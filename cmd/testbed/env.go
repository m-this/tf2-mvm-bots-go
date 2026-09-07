package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/m-this/tf2-mvm-bots-go/internal/plugin"
)

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

/*
bedName is which test-bed this runner drives.

One machine can run two: each is a compose project with a port of its own, and
the second builds its tree over the first's game volume rather than downloading
the game again. TESTBED_PROJECT names the bed. A bed that is not the first must
also be given TESTBED_PORT, because two servers on 27025 is one server and a
runner reading the wrong one.
*/
const firstBed = "mvmbots-testbed"

func bedName() string { return envOr("TESTBED_PROJECT", firstBed) }

func port() (string, error) {
	p := os.Getenv("TESTBED_PORT")
	if p == "" && bedName() != firstBed {
		return "", fmt.Errorf("TESTBED_PROJECT=%s needs a TESTBED_PORT of its own; the first bed has 27025", bedName())
	}
	if p == "" {
		return "27025", nil
	}
	return p, nil
}

func address(port string) string { return "127.0.0.1:" + port }

func password() string { return envOr("TESTBED_RCONPW", "testbed") }

func container() string { return envOr("TESTBED_CONTAINER", bedName()+"-srcds-1") }

/*
	repoRoot is the plugin tree, not this repository's root

The test-bed lives here and everything it runs lives there: build.sh, the
compose file, the popfiles, the map configs and the results. It used to find its
own working tree by walking up to a go.mod, which was the same directory; since
mvm-x2c it is not, so the path comes from internal/plugin like every other
reader of that tree.
*/
func repoRoot() (string, error) {
	return plugin.Dir()
}

func compile(ctx context.Context, root string) error {
	cmd := exec.CommandContext(ctx, "sh", filepath.Join(root, "testbed", "build.sh"))
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("the build failed:\n%s", tail(string(out), 20))
	}
	return nil
}

func tail(s string, lines int) string {
	all := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(all) > lines {
		all = all[len(all)-lines:]
	}
	return strings.Join(all, "\n")
}

/*
hold takes the test-bed, and says who has it rather than waiting.

Two runners is not a slow run, it is two runs measuring each other's map
changes. That happened three times in one session before this existed, and each
time the results looked ordinary.

The refusal used to be a bare pid, which was a dead end ten times over: it says
nothing about what is being played, which bed it is, or when it will be free,
and nothing about whether the process is still there. What that invited was
worse than the wait. A session that could not read the lock reached for a
compose command by hand, and compose.yml defaults its project name to the first
bed, so a line typed without TESTBED_PROJECT recreates somebody else's server.
So the refusal reads the run record and names the run, the map, the arm and the
deadline, and points at the other beds. See mvm-d84.
*/
func hold(path string) (func(), error) {
	// 0o600: the lock is this developer's own, and the run record beside it is
	// what anybody else reads.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("%s is already in use.\n%s", bedName(), heldBy())
	}
	if err := file.Truncate(0); err == nil {
		// Best effort: the lock is the flock. The pid is here so a reader with
		// no run record still has something to look at.
		_, _ = fmt.Fprintf(file, "pid %d", os.Getpid())
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
		_ = os.Remove(path)
	}, nil
}

/*
heldBy is what the runner holding the bed is doing, and what to do instead.

A flock the kernel has already dropped cannot be the one refusing here, so the
holder is alive whatever its record says; a record that looks stale next to a
live lock is a runner that has not written a poll yet.
*/
func heldBy() string {
	var b strings.Builder
	s, found, err := readState(bedName())
	switch {
	case err != nil || !found:
		b.WriteString("  there is no run record for it, so it was started by something older than mvm-c4a\n")
	default:
		b.WriteString(describe(s))
	}
	b.WriteString("  a second bed is TESTBED_PROJECT with a TESTBED_PORT of its own; `testbed -bed list` says which exist\n")
	b.WriteString("  do not run compose against this tree by hand: it defaults to " + firstBed + " and would recreate the server above\n")
	return b.String()
}
