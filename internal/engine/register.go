package engine

/*
Registering a callback that is not a timer: a console command, a game event.

CreateTimer already takes its callback by name, and this is the same shape for
the other two. The function is a value in the one way the subset allows: named,
as an argument to an extern that declares it takes one. A name in a string would
throw away the one thing worth checking, which is that the callback has the
signature the registration expects.
*/

// RegisterCalls are the answers.
type RegisterCalls struct {
	RegConsoleCmd                  func(name string)
	HookEvent                      func(name string)
	CreateTimerWith                func(interval float32, data int32, flags int32) Timer
	ResetIntentionInterface        func(client int32)
	SetShoppedThisBreak            func(client int32, shopped bool)
	SetBeingRevived                func(client int32, reviving bool)
	EventInt                       func(e Event, key string) int32
	EventBool                      func(e Event, key string) bool
	ResetSpyIntel                  func()
	SetupSniperSpotHints           func()
	NestRelocationResetAll         func()
	DebugFaultsOnWaveStart         func()
	DebugFaultsOnWaveStartEmpty    func()
	PublishActiveFeatures          func()
	ThreatPortAuditReport          func()
	NestRelocationStopEvaluating   func()
	TeleporterForgetGivingUp       func()
	DisposableForgetGivingUp       func()
	QueueBehaviourReset            func()
	RemoveOrphanedWearables        func()
	ManageDefenderBots             func(force bool)
	FreeChosenBotTeam              func()
	HasUpgraded                    func(client int32) bool
	SetHasUpgraded                 func(client int32, upgraded bool)
	SetHasBoughtUpgrades           func(client int32, bought bool)
	GrantOrRemoveAllUpgrades       func(client int32, remove bool, refund bool)
	UpdateChosenBotTeamComposition func()
	ReseatOnBreak                  func()
	SetNextReadyTime               func(when float32)
	RemoveAllDefenderBots          func(reason string)
}

var registrations RegisterCalls

// InstallRegistrations puts a set of answers behind them.
func InstallRegistrations(c RegisterCalls) func() {
	previous := registrations
	registrations = c
	return func() { registrations = previous }
}

// RegConsoleCmd puts a command on the console, taking its callback by name.
//
//sp:native RegConsoleCmd
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func RegConsoleCmd(name string, callback func(client int32, args int32) Outcome) {
	if registrations.RegConsoleCmd == nil {
		missing("RegConsoleCmd")
	}
	registrations.RegConsoleCmd(name)
}

// HookEvent listens for a game event, taking its callback by name.
//
//sp:native HookEvent
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func HookEvent(name string, callback func(event Event, name string, dontBroadcast bool) Outcome) {
	if registrations.HookEvent == nil {
		missing("HookEvent")
	}
	registrations.HookEvent(name)
}

/*
CreateTimerWith is CreateTimer with the cell a timer carries to its callback.

The same native as the one in engineer.go, declared again because the callback
takes the cell: a timer that carries a userid and one that carries nothing have
different callbacks, and the signature is the thing worth checking.
*/
//
//sp:native CreateTimer
//
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func CreateTimerWith(interval float32, callback func(timer Timer, data int32) Outcome, data int32, flags int32) Timer {
	if registrations.CreateTimerWith == nil {
		missing("CreateTimer")
	}
	return registrations.CreateTimerWith(interval, data, flags)
}

// ResetIntentionInterface makes the bot throw away what it was doing and decide
// again from the top.
//
//sp:plugin ResetIntentionInterface
func ResetIntentionInterface(client int32) {
	if registrations.ResetIntentionInterface == nil {
		missing("ResetIntentionInterface")
	}
	registrations.ResetIntentionInterface(client)
}

// SetShoppedThisBreak writes the shopping flag, which a break clears for
// everybody.
//
//sp:slotset g_bShoppedThisBreak
func SetShoppedThisBreak(client int32, shopped bool) {
	if registrations.SetShoppedThisBreak == nil {
		missing("g_bShoppedThisBreak")
	}
	registrations.SetShoppedThisBreak(client, shopped)
}

// SetBeingRevived says somebody has a revive marker up over this player.
//
//sp:slotset g_bIsBeingRevived
func SetBeingRevived(client int32, reviving bool) {
	if registrations.SetBeingRevived == nil {
		missing("g_bIsBeingRevived")
	}
	registrations.SetBeingRevived(client, reviving)
}

// SpyKilled is g_bSpyKilled, set while a defender spy's death is being handled.
//
//sp:global g_bSpyKilled
func SpyKilled() bool { return false }

// EventInt reads a number off a game event.
//
//sp:method GetInt
func (e Event) EventInt(key string) int32 {
	if registrations.EventInt == nil {
		missing("Event.GetInt")
	}
	return registrations.EventInt(e, key)
}

// EventBool reads a flag off one.
//
//sp:method GetBool
func (e Event) EventBool(key string) bool {
	if registrations.EventBool == nil {
		missing("Event.GetBool")
	}
	return registrations.EventBool(e, key)
}

// ResetSpyIntel forgets what the bots think they know about spies. Ported,
// spycheck.
//
//sp:body ResetSpyIntel
func ResetSpyIntel() {
	if registrations.ResetSpyIntel == nil {
		missing("ResetSpyIntel")
	}
	registrations.ResetSpyIntel()
}

// SetupSniperSpotHints reads the map's sniping spots again.
//
//sp:plugin SetupSniperSpotHints
func SetupSniperSpotHints() {
	if registrations.SetupSniperSpotHints == nil {
		missing("SetupSniperSpotHints")
	}
	registrations.SetupSniperSpotHints()
}

// NestRelocationResetAll forgets every engineer's relocation state. Ported,
// engineeridle.
//
//sp:body EngineerNestRelocation_ResetAll
func NestRelocationResetAll() {
	if registrations.NestRelocationResetAll == nil {
		missing("EngineerNestRelocation_ResetAll")
	}
	registrations.NestRelocationResetAll()
}

// DebugFaultsOnWaveStart starts the fault trace, which does nothing unless a
// debug convar is set. Ported, faults.
//
//sp:body DebugFaults_OnWaveStart
func DebugFaultsOnWaveStart() {
	if registrations.DebugFaultsOnWaveStart == nil {
		missing("DebugFaults_OnWaveStart")
	}
	registrations.DebugFaultsOnWaveStart()
}

// DebugFaultsOnWaveStartEmpty is the same for a wave that begins with no bots.
// Ported, faults.
//
//sp:body DebugFaults_OnWaveStartEmpty
func DebugFaultsOnWaveStartEmpty() {
	if registrations.DebugFaultsOnWaveStartEmpty == nil {
		missing("DebugFaults_OnWaveStartEmpty")
	}
	registrations.DebugFaultsOnWaveStartEmpty()
}

// PublishActiveFeatures says which feature switches are on, late enough that
// server.cfg has certainly run.
//
//sp:plugin PublishActiveFeatures
func PublishActiveFeatures() {
	if registrations.PublishActiveFeatures == nil {
		missing("PublishActiveFeatures")
	}
	registrations.PublishActiveFeatures()
}

// ThreatPortAuditReport says what the threat port disagreed about.
//
//sp:plugin ThreatPortAudit_Report
func ThreatPortAuditReport() {
	if registrations.ThreatPortAuditReport == nil {
		missing("ThreatPortAudit_Report")
	}
	registrations.ThreatPortAuditReport()
}

// NestRelocationStopEvaluating drops whatever the relocation queue has left,
// because it is about a bomb that is about to move. Ported, engineeridle.
//
//sp:body EngineerNestRelocation_StopEvaluating
func NestRelocationStopEvaluating() {
	if registrations.NestRelocationStopEvaluating == nil {
		missing("EngineerNestRelocation_StopEvaluating")
	}
	registrations.NestRelocationStopEvaluating()
}

// TeleporterForgetGivingUp gives the exit spot another chance. Ported,
// engineerbuildteleporter.
//
//sp:body EngineerTeleporter_ForgetGivingUp
func TeleporterForgetGivingUp() {
	if registrations.TeleporterForgetGivingUp == nil {
		missing("EngineerTeleporter_ForgetGivingUp")
	}
	registrations.TeleporterForgetGivingUp()
}

// DisposableForgetGivingUp is the same for the disposable sentry. Ported,
// engineerbuilddisposable.
//
//sp:body EngineerDisposable_ForgetGivingUp
func DisposableForgetGivingUp() {
	if registrations.DisposableForgetGivingUp == nil {
		missing("EngineerDisposable_ForgetGivingUp")
	}
	registrations.DisposableForgetGivingUp()
}

// QueueBehaviourReset drains a rethink across the clients. Ported,
// behaviourreset.
//
//sp:body QueueBehaviourReset
func QueueBehaviourReset() {
	if registrations.QueueBehaviourReset == nil {
		missing("QueueBehaviourReset")
	}
	registrations.QueueBehaviourReset()
}

// RemoveOrphanedWearables sweeps up the hats the game refused. Ported,
// cosmetics.
//
//sp:body RemoveOrphanedWearables
func RemoveOrphanedWearables() {
	if registrations.RemoveOrphanedWearables == nil {
		missing("RemoveOrphanedWearables")
	}
	registrations.RemoveOrphanedWearables()
}

// ManageDefenderBots adds or removes bots to match what the server asked for.
//
//sp:plugin ManageDefenderBots
func ManageDefenderBots(force bool) {
	if registrations.ManageDefenderBots == nil {
		missing("ManageDefenderBots")
	}
	registrations.ManageDefenderBots(force)
}

// FreeChosenBotTeam drops the lineup the players picked, which the bots on the
// field have already been built from.
//
//sp:plugin FreeChosenBotTeam
func FreeChosenBotTeam() {
	if registrations.FreeChosenBotTeam == nil {
		missing("FreeChosenBotTeam")
	}
	registrations.FreeChosenBotTeam()
}

// ManagerMode is redbots_manager_mode, which says who starts the bots.
//
//sp:global redbots_manager_mode
func ManagerMode() ConVar { return 0 }

// ManagerModeAutoBots is MANAGER_MODE_AUTO_BOTS, the mode where the mod starts
// them itself.
//
//sp:global MANAGER_MODE_AUTO_BOTS
func ManagerModeAutoBots() int32 { return 0 }

// KeepBotUpgrades is redbots_manager_keep_bot_upgrades, whether a failed wave
// leaves the bots what they bought.
//
//sp:global redbots_manager_keep_bot_upgrades
func KeepBotUpgrades() ConVar { return 0 }

// BotLineupMode is redbots_manager_bot_lineup_mode, how the team is composed.
//
//sp:global redbots_manager_bot_lineup_mode
func BotLineupMode() ConVar { return 0 }

// LineupModeChoose is BOT_LINEUP_MODE_CHOOSE, the mode where the players pick.
//
//sp:global BOT_LINEUP_MODE_CHOOSE
func LineupModeChoose() int32 { return 0 }

// HasUpgraded says this bot has been through the upgrade station.
//
//sp:slot g_bHasUpgraded
func HasUpgraded(client int32) bool {
	if registrations.HasUpgraded == nil {
		missing("g_bHasUpgraded")
	}
	return registrations.HasUpgraded(client)
}

// SetHasUpgraded writes it.
//
//sp:slotset g_bHasUpgraded
func SetHasUpgraded(client int32, upgraded bool) {
	if registrations.SetHasUpgraded == nil {
		missing("g_bHasUpgraded")
	}
	registrations.SetHasUpgraded(client, upgraded)
}

// SetHasBoughtUpgrades writes the loadout side of the same fact. Ported,
// loadouts.
//
//sp:slotset g_bHasBoughtUpgrades
func SetHasBoughtUpgrades(client int32, bought bool) {
	if registrations.SetHasBoughtUpgrades == nil {
		missing("g_bHasBoughtUpgrades")
	}
	registrations.SetHasBoughtUpgrades(client, bought)
}

// GrantOrRemoveAllUpgrades makes the population manager forget what this bot
// bought, so it can go and buy again.
//
//sp:native VS_GrantOrRemoveAllUpgrades
func GrantOrRemoveAllUpgrades(client int32, remove bool, refund bool) {
	if registrations.GrantOrRemoveAllUpgrades == nil {
		missing("VS_GrantOrRemoveAllUpgrades")
	}
	registrations.GrantOrRemoveAllUpgrades(client, remove, refund)
}

// UpdateChosenBotTeamComposition works out the lineup the next wave gets.
//
//sp:plugin UpdateChosenBotTeamComposition
func UpdateChosenBotTeamComposition() {
	if registrations.UpdateChosenBotTeamComposition == nil {
		missing("UpdateChosenBotTeamComposition")
	}
	registrations.UpdateChosenBotTeamComposition()
}

// ReseatOnBreak applies a lineup that was retyped mid-wave, which is held until
// the break.
//
//sp:plugin Reseat_OnBreak
func ReseatOnBreak() {
	if registrations.ReseatOnBreak == nil {
		missing("Reseat_OnBreak")
	}
	registrations.ReseatOnBreak()
}

// KickBots is redbots_manager_kick_bots, whether a break clears the team out.
//
//sp:global redbots_manager_kick_bots
func KickBots() ConVar { return 0 }

// ReadyCooldown is redbots_manager_ready_cooldown, how long before the players
// may ready up again.
//
//sp:global redbots_manager_ready_cooldown
func ReadyCooldown() ConVar { return 0 }

// ManagerModeReadyBots is MANAGER_MODE_READY_BOTS, the mode where the players
// ready up.
//
//sp:global MANAGER_MODE_READY_BOTS
func ManagerModeReadyBots() int32 { return 0 }

// NextReadyTime is g_flNextReadyTime, the clock the global ready cooldown ends
// on.
//
//sp:global g_flNextReadyTime
func NextReadyTime() float32 { return 0 }

// SetNextReadyTime writes it.
//
//sp:globalset g_flNextReadyTime
func SetNextReadyTime(when float32) {
	if registrations.SetNextReadyTime == nil {
		missing("g_flNextReadyTime")
	}
	registrations.SetNextReadyTime(when)
}

// RemoveAllDefenderBots clears the team out, saying why.
//
//sp:plugin RemoveAllDefenderBots
func RemoveAllDefenderBots(reason string) {
	if registrations.RemoveAllDefenderBots == nil {
		missing("RemoveAllDefenderBots")
	}
	registrations.RemoveAllDefenderBots(reason)
}
