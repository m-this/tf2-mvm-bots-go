/*
Package rostercounts is the counting half of source/tf2_defenderbots.sp: who is
on the team, who among them is a person, and who has said they are ready.
*/
package rostercounts

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

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
