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
	ArchipelagoRecheck          func()
	DebugFaultsOnGameFrame      func()
	SetPopulationManager        func(entity int32)
	DHooksOnEntityCreated       func(entity int32, classname Text)
	SetDetonatingPlayer         func(client int32)
	NoticeThreat                func(client int32, threat int32)
	CreateTimerData             func(interval float32, data int32) Timer
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

/*
The plugin's own forwards reach these.

Each is a function that still lives in tf2_defenderbots.sp or in one of the
files it includes, called from a forward that has been ported.
*/

// ArchipelagoRecheck asks the campaign plugin whether it is there. Ported,
// archipelago.
//
//sp:body Archipelago_Recheck
func ArchipelagoRecheck() {
	if settings.ArchipelagoRecheck == nil {
		missing("Archipelago_Recheck")
	}
	settings.ArchipelagoRecheck()
}

// DebugFaultsOnGameFrame is the fault watcher's frame. Ported, faults.
//
//sp:body DebugFaults_OnGameFrame
func DebugFaultsOnGameFrame() {
	if settings.DebugFaultsOnGameFrame == nil {
		missing("DebugFaults_OnGameFrame")
	}
	settings.DebugFaultsOnGameFrame()
}

// SetPopulationManager writes g_iPopulationManager, the info_populator the
// wave record is read off.
//
//sp:globalset g_iPopulationManager
func SetPopulationManager(entity int32) {
	if settings.SetPopulationManager == nil {
		missing("g_iPopulationManager")
	}
	settings.SetPopulationManager(entity)
}

// DHooksOnEntityCreated hooks whatever the entity needs hooking for. Still in
// dhooks.sp.
//
//sp:plugin DHooks_OnEntityCreated
func DHooksOnEntityCreated(entity int32, classname Text) {
	if settings.DHooksOnEntityCreated == nil {
		missing("DHooks_OnEntityCreated")
	}
	settings.DHooksOnEntityCreated(entity, classname)
}

// SetDetonatingPlayer writes g_iDetonatingPlayer, the buster winding up.
//
//sp:globalset g_iDetonatingPlayer
func SetDetonatingPlayer(client int32) {
	if settings.SetDetonatingPlayer == nil {
		missing("g_iDetonatingPlayer")
	}
	settings.SetDetonatingPlayer(client)
}

// NoticeThreat tells a bot about somebody it has not seen. Ported, threat.
//
//sp:body TFBot_NoticeThreat
func NoticeThreat(client int32, threat int32) {
	if settings.NoticeThreat == nil {
		missing("TFBot_NoticeThreat")
	}
	settings.NoticeThreat(client, threat)
}

// CreateTimerData is CreateTimer with a cell and no flags, which is the third
// shape the plugin writes.
//
//sp:native CreateTimer
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func CreateTimerData(interval float32, callback func(timer Timer, data Cell) Outcome, data int32) Timer {
	if settings.CreateTimerData == nil {
		missing("CreateTimer")
	}
	return settings.CreateTimerData(interval, data)
}
