package scan

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// The scans over entities rather than clients: the same loop with
// FindEntityByClassname in place of the slot range, and the same shape of bug:
// the walk ends when the engine says there is nothing more, and nothing bounds
// it otherwise.

// ObjectKind is which entity scan this is: what is walked, and what is asked of
// each one.
//
//sp:name ScanObjectKind
type ObjectKind int32

const (
	// ObjectSappable is any enemy building a spy could sap.
	ObjectSappable ObjectKind = 0
	// ObjectTeleporter is an enemy teleporter.
	ObjectTeleporter ObjectKind = 1
	// ObjectCurrency is money on the ground.
	ObjectCurrency ObjectKind = 2
)

// NearestSappableObject is util.sp:1325, GetNearestSappableObject: the closest
// enemy building a spy could sap.
//
//sp:default maxDistance 1000.0
//sp:name GetNearestSappableObject
func NearestSappableObject(client int32, maxDistance float32) int32 {
	origin := engine.Origin(client)
	myTeam := engine.GetClientTeam(client)

	return nearestObject(origin, myTeam, maxDistance, "obj_*", ObjectSappable)
}

// NearestEnemyTeleporter is util.sp:1363, GetNearestEnemyTeleporter.
//
//sp:default maxDistance 999999.0
//sp:name GetNearestEnemyTeleporter
func NearestEnemyTeleporter(client int32, maxDistance float32) int32 {
	origin := engine.Origin(client)
	myTeam := engine.GetClientTeam(client)

	return nearestObject(origin, myTeam, maxDistance, "obj_teleporter", ObjectTeleporter)
}

// NearestCurrencyPack is util.sp:1358, GetNearestCurrencyPack: the closest
// money the bot could pick up. Money has no team, so none is asked for.
//
//sp:default maxDistance 999999.0
//sp:name GetNearestCurrencyPack
func NearestCurrencyPack(client int32, maxDistance float32) int32 {
	origin := engine.Origin(client)

	return nearestObject(origin, 0, maxDistance, "item_currency*", ObjectCurrency)
}

// nearestObject is the one entity loop: the closest entity of the classname,
// within maxDistance, that the kind's questions accept.
func nearestObject(origin [3]float32, myTeam int32, maxDistance float32, classname string, kind ObjectKind) int32 {
	bestDistance := float32(NoDistance)
	bestEnt := int32(-1)
	ent := int32(-1)

	for {
		ent = engine.FindEntityByClassname(ent, classname)
		if ent == -1 {
			break
		}
		if !objectWanted(kind, ent, myTeam) {
			continue
		}
		distance := engine.VectorDistance(origin, AbsOrigin(ent))

		if distance <= bestDistance && distance <= maxDistance {
			bestDistance = distance
			bestEnt = ent
		}
	}

	return bestEnt
}

// objectWanted is what each entity scan asks, in the order it asked it.
func objectWanted(kind ObjectKind, ent int32, myTeam int32) bool {
	switch kind {
	case ObjectSappable:
		return engine.ObjectType(ent) != engine.ObjectSapper() && buildingSappable(ent, myTeam)
	case ObjectTeleporter:
		return buildingSappable(ent, myTeam)
	}
	// This pack has already been distributed to the team, or has not reached
	// the ground yet.
	return engine.EntProp(ent, engine.PropSend(), "m_bDistributed") != 1 &&
		engine.EntityFlags(ent)&engine.FlagOnGround() != 0
}

// buildingSappable is an enemy building standing on its own with no sapper on
// it yet.
func buildingSappable(ent int32, myTeam int32) bool {
	if engine.EntityTeamNumber(ent) == myTeam {
		return false
	}
	if engine.IsPlacing(ent) {
		return false
	}
	if engine.IsCarried(ent) {
		return false
	}
	return !engine.HasSapper(ent)
}
