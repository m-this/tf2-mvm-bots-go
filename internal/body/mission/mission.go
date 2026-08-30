/*
Package mission is the part of source/redbots3/util.sp that answers questions
about the mission and about who is playing it.
*/
package mission

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// IsTFBotPlayer says the slot is one of the game's own bots.
//
//sp:name IsTFBotPlayer
func IsTFBotPlayer(client int32) bool {
	// TODO: change this, as it's not entirely reliable
	return engine.IsFakeClient(client)
}

// IsFinalWave says this is the last one the mission has.
//
//sp:name IsFinalWave
func IsFinalWave() bool {
	rsrc := engine.FindEntityByClassname(engine.MaxClients()+1, "tf_objective_resource")

	if rsrc != -1 {
		if engine.WaveCount(rsrc) == engine.MaxWaveCount(rsrc) {
			return true
		}
	} else {
		engine.LogError("IsFinalWave: Could find entity tf_objective_resource!")
	}

	return false
}

// IsSentryBusterRobot says the robot is the one whose whole job is to walk into
// a nest and detonate.
//
//sp:name IsSentryBusterRobot
func IsSentryBusterRobot(client int32) bool {
	if IsTFBotPlayer(client) {
		return engine.TFBotMission(client) == engine.MissionDestroySentries()
	}

	model := engine.ClientModel(client)

	return engine.StrEqual(model, "models/bots/demo/bot_sentry_buster.mdl")
}

// SelectRandomReachableEnemy is one of the robots out of their spawn, chosen at
// random, and -1 when there are none.
//
//sp:name SelectRandomReachableEnemy
func SelectRandomReachableEnemy(actor int32) int32 {
	opposingTeam := engine.PlayerEnemyTeam(actor)

	var playerarray [65]int32

	playercount := int32(0)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if i == actor {
			continue
		}

		if !engine.IsClientInGame(i) {
			continue
		}

		if !engine.IsPlayerAlive(i) {
			continue
		}

		if engine.PlayerTeam(i) != opposingTeam {
			continue
		}

		if engine.IsPointInRespawnRoom(engine.WorldSpaceCenter(i)) {
			continue
		}

		if IsSentryBusterRobot(i) {
			continue
		}

		playerarray[playercount] = i
		playercount++
	}

	if playercount > 0 {
		return playerarray[engine.RandomInt(0, playercount-1)]
	}

	return -1
}
