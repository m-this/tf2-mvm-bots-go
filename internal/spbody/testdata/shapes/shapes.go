// Package shapes is the golden input: one of everything the subset has, so a
// change to the emitter shows up as a diff a person can read.
package shapes

// Priority is a named integer, which is how the subset writes an enum.
type Priority int32

const (
	PriorityIdle Priority = iota
	PriorityBusy
	PriorityUrgent
)

// Slots is a plain constant, which becomes a define so it can size an array.
const Slots = 4

// Sample is a named struct, which becomes an enum struct.
type Sample struct {
	Client int32
	Score  float32
	Recent [Slots]int32
}

// seen is package state, which becomes a SourcePawn global. Its initialiser is
// written once at load, so it is a constant or an array of them.
var seen [Slots]int32 = [Slots]int32{1, 2}

// worst is the scalar shape of the same thing.
var worst Priority = PriorityIdle

// Note records a client and returns what the highest priority seen so far is,
// which is the read-and-write a state emission has to get right.
func Note(slot int32, p Priority) Priority {
	seen[slot]++
	if p > worst {
		worst = p
	}
	return worst
}

// Rank folds the sample into a priority, and shows the switch, the tagged
// return and the float comparison in one place.
func Rank(s Sample, threshold float32) Priority {
	if s.Score > threshold {
		return PriorityUrgent
	}
	switch s.Client {
	case 0:
		return PriorityIdle
	case 1, 2:
		return PriorityBusy
	}
	return PriorityIdle
}

// SumRecent shows the range loop, the array index and the conversion the
// generator has to spell as a call.
func SumRecent(s Sample) (total int32, average float32) {
	total = 0
	for i := range s.Recent {
		total += s.Recent[i]
	}
	return total, float32(total) / float32(Slots)
}

// Centre is the vector shape: SourcePawn returns a cell, so an array result
// becomes the parameter the caller supplies and this fills.
func Centre(s Sample) (centre [3]float32) {
	centre[0] = s.Score
	centre[1] = float32(s.Client)
	centre[2] = 0.0
	return centre
}

// Offset shows the call site: a declaration and a call, never an expression.
func Offset(s Sample) float32 {
	centre := Centre(s)
	return centre[0] + centre[1]
}

// Clamp shows min and max, the local declaration with no initialiser, and the
// defaults the SourcePawn callers of the ported functions rely on.
//
//sp:default low 0
//sp:default high 100
func Clamp(v int32, low int32, high int32) int32 {
	var out int32
	out = min(max(v, low), high)
	return out
}
