/*
Package plugin resolves the SourcePawn tree this repository owns.

It used to live in another repository, which is what made that repository
impossible to archive: half of this one asked it where the plugin was. The tree
is under plugin/ now and nothing reads the old repository.
*/
package plugin

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

/*
Dir is the plugin tree, found by this repository's go.mod.

Counted from the module root rather than in "..", because a test runs in
internal/something and a command runs at the root, and the same count cannot be
right for both.
*/
func Dir() (string, error) {
	root, err := repoRoot()
	if err != nil {
		return "", err
	}

	dir := os.Getenv(DirEnv)
	if dir == "" {
		dir = filepath.Join(root, "plugin")
	} else if !filepath.IsAbs(dir) {
		dir = filepath.Join(root, dir)
	}

	if _, err := os.Stat(filepath.Join(dir, "source")); err != nil {
		return "", fmt.Errorf("no plugin tree at %s: %w", dir, err)
	}

	return dir, nil
}

// DirEnv names the variable that points somewhere else, which the test-bed uses
// to build a tree it has staged.
const DirEnv = "MVMBOTS_PLUGIN"

// Path is one file inside the tree.
func Path(parts ...string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{dir}, parts...)...), nil
}

// Read is one file's contents.
func Read(parts ...string) ([]byte, error) {
	path, err := Path(parts...)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path) //nolint:gosec // the path is this repository's own tree
}

// repoRoot walks up for the go.mod, which is the one landmark that does not
// move with the caller.
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
			return "", errors.New("no go.mod above the working directory, so the repository root is unknown")
		}
		dir = parent
	}
}

/*
SkipOrFail is how a test reacts to a missing plugin tree.

It should never be missing now that it is in this repository, which is the
point: what used to be a skip because somebody had not cloned the other
repository is a broken checkout, and RequireEnv makes make check say so.
*/
func SkipOrFail(t testing.TB) string {
	t.Helper()

	dir, err := Dir()
	if err == nil {
		return dir
	}
	if os.Getenv(RequireEnv) == "" {
		t.Skipf("no plugin tree: %v", err)
	}
	t.Fatalf("no plugin tree: %v (%s is set, so this is a failure and not a skip)", err, RequireEnv)
	return ""
}

// RequireEnv turns a missing plugin tree from a skip into a failure. make check
// sets it.
const RequireEnv = "MVMBOTS_REQUIRE_PLUGIN"

// ReadPath is Read taking the path in pieces.
func ReadPath(parts ...string) (string, error) {
	body, err := Read(parts...)
	return string(body), err
}
