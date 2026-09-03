package engine

/*
Standing in front of a function the game calls.

A native or an SDKCall is a call the plugin makes. A detour and a hook are the
other direction: the engine enters the plugin, with the arguments in a
DHookParam and the answer expected in a DHookReturn, and the callback says
whether it answered for the game or let it through.
*/

// DHookCalls are the answers.
type DHookCalls struct {
	DetourFromConf       func(g GameData, name string) Detour
	EnableDetour         func(d Detour, when int32)
	CloseDetour          func(d Detour)
	HookFromConf         func(g GameData, name string) Hook
	HookEntity           func(h Hook, when int32, entity int32)
	HookRaw              func(h Hook, when int32, address Address)
	ParamAt              func(p DHookParam, index int32) int32
	SetReturnBool        func(r DHookReturn, value bool)
	SetGameRulesProp     func(prop string, value int32)
	SetPlayerClass       func(client int32, class Class, weapons bool, persist bool)
	VisionBot            func(vision Address) Bot
	SetTouchCredits      func(on bool)
	TouchCredits         func() bool
	SetPlayerKilled      func(on bool)
	PlayerKilled         func() bool
	SetEngineerKilled    func(on bool)
	EngineerKilled       func() bool
	SetSpyKilled         func(on bool)
	HasCustomLoadout     func(client int32) bool
	PrepareCustomLoadout func(client int32)
}

var dhooks DHookCalls

// InstallDHooks puts a set of answers behind them.
func InstallDHooks(c DHookCalls) func() {
	previous := dhooks
	dhooks = c
	return func() { dhooks = previous }
}

// Detour is a function the plugin stands in front of everywhere it is called.
//
//sp:tag DynamicDetour
type Detour int32

// Hook is a virtual the plugin stands in front of on one object at a time.
//
//sp:tag DynamicHook
type Hook int32

// DHookReturn is what the hooked function is about to answer.
//
//sp:tag DHookReturn
type DHookReturn int32

// DHookParam is the arguments it was called with.
//
//sp:tag DHookParam
type DHookParam int32

// Mres is MRESReturn, whether the callback answered for the game.
//
//sp:tag MRESReturn
type Mres int32

// MresIgnored is MRES_Ignored: let the game carry on.
//
//sp:global MRES_Ignored
func MresIgnored() Mres { return 0 }

// MresSupercede is MRES_Supercede: the callback answered, so the game does not
// run at all.
//
//sp:global MRES_Supercede
func MresSupercede() Mres { return 0 }

// HookPre is Hook_Pre, before the function runs.
//
//sp:global Hook_Pre
func HookPre() int32 { return 0 }

// HookPost is Hook_Post, after it does.
//
//sp:global Hook_Post
func HookPost() int32 { return 0 }

// NoDetour is null, what a detour that could not be built is.
//
//sp:global null
func NoDetour() Detour { return 0 }

// NoHook is null, the same for a hook.
//
//sp:global null
func NoHook() Hook { return 0 }

// DetourFromConf builds one from the gamedata file.
//
//sp:native DynamicDetour.FromConf
func DetourFromConf(g GameData, name string) Detour {
	if dhooks.DetourFromConf == nil {
		missing("DynamicDetour.FromConf")
	}
	return dhooks.DetourFromConf(g, name)
}

// Close releases the detour handle. The detour itself outlives it.
//
//sp:delete Close
func (d Detour) Close() {
	if dhooks.CloseDetour == nil {
		missing("delete DynamicDetour")
	}
	dhooks.CloseDetour(d)
}

// HookFromConf builds a virtual hook from the gamedata file.
//
//sp:native DynamicHook.FromConf
func HookFromConf(g GameData, name string) Hook {
	if dhooks.HookFromConf == nil {
		missing("DynamicHook.FromConf")
	}
	return dhooks.HookFromConf(g, name)
}

// Get is one argument the hooked function was called with.
//
//sp:method Get
func (p DHookParam) Get(index int32) int32 {
	if dhooks.ParamAt == nil {
		missing("DHookParam.Get")
	}
	return dhooks.ParamAt(p, index)
}

// SetBool writes the answer the callback is giving for the game.
//
//sp:propertyset Value
func (r DHookReturn) SetBool(value bool) {
	if dhooks.SetReturnBool == nil {
		missing("DHookReturn.Value")
	}
	dhooks.SetReturnBool(r, value)
}

// SetGameRulesProp writes a game rules property, which is how the mod lies to
// the vision code about whether this is Mann vs Machine.
//
//sp:native GameRules_SetProp
func SetGameRulesProp(prop string, value int32) {
	if dhooks.SetGameRulesProp == nil {
		missing("GameRules_SetProp")
	}
	dhooks.SetGameRulesProp(prop, value)
}

// SetPlayerClass moves a player to another class.
//
//sp:native TF2_SetPlayerClass
func SetPlayerClass(client int32, class Class, weapons bool, persist bool) {
	if dhooks.SetPlayerClass == nil {
		missing("TF2_SetPlayerClass")
	}
	dhooks.SetPlayerClass(client, class, weapons, persist)
}

// VisionBot is the bot a vision interface belongs to.
//
//sp:cast IVision
func VisionBot(vision Address) Vision {
	return Vision(vision)
}

// SetTouchCredits writes m_bTouchCredits.
//
//sp:globalset m_bTouchCredits
func SetTouchCredits(on bool) {
	if dhooks.SetTouchCredits == nil {
		missing("m_bTouchCredits")
	}
	dhooks.SetTouchCredits(on)
}

// TouchCredits reads it.
//
//sp:global m_bTouchCredits
func TouchCredits() bool {
	if dhooks.TouchCredits == nil {
		missing("m_bTouchCredits")
	}
	return dhooks.TouchCredits()
}

// SetPlayerKilled writes m_bPlayerKilled.
//
//sp:globalset m_bPlayerKilled
func SetPlayerKilled(on bool) {
	if dhooks.SetPlayerKilled == nil {
		missing("m_bPlayerKilled")
	}
	dhooks.SetPlayerKilled(on)
}

// PlayerKilled reads it.
//
//sp:global m_bPlayerKilled
func PlayerKilled() bool {
	if dhooks.PlayerKilled == nil {
		missing("m_bPlayerKilled")
	}
	return dhooks.PlayerKilled()
}

// SetEngineerKilled writes m_bEngineerKilled.
//
//sp:globalset m_bEngineerKilled
func SetEngineerKilled(on bool) {
	if dhooks.SetEngineerKilled == nil {
		missing("m_bEngineerKilled")
	}
	dhooks.SetEngineerKilled(on)
}

// EngineerKilled reads it.
//
//sp:global m_bEngineerKilled
func EngineerKilled() bool {
	if dhooks.EngineerKilled == nil {
		missing("m_bEngineerKilled")
	}
	return dhooks.EngineerKilled()
}

// SetSpyKilled writes g_bSpyKilled.
//
//sp:globalset g_bSpyKilled
func SetSpyKilled(on bool) {
	if dhooks.SetSpyKilled == nil {
		missing("g_bSpyKilled")
	}
	dhooks.SetSpyKilled(on)
}

// HasCustomLoadout says the bot's weapons have already been worked out.
//
//sp:slot g_bHasCustomLoadout
func HasCustomLoadout(client int32) bool {
	if dhooks.HasCustomLoadout == nil {
		missing("g_bHasCustomLoadout")
	}
	return dhooks.HasCustomLoadout(client)
}

// PrepareCustomLoadout works them out. Ported, loadouts.
//
//sp:body PrepareCustomLoadout
func PrepareCustomLoadout(client int32) {
	if dhooks.PrepareCustomLoadout == nil {
		missing("PrepareCustomLoadout")
	}
	dhooks.PrepareCustomLoadout(client)
}

// ConditionImmuneToPushback is TFCond_ImmuneToPushback.
//
//sp:global TFCond_ImmuneToPushback
func ConditionImmuneToPushback() Condition { return 0 }

// TimerGiveCustomLoadout is the callback that hands a bot its weapons a tick
// later. Ported, loadouts.
//
//sp:callback Timer_GiveCustomLoadout
//nolint:revive // unused-parameter: a name handed to CreateTimer, never called
func TimerGiveCustomLoadout(timer Timer, client int32) Outcome { return 0 }

/*
The callback names the hook registrations hand out.

SourcePawn has one DHookCallback type covering every shape a callback can take,
so these all declare the same Go signature: what matters is the name, and the
real signature is on the generated function itself.
*/

// DHookCallback is a callback's name, which SourcePawn compares directly
// against INVALID_FUNCTION and hands to a registration.
//
//sp:tag DHookCallback
type DHookCallback int32

// InvalidFunction is INVALID_FUNCTION, which is how a registration says it
// wants only one side of the pair.
//
//sp:global INVALID_FUNCTION
func InvalidFunction() DHookCallback { return 0 }

// DHookLoadUpgradesFilePost is the callback of that name.
//
//sp:global DHookCallback_LoadUpgradesFile_Post
func DHookLoadUpgradesFilePost() DHookCallback { return 0 }

// DHookManageRegularWeaponsPre is the callback of that name.
//
//sp:global DHookCallback_ManageRegularWeapons_Pre
func DHookManageRegularWeaponsPre() DHookCallback { return 0 }

// DHookManageRegularWeaponsPost is the callback of that name.
//
//sp:global DHookCallback_ManageRegularWeapons_Post
func DHookManageRegularWeaponsPost() DHookCallback { return 0 }

// DHookManageBuilderWeaponsPre is the callback of that name.
//
//sp:global DHookCallback_ManageBuilderWeapons_Pre
func DHookManageBuilderWeaponsPre() DHookCallback { return 0 }

// DHookMyTouchPre is the callback of that name.
//
//sp:global DHookCallback_MyTouch_Pre
func DHookMyTouchPre() DHookCallback { return 0 }

// DHookMyTouchPost is the callback of that name.
//
//sp:global DHookCallback_MyTouch_Post
func DHookMyTouchPost() DHookCallback { return 0 }

// DHookIsBotPre is the callback of that name.
//
//sp:global DHookCallback_IsBot_Pre
func DHookIsBotPre() DHookCallback { return 0 }

// DHookEventKilledPre is the callback of that name.
//
//sp:global DHookCallback_EventKilled_Pre
func DHookEventKilledPre() DHookCallback { return 0 }

// DHookEventKilledPost is the callback of that name.
//
//sp:global DHookCallback_EventKilled_Post
func DHookEventKilledPost() DHookCallback { return 0 }

// DHookIsVisibleEntityNoticedPre is the callback of that name.
//
//sp:global DHookCallback_IsVisibleEntityNoticed_Pre
func DHookIsVisibleEntityNoticedPre() DHookCallback { return 0 }

// DHookIsVisibleEntityNoticedPost is the callback of that name.
//
//sp:global DHookCallback_IsVisibleEntityNoticed_Post
func DHookIsVisibleEntityNoticedPost() DHookCallback { return 0 }

// DHookIsIgnoredPre is the callback of that name.
//
//sp:global DHookCallback_IsIgnored_Pre
func DHookIsIgnoredPre() DHookCallback { return 0 }

// HookMyTouch is m_hMyTouch.
//
//sp:global m_hMyTouch
func HookMyTouch() Hook { return 0 }

// HookIsBot is m_hIsBot.
//
//sp:global m_hIsBot
func HookIsBot() Hook { return 0 }

// HookEventKilled is m_hEventKilled.
//
//sp:global m_hEventKilled
func HookEventKilled() Hook { return 0 }

// HookIsVisibleEntityNoticed is m_hIsVisibleEntityNoticed.
//
//sp:global m_hIsVisibleEntityNoticed
func HookIsVisibleEntityNoticed() Hook { return 0 }

// HookIsIgnored is m_hIsIgnored.
//
//sp:global m_hIsIgnored
func HookIsIgnored() Hook { return 0 }

// Enable turns on one side of a detour.
//
//sp:method Enable
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func (d Detour) Enable(when int32, callback DHookCallback) {
	if dhooks.EnableDetour == nil {
		missing("DynamicDetour.Enable")
	}
	dhooks.EnableDetour(d, when)
}

// SameCallback is != between two callback names, which SourcePawn compares
// directly and Go will not.
//
//sp:same SameCallback
//nolint:revive // unused-parameter: the emitter writes the comparison, not a call
func SameCallback(a func(pThis int32) Mres, b func(pThis int32) Mres) bool { return false }

// HookEntity arms the hook on one entity.
//
//sp:method HookEntity
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func (h Hook) HookEntity(when int32, entity int32, callback DHookCallback) {
	if dhooks.HookEntity == nil {
		missing("DynamicHook.HookEntity")
	}
	dhooks.HookEntity(h, when, entity)
}

// HookRaw arms it on something that is not an entity, by address, which is how
// a vision interface is reached.
//
//sp:method HookRaw
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func (h Hook) HookRaw(when int32, address Address, callback DHookCallback) {
	if dhooks.HookRaw == nil {
		missing("DynamicHook.HookRaw")
	}
	dhooks.HookRaw(h, when, address)
}

// AddressOfVision is the vision interface read as a memory address, which is
// what HookRaw takes.
//
//sp:cast Address
func AddressOfVision(v Vision) Address {
	return Address(v)
}

// Bot is the bot a vision interface belongs to.
//
//sp:method GetBot
func (v Vision) Bot() Bot {
	if dhooks.VisionBot == nil {
		missing("IVision.GetBot")
	}
	return dhooks.VisionBot(Address(v))
}
