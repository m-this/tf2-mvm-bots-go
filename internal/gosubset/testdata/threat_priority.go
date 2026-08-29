// The threat ranking of source/redbots3/nextbot_behavior.sp, written in the
// subset. The engine reads are done by the caller and arrive as a Threat.
package decisions

type ThreatPriority int32

const (
	ThreatPriorityNone ThreatPriority = iota
	ThreatPriorityBomb
	ThreatPriorityGiant
	ThreatPriorityGiantBomb
	ThreatPrioritySupport
	ThreatPriorityMedic
	ThreatPriorityUrgent
)

const (
	urgentRange   float32 = 512.0
	priorityRange float32 = 3000.0
)

const (
	classSniper   int32 = 2
	classMedic    int32 = 5
	classEngineer int32 = 9
)

type Threat struct {
	Class      int32
	RangeSq    float32
	IsPlayer   bool
	IsMiniBoss bool
	HasFlag    bool
	Visible    bool
}

func threatPriorityOf(t Threat) ThreatPriority {
	if t.RangeSq < urgentRange*urgentRange {
		return ThreatPriorityUrgent
	}
	if t.RangeSq > priorityRange*priorityRange {
		return ThreatPriorityNone
	}
	if !t.IsPlayer {
		return ThreatPriorityNone
	}

	switch t.Class {
	case classMedic:
		return ThreatPriorityMedic
	case classSniper, classEngineer:
		return ThreatPrioritySupport
	}

	if t.IsMiniBoss && t.HasFlag {
		return ThreatPriorityGiantBomb
	}
	if t.IsMiniBoss {
		return ThreatPriorityGiant
	}
	if t.HasFlag {
		return ThreatPriorityBomb
	}
	return ThreatPriorityNone
}

// MoreDangerousThreat answers 0 for the first threat and 1 for the second.
func MoreDangerousThreat(first Threat, second Threat, priorityFeature bool) int32 {
	if first.Visible && !second.Visible {
		return 0
	}
	if second.Visible && !first.Visible {
		return 1
	}

	firstPriority := threatPriorityOf(first)
	secondPriority := threatPriorityOf(second)

	if priorityFeature && firstPriority != secondPriority {
		if firstPriority > secondPriority {
			return 0
		}
		return 1
	}
	if first.RangeSq < second.RangeSq {
		return 0
	}
	return 1
}
