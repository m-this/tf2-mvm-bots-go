package scan

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// The two scans over buildings rather than clients. They are the same loop as
// the client ones with FindEntityByClassname in place of the slot range, and
// they carry the same shape of bug: the walk ends when the engine says there is
// nothing more, and nothing bounds it otherwise.

// NearestSappableObject is util.sp:1325, GetNearestSappableObject: the closest
// enemy building a spy could sap.
//
//sp:default maxDistance 1000.0
func NearestSappableObject(client int32, maxDistance float32) int32 {
	origin := engine.Origin(client)
	myTeam := engine.GetClientTeam(client)

	bestDistance := float32(999999.0)
	bestEnt := int32(-1)

	ent := int32(-1)
	for {
		ent = engine.FindEntityByClassname(ent, "obj_*")
		if ent == -1 {
			break
		}
		if engine.ObjectType(ent) == engine.ObjectSapper() {
			continue
		}
		if engine.EntityTeamNumber(ent) == myTeam {
			continue
		}
		if engine.IsPlacing(ent) {
			continue
		}
		if engine.IsCarried(ent) {
			continue
		}
		if engine.HasSapper(ent) {
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

// NearestEnemyTeleporter is util.sp:1363, GetNearestEnemyTeleporter.
//
//sp:default maxDistance 999999.0
func NearestEnemyTeleporter(client int32, maxDistance float32) int32 {
	origin := engine.Origin(client)
	myTeam := engine.GetClientTeam(client)

	bestDistance := float32(999999.0)
	bestEnt := int32(-1)
	ent := int32(-1)

	for {
		ent = engine.FindEntityByClassname(ent, "obj_teleporter")
		if ent == -1 {
			break
		}
		if engine.EntityTeamNumber(ent) == myTeam {
			continue
		}
		if engine.IsPlacing(ent) {
			continue
		}
		if engine.IsCarried(ent) {
			continue
		}
		if engine.HasSapper(ent) {
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
