package engine

/*
A server owner changed a setting, and the mod has to catch up with it.

Each of these is the hook on one convar. The value arrives as text, because that
is what a convar change hands a callback.
*/

// SettingsCalls are the answers.
type SettingsCalls struct {
	SetConVarInt                      func(c ConVar, value int32)
	IsNoConVar                        func(c ConVar) bool
	FreeChosenBotTeamAnnouncing       func(announce bool)
	ArchipelagoRecheck                func()
	DebugFaultsOnGameFrame            func()
	SetPopulationManager              func(entity int32)
	DHooksOnEntityCreated             func(entity int32, classname Text)
	SetDetonatingPlayer               func(client int32)
	NoticeThreat                      func(client int32, threat int32)
	CreateTimerData                   func(interval float32, data int32) Timer
	SetAddingBotTime                  func(when float32)
	SetPlayerForcedPref               func(client int32)
	PlayerForcedPref                  func() int32
	ResetMapHintNests                 func()
	ConfigLoadServerLoadout           func()
	ConfigLoadBotNames                func()
	ConfigLoadMap                     func()
	CreateBotPreferenceMenu           func()
	ResetSpawnExitWatch               func(client int32)
	ResetLoadouts                     func(client int32)
	ForgetBotSeat                     func(client int32)
	ForgetBotCosmetics                func(client int32)
	ReseatOnMapStart                  func()
	CreateTimerFlags                  func(interval float32, flags int32) Timer
	CompareTextTo                     func(a Text, b string) int32
	UpdateChosenBotTeamCompositionFor func(caller int32)
	AllowBotRedo                      func() bool
	ShowDefenderBotTeamSetupMenu      func(client int32, itemPosition int32, initialize bool, numBotsToAdd int32)
	TeamHumanClientCount              func(team Team) int32
	StartBotVote                      func(client int32) bool
	HavePlayersChosenBotTeam          func() bool
	AddingBotTime                     func() float32
	CompareTextCased                  func(a Text, b string, caseSensitive bool) int32
	AddDefenderTFBotClass             func(count int32, class Text)
	AddBotsBasedOnLineupModeCount     func(count int32)
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

/*
The map's start and a client's departure reach into every subject at once.

Both are the plugin's own forwards, so each is a list of resets that belong to
other packages. Every one of these is generated already; this is how a forward
in one package calls into another.
*/

// SetAddingBotTime writes g_flAddingBotTime, when the last bot was asked for.
//
//sp:globalset g_flAddingBotTime
func SetAddingBotTime(when float32) {
	if settings.SetAddingBotTime == nil {
		missing("g_flAddingBotTime")
	}
	settings.SetAddingBotTime(when)
}

// SetPlayerForcedPref writes g_iPlayerForcedPref, the one player whose
// preferences stand in for everybody's.
//
//sp:globalset g_iPlayerForcedPref
func SetPlayerForcedPref(client int32) {
	if settings.SetPlayerForcedPref == nil {
		missing("g_iPlayerForcedPref")
	}
	settings.SetPlayerForcedPref(client)
}

// PlayerForcedPref reads it.
//
//sp:global g_iPlayerForcedPref
func PlayerForcedPref() int32 {
	if settings.PlayerForcedPref == nil {
		missing("g_iPlayerForcedPref")
	}
	return settings.PlayerForcedPref()
}

// ResetMapHintNests drops the nest spots the last map's hints put down. Ported,
// nesthint.
//
//sp:body ResetMapHintNests
func ResetMapHintNests() {
	if settings.ResetMapHintNests == nil {
		missing("ResetMapHintNests")
	}
	settings.ResetMapHintNests()
}

// ConfigLoadServerLoadout reads the server's own loadout file. Ported,
// playerpref.
//
//sp:body Config_LoadServerLoadout
func ConfigLoadServerLoadout() {
	if settings.ConfigLoadServerLoadout == nil {
		missing("Config_LoadServerLoadout")
	}
	settings.ConfigLoadServerLoadout()
}

// ConfigLoadBotNames reads the names the bots are drawn from. Ported,
// mapconfig.
//
//sp:body Config_LoadBotNames
func ConfigLoadBotNames() {
	if settings.ConfigLoadBotNames == nil {
		missing("Config_LoadBotNames")
	}
	settings.ConfigLoadBotNames()
}

// ConfigLoadMap reads the file for the map being played. Ported, mapconfig.
//
//sp:body Config_LoadMap
func ConfigLoadMap() {
	if settings.ConfigLoadMap == nil {
		missing("Config_LoadMap")
	}
	settings.ConfigLoadMap()
}

// CreateBotPreferenceMenu builds the menu once, for everybody. Ported,
// prefmenu.
//
//sp:body CreateBotPreferenceMenu
func CreateBotPreferenceMenu() {
	if settings.CreateBotPreferenceMenu == nil {
		missing("CreateBotPreferenceMenu")
	}
	settings.CreateBotPreferenceMenu()
}

// ResetSpawnExitWatch forgets what that client was doing on the way out of
// spawn. Ported, spawnexit.
//
//sp:body ResetSpawnExitWatch
func ResetSpawnExitWatch(client int32) {
	if settings.ResetSpawnExitWatch == nil {
		missing("ResetSpawnExitWatch")
	}
	settings.ResetSpawnExitWatch(client)
}

// ResetLoadouts forgets the weapons that client was given. Ported, loadouts.
//
//sp:body ResetLoadouts
func ResetLoadouts(client int32) {
	if settings.ResetLoadouts == nil {
		missing("ResetLoadouts")
	}
	settings.ResetLoadouts(client)
}

// ForgetBotSeat gives the seat back. Ported, playerpref.
//
//sp:body ForgetBotSeat
func ForgetBotSeat(client int32) {
	if settings.ForgetBotSeat == nil {
		missing("ForgetBotSeat")
	}
	settings.ForgetBotSeat(client)
}

// ForgetBotCosmetics drops the hats that client was wearing. Ported,
// cosmetics.
//
//sp:body ForgetBotCosmetics
func ForgetBotCosmetics(client int32) {
	if settings.ForgetBotCosmetics == nil {
		missing("ForgetBotCosmetics")
	}
	settings.ForgetBotCosmetics(client)
}

// ReseatOnMapStart drops a pending lineup change. Ported, seating.
//
//sp:body Reseat_OnMapStart
func ReseatOnMapStart() {
	if settings.ReseatOnMapStart == nil {
		missing("Reseat_OnMapStart")
	}
	settings.ReseatOnMapStart()
}

// CreateTimerFlags is CreateTimer with no data cell, which is the fourth shape
// the plugin writes.
//
//sp:native CreateTimer
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func CreateTimerFlags(interval float32, callback func(timer Timer) Outcome, flags int32) Timer {
	if settings.CreateTimerFlags == nil {
		missing("CreateTimer")
	}
	return settings.CreateTimerFlags(interval, flags)
}

// CompareTextTo is strcmp against a literal, with the case flag left at its
// default, which is how the plugin writes it.
//
//sp:native strcmp
func CompareTextTo(a Text, b string) int32 {
	if settings.CompareTextTo == nil {
		missing("strcmp")
	}
	return settings.CompareTextTo(a, b)
}

// UpdateChosenBotTeamCompositionFor decides the lineup and names who asked.
// Ported, seating.
//
//sp:body UpdateChosenBotTeamComposition
func UpdateChosenBotTeamCompositionFor(caller int32) {
	if settings.UpdateChosenBotTeamCompositionFor == nil {
		missing("UpdateChosenBotTeamComposition")
	}
	settings.UpdateChosenBotTeamCompositionFor(caller)
}

// AllowBotRedo reads g_bAllowBotTeamRedo.
//
//sp:global g_bAllowBotTeamRedo
func AllowBotRedo() bool {
	if settings.AllowBotRedo == nil {
		missing("g_bAllowBotTeamRedo")
	}
	return settings.AllowBotRedo()
}

// ShowDefenderBotTeamSetupMenu puts the lineup menu up. Ported, teammenu.
//
//sp:body ShowDefenderBotTeamSetupMenu
func ShowDefenderBotTeamSetupMenu(client int32, itemPosition int32, initialize bool, numBotsToAdd int32) {
	if settings.ShowDefenderBotTeamSetupMenu == nil {
		missing("ShowDefenderBotTeamSetupMenu")
	}
	settings.ShowDefenderBotTeamSetupMenu(client, itemPosition, initialize, numBotsToAdd)
}

// TeamHumanClientCount is how many people, not bots, are on that team.
//
//sp:native GetTeamHumanClientCount
func TeamHumanClientCount(team Team) int32 {
	if settings.TeamHumanClientCount == nil {
		missing("GetTeamHumanClientCount")
	}
	return settings.TeamHumanClientCount(team)
}

// StartBotVote puts the bot vote up. Ported, prefmenu.
//
//sp:body StartBotVote
func StartBotVote(client int32) bool {
	if settings.StartBotVote == nil {
		missing("StartBotVote")
	}
	return settings.StartBotVote(client)
}

// ExtraBots is redbots_manager_extra_bots, how many over the team size a
// player may ask for.
//
//sp:global redbots_manager_extra_bots
func ExtraBots() ConVar { return 0 }

// PluginBadLoad is Plugin_BadLoad, which a command returns to say it refused
// for a reason the caller has to fix first.
//
//sp:global Plugin_BadLoad
func PluginBadLoad() Outcome { return 2 }

// CompareTextCased is strcmp with the case flag written out.
//
//sp:native strcmp
func CompareTextCased(a Text, b string, caseSensitive bool) int32 {
	if settings.CompareTextCased == nil {
		missing("strcmp")
	}
	return settings.CompareTextCased(a, b, caseSensitive)
}

// AddDefenderTFBotClass is AddDefenderTFBot with only the count and the class,
// which is how the extra-bot command calls it. Ported, manage.
//
//sp:body AddDefenderTFBot
func AddDefenderTFBotClass(count int32, class Text) {
	if settings.AddDefenderTFBotClass == nil {
		missing("AddDefenderTFBot")
	}
	settings.AddDefenderTFBotClass(count, class)
}

// AddBotsBasedOnLineupModeCount is AddBotsBasedOnLineupMode with the time
// adjustment left at its default. Ported, manage.
//
//sp:body AddBotsBasedOnLineupMode
func AddBotsBasedOnLineupModeCount(count int32) {
	if settings.AddBotsBasedOnLineupModeCount == nil {
		missing("AddBotsBasedOnLineupMode")
	}
	settings.AddBotsBasedOnLineupModeCount(count)
}

// HavePlayersChosenBotTeam says the lineup is settled enough to seat bots on.
// Ported, seating.
//
//sp:body HavePlayersChosenBotTeam
func HavePlayersChosenBotTeam() bool {
	if settings.HavePlayersChosenBotTeam == nil {
		missing("HavePlayersChosenBotTeam")
	}
	return settings.HavePlayersChosenBotTeam()
}

// AddingBotTime reads g_flAddingBotTime.
//
//sp:global g_flAddingBotTime
func AddingBotTime() float32 {
	if settings.AddingBotTime == nil {
		missing("g_flAddingBotTime")
	}
	return settings.AddingBotTime()
}
