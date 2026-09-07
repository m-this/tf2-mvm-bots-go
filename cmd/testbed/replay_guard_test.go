package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

/*
A replayed server.cfg cannot land where git would commit it.

A player's cfg was copied into the plugin tree so a run could name it, a later
`git add -A testbed/` swept it in, and his rcon_password went to a public
repository. Removing it from the tip did not remove it from the history.
*/
func TestReplayRefusesAPathGitWouldCommit(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init")

	tracked := filepath.Join(dir, "server.cfg")
	write(t, tracked, "sm_redbots_manager_mode 2\nrcon_password hunter2\n")

	if _, err := readServerCfg(tracked); err == nil {
		t.Fatal("a cfg inside a git working tree was accepted")
	} else if !strings.Contains(err.Error(), "ignored") {
		t.Fatalf("the refusal does not say why: %v", err)
	}
}

func TestReplayTakesAnIgnoredPath(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init")
	write(t, filepath.Join(dir, ".gitignore"), "replays/\n")

	if err := os.Mkdir(filepath.Join(dir, "replays"), 0o750); err != nil {
		t.Fatal(err)
	}
	ignored := filepath.Join(dir, "replays", "somebody.cfg")
	write(t, ignored, "sm_redbots_manager_mode 2\nrcon_password hunter2\n")

	got, err := readServerCfg(ignored)
	if err != nil {
		t.Fatalf("an ignored cfg was refused: %v", err)
	}
	if got["TESTBED_BOT_MANAGER_MODE"] != "2" {
		t.Fatalf("read %v, want the manager mode", got)
	}
}

// The credentials in the file are not read, whatever else is. The whitelist is
// the guard; this is the assertion that it stays one.
func TestCredentialsAreNotCarried(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "server.cfg")
	write(t, cfg, "sm_redbots_manager_mode 2\nrcon_password hunter2\nsv_password letmein\nsv_setsteamaccount ABCDEF\n")

	got, err := readServerCfg(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range got {
		for _, secret := range []string{"hunter2", "letmein", "ABCDEF"} {
			if strings.Contains(value, secret) {
				t.Fatalf("%s carries %q out of the player's cfg", key, secret)
			}
		}
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("git is not usable here: %v\n%s", err, out)
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
