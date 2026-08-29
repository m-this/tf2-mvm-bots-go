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
		"scan_world.inc": worldInclude(w) + scanStubs(w) + entityInclude(cannedEntities()),
	})
	if err != nil {
		t.Fatalf("running the scan probe under spshell: %v", err)
	}

	want := goScanCells(w)
	compareCells(t, want, cells)
}

// goScanCells runs the Go in the order the probe runs it.
func goScanCells(w world) []int32 {
	in := &installed{w: w, entities: cannedEntities()}
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

		// FindEnemyNearestToMe, over the filters its callers switch on.
		emit(scan.EnemyNearestToMe(client, 900000.0, false, false, false, engine.ClassUnknown()))
		emit(scan.EnemyNearestToMe(client, 900000.0, true, false, false, engine.ClassUnknown()))
		emit(scan.EnemyNearestToMe(client, 900000.0, false, true, false, engine.ClassUnknown()))
		emit(scan.EnemyNearestToMe(client, 900000.0, false, false, true, engine.ClassUnknown()))
		emit(scan.EnemyNearestToMe(client, 900000.0, false, false, false, engine.Class(2)))
		emit(scan.EnemyNearestToMe(client, 5934.8125, false, false, false, engine.ClassUnknown()))

		// The two building scans, and the spy's four passes over them.
		emit(scan.NearestSappableObject(client, 1000.0))
		emit(scan.NearestSappableObject(client, 999999.0))
		emit(scan.NearestEnemyTeleporter(client, 999999.0))
		emit(scan.NearestEnemyTeleporter(client, 1000.0))
		emit(scan.BestTargetForSpy(client, 900000.0))

		// The spy's four, over the two filters the callers vary and the
		// speed check that is off unless a caller asks for it.
		var here [3]float32
		here[0] = 200.0
		emit(scan.NearestSappablePlayer(client, 900000.0, false, engine.ClassUnknown(), 0.0))
		emit(scan.NearestSappablePlayer(client, 900000.0, true, engine.ClassUnknown(), 0.0))
		emit(scan.NearestSappablePlayer(client, 900000.0, false, engine.Class(2), 340.0))
		emit(scan.FarthestSappablePlayer(client, 900000.0, false, engine.ClassUnknown(), 0.0))
		emit(scan.FarthestSappablePlayer(client, 900000.0, false, engine.ClassUnknown(), 340.0))
		emit(scan.NearestSappablePlayerHealingSomeone(client, 900000.0, false, engine.ClassUnknown(), 0.0))
		emit(scan.NearestSappablePlayerHealingSomeone(client, 900000.0, true, engine.ClassUnknown(), 340.0))
		emit(scan.EnemyPlayerNearestToPosition(client, here, 900000.0))
		emit(scan.EnemyPlayerNearestToPosition(client, here, 100.0))
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
	writeBools(&b, "gGiant", w.giant[:])
	writeBools(&b, "gDazed", w.dazed[:])
	writeInts(&b, "gClass", classCells(w))
	writeInts(&b, "gTfTeam", teamCells(w))
	writeBools(&b, "gSapped", w.sapped[:])
	writeBools(&b, "gBonked", w.bonked[:])
	writeBools(&b, "gMedigun", w.medigun[:])
	writeBools(&b, "gHealing", w.healing[:])
	writeInts(&b, "gWeapon", w.weapon[:])
	writeFloats(&b, "gMaxSpeed", w.maxSpeed[:])
	b.WriteString(`
/* The three SourceMod tags the ported signature keeps. Declared here because
   SourceMod's own includes are not in the standalone SourcePawn, with only the
   constants the scans name. */
enum TFClassType
{
	TFClass_Unknown = 0,
	TFClass_Engineer = 9
};

enum TFTeam
{
	TFTeam_Unassigned = 0
};

enum TFCond
{
	TFCond_Dazed = 17,
	TFCond_Sapped = 15,
	TFCond_Bonked = 16
};

enum PropType
{
	Prop_Send = 1
};

enum TFWeaponType
{
	TF_WEAPON_NONE = 0,
	TF_WEAPON_MEDIGUN = 29
};

`)
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
	fmt.Fprintf(&b, "stock bool TF2_IsMiniBoss(int client) { Trace(%d, client); return gGiant[client]; }\n", traceIsMiniBoss)
	fmt.Fprintf(&b, `
stock bool TF2_IsPlayerInCondition(int client, TFCond condition)
{
	Trace(%d, client);

	switch (condition)
	{
		case TFCond_Dazed:  { return gDazed[client]; }
		case TFCond_Sapped: { return gSapped[client]; }
		case TFCond_Bonked: { return gBonked[client]; }
	}
	return false;
}
`, traceIsPlayerInCondition)
	fmt.Fprintf(&b, "stock TFClassType TF2_GetPlayerClass(int client) { Trace(%d, client); return view_as<TFClassType>(gClass[client]); }\n", tracePlayerClass)
	fmt.Fprintf(&b, "stock TFTeam TF2_GetClientTeam(int client) { Trace(%d, client); return view_as<TFTeam>(gTfTeam[client]); }\n", tracePlayerTeam)
	fmt.Fprintf(&b, "stock float GetEntPropFloat(int entity, PropType propType, const char[] prop) { Trace(%d, entity); return gMaxSpeed[entity]; }\n", traceEntPropFloat)
	fmt.Fprintf(&b, "stock int GetEntPropEnt(int weapon, PropType propType, const char[] prop) { Trace(%d, weapon); return gHealing[weapon - 200] ? 1 : -1; }\n", traceEntPropEnt)
	fmt.Fprintf(&b, "stock int BaseCombatCharacter_GetActiveWeapon(int client) { Trace(%d, client); return gWeapon[client]; }\n", traceActiveWeapon)
	fmt.Fprintf(&b, "stock TFWeaponType TF2Util_GetWeaponID(int weapon) { Trace(%d, weapon); return gMedigun[weapon - 200] ? TF_WEAPON_MEDIGUN : TF_WEAPON_NONE; }\n", traceWeaponID)
	fmt.Fprintf(&b, "stock TFTeam GetPlayerEnemyTeam(int client) { Trace(%d, client); return gTfTeam[client] == 2 ? view_as<TFTeam>(3) : view_as<TFTeam>(2); }\n", tracePlayerEnemyTeam)
	return b.String()
}

func classCells(w world) []int32 {
	out := make([]int32, len(w.class))
	for i, c := range w.class {
		out[i] = int32(c)
	}
	return out
}

func teamCells(w world) []int32 {
	out := make([]int32, len(w.tfTeam))
	for i, t := range w.tfTeam {
		out[i] = int32(t)
	}
	return out
}
