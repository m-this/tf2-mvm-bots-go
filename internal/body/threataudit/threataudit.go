/*
Package threataudit is what a robot is worth killing first.

It was three things: the chain that shipped, the table's answer, and an audit
that played one against the other in a running game. The audit closed on
2026-09-06, three waves of Decoy with the table armed, 137326 and 262474
comparisons in the two waves it reported and not one disagreement, on top of the
differential test that proves the decision and the table agree over the whole
domain under SourcePawn's own VM. So the chain and the audit are gone and this
is the only answer left. See mvm-z83.47.
*/
package threataudit

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
ThreatPriority is the question, asked of the generated table.

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
//sp:name ThreatPriority
func ThreatPriority(threat int32, rangeSq float32) int32 {
	isPlayer := engine.IsPlayer(threat)
	inGame := isPlayer && engine.IsClientInGame(threat)

	if !inGame {
		return engine.ThreatPriorityOf(rangeSq, isPlayer, false, engine.ClassUnknown(), false, false)
	}

	return engine.ThreatPriorityOf(rangeSq, isPlayer, true, engine.PlayerClass(threat),
		engine.IsMiniBoss(threat), engine.HasTheFlag(threat))
}
