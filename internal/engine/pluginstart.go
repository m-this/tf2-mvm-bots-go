package engine

/*
Starting the plugin: the registrations, the gamedata and the one-time loads.

Everything here is wiring. What it wires is decided elsewhere; this is the file
that says which generated function answers which command, event and convar.
*/

// PluginStartCalls are the answers.
type PluginStartCalls struct {
	RegAdminCmd              func(name string, flags int32, description string)
	HookConVarChange         func(c ConVar)
	AddCommandListener       func(command string)
	AddNormalSoundHook       func()
	SetFailState             func(format string, args []any)
	NewGameData              func(name string) GameData
	CloseGameData            func(g GameData)
	GameConfAddress          func(g GameData, name string) Address
	InitOffsets              func(g GameData)
	InitMvMUpgrades          func(g GameData)
	InitSDKCalls             func(g GameData) bool
	InitDHooks               func(g GameData) bool
	SetUpgradesAddress       func(address Address)
	LateLoad                 func() bool
	LoadFeatures             func()
	BluAssistInit            func()
	DebugFaultsInit          func()
	LoadLoadoutFuncs         func()
	LoadPreferences          func()
	InitNextBotPathing       func()
	ArchipelagoInit          func()
	InitGameEventHooks       func()
	InitMapConfig            func()
	SetChosenClasses         func(l List)
	SetChosenSeats           func(l List)
	SetBotNames              func(l List)
	SetPlayerPrefPath        func(path Text)
	BuildPathInto            func(out Text, maxlen int32, format string)
	UpgradesAddress          func() Address
	FindGameConsoleVariables func()
	SetLateLoad              func(late bool)
	CreateNative             func(name string)
	MarkNativeAsOptional     func(name string)
	RegPluginLibrary         func(name string)
}

var pluginStarts PluginStartCalls

// InstallPluginStarts puts a set of answers behind them.
func InstallPluginStarts(c PluginStartCalls) func() {
	previous := pluginStarts
	pluginStarts = c
	return func() { pluginStarts = previous }
}

// RegAdminCmd registers a command only an admin may run.
//
//sp:native RegAdminCmd
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func RegAdminCmd(name string, callback func(client int32, args int32) Outcome, flags int32, description string) {
	if pluginStarts.RegAdminCmd == nil {
		missing("RegAdminCmd")
	}
	pluginStarts.RegAdminCmd(name, flags, description)
}

// RegAdminCmdPlain is the same with no description, which is the form most of
// the dump commands take.
//
//sp:native RegAdminCmd
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func RegAdminCmdPlain(name string, callback func(client int32, args int32) Outcome, flags int32) {
	if pluginStarts.RegAdminCmd == nil {
		missing("RegAdminCmd")
	}
	pluginStarts.RegAdminCmd(name, flags, "")
}

// RegConsoleCmdPlain is RegConsoleCmd with no description.
//
//sp:native RegConsoleCmd
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func RegConsoleCmdPlain(name string, callback func(client int32, args int32) Outcome) {
	if pluginStarts.RegAdminCmd == nil {
		missing("RegConsoleCmd")
	}
}

// HookConVarChange asks to be told when that convar's value moves.
//
//sp:native HookConVarChange
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func HookConVarChange(c ConVar, callback func(convar ConVar, before Text, after Text)) {
	if pluginStarts.HookConVarChange == nil {
		missing("HookConVarChange")
	}
	pluginStarts.HookConVarChange(c)
}

// AddCommandListener watches a command the game itself owns.
//
//sp:native AddCommandListener
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func AddCommandListener(callback func(client int32, command Text, argc int32) Outcome, command string) {
	if pluginStarts.AddCommandListener == nil {
		missing("AddCommandListener")
	}
	pluginStarts.AddCommandListener(command)
}

// AddNormalSoundHook watches every sound the server plays.
//
//sp:native AddNormalSoundHook
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func AddNormalSoundHook(callback func(clients [101]int32, numClients int32, sample Text, entity int32, channel int32, volume float32, level int32, pitch int32, flags int32, soundEntry Text, seed int32) Outcome) {
	if pluginStarts.AddNormalSoundHook == nil {
		missing("AddNormalSoundHook")
	}
	pluginStarts.AddNormalSoundHook()
}

// SetFailState stops the plugin, saying why. Nothing after it runs.
//
//sp:native SetFailState
func SetFailState(format string, args ...any) {
	if pluginStarts.SetFailState == nil {
		missing("SetFailState")
	}
	pluginStarts.SetFailState(format, args)
}

// GameData is a parsed gamedata file, which the caller owns.
//
//sp:tag GameData
type GameData int32

// NoGameData is null, what a missing file parses to.
//
//sp:global null
func NoGameData() GameData { return 0 }

// NewGameData parses one by name.
//
//sp:new GameData
func NewGameData(name string) GameData {
	if pluginStarts.NewGameData == nil {
		missing("new GameData")
	}
	return pluginStarts.NewGameData(name)
}

// Close releases it.
//
//sp:delete Close
func (g GameData) Close() {
	if pluginStarts.CloseGameData == nil {
		missing("delete GameData")
	}
	pluginStarts.CloseGameData(g)
}

// GameConfAddress is an address the gamedata file names.
//
//sp:native GameConfGetAddress
func GameConfAddress(g GameData, name string) Address {
	if pluginStarts.GameConfAddress == nil {
		missing("GameConfGetAddress")
	}
	return pluginStarts.GameConfAddress(g, name)
}

// InitOffsets reads every offset the mod needs. Still in offsets.sp.
//
//sp:plugin InitOffsets
func InitOffsets(g GameData) {
	if pluginStarts.InitOffsets == nil {
		missing("InitOffsets")
	}
	pluginStarts.InitOffsets(g)
}

// InitMvMUpgrades reads the upgrade offsets. Still in tf_upgrades.sp.
//
//sp:plugin InitMvMUpgrades
func InitMvMUpgrades(g GameData) {
	if pluginStarts.InitMvMUpgrades == nil {
		missing("InitMvMUpgrades")
	}
	pluginStarts.InitMvMUpgrades(g)
}

// InitSDKCalls prepares every SDKCall, and says whether it could. Still in
// sdkcalls.sp.
//
//sp:plugin InitSDKCalls
func InitSDKCalls(g GameData) bool {
	if pluginStarts.InitSDKCalls == nil {
		missing("InitSDKCalls")
	}
	return pluginStarts.InitSDKCalls(g)
}

// InitDHooks prepares every detour, and says whether it could. Still in
// dhooks.sp.
//
//sp:plugin InitDHooks
func InitDHooks(g GameData) bool {
	if pluginStarts.InitDHooks == nil {
		missing("InitDHooks")
	}
	return pluginStarts.InitDHooks(g)
}

// SetUpgradesAddress writes g_pMannVsMachineUpgrades.
//
//sp:globalset g_pMannVsMachineUpgrades
func SetUpgradesAddress(address Address) {
	if pluginStarts.SetUpgradesAddress == nil {
		missing("g_pMannVsMachineUpgrades")
	}
	pluginStarts.SetUpgradesAddress(address)
}

// LateLoad says the plugin was loaded onto a running server, so the world
// already exists and has to be looked at rather than waited for.
//
//sp:global g_bLateLoad
func LateLoad() bool {
	if pluginStarts.LateLoad == nil {
		missing("g_bLateLoad")
	}
	return pluginStarts.LateLoad()
}

// LoadFeatures publishes what this build can do. Ported, features.
//
//sp:plugin LoadFeatures
func LoadFeatures() {
	if pluginStarts.LoadFeatures == nil {
		missing("LoadFeatures")
	}
	pluginStarts.LoadFeatures()
}

// BluAssistInit prepares the BLU-side helper. Ported, bluassist.
//
//sp:body BluAssist_Init
func BluAssistInit() {
	if pluginStarts.BluAssistInit == nil {
		missing("BluAssist_Init")
	}
	pluginStarts.BluAssistInit()
}

// DebugFaultsInit prepares the fault watcher. Ported, faults.
//
//sp:body DebugFaults_Init
func DebugFaultsInit() {
	if pluginStarts.DebugFaultsInit == nil {
		missing("DebugFaults_Init")
	}
	pluginStarts.DebugFaultsInit()
}

// LoadLoadoutFunctions prepares the loadout tables. Ported, loadouts.
//
//sp:body LoadLoadoutFunctions
func LoadLoadoutFunctions() {
	if pluginStarts.LoadLoadoutFuncs == nil {
		missing("LoadLoadoutFunctions")
	}
	pluginStarts.LoadLoadoutFuncs()
}

// LoadPreferencesData reads the saved player preferences. Ported, playerpref.
//
//sp:body LoadPreferencesData
func LoadPreferencesData() {
	if pluginStarts.LoadPreferences == nil {
		missing("LoadPreferencesData")
	}
	pluginStarts.LoadPreferences()
}

// InitNextBotPathing prepares the path objects every bot keeps. Ported,
// botreset.
//
//sp:body InitNextBotPathing
func InitNextBotPathing() {
	if pluginStarts.InitNextBotPathing == nil {
		missing("InitNextBotPathing")
	}
	pluginStarts.InitNextBotPathing()
}

// ArchipelagoInit prepares the campaign seam. Ported, archipelago.
//
//sp:body Archipelago_Init
func ArchipelagoInit() {
	if pluginStarts.ArchipelagoInit == nil {
		missing("Archipelago_Init")
	}
	pluginStarts.ArchipelagoInit()
}

// InitGameEventHooks asks for every game event the mod watches. Ported,
// gameevents.
//
//sp:body InitGameEventHooks
func InitGameEventHooks() {
	if pluginStarts.InitGameEventHooks == nil {
		missing("InitGameEventHooks")
	}
	pluginStarts.InitGameEventHooks()
}

// InitMapConfig empties the record and makes its lists, which is the enum
// struct's own method.
//
//sp:plugin g_arrMapConfig.Initialize
func InitMapConfig() {
	if pluginStarts.InitMapConfig == nil {
		missing("g_arrMapConfig.Initialize")
	}
	pluginStarts.InitMapConfig()
}

// SetChosenBotClasses writes g_adtChosenBotClasses.
//
//sp:globalset g_adtChosenBotClasses
func SetChosenBotClasses(l List) {
	if pluginStarts.SetChosenClasses == nil {
		missing("g_adtChosenBotClasses")
	}
	pluginStarts.SetChosenClasses(l)
}

// SetChosenBotSeats writes g_adtChosenBotSeats.
//
//sp:globalset g_adtChosenBotSeats
func SetChosenBotSeats(l List) {
	if pluginStarts.SetChosenSeats == nil {
		missing("g_adtChosenBotSeats")
	}
	pluginStarts.SetChosenSeats(l)
}

// SetBotNames writes m_adtBotNames.
//
//sp:globalset m_adtBotNames
func SetBotNames(l List) {
	if pluginStarts.SetBotNames == nil {
		missing("m_adtBotNames")
	}
	pluginStarts.SetBotNames(l)
}

// NameMax is MAX_NAME_LENGTH, the block size the bot names list wants.
//
//sp:global MAX_NAME_LENGTH
func NameMax() int32 { return 32 }

// PlayerPrefPath is g_sPlayerPrefPath, the file the preferences are kept in.
//
//sp:global g_sPlayerPrefPath
func PlayerPrefPath() Text {
	if pluginStarts.SetPlayerPrefPath == nil {
		missing("g_sPlayerPrefPath")
	}
	return Text{}
}

// BuildPathInto is BuildPath writing into a buffer that already exists, which
// the preference path is: the plugin declares it and this fills it.
//
//sp:native BuildPath before Path_SM
func BuildPathInto(out Text, maxlen int32, format string) {
	if pluginStarts.BuildPathInto == nil {
		missing("BuildPath")
	}
	pluginStarts.BuildPathInto(out, maxlen, format)
}

// UpgradesAddress reads g_pMannVsMachineUpgrades, the table the station's
// upgrades are read out of.
//
//sp:global g_pMannVsMachineUpgrades
func UpgradesAddress() Address {
	if pluginStarts.UpgradesAddress == nil {
		missing("g_pMannVsMachineUpgrades")
	}
	return pluginStarts.UpgradesAddress()
}

// FindGameConsoleVariables looks up the game's own convars. Ported, seating.
//
//sp:body FindGameConsoleVariables
func FindGameConsoleVariables() {
	if pluginStarts.FindGameConsoleVariables == nil {
		missing("FindGameConsoleVariables")
	}
	pluginStarts.FindGameConsoleVariables()
}

/*
Asking to be loaded, and what the plugin offers once it is.

The six natives are the test bed's window into a bot: two facts, whether it is
pathing and how far it has to walk, that look identical from every angle a
watcher has and that only this plugin can tell apart.
*/

// AplRes is APLRes, the answer AskPluginLoad2 gives.
//
//sp:tag APLRes
type AplRes int32

// AplResSuccess is APLRes_Success.
//
//sp:global APLRes_Success
func AplResSuccess() AplRes { return 0 }

// SetLateLoad writes g_bLateLoad.
//
//sp:globalset g_bLateLoad
func SetLateLoad(late bool) {
	if pluginStarts.SetLateLoad == nil {
		missing("g_bLateLoad")
	}
	pluginStarts.SetLateLoad(late)
}

// CreateNative offers one under that name.
//
//sp:native CreateNative
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func CreateNative(name string, callback func(plugin int32, params int32) int32) {
	if pluginStarts.CreateNative == nil {
		missing("CreateNative")
	}
	pluginStarts.CreateNative(name)
}

// MarkNativeAsOptional says a native this plugin calls is allowed to be
// missing, which without it fails the whole load rather than the call.
//
//sp:native MarkNativeAsOptional
func MarkNativeAsOptional(name string) {
	if pluginStarts.MarkNativeAsOptional == nil {
		missing("MarkNativeAsOptional")
	}
	pluginStarts.MarkNativeAsOptional(name)
}

// RegPluginLibrary publishes the library name other plugins test for.
//
//sp:native RegPluginLibrary
func RegPluginLibrary(name string) {
	if pluginStarts.RegPluginLibrary == nil {
		missing("RegPluginLibrary")
	}
	pluginStarts.RegPluginLibrary(name)
}

// NativeGetPathLength is the native of that name. Ported, statnatives.
//
//sp:callback Native_GetPathLength
//nolint:revive // unused-parameter: a name handed to CreateNative, never called
func NativeGetPathLength(plugin int32, params int32) int32 { return 0 }

// NativeIsPathing is the native of that name. Ported, statnatives.
//
//sp:callback Native_IsPathing
//nolint:revive // unused-parameter: a name handed to CreateNative, never called
func NativeIsPathing(plugin int32, params int32) int32 { return 0 }

// NativePathFailed is the native of that name. Ported, statnatives.
//
//sp:callback Native_PathFailed
//nolint:revive // unused-parameter: a name handed to CreateNative, never called
func NativePathFailed(plugin int32, params int32) int32 { return 0 }

// NativePathFailures is the native of that name. Ported, statnatives.
//
//sp:callback Native_PathFailures
//nolint:revive // unused-parameter: a name handed to CreateNative, never called
func NativePathFailures(plugin int32, params int32) int32 { return 0 }

// NativeRangeRepairStalls is the native of that name. Ported, statnatives.
//
//sp:callback Native_RangeRepairStalls
//nolint:revive // unused-parameter: a name handed to CreateNative, never called
func NativeRangeRepairStalls(plugin int32, params int32) int32 { return 0 }

// NativeGetAttackTarget is the native of that name. Ported, statnatives.
//
//sp:callback Native_GetAttackTarget
//nolint:revive // unused-parameter: a name handed to CreateNative, never called
func NativeGetAttackTarget(plugin int32, params int32) int32 { return 0 }

// MapConfigRecord is esMapConfiguration, the per-map record. An enum struct
// with methods on it, which the generator has no form for, so the plugin keeps
// the type and this names it.
//
//sp:tag esMapConfiguration
type MapConfigRecord int32
