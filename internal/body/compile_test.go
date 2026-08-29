package body_test

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
	"github.com/m-this/tf2-mvm-bots-go/internal/spshell"
	"github.com/m-this/tf2-mvm-bots-go/internal/upstream"
)

/*
	TestGeneratedDHooksCompileUnderTheShippedCompiler

The bodies are proved by running. The DHook callbacks around them are proved by
compiling, and it has to be the compiler the plugin ships with and the DHooks
include the plugin ships with: a callback shape the standalone SourcePawn would
accept and SourceMod would not is exactly the failure this covers.
*/
func TestGeneratedDHooksCompileUnderTheShippedCompiler(t *testing.T) {
	local := spshell.ForTest(t)
	shipped, err := local.WithSourceMod(upstream.SkipOrFail(t))
	if err != nil {
		t.Skipf("no SourceMod compiler: %v", err)
	}

	generated, err := body.Generate("../..")
	if err != nil {
		t.Fatalf("generating the bodies: %v", err)
	}
	if err := shipped.Compile(t.Context(), "testdata/roster_smoke.sp", map[string]string{
		"roster.sp":        string(generated["sourcepawn/roster.sp"]),
		"roster_dhooks.sp": string(generated["sourcepawn/roster_dhooks.sp"]),
	}); err != nil {
		t.Fatalf("compiling the generated DHook callbacks: %v", err)
	}
}
