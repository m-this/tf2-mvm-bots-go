/*
Package buildarea is the part of source/redbots3/util.sp that picks the ground an
engineer builds on when the map names none.

The mesh is walked once and every area that survives the filters is dropped into
one of five lists, best first. The lists are the whole of the decision: a nest
forward of the bomb and visible to it beats one that is only forward, which beats
one that is only visible, and the last two are the ones kept so that an engineer
always has somewhere rather than nowhere.
*/
package buildarea

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// How near the hatch a nest may not be: the bomb ends up there and a sentry on
// top of it covers nothing on the way.
//
//sp:name NEST_HATCH_CLEARANCE
const hatchClearance = 180.0

// PickBuildArea is the ground this engineer should hold.
//
//sp:name PickBuildArea
//sp:default sentryRange 1300.0
func PickBuildArea(client int32, sentryRange float32) engine.Area {
	areaCount := engine.NavAreaCount()

	if areaCount <= 0 {
		return engine.NullArea()
	}

	found, bombinfo := engine.GetBombInfo()

	if !found {
		return PickBuildAreaPreRound(client, 1300.0)
	}

	var targetPos [3]float32

	targetPos[0] = bombinfo.Position[0]
	targetPos[1] = bombinfo.Position[1]
	targetPos[2] = bombinfo.Position[2] + 40.0

	configured := engine.PickConfiguredNestArea(client, targetPos, sentryRange)

	if configured != engine.NullArea() {
		return configured
	}

	hinted := engine.PickMapHintNestArea(client, targetPos, sentryRange)

	if hinted != engine.NullArea() {
		return hinted
	}

	bombArea := engine.NavArea(engine.NearestNavArea(targetPos, false, 90000.0, false, true, engine.TeamAny()))

	if bombArea == engine.NavArea(engine.NullArea()) {
		return engine.NullArea()
	}

	if bombArea.HasAttributeTF(engine.BlueSpawnRoom()) || bombArea.HasAttributeTF(engine.RedSpawnRoom()) {
		return engine.NullArea()
	}

	// Areas forward of the bomb within some distance and visible to bomb.
	forwardVisibleAreas := engine.NewList()
	defer forwardVisibleAreas.Close()

	// Areas forward of the bomb but not necessarily visible.
	forwardAreas := engine.NewList()
	defer forwardAreas.Close()

	// Areas visible to the bomb but not nescessarily forward of it.
	visibleAreasAround := engine.NewList()
	defer visibleAreasAround.Close()

	// Any of the above, but further up the path than an engineer should nest.
	areasTooFarUp := engine.NewList()
	defer areasTooFarUp.Close()

	// On top of the bomb, which is a nest only when the map offers nothing else.
	areasTooClose := engine.NewList()
	defer areasTooClose.Close()

	limit := engine.NestDistanceLimit()

	for i := int32(0); i < areaCount; i++ {
		area := engine.AllNavAreas().NavAreaAt(i)

		if area == engine.NavArea(engine.NullArea()) {
			continue
		}

		// Area in spawn
		if area.HasAttributeTF(engine.BlueSpawnRoom()) || area.HasAttributeTF(engine.RedSpawnRoom()) {
			continue
		}

		/* BLOCKED is the one nav attribute that changes during a mission: gates and
		func_nav_blocker set it. PickBuildAreaPreRound has always checked it and this one never
		did, so a nest picked after a gate closed could sit on ground the mesh calls unreachable */
		if area.HasAttributeTF(engine.AttributeBlocked()) {
			continue
		}

		// TODO
		// Better solution because this will break on all non mvm maps.
		// Most likely areachable area
		if !area.HasAttributeTF(engine.AttributeBombDrop()) {
			continue
		}

		bombTargetDistanceAtArea := engine.TravelDistanceToBombTarget(engine.Area(area))
		bombTargetDistanceAtBomb := engine.TravelDistanceToBombTarget(engine.Area(bombArea))

		if bombTargetDistanceAtArea < hatchClearance {
			continue
		}

		/* Further up the path than an engineer nests. Kept, because the bomb spends the start of
		every wave up there and this is where the forward lists would otherwise be empty: better a
		nest too far forward than an engineer that never builds one */
		if limit > 0.0 && bombTargetDistanceAtArea > limit {
			areasTooFarUp.Push(int32(area))
			continue
		}

		areaCenter := area.Center()
		areaCenter[2] += 50.0

		areaDistanceToBomb := engine.VectorDistance(areaCenter, targetPos)

		if areaDistanceToBomb >= sentryRange {
			continue
		}

		/* Close enough to the bomb that the sentry never uses its range
		Kept rather than dropped, and kept last: a nest on top of the bomb is bad and no nest at
		all is worse, and a map whose every area near the bomb is this close is a map where the
		engineer would otherwise stand around with 300 metal */
		if !engine.IsNestRangeSane(areaDistanceToBomb, sentryRange) {
			areasTooClose.Push(int32(area))
			continue
		}

		areaVisibleToBomb := area.IsEntirelyVisible(targetPos)

		if areaVisibleToBomb {
			visibleAreasAround.Push(int32(area))
		}

		if bombTargetDistanceAtBomb > bombTargetDistanceAtArea {
			if areaDistanceToBomb <= sentryRange*engine.RandomFloat(0.8, 1.75) && areaVisibleToBomb {
				forwardVisibleAreas.Push(int32(area))
			}

			forwardAreas.Push(int32(area))
		}
	}

	randomArea := engine.NullArea()

	//nolint:gocritic // ifElseChain: the shipped file is this chain of tiers, best first, and the port keeps its shape
	if forwardVisibleAreas.Length() > 0 {
		randomArea = engine.BestNestArea(client, forwardVisibleAreas, targetPos, sentryRange)
	} else if forwardAreas.Length() > 0 {
		randomArea = engine.BestNestArea(client, forwardAreas, targetPos, sentryRange)
	} else if visibleAreasAround.Length() > 0 {
		randomArea = engine.BestNestArea(client, visibleAreasAround, targetPos, sentryRange)
	} else if areasTooFarUp.Length() > 0 {
		randomArea = engine.BestNestArea(client, areasTooFarUp, targetPos, sentryRange)
	} else if areasTooClose.Length() > 0 {
		randomArea = engine.BestNestArea(client, areasTooClose, targetPos, sentryRange)
	}

	if engine.ManagerDebug().Bool() {
		engine.PrintToServer("PickBuildArea %i ForwardVisibleAreas | %i ForwardAreas | %i VisibleAreasAroundBomb | %i AreasTooFarUp | %i AreasTooClose",
			forwardVisibleAreas.Length(), forwardAreas.Length(), visibleAreasAround.Length(), areasTooFarUp.Length(), areasTooClose.Length())
	}

	return randomArea
}

/*
PickBuildAreaPreRound is the same question before a wave starts, when there is no
bomb to measure from.

The hatch stands in for it: that is where the bomb ends up, so ground that covers
the hatch is ground that will matter. The tiers are one shorter than the wave-time
ones, because "forward of the bomb" has no meaning yet.
*/
//
//sp:name PickBuildAreaPreRound
//sp:default sentryRange 1300.0
func PickBuildAreaPreRound(client int32, sentryRange float32) engine.Area {
	areaCount := engine.NavAreaCount()

	if areaCount <= 0 {
		return engine.NullArea()
	}

	limit := engine.NestDistanceLimit()

	hatch := engine.BombHatchPosition()
	hatch[2] += 40.0

	configured := engine.PickConfiguredNestArea(client, hatch, sentryRange)

	if configured != engine.NullArea() {
		return configured
	}

	hinted := engine.PickMapHintNestArea(client, hatch, sentryRange)

	if hinted != engine.NullArea() {
		return hinted
	}

	// Near enough to the hatch to nest, and with a line to it
	coveringAreas := engine.NewList()
	defer coveringAreas.Close()

	// Near enough to nest, seeing the hatch or not
	nestingAreas := engine.NewList()
	defer nestingAreas.Close()

	// On the path, but further up it than an engineer should nest
	areasTooFarUp := engine.NewList()
	defer areasTooFarUp.Close()

	// On top of the hatch, which is a nest only when the map offers nothing else
	areasTooClose := engine.NewList()
	defer areasTooClose.Close()

	for i := int32(0); i < areaCount; i++ {
		area := engine.AllNavAreas().NavAreaAt(i)

		if area == engine.NavArea(engine.NullArea()) {
			continue
		}

		if area.HasAttributeTF(engine.BlueSpawnRoom()) || area.HasAttributeTF(engine.RedSpawnRoom()) {
			continue
		}

		if area.HasAttributeTF(engine.AttributeBlocked()) {
			continue
		}

		// TODO
		// Better solution because this will break on all non mvm maps.
		if !area.HasAttributeTF(engine.AttributeBombDrop()) {
			continue
		}

		distance := engine.TravelDistanceToBombTarget(engine.Area(area))

		if distance < hatchClearance {
			continue
		}

		if limit > 0.0 && distance > limit {
			areasTooFarUp.Push(int32(area))
			continue
		}

		center := area.Center()
		center[2] += 50.0

		/* Sitting on the hatch is not nesting, whichever tier the area would have landed in
		The clearance above is a travel distance along the bomb path and says nothing about a
		ledge directly over the hatch, which is a short walk and no distance at all */
		if !engine.IsNestRangeSane(engine.VectorDistance(center, hatch), sentryRange) {
			areasTooClose.Push(int32(area))
			continue
		}

		nestingAreas.Push(int32(area))

		if area.IsEntirelyVisible(hatch) {
			coveringAreas.Push(int32(area))
		}
	}

	bestArea := engine.NullArea()

	//nolint:gocritic // ifElseChain: the shipped file is this chain of tiers, best first, and the port keeps its shape
	if coveringAreas.Length() > 0 {
		bestArea = engine.BestNestArea(client, coveringAreas, hatch, sentryRange)
	} else if nestingAreas.Length() > 0 {
		bestArea = engine.BestNestArea(client, nestingAreas, hatch, sentryRange)
	} else if areasTooFarUp.Length() > 0 {
		bestArea = engine.BestNestArea(client, areasTooFarUp, hatch, sentryRange)
	} else if areasTooClose.Length() > 0 {
		bestArea = engine.BestNestArea(client, areasTooClose, hatch, sentryRange)
	}

	if engine.ManagerDebug().Bool() {
		engine.PrintToServer("PickBuildAreaPreRound %i CoveringAreas | %i NestingAreas | %i AreasTooFarUp | %i AreasTooClose",
			coveringAreas.Length(), nestingAreas.Length(), areasTooFarUp.Length(), areasTooClose.Length())
	}

	return bestArea
}
