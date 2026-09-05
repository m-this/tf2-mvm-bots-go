package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/m-this/tf2-mvm-bots-go/internal/lab"
	"github.com/m-this/tf2-mvm-bots-go/internal/wave"
)

/*
playArms interleaves the arms and alternates which one goes first.

Arm at a time, which is what this used to do, made "the first arm" and "the
first round on a freshly recreated server" the same thing. Five watchdog trips
in one day all landed in the arm that ran first and none in the second, across
four features, two of which provably never fired. The crash column was reading
arm order. See mvm-p4x.

Alternating costs nothing and removes it: each round plays every arm, and the
order flips every round, so whatever the first round of a server is worth is
shared out instead of being paid by one arm.
*/
func playArms(ctx context.Context, l lab.Lab, list arms, o options) ([]wave.Arm, error) {
	got := make([]wave.Arm, len(list))
	for i, a := range list {
		got[i] = wave.Arm{Name: a.name}
	}

	for round := 1; round <= o.attempts; round++ {
		for _, at := range roundOrder(len(list), round) {
			o.say("=== %s attempt %d of %d", list[at].name, round, o.attempts)
			if err := playInto(ctx, l, list[at], o, round, &got[at]); err != nil {
				/* What completed is reported with the refusal.

				A noise-floor run played five attempts of six, refused the
				sixth, and reported nothing: five files of good data read as a
				failure. The refusal stands and the run exits non-zero; the
				caller sees what it has and decides whether it is enough. */
				if errors.Is(err, lab.ErrPrecondition) {
					o.say("stopping here: %v", err)
					o.say("%d of %d rounds completed; what follows is the completed part", round-1, o.attempts)
				}
				return got, err
			}
		}
	}
	return got, nil
}

// Odd rounds in order, even rounds reversed, so no arm keeps the first slot.
func roundOrder(count, round int) []int {
	order := make([]int, count)
	for i := range order {
		if round%2 == 1 {
			order[i] = i
		} else {
			order[i] = count - 1 - i
		}
	}
	return order
}

func playInto(ctx context.Context, l lab.Lab, a arm, o options, round int, got *wave.Arm) error {
	got.Attempts++

	path := filepath.Join(o.out, fmt.Sprintf("%s-%s-%d.jsonl", o.tag, a.name, round))
	results, crashed, err := playOnce(ctx, l, a, o, path)
	switch {
	case errors.Is(err, context.Canceled):
		return err
	case errors.Is(err, lab.ErrPrecondition):
		// Loud, and the run stops: a precondition that fails once fails the
		// same way every time, and grinding through the rest wastes an hour
		// to produce nothing.
		return err
	case err != nil:
		o.say("attempt %d did not finish: %v", round, err)
	}

	if crashed {
		got.Crashes++
		if words := lastWords(ctx); words != "" {
			o.say("the server's last words:\n%s", words)
		}
	}
	if len(results) == 0 {
		got.Empty++
		o.say("attempt %d produced no wave result", round)
		return nil
	}
	got.Results = append(got.Results, results...)
	o.say("attempt %d: %d waves, %d cleared", round, len(results), cleared(results))

	/* Said on every run, whatever the run was measuring

	Six defenders doing the work of five still clears waves, so an arm comparison
	never shows a bot that was handed nothing to do. The stock sniper sat in these
	files for a day: the samples naming his empty stack were written every half
	second and nothing read them. See mvm-bj8. */
	if report := wave.IdleReport(path); report != "" {
		o.say("%s", report)
	}

	return nil
}

/*
checkPuppets refuses an attempt that asked for a player and got none.

RED is six seats and the host already holds one, so a run that forgot to bring
-defenders down finds the last seat refused, and the seat it loses is the
puppet: the mod fills its own before this is asked. Nothing else about the file
would say so, and a full results file with no player in it reads as the medic
ignoring a call it never received.
*/
func checkPuppets(l lab.Lab, o options) error {
	if o.puppets.count == 0 {
		return nil
	}

	roster, err := l.Roster()
	if err != nil {
		return err
	}
	if roster.Puppets < o.puppets.count {
		return fmt.Errorf("%w: %d puppets are on RED and the run asked for %d, so drop -defenders and -team by one for each",
			lab.ErrPrecondition, roster.Puppets, o.puppets.count)
	}

	status, err := l.PuppetStatus()
	if err != nil {
		return err
	}
	o.say("%s", status)

	return nil
}

func cleared(results []wave.Result) int {
	n := 0
	for _, r := range results {
		if r.Outcome == "cleared" {
			n++
		}
	}
	return n
}

func playOnce(ctx context.Context, l lab.Lab, a arm, o options, path string) ([]wave.Result, bool, error) {
	if err := clearStats(ctx, o.root); err != nil {
		return nil, false, err
	}
	if err := l.LoadMission(ctx, o.mapName, o.mission); err != nil {
		return nil, false, err
	}

	/* The arm goes on after the map load, not before.
	   A map load execs server.cfg, and that file puts the container's own values back: an arm set
	   first is an arm the server has forgotten by the time the wave starts. */
	if _, err := l.Do("sm_redbots_manager_team_composition \"" + o.team + "\""); err != nil {
		return nil, false, err
	}

	/* Before the arm cvars, not after: the arm is what is under test and gets
	   the last word, so a run comparing the nextbot player test against itself
	   can still turn it off in one of its arms. */
	if err := l.SeatPuppets(o.puppets.count); err != nil {
		return nil, false, err
	}

	for _, pair := range strings.Split(a.cvars, ",") {
		if pair = strings.TrimSpace(pair); pair == "" {
			continue
		}
		key, value, found := strings.Cut(pair, "=")
		if !found {
			return nil, false, fmt.Errorf("an arm cvar is key=value, not %q", pair)
		}
		if _, err := l.Do(key + " " + value); err != nil {
			return nil, false, err
		}
	}

	/* Every attempt starts from the same wave, between waves.

	A mission that is not reloaded carries on from the wave it reached, so
	each attempt started wherever the last one stopped: one noise-floor run
	played waves 1 and 2 against 2 and 3, then 6 and 7 against 4 and 5, and
	wave 8 of a mission is not wave 1. A wave already in flight was counted
	for the arm that found it running. The jump restarts the round at the
	wave asked for, or the first, and leaves the server between waves, which
	is what the settle below expects to find. */
	start := max(o.jump, 1)
	o.say("starting at wave %d", start)
	if err := l.JumpToWave(ctx, start); err != nil {
		return nil, false, err
	}
	if err := l.Settle(ctx, o.defenders, 3*time.Minute); err != nil {
		return nil, false, err
	}
	if err := checkPuppets(l, o); err != nil {
		return nil, false, err
	}
	if o.jump > 0 {
		o.say("jumping to wave %d", o.jump)
		if err := l.JumpToWave(ctx, o.jump); err != nil {
			return nil, false, err
		}
	}

	results, crashed, reason := waitForWaves(ctx, l, o)
	if err := copyStats(ctx, o.root, path); err != nil {
		return results, crashed, err
	}
	if err := writeRunRecord(path, runRecord{
		Tag: o.tag, Arm: a.name, Cvars: a.cvars, Map: o.mapName, Mission: o.mission,
		Team: o.team, Defenders: o.defenders, Puppets: o.puppets.count, PuppetCalls: o.puppets.calls,
		Waves: o.waves, StartWave: max(o.jump, 1), Plugin: o.plugin, At: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return results, crashed, err
	}
	if len(results) == 0 && reason != "" && !crashed {
		// A run that produced nothing for a reason the watcher can name is
		// worth saying out loud, and worth keeping out of the numbers.
		o.say("nothing usable from this attempt: %s", reason)
	}
	return results, crashed, nil
}

/*
waitForWaves watches a running wave rather than only waiting for it.

A wave that goes wrong looks like a slow one for the first minute and like a
finished one at the end, so the difference has to be caught while it happens.
The watcher's reasons name what went wrong, which is worth more than a timeout
and an empty file.
*/
func waitForWaves(ctx context.Context, l lab.Lab, o options) ([]wave.Result, bool, string) {
	// Named for the bed: two beds staging into one file read each other's waves.
	staged := filepath.Join(os.TempDir(), bed()+"-stats.jsonl")

	watcher := &lab.Watcher{
		WantDefenders: o.defenders,
		// A wave lasts minutes and the polls are twenty seconds apart, so five
		// polls is well past a between-rounds lull and well short of a wave.
		PatienceRobots: 5,
		PatienceSilent: 6,
		// Three reads apart, so a crash is called after a minute of silence
		// rather than on one long frame.
		PatienceQuiet: 3,
	}

	var found []wave.Result
	health, err := l.Wait(ctx, watcher, 20*time.Second, o.timeout, func() (int, int, bool) {
		lines, results := readStagedWithLines(ctx, o.root, staged)
		found = results
		begun := wave.Begun(staged)

		/* The call rides on the poll rather than on a clock of its own.

		Twenty seconds against a ten second answer time is a player who calls,
		waits, and calls again, which is what a player does when nobody comes.
		It also leaves half of every gap with no call live, so a beam that sits
		on the puppet throughout is the player rule and not the call. */
		if o.puppets.calls && begun {
			if n, err := l.CallForMedic(); err != nil {
				o.say("the medic call did not reach a puppet: %v", err)
			} else if n == 0 {
				o.say("no puppet was alive to call for a medic")
			}
		}

		if len(results) >= o.waves {
			return lines, len(results), begun
		}
		return lines, 0, begun
	})
	if err != nil {
		return found, false, "cancelled"
	}

	if len(found) >= o.waves {
		return found, false, ""
	}
	o.say("%s", health.Reason)
	return found, health.Fatal, health.Reason
}

// The line count is what tells a quiet wave from a dead plugin.
func readStagedWithLines(ctx context.Context, root, staged string) (int, []wave.Result) {
	if err := copyStats(ctx, root, staged); err != nil {
		return 0, nil
	}
	body, err := os.ReadFile(staged)
	if err != nil {
		return 0, nil
	}
	lines := bytes.Count(body, []byte("\n"))

	results, err := wave.Read(staged)
	if err != nil {
		return lines, nil
	}
	return lines, results
}

const remoteStats = "/home/steam/tf-dedicated/tf/addons/sourcemod/logs/mvmbots_stats.jsonl"

func clearStats(ctx context.Context, _ string) error {
	return exec.CommandContext(ctx, "docker", "exec", container(),
		"sh", "-c", "rm -f "+remoteStats).Run()
}

func copyStats(ctx context.Context, _, to string) error {
	// 0o750: the results are this developer's own, and nothing else on the
	// machine reads them.
	if err := os.MkdirAll(filepath.Dir(to), 0o750); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "docker", "cp", container()+":"+remoteStats, to)
	cmd.Stderr = nil
	return cmd.Run()
}
