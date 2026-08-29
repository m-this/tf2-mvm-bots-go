package body_test

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
	"github.com/m-this/tf2-mvm-bots-go/internal/spshell"
	"github.com/m-this/tf2-mvm-bots-go/internal/upstream"
)

/*
	TestTheGeneratedFilesCompileUnderTheShippedCompiler

The bodies are proved by running, under the standalone SourcePawn, with the
engine stubbed. What that cannot reach is everything SourceMod owns: DHookParam
and MRESReturn, the real GetVectorDistance, a default the plugin's own call
sites rely on. So the generated files are also compiled with the compiler and
the includes the plugin ships with, which is where a shape the standalone would
accept and SourceMod would not shows up.
*/
func TestTheGeneratedFilesCompileUnderTheShippedCompiler(t *testing.T) {
	local := spshell.ForTest(t)
	shipped, err := local.WithSourceMod(upstream.SkipOrFail(t))
	if err != nil {
		t.Skipf("no SourceMod compiler: %v", err)
	}

	generated, err := body.Generate("../..")
	if err != nil {
		t.Fatalf("generating the bodies: %v", err)
	}
	files := map[string]string{
		"roster.sp":        string(generated["sourcepawn/roster.sp"]),
		"roster_dhooks.sp": string(generated["sourcepawn/roster_dhooks.sp"]),
		"scan.sp":          string(generated["sourcepawn/scan.sp"]),
	}
	for _, smoke := range []string{"testdata/roster_smoke.sp", "testdata/scan_smoke.sp"} {
		if err := shipped.Compile(t.Context(), smoke, files); err != nil {
			t.Errorf("compiling %s: %v", smoke, err)
		}
	}
}
