package tables_test

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/upstream"
)

// readUpstream is the pinned plugin file these proofs read. The pin, and why
// there is one, is internal/upstream. No plugin repository is a skip; a plugin
// repository missing the file at the pin is a failure.
func readUpstream(t *testing.T, parts ...string) string {
	t.Helper()

	if _, err := upstream.Dir(); err != nil {
		t.Skipf("no plugin repository, set MVMBOTS_UPSTREAM: %v", err)
	}
	body, err := upstream.Read(parts...)
	if err != nil {
		t.Fatalf("reading %v at %s: %v", parts, upstream.Rev, err)
	}
	return body
}

// TestPinIsNotBehindHEAD says when the pin has fallen behind, without failing:
// the plugin moving on is normal, and only the proofs going stale is a problem.
func TestPinIsNotBehindHEAD(t *testing.T) {
	head, err := upstream.Head()
	if err != nil {
		t.Skipf("no upstream HEAD: %v", err)
	}
	if head != upstream.Rev {
		t.Logf("pinned at %s, upstream HEAD is %s: move the pin and re-read the diff", upstream.Rev, head)
	}
}
