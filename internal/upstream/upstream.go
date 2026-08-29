/*
Package upstream reads the plugin repository at one pinned revision.

Every proof in this repository that compares against the plugin reads it through
here, so the revision is one fact and not one per package. The alternative was
tried: internal/tables pinned a0f9490 while internal/spgen read HEAD, and the
two answered different questions about the same file.
*/
package upstream

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

/*
Rev is the plugin revision the proofs are checked against.

Read from a git object, not from the working tree. The working tree is edited by
whoever is working in that repository, and a run caught features.sp mid-save at
21 features against the table's 22 and reported five failures that were not
real. A test that fails because somebody saved a file is a test people learn to
ignore, and these proofs are the whole argument that the port is safe to adopt.

Moving the pin is a deliberate act with a diff. Drift between it and HEAD is a
fact somebody decides about, which is what TestPinIsNotBehindHEAD reports.
*/
const Rev = "b48ad70"

/*
Dir resolves the plugin repository, or returns an error naming where it looked.

A relative MVMBOTS_UPSTREAM is read from the repository root, not from the
package under test. Resolving it here rather than in the Makefile is what stops
a wrong path turning the proofs into silent skips, which it already did.
*/
func Dir() (string, error) {
	dir := os.Getenv("MVMBOTS_UPSTREAM")
	if dir == "" {
		dir = filepath.Join("..", "..", "..", "tf2-mvm-bots")
	}
	if !filepath.IsAbs(dir) {
		if abs, err := filepath.Abs(dir); err == nil {
			if _, err := os.Stat(filepath.Join(abs, ".git")); err != nil {
				dir = filepath.Join("..", "..", dir)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return "", err
	}
	return dir, nil
}

// Read returns one file at Rev, with the path parts joined the way git wants
// them rather than the way the host spells a path.
func Read(parts ...string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	path := strings.Join(parts, "/")
	body, err := exec.Command("git", "-C", dir, "show", Rev+":"+path).Output()
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// Head is the plugin's current revision, short, for the drift report.
func Head() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
