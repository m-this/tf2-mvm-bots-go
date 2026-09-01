package engine

/*
The threat priority table, and the ranges and ranks it is written in terms of.

internal/threat authors the decision and emits threat_priority.sp; this is how a
body reaches what it emitted.
*/

// ThreatAuditCalls are the answers.
type ThreatAuditCalls struct {
	ThreatPriorityOf func(rangeSq float32, isPlayer bool, inGame bool, playerClass Class, giant bool, carrier bool) int32
}

var threatAudits ThreatAuditCalls

// InstallThreatAudits puts a set of answers behind them.
func InstallThreatAudits(c ThreatAuditCalls) func() {
	previous := threatAudits
	threatAudits = c
	return func() { threatAudits = previous }
}

// ThreatPriorityOf is the table's answer, taking what is known about a threat
// rather than an entity index. Ported, internal/threat.
//
//sp:body ThreatPriorityOf
func ThreatPriorityOf(rangeSq float32, isPlayer bool, inGame bool, playerClass Class, giant bool, carrier bool) int32 {
	if threatAudits.ThreatPriorityOf == nil {
		missing("ThreatPriorityOf")
	}
	return threatAudits.ThreatPriorityOf(rangeSq, isPlayer, inGame, playerClass, giant, carrier)
}

// ThreatUrgentRange is THREAT_URGENT_RANGE, close enough to be killing the bot.
//
//sp:global THREAT_URGENT_RANGE
func ThreatUrgentRange() float32 { return 0 }

// ThreatPriorityRange is THREAT_PRIORITY_RANGE, past which nothing is worth
// walking the aim across the map for.
//
//sp:global THREAT_PRIORITY_RANGE
func ThreatPriorityRange() float32 { return 0 }

// The ranks, in the order the table gives them.

// ThreatPriorityUrgent is THREAT_PRIORITY_URGENT.
//
//sp:global THREAT_PRIORITY_URGENT
func ThreatPriorityUrgent() int32 { return 0 }

// ThreatPriorityNone is THREAT_PRIORITY_NONE.
//
//sp:global THREAT_PRIORITY_NONE
func ThreatPriorityNone() int32 { return 0 }

// ThreatPriorityMedic is THREAT_PRIORITY_MEDIC.
//
//sp:global THREAT_PRIORITY_MEDIC
func ThreatPriorityMedic() int32 { return 0 }

// ThreatPrioritySupport is THREAT_PRIORITY_SUPPORT.
//
//sp:global THREAT_PRIORITY_SUPPORT
func ThreatPrioritySupport() int32 { return 0 }

// ThreatPriorityGiantBomb is THREAT_PRIORITY_GIANT_BOMB.
//
//sp:global THREAT_PRIORITY_GIANT_BOMB
func ThreatPriorityGiantBomb() int32 { return 0 }

// ThreatPriorityGiant is THREAT_PRIORITY_GIANT.
//
//sp:global THREAT_PRIORITY_GIANT
func ThreatPriorityGiant() int32 { return 0 }

// ThreatPriorityBomb is THREAT_PRIORITY_BOMB.
//
//sp:global THREAT_PRIORITY_BOMB
func ThreatPriorityBomb() int32 { return 0 }
