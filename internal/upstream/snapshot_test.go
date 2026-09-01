package upstream_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/upstream"
)

/*
	TestSnapshotMatchesTheRepository is what makes the snapshot evidence

A copy of somebody else's file proves nothing on its own: it proves what it says
only if it is what they wrote. So wherever the plugin repository is present, every
snapshotted file is read out of its git history and compared with the copy here.

It skips when the repository is not there, which is the state the snapshot exists
to make survivable, and it is not a hole: the day the repository is archived this
test stops running and the snapshot stops changing, because tools/snapshot.sh is
the only thing that writes it and it needs the repository too.
*/
func TestSnapshotMatchesTheRepository(t *testing.T) {
	dir, err := upstream.Dir()
	if err != nil {
		t.Skipf("no plugin repository: %v", err)
	}

	files, err := upstream.SnapshotFiles()
	if err != nil {
		t.Fatalf("walking the snapshot: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("the snapshot is empty, so the proofs that read it prove nothing")
	}

	for _, f := range files {
		t.Run(f.Rev+"/"+f.Path, func(t *testing.T) {
			out, err := exec.Command("git", "-C", dir, "show", f.Rev+":"+f.Path).Output()
			if err != nil {
				t.Fatalf("reading %s at %s from %s: %v", f.Path, f.Rev, filepath.Base(dir), err)
			}
			if string(out) != f.Body {
				t.Errorf("the snapshot of %s at %s is not what the repository holds; run tools/snapshot.sh", f.Path, f.Rev)
			}
		})
	}
}

// TestEverySnapshotIsRead catches a file left in the snapshot after the port
// that read it stopped needing it, which is 700 KiB nobody would notice.
func TestEverySnapshotIsRead(t *testing.T) {
	files, err := upstream.SnapshotFiles()
	if err != nil {
		t.Fatalf("walking the snapshot: %v", err)
	}

	for _, f := range files {
		if !strings.HasSuffix(f.Path, ".sp") {
			t.Errorf("%s is not SourcePawn; the snapshot holds shipped plugin files and nothing else", f.Path)
		}
	}
}
