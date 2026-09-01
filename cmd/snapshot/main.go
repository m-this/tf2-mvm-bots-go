/*
Command snapshot writes internal/upstream/shipped from the plugin repository.

One file per Body with a Shipped path, at the revision that Body reads it at:
the global pin, or its own when it names one. See tools/snapshot.sh for why the
snapshot exists at all.
*/
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
	"github.com/m-this/tf2-mvm-bots-go/internal/upstream"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "snapshot:", err)
		os.Exit(1)
	}
}

func run() error {
	dir, err := upstream.Dir()
	if err != nil {
		return fmt.Errorf("the plugin repository is what the snapshot is taken from: %w", err)
	}

	root := filepath.Join("internal", "upstream", "shipped")
	if err := os.RemoveAll(root); err != nil {
		return err
	}

	// A file pinned at two revisions would be two snapshots, which is fine and
	// worth saying out loud, so this counts what it wrote rather than what it
	// was asked for.
	written := map[string]bool{}

	for _, b := range append(append([]body.Body{}, body.All...), body.Actions...) {
		if b.Shipped == "" {
			continue
		}

		rev := b.Rev
		if rev == "" {
			rev = upstream.Rev
		}

		key := rev + ":" + b.Shipped
		if written[key] {
			continue
		}

		out, err := exec.Command("git", "-C", dir, "show", key).Output()
		if err != nil {
			return fmt.Errorf("reading %s: %w", key, err)
		}

		path := filepath.Join(root, rev, filepath.FromSlash(b.Shipped))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(path, out, 0o600); err != nil {
			return err
		}

		written[key] = true
	}

	fmt.Printf("snapshot: %d files\n", len(written))
	return nil
}
