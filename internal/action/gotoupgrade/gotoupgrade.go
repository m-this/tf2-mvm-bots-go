/*
Package gotoupgrade is source/redbots3/behavior/gotoupgrade.sp.

Walking to the upgrade station between waves. Twelfth behaviour across, and the
one that shows the text path: six maps have a station the nav mesh cannot get a
bot to, and each is named by a hard coded position.

//sp:action DefenderGotoUpgrade CTFBotGotoUpgrade
*/
package gotoupgrade

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

//sp:name m_iStation
var station [Slots]int32

// OnStart picks a station, and pretends the bot is at one when there is none it
// could reach.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	station[actor] = engine.FindClosestUpgradeStation(actor)

	if station[actor] <= engine.MaxClients() || !engine.IsValidEntity(station[actor]) {
		// We couldn't find an upgrade station to path to, so let's just pretend we're at one
		engine.SetInUpgradeZone(actor, true)
	} else if engine.RoundState() == engine.RoundStateRunning() {
		myOrigin := engine.Origin(actor)

		// The closest station is so far away, pretend we're in it
		if engine.VectorDistance(myOrigin, engine.WorldSpaceCenter(station[actor])) >= 1000.0 {
			engine.SetInUpgradeZone(actor, true)
		}
	}

	return engine.Continue()
}

// Update walks there, tracing for ground the bot can stand on in front of it.
func Update(actor int32) engine.Outcome {
	if engine.IsInUpgradeZone(actor) {
		return engine.ChangeTo(engine.Upgrade(), "Reached upgrade station; buying upgrades")
	}

	theStation := station[actor]

	// Moved from OnStart for technical reasons
	hasGoal, center := MapUpgradeStationGoal()

	if !hasGoal {
		if theStation <= engine.MaxClients() || !engine.IsValidEntity(theStation) {
			return engine.Done("No upgrade station to path to")
		}

		area := engine.NearestNavArea(engine.WorldSpaceCenter(theStation), true, 1000.0, false, false, engine.TeamAny())

		if area == engine.NullArea() {
			return engine.Continue()
		}

		center = engine.RandomPointIn(area)

		center[2] += 50.0

		engine.TraceRayFilter(center, engine.WorldSpaceCenter(theStation), engine.MaskPlayerSolid(), engine.RayTypeEndPoint(), engine.IgnoreActors())
		center = engine.TraceEndPosition()
	}

	myBot := engine.NextBotOf(actor)

	if engine.RepathTime(actor) <= engine.GameTime() {
		engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(1.0, 2.0))
		engine.RepathToPos(actor, myBot, center)
	}

	if engine.PathFailedFor(actor) {
		engine.NudgeTowardsGoal(actor, myBot, center)
	} else {
		engine.PathOf(actor).Update(myBot)
	}

	return engine.Continue()
}

// OnEnd forgets the station.
func OnEnd(actor int32) {
	station[actor] = -1
}

// OnNavAreaChanged bails out if the bot has wandered out of spawn mid-wave.
//
//nolint:revive // unused-parameter: the signature is the engine's, not ours
func OnNavAreaChanged(actor int32, newArea engine.NavArea, oldArea engine.NavArea) engine.Outcome {
	// If we are for some reason not in our spawn room during an active game, just bail out
	if newArea != 0 && engine.RoundState() == engine.RoundStateRunning() {
		spawnRoomFlag := engine.BlueSpawnRoom()

		if engine.PlayerTeam(actor) == engine.TeamRed() {
			spawnRoomFlag = engine.RedSpawnRoom()
		}

		if !newArea.HasAttributeTF(spawnRoomFlag) {
			return engine.TryDone(engine.ResultImportant(), "I am not in a spawn room")
		}
	}

	return engine.TryContinue()
}

// FindClosestUpgradeStation picks one of the stations a bot can actually walk
// to, at random.
//
//sp:name FindClosestUpgradeStation
func FindClosestUpgradeStation(actor int32) int32 {
	var stations [Slots]int32
	stationcount := int32(0)

	i := int32(-1)

	for {
		i = engine.FindEntityByClassname(i, "func_upgradestation")
		if i == -1 {
			break
		}
		if !engine.IsUpgradeStationEnabled(i) {
			continue
		}

		area := engine.NearestNavArea(engine.WorldSpaceCenter(i), true, 8000.0, false, false, engine.TeamAny())

		if area == engine.NullArea() {
			continue
		}

		center := area.Center()

		center[2] += 50.0

		engine.TraceRay(center, engine.WorldSpaceCenter(i), engine.MaskPlayerSolid(), engine.RayTypeEndPoint())
		center = engine.TraceEndPosition()

		if !engine.IsPathToVectorPossible(actor, center) {
			continue
		}

		stations[stationcount] = i
		stationcount++
	}

	if stationcount == 0 {
		return -1
	}

	return stations[engine.RandomInt(0, stationcount-1)]
}

// MapUpgradeStationGoal is the hard coded spot for the six maps whose station
// the nav mesh cannot get a bot to.
//
// switch over which prefix a map name contains is not one SourcePawn can write
//
//sp:name GetMapUpgradeStationGoal
//nolint:gocritic // ifElseChain: the shipped file is a chain of six, and a
func MapUpgradeStationGoal() (found bool, buffer [3]float32) {
	mapName := engine.CurrentMap()

	if engine.StrContains(mapName, "mvm_mannworks", true) != -1 {
		buffer = [3]float32{-643.9, -2635.2, 384.0}
		return true, buffer
	} else if engine.StrContains(mapName, "mvm_teien", true) != -1 {
		buffer = [3]float32{4613.1, -6561.9, 260.0}
		return true, buffer
	} else if engine.StrContains(mapName, "mvm_sequoia", true) != -1 {
		buffer = [3]float32{-5117.0, -377.3, 4.5}
		return true, buffer
	} else if engine.StrContains(mapName, "mvm_highground", true) != -1 {
		buffer = [3]float32{-2013.0, 4561.0, 448.0}
		return true, buffer
	} else if engine.StrContains(mapName, "mvm_newnormandy", true) != -1 {
		buffer = [3]float32{-345.0, 4178.0, 205.0}
		return true, buffer
	} else if engine.StrContains(mapName, "mvm_snowfall", true) != -1 {
		buffer = [3]float32{-26.0, 792.0, -159.0}
		return true, buffer
	}

	return false, buffer
}
