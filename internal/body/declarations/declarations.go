/*
Package declarations is what tf2_defenderbots.sp used to declare: the two mode
enums, the mod's globals and every convar handle it owns.

Nothing here is read from Go. It exists so that adding a convar or a per-client
field is a Go edit rather than a SourcePawn one, which is what "SourcePawn is a
build artifact" has to mean to be worth anything.

Three things stayed behind, because the generator has no form for them: the
three enum structs, which carry methods, the plugin's own myinfo, and the
include list, which is its build order.
*/
package declarations

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is MAXPLAYERS + 1, the client array size.
const Slots = 65

// ManagerMode is which of the three ways the mod runs.
const (
	//sp:name MANAGER_MODE_MANUAL_BOTS
	ManagerModeManualBots = 0
	//sp:name MANAGER_MODE_READY_BOTS
	ManagerModeReadyBots = 1
	//sp:name MANAGER_MODE_AUTO_BOTS
	ManagerModeAutoBots = 2
)

// BotLineupMode is how the bot team's classes are decided.
const (
	//sp:name BOT_LINEUP_MODE_RANDOM
	BotLineupModeRandom = 0
	//sp:name BOT_LINEUP_MODE_PREFERENCE
	BotLineupModePreference = 1
	//sp:name BOT_LINEUP_MODE_CHOOSE
	BotLineupModeChoose = 2
	//sp:name BOT_LINEUP_MODE_PREFERENCE_CHOOSE
	BotLineupModePreferenceChoose = 3
)

/*
The mod's own state, none of it per client.
*/

//sp:name g_bLateLoad
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var lateLoad bool

//sp:name g_bBotsEnabled
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var botsEnabled bool

//sp:name g_flAddingBotTime
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var addingBotTime float32

//sp:name g_flNextReadyTime
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var nextReadyTime float32

//sp:name g_iDetonatingPlayer
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var detonatingPlayer int32 = -1

//sp:name g_adtChosenBotClasses
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var chosenBotClasses engine.List

/*
chosenBotSeats is the seat of the team composition each of those classes was
named in, index for index.

Only the composition names seats, and the lineup mode's classes are appended
after its own, so this list is the shorter of the two and an index past its end
is a bot sitting in no seat.
*/
//
//sp:name g_adtChosenBotSeats
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var chosenBotSeats engine.List

//sp:name g_bBotClassesLocked
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var botClassesLocked bool

//sp:name g_iUIDBotSummoner
//nolint:unused,revive // emitted, not read from Go, and var-declaration: the shipped declaration writes the zero out and the emitter follows it
var uidBotSummoner int32 = 0

//sp:name g_bAllowBotTeamRedo
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var allowBotTeamRedo bool

/*
One entry per client, for the bots.
*/

//sp:name g_bIsDefenderBot
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var isDefenderBot [Slots]bool

//sp:name g_bIsBeingRevived
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var isBeingRevived [Slots]bool

//sp:name g_bHasUpgraded
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var hasUpgraded [Slots]bool

/*
shoppedThisBreak is whether this bot has done its shopping since the last wave
started.

Readiness used to stand in for this, and it stopped being able to: with a person
on RED every bot is held ready from the first frame of the break so the person
alone decides when the wave starts. Every "has he finished preparing" test that
read the ready flag then answered yes before he had bought anything, which
skipped the shopping trip and, from the second wave on, skipped it for the rest
of the mission.
*/
//
//sp:name g_bShoppedThisBreak
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var shoppedThisBreak [Slots]bool

//sp:name g_arrExtraButtons
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var extraButtons [Slots]engine.ButtonInput

//sp:name m_flDeadRethinkTime
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var deadRethinkTime [Slots]float32

//sp:name g_iBuybackNumber
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var buybackNumber [Slots]int32

//sp:name g_iBuyUpgradesNumber
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var buyUpgradesNumber [Slots]int32

//sp:name m_flNextSnipeFireTime
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var nextSnipeFireTime [Slots]float32

//sp:name m_flNextRollTime
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var nextRollTime [Slots]float32

/*
One entry per client, for the people.
*/

//sp:name g_bChoosingBotClasses
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var choosingBotClasses [Slots]bool

//sp:name g_flEnableBotsCooldown
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var enableBotsCooldown [Slots]float32

//sp:name m_flLastReadyInputTime
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var lastReadyInputTime [Slots]float32

/*
The map's config, and the world.
*/

//sp:name g_arrMapConfig
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var mapConfig engine.MapConfigRecord

//sp:name m_adtBotNames
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var botNames engine.List

//sp:name g_iPopulationManager
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var populationManager int32 = -1

//sp:name g_pMannVsMachineUpgrades
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var mannVsMachineUpgrades engine.Address

/*
Every convar this mod owns, and the seven of the game's own it looks up.
*/

//sp:name redbots_manager_debug
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerDebug engine.ConVar

//sp:name redbots_manager_debug_actions
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerDebugActions engine.ConVar

//sp:name redbots_manager_mode
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerMode engine.ConVar

//sp:name redbots_manager_bot_lineup_mode
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotLineupMode engine.ConVar

//sp:name redbots_manager_use_custom_loadouts
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerUseCustomLoadouts engine.ConVar

//sp:name redbots_manager_class_blacklist
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerClassBlacklist engine.ConVar

//sp:name redbots_manager_team_composition
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerTeamComposition engine.ConVar

//sp:name redbots_manager_kick_bots
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerKickBots engine.ConVar

//sp:name redbots_manager_min_players
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerMinPlayers engine.ConVar

//sp:name redbots_manager_defender_team_size
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerDefenderTeamSize engine.ConVar

//sp:name redbots_manager_ready_cooldown
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerReadyCooldown engine.ConVar

//sp:name redbots_manager_keep_bot_upgrades
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerKeepBotUpgrades engine.ConVar

//sp:name redbots_manager_bot_upgrade_interval
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotUpgradeInterval engine.ConVar

//sp:name redbots_manager_engineer_nest_depth
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerEngineerNestDepth engine.ConVar

//sp:name redbots_manager_engineer_nest_relocate
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerEngineerNestRelocate engine.ConVar

//sp:name redbots_manager_engineer_nest_relocate_score_gain_min
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerEngineerNestRelocateScoreGainMin engine.ConVar

//sp:name redbots_manager_bot_use_upgrades
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotUseUpgrades engine.ConVar

//sp:name redbots_manager_spawn_nav_recovery
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerSpawnNavRecovery engine.ConVar

//sp:name redbots_manager_spawn_nav_recovery_radius
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerSpawnNavRecoveryRadius engine.ConVar

//sp:name redbots_manager_spawn_nav_recovery_time
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerSpawnNavRecoveryTime engine.ConVar

//sp:name redbots_manager_bot_hats
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotHats engine.ConVar

//sp:name redbots_manager_bot_hat_effects
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotHatEffects engine.ConVar

//sp:name redbots_manager_bot_buyback_chance
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotBuybackChance engine.ConVar

//sp:name redbots_manager_bot_buy_upgrades_chance
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotBuyUpgradesChance engine.ConVar

//sp:name redbots_manager_bot_max_tank_attackers
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotMaxTankAttackers engine.ConVar

//sp:name redbots_manager_bot_aim_skill
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotAimSkill engine.ConVar

//sp:name redbots_manager_bot_reflect_skill
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotReflectSkill engine.ConVar

//sp:name redbots_manager_bot_reflect_chance
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotReflectChance engine.ConVar

//sp:name redbots_manager_bot_backstab_skill
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotBackstabSkill engine.ConVar

//sp:name redbots_manager_bot_hear_spy_range
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotHearSpyRange engine.ConVar

//sp:name redbots_manager_bot_notice_spy_time
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotNoticeSpyTime engine.ConVar

//sp:name redbots_manager_extra_bots
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerExtraBots engine.ConVar

//sp:name redbots_manager_bot_request_credits
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotRequestCredits engine.ConVar

//sp:name redbots_manager_bot_rtd_variance
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotRtdVariance engine.ConVar

//sp:name nb_blind
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var nbBlind engine.ConVar

//sp:name tf_bot_path_lookahead_range
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var tfBotPathLookaheadRange engine.ConVar

//sp:name tf_bot_health_critical_ratio
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var tfBotHealthCriticalRatio engine.ConVar

//sp:name tf_bot_health_ok_ratio
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var tfBotHealthOkRatio engine.ConVar

//sp:name tf_bot_ammo_search_range
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var tfBotAmmoSearchRange engine.ConVar

//sp:name tf_bot_health_search_far_range
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var tfBotHealthSearchFarRange engine.ConVar

//sp:name tf_bot_health_search_near_range
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var tfBotHealthSearchNearRange engine.ConVar

/*
botTeamCompositions is the three preset lineups AddBotsWithPresetTeamComp draws
from.

Both are dead: nothing calls that function and nothing else reads this table,
which is mvm-z83.80. They stayed because a port does not delete what it does not
understand.
*/
//
//sp:name g_sBotTeamCompositions
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var botTeamCompositions = [3][6]string{
	{"scout", "soldier", "demoman", "heavyweapons", "engineer", "medic"},
	{"scout", "heavyweapons", "heavyweapons", "heavyweapons", "engineer", "sniper"},
	{"scout", "heavyweapons", "heavyweapons", "pyro", "engineer", "demoman"},
}
