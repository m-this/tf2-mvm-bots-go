package tables_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

/*
	The plugin revision these proofs are checked against

Read from a git object, not from the working tree. The working tree is edited by
whoever is working in that repository, and a run caught features.sp mid-save at
21 features against the table's 22 and reported five failures that were not
real. A test that fails because somebody saved a file is a test people learn to
ignore, and these are the whole argument that the tables are safe to adopt.

Moving the pin is a deliberate act with a diff. Drift between it and HEAD is a
fact somebody decides about, which is what TestPinIsNotBehindHEAD reports.
*/
const upstreamRev = "a0f9490"

func upstreamDir(t *testing.T) string {
	t.Helper()

	dir := os.Getenv("MVMBOTS_UPSTREAM")
	if dir == "" {
		dir = filepath.Join("..", "..", "..", "tf2-mvm-bots")
	}
	// A relative MVMBOTS_UPSTREAM is read from the repository root, not from
	// this package. Resolving it here rather than in the Makefile is what stops
	// a wrong path turning the proofs into silent skips, which it already did.
	if !filepath.IsAbs(dir) {
		if abs, err := filepath.Abs(dir); err == nil {
			if _, err := os.Stat(filepath.Join(abs, ".git")); err != nil {
				dir = filepath.Join("..", "..", dir)
			}
		}
	}

	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Skipf("upstream plugin not a git repository at %s, set MVMBOTS_UPSTREAM: %v", dir, err)
	}
	return dir
}

func readUpstream(t *testing.T, parts ...string) string {
	t.Helper()

	path := strings.Join(parts, "/")
	cmd := exec.Command("git", "-C", upstreamDir(t), "show", upstreamRev+":"+path)
	body, err := cmd.Output()
	if err != nil {
		t.Fatalf("git show %s:%s: %v", upstreamRev, path, err)
	}
	return string(body)
}

// TestPinIsNotBehindHEAD says when the pin has fallen behind, without failing:
// the plugin moving on is normal, and only the proofs going stale is a problem.
func TestPinIsNotBehindHEAD(t *testing.T) {
	dir := upstreamDir(t)

	head, err := exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		t.Skipf("no HEAD in %s: %v", dir, err)
	}
	if got := strings.TrimSpace(string(head)); got != upstreamRev {
		t.Logf("pinned at %s, upstream HEAD is %s: move the pin and re-read the diff", upstreamRev, got)
	}
}
