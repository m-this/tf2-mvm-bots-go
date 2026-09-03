package engine

/*
The game events the mod hooks, and what its own handlers reach.

Every one of these is a function some generated file already emits; the
declarations here are how one generated file calls into another.
*/

// EventCalls are the answers.
type EventCalls struct {
	PrefFlagOf                   func(index int32) int32
	FileExists                   func(path Text) bool
	ImportFromFile               func(kv KeyValues, path Text) bool
	ExportToFile                 func(kv KeyValues, path Text) bool
	PrintHintText                func(client int32, format string, args []any)
	PrintHintTextToAll           func(format string, args []any)
	DisplayPanelBotPercentages   func(client int32, classPercents [9]float32)
	CheckCommandAccess           func(client int32, command string, flags int32) bool
	EnableBotsCooldown           func(client int32) float32
	SetEnableBotsCooldown        func(client int32, when float32)
	IsValidAttributeName         func(name string) bool
	SetDefenderBotFlag           func(client int32, ours bool)
	SetRandomNameOnBot           func(client int32)
	RespawnPlayer                func(client int32)
	AddBotAttribute              func(client int32, attribute int32)
	MarkNeedsNamePurge           func(client int32)
	SetCurrencyWithBundles       func(client int32, credits int32)
	StartingCurrency             func(populationManager int32) int32
	AcquiredCreditsOfAllWaves    func() int32
	SetFakeClientConVar          func(client int32, name string, value string)
	HookTouchPost                func(entity int32)
	DefenderBotTouchPost         func(entity int32, other int32)
	HookDefenderBot              func(client int32)
	RemoveAllDefenderBotsWhen    func(reason string, danceInstead bool)
	NoteSpySighting              func(origin [3]float32)
	IsFinalWave                  func() bool
	NestRelocationOnWaveComplete func()
	ClearSniperStall             func(client int32)
	HandleTeamPlayerCountChanged func(team Team, whoChanging int32)
	HookEventPre                 func(name string)
	NoteMedicCall                func(client int32)
	BluAssistOnRobotSpawn        func(client int32)
	GiveBotCosmeticsSoon         func(client int32)
	CanBuyUpgradesNow            func(client int32) bool
}

var events EventCalls

// InstallEvents puts a set of answers behind them.
func InstallEvents(c EventCalls) func() {
	previous := events
	events = c
	return func() { events = previous }
}

// EventHookModePre asks to be called before the game handles the event, which
// is the only way to change it. Used for the mission update.
//
//sp:global EventHookMode_Pre
func EventHookModePre() int32 { return 0 }

// NoteMedicCall records that a player asked for a medic. Ported, mediccall.
//
//sp:body NoteMedicCall
func NoteMedicCall(client int32) {
	if events.NoteMedicCall == nil {
		missing("NoteMedicCall")
	}
	events.NoteMedicCall(client)
}

// BluAssistOnRobotSpawn bends a robot the assist is meant to help. Ported,
// blu_assist.
//
//sp:body BluAssist_OnRobotSpawn
func BluAssistOnRobotSpawn(client int32) {
	if events.BluAssistOnRobotSpawn == nil {
		missing("BluAssist_OnRobotSpawn")
	}
	events.BluAssistOnRobotSpawn(client)
}

// GiveBotCosmeticsSoon dresses a bot half a second after it spawns. Ported,
// cosmetics.
//
//sp:body GiveBotCosmeticsSoon
func GiveBotCosmeticsSoon(client int32) {
	if events.GiveBotCosmeticsSoon == nil {
		missing("GiveBotCosmeticsSoon")
	}
	events.GiveBotCosmeticsSoon(client)
}

// CanBuyUpgradesNow says the station is open to this bot. Ported, botqueries.
//
//sp:body CanBuyUpgradesNow
func CanBuyUpgradesNow(client int32) bool {
	if events.CanBuyUpgradesNow == nil {
		missing("CanBuyUpgradesNow")
	}
	return events.CanBuyUpgradesNow(client)
}

// HookEventPre asks to be called before the game handles the event, which is
// the only way to change one. The mission update is the only one the mod does
// that with.
//
//sp:native HookEvent after EventHookMode_Pre
//nolint:revive // unused-parameter: the callback is a name the emitter writes
func HookEventPre(name string, callback func(event Event, name string, dontBroadcast bool) Outcome) {
	if events.HookEventPre == nil {
		missing("HookEvent")
	}
	events.HookEventPre(name)
}

// NoteSpySighting records where a robot spy was last seen. Ported, spycheck.
//
//sp:body NoteSpySighting
func NoteSpySighting(origin [3]float32) {
	if events.NoteSpySighting == nil {
		missing("NoteSpySighting")
	}
	events.NoteSpySighting(origin)
}

// IsFinalWave says this is the last one of the mission. Ported, mission.
//
//sp:body IsFinalWave
func IsFinalWave() bool {
	if events.IsFinalWave == nil {
		missing("IsFinalWave")
	}
	return events.IsFinalWave()
}

// EngineerNestRelocationOnWaveComplete decides whether the nest should move
// before the shopping trip tears it down. Ported, engineeridle.
//
//sp:body EngineerNestRelocation_OnWaveComplete
func EngineerNestRelocationOnWaveComplete() {
	if events.NestRelocationOnWaveComplete == nil {
		missing("EngineerNestRelocation_OnWaveComplete")
	}
	events.NestRelocationOnWaveComplete()
}

// ClearSniperStall forgets how long a sniper has been standing still. Ported,
// stuckwatch.
//
//sp:body ClearSniperStall
func ClearSniperStall(client int32) {
	if events.ClearSniperStall == nil {
		missing("ClearSniperStall")
	}
	events.ClearSniperStall(client)
}

// HandleTeamPlayerCountChanged reseats the team when a person joins or leaves.
// Still in tf2_defenderbots.sp.
//
//sp:body HandleTeamPlayerCountChanged
func HandleTeamPlayerCountChanged(team Team, whoChanging int32) {
	if events.HandleTeamPlayerCountChanged == nil {
		missing("HandleTeamPlayerCountChanged")
	}
	events.HandleTeamPlayerCountChanged(team, whoChanging)
}

/*
Turning a bot the server made into one of ours.

Everything below runs once, a fifth of a second after the bot's first spawn: the
game has finished building it by then, and the flag that says it is ours is set
in the middle of the list, which is why the cosmetics are asked for again after
it.
*/

// RespawnPlayer puts the bot back, which is the only way custom weapons take.
//
//sp:native TF2_RespawnPlayer
func RespawnPlayer(client int32) {
	if events.RespawnPlayer == nil {
		missing("TF2_RespawnPlayer")
	}
	events.RespawnPlayer(client)
}

// AddBotAttribute turns one of the game's own bot flags on.
//
//sp:plugin VS_AddBotAttribute
func AddBotAttribute(client int32, attribute int32) {
	if events.AddBotAttribute == nil {
		missing("VS_AddBotAttribute")
	}
	events.AddBotAttribute(client, attribute)
}

// MarkNeedsNamePurge tells the game the name is about to change.
//
//sp:native BaseEntity_MarkNeedsNamePurge
func MarkNeedsNamePurge(client int32) {
	if events.MarkNeedsNamePurge == nil {
		missing("BaseEntity_MarkNeedsNamePurge")
	}
	events.MarkNeedsNamePurge(client)
}

// SetCurrencyWithBundles writes a bot's credits, Archipelago's bundles
// included. Ported, archipelago.
//
//sp:body SetCurrencyWithBundles
func SetCurrencyWithBundles(client int32, credits int32) {
	if events.SetCurrencyWithBundles == nil {
		missing("SetCurrencyWithBundles")
	}
	events.SetCurrencyWithBundles(client, credits)
}

// StartingCurrency is what the mission starts everybody with.
//
//sp:body GetStartingCurrency
func StartingCurrency(populationManager int32) int32 {
	if events.StartingCurrency == nil {
		missing("GetStartingCurrency")
	}
	return events.StartingCurrency(populationManager)
}

// AcquiredCreditsOfAllWaves is what the game's own record says has been picked
// up so far.
//
//sp:body GetAcquiredCreditsOfAllWaves
func AcquiredCreditsOfAllWaves() int32 {
	if events.AcquiredCreditsOfAllWaves == nil {
		missing("GetAcquiredCreditsOfAllWaves")
	}
	return events.AcquiredCreditsOfAllWaves()
}

// PopulationManager is g_iPopulationManager, the entity the mission is read
// off.
//
//sp:global g_iPopulationManager
func PopulationManager() int32 { return -1 }

// SetFakeClientConVar writes a convar on a bot, which is how its field of view
// is set.
//
//sp:native SetFakeClientConVar
func SetFakeClientConVar(client int32, name string, value string) {
	if events.SetFakeClientConVar == nil {
		missing("SetFakeClientConVar")
	}
	events.SetFakeClientConVar(client, name, value)
}

// HookTouchPost hooks the touch the credit pickup is noticed by.
//
//sp:native SDKHook after SDKHook_TouchPost DefenderBot_TouchPost
func HookTouchPost(entity int32) {
	if events.HookTouchPost == nil {
		missing("SDKHook")
	}
	events.HookTouchPost(entity)
}

// HookDefenderBot puts the mod's detours on one bot. Still in dhooks.sp.
//
//sp:plugin DHooks_DefenderBot
func HookDefenderBot(client int32) {
	if events.HookDefenderBot == nil {
		missing("DHooks_DefenderBot")
	}
	events.HookDefenderBot(client)
}

// BotProjectileShield is CTFBot_PROJECTILE_SHIELD, which lets a medic bot use
// the vaccinator's bubble.
//
//sp:global CTFBot_PROJECTILE_SHIELD
func BotProjectileShield() int32 { return 0 }

// RemoveAllDefenderBotsWhen kicks the team, or leaves it dancing on the final
// wave. Ported, manage.
//
//sp:body RemoveAllDefenderBots
func RemoveAllDefenderBotsWhen(reason string, danceInstead bool) {
	if events.RemoveAllDefenderBotsWhen == nil {
		missing("RemoveAllDefenderBots")
	}
	events.RemoveAllDefenderBotsWhen(reason, danceInstead)
}

// SetDefenderBotFlag marks a bot as one of ours, which is what every other
// question about it turns on.
//
//sp:slotset g_bIsDefenderBot
func SetDefenderBotFlag(client int32, ours bool) {
	if events.SetDefenderBotFlag == nil {
		missing("g_bIsDefenderBot")
	}
	events.SetDefenderBotFlag(client, ours)
}

// SetRandomNameOnBotFor gives it a name off the list. Ported, botnames.
//
//sp:body SetRandomNameOnBot
func SetRandomNameOnBotFor(client int32) {
	if events.SetRandomNameOnBot == nil {
		missing("SetRandomNameOnBot")
	}
	events.SetRandomNameOnBot(client)
}

// RequestCredits is redbots_manager_bot_request_credits, whether bots ask the
// credits plugin for what they are owed.
//
//sp:global redbots_manager_bot_request_credits
func RequestCredits() ConVar { return 0 }

// IsValidAttributeName says the schema knows that attribute, which a custom
// one may not be.
//
//sp:native TF2Attrib_IsValidAttributeName
func IsValidAttributeName(name string) bool {
	if events.IsValidAttributeName == nil {
		missing("TF2Attrib_IsValidAttributeName")
	}
	return events.IsValidAttributeName(name)
}

// CheckCommandAccess says the player holds that admin flag.
//
//sp:native CheckCommandAccess after true
func CheckCommandAccess(client int32, command string, flags int32) bool {
	if events.CheckCommandAccess == nil {
		missing("CheckCommandAccess")
	}
	return events.CheckCommandAccess(client, command, flags)
}

// AdmFlagGeneric is ADMFLAG_GENERIC, the flag an ordinary admin has.
//
//sp:global ADMFLAG_GENERIC
func AdmFlagGeneric() int32 { return 0 }

// NullString is NULL_STRING, which CheckCommandAccess takes where a command
// name would go when the flag alone is the question.
//
//sp:global NULL_STRING
func NullString() string { return "" }

// EnableBotsCooldown is g_flEnableBotsCooldown, how long a player who switched
// from BLUE is barred from starting the bots.
//
//sp:slot g_flEnableBotsCooldown
func EnableBotsCooldown(client int32) float32 {
	if events.EnableBotsCooldown == nil {
		missing("g_flEnableBotsCooldown")
	}
	return events.EnableBotsCooldown(client)
}

// SetEnableBotsCooldown writes it.
//
//sp:slotset g_flEnableBotsCooldown
func SetEnableBotsCooldown(client int32, when float32) {
	if events.SetEnableBotsCooldown == nil {
		missing("g_flEnableBotsCooldown")
	}
	events.SetEnableBotsCooldown(client, when)
}

// FileExists says the path is there, which a config the server has not written
// is not.
//
//sp:native FileExists
func FileExists(path Text) bool {
	if events.FileExists == nil {
		missing("FileExists")
	}
	return events.FileExists(path)
}

// ImportFromFile reads a KeyValues off disk, and says whether it parsed.
//
//sp:method ImportFromFile
func (kv KeyValues) ImportFromFile(path Text) bool {
	if events.ImportFromFile == nil {
		missing("KeyValues.ImportFromFile")
	}
	return events.ImportFromFile(kv, path)
}

// ExportToFile writes one back.
//
//sp:method ExportToFile
func (kv KeyValues) ExportToFile(path Text) bool {
	if events.ExportToFile == nil {
		missing("KeyValues.ExportToFile")
	}
	return events.ExportToFile(kv, path)
}

// PrintHintText puts a line in the middle of one player's screen.
//
//sp:native PrintHintText
func PrintHintText(client int32, format string, args ...any) {
	if events.PrintHintText == nil {
		missing("PrintHintText")
	}
	events.PrintHintText(client, format, args)
}

// PrintHintTextToAll does it for everybody.
//
//sp:native PrintHintTextToAll
func PrintHintTextToAll(format string, args ...any) {
	if events.PrintHintTextToAll == nil {
		missing("PrintHintTextToAll")
	}
	events.PrintHintTextToAll(format, args)
}

// DisplayPanelBotPercentages shows each class's share of the draw. Ported,
// panels.
//
//sp:body CreateDisplayPanelBotPercentages
func DisplayPanelBotPercentages(client int32, classPercents [9]float32) {
	if events.DisplayPanelBotPercentages == nil {
		missing("CreateDisplayPanelBotPercentages")
	}
	events.DisplayPanelBotPercentages(client, classPercents)
}

// TextOfPath reads a path buffer where a Text is expected. SourcePawn has one
// char[]; this says the two are the same value and emits nothing.
//
//sp:same TextOfPath
func TextOfPath(from [256]byte) Text {
	var out Text
	copy(out[:], from[:])
	return out
}
