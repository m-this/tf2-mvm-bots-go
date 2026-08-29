package upstream

import (
	"os"
	"testing"
)

/*
	SkipOrFail is how a test reacts to an unreachable plugin repository

A developer without the sibling checkout gets a skip that names what is
missing. The gate sets RequireEnv and gets a failure, because three packages
resolved the path themselves, got it wrong, and skipped in silence: bindgen and
navmesh each looked green in under a second while running none of their proofs.
*/
func SkipOrFail(t testing.TB) string {
	t.Helper()

	dir, err := Dir()
	if err == nil {
		return dir
	}
	if os.Getenv(RequireEnv) == "" {
		t.Skipf("no plugin repository, set MVMBOTS_UPSTREAM: %v", err)
	}
	t.Fatalf("no plugin repository: %v (%s is set, so this is a failure and not a skip)", err, RequireEnv)
	return ""
}

// RequireEnv names the variable that turns an unreachable plugin repository
// from a skip into a failure. make check sets it.
const RequireEnv = "MVMBOTS_REQUIRE_UPSTREAM"
