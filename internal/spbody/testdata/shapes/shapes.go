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

// Clamp shows min and max, and the local declaration with no initialiser.
func Clamp(v int32, low int32, high int32) int32 {
	var out int32
	out = min(max(v, low), high)
	return out
}
