package body_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
	"github.com/m-this/tf2-mvm-bots-go/internal/body/scan"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
	"github.com/m-this/tf2-mvm-bots-go/internal/spshell"
)

/*
	TestPortedScansAgreeWithTheGoTheyCameFrom

util.sp's client loop, in the form it was ported into. The proof is the one the
bodies get: the same canned world on both sides, and the answer and the sequence
of engine calls both have to match, so a loop that skipped a slot the shipped
one visits fails here even when the count comes out right.
*/
func TestPortedScansAgreeWithTheGoTheyCameFrom(t *testing.T) {
	tc := spshell.ForTest(t)
	w := cannedWorld()

	generated, err := body.Generate("../..")
	if err != nil {
		t.Fatalf("generating the bodies: %v", err)
	}
	cells, err := tc.Run(t.Context(), "testdata/scan_probe.sp", map[string]string{
		"scan.sp":        string(generated["sourcepawn/scan.sp"]),
		"scan_world.inc": worldInclude(w) + scanStubs(w),
	})
	if err != nil {
		t.Fatalf("running the scan probe under spshell: %v", err)
	}

	want := goScanCells(w)
	compareCells(t, want, cells)
}

// goScanCells runs the Go in the order the probe runs it.
func goScanCells(w world) []int32 {
	in := &installed{w: w}
	defer engine.Install(in.calls())()

	var out []int32
	emit := func(result int32) {
		out = append(out, result, int32(len(in.trace))) //nolint:gosec // G115: record bounds the trace at traceCells
		out = append(out, in.trace...)
		in.trace = in.trace[:0]
	}
	for client := int32(1); client <= worldSlots; client++ {
		emit(scan.NearestEnemyCount(client, 200.0, false))
		emit(scan.NearestEnemyCount(client, 200.0, true))
		emit(scan.NearestEnemyCount(client, 100000.0, false))
		// The SourcePawn caller omits this one, so the default is what
		// the generated declaration says it is.
		emit(scan.NearestEnemyCount(client, 100000.0, false))
		emit(scan.NearestEnemyCount(client, 5934.8125, false))
	}
	return out
}

// scanStubs is the rest of the engine the scan reaches, on top of the world the
// bodies share. WorldSpaceCenter returns its array, which is the plugin's own
// shape and the one spcomp will not let a caller assign to a sized local.
func scanStubs(w world) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\nint MaxClients = WORLD_SLOTS;\n\n")
	writeBools(&b, "gBuster", w.buster[:])
	writeBools(&b, "gUber", w.uber[:])
	writeBools(&b, "gStealth", w.stealth[:])
	writeBools(&b, "gExposed", w.exposed[:])
	writeVectors(&b, "gCentre", w.centre[:])
	fmt.Fprintf(&b, "\nstock bool IsSentryBusterRobot(int client) { Trace(%d, client); return gBuster[client]; }\n", traceIsSentryBusterRobot)
	fmt.Fprintf(&b, "stock bool TF2_IsInvulnerable(int client) { Trace(%d, client); return gUber[client]; }\n", traceIsInvulnerable)
	fmt.Fprintf(&b, "stock bool TF2_IsStealthed(int client) { Trace(%d, client); return gStealth[client]; }\n", traceIsStealthed)
	fmt.Fprintf(&b, "stock bool IsCloakedPlayerExposed(int client) { Trace(%d, client); return gExposed[client]; }\n", traceIsCloakedPlayerExposed)
	fmt.Fprintf(&b, `
stock float[] WorldSpaceCenter(int entity)
{
	Trace(%d, entity);

	float centre[3];
	for (int axis = 0; axis < 3; axis++)
		centre[axis] = gCentre[entity][axis];

	return centre;
}

stock float GetVectorDistance(const float a[3], const float b[3])
{
	Trace(%d, 0);

	/* The square, not the distance. The standalone SourcePawn has no square
	   root, and what this has to be is the same function on both sides:
	   what is under test is the translation, not SourceMod's arithmetic. */
	float sum = 0.0;
	for (int axis = 0; axis < 3; axis++)
	{
		float d = a[axis] - b[axis];
		sum += d * d;
	}
	return sum;
}
`, traceWorldSpaceCenter, traceVectorDistance)
	return b.String()
}
