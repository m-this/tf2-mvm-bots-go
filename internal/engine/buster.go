package engine

/*
The sentry buster, and what a bot does about one.

The buster is the one robot in the mode that kills a defender who does nothing,
and it announces itself. Everything here is either the plugin's own or a
constant it defines, and each goes when the port reaches the file it lives in.
*/

// BusterCalls are the answers.
type BusterCalls struct {
	DetonatingPlayer     func() int32
	FindSentryBusterNear func(origin [3]float32, enemyTeam Team, maxRange float32) int32
	IsInUpgradeZone      func(client int32) bool
}

var busters BusterCalls

// InstallBusters puts a set of answers behind them.
func InstallBusters(c BusterCalls) func() {
	previous := busters
	busters = c
	return func() { busters = previous }
}

// BusterBlastRange is what the explosion reaches. Valve's own is smaller, and a
// bot that stops running early is dead.
//
//sp:global BUSTER_BLAST_RANGE
func BusterBlastRange() float32 { return 400.0 }

// BusterFleeRange is how close a live buster has to be before a bot drops what
// it is doing and runs.
//
//sp:global BUSTER_FLEE_RANGE
func BusterFleeRange() float32 { return 700.0 }

// ConceptIncoming is MP_CONCEPT_PLAYER_INCOMING.
//
//sp:global MP_CONCEPT_PLAYER_INCOMING
func ConceptIncoming() int32 { return 3 }

// DetonatingPlayer is the buster that has started its countdown, and an invalid
// index when none has.
//
//sp:global g_iDetonatingPlayer
func DetonatingPlayer() int32 {
	if busters.DetonatingPlayer == nil {
		missing("g_iDetonatingPlayer")
	}
	return busters.DetonatingPlayer()
}

// FindSentryBusterNear is a live buster within range, or -1.
//
//sp:plugin FindSentryBusterNear
func FindSentryBusterNear(origin [3]float32, enemyTeam Team, maxRange float32) int32 {
	if busters.FindSentryBusterNear == nil {
		missing("FindSentryBusterNear")
	}
	return busters.FindSentryBusterNear(origin, enemyTeam, maxRange)
}

// IsInUpgradeZone says the bot is at the station between waves.
//
//sp:plugin TF2_IsInUpgradeZone
func IsInUpgradeZone(client int32) bool {
	if busters.IsInUpgradeZone == nil {
		missing("TF2_IsInUpgradeZone")
	}
	return busters.IsInUpgradeZone(client)
}
