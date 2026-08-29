/*
Package scan is util.sp's client loop, ported one function at a time.

util.sp 1183-1690 holds nine variants of the same scan over client slots. They
move here as they are, one per commit, and the duplication is collapsed once
they are all here and their behaviour is pinned. Collapsing them on the way
across would mean a run that moved could not say whether the port or the
collapse moved it, which is mvm-z83.41.

The bug they share, mvm-ds3, is not fixed here. Every one of them loops player
slots and a tank occupies none, and that stays true in the port.
*/
package scan

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// NearestEnemyCount is util.sp:1398, GetNearestEnemyCount: how many enemies are
// within max_distance of the client, not counting the client.
//
//sp:default ignoreUber false
func NearestEnemyCount(client int32, maxDistance float32, ignoreUber bool) int32 {
	origin := engine.Origin(client)

	myTeam := engine.GetClientTeam(client)
	count := int32(0)

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
		if engine.GetClientTeam(i) == myTeam {
			continue
		}
		// Usually not a threat
		if engine.IsSentryBusterRobot(i) {
			continue
		}
		if ignoreUber && engine.IsInvulnerable(i) {
			continue
		}
		if engine.IsStealthed(i) && !engine.IsCloakedPlayerExposed(i) {
			continue
		}
		if engine.VectorDistance(engine.WorldSpaceCenter(i), origin) <= maxDistance {
			count++
		}
	}

	return count
}

// EnemyNearestToMe is util.sp:1183, FindEnemyNearestToMe: the closest enemy
// within max_distance, or -1. The four filters after the team check are the
// ones its callers switch on.
//
//sp:default giantsOnly false
//sp:default ignoreUber false
//sp:default stunnedOnly false
//sp:default class TFClass_Unknown
func EnemyNearestToMe(client int32, maxDistance float32, giantsOnly bool, ignoreUber bool, stunnedOnly bool, class engine.Class) int32 {
	origin := engine.WorldSpaceCenter(client)

	bestDistance := float32(999999.0)
	bestEntity := int32(-1)
	enemyTeam := engine.PlayerEnemyTeam(client)

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
		if giantsOnly && !engine.IsMiniBoss(i) {
			continue
		}
		if ignoreUber && engine.IsInvulnerable(i) {
			continue
		}
		if stunnedOnly && !engine.IsPlayerInCondition(i, engine.ConditionDazed()) {
			continue
		}
		if class > engine.ClassUnknown() && engine.PlayerClass(i) != class {
			continue
		}
		if engine.IsStealthed(i) && !engine.IsCloakedPlayerExposed(i) {
			continue
		}
		distance := engine.VectorDistance(engine.WorldSpaceCenter(i), origin)

		if distance <= bestDistance && distance <= maxDistance {
			bestDistance = distance
			bestEntity = i
		}
	}

	return bestEntity
}
