/*
Package rostercounts is the counting half of source/tf2_defenderbots.sp: who is
on the team, who among them is a person, and who has said they are ready.
*/
package rostercounts

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// BuyUpgradesMaxTime is how long a shopping trip is given, owned here because
// this file is the first to read it and a define has to come before its reader.
//
//sp:name BUY_UPGRADES_MAX_TIME
const BuyUpgradesMaxTime = 30.0

/*
IsDefenderBot says the slot holds one of ours.

The flag is the answer whenever it has been set. It has not been for a bot the
server made before the mod noticed it, and the name is what is left to go on:
every bot this mod adds carries the identity name.
*/
//
//sp:name IsDefenderBot
func IsDefenderBot(client int32) bool {
	if engine.DefenderBotFlag(client) {
		return true
	}

	if !engine.IsFakeClient(client) {
		return false
	}

	_, clientName := engine.ClientName(client)

	return engine.StrContains(clientName, engine.TFBotIdentityName(), true) != -1
}

// GetDefenderBotCount is how many of ours are on that team.
//
//sp:name GetDefenderBotCount
func GetDefenderBotCount(team engine.Team) int32 {
	count := int32(0)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if engine.IsClientInGame(i) && engine.DefenderBotFlag(i) && engine.ClientTeam(i) == team {
			count++
		}
	}

	return count
}

// GetHumanAndDefenderBotCount is everybody on that team who is not somebody
// else's bot.
//
//sp:name GetHumanAndDefenderBotCount
func GetHumanAndDefenderBotCount(team engine.Team) int32 {
	count := int32(0)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if engine.IsClientInGame(i) && (IsDefenderBot(i) || !engine.IsFakeClient(i)) && engine.ClientTeam(i) == team {
			count++
		}
	}

	return count
}

// GetRealPlayerCount is how many people are on the server.
//
//sp:name GetRealPlayerCount
func GetRealPlayerCount() int32 {
	count := int32(0)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if engine.IsClientInGame(i) && !engine.IsFakeClient(i) {
			count++
		}
	}

	return count
}

// GetCountOfPlayersChoosingBotClasses is how many have the lineup menu open,
// which the vote waits on.
//
//sp:name GetCountOfPlayersChoosingBotClasses
func GetCountOfPlayersChoosingBotClasses() int32 {
	count := int32(0)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if engine.IsClientInGame(i) && engine.ChoosingBotClasses(i) {
			count++
		}
	}

	return count
}

/*
ExtendUpgradeTimeForNewBots gives a bot that joined late enough of the break to
shop in.

Only when there is little left: a bot added with twenty seconds on the clock
cannot buy anything and starts the wave stock, which is a seat wasted for the
whole wave rather than a break made slightly longer.
*/
//
//sp:name ExtendUpgradeTimeForNewBots
func ExtendUpgradeTimeForNewBots() {
	restartRoundTime := engine.GameRulesFloat("m_flRestartRoundTime")

	if restartRoundTime <= 0 {
		return
	}

	if restartRoundTime-engine.GameTime() <= BuyUpgradesMaxTime {
		// Add a little more time for the new bot to ready.
		engine.SetGameRulesFloat("m_flRestartRoundTime", restartRoundTime+BuyUpgradesMaxTime)
	}
}

/*
ClearBuildingsBeforeKick takes an engineer's nest down with him.

A sentry whose owner has left is somebody else's problem: it keeps shooting, it
cannot be upgraded or repaired, and nothing will ever remove it. Walked
backwards because removing one shortens the list.
*/
//
//sp:name ClearBuildingsBeforeKick
func ClearBuildingsBeforeKick(client int32) {
	for i := engine.PlayerObjectCount(client) - 1; i >= 0; i-- {
		building := engine.PlayerObject(client, i)

		if engine.IsValidEntity(building) {
			engine.RemoveEntity(building)
		}
	}

	// The one the game already took out of that list, because he was carrying
	// it.
	carried := engine.CarriedObject(client)

	if carried != -1 && engine.IsValidEntity(carried) {
		engine.RemoveEntity(carried)
	}
}

// RecycleDefenderBots clears the team out so a new lineup can be seated, and
// says how many seats it freed.
//
//sp:name RecycleDefenderBots
func RecycleDefenderBots() int32 {
	kicked := int32(0)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if engine.IsClientInGame(i) && IsDefenderBot(i) && engine.ClientTeam(i) == engine.TeamRed() {
			kicked++
			ClearBuildingsBeforeKick(i)
			engine.KickClient(i, "BotManager3: the team changed")
		}
	}

	return kicked
}
