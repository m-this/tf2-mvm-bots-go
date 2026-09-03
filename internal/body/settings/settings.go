/*
Package settings is the convar hooks out of source/tf2_defenderbots.sp: what the
mod does when a server owner changes one of its settings.
*/
package settings

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// ConVarChangedBotLineupMode reacts to the lineup mode being changed.
//
//sp:name ConVarChanged_BotLineupMode
//nolint:revive // unused-parameter: a convar hook is handed all three and this one reads the new value
func ConVarChangedBotLineupMode(convar engine.ConVar, oldValue engine.Text, newValue engine.Text) {
	mode := engine.StringToInt(newValue)

	switch mode {
	case engine.LineupModeRandom():
		engine.UpdateChosenBotTeamComposition()
	case engine.LineupModePreference():
		engine.UpdateChosenBotTeamComposition()
	case engine.LineupModeChoose():
		engine.FreeChosenBotTeamAnnouncing(true)
	case engine.LineupModePreferenceChoose():
		engine.FreeChosenBotTeamAnnouncing(true)
		engine.UpdateChosenBotTeamComposition()
	}
}

/*
	ConVarChangedTeamComposition reacts to the named team being retyped

Kicking a bot in the middle of a wave loses whatever it was doing and drops its
buildings, so a lineup change that arrives mid-wave is held until the break.
*/
//
//sp:name ConVarChanged_TeamComposition
//nolint:revive // unused-parameter: a convar hook is handed the convar and both values
func ConVarChangedTeamComposition(convar engine.ConVar, before engine.Text, after engine.Text) {
	if engine.StrEqualText(before, after) {
		return
	}

	if engine.RoundState() == engine.RoundStateRunning() {
		engine.SetReseatPending(true)
		engine.LogMessage("Reseat: the lineup changed mid-wave, holding it until the break")
		engine.PrintToChatAll("%s The new lineup takes effect when this wave ends.", engine.PluginPrefix())
		return
	}

	engine.ReseatDefenderBots()
}

/*
	ConVarChangedDefenderTeamSize asks the game for the seats and settles for what it gives

Reported as "i tried changing the number to 12 but then it stops adding bots",
which left a player with fewer bots at twelve than at six. Nothing here ever
touched tf_mvm_defenders_team_size, so the game kept refusing RED past its own
number while this asked for twelve, and the bots went nowhere.

Mann vs Machine is built around six: the upgrade station, the ready panel and the
scoreboard all assume it. Whether the game accepts more is the game's answer and
not ours, so this asks and then reads back rather than assuming either way. What
it reads back is the ceiling, and this convar is clamped to it so every place
that reads the number gets one the game will honour.

Failing loudly at seven beats spawning nothing at twelve.
*/
//
//sp:name ConVarChanged_DefenderTeamSize
//nolint:revive // unused-parameter: a convar hook is handed both values and this one reads the convar
func ConVarChangedDefenderTeamSize(convar engine.ConVar, before engine.Text, after engine.Text) {
	wanted := convar.Int()

	gameSize := engine.FindConVar("tf_mvm_defenders_team_size")

	if gameSize == engine.NoConVar() {
		engine.LogMessage("Team size: the game has no tf_mvm_defenders_team_size, so %d is taken as given", wanted)
		return
	}

	if gameSize.Int() != wanted {
		gameSize.SetInt(wanted)
	}

	allowed := gameSize.Int()

	if allowed == wanted {
		return
	}

	engine.LogMessage("Team size: asked the game for %d, it allows %d. Clamping, or no bot would be added at all",
		wanted, allowed)
	engine.PrintToChatAll("%s RED holds %d, not %d: the game refused the rest.", engine.PluginPrefix(), allowed, wanted)

	// The hook fires again on this write, and the second pass returns at the
	// equality above.
	convar.SetInt(allowed)
}

/*
	ConVarChangedManagerMode reacts to the manager mode being changed

It reads the new value and does nothing with it. Left that way on purpose:
mvm-z83.41 says a port does not carry a fix, and mvm-z83.79 is where the
question of what it was meant to catch belongs.
*/
//
//sp:name ConVarChanged_ManagerMode
//nolint:revive // unused-parameter: a convar hook is handed all three
func ConVarChangedManagerMode(convar engine.ConVar, oldValue engine.Text, newValue engine.Text) {
	mode := engine.StringToInt(newValue)

	_ = mode
}
