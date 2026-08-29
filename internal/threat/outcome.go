package threat

// Priority is what a target is worth shooting first. The numbers are an order
// and not a measurement: the decision compares two of them and takes the
// larger, so only the ordering means anything.
type Priority int32

// The seven answers, in the plugin's declared order. The order is the
// behaviour: THREAT_PRIORITY_URGENT is last because it has to beat everything.
const (
	PriorityNone Priority = iota
	PriorityBomb
	PriorityGiant
	PriorityGiantBomb
	PrioritySupport
	PriorityMedic
	PriorityUrgent
	numPriorities
)

// Enum is the SourcePawn constant this answer is, so a failure message names
// what the plugin calls it and the generator writes the enum from one place.
func (p Priority) Enum() string {
	if s, ok := priorityEnum[p]; ok {
		return s
	}
	return "THREAT_PRIORITY_INVALID"
}

func (p Priority) String() string {
	if s, ok := priorityName[p]; ok {
		return s
	}
	return "Priority(?)"
}

var priorityEnum = map[Priority]string{
	PriorityNone:      "THREAT_PRIORITY_NONE",
	PriorityBomb:      "THREAT_PRIORITY_BOMB",
	PriorityGiant:     "THREAT_PRIORITY_GIANT",
	PriorityGiantBomb: "THREAT_PRIORITY_GIANT_BOMB",
	PrioritySupport:   "THREAT_PRIORITY_SUPPORT",
	PriorityMedic:     "THREAT_PRIORITY_MEDIC",
	PriorityUrgent:    "THREAT_PRIORITY_URGENT",
}

var priorityName = map[Priority]string{
	PriorityNone: "None", PriorityBomb: "Bomb", PriorityGiant: "Giant",
	PriorityGiantBomb: "GiantBomb", PrioritySupport: "Support",
	PriorityMedic: "Medic", PriorityUrgent: "Urgent",
}

// Priorities is every answer, in declared order.
func Priorities() []Priority {
	all := make([]Priority, 0, numPriorities)
	for p := PriorityNone; p < numPriorities; p++ {
		all = append(all, p)
	}
	return all
}
