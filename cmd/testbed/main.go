/*
testbed runs an A/B on the defender mod and says what it found.

	testbed -map mvm_decoy -mission mvm_decoy_advanced \
	        -arm on:sm_redbots_feature_watch_idle_bots=1 \
	        -arm off:sm_redbots_feature_watch_idle_bots=0 \
	        -attempts 3 -waves 2

It replaces a pile of shell that produced measurements nobody should have
believed. What it refuses to do is the point:

  - one arm at a time from start to finish. That made "the first arm" and "the
    first round on a freshly recreated server" the same thing, and five watchdog
    trips in a day all landed in whichever arm ran first. The arms interleave and
    the order flips every round.
  - two runners at once. A lock file holds the test-bed, and a second runner
    says who has it rather than quietly fighting for the map.
  - a stale plugin. The version the server has loaded is compared with the one
    on disk, and a mismatch stops the run instead of measuring a two hour old
    build.
  - an empty run. RED must hold defenders and the mission must be the one asked
    for, or the attempt is a refusal rather than a file full of zeros.
*/
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/m-this/tf2-mvm-bots-go/internal/lab"
	"github.com/m-this/tf2-mvm-bots-go/internal/rcon"
	"github.com/m-this/tf2-mvm-bots-go/internal/wave"
)

type arms []arm

type arm struct {
	name  string
	cvars string
}

func (a *arms) String() string { return fmt.Sprint(*a) }

func (a *arms) Set(value string) error {
	name, cvars, found := strings.Cut(value, ":")
	if !found || name == "" {
		return fmt.Errorf("an arm is name:cvars, not %q", value)
	}
	*a = append(*a, arm{name: name, cvars: cvars})
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "\ntestbed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		mapName  = flag.String("map", "mvm_decoy", "the map to play")
		mission  = flag.String("mission", "", "the popfile to play, empty for the map's own")
		waves    = flag.Int("waves", 2, "wave results to wait for per attempt")
		attempts = flag.Int("attempts", 3, "attempts per arm")
		timeout  = flag.Duration("timeout", 25*time.Minute, "how long one attempt may take")
		team     = flag.String("team", "scout,soldier,demoman,heavyweapons,engineer,medic", "the RED lineup")
		defend   = flag.Int("defenders", 6, "defenders RED must hold before a wave counts")
		out      = flag.String("out", "results", "where to write the run")
		tag      = flag.String("tag", "run", "what to call this run")
		build    = flag.Bool("build", true, "compile and restart the server before the first attempt")
		jumpTo   = flag.Int("wave", 0, "start at this wave of the mission, 0 for the first")
		down     = flag.Bool("down", false, "stop the server when the run is done")
		maps     = flag.String("maps", "", "run every map in this list instead of -map, space separated")
		list     arms
	)
	/* The puppets, which is how a fault that needs a person on RED gets measured

	Each one takes a RED seat, so -defenders and -team come down by one for each
	or the settle refuses the attempt. See mvm-n4s. */
	puppets := flag.Int("puppets", 0, "bodies to seat on RED standing in for players, each taking a defender's seat")
	puppetClass := flag.String("puppet-class", "", "the class they join as, empty for the plugin's own (scout)")
	puppetCalls := flag.Bool("puppet-calls", false, "have them press MEDIC! at every poll while a wave runs")
	replay := flag.String("replay", "", "a player's server.cfg, from a debug bundle, whose settings this run plays instead of the flags")
	flag.Var(&list, "arm", "name:cvars, repeatable. Comma separated cvars, key=value")
	flag.Parse()

	if len(list) == 0 {
		return errors.New("no arms: give at least one -arm name:cvars")
	}
	if *puppetCalls && *puppets == 0 {
		return errors.New("-puppet-calls with no puppets: nothing would press the call")
	}

	replayed := map[string]string{}
	if *replay != "" {
		found, err := readServerCfg(*replay)
		if err != nil {
			return err
		}
		replayed = found
	}

	root, err := repoRoot()
	if err != nil {
		return err
	}

	release, err := hold(filepath.Join(root, "testbed", ".lock"))
	if err != nil {
		return err
	}
	defer release()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	say := func(format string, args ...any) {
		fmt.Printf("[testbed] "+format+"\n", args...)
	}

	// Said out loud, because a run that quietly played settings other than the
	// ones on the command line is the whole fault this flag exists for.
	for _, setting := range sortedPairs(replayed) {
		say("replaying %s", setting)
	}
	l := lab.Lab{
		Client: rcon.Client{Addr: address(), Password: password(), Timeout: 15 * time.Second},
		Say:    say,
	}

	if *build {
		say("building")
		if err := compile(ctx, root); err != nil {
			return err
		}
		say("restarting the server onto it")
		compose := filepath.Join(root, "testbed", "compose.yml")
		if err := lab.Compose(ctx, compose, containerEnv(*mapName, *team, *defend, puppet{count: *puppets, class: *puppetClass}, replayed), "up", "-d", "--force-recreate"); err != nil {
			return err
		}
	}
	if err := l.WaitForRcon(ctx, 20*time.Minute); err != nil {
		return err
	}
	if err := checkVersion(root, l, say); err != nil {
		return err
	}

	if *down {
		defer func() {
			say("stopping the server")
			_ = lab.Compose(context.Background(), filepath.Join(root, "testbed", "compose.yml"), nil, "stop")
		}()
	}

	// A sweep is the same run repeated, so it is a loop here, not a script reimplementing the preconditions.
	played := strings.Fields(*maps)
	if len(played) == 0 {
		played = []string{*mapName}
	}

	for _, name := range played {
		results, err := playArms(ctx, l, list, options{
			root: root, mapName: name, mission: *mission, waves: *waves,
			attempts: *attempts, timeout: *timeout, team: *team, defenders: *defend,
			out: filepath.Join(root, *out), tag: *tag, jump: *jumpTo, say: say,
			puppets: puppet{count: *puppets, class: *puppetClass, calls: *puppetCalls},
		})
		if err != nil {
			return err
		}
		fmt.Print(report(*tag, name, *mission, results))
	}
	return nil
}

type options struct {
	root, mapName, mission, team, out, tag string
	waves, attempts, defenders, jump       int
	timeout                                time.Duration
	say                                    func(string, ...any)
	puppets                                puppet
}

/*
puppet is the run's stand-in for a player, in the three things a run decides
about it: how many, what class, and whether they call for a medic.

Named rather than three ints and a bool at every call site, because count and
class are both what the seat is and would swap silently.
*/
type puppet struct {
	count int
	class string
	calls bool
}

func report(tag, mapName, mission string, got []wave.Arm) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n=== %s on %s", tag, mapName)
	if mission != "" {
		fmt.Fprintf(&b, ", %s", mission)
	}
	b.WriteString(" ===\n")

	switch len(got) {
	case 1:
		a := got[0]
		fmt.Fprintf(&b, "\n%s: %d attempts, %d waves, %d cleared, %d crashes, %d empty\n",
			a.Name, a.Attempts, len(a.Results), a.Cleared(), a.Crashes, a.Empty)
	default:
		// The last arm is the control: the off switch is written last by habit,
		// and the treated arm is the one being asked about.
		control := got[len(got)-1]
		for _, treated := range got[:len(got)-1] {
			b.WriteString(wave.Compare(treated, control))
		}
	}
	return b.String()
}

var versionInSource = regexp.MustCompile(`version\s*=\s*"([^"]+)"`)

// checkVersion refuses a run against a plugin the server loaded before the last
// build. That mistake cost a full A/B: --no-build left a two hour old mod in
// memory and the run reported the fix doing nothing.
func checkVersion(root string, l lab.Lab, say func(string, ...any)) error {
	source, err := os.ReadFile(filepath.Join(root, "source", "tf2_defenderbots.sp"))
	if err != nil {
		return err
	}
	m := versionInSource.FindSubmatch(source)
	if m == nil {
		return errors.New("cannot find the version in source/tf2_defenderbots.sp")
	}
	want := string(m[1])

	got, err := l.PluginVersion()
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("%w: the server has %s loaded and the source says %s, so build first",
			lab.ErrPrecondition, got, want)
	}
	say("the server is running %s, which is what the source says", got)
	return nil
}

/*
containerEnv is the run's shape, in the variables compose.yml reads.

The container writes server.cfg from these at start up, and the map load reads
that file. Setting the lineup over rcon instead does not survive, because the
exec that follows a map load puts the file's values back: a run that set the
team and then loaded the map found RED empty and refused, which is how this was
found.
*/
func sortedPairs(of map[string]string) []string {
	out := make([]string, 0, len(of))
	for key, value := range of {
		out = append(out, key+"="+value)
	}
	sort.Strings(out)

	return out
}

func containerEnv(mapName, team string, size int, p puppet, replayed map[string]string) []string {
	env := map[string]string{
		"TESTBED_MAP":           mapName,
		"TESTBED_BOT_TEAM_COMP": team,
		"TESTBED_BOT_TEAM_SIZE": strconv.Itoa(size),
		"TESTBED_HOST":          "1",
		"TESTBED_PUPPETS":       strconv.Itoa(p.count),
		"TESTBED_PORT":          envOr("TESTBED_PORT", "27025"),
		"TESTBED_RCONPW":        envOr("TESTBED_RCONPW", "testbed"),
	}

	// The class only when it was asked for, so an empty flag leaves the
	// plugin's own default rather than writing an empty joinclass.
	if p.class != "" {
		env["TESTBED_PUPPET_CLASS"] = p.class
	}

	// A replayed server.cfg wins, because the point of naming one is to play
	// what it plays rather than what the flags default to.
	for key, value := range replayed {
		env[key] = value
	}

	out := make([]string, 0, len(env))
	for key, value := range env {
		out = append(out, key+"="+value)
	}
	sort.Strings(out)

	return out
}
