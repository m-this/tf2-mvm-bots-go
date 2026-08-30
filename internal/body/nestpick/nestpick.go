/*
Package nestpick is the part of source/redbots3/util.sp that turns the map's own
list of nest spots into the areas an engineer may choose between.

The zone rule is here: a spot in a named zone belongs to whichever engineer took
it first, so the second one is offered the rest.
*/
package nestpick

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// CollectConfiguredNestAreas is the nav area under each named spot, skipping the
// spots the mesh does not cover.
//
//sp:name CollectConfiguredNestAreas
func CollectConfiguredNestAreas(spots engine.List, out engine.List) {
	for i := int32(0); i < spots.Length(); i++ {
		spot := spots.GetArray(i)

		area := engine.NearestNavArea(spot, false, 500.0, false, true, engine.TeamAny())

		if area != engine.NullArea() {
			out.Push(int32(area))
		}
	}
}

// IsNestZoneTaken says another engineer is already holding a spot in that zone.
//
//sp:name IsNestZoneTaken
//sp:const zone
func IsNestZoneTaken(client int32, zone engine.Text) bool {
	spots := engine.EngineerNestSpots()
	zones := engine.EngineerNestZones()

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if i == client || !engine.IsClientInGame(i) || engine.NestAreaOf(i) == engine.NullArea() {
			continue
		}

		held := engine.NestAreaOf(i).Center()

		for s := int32(0); s < spots.Length() && s < zones.Length(); s++ {
			other := zones.GetString(s)

			if !engine.StrEqualText(other, zone) {
				continue
			}

			spot := spots.GetArray(s)

			if engine.VectorDistance(held, spot) < engine.NestSpotMatchRange() {
				return true
			}
		}
	}

	return false
}

// CollectZonedNestAreas offers the spots no other engineer's zone has taken, and
// the whole list when that leaves nothing.
//
//sp:name CollectZonedNestAreas
func CollectZonedNestAreas(client int32, out engine.List) {
	spots := engine.EngineerNestSpots()
	zones := engine.EngineerNestZones()

	if spots.Length() == 0 {
		return
	}

	free := engine.NewBlocks(3)
	defer free.Close()

	for i := int32(0); i < spots.Length(); i++ {
		var zone engine.Text

		if i < zones.Length() {
			zone = zones.GetString(i)
		}

		if engine.Feature(engine.FeatureNestZones()) && zone[0] != 0 && IsNestZoneTaken(client, zone) {
			continue
		}

		spot := spots.GetArray(i)
		free.PushArray(spot)
	}

	offered := spots

	if free.Length() > 0 {
		offered = free
	}

	CollectConfiguredNestAreas(offered, out)
}

/*
PickConfiguredNestArea is the best of the ground the map names.

Rottenburg has a spot that only works when a tank is rolling and one that must be
left empty when it is: a sentry parked on the tank's path is a sentry the tank
drives through. Which of the two lists applies is a property of the wave, so it is
asked here rather than baked into the file.
*/
//
//sp:name PickConfiguredNestArea
//sp:const target
func PickConfiguredNestArea(client int32, target [3]float32, sentryRange float32) engine.Area {
	areas := engine.NewList()
	defer areas.Close()

	CollectZonedNestAreas(client, areas)

	if engine.IsTankWave() {
		CollectConfiguredNestAreas(engine.NestTankOnlySpots(), areas)
	} else {
		CollectConfiguredNestAreas(engine.NestNoTankSpots(), areas)
	}

	best := engine.NullArea()

	if areas.Length() > 0 {
		best = engine.BestNestArea(client, areas, target, sentryRange)
	}

	return best
}
