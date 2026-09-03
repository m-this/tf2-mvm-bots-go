package engine

/*
Creating the mod's own convars, which is what OnPluginStart spends most of its
lines on.

One setter per convar rather than one call taking a name: the name is what
SourcePawn writes on the left of the assignment and there is nothing generic to
hold. The readers are elsewhere, beside whatever reads them.
*/

// ConVarSetCalls are the answers.
type ConVarSetCalls struct {
	SetConVar     func(name string, c ConVar)
	CreateConVar  func(name string, def string, description string, flags int32) ConVar
	CreateBounded func(name string, def string, description string, flags int32, hasMin bool, low float32, hasMax bool, high float32) ConVar
	CreateFloored func(name string, def string, description string, flags int32, hasMin bool, low float32) ConVar
}

var conVarSets ConVarSetCalls

// InstallConVarSets puts a set of answers behind them.
func InstallConVarSets(c ConVarSetCalls) func() {
	previous := conVarSets
	Fill(&c)
	conVarSets = c
	return func() { conVarSets = previous }
}

// CreateConVar makes one with a description and flags.
//
//sp:native CreateConVar
func CreateConVar(name string, def string, description string, flags int32) ConVar {
	return conVarSets.CreateConVar(name, def, description, flags)
}

// CreateBoundedConVar makes one clamped at both ends.
//
//sp:native CreateConVar
func CreateBoundedConVar(name string, def string, description string, flags int32, hasMin bool, low float32, hasMax bool, high float32) ConVar {
	return conVarSets.CreateBounded(name, def, description, flags, hasMin, low, hasMax, high)
}

// CreateFlooredConVar makes one clamped at the bottom only.
//
//sp:native CreateConVar
func CreateFlooredConVar(name string, def string, description string, flags int32, hasMin bool, low float32) ConVar {
	return conVarSets.CreateFloored(name, def, description, flags, hasMin, low)
}

// FcvarNone is FCVAR_NONE.
//
//sp:global FCVAR_NONE
func FcvarNone() int32 { return 0 }

// FcvarNotify is FCVAR_NOTIFY, which tells everybody the value changed.
//
//sp:global FCVAR_NOTIFY
func FcvarNotify() int32 { return 0 }

// set records one, which is all a Go process can do with it.
func (c ConVarSetCalls) set(name string, value ConVar) {
	if c.SetConVar == nil {
		missing(name)
	}
	c.SetConVar(name, value)
}

// SetAimSkill writes redbots_manager_bot_aim_skill.
//
//sp:globalset redbots_manager_bot_aim_skill
func SetAimSkill(c ConVar) { conVarSets.set("redbots_manager_bot_aim_skill", c) }

// SetBackstabSkill writes redbots_manager_bot_backstab_skill.
//
//sp:globalset redbots_manager_bot_backstab_skill
func SetBackstabSkill(c ConVar) { conVarSets.set("redbots_manager_bot_backstab_skill", c) }

// SetBuyUpgradesChance writes redbots_manager_bot_buy_upgrades_chance.
//
//sp:globalset redbots_manager_bot_buy_upgrades_chance
func SetBuyUpgradesChance(c ConVar) { conVarSets.set("redbots_manager_bot_buy_upgrades_chance", c) }

// SetBuybackChance writes redbots_manager_bot_buyback_chance.
//
//sp:globalset redbots_manager_bot_buyback_chance
func SetBuybackChance(c ConVar) { conVarSets.set("redbots_manager_bot_buyback_chance", c) }

// SetBotHatEffects writes redbots_manager_bot_hat_effects.
//
//sp:globalset redbots_manager_bot_hat_effects
func SetBotHatEffects(c ConVar) { conVarSets.set("redbots_manager_bot_hat_effects", c) }

// SetBotHats writes redbots_manager_bot_hats.
//
//sp:globalset redbots_manager_bot_hats
func SetBotHats(c ConVar) { conVarSets.set("redbots_manager_bot_hats", c) }

// SetHearSpyRange writes redbots_manager_bot_hear_spy_range.
//
//sp:globalset redbots_manager_bot_hear_spy_range
func SetHearSpyRange(c ConVar) { conVarSets.set("redbots_manager_bot_hear_spy_range", c) }

// SetBotLineupMode writes redbots_manager_bot_lineup_mode.
//
//sp:globalset redbots_manager_bot_lineup_mode
func SetBotLineupMode(c ConVar) { conVarSets.set("redbots_manager_bot_lineup_mode", c) }

// SetMaxTankAttackers writes redbots_manager_bot_max_tank_attackers.
//
//sp:globalset redbots_manager_bot_max_tank_attackers
func SetMaxTankAttackers(c ConVar) { conVarSets.set("redbots_manager_bot_max_tank_attackers", c) }

// SetNoticeSpyTime writes redbots_manager_bot_notice_spy_time.
//
//sp:globalset redbots_manager_bot_notice_spy_time
func SetNoticeSpyTime(c ConVar) { conVarSets.set("redbots_manager_bot_notice_spy_time", c) }

// SetReflectChance writes redbots_manager_bot_reflect_chance.
//
//sp:globalset redbots_manager_bot_reflect_chance
func SetReflectChance(c ConVar) { conVarSets.set("redbots_manager_bot_reflect_chance", c) }

// SetReflectSkill writes redbots_manager_bot_reflect_skill.
//
//sp:globalset redbots_manager_bot_reflect_skill
func SetReflectSkill(c ConVar) { conVarSets.set("redbots_manager_bot_reflect_skill", c) }

// SetRequestCredits writes redbots_manager_bot_request_credits.
//
//sp:globalset redbots_manager_bot_request_credits
func SetRequestCredits(c ConVar) { conVarSets.set("redbots_manager_bot_request_credits", c) }

// SetRtdVariance writes redbots_manager_bot_rtd_variance.
//
//sp:globalset redbots_manager_bot_rtd_variance
func SetRtdVariance(c ConVar) { conVarSets.set("redbots_manager_bot_rtd_variance", c) }

// SetUpgradeInterval writes redbots_manager_bot_upgrade_interval.
//
//sp:globalset redbots_manager_bot_upgrade_interval
func SetUpgradeInterval(c ConVar) { conVarSets.set("redbots_manager_bot_upgrade_interval", c) }

// SetUseUpgrades writes redbots_manager_bot_use_upgrades.
//
//sp:globalset redbots_manager_bot_use_upgrades
func SetUseUpgrades(c ConVar) { conVarSets.set("redbots_manager_bot_use_upgrades", c) }

// SetClassBlacklist writes redbots_manager_class_blacklist.
//
//sp:globalset redbots_manager_class_blacklist
func SetClassBlacklist(c ConVar) { conVarSets.set("redbots_manager_class_blacklist", c) }

// SetManagerDebug writes redbots_manager_debug.
//
//sp:globalset redbots_manager_debug
func SetManagerDebug(c ConVar) { conVarSets.set("redbots_manager_debug", c) }

// SetDebugActions writes redbots_manager_debug_actions.
//
//sp:globalset redbots_manager_debug_actions
func SetDebugActions(c ConVar) { conVarSets.set("redbots_manager_debug_actions", c) }

// SetDefenderTeamSize writes redbots_manager_defender_team_size.
//
//sp:globalset redbots_manager_defender_team_size
func SetDefenderTeamSize(c ConVar) { conVarSets.set("redbots_manager_defender_team_size", c) }

// SetNestDepth writes redbots_manager_engineer_nest_depth.
//
//sp:globalset redbots_manager_engineer_nest_depth
func SetNestDepth(c ConVar) { conVarSets.set("redbots_manager_engineer_nest_depth", c) }

// SetNestRelocateScoreGainMin writes redbots_manager_engineer_nest_relocate_score_gain_min.
//
//sp:globalset redbots_manager_engineer_nest_relocate_score_gain_min
func SetNestRelocateScoreGainMin(c ConVar) {
	conVarSets.set("redbots_manager_engineer_nest_relocate_score_gain_min", c)
}

// SetExtraBots writes redbots_manager_extra_bots.
//
//sp:globalset redbots_manager_extra_bots
func SetExtraBots(c ConVar) { conVarSets.set("redbots_manager_extra_bots", c) }

// SetKeepBotUpgrades writes redbots_manager_keep_bot_upgrades.
//
//sp:globalset redbots_manager_keep_bot_upgrades
func SetKeepBotUpgrades(c ConVar) { conVarSets.set("redbots_manager_keep_bot_upgrades", c) }

// SetKickBots writes redbots_manager_kick_bots.
//
//sp:globalset redbots_manager_kick_bots
func SetKickBots(c ConVar) { conVarSets.set("redbots_manager_kick_bots", c) }

// SetMinPlayers writes redbots_manager_min_players.
//
//sp:globalset redbots_manager_min_players
func SetMinPlayers(c ConVar) { conVarSets.set("redbots_manager_min_players", c) }

// SetManagerMode writes redbots_manager_mode.
//
//sp:globalset redbots_manager_mode
func SetManagerMode(c ConVar) { conVarSets.set("redbots_manager_mode", c) }

// SetReadyCooldown writes redbots_manager_ready_cooldown.
//
//sp:globalset redbots_manager_ready_cooldown
func SetReadyCooldown(c ConVar) { conVarSets.set("redbots_manager_ready_cooldown", c) }

// SetSpawnNavRecovery writes redbots_manager_spawn_nav_recovery.
//
//sp:globalset redbots_manager_spawn_nav_recovery
func SetSpawnNavRecovery(c ConVar) { conVarSets.set("redbots_manager_spawn_nav_recovery", c) }

// SetSpawnNavRecoveryRadius writes redbots_manager_spawn_nav_recovery_radius.
//
//sp:globalset redbots_manager_spawn_nav_recovery_radius
func SetSpawnNavRecoveryRadius(c ConVar) {
	conVarSets.set("redbots_manager_spawn_nav_recovery_radius", c)
}

// SetSpawnNavRecoveryTime writes redbots_manager_spawn_nav_recovery_time.
//
//sp:globalset redbots_manager_spawn_nav_recovery_time
func SetSpawnNavRecoveryTime(c ConVar) { conVarSets.set("redbots_manager_spawn_nav_recovery_time", c) }

// SetTeamComposition writes redbots_manager_team_composition.
//
//sp:globalset redbots_manager_team_composition
func SetTeamComposition(c ConVar) { conVarSets.set("redbots_manager_team_composition", c) }

// SetUseCustomLoadouts writes redbots_manager_use_custom_loadouts.
//
//sp:globalset redbots_manager_use_custom_loadouts
func SetUseCustomLoadouts(c ConVar) { conVarSets.set("redbots_manager_use_custom_loadouts", c) }

// SetNestRelocateConVar writes redbots_manager_engineer_nest_relocate. The
// plain name is taken by the per-client nest slot in engineer.go.
//
//sp:globalset redbots_manager_engineer_nest_relocate
func SetNestRelocateConVar(c ConVar) {
	conVarSets.set("redbots_manager_engineer_nest_relocate", c)
}
