/*
Package nestscore is the part of source/redbots3/util.sp that decides which
ground an engineer should hold.

A nest is worth having for three things and they are scored separately: the range
to what the sentry is meant to cover, how much of the ground the robots cross it
can see, and whether another engineer is already there.
*/
package nestscore

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// How near two nests may be before the second one is worth less.
//
//sp:name NEST_SPACING
const nestSpacing = 500.0

// How many pieces of the approach are sampled, and what seeing all of them is
// worth.
const (
	//sp:name MAX_APPROACH_SAMPLES
	maxApproachSamples = 24
	//sp:name NEST_SIGHT_SCORE
	nestSightScore = 80.0
)

/*
CollectBombApproachAreas is the ground the robots cross to reach the target,
sampled rather than walked.

The mesh around a nest holds hundreds of areas and the sight test is a trace per
pair, so the whole list is a frame the watchdog kills. A stride over it is the same
shape of answer for a fixed price.
*/
//
//sp:name CollectBombApproachAreas
//sp:const target
func CollectBombApproachAreas(target [3]float32, sentryRange float32, out engine.List) {
	areas := engine.CollectAreasInRadius(target, sentryRange)
	defer areas.Close()

	count := areas.Count()

	stride := int32(1)

	if count > maxApproachSamples {
		stride = count / maxApproachSamples
	}

	for i := int32(0); i < count && out.Length() < maxApproachSamples; i += stride {
		area := areas.Get(i)

		if !area.HasAttributeTF(engine.AttributeBombDrop()) {
			continue
		}

		if area.HasAttributeTF(engine.BlueSpawnRoom()) || area.HasAttributeTF(engine.RedSpawnRoom()) {
			continue
		}

		out.Push(int32(area))
	}
}

// NestSightScore is how much of the sampled approach this area can see, as a
// share of the whole.
//
//sp:name NestSightScore
func NestSightScore(area engine.NavArea, approach engine.List) float32 {
	if approach == engine.NoList() || approach.Length() == 0 {
		return 0.0
	}

	seen := int32(0)

	for i := int32(0); i < approach.Length(); i++ {
		if area.IsCompletelyVisible(engine.Area(approach.Get(i))) {
			seen++
		}
	}

	return (float32(seen) / float32(approach.Length())) * nestSightScore
}

/*
ScoreNestArea is what one piece of ground is worth to this engineer.

The range term is a distance from ideal rather than a distance: a sentry too close
to the bomb is as wrong as one too far, and the ideal is nearer for a Gunslinger,
whose mini is cheap enough to put in the fight.
*/
//
//sp:name ScoreNestArea
//sp:const target
//sp:default approach null
func ScoreNestArea(client int32, area engine.NavArea, target [3]float32, sentryRange float32, approach engine.List) float32 {
	disposable := engine.GunslingerEquipped(client)

	center := area.Center()
	center[2] += 50.0

	areaRange := engine.VectorDistance(center, target)

	ideal := sentryRange * 0.75

	if disposable {
		ideal = sentryRange * 0.35
	}

	score := 100.0 - (engine.FloatAbs(areaRange-ideal)/sentryRange)*100.0

	if !disposable {
		height := center[2] - target[2]

		if height > 0.0 {
			score += engine.MinFloat(height, 300.0) * 0.1
		}
	}

	score += engine.MinFloat(area.SizeX(), area.SizeY()) * 0.05

	score += NestSightScore(area, approach)

	score += NestCrowdingPenalty(client, area, center)

	return score
}

// NestCrowdingPenalty is what another engineer already standing here costs.
//
//sp:name NestCrowdingPenalty
//sp:const center
func NestCrowdingPenalty(client int32, area engine.NavArea, center [3]float32) float32 {
	penalty := float32(0.0)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if i == client || !engine.IsClientInGame(i) {
			continue
		}

		if engine.NestAreaOf(i) == engine.Area(area) {
			penalty -= 100.0
		} else if engine.NestAreaOf(i) != engine.NullArea() {
			other := engine.NestAreaOf(i).Center()

			if engine.VectorDistance(center, other) < nestSpacing {
				penalty -= 50.0
			}
		}

		sentry := engine.ObjectOfType(i, engine.ObjectSentry())

		if sentry != engine.InvalidEntReference() && engine.VectorDistance(center, engine.AbsOriginOf(sentry)) < nestSpacing {
			penalty -= 100.0
		}
	}

	return penalty
}

// BestNestArea is the highest scoring of the areas offered, with the approach
// sampled once for the whole list rather than per area.
//
//sp:name BestNestArea
//sp:const target
func BestNestArea(client int32, areas engine.List, target [3]float32, sentryRange float32) engine.Area {
	best := engine.NullArea()
	bestScore := float32(0.0)

	// The ground the robots cross to reach the target, sampled once for the whole list
	approach := engine.NewList()
	defer approach.Close()

	CollectBombApproachAreas(target, sentryRange, approach)

	for i := int32(0); i < areas.Length(); i++ {
		area := engine.NavArea(areas.Get(i))
		score := ScoreNestArea(client, area, target, sentryRange, approach)

		if best == engine.NullArea() || score > bestScore {
			best = engine.Area(area)
			bestScore = score
		}
	}

	return best
}
