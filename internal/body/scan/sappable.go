package scan

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// The spy's four scans. They are the client loop again with two more questions
// on the end, and the only thing that separates the nearest from the farthest is
// which way the comparison points and what it starts at.

// PlayerSappable is util.sp:1437, IsPlayerSappable.
func PlayerSappable(client int32) bool {
	if engine.IsPlayerInCondition(client, engine.ConditionSapped()) {
		return false
	}
	if engine.IsInvulnerable(client) {
		return false
	}
	if engine.IsPlayerInCondition(client, engine.ConditionBonked()) {
		return false
	}
	return true
}

// PlayerHealingSomething is util.sp:1690, IsPlayerHealingSomething.
func PlayerHealingSomething(client int32) bool {
	weapon := engine.ActiveWeapon(client)

	if weapon == -1 {
		return false
	}

	return engine.WeaponID(weapon) == engine.WeaponMedigun() &&
		engine.EntPropEnt(weapon, engine.PropSend(), "m_hHealingTarget") != -1
}

// NearestSappablePlayer is util.sp:1451, GetNearestSappablePlayer.
//
//sp:default giantsOnly false
//sp:default class TFClass_Unknown
//sp:default speedCheck 0.0
func NearestSappablePlayer(client int32, maxDistance float32, giantsOnly bool, class engine.Class, speedCheck float32) int32 {
	origin := engine.Origin(client)

	enemyTeam := PlayerEnemyTeam(client)
	bestDistance := float32(999999.0)
	bestEntity := int32(-1)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if !sappableCandidate(client, i, enemyTeam, giantsOnly, class, speedCheck) {
			continue
		}
		if !PlayerSappable(i) {
			continue
		}
		distance := engine.VectorDistance(WorldSpaceCenter(i), origin)

		if distance <= bestDistance && distance <= maxDistance {
			bestDistance = distance
			bestEntity = i
		}
	}

	return bestEntity
}

// FarthestSappablePlayer is util.sp:1501, GetFarthestSappablePlayer. Same loop,
// starting at zero and keeping the largest.
//
//sp:default giantsOnly false
//sp:default class TFClass_Unknown
//sp:default speedCheck 0.0
func FarthestSappablePlayer(client int32, maxDistance float32, giantsOnly bool, class engine.Class, speedCheck float32) int32 {
	origin := engine.Origin(client)

	enemyTeam := PlayerEnemyTeam(client)
	bestDistance := float32(0.0)
	bestEntity := int32(-1)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if !sappableCandidate(client, i, enemyTeam, giantsOnly, class, speedCheck) {
			continue
		}
		if !PlayerSappable(i) {
			continue
		}
		distance := engine.VectorDistance(WorldSpaceCenter(i), origin)

		if distance >= bestDistance && distance <= maxDistance {
			bestDistance = distance
			bestEntity = i
		}
	}

	return bestEntity
}

// NearestSappablePlayerHealingSomeone is util.sp:1638. The same loop again with
// the medigun question in front of the sappable one, in that order, because the
// order is what the engine call trace records.
//
//sp:default giantsOnly false
//sp:default class TFClass_Unknown
//sp:default speedCheck 0.0
func NearestSappablePlayerHealingSomeone(client int32, maxDistance float32, giantsOnly bool, class engine.Class, speedCheck float32) int32 {
	origin := engine.Origin(client)

	enemyTeam := PlayerEnemyTeam(client)
	bestDistance := float32(999999.0)
	bestEntity := int32(-1)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if !sappableCandidate(client, i, enemyTeam, giantsOnly, class, speedCheck) {
			continue
		}
		if !PlayerHealingSomething(i) {
			continue
		}
		if !PlayerSappable(i) {
			continue
		}
		distance := engine.VectorDistance(WorldSpaceCenter(i), origin)

		if distance <= bestDistance && distance <= maxDistance {
			bestDistance = distance
			bestEntity = i
		}
	}

	return bestEntity
}

// sappableCandidate is the run of seven questions the three loops above ask
// before they ask anything of their own, in the order they ask them. It is the
// one place they were already identical in util.sp, so lifting it out changes
// no order and no answer; the rest of the collapse is mvm-z83.35 and waits
// until every variant is across.
func sappableCandidate(client int32, i int32, enemyTeam engine.Team, giantsOnly bool, class engine.Class, speedCheck float32) bool {
	if i == client {
		return false
	}
	if !engine.IsClientInGame(i) {
		return false
	}
	if !engine.IsPlayerAlive(i) {
		return false
	}
	if engine.PlayerTeam(i) != enemyTeam {
		return false
	}
	if engine.IsSentryBusterRobot(i) {
		return false
	}
	if giantsOnly && !engine.IsMiniBoss(i) {
		return false
	}
	if class > engine.ClassUnknown() && engine.PlayerClass(i) != class {
		return false
	}
	// Not fast enough
	if speedCheck > 0.0 && engine.EntPropFloat(i, engine.PropSend(), "m_flMaxspeed") < speedCheck {
		return false
	}
	return true
}

// EnemyPlayerNearestToPosition is util.sp:1550,
// GetEnemyPlayerNearestToPosition: the shortest of the loops, measuring from a
// position the caller supplies rather than from where the client stands.
func EnemyPlayerNearestToPosition(client int32, position [3]float32, maxDistance float32) int32 {
	enemyTeam := PlayerEnemyTeam(client)
	bestDistance := float32(999999.0)
	bestEntity := int32(-1)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if i == client {
			continue
		}
		if !engine.IsClientInGame(i) {
			continue
		}
		if !engine.IsPlayerAlive(i) {
			continue
		}
		if engine.PlayerTeam(i) != enemyTeam {
			continue
		}
		if engine.IsSentryBusterRobot(i) {
			continue
		}
		distance := engine.VectorDistance(WorldSpaceCenter(i), position)

		if distance <= bestDistance && distance <= maxDistance {
			bestDistance = distance
			bestEntity = i
		}
	}

	return bestEntity
}
