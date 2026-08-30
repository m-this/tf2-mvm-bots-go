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

// How many icons the wave bar holds, and the one a tank puts there.
const (
	//sp:name MVM_WAVE_CLASS_ICONS_MAX
	waveClassIconsMax = 12
	//sp:name MVM_TANK_CLASS_ICON
	tankClassIcon = "tank"
)

/*
WaveHasClassIcon says the coming wave has that kind of robot in it.

The wave bar is what a player reads before the wave starts, and it is the only
thing that says what is coming: tf_objective_resource carries it, and a question
asked before the game has filled it in sees an empty wave.
*/
//
//sp:name WaveHasClassIcon
func WaveHasClassIcon(needle string) bool {
	rsrc := engine.FindEntityByClassname(engine.MaxClients()+1, "tf_objective_resource")

	if rsrc == -1 {
		return false
	}

	for i := int32(0); i < waveClassIconsMax; i++ {
		icon := engine.WaveClassName(rsrc, i)

		if icon[0] != 0 && engine.StrContains(icon, needle, false) != -1 {
			return true
		}
	}

	return false
}

// IsTankWave says a tank is coming.
//
//sp:name IsTankWave
func IsTankWave() bool {
	return WaveHasClassIcon(tankClassIcon)
}

// WaveHasExplosiveRobots is what blast resistance is priced against.
//
//sp:name WaveHasExplosiveRobots
func WaveHasExplosiveRobots() bool {
	return WaveHasClassIcon("demo") || WaveHasClassIcon("soldier") || IsTankWave()
}

// WaveHasBulletRobots is what bullet resistance is priced against.
//
//sp:name WaveHasBulletRobots
func WaveHasBulletRobots() bool {
	return WaveHasClassIcon("heavy") || WaveHasClassIcon("scout") || WaveHasClassIcon("sniper")
}

// WaveHasFireRobots is what fire resistance is priced against.
//
//sp:name WaveHasFireRobots
func WaveHasFireRobots() bool {
	return WaveHasClassIcon("pyro")
}
