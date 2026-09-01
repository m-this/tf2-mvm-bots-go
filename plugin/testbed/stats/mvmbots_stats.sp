/* What the bots did with a wave, written down so it can be compared with what they did last time
 *
 * The mod is judged by play, and play is an opinion until somebody counts something. This counts
 * the few things that are not opinions: whether the wave was cleared, how long it took, how many
 * robots died and to whom, how many defenders died and to what, and what the engineers lost.
 *
 * One JSON object per line, appended, never rewritten. A wave is a line. That format is chosen
 * so a run that crashes halfway still leaves everything it measured, and so two runs can be
 * compared with nothing more than a file each.
 *
 * It is a test-bed plugin and belongs on a test server. It hooks events, writes a file, and does
 * nothing to the game.
 */

#include <sourcemod>
#include <tf2_stocks>
#include <sdkhooks>
#include <tf2utils>
#include <actions>

#pragma semicolon 1
#pragma newdecls required

#define PLUGIN_VERSION "1.0.0"

//A wave is hundreds of deaths and a line is written per wave, so the file is small on purpose
#define STATS_LINE_LENGTH	3072

/* A frame the server did not finish inside its own tick

The mod runs a path computation or two per bot per second and there are a dozen bots, so the
question of whether it fits in a tick is a real one and it is not answerable by watching. The
server runs at 66 ticks a second, which is a budget of about fifteen milliseconds a frame; twice
that is a frame somebody felt, and four times it is the watchdog's territory.

Counted rather than averaged. A mean frame time hides exactly the thing worth finding, which is
the one frame in a thousand that took a third of a second. */
#define FRAME_BUDGET_MS		15.0
#define FRAME_SLOW_MS		30.0
#define FRAME_STALL_MS		100.0

/* Every frame long enough to be worth a line of its own, with when it happened

The per-wave counts say how many frames went over a hundred milliseconds and how bad the worst was.
They do not say when, and when is the whole question: the watchdog killed three runs on the frame a
wave starts on and none in the middle of one, so a count that mixes the two answers nothing.

A quarter of a second is four times the tick and a quarter of the way to the watchdog, and frames
that long are rare enough that a line each costs nothing. */
#define FRAME_REPORT_MS		250.0

/* How far a building may sit from the nest before it is worth writing down as wrong

A dispenser is meant to be beside the sentry. One at the other end of the map feeds nothing, and
it is the shape of the bug the reach deadline used to cause, so the distance goes in the file
rather than a verdict about it. */
#define ENGINEER_LINE_LENGTH	512

/* How often the engineers are looked at while a wave runs

What each one had when the wave began says what the between-rounds time bought. It says nothing
about the eight minutes of a Bigrock wave he spent with no sentry at all, which is the shape most
"the engineer misbehaves on this map" reports actually have: not a nest he never built, but one he
could not keep.

Five seconds is a hundred and some samples in a long wave and a handful in a short one, which is
enough to tell "lost it once and rebuilt" from "never had one". */
#define ENGINEER_SAMPLE_INTERVAL	5.0

/* How often a building's health is read, which is not the same question as how often it is sampled
 *
 * Uptime is a fraction of the wave and five seconds resolves it fine. Repair is a sum of health put
 * back, and a sentry that loses two hundred and gets it back inside one interval is invisible at
 * five seconds and worth a hundred and fifty at half a second. Two engineers with three buildings
 * each is six property reads twice a second, which is nothing. */
#define REPAIR_SAMPLE_INTERVAL	0.5

/* How often an engineer's feet are read during a break, and what counts as him not having walked

The break is the whole of the setup question: what he built, in what order, and how much of the
clock he spent getting to where he built it. A quarter of a second is fine for a distance summed
over a two minute break, and it is short enough that a teleport lands in one sample rather than
being smeared over several and read as walking.

An engineer moves at 300 units a second, so a quarter second is 75 units and no more. Two hundred
is comfortably past anything he can do on foot with a stutter in the way, and comfortably under
the map-crossing jump a teleport back to the nest is. A displacement over the line is a teleport
and is kept apart from the walk, because the two are the opposite measurement: the walk is the
cost being removed and the teleport is what removes it.

A respawn is also a jump and is not a teleport, so the last position is dropped whenever he is
dead and seeded again when he comes back.

The telemetry sampler already walks positions and printBreak already sums a distance out of them,
and neither is usable here: it samples every five seconds, which is fifteen hundred units of
walking, so it cannot tell a teleport from a sprint and rounds a winding path down to the straight
line between two points five seconds apart. */
#define SETUP_SAMPLE_INTERVAL	0.25
#define SETUP_TELEPORT_UNITS	200.0

//Sentry, dispenser, entrance, exit: the same four slots the repair sampling keys on, same order
#define SETUP_SLOTS	4

public Plugin myinfo =
{
	name = "MvM Defender Bots: wave statistics",
	author = "m-this",
	description = "Records the result of every MvM wave as one line of JSON",
	version = PLUGIN_VERSION,
	url = "https://github.com/m-this/tf2-mvm-bots"
};

//The projectile whose hit was counted last, so one blast into a crowd is one hit
static int g_iLastCountedProjectile = -1;

ConVar g_cvPath;

char g_sMap[PLATFORM_MAX_PATH];

int g_iWave;
float g_flWaveStart;

/* How a defender died, which is not the same question as who killed him

"Killed by a Spy" and "killed by a knife in the back" are different failures: the first is a bot
that lost a fight, the second is a bot that never had one. Counting the causes is what tells a
hundred-Spy wave apart from a hundred-Heavy wave in the numbers */
enum
{
	DEATH_CAUSE_BULLET = 0,
	DEATH_CAUSE_EXPLOSION,
	DEATH_CAUSE_FIRE,
	DEATH_CAUSE_MELEE,
	DEATH_CAUSE_BACKSTAB,
	DEATH_CAUSE_HEADSHOT,
	DEATH_CAUSE_FALL,
	DEATH_CAUSE_OTHER,
	DEATH_CAUSE_COUNT
}

//Everything counted for the wave in progress. Reset when one begins, written when one ends
enum struct WaveCounters
{
	int robotKills;
	int giantKills;
	int tankKills;
	int sentryKills;
	int defenderDeaths;
	int backstabs;
	int busterDetonations;
	int sentriesLost;
	int dispensersLost;
	/* What the wrench actually did, which nothing has ever counted
	
	An engineer was watched standing still with the wrench out and the sentry losing health, and the
	only numbers to argue about it with were "he had a sentry up 80% of the wave" and "18 sentries
	lost". Both are consistent with an engineer who repairs perfectly and one who never swings.
	Health put back in is the difference between them. */
	int buildingRepaired;
	int buildingDamageTaken;
	/* Contribution, not just body count

	Waves cleared says whether the team held. It does not say who held it, and two builds that
	both clear five waves can be doing completely different work to get there. Damage, healing
	and what the sentry did are the numbers that separate them */
	int damageDealt;
	int damageToTanks;
	int sentryDamage;
	int healingDone;
	/* Healing split by who did it, and the scoreboard's own answer beside it
	
	One total cannot say whether a wave was held by a medic or by a dispenser, and the engineer's
	dispenser is in there: player_healed names the engineer as the healer for it. The scoreboard
	number is read separately, from the player manager the Tab screen reads, because two
	instruments that should agree are how three broken counters were caught in one day. */
	int healingByClass[view_as<int>(TFClass_Engineer) + 1];
	int healingScoreboard;
	int ubersDeployed;
	int damageByClass[view_as<int>(TFClass_Engineer) + 1];
	/* What the defenders did to themselves, which nothing has ever counted

	A soldier firing a rocket at a tank he is stood against takes the blast himself, and from the
	scoreboard that is indistinguishable from a soldier who fought well and got shot: damage up,
	kills up, deaths up. It was found by somebody watching it happen and saying so.

	Self damage separates the two without an opinion in it. A class with a column here is a class
	whose own weapon is one of the things killing it, and the number is comparable between two
	builds, which is the whole point. */
	/* The Demoman's two weapons, counted apart
	
	He is the weakest seat on the team by damage and has two entirely different ways of dealing it,
	and "demoman 1608 a wave" cannot say whether the pipes are missing or the stickies are never
	detonated. The inflictor is already in hand here, so the split costs nothing. */
	int demoPipeDamage;
	int demoStickyDamage;
	int demoMeleeDamage;
	int soldierRocketDamage;
	int soldierOtherDamage;
	/* Explosive projectiles thrown, against the ones that hurt something
	
	The Soldier and the Demoman are the two seats that fight with an arcing projectile and they are
	the two lowest scoring seats on the team. "He does a thousand a wave" cannot tell a bot that
	never shoots from a bot whose every shot goes past the robot, and those want opposite fixes. */
	int jarsThrown;
	int projectilesFired[view_as<int>(TFClass_Engineer) + 1];
	int projectilesHit[view_as<int>(TFClass_Engineer) + 1];
	int selfDamageByClass[view_as<int>(TFClass_Engineer) + 1];

	//What the team bought at the station, which nothing here has ever counted
	int upgradesBought;
	int upgradeCreditsSpent;

	/* The whole life of the wave's money
	
	dropped is what the wave paid out, picked up is what the team walked over, bonus is what the
	game adds for leaving none of it, spent is what went over the counter at the station, and in
	hand is what was still in their pockets when the wave ended. Dropped minus picked up expired on
	the floor; picked up minus spent is money nobody got round to using. Those are two different
	faults and the first version of this could tell neither. */
	int creditsDropped;
	int creditsAcquired;
	int creditsBonus;
	int creditsSpent;
	int creditsInHand;
	int selfDeathsByClass[view_as<int>(TFClass_Engineer) + 1];
	/* Who killed what, on both sides

	"Five waves cleared" hides everything worth knowing. Which defender class does the killing
	says which one is worth its seat, and what kills the defenders says what the team has no
	answer to */
	int killsByClass[view_as<int>(TFClass_Engineer) + 1];
	int giantKillsByClass[view_as<int>(TFClass_Engineer) + 1];
	int deathsToClass[view_as<int>(TFClass_Engineer) + 1];
	int deathsToSentry;
	int deathsToTank;
	//How it happened, which is a different question from who did it
	int deathsByCause[DEATH_CAUSE_COUNT];
	//What the server's own frame times did while the wave ran, which is the mod's cost to the tick
	int frames;
	int framesSlow;
	int framesStalled;
	float frameWorstMs;
	float frameTotalMs;

	void Reset()
	{
		this.frames = 0;
		this.framesSlow = 0;
		this.framesStalled = 0;
		this.frameWorstMs = 0.0;
		this.frameTotalMs = 0.0;
		this.robotKills = 0;
		this.giantKills = 0;
		this.tankKills = 0;
		this.sentryKills = 0;
		this.defenderDeaths = 0;
		this.backstabs = 0;
		this.busterDetonations = 0;
		this.sentriesLost = 0;
		this.buildingRepaired = 0;
		this.buildingDamageTaken = 0;
		this.dispensersLost = 0;
		this.upgradesBought = 0;
		this.upgradeCreditsSpent = 0;
		this.creditsDropped = 0;
		this.creditsAcquired = 0;
		this.creditsBonus = 0;
		this.creditsSpent = 0;
		this.creditsInHand = 0;
		this.damageDealt = 0;
		this.damageToTanks = 0;
		this.sentryDamage = 0;
		this.healingDone = 0;
		this.healingScoreboard = 0;

		for (int c = 0; c <= view_as<int>(TFClass_Engineer); c++)
			this.healingByClass[c] = 0;
		this.ubersDeployed = 0;
		
		this.demoPipeDamage = 0;
		this.demoStickyDamage = 0;
		this.demoMeleeDamage = 0;
		this.soldierRocketDamage = 0;
		this.soldierOtherDamage = 0;
		
		this.jarsThrown = 0;
		
		for (int i = 0; i < sizeof(this.projectilesFired); i++)
		{
			this.projectilesFired[i] = 0;
			this.projectilesHit[i] = 0;
		}
		this.deathsToSentry = 0;
		this.deathsToTank = 0;
		
		for (int i = 0; i < DEATH_CAUSE_COUNT; i++)
			this.deathsByCause[i] = 0;
		
		for (int i = 0; i < sizeof(this.damageByClass); i++)
		{
			this.damageByClass[i] = 0;
			this.selfDamageByClass[i] = 0;
			this.selfDeathsByClass[i] = 0;
			this.killsByClass[i] = 0;
			this.giantKillsByClass[i] = 0;
			this.deathsToClass[i] = 0;
		}
	}
}

WaveCounters g_Wave;

/* Two facts the bots plugin knows about itself and nothing else can work out
 *
 * Optional, because this plugin has to load on a server without the mod: asking for a native that
 * is not there is a load failure, and a statistics plugin that refuses to start is worse than one
 * that records two fields as unknown.
 */
native float Defenderbots_GetPathLength(int client);
native int Defenderbots_GetAttackTarget(int client);
native bool Defenderbots_IsPathing(int client);
native bool Defenderbots_PathFailed(int client);
native int Defenderbots_PathFailures(int client);
native int Defenderbots_RangeRepairStalls(int client);

static bool g_bHasPathNatives;
static bool g_bHasTargetNative;

public APLRes AskPluginLoad2(Handle myself, bool late, char[] error, int err_max)
{
	MarkNativeAsOptional("Defenderbots_GetPathLength");
	MarkNativeAsOptional("Defenderbots_GetAttackTarget");
	MarkNativeAsOptional("Defenderbots_IsPathing");
	MarkNativeAsOptional("Defenderbots_PathFailed");
	MarkNativeAsOptional("Defenderbots_PathFailures");
	MarkNativeAsOptional("Defenderbots_RangeRepairStalls");

	return APLRes_Success;
}

public void OnAllPluginsLoaded()
{
	g_bHasTargetNative = GetFeatureStatus(FeatureType_Native, "Defenderbots_GetAttackTarget") == FeatureStatus_Available;
	g_bHasPathNatives = GetFeatureStatus(FeatureType_Native, "Defenderbots_GetPathLength") == FeatureStatus_Available
		&& GetFeatureStatus(FeatureType_Native, "Defenderbots_IsPathing") == FeatureStatus_Available;

	if (!g_bHasPathNatives)
		LogMessage("mvmbots_stats: the bots plugin is not exporting its path state, so path_len will read -1");
}

public void OnPluginStart()
{
	g_cvPath = CreateConVar("mvmbots_stats_path", "logs/mvmbots_stats.jsonl",
		"Where to append wave results, relative to addons/sourcemod unless it starts with a slash.");

	HookEvent("mvm_begin_wave", Event_WaveBegin);
	HookEvent("mvm_wave_complete", Event_WaveComplete);
	HookEvent("mvm_wave_failed", Event_WaveFailed);
	HookEvent("mvm_mission_complete", Event_MissionComplete);
	HookEvent("player_death", Event_PlayerDeath);
	HookEvent("object_destroyed", Event_ObjectDestroyed);
	HookEvent("player_healed", Event_PlayerHealed);
	HookEvent("player_chargedeployed", Event_ChargeDeployed);
	HookEvent("player_spawn", Event_PlayerSpawn);
	HookEvent("player_builtobject", Event_BuiltObject);

	g_Wave.Reset();
}

/* The wall clock either side of a frame, which is the only honest way to ask what a frame cost

GetGameFrameTime is the tick interval the server intends, not the time it spent, so it says
fifteen milliseconds through a stall as happily as through an idle frame. The gap between one
frame starting and the next one starting is what actually elapsed. */
static float g_flLastFrame;

/* The worst frame since the last wave began, counted whether or not one is running

The per-wave numbers only ever covered frames inside a wave, and the frame that matters is the one
the wave starts on: every robot spawns and begins pathing at once, and the watchdog kills the
server there rather than in the middle of a wave. Three runs of an A/B died on it, all of them
immediately after "NextBot tickrate changed from 0 to 7", and none of them had written a wave
result yet, so nothing in the file said how close the frames had been getting.

Reported on the wave_begin line, which is the first thing written after that frame. */
static float g_flWorstFrameBetween;

/* When the worst one happened, in seconds since the map started

Without it the number is unattributable, and I have already read it wrong once: a gap of one and a
half seconds before wave one reads as a server about to be killed by the watchdog, and if it lands
twenty seconds into a map that has just finished loading it is the server starting up and nothing
to do with a wave at all. */
static float g_flWorstFrameAt;
static float g_flMapStart;

/* How much of the wave each engineer spent with something standing

Counted in samples rather than seconds because the sample is what is actually observed, and a
fraction of samples is a fraction of the wave whatever the interval is set to. */
static int g_iEngineerSamples[MAXPLAYERS + 1];
static int g_iSamplesWithSentry[MAXPLAYERS + 1];
static int g_iSamplesWithLevel3[MAXPLAYERS + 1];
static int g_iSamplesWithDispenser[MAXPLAYERS + 1];

/* Sampled off the frame hook rather than a repeating timer

A timer without TIMER_FLAG_NO_MAPCHANGE dies with the map, and one created in OnPluginStart is
created once. The first results file this wrote had a sample count of zero for every engineer on
every map, which is a measurement that says nothing and looks like a measurement. The frame hook
is already running for the frame times and cannot be killed out from under this. */
static float g_flNextEngineerSample;

static void ResetEngineerSamples()
{
	for (int i = 1; i <= MaxClients; i++)
	{
		g_iEngineerSamples[i] = 0;
		g_iSamplesWithSentry[i] = 0;
		g_iSamplesWithLevel3[i] = 0;
		g_iSamplesWithDispenser[i] = 0;
	}

	g_flNextEngineerSample = 0.0;
}

static void SampleEngineers()
{
	//Only while a wave is being played: between rounds he is building, and that is not uptime
	if (g_flWaveStart <= 0.0 || GameRules_GetRoundState() != RoundState_RoundRunning)
		return;

	if (GetGameTime() < g_flNextEngineerSample)
		return;

	g_flNextEngineerSample = GetGameTime() + ENGINEER_SAMPLE_INTERVAL;

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || TF2_GetClientTeam(i) != TFTeam_Red)
			continue;

		if (TF2_GetPlayerClass(i) != TFClass_Engineer)
			continue;

		g_iEngineerSamples[i]++;

		int sentry = FindOwnedObject(i, TFObject_Sentry);

		if (sentry != -1)
		{
			g_iSamplesWithSentry[i]++;

			if (GetEntProp(sentry, Prop_Send, "m_iUpgradeLevel") >= 3)
				g_iSamplesWithLevel3[i]++;
		}

		if (FindOwnedObject(i, TFObject_Dispenser) != -1)
			g_iSamplesWithDispenser[i]++;
	}
}

/* Health put back into a building, and health taken out of it
 *
 * There is no repair event to hook: the game fires nothing when a wrench connects, and the metal
 * an engineer spends covers building and upgrading as well. So this reads the health twice a second
 * and adds up which way it moved.
 *
 * Keyed by owner and object type rather than entity index. Indices are reused, and a fresh sentry
 * standing where a dead one stood would otherwise read as a hundred and eighty points of repair.
 *
 * Two things are skipped rather than counted. A building still going up is gaining health because
 * it is being built, not repaired, and an upgrade raises maximum health and fills it, which is a
 * jump of several hundred that has nothing to do with the wrench. Both would swamp the number they
 * are sitting in. */
static int g_iLastObjectHealth[MAXPLAYERS + 1][4];
static int g_iLastObjectLevel[MAXPLAYERS + 1][4];
static float g_flNextRepairSample;

//A teleporter entrance and a teleporter exit are the same object type and different buildings
static int FindOwnedObjectOfMode(int client, TFObjectType type, int mode)
{
	int count = TF2Util_GetPlayerObjectCount(client);

	for (int i = 0; i < count; i++)
	{
		int owned = TF2Util_GetPlayerObject(client, i);

		if (TF2_GetObjectType(owned) != type)
			continue;

		int ownedMode = HasEntProp(owned, Prop_Send, "m_iObjectMode")
			? GetEntProp(owned, Prop_Send, "m_iObjectMode") : 0;

		if (ownedMode == mode)
			return owned;
	}

	return -1;
}

static void ResetRepairSamples()
{
	for (int i = 1; i <= MaxClients; i++)
	{
		for (int t = 0; t < 4; t++)
		{
			g_iLastObjectHealth[i][t] = -1;
			g_iLastObjectLevel[i][t] = -1;
		}
	}

	g_flNextRepairSample = 0.0;
}

static void SampleRepairs()
{
	if (GetGameTime() < g_flNextRepairSample)
		return;

	g_flNextRepairSample = GetGameTime() + REPAIR_SAMPLE_INTERVAL;

	/* Four slots, because an engineer owns four buildings and only three types
	
	Both teleporters are TFObject_Teleporter, so keying on type alone puts the entrance and the
	exit in one slot and FindOwnedObject hands back whichever comes first in his object list. Two
	entities taking turns in one slot is two health readings being subtracted from each other. It
	is quiet here because both usually sit at full health, which is exactly the kind of quiet that
	stays wrong until somebody kills a teleporter. */
	static TFObjectType types[4];
	static int modes[4];
	types[0] = TFObject_Sentry;      modes[0] = 0;
	types[1] = TFObject_Dispenser;   modes[1] = 0;
	types[2] = TFObject_Teleporter;  modes[2] = 0;
	types[3] = TFObject_Teleporter;  modes[3] = 1;

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || TF2_GetClientTeam(i) != TFTeam_Red)
			continue;

		for (int t = 0; t < 4; t++)
		{
			int building = FindOwnedObjectOfMode(i, types[t], modes[t]);

			//Gone, or not finished. Either way the next reading is a seed and not a difference
			if (building == -1 || GetEntProp(building, Prop_Send, "m_bBuilding") != 0)
			{
				g_iLastObjectHealth[i][t] = -1;
				g_iLastObjectLevel[i][t] = -1;

				continue;
			}

			int health = GetEntProp(building, Prop_Data, "m_iHealth");
			int level = GetEntProp(building, Prop_Send, "m_iUpgradeLevel");

			if (g_iLastObjectHealth[i][t] >= 0 && g_iLastObjectLevel[i][t] == level)
			{
				int moved = health - g_iLastObjectHealth[i][t];

				if (moved > 0)
					g_Wave.buildingRepaired += moved;
				else
					g_Wave.buildingDamageTaken += -moved;
			}

			g_iLastObjectHealth[i][t] = health;
			g_iLastObjectLevel[i][t] = level;
		}
	}
}

/* What the break bought, per engineer: the order he built in and the ground he covered doing it

Peppy asked for the teleporter entrance to go up before the nest so the engineer does not walk back
to spawn after building it, and mvm-dh8 is that change. Neither half of it can be judged by
watching. "He seems quicker now" is the same sentence whether the order changed or not.

So the break is measured rather than described. Every building carries the second of the break it
was placed on, which is the order and the cost of the order in one number, and his feet are read
four times a second, which splits into ground walked and ground teleported over. A change that
works moves the entrance timestamp towards zero and the walk down, and leaves the sentry standing
and upgraded at the gate; a change that only feels faster moves neither.

Reset when a break begins and never when one ends, because the wave_begin event and the round state
leaving BetweenRounds are not ordered against each other, and the numbers have to survive whichever
comes first. */
static float g_flBreakStart;
static float g_flBreakSeconds;
static bool g_bInBreak;

static float g_flSetupWalked[MAXPLAYERS + 1];
static float g_flSetupTeleported[MAXPLAYERS + 1];
static int g_iSetupTeleports[MAXPLAYERS + 1];
static float g_vSetupLast[MAXPLAYERS + 1][3];
static bool g_bSetupLastValid[MAXPLAYERS + 1];
static float g_flSetupBuiltAt[MAXPLAYERS + 1][SETUP_SLOTS];
static float g_flNextSetupSample;

//Which of the four an object is, or none: a sapper and a disposable sentry are neither
static int SetupSlot(TFObjectType type, int mode)
{
	if (type == TFObject_Sentry)
		return 0;

	if (type == TFObject_Dispenser)
		return 1;

	if (type == TFObject_Teleporter)
		return mode == 0 ? 2 : 3;

	return -1;
}

static void ResetSetup()
{
	for (int i = 1; i <= MaxClients; i++)
	{
		g_flSetupWalked[i] = 0.0;
		g_flSetupTeleported[i] = 0.0;
		g_iSetupTeleports[i] = 0;
		g_bSetupLastValid[i] = false;

		for (int t = 0; t < SETUP_SLOTS; t++)
			g_flSetupBuiltAt[i][t] = -1.0;
	}

	g_flNextSetupSample = 0.0;
}

static void SampleSetup()
{
	bool between = GameRules_GetRoundState() == RoundState_BetweenRounds;

	if (between != g_bInBreak)
	{
		g_bInBreak = between;

		if (between)
		{
			ResetSetup();
			g_flBreakStart = GetGameTime();
			g_flBreakSeconds = 0.0;
		}
		else if (g_flBreakStart > 0.0)
		{
			g_flBreakSeconds = GetGameTime() - g_flBreakStart;
		}
	}

	if (!g_bInBreak || GetGameTime() < g_flNextSetupSample)
		return;

	g_flNextSetupSample = GetGameTime() + SETUP_SAMPLE_INTERVAL;

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || TF2_GetClientTeam(i) != TFTeam_Red)
			continue;

		if (TF2_GetPlayerClass(i) != TFClass_Engineer)
			continue;

		//Dead is not still, and coming back is not a teleport
		if (!IsPlayerAlive(i))
		{
			g_bSetupLastValid[i] = false;

			continue;
		}

		float at[3]; GetClientAbsOrigin(i, at);

		if (g_bSetupLastValid[i])
		{
			float moved = GetVectorDistance(g_vSetupLast[i], at);

			if (moved > SETUP_TELEPORT_UNITS)
			{
				g_iSetupTeleports[i]++;
				g_flSetupTeleported[i] += moved;
			}
			else
			{
				g_flSetupWalked[i] += moved;
			}
		}

		g_vSetupLast[i] = at;
		g_bSetupLastValid[i] = true;
	}
}

/* The second of the break a building was placed on, kept for the first placement only

Minus one is "he did not put this one down during this break", which is not the same as "it is not
there": a teleporter that survived the last wave is still standing and does not get built again.
What stands is on the engineer line, which reads the levels of all four. This says what the break
cost, so a building he did not have to build costs nothing and belongs out of the average.

He puts one down, tears it up and puts it down again on plenty of maps, and the second placement is
a different question from the order he opened with. The first is the order. */
static void Event_BuiltObject(Event event, const char[] name, bool dontBroadcast)
{
	if (!g_bInBreak || g_flBreakStart <= 0.0)
		return;

	int builder = GetClientOfUserId(event.GetInt("userid"));

	if (builder < 1 || !IsClientInGame(builder) || TF2_GetClientTeam(builder) != TFTeam_Red)
		return;

	int building = event.GetInt("index");

	if (!IsValidEntity(building))
		return;

	int mode = HasEntProp(building, Prop_Send, "m_iObjectMode")
		? GetEntProp(building, Prop_Send, "m_iObjectMode") : 0;

	int slot = SetupSlot(TF2_GetObjectType(building), mode);

	if (slot < 0 || g_flSetupBuiltAt[builder][slot] >= 0.0)
		return;

	g_flSetupBuiltAt[builder][slot] = GetGameTime() - g_flBreakStart;
}

/* How long the break has run, whether or not it has ended

Every break in the first run of this came out as zero seconds. The wave_begin event fires while the
round state is still BetweenRounds, so the state change this used to be computed on had not
happened yet when the line was written, and a walk of seventeen thousand units was reported with
nothing to divide it by. Read live instead, and kept if the break really has ended. */
static float BreakSeconds()
{
	if (g_bInBreak && g_flBreakStart > 0.0)
		return GetGameTime() - g_flBreakStart;

	return g_flBreakSeconds;
}

//One line per engineer, written where the wave begins, saying what the break before it bought
static void WriteSetup()
{
	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || TF2_GetClientTeam(i) != TFTeam_Red)
			continue;

		if (TF2_GetPlayerClass(i) != TFClass_Engineer)
			continue;

		char name[MAX_NAME_LENGTH]; GetClientName(i, name, sizeof(name));

		char line[ENGINEER_LINE_LENGTH];
		FormatEx(line, sizeof(line),
			"{\"event\":\"setup\",\"map\":\"%s\",\"wave\":%d,\"who\":\"%s\",\"break_s\":%.1f,"
			... "\"walked\":%.0f,\"teleports\":%d,\"teleported\":%.0f,"
			... "\"sentry_at_s\":%.1f,\"dispenser_at_s\":%.1f,\"entrance_at_s\":%.1f,\"exit_at_s\":%.1f}",
			g_sMap, g_iWave, name, BreakSeconds(),
			g_flSetupWalked[i], g_iSetupTeleports[i], g_flSetupTeleported[i],
			g_flSetupBuiltAt[i][0], g_flSetupBuiltAt[i][1], g_flSetupBuiltAt[i][2], g_flSetupBuiltAt[i][3]);

		WriteLine(line);
	}
}

public void OnGameFrame()
{
	float now = GetEngineTime();

	if (g_flLastFrame > 0.0)
	{
		float sinceLast = (now - g_flLastFrame) * 1000.0;

		if (sinceLast > g_flWorstFrameBetween)
		{
			g_flWorstFrameBetween = sinceLast;
			g_flWorstFrameAt = GetGameTime() - g_flMapStart;
		}

		if (sinceLast > FRAME_REPORT_MS)
			ReportStall(sinceLast);
	}

	if (g_flLastFrame > 0.0 && g_flWaveStart > 0.0)
	{
		float ms = (now - g_flLastFrame) * 1000.0;

		g_Wave.frames++;
		g_Wave.frameTotalMs += ms;

		if (ms > g_Wave.frameWorstMs)
			g_Wave.frameWorstMs = ms;

		if (ms > FRAME_STALL_MS)
			g_Wave.framesStalled++;
		else if (ms > FRAME_SLOW_MS)
			g_Wave.framesSlow++;
	}

	g_flLastFrame = now;

	SampleEngineers();
	SampleRepairs();
	SampleSetup();
	SampleTelemetry();
	CollectWaveCredits();
}

static void ReportStall(float ms)
{
	char line[ENGINEER_LINE_LENGTH];
	FormatEx(line, sizeof(line),
		"{\"event\":\"stall\",\"map\":\"%s\",\"wave\":%d,\"ms\":%.0f,\"round\":%d,\"in_wave\":%d}",
		g_sMap, g_iWave, ms, GameRules_GetRoundState(), g_flWaveStart > 0.0 ? 1 : 0);

	WriteLine(line);
}

public void OnMapStart()
{
	GetCurrentMap(g_sMap, sizeof(g_sMap));

	/* The clock starts again, because a map load is not a frame

	OnGameFrame does not run while the server is loading, so the gap between the last frame of the
	old map and the first frame of the new one is the whole load. Left running across the change,
	that gap was reported as the worst frame before wave one: 1256ms on Coaltown, which read as a
	server a quarter of a second from the watchdog and sent me looking for what was on that frame.
	Nothing was. It was the map loading, which every server does and no watchdog counts. */
	g_flLastFrame = 0.0;
	g_flWorstFrameBetween = 0.0;
	g_flWorstFrameAt = 0.0;
	g_flMapStart = GetGameTime();

	g_iWave = 0;
	g_flWaveStart = 0.0;
	g_Wave.Reset();

	//A map load is not the end of a break, and the round state on the first frame is not read yet
	g_bInBreak = false;
	g_flBreakStart = 0.0;
	g_flBreakSeconds = 0.0;
	ResetSetup();
}

static void Event_WaveBegin(Event event, const char[] name, bool dontBroadcast)
{
	//The game counts from zero and everybody else counts from one
	g_iWave = event.GetInt("wave_index") + 1;
	g_flWaveStart = GetGameTime();

	g_Wave.Reset();

	ResetEngineerSamples();
	ResetRepairSamples();
	SnapshotScoreboardHealing();

	/* The features that are on go in the file with the numbers they produced

	A results file whose settings are not recorded is a file nobody can compare with anything: two
	runs of the same mission look identical on disk and were not the same mod. The bots plugin
	publishes the set, and this only copies it. */
	char features[512];
	ConVar cvFeatures = FindConVar("sm_redbots_features_active");
	
	if (cvFeatures != null)
		cvFeatures.GetString(features, sizeof(features));
	
	char line[STATS_LINE_LENGTH];
	FormatEx(line, sizeof(line), "{\"event\":\"wave_begin\",\"map\":\"%s\",\"wave\":%d,\"red\":%d,\"bots\":%d,"
		... "\"worst_frame_before_ms\":%.0f,\"worst_frame_at_s\":%.0f,\"features\":\"%s\"}",
		g_sMap, g_iWave, CountTeam(TFTeam_Red, false), CountTeam(TFTeam_Red, true),
		g_flWorstFrameBetween, g_flWorstFrameAt, features);

	WriteLine(line);

	g_flWorstFrameBetween = 0.0;
	g_flWorstFrameAt = 0.0;

	/* Both ends of the wave, because the two questions are different

	At the beginning it says what the engineer had time to finish between rounds, which is where a
	teleporter comes from and where a nest that never reached level three shows up. At the end it
	says what survived. */
	WriteEngineers("begin");
	WriteSetup();
}

static void Event_WaveComplete(Event event, const char[] name, bool dontBroadcast)
{
	WriteWaveResult("cleared");
}

/* What every engineer had standing, and where, written as its own line per engineer

Not folded into the wave line, because there is one of these per engineer and the wave line is a
fixed shape that two runs are diffed on. What it is for is the complaint that reads "the engineer
misbehaves on this map": a nest that never got a level three sentry, a dispenser at the other end
of the map from it, a teleporter that never went up. All three are distances and levels rather
than opinions, and a map that produces them every wave is a map with bad data or bad ground. */
void WriteEngineers(const char[] when)
{
	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || TF2_GetClientTeam(i) != TFTeam_Red)
			continue;

		if (TF2_GetPlayerClass(i) != TFClass_Engineer)
			continue;

		int sentry = FindOwnedObject(i, TFObject_Sentry);
		int dispenser = FindOwnedObject(i, TFObject_Dispenser);
		int entrance = FindOwnedTeleporter(i, TFObjectMode_Entrance);
		int teleExit = FindOwnedTeleporter(i, TFObjectMode_Exit);

		float sentryAt[3];
		bool haveSentry = sentry != -1;

		if (haveSentry)
			GetEntPropVector(sentry, Prop_Send, "m_vecOrigin", sentryAt);

		char name[MAX_NAME_LENGTH]; GetClientName(i, name, sizeof(name));

		char line[ENGINEER_LINE_LENGTH];
		FormatEx(line, sizeof(line),
			"{\"event\":\"engineer\",\"map\":\"%s\",\"wave\":%d,\"when\":\"%s\",\"who\":\"%s\","
			... "\"sentry\":%d,\"dispenser\":%d,\"entrance\":%d,\"exit\":%d,"
			... "\"dispenser_from_sentry\":%.0f,\"exit_from_sentry\":%.0f,\"alive\":%d,"
			... "\"samples\":%d,\"with_sentry\":%d,\"with_level3\":%d,\"with_dispenser\":%d}",
			g_sMap, g_iWave, when, name,
			BuildingLevel(sentry), BuildingLevel(dispenser), BuildingLevel(entrance), BuildingLevel(teleExit),
			haveSentry ? RangeToBuilding(sentryAt, dispenser) : -1.0,
			haveSentry ? RangeToBuilding(sentryAt, teleExit) : -1.0,
			IsPlayerAlive(i) ? 1 : 0,
			g_iEngineerSamples[i], g_iSamplesWithSentry[i], g_iSamplesWithLevel3[i], g_iSamplesWithDispenser[i]);

		WriteLine(line);
	}
}

//Level, or zero when there is no such building, which is the answer the file wants either way
static int BuildingLevel(int building)
{
	if (building == -1)
		return 0;

	return GetEntProp(building, Prop_Send, "m_iUpgradeLevel");
}

static float RangeToBuilding(const float from[3], int building)
{
	if (building == -1)
		return -1.0;

	float at[3]; GetEntPropVector(building, Prop_Send, "m_vecOrigin", at);

	return GetVectorDistance(from, at);
}

static int FindOwnedObject(int client, TFObjectType type)
{
	int count = TF2Util_GetPlayerObjectCount(client);

	for (int i = 0; i < count; i++)
	{
		int owned = TF2Util_GetPlayerObject(client, i);

		if (TF2_GetObjectType(owned) == type)
			return owned;
	}

	return -1;
}

static int FindOwnedTeleporter(int client, TFObjectMode mode)
{
	int count = TF2Util_GetPlayerObjectCount(client);

	for (int i = 0; i < count; i++)
	{
		int owned = TF2Util_GetPlayerObject(client, i);

		if (TF2_GetObjectType(owned) == TFObject_Teleporter && TF2_GetObjectMode(owned) == mode)
			return owned;
	}

	return -1;
}

static void Event_WaveFailed(Event event, const char[] name, bool dontBroadcast)
{
	WriteWaveResult("lost");
}

static void Event_MissionComplete(Event event, const char[] name, bool dontBroadcast)
{
	char line[STATS_LINE_LENGTH];
	FormatEx(line, sizeof(line), "{\"event\":\"mission_complete\",\"map\":\"%s\",\"waves\":%d}", g_sMap, g_iWave);

	WriteLine(line);
}

#include "generated/wave_write.sp"

static void Event_PlayerDeath(Event event, const char[] name, bool dontBroadcast)
{
	int victim = GetClientOfUserId(event.GetInt("userid"));

	if (victim < 1 || !IsClientInGame(victim))
		return;

	char weapon[64]; event.GetString("weapon", weapon, sizeof(weapon));

	int attacker = GetClientOfUserId(event.GetInt("attacker"));

	if (TF2_GetClientTeam(victim) == TFTeam_Blue)
	{
		g_Wave.robotKills++;

		bool giant = HasEntProp(victim, Prop_Send, "m_bIsMiniBoss") && GetEntProp(victim, Prop_Send, "m_bIsMiniBoss");

		if (giant)
			g_Wave.giantKills++;

		//Which defender did it. A robot that killed itself leaves no attacker, and belongs to nobody
		if (attacker > 0 && IsClientInGame(attacker) && TF2_GetClientTeam(attacker) == TFTeam_Red)
		{
			int class = view_as<int>(TF2_GetPlayerClass(attacker));

			if (class >= 0 && class < sizeof(g_Wave.killsByClass))
			{
				g_Wave.killsByClass[class]++;

				if (giant)
					g_Wave.giantKillsByClass[class]++;
			}
		}

		/* A sentry buster kills itself, so the death has no attacker and the weapon is its own
		explosion. Counting them says whether the engineers are losing nests to something the
		team could have shot first */
		if (StrContains(weapon, "sentry_buster", false) != -1)
			g_Wave.busterDetonations++;

		if (StrContains(weapon, "obj_sentrygun", false) != -1 || StrContains(weapon, "sentry", false) != -1)
			g_Wave.sentryKills++;

		return;
	}

	//A defender who is his own killer, which no scoreboard has ever told apart from being shot
	if (attacker == victim && TF2_GetClientTeam(victim) == TFTeam_Red)
	{
		int selfClass = view_as<int>(TF2_GetPlayerClass(victim));
		
		if (selfClass >= 0 && selfClass < sizeof(g_Wave.selfDeathsByClass))
			g_Wave.selfDeathsByClass[selfClass]++;
	}

	if (TF2_GetClientTeam(victim) != TFTeam_Red)
		return;

	g_Wave.defenderDeaths++;

	/* A defender died while a charge sat in a medigun
	
	The stock rule fires the uber when the patient is under half health, and a giant with crits takes
	a patient from whole to dead without ever being seen at half. So the charge is still there when
	the body hits the floor, which is the shape of "the giant heavy keeps wiping everything" and is
	invisible in a wave total that only counts ubers spent. */
	for (int medic = 1; medic <= MaxClients; medic++)
	{
		if (!IsClientInGame(medic) || TF2_GetClientTeam(medic) != TFTeam_Red)
			continue;

		if (TF2_GetPlayerClass(medic) != TFClass_Medic || !IsPlayerAlive(medic))
			continue;

		int medigun = GetPlayerWeaponSlot(medic, 1);

		if (medigun == -1 || !HasEntProp(medigun, Prop_Send, "m_flChargeLevel"))
			continue;

		float charge = GetEntPropFloat(medigun, Prop_Send, "m_flChargeLevel");

		if (charge < 1.0)
			continue;

		char line[ENGINEER_LINE_LENGTH];
		char who[MAX_NAME_LENGTH]; GetClientName(victim, who, sizeof(who));

		FormatEx(line, sizeof(line),
			"{\"event\":\"uber_held\",\"map\":\"%s\",\"wave\":%d,\"died\":\"%s\",\"class\":\"%s\","
			... "\"charge\":%.2f}",
			g_sMap, g_iWave, who, ClassName(TF2_GetPlayerClass(victim)), charge);

		WriteLine(line);
	}

	/* What killed the defender

	The robot's class is the answer most of the time. A sentry is not a class and neither is a
	tank, and both kill defenders, so they get counted on their own: a team losing people to a
	robot Engineer's sentry has a different problem from one losing them to giant Heavies */
	if (StrContains(weapon, "obj_sentrygun", false) != -1 || StrContains(weapon, "sentry", false) != -1)
	{
		g_Wave.deathsToSentry++;
	}
	else if (attacker < 1 || !IsClientInGame(attacker))
	{
		//Tanks are not players, so a tank kill arrives with nobody holding the gun
		g_Wave.deathsToTank++;
	}
	else
	{
		int class = view_as<int>(TF2_GetPlayerClass(attacker));

		if (class >= 0 && class < sizeof(g_Wave.deathsToClass))
			g_Wave.deathsToClass[class]++;
	}

	//A defender who died to a knife in the back is a defender who never saw the Spy
	int customKill = event.GetInt("customkill");

	if (customKill == TF_CUSTOM_BACKSTAB || StrContains(weapon, "knife", false) != -1)
		g_Wave.backstabs++;

	g_Wave.deathsByCause[DeathCause(customKill, event.GetInt("damagebits"))]++;
}


/* Damage is counted where it lands, not where it was fired

A player_hurt event cannot tell an engineer's shotgun from his sentry: both name him as the
attacker. The damage hook can, because it carries the inflictor, and the sentry is the whole
reason an engineer is on the team */
public void OnEntityCreated(int entity, const char[] classname)
{
	//Tanks are not players, so nothing else here would ever see damage done to one
	if (StrEqual(classname, "tank_boss"))
		SDKHook(entity, SDKHook_OnTakeDamagePost, OnTankDamagePost);
	
	if (IsCountedProjectile(classname) || IsThrownJar(classname))
		RequestFrame(Frame_CountProjectile, EntIndexToEntRef(entity));
}

//Whether this is one of the arcing projectiles whose hit rate is worth knowing
static bool IsCountedProjectile(const char[] classname)
{
	return StrEqual(classname, "tf_projectile_rocket")
		|| StrEqual(classname, "tf_projectile_pipe")
		|| StrEqual(classname, "tf_projectile_pipe_remote");
}

/* Jars, counted separately because they are thrown rather than fired
 *
 * A Scout was measured holding Mad Milk for seventy five seconds at a stretch, which is three
 * times its recharge, so the question is whether the bottle is ever leaving his hand at all. The
 * weapon share cannot answer that: holding it and throwing it look the same from a slot number.
 */
static bool IsThrownJar(const char[] classname)
{
	return StrEqual(classname, "tf_projectile_jar")
		|| StrEqual(classname, "tf_projectile_jar_milk")
		|| StrEqual(classname, "tf_projectile_jar_gas")
		|| StrEqual(classname, "tf_projectile_cleaver");
}

/* Counted a frame later, because the owner is not set when the entity is created
 *
 * OnEntityCreated fires before the game attaches the projectile to whoever fired it, so asking for
 * the owner here answers nobody for every shot in the mission.
 */
public void Frame_CountProjectile(any ref)
{
	int projectile = EntRefToEntIndex(ref);
	
	if (projectile == INVALID_ENT_REFERENCE || !IsValidEntity(projectile))
		return;
	
	char classname[64]; GetEntityClassname(projectile, classname, sizeof(classname));
	
	/* A jar is counted before the owner is resolved, and on purpose
	
	Everything below needs to know whose shot it was, and a jar does not: the question is only
	whether one ever leaves anybody's hand. Counting it after the owner checks would mean a jar
	whose owner handle is not set yet reads exactly like a jar that was never thrown, and this
	file has already produced three counters that could not return anything but zero. */
	if (IsThrownJar(classname))
	{
		g_Wave.jarsThrown++;
		
		return;
	}
	
	int owner = GetEntPropEnt(projectile, Prop_Send, "m_hOwnerEntity");
	
	//A pipe belongs to its thrower; the owner handle is not always the one that is set
	if ((owner < 1 || owner > MaxClients) && HasEntProp(projectile, Prop_Send, "m_hThrower"))
		owner = GetEntPropEnt(projectile, Prop_Send, "m_hThrower");
	
	if (owner < 1 || owner > MaxClients || !IsClientInGame(owner))
		return;
	
	if (TF2_GetClientTeam(owner) != TFTeam_Red)
		return;
	
	
	int class = view_as<int>(TF2_GetPlayerClass(owner));
	
	if (class >= 0 && class < sizeof(g_Wave.projectilesFired))
		g_Wave.projectilesFired[class]++;
}

static void Event_PlayerSpawn(Event event, const char[] name, bool dontBroadcast)
{
	int client = GetClientOfUserId(event.GetInt("userid"));
	
	if (client < 1 || !IsClientInGame(client))
		return;
	
	/* Both teams, for two different counts
	
	The robots are hooked for the damage the defenders put into them. The defenders are hooked for
	the damage they put into themselves, and that hook used to sit behind the return below: it was
	attached to robots only, so it could never once fire, and the results file reported that nobody
	had ever hurt themselves while the same file counted Demomen killing themselves. A measurement
	that cannot produce a number looks exactly like a number of zero. */
	TFTeam team = TF2_GetClientTeam(client);
	
	if (team == TFTeam_Blue)
	{
		SDKUnhook(client, SDKHook_OnTakeDamagePost, OnRobotDamagePost);
		SDKHook(client, SDKHook_OnTakeDamagePost, OnRobotDamagePost);
	}
	else if (team == TFTeam_Red)
	{
		SDKUnhook(client, SDKHook_OnTakeDamagePost, OnDefenderDamagePost);
		SDKHook(client, SDKHook_OnTakeDamagePost, OnDefenderDamagePost);
	}
}

static void OnRobotDamagePost(int victim, int attacker, int inflictor, float damage, int damagetype)
{
	CountDefenderDamage(attacker, inflictor, damage, false);
}

/* A defender hurting himself, which is his own weapon and nobody else's fault
 *
 * Only when he is both ends of it: teammates cannot hurt each other in MvM, so anything a defender
 * does to a defender is a man standing too close to his own explosion.
 */
static void OnDefenderDamagePost(int victim, int attacker, int inflictor, float damage, int damagetype)
{
	if (victim != attacker || victim < 1 || victim > MaxClients || !IsClientInGame(victim))
		return;
	
	if (TF2_GetClientTeam(victim) != TFTeam_Red)
		return;
	
	int class = view_as<int>(TF2_GetPlayerClass(victim));
	
	if (class >= 0 && class < sizeof(g_Wave.selfDamageByClass))
		g_Wave.selfDamageByClass[class] += RoundToNearest(damage);

	/* Which of his own weapons did it
	
	The demoman takes a third of his own output back on Rottenburg, and the class total cannot say
	whether that is a pipe fired at something that walked into him or a sticky trap he stood in.
	Those are different faults with different fixes. */
	char what[64] = "unknown";

	if (inflictor > 0 && IsValidEntity(inflictor))
		GetEntityClassname(inflictor, what, sizeof(what));

	char line[ENGINEER_LINE_LENGTH];
	char who[MAX_NAME_LENGTH]; GetClientName(victim, who, sizeof(who));

	FormatEx(line, sizeof(line),
		"{\"event\":\"selfhurt\",\"map\":\"%s\",\"wave\":%d,\"who\":\"%s\",\"class\":\"%s\","
		... "\"weapon\":\"%s\",\"damage\":%d,\"hp\":%d}",
		g_sMap, g_iWave, who, ClassName(TF2_GetPlayerClass(victim)), what,
		RoundToNearest(damage), GetClientHealth(victim));

	WriteLine(line);
}

static void OnTankDamagePost(int victim, int attacker, int inflictor, float damage, int damagetype)
{
	CountDefenderDamage(attacker, inflictor, damage, true);
}

static void CountDefenderDamage(int attacker, int inflictor, float damage, bool tank)
{
	if (attacker < 1 || attacker > MaxClients || !IsClientInGame(attacker))
		return;
	
	if (TF2_GetClientTeam(attacker) != TFTeam_Red)
		return;
	
	int amount = RoundToNearest(damage);
	
	g_Wave.damageDealt += amount;
	
	if (tank)
		g_Wave.damageToTanks += amount;
	
	int class = view_as<int>(TF2_GetPlayerClass(attacker));
	
	if (class >= 0 && class < sizeof(g_Wave.damageByClass))
		g_Wave.damageByClass[class] += amount;
	
	if (inflictor > MaxClients && IsValidEntity(inflictor))
	{
		char classname[32]; GetEntityClassname(inflictor, classname, sizeof(classname));
		
		if (StrEqual(classname, "obj_sentrygun"))
			g_Wave.sentryDamage += amount;
		
		if (class == view_as<int>(TFClass_DemoMan))
		{
			if (StrEqual(classname, "tf_projectile_pipe_remote"))
				g_Wave.demoStickyDamage += amount;
			else if (StrEqual(classname, "tf_projectile_pipe"))
				g_Wave.demoPipeDamage += amount;
		}
		else if (class == view_as<int>(TFClass_Soldier) && StrEqual(classname, "tf_projectile_rocket"))
		{
			g_Wave.soldierRocketDamage += amount;
		}
		
		/* One projectile counts as one hit however many robots it catches
		
		Otherwise a rocket into a crowd reads as five hits and the rate goes over a hundred. */
		if (IsCountedProjectile(classname) && inflictor != g_iLastCountedProjectile)
		{
			g_iLastCountedProjectile = inflictor;
			
			if (class >= 0 && class < sizeof(g_Wave.projectilesHit))
				g_Wave.projectilesHit[class]++;
		}
	}
	else if (inflictor == attacker)
	{
		//The inflictor is the man himself, which means a hitscan weapon or a swing
		if (class == view_as<int>(TFClass_DemoMan))
			g_Wave.demoMeleeDamage += amount;
		else if (class == view_as<int>(TFClass_Soldier))
			g_Wave.soldierOtherDamage += amount;
	}
}

/* The custom kill says the special cases, the damage bits say the ordinary ones

Order matters: a backstab is also a melee hit and a headshot is also a bullet, so the specific
answer has to be asked for first or every stab is filed as "melee" */
static int DeathCause(int customKill, int damageBits)
{
	switch (customKill)
	{
		case TF_CUSTOM_BACKSTAB:
			return DEATH_CAUSE_BACKSTAB;
		
		case TF_CUSTOM_HEADSHOT, TF_CUSTOM_HEADSHOT_DECAPITATION:
			return DEATH_CAUSE_HEADSHOT;
		
		case TF_CUSTOM_BURNING, TF_CUSTOM_BURNING_FLARE:
			return DEATH_CAUSE_FIRE;
	}
	
	if (damageBits & DMG_BURN)
		return DEATH_CAUSE_FIRE;
	
	if (damageBits & DMG_BLAST)
		return DEATH_CAUSE_EXPLOSION;
	
	if (damageBits & DMG_CLUB)
		return DEATH_CAUSE_MELEE;
	
	if (damageBits & DMG_BULLET)
		return DEATH_CAUSE_BULLET;
	
	if (damageBits & DMG_FALL)
		return DEATH_CAUSE_FALL;
	
	return DEATH_CAUSE_OTHER;
}

static void Event_PlayerHealed(Event event, const char[] name, bool dontBroadcast)
{
	int healer = GetClientOfUserId(event.GetInt("healer"));
	
	if (healer < 1 || !IsClientInGame(healer) || TF2_GetClientTeam(healer) != TFTeam_Red)
		return;
	
	int amount = event.GetInt("amount");

	g_Wave.healingDone += amount;
	g_Wave.healingByClass[view_as<int>(TF2_GetPlayerClass(healer))] += amount;
}

/* The number the Tab screen shows, which is not the number the event adds up
 *
 * The scoreboard reads m_iHealing off tf_player_manager and it counts for the whole match, so a
 * wave's worth of it is the difference across the wave. Kept beside the event sum rather than
 * instead of it: they measure the same thing by different routes, and the day they disagree is
 * the day one of them is broken and worth knowing about.
 */
static int g_iHealingAtWaveStart[MAXPLAYERS + 1];

static int PlayerManager()
{
	return FindEntityByClassname(-1, "tf_player_manager");
}

static int ScoreboardHealing(int client)
{
	int manager = PlayerManager();

	if (manager == -1 || !HasEntProp(manager, Prop_Send, "m_iHealing"))
		return -1;

	return GetEntProp(manager, Prop_Send, "m_iHealing", 4, client);
}

static void SnapshotScoreboardHealing()
{
	for (int i = 1; i <= MaxClients; i++)
		g_iHealingAtWaveStart[i] = IsClientInGame(i) ? ScoreboardHealing(i) : 0;
}

/* What this wave paid out and how much of it the team actually picked up
 *
 * Money that is not walked over expires where it fell, and it is the whole upgrade budget: a team
 * that leaves a third of it on the floor plays every later wave a third under-equipped. Nothing
 * here has ever counted it, and the game keeps both halves of the number itself.
 */
//Everybody's balance at the last sample, so a fall in it can be read as money spent
static int g_iCurrencyLastSeen[MAXPLAYERS + 1];

static void CollectWaveCredits()
{
	int ent = FindEntityByClassname(MaxClients + 1, "tf_mann_vs_machine_stats");

	if (ent == -1)
		return;

	g_Wave.creditsDropped = GetEntProp(ent, Prop_Send, "m_currentWaveStats", _, 0);
	g_Wave.creditsAcquired = GetEntProp(ent, Prop_Send, "m_currentWaveStats", _, 1);
	g_Wave.creditsBonus = GetEntProp(ent, Prop_Send, "m_currentWaveStats", _, 2);

	/* What the team is holding, and what left their pockets since the last sample
	
	The station takes the money without saying how much, so it is read off the players: a fall in
	somebody's balance between two samples is money spent, and the balance itself is money nobody
	has used yet. A wave that ends with a thousand credits in hand is not the same problem as one
	that ends with a thousand on the floor. */
	int inHand = 0;

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || TF2_GetClientTeam(i) != TFTeam_Red)
			continue;

		if (!HasEntProp(i, Prop_Send, "m_nCurrency"))
			continue;

		int now = GetEntProp(i, Prop_Send, "m_nCurrency");

		inHand += now;

		if (g_iCurrencyLastSeen[i] > now)
			g_Wave.creditsSpent += g_iCurrencyLastSeen[i] - now;

		g_iCurrencyLastSeen[i] = now;
	}

	g_Wave.creditsInHand = inHand;
}


void CollectScoreboardHealing()
{
	g_Wave.healingScoreboard = 0;

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || TF2_GetClientTeam(i) != TFTeam_Red)
			continue;

		int now = ScoreboardHealing(i);

		//A player who joined mid-wave has no start to subtract, and neither has a missing manager
		if (now < 0 || g_iHealingAtWaveStart[i] < 0)
			continue;

		g_Wave.healingScoreboard += now - g_iHealingAtWaveStart[i];
	}
}

/* Every purchase at the upgrade station, counted per wave
 *
 * Whether the bots shop at all is the first question of any argument about why a wave beat them,
 * and nothing here could answer it: max health is not an upgrade Mann vs Machine sells, so a team
 * of stock-health bots says nothing either way. */
public void OnClientCommandKeyValues_Post(int client, KeyValues kv)
{
	if (client < 1 || client > MaxClients || !IsClientInGame(client))
		return;

	if (TF2_GetClientTeam(client) != TFTeam_Red)
		return;

	char name[32]; kv.GetSectionName(name, sizeof(name));

	if (!StrEqual(name, "MVM_Upgrade") || !kv.JumpToKey("upgrade"))
		return;

	int count = kv.GetNum("count", 1);
	kv.GoBack();

	if (count == 0)
		return;

	g_Wave.upgradesBought += count;
}

static void Event_ChargeDeployed(Event event, const char[] name, bool dontBroadcast)
{
	int client = GetClientOfUserId(event.GetInt("userid"));
	
	if (client > 0 && IsClientInGame(client) && TF2_GetClientTeam(client) == TFTeam_Red)
		g_Wave.ubersDeployed++;
}

static void Event_ObjectDestroyed(Event event, const char[] name, bool dontBroadcast)
{
	int owner = GetClientOfUserId(event.GetInt("userid"));

	if (owner < 1 || !IsClientInGame(owner) || TF2_GetClientTeam(owner) != TFTeam_Red)
		return;

	switch (view_as<TFObjectType>(event.GetInt("objecttype")))
	{
		case TFObject_Sentry: g_Wave.sentriesLost++;
		case TFObject_Dispenser: g_Wave.dispensersLost++;
	}
}

//How many are on a team, and how many of those are the mod's rather than people
int CountTeam(TFTeam team, bool fakeOnly)
{
	int count = 0;

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || TF2_GetClientTeam(i) != team)
			continue;

		if (fakeOnly && !IsFakeClient(i))
			continue;

		count++;
	}

	return count;
}

/* Append one line, opening and closing the file each time
 *
 * A wave is minutes apart from the next one, so holding a handle open for the length of a run
 * buys nothing and loses everything written since the last flush if the server goes down */
void WriteLine(const char[] line)
{
	char configured[PLATFORM_MAX_PATH]; g_cvPath.GetString(configured, sizeof(configured));

	char path[PLATFORM_MAX_PATH];

	if (configured[0] == '/')
		strcopy(path, sizeof(path), configured);
	else
		BuildPath(Path_SM, path, sizeof(path), "%s", configured);

	File file = OpenFile(path, "a");

	if (file == null)
	{
		LogError("mvmbots_stats: cannot open %s for writing", path);
		return;
	}

	/* WriteString rather than WriteLine, because WriteLine formats through a 2048 byte buffer
	
	The wave result is the longest line this writes and it grew past that: three of four results in
	a Bavarian Botbash run came out exactly 2047 characters long, which is not valid JSON, so every
	reader skipped them and the run looked like it had played one wave. An instrument that drops the
	measurement and says nothing is worse than no instrument. */
	file.WriteString(line, false);
	file.WriteString("\n", false);

	delete file;
}

/* Where every bot was and what it was doing, written down instead of watched
 *
 * Five separate faults this week were found by somebody playing the game and noticing: an engineer
 * stood in a house, a medic in the middle of the map, a dispenser beside a teleporter entrance. All
 * five looked identical from outside, all five took a round of guessing to name, and two of those
 * guesses were wrong. The reason is that nothing wrote down what the bots were doing, so the only
 * instrument was a person watching one of six bots at a time.
 *
 * These lines are that instrument. Facts, not verdicts: where he is, what is in his hands, what his
 * behaviour stack says, who his medigun is on. A verdict about whether that is good belongs in the
 * report, where it can be changed without another run.
 */
#define TELEMETRY_SAMPLE_INTERVAL	5.0
#define TELEMETRY_LINE_LENGTH		1280

/* How far from a building somebody counts as being served by it
 *
 * A dispenser's own heal radius, and the range a level three sentry shoots at. The dispenser
 * number is what answers "is it in a place that is any use to the team", which is the question a
 * spot walked by hand is meant to settle and nothing has ever checked.
 */
#define DISPENSER_SERVE_RANGE	450.0
#define SENTRY_SERVE_RANGE		1100.0

static float g_flNextTelemetrySample;
static char g_sTelemetryStack[512];

static void CollectTelemetryActionName(BehaviorAction action)
{
	char name[64]; action.GetName(name, sizeof(name));

	if (g_sTelemetryStack[0] != '\0')
		StrCat(g_sTelemetryStack, sizeof(g_sTelemetryStack), " < ");

	StrCat(g_sTelemetryStack, sizeof(g_sTelemetryStack), name);
}

/* Everything with a behaviour, and nothing without one
 *
 * ActionsManager.Iterator throws on a client that is not a NextBot, and a thrown native takes the
 * whole callback with it: the seat-holder mvmbots_host puts on RED is an ordinary fake client, so
 * the first sample of every frame died on it and not one telemetry line reached the file.
 *
 * The two plugins ship in the same directory and one exists to keep the other's server running, so
 * asking it what it named its seat is cheaper than guessing at a property that tells NextBots apart
 * from fake clients. */
static bool HasBehaviour(int client)
{
	static ConVar hostName;

	if (hostName == null)
		hostName = FindConVar("mvmbots_host_name");

	if (hostName == null)
		return true;

	char seat[MAX_NAME_LENGTH]; hostName.GetString(seat, sizeof(seat));
	char name[MAX_NAME_LENGTH]; GetClientName(client, name, sizeof(name));

	return !StrEqual(name, seat);
}

static void TelemetryActionStack(int client, char[] buffer, int maxlength)
{
	g_sTelemetryStack[0] = '\0';

	if (HasBehaviour(client))
		ActionsManager.Iterator(client, CollectTelemetryActionName);

	strcopy(buffer, maxlength, g_sTelemetryStack);
}

//Live enemies within range of a point that it can actually see, which is what a sentry is worth
static int EnemiesServedBy(int building, const float at[3], float range)
{
	int count = 0;

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || !IsPlayerAlive(i) || TF2_GetClientTeam(i) != TFTeam_Blue)
			continue;

		float theirs[3]; GetClientAbsOrigin(i, theirs);

		if (GetVectorDistance(at, theirs) > range)
			continue;

		theirs[2] += 40.0;

		TR_TraceRayFilter(at, theirs, MASK_SHOT, RayType_EndPoint, TraceIgnoreBuilding, building);

		if (TR_DidHit())
			continue;

		count++;
	}

	return count;
}

public bool TraceIgnoreBuilding(int entity, int mask, any data)
{
	return entity != data && (entity < 1 || entity > MaxClients);
}

//Defenders within range of a point, which is what a dispenser is worth wherever somebody put it
static int TeammatesServedBy(const float at[3], float range)
{
	int count = 0;

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || !IsPlayerAlive(i) || TF2_GetClientTeam(i) != TFTeam_Red)
			continue;

		float theirs[3]; GetClientAbsOrigin(i, theirs);

		if (GetVectorDistance(at, theirs) <= range)
			count++;
	}

	return count;
}

/* How close the nearest robot is, which is the whole of "is he standing too far forward"
 *
 * A Soldier and a Demoman fight with a projectile that arcs and splashes. Too far and it lands
 * behind a moving robot; too close and the splash is on the man who fired it. Neither is visible
 * from a damage total, and both have been guessed at.
 */
static float RangeToNearestEnemy(int client)
{
	float mine[3]; GetClientAbsOrigin(client, mine);
	float best = -1.0;
	
	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || !IsPlayerAlive(i) || TF2_GetClientTeam(i) != TFTeam_Blue)
			continue;
		
		float theirs[3]; GetClientAbsOrigin(i, theirs);
		
		float range = GetVectorDistance(mine, theirs);
		
		if (best < 0.0 || range < best)
			best = range;
	}
	
	return best;
}

/* What the bot is actually pointing at, and whether it is pressing the trigger
 *
 * "He got stuck firing at a wall" was reported from play and there was no way to check it. The
 * action stack says which behaviour is running, the position says where he is, and neither of them
 * can tell a bot shooting a robot apart from a bot shooting the wall in front of the robot.
 *
 * A trace from the eye along the view angles is the same ray the weapon fires, so what it hits is
 * what the shot hits. Once a sample, one trace per bot.
 *
 * Named for the report rather than for the entity: "world" and "robot" are the answer to the
 * question being asked, and obj_sentrygun with somebody else's name on it is not. */
//How many things are healing this one, which is what makes a giant unkillable
static int GetHealerCount(int client)
{
	return HasEntProp(client, Prop_Send, "m_nNumHealers") ? GetEntProp(client, Prop_Send, "m_nNumHealers") : 0;
}

//The game's own word for a giant, which is the same one the kill counter reads
static bool IsGiant(int client)
{
	return HasEntProp(client, Prop_Send, "m_bIsMiniBoss") && GetEntProp(client, Prop_Send, "m_bIsMiniBoss") != 0;
}

static void AimTrace(int client, char[] what, int length, float &range)
{
	float eye[3]; GetClientEyePosition(client, eye);
	float angles[3]; GetClientEyeAngles(client, angles);

	Handle trace = TR_TraceRayFilterEx(eye, angles, MASK_SHOT, RayType_Infinite, TraceIgnoreShooter, client);

	if (!TR_DidHit(trace))
	{
		strcopy(what, length, "nothing");
		range = -1.0;

		delete trace;

		return;
	}

	float end[3]; TR_GetEndPosition(end, trace);
	range = GetVectorDistance(eye, end);

	int hit = TR_GetEntityIndex(trace);

	delete trace;

	if (hit <= 0)
	{
		strcopy(what, length, "world");

		return;
	}

	if (hit <= MaxClients)
	{
		if (TF2_GetClientTeam(hit) != TFTeam_Blue)
		{
			strcopy(what, length, "teammate");

			return;
		}

		/* Which robot, not just "a robot"
		
		A wave that wipes the team is usually one robot the bots should have shot first, and a giant
		with two medics behind it is the case a player reported. "robot" cannot tell a giant medic
		from a scout, so the answer names the class, and says giant when the robot is one. */
		FormatEx(what, length, "%s%s", IsGiant(hit) ? "giant " : "", ClassName(TF2_GetPlayerClass(hit)));

		return;
	}

	char class[64]; GetEntityClassname(hit, class, sizeof(class));

	//A building of his own and a building of somebody else's are different answers
	if (StrContains(class, "obj_") == 0)
	{
		int owner = GetEntPropEnt(hit, Prop_Send, "m_hBuilder");

		Format(what, length, "%s%s", owner == client ? "own_" : "", class[4]);

		return;
	}

	strcopy(what, length, class);
}

public bool TraceIgnoreShooter(int entity, int mask, any data)
{
	return entity != data;
}

static void WriteBotTelemetry(int client, float when, float clock)
{
	float at[3]; GetClientAbsOrigin(client, at);

	char name[MAX_NAME_LENGTH]; GetClientName(client, name, sizeof(name));
	char stack[512]; TelemetryActionStack(client, stack, sizeof(stack));

	int weapon = GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon");
	char weaponClass[64] = "none";
	int slot = -1;

	if (weapon != -1)
	{
		GetEntityClassname(weapon, weaponClass, sizeof(weaponClass));
		slot = TF2Util_GetWeaponSlot(weapon);
	}

	//Who the medigun is actually on, which is the difference between a medic healing and a medic walking
	char healing[MAX_NAME_LENGTH] = "";
	int medigun = GetPlayerWeaponSlot(client, 1);

	if (medigun != -1 && HasEntProp(medigun, Prop_Send, "m_hHealingTarget"))
	{
		int patient = GetEntPropEnt(medigun, Prop_Send, "m_hHealingTarget");

		if (patient > 0 && patient <= MaxClients && IsClientInGame(patient))
			GetClientName(patient, healing, sizeof(healing));
	}

	char aim[64]; float aimRange;
	AimTrace(client, aim, sizeof(aim), aimRange);

	/* The robot this bot chose, by class, and whether anything is healing it
	
	The crosshair says where he is pointing; this says who he picked. A giant with two medics behind
	it is the case a player reported, and the question it asks is whether anybody chose a medic. */
	char picked[64] = "";
	int target = g_bHasTargetNative ? Defenderbots_GetAttackTarget(client) : -1;

	if (target > 0 && target <= MaxClients && IsClientInGame(target) && IsPlayerAlive(target))
	{
		FormatEx(picked, sizeof(picked), "%s%s%s", IsGiant(target) ? "giant " : "",
			ClassName(TF2_GetPlayerClass(target)), GetHealerCount(target) > 0 ? " (healed)" : "");
	}

	/* Whether he is walking, and whether there is anything to walk along
	
	A path length of zero with pathing true is the failure this pair exists for: the bot asked to
	go somewhere, the query came back with nothing, and it walks along nothing while every other
	field says it is travelling. That has been the cause of at least three reported faults and it
	has never been visible in a results file. */
	float pathLength = g_bHasPathNatives ? Defenderbots_GetPathLength(client) : -1.0;
	bool pathing = g_bHasPathNatives && Defenderbots_IsPathing(client);

	/* The length is not enough, and believing it was cost most of an evening
	
	A refused computation leaves the path object holding the last one that worked, so the length
	stays healthy while the bot has nowhere to go. path_failed is the flag the mod already keeps
	and has never published. */
	bool pathFailed = g_bHasPathNatives && Defenderbots_PathFailed(client);
	int pathFailures = g_bHasPathNatives ? Defenderbots_PathFailures(client) : -1;
	int repairStalls = g_bHasPathNatives ? Defenderbots_RangeRepairStalls(client) : -1;

	bool firing = (GetEntProp(client, Prop_Data, "m_nButtons") & IN_ATTACK) != 0;

	char line[TELEMETRY_LINE_LENGTH];
	FormatEx(line, sizeof(line),
		"{\"event\":\"bot\",\"map\":\"%s\",\"wave\":%d,\"t\":%.1f,\"clock\":%.1f,\"who\":\"%s\",\"class\":\"%s\","
		... "\"at\":[%.0f,%.0f,%.0f],\"hp\":%d,\"maxhp\":%d,\"weapon\":\"%s\",\"slot\":%d,"
		... "\"nearest_enemy\":%.0f,\"aim\":\"%s\",\"aim_range\":%.0f,\"firing\":%d,"
		... "\"path_len\":%.0f,\"pathing\":%d,\"path_failed\":%d,\"path_failures\":%d,\"repair_stalls\":%d,"
		... "\"healing\":\"%s\",\"picked\":\"%s\",\"action\":\"%s\"}",
		g_sMap, g_iWave, when, clock, name, ClassName(TF2_GetPlayerClass(client)),
		at[0], at[1], at[2], GetClientHealth(client), TF2Util_GetEntityMaxHealth(client),
		weaponClass, slot, RangeToNearestEnemy(client), aim, aimRange, firing ? 1 : 0,
		pathLength, pathing ? 1 : 0, pathFailed ? 1 : 0, pathFailures, repairStalls,
		healing, picked, stack);

	WriteLine(line);
}

static void WriteBuildingTelemetry(int owner, int building, float when, float clock)
{
	float at[3]; GetEntPropVector(building, Prop_Send, "m_vecOrigin", at);

	char ownerName[MAX_NAME_LENGTH]; GetClientName(owner, ownerName, sizeof(ownerName));
	char class[64]; GetEntityClassname(building, class, sizeof(class));

	float eye[3]; eye = at;
	eye[2] += 40.0;

	bool disposable = HasEntProp(building, Prop_Send, "m_bDisposableBuilding")
		&& GetEntProp(building, Prop_Send, "m_bDisposableBuilding") != 0;

	int kills = HasEntProp(building, Prop_Send, "m_iKills")
		? GetEntProp(building, Prop_Send, "m_iKills") : -1;

	char line[TELEMETRY_LINE_LENGTH];
	FormatEx(line, sizeof(line),
		"{\"event\":\"building\",\"map\":\"%s\",\"wave\":%d,\"t\":%.1f,\"clock\":%.1f,\"owner\":\"%s\","
		... "\"type\":\"%s\",\"mode\":%d,\"level\":%d,\"hp\":%d,\"maxhp\":%d,\"at\":[%.0f,%.0f,%.0f],"
		... "\"disposable\":%d,\"kills\":%d,\"enemies_seen\":%d,\"teammates_near\":%d,\"sapped\":%d,"
		... "\"shells\":%d}",
		g_sMap, g_iWave, when, clock, ownerName, class,
		HasEntProp(building, Prop_Send, "m_iObjectMode") ? GetEntProp(building, Prop_Send, "m_iObjectMode") : 0,
		GetEntProp(building, Prop_Send, "m_iUpgradeLevel"),
		GetEntProp(building, Prop_Data, "m_iHealth"), GetEntProp(building, Prop_Send, "m_iMaxHealth"),
		at[0], at[1], at[2], disposable ? 1 : 0, kills,
		EnemiesServedBy(building, eye, SENTRY_SERVE_RANGE),
		TeammatesServedBy(at, DISPENSER_SERVE_RANGE),
		GetEntProp(building, Prop_Send, "m_bHasSapper") != 0 ? 1 : 0,
		/* What a sentry has left to fire, which nothing here counted
		
		A player reported Rescue Ranger engineers refusing to reload their sentry. A bolt repairs a
		building and does not reload one, so a sentry at full health with an empty magazine is a
		state the mod can sit in forever, and no number here could see it. */
		HasEntProp(building, Prop_Send, "m_iAmmoShells") ? GetEntProp(building, Prop_Send, "m_iAmmoShells") : -1);

	WriteLine(line);
}

/* Every building in the world, counted against its builder rather than his object list
 *
 * A play-test reported an engineer with two dispensers. The per-owner samples above walk
 * TF2Util_GetPlayerObject, which is the game's own list and holds one entry per type, so a second
 * dispenser on the same builder is exactly the thing that list cannot show. This one walks the
 * entities instead and writes a line when a builder holds more than one of a type.
 *
 * Disposable sentries are skipped: a second one of those is the upgrade working.
 */
static void SampleDuplicateBuildings(float when, float clock)
{
	static const char types[][] = {"obj_dispenser", "obj_sentrygun", "obj_teleporter"};

	for (int t = 0; t < sizeof(types); t++)
	{
		int held[MAXPLAYERS + 1][2];
		int built[MAXPLAYERS + 1][2];

		for (int i = 1; i <= MaxClients; i++)
		{
			held[i][0] = 0;
			held[i][1] = 0;
			built[i][0] = 0;
			built[i][1] = 0;
		}

		int entity = -1;

		while ((entity = FindEntityByClassname(entity, types[t])) != -1)
		{
			int builder = GetEntPropEnt(entity, Prop_Send, "m_hBuilder");

			/* A building whose builder is gone, which the game's one-per-engineer limit cannot see
			
			A bot that is kicked to make room for a player, or replaced between waves, leaves its
			buildings standing with nobody holding them. The next engineer builds his own, and what
			a person sees is one engineer with two dispensers. Reported from play with a photograph
			and one engineer on the team. */
			if (builder < 1 || builder > MaxClients || !IsClientInGame(builder))
			{
				float at[3]; GetEntPropVector(entity, Prop_Send, "m_vecOrigin", at);
				char line[ENGINEER_LINE_LENGTH];

				FormatEx(line, sizeof(line),
					"{\"event\":\"orphan\",\"map\":\"%s\",\"wave\":%d,\"t\":%.1f,\"clock\":%.1f,"
					... "\"type\":\"%s\",\"at\":[%.0f,%.0f,%.0f],\"builder\":%d}",
					g_sMap, g_iWave, when, clock, types[t], at[0], at[1], at[2], builder);

				WriteLine(line);

				continue;
			}

			if (HasEntProp(entity, Prop_Send, "m_bDisposableBuilding")
				&& GetEntProp(entity, Prop_Send, "m_bDisposableBuilding") != 0)
				continue;

			/* A blueprint is not a building, and it is owned by the same engineer
			
			The first run of this counted one: an engineer walking to a spot with a dispenser on his
			toolbox holds a placing entity and the one already standing, which reads as two. It is
			also what a person watching the game sees as a second dispenser, so it is written down
			rather than skipped quietly. */
			if (GetEntProp(entity, Prop_Send, "m_bPlacing") != 0 || GetEntProp(entity, Prop_Send, "m_bCarried") != 0)
			{
				char ghostOwner[MAX_NAME_LENGTH]; GetClientName(builder, ghostOwner, sizeof(ghostOwner));
				char ghostLine[ENGINEER_LINE_LENGTH];

				FormatEx(ghostLine, sizeof(ghostLine),
					"{\"event\":\"ghost\",\"map\":\"%s\",\"wave\":%d,\"t\":%.1f,\"clock\":%.1f,"
					... "\"owner\":\"%s\",\"type\":\"%s\",\"carried\":%d}",
					g_sMap, g_iWave, when, clock, ghostOwner, types[t],
					GetEntProp(entity, Prop_Send, "m_bCarried") != 0 ? 1 : 0);

				WriteLine(ghostLine);
				continue;
			}

			int mode = HasEntProp(entity, Prop_Send, "m_iObjectMode")
				? GetEntProp(entity, Prop_Send, "m_iObjectMode") : 0;

			held[builder][mode == 1 ? 1 : 0]++;

			if (GetEntProp(entity, Prop_Send, "m_bBuilding") == 0)
				built[builder][mode == 1 ? 1 : 0]++;
		}

		for (int i = 1; i <= MaxClients; i++)
		{
			for (int mode = 0; mode < 2; mode++)
			{
				if (held[i][mode] < 2 || !IsClientInGame(i))
					continue;

				char name[MAX_NAME_LENGTH]; GetClientName(i, name, sizeof(name));
				char line[ENGINEER_LINE_LENGTH];

				/* And what the mod's own question answers for the same client
				
				held and built come from walking the entities by m_hBuilder. Everything in the mod
				asks TF2Util_GetPlayerObject instead, and nothing has ever compared the two. If the
				list is short the gates cannot see the second building, refuse nothing, and the
				engineer is free to build again: that would put the fix in GetObjectOfType rather
				than in any caller. If the two agree, the list is fine and the second was built in
				a frame where the first was not in it yet. */
				int listed = 0;
				int objects = TF2Util_GetPlayerObjectCount(i);

				for (int o = 0; o < objects; o++)
				{
					int owned = TF2Util_GetPlayerObject(i, o);

					if (owned == -1 || !IsValidEntity(owned))
						continue;

					char owning[64]; GetEntityClassname(owned, owning, sizeof(owning));

					if (StrEqual(owning, types[t]))
						listed++;
				}

				FormatEx(line, sizeof(line),
					"{\"event\":\"duplicate\",\"map\":\"%s\",\"wave\":%d,\"t\":%.1f,\"clock\":%.1f,"
					... "\"owner\":\"%s\",\"type\":\"%s\",\"mode\":%d,\"held\":%d,\"built\":%d,\"listed\":%d}",
					g_sMap, g_iWave, when, clock, name, types[t], mode, held[i][mode], built[i][mode], listed);

				WriteLine(line);
			}
		}
	}
}

/* Sampled in both round states, unlike the engineer uptime above
 *
 * Half of what has gone wrong went wrong between waves: the walk to the front, the shopping trip,
 * the toolbox still set to the last building. Sampling only while a wave runs is sampling the half
 * that was never the problem.
 */
static void SampleTelemetry()
{
	if (GetGameTime() < g_flNextTelemetrySample)
		return;

	g_flNextTelemetrySample = GetGameTime() + TELEMETRY_SAMPLE_INTERVAL;

	/* Seconds into the wave, and the server clock beside it
	
	Between waves there is no wave to be seconds into, so t is zero for every sample in the break.
	That makes a whole break's worth of samples look like one instant: reading the file back, one
	dispenser sampled fourteen times came out as fourteen dispensers, which is a bug this file was
	built to find and briefly invented instead. The clock is what tells two samples apart. */
	float when = g_flWaveStart > 0.0 ? GetGameTime() - g_flWaveStart : 0.0;
	float clock = GetGameTime();

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || !IsFakeClient(i) || TF2_GetClientTeam(i) != TFTeam_Red)
			continue;

		if (!IsPlayerAlive(i) || !HasBehaviour(i))
			continue;

		WriteBotTelemetry(i, when, clock);

		int objects = TF2Util_GetPlayerObjectCount(i);

		for (int n = 0; n < objects; n++)
			WriteBuildingTelemetry(i, TF2Util_GetPlayerObject(i, n), when, clock);
	}

	SampleDuplicateBuildings(when, clock);
}

//The name the rest of the file already writes into its keys, so the two agree without a lookup
static char[] ClassName(TFClassType class)
{
	char name[16];

	switch (class)
	{
		case TFClass_Scout:		name = "scout";
		case TFClass_Sniper:	name = "sniper";
		case TFClass_Soldier:	name = "soldier";
		case TFClass_DemoMan:	name = "demoman";
		case TFClass_Medic:		name = "medic";
		case TFClass_Heavy:		name = "heavy";
		case TFClass_Pyro:		name = "pyro";
		case TFClass_Spy:		name = "spy";
		case TFClass_Engineer:	name = "engineer";
		default:				name = "unknown";
	}

	return name;
}
