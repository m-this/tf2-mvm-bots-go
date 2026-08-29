package spgen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/sp"
	"github.com/m-this/tf2-mvm-bots-go/internal/tf"
	"github.com/m-this/tf2-mvm-bots-go/internal/threat"
)

/*
	The threat priority table

Different in shape from the action selection table above, and the difference is
worth stating rather than leaving as an absence. Action selection has to be
walked one question at a time because three of its predicates have side effects,
which is mvm-z83.40. Nothing this decision reads costs anything: six reads, no
writes, no randomness. So the edge fills a record and indexes, and there is no
walk to get wrong.

The range comes in as the float32 the plugin already holds and is reduced to a
band by the two comparisons the plugin makes, in the emitted code, so the
boundary behaviour is the boundary behaviour rather than something the caller
decided.
*/

// threatCells is the flat table, indexed the way ThreatIndex below indexes it.
func threatCells() []int32 {
	cells := make([]int32, threatTableSize())
	for _, band := range threat.Bands() {
		for _, class := range tf.Classes() {
			for bits := range threatFlagCount {
				t := threat.Threat{
					IsPlayer: bits&1 != 0,
					InGame:   bits&2 != 0,
					Giant:    bits&4 != 0,
					Carrier:  bits&8 != 0,
					Class:    class,
				}
				cells[threatIndex(band, class, bits)] = int32(threat.PriorityOf(threatAt(t, band)))
			}
		}
	}
	return cells
}

// threatFlagCount is the four booleans the record carries beside the band and
// the class.
const threatFlagCount = 1 << 4

func threatTableSize() int {
	return threat.NumBands * int(tf.NumClasses) * threatFlagCount
}

// threatIndex is the one place the row-major order is written. The emitted
// SourcePawn computes the same expression, and TestThreatTableIndexesTheSame
// holds the two together.
func threatIndex(band threat.Band, class tf.Class, bits int) int {
	return (int(band)*int(tf.NumClasses)+int(class))*threatFlagCount + bits
}

// threatAt puts a record in the middle of a band, which is all the table can
// say: the band is what the decision reads.
func threatAt(t threat.Threat, band threat.Band) threat.Threat {
	urgent := threat.UrgentRange * threat.UrgentRange
	far := threat.PriorityRange * threat.PriorityRange
	switch band {
	case threat.BandUrgent:
		t.RangeSquared = 0
	case threat.BandTooFar:
		t.RangeSquared = far + 1
	default:
		t.RangeSquared = (urgent + far) / 2
	}
	return t
}

// EmitThreatPriority writes the threat priority side of the plugin: the two
// ranges, the priority enum, the band function and the table lookup.
func EmitThreatPriority() []byte {
	var b strings.Builder

	b.WriteString(`/* Generated from internal/threat. Do not edit.

What a robot is worth killing first. The numbers are an order, not a measurement: the caller
compares two of them and takes the larger, so only the ordering means anything.

Anything inside THREAT_URGENT_RANGE outranks the list. A bot that ignores the Heavy in front of it
to shoot a Sniper across the map dies holding a good idea.

Both distances were widened after measuring. At 400 units the order was costing more than it
bought: ten runs on Decoy put defender deaths at 54 against the old code's 43, for the same waves
cleared. 400 is a rocket's splash, not a firefight, so a bot would walk its aim off the Heavy
shooting it as soon as anything better appeared anywhere. And a priority target beyond
THREAT_PRIORITY_RANGE is not a target, it is a plan: past that the nearest one wins. */

`)
	fmt.Fprintf(&b, "#define THREAT_URGENT_RANGE\t\t%s\n", sp.FloatLiteral(threat.UrgentRange))
	fmt.Fprintf(&b, "#define THREAT_PRIORITY_RANGE\t%s\n\n", sp.FloatLiteral(threat.PriorityRange))

	b.WriteString("enum\n{\n")
	for i, p := range threat.Priorities() {
		if i == 0 {
			fmt.Fprintf(&b, "\t%s = 0,\n", p.Enum())
			continue
		}
		fmt.Fprintf(&b, "\t%s,\n", p.Enum())
	}
	b.WriteString("};\n\n")

	fmt.Fprintf(&b, "#define THREAT_BANDS\t\t%d\n", threat.NumBands)
	fmt.Fprintf(&b, "#define THREAT_CLASSES\t\t%d\n", int(tf.NumClasses))
	fmt.Fprintf(&b, "#define THREAT_FLAGS\t\t%d\n\n", threatFlagCount)

	b.WriteString(`/* Which side of the two range comparisons a target falls on

Both tests are strict, so a range exactly on a boundary is in the band above it. This is the whole
of what the decision reads about a distance. */
stock int ThreatBand(float rangeSq)
{
	if (rangeSq < THREAT_URGENT_RANGE * THREAT_URGENT_RANGE)
		return 0;
	
	if (rangeSq > THREAT_PRIORITY_RANGE * THREAT_PRIORITY_RANGE)
		return 2;
	
	return 1;
}

`)
	b.WriteString("static const int g_ThreatPriority[THREAT_BANDS * THREAT_CLASSES * THREAT_FLAGS] =\n{\n")
	cells := threatCells()
	for i := 0; i < len(cells); i += threatFlagCount {
		row := make([]string, 0, threatFlagCount)
		for _, cell := range cells[i : i+threatFlagCount] {
			row = append(row, strconv.FormatInt(int64(cell), 10))
		}
		fmt.Fprintf(&b, "\t%s,\n", strings.Join(row, ", "))
	}
	b.WriteString("};\n\n")

	b.WriteString(`/* What one threat is worth, from what the caller already knows about it

A record rather than an entity index, which is the point of the move. Whoever fills this decides
what counts as a threat, and this decides what it is worth. Every scan in the mod that finds one
walks player slots, and a tank occupies none, which is mvm-ds3: not fixed here, and fixable here
for the first time.

isPlayer and inGame are read as one pair because the shipped code tested them together, and the
caller has to fill them the same way: the shipped test is || and short circuits, so IsClientInGame
is never reached for a non-player. Calling it anyway to fill this record throws "Client index is
invalid" on every tank in the game. Fill inGame as isPlayer && IsClientInGame(threat). */
stock int ThreatPriorityOf(float rangeSq, bool isPlayer, bool inGame, TFClassType pclass, bool giant, bool carrier)
{
	int flags = (isPlayer ? 1 : 0) | (inGame ? 2 : 0) | (giant ? 4 : 0) | (carrier ? 8 : 0);
	int index = (ThreatBand(rangeSq) * THREAT_CLASSES + view_as<int>(pclass)) * THREAT_FLAGS + flags;
	
	return g_ThreatPriority[index];
}
`)
	return []byte(b.String())
}
