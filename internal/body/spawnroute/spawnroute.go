/*
Package spawnroute is the part of source/redbots3/util.sp that reads the way out
of spawn, which is where a teleporter entrance belongs.

A route rather than a spot: an entrance goes where a player leaving spawn walks
into it, so the points are sampled along the path the nav mesh itself would take.
*/
package spawnroute

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// NearestSpawnPoint is where this bot's team respawns, and false when the map
// names no such thing.
//
//sp:name NearestSpawnPoint
func NearestSpawnPoint(actor int32) (found bool, spawn [3]float32) {
	point := int32(-1)
	nearest := int32(-1)
	nearestRange := float32(-1.0)

	for {
		point = engine.FindEntityByClassname(point, "info_player_teamspawn")

		if point == -1 {
			break
		}

		if engine.EntProp(point, engine.PropData(), "m_iTeamNum") != int32(engine.TeamRed()) {
			continue
		}

		origin := engine.EntPropVector(point, engine.PropData(), "m_vecAbsOrigin")

		pointRange := engine.VectorDistance(engine.AbsOriginOf(actor), origin)

		if nearestRange < 0.0 || pointRange < nearestRange {
			nearestRange = pointRange
			nearest = point
		}
	}

	if nearest == -1 {
		return false, spawn
	}

	spawn = engine.EntPropVector(nearest, engine.PropData(), "m_vecAbsOrigin")

	return true, spawn
}

/*
Points samples the way out of spawn: where a teleporter goes, and where
the man stands to put it there, one pair per step along the route.

Returns how many points the route was long enough for, which is none when there is
no route at all and none when the engineer is already inside the spawn room.
*/
//
//sp:name SpawnRoutePoints
//sp:const spawn
//sp:mutates spots
//sp:mutates stands
func Points(actor int32, spawn [3]float32, first float32, step float32, reach float32,
	spots [8][3]float32, stands [8][3]float32, pointsMax int32,
) int32 {
	engine.CombatOf(actor).UpdateLastKnownArea()

	route := engine.NewRoute(engine.FilterIgnoreActors(), engine.FilterOnlyActors())
	defer route.Close()

	found := int32(0)

	if route.Compute(engine.NextBotOf(actor), spawn) {
		length := route.Length()

		for i := int32(0); i < pointsMax; i++ {
			fromSpawn := first + step*float32(i)

			// The route runs out, and a point past the far end of it is not on the way out of spawn
			if length <= fromSpawn+reach {
				break
			}

			spots[i] = route.PositionAlong(length - fromSpawn)
			stands[i] = route.PositionAlong(length - fromSpawn - reach)

			found++
		}
	}

	return found
}

/*
Out samples the same way out of spawn from its near end: the route from where he
stands, in spawn, to the nest. The spot is first + step*i along it, and the stand
is a build's reach short of the spot on the spawn side, which is the side he
arrives from when the entrance goes up before the nest.

Returns how many points the route was long enough for, which is none when there is
no route to the nest.
*/
//
//sp:name SpawnRouteOut
//sp:const nest
//sp:mutates spots
//sp:mutates stands
func Out(actor int32, nest [3]float32, first float32, step float32, reach float32,
	spots [8][3]float32, stands [8][3]float32, pointsMax int32,
) int32 {
	engine.CombatOf(actor).UpdateLastKnownArea()

	route := engine.NewRoute(engine.FilterIgnoreActors(), engine.FilterOnlyActors())
	defer route.Close()

	found := int32(0)

	if route.Compute(engine.NextBotOf(actor), nest) {
		length := route.Length()

		for i := int32(0); i < pointsMax; i++ {
			fromSpawn := first + step*float32(i)

			// A point past the nest is not on the way out of spawn
			if length <= fromSpawn {
				break
			}

			spots[i] = route.PositionAlong(fromSpawn)
			stands[i] = route.PositionAlong(fromSpawn - reach)

			found++
		}
	}

	return found
}
