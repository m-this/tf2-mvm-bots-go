package engine

/*
The per-tick command a bot sends, and everything the mod changes about it.

This runs once for every bot on every tick, so what is here is what could not be
decided anywhere cheaper: buttons the behaviour layer asked to be held or held
off, the weapon-specific reasons to stop firing, and the aim help.
*/

// RunCmdCalls are the answers.
type RunCmdCalls struct {
	PluginBotSimulateFrame  func(client int32)
	IsPlayingHorn           func(weapon int32) bool
	LastAccuracyCheck       func(weapon int32) float32
	MonitorKnownEntities    func(client int32, vision Vision)
	UseWeaponAbilities      func(client int32, weapon int32, bot Bot, threat Known)
	UsePowerupBottle        func(client int32, weapon int32, bot Bot, threat Known)
	CanWeaponAirblast       func(weapon int32) bool
	UtilizeCompressionBlast func(client int32, bot Bot, threat Known, mode int32)
	ShouldBuybackIntoGame   func(client int32) bool
	PlayerBuyback           func(client int32)
	ObserverMode            func(client int32) int32
	NextSnipeFireTime       func(client int32) float32
	SetNextSnipeFireTime    func(client int32, when float32)
	DeadRethinkTime         func(client int32) float32
	NextRollTime            func(client int32) float32
	WatchDefenderSpawnExit  func(client int32)
	SelectTargetPoint       func(i Intention, entity int32) [3]float32
	FakeClientCommand       func(client int32, command string)
	RandomFloat             func(low float32, high float32) float32
}

var runCmds RunCmdCalls

// InstallRunCmds puts a set of answers behind them.
func InstallRunCmds(c RunCmdCalls) func() {
	previous := runCmds
	Fill(&c)
	runCmds = c
	return func() { runCmds = previous }
}

// ButtonBack is IN_BACK.
//
//sp:global IN_BACK
func ButtonBack() int32 { return 0 }

// ButtonTurnLeft is IN_LEFT, the keyboard turn and not the strafe.
//
//sp:global IN_LEFT
func ButtonTurnLeft() int32 { return 0 }

// ButtonTurnRight is IN_RIGHT.
//
//sp:global IN_RIGHT
func ButtonTurnRight() int32 { return 0 }

// ObserverModeFreezecam is OBS_MODE_FREEZECAM.
//
//sp:global OBS_MODE_FREEZECAM
func ObserverModeFreezecam() int32 { return 0 }

// ObserverModeDeathcam is OBS_MODE_DEATHCAM.
//
//sp:global OBS_MODE_DEATHCAM
func ObserverModeDeathcam() int32 { return 0 }

// PluginBotSimulateFrame runs a plugin-driven bot's frame. Ported, pathing.
//
//sp:body PluginBot_SimulateFrame
func PluginBotSimulateFrame(client int32) { runCmds.PluginBotSimulateFrame(client) }

// IsPlayingHorn says the buff banner is mid-blow, which is when holding fire
// stops meaning anything. Still in offsets.sp.
//
//sp:body IsPlayingHorn
func IsPlayingHorn(weapon int32) bool { return runCmds.IsPlayingHorn(weapon) }

// LastAccuracyCheck is when the revolver last decided its shot was accurate.
// Still in offsets.sp.
//
//sp:body GetLastAccuracyCheck
func LastAccuracyCheck(weapon int32) float32 { return runCmds.LastAccuracyCheck(weapon) }

// MonitorKnownEntities keeps the bot's memory of what it has seen current.
// Ported, botqueries.
//
//sp:body MonitorKnownEntities
func MonitorKnownEntities(client int32, vision Vision) { runCmds.MonitorKnownEntities(client, vision) }

// UseWeaponAbilities fires whatever the weapon offers for free. Ported,
// botqueries.
//
//sp:body OpportunisticallyUseWeaponAbilities
func UseWeaponAbilities(client int32, weapon int32, bot Bot, threat Known) {
	runCmds.UseWeaponAbilities(client, weapon, bot, threat)
}

// UsePowerupBottle drinks the canteen when it is worth drinking. Ported,
// bottle.
//
//sp:body OpportunisticallyUsePowerupBottle
func UsePowerupBottle(client int32, weapon int32, bot Bot, threat Known) {
	runCmds.UsePowerupBottle(client, weapon, bot, threat)
}

// CanWeaponAirblast says the flamethrower has the ammo for a blast. Ported,
// state.
//
//sp:body CanWeaponAirblast
func CanWeaponAirblast(weapon int32) bool { return runCmds.CanWeaponAirblast(weapon) }

// UtilizeCompressionBlast pushes what is in front of the bot. Ported,
// botqueries.
//
//sp:body UtilizeCompressionBlast
func UtilizeCompressionBlast(client int32, bot Bot, threat Known, mode int32) {
	runCmds.UtilizeCompressionBlast(client, bot, threat, mode)
}

// ShouldBuybackIntoGame says a dead bot has decided to pay its way back in.
// Ported, botqueries.
//
//sp:body ShouldBuybackIntoGame
func ShouldBuybackIntoGame(client int32) bool { return runCmds.ShouldBuybackIntoGame(client) }

// PlayerBuyback pays it. Ported, stocks.
//
//sp:body PlayerBuyback
func PlayerBuyback(client int32) { runCmds.PlayerBuyback(client) }

// ObserverMode is what a dead player is watching through, which decides
// whether buying back is possible at all.
//
//sp:native BasePlayer_GetObserverMode
func ObserverMode(client int32) int32 { return runCmds.ObserverMode(client) }

// AimSkill is redbots_manager_bot_aim_skill, how much the mod helps a bot aim.
//
//sp:global redbots_manager_bot_aim_skill
func AimSkill() ConVar { return 0 }

// RtdVariance is redbots_manager_bot_rtd_variance, the spread on how often a
// bot rolls the dice, and below COMMAND_MAX_RATE it never does.
//
//sp:global redbots_manager_bot_rtd_variance
func RtdVariance() ConVar { return 0 }

// SniperReactionTime is SNIPER_REACTION_TIME, the pause a sniper takes before
// firing at a threat it has just seen.
//
//sp:global SNIPER_REACTION_TIME
func SniperReactionTime() float32 { return 0.5 }

// NextSnipeFireTime reads m_flNextSnipeFireTime for one client.
//
//sp:slot m_flNextSnipeFireTime
func NextSnipeFireTime(client int32) float32 { return runCmds.NextSnipeFireTime(client) }

// SetNextSnipeFireTime writes it.
//
//sp:slotset m_flNextSnipeFireTime
func SetNextSnipeFireTime(client int32, when float32) { runCmds.SetNextSnipeFireTime(client, when) }

// DeadRethinkTime reads m_flDeadRethinkTime for one client.
//
//sp:slot m_flDeadRethinkTime
func DeadRethinkTime(client int32) float32 { return runCmds.DeadRethinkTime(client) }

// NextRollTime reads m_flNextRollTime for one client.
//
//sp:slot m_flNextRollTime
func NextRollTime(client int32) float32 { return runCmds.NextRollTime(client) }

// WatchDefenderSpawnExit notices a bot leaving spawn. Ported, spawnexit.
//
//sp:body WatchDefenderSpawnExit
func WatchDefenderSpawnExit(client int32) { runCmds.WatchDefenderSpawnExit(client) }

// SelectTargetPointOf is where the intention interface says to aim at that
// entity, which is the same answer the behaviour layer would give.
//
//sp:method SelectTargetPoint into
func (i Intention) SelectTargetPointOf(entity int32) (aimPos [3]float32) {
	return runCmds.SelectTargetPoint(i, entity)
}
