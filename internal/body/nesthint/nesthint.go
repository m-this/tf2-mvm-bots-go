/*
Package nesthint is the part of source/redbots3/util.sp that reads the nest spots
the map itself carries, and the two distances a nest is judged against.

A map's own bot_hint_sentrygun entities are what Valve put there for the game's
engineer, and they are better ground than anything this mod can score from the mesh
alone. They are collected once per map and kept.
*/
package nesthint

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// More hint entities than any map has, which is the point: past this the entity
// list is not what we think it is.
//
//sp:name MAX_MAP_HINT_NESTS
const maxMapHintNests = 64

// How near the bomb a nest may be before it is inside the fight rather than
// covering it.
//
//sp:name NEST_MIN_BOMB_RANGE_FRACTION
const minBombRangeFraction = 0.34

var (
	//sp:name g_adtMapHintNests
	mapHintNests engine.List
	//sp:name g_bMapHintNestsLoaded
	mapHintNestsLoaded bool
)

// ResetMapHintNests forgets them, which a map change has to do.
//
//sp:name ResetMapHintNests
func ResetMapHintNests() {
	mapHintNestsLoaded = false

	if mapHintNests != engine.NoList() {
		mapHintNests.Clear()
	}
}

// CollectMapHintNests adds every entity of that class the map placed.
//
//sp:name CollectMapHintNests
func CollectMapHintNests(classname string) {
	entity := int32(-1)

	for {
		entity = engine.FindEntityByClassname(entity, classname)

		if entity == -1 {
			break
		}

		if mapHintNests.Length() >= maxMapHintNests {
			return
		}

		origin := engine.AbsOriginOf(entity)

		if engine.IsZeroVector(origin) {
			continue
		}

		mapHintNests.PushArray(origin)
	}
}

// MapHintNests is the list, collected on the first ask and kept after that.
//
//sp:name MapHintNests
//sp:borrowed
func MapHintNests() engine.List {
	if mapHintNests == engine.NoList() {
		mapHintNests = engine.NewBlocks(3)
	}

	if mapHintNestsLoaded {
		return mapHintNests
	}

	mapHintNestsLoaded = true

	CollectMapHintNests("bot_hint_sentrygun")
	CollectMapHintNests("bot_hint_engineer_nest")

	if engine.ManagerDebug().Bool() {
		engine.PrintToServer("MapHintNests: %d nest spots from the map's own entities", mapHintNests.Length())
	}

	return mapHintNests
}

// PickMapHintNestArea is the best of the ground the map's own entities name.
//
//sp:name PickMapHintNestArea
//sp:const target
func PickMapHintNestArea(client int32, target [3]float32, sentryRange float32) engine.Area {
	spots := MapHintNests()

	if spots.Length() == 0 {
		return engine.NullArea()
	}

	areas := engine.NewList()
	defer areas.Close()

	for i := int32(0); i < spots.Length(); i++ {
		spot := spots.GetArray(i)

		area := engine.NearestNavArea(spot, false, 500.0, false, true, engine.TeamAny())

		if area == engine.NullArea() {
			continue
		}

		tfArea := engine.NavArea(area)

		// A spot inside either spawn room is one the engineer cannot hold, whoever it was put there for
		if tfArea.HasAttributeTF(engine.BlueSpawnRoom()) || tfArea.HasAttributeTF(engine.RedSpawnRoom()) {
			continue
		}

		areas.Push(int32(area))
	}

	best := engine.BestNestArea(client, areas, target, sentryRange)

	return best
}

// IsNestRangeSane says the ground is far enough from the bomb to shoot at it and
// near enough to reach it.
//
//sp:name IsNestRangeSane
func IsNestRangeSane(rangeToBomb float32, sentryRange float32) bool {
	return rangeToBomb >= sentryRange*minBombRangeFraction && rangeToBomb < sentryRange
}

// BombPathLength is the longest route to the bomb's target anywhere on the mesh,
// which is what a nest's depth is measured against.
//
//sp:name BombPathLength
func BombPathLength() float32 {
	areaCount := engine.NavAreaCount()
	longest := float32(0.0)

	for i := int32(0); i < areaCount; i++ {
		area := engine.AllNavAreas().NavAreaAt(i)

		if area == engine.NavArea(engine.NullArea()) {
			continue
		}

		if area.HasAttributeTF(engine.BlueSpawnRoom()) || area.HasAttributeTF(engine.RedSpawnRoom()) {
			continue
		}

		longest = engine.MaxFloat(longest, engine.TravelDistanceToBombTarget(area))
	}

	return longest
}

// NestDistanceLimit is how far along that route an engineer may hold ground.
//
//sp:name NestDistanceLimit
func NestDistanceLimit() float32 {
	length := BombPathLength()

	if length <= 0.0 {
		return 0.0
	}

	return length * engine.NestDepth().Float()
}
