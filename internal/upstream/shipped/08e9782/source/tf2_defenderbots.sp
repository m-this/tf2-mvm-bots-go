/* --------------------------------------------------
MvM Defender TFBots
April 08 2024
Author: ★ Officer Spy ★
-------------------------------------------------- */
#include <sourcemod>
#include <tf2_stocks>
#include <dhooks>
#include <tf2attributes>
#include <tf_econ_data>
#include <tf2utils>
#include <cbasenpc>
#include <cbasenpc/tf/nav>
#include <ripext>

#define _disable_actions_query_result_type
#define _disable_actions_event_result_priority_type
#include <actions>

#pragma semicolon 1
#pragma newdecls required

// #define TESTING_ONLY

#define MOD_REQUEST_CREDITS
#define MOD_CUSTOM_ATTRIBUTES
#define MOD_ROLL_THE_DICE_REVAMPED

#define METHOD_MVM_UPGRADES

#define CHANGETEAM_RESTRICTIONS

// #define TFBOT_CUSTOM_SPY_CONTACT

#define EXTRA_PLUGINBOT

// #define VALIDATE_ENTITY_TANKBOSS

// #define IDLEBOT_AIMING

#define PLUGIN_PREFIX	"[BotManager]"
#define TFBOT_IDENTITY_NAME	"TFBOT_SEX_HAVER"

enum
{
	MANAGER_MODE_MANUAL_BOTS = 0,
	MANAGER_MODE_READY_BOTS,
	MANAGER_MODE_AUTO_BOTS
}

enum
{
	BOT_LINEUP_MODE_RANDOM,
	BOT_LINEUP_MODE_PREFERENCE,
	BOT_LINEUP_MODE_CHOOSE,
	BOT_LINEUP_MODE_PREFERENCE_CHOOSE
}

//A zone name is a short label like "inside": long enough to read, short enough to keep in a config
#define NEST_ZONE_LENGTH	24

enum struct esMapConfiguration
{
	ArrayList adtSniperSpot;
	ArrayList adtEngineerNestLocation;
	//One zone name per nest spot, same order. Empty when the map does not name one
	ArrayList adtEngineerNestZone;
	ArrayList adtTeleporterEntranceLocation;
	ArrayList adtTeleporterExitLocation;
	ArrayList adtDispenserLocation;
	//One zone name per dispenser spot, same order, so a nest in a zone takes the dispenser in it
	ArrayList adtDispenserZone;
	//Nests that only apply to a wave with a tank in it, and nests that only apply to one without
	ArrayList adtNestTankOnlyLocation;
	ArrayList adtNestNoTankLocation;
	//The lineup this map wants, comma separated, empty when it does not care
	char strComposition[128];
	/* Whether the engineers are expected to pick the nest up and move it between waves

	Mannhattan's gates move the front, and Rottenburg wants a different nest for a tank wave than
	for one without. On a map like that a disposable sentry covers the ground while the real one is
	in a toolbox, and is worth buying. On every other map it is a hundred and fifty credits for a
	second sentry nobody moves, which is what the guides mean when they say never. */
	bool bMovingNests;
	
	void Initialize()
	{
		this.adtSniperSpot = new ArrayList(3);
		this.adtEngineerNestLocation = new ArrayList(3);
		this.adtEngineerNestZone = new ArrayList(ByteCountToCells(NEST_ZONE_LENGTH));
		this.adtTeleporterEntranceLocation = new ArrayList(3);
		this.adtTeleporterExitLocation = new ArrayList(3);
		this.adtDispenserLocation = new ArrayList(3);
		this.adtDispenserZone = new ArrayList(ByteCountToCells(NEST_ZONE_LENGTH));
		this.adtNestTankOnlyLocation = new ArrayList(3);
		this.adtNestNoTankLocation = new ArrayList(3);
	}
	void Reset()
	{
		this.adtSniperSpot.Clear();
		this.adtEngineerNestLocation.Clear();
		this.adtEngineerNestZone.Clear();
		this.adtTeleporterEntranceLocation.Clear();
		this.adtTeleporterExitLocation.Clear();
		this.adtDispenserLocation.Clear();
		this.adtDispenserZone.Clear();
		this.adtNestTankOnlyLocation.Clear();
		this.adtNestNoTankLocation.Clear();
		this.strComposition[0] = '\0';
		this.bMovingNests = false;
	}
}

enum struct esButtonInput
{
	int iPress;
	float flPressTime;
	int iRelease;
	float flReleaseTime;
	float flKeySpeed;
	
	void Reset()
	{
		this.iPress = 0;
		this.flPressTime = 0.0;
		this.iRelease = 0;
		this.flReleaseTime = 0.0;
		this.flKeySpeed = 0.0;
	}
	
	void PressButtons(int buttons, float duration = -1.0)
	{
		this.iPress = buttons;
		this.flPressTime = duration > 0.0 ? GetGameTime() + duration : 0.0;
	}
	
	void ReleaseButtons(int buttons, float duration = -1.0)
	{
		this.iRelease = buttons;
		this.flReleaseTime = duration > 0.0 ? GetGameTime() + duration : 0.0;
	}
}

//Globals
bool g_bLateLoad;
bool g_bBotsEnabled;
float g_flAddingBotTime;
float g_flNextReadyTime;
int g_iDetonatingPlayer = -1;
ArrayList g_adtChosenBotClasses;

/* The seat of the team composition each of those classes was named in, index for index
Only the composition names seats, and the lineup mode's classes are appended after its own, so this
list is the shorter of the two and an index past its end is a bot sitting in no seat */
ArrayList g_adtChosenBotSeats;
bool g_bBotClassesLocked;
int g_iUIDBotSummoner = 0;
bool g_bAllowBotTeamRedo;

//For defender bots
bool g_bIsDefenderBot[MAXPLAYERS + 1];
bool g_bIsBeingRevived[MAXPLAYERS + 1];
bool g_bHasUpgraded[MAXPLAYERS + 1];

/* Whether this bot has done its shopping since the last wave started

Readiness used to stand in for this, and it stopped being able to: with a person on RED every bot
is held ready from the first frame of the break so the person alone decides when the wave starts.
Every "has he finished preparing" test that read the ready flag then answered yes before he had
bought anything, which skipped the shopping trip and, from the second wave on, skipped it for the
rest of the mission. */
bool g_bShoppedThisBreak[MAXPLAYERS + 1];
esButtonInput g_arrExtraButtons[MAXPLAYERS + 1];
static float m_flDeadRethinkTime[MAXPLAYERS + 1];
int g_iBuybackNumber[MAXPLAYERS + 1];
int g_iBuyUpgradesNumber[MAXPLAYERS + 1];

#if !defined IDLEBOT_AIMING
static float m_flNextSnipeFireTime[MAXPLAYERS + 1];
#endif

#if defined MOD_ROLL_THE_DICE_REVAMPED
static float m_flNextRollTime[MAXPLAYERS + 1];
#endif

//For other players
bool g_bChoosingBotClasses[MAXPLAYERS + 1];

#if defined CHANGETEAM_RESTRICTIONS
float g_flEnableBotsCooldown[MAXPLAYERS + 1];
#endif

static float m_flLastCommandTime[MAXPLAYERS + 1];
static float m_flLastReadyInputTime[MAXPLAYERS + 1];

//Config
esMapConfiguration g_arrMapConfig;
static ArrayList m_adtBotNames;

//Global entities
int g_iPopulationManager = -1;

ConVar redbots_manager_debug;
ConVar redbots_manager_debug_actions;
ConVar redbots_manager_mode;
ConVar redbots_manager_bot_lineup_mode;
ConVar redbots_manager_use_custom_loadouts;
ConVar redbots_manager_class_blacklist;
ConVar redbots_manager_team_composition;
ConVar redbots_manager_kick_bots;
ConVar redbots_manager_min_players;
ConVar redbots_manager_defender_team_size;
ConVar redbots_manager_ready_cooldown;
ConVar redbots_manager_keep_bot_upgrades;
ConVar redbots_manager_bot_upgrade_interval;
ConVar redbots_manager_engineer_nest_depth;
ConVar redbots_manager_engineer_nest_relocate;
ConVar redbots_manager_engineer_nest_relocate_score_gain_min;
ConVar redbots_manager_bot_use_upgrades;
ConVar redbots_manager_spawn_nav_recovery;
ConVar redbots_manager_spawn_nav_recovery_radius;
ConVar redbots_manager_spawn_nav_recovery_time;
ConVar redbots_manager_bot_hats;
ConVar redbots_manager_bot_hat_effects;
ConVar redbots_manager_bot_buyback_chance;
ConVar redbots_manager_bot_buy_upgrades_chance;
ConVar redbots_manager_bot_max_tank_attackers;
ConVar redbots_manager_bot_aim_skill;
ConVar redbots_manager_bot_reflect_skill;
ConVar redbots_manager_bot_reflect_chance;
ConVar redbots_manager_bot_backstab_skill;
ConVar redbots_manager_bot_hear_spy_range;
ConVar redbots_manager_bot_notice_spy_time;
ConVar redbots_manager_extra_bots;

#if defined MOD_REQUEST_CREDITS
ConVar redbots_manager_bot_request_credits;
#endif

#if defined MOD_ROLL_THE_DICE_REVAMPED
ConVar redbots_manager_bot_rtd_variance;
#endif

ConVar nb_blind;
ConVar tf_bot_path_lookahead_range;
ConVar tf_bot_health_critical_ratio;
ConVar tf_bot_health_ok_ratio;
ConVar tf_bot_ammo_search_range;
ConVar tf_bot_health_search_far_range;
ConVar tf_bot_health_search_near_range;

#if defined METHOD_MVM_UPGRADES
Address g_pMannVsMachineUpgrades;
#endif

#include "redbots3/archipelago.sp"
#include "redbots3/generated/features.sp"
#include "redbots3/generated/scan.sp"
#include "redbots3/blu_assist.sp"
#include "redbots3/util.sp"
#include "redbots3/generated/weapon_tuning.sp"
#include "redbots3/generated/uber.sp"
#include "redbots3/demoman_stickies.sp"
#include "redbots3/offsets.sp"
#include "redbots3/sdkcalls.sp"
#include "redbots3/loadouts.sp"
#include "redbots3/cosmetics.sp"
#include "redbots3/dhooks.sp"
#include "redbots3/events.sp"
#include "redbots3/player_pref.sp"
#include "redbots3/menu.sp"
#include "redbots3/tf_upgrades.sp"
#include "redbots3/debug_faults.sp"
#include "redbots3/generated/threat_priority.sp"
#include "redbots3/nextbot_behavior.sp"
#include "redbots3/botaim.sp"

public Plugin myinfo =
{
	name = "Defender TFBots",
	author = "Officer Spy",
	description = "TFBots that play Mann vs. Machine",
	/* This fork's version, not upstream's. The tags here restarted at v2.0.0 because the fork is
	far enough from 1.5.5 that the old number said nothing about what is running. Leaving myinfo on
	1.5.5 meant `sm plugins list` and every play-test report named a build nobody could identify. */
	version = "2.44.0",
	url = "https://github.com/OfficerSpy/TF2-MvM-Defender-TFBots"
};

/* The Archipelago plugin can come and go while this one runs, so its native is looked for again
whenever any plugin registers or drops a library rather than once at startup. */
public void OnLibraryAdded(const char[] name)
{
	if (StrEqual(name, "tf2_archipelago"))
		Archipelago_Recheck();
}

public void OnLibraryRemoved(const char[] name)
{
	if (StrEqual(name, "tf2_archipelago"))
		Archipelago_Recheck();
}

public void OnAllPluginsLoaded()
{
	Archipelago_Recheck();
}

public void OnPluginStart()
{
	Archipelago_Init();
	
#if defined TESTING_ONLY
	BuildPath(Path_SM, g_sPlayerPrefPath, PLATFORM_MAX_PATH, "data/testing/db_botpref.txt");
	PrintToServer("[BOTS MANAGER] DEBUG BUILD: FOR DEV USE ONLY");
#else
	BuildPath(Path_SM, g_sPlayerPrefPath, PLATFORM_MAX_PATH, "data/db_botpref.txt");
#endif
	
	redbots_manager_debug = CreateConVar("sm_redbots_manager_debug", "0", _, FCVAR_NONE);
	redbots_manager_debug_actions = CreateConVar("sm_redbots_manager_debug_actions", "0", _, FCVAR_NONE);
	redbots_manager_mode = CreateConVar("sm_redbots_manager_mode", "0", "What mode of the mod the use.", FCVAR_NOTIFY);
	redbots_manager_bot_lineup_mode = CreateConVar("sm_redbots_manager_bot_lineup_mode", "0", "How bot team composition is decided.", FCVAR_NOTIFY);
	redbots_manager_use_custom_loadouts = CreateConVar("sm_redbots_manager_use_custom_loadouts", "0", "Let's bots use different weapons.", FCVAR_NOTIFY);
	redbots_manager_class_blacklist = CreateConVar("sm_redbots_manager_class_blacklist", "", "Classes the bots never play, comma-separated. Example: sniper,spy", FCVAR_NOTIFY);
	redbots_manager_team_composition = CreateConVar("sm_redbots_manager_team_composition", "", "The classes the bots fill RED with, in order, comma-separated. Overrides the lineup mode and the blacklist. Example: engineer,medic,heavyweapons,soldier,demoman", FCVAR_NOTIFY);
	redbots_manager_kick_bots = CreateConVar("sm_redbots_manager_kick_bots", "0", "Kick bots on wave failure/completion. A kicked bot is replaced by a new one that owns nothing, so this throws away every upgrade the team bought.", FCVAR_NOTIFY);
	redbots_manager_min_players = CreateConVar("sm_redbots_manager_min_players", "3", "Minimum players for normal missions. Other difficulties are adjusted based on this value. Set to -1 to disable entirely.", FCVAR_NOTIFY, true, -1.0, true, float(MAXPLAYERS));
	redbots_manager_defender_team_size = CreateConVar("sm_redbots_manager_defender_team_size", "6", "How many seats RED holds. Mann vs Machine has its own limit and this is clamped to it.", FCVAR_NOTIFY, true, 1.0, true, float(MAXPLAYERS));
	redbots_manager_ready_cooldown = CreateConVar("sm_redbots_manager_ready_cooldown", "30.0", _, FCVAR_NOTIFY, true, 0.0);
	redbots_manager_keep_bot_upgrades = CreateConVar("sm_redbots_manager_keep_bot_upgrades", "1", "Let bots that survive a failed wave keep what they bought, instead of refunding it and making them shop again from nothing.", FCVAR_NOTIFY);
	redbots_manager_bot_upgrade_interval = CreateConVar("sm_redbots_manager_bot_upgrade_interval", "0.1", _, FCVAR_NOTIFY);
	redbots_manager_engineer_nest_depth = CreateConVar("sm_redbots_manager_engineer_nest_depth", "0.4", "How far up the bomb path an engineer will build, as a fraction of the whole path measured from the hatch. 1.0 is the robots' spawn door.", FCVAR_NOTIFY, true, 0.05, true, 1.0);
	//Off until the watchdog crash in TODO item 10 is found. Turning it on trips the server's
	//watchdog at the first wave transition, reliably, on mvm_decoy with two engineers
	redbots_manager_engineer_nest_relocate = CreateConVar("sm_redbots_manager_engineer_nest_relocate", "0", "Let engineers move their nest between waves when a better spot opens up. Crashes the server, see TODO item 10.", FCVAR_NOTIFY);
	redbots_manager_engineer_nest_relocate_score_gain_min = CreateConVar("sm_redbots_manager_engineer_nest_relocate_score_gain_min", "40.0", "How much better a nest spot has to score than the one an engineer holds before he moves to it between waves. 0 makes him move for any improvement at all.", FCVAR_NOTIFY, true, 0.0, true, 200.0);
	redbots_manager_bot_use_upgrades = CreateConVar("sm_redbots_manager_bot_use_upgrades", "1", "Enable bots to buy upgrades.", FCVAR_NOTIFY);
	redbots_manager_spawn_nav_recovery = CreateConVar("sm_redbots_spawn_nav_recovery", "1", "Recover prepared defender bots that cannot leave spawn.", FCVAR_NOTIFY, true, 0.0, true, 1.0);
	redbots_manager_spawn_nav_recovery_radius = CreateConVar("sm_redbots_spawn_nav_recovery_radius", "512.0", "Distance outside a RED spawn brush that still counts as spawn for navigation recovery.", FCVAR_NOTIFY, true, 0.0, true, 4096.0);
	redbots_manager_spawn_nav_recovery_time = CreateConVar("sm_redbots_spawn_nav_recovery_time", "12.0", "Maximum seconds a prepared defender may remain in or near spawn before recovery.", FCVAR_NOTIFY, true, 1.0, true, 120.0);
	redbots_manager_bot_hats = CreateConVar("sm_redbots_manager_bot_hats", "1", "Give every defender bot a random hat its class can wear. Looks only.", FCVAR_NOTIFY);
	redbots_manager_bot_hat_effects = CreateConVar("sm_redbots_manager_bot_hat_effects", "0", "Put a random unusual effect on that hat. Needs the hats above.", FCVAR_NOTIFY);
	redbots_manager_bot_buyback_chance = CreateConVar("sm_redbots_manager_bot_buyback_chance", "5", "Chance for bots to buyback into the game.", FCVAR_NOTIFY);
	redbots_manager_bot_buy_upgrades_chance = CreateConVar("sm_redbots_manager_bot_buy_upgrades_chance", "50", "Chance for bots to buy upgrades in the middle of a game.", FCVAR_NOTIFY);
	redbots_manager_bot_max_tank_attackers = CreateConVar("sm_redbots_manager_bot_max_tank_attackers", "3", _, FCVAR_NOTIFY);
	redbots_manager_bot_aim_skill = CreateConVar("sm_redbots_manager_bot_aim_skill", "0", _, FCVAR_NOTIFY);
	redbots_manager_bot_reflect_skill = CreateConVar("sm_redbots_manager_bot_reflect_skill", "1", _, FCVAR_NOTIFY);
	redbots_manager_bot_reflect_chance = CreateConVar("sm_redbots_manager_bot_reflect_chance", "100.0", _, FCVAR_NOTIFY);
	redbots_manager_bot_backstab_skill = CreateConVar("sm_redbots_manager_bot_backstab_skill", "0", _, FCVAR_NOTIFY);
	redbots_manager_bot_hear_spy_range = CreateConVar("sm_redbots_manager_bot_hear_spy_range", "3000.0", _, FCVAR_NOTIFY);
	redbots_manager_bot_notice_spy_time = CreateConVar("sm_redbots_manager_bot_notice_spy_time", "0.0", _, FCVAR_NOTIFY);
	redbots_manager_extra_bots = CreateConVar("sm_redbots_manager_extra_bots", "1", "How many more bots we are allowed to request beyond the team size", FCVAR_NOTIFY);
	
#if defined MOD_REQUEST_CREDITS
	redbots_manager_bot_request_credits = CreateConVar("sm_redbots_manager_bot_request_credits", "1", _, FCVAR_NOTIFY);
#endif
	
#if defined MOD_ROLL_THE_DICE_REVAMPED
	redbots_manager_bot_rtd_variance = CreateConVar("sm_redbots_manager_bot_rtd_variance", "15.0", _, FCVAR_NOTIFY);
#endif
	
	HookConVarChange(redbots_manager_defender_team_size, ConVarChanged_DefenderTeamSize);
	HookConVarChange(redbots_manager_mode, ConVarChanged_ManagerMode);
	HookConVarChange(redbots_manager_bot_lineup_mode, ConVarChanged_BotLineupMode);
	HookConVarChange(redbots_manager_team_composition, ConVarChanged_TeamComposition);
	
	RegConsoleCmd("sm_votebots", Command_Votebots);
	RegConsoleCmd("sm_vb", Command_Votebots);
	RegConsoleCmd("sm_botpref", Command_BotPreferences);
	RegConsoleCmd("sm_botpreferences", Command_BotPreferences);
	RegConsoleCmd("sm_viewbotchances", Command_ShowBotChances);
	RegConsoleCmd("sm_botchances", Command_ShowBotChances);
	RegConsoleCmd("sm_viewbotlineup", Command_ShowNewBotTeamComposition);
	RegConsoleCmd("sm_botlineup", Command_ShowNewBotTeamComposition);
	RegConsoleCmd("sm_rerollbotclasses", Command_RerollNewBotTeamComposition);
	RegConsoleCmd("sm_rerollbots", Command_RerollNewBotTeamComposition);
	RegConsoleCmd("sm_rollbots", Command_RerollNewBotTeamComposition);
	RegConsoleCmd("sm_playwithbots", Command_JoinBluePlayWithBots);
	RegConsoleCmd("sm_requestbot", Command_RequestExtraBot);
	RegConsoleCmd("sm_choosebotteam", Command_ChooseBotClasses);
	RegConsoleCmd("sm_cbt", Command_ChooseBotClasses);
	RegConsoleCmd("sm_redobots", Command_RedoBotTeamLineup);
	
#if defined TESTING_ONLY
	RegConsoleCmd("sm_bots_start_now", Command_BotsReadyNow);
#endif
	
	RegAdminCmd("sm_addbots", Command_AddBots, ADMFLAG_GENERIC);
	RegAdminCmd("sm_purgebots", Command_RemoveAllBots, ADMFLAG_GENERIC);
	RegAdminCmd("sm_redbots_reseat", Command_ReseatBots, ADMFLAG_GENERIC, "Reload the loadout file and rebuild RED from the current lineup");
	RegAdminCmd("sm_dump_credits", Command_DumpCredits, ADMFLAG_GENERIC, "What every player on RED is holding");
	RegAdminCmd("sm_botmanager_stop", Command_StopManagingBots, ADMFLAG_GENERIC);
	RegAdminCmd("sm_view_bot_upgrades", Command_ViewBotUpgrades, ADMFLAG_GENERIC);
	//Not RegAdminCmd: it prints where the caller is standing and changes nothing, and needing
	//an admin entry to write down a nest spot is a gate in front of the only way to author one
	RegConsoleCmd("sm_dump_spot", Command_DumpSpot);
	RegAdminCmd("sm_dump_upgrades", Command_DumpUpgrades, ADMFLAG_GENERIC);
	RegAdminCmd("sm_dump_hats", Command_DumpHats, ADMFLAG_GENERIC);
	RegAdminCmd("sm_dump_nest", Command_DumpNest, ADMFLAG_GENERIC);
	RegAdminCmd("sm_dump_medic", Command_DumpMedic, ADMFLAG_GENERIC);
	RegAdminCmd("sm_dump_front", Command_DumpFront, ADMFLAG_GENERIC);
	RegAdminCmd("sm_dump_spawn_nav", Command_DumpSpawnNav, ADMFLAG_GENERIC);
	RegAdminCmd("sm_recover_spawn_bots", Command_RecoverSpawnBots, ADMFLAG_GENERIC);
	
	AddCommandListener(Listener_TournamentPlayerReadystate, "tournament_player_readystate");
	AddCommandListener(Listener_VoiceMenu, "voicemenu");
	
	AddNormalSoundHook(SoundHook_General);
	
	InitGameEventHooks();
	
	GameData hGamedata = new GameData("tf2.defenderbots");
	
	if (hGamedata)
	{
		InitOffsets(hGamedata);
		
		bool bFailed = false;
		
#if defined METHOD_MVM_UPGRADES
		InitMvMUpgrades(hGamedata);
		
		g_pMannVsMachineUpgrades = GameConfGetAddress(hGamedata, "MannVsMachineUpgrades");
		
		if (!g_pMannVsMachineUpgrades)
			LogError("OnPluginStart: Failed to find Address to g_MannVsMachineUpgrades!");
#if defined TESTING_ONLY
		else
			LogMessage("OnPluginStart: Found \"g_MannVsMachineUpgrades\" @ 0x%X", g_pMannVsMachineUpgrades);
#endif
#endif
		
		if (!InitSDKCalls(hGamedata))
			bFailed = true;
		
		if (!InitDHooks(hGamedata))
			bFailed = true;
		
		delete hGamedata;
		
		if (bFailed)
			SetFailState("Gamedata failed!");
	}
	else
	{
		SetFailState("Failed to load gamedata file tf2.defenderbots.txt");
	}
	
	if (g_bLateLoad)
	{
		g_iPopulationManager = FindEntityByClassname(MaxClients + 1, "info_populator");
	}
	
	LoadFeatures();
	BluAssist_Init();
	DebugFaults_Init();
	LoadLoadoutFunctions();
	LoadPreferencesData();
	
	g_adtChosenBotClasses = new ArrayList(TF2_CLASS_MAX_NAME_LENGTH);
	g_adtChosenBotSeats = new ArrayList();
	m_adtBotNames = new ArrayList(MAX_NAME_LENGTH);
	g_arrMapConfig.Initialize();
	
	InitNextBotPathing();
	
#if defined IDLEBOT_AIMING
	InitTFBotAim();
#endif
	
	FindGameConsoleVariables();
}

public void OnPluginEnd()
{
	RemoveAllDefenderBots("BM3 OnPluginEnd");
}

public APLRes AskPluginLoad2(Handle myself, bool late, char[] error, int err_max)
{
	g_bLateLoad = late;
	
	/* Two facts about a bot that nothing outside this plugin can work out for itself
	
	"He is not moving" and "he has nowhere to walk" look identical from every angle a watcher has,
	and telling them apart has been the hard part of five separate faults. The path length and
	whether the bot believes it is walking are both held in here, so they are published from here.
	
	Exported rather than logged because the test bed wants them per sample beside everything else it
	records. Nothing in this plugin depends on anybody calling them. */
	CreateNative("Defenderbots_GetPathLength", Native_GetPathLength);
	CreateNative("Defenderbots_IsPathing", Native_IsPathing);
	CreateNative("Defenderbots_PathFailed", Native_PathFailed);
	CreateNative("Defenderbots_PathFailures", Native_PathFailures);
	CreateNative("Defenderbots_RangeRepairStalls", Native_RangeRepairStalls);
	CreateNative("Defenderbots_GetAttackTarget", Native_GetAttackTarget);
	
	/* The one native this plugin asks for rather than offers, and it is allowed to be missing
	
	Without this line an unresolved native does not fail at the call, it fails the whole plugin
	load: a server that has never heard of Archipelago would get no defender bots at all. The
	runtime check in archipelago.sp only ever runs on a plugin that loaded. */
	MarkNativeAsOptional("TF2AP_GetBundleCredits");
	
	RegPluginLibrary("tf2_defenderbots");
	
	return APLRes_Success;
}

static any Native_GetPathLength(Handle plugin, int numParams)
{
	int client = GetNativeCell(1);
	
	if (client < 1 || client > MaxClients || !IsClientInGame(client) || !g_bIsDefenderBot[client])
		return -1.0;
	
	//Unguarded, as everywhere else that reads it: the path is made with the bot and outlives it
	return m_pPath[client].GetLength();
}

/* Whether the last computation came back with nothing, which the length cannot tell anybody
 *
 * A refused computation leaves the path object holding whatever it held before, so GetLength keeps
 * returning the old answer and a failing bot reads as a bot with a perfectly good path. Measured on
 * Decoy: the medic reported a path 10400 units long, constant to within fifty units over eighty
 * seconds, while the nearest teammate stood four hundred units away. Every one of those samples was
 * a failure wearing the length of the last success. */
static any Native_PathFailed(Handle plugin, int numParams)
{
	int client = GetNativeCell(1);
	
	if (client < 1 || client > MaxClients || !IsClientInGame(client) || !g_bIsDefenderBot[client])
		return false;
	
	return PathFailedFor(client);
}

static any Native_PathFailures(Handle plugin, int numParams)
{
	int client = GetNativeCell(1);
	
	if (client < 1 || client > MaxClients || !IsClientInGame(client) || !g_bIsDefenderBot[client])
		return -1;
	
	return PathFailuresOf(client);
}

/* Bolts fired at a sentry that gained nothing for three seconds, counted rather than sampled
 *
 * The state this happens in is rare enough that a five second sampler saw it zero times in a
 * hundred and thirty seven engineer samples. A counter does not care how rare it is. */
static any Native_RangeRepairStalls(Handle plugin, int numParams)
{
	int client = GetNativeCell(1);
	
	if (client < 1 || client > MaxClients || !IsClientInGame(client) || !g_bIsDefenderBot[client])
		return -1;
	
	return RangeRepairStallsOf(client);
}

/* Who this bot decided to shoot, which is the decision rather than where the crosshair happens to
be pointing. A wave that wipes the team is usually one robot nobody chose. */
static any Native_GetAttackTarget(Handle plugin, int numParams)
{
	int client = GetNativeCell(1);
	
	if (client < 1 || client > MaxClients || !IsClientInGame(client) || !g_bIsDefenderBot[client])
		return -1;
	
	return m_iAttackTarget[client];
}

static any Native_IsPathing(Handle plugin, int numParams)
{
	int client = GetNativeCell(1);
	
	if (client < 1 || client > MaxClients || !IsClientInGame(client) || !g_bIsDefenderBot[client])
		return false;
	
	return g_arrPluginBot[client].bPathing;
}

/* The features that are on, published once the server's own configs have run

They were published only when a wave began, and the statistics plugin reads the list in its own
handler for that same event. Whichever of the two hooks first is whichever SourceMod loaded first,
so the first wave of every run recorded an empty list: a results file that does not say which mod
produced it, which is the one thing the list exists for.

OnConfigsExecuted is after server.cfg and before anybody plays, which is where the answer stops
being the defaults and starts being what the server was asked for. */
public void OnConfigsExecuted()
{
	PublishActiveFeatures();
}

public void OnGameFrame()
{
	DebugFaults_OnGameFrame();
}

public void OnMapStart()
{
	g_bBotsEnabled = false;
	g_flAddingBotTime = 0.0;
	g_flNextReadyTime = 0.0;
	g_bBotClassesLocked = false;
	g_bAllowBotTeamRedo = false;
	Reseat_OnMapStart();
	
	ResetMapHintNests();
	
	Config_LoadMap();
	Config_LoadBotNames();
	Config_LoadServerLoadout();
	

	CreateBotPreferenceMenu();
}

public void OnClientDisconnect(int client)
{
	if (client == g_iPlayerForcedPref)
		g_iPlayerForcedPref = -1;
	
	if (!IsFakeClient(client))
		CreateTimer(0.1, Timer_RefillDefenderTeam, _, TIMER_FLAG_NO_MAPCHANGE);
	
	g_bIsDefenderBot[client] = false;
	ResetSpawnExitWatch(client);
	
	g_bChoosingBotClasses[client] = false;
	
	ResetLoadouts(client);
	ForgetBotSeat(client);
	ForgetBotCosmetics(client);
}

public void OnClientPutInServer(int client)
{
	if (!IsFakeClient(client))
		MakeRoomForHumanPlayer(client);

	
	//The name is set at tf_bot_add, so one of ours is known here, before it picks its loadout
	if (IsDefenderBot(client))
		TakeBotSeat(client);

	g_bHasUpgraded[client] = false;
	g_bShoppedThisBreak[client] = false;
	//A slot is reused, and a call left on the clock by whoever had it is not this player's call
	ForgetMedicCall(client);
	g_arrExtraButtons[client].Reset();
	m_flDeadRethinkTime[client] = 0.0;
	g_iBuybackNumber[client] = 0;
	g_iBuyUpgradesNumber[client] = 0;
	
#if defined MOD_ROLL_THE_DICE_REVAMPED
	m_flNextRollTime[client] = 0.0;
#endif
	
#if defined CHANGETEAM_RESTRICTIONS
	g_flEnableBotsCooldown[client] = 0.0;
#endif
	
	m_flLastCommandTime[client] = GetGameTime();
	m_flLastReadyInputTime[client] = 0.0;
	
	g_bHasBoughtUpgrades[client] = false;
	
	ResetNextBot(client);
	ResetSpawnExitWatch(client);
	
#if defined IDLEBOT_AIMING
	BotAim(client).Reset();
#endif
}

public void OnEntityCreated(int entity, const char[] classname)
{
	if (StrEqual(classname, "info_populator"))
		g_iPopulationManager = entity;
	
	DHooks_OnEntityCreated(entity, classname);
}

/* NOTE: This forward is not consistent with nextbot functionalities such as Action::Update
Nextbot behavior updates are based on the value of convar nb_update_frequency
This forward is only called every time CBasePlayer::PlayerRunCommand is called, which updates on its own interval
So what gets done in here will never always be consistent with the nextbot behavior actions */
public Action OnPlayerRunCmd(int client, int &buttons, int &impulse, float vel[3], float angles[3], int &weapon, int &subtype, int &cmdnum, int &tickcount, int &seed, int mouse[2])
{
	if (g_bIsDefenderBot[client] == false)
		return Plugin_Continue;
	
	if (IsPlayerAlive(client))
	{
		WatchDefenderSpawnExit(client);

		if (g_arrExtraButtons[client].iPress != 0)
		{
			if (g_arrExtraButtons[client].iPress & IN_BACK)
				vel[0] -= PLAYER_SIDESPEED;
			
			if (g_arrExtraButtons[client].iPress & IN_FORWARD)
				vel[0] += PLAYER_SIDESPEED;
			
			if (g_arrExtraButtons[client].iPress & IN_MOVELEFT)
				vel[1] -= PLAYER_SIDESPEED;
			
			if (g_arrExtraButtons[client].iPress & IN_MOVERIGHT)
				vel[1] += PLAYER_SIDESPEED;
			
			if (g_arrExtraButtons[client].iPress & IN_LEFT)
				angles[1] -= g_arrExtraButtons[client].flKeySpeed;
			
			if (g_arrExtraButtons[client].iPress & IN_RIGHT)
				angles[1] += g_arrExtraButtons[client].flKeySpeed;
			
			buttons |= g_arrExtraButtons[client].iPress;
			
			//We are told to hold these inputs down for a specific time, don't clear until it expires
			if (g_arrExtraButtons[client].flPressTime <= GetGameTime())
				g_arrExtraButtons[client].iPress = 0;
		}
		
		if (g_arrExtraButtons[client].iRelease != 0)
		{
			buttons &= ~g_arrExtraButtons[client].iRelease;
			
			if (g_arrExtraButtons[client].flReleaseTime <= GetGameTime())
				g_arrExtraButtons[client].iRelease = 0;
		}
		
#if defined EXTRA_PLUGINBOT
		PluginBot_SimulateFrame(client);
#endif
		
#if defined IDLEBOT_AIMING
		if (m_ctReload[client] > GetGameTime())
		{
			buttons |= IN_RELOAD;
		}
		
		if (m_ctFire[client] > GetGameTime())
		{
			buttons |= IN_ATTACK;
		}
		
		if (m_ctAltFire[client] > GetGameTime())
		{
			buttons |= IN_ATTACK2;
		}
#endif
		
		if (GameRules_GetRoundState() != RoundState_BetweenRounds)
		{
			int myWeapon = BaseCombatCharacter_GetActiveWeapon(client);
			int weaponID = myWeapon != -1 ? TF2Util_GetWeaponID(myWeapon) : -1;
			
			if (buttons & IN_ATTACK)
			{
				switch (weaponID)
				{
#if !defined IDLEBOT_AIMING
					case TF_WEAPON_MINIGUN:
					{
						//Don't keep spinning the minigun if it ran out of ammo
						if (!HasAmmo(myWeapon))
							buttons &= ~IN_ATTACK;
					}
#endif
					case TF_WEAPON_SNIPERRIFLE_CLASSIC:
					{
						//For the classic, let go on a full charge
						if (GetEntPropFloat(myWeapon, Prop_Send, "m_flChargedDamage") >= 150.0)
							buttons &= ~IN_ATTACK;
					}
					case TF_WEAPON_BUFF_ITEM:
					{
						//Once we blow the horn, stop pressing the fire button
						if (IsPlayingHorn(myWeapon))
							buttons &= ~IN_ATTACK;
					}
					case TF_WEAPON_REVOLVER:
					{
						if (CanRevolverHeadshot(myWeapon))
						{
							//Don;t fire if our shot won't be very accurate
							if (!(GetGameTime() - GetLastAccuracyCheck(myWeapon) > 1.0))
								buttons &= ~IN_ATTACK;
						}
					}
				}
			}
			
			INextBot myBot = CBaseNPC_GetNextBotOfEntity(client);
			IVision myVision = myBot.GetVisionInterface();
			
			MonitorKnownEntities(client, myVision);
			
			CKnownEntity threat = myVision.GetPrimaryKnownThreat(false);
			
			OpportunisticallyUseWeaponAbilities(client, myWeapon, myBot, threat);
			OpportunisticallyUsePowerupBottle(client, myWeapon, myBot, threat);
			
			if ((weaponID == TF_WEAPON_FLAMETHROWER || weaponID == TF_WEAPON_FLAME_BALL) && CanWeaponAirblast(myWeapon))
				UtilizeCompressionBlast(client, myBot, threat, 1);
			
#if defined IDLEBOT_AIMING
			if (threat)
			{
				//TODO: disable on engineers for now until we make a proper better behavior
				if (TF2_GetPlayerClass(client) != TFClass_Engineer)
					BotAim(client).AimHeadTowardsEntity(threat.GetEntity(), CRITICAL, 0.1);
			}
#else
			if (WeaponID_IsSniperRifle(weaponID))
			{
				if (TF2_IsPlayerInCondition(client, TFCond_Zoomed))
				{
					if (redbots_manager_bot_aim_skill.IntValue >= 1)
					{
						//TODO: this needs to be more precise with actually getting our current m_lookAtSubject in PlayerBody as this can cause jittery aim
						if (threat && IsLineOfFireClearEntity(client, GetEyePosition(client), threat.GetEntity()))
						{
							//Help aim towards the desired target point
							float aimPos[3]; myBot.GetIntentionInterface().SelectTargetPoint(threat.GetEntity(), aimPos);
							SnapViewToPosition(client, aimPos);
							
							if (m_flNextSnipeFireTime[client] <= GetGameTime())
								VS_PressFireButton(client);
						}
						else
						{
							//Delay to give a reaction time the next time we can see a threat
							m_flNextSnipeFireTime[client] = GetGameTime() + SNIPER_REACTION_TIME;
						}
					}
					else
					{
						if (threat && threat.IsVisibleInFOVNow() && myBot.GetBodyInterface().IsHeadAimingOnTarget())
						{
							if (m_flNextSnipeFireTime[client] <= GetGameTime())
								VS_PressFireButton(client);
						}
						else
						{
							m_flNextSnipeFireTime[client] = GetGameTime() + SNIPER_REACTION_TIME;
						}
					}
				}
				else
				{
					//Set a reaction time when we're not scoped in
					m_flNextSnipeFireTime[client] = GetGameTime() + SNIPER_REACTION_TIME;
				}
			}
			else
			{
				if (threat)
				{
					//Exclude certain things for scenarios where aim shouldn't be altered
					//TODO: replace this with a variable to control this
					if (IsCombatWeapon(client, myWeapon) && weaponID != TF_WEAPON_KNIFE && TF2_GetPlayerClass(client) != TFClass_Engineer && weaponID != TF_WEAPON_BONESAW)
					{
						int iThreat = threat.GetEntity();
						
						if (redbots_manager_bot_aim_skill.IntValue >= 2)
						{
							/* NOTE: this used to be handled in CTFBotMainAction_SelectTargetPoint, but it seems that function doesn't always get called when the bot is up close to it
							The bot will look up, but then start looking towards the center again and stop firing before going to look up and fire again
							It then just repeats this process over and over unless it gets away from the tank */
							if (weaponID == TF_WEAPON_FLAMETHROWER && IsBaseBoss(iThreat) && myBot.IsRangeLessThan(iThreat, FLAMETHROWER_REACH_RANGE))
							{
								float aimPos[3]; GetFlameThrowerAimForTank(iThreat, aimPos);
								SnapViewToPosition(client, aimPos);
								buttons |= IN_ATTACK;
							}
							else if (!threat.IsVisibleInFOVNow() && IsLineOfFireClearEntity(client, GetEyePosition(client), iThreat))
							{
								//We're not currently facing our threat, so let's quickly turn towards them
								float aimPos[3]; myBot.GetIntentionInterface().SelectTargetPoint(iThreat, aimPos);
								SnapViewToPosition(client, aimPos);
							}
						}
						else if (redbots_manager_bot_aim_skill.IntValue == 1)
						{
							if (weaponID == TF_WEAPON_FLAMETHROWER && IsBaseBoss(iThreat) && myBot.IsRangeLessThan(iThreat, FLAMETHROWER_REACH_RANGE))
							{
								float aimPos[3]; GetFlameThrowerAimForTank(iThreat, aimPos);
								SnapViewToPosition(client, aimPos);
								buttons |= IN_ATTACK;
							}
							else if (!threat.IsVisibleRecently() && IsLineOfFireClearEntity(client, GetEyePosition(client), iThreat))
							{
								float aimPos[3]; myBot.GetIntentionInterface().SelectTargetPoint(iThreat, aimPos);
								SnapViewToPosition(client, aimPos);
							}
						}
						else
						{
							if (weaponID == TF_WEAPON_FLAMETHROWER && IsBaseBoss(iThreat) && myBot.IsRangeLessThan(iThreat, FLAMETHROWER_REACH_RANGE))
							{
								float aimPos[3];
								GetFlameThrowerAimForTank(iThreat, aimPos);
								SnapViewToPosition(client, aimPos); //TODO: replace with AimHeadTowards
								buttons |= IN_ATTACK;
							}
						}
					}
				}
			}
#endif
			
#if defined MOD_ROLL_THE_DICE_REVAMPED
			if (redbots_manager_bot_rtd_variance.FloatValue >= COMMAND_MAX_RATE)
			{
				if (threat && threat.IsVisibleInFOVNow() && m_flNextRollTime[client] <= GetGameTime())
				{
					m_flNextRollTime[client] = GetGameTime() + GetRandomFloat(COMMAND_MAX_RATE, redbots_manager_bot_rtd_variance.FloatValue);
					FakeClientCommand(client, "sm_rtd");
				}
			}
#endif
			
#if defined TESTING_ONLY
			if (GetEntityFlags(client) & FL_ONGROUND == 0 && !TF2_IsJumping(client))
			{
				//TFBots have no air control in mvm, keep us moving
				PathFollower myPath = myBot.GetCurrentPath();
				
				if (myPath)
				{
					Segment pGoal = myPath.GetCurrentGoal();
					
					if (pGoal)
					{
						float vGoal[3]; pGoal.GetPosition(vGoal);
						MovePlayerTowardsGoal(client, vGoal, vel);
					}
				}
			}
#endif
		}
		
		//TODO: is this too expensive? use global per-player variable otherwise
		if (TF2_IsInUpgradeZone(client) && ActionsManager.LookupEntityActionByName(client, "DefenderUpgrade") != INVALID_ACTION)
		{
			//Because of CTFBot::AvoidPlayers, do not let ourselves move away from other players while upgrading
			vel = NULL_VECTOR;
		}
		
#if defined IDLEBOT_AIMING
		BotAim(client).Upkeep();
		BotAim(client).FireWeaponAtEnemy();
#endif
	}
	else
	{
		if (m_flDeadRethinkTime[client] <= GetGameTime())
		{
			//Think every second while we're dead
			m_flDeadRethinkTime[client] = GetGameTime() + 1.0;
			
			int iObsMode = BasePlayer_GetObserverMode(client);
			
			if (iObsMode == OBS_MODE_FREEZECAM || iObsMode == OBS_MODE_DEATHCAM)
			{
				//We can't buyback right now, so don't even think about it
				g_iBuybackNumber[client] = 0;
			}
			else
			{
				//Randomly think about buying back
				g_iBuybackNumber[client] = GetRandomInt(1, 100);
			}
			
			if (ShouldBuybackIntoGame(client))
				PlayerBuyback(client);
			
			if (redbots_manager_debug.BoolValue)
				PrintToChatAll("[OnPlayerRunCmd] g_iBuybackNumber[%d] = %d", client, g_iBuybackNumber[client]);
		}
		
		
	}
	
	return Plugin_Continue;
}

public void TF2_OnConditionAdded(int client, TFCond condition)
{
	if (condition == TFCond_Taunting && TF2_GetClientTeam(client) == TFTeam_Blue && IsSentryBusterRobot(client))
	{
		//Keep track of the player that is detonating
		g_iDetonatingPlayer = client;
		CreateTimer(2.0, Timer_ForgetDetonatingPlayer, client);
	}
}

public Action Command_Votebots(int client, int args)
{
	if (g_bBotsEnabled)
	{
		ReplyToCommand(client, "%s Bots are already enabled for this round.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (redbots_manager_mode.IntValue != MANAGER_MODE_MANUAL_BOTS)
	{
		ReplyToCommand(client, "%s This is only allowed in MANAGER_MODE_MANUAL_BOTS.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (g_flNextReadyTime > GetGameTime())
	{
		ReplyToCommand(client, "%s You're going too fast!", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (IsServerFull())
	{
		ReplyToCommand(client, "%s Server is at max capacity.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (GameRules_GetRoundState() != RoundState_BetweenRounds)
	{
		ReplyToCommand(client, "%s This cannot be used at this time.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (IsVoteInProgress())
	{
		ReplyToCommand(client, "%s A vote is already in progress.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (redbots_manager_bot_lineup_mode.IntValue == BOT_LINEUP_MODE_CHOOSE)
	{
		if (!HavePlayersChosenBotTeam())
		{
			if (g_bChoosingBotClasses[client])
			{
				ReplyToCommand(client, "%s You are already choosing the next team lineup.", PLUGIN_PREFIX);
				return Plugin_Handled;
			}
			
			if (GetCountOfPlayersChoosingBotClasses() > 0)
			{
				ReplyToCommand(client, "%s Someone is currently choosing the next team lineup.", PLUGIN_PREFIX);
				return Plugin_Handled;
			}
			
			ReplyToCommand(client, "%s Choose your bot team lineup first! Use command !choosebotteam or !cbt", PLUGIN_PREFIX);
			return Plugin_BadLoad;
		}
	}
	
	switch (TF2_GetClientTeam(client))
	{
		case TFTeam_Red:
		{
#if defined CHANGETEAM_RESTRICTIONS
			float botBanTime = g_flEnableBotsCooldown[client] - GetGameTime();
			
			if (botBanTime > 0.0)
			{
				ReplyToCommand(client, "%s You cannot start the bots at this time.", PLUGIN_PREFIX);
				LogAction(client, -1, "MANAGER_MODE_MANUAL_BOTS: %L tried to start the bots on cooldown. (%f seconds)", client, botBanTime);
				
				return Plugin_Handled;
			}
#endif
			
			if (GetHumanAndDefenderBotCount(TFTeam_Red) < redbots_manager_defender_team_size.IntValue)
			{
				StartBotVote(client);
				return Plugin_Handled;
			}
			else
			{
				ReplyToCommand(client, "%s RED team is full.", PLUGIN_PREFIX);
				return Plugin_Handled;
			}
		}
		default:
		{
			ReplyToCommand(client, "%s You cannot use this command on this team.", PLUGIN_PREFIX);
			return Plugin_Handled;
		}
	}
}

public Action Command_BotPreferences(int client, int args)
{
	DisplayMenu(g_hBotPreferenceMenu, client, MENU_TIME_FOREVER);
	return Plugin_Handled;
}

public Action Command_ShowBotChances(int client, int args)
{
	ShowCurrentBotClassChances(client);
	return Plugin_Handled;
}

public Action Command_ShowNewBotTeamComposition(int client, int args)
{
	if (!CreateDisplayPanelBotTeamComposition(client))
	{
		ReplyToCommand(client, "%s There is no bot lineup currently active.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	ReplyToCommand(client, "Use command !rerollbotclasses to reshuffle the bot class lineup.");
	
	return Plugin_Handled;
}

public Action Command_RerollNewBotTeamComposition(int client, int args)
{
#if !defined TESTING_ONLY
	if (TF2_GetClientTeam(client) != TFTeam_Red)
	{
		ReplyToCommand(client, "%s Your team is not allowed to use this.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
#endif
	
	switch (redbots_manager_bot_lineup_mode.IntValue)
	{
		case BOT_LINEUP_MODE_CHOOSE:
		{
			ReplyToCommand(client, "%s This cannot be used with the current lineup mode.", PLUGIN_PREFIX);
			return Plugin_Handled;
		}
	}
	
	UpdateChosenBotTeamComposition(client);
	CreateDisplayPanelBotTeamComposition(client);
	
	return Plugin_Handled;
}

public Action Command_JoinBluePlayWithBots(int client, int args)
{
	if (redbots_manager_mode.IntValue < MANAGER_MODE_MANUAL_BOTS)
	{
		ReplyToCommand(client, "%s Currently not allowed.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (g_bBotsEnabled)
	{
		ReplyToCommand(client, "%s Bots are already enabled for this round.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (TF2_GetClientTeam(client) != TFTeam_Blue)
	{
		ReplyToCommand(client, "%s Your team is not allowed to use this.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (GetHumanAndDefenderBotCount(TFTeam_Red) > 0)
	{
		ReplyToCommand(client, "%s You cannot use this with players on RED team.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	AddRandomDefenderBots(redbots_manager_defender_team_size.IntValue); //TODO: replace me with a smarter team comp
	g_bBotsEnabled = true;
	PrintToChatAll("%s You will play a game with bots.", PLUGIN_PREFIX);
	
	return Plugin_Handled;
}

public Action Command_RequestExtraBot(int client, int args)
{
	if (!g_bBotsEnabled)
	{
		ReplyToCommand(client, "%s Bots aren't enabled.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (g_flAddingBotTime > GetGameTime())
	{
		return Plugin_Handled;
	}
	
	if (TF2_GetClientTeam(client) != TFTeam_Red)
	{
		ReplyToCommand(client, "%s Your team is not allowed to use this.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (IsServerFull())
	{
		ReplyToCommand(client, "%s It is currently not possible to add any more.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	int defenderLimit = redbots_manager_defender_team_size.IntValue + redbots_manager_extra_bots.IntValue;
	
	if (GetHumanAndDefenderBotCount(TFTeam_Red) >= defenderLimit)
	{
		ReplyToCommand(client, "%s You already have an additional bot.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	g_flAddingBotTime = GetGameTime() + 0.1;
	
	if (args > 0)
	{
		char arg1[TF2_CLASS_MAX_NAME_LENGTH]; GetCmdArg(1, arg1, sizeof(arg1));
		
		if (strcmp(arg1, "random", false) == 0)
		{
			AddRandomDefenderBots(1);
			return Plugin_Handled;
		}
		
		TFClassType class = TF2_GetClassIndexFromString(arg1);
		
		if (class == TFClass_Unknown)
		{
			ReplyToCommand(client, "%s Invalid class specified: %s.", PLUGIN_PREFIX, arg1);
			return Plugin_Handled;
		}
		
		AddDefenderTFBot(1, arg1);
		PrintToChatAll("%s %N requested an additional \"%s\" bot.", PLUGIN_PREFIX, client, arg1);
		
		return Plugin_Handled;
	}
	
	AddBotsBasedOnLineupMode(1);
	PrintToChatAll("%s %N requested an additional bot.", PLUGIN_PREFIX, client);
	
	return Plugin_Handled;
}

public Action Command_ChooseBotClasses(int client, int args)
{
	if (g_bBotsEnabled)
	{
		ReplyToCommand(client, "%s Bots are already enabled.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (redbots_manager_bot_lineup_mode.IntValue != BOT_LINEUP_MODE_CHOOSE)
	{
		ReplyToCommand(client, "%s Not allowed in the current manager lineup mode.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (TF2_GetClientTeam(client) != TFTeam_Red)
	{
		ReplyToCommand(client, "%s Your team is not allowed to use this.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (g_bBotClassesLocked)
	{
		ReplyToCommand(client, "%s Someone has already chosen the lineup for the next game.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (g_bChoosingBotClasses[client])
	{
		ReplyToCommand(client, "%s You are already choosing the next team lineup.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (GetCountOfPlayersChoosingBotClasses() > 0)
	{
		ReplyToCommand(client, "%s Someone is currently choosing the next team lineup.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (GameRules_GetRoundState() != RoundState_BetweenRounds)
	{
		ReplyToCommand(client, "%s This can only be used between waves.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	int redTeamCount = GetHumanAndDefenderBotCount(TFTeam_Red);
	int defenderTeamSize = redbots_manager_defender_team_size.IntValue;
	
	if (redTeamCount >= defenderTeamSize)
	{
		ReplyToCommand(client, "%s You are not solo.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	//Should only be able to call this while solo, so current team count should always be 1
	ShowDefenderBotTeamSetupMenu(client, _, true, defenderTeamSize - redTeamCount);
	PrintToChatAll("%N is choosing the current bot team lineup.", client);
	
	return Plugin_Handled;
}

public Action Command_RedoBotTeamLineup(int client, int args)
{
	if (!g_bBotsEnabled)
	{
		ReplyToCommand(client, "%s The bots aren't here, dummy.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (!g_bAllowBotTeamRedo)
	{
		ReplyToCommand(client, "%s This is currently not allowed.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (TF2_GetClientTeam(client) != TFTeam_Red)
	{
		ReplyToCommand(client, "%s Your team is not allowed to use this.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (g_bChoosingBotClasses[client])
	{
		ReplyToCommand(client, "%s You are already choosing the next team lineup.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	if (GetCountOfPlayersChoosingBotClasses() > 0)
	{
		ReplyToCommand(client, "%s Someone is currently choosing the next team lineup.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}
	
	switch (redbots_manager_bot_lineup_mode.IntValue)
	{
		case BOT_LINEUP_MODE_RANDOM:
		{
			g_bBotsEnabled = false;
			RemoveAllDefenderBots("DB redo bots");
			g_bBotClassesLocked = false;
			UpdateChosenBotTeamComposition();
		}
		case BOT_LINEUP_MODE_PREFERENCE:
		{
			g_bBotsEnabled = false;
			RemoveAllDefenderBots("DB redo bots");
			g_bBotClassesLocked = false;
			UpdateChosenBotTeamComposition();
		}
		case BOT_LINEUP_MODE_CHOOSE:
		{
			g_bBotsEnabled = false;
			RemoveAllDefenderBots("DB redo bots");
			FreeChosenBotTeam(false);
			Command_ChooseBotClasses(client, 0); //Lazy
		}
		case BOT_LINEUP_MODE_PREFERENCE_CHOOSE:
		{
			g_bBotsEnabled = false;
			RemoveAllDefenderBots("DB redo bots");
			g_bBotClassesLocked = false;
			UpdateChosenBotTeamComposition();
		}
	}
	
	//Solo players are always allowed to repick their bot lineup
	g_bAllowBotTeamRedo = GetTeamHumanClientCount(TFTeam_Red) == 1;
	
	PrintToChatAll("%s %N has decided to repick the bot team lineup.", PLUGIN_PREFIX, client);
	LogAction(client, -1, "%L triggered defender bot redo", client);
	
	return Plugin_Handled;
}

#if defined TESTING_ONLY
public Action Command_BotsReadyNow(int client, int args)
{
	int target = GetClientAimTarget(client);
	SpawnSapper(client, target);
	
	return Plugin_Handled;
}
#endif

public Action Command_AddBots(int client, int args)
{
	if (args > 0)
	{
		char arg1[3]; GetCmdArg(1, arg1, sizeof(arg1));
		int amount = StringToInt(arg1);
		AddBotsBasedOnLineupMode(amount, false);
		
		return Plugin_Handled;
	}
	
	CreateDisplayMenuAddDefenderBots(client);
	return Plugin_Handled;
}

public Action Command_RemoveAllBots(int client, int args)
{
	if (args > 0)
	{
		char arg1[3]; GetCmdArg(1, arg1, sizeof(arg1));
		
		if (StringToInt(arg1) == 1)
			ManageDefenderBots(false);
	}
	
	RemoveAllDefenderBots("Admin request");
	ShowActivity2(client, "[SM] ", "Purged all bots.");
	
	return Plugin_Handled;
}

public Action Command_StopManagingBots(int client, int args)
{
	ManageDefenderBots(false);
	ReplyToCommand(client, "Stopped manaing bots.");
	
	return Plugin_Handled;
}

/* Every upgrade the game holds, by the index it holds it at

tf2-archipelago names an upgrade by counting "attribute" lines in
scripts/items/mvm_upgrades.txt and taking the Nth one, which assumes the game numbers them in
file order and skips nothing. That assumption has never been checked against the game.

This prints what the game itself says, so the two can be compared. If they disagree, every
purchase that mod reports is named after the wrong upgrade, and the fix is there rather than here.

An upgrade with no attribute is printed as well rather than skipped: a gap in the numbering is
exactly the thing that would break counting lines, so hiding it would hide the answer. */
//A list far longer than this is the manager not being what we think it is, not a big mission
#define DUMP_UPGRADES_MAX	1024

public Action Command_DumpUpgrades(int client, int args)
{
	CMannVsMachineUpgradeManager manager = CMannVsMachineUpgradeManager();
	
	if (manager.Address == Address_Null)
	{
		ReplyToCommand(client, "[SM] The upgrade manager is not up yet. Load an MvM map first.");
		return Plugin_Handled;
	}
	
	int count = manager.Count();
	
	if (count < 1 || count > DUMP_UPGRADES_MAX)
	{
		ReplyToCommand(client, "[SM] The manager says it holds %d upgrades, which is not believable.", count);
		return Plugin_Handled;
	}
	
	ReplyToCommand(client, "[SM] %d upgrades, by the index the game uses:", count);
	LogMessage("sm_dump_upgrades: %d upgrades", count);
	
	for (int i = 0; i < count; i++)
	{
		CMannVsMachineUpgrades upgrade = manager.GetUpgradeByIndex(i);
		
		char attribute[MAX_ATTRIBUTE_DESCRIPTION_LENGTH];
		
		if (upgrade.Address != Address_Null)
			attribute = upgrade.m_szAttribute();
		
		ReplyToCommand(client, "%d %s", i, attribute[0] == '\0' ? "(none)" : attribute);
		LogMessage("%d %s", i, attribute[0] == '\0' ? "(none)" : attribute);
	}
	
	return Plugin_Handled;
}

/* Where you are standing, as a map configuration block

The nest, teleporter and sniper locations in configs/defenderbots/map are all somebody standing on
the ground they meant and writing down where that was. This prints the line to write down, so the
map data can be authored in the map instead of guessed from a compiled brush

Usage: sm_dump_spot <block> [aim]

Standing on the spot is the accurate way and stays the default. The aim mode is for noclip: it
traces the crosshair to the world and writes down what it hit, so a whole map can be marked from
above without landing on every spot. It refuses a trace that hits nothing, since a spot in the
skybox is worse than no spot */
#define DUMP_SPOT_AIM_RANGE	8192.0

public Action Command_DumpSpot(int client, int args)
{
	if (client < 1 || !IsClientInGame(client))
	{
		ReplyToCommand(client, "[SM] This command requires standing somewhere in the map.");
		return Plugin_Handled;
	}
	
	char block[64] = "EngineerNest";
	
	if (args >= 1)
		GetCmdArg(1, block, sizeof(block));
	
	char mode[16];
	
	if (args >= 2)
		GetCmdArg(2, mode, sizeof(mode));
	
	float origin[3];
	
	if (StrEqual(mode, "aim", false))
	{
		if (!TraceAimToWorld(client, origin))
		{
			ReplyToCommand(client, "[SM] Your crosshair is not on anything within %.0f units.", DUMP_SPOT_AIM_RANGE);
			return Plugin_Handled;
		}
	}
	else
	{
		GetClientAbsOrigin(client, origin);
	}
	
	char mapName[PLATFORM_MAX_PATH]; GetCurrentMap(mapName, sizeof(mapName));
	
	ReplyToCommand(client, "[SM] %s on %s:", block, mapName);
	ReplyToCommand(client, "\t\t\t\"origin\" \"%.0f %.0f %.0f\"", origin[0], origin[1], origin[2]);
	
	LogMessage("%s %s: \"origin\" \"%.0f %.0f %.0f\"", mapName, block, origin[0], origin[1], origin[2]);
	
	return Plugin_Handled;
}

//The world under the crosshair. Brushes and props only: a spot written down off a teammate's head is a spot nobody can build on
static bool TraceAimToWorld(int client, float result[3])
{
	float eyes[3]; GetClientEyePosition(client, eyes);
	float angles[3]; GetClientEyeAngles(client, angles);
	
	Handle trace = TR_TraceRayFilterEx(eyes, angles, MASK_SOLID, RayType_Infinite, TraceFilter_IgnorePlayers, client);
	
	bool hit = TR_DidHit(trace);
	
	if (hit)
		TR_GetEndPosition(result, trace);
	
	delete trace;
	
	if (!hit || GetVectorDistance(eyes, result) > DUMP_SPOT_AIM_RANGE)
		return false;
	
	return true;
}

static bool TraceFilter_IgnorePlayers(int entity, int mask, any data)
{
	return entity > MaxClients;
}

public Action Command_ViewBotUpgrades(int client, int args)
{
	if (args < 1)
	{
		ReplyToCommand(client, "[SM] Usage: sm_view_bot_upgrades <#userid|name> <slot>");
		return Plugin_Handled;
	}
	
	char arg[65]; GetCmdArg(1, arg, sizeof(arg));
	
	char target_name[MAX_TARGET_LENGTH];
	int target_list[MAXPLAYERS], target_count;
	bool tn_is_ml;
	
	if ((target_count = ProcessTargetString(
			arg,
			client,
			target_list,
			MAXPLAYERS,
			COMMAND_FILTER_ALIVE,
			target_name,
			sizeof(target_name),
			tn_is_ml)) <= 0)
	{
		ReplyToTargetError(client, target_count);
		return Plugin_Handled;
	}
	
	int slot = -1;
	
	if (args >= 2)
	{
		char arg2[3]; GetCmdArg(2, arg2, sizeof(arg2));
		
		slot = StringToInt(arg2);
	}
	
	for (int i = 0; i < target_count; i++)
		ShowPlayerUpgrades(client, target_list[i], slot);
	
	return Plugin_Handled;
}

public Action Command_ForcePlayerPreference(int client, int args)
{
	if (args < 1)
	{
		ReplyToCommand(client, "[SM] Usage: sm_db_use_pref_of_player <#userid|name>");
		return Plugin_Handled;
	}
	
	char arg[4]; GetCmdArg(1, arg, sizeof(arg));
	
	//TODO: this is a terrible mockup, please change this
	//We only want one target at a time here
	if (!strcmp(arg, "@me"))
	{
		g_iPlayerForcedPref = client;
		return Plugin_Handled;
	}
	
	//TODO: for admin to force use someone else's instead
	
	return Plugin_Handled;
}

public void ConVarChanged_ManagerMode(ConVar convar, const char[] oldValue, const char[] newValue)
{
	int mode = StringToInt(newValue);
	
	//TODO: really only here for legacy reasons
	//Catch all cases of everything!
}

public void ConVarChanged_BotLineupMode(ConVar convar, const char[] oldValue, const char[] newValue)
{
	int mode = StringToInt(newValue);
	
	switch (mode)
	{
		case BOT_LINEUP_MODE_RANDOM:
		{
			UpdateChosenBotTeamComposition();
		}
		case BOT_LINEUP_MODE_PREFERENCE:
		{
			UpdateChosenBotTeamComposition();
		}
		case BOT_LINEUP_MODE_CHOOSE:
		{
			FreeChosenBotTeam(true);
		}
		case BOT_LINEUP_MODE_PREFERENCE_CHOOSE:
		{
			FreeChosenBotTeam(true);
			UpdateChosenBotTeamComposition();
		}
	}
}

public Action Listener_TournamentPlayerReadystate(int client, const char[] command, int argc)
{
	//Always let bots pass
	if (g_bIsDefenderBot[client])
		return Plugin_Continue;
	
	switch (redbots_manager_mode.IntValue)
	{
		case MANAGER_MODE_MANUAL_BOTS:
		{
			if (TF2_GetClientTeam(client) != TFTeam_Red)
				return Plugin_Continue;
			
			//Admin probably added bots
			if (GetDefenderBotCount(TFTeam_Red) > 0)
				return Plugin_Continue;
			
			char arg1[2]; GetCmdArg(1, arg1, sizeof(arg1));
			int value = StringToInt(arg1);
			
			//0 means we unready, let it pass
			if (value < 1)
				return Plugin_Continue;
			
			//Allow players that are ready to unready
			if (IsPlayerReady(client))
				return Plugin_Continue;
			
			if (redbots_manager_min_players.IntValue != -1)
			{
				eMissionDifficulty difficulty = GetMissionDifficulty();
				int defenderTeamSize = redbots_manager_defender_team_size.IntValue;
				int minPlayers = redbots_manager_min_players.IntValue;
				int trueMinPlayers;
				
				switch (difficulty)
				{
					case MISSION_NORMAL:
					{
						//Don't go over the max amount of red players
						trueMinPlayers = minPlayers > defenderTeamSize ? defenderTeamSize : minPlayers;
						
						//Block ready status if we don't have enough players
						if (GetHumanAndDefenderBotCount(TFTeam_Red) < trueMinPlayers)
						{
							PrintToChat(client, "%s More players are required.", PLUGIN_PREFIX);
							return Plugin_Handled;
						}
					}
					case MISSION_INTERMEDIATE:
					{
						trueMinPlayers = minPlayers + 1 > defenderTeamSize ? defenderTeamSize : minPlayers + 1;
						
						if (GetHumanAndDefenderBotCount(TFTeam_Red) < trueMinPlayers)
						{
							PrintToChat(client, "%s More players are required.", PLUGIN_PREFIX);
							return Plugin_Handled;
						}
					}
					case MISSION_ADVANCED:
					{
						trueMinPlayers = minPlayers + 2 > defenderTeamSize ? defenderTeamSize : minPlayers + 2;
						
						if (GetHumanAndDefenderBotCount(TFTeam_Red) < trueMinPlayers)
						{
							PrintToChat(client, "%s More players are required.", PLUGIN_PREFIX);
							return Plugin_Handled;
						}
					}
					case MISSION_EXPERT:
					{
						trueMinPlayers = minPlayers + 3 > defenderTeamSize ? defenderTeamSize : minPlayers + 3;
						
						if (GetHumanAndDefenderBotCount(TFTeam_Red) < trueMinPlayers)
						{
							PrintToChat(client, "%s More players are required.", PLUGIN_PREFIX);
							return Plugin_Handled;
						}
					}
					case MISSION_NIGHTMARE:
					{
						trueMinPlayers = minPlayers + 4 > defenderTeamSize ? defenderTeamSize : minPlayers + 4;
						
						if (GetHumanAndDefenderBotCount(TFTeam_Red) < trueMinPlayers)
						{
							PrintToChat(client, "%s More players are required.", PLUGIN_PREFIX);
							return Plugin_Handled;
						}
					}
					default:	LogError("Listener_Readystate: Unknown difficulty returned!");
				}
			}
		}
		case MANAGER_MODE_READY_BOTS:
		{
			if (TF2_GetClientTeam(client) != TFTeam_Red)
				return Plugin_Continue;
			
			if (GetDefenderBotCount(TFTeam_Red) > 0)
				return Plugin_Continue;
			
			if (!ShouldProcessCommand(client))
				return Plugin_Handled;
			
			if (g_bBotsEnabled)
			{
				//Bots already going, okay to pass
				return Plugin_Continue;
			}
			else
			{
				if (g_flNextReadyTime > GetGameTime())
				{
					PrintToChat(client, "%s You're going too fast!", PLUGIN_PREFIX);
					
					//Give more time to ready dawg
					return Plugin_Handled;
				}
				
#if defined CHANGETEAM_RESTRICTIONS
				float botBanTime = g_flEnableBotsCooldown[client] - GetGameTime();
				
				if (botBanTime > 0.0)
				{
					ReplyToCommand(client, "%s You cannot start the bots at this time.", PLUGIN_PREFIX);
					LogAction(client, -1, "MANAGER_MODE_READY_BOTS: %L tried to start the bots on cooldown. (%f seconds)", client, botBanTime);
					
					return Plugin_Handled;
				}
#endif
				
				if (redbots_manager_bot_lineup_mode.IntValue == BOT_LINEUP_MODE_CHOOSE)
				{
					if (!HavePlayersChosenBotTeam())
					{
						if (GetCountOfPlayersChoosingBotClasses() > 0)
						{
							PrintToChat(client, "%s Someone is currently choosing the next team lineup.", PLUGIN_PREFIX);
							return Plugin_Handled;
						}
						
						PrintToChat(client, "%s Choose your bot team lineup first! Use command !choosebotteam or !cbt", PLUGIN_PREFIX);
						return Plugin_BadLoad;
					}
				}
				
				if (m_flLastReadyInputTime[client] <= GetGameTime())
				{
					m_flLastReadyInputTime[client] = GetGameTime() + 3.0;
					PrintToChat(client, "%s Press ready again to start the bots.", PLUGIN_PREFIX);
					
					return Plugin_Handled;
				}
				else
				{
					ManageDefenderBots(true);
					g_iUIDBotSummoner = GetClientUserId(client);
					
					return Plugin_Handled;
				}
			}
		}
	}
	
	return Plugin_Continue;
}

public Action SoundHook_General(int clients[MAXPLAYERS], int &numClients, char sample[PLATFORM_MAX_PATH], int &entity, int &channel, float &volume, int &level, int &pitch, int &flags, char soundEntry[PLATFORM_MAX_PATH], int &seed)
{
	if (channel == SNDCHAN_VOICE && volume > 0.0 && BaseEntity_IsPlayer(entity))
	{
		if (StrContains(sample, "spy_mvm_LaughShort", false) != -1)
		{
			if (TF2_IsPlayerInCondition(entity, TFCond_Disguised) && !TF2_IsStealthed(entity))
			{
				/* Robots have robotic voices even when disguised so any
				defender bot that can see him right now will call him out */
				for (int i = 1; i <= MaxClients; i++)
				{
					if (i == entity)
						continue;
					
					if (!IsClientInGame(i))
						continue;
					
					if (g_bIsDefenderBot[i] == false)
						continue;
					
					if (GetClientTeam(entity) == GetClientTeam(i))
						continue;
					
					if (GetVectorDistance(GetAbsOrigin(i), WorldSpaceCenter(entity)) > redbots_manager_bot_hear_spy_range.FloatValue)
						continue;
					
					if (IsLineOfFireClearEntity(i, GetEyePosition(i), entity))
					{
						DataPack pack;
						CreateDataTimer(redbots_manager_bot_notice_spy_time.FloatValue, Timer_RealizeSpy, pack, TIMER_FLAG_NO_MAPCHANGE);
						pack.WriteCell(GetClientUserId(i));
						pack.WriteCell(GetClientUserId(entity));
						pack.Reset();
					}
				}
			}
		}
	}
	
	return Plugin_Continue;
}

public Action Timer_CheckBotImbalance(Handle timer)
{
	if (!g_bBotsEnabled)
		return Plugin_Stop;
	
	switch (redbots_manager_mode.IntValue)
	{
		case MANAGER_MODE_MANUAL_BOTS, MANAGER_MODE_READY_BOTS:
		{
			//Bots are added pre-round, but we can also monitor them during the round
			if (GameRules_GetRoundState() != RoundState_BetweenRounds && GameRules_GetRoundState() != RoundState_RoundRunning)
				return Plugin_Stop;
			
			int defenderCount = GetHumanAndDefenderBotCount(TFTeam_Red);
			
			if (defenderCount < redbots_manager_defender_team_size.IntValue)
			{
				int amount = redbots_manager_defender_team_size.IntValue - defenderCount;
				AddBotsBasedOnLineupMode(amount);
			}
		}
		case MANAGER_MODE_AUTO_BOTS:
		{
			//Bots are added when rhe wave begins, only monitor them during the round
			if (GameRules_GetRoundState() != RoundState_RoundRunning)
				return Plugin_Stop;
			
			int defenderCount = GetHumanAndDefenderBotCount(TFTeam_Red);
			
			if (defenderCount < redbots_manager_defender_team_size.IntValue)
			{
				int amount = redbots_manager_defender_team_size.IntValue - defenderCount;
				AddBotsBasedOnLineupMode(amount);
			}
		}
	}
	
	return Plugin_Continue;
}

public Action Timer_ForgetDetonatingPlayer(Handle timer, any data)
{
	//They should have detonated by now
	
	//Another player might have started detonating
	//Don't forget the newest one so soon
	if (g_iDetonatingPlayer == data)
		g_iDetonatingPlayer = -1;
	
	return Plugin_Stop;
}

static float m_flHumanReadinessTime = -1.0;
static bool m_bHumansOnRed;
static bool m_bAnyHumanReadyOnRed;

/* Whether anybody who is not a bot is on RED, and whether any of them has said the team is ready

Read once per bot per frame, answered once per frame: the roster cannot change between two bots of
the same tick, and a walk of every client slot inside something that reads like a cheap question is
how four of this mod's per-frame costs got there. */
void RefreshHumanReadiness()
{
	if (m_flHumanReadinessTime == GetGameTime())
		return;
	
	m_flHumanReadinessTime = GetGameTime();
	m_bHumansOnRed = false;
	m_bAnyHumanReadyOnRed = false;
	
	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || IsFakeClient(i) || TF2_GetClientTeam(i) != TFTeam_Red)
			continue;
		
		m_bHumansOnRed = true;
		
		if (IsPlayerReady(i))
		{
			m_bAnyHumanReadyOnRed = true;
			break;
		}
	}
}

bool AnyHumanOnRed()
{
	RefreshHumanReadiness();
	
	return m_bHumansOnRed;
}

bool AnyHumanReadyOnRed()
{
	RefreshHumanReadiness();
	
	return m_bAnyHumanReadyOnRed;
}

public void Timer_ReadyPlayer(Handle timer, int data)
{
	if (!IsClientInGame(data))
		return;
	
	SetPlayerReady(data, true);
}

public void Timer_RealizeSpy(Handle timer, DataPack pack)
{
	int client = GetClientOfUserId(pack.ReadCell());
	
	if (client == 0)
		return;
	
	int threat = GetClientOfUserId(pack.ReadCell());
	
	if (threat == 0)
		return;
	
	TFBot_NoticeThreat(client, threat);
}

public void DefenderBot_TouchPost(int entity, int other)
{
	//Call out enemy spies upon contact
	if (BaseEntity_IsPlayer(other) && GetClientTeam(other) != GetClientTeam(entity) && TF2_IsPlayerInCondition(other, TFCond_Disguised))
	{
#if defined TFBOT_CUSTOM_SPY_CONTACT
		DataPack pack;
		CreateDataTimer(redbots_manager_bot_notice_spy_time.FloatValue, Timer_RealizeSpy, pack, TIMER_FLAG_NO_MAPCHANGE);
		pack.WriteCell(GetClientUserId(entity));
		pack.WriteCell(GetClientUserId(other));
		pack.Reset();
#else
		TFBot_NoticeThreat(entity, other);
#endif
	}
}

void FindGameConsoleVariables()
{
	nb_blind = FindConVar("nb_blind");
	tf_bot_path_lookahead_range = FindConVar("tf_bot_path_lookahead_range");
	tf_bot_health_critical_ratio = FindConVar("tf_bot_health_critical_ratio");
	tf_bot_health_ok_ratio = FindConVar("tf_bot_health_ok_ratio");
	tf_bot_ammo_search_range = FindConVar("tf_bot_ammo_search_range");
	tf_bot_health_search_far_range = FindConVar("tf_bot_health_search_far_range");
	tf_bot_health_search_near_range = FindConVar("tf_bot_health_search_near_range");
}

bool FakeClientCommandThrottled(int client, const char[] command)
{
	if (m_flLastCommandTime[client] > GetGameTime())
		return false;
	
	FakeClientCommand(client, command);
	
	m_flLastCommandTime[client] = GetGameTime() + 0.4;
	
	return true;
}

void MakePlayerDance(int client)
{
	if (IsPlayerAlive(client))
	{
		//TODO: tauntem
	}
}

/* What a bot is actually carrying, by name, printed wherever the command came from

Every line of this used PrintToChat, which needs a client, and rcon has not got one: run from the
console it printed nothing at all and looked like a bot with no upgrades. That is the one place
anybody would run it from on a test server.

The attribute index alone was not much better. "INDEX 56, VALUE 0.800000" is the answer to a
question nobody asked; the schema has the name and it costs a lookup. */
static void ShowUpgradesOn(int client, int entity, const char[] what)
{
	int attribIndexes[MAX_RUNTIME_ATTRIBUTES];
	int count = TF2Attrib_ListDefIndices(entity, attribIndexes, sizeof(attribIndexes));
	
	ReplyToCommand(client, "%s: %d upgrades", what, count);
	
	for (int i = 0; i < count; i++)
	{
		Address pAttr = TF2Attrib_GetByDefIndex(entity, attribIndexes[i]);
		float value = TF2Attrib_GetValue(pAttr);
		
		char name[128];
		
		if (!TF2Econ_GetAttributeName(attribIndexes[i], name, sizeof(name)))
			strcopy(name, sizeof(name), "(unnamed)");
		
		ReplyToCommand(client, "  %-48s %.3f", name, value);
	}
}

void ShowPlayerUpgrades(int client, int target, int slot)
{
	char who[MAX_NAME_LENGTH]; GetClientName(target, who, sizeof(who));
	
	if (slot == -1)
	{
		char label[160]; FormatEx(label, sizeof(label), "%s, on himself", who);
		
		ShowUpgradesOn(client, target, label);
		
		return;
	}
	
	int weapon = GetPlayerWeaponSlot(target, slot);
	
	if (weapon == -1)
	{
		ReplyToCommand(client, "%s has nothing in slot %d.", who, slot);
		
		return;
	}
	
	char label[160]; FormatEx(label, sizeof(label), "%s, slot %d", who, slot);
	
	ShowUpgradesOn(client, weapon, label);
}

int GetHumanAndDefenderBotCount(TFTeam team)
{
	int count = 0;
	
	for (int i = 1; i <= MaxClients; i++)
		if (IsClientInGame(i) && (IsDefenderBot(i) || !IsFakeClient(i)) && TF2_GetClientTeam(i) == team)
			count++;
	
	return count;
}

/* One of ours, including one that has not spawned yet
g_bIsDefenderBot is set on the first spawn, and between tf_bot_add and that spawn the bot is on the
team and counts for nothing. The top-up timer runs every second in that window and adds one more,
which is how a six man team ends up with seven or eight members. The name is set at tf_bot_add */
bool IsDefenderBot(int client)
{
	if (g_bIsDefenderBot[client])
		return true;
	
	if (!IsFakeClient(client))
		return false;
	
	char clientName[MAX_NAME_LENGTH]; GetClientName(client, clientName, sizeof(clientName));
	
	return StrContains(clientName, TFBOT_IDENTITY_NAME) != -1;
}

int GetDefenderBotCount(TFTeam team)
{
	int count = 0;
	
	for (int i = 1; i <= MaxClients; i++)
		if (IsClientInGame(i) && g_bIsDefenderBot[i] && TF2_GetClientTeam(i) == team)
			count++;
	
	return count;
}

int GetCountOfPlayersChoosingBotClasses()
{
	int count = 0;
	
	for (int i = 1; i <= MaxClients; i++)
		if (IsClientInGame(i) && g_bChoosingBotClasses[i])
			count++;
	
	return count;
}

/* Used to check players last command input
Usually for preventing palyers from sending a command multiple times in a single frame */
bool ShouldProcessCommand(int client)
{
	if (m_flLastCommandTime[client] > GetGameTime())
		return false;
	
	m_flLastCommandTime[client] = GetGameTime() + COMMAND_MAX_RATE;
	return true;
}

int GetRealPlayerCount()
{
	int count = 0;
	
	for (int i = 1; i <= MaxClients; i++)
		if (IsClientInGame(i) && !IsFakeClient(i))
			count++;
	
	return count;
}

/* Put a bot back in the slot a player just left

Runs a tick after the disconnect, because the leaving player is still in the game at the point the
forward fires and would otherwise still be counted. Nobody is left to play with if the last player
leaves, so an empty defending team is left empty rather than filled with six bots holding a hatch
for no one */
static Action Timer_RefillDefenderTeam(Handle timer)
{
	if (!g_bBotsEnabled)
		return Plugin_Stop;
	
	if (GetRealPlayerCount() < 1)
		return Plugin_Stop;
	
	int missing = redbots_manager_defender_team_size.IntValue - GetHumanAndDefenderBotCount(TFTeam_Red);
	
	if (missing > 0)
		AddBotsBasedOnLineupMode(missing);
	
	return Plugin_Stop;
}

/* Free a defender slot for somebody who just connected

The bots are not kicked between waves any more, because a kicked bot is a bot that paid for its
upgrades and left them behind. That leaves nothing to open a slot: the game caps the defending
team, the bots fill it, and a player who joins the server after the mission started is told the
team is full for the rest of it.

A dead bot goes first, since kicking one costs the team nothing it still had */
void MakeRoomForHumanPlayer(int client)
{
	if (GetHumanAndDefenderBotCount(TFTeam_Red) < redbots_manager_defender_team_size.IntValue)
		return;
	
	int victim = -1;
	
	for (int i = 1; i <= MaxClients; i++)
	{
		if (i == client || !IsClientInGame(i) || !g_bIsDefenderBot[i])
			continue;
		
		if (TF2_GetClientTeam(i) != TFTeam_Red)
			continue;
		
		if (!IsPlayerAlive(i))
		{
			victim = i;
			break;
		}
		
		if (victim == -1)
			victim = i;
	}
	
	if (victim == -1)
		return;
	
	KickClient(victim, "BotManager3: Making room for a player");
}

void RemoveAllDefenderBots(char[] reason = "", bool bDanceInstead = false)
{
	for (int i = 1; i <= MaxClients; i++)
	{
		if (IsClientInGame(i) && g_bIsDefenderBot[i])
		{
			//We dance on the final wave instead
			if (bDanceInstead)
			{
				MakePlayerDance(i);
				continue;
			}
			
			KickClient(i, reason);
		}
	}
}

static int m_iFindNameTries[MAXPLAYERS + 1];
void SetRandomNameOnBot(int client)
{
	char newName[MAX_NAME_LENGTH]; GetRandomDefenderBotName(newName, sizeof(newName));
	const int maxTries = 10;
	
	if (m_adtBotNames.Length > 0 && DoesAnyPlayerUseThisName(newName) && m_iFindNameTries[client] < maxTries)
	{
		m_iFindNameTries[client]++;
		
		//Someone's already using my name, mock them for it and try again
		PrintToChatAll("%s : %s", newName, g_sPlayerUseMyNameResponse[GetRandomInt(0, sizeof(g_sPlayerUseMyNameResponse) - 1)]);
		SetRandomNameOnBot(client);
		
		return;
	}
	
	m_iFindNameTries[client] = 0;
	SetClientName(client, newName);
}

void GetRandomDefenderBotName(char[] buffer, int maxlen)
{
	if (m_adtBotNames.Length == 0)
	{
		strcopy(buffer, maxlen, "You forgot to give me a name!");
		return;
	}
	
	char botName[MAX_NAME_LENGTH]; m_adtBotNames.GetString(GetRandomInt(0, m_adtBotNames.Length - 1), botName, sizeof(botName));
	
	strcopy(buffer, maxlen, botName);
}

void ManageDefenderBots(bool bManage, bool bAddBots = true)
{
	if (bManage)
	{
		if (bAddBots)
			AddBotsFromChosenTeamComposition();
		
		CreateTimer(1.0, Timer_CheckBotImbalance, _, TIMER_FLAG_NO_MAPCHANGE | TIMER_REPEAT);
		g_bBotsEnabled = true;
		
		PrintToChatAll("%s Bots have been enabled.", PLUGIN_PREFIX);
	}
	else
	{
		g_bBotsEnabled = false;
	}
}

void AddBotsBasedOnLineupMode(int count, bool bAdjustTime = true)
{
	LogMessage("Fill: asked for %d, RED holds %d of %d", count, GetHumanAndDefenderBotCount(TFTeam_Red),
		redbots_manager_defender_team_size.IntValue);

	//The lineup mode fills what the named team left, and not the whole ask again: a three seat team
	//and an ask for six used to be nine bots on RED
	count -= AddBotsFromTeamComposition(count);

	if (count < 1)
	{
		if (bAdjustTime)
			ExtendUpgradeTimeForNewBots();

		return;
	}
	LogMessage("Fill: the lineup mode adds %d more", count);

	switch (redbots_manager_bot_lineup_mode.IntValue)
	{
		case BOT_LINEUP_MODE_RANDOM:
		{
			AddRandomDefenderBots(count);
		}
		case BOT_LINEUP_MODE_PREFERENCE, BOT_LINEUP_MODE_CHOOSE, BOT_LINEUP_MODE_PREFERENCE_CHOOSE:
		{
			AddBotsBasedOnPreferences(count);
		}
		default:
		{
			ThrowError("Unhandled lineup mode %d", redbots_manager_bot_lineup_mode.IntValue);
		}
	}
	
	if (bAdjustTime)
		ExtendUpgradeTimeForNewBots();
}

void ExtendUpgradeTimeForNewBots()
{
	float restartRoundTime = GameRules_GetPropFloat("m_flRestartRoundTime");

	if (restartRoundTime <= 0)
		return;

	if (restartRoundTime - GetGameTime() <= BUY_UPGRADES_MAX_TIME)
	{
		//Add a little more time for the new bot to ready
		GameRules_SetPropFloat("m_flRestartRoundTime", restartRoundTime + BUY_UPGRADES_MAX_TIME);
	}
}

/* The seats sm_redbots_manager_team_composition still wants filled, in its order, at most count of
them. Zero when the convar is empty, and the caller falls back to the lineup mode

The list is what the team should look like, not what to add: every call counts the bots already on
RED against it first and names only what is missing. So a top-up in the middle of a wave converges on
the same team as the first fill, whatever order the seats emptied in. A list shorter than the seats
leaves the rest to the lineup mode.

A seat is where a class sits in that list, counted from 1, and it comes out alongside the class name
because the loadout file can name one seat rather than every engineer at once */
int CollectMissingTeamComposition(ArrayList classes, ArrayList seats, int count)
{
	char list[128]; GetWantedTeamComposition(list, sizeof(list));

	if (list[0] == '\0')
		return 0;

	char wanted[MAXPLAYERS + 1][TF2_CLASS_MAX_NAME_LENGTH];
	int total = ExplodeString(list, ",", wanted, sizeof(wanted), sizeof(wanted[]));

	//How many bots of each class already hold a seat
	int held[view_as<int>(TFClass_Engineer) + 1];

	for (int i = 1; i <= MaxClients; i++)
		if (IsClientInGame(i) && IsDefenderBot(i) && TF2_GetClientTeam(i) == TFTeam_Red)
			held[view_as<int>(TF2_GetPlayerClass(i))]++;

	int collected = 0;

	for (int i = 0; i < total && collected < count; i++)
	{
		TrimString(wanted[i]);

		TFClassType class = TF2_GetClassIndexFromString(wanted[i]);

		if (class == TFClass_Unknown)
			continue;

		if (held[view_as<int>(class)] > 0)
		{
			held[view_as<int>(class)]--;
			continue;
		}

		classes.PushString(wanted[i]);
		seats.Push(i + 1);
		collected++;
	}

	return collected;
}

/* A reseat waiting for the break, because the composition was retyped mid-wave

Kicking a bot in the middle of a wave loses whatever it was doing and drops its buildings, and the
replacement walks in from spawn with the bomb halfway home. The break is where a lineup change is
free */
static bool m_bReseatPending;

//A whole-team recycle waiting for the same break, asked for by sm_redbots_reseat
static bool m_bRecyclePending;

/* What every bot on RED is holding, so a lineup change can be checked for losing it

A bot that changes class leaves and comes back, and the join path is what decides its balance. This
prints the number that path arrived at, next to the wave accounting it should match, because reading
SetCurrencyWithBundles is not the same as knowing what it produced */
public Action Command_DumpCredits(int client, int args)
{
	int earned = GetStartingCurrency(g_iPopulationManager) + GetAcquiredCreditsOfAllWaves();

	ReplyToCommand(client, "[SM] starting plus acquired is %d, before anything Archipelago paid", earned);

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || TF2_GetClientTeam(i) != TFTeam_Red)
			continue;

		char name[MAX_NAME_LENGTH]; GetClientName(i, name, sizeof(name));

		ReplyToCommand(client, "[SM] %-20s %-14s %6d credits%s", name, g_sRawPlayerClassNames[view_as<int>(TF2_GetPlayerClass(i))],
			TF2_GetCurrency(i), IsDefenderBot(i) ? "" : " (human)");
	}

	return Plugin_Handled;
}

/* Take a bot's buildings down before it leaves the server

A sentry outlives the engineer that placed it, and the mod holds hooks on both. Kicking the owner
and leaving the building standing is the shape of crash a reseat can cause and an ordinary wave
cannot, so every path that removes a bot mid-mission goes through here first.

Everything it owns, not only the sentry: a dispenser and both ends of a teleporter are the same
question */
void ClearBuildingsBeforeKick(int client)
{
	for (int i = PlayerObjectCount(client) - 1; i >= 0; i--)
	{
		int building = TF2Util_GetPlayerObject(client, i);

		if (IsValidEntity(building))
			RemoveEntity(building);
	}

	//The one the game already took out of that list, because he was carrying it
	int carried = TF2_GetCarriedObject(client);

	if (carried != -1 && IsValidEntity(carried))
		RemoveEntity(carried);
}

/* Send the whole team back through the join path, and say how many went

A lineup change kicks only the bots the new list has no seat for, which is right and is nobody at
all when the classes did not move. A loadout change moves no class and still has to reach every bot,
because a weapon is handed out on the way in and never again.

So this is the blunt one: everybody out, and the fill timer builds the team again from the
composition and the loadout file as they now stand */
int RecycleDefenderBots()
{
	int kicked = 0;

	for (int i = 1; i <= MaxClients; i++)
	{
		if (IsClientInGame(i) && IsDefenderBot(i) && TF2_GetClientTeam(i) == TFTeam_Red)
		{
			kicked++;
			ClearBuildingsBeforeKick(i);
			KickClient(i, "BotManager3: the team changed");
		}
	}

	return kicked;
}

/* Reload the loadout file and put the team back together with it

The launcher rewrites configs/defenderbots/loadout.cfg and then calls this, which is the whole of
applying a loadout change to a running server. Nothing else reads that file between map changes */
public Action Command_ReseatBots(int client, int args)
{
	Config_LoadServerLoadout();

	if (GameRules_GetRoundState() == RoundState_RoundRunning)
	{
		m_bRecyclePending = true;
		ReplyToCommand(client, "%s The new team takes effect when this wave ends.", PLUGIN_PREFIX);
		PrintToChatAll("%s The new team takes effect when this wave ends.", PLUGIN_PREFIX);
		return Plugin_Handled;
	}

	int kicked = RecycleDefenderBots();
	LogMessage("Reseat: the team was retyped, recycled %d bot(s)", kicked);
	ReplyToCommand(client, "%s Rebuilding %d bot(s) from the new team.", PLUGIN_PREFIX, kicked);

	if (kicked > 0)
		PrintToChatAll("%s Rebuilding %d bot(s) from the new team...", PLUGIN_PREFIX, kicked);

	return Plugin_Handled;
}

/* Kick the bots the lineup no longer asks for, and say how many went

CollectMissingTeamComposition only ever names seats nobody holds, so a full RED is already the team
it wants whatever the convar now says. Retyping the composition mid-mission therefore did nothing at
all. Kicking the surplus is what makes the next top-up converge on the new list.

Exactly as many bots go as the new list has seats nobody holds, so the team size never dips further
than the refill covers, and a list that names classes RED already fields kicks nobody.

A kicked bot comes back through the join path, where the seat, the loadout and
SetCurrencyWithBundles already live. Nothing here changes a class or refunds anything */
int ReseatDefenderBots()
{
	char list[128]; GetWantedTeamComposition(list, sizeof(list));

	if (list[0] == '\0')
		return 0;

	char wanted[MAXPLAYERS + 1][TF2_CLASS_MAX_NAME_LENGTH];
	int total = ExplodeString(list, ",", wanted, sizeof(wanted), sizeof(wanted[]));

	//Bots of a class the list no longer asks for, and the clients holding them
	int spare[view_as<int>(TFClass_Engineer) + 1];
	ArrayList bots = new ArrayList();

	for (int i = 1; i <= MaxClients; i++)
	{
		if (IsClientInGame(i) && IsDefenderBot(i) && TF2_GetClientTeam(i) == TFTeam_Red)
		{
			spare[view_as<int>(TF2_GetPlayerClass(i))]++;
			bots.Push(i);
		}
	}

	//Seats the list names that nobody holds, which is what there is room to kick
	int missing = 0;

	for (int i = 0; i < total; i++)
	{
		TrimString(wanted[i]);

		TFClassType class = TF2_GetClassIndexFromString(wanted[i]);

		//A blank or a typo leaves the seat to the lineup mode, so it asks for nobody in particular
		if (class == TFClass_Unknown)
			continue;

		if (spare[view_as<int>(class)] > 0)
			spare[view_as<int>(class)]--;
		else
			missing++;
	}

	int kicked = 0;

	for (int i = 0; i < bots.Length && kicked < missing; i++)
	{
		int client = bots.Get(i);

		//Rechecked rather than trusted: the list was taken before the first kick
		if (!IsClientInGame(client))
			continue;

		int class = view_as<int>(TF2_GetPlayerClass(client));

		if (spare[class] < 1)
			continue;

		spare[class]--;
		kicked++;
		ClearBuildingsBeforeKick(client);
		KickClient(client, "BotManager3: the lineup changed");
	}

	delete bots;

	if (kicked > 0)
	{
		LogMessage("Reseat: the lineup wants %d seat(s) nobody holds, kicked %d bot(s) for them", missing, kicked);
		PrintToChatAll("%s Changing %d bot(s) to match the new lineup...", PLUGIN_PREFIX, kicked);
	}

	return kicked;
}

public void ConVarChanged_TeamComposition(ConVar convar, const char[] before, const char[] after)
{
	if (StrEqual(before, after))
		return;

	if (GameRules_GetRoundState() == RoundState_RoundRunning)
	{
		m_bReseatPending = true;
		LogMessage("Reseat: the lineup changed mid-wave, holding it until the break");
		PrintToChatAll("%s The new lineup takes effect when this wave ends.", PLUGIN_PREFIX);
		return;
	}

	ReseatDefenderBots();
}

//The break has opened, so a lineup change that arrived mid-wave can happen now
void Reseat_OnBreak()
{
	if (m_bRecyclePending)
	{
		m_bRecyclePending = false;
		m_bReseatPending = false;
		LogMessage("Reseat: recycled %d bot(s) held from mid-wave", RecycleDefenderBots());
		return;
	}

	if (!m_bReseatPending)
		return;

	m_bReseatPending = false;
	ReseatDefenderBots();
}

//The round the pending reseat was waiting on ended with the map, and the bots it meant are gone
static void Reseat_OnMapStart()
{
	m_bReseatPending = false;
	m_bRecyclePending = false;
}

/* Fill the empty seats from the named team, and say how many it filled
Zero when the convar named nothing to add, and the caller asks the lineup mode for the rest */
int AddBotsFromTeamComposition(int count)
{
	ArrayList classes = new ArrayList(TF2_CLASS_MAX_NAME_LENGTH);
	ArrayList seats = new ArrayList();
	int added = CollectMissingTeamComposition(classes, seats, count);

	char class[TF2_CLASS_MAX_NAME_LENGTH];

	for (int i = 0; i < classes.Length; i++)
	{
		classes.GetString(i, class, sizeof(class));
		NoteBotSeatPending(seats.Get(i));
		AddDefenderTFBot(1, class, "red", "expert");
	}

	delete classes;
	delete seats;

	if (added > 0)
		PrintToChatAll("%s Adding %d bot(s)...", PLUGIN_PREFIX, added);

	LogMessage("Fill: the named team filled %d of %d", added, count);

	return added;
}

/* Decide what to do when a player decides to change their team
This is to prevent abuse of the system by leaving RED players with unfavorable teams */
void HandleTeamPlayerCountChanged(TFTeam team, int iWhoChanging = -1)
{
	if (GameRules_GetRoundState() != RoundState_BetweenRounds)
		return;
	
	if (redbots_manager_mode.IntValue == MANAGER_MODE_MANUAL_BOTS)
	{
		if (iWhoChanging > 0 && iWhoChanging == GetClientOfUserId(g_iUIDBotSummoner) && IsVoteInProgress())
		{
			//He started the bot vote then changed teams, cancel it
			CancelVote();
		}
	}
	switch (redbots_manager_bot_lineup_mode.IntValue)
	{
		case BOT_LINEUP_MODE_CHOOSE, BOT_LINEUP_MODE_PREFERENCE_CHOOSE:
		{
			//Allow the classes to be picked again, but don't clear current list
			g_bBotClassesLocked = false;
			PrintToChatTeam(team, "%s You can repick your bot team lineup.", PLUGIN_PREFIX);
		}
	}
	
	if (!g_bBotsEnabled)
		return;
	
	if (iWhoChanging > 0 && GetClientOfUserId(g_iUIDBotSummoner) == iWhoChanging)
	{
		//The summoner changed teams, allow RED team to repick their bots
		g_bAllowBotTeamRedo = true;
		PrintToChatTeam(team, "%s Use command !redobots to repick your bot team lineup.", PLUGIN_PREFIX);
	}
	
	int iWhoToUnready = -1;
	int iReadyCount = 0;
	int iMemberCount = 0;
	
	for (int i = 1; i <= MaxClients; i++)
	{
		//Whoever is changing teams won't count to the team count
		if (i == iWhoChanging)
			continue;
		
		if (!IsClientInGame(i))
			continue;
		
		if (TF2_GetClientTeam(i) != team)
			continue;
		
		if (IsPlayerReady(i))
		{
			if (iWhoToUnready != -1)
			{
				if (g_bIsDefenderBot[iWhoToUnready])
				{
					//Always prefer to unready human players first
					if (!g_bIsDefenderBot[i])
						iWhoToUnready = i;
				}
			}
			else
			{
				iWhoToUnready = i;
			}
			
			iReadyCount++;
		}
		
		iMemberCount++;
	}
	
	//Are all remaining members of the team ready?
	if (iReadyCount == iMemberCount)
	{
		//Unready one member to prevent starting the game and allow another bot to enter
		SetPlayerReady(iWhoToUnready, false);
		
		if (g_bIsDefenderBot[iWhoToUnready])
		{
			//Ready up the bot again after some time
			CreateTimer(0.2, Timer_ReadyPlayer, iWhoToUnready, TIMER_FLAG_NO_MAPCHANGE);
		}
	}
}

void AddDefenderTFBot(int count, char[] class, char[] team = "red", char[] difficulty = "expert", bool quotaManaged = false, bool honorBlacklist = true)
{
	char allowed[TF2_CLASS_MAX_NAME_LENGTH]; strcopy(allowed, sizeof(allowed), class);

	if (honorBlacklist)
		PickAllowedBotClass(class, allowed, sizeof(allowed));

	//Says why a bot is the class it is, which nothing did: the wanted class, what the blacklist left
	//of it, and whether the lineup was typed into the console or read off the map config
	if (!StrEqual(class, allowed) || redbots_manager_debug.BoolValue)
	{
		char typed[128]; redbots_manager_team_composition.GetString(typed, sizeof(typed));

		LogMessage("Adding %s (wanted %s), lineup from %s", allowed, class,
			typed[0] != '\0' ? "the convar" : (g_arrMapConfig.strComposition[0] != '\0' ? "the map config" : "the lineup mode"));
	}

	//Send command as many times as needed because custom names aren't supported when adding multiple
	for (int i = 0; i < count; i++)
		ServerCommand("tf_bot_add %d %s %s %s %s %s", 1, allowed, team, difficulty, quotaManaged ? "" : "noquota", TFBOT_IDENTITY_NAME);
}

//A blacklisted class becomes a random class that is not, so every path that adds a bot obeys the list
void PickAllowedBotClass(const char[] wanted, char[] buffer, int maxlen)
{
	strcopy(buffer, maxlen, wanted);

	if (!IsBotClassBlacklisted(wanted))
		return;

	char candidates[9][TF2_CLASS_MAX_NAME_LENGTH];
	int total = 0;

	for (int i = view_as<int>(TFClass_Scout); i <= view_as<int>(TFClass_Engineer); i++)
	{
		if (!IsBotClassBlacklisted(g_sRawPlayerClassNames[i]))
			strcopy(candidates[total++], TF2_CLASS_MAX_NAME_LENGTH, g_sRawPlayerClassNames[i]);
	}

	//Everything is blacklisted, which cannot be meant: the list is ignored
	if (total == 0)
		return;

	strcopy(buffer, maxlen, candidates[GetRandomInt(0, total - 1)]);
}

/* The lineup to fill RED with, or an empty string to leave it to the lineup mode

The convar wins over the map. Somebody who typed a team into the console is answering a question
the map file guessed at, and the map is a default rather than an instruction.

The map's own answer exists because the right team is not the same on every map: Mannworks is
full of deflector Heavies, which eat a Soldier's rockets and do nothing to a second Heavy, and
Coal Town is one long bottleneck full of Spies, where a Pyro is worth more than the reach */
void GetWantedTeamComposition(char[] out, int maxlen)
{
	redbots_manager_team_composition.GetString(out, maxlen);

	if (out[0] != '\0')
		return;

	strcopy(out, maxlen, g_arrMapConfig.strComposition);
}

//Does sm_redbots_manager_team_composition ask for this class anywhere in the team it names?
bool IsClassInTeamComposition(const char[] class, bool bTypedTeamOnly = false)
{
	char list[128];

	if (bTypedTeamOnly)
		redbots_manager_team_composition.GetString(list, sizeof(list));
	else
		GetWantedTeamComposition(list, sizeof(list));

	if (list[0] == '\0')
		return false;

	TFClassType wanted = TF2_GetClassIndexFromString(class);

	if (wanted == TFClass_Unknown)
		return false;

	char entries[MAXPLAYERS + 1][TF2_CLASS_MAX_NAME_LENGTH];
	int count = ExplodeString(list, ",", entries, sizeof(entries), sizeof(entries[]));

	for (int i = 0; i < count; i++)
	{
		TrimString(entries[i]);

		if (TF2_GetClassIndexFromString(entries[i]) == wanted)
			return true;
	}

	return false;
}

bool IsBotClassBlacklisted(const char[] class)
{
	/* A team somebody typed out is more specific than the blacklist, so what it asks for is never
	blacklisted. The map config's own composition is not that: it is this mod's guess at a good team
	for the map, and a guess does not get to overrule a class the server was told never to play.
	Reported from a play-test as seats set to "Let the mod pick" drawing unticked classes. */
	if (IsClassInTeamComposition(class, true))
		return false;

	char list[128]; redbots_manager_class_blacklist.GetString(list, sizeof(list));

	if (list[0] == '\0')
		return false;

	TFClassType wanted = TF2_GetClassIndexFromString(class);

	if (wanted == TFClass_Unknown)
		return false;

	char entries[9][TF2_CLASS_MAX_NAME_LENGTH];
	int count = ExplodeString(list, ",", entries, sizeof(entries), sizeof(entries[]));

	for (int i = 0; i < count; i++)
	{
		TrimString(entries[i]);

		if (TF2_GetClassIndexFromString(entries[i]) == wanted)
			return true;
	}

	return false;
}

/* Ask the game for the seats first, and settle for what it gives.

Reported as "i tried changing the number to 12 but then it stops adding bots", which left a player
with fewer bots at twelve than at six. Nothing here ever touched tf_mvm_defenders_team_size, so the
game kept refusing RED past its own number while this asked for twelve, and the bots went nowhere.

Mann vs Machine is built around six: the upgrade station, the ready panel and the scoreboard all
assume it. Whether the game accepts more is the game's answer and not ours, so this asks and then
reads back rather than assuming either way. What it reads back is the ceiling, and this convar is
clamped to it so every place that reads the number gets one the game will honour.

Failing loudly at seven beats spawning nothing at twelve. */
public void ConVarChanged_DefenderTeamSize(ConVar convar, const char[] before, const char[] after)
{
	int wanted = convar.IntValue;

	ConVar gameSize = FindConVar("tf_mvm_defenders_team_size");

	if (gameSize == null)
	{
		LogMessage("Team size: the game has no tf_mvm_defenders_team_size, so %d is taken as given", wanted);
		return;
	}

	if (gameSize.IntValue != wanted)
		gameSize.SetInt(wanted);

	int allowed = gameSize.IntValue;

	if (allowed == wanted)
		return;

	LogMessage("Team size: asked the game for %d, it allows %d. Clamping, or no bot would be added at all",
		wanted, allowed);
	PrintToChatAll("%s RED holds %d, not %d: the game refused the rest.", PLUGIN_PREFIX, allowed, wanted);

	//The hook fires again on this write, and the second pass returns at the equality above
	convar.SetInt(allowed);
}

void AddRandomDefenderBots(int amount)
{
	PrintToChatAll("%s Adding %d bot(s)...", PLUGIN_PREFIX, amount);
	
	for (int i = 1; i <= amount; i++)
		AddDefenderTFBot(1, g_sRawPlayerClassNames[GetRandomInt(1, 9)], "red", "expert");
}

void AddBotsWithPresetTeamComp(int count = 6, int teamType = 0)
{
	int total = 0;
	
	for (int i = 0; i < count; i++)
	{
		//We're done here
		if (total >= count)
			break;
		
		//We asked for more than the array size, cycle back from the beginning
		if (i >= sizeof(g_sBotTeamCompositions[]))
			i = 0;
		
		AddDefenderTFBot(1, g_sBotTeamCompositions[teamType][i], "red", "expert");
		total++;
	}
}

void SetupSniperSpotHints()
{
	if (g_arrMapConfig.adtSniperSpot.Length > 0)
	{
		for (int i = 0; i < g_arrMapConfig.adtSniperSpot.Length; i++)
		{
			float vec[3]; g_arrMapConfig.adtSniperSpot.GetArray(i, vec);
			int ent = CreateEntityByName("func_tfbot_hint");
			
			if (ent != -1)
			{
				DispatchKeyValueVector(ent, "origin", vec);
				DispatchKeyValue(ent, "team", "2");
				DispatchKeyValue(ent, "hint", "0");
				DispatchSpawn(ent);
			}
		}
	}
	else
	{
		//No custom hints specified, so we'll just override any existing ones
		int ent = -1;
		
		while ((ent = FindEntityByClassname(ent, "func_tfbot_hint")) != -1)
			DispatchKeyValue(ent, "team", "0");
		
		LogError("SetupSniperSpotHints: No hints specified by configuration, overriding other hint entities!");
	}
}

bool HavePlayersChosenBotTeam()
{
	//If someone is choosing the lineup right now, we're not ready yet
	if (GetCountOfPlayersChoosingBotClasses() > 0)
		return false;
	
	//Always ready if our team is full
	if (GetTeamClientCount(TFTeam_Red) >= redbots_manager_defender_team_size.IntValue)
		return true;
	
	/* If strictly requiring a chosen lineup, the list will only ever be made up of classes picked by a player
	If it is empty, no one chose anything yet */
	return g_adtChosenBotClasses.Length > 0;
}

void FreeChosenBotTeam(bool bAnnounce = false)
{
	g_adtChosenBotClasses.Clear();
	g_adtChosenBotSeats.Clear();
	g_bBotClassesLocked = false;
	
	if (bAnnounce)
		PrintToChatAll("%s Bot team lineup can now be changed.", PLUGIN_PREFIX);
}

void UpdateChosenBotTeamComposition(int caller = -1)
{
	//A player has already chosen their team, don't let it change
	if (g_bBotClassesLocked)
	{
		if (caller != -1)
			PrintToChat(caller, "%s Bot team lineup is locked for the next game.");
		
		return;
	}
	
	//If someone is selecting the team, don't change it
	if (GetCountOfPlayersChoosingBotClasses() > 0)
	{
		if (caller != -1)
			PrintToChat(caller, "%s Someone is currently choosing the bot team lineup.");
		
		return;
	}
	
	g_adtChosenBotClasses.Clear();
	g_adtChosenBotSeats.Clear();
	
	int newBotsToAdd = redbots_manager_defender_team_size.IntValue - GetHumanAndDefenderBotCount(TFTeam_Red);
	
	if (newBotsToAdd < 1)
		return;
	
	/* The named team is decided here, where every lineup is decided, and not where the bots are added
	The wave begins by adding this list and nothing else, so a team named in the convar that only the
	top-up timer ever read was a team that never played */
	newBotsToAdd -= CollectMissingTeamComposition(g_adtChosenBotClasses, g_adtChosenBotSeats, newBotsToAdd);
	
	//Whatever seats the named team left over are the lineup mode's to fill
	if (newBotsToAdd > 0)
		ChooseBotClassesFromLineupMode(newBotsToAdd);
	
	if (caller != -1)
		PrintToChatAll("%s %N changed the bot team lineup", PLUGIN_PREFIX, caller);
	else
		PrintToChatAll("%s Bot lineup changed", PLUGIN_PREFIX);
}

/* Name count more classes for the chosen lineup, the way the lineup mode says to */
void ChooseBotClassesFromLineupMode(int count)
{
	switch (redbots_manager_bot_lineup_mode.IntValue)
	{
		case BOT_LINEUP_MODE_RANDOM:
		{
			for (int i = 1; i <= count; i++)
				g_adtChosenBotClasses.PushString(g_sRawPlayerClassNames[GetRandomInt(TFClass_Scout, TFClass_Engineer)]);
		}
		case BOT_LINEUP_MODE_PREFERENCE, BOT_LINEUP_MODE_PREFERENCE_CHOOSE:
		{
			ArrayList adtClassPref = new ArrayList(TF2_CLASS_MAX_NAME_LENGTH);
			
			CollectPlayerBotClassPreferences(adtClassPref);
			
			if (adtClassPref.Length > 0)
			{
				//Choose the class lineup based on players' preferences
				for (int i = 1; i <= count; i++)
				{
					char class[TF2_CLASS_MAX_NAME_LENGTH]; adtClassPref.GetString(GetRandomInt(0, adtClassPref.Length - 1), class, sizeof(class));
					
					g_adtChosenBotClasses.PushString(class);
				}
			}
			else
			{
				//No prefernces, the lineup is random
				for (int i = 1; i <= count; i++)
					g_adtChosenBotClasses.PushString(g_sRawPlayerClassNames[GetRandomInt(TFClass_Scout, TFClass_Engineer)]);
			}
			
			delete adtClassPref;
		}
		default:
		{
			ThrowError("Unknown lineup mode %d", redbots_manager_bot_lineup_mode.IntValue);
		}
	}
}

/* Fill the seats the lineup asked for that are still empty, rather than the whole lineup

This added every entry and counted nobody, which is right only while RED is empty until the wave
begins, because in AUTO_BOTS the mod is what fills it. tf2-archipelago fills the team between
waves instead, so the bots shop and the engineer builds his nest before the wave rather than
arriving with it, and the two adds stacked: a six man team came up at ten after wave one started.

Counted the way the top-up timer counts, and a class already standing there spends its own entry,
so what gets added is the classes the lineup named and the team has not got. */
void AddBotsFromChosenTeamComposition()
{
	//Once we add them it's not locked anymore
	g_bBotClassesLocked = false;
	
	int seats = redbots_manager_defender_team_size.IntValue - GetHumanAndDefenderBotCount(TFTeam_Red);
	
	if (seats < 1)
	{
		if (redbots_manager_debug.BoolValue)
			PrintToServer("AddBotsFromChosenTeamComposition: RED is already full, added nobody");
		
		return;
	}
	
	//A bot that is on the team but has not spawned yet has no class, and only the seat count sees it
	int held[view_as<int>(TFClass_Engineer) + 1];
	
	for (int i = 1; i <= MaxClients; i++)
		if (IsClientInGame(i) && IsDefenderBot(i) && TF2_GetClientTeam(i) == TFTeam_Red)
			held[view_as<int>(TF2_GetPlayerClass(i))]++;
	
	char class[TF2_CLASS_MAX_NAME_LENGTH];
	int added = 0;
	
	for (int i = 0; i < g_adtChosenBotClasses.Length && added < seats; i++)
	{
		g_adtChosenBotClasses.GetString(i, class, sizeof(class));
		
		TFClassType type = TF2_GetClassIndexFromString(class);
		
		if (type != TFClass_Unknown && held[view_as<int>(type)] > 0)
		{
			held[view_as<int>(type)]--;
			continue;
		}
		
		//The seat belongs to the entry, so an entry that was skipped does not spend one
		NoteBotSeatPending(i < g_adtChosenBotSeats.Length ? g_adtChosenBotSeats.Get(i) : 0);
		
		AddDefenderTFBot(1, class, "red", "expert");
		added++;
	}
	
	if (added > 0)
		PrintToChatAll("%s Added %d bot(s).", PLUGIN_PREFIX, added);
}

eMissionDifficulty GetMissionDifficulty()
{
	int rsrc = FindEntityByClassname(MaxClients + 1, "tf_objective_resource");
	
	if (rsrc == -1)
	{
		LogError("GetMissionDifficulty: Could not find entity tf_objective_resource!");
		return MISSION_UNKNOWN;
	}
	
	char missionName[PLATFORM_MAX_PATH]; TF2_GetMvMPopfileName(rsrc, missionName, sizeof(missionName));
	
	//Remove unnecessary
	ReplaceString(missionName, sizeof(missionName), "scripts/population/", "");
	ReplaceString(missionName, sizeof(missionName), ".pop", "");
	
	eMissionDifficulty type = Config_GetMissionDifficultyFromName(missionName);
	
	//No config file specified a difficulty, search for one ourselves
	if (type == MISSION_UNKNOWN)
	{
		char mapName[PLATFORM_MAX_PATH]; GetCurrentMap(mapName, sizeof(mapName));
		
		//Searching by prefix or suffix
		if (StrEqual(missionName, mapName) || StrContains(missionName, "_norm_", false) != -1)
		{
			//If the mission name is the same as the map's name, it's typically a normal mission
			type = MISSION_NORMAL;
		}
		else if (StrContains(missionName, "_intermediate", false) != -1 || StrContains(missionName, "_int_", false) != -1)
		{
			type = MISSION_INTERMEDIATE;
		}
		else if (StrContains(missionName, "_advanced", false) != -1 || StrContains(missionName, "_adv_", false) != -1)
		{
			type = MISSION_ADVANCED;
		}
		else if (StrContains(missionName, "_expert", false) != -1 || StrContains(missionName, "_exp_", false) != -1)
		{
			type = MISSION_EXPERT;
		}
		else if (StrContains(missionName, "_night_", false) != -1)
		{
			//NOTE: No official mission actually uses this
			type = MISSION_NIGHTMARE;
		}
	}
	
	if (redbots_manager_debug.BoolValue)
		PrintToChatAll("GetMissionDifficulty: Current difficulty is %d", type);
	
	return type;
}

void Config_LoadMap()
{
	g_arrMapConfig.Reset();
	
	char mapName[PLATFORM_MAX_PATH]; GetCurrentMap(mapName, sizeof(mapName));
	char filePath[PLATFORM_MAX_PATH]; BuildPath(Path_SM, filePath, sizeof(filePath), "configs/defenderbots/map/%s.cfg", mapName);
	
	KeyValues kv = new KeyValues("MapConfig");
	
	if (!kv.ImportFromFile(filePath))
	{
		CloseHandle(kv);
		LogError("Config_LoadMap: File not found (%s)", filePath);
		return;
	}
	
	Config_LoadLocations(kv, "SniperSpot", g_arrMapConfig.adtSniperSpot);
	Config_LoadNestSpots(kv, "EngineerNest", g_arrMapConfig.adtEngineerNestLocation, g_arrMapConfig.adtEngineerNestZone);
	Config_LoadLocations(kv, "TeleporterEntrance", g_arrMapConfig.adtTeleporterEntranceLocation);
	Config_LoadLocations(kv, "TeleporterExit", g_arrMapConfig.adtTeleporterExitLocation);
	Config_LoadNestSpots(kv, "DispenserSpot", g_arrMapConfig.adtDispenserLocation, g_arrMapConfig.adtDispenserZone);
	Config_LoadLocations(kv, "NestTankOnly", g_arrMapConfig.adtNestTankOnlyLocation);
	Config_LoadLocations(kv, "NestNoTank", g_arrMapConfig.adtNestNoTankLocation);
	kv.GetString("Composition", g_arrMapConfig.strComposition, sizeof(g_arrMapConfig.strComposition), "");
	g_arrMapConfig.bMovingNests = kv.GetNum("MovingNests", 0) != 0;
	
	CloseHandle(kv);
	
	/* One line, always. Whoever authored a map file needs to know it was read, and a typo in a
	block name is otherwise silent: the block is skipped, the list stays empty, and the bots fall
	back to the nav mesh as though nobody had written anything */
	LogMessage("Config_LoadMap: %s: %d sniper, %d nest, %d nest-tank, %d nest-notank, %d dispenser, %d tele-in, %d tele-out, moving nests %d",
		mapName,
		g_arrMapConfig.adtSniperSpot.Length,
		g_arrMapConfig.adtEngineerNestLocation.Length,
		g_arrMapConfig.adtNestTankOnlyLocation.Length,
		g_arrMapConfig.adtNestNoTankLocation.Length,
		g_arrMapConfig.adtDispenserLocation.Length,
		g_arrMapConfig.adtTeleporterEntranceLocation.Length,
		g_arrMapConfig.adtTeleporterExitLocation.Length,
		g_arrMapConfig.bMovingNests);
}

/* Every "origin" under a named block, in map order

A block this map does not have leaves the list empty, which is what every caller falls back on:
the map configurations are hand written one map at a time, so most of them define some blocks and
not others */
/* Nest spots, and the zone each one covers

A map with an inside and an outside wants an engineer on each, and nothing in the score says so:
two spots that both look good get both engineers, and half the map is unheld. A zone is the
mapper saying "these are the same piece of ground", so the picker can spread across them.

A spot with no zone belongs to no group and competes normally */
void Config_LoadNestSpots(KeyValues kv, const char[] key, ArrayList locations, ArrayList zones)
{
	if (!kv.JumpToKey(key))
		return;
	
	if (kv.GotoFirstSubKey(false))
	{
		do
		{
			float vec[3]; kv.GetVector("origin", vec);
			locations.PushArray(vec);
			
			char zone[NEST_ZONE_LENGTH]; kv.GetString("zone", zone, sizeof(zone), "");
			zones.PushString(zone);
		} while (kv.GotoNextKey(false));
		
		kv.GoBack();
	}
	
	kv.GoBack();
}

void Config_LoadLocations(KeyValues kv, const char[] key, ArrayList locations)
{
	if (!kv.JumpToKey(key))
		return;
	
	if (kv.GotoFirstSubKey(false))
	{
		do
		{
			float vec[3]; kv.GetVector("origin", vec);
			locations.PushArray(vec);
		} while (kv.GotoNextKey(false));
		
		kv.GoBack();
	}
	
	kv.GoBack();
}

void Config_LoadBotNames()
{
	char filePath[PLATFORM_MAX_PATH]; BuildPath(Path_SM, filePath, sizeof(filePath), "configs/defenderbots/bot_names.txt");
	File hConfigFile = OpenFile(filePath, "r");
	char currentLine[MAX_NAME_LENGTH + 1];
	
	if (hConfigFile == null)
	{
		LogError("Config_LoadBotNames: Could not locate file %s!", filePath);
		return;
	}
	
	m_adtBotNames.Clear();
	
	while (ReadFileLine(hConfigFile, currentLine, sizeof(currentLine)))
	{
		TrimString(currentLine);
		
		if (strlen(currentLine) > 0)
			m_adtBotNames.PushString(currentLine);
	}
	
	delete hConfigFile;
}

eMissionDifficulty Config_GetMissionDifficultyFromName(char[] missionName)
{
	char filePath[PLATFORM_MAX_PATH];
	
	for (eMissionDifficulty i = MISSION_NORMAL; i < MISSION_MAX_COUNT; i++)
	{
		BuildPath(Path_SM, filePath, sizeof(filePath), g_sMissionDifficultyFilePaths[i]);
		
		File hOpenedFile = OpenFile(filePath, "r");
		
		if (hOpenedFile == null)
		{
			if (redbots_manager_debug.BoolValue)
				LogMessage("Config_GetMissionDifficultyFromName: Could not locate file %s. Skipping...", filePath);
			
			continue;
		}
		
		char currentLine[PLATFORM_MAX_PATH];
		
		while (ReadFileLine(hOpenedFile, currentLine, sizeof(currentLine)))
		{
			TrimString(currentLine);
			
			if (StrEqual(currentLine, missionName))
			{
				//Current line matches with the mission name in the file, this is it
				delete hOpenedFile;
				return i;
			}
		}
		
		delete hOpenedFile;
	}
	
	return MISSION_UNKNOWN;
}
