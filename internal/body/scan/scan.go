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
//sp:name GetNearestEnemyCount
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
		if engine.VectorDistance(WorldSpaceCenter(i), origin) <= maxDistance {
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
//sp:name FindEnemyNearestToMe
func EnemyNearestToMe(client int32, maxDistance float32, giantsOnly bool, ignoreUber bool, stunnedOnly bool, class engine.Class) int32 {
	origin := WorldSpaceCenter(client)

	bestDistance := float32(999999.0)
	bestEntity := int32(-1)
	enemyTeam := PlayerEnemyTeam(client)

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
		distance := engine.VectorDistance(WorldSpaceCenter(i), origin)

		if distance <= bestDistance && distance <= maxDistance {
			bestDistance = distance
			bestEntity = i
		}
	}

	return bestEntity
}

// BestTargetForSpy is util.sp:1235, GetBestTargetForSpy: the four passes a spy
// makes over the enemy team, and then the healer behind whoever it found.
//
//sp:name GetBestTargetForSpy
func BestTargetForSpy(client int32, maxDistance float32) int32 {
	// The shipped code writes this and overwrites it on the next line. It is
	// dead there too, and it stays, because the port is behaviour identical
	// and a tidy that rides along cannot be told from one that is not.
	target := int32(-1) //nolint:ineffassign,wastedassign // util.sp:1237, kept as shipped

	// Find the nearest enemy engineer
	target = EnemyNearestToMe(client, maxDistance, false, true, false, engine.ClassEngineer())

	// Find the nearest stunned enemy
	if target == -1 {
		target = EnemyNearestToMe(client, maxDistance, false, true, true, engine.ClassUnknown())
	}

	// Find the nearest enemy giant
	if target == -1 {
		target = EnemyNearestToMe(client, maxDistance, true, true, false, engine.ClassUnknown())
	}

	// Find the nearest enemy
	if target == -1 {
		target = EnemyNearestToMe(client, maxDistance, false, true, false, engine.ClassUnknown())
	}

	// Target their healer first, if they have one
	if target != -1 {
		myTeam := engine.GetClientTeam(client)

		for i := int32(0); i < engine.NumHealers(target); i++ {
			healer := engine.PlayerHealer(target, i)

			if healer != -1 && engine.IsPlayer(healer) && engine.GetClientTeam(healer) != myTeam {
				target = healer
				break
			}
		}
	}

	return target
}
