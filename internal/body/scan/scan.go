/*
Package scan is util.sp's client loop.

util.sp 1183-1690 held nine variants of one scan: walk the client slots or the
entities of one classname, filter, measure a distance, keep the best. They came
across one at a time and were collapsed once every one was here and its
behaviour pinned: nearestClient and nearestObject are the two loops, and each
variant is the questions it asks, as named predicates. Which questions, and in
which order, is exactly what the shipped loops asked: internal/body's
TestScanTracesArePinned holds every answer and every engine call.

The bug they share, mvm-ds3, is not fixed here. Every one of them loops player
slots and a tank occupies none, and that stays true.
*/
package scan

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Kind is which of the shipped loops a scan is: the five questions every
// client scan asks come first, and the kind is what it asks after them.
type Kind int32

const (
	// KindEnemy is FindEnemyNearestToMe's filters.
	KindEnemy Kind = 0
	// KindSappable is the spy's questions.
	KindSappable Kind = 1
	// KindSappableHealing is the spy's questions behind the medigun one.
	KindSappableHealing Kind = 2
	// KindAnyEnemy asks nothing more.
	KindAnyEnemy Kind = 3
)

// NoDistance is what the nearest search starts from, and the farthest search
// ends under: the plugin's own sentinel.
//
//sp:name SCAN_NO_DISTANCE
const NoDistance = 999999.0

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

	return nearestClient(client, origin, maxDistance, KindEnemy, false, giantsOnly, ignoreUber, stunnedOnly, class, 0.0)
}

// EnemyPlayerNearestToPosition is util.sp:1550,
// GetEnemyPlayerNearestToPosition: the shortest of the loops, measuring from a
// position the caller supplies rather than from where the client stands.
//
//sp:name GetEnemyPlayerNearestToPosition
func EnemyPlayerNearestToPosition(client int32, position [3]float32, maxDistance float32) int32 {
	return nearestClient(client, position, maxDistance, KindAnyEnemy, false, false, false, false, engine.ClassUnknown(), 0.0)
}

/*
nearestClient is the one client loop: the enemy nearest to origin within
maxDistance, or the farthest, that the kind's questions accept.

The five questions every variant asked come first, then the kind's own, then
the distance. A candidate the kind refuses is never measured, which is the
order the shipped loops had and the order the pinned traces hold.
*/
func nearestClient(client int32, origin [3]float32, maxDistance float32, kind Kind, farthest bool,
	giantsOnly bool, ignoreUber bool, stunnedOnly bool, class engine.Class, speedCheck float32,
) int32 {
	enemyTeam := PlayerEnemyTeam(client)
	bestDistance := engine.ChooseFloat(farthest, 0.0, NoDistance)
	bestEntity := int32(-1)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if !enemyCandidate(client, i, enemyTeam) {
			continue
		}
		if !wantedBy(kind, i, giantsOnly, ignoreUber, stunnedOnly, class, speedCheck) {
			continue
		}
		distance := engine.VectorDistance(WorldSpaceCenter(i), origin)

		if distance > maxDistance {
			continue
		}
		if farthest && distance < bestDistance {
			continue
		}
		if !farthest && distance > bestDistance {
			continue
		}
		bestDistance = distance
		bestEntity = i
	}

	return bestEntity
}

// enemyCandidate is the five questions every client scan asks first, in the
// order it asks them: another slot, in the game, alive, on the other team, and
// not a sentry buster.
func enemyCandidate(client int32, i int32, enemyTeam engine.Team) bool {
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
	return !engine.IsSentryBusterRobot(i)
}

// wantedBy is what a kind asks after the five, in its own order.
func wantedBy(kind Kind, i int32, giantsOnly bool, ignoreUber bool, stunnedOnly bool, class engine.Class, speedCheck float32) bool {
	switch kind {
	case KindEnemy:
		return enemyWanted(i, giantsOnly, ignoreUber, stunnedOnly, class)
	case KindSappable:
		return sappableWanted(i, giantsOnly, class, speedCheck) && PlayerSappable(i)
	case KindSappableHealing:
		return sappableWanted(i, giantsOnly, class, speedCheck) && PlayerHealingSomething(i) && PlayerSappable(i)
	}
	return true
}

// enemyWanted is FindEnemyNearestToMe's four switches and the cloak question.
func enemyWanted(i int32, giantsOnly bool, ignoreUber bool, stunnedOnly bool, class engine.Class) bool {
	if giantsOnly && !engine.IsMiniBoss(i) {
		return false
	}
	if ignoreUber && engine.IsInvulnerable(i) {
		return false
	}
	if stunnedOnly && !engine.IsPlayerInCondition(i, engine.ConditionDazed()) {
		return false
	}
	if class > engine.ClassUnknown() && engine.PlayerClass(i) != class {
		return false
	}
	return !engine.IsStealthed(i) || engine.IsCloakedPlayerExposed(i)
}

// BestTargetForSpy is util.sp:1235, GetBestTargetForSpy: the four passes a spy
// makes over the enemy team, and then the healer behind whoever it found.
//
//sp:name GetBestTargetForSpy
func BestTargetForSpy(client int32, maxDistance float32) int32 {
	// Find the nearest enemy engineer
	target := EnemyNearestToMe(client, maxDistance, false, true, false, engine.ClassEngineer())

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
