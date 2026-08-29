package body_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
	"github.com/m-this/tf2-mvm-bots-go/internal/body/roster"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
	"github.com/m-this/tf2-mvm-bots-go/internal/spshell"
)

// worldSlots is how many client slots the canned world has. Small on purpose:
// the proof here is that two translations of one function agree, and a wider
// sweep would say the same thing more slowly.
const worldSlots = 8

// The trace ids. One per engine call, and the emitted trace is the sequence of
// them with the argument that was asked about, so a body that answered right by
// asking the wrong questions still fails.
const (
	traceIsClientInGame = iota + 1
	traceIsPlayerAlive
	traceGetClientTeam
	traceHasAmmo
	traceClip1
	traceIsDefenderBot
	traceSetTouchCredits
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
}

// cannedWorld is one world with every case the bodies branch on in it: a slot
// out of the game, one in the game and dead, several teams, a weapon with no
// ammo, and a clip the SDKCall answers as negative.
func cannedWorld() world {
	var w world
	for i := 1; i <= worldSlots; i++ {
		w.inGame[i] = i != 3
		w.alive[i] = i%2 == 1
		w.defender[i] = i <= 4
		w.team[i] = int32(i % 3)
		w.hasAmmo[i] = i != 5
		w.clip[i] = int32(i) - 2
	}
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
		IsDefenderBot:  func(c int32) bool { in.record(traceIsDefenderBot, c); return in.w.defender[c] },
		SetTouchCredits: func(touching bool) {
			in.record(traceSetTouchCredits, boolCell(touching))
		},
	}
}

func boolCell(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

// goCells runs the Go the bodies were written as, in the order the probe runs
// them, and produces the same cell stream the probe prints.
func goCells(w world) []int32 {
	in := &installed{w: w}
	defer engine.Install(in.calls())()

	var out []int32
	emit := func(result int32) {
		out = append(out, result, int32(len(in.trace))) //nolint:gosec // G115: record bounds the trace at traceCells
		out = append(out, in.trace...)
		in.trace = in.trace[:0]
	}
	for team := int32(0); team < 4; team++ {
		emit(roster.AliveOnTeam(worldSlots, team))
	}
	for weapon := int32(0); weapon <= worldSlots; weapon++ {
		emit(roster.LoadedRounds(weapon))
	}
	for player := int32(1); player <= worldSlots; player++ {
		roster.MyTouchPre(0, player)
		emit(0)
	}
	for client := int32(1); client <= worldSlots; client++ {
		for _, touching := range []bool{false, true} {
			supercede, value := roster.IsBotPre(client, touching)
			out = append(out, boolCell(supercede))
			emit(boolCell(value))
		}
	}
	return out
}

/*
	TestGeneratedBodiesAgreeWithTheGoTheyCameFrom

The deliverable of mvm-bis. A body that calls the engine cannot be run here, so
the engine is a stub on both sides answering out of one canned world, and what
is compared is the answer and the sequence of calls that produced it.

Nothing is sampled. Every case the probe runs is compared cell for cell.
*/
func TestGeneratedBodiesAgreeWithTheGoTheyCameFrom(t *testing.T) {
	tc := spshell.ForTest(t)
	w := cannedWorld()

	generated, err := body.Generate("../..")
	if err != nil {
		t.Fatalf("generating the bodies: %v", err)
	}
	cells, err := tc.Run(t.Context(), "testdata/roster_probe.sp", map[string]string{
		"roster.sp":        string(generated["sourcepawn/roster.sp"]),
		"roster_world.inc": worldInclude(w),
	})
	if err != nil {
		t.Fatalf("running the probe under spshell: %v", err)
	}

	want := goCells(w)
	if len(cells) != len(want) {
		t.Fatalf("the probe printed %d cells, the Go produced %d", len(cells), len(want))
	}
	mismatches := 0
	const reportAtMost = 10
	for i := range want {
		if cells[i] == want[i] {
			continue
		}
		mismatches++
		if mismatches <= reportAtMost {
			t.Errorf("cell %d: the Go says %d, the generated SourcePawn says %d", i, want[i], cells[i])
		}
	}
	if mismatches > reportAtMost {
		t.Errorf("%d further disagreements", mismatches-reportAtMost)
	}
	t.Logf("compared %d cells, answers and call traces both", len(want))
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
	fmt.Fprintf(&b, "stock bool IsDefenderBot(int client) { Trace(%d, client); return gDefender[client]; }\n", traceIsDefenderBot)
	fmt.Fprintf(&b, "stock void SetTouchCredits(bool touching) { Trace(%d, view_as<int>(touching)); }\n", traceSetTouchCredits)
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
	b.WriteString("\t}\n\treturn 0;\n}\n")
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
