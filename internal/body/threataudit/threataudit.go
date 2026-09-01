/*
Package threataudit is the threat priority scaffolding out of
source/redbots3/nextbot_behavior.sp: the chain that shipped, the table's answer,
and the audit that plays one against the other in a running game.

All of it is meant to go. mvm-z83.47 says what closes it: play three waves
armed, read the per wave line, and require compared to be a large number and
disagreed to be zero. The chain and the audit are deleted then, and
ThreatPriorityGenerated becomes the only answer.
*/
package threataudit

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
ThreatPriority is what a robot is worth killing first, as the chain that
shipped.

Every guide written about this mode says the same order and none of it was here:
the Medic first because a giant being healed cannot be killed at all, then the
Sniper and the Engineer because they are the two the rest of the team cannot
reach, then giants, then whoever is holding the bomb. A robot close enough to be
killing the bot outranks all of it, because a priority target is worth nothing
to a corpse.
*/
//
//sp:name ThreatPriority
func ThreatPriority(threat int32, rangeSq float32) int32 {
	if rangeSq < engine.ThreatUrgentRange()*engine.ThreatUrgentRange() {
		return engine.ThreatPriorityUrgent()
	}

	// Too far to be worth walking the aim across the map for.
	if rangeSq > engine.ThreatPriorityRange()*engine.ThreatPriorityRange() {
		return engine.ThreatPriorityNone()
	}

	if !engine.IsPlayer(threat) || !engine.IsClientInGame(threat) {
		return engine.ThreatPriorityNone()
	}

	switch engine.PlayerClass(threat) {
	// A giant with a Medic on it is not killable until the Medic is dead.
	case engine.ClassMedic():
		return engine.ThreatPriorityMedic()

	// The two the rest of the team cannot get to: one sits out of reach, the
	// other builds.
	case engine.ClassSniper(), engine.ClassEngineer():
		return engine.ThreatPrioritySupport()
	}

	giant := engine.IsMiniBoss(threat)
	carrier := engine.HasTheFlag(threat)

	// Carrying the bomb halves a robot's speed, except a giant's, so that one
	// is still running.
	if giant && carrier {
		return engine.ThreatPriorityGiantBomb()
	}

	if giant {
		return engine.ThreatPriorityGiant()
	}

	if carrier {
		return engine.ThreatPriorityBomb()
	}

	return engine.ThreatPriorityNone()
}

/*
ThreatPriorityGenerated is the same question, asked of the generated table.

The record is what the move in mvm-z83.6 was for: the decision takes what is
known about a threat rather than an entity index, so something that occupies no
player slot can still be ranked. Every threat scan in this mod walks player slots
and a tank occupies none, which is mvm-ds3, and this does not fix it. It makes
fixing it possible.

Every field after isPlayer is filled behind it, not beside it, and the first
version of this was not: all three throw when asked about something that is not
a player. Measured, TF2_HasTheFlag threw 3933 times over four waves on tank_boss
and obj_attachment_sapper, and each one aborted the whole threat choice for that
tick. See mvm-z83.46.
*/
//
//sp:name ThreatPriorityGenerated
func ThreatPriorityGenerated(threat int32, rangeSq float32) int32 {
	isPlayer := engine.IsPlayer(threat)
	inGame := isPlayer && engine.IsClientInGame(threat)

	if !inGame {
		return engine.ThreatPriorityOf(rangeSq, isPlayer, false, engine.ClassUnknown(), false, false)
	}

	return engine.ThreatPriorityOf(rangeSq, isPlayer, true, engine.PlayerClass(threat),
		engine.IsMiniBoss(threat), engine.HasTheFlag(threat))
}

//sp:name g_iThreatSplits
var threatSplits int32

//sp:name g_iThreatCompared
var threatCompared int32

/*
ThreatPortAuditReport says how much was compared, not only what disagreed.

Zero disagreements and never having run look identical in a log that only writes
on a disagreement, and reading the first as the second is the fault mvm-z83.23 is
about.
*/
//
//sp:name ThreatPortAudit_Report
func ThreatPortAuditReport() {
	if threatCompared == 0 {
		return
	}

	engine.LogMessage("threat audit: %d compared, %d disagreed", threatCompared, threatSplits)

	threatCompared = 0
}

/*
ThreatPortAudit is where the generated answer and the shipped chain part company.

The differential test proves the decision and the table agree on every
combination it can be asked about. It cannot prove the edge fills the record the
way the chain reads it, because it drives both sides from the same record. Only
a running game can answer that.

It runs on the armed side only, so the other arm pays nothing, and it stops
writing after twenty lines because a disagreement that happens at all is the
finding and a log full of them is not more of one.
*/
//
//sp:name ThreatPortAudit
func ThreatPortAudit(threat int32, rangeSq float32) {
	threatCompared++

	if threatSplits >= 20 {
		return
	}

	shipped := ThreatPriority(threat, rangeSq)
	fromTable := ThreatPriorityGenerated(threat, rangeSq)

	if shipped == fromTable {
		return
	}

	threatSplits++

	classname := engine.EntityClassname(threat)

	engine.LogMessage("threat audit: entity %d/%s rangeSq %.0f, chain says %d, table says %d",
		threat, classname, rangeSq, shipped, fromTable)
}
