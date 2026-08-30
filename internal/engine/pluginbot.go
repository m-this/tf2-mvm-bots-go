package engine

/*
The plugin's own bot record, which is how a behaviour walks somebody without the
game's path following.

g_arrPluginBot is an array of enum structs, one per client, so reaching a field
of one is a slot and then a property. Writing one is the same with a value,
which is what //sp:propertyset is.
*/

// PluginBotCalls are the answers.
type PluginBotCalls struct {
	PluginBot         func(actor int32) PluginBot
	SetPathing        func(b PluginBot, pathing bool)
	SetPathGoalVector func(b PluginBot, goal [3]float32)
}

var pluginBots PluginBotCalls

// InstallPluginBots puts a set of answers behind them.
func InstallPluginBots(c PluginBotCalls) func() {
	previous := pluginBots
	pluginBots = c
	return func() { pluginBots = previous }
}

// PluginBot is the plugin's per-client record.
//
//sp:tag PluginBot
type PluginBot int32

// PluginBotOf is the record for that client.
//
//sp:slot g_arrPluginBot
func PluginBotOf(actor int32) PluginBot {
	if pluginBots.PluginBot == nil {
		missing("g_arrPluginBot")
	}
	return pluginBots.PluginBot(actor)
}

// SetPathing turns the plugin's own walking on or off.
//
//sp:propertyset bPathing
func (b PluginBot) SetPathing(pathing bool) {
	if pluginBots.SetPathing == nil {
		missing("PluginBot.bPathing")
	}
	pluginBots.SetPathing(b, pathing)
}

// SetPathGoalVector says where that walking is going.
//
//sp:method SetPathGoalVector
func (b PluginBot) SetPathGoalVector(goal [3]float32) {
	if pluginBots.SetPathGoalVector == nil {
		missing("PluginBot.SetPathGoalVector")
	}
	pluginBots.SetPathGoalVector(b, goal)
}
