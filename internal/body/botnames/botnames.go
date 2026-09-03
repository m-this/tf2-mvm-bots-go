/*
Package botnames is how a bot out of source/tf2_defenderbots.sp gets its name:
drawn from the list the names file filled, and drawn again when a player
already has it.
*/
package botnames

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

// MaxTries is how many redraws a bot gets before it keeps a name somebody
// else has.
const MaxTries = 10

// What a bot says to the player who took its name.
//
//sp:name g_sPlayerUseMyNameResponse
var playerUseMyNameResponse = [2]string{
	"You're very funny for using my name.",
	"You totally stole my name.",
}

//sp:name m_iFindNameTries
var findNameTries [slots.Count]int32

/*
SetRandomNameOnBot names the bot, and mocks whoever is already using the name
it drew before drawing again.

Bounded: a server full of players who each took a name from the list would
otherwise draw for ever.
*/
//
//sp:name SetRandomNameOnBot
func SetRandomNameOnBot(client int32) {
	var newName engine.Text
	GetRandomDefenderBotName(newName, 512)

	if engine.BotNames().Length() > 0 && engine.DoesAnyPlayerUseThisName(newName) && findNameTries[client] < MaxTries {
		findNameTries[client]++

		// Someone's already using my name, mock them for it and try again.
		engine.PrintToChatAll("%s : %s", newName, playerUseMyNameResponse[engine.RandomInt(0, int32(len(playerUseMyNameResponse)-1))])
		SetRandomNameOnBot(client)

		return
	}

	findNameTries[client] = 0
	engine.SetClientName(client, newName)
}

// GetRandomDefenderBotName draws one from the list, or says so when the list
// is empty.
//
//sp:name GetRandomDefenderBotName
//sp:length buffer maxlen
//nolint:revive,ineffassign,staticcheck,wastedassign // the write is the point: SourcePawn passes the buffer by reference and //sp:length carries its size
func GetRandomDefenderBotName(buffer engine.Text, maxlen int32) {
	if engine.BotNames().Length() == 0 {
		buffer = engine.TextFrom("You forgot to give me a name!")
		return
	}

	botName := engine.BotNames().GetString(engine.RandomInt(0, engine.BotNames().Length()-1))

	buffer = engine.CopyTextInto(botName)
}
