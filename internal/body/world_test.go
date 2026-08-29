package body_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
	"github.com/m-this/tf2-mvm-bots-go/internal/sp"
)

// worldSlots is how many client slots the canned world has. Small on purpose:
// the proof here is that two translations of one function agree, and a wider
// sweep would say the same thing more slowly.
const worldSlots = 12

// The trace ids. One per engine call, and the emitted trace is the sequence of
// them with the argument that was asked about, so a body that answered right by
// asking the wrong questions still fails.
const (
	traceIsClientInGame = iota + 1
	traceIsPlayerAlive
	traceGetClientTeam
	traceHasAmmo
	traceClip1
	traceOrigin
	traceVectorDistance
	traceIsSentryBusterRobot
	traceIsInvulnerable
	traceIsStealthed
	traceIsCloakedPlayerExposed
	traceWorldSpaceCenter
	traceIsMiniBoss
	traceIsPlayerInCondition
	tracePlayerClass
	tracePlayerTeam
	tracePlayerEnemyTeam
)

// world is what the stubs answer, indexed by slot. Slot 0 is never a client and
// is there so the arrays index the way SourcePawn's do.
type world struct {
	inGame   [worldSlots + 1]bool
	alive    [worldSlots + 1]bool
	defender [worldSlots + 1]bool
	team     [worldSlots + 1]int32
	hasAmmo  [worldSlots + 1]bool
	clip     [worldSlots + 1]int32
	origin   [worldSlots + 1][3]float32
	buster   [worldSlots + 1]bool
	uber     [worldSlots + 1]bool
	stealth  [worldSlots + 1]bool
	exposed  [worldSlots + 1]bool
	centre   [worldSlots + 1][3]float32
	giant    [worldSlots + 1]bool
	dazed    [worldSlots + 1]bool
	class    [worldSlots + 1]engine.Class
	tfTeam   [worldSlots + 1]engine.Team
}

// cannedWorld is one world with every case the bodies branch on in it: a slot
// out of the game, one in the game and dead, several teams, a weapon with no
// ammo, and a clip the SDKCall answers as negative.
func cannedWorld() world {
	var w world
	// Every predicate is decorrelated from every other on purpose. A world
	// where being alive and being on the enemy team are the same parity
	// filters out every candidate at once and says nothing about the loop.
	for i := 1; i <= worldSlots; i++ {
		w.inGame[i] = i != 3
		w.alive[i] = i != 5
		w.defender[i] = i <= 4
		w.team[i] = int32(i % 3)
		w.hasAmmo[i] = i != 5
		w.clip[i] = int32(i) - 2
		w.origin[i] = [3]float32{float32(i) * 64.5, float32(i) * -3.25, float32(i)}
		w.buster[i] = i == 7
		w.uber[i] = i == 4 || i == 10
		// Slot 2 is cloaked and unseen, slot 6 is cloaked and exposed, so
		// both sides of that pair of questions are walked.
		w.stealth[i] = i == 2 || i == 6
		w.exposed[i] = i == 6
		w.centre[i] = [3]float32{float32(i) * 64.5, float32(i) * -3.25, float32(i) + 41.0}
		w.giant[i] = i%3 == 0
		w.dazed[i] = i%5 == 0
		w.class[i] = engine.Class(i % 4)
		w.tfTeam[i] = engine.Team(2 + (i/2)%2)
	}
	// Two enemies of slot 1 in the same place. FindEnemyNearestToMe compares
	// with <=, so the later of two equally close ones is what it picks, and a
	// world with no tie in it cannot tell that apart from <.
	w.centre[11] = w.centre[10]
	return w
}

// installed is the world behind the engine calls, with the trace it records.
type installed struct {
	w     world
	trace []int32
}

// traceCells is the trace array the probe declares. The Go side holds itself to
// the same bound, so a body that called the engine more times than the probe can
// record fails here rather than writing past the end of a SourcePawn array.
const traceCells = 512

func (in *installed) record(id, arg int32) {
	if len(in.trace)+2 > traceCells {
		panic("the body made more engine calls than the probe's trace holds")
	}
	in.trace = append(in.trace, id, arg)
}

func (in *installed) calls() engine.Calls {
	return engine.Calls{
		IsClientInGame: func(c int32) bool { in.record(traceIsClientInGame, c); return in.w.inGame[c] },
		IsPlayerAlive:  func(c int32) bool { in.record(traceIsPlayerAlive, c); return in.w.alive[c] },
		GetClientTeam:  func(c int32) int32 { in.record(traceGetClientTeam, c); return in.w.team[c] },
		HasAmmo:        func(x int32) bool { in.record(traceHasAmmo, x); return in.w.hasAmmo[x] },
		Clip1:          func(x int32) int32 { in.record(traceClip1, x); return in.w.clip[x] },
		Origin:         func(c int32) [3]float32 { in.record(traceOrigin, c); return in.w.origin[c] },
		// Not traced: SourcePawn reads a variable here, and a variable
		// read is not a call the two sides could disagree about.
		MaxClients: func() int32 { return worldSlots },
		VectorDistance: func(a, b [3]float32) float32 {
			in.record(traceVectorDistance, 0)
			return vectorDistance(a, b)
		},
		IsSentryBusterRobot:    func(c int32) bool { in.record(traceIsSentryBusterRobot, c); return in.w.buster[c] },
		IsInvulnerable:         func(c int32) bool { in.record(traceIsInvulnerable, c); return in.w.uber[c] },
		IsStealthed:            func(c int32) bool { in.record(traceIsStealthed, c); return in.w.stealth[c] },
		IsCloakedPlayerExposed: func(c int32) bool { in.record(traceIsCloakedPlayerExposed, c); return in.w.exposed[c] },
		WorldSpaceCenter: func(e int32) [3]float32 {
			in.record(traceWorldSpaceCenter, e)
			return in.w.centre[e]
		},
		IsMiniBoss: func(c int32) bool { in.record(traceIsMiniBoss, c); return in.w.giant[c] },
		IsPlayerInCondition: func(c int32, cond engine.Condition) bool {
			in.record(traceIsPlayerInCondition, c)
			return cond == engine.ConditionDazed() && in.w.dazed[c]
		},
		PlayerClass: func(c int32) engine.Class { in.record(tracePlayerClass, c); return in.w.class[c] },
		PlayerTeam:  func(c int32) engine.Team { in.record(tracePlayerTeam, c); return in.w.tfTeam[c] },
		PlayerEnemyTeam: func(c int32) engine.Team {
			in.record(tracePlayerEnemyTeam, c)
			// Red fights blue, and every other slot is on each.
			if in.w.tfTeam[c] == 2 {
				return 3
			}
			return 2
		},
	}
}

// vectorDistance is the stub, and it is the square rather than the distance:
// the standalone SourcePawn has no square root, and what the probe needs is the
// same function on both sides rather than SourceMod's arithmetic. Accumulated
// in float32 in the order the probe accumulates it, so the two are one float
// and not two that print alike.
func vectorDistance(a, b [3]float32) float32 {
	sum := float32(0)
	for axis := range a {
		d := a[axis] - b[axis]
		sum += d * d
	}
	return sum
}

func boolCell(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

// worldInclude writes the canned world and the stubs that answer from it. The
// stubs are the engine as far as the generated body is concerned: SDKCall is
// one of them, because SourceMod's is not here either.
func worldInclude(w world) string {
	var b strings.Builder
	fmt.Fprintf(&b, "#define WORLD_SLOTS %d\n\n", worldSlots)
	writeBools(&b, "gInGame", w.inGame[:])
	writeBools(&b, "gAlive", w.alive[:])
	writeBools(&b, "gDefender", w.defender[:])
	writeBools(&b, "gHasAmmo", w.hasAmmo[:])
	writeInts(&b, "gTeam", w.team[:])
	writeInts(&b, "gClip", w.clip[:])
	writeVectors(&b, "gOrigin", w.origin[:])
	b.WriteString(`
int gTrace[512];
int gTraceLen = 0;

static void Trace(int id, int arg)
{
	gTrace[gTraceLen++] = id;
	gTrace[gTraceLen++] = arg;
}

`)
	fmt.Fprintf(&b, "stock bool IsClientInGame(int client) { Trace(%d, client); return gInGame[client]; }\n", traceIsClientInGame)
	fmt.Fprintf(&b, "stock bool IsPlayerAlive(int client) { Trace(%d, client); return gAlive[client]; }\n", traceIsPlayerAlive)
	fmt.Fprintf(&b, "stock int GetClientTeam(int client) { Trace(%d, client); return gTeam[client]; }\n", traceGetClientTeam)
	b.WriteString(`
/* The two SDKCall handles, and the SDKCall that reaches them. SourceMod's takes
   any number of arguments; both of ours take one, and a stub that accepts what
   the generated code actually writes is a stub that stays readable. */
#define m_hHasAmmo 1
#define m_hClip1   2

stock any SDKCall(int handle, int weapon)
{
	switch (handle)
	{
`)
	fmt.Fprintf(&b, "\t\tcase m_hHasAmmo: { Trace(%d, weapon); return gHasAmmo[weapon]; }\n", traceHasAmmo)
	fmt.Fprintf(&b, "\t\tcase m_hClip1:   { Trace(%d, weapon); return gClip[weapon]; }\n", traceClip1)
	b.WriteString("\t}\n\treturn 0;\n}\n\n")
	fmt.Fprintf(&b, "stock void GetClientAbsOrigin(int client, float origin[3])\n{\n\tTrace(%d, client);\n\tfor (int axis = 0; axis < 3; axis++)\n\t\torigin[axis] = gOrigin[client][axis];\n}\n", traceOrigin)
	return b.String()
}

func writeBools(b *strings.Builder, name string, values []bool) {
	fmt.Fprintf(b, "bool %s[%d] = {", name, len(values))
	for i, v := range values {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "%t", v)
	}
	b.WriteString("};\n")
}

func writeVectors(b *strings.Builder, name string, values [][3]float32) {
	fmt.Fprintf(b, "float %s[%d][3] =\n{\n", name, len(values))
	for _, v := range values {
		fmt.Fprintf(b, "\t{%s, %s, %s},\n", sp.FloatLiteral(v[0]), sp.FloatLiteral(v[1]), sp.FloatLiteral(v[2]))
	}
	b.WriteString("};\n")
}

func writeInts(b *strings.Builder, name string, values []int32) {
	fmt.Fprintf(b, "int %s[%d] = {", name, len(values))
	for i, v := range values {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "%d", v)
	}
	b.WriteString("};\n")
}

// compareCells is the comparison both probes make: cell for cell, nothing
// sampled, and a bounded report so a broken generator names ten disagreements
// rather than a thousand.
func compareCells(t *testing.T, want, got []int32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("the probe printed %d cells, the Go produced %d", len(got), len(want))
	}
	mismatches := 0
	const reportAtMost = 10
	for i := range want {
		if got[i] == want[i] {
			continue
		}
		mismatches++
		if mismatches <= reportAtMost {
			t.Errorf("cell %d: the Go says %d, the generated SourcePawn says %d", i, want[i], got[i])
		}
	}
	if mismatches > reportAtMost {
		t.Errorf("%d further disagreements", mismatches-reportAtMost)
	}
	t.Logf("compared %d cells, answers and call traces both", len(want))
}
