package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/m-this/tf2-mvm-bots-go/internal/lab"
)

/*
How the server is started, and the only thing a native run differs by.

There used to be two runners. testbed/run-native.sh was 228 lines of shell that
played a mission on a native server with none of the guards this one was written
for: it did not check the loaded plugin against the source, did not interleave
arms, did not record the machine, did not watch the wave, did not hold the bed
lock, and reached the server through rcon.py. Every fault the package comment
enumerates was available again by taking that path, and the README offered it as
a peer. Worse, it wrote results files the reports could not tell apart from a
real run's.

It was not optional work: mvm-y7e is open on native Linux servers crashing far
more than Docker, and the measurement that would settle it is the one the runner
could not vouch for. So native is a mode here rather than a second program, and
this is the whole of the difference between them. See mvm-tjb.
*/
type server interface {
	// Recreate puts the server on a map from nothing, which is what a first
	// map and every map of a sweep both do. A changelevel is not the same
	// thing: it drops the connection under the exec that writes the lineup.
	Recreate(ctx context.Context, mapName string, env []string) error
	Stop(ctx context.Context) error

	// ClearStats and CopyStats move the statistics file the plugin writes.
	ClearStats(ctx context.Context) error
	CopyStats(ctx context.Context, to string) error

	// LogSince is what the server printed since a moment, for crash triage. The
	// window matters more than the content: an unbounded read counts crashes
	// from hours ago, which is mvm-427.
	LogSince(ctx context.Context, since time.Time) string
	// NewestCore is the core to look at and the command that symbolises it.
	NewestCore(ctx context.Context) (core, symbolise string)
	// Space is the free space where the game lives, the cores sitting in it and
	// what they weigh. Not every server can be asked, which is the last return.
	Space(ctx context.Context) (freeMB int64, cores int, weightMB int64, ok bool)
}

// The statistics file, at the path the plugin writes it to under the game tree.
const statsUnderGame = "addons/sourcemod/logs/mvmbots_stats.jsonl"

/*
dockerBed is the container test-bed, which is what a run means unless it says
otherwise. One compose project per bed, one container in it.
*/
type dockerBed struct {
	compose string
	root    string
}

const dockerGameDir = "/home/steam/tf-dedicated"

func (d dockerBed) Recreate(ctx context.Context, _ string, env []string) error {
	return lab.Compose(ctx, d.compose, env, "up", "-d", "--force-recreate")
}

func (d dockerBed) Stop(ctx context.Context) error {
	return lab.Compose(ctx, d.compose, nil, "stop")
}

func (d dockerBed) ClearStats(ctx context.Context) error {
	return exec.CommandContext(ctx, "docker", "exec", container(),
		"sh", "-c", "rm -f "+dockerGameDir+"/tf/"+statsUnderGame).Run()
}

func (d dockerBed) CopyStats(ctx context.Context, to string) error {
	if err := os.MkdirAll(filepath.Dir(to), 0o750); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "docker", "cp", container()+":"+dockerGameDir+"/tf/"+statsUnderGame, to)
	cmd.Stderr = nil
	return cmd.Run()
}

func (d dockerBed) LogSince(ctx context.Context, since time.Time) string {
	out, err := exec.CommandContext(ctx, "docker", "logs", "--since", stamp(since), container()).CombinedOutput()
	if err != nil {
		return ""
	}
	return string(out)
}

func (d dockerBed) NewestCore(ctx context.Context) (string, string) {
	out, err := exec.CommandContext(ctx, "docker", "exec", container(),
		"sh", "-c", "ls -t "+dockerGameDir+"/core.* 2>/dev/null | head -1").Output()
	if err != nil {
		return "", ""
	}
	core := strings.TrimSpace(string(out))
	if core == "" {
		return "", ""
	}
	local := filepath.Join(os.TempDir(), filepath.Base(core))
	return core, fmt.Sprintf("docker cp %s:%s %s && TESTBED_NATIVE_ROOT=%s sh %s %s",
		container(), core, local, dockerGameDir, filepath.Join(d.root, "testbed", "symbolise-core.sh"), local)
}

func (d dockerBed) Space(ctx context.Context) (int64, int, int64, bool) {
	return spaceUnder(ctx, dockerGameDir, func(command string) *exec.Cmd {
		return exec.CommandContext(ctx, "docker", "exec", container(), "sh", "-c", command)
	})
}

/*
nativeBed is srcds run as a process on this machine, which is the thing the
container cannot show.

A native server is reported to crash far more often than the same mod under
Docker, and Docker restarts srcds by itself, so the same crash reads as a hiccup
in one and ends the session in the other: mvm-y7e. The game tree is a copy and
never a share, because installing plugins over a tree a live server is reading
is how the container bed once produced five crashes in ten minutes.
*/
type nativeBed struct {
	root string // the plugin tree
	game string // TESTBED_NATIVE_ROOT, the game copy this bed plays
	port string

	running *exec.Cmd
	log     string
}

// NativeRootEnv names the game copy a native run plays. Its default is the one
// seed-native.sh writes.
const NativeRootEnv = "TESTBED_NATIVE_ROOT"

func nativeRoot() string {
	if root := os.Getenv(NativeRootEnv); root != "" {
		return root
	}
	return filepath.Join(os.Getenv("HOME"), "tf2-native", "tf-dedicated")
}

/*
Recreate installs the mod into the game copy and starts srcds on the map.

The installers are entrypoint.sh's, sourced rather than reimplemented: the
container and a native run must install the same files the same way, or the two
beds differ in something other than the one thing the native bed exists to vary.
TESTBED_DEFINE_ONLY stops the script running its own start-up.
*/
func (n *nativeBed) Recreate(ctx context.Context, mapName string, env []string) error {
	if err := n.Stop(ctx); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(n.game, "srcds_linux")); err != nil {
		return fmt.Errorf("%w: no game at %s; seed it once with testbed/seed-native.sh, or set %s",
			lab.ErrPrecondition, n.game, NativeRootEnv)
	}

	install := exec.CommandContext(ctx, "sh", "-c",
		". "+shellQuote(filepath.Join(n.root, "testbed", "entrypoint.sh"))+"; install_addons && install_server_cfg")
	install.Dir = n.root
	install.Env = append(n.env(env), "TESTBED_DEFINE_ONLY=1")
	if out, err := install.CombinedOutput(); err != nil {
		return fmt.Errorf("installing the mod into %s failed:\n%s", n.game, tail(string(out), 20))
	}

	_ = os.Remove(filepath.Join(n.game, "tf", statsUnderGame))
	// Cores land in the working directory. The soft limit is nought on most
	// machines, and a crash with no core is a crash nobody can look into.
	for _, core := range coreFiles(n.game) {
		_ = os.Remove(core)
	}

	n.log = filepath.Join(n.root, "testbed", "build", "native-server.log")
	logFile, err := os.OpenFile(n.log, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = logFile.Close() }()

	/* Started with its own stdin closed and its output to a file.

	srcds reads the console, so a backgrounded one sharing this terminal stops
	on SIGTTIN. Its output stops after the Steam banner whatever happens,
	because it takes the console over from there, which is why readiness is
	asked of rcon and never of this log. */
	//nolint:gosec // the binary is the game copy this bed was pointed at
	cmd := exec.Command(filepath.Join(n.game, "srcds_linux"),
		"-game", "tf", "-console", "-usercon", "-norestart",
		"-ip", "0", "-port", n.port, "-tickrate", "66",
		"+fps_max", "120", "+maxplayers", "32", "+map", mapName,
		"+rcon_password", password(), "+sv_setsteamaccount", "0",
		"+servercfgfile", "server.cfg")
	cmd.Dir = n.game
	cmd.Stdin = nil
	cmd.Stdout, cmd.Stderr = logFile, logFile
	cmd.Env = append(n.env(env),
		"LD_LIBRARY_PATH=.:bin:"+filepath.Join(n.game, "bin")+":"+os.Getenv("LD_LIBRARY_PATH"))
	// The core limit is raised for the child alone, so a crash leaves something
	// to symbolise without changing this process's own limits.
	cmd.SysProcAttr = coreDumping()

	if err := cmd.Start(); err != nil {
		return err
	}
	n.running = cmd
	// Reaped here so a server that exits does not sit as a zombie for the rest
	// of the run, and so Stop has something to wait on.
	go func() { _ = cmd.Wait() }()
	return nil
}

/*
env is the run's settings for entrypoint.sh and for srcds.

The same variables the compose file passes the container, so a native run and a
container run of the same mission differ in the one thing this mode exists to
vary. The names lose the TESTBED_ prefix on the way in, which is the translation
compose.yml does in YAML.
*/
func (n *nativeBed) env(from []string) []string {
	out := append([]string{}, os.Environ()...)
	out = append(out,
		"STAGE="+filepath.Join(n.root, "testbed", "build", "package"),
		"STEAMAPPDIR="+n.game,
		"STEAMAPP=tf",
		"STATS_FILE=mvmbots_stats.jsonl",
		"SRCDS_RCONPW="+password(),
	)
	for _, pair := range from {
		key, value, found := strings.Cut(pair, "=")
		if !found {
			continue
		}
		if plain, is := strings.CutPrefix(key, "TESTBED_BOT_"); is {
			out = append(out, "BOT_"+plain+"="+value)
		}
		out = append(out, pair)
	}
	return out
}

func (n *nativeBed) Stop(_ context.Context) error {
	if n.running == nil || n.running.Process == nil {
		return nil
	}
	_ = n.running.Process.Kill()
	n.running = nil
	return nil
}

func (n *nativeBed) ClearStats(context.Context) error {
	err := os.Remove(filepath.Join(n.game, "tf", statsUnderGame))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (n *nativeBed) CopyStats(_ context.Context, to string) error {
	body, err := os.ReadFile(filepath.Join(n.game, "tf", statsUnderGame)) //nolint:gosec // the game copy this bed plays
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o750); err != nil {
		return err
	}
	return os.WriteFile(to, body, 0o600)
}

/*
LogSince is the server's own log from a point in time.

srcds writes no timestamps, so the window is taken by size: what was in the file
when the attempt began is skipped, and what came after it is this attempt's. The
file is truncated at every Recreate, which is what makes the offset meaningful.
*/
func (n *nativeBed) LogSince(_ context.Context, since time.Time) string {
	if n.log == "" {
		return ""
	}
	file, err := os.Open(n.log) //nolint:gosec // the log this bed just wrote
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil || info.ModTime().Before(since) {
		return ""
	}
	body, err := os.ReadFile(n.log) //nolint:gosec // the log this bed just wrote
	if err != nil {
		return ""
	}
	return string(body)
}

func (n *nativeBed) NewestCore(context.Context) (string, string) {
	cores := coreFiles(n.game)
	if len(cores) == 0 {
		return "", ""
	}
	newest := cores[0]
	return newest, fmt.Sprintf("TESTBED_NATIVE_ROOT=%s sh %s %s",
		n.game, filepath.Join(n.root, "testbed", "symbolise-core.sh"), newest)
}

func (n *nativeBed) Space(ctx context.Context) (int64, int, int64, bool) {
	return spaceUnder(ctx, n.game, func(command string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", command)
	})
}

// coreFiles is the cores under a game tree, newest first.
func coreFiles(game string) []string {
	found, err := filepath.Glob(filepath.Join(game, "core.*"))
	if err != nil || len(found) == 0 {
		return nil
	}
	newest := func(path string) time.Time {
		info, err := os.Stat(path)
		if err != nil {
			return time.Time{}
		}
		return info.ModTime()
	}
	for i := 1; i < len(found); i++ {
		for j := i; j > 0 && newest(found[j]).After(newest(found[j-1])); j-- {
			found[j], found[j-1] = found[j-1], found[j]
		}
	}
	return found
}

/*
spaceUnder is df and the core count over one directory, run wherever the caller
says. One command rather than three, because the container round trip is the
expensive part and this runs before every run.
*/
func spaceUnder(_ context.Context, dir string, shell func(string) *exec.Cmd) (freeMB int64, cores int, weightMB int64, ok bool) {
	out, err := shell("df -Pm " + dir + " | tail -1; ls " + dir + "/core.* 2>/dev/null | wc -l; du -cm " + dir + "/core.* 2>/dev/null | tail -1").Output()
	if err != nil {
		return 0, 0, 0, false
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) < 2 {
		return 0, 0, 0, false
	}
	if fields := strings.Fields(lines[0]); len(fields) >= 4 {
		freeMB, _ = strconv.ParseInt(fields[3], 10, 64)
	}
	cores, _ = strconv.Atoi(strings.TrimSpace(lines[1]))
	if len(lines) >= 3 {
		if fields := strings.Fields(lines[2]); len(fields) >= 1 {
			weightMB, _ = strconv.ParseInt(fields[0], 10, 64)
		}
	}
	return freeMB, cores, weightMB, true
}

// shellQuote is one argument for sh -c, which is the only place this program
// builds a shell command line rather than an argv.
func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

/*
serverFor is the bed a run plays on, and the one place the two modes are chosen
between.

Everything downstream takes the interface, so a native run is refused by the
same preconditions, watched by the same watcher and written into the same
records as a container run. That is what makes a results file unambiguous about
where it came from.
*/
func serverFor(native bool, root, port string) server {
	if native {
		return &nativeBed{root: root, game: nativeRoot(), port: port}
	}
	return dockerBed{compose: filepath.Join(root, "testbed", "compose.yml"), root: root}
}

// containerOf is what the run record calls the thing it is playing on. A native
// run has no container, and saying so is better than naming one that is not there.
func containerOf(native bool) string {
	if native {
		return "srcds under " + nativeRoot()
	}
	return container()
}
