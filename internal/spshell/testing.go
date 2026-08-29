package spshell

import (
	"errors"
	"os"
	"testing"
)

// RequireEnv names the variable that turns an absent toolchain from a skip into
// a failure. make check sets it, so the gate cannot pass by running nothing.
const RequireEnv = "MVMBOTS_REQUIRE_SPSHELL"

/*
	ForTest is the toolchain every differential test starts from

A developer with no clang and no network gets a skip that names what is missing.
The gate gets a failure, because a differential test that silently does not run
is the same as not having one.

testing.TB rather than *testing.T, because the float literal fuzz needs it too.
*/
func ForTest(t testing.TB) Toolchain {
	t.Helper()
	tc, err := ToolchainFromEnv()
	if err == nil {
		return tc
	}
	if errors.Is(err, ErrNoToolchain) && os.Getenv(RequireEnv) == "" {
		t.Skipf("no standalone SourcePawn toolchain, run make toolchain: %v", err)
	}
	t.Fatalf("%v (%s is set, so this is a failure and not a skip)", err, RequireEnv)
	return Toolchain{}
}
