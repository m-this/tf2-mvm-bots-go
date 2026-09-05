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
bed is which test-bed this runner drives.

One machine can run two: each is a compose project with a port of its own, and
the second builds its tree over the first's game volume rather than downloading
the game again. TESTBED_PROJECT names the bed. A bed that is not the first must
also be given TESTBED_PORT, because two servers on 27025 is one server and a
runner reading the wrong one.
*/
const firstBed = "mvmbots-testbed"

func bed() string { return envOr("TESTBED_PROJECT", firstBed) }

func port() (string, error) {
	p := os.Getenv("TESTBED_PORT")
	if p == "" && bed() != firstBed {
		return "", fmt.Errorf("TESTBED_PROJECT=%s needs a TESTBED_PORT of its own; the first bed has 27025", bed())
	}
	if p == "" {
		return "27025", nil
	}
	return p, nil
}

func address(port string) string { return "127.0.0.1:" + port }

func password() string { return envOr("TESTBED_RCONPW", "testbed") }

func container() string { return envOr("TESTBED_CONTAINER", bed()+"-srcds-1") }

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
*/
func hold(path string) (func(), error) {
	// 0o600: the lock holds a pid so the message can name who has the bed,
	// and nobody else needs to read it.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		held, _ := os.ReadFile(path)
		_ = file.Close()
		return nil, fmt.Errorf("the test-bed is already in use by %s", strings.TrimSpace(string(held)))
	}
	if err := file.Truncate(0); err == nil {
		// Best effort: the lock is the flock, and the pid in the file is
		// only there so the message names who has it.
		_, _ = fmt.Fprintf(file, "pid %d", os.Getpid())
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
		_ = os.Remove(path)
	}, nil
}
