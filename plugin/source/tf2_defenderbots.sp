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
float m_flDeadRethinkTime[MAXPLAYERS + 1]; //no longer static: generated/lifecycle.sp clears it, and a file-static is invisible from an included file
int g_iBuybackNumber[MAXPLAYERS + 1];
int g_iBuyUpgradesNumber[MAXPLAYERS + 1];

static float m_flNextSnipeFireTime[MAXPLAYERS + 1];

#if defined MOD_ROLL_THE_DICE_REVAMPED
float m_flNextRollTime[MAXPLAYERS + 1]; //no longer static: generated/lifecycle.sp clears it, and a file-static is invisible from an included file
#endif

//For other players
bool g_bChoosingBotClasses[MAXPLAYERS + 1];

#if defined CHANGETEAM_RESTRICTIONS
float g_flEnableBotsCooldown[MAXPLAYERS + 1];
#endif

float m_flLastReadyInputTime[MAXPLAYERS + 1]; //no longer static: generated/lifecycle.sp clears it, and a file-static is invisible from an included file

//Config
esMapConfiguration g_arrMapConfig;
ArrayList m_adtBotNames; //no longer static: generated/mapconfig.sp fills it, and a file-static is invisible from an included file

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
#include "redbots3/generated/archipelago.sp"
#include "redbots3/generated/features.sp"
#include "redbots3/generated/loadout.sp"
#include "redbots3/generated/entity.sp"
#include "redbots3/generated/state.sp"
#include "redbots3/generated/mission.sp"
#include "redbots3/generated/weapons.sp"
#include "redbots3/generated/finders.sp"
#include "redbots3/generated/nestspot.sp"
#include "redbots3/generated/buildings.sp"
#include "redbots3/generated/scan.sp"
#include "redbots3/generated/blu_assist.sp"
#include <stocklib_officerspy/tf/tf_bot>
#include <stocklib_officerspy/tf/tf_player>
#include <stocklib_officerspy/tf/tf_obj>
#include <stocklib_officerspy/tf/tf_objective_resource>
#include <stocklib_officerspy/tf/stocklib_extra_vscript>
#include <stocklib_officerspy/econ_item_view>
#include <stocklib_officerspy/tf/tf_weaponbase>
#include <stocklib_officerspy/tf/entity_capture_flag>
#include <stocklib_officerspy/shared/util_shared>
#include <stocklib_officerspy/mathlib/vector>
#include "redbots3/generated/shared.sp"

/* The three preset lineups AddBotsWithPresetTeamComp draws from

Both are dead: nothing calls that function, which is mvm-z83.71. They stayed
behind when the rest of util.sp was ported because a two dimensional table of
names is a shape the generator has no form for, and porting a dead table to add
one would be work for something that is going to be deleted. */
char g_sBotTeamCompositions[][][] =
{
	{"scout", "soldier", "demoman", "heavyweapons", "engineer", "medic"},
	{"scout", "heavyweapons", "heavyweapons", "heavyweapons", "engineer", "sniper"},
	{"scout", "heavyweapons", "heavyweapons", "pyro", "engineer", "demoman"}
};
#include "redbots3/generated/roster_counts.sp"
#include "redbots3/generated/humans.sp"
#include "redbots3/generated/mapconfig.sp"
#include "redbots3/generated/botnames.sp"
#include "redbots3/generated/manage.sp"
#include "redbots3/generated/lineoffire.sp"
#include "redbots3/generated/nestscore.sp"
#include "redbots3/generated/nestpick.sp"
#include "redbots3/generated/nesthint.sp"
#include "redbots3/generated/buildarea.sp"
#include "redbots3/generated/nestmove.sp"
#include "redbots3/generated/bombinfo.sp"
#include "redbots3/generated/stocks.sp"
#include "redbots3/generated/angles.sp"
#include "redbots3/generated/movement.sp"
#include "redbots3/generated/reflect.sp"
#include "redbots3/generated/spawnroute.sp"
#include "redbots3/generated/sapper.sp"
#include "redbots3/generated/econitem.sp"
#include "redbots3/generated/chat.sp"
#include "redbots3/generated/actionstack.sp"
#include "redbots3/generated/weapon_tuning.sp"
#include "redbots3/generated/uber.sp"
#include "redbots3/generated/demoman_stickies.sp"
#include "redbots3/offsets.sp"
#include "redbots3/sdkcalls.sp"
#include "redbots3/generated/loadouts.sp"
#include "redbots3/generated/cosmetics.sp"
#include "redbots3/dhooks.sp"
#include "redbots3/generated/gameevents.sp"
#include "redbots3/generated/behaviourreset.sp"
#include "redbots3/generated/playerpref.sp"
#include "redbots3/generated/seating.sp"
#include "redbots3/generated/upgradereport.sp"
#include "redbots3/generated/teamchange.sp"
#include "redbots3/generated/composition.sp"
#include "redbots3/generated/settings.sp"
#include "redbots3/generated/lifecycle.sp"
#include "redbots3/generated/commands.sp"
#include "redbots3/generated/readystate.sp"
#include "redbots3/generated/teammenu.sp"
#include "redbots3/generated/prefmenu.sp"
#include "redbots3/generated/addmenu.sp"
#include "redbots3/generated/panels.sp"
#include "redbots3/tf_upgrades.sp"
#include "redbots3/generated/debug_faults.sp"
#include "redbots3/generated/threat_priority.sp"

/* Replicate the behaviour of PathFollower's PluginBot

An enum struct with methods on it, which the generator has no form for: it emits
a record's fields and nothing else. It sits here rather than in the generated
preamble for that reason alone, and it is what mvm-z83.74 would have to grow to
take. */
enum struct esPluginBot
{
	bool bPathing;
	float vecPathGoal[3];
	int iPathGoalEntity;
	
	void Reset()
	{
		this.bPathing = false;
		this.vecPathGoal = NULL_VECTOR;
		this.iPathGoalEntity = -1;
	}
	
	bool HasPathGoalVector()
	{
		return !Vector_IsZero(this.vecPathGoal);
	}
	
	bool HasPathGoalEntity()
	{
		return this.iPathGoalEntity != -1;
	}
	
	void SetPathGoalVector(const float vec[3])
	{
		//You can only set one or the other, not both
		this.iPathGoalEntity = -1;
		this.vecPathGoal = vec;
	}
	
	void SetPathGoalEntity(int entity)
	{
		this.vecPathGoal = NULL_VECTOR;
		this.iPathGoalEntity = entity;
	}
}

esPluginBot g_arrPluginBot[MAXPLAYERS + 1];

/* What nextbot_behavior.sp used to include

The file held nothing but this list and six declarations by the end, and the
declarations moved to the generated preamble. The order still matters: a
behaviour that declares a constant has to come before the file that reads it,
and behavior/engineeridle.sp declares a static with the same name as the
game-facing override in hooks. */
#include "redbots3/generated/attack.sp"
#include "redbots3/generated/markgiant.sp"
#include "redbots3/generated/collectmoney.sp"
#include "redbots3/generated/gotoupgrade.sp"
#include "redbots3/generated/attributes.sp"
#include "redbots3/generated/upgrade_rank.sp"
#include "redbots3/generated/upgrade_rules.sp"
#include "redbots3/generated/upgrade.sp"
#include "redbots3/generated/getammo.sp"
#include "redbots3/generated/movetofront.sp"
#include "redbots3/generated/gethealth.sp"
#include "redbots3/generated/engineeridle.sp"
#include "redbots3/generated/engineerbuildsentrygun.sp"
#include "redbots3/generated/engineerbuilddispenser.sp"
#include "redbots3/generated/engineerbuildteleporter.sp"
#include "redbots3/generated/engineerbuilddisposable.sp"
#include "redbots3/generated/spycheck.sp"
#include "redbots3/generated/stickytrap.sp"
#include "redbots3/generated/spylurk.sp"
#include "redbots3/generated/spysap.sp"
#include "redbots3/generated/spysapplayer.sp"
#include "redbots3/generated/medicrevive.sp"
#include "redbots3/generated/medic.sp"
#include "redbots3/generated/attackforuber.sp"
#include "redbots3/generated/evadebuster.sp"
#include "redbots3/generated/campbomb.sp"
#include "redbots3/generated/attacktank.sp"
#include "redbots3/generated/destroyteleporter.sp"
#include "redbots3/generated/guardpoint.sp"
#include "redbots3/generated/collectnearmoney.sp"
#include "redbots3/generated/botqueries.sp"
#include "redbots3/generated/readiness.sp"
#include "redbots3/generated/pathing.sp"
#include "redbots3/generated/stuckwatch.sp"
#include "redbots3/generated/mediccall.sp"
#include "redbots3/generated/spawnexit.sp"
#include "redbots3/generated/scoutjump.sp"
#include "redbots3/generated/bottle.sp"
#include "redbots3/generated/medicnudge.sp"
#include "redbots3/generated/threataudit.sp"
#include "redbots3/generated/dispatch.sp"
#include "redbots3/generated/botreset.sp"
#include "redbots3/generated/hooks.sp"
#include "redbots3/generated/statnatives.sp"
#include "redbots3/generated/botcommands.sp"
#include "redbots3/generated/aimweapons.sp"

public Plugin myinfo =
{
	name = "Defender TFBots",
	author = "Officer Spy",
	description = "TFBots that play Mann vs. Machine",
	/* This fork's version, not upstream's. The tags here restarted at v2.0.0 because the fork is
	far enough from 1.5.5 that the old number said nothing about what is running. Leaving myinfo on
	1.5.5 meant `sm plugins list` and every play-test report named a build nobody could identify. */
	version = "2.46.0",
	url = "https://github.com/OfficerSpy/TF2-MvM-Defender-TFBots"
};

public void OnPluginStart()
{
	Archipelago_Init();
	
	BuildPath(Path_SM, g_sPlayerPrefPath, PLATFORM_MAX_PATH, "data/db_botpref.txt");
	
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
	
	
	FindGameConsoleVariables();
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
		
		
		if (GameRules_GetRoundState() != RoundState_BetweenRounds)
		{
			int myWeapon = BaseCombatCharacter_GetActiveWeapon(client);
			int weaponID = myWeapon != -1 ? TF2Util_GetWeaponID(myWeapon) : -1;
			
			if (buttons & IN_ATTACK)
			{
				switch (weaponID)
				{
					case TF_WEAPON_MINIGUN:
					{
						//Don't keep spinning the minigun if it ran out of ammo
						if (!HasAmmo(myWeapon))
							buttons &= ~IN_ATTACK;
					}
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
			
		}
		
		//TODO: is this too expensive? use global per-player variable otherwise
		if (TF2_IsInUpgradeZone(client) && ActionsManager.LookupEntityActionByName(client, "DefenderUpgrade") != INVALID_ACTION)
		{
			//Because of CTFBot::AvoidPlayers, do not let ourselves move away from other players while upgrading
			vel = NULL_VECTOR;
		}
		
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

public void ConVarChanged_ManagerMode(ConVar convar, const char[] oldValue, const char[] newValue)
{
	int mode = StringToInt(newValue);
	
	//TODO: really only here for legacy reasons
	//Catch all cases of everything!
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

