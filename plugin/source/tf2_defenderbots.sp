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

float m_flNextSnipeFireTime[MAXPLAYERS + 1]; //no longer static: generated/lifecycle.sp reads and writes it, and a file-static is invisible from an included file

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
#include "redbots3/generated/dumpspot.sp"
#include "redbots3/generated/soundhook.sp"
#include "redbots3/generated/pluginstart.sp"
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

/* Every upgrade the game holds, by the index it holds it at

tf2-archipelago names an upgrade by counting "attribute" lines in
scripts/items/mvm_upgrades.txt and taking the Nth one, which assumes the game numbers them in
file order and skips nothing. That assumption has never been checked against the game.

This prints what the game itself says, so the two can be compared. If they disagree, every
purchase that mod reports is named after the wrong upgrade, and the fix is there rather than here.

An upgrade with no attribute is printed as well rather than skipped: a gap in the numbering is
exactly the thing that would break counting lines, so hiding it would hide the answer. */
//A list far longer than this is the manager not being what we think it is, not a big mission

/* Where you are standing, as a map configuration block

The nest, teleporter and sniper locations in configs/defenderbots/map are all somebody standing on
the ground they meant and writing down where that was. This prints the line to write down, so the
map data can be authored in the map instead of guessed from a compiled brush

Usage: sm_dump_spot <block> [aim]

Standing on the spot is the accurate way and stays the default. The aim mode is for noclip: it
traces the crosshair to the world and writes down what it hit, so a whole map can be marked from
above without landing on every spot. It refuses a trace that hits nothing, since a spot in the
skybox is worse than no spot */

