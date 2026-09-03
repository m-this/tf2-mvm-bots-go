/*
Package pluginstart is OnPluginStart out of source/tf2_defenderbots.sp.

Everything here is wiring. What it wires is decided elsewhere; this is the file
that says which generated function answers which command, event and convar, and
it is the one place that has to know every callback by name.
*/
package pluginstart

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// OnPluginStart is the whole of it.
//
//sp:name OnPluginStart
//sp:public
func OnPluginStart() {
	engine.ArchipelagoInit()

	engine.BuildPathInto(engine.PlayerPrefPath(), engine.PathMax(), "data/db_botpref.txt")

	engine.SetManagerDebug(engine.CreateConVar("sm_redbots_manager_debug", "0", "", engine.FcvarNone()))
	engine.SetDebugActions(engine.CreateConVar("sm_redbots_manager_debug_actions", "0", "", engine.FcvarNone()))
	engine.SetManagerMode(engine.CreateConVar("sm_redbots_manager_mode", "0", "What mode of the mod the use.", engine.FcvarNotify()))
	engine.SetBotLineupMode(engine.CreateConVar("sm_redbots_manager_bot_lineup_mode", "0", "How bot team composition is decided.", engine.FcvarNotify()))
	engine.SetUseCustomLoadouts(engine.CreateConVar("sm_redbots_manager_use_custom_loadouts", "0", "Let's bots use different weapons.", engine.FcvarNotify()))
	engine.SetClassBlacklist(engine.CreateConVar("sm_redbots_manager_class_blacklist", "", "Classes the bots never play, comma-separated. Example: sniper,spy", engine.FcvarNotify()))
	engine.SetTeamComposition(engine.CreateConVar("sm_redbots_manager_team_composition", "", "The classes the bots fill RED with, in order, comma-separated. Overrides the lineup mode and the blacklist. Example: engineer,medic,heavyweapons,soldier,demoman", engine.FcvarNotify()))
	engine.SetKickBots(engine.CreateConVar("sm_redbots_manager_kick_bots", "0", "Kick bots on wave failure/completion. A kicked bot is replaced by a new one that owns nothing, so this throws away every upgrade the team bought.", engine.FcvarNotify()))
	engine.SetMinPlayers(engine.CreateBoundedConVar("sm_redbots_manager_min_players", "3", "Minimum players for normal missions. Other difficulties are adjusted based on this value. Set to -1 to disable entirely.", engine.FcvarNotify(), true, -1.0, true, float32(engine.MaxTargets())))
	engine.SetDefenderTeamSize(engine.CreateBoundedConVar("sm_redbots_manager_defender_team_size", "6", "How many seats RED holds. Mann vs Machine has its own limit and this is clamped to it.", engine.FcvarNotify(), true, 1.0, true, float32(engine.MaxTargets())))
	engine.SetReadyCooldown(engine.CreateFlooredConVar("sm_redbots_manager_ready_cooldown", "30.0", "", engine.FcvarNotify(), true, 0.0))
	engine.SetKeepBotUpgrades(engine.CreateConVar("sm_redbots_manager_keep_bot_upgrades", "1", "Let bots that survive a failed wave keep what they bought, instead of refunding it and making them shop again from nothing.", engine.FcvarNotify()))
	engine.SetUpgradeInterval(engine.CreateConVar("sm_redbots_manager_bot_upgrade_interval", "0.1", "", engine.FcvarNotify()))
	engine.SetNestDepth(engine.CreateBoundedConVar("sm_redbots_manager_engineer_nest_depth", "0.4", "How far up the bomb path an engineer will build, as a fraction of the whole path measured from the hatch. 1.0 is the robots' spawn door.", engine.FcvarNotify(), true, 0.05, true, 1.0))
	engine.SetNestRelocateConVar(engine.CreateConVar("sm_redbots_manager_engineer_nest_relocate", "0", "Let engineers move their nest between waves when a better spot opens up. Crashes the server, see TODO item 10.", engine.FcvarNotify()))
	engine.SetNestRelocateScoreGainMin(engine.CreateBoundedConVar("sm_redbots_manager_engineer_nest_relocate_score_gain_min", "40.0", "How much better a nest spot has to score than the one an engineer holds before he moves to it between waves. 0 makes him move for any improvement at all.", engine.FcvarNotify(), true, 0.0, true, 200.0))
	engine.SetUseUpgrades(engine.CreateConVar("sm_redbots_manager_bot_use_upgrades", "1", "Enable bots to buy upgrades.", engine.FcvarNotify()))
	engine.SetSpawnNavRecovery(engine.CreateBoundedConVar("sm_redbots_spawn_nav_recovery", "1", "Recover prepared defender bots that cannot leave spawn.", engine.FcvarNotify(), true, 0.0, true, 1.0))
	engine.SetSpawnNavRecoveryRadius(engine.CreateBoundedConVar("sm_redbots_spawn_nav_recovery_radius", "512.0", "Distance outside a RED spawn brush that still counts as spawn for navigation recovery.", engine.FcvarNotify(), true, 0.0, true, 4096.0))
	engine.SetSpawnNavRecoveryTime(engine.CreateBoundedConVar("sm_redbots_spawn_nav_recovery_time", "12.0", "Maximum seconds a prepared defender may remain in or near spawn before recovery.", engine.FcvarNotify(), true, 1.0, true, 120.0))
	engine.SetBotHats(engine.CreateConVar("sm_redbots_manager_bot_hats", "1", "Give every defender bot a random hat its class can wear. Looks only.", engine.FcvarNotify()))
	engine.SetBotHatEffects(engine.CreateConVar("sm_redbots_manager_bot_hat_effects", "0", "Put a random unusual effect on that hat. Needs the hats above.", engine.FcvarNotify()))
	engine.SetBuybackChance(engine.CreateConVar("sm_redbots_manager_bot_buyback_chance", "5", "Chance for bots to buyback into the game.", engine.FcvarNotify()))
	engine.SetBuyUpgradesChance(engine.CreateConVar("sm_redbots_manager_bot_buy_upgrades_chance", "50", "Chance for bots to buy upgrades in the middle of a game.", engine.FcvarNotify()))
	engine.SetMaxTankAttackers(engine.CreateConVar("sm_redbots_manager_bot_max_tank_attackers", "3", "", engine.FcvarNotify()))
	engine.SetAimSkill(engine.CreateConVar("sm_redbots_manager_bot_aim_skill", "0", "", engine.FcvarNotify()))
	engine.SetReflectSkill(engine.CreateConVar("sm_redbots_manager_bot_reflect_skill", "1", "", engine.FcvarNotify()))
	engine.SetReflectChance(engine.CreateConVar("sm_redbots_manager_bot_reflect_chance", "100.0", "", engine.FcvarNotify()))
	engine.SetBackstabSkill(engine.CreateConVar("sm_redbots_manager_bot_backstab_skill", "0", "", engine.FcvarNotify()))
	engine.SetHearSpyRange(engine.CreateConVar("sm_redbots_manager_bot_hear_spy_range", "3000.0", "", engine.FcvarNotify()))
	engine.SetNoticeSpyTime(engine.CreateConVar("sm_redbots_manager_bot_notice_spy_time", "0.0", "", engine.FcvarNotify()))
	engine.SetExtraBots(engine.CreateConVar("sm_redbots_manager_extra_bots", "1", "How many more bots we are allowed to request beyond the team size", engine.FcvarNotify()))
	engine.SetRequestCredits(engine.CreateConVar("sm_redbots_manager_bot_request_credits", "1", "", engine.FcvarNotify()))
	engine.SetRtdVariance(engine.CreateConVar("sm_redbots_manager_bot_rtd_variance", "15.0", "", engine.FcvarNotify()))

	engine.HookConVarChange(engine.DefenderTeamSize(), engine.ConVarChangedDefenderTeamSize)
	engine.HookConVarChange(engine.ManagerMode(), engine.ConVarChangedManagerMode)
	engine.HookConVarChange(engine.BotLineupMode(), engine.ConVarChangedBotLineupMode)
	engine.HookConVarChange(engine.TeamComposition(), engine.ConVarChangedTeamComposition)

	engine.RegConsoleCmd("sm_votebots", engine.CommandVotebots)
	engine.RegConsoleCmd("sm_vb", engine.CommandVotebots)
	engine.RegConsoleCmd("sm_botpref", engine.CommandBotPreferences)
	engine.RegConsoleCmd("sm_botpreferences", engine.CommandBotPreferences)
	engine.RegConsoleCmd("sm_viewbotchances", engine.CommandShowBotChances)
	engine.RegConsoleCmd("sm_botchances", engine.CommandShowBotChances)
	engine.RegConsoleCmd("sm_viewbotlineup", engine.CommandShowNewBotTeamComposition)
	engine.RegConsoleCmd("sm_botlineup", engine.CommandShowNewBotTeamComposition)
	engine.RegConsoleCmd("sm_rerollbotclasses", engine.CommandRerollNewBotTeamComposition)
	engine.RegConsoleCmd("sm_rerollbots", engine.CommandRerollNewBotTeamComposition)
	engine.RegConsoleCmd("sm_rollbots", engine.CommandRerollNewBotTeamComposition)
	engine.RegConsoleCmd("sm_playwithbots", engine.CommandJoinBluePlayWithBots)
	engine.RegConsoleCmd("sm_requestbot", engine.CommandRequestExtraBot)
	engine.RegConsoleCmd("sm_choosebotteam", engine.CommandChooseBotClasses)
	engine.RegConsoleCmd("sm_cbt", engine.CommandChooseBotClasses)
	engine.RegConsoleCmd("sm_redobots", engine.CommandRedoBotTeamLineup)

	engine.RegAdminCmdPlain("sm_addbots", engine.CommandAddBots, engine.AdmFlagGeneric())
	engine.RegAdminCmdPlain("sm_purgebots", engine.CommandRemoveAllBots, engine.AdmFlagGeneric())
	engine.RegAdminCmd("sm_redbots_reseat", engine.CommandReseatBots, engine.AdmFlagGeneric(), "Reload the loadout file and rebuild RED from the current lineup")
	engine.RegAdminCmd("sm_dump_credits", engine.CommandDumpCredits, engine.AdmFlagGeneric(), "What every player on RED is holding")
	engine.RegAdminCmdPlain("sm_botmanager_stop", engine.CommandStopManagingBots, engine.AdmFlagGeneric())
	engine.RegAdminCmdPlain("sm_view_bot_upgrades", engine.CommandViewBotUpgrades, engine.AdmFlagGeneric())
	/* Not RegAdminCmd: it prints where the caller is standing and changes
	nothing, and needing an admin entry to write down a nest spot is a gate in
	front of the only way to author one */
	engine.RegConsoleCmdPlain("sm_dump_spot", engine.CommandDumpSpot)
	engine.RegAdminCmdPlain("sm_dump_upgrades", engine.CommandDumpUpgrades, engine.AdmFlagGeneric())
	engine.RegAdminCmdPlain("sm_dump_hats", engine.CommandDumpHats, engine.AdmFlagGeneric())
	engine.RegAdminCmdPlain("sm_dump_nest", engine.CommandDumpNest, engine.AdmFlagGeneric())
	engine.RegAdminCmdPlain("sm_dump_medic", engine.CommandDumpMedic, engine.AdmFlagGeneric())
	engine.RegAdminCmdPlain("sm_dump_front", engine.CommandDumpFront, engine.AdmFlagGeneric())
	engine.RegAdminCmdPlain("sm_dump_spawn_nav", engine.CommandDumpSpawnNav, engine.AdmFlagGeneric())
	engine.RegAdminCmdPlain("sm_recover_spawn_bots", engine.CommandRecoverSpawnBots, engine.AdmFlagGeneric())

	engine.AddCommandListener(engine.ListenerTournamentPlayerReadystate, "tournament_player_readystate")
	engine.AddCommandListener(engine.ListenerVoiceMenu, "voicemenu")

	engine.AddNormalSoundHook(engine.SoundHookGeneral)

	engine.InitGameEventHooks()

	hGamedata := engine.NewGameData("tf2.defenderbots")

	if hGamedata != engine.NoGameData() {
		engine.InitOffsets(hGamedata)

		bFailed := false

		engine.InitMvMUpgrades(hGamedata)

		engine.SetUpgradesAddress(engine.GameConfAddress(hGamedata, "MannVsMachineUpgrades"))

		if engine.UpgradesAddress() == engine.NoAddress() {
			engine.LogError("OnPluginStart: Failed to find Address to g_MannVsMachineUpgrades!")
		}

		if !engine.InitSDKCalls(hGamedata) {
			bFailed = true
		}

		if !engine.InitDHooks(hGamedata) {
			bFailed = true
		}

		hGamedata.Close()

		if bFailed {
			engine.SetFailState("Gamedata failed!")
		}
	} else {
		engine.SetFailState("Failed to load gamedata file tf2.defenderbots.txt")
	}

	if engine.LateLoad() {
		engine.SetPopulationManager(engine.FindEntityByClassname(engine.MaxClients()+1, "info_populator"))
	}

	engine.LoadFeatures()
	engine.BluAssistInit()
	engine.DebugFaultsInit()
	engine.LoadLoadoutFunctions()
	engine.LoadPreferencesData()

	engine.SetChosenBotClasses(engine.NewListSized(engine.ClassNameMax()))
	engine.SetChosenBotSeats(engine.NewList())
	engine.SetBotNames(engine.NewListSized(engine.NameMax()))
	engine.InitMapConfig()

	engine.InitNextBotPathing()

	engine.FindGameConsoleVariables()
}

/*
	AskPluginLoad2 says the plugin will load, and publishes what it offers

The six natives are the test bed's window into a bot. "He is not moving" and "he
has nowhere to walk" look identical from every angle a watcher has, and telling
them apart has been the hard part of five separate faults; the path length and
whether the bot believes it is walking are both held in here, so they are
published from here. Exported rather than logged because the test bed wants them
per sample beside everything else it records. Nothing in this plugin depends on
anybody calling them.
*/
//
//sp:name AskPluginLoad2
//sp:public
//sp:writable errorText
//nolint:revive // unused-parameter: the loader hands all four and this one reads the late flag
func AskPluginLoad2(myself engine.Timer, late bool, errorText engine.Text, errMax int32) engine.AplRes {
	engine.SetLateLoad(late)

	engine.CreateNative("Defenderbots_GetPathLength", engine.NativeGetPathLength)
	engine.CreateNative("Defenderbots_IsPathing", engine.NativeIsPathing)
	engine.CreateNative("Defenderbots_PathFailed", engine.NativePathFailed)
	engine.CreateNative("Defenderbots_PathFailures", engine.NativePathFailures)
	engine.CreateNative("Defenderbots_RangeRepairStalls", engine.NativeRangeRepairStalls)
	engine.CreateNative("Defenderbots_GetAttackTarget", engine.NativeGetAttackTarget)

	/* The one native this plugin asks for rather than offers, and it is
	allowed to be missing.

	Without this line an unresolved native does not fail at the call, it fails
	the whole plugin load: a server that has never heard of Archipelago would
	get no defender bots at all. The runtime check in archipelago.sp only ever
	runs on a plugin that loaded. */
	engine.MarkNativeAsOptional("TF2AP_GetBundleCredits")

	engine.RegPluginLibrary("tf2_defenderbots")

	return engine.AplResSuccess()
}
