package scan

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// The spy's scans: the client loop with his questions after the five, and the
// only thing that separates the nearest from the farthest is which way the
// comparison points and what it starts at.

// PlayerSappable is util.sp:1437, IsPlayerSappable.
//
//sp:name IsPlayerSappable
func PlayerSappable(client int32) bool {
	if engine.IsPlayerInCondition(client, engine.ConditionSapped()) {
		return false
	}
	if engine.IsInvulnerable(client) {
		return false
	}
	return !engine.IsPlayerInCondition(client, engine.ConditionBonked())
}

// PlayerHealingSomething is util.sp:1690, IsPlayerHealingSomething.
//
//sp:name IsPlayerHealingSomething
func PlayerHealingSomething(client int32) bool {
	weapon := engine.ActiveWeapon(client)

	if weapon == -1 {
		return false
	}

	return engine.WeaponID(weapon) == engine.WeaponMedigun() &&
		engine.EntPropEnt(weapon, engine.PropSend(), "m_hHealingTarget") != -1
}

// sappableWanted is the three switches the spy's callers vary: giants only, one
// class, and a floor on speed.
func sappableWanted(i int32, giantsOnly bool, class engine.Class, speedCheck float32) bool {
	if giantsOnly && !engine.IsMiniBoss(i) {
		return false
	}
	if class > engine.ClassUnknown() && engine.PlayerClass(i) != class {
		return false
	}
	// Not fast enough
	return speedCheck <= 0.0 || engine.EntPropFloat(i, engine.PropSend(), "m_flMaxspeed") >= speedCheck
}

// NearestSappablePlayer is util.sp:1451, GetNearestSappablePlayer.
//
//sp:default giantsOnly false
//sp:default class TFClass_Unknown
//sp:default speedCheck 0.0
//sp:name GetNearestSappablePlayer
func NearestSappablePlayer(client int32, maxDistance float32, giantsOnly bool, class engine.Class, speedCheck float32) int32 {
	origin := engine.Origin(client)

	return nearestClient(client, origin, maxDistance, KindSappable, false, giantsOnly, false, false, class, speedCheck)
}

// FarthestSappablePlayer is util.sp:1501, GetFarthestSappablePlayer. Same loop,
// starting at zero and keeping the largest.
//
//sp:default giantsOnly false
//sp:default class TFClass_Unknown
//sp:default speedCheck 0.0
//sp:name GetFarthestSappablePlayer
func FarthestSappablePlayer(client int32, maxDistance float32, giantsOnly bool, class engine.Class, speedCheck float32) int32 {
	origin := engine.Origin(client)

	return nearestClient(client, origin, maxDistance, KindSappable, true, giantsOnly, false, false, class, speedCheck)
}

// NearestSappablePlayerHealingSomeone is util.sp:1638. The same loop again with
// the medigun question in front of the sappable one, in that order, because the
// order is what the engine call trace records.
//
//sp:default giantsOnly false
//sp:default class TFClass_Unknown
//sp:default speedCheck 0.0
//sp:name GetNearestSappablePlayerHealingSomeone
func NearestSappablePlayerHealingSomeone(client int32, maxDistance float32, giantsOnly bool, class engine.Class, speedCheck float32) int32 {
	origin := engine.Origin(client)

	return nearestClient(client, origin, maxDistance, KindSappableHealing, false, giantsOnly, false, false, class, speedCheck)
}
