/*
Package evadebuster is source/redbots3/behavior/evadebuster.sp.

Getting out of the way of a sentry buster.

The buster is the one robot in the mode that kills a defender who does nothing,
and it announces itself: it spawns visible, it walks the whole map, and it takes
three seconds to detonate once it arrives. A play-test found the bots standing
in all of it.

This used to be dead code. Nothing suspended for the action, so nothing here
ever ran, and what was written would not have helped much if it had: it started
only once the buster was already taunting, and it escaped to the first nav area
more than 500 units away, which is inside the blast. An engineer, at that
moment, walked towards his sentry.

So there are two answers, at two distances, and only the second one is here:

	far   the engineer picks the sentry up and walks it out of the buster's
	      way, which is CTFBotMvMEngineerIdle's job because the machinery to
	      carry a building already lives there
	near  everybody runs, this file, and an engineer runs like anybody else.
	      A sentry is worth less than the engineer who can rebuild it

//sp:action DefenderEvadeBuster CTFBotEvadeBuster
*/
package evadebuster

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

// How far out to look for ground to run to.
//
//sp:name BUSTER_ESCAPE_SEARCH_RANGE
const escapeSearchRange = 1500.0

// A wave is not spent running: past this the bot goes back to fighting whatever
// the buster does.
//
//sp:name BUSTER_EVADE_MAX_TIME
const evadeMaxTime = 8.0

// maxAreas caps the collector walk. Every wave has one buster and every bot
// near it runs this, and the count is the map's.
const maxAreas = 256

//sp:name m_ctEvadeBusterGiveUp
var evadeBusterGiveUp [slots.Count]float32

// OnStart starts the clock and tells the team what is coming.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	evadeBusterGiveUp[actor] = engine.GameTime() + evadeMaxTime

	engine.SpeakConceptIfAllowed(actor, engine.ConceptIncoming())

	return engine.Continue()
}

// Update runs, until the clock runs out or there is nothing to run from.
func Update(actor int32) engine.Outcome {
	if evadeBusterGiveUp[actor] < engine.GameTime() {
		return engine.Done("Ran from the buster for long enough")
	}

	buster := Threat(actor)

	if buster == -1 {
		return engine.Done("No buster to run from")
	}

	myBot := engine.NextBotOf(actor)
	busterOrigin := engine.WorldSpaceCenter(buster)

	found, escape := FindEscape(actor, busterOrigin)

	if !found {
		return engine.Done("Nowhere to run")
	}

	if engine.RepathTime(actor) <= engine.GameTime() {
		engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(0.3, 0.4))
		engine.RepathToPos(actor, myBot, escape)
	}

	engine.PathOf(actor).Update(myBot)

	return engine.Continue()
}

/*
FindEscape is the ground furthest from the blast, of what the bot can see a way
to from here.

Furthest rather than first past a threshold, which is what this used to take: a
buster standing in a corridor makes most of the areas within a radius worse than
the one the bot is on, and the first one the collector happens to hand back is as
likely to be the far side of the buster as the near side of the exit.

No path is computed per candidate. One path query per area, at four a second,
for every bot near a buster, costs more than picking a spot the bot cannot quite
reach and being handed the next one a tenth of a second later.
*/
//
//sp:name CTFBotEvadeBuster_FindEscape
func FindEscape(actor int32, busterOrigin [3]float32) (found bool, escape [3]float32) {
	myOrigin := engine.Origin(actor)

	hAreas := engine.CollectAreasInRadius(myOrigin, escapeSearchRange)
	defer hAreas.Close()

	// The ground the bot is standing on, so that a bot with nowhere better still has an answer
	bestDistance := engine.VectorDistance(myOrigin, busterOrigin)

	count := hAreas.Count()

	// Every wave has one buster and every bot near it runs this. The count is the map's, so cap it
	if count > maxAreas {
		count = maxAreas
	}

	for i := int32(0); i < count; i++ {
		area := hAreas.Area(i)
		center := area.Center()

		distance := engine.VectorDistance(center, busterOrigin)

		if distance <= bestDistance {
			continue
		}

		bestDistance = distance
		escape = center
		found = true
	}

	return found, escape
}

/*
Threat is the buster this bot has to get away from, or -1.

A buster that has started its detonation is a threat at blast range whatever it
is doing. One that has not is a threat only when it is close enough that it
could arrive before the bot is gone, which is what keeps a team from spending
the wave backing away from a robot walking the length of the map.
*/
//
//sp:name CTFBotEvadeBuster_Threat
func Threat(client int32) int32 {
	myOrigin := engine.Origin(client)
	enemyTeam := engine.PlayerEnemyTeam(client)

	if engine.IsValidClientIndex(engine.DetonatingPlayer()) && engine.IsPlayerAlive(engine.DetonatingPlayer()) &&
		engine.PlayerTeam(engine.DetonatingPlayer()) == enemyTeam {
		theirOrigin := engine.Origin(engine.DetonatingPlayer())

		if engine.VectorDistance(myOrigin, theirOrigin) <= engine.BusterBlastRange()*2.0 {
			return engine.DetonatingPlayer()
		}
	}

	return engine.FindSentryBusterNear(myOrigin, enemyTeam, engine.BusterFleeRange())
}

// IsPossible says whether running is worth doing.
//
//sp:name CTFBotEvadeBuster_IsPossible
func IsPossible(client int32) bool {
	if !engine.IsPlayerAlive(client) {
		return false
	}

	/* A bot at the upgrade station is between waves and there is no buster walking towards it.
	Leaving the station mid-purchase is also how a bot ends up owing the wave a ready-up */
	if engine.IsInUpgradeZone(client) {
		return false
	}

	return Threat(client) != -1
}
