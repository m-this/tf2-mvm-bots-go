package engine

/*
A server owner changed a setting, and the mod has to catch up with it.

Each of these is the hook on one convar. The value arrives as text, because that
is what a convar change hands a callback.
*/

// SettingsCalls are the answers.
type SettingsCalls struct {
	SetConVarInt                func(c ConVar, value int32)
	IsNoConVar                  func(c ConVar) bool
	FreeChosenBotTeamAnnouncing func(announce bool)
}

var settings SettingsCalls

// InstallSettings puts a set of answers behind them.
func InstallSettings(c SettingsCalls) func() {
	previous := settings
	settings = c
	return func() { settings = previous }
}

// SetInt writes a convar as a number, which fires its own change hook.
//
//sp:method SetInt
func (c ConVar) SetInt(value int32) {
	if settings.SetConVarInt == nil {
		missing("ConVar.SetInt")
	}
	settings.SetConVarInt(c, value)
}

// FreeChosenBotTeamAnnouncing drops the held lineup and says so in chat.
// Ported, seating.
//
//sp:body FreeChosenBotTeam
func FreeChosenBotTeamAnnouncing(announce bool) {
	if settings.FreeChosenBotTeamAnnouncing == nil {
		missing("FreeChosenBotTeam")
	}
	settings.FreeChosenBotTeamAnnouncing(announce)
}
