/*
Package upstream reads the plugin repository at one pinned revision.

Every proof in this repository that compares against the plugin reads it through
here, so the revision is one fact and not one per package. The alternative was
tried: internal/tables pinned a0f9490 while internal/spgen read HEAD, and the
two answered different questions about the same file.
*/
package upstream

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
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
	root, err := repoRoot()
	if err != nil {
		return "", err
	}
	dir := os.Getenv(DirEnv)
	if dir == "" {
		// The archived plugin repository, wanted by nothing but the
		// snapshot: everything else reads plugin/ in this repository.
		dir = filepath.Join(root, "..", "tf2-mvm-bots")
	} else if !filepath.IsAbs(dir) {
		dir = filepath.Join(root, dir)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return "", fmt.Errorf("no plugin repository at %s: %w", dir, err)
	}
	return dir, nil
}

// DirEnv names the variable that points at the plugin repository.
const DirEnv = "MVMBOTS_UPSTREAM"

/*
	repoRoot is this repository, found by its go.mod

It used to be counted in directories, "..", "..", "..", which is right for a
test running in internal/something and wrong for a command running at the root.
cmd/testbed is that command, and after mvm-x2c it is the biggest reader of the
plugin tree, so the depth is worked out rather than assumed.
*/
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("not inside the repository: no go.mod above the working directory")
		}
		dir = parent
	}
}

// Read returns one file at Rev, with the path parts joined the way git wants
// them rather than the way the host spells a path.
func Read(parts ...string) (string, error) {
	return ReadAt("", parts...)
}

/*
ReadAt is Read at a revision of the caller's choosing, and at the pin when that
is empty.

It reads the snapshot under shipped/ first. That snapshot is the whole reason
the plugin repository can be archived: the proofs are what tie this repository
to it, and a proof that needs another repository's git history is a proof that
dies with that repository. The snapshot is 700 KiB of text and it is the
evidence, so it is versioned here beside what it proves.

The git path is still taken when the snapshot has no answer, and
TestSnapshotMatchesTheRepository checks the two agree wherever the repository is
present.
*/
func ReadAt(rev string, parts ...string) (string, error) {
	if rev == "" {
		rev = Rev
	}
	path := strings.Join(parts, "/")

	if body, err := snapshot.ReadFile("shipped/" + rev + "/" + path); err == nil {
		return string(body), nil
	}

	dir, err := Dir()
	if err != nil {
		return "", err
	}
	body, err := exec.Command("git", "-C", dir, "show", rev+":"+path).Output()
	if err != nil {
		return "", err
	}
	return string(body), nil
}

/*
snapshot is every shipped file a proof reads, at the revision it reads it.

Written by tools/snapshot.sh from the plugin repository. Adding a Body with a
Shipped path means adding its file here, which the snapshot test says out loud
when it has not been done.
*/
//go:embed shipped
var snapshot embed.FS

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

// SnapshotFile is one file the snapshot holds, and the revision it was taken
// at.
type SnapshotFile struct {
	Rev  string
	Path string
	Body string
}

// SnapshotFiles is everything under shipped, so a test can walk it without
// knowing what a Body asked for.
func SnapshotFiles() ([]SnapshotFile, error) {
	var out []SnapshotFile

	err := fs.WalkDir(snapshot, "shipped", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rest, _ := strings.CutPrefix(name, "shipped/")
		rev, path, found := strings.Cut(rest, "/")
		if !found {
			return fmt.Errorf("%s is not under a revision directory", name)
		}
		body, err := snapshot.ReadFile(name)
		if err != nil {
			return err
		}
		out = append(out, SnapshotFile{Rev: rev, Path: path, Body: string(body)})
		return nil
	})

	return out, err
}
