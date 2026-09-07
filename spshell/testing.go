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
	return ForTestRequiring(t, RequireEnv)
}

/*
	ForTestRequiring is ForTest with the gate's variable named by the caller

This package is imported by more than one repository, and each has its own gate
with its own variable. A repository that set MVMBOTS_REQUIRE_SPSHELL to mean its
own gate would be naming this one, and the two would drift the first time either
Makefile changed. So the caller says which variable is theirs.
*/
func ForTestRequiring(t testing.TB, requireEnv string) Toolchain {
	t.Helper()
	tc, err := ToolchainFromEnv()
	if err == nil {
		return tc
	}
	if errors.Is(err, ErrNoToolchain) && os.Getenv(requireEnv) == "" {
		t.Skipf("no standalone SourcePawn toolchain, run make toolchain: %v", err)
	}
	t.Fatalf("%v (%s is set, so this is a failure and not a skip)", err, requireEnv)
	return Toolchain{}
}
