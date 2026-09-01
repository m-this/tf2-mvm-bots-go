package tables_test

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/upstream"
)

/*
	readUpstream is the plugin file these proofs read

The shipped text, from the snapshot under internal/upstream rather than from the
plugin tree: what these check is what the plugin said, and the port deletes that
from the tree as it goes. Reading the tree would make a proof pass by having
nothing left to disagree with.
*/
func readUpstream(t *testing.T, parts ...string) string {
	t.Helper()

	body, err := upstream.Read(parts...)
	if err != nil {
		t.Fatalf("reading %v: %v", parts, err)
	}
	return body
}
