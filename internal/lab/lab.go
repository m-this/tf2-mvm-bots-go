/*
Package lab drives one test-bed server.

Every rule here exists because breaking it produced a run that looked fine and
measured nothing:

  - one runner at a time, held by a lock file. Two scripts waiting on the same
    "is a run going" check both started, and each map change pulled the other's
    mission out from under it.
  - the map is loaded before the mission is named. A changelevel resets
    tf_mvm_popfile to the map's own mission, and naming a mission on a map that
    has been up for hours reports the right popfile with none of its robots.
  - the plugin the server has loaded is checked against the one on disk. A run
    with --no-build measured a two hour old build and said the fix did nothing.
  - RED is checked for defenders and BLU for robots before the wave counts.
    Twenty two robots and an empty RED still produces a file, and the file is
    empty of everything that matters.
*/
package lab

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/m-this/tf2-mvm-bots-go/internal/rcon"
)

// Lab is the running server, reached over rcon. Every command the test-bed
// sends goes through here, so a server that stopped answering is one error and
// not one per call site.
type Lab struct {
	Client rcon.Client
	Say    func(format string, args ...any)
}

// ErrPrecondition is a run that must not be believed. Separate from a transport
// error because this one means the server is fine and the run is not.
var ErrPrecondition = errors.New("the run did not meet its preconditions")

func (l Lab) say(format string, args ...any) {
	if l.Say != nil {
		l.Say(format, args...)
	}
}

// Do sends one console command and returns what the server printed.
func (l Lab) Do(command string) (string, error) {
	out, err := l.Client.Do(command)
	if err != nil {
		return out, fmt.Errorf("%q: %w", command, err)
	}
	return out, nil
}

// WaitForRcon blocks until the server answers, which is the whole of it being up.
func (l Lab) WaitForRcon(ctx context.Context, limit time.Duration) error {
	deadline := time.Now().Add(limit)
	for {
		if _, err := l.Client.Do("status"); err == nil {
			return nil
		} else if errors.Is(err, rcon.ErrAuth) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("the server did not answer rcon within %s", limit)
		}
		if err := sleep(ctx, 5*time.Second); err != nil {
			return err
		}
	}
}

var (
	mapLine    = regexp.MustCompile(`(?m)^map\s+:\s+(\S+)`)
	popLine    = regexp.MustCompile(`Current popfile is:\s*(\S+)`)
	versionRow = regexp.MustCompile(`"Defender TFBots"\s+\(([^)]+)\)`)
)

// Roster is who is on the server: the humans, and the bots the mod put there.
type Roster struct {
	Humans    int
	Bots      int
	Robots    int // BLU, which the game names TFBot
	Defenders int // ours, which the mod gives a name of its own
	Puppets   int // the bodies a run seats to stand in for players
	Host      bool
}

var rosterLine = regexp.MustCompile(`mvmbots_roster red=(\d+) blu=(\d+) humans=(\d+) host=(\d+) puppets=(\d+)`)

/*
Roster asks the game who is on each team.

Through the host plugin rather than through status, because status lists names
and never says which side anybody is on. The robots are named by class on most
maps, so a runner counting "not a robot name" as a defender read fourteen Pyros
on BLU as a full RED and passed a run that had no defenders at all.
*/
func (l Lab) Roster() (Roster, error) {
	out, err := l.Do("mvmbots_roster")
	if err != nil {
		return Roster{}, err
	}
	return readRoster(out)
}

// readRoster is the line on its own, so who counts as a defender can be tested
// without a server.
func readRoster(out string) (Roster, error) {
	m := rosterLine.FindStringSubmatch(out)
	if m == nil {
		return Roster{}, fmt.Errorf("%w: the host plugin did not answer mvmbots_roster, so nobody can say who is on which team", ErrPrecondition)
	}

	var r Roster
	red, _ := strconv.Atoi(m[1])
	r.Robots, _ = strconv.Atoi(m[2])
	r.Humans, _ = strconv.Atoi(m[3])
	host, _ := strconv.Atoi(m[4])
	r.Puppets, _ = strconv.Atoi(m[5])
	r.Host = host > 0

	// The host and the puppets hold RED seats and neither plays the mission,
	// so neither is a defender. A puppet counted as one is a run that thinks
	// it has six bots and has five.
	r.Defenders = red - host - r.Humans - r.Puppets
	if r.Defenders < 0 {
		r.Defenders = 0
	}
	r.Bots = red + r.Robots - r.Humans
	return r, nil
}

/*
SeatPuppets asks for that many bodies on RED standing in for players.

The mod is told to answer the player question by the nextbot at the same time,
because a puppet it reads as one of its own bots is a body with a name and
nothing else: IsTFBotPlayer is IsFakeClient without that switch, and every seat
a plugin can create is a fake client. See mvm-z83.93.
*/
func (l Lab) SeatPuppets(count int) error {
	if count > 0 {
		if _, err := l.Do("sm_redbots_feature_bot_test_by_nextbot 1"); err != nil {
			return err
		}
	}
	_, err := l.Do("mvmbots_puppet_count " + strconv.Itoa(count))
	return err
}

var calledLine = regexp.MustCompile(`mvmbots_puppet_call called=(\d+)`)

// CallForMedic presses MEDIC! on every seated puppet and says how many pressed
// it. Nought is a run measuring nothing, so the caller is the one to complain.
func (l Lab) CallForMedic() (int, error) {
	out, err := l.Do("mvmbots_puppet_call")
	if err != nil {
		return 0, err
	}
	m := calledLine.FindStringSubmatch(out)
	if m == nil {
		return 0, fmt.Errorf("%w: the host plugin did not answer mvmbots_puppet_call, so no puppet is on RED", ErrPrecondition)
	}
	n, _ := strconv.Atoi(m[1])
	return n, nil
}

// PuppetStatus is what the puppets are doing right now, one line each, which is
// how a call is watched while the wave runs rather than read afterwards.
func (l Lab) PuppetStatus() (string, error) {
	out, err := l.Do("mvmbots_puppet_status")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// CurrentMap is the map the server is playing, read out of status.
func (l Lab) CurrentMap() (string, error) {
	out, err := l.Do("status")
	if err != nil {
		return "", err
	}
	if m := mapLine.FindStringSubmatch(out); m != nil {
		return m[1], nil
	}
	return "", fmt.Errorf("status said nothing about a map: %q", trim(out))
}

// PopFile is the mission the server has loaded.
func (l Lab) PopFile() (string, error) {
	out, err := l.Do("tf_mvm_popfile")
	if err != nil {
		return "", err
	}
	if m := popLine.FindStringSubmatch(out); m != nil {
		return m[1], nil
	}
	return "", fmt.Errorf("could not read the popfile from %q", trim(out))
}

// PluginVersion is the version of the mod the server has loaded, which is not
// always the version on disk.
func (l Lab) PluginVersion() (string, error) {
	out, err := l.Do("sm plugins list")
	if err != nil {
		return "", err
	}
	if m := versionRow.FindStringSubmatch(out); m != nil {
		return m[1], nil
	}
	return "", fmt.Errorf("the defender mod is not in the plugin list: %q", trim(out))
}

/*
LoadMission puts the server on a map and a mission, in that order.

The order is the whole point. A changelevel resets tf_mvm_popfile, so naming the
mission first throws the name away; and naming it on a map already up leaves
missions like mvm_mannworks_intermediate2 reporting themselves as loaded with
none of their robots built.
*/
func (l Lab) LoadMission(ctx context.Context, mapName, mission string) error {
	/* server.cfg runs on map load, and the first map can load before the entrypoint has written
	   ours. Executing it by hand costs nothing when it was already read, and is the difference
	   between measuring six bots and none: without it the mod has no team composition and RED stays
	   empty, which the settle step then refuses. */
	if _, err := l.Do("exec server.cfg"); err != nil {
		l.say("could not exec server.cfg, which usually means RED will stay empty: %v", err)
	}

	/* Only load the map when the server is not already on it.

	A changelevel costs more than the time. The mod disables its bots in
	OnMapStart and waits to be started again, and a run that changed level onto
	the map it was already on found RED at nought defenders every time, on a map
	that plays perfectly from a fresh container. The container recreate above is
	already a map load, and it is the one the mission is named after. */
	if current, err := l.CurrentMap(); err != nil || current != mapName {
		l.say("loading %s", mapName)
		if _, err := l.Do("changelevel " + mapName); err != nil {
			// A changelevel drops the connection, which is not a failure.
			l.say("the changelevel closed the connection, which is expected")
		}
		if err := sleep(ctx, 10*time.Second); err != nil {
			return err
		}
		if err := l.WaitForRcon(ctx, 3*time.Minute); err != nil {
			return err
		}
		if err := l.waitForMap(ctx, mapName); err != nil {
			return err
		}
		if err := sleep(ctx, 20*time.Second); err != nil {
			return err
		}
	} else {
		l.say("already on %s", mapName)
	}

	if mission == "" {
		return nil
	}
	l.say("naming mission %s", mission)
	if _, err := l.Do("tf_mvm_popfile " + mission); err != nil {
		return err
	}
	if err := sleep(ctx, 15*time.Second); err != nil {
		return err
	}

	loaded, err := l.PopFile()
	if err != nil {
		return err
	}
	if !strings.Contains(loaded, mission) {
		return fmt.Errorf("%w: asked for %s and the server is playing %s", ErrPrecondition, mission, loaded)
	}

	// The map load reads server.cfg again, and the mission was named after it.
	// Executing it here is what puts the lineup back for the round about to start.
	if _, err := l.Do("exec server.cfg"); err != nil {
		l.say("could not exec server.cfg after the map load: %v", err)
	}
	return nil
}

func (l Lab) waitForMap(ctx context.Context, want string) error {
	deadline := time.Now().Add(3 * time.Minute)
	for {
		if got, err := l.CurrentMap(); err == nil && got == want {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%w: the server never reached %s", ErrPrecondition, want)
		}
		if err := sleep(ctx, 5*time.Second); err != nil {
			return err
		}
	}
}

/*
Settle gets the round going and then holds the run to its lineup.

The order took a deadlock to get right. The mod fills RED around the round
starting, so waiting for the full lineup before nudging the round waits for
something the nudge is what causes. The shell waited for any bot at all and then
nudged, which worked and never checked the lineup it got.

So both: wait for the mod to show signs of life, nudge, then insist on the
lineup. That last wait is the precondition, and failing it is a refusal, because
a wave nobody fought writes a file of zeros that reads as a mission nobody could
win.
*/
func (l Lab) Settle(ctx context.Context, wantDefenders int, limit time.Duration) error {
	l.say("waiting for the mod to start filling RED")
	if err := l.waitForDefenders(ctx, 1, limit/2); err != nil {
		l.say("nobody on RED yet, nudging the round anyway")
	}
	if err := sleep(ctx, 5*time.Second); err != nil {
		return err
	}
	if _, err := l.Do("mp_tournament_restart"); err != nil {
		return err
	}

	l.say("waiting for RED to hold %d", wantDefenders)
	return l.waitForDefenders(ctx, wantDefenders, limit)
}

// waitForDefenders is bounded, and says how far RED actually got: a mod that
// never started and one that filled four of six are different faults.
func (l Lab) waitForDefenders(ctx context.Context, want int, limit time.Duration) error {
	deadline := time.Now().Add(limit)
	best := 0
	for {
		roster, err := l.Roster()
		if err == nil {
			if roster.Defenders > best {
				best = roster.Defenders
			}
			if roster.Defenders >= want {
				l.say("RED holds %d defenders", roster.Defenders)
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%w: RED reached %d defenders and not %d", ErrPrecondition, best, want)
		}
		if err := sleep(ctx, 5*time.Second); err != nil {
			return err
		}
	}
}

/*
Compose runs a docker compose subcommand against the test-bed.

env is passed through because compose.yml reads the run's shape from it:
TESTBED_MAP, TESTBED_BOT_TEAM_COMP and the rest become the server.cfg the
container writes. A compose started without them writes a server.cfg naming no
lineup, and the mod then fills RED with nobody.
*/
func Compose(ctx context.Context, file string, env []string, args ...string) error {
	full := append([]string{"compose", "-f", file}, args...)
	cmd := exec.CommandContext(ctx, "docker", full...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	return cmd.Run()
}

func sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

func trim(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}

/*
JumpToWave starts the mission partway in, which is how a wave three report is
made without playing waves one and two for half an hour.

The jump is a cheat command, so cheats go on for it and back to what they were
after. Restoring the previous value rather than zero: a server somebody had set
cheats on deliberately should not have them turned off by a measurement.
*/
func (l Lab) JumpToWave(ctx context.Context, wave int) error {
	before, err := l.Do("sv_cheats")
	if err != nil {
		return err
	}
	restore := "0"
	if strings.Contains(before, `"sv_cheats" = "1"`) {
		restore = "1"
	}

	if err := sleep(ctx, 5*time.Second); err != nil {
		return err
	}
	if _, err := l.Do("sv_cheats 1"); err != nil {
		return err
	}
	if _, err := l.Do(fmt.Sprintf("tf_mvm_jump_to_wave %d", wave)); err != nil {
		return err
	}
	_, err = l.Do("sv_cheats " + restore)
	return err
}
