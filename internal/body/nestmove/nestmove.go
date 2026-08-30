/*
Package nestmove is the part of source/redbots3/util.sp that decides when an
engineer should move: away from a buster now, or to better ground between waves.
*/
package nestmove

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// How far around the sentry to look for somewhere out of the blast.
//
//sp:name SENTRY_HAUL_SEARCH_RANGE
const haulSearchRange = 1200.0

/*
PickBusterRetreatArea is ground to carry the sentry to, away from a buster.

Anywhere it ends up has to beat where it stands now by a blast, or it was not
worth moving.
*/
//
//sp:name PickBusterRetreatArea
func PickBusterRetreatArea(sentry int32, buster int32) engine.Area {
	sentryOrigin := engine.AbsOriginOf(sentry)
	busterOrigin := engine.WorldSpaceCenter(buster)

	// Anywhere the sentry ends up has to beat where it stands now by a blast, or it was not worth moving
	bestDistance := engine.VectorDistance(sentryOrigin, busterOrigin) + engine.BusterBlastRange()
	best := engine.NullArea()

	areas := engine.CollectAreasInRadius(sentryOrigin, haulSearchRange)
	defer areas.Close()

	count := areas.Count()

	// One engineer, once per buster, but the count belongs to the map rather than to this
	if count > 256 {
		count = 256
	}

	for i := int32(0); i < count; i++ {
		area := areas.Get(i)

		if area.HasAttributeTF(engine.BlueSpawnRoom()) || area.HasAttributeTF(engine.RedSpawnRoom()) {
			continue
		}

		center := area.Center()

		distance := engine.VectorDistance(center, busterOrigin)

		if distance <= bestDistance {
			continue
		}

		bestDistance = distance
		best = engine.Area(area)
	}

	return best
}

/*
ShouldRelocateNest is whether better ground is worth the walk, asked once per wave.

The gain is the difference between what the candidate scores and what the ground he
holds scores, both against the same sampled approach, so the two numbers are
comparable. A small gain is not worth a sentry in a toolbox.
*/
//
//sp:name ShouldRelocateNest
//sp:default sentryRange 1300.0
func ShouldRelocateNest(client int32, sentryRange float32) (yes bool, destination engine.Area) {
	destination = engine.NullArea()

	current := engine.NestAreaOf(client)

	// No nest yet, so there is nothing to compare against and the ordinary picker will build one
	if current == engine.NullArea() {
		return false, destination
	}

	var target [3]float32

	found, bombinfo := engine.GetBombInfo()

	if found {
		target[0] = bombinfo.Position[0]
		target[1] = bombinfo.Position[1]
		target[2] = bombinfo.Position[2]
	} else {
		target = engine.BombHatchPosition()
	}

	target[2] += 40.0

	candidate := engine.PickBuildAreaRanged(client, sentryRange)

	if candidate == engine.NullArea() || candidate == current {
		return false, destination
	}

	approach := engine.NewList()
	defer approach.Close()

	engine.CollectBombApproachAreas(target, sentryRange, approach)

	gain := engine.ScoreNestArea(client, engine.NavArea(candidate), target, sentryRange, approach) -
		engine.ScoreNestArea(client, engine.NavArea(current), target, sentryRange, approach)

	if engine.ManagerDebug().Bool() {
		engine.PrintToServer("ShouldRelocateNest: %N would gain %.1f by moving", client, gain)
	}

	if gain < engine.NestRelocateScoreGainMin().Float() {
		return false, destination
	}

	destination = candidate

	return true, destination
}
