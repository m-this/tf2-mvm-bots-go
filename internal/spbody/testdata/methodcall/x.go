// Package methodcall is the methodmap shape: SourceMod's API is written on a
// receiver, and there is no plain function behind it to call instead.
package methodcall

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Threatened says whether the bot has noticed anything, which is the question
// every behaviour in the plugin opens with.
func Threatened(actor int32) bool {
	bot := engine.NextBotOf(actor)
	return bot.Vision().PrimaryKnownThreat(false) != 0
}
