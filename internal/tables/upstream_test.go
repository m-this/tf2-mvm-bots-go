package tables_test

import (
	"os"
	"path/filepath"
	"testing"
)

// upstreamDir is the plugin repository the generated SourcePawn replaces. The
// round-trip proofs read the real files out of it rather than a copy, because a
// copy is the same duplication the table exists to remove.
func upstreamDir(t *testing.T) string {
	t.Helper()

	dir := os.Getenv("MVMBOTS_UPSTREAM")
	if dir == "" {
		dir = filepath.Join("..", "..", "..", "tf2-mvm-bots")
	}

	if _, err := os.Stat(dir); err != nil {
		t.Skipf("upstream plugin not at %s, set MVMBOTS_UPSTREAM: %v", dir, err)
	}
	return dir
}

func readUpstream(t *testing.T, parts ...string) string {
	t.Helper()

	path := filepath.Join(append([]string{upstreamDir(t)}, parts...)...)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(body)
}
