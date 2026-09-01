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

#define SENTRY_MAX_RANGE 1100.0

//WeaponData > Range in file tf_weapon_medigun.txt
#define WEAPON_MEDIGUN_RANGE	450.0

//CTFWeaponBuilder::InternalGetEffectBarRechargeTime
#define SAPPER_RECHARGE_TIME	15.0

//Raw value found in CBaseObject::FindBuildPointOnPlayer
#define SAPPER_PLAYER_BUILD_ON_RANGE	160.0

//ConVar cl_sidespeed
#define PLAYER_SIDESPEED	450.0

//Raw value found in CTFBotMainAction::FireWeaponAtEnemy
#define TFBOT_MELEE_ATTACK_RANGE	250.0

//PlayerLocomotion::GetStepHeight
#define TFBOT_STEP_HEIGHT	18.0

#define SNIPER_REACTION_TIME	0.5

enum //medigun_resist_types_t
{
	MEDIGUN_BULLET_RESIST = 0,
	MEDIGUN_BLAST_RESIST,
	MEDIGUN_FIRE_RESIST,
	MEDIGUN_NUM_RESISTS
}

//The game's own medigun_weapontypes_t, and its own names: 1 is the Kritzkrieg, not an uber
enum //medigun_weapontypes_t
{
	MEDIGUN_STANDARD = 0,
	MEDIGUN_CRITBOOST,
	MEDIGUN_MEGAHEAL,
	MEDIGUN_RESIST
}

enum struct BombInfo_t
{
	float vPosition[3];
	float flMinBattleFront;
	float flMaxBattleFront
}

enum
{
	TF_LOADOUT_SLOT_PRIMARY   =  0,
	TF_LOADOUT_SLOT_SECONDARY =  1,
	TF_LOADOUT_SLOT_MELEE     =  2,
	TF_LOADOUT_SLOT_UTILITY   =  3,
	TF_LOADOUT_SLOT_BUILDING  =  4,
	TF_LOADOUT_SLOT_PDA       =  5,
	TF_LOADOUT_SLOT_PDA2      =  6,
	TF_LOADOUT_SLOT_HEAD      =  7,
	TF_LOADOUT_SLOT_MISC      =  8,
	TF_LOADOUT_SLOT_ACTION    =  9,
	TF_LOADOUT_SLOT_MISC2     = 10,
	TF_LOADOUT_SLOT_TAUNT     = 11,
	TF_LOADOUT_SLOT_TAUNT2    = 12,
	TF_LOADOUT_SLOT_TAUNT3    = 13,
	TF_LOADOUT_SLOT_TAUNT4    = 14,
	TF_LOADOUT_SLOT_TAUNT5    = 15,
	TF_LOADOUT_SLOT_TAUNT6    = 16,
	TF_LOADOUT_SLOT_TAUNT7    = 17,
	TF_LOADOUT_SLOT_TAUNT8    = 18,
}

enum eMissionDifficulty
{
	MISSION_UNKNOWN = 0,
	MISSION_NORMAL,
	MISSION_INTERMEDIATE,
	MISSION_ADVANCED,
	MISSION_EXPERT,
	MISSION_NIGHTMARE,
	MISSION_MAX_COUNT
}

enum
{
	STATS_CREDITS_DROPPED = 0,
	STATS_CREDITS_ACQUIRED,
	STATS_CREDITS_BONUS,
	STATS_PLAYER_DEATHS,
	STATS_BUYBACKS
}

char g_sPlayerUseMyNameResponse[][] =
{
	"You're very funny for using my name.",
	"You totally stole my name."
};

//NOTE: Make sure this matches with the eMissionDifficulty enum size
char g_sMissionDifficultyFilePaths[][] =
{
	"",
	"configs/defenderbots/mission/mission_normal.txt",
	"configs/defenderbots/mission/mission_intermediate.txt",
	"configs/defenderbots/mission/mission_advanced.txt",
	"configs/defenderbots/mission/mission_expert.txt",
	"configs/defenderbots/mission/mission_nightmare.txt"
};

char g_sBotTeamCompositions[][][] =
{
	{"scout", "soldier", "demoman", "heavyweapons", "engineer", "medic"},
	{"scout", "heavyweapons", "heavyweapons", "heavyweapons", "engineer", "sniper"},
	{"scout", "heavyweapons", "heavyweapons", "pyro", "engineer", "demoman"}
};

char g_sRawPlayerClassNames[][] =
{
	"undefined",
	"scout",
	"sniper",
	"soldier",
	"demoman",
	"medic",
	"heavyweapons",
	"pyro",
	"spy",
	"engineer",
	"civilian",
	"",
	"random"
};

static bool TraceFilter_TFBot(int entity, int contentsMask, StringMap data)
{
	//NextBotTraceFilterIgnoreActors
	if (CBaseEntity(entity).IsCombatCharacter())
		return false;
	
	//CTraceFilterIgnoreFriendlyCombatItems
	int iPassEnt = -1;
	data.GetValue("m_pPassEnt", iPassEnt);
	
	int iCollisionGroup;
	data.GetValue("m_collisionGroup", iCollisionGroup);
	
	int iIgnoreTeam;
	data.GetValue("m_iIgnoreTeam", iIgnoreTeam);
	
	if (BaseEntity_IsCombatItem(entity))
	{
		if (BaseEntity_GetTeamNumber(entity) == iIgnoreTeam)
			return false;
		
		//m_bCallerIsProjectile is false here
	}
	
	//CTraceFilterSimple as BaseClass of CTraceFilterIgnoreFriendlyCombatItems
	if (!StandardFilterRules(entity, contentsMask))
		return false;
	
	if (iPassEnt != -1)
	{
		if (!PassServerEntityFilter(entity, iPassEnt))
			return false;
	}
	
	if (!ShouldCollide(entity, iCollisionGroup, contentsMask))
		return false;
	
	if (!TFGameRules_ShouldCollide(iCollisionGroup, BaseEntity_GetCollisionGroup(entity)))
		return false;
	
	//CTraceFilterChain checks if both filters are true
	return true;
}

//CNavArea::GetRandomPoint
void CNavArea_GetRandomPoint(CNavArea area, float buffer[3])
{
	float eLo[3], eHi[3];
	area.GetExtent(eLo, eHi);
	
	float spot[3];
	spot[0] = GetRandomFloat(eLo[0], eHi[0]);
	spot[1] = GetRandomFloat(eLo[1], eHi[1]);
	spot[2] = area.GetZ(spot[0], spot[1]);
	
	buffer = spot;
}

bool IsTFBotPlayer(int client)
{
	//TODO: change this, as it's not entirely reliable
	return IsFakeClient(client);
}

bool IsFinalWave()
{
	int rsrc = FindEntityByClassname(MaxClients + 1, "tf_objective_resource");
	
	if (rsrc != -1)
	{
		if (TF2_GetMannVsMachineWaveCount(rsrc) == TF2_GetMannVsMachineMaxWaveCount(rsrc))
			return true;
	}
	else
	{
		LogError("IsFinalWave: Could find entity tf_objective_resource!");
	}
	
	return false;
}

//Set up an entity for item creation
int EconItemCreateNoSpawn(char[] classname, int itemDefIndex, int level, int quality)
{
	int item = CreateEntityByName(classname);
	
	if (item != -1)
	{
		SetEntProp(item, Prop_Send, "m_iItemDefinitionIndex", itemDefIndex);
		SetEntProp(item, Prop_Send, "m_bInitialized", 1);
		
		//SetEntProp doesn't work here...
		static int iOffsetEntityQuality = -1;
		
		if (iOffsetEntityQuality == -1)
			iOffsetEntityQuality = FindSendPropInfo("CEconEntity", "m_iEntityQuality");
		
		static int iOffsetEntityLevel = -1;
		
		if (iOffsetEntityLevel == -1)
			iOffsetEntityLevel = FindSendPropInfo("CEconEntity", "m_iEntityLevel");
		
		SetEntData(item, iOffsetEntityQuality, quality);
		SetEntData(item, iOffsetEntityLevel, level);
		
		if (StrEqual(classname, "tf_weapon_builder", false))
		{
			/* NOTE: After the 2023-10-09 update, not setting netprop m_iObjectType
			will crash all client games (but the server will remain fine)
			I suspect the client's game code change and not setting it cause it to read garbage */
			SetEntProp(item, Prop_Send, "m_iObjectType", 3); //Set to OBJ_ATTACHMENT_SAPPER?
			
			bool isSapper = IsItemDefIndexSapper(itemDefIndex);
			
			if (isSapper)
				SetEntProp(item, Prop_Data, "m_iSubType", 3);
			
			SetEntProp(item, Prop_Send, "m_aBuildableObjectTypes", isSapper ? 0 : 1, _, 0); //OBJ_DISPENSER
			SetEntProp(item, Prop_Send, "m_aBuildableObjectTypes", isSapper ? 0 : 1, _, 1); //OBJ_TELEPORTER
			SetEntProp(item, Prop_Send, "m_aBuildableObjectTypes", isSapper ? 0 : 1, _, 2); //OBJ_SENTRYGUN
			SetEntProp(item, Prop_Send, "m_aBuildableObjectTypes", isSapper ? 1 : 0, _, 3); //OBJ_ATTACHMENT_SAPPER
		}
		else if (StrEqual(classname, "tf_weapon_sapper", false))
		{
			SetEntProp(item, Prop_Send, "m_iObjectType", 3);
			SetEntProp(item, Prop_Data, "m_iSubType", 3);
			SetEntProp(item, Prop_Send, "m_aBuildableObjectTypes", 0, _, 0);
			SetEntProp(item, Prop_Send, "m_aBuildableObjectTypes", 0, _, 1);
			SetEntProp(item, Prop_Send, "m_aBuildableObjectTypes", 0, _, 2);
			SetEntProp(item, Prop_Send, "m_aBuildableObjectTypes", 1, _, 3);
		}
	}
	else
	{
		LogError("EconItemCreateNoSpawn: Failed to create entity.");
	}
	
	return item;
}

//Call this when you're ready to spawn it
void EconItemSpawnGiveTo(int item, int client)
{
	DispatchSpawn(item);
	
	if (TF2Util_IsEntityWearable(item))
	{
		TF2Util_EquipPlayerWearable(client, item);
	}
	else
	{
		EquipPlayerWeapon(client, item);
	}
	
	//NOTE: bot items are always visible in PvE, so m_bValidatedAttachedEntity does not need setting
}

int GiveItemToPlayer(int client, char[] classname, int itemDefIndex, int level, int quality)
{
	int item = EconItemCreateNoSpawn(classname, itemDefIndex, level, quality);
	
	if (item != -1)
	{
		EconItemView_SetItemID(item, GetRandomInt(1, 2048));
		EconItemSpawnGiveTo(item, client);
	}
	
	return item;
}

bool EquipWeaponSlot(int client, int slot)
{
	int weapon = GetPlayerWeaponSlot(client, slot);
	
	if (weapon != -1)
		return TF2Util_SetPlayerActiveWeapon(client, weapon);
	
	return false;
}

float GetTimeSinceWeaponFired(int client)
{
	int iWeapon = BaseCombatCharacter_GetActiveWeapon(client);
	
	if (iWeapon == -1)
		return 9999.0;
		
	float flLastFireTime = GetEntPropFloat(iWeapon, Prop_Send, "m_flLastFireTime");
	
	if (flLastFireTime <= 0.0)
		return 9999.0;
		
	return GetGameTime() - flLastFireTime;
}

//CWeaponMedigun::GetMedigunType
int GetMedigunType(int weapon)
{
	return TF2Attrib_HookValueInt(0, "set_weapon_mode", weapon);
}

int GetResistType(int weapon)
{
	return GetEntProp(weapon, Prop_Send, "m_nChargeResistType");
}

bool HasSniperRifle(int client)
{
	int iWeapon = GetPlayerWeaponSlot(client, TFWeaponSlot_Primary);
	
	if (iWeapon == -1)
		return false;
	
	return WeaponID_IsSniperRifle(TF2Util_GetWeaponID(iWeapon));
}

//More than an engineer is supposed to own, which is the point: he is not supposed to and he does
#define MAX_PLAYER_OBJECTS	8

/* Why a build ended without a building, said out loud
 *
 * A nest that is standing for two fifths of a wave is the engineer's whole problem, and it was
 * invisible: every build action has half a dozen ways to end and none of them left a trace, so
 * "he never built one" and "he built three and lost three" and "he gave up after twelve seconds"
 * all looked the same from a results file.
 *
 * Printed rather than counted, because the interesting thing is the sequence: which reason, in
 * which order, at what point in the wave. */
stock void LogBuildFailure(int actor, const char[] what, const char[] why)
{
	if (actor < 1 || actor > MaxClients || !IsClientInGame(actor))
		return;
	
	PrintToServer("[defenderbots] %s failed for %N at %.1f: %s", what, actor, GetGameTime(), why);

	//The console is a stream nobody can count per run; the log is a file with the run in it
	LogMessage("Build: %s for %N at %.1f: %s", what, actor, GetGameTime(), why);
}

/* Whether the toolbox in his hands is set to build the thing this action came here to build

Every build action used to ask only whether he was holding the toolbox at all, and the toolbox
remembers what it was last told to make. So an engineer walking from one build straight into the
next never re-issued the command, and pressed fire on a toolbox still set to the last job.

Measured on Coaltown: he finishes the dispenser at his nest, walks to the spawn to put down the
teleporter entrance, and the entrance never happens because the toolbox is still set to dispenser.
What goes down at the spawn is a second dispenser, which is both the "dispenser right beside the
teleporter entrance" and the "two dispensers for one engineer" from play. */
stock bool IsBuilderSetTo(int client, TFObjectType type, TFObjectMode mode = TFObjectMode_None)
{
	int weapon = BaseCombatCharacter_GetActiveWeapon(client);
	
	if (weapon < 1 || TF2Util_GetWeaponID(weapon) != TF_WEAPON_BUILDER)
		return false;
	
	if (GetEntProp(weapon, Prop_Send, "m_iObjectType") != view_as<int>(type))
		return false;
	
	//Only the teleporter has two of them, and putting an entrance down for an exit is the same bug
	if (type == TFObject_Teleporter && GetEntProp(weapon, Prop_Send, "m_iObjectMode") != view_as<int>(mode))
		return false;
	
	return true;
}

/* Every building of the type, not the first one found

An engineer is not meant to be able to hold two dispensers, and one was measured holding two on
Coaltown: the working one at his nest, and a second at the spawn a teleporter's width from his
entrance. Taking down "the" dispenser between waves took down whichever came first in his object
list, so the other one outlived it, and then outlived every wave after that. The nest was rebuilt
each break and the stray never was. Reported as two dispensers for one engineer.

Collected before any of them is detonated, because detonating edits the list being walked. */
/* How many buildings this player owns, and none for a player who has left

TF2Util_GetPlayerObjectCount throws on a client that is not in game, and a thrown native takes the
whole callback with it. An action's OnEnd is the one place that reliably asks about a bot after he
has gone: the seat refill kicks bots between waves, which ends their actions, and k-kaneta's log of
2026-08-27 has the trace twice on Mannworks with the engineer's sentry action named in it.

Zero rather than a refusal, because every caller loops over the answer and a player with no
buildings is the truth about a player who is not there. */
stock int PlayerObjectCount(int client)
{
	if (client <= 0 || client > MaxClients || !IsClientInGame(client))
		return 0;

	return TF2Util_GetPlayerObjectCount(client);
}

void DetonateObjectOfType(int client, TFObjectType iType, TFObjectMode iMode = TFObjectMode_None, bool bIgnoreSapperState = false)
{
	int found[MAX_PLAYER_OBJECTS];
	int count = 0;
	
	int iNumObjects = PlayerObjectCount(client);
	
	for (int i = 0; i < iNumObjects && count < MAX_PLAYER_OBJECTS; i++)
	{
		int iObj = TF2Util_GetPlayerObject(client, i);
		
		if (TF2_GetObjectType(iObj) != iType)
			continue;
		
		if (iType == TFObject_Teleporter && TF2_GetObjectMode(iObj) != iMode)
			continue;
		
		if (TF2_IsDisposableBuilding(iObj))
			continue;
		
		if (!bIgnoreSapperState && (TF2_HasSapper(iObj) || TF2_IsPlasmaDisabled(iObj)))
			continue;
		
		found[count++] = EntIndexToEntRef(iObj);
	}
	
	for (int i = 0; i < count; i++)
	{
		int iObj = EntRefToEntIndex(found[i]);
		
		if (iObj == INVALID_ENT_REFERENCE || !IsValidEntity(iObj))
			continue;
		
		Event hEvent = CreateEvent("object_removed");
		
		if (hEvent)
		{
			hEvent.SetInt("userid", GetClientUserId(client));
			hEvent.SetInt("objecttype", iType);
			hEvent.SetInt("index", iObj);
			hEvent.Fire();
		}
		
		TF2_DetonateObject(iObj);
	}
}

/* A building of this type he owns, counting the one in his hands
 *
 * The game takes a building out of the player's object list the moment he picks it up, so every
 * question the mod asks about what an engineer has answered no while he was carrying it. What
 * follows from that is a second one: the dispenser gate sees none, sends him to build, and when he
 * finally puts the carried one down there are two dispensers and one engineer. Reported from play
 * with a photograph.
 *
 * The carried one is his by any reading of the question, so it is counted here rather than at each
 * of the twenty call sites that ask. */
int HasObjectOfType(int client, TFObjectType iObjectType, TFObjectMode iObjectMode = TFObjectMode_None)
{
	int standing = GetObjectOfType(client, iObjectType, iObjectMode);

	if (standing != INVALID_ENT_REFERENCE)
		return standing;

	int carried = TF2_GetCarriedObject(client);

	if (carried == -1 || !IsValidEntity(carried))
		return INVALID_ENT_REFERENCE;

	if (TF2_GetObjectType(carried) != iObjectType)
		return INVALID_ENT_REFERENCE;

	if (iObjectType == TFObject_Teleporter && TF2_GetObjectMode(carried) != iObjectMode)
		return INVALID_ENT_REFERENCE;

	return carried;
}

int GetObjectOfType(int client, TFObjectType iObjectType, TFObjectMode iObjectMode = TFObjectMode_None)
{
	int iNumObjects = PlayerObjectCount(client);
	
	for (int i = 0; i < iNumObjects; i++)
	{
		int iObj = TF2Util_GetPlayerObject(client, i);
		
		if (TF2_GetObjectType(iObj) != iObjectType)
			continue;
		
		if (iObjectType == TFObject_Teleporter && TF2_GetObjectMode(iObj) != iObjectMode)
			continue;
		
		if (TF2_IsDisposableBuilding(iObj))
			continue;
		
		return iObj;
	}
	
	return -1;
}

bool IsSentryBusterRobot(int client)
{
	if (IsTFBotPlayer(client))
		return GetTFBotMission(client) == CTFBot_MISSION_DESTROY_SENTRIES;
	
	char model[PLATFORM_MAX_PATH]; GetClientModel(client, model, PLATFORM_MAX_PATH);
	
	return StrEqual(model, "models/bots/demo/bot_sentry_buster.mdl");
}

//What the explosion reaches. Valve's own is smaller, and a bot that stops running early is dead
#define BUSTER_BLAST_RANGE	400.0

//How close a live buster has to be before a bot drops what it is doing and runs
#define BUSTER_FLEE_RANGE	700.0

/* How far away a buster has to be for the engineer to still have time to move the sentry

A buster walks faster than an engineer carries. Further out than this and the engineer would put
the sentry down and pick it up again for every robot that walks past the nest */
#define BUSTER_HAUL_RANGE	1800.0

/* The nearest live sentry buster to a point, or -1 for none

A buster is a mission robot, so there is at most one of them worth caring about at a time, but
nothing in the game promises that: the loop reads them all and keeps the closest.

Busters still in the spawn room are skipped. One walks the whole length of the map to reach a
sentry, and a team that starts running when it leaves the door spends the wave running */
/* Where to actually build inside a nest area

The area centre is not the spot. A nav area on a ledge runs back from the edge,
so its centre is a couple of metres behind the drop and a sentry there cannot see
what is underneath it. Reported on Decoy and Mannhattan: engineers building too
far back from the high ground.

When the nest came out of a map config, the authored origin is the answer,
because somebody stood on it. Anything else falls back to the centre */
#define NEST_SPOT_MATCH_RANGE	400.0

void NestBuildPosition(CNavArea area, float out[3])
{
	//Before the GetCenter, not after it: reading the centre of a null area reads through a null
	if (area == NULL_AREA)
	{
		out[0] = 0.0; out[1] = 0.0; out[2] = 0.0;
		return;
	}

	area.GetCenter(out);

	float best = NEST_SPOT_MATCH_RANGE;

	NestSpotFromList(g_arrMapConfig.adtEngineerNestLocation, out, best);
	NestSpotFromList(g_arrMapConfig.adtNestTankOnlyLocation, out, best);
	NestSpotFromList(g_arrMapConfig.adtNestNoTankLocation, out, best);
}

/* Where a man stands to put a building on a spot, on the side the attempt asks for

A building goes down in front of him and never under him, so the place to stand is a build's
reach short of the spot with the spot in front of him. Attempt zero is the side he is coming
from, which costs him no walking at all; each one after it is a step round the spot, so a spot
with a wall on one side is reached from another.

Shared because the dispenser, the teleporter and the sentry all need it, and they all learned it
separately before anybody wrote it down once. */
/* How far off the ring point the nav mesh may be, and how much height still counts as beside it

A spot on raised ground has ninety units of thin air around it, and the height of the ring point
used to be the height of the spot whatever was underneath. Coaltown's right-hand building is where
that showed: the engineer paths at a coordinate hanging over the floor below, the nav mesh walks
him to the ground in front of the building instead, and the arrival test never comes true. He
stands there holding the toolbox until the give-up clock runs out. Reported from play.

So the point is put wherever the nav mesh says the ground is, and refused when that ground is a
storey off the spot: standing under a ledge is not standing beside it, and the next side round is
a better answer than a building placed from down there. */
#define BUILD_STAND_SEARCH	120.0
#define BUILD_STAND_STOREY	100.0

bool BuildStandPoint(const float spot[3], const float from[3], int attempt, int attempts, float reach, float stand[3])
{
	float away[3]; SubtractVectors(from, spot, away);
	
	away[2] = 0.0;
	
	//He is standing on it, so any side will do to start from
	if (NormalizeVector(away, away) < 1.0)
	{
		away[0] = 1.0;
		away[1] = 0.0;
	}
	
	float yaw = ArcTangent2(away[1], away[0]) + DegToRad(360.0 / float(attempts) * float(attempt));
	
	stand[0] = spot[0] + Cosine(yaw) * reach;
	stand[1] = spot[1] + Sine(yaw) * reach;
	stand[2] = spot[2];
	
	CNavArea area = TheNavMesh.GetNearestNavArea(stand, false, BUILD_STAND_SEARCH, false, true, TEAM_ANY);
	
	if (area == NULL_AREA)
		return false;
	
	float ground[3]; area.GetClosestPointOnArea(stand, ground);
	
	if (FloatAbs(ground[2] - spot[2]) > BUILD_STAND_STOREY)
		return false;
	
	stand = ground;
	
	return true;
}

/* The RED spawn nearest the engineer, for a map that names no teleporter entrance

Which is most of them: Decoy names one exit and no entrance, so the engineers never built a
teleporter at all, on any map, however long the players waited between waves. */
bool NearestSpawnPoint(int actor, float spawn[3])
{
	int point = -1;
	int nearest = -1;
	float nearestRange = -1.0;
	
	while ((point = FindEntityByClassname(point, "info_player_teamspawn")) != -1)
	{
		if (GetEntProp(point, Prop_Data, "m_iTeamNum") != view_as<int>(TFTeam_Red))
			continue;
		
		float origin[3]; GetEntPropVector(point, Prop_Data, "m_vecAbsOrigin", origin);
		
		float range = GetVectorDistance(GetAbsOrigin(actor), origin);
		
		if (nearestRange < 0.0 || range < nearestRange)
		{
			nearestRange = range;
			nearest = point;
		}
	}
	
	if (nearest == -1)
		return false;
	
	GetEntPropVector(nearest, Prop_Data, "m_vecAbsOrigin", spawn);
	
	return true;
}

/* Every point on the way out of spawn an attempt might use, and where to stand to build on each

The old answer was the spawn point plus the direction to the nest times a distance, and a
direction is not a route: on a map with a wall between the two, the first attempt is inside the
wall and every attempt after it is further inside. Nothing was ever placed and nothing could be.

So it reads the nav mesh's own route instead, computed from the engineer to the spawn point and
measured back from the spawn end. The first point is a little way out of the door, and each one
after it is a step further along the way a player actually walks.

All of them at once, and not one per attempt, and that is the part that was wrong when this was
first written. The route is computed from wherever the engineer is standing. Once he has walked
out to the first point, the route left between him and spawn is shorter than the second point is
from spawn, so the second ask fails and he gives up having tried exactly one place. Sampling here,
while he is still at his nest, reads the route from the far end of the whole journey, and the
world coordinates it hands back stay true however far he walks afterwards.

Returns how many points the route was long enough for, which is none when there is no route at all
and none when the engineer is already inside the spawn room. */
int SpawnRoutePoints(int actor, const float spawn[3], float first, float step, float reach,
	float spots[][3], float stands[][3], int pointsMax)
{
	CBaseCombatCharacter(actor).UpdateLastKnownArea();
	
	PathFollower route = PathFollower(_, Path_FilterIgnoreActors, Path_FilterOnlyActors);
	
	int found = 0;
	
	if (route.ComputeToPos(CBaseNPC_GetNextBotOfEntity(actor), spawn))
	{
		float length = route.GetLength();
		
		for (int i = 0; i < pointsMax; i++)
		{
			float fromSpawn = first + step * float(i);
			
			//The route runs out, and a point past the far end of it is not on the way out of spawn
			if (length <= fromSpawn + reach)
				break;
			
			route.GetPosition(length - fromSpawn, spots[i]);
			route.GetPosition(length - fromSpawn - reach, stands[i]);
			
			found++;
		}
	}
	
	route.Destroy();
	
	return found;
}

/* The zone the map gave the nest this engineer is holding, empty when it named none

A zone is what lets a map say "this dispenser belongs to that nest" instead of leaving it to
whichever happens to be nearest. Coaltown needed it: the ground behind the wall on the right is
eight hundred units from the nest it serves and two hundred from a different one, so nearest is
the wrong answer and no distance rule fixes that. */
void NestZoneOf(CNavArea area, char[] zone, int maxlength)
{
	zone[0] = '\0';

	if (area == NULL_AREA)
		return;

	ArrayList spots = g_arrMapConfig.adtEngineerNestLocation;
	ArrayList zones = g_arrMapConfig.adtEngineerNestZone;

	float centre[3]; area.GetCenter(centre);

	float best = NEST_SPOT_MATCH_RANGE;

	for (int i = 0; i < spots.Length && i < zones.Length; i++)
	{
		float spot[3]; spots.GetArray(i, spot);

		float distance = GetVectorDistance(centre, spot);

		if (distance < best)
		{
			best = distance;
			zones.GetString(i, zone, maxlength);
		}
	}
}

//The authored spot nearest the centre we already have, when one is close enough to be this nest
static void NestSpotFromList(ArrayList spots, float inout[3], float &best)
{
	float centre[3]; centre = inout;

	for (int i = 0; i < spots.Length; i++)
	{
		float spot[3]; spots.GetArray(i, spot);

		float distance = GetVectorDistance(centre, spot);

		if (distance < best)
		{
			best = distance;
			inout = spot;
		}
	}
}

/* Somebody for the medic to point the medigun at

The beam already attached is the plain answer, and it is what the game itself tracks. Without one,
anybody alive and close enough to catch the beam counts, so he keeps the medigun out while walking
into range rather than swapping twice on the way */
#define MEDIGUN_HEAL_RANGE	450.0

bool MedicHasPatient(int client, int medigun)
{
	if (GetEntPropEnt(medigun, Prop_Send, "m_hHealingTarget") != -1)
		return true;

	for (int i = 1; i <= MaxClients; i++)
	{
		if (i == client || !IsClientInGame(i) || !IsPlayerAlive(i))
			continue;

		if (GetClientTeam(i) != GetClientTeam(client))
			continue;

		if (GetVectorDistance(WorldSpaceCenter(client), WorldSpaceCenter(i)) < MEDIGUN_HEAL_RANGE)
			return true;
	}

	return false;
}

/* A friendly dispenser close enough to the ground being held to hold from instead

The engineers are told where to build. Nobody tells the rest of the team, so a Heavy holding the
bomb twenty metres from a dispenser walks off to a health pack when he is hurt and the bomb is
unguarded while he does it. Standing on the dispenser instead is the same guard position, and it
heals and reloads him without leaving.

Only for a bot that wants it. A healthy bot with full ammo has no business crowding the dispenser
and giving one rocket two bodies to hit */
#define DISPENSER_GUARD_RANGE		600.0
#define DISPENSER_GUARD_HEALTH_RATIO	0.8

int FindFriendlyDispenserNear(int client, const float origin[3], float maxRange = DISPENSER_GUARD_RANGE)
{
	float bestDistance = maxRange;
	int best = -1;

	int dispenser = -1;

	while ((dispenser = FindEntityByClassname(dispenser, "obj_dispenser")) != -1)
	{
		if (GetEntProp(dispenser, Prop_Send, "m_bPlacing") || GetEntProp(dispenser, Prop_Send, "m_bBuilding"))
			continue;

		if (BaseEntity_GetTeamNumber(dispenser) != GetClientTeam(client))
			continue;

		float distance = GetVectorDistance(GetAbsOrigin(dispenser), origin);

		if (distance < bestDistance)
		{
			bestDistance = distance;
			best = dispenser;
		}
	}

	return best;
}

//Hurt, or short of ammo. Either is a reason to hold the ground from on top of the dispenser
bool WantsDispenser(int client)
{
	if (float(GetClientHealth(client)) < float(TF2Util_GetEntityMaxHealth(client)) * DISPENSER_GUARD_HEALTH_RATIO)
		return true;

	return IsAmmoLow(client);
}

int FindSentryBusterNear(const float origin[3], TFTeam enemyTeam, float maxRange)
{
	float bestDistance = maxRange;
	int best = -1;

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || !IsPlayerAlive(i))
			continue;

		if (TF2_GetClientTeam(i) != enemyTeam)
			continue;

		if (!IsSentryBusterRobot(i))
			continue;

		if (TF2Util_IsPointInRespawnRoom(WorldSpaceCenter(i)))
			continue;

		float distance = GetVectorDistance(WorldSpaceCenter(i), origin);

		if (distance < bestDistance)
		{
			bestDistance = distance;
			best = i;
		}
	}

	return best;
}

int FindBotNearestToBombNearestToHatch(int client)
{
	int iBomb = FindBombNearestToHatch();
	
	if (iBomb <= 0)
		return -1;
	
	float flOrigin[3]; flOrigin = WorldSpaceCenter(iBomb);
	
	float flBestDistance = 999999.0;
	int iBestEntity = -1;
	
	for (int i = 1; i <= MaxClients; i++)
	{
		if (i == client)
			continue;
		
		if (!IsClientInGame(i))
			continue;
		
		if (!IsPlayerAlive(i))
			continue;
		
		if (TF2_GetClientTeam(i) != GetPlayerEnemyTeam(client))
			continue;
		
		if (TF2Util_IsPointInRespawnRoom(WorldSpaceCenter(i)))
			continue;
		
		if (IsSentryBusterRobot(i))
			continue;
		
		float flDistance = GetVectorDistance(WorldSpaceCenter(i), flOrigin);
		
		if (flDistance <= flBestDistance)
		{
			flBestDistance = flDistance;
			iBestEntity = i;
		}
	}
	
	return iBestEntity;
}

int FindBombNearestToHatch()
{
	float flOrigin[3]; flOrigin = GetBombHatchPosition();
	
	float flBestDistance = 999999.0;
	int iBestEntity = -1;
	
	int iEnt = -1;
	
	while ((iEnt = FindEntityByClassname(iEnt, "item_teamflag")) != -1)
	{
		if (CaptureFlag_IsHome(iEnt))
			continue;
		
		float flDistance = GetVectorDistance(flOrigin, WorldSpaceCenter(iEnt));
		
		if (flDistance <= flBestDistance)
		{
			flBestDistance = flDistance;
			iBestEntity = iEnt;
		}
	}
	
	return iBestEntity;
}

int SelectRandomReachableEnemy(int actor)
{
	TFTeam opposingTFTeam = GetPlayerEnemyTeam(actor);
	
	int playerarray[MAXPLAYERS + 1];
	int playercount;
	
	for (int i = 1; i <= MaxClients; i++)
	{
		if (i == actor)
			continue;
		
		if (!IsClientInGame(i))
			continue;
		
		if (!IsPlayerAlive(i))
			continue;
		
		if (TF2_GetClientTeam(i) != opposingTFTeam)
			continue;
		
		if (TF2Util_IsPointInRespawnRoom(WorldSpaceCenter(i)))
			continue;
		
		if (IsSentryBusterRobot(i))
			continue;
		
		playerarray[playercount] = i;
		playercount++;
	}
	
	if (playercount > 0)
		return playerarray[GetRandomInt(0, playercount-1)];
	
	return -1;
}

bool IsHealedByMedic(int client)
{
	for (int i = 0; i < TF2_GetNumHealers(client); i++)
	{
		int iHealerIndex = TF2Util_GetPlayerHealer(client, i);
		
		//Not a player.
		if (!BaseEntity_IsPlayer(iHealerIndex))
			continue;
		
		return true;
	}
	
	return false;
}

float[] GetBombHatchPosition(bool bUseAbsOrigin = false)
{
	float vOrigin[3];

	int iHole = FindEntityByClassname(-1, "func_capturezone");
	
	if (iHole != -1)
		vOrigin = bUseAbsOrigin ? GetAbsOrigin(iHole) : WorldSpaceCenter(iHole);
	
	return vOrigin;
}

int GetAcquiredCreditsOfAllWaves(bool withBonus = true)
{
	int ent = FindEntityByClassname(MaxClients + 1, "tf_mann_vs_machine_stats");
	
	if (ent == -1)
	{
		LogError("GetAcquiredCreditsOfAllWaves: Could not find entity tf_mann_vs_machine_stats!");
		return 0;
	}
	
	int total = GetEntProp(ent, Prop_Send, "m_runningTotalWaveStats", _, STATS_CREDITS_ACQUIRED);
	total += GetEntProp(ent, Prop_Send, "m_previousWaveStats", _, STATS_CREDITS_ACQUIRED);
	total += GetEntProp(ent, Prop_Send, "m_currentWaveStats", _, STATS_CREDITS_ACQUIRED);
	
	if (withBonus)
	{
		total += GetEntProp(ent, Prop_Send, "m_runningTotalWaveStats", _, STATS_CREDITS_BONUS);
		total += GetEntProp(ent, Prop_Send, "m_previousWaveStats", _, STATS_CREDITS_BONUS);
		total += GetEntProp(ent, Prop_Send, "m_currentWaveStats", _, STATS_CREDITS_BONUS);
	}
	
	return total;
}

int GerNearestTeammate(int client, const float max_distance)
{
	float origin[3]; origin = WorldSpaceCenter(client);
	
	float bestDistance = 999999.0;
	int bestEntity = -1;
	
	for (int i = 1; i <= MaxClients; i++)
	{
		if (i == client)
			continue;
		
		if (!IsClientInGame(i))
			continue;
		
		if (!IsPlayerAlive(i))
			continue;
		
		if (GetClientTeam(i) != GetClientTeam(client))
			continue;
		
		float distance = GetVectorDistance(WorldSpaceCenter(i), origin);
		
		if (distance <= bestDistance && distance <= max_distance)
		{
			bestDistance = distance;
			bestEntity = i;
		}
	}
	
	return bestEntity;
}

int GetNearestReviveMarker(int client, const float max_distance)
{
	float origin[3]; GetClientAbsOrigin(client, origin);
	
	float bestDistance = 999999.0;
	int bestEntity = -1;
	
	int iEnt = -1;
	while ((iEnt = FindEntityByClassname(iEnt, "entity_revive_marker")) != -1)
	{
		if (BaseEntity_GetTeamNumber(iEnt) != GetClientTeam(client))
			continue;
		
		float distance = GetVectorDistance(origin, GetAbsOrigin(iEnt));
		
		if (distance <= bestDistance && distance <= max_distance)
		{
			bestDistance = distance;
			bestEntity = iEnt;
		}
	}
	
	return bestEntity;
}

int PowerupBottle_GetNumCharges(int bottle)
{
	return GetEntProp(bottle, Prop_Send, "m_usNumCharges");
}

//CTFPowerupBottle::GetPowerupType
int PowerupBottle_GetType(int bottle)
{
	return GetEntProp(bottle, Prop_Send, "m_usAdvancedType");
}

int GetPowerupBottle(int client)
{
	int ent = -1;
	
	while ((ent = FindEntityByClassname(ent, "tf_powerup_bottle")) != -1)
		if (BaseEntity_GetOwnerEntity(ent) == client)
			break;
	
	return ent;
}

//CTFFlameThrower::CanAirBlast
bool CanWeaponAirblast(int weapon)
{
	return TF2Attrib_HookValueInt(0, "airblast_disabled", weapon) == 0;
}

/* How many live enemies stand within radius of a point

Counts the robot at the point too, so one alone answers one. Used to decide whether a rocket is
worth aiming at the ground: splash pays when it catches a crowd and costs damage when it does not */
int CountEnemiesNearPosition(int client, const float origin[3], float radius)
{
	int count = 0;
	TFTeam enemyTeam = GetPlayerEnemyTeam(client);
	
	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || !IsPlayerAlive(i))
			continue;
		
		if (TF2_GetClientTeam(i) != enemyTeam)
			continue;
		
		if (GetVectorDistance(WorldSpaceCenter(i), origin) <= radius)
			count++;
	}
	
	return count;
}

/* To trigger the robo sapper, you need to do several things
- set a builder
- set the object mode to MODE_SAPPER_ANTI_ROBOT or MODE_SAPPER_ANTI_ROBOT_RADIUS
- parent the sapper to some entity
- set the entity the sapper is being built on
- then fire input Enable to call CObjectSapper::OnGoActive */
int SpawnSapper(int owner, int entity, int weapon = -1)
{
	int sapper = CreateEntityByName("obj_attachment_sapper");
	
	if (sapper != -1)
	{
		AcceptEntityInput(sapper, "SetBuilder", owner);
		
		if (weapon > 0)
			TF2_SetObjectMode(sapper, GetEntProp(weapon, Prop_Send, "m_iObjectMode"));
		
		ParentEntity(entity, sapper, BaseEntity_IsPlayer(entity) ? "head" : "weapon_bone");
		SetEntPropEnt(sapper, Prop_Send, "m_hBuiltOnEntity", entity);
		SetEntProp(sapper, Prop_Send, "m_bBuilding", 1);
		DispatchSpawn(sapper);
		RemoveEffects(sapper, EF_NODRAW);
	}
	
	return sapper;
}

void RemoveEffects(int entity, int nEffects)
{
	SetEntProp(entity, Prop_Send, "m_fEffects", GetEntProp(entity, Prop_Send, "m_fEffects") & ~nEffects);
	
	if (nEffects & EF_NODRAW)
		CBaseEntity(entity).DispatchUpdateTransmitState();
}

//Based on CTFKnife::CanPerformBackstabAgainstTarget
bool HasBackstabPotential(int client)
{
	//These are MvM-specific conditions, where stunned bots are usually allowed to be backstabbed
	if (TF2_GetClientTeam(client) == TFTeam_Blue)
	{
		if (TF2_IsPlayerInCondition(client, TFCond_MVMBotRadiowave))
			return true;
		
		if (TF2_IsPlayerInCondition(client, TFCond_Sapped) && !TF2_IsMiniBoss(client))
			return true;
	}
	
	return false;
}

int GetControlPointByID(int pointID)
{
	int ent = -1;
	
	while ((ent = FindEntityByClassname(ent, "team_control_point")) != -1)
		if (GetEntProp(ent, Prop_Data, "m_iPointIndex") == pointID)
			return ent;
	
	return -1;
}

//Return a capture area trigger associated with a control point that the team can capture
int GetCapturableAreaTrigger(TFTeam team)
{
	int trigger = -1;
	
	while ((trigger = FindEntityByClassname(trigger, "trigger_*")) != -1)
	{
		//Only want capture areas
		if (!HasEntProp(trigger, Prop_Data, "CTriggerAreaCaptureCaptureThink"))
			continue;
		
		//Ignore disabled triggers
		if (GetEntProp(trigger, Prop_Data, "m_bDisabled"))
			continue;
		
		//Apparently some community maps don't disable the trigger when capped
		char sCapPointName[32]; GetEntPropString(trigger, Prop_Data, "m_iszCapPointName", sCapPointName, sizeof(sCapPointName));
		
		//Trigger has no point associated with it
		if (strlen(sCapPointName) < 3)
			continue;
		
		//Now find the matching control point
		int point = -1;
		
		while ((point = FindEntityByClassname(point, "team_control_point")) != -1)
		{
			int iPointIndex = GetEntProp(point, Prop_Data, "m_iPointIndex");
			
			if (!TFGameRules_TeamMayCapturePoint(team, iPointIndex))
				continue;
			
			char sName[32]; GetEntPropString(point, Prop_Data, "m_iName", sName, sizeof(sName));
			
			if (strcmp(sName, sCapPointName, false) == 0)
				return trigger;
		}
	}
	
	return -1;
}

//CTFRevolver::CanHeadshot
bool CanRevolverHeadshot(int weapon)
{
	return TF2Attrib_HookValueInt(0, "set_weapon_mode", weapon) == 1;
}

bool IsPlayerMoving(int client)
{
	float vec[3]; CBaseEntity(client).GetAbsVelocity(vec);
	
	return !IsZeroVector(vec);
}

bool CanWeaponAddUberOnHit(int weapon)
{
	return TF2Attrib_HookValueFloat(0.0, "add_onhit_ubercharge", weapon) > 0.0;
}

bool IsCloakedPlayerExposed(int client)
{
	if (TF2_IsPlayerInCondition(client, TFCond_OnFire))
		return true;
	
	if (TF2_IsPlayerInCondition(client, TFCond_Jarated))
		return true;
	
	if (TF2_IsPlayerInCondition(client, TFCond_CloakFlicker))
		return true;
	
	if (TF2_IsPlayerInCondition(client, TFCond_Bleeding))
		return true;
	
	if (TF2_IsPlayerInCondition(client, TFCond_Milked))
		return true;
	
	if (TF2_IsPlayerInCondition(client, TFCond_Gas))
		return true;
	
	return false;
}

int GetHealerOfPlayer(int client, bool bPlayerOnly = false)
{
	for (int i = 0; i < TF2_GetNumHealers(client); i++)
	{
		int healer = TF2Util_GetPlayerHealer(client, i);
		
		if (healer != -1)
		{
			if (bPlayerOnly && !BaseEntity_IsPlayer(healer))
				continue;
			
			return healer;
		}
	}
	
	return -1;
}

bool IsHealedByObject(int client)
{
	for (int i = 0; i < TF2_GetNumHealers(client); i++)
	{
		int healer = TF2Util_GetPlayerHealer(client, i);
		
		if (!BaseEntity_IsBaseObject(healer))
			continue;
		
		return true;
	}
	
	return false;
}

//Return the only entity we can see, -2 if we can see them both
int FindOnlyOneVisibleEntity(int client, int ent1, int ent2)
{
	if (!IsLineOfFireClearEntity(client, GetEyePosition(client), ent1))
	{
		return ent2;
	}
	
	if (!IsLineOfFireClearEntity(client, GetEyePosition(client), ent2))
	{
		return ent1;
	}
	
	return -2;
}

bool CanUsePrimayWeapon(int client)
{
	if (TF2_IsPlayerInCondition(client, TFCond_MeleeOnly))
		return false;
	
	int weapon = GetPlayerWeaponSlot(client, TFWeaponSlot_Primary);
	
	if (weapon == -1)
		return false;
	
	return true;
}

//bool CTFBot::IsLineOfFireClear( const Vector &from, const Vector &to ) const
bool IsLineOfFireClearPosition(int client, const float from[3], const float to[3])
{
	StringMap adtProperties = new StringMap();
	adtProperties.SetValue("m_pPassEnt", client);
	adtProperties.SetValue("m_collisionGroup", COLLISION_GROUP_NONE);
	adtProperties.SetValue("m_iIgnoreTeam", GetClientTeam(client));
	
	TR_TraceRayFilter(from, to, MASK_SOLID_BRUSHONLY, RayType_EndPoint, TraceFilter_TFBot, adtProperties);
	adtProperties.Close();
	
	return !TR_DidHit();
}

//bool CTFBot::IsLineOfFireClear( const Vector &from, CBaseEntity *who ) const
bool IsLineOfFireClearEntity(int client, const float from[3], int who)
{
	StringMap adtProperties = new StringMap();
	adtProperties.SetValue("m_pPassEnt", client);
	adtProperties.SetValue("m_collisionGroup", COLLISION_GROUP_NONE);
	adtProperties.SetValue("m_iIgnoreTeam", GetClientTeam(client));
	
	TR_TraceRayFilter(from, WorldSpaceCenter(who), MASK_SOLID_BRUSHONLY, RayType_EndPoint, TraceFilter_TFBot, adtProperties);
	adtProperties.Close();
	
	return !TR_DidHit() || TR_GetEntityIndex() == who;
}

bool GetBombInfo(BombInfo_t info)
{
	int iAreaCount = TheNavAreas.Count;

	if (iAreaCount <= 0)
		return false;

	float hatch_dist = 0.0;
	
	for (int i = 0; i < (iAreaCount - 1); i++)
	{
		CTFNavArea area = view_as<CTFNavArea>(TheNavAreas.Get(i));
		
		//Skip spawn areas
		if (area.HasAttributeTF(BLUE_SPAWN_ROOM) || area.HasAttributeTF(RED_SPAWN_ROOM))
		{
			continue;
		}
		
		float m_flBombTargetDistance = GetTravelDistanceToBombTarget(area);
		
		hatch_dist = MaxFloat(MaxFloat(m_flBombTargetDistance, hatch_dist), 0.0);
	}
	
	int closest_flag = INVALID_ENT_REFERENCE;
	float closest_flag_pos[3];
	
	int flag = -1;
	while ((flag = FindEntityByClassname(flag, "item_teamflag")) != -1)
	{
		//Ignore bombs not in play
		if (GetEntProp(flag, Prop_Send, "m_nFlagStatus") == TF_FLAGINFO_HOME)
			continue;
		
		float flag_pos[3];
		
		int owner = BaseEntity_GetOwnerEntity(flag);
		
		if (IsValidClientIndex(owner))
		{
			flag_pos = GetAbsOrigin(owner);
		}
		else
		{
			flag_pos = WorldSpaceCenter(flag);
		}
		
		CTFNavArea area = view_as<CTFNavArea>(TheNavMesh.GetNearestNavArea(flag_pos));
		
		if (area == NULL_AREA)
			continue;
		
		if (area.HasAttributeTF(BLUE_SPAWN_ROOM) || area.HasAttributeTF(RED_SPAWN_ROOM))
			continue;
		
		float m_flBombTargetDistance = GetTravelDistanceToBombTarget(area);
		
		if (m_flBombTargetDistance < hatch_dist) 
		{
			closest_flag = flag;
			hatch_dist = m_flBombTargetDistance;
			closest_flag_pos = flag_pos;
		}
	}
	
	float range_fwd   = 2300.0;
	float range_back  = 1000.0;
	
	info.vPosition = closest_flag_pos;
	info.flMaxBattleFront = hatch_dist + range_back;
	info.flMinBattleFront = hatch_dist - range_fwd;
	
	return (closest_flag != INVALID_ENT_REFERENCE);
}

bool IsUpgradeStationEnabled(int station)
{
	static int iOffsetIsEnabled = -1;
	
	//m_bIsEnabled
	if (iOffsetIsEnabled == -1)
		iOffsetIsEnabled = FindDataMapInfo(station, "m_nStartDisabled") + 28;
	
	return GetEntData(station, iOffsetIsEnabled, 1);
}

float[] GetAbsAngles(int entity)
{
	float vec[3]; CBaseEntity(entity).GetAbsAngles(vec);
	
	return vec;
}

/* The whole bomb path: the travel distance to the hatch from the far end of it

Spawn areas are left out because they carry no distance to the hatch, so the far end this finds is
the first area the robots walk into rather than the room they walk out of */
float BombPathLength()
{
	int iAreaCount = TheNavAreas.Count;
	float longest = 0.0;

	for (int i = 0; i < iAreaCount; i++)
	{
		CTFNavArea area = view_as<CTFNavArea>(TheNavAreas.Get(i));

		if (area == NULL_AREA)
			continue;

		if (area.HasAttributeTF(BLUE_SPAWN_ROOM) || area.HasAttributeTF(RED_SPAWN_ROOM))
			continue;

		longest = MaxFloat(longest, GetTravelDistanceToBombTarget(area));
	}

	return longest;
}

/* How far from the hatch an engineer may nest, in travel distance

A fraction of the bomb path rather than a number of units. The path is a few thousand units on
Decoy and several times that on Rottenburg, and the same number of units is a defensible choke on
one map and the robots' spawn door on the other.

An engineer is worth more than the ground it holds: a nest at the front collapses to the first
giant that walks into it, and the wave then runs at a team with no sentry for the rest of it.
Nesting back means the sentry is still alive when the bomb reaches the ground that matters.

Zero when the map has no bomb path to measure, and the caller then takes what it can get */
float NestDistanceLimit()
{
	float length = BombPathLength();

	if (length <= 0.0)
		return 0.0;

	return length * redbots_manager_engineer_nest_depth.FloatValue;
}

//Where each engineer holds, read by the nest scoring below and written by the engineer behaviours
CNavArea m_aNestArea[MAXPLAYERS + 1] = {NULL_AREA, ...};

/* Where an engineer's nest is moving to once the wave is over, NULL_AREA when it is staying put

The bomb does not take the same route for a whole mission. Mannhattan opens and closes gates,
maps drop barricades, and which way the robots come changes from one wave to the next, so ground
that covered the approach in wave one can be facing a wall in wave three. Written once per
between-waves period and read by the engineer behaviours, which are the ones that own the
buildings standing on the old ground */
CNavArea m_aNestAreaRelocate[MAXPLAYERS + 1] = {NULL_AREA, ...};

/* The engineer items a bot has to play differently to play at all

An item definition index rather than a weapon id: the game gives the Gunslinger and the stock
wrench the same TF_WEAPON_WRENCH, and it is the item that decides whether this engineer holds a
level three or spends mini sentries */
#define TF_ITEMDEF_GUNSLINGER		142
#define TF_ITEMDEF_EUREKA_EFFECT	589
#define TF_ITEMDEF_RESCUE_RANGER	997
#define TF_ITEMDEF_WRANGLER		140
#define TF_ITEMDEF_WIDOWMAKER		527
#define TF_ITEMDEF_SHORT_CIRCUIT	528

int GetLoadoutSlotItemDefinitionIndex(int client, int slot)
{
	int weapon = GetPlayerWeaponSlot(client, slot);
	
	if (weapon < 1 || !HasEntProp(weapon, Prop_Send, "m_iItemDefinitionIndex"))
		return -1;
	
	return GetEntProp(weapon, Prop_Send, "m_iItemDefinitionIndex");
}

bool TF2_IsGunslingerEquipped(int client)
{
	return GetLoadoutSlotItemDefinitionIndex(client, TFWeaponSlot_Melee) == TF_ITEMDEF_GUNSLINGER;
}

bool TF2_IsRescueRangerEquipped(int client)
{
	return GetLoadoutSlotItemDefinitionIndex(client, TFWeaponSlot_Primary) == TF_ITEMDEF_RESCUE_RANGER;
}

/* Whether this engineer's gun is paid for out of the metal supply

Every shot from one of these is a sentry repair that does not happen, which is what makes the
metal upgrades the first thing such an engineer should buy rather than a convenience */
bool EngineerGunSpendsMetal(int client)
{
	if (TF2_GetPlayerClass(client) != TFClass_Engineer)
		return false;

	switch (GetLoadoutSlotItemDefinitionIndex(client, TFWeaponSlot_Primary))
	{
		case TF_ITEMDEF_WIDOWMAKER, TF_ITEMDEF_RESCUE_RANGER: return true;
	}

	return GetLoadoutSlotItemDefinitionIndex(client, TFWeaponSlot_Secondary) == TF_ITEMDEF_SHORT_CIRCUIT;
}

//A nest on top of the hatch has nothing in front of it to shoot at
#define NEST_HATCH_CLEARANCE 180.0

//Two nests closer together than this cover the same ground twice and die to the same blast
#define NEST_SPACING 500.0

//How far from the sentry to look for ground to move it to, away from a buster
#define SENTRY_HAUL_SEARCH_RANGE 1200.0

/* How close to the bomb a nest is allowed to be, as a fraction of the sentry's range

A third of it. Closer than that and the sentry spends none of its range: the robots are already
on top of it when it opens fire, the giant that walks in melees it, and the engineer holding it
is standing in the fight rather than behind it */
#define NEST_MIN_BOMB_RANGE_FRACTION 0.34

/* How many pieces of the approach to sample, and what seeing all of them is worth

Bounded because the term is computed for every candidate area: a map hands out as many areas
within a sentry's range of the bomb as its mesh happens to have, and a score is not worth an
unbounded loop. Two dozen spread across the ground is enough to tell a ledge over the choke from
a corner behind a wall */
#define MAX_APPROACH_SAMPLES	24
#define NEST_SIGHT_SCORE	80.0

/* The ground the robots have to cross to reach the target

Areas on the bomb path within a sentry's range of it, taken with a stride so that a mesh with
hundreds of them still describes the whole approach rather than one corner of it */
static void CollectBombApproachAreas(const float target[3], float SentryRange, ArrayList out)
{
	AreasCollector hAreas = TheNavMesh.CollectAreasInRadius(target, SentryRange);

	int count = hAreas.Count();
	int stride = count > MAX_APPROACH_SAMPLES ? count / MAX_APPROACH_SAMPLES : 1;

	for (int i = 0; i < count && out.Length < MAX_APPROACH_SAMPLES; i += stride)
	{
		CTFNavArea area = view_as<CTFNavArea>(hAreas.Get(i));

		if (!area.HasAttributeTF(BOMB_DROP))
			continue;

		if (area.HasAttributeTF(BLUE_SPAWN_ROOM) || area.HasAttributeTF(RED_SPAWN_ROOM))
			continue;

		out.Push(area);
	}

	delete hAreas;
}

/* What this area can actually shoot at, which is the thing a play-test said was missing

"Their sentries are blocked by the walls." A nest was tested against one point, the bomb, and a
line to one point says nothing about a lane. A spot with the bomb visible through a doorway and
a wall across everything either side of it passed, and the sentry built there fires at whatever
crosses the doorway and nothing else.

The nav mesh already knows this. Visibility between areas is computed when the mesh is built, so
asking it is a lookup rather than a trace, and the whole approach can be asked about for the cost
of the one trace this replaces.

A mesh built without visibility data answers no to everything. Then every candidate scores zero
here and the other terms decide, which is what happened before this existed */
static float NestSightScore(CTFNavArea area, ArrayList approach)
{
	if (approach == null || approach.Length == 0)
		return 0.0;

	int seen = 0;

	for (int i = 0; i < approach.Length; i++)
	{
		if (area.IsCompletelyVisible(view_as<CNavArea>(approach.Get(i))))
			seen++;
	}

	return (float(seen) / float(approach.Length)) * NEST_SIGHT_SCORE;
}

/* How good a nest this area is, higher being better

Every candidate handed to this has already passed the tests that make it a nest at all: on the
bomb path, out of the spawn rooms, not on top of the hatch, within nesting depth. What is left is
a matter of degree, and picking the best of them beats the random pick this replaces, which was
free to put the sentry in a corridor behind the fight while a ledge over the choke went unused.

  range     a sentry wants the robots inside its range and not on top of it, so the ideal is most
            of the way out: close in, it is meleed by the first giant to arrive
  height    ground above the path is worth holding: the sentry shoots down onto the robots and the
            splash that answers it mostly does not reach back up
  room      a wide area fits the sentry, the dispenser and the engineer between them
  spacing   two engineers nesting on the same ledge is one blast killing both nests

An engineer carrying a Gunslinger scores the opposite way on range and height. The mini sentry is
built in two seconds and dies in one, so it is spent rather than held: it wants to be near the
robots where it is worth the metal, not on a ledge where it plinks */
float ScoreNestArea(int client, CTFNavArea area, const float target[3], float SentryRange, ArrayList approach = null)
{
	bool disposable = TF2_IsGunslingerEquipped(client);
	
	float center[3]; area.GetCenter(center);
	center[2] += 50.0;
	
	float range = GetVectorDistance(center, target);
	float ideal = SentryRange * (disposable ? 0.35 : 0.75);
	float score = 100.0 - (FloatAbs(range - ideal) / SentryRange) * 100.0;
	
	if (!disposable)
	{
		float height = center[2] - target[2];
		
		if (height > 0.0)
			score += MinFloat(height, 300.0) * 0.1;
	}
	
	score += MinFloat(area.GetSizeX(), area.GetSizeY()) * 0.05;
	
	score += NestSightScore(area, approach);
	
	score += NestCrowdingPenalty(client, area, center);
	
	return score;
}

/* What this ground is worth less for somebody else already being on it

The old rule skipped the one case that matters. An engineer whose nest area was this very area
was passed over with a continue, so two engineers who scored the same area best both walked to it
and stood there placing a sentry on top of a sentry, while an area merely near a held one was
penalised. The same ground is the strongest reason to pick different ground, not a free pass.

A sentry that is already standing there counts as well, whoever built it: a bot who has not
chosen a nest yet, and a human engineer, are both invisible to a rule that only reads other
bots' intentions. */
static float NestCrowdingPenalty(int client, CTFNavArea area, const float center[3])
{
	float penalty = 0.0;
	
	for (int i = 1; i <= MaxClients; i++)
	{
		if (i == client || !IsClientInGame(i))
			continue;
		
		if (m_aNestArea[i] == view_as<CNavArea>(area))
		{
			penalty -= 100.0;
		}
		else if (m_aNestArea[i] != NULL_AREA)
		{
			float other[3]; m_aNestArea[i].GetCenter(other);
			
			if (GetVectorDistance(center, other) < NEST_SPACING)
				penalty -= 50.0;
		}
		
		int sentry = GetObjectOfType(i, TFObject_Sentry);
		
		if (sentry != INVALID_ENT_REFERENCE && GetVectorDistance(center, GetAbsOrigin(sentry)) < NEST_SPACING)
			penalty -= 100.0;
	}
	
	return penalty;
}

/* The best scoring area in the list, or NULL_AREA for an empty one

The lists are tiers: a caller asks for the best of the areas that see the bomb before it asks for
the best of the areas that merely face it, so the score only ever orders areas that are already
equally good on the thing that matters most */
CNavArea BestNestArea(int client, ArrayList areas, const float target[3], float SentryRange)
{
	CNavArea best = NULL_AREA;
	float bestScore = 0.0;
	
	//The ground the robots cross to reach the target, sampled once for the whole list
	ArrayList approach = new ArrayList();
	CollectBombApproachAreas(target, SentryRange, approach);
	
	for (int i = 0; i < areas.Length; i++)
	{
		CTFNavArea area = view_as<CTFNavArea>(areas.Get(i));
		float score = ScoreNestArea(client, area, target, SentryRange, approach);
		
		if (best == NULL_AREA || score > bestScore)
		{
			best = view_as<CNavArea>(area);
			bestScore = score;
		}
	}
	
	delete approach;
	
	return best;
}

/* The authored nests worth offering this engineer

A zone is a piece of ground the map names, and the point of naming it is that somebody should be
on each. So the spots in a zone another engineer already holds are left out, which sends the
second engineer inside when the first one took the courtyard. When every zone is spoken for, or
the map names none, the whole list comes back and the score decides as before */
static void CollectZonedNestAreas(int client, ArrayList out)
{
	ArrayList spots = g_arrMapConfig.adtEngineerNestLocation;
	ArrayList zones = g_arrMapConfig.adtEngineerNestZone;
	
	if (spots.Length == 0)
		return;
	
	ArrayList free = new ArrayList(3);
	
	for (int i = 0; i < spots.Length; i++)
	{
		char zone[NEST_ZONE_LENGTH];
		
		if (i < zones.Length)
			zones.GetString(i, zone, sizeof(zone));
		
		if (Feature(FEATURE_NEST_ZONES) && zone[0] != '\0' && IsNestZoneTaken(client, zone))
			continue;
		
		float spot[3]; spots.GetArray(i, spot);
		free.PushArray(spot);
	}
	
	CollectConfiguredNestAreas(free.Length > 0 ? free : spots, out);
	
	delete free;
}

//Is another engineer's nest one of the spots this zone names?
static bool IsNestZoneTaken(int client, const char[] zone)
{
	ArrayList spots = g_arrMapConfig.adtEngineerNestLocation;
	ArrayList zones = g_arrMapConfig.adtEngineerNestZone;
	
	for (int i = 1; i <= MaxClients; i++)
	{
		if (i == client || !IsClientInGame(i) || m_aNestArea[i] == NULL_AREA)
			continue;
		
		float held[3]; m_aNestArea[i].GetCenter(held);
		
		for (int s = 0; s < spots.Length && s < zones.Length; s++)
		{
			char other[NEST_ZONE_LENGTH]; zones.GetString(s, other, sizeof(other));
			
			if (!StrEqual(other, zone))
				continue;
			
			float spot[3]; spots.GetArray(s, spot);
			
			if (GetVectorDistance(held, spot) < NEST_SPOT_MATCH_RANGE)
				return true;
		}
	}
	
	return false;
}

/* The nest the map configuration asks for, or NULL_AREA when it asks for nothing

A hand placed nest is there because somebody stood on that ground and decided it was the spot, so
it outranks anything the nav mesh reasoning below arrives at. The spots still go through the
score, which is what spreads several engineers across several of them and what keeps a spot from
being used when another engineer already holds it */
CNavArea PickConfiguredNestArea(int client, const float target[3], float SentryRange)
{
	ArrayList areas = new ArrayList();
	
	CollectZonedNestAreas(client, areas);
	
	/* Rottenburg has a spot that only works when a tank is rolling and one that must be left
	empty when it is: a sentry parked on the tank's path is a sentry the tank drives through.
	Which of the two lists applies is a property of the wave, so it is asked here rather than
	baked into the file */
	if (IsTankWave())
		CollectConfiguredNestAreas(g_arrMapConfig.adtNestTankOnlyLocation, areas);
	else
		CollectConfiguredNestAreas(g_arrMapConfig.adtNestNoTankLocation, areas);
	
	CNavArea best = NULL_AREA;
	
	if (areas.Length > 0)
		best = BestNestArea(client, areas, target, SentryRange);
	
	delete areas;
	
	return best;
}

//The nav areas under a list of authored origins, appended to out
static void CollectConfiguredNestAreas(ArrayList spots, ArrayList out)
{
	for (int i = 0; i < spots.Length; i++)
	{
		float spot[3]; spots.GetArray(i, spot);
		
		CNavArea area = TheNavMesh.GetNearestNavArea(spot, false, 500.0, false, true, TEAM_ANY);
		
		if (area != NULL_AREA)
			out.Push(area);
	}
}

/* Does the wave being fought, or the one about to be, contain a tank?

The rest of the mod finds a tank by looking for a live tank_boss, which is the right answer for
shooting at one and the wrong answer for building. An engineer picks its nest and builds during
the between-waves period, when no tank exists yet, so asking the world is always a no.

m_iszMannVsMachineWaveClassNames is the row of class icons the wave bar draws. The game fills it
in before the wave starts, and a wave with a tank in it carries the "tank" icon */
#define MVM_WAVE_CLASS_ICONS_MAX	12
#define MVM_TANK_CLASS_ICON			"tank"

/* Does the coming wave carry robots of this kind?

The icon names are the wave bar's, so they are the mission's own answer and they are filled in
before the wave starts. Matched as a substring because the variants are all suffixed: a wave with
demoknights and burst demos carries "demoknight" and "demo_burst", and both of them throw
explosives at the team */
bool WaveHasClassIcon(const char[] needle)
{
	int rsrc = FindEntityByClassname(MaxClients + 1, "tf_objective_resource");
	
	if (rsrc == -1)
		return false;
	
	for (int i = 0; i < MVM_WAVE_CLASS_ICONS_MAX; i++)
	{
		char icon[64]; TF2_GetMannVsMachineWaveClassName(rsrc, i, icon, sizeof(icon));
		
		if (icon[0] != '\0' && StrContains(icon, needle, false) != -1)
			return true;
	}
	
	return false;
}

bool IsTankWave()
{
	return WaveHasClassIcon(MVM_TANK_CLASS_ICON);
}

//What the coming wave will actually kill the team with
bool WaveHasExplosiveRobots()
{
	return WaveHasClassIcon("demo") || WaveHasClassIcon("soldier") || IsTankWave();
}

bool WaveHasBulletRobots()
{
	return WaveHasClassIcon("heavy") || WaveHasClassIcon("scout") || WaveHasClassIcon("sniper");
}

bool WaveHasFireRobots()
{
	return WaveHasClassIcon("pyro");
}

/* The nest spots the map itself carries, and where they come from

The TODO in this repository said the compiled map does not hold this data and that somebody has
to stand on the ground and decide. That is true of three of the seven official maps. It is not
true of the other four, and it is not true of most community maps built after them.

Decoy, Bigrock, Rottenburg and Mannhattan ship thirteen to twenty-seven bot_hint_sentrygun and
bot_hint_engineer_nest entities each. They are there for the engineer robots, so they are BLU's
and they face the wrong way, and neither of those things matters. A sentry shoots in a circle,
and the ground under a spot Valve chose for one team to hold a corridor from is the same ground
the other team holds the same corridor from. What the entity is really saying is that a level
three fits here, that it has a line down the lane, and that a mapper put it there on purpose.

They are read at runtime rather than copied into the configuration files, so a community map that
built on the same prefabs is covered without anybody authoring anything.

None of it is trusted blindly. Every spot goes through the same nest score as everything else, so
one deep in the robots' half loses to the nav mesh reasoning below on distance to the bomb. A
hand written EngineerNest block still outranks all of it: somebody stood there. */

//Put a limit on everything. Mannhattan carries twenty-seven; a map with a thousand is broken
#define MAX_MAP_HINT_NESTS	64

static ArrayList g_adtMapHintNests;
static bool g_bMapHintNestsLoaded;

//The map changed, so what the last map's entities said about it is worth nothing
void ResetMapHintNests()
{
	g_bMapHintNestsLoaded = false;

	if (g_adtMapHintNests != null)
		g_adtMapHintNests.Clear();
}

static void CollectMapHintNests(const char[] classname)
{
	int entity = -1;

	while ((entity = FindEntityByClassname(entity, classname)) != -1)
	{
		if (g_adtMapHintNests.Length >= MAX_MAP_HINT_NESTS)
			return;

		float origin[3]; origin = GetAbsOrigin(entity);

		if (IsZeroVector(origin))
			continue;

		g_adtMapHintNests.PushArray(origin);
	}
}

/* Read the first time an engineer asks, rather than at map start

The entities are the map's own and they are spawned long before this, but reading them late costs
nothing and does not depend on when a forward happens to fire relative to the level's entities */
static ArrayList MapHintNests()
{
	if (g_adtMapHintNests == null)
		g_adtMapHintNests = new ArrayList(3);

	if (g_bMapHintNestsLoaded)
		return g_adtMapHintNests;

	g_bMapHintNestsLoaded = true;

	CollectMapHintNests("bot_hint_sentrygun");
	CollectMapHintNests("bot_hint_engineer_nest");

	if (redbots_manager_debug.BoolValue)
		PrintToServer("MapHintNests: %d nest spots from the map's own entities", g_adtMapHintNests.Length);

	return g_adtMapHintNests;
}

//The best of the map's own nest spots, or NULL_AREA when the map carries none
CNavArea PickMapHintNestArea(int client, const float target[3], float SentryRange)
{
	ArrayList spots = MapHintNests();

	if (spots.Length == 0)
		return NULL_AREA;

	ArrayList areas = new ArrayList();

	for (int i = 0; i < spots.Length; i++)
	{
		float spot[3]; spots.GetArray(i, spot);

		CNavArea area = TheNavMesh.GetNearestNavArea(spot, false, 500.0, false, true, TEAM_ANY);

		if (area == NULL_AREA)
			continue;

		CTFNavArea tfArea = view_as<CTFNavArea>(area);

		//A spot inside either spawn room is one the engineer cannot hold, whoever it was put there for
		if (tfArea.HasAttributeTF(BLUE_SPAWN_ROOM) || tfArea.HasAttributeTF(RED_SPAWN_ROOM))
			continue;

		areas.Push(area);
	}

	CNavArea best = BestNestArea(client, areas, target, SentryRange);

	delete areas;

	return best;
}

/* Whether a spot is close enough to the bomb to be worth holding, and far enough to survive it

Both bounds matter and only one of them existed. A nest further up the path than the depth limit
is a nest the wave walks past; a nest on top of the bomb is a sentry the first giant meleed, with
a dispenser nobody but the engineer stands at, which is what a play-test found sitting on Decoy's
hatch. Valve keeps its own engineer robots 1300 units off the bomb for the same reason */
static bool IsNestRangeSane(float rangeToBomb, float SentryRange)
{
	return rangeToBomb >= SentryRange * NEST_MIN_BOMB_RANGE_FRACTION && rangeToBomb < SentryRange;
}

CNavArea PickBuildArea(int client, float SentryRange = 1300.0)
{
	int iAreaCount = TheNavAreas.Count;

	if (iAreaCount <= 0)
		return NULL_AREA;
	
	BombInfo_t bombinfo;
	
	if (!GetBombInfo(bombinfo)) 
	{	
		return PickBuildAreaPreRound(client);
	}
	
	float vecTargetPos[3];
	vecTargetPos[0] = bombinfo.vPosition[0];
	vecTargetPos[1] = bombinfo.vPosition[1];
	vecTargetPos[2] = bombinfo.vPosition[2] + 40.0;
	
	CNavArea configured = PickConfiguredNestArea(client, vecTargetPos, SentryRange);
	
	if (configured != NULL_AREA)
		return configured;
	
	CNavArea hinted = PickMapHintNestArea(client, vecTargetPos, SentryRange);
	
	if (hinted != NULL_AREA)
		return hinted;
	
	CTFNavArea bombArea = TheNavMesh.GetNearestNavArea(vecTargetPos, false, 90000.0, false, true, TEAM_ANY);
	
	if (bombArea == NULL_AREA)
	{
		return NULL_AREA;
	}
	
	if (bombArea.HasAttributeTF(BLUE_SPAWN_ROOM) || bombArea.HasAttributeTF(RED_SPAWN_ROOM))
	{
		return NULL_AREA;
	}

	//Areas forward of the bomb within some distance and visible to bomb.
	ArrayList ForwardVisibleAreas = new ArrayList();
	//Areas forward of the bomb but not necessarily visible.
	ArrayList ForwardAreas        = new ArrayList();
	//Areas visible to the bomb but not nescessarily forward of it.
	ArrayList VisibleAreasAround  = new ArrayList();
	//Any of the above, but further up the path than an engineer should nest.
	ArrayList AreasTooFarUp       = new ArrayList();
	//On top of the bomb, which is a nest only when the map offers nothing else.
	ArrayList AreasTooClose       = new ArrayList();

	float limit = NestDistanceLimit();
	
	for (int i = 0; i < iAreaCount; i++)
	{	
		CTFNavArea area = view_as<CTFNavArea>(TheNavAreas.Get(i));
		
		if (area == NULL_AREA)
			continue;
		
		//Area in spawn
		if (area.HasAttributeTF(BLUE_SPAWN_ROOM) || area.HasAttributeTF(RED_SPAWN_ROOM))
			continue;
		
		/* BLOCKED is the one nav attribute that changes during a mission: gates and
		func_nav_blocker set it. PickBuildAreaPreRound has always checked it and this one never
		did, so a nest picked after a gate closed could sit on ground the mesh calls unreachable */
		if (area.HasAttributeTF(BLOCKED))
			continue;
		
		//TODO
		//Better solution because this will break on all non mvm maps.
		//Most likely areachable area
		if (!area.HasAttributeTF(BOMB_DROP))
			continue;
		
		float m_flBombTargetDistanceAtArea = GetTravelDistanceToBombTarget(area);
		float m_flBombTargetDistanceAtBomb = GetTravelDistanceToBombTarget(bombArea);
		
		if (m_flBombTargetDistanceAtArea < NEST_HATCH_CLEARANCE)
			continue;

		/* Further up the path than an engineer nests. Kept, because the bomb spends the start of
		every wave up there and this is where the forward lists would otherwise be empty: better a
		nest too far forward than an engineer that never builds one */
		if (limit > 0.0 && m_flBombTargetDistanceAtArea > limit)
		{
			AreasTooFarUp.Push(area);
			continue;
		}
		
		float areaCenter[3]; area.GetCenter(areaCenter);
		areaCenter[2] += 50.0;
		
		float flAreaDistanceToBomb = GetVectorDistance(areaCenter, vecTargetPos);
		
		if (flAreaDistanceToBomb >= SentryRange)
			continue;
		
		/* Close enough to the bomb that the sentry never uses its range
		Kept rather than dropped, and kept last: a nest on top of the bomb is bad and no nest at
		all is worse, and a map whose every area near the bomb is this close is a map where the
		engineer would otherwise stand around with 300 metal */
		if (!IsNestRangeSane(flAreaDistanceToBomb, SentryRange))
		{
			AreasTooClose.Push(area);
			continue;
		}
		
		bool bAreaVisibleToBomb = area.IsEntirelyVisible(vecTargetPos);
		
		if (bAreaVisibleToBomb)
		{
			VisibleAreasAround.Push(area);
		}
		
		if (m_flBombTargetDistanceAtBomb > m_flBombTargetDistanceAtArea)
		{
			if (flAreaDistanceToBomb <= SentryRange * GetRandomFloat(0.8, 1.75) && bAreaVisibleToBomb)
			{
				ForwardVisibleAreas.Push(area);
			}
			
			ForwardAreas.Push(area);
		}
	}
	
	CNavArea randomArea = NULL_AREA;
	
	if (ForwardVisibleAreas.Length     > 0) randomArea = BestNestArea(client, ForwardVisibleAreas, vecTargetPos, SentryRange);
	else if (ForwardAreas.Length       > 0) randomArea = BestNestArea(client, ForwardAreas,        vecTargetPos, SentryRange);
	else if (VisibleAreasAround.Length > 0) randomArea = BestNestArea(client, VisibleAreasAround,  vecTargetPos, SentryRange);
	else if (AreasTooFarUp.Length      > 0) randomArea = BestNestArea(client, AreasTooFarUp,       vecTargetPos, SentryRange);
	else if (AreasTooClose.Length      > 0) randomArea = BestNestArea(client, AreasTooClose,       vecTargetPos, SentryRange);
	
	if (redbots_manager_debug.BoolValue)
		PrintToServer("PickBuildArea %i ForwardVisibleAreas | %i ForwardAreas | %i VisibleAreasAroundBomb | %i AreasTooFarUp | %i AreasTooClose", ForwardVisibleAreas.Length, ForwardAreas.Length, VisibleAreasAround.Length, AreasTooFarUp.Length, AreasTooClose.Length);
	
	ForwardVisibleAreas.Close();
	ForwardAreas.Close();
	VisibleAreasAround.Close();
	AreasTooFarUp.Close();
	AreasTooClose.Close();
	
	return randomArea;
}

/* Ground to move a sentry to when a buster is walking towards it, or NULL_AREA for none

Not a nest. The sentry is going back where it was as soon as the buster is dead, so this asks for
one thing only: that the blast happens somewhere else. Far enough that the sentry survives it,
near enough that the engineer is not carrying a building across the map while the wave arrives.

Nothing here is clever about where the buster will walk. A buster chases the sentry that hurt it
most, so it follows, and what the engineer buys is the time it takes to walk the difference. That
time is what the team kills it in, and the blast that does land lands away from the dispenser and
away from whoever was standing at the nest */
CNavArea PickBusterRetreatArea(int sentry, int buster)
{
	float sentryOrigin[3]; sentryOrigin = GetAbsOrigin(sentry);
	float busterOrigin[3]; busterOrigin = WorldSpaceCenter(buster);

	//Anywhere the sentry ends up has to beat where it stands now by a blast, or it was not worth moving
	float bestDistance = GetVectorDistance(sentryOrigin, busterOrigin) + BUSTER_BLAST_RANGE;
	CNavArea best = NULL_AREA;

	AreasCollector hAreas = TheNavMesh.CollectAreasInRadius(sentryOrigin, SENTRY_HAUL_SEARCH_RANGE);

	int count = hAreas.Count();

	//One engineer, once per buster, but the count belongs to the map rather than to this
	if (count > 256)
		count = 256;

	for (int i = 0; i < count; i++)
	{
		CTFNavArea area = view_as<CTFNavArea>(hAreas.Get(i));

		if (area.HasAttributeTF(BLUE_SPAWN_ROOM) || area.HasAttributeTF(RED_SPAWN_ROOM))
			continue;

		float center[3]; area.GetCenter(center);

		float distance = GetVectorDistance(center, busterOrigin);

		if (distance <= bestDistance)
			continue;

		bestDistance = distance;
		best = view_as<CNavArea>(area);
	}

	delete hAreas;

	return best;
}

/* Where an engineer nests before a wave begins

There is no bomb yet, so the hatch is what there is to measure from: the areas on the bomb path
within nesting distance of it, preferring the ones close enough to see it, which is the ground the
robots have to cross last and the ground worth holding.

It used to anchor on the robots' spawn door and put the sentry in front of that. That reads well
and plays badly. The nest meets the whole wave at full strength with no team around it yet, the
first giant walks through it, and the engineer spends the rest of the wave rebuilding somewhere
behind while the team fights without a sentry. On a short map like Decoy the door is also most of
the map away from anything that needs defending */
CNavArea PickBuildAreaPreRound(int client, float SentryRange = 1300.0)
{
	int iAreaCount = TheNavAreas.Count;

	if (iAreaCount <= 0)
		return NULL_AREA;

	float limit = NestDistanceLimit();
	float hatch[3]; hatch = GetBombHatchPosition();
	hatch[2] += 40.0;
	
	CNavArea configured = PickConfiguredNestArea(client, hatch, SentryRange);
	
	if (configured != NULL_AREA)
		return configured;

	CNavArea hinted = PickMapHintNestArea(client, hatch, SentryRange);

	if (hinted != NULL_AREA)
		return hinted;

	//Near enough to the hatch to nest, and with a line to it
	ArrayList CoveringAreas = new ArrayList();
	//Near enough to nest, seeing the hatch or not
	ArrayList NestingAreas  = new ArrayList();
	//On the path, but further up it than an engineer should nest
	ArrayList AreasTooFarUp = new ArrayList();
	//On top of the hatch, which is a nest only when the map offers nothing else
	ArrayList AreasTooClose = new ArrayList();

	for (int i = 0; i < iAreaCount; i++)
	{
		CTFNavArea area = view_as<CTFNavArea>(TheNavAreas.Get(i));

		if (area == NULL_AREA)
			continue;

		if (area.HasAttributeTF(BLUE_SPAWN_ROOM) || area.HasAttributeTF(RED_SPAWN_ROOM))
			continue;

		if (area.HasAttributeTF(BLOCKED))
			continue;

		//TODO
		//Better solution because this will break on all non mvm maps.
		if (!area.HasAttributeTF(BOMB_DROP))
			continue;

		float distance = GetTravelDistanceToBombTarget(area);

		if (distance < NEST_HATCH_CLEARANCE)
			continue;

		if (limit > 0.0 && distance > limit)
		{
			AreasTooFarUp.Push(area);
			continue;
		}

		float center[3]; area.GetCenter(center);
		center[2] += 50.0;

		/* Sitting on the hatch is not nesting, whichever tier the area would have landed in
		The clearance above is a travel distance along the bomb path and says nothing about a
		ledge directly over the hatch, which is a short walk and no distance at all */
		if (!IsNestRangeSane(GetVectorDistance(center, hatch), SentryRange))
		{
			AreasTooClose.Push(area);
			continue;
		}

		NestingAreas.Push(area);

		if (area.IsEntirelyVisible(hatch))
			CoveringAreas.Push(area);
	}

	CNavArea bestArea = NULL_AREA;

	if (CoveringAreas.Length > 0)       bestArea = BestNestArea(client, CoveringAreas, hatch, SentryRange);
	else if (NestingAreas.Length > 0)   bestArea = BestNestArea(client, NestingAreas,  hatch, SentryRange);
	else if (AreasTooFarUp.Length > 0)  bestArea = BestNestArea(client, AreasTooFarUp, hatch, SentryRange);
	else if (AreasTooClose.Length > 0)  bestArea = BestNestArea(client, AreasTooClose, hatch, SentryRange);

	if (redbots_manager_debug.BoolValue)
		PrintToServer("PickBuildAreaPreRound %i CoveringAreas | %i NestingAreas | %i AreasTooFarUp | %i AreasTooClose", CoveringAreas.Length, NestingAreas.Length, AreasTooFarUp.Length, AreasTooClose.Length);

	CoveringAreas.Close();
	NestingAreas.Close();
	AreasTooFarUp.Close();
	AreasTooClose.Close();

	return bestArea;
}

/* Whether the ground this engineer holds has stopped being the best ground on offer

Both areas go through ScoreNestArea against the same approach sample, so the two numbers are
comparable: this asks what the nest picker would choose if it ran again now, and by how much that
beats what the engineer already has.

The gain has to be worth what moving costs. A relocation is a rebuild or a carry, either of which
spends the opening of the next wave walking rather than shooting, so the default threshold is half
of NEST_SIGHT_SCORE: what a nest gains by going from seeing half the approach to seeing all of it.
Under that the difference is mostly the range and room terms wobbling as the bomb's reset position
shifts between waves, and that is not a reason to give up a standing level three */
bool ShouldRelocateNest(int client, CNavArea &destination, float SentryRange = 1300.0)
{
	destination = NULL_AREA;
	
	CNavArea current = m_aNestArea[client];
	
	//No nest yet, so there is nothing to compare against and the ordinary picker will build one
	if (current == NULL_AREA)
		return false;
	
	float target[3];
	BombInfo_t bombinfo;
	
	if (GetBombInfo(bombinfo))
	{
		target[0] = bombinfo.vPosition[0];
		target[1] = bombinfo.vPosition[1];
		target[2] = bombinfo.vPosition[2];
	}
	else
	{
		target = GetBombHatchPosition();
	}
	
	target[2] += 40.0;
	
	CNavArea candidate = PickBuildArea(client, SentryRange);
	
	if (candidate == NULL_AREA || candidate == current)
		return false;
	
	ArrayList approach = new ArrayList();
	CollectBombApproachAreas(target, SentryRange, approach);
	
	float gain = ScoreNestArea(client, view_as<CTFNavArea>(candidate), target, SentryRange, approach)
		- ScoreNestArea(client, view_as<CTFNavArea>(current), target, SentryRange, approach);
	
	delete approach;
	
	if (redbots_manager_debug.BoolValue)
		PrintToServer("ShouldRelocateNest: %N would gain %.1f by moving", client, gain);
	
	if (gain < redbots_manager_engineer_nest_relocate_score_gain_min.FloatValue)
		return false;
	
	destination = candidate;
	
	return true;
}

stock bool DoesAnyPlayerUseThisName(const char[] name)
{
	char playerName[MAX_NAME_LENGTH];
	
	for (int i = 1; i <= MaxClients; i++)
		if (IsClientConnected(i) && GetClientName(i, playerName, sizeof(playerName)) && StrEqual(playerName, name, false))
			return true;
	
	return false;
}

stock int ReadInt(Address pAddr)
{
	if (pAddr == Address_Null)
		return -1;
	
	return LoadFromAddress(pAddr, NumberType_Int32);
}

//Somewhat borrowed from [L4D2] Survivor Bot AI Improver
stock void SnapViewToPosition(int iClient, const float fPos[3])
{
	float clientEyePos[3]; GetClientEyePosition(iClient, clientEyePos);
	
	float fDesiredDir[3]; MakeVectorFromPoints(clientEyePos, fPos, fDesiredDir);
	GetVectorAngles(fDesiredDir, fDesiredDir);

	float clientEyeAng[3]; GetClientEyeAngles(iClient, clientEyeAng);
	
	float fEyeAngles[3];
	fEyeAngles[0] = (clientEyeAng[0] + NormalizeAngle(fDesiredDir[0] - clientEyeAng[0]));
	fEyeAngles[1] = (clientEyeAng[1] + NormalizeAngle(fDesiredDir[1] - clientEyeAng[1]));
	fEyeAngles[2] = 0.0;

	TeleportEntity(iClient, NULL_VECTOR, fEyeAngles, NULL_VECTOR);
}

stock float NormalizeAngle(float fAngle)
{
	fAngle = (fAngle - RoundToFloor(fAngle / 360.0) * 360.0);
	if (fAngle > 180.0)fAngle -= 360.0;
	else if (fAngle < -180.0)fAngle += 360.0;
	return fAngle;
}

stock bool IsValidClientIndex(int client)
{
	return client > 0 && client <= MaxClients && IsClientInGame(client);
}

stock bool IsBaseBoss(int entity)
{
	return HasEntProp(entity, Prop_Send, "m_lastHealthPercentage");
}

stock bool IsPlayerReady(int client)
{
	return view_as<bool>(GameRules_GetProp("m_bPlayerReady", 1, client));
}

stock bool IsMeleeWeapon(int entity)
{
	//THINKFUNC Smack
	return HasEntProp(entity, Prop_Data, "CTFWeaponBaseMeleeSmack");
}

stock bool IsZeroVector(float origin[3])
{
	return origin[0] == NULL_VECTOR[0] && origin[1] == NULL_VECTOR[1] && origin[2] == NULL_VECTOR[2];
}

/* Every action the bot is running, innermost first, for the dump commands

Reasoning about which behaviour has a bot from the outside is guesswork, and this session spent two
rounds of it on one medic. The stack says it outright. */
static char m_sActionStack[512];

static void CollectActionName(BehaviorAction action)
{
	char name[ACTION_NAME_LENGTH]; action.GetName(name, sizeof(name));
	
	if (m_sActionStack[0] != '\0')
		StrCat(m_sActionStack, sizeof(m_sActionStack), " < ");
	
	StrCat(m_sActionStack, sizeof(m_sActionStack), name);
}

stock void ActionStackOf(int client, char[] buffer, int maxlength)
{
	m_sActionStack[0] = '\0';
	
	ActionsManager.Iterator(client, CollectActionName);
	
	strcopy(buffer, maxlength, m_sActionStack);
}

/* How long an engineer's walk to a build spot is allowed to take

A flat clock is a clock that is right for one distance. Twelve seconds is generous inside a nest
and nowhere near enough to come back from the upgrade station, and what expiring meant was building
where he stood: a sentry six hundred units from its nest on Decoy, a dispenser beside the
teleporter entrance instead of on the ground the map names for it.

So the walk is priced by how far it is, floored so a short one still has room to go round the spot,
and capped so nothing waits on an engineer for ever. */
#define BUILD_WALK_SPEED	180.0
#define BUILD_WALK_TIME_MIN	12.0
#define BUILD_WALK_TIME_MAX	40.0

stock float BuildReachTime(const float from[3], const float to[3])
{
	float seconds = BUILD_WALK_TIME_MIN + GetVectorDistance(from, to) / BUILD_WALK_SPEED;
	
	return seconds > BUILD_WALK_TIME_MAX ? BUILD_WALK_TIME_MAX : seconds;
}

/* Saying it again is not free, so it is only said when the answer changes

Readiness is reasserted every frame, on purpose, because several places set a bot ready and
gating each of them would be four chances to miss one. What that turned into on the wire was a
tournament_player_readystate command per bot per frame, and a ready screen flickering through
every one of them. The command is the announcement; the flag is the state. */
stock void SetPlayerReady(int client, bool state)
{
	if (IsPlayerReady(client) == state)
		return;
	
	FakeClientCommand(client, "tournament_player_readystate %d", state);
}

stock bool IsPluginMvMCreditsLoaded()
{
	//tf_mvm_credits
	return FindConVar("sm_mvmcredits_version") != null;
}

stock bool IsPluginRTDLoaded()
{
	//rtd
	return FindConVar("sm_rtd2_version") != null;
}

stock void UseActionSlotItem(int client)
{
	KeyValues kv = new KeyValues("use_action_slot_item_server");
	FakeClientCommandKeyValues(client, kv);
	delete kv;
}

stock void PlayerBuyback(int client)
{
	FakeClientCommand(client, "td_buyback");
}

//From stocksoup/entity_tools.inc
stock bool ParentEntity(int parent, int attachment, const char[] attachPoint = "",
		bool maintainOffset = false) {
	SetVariantString("!activator");
	AcceptEntityInput(attachment, "SetParent", parent, attachment, 0);
	
	if (strlen(attachPoint) > 0) {
		SetVariantString(attachPoint);
		AcceptEntityInput(attachment,
				maintainOffset? "SetParentAttachmentMaintainOffset" : "SetParentAttachment",
				parent, parent);
	}
}

//TODO: should use an actual call to CBaseEntity::IsDeflectable
stock bool CanBeReflected(int projectile)
{
	char classname[32]; GetEntityClassname(projectile, classname, sizeof(classname));
	
	if (StrEqual(classname, "tf_projectile_arrow", false)
	|| StrEqual(classname, "tf_projectile_ball_ornament", false)
	|| StrEqual(classname, "tf_projectile_cleaver", false)
	|| StrEqual(classname, "tf_projectile_energy_ball", false)
	|| StrEqual(classname, "tf_projectile_flare", false)
	|| StrEqual(classname, "tf_projectile_healing_bolt", false)
	|| StrContains(classname, "tf_projectile_jar", false) != -1
	|| StrEqual(classname, "tf_projectile_pipe", false)
	|| StrEqual(classname, "tf_projectile_rocket", false)
	|| StrEqual(classname, "tf_projectile_sentryrocket", false)
	|| StrEqual(classname, "tf_projectile_stun_ball", false)
	|| StrEqual(classname, "tf_projectile_balloffire", false))
	{
		return true;
	}
	
	return false;
}

stock bool IsItemDefIndexSapper(int itemDefIndex)
{
	switch (itemDefIndex)
	{
		case 735, 736, 810, 831, 933, 1080, 1102:
		{
			return true;
		}
	}
	
	return false;
}

stock float AngleDiff( float destAngle, float srcAngle )
{
	return AngleNormalize(destAngle - srcAngle);
}

stock float AngleNormalize( float angle )
{
	angle = angle - 360.0 * RoundToFloor(angle / 360.0);
	while (angle > 180.0) angle -= 360.0;
	while (angle < -180.0) angle += 360.0;
	return angle;
}

stock float[] GetAbsVelocity(int entity)
{
	float vec[3];

	CBaseEntity(entity).GetAbsVelocity(vec);
	
	return vec;
}

stock float VMX_VectorNormalize(float a1[3])
{
	float flLength = GetVectorLength(a1, true) + 0.0000000001;
	float v4 = (1.0 / SquareRoot(flLength)); 
	float den = v4 * ((3.0 - ((v4 * v4) * flLength)) * 0.5);
	
	ScaleVector(a1, den);
	
	return den * flLength;
}

stock float[] GetEyePosition(int client)
{
	float vec[3]; BaseEntity_EyePosition(client, vec);
	
	return vec;
}

stock float ApproachAngle( float target, float value, float speed )
{
	float delta = AngleDiff(target, value);
	
	if (speed < 0.0) 
		speed = -speed;
	
	if (delta > speed) 
		value += speed;
	else if (delta < -speed) 
		value -= speed;
	else
		value = target;
	
	return AngleNormalize(value);
}

stock float GetCurrentCharge(int iWeapon)
{
	if (!HasEntProp(iWeapon, Prop_Send, "m_flChargeBeginTime"))
		return 0.0;
	
	float flCharge = 0.0;
	
	float flChargeBeginTime = GetEntPropFloat(iWeapon, Prop_Send, "m_flChargeBeginTime");
	
	if (flChargeBeginTime != 0.0)
	{
		flCharge = MinFloat(1.0, GetGameTime() - flChargeBeginTime);
	}
	
	return flCharge;
}

stock bool IsServerFull()
{
	return GetClientCount(false) >= MaxClients;
}

//From stocksoup/memory.inc
stock Address DereferencePointer(Address addr) {
	// maybe someday we'll do 64-bit addresses
	return view_as<Address>(LoadFromAddress(addr, NumberType_Int32));
}

stock void TFBot_NoticeThreat(int tfbot, int threat)
{
	//UpdateDelayedThreatNotices is called in CTFBotTacticalMonitor::Update, but that behavior can be interrupted so we use it here to ensure he's noticed
	OSLib_RunScriptCode(tfbot, _, _, "self.DelayedThreatNotice(EntIndexToHScript(%d),0);self.UpdateDelayedThreatNotices()", threat);
}

stock void PrintToChatTeam(int team, const char[] format, any ...)
{
	char buffer[254];
	
	for (int i = 1; i <= MaxClients; i++)
	{
		if (IsClientInGame(i) && GetClientTeam(i) == team)
		{
			SetGlobalTransTarget(i);
			VFormat(buffer, sizeof(buffer), format, 3);
			PrintToChat(i, "%s", buffer);
		}
	}
}

stock int GetTeamHumanClientCount(int team)
{
	int count = 0;
	
	for (int i = 1; i <= MaxClients; i++)
		if (IsClientInGame(i) && !IsFakeClient(i) && GetClientTeam(i) == team)
			count++;
	
	return count;
}

/* TODO: remove this as we have a better way to do this
we are only doing this for right now until we can solve a potential issue */
stock int TEMP_GetPlayerMaxHealth(int client)
{
	return GetEntProp(GetPlayerResourceEntity(), Prop_Send, "m_iMaxHealth", _, client);
}

//This seems heavily based on PlayerLocomotion::Approach
stock void MovePlayerTowardsGoal(int client, const float vGoal[3], float vVel[3])
{
	//WASD Movement
	float forward3D[3];
	BasePlayer_EyeVectors(client, forward3D);
	
	float vForward[3];
	vForward[0] = forward3D[0];
	vForward[1] = forward3D[1];
	NormalizeVector(vForward, vForward);
	
	float right[3] 
	right[0] = vForward[1];
	right[1] = -vForward[0];

	//PlayerLocomotion::GetFeet
	float vFeet[3]; GetClientAbsOrigin(client, vFeet);
	
	float to[3]; 
	SubtractVectors(vGoal, vFeet, to);

	NormalizeVector(to, to);

	float ahead = GetVectorDotProduct(to, vForward);
	float side  = GetVectorDotProduct(to, right);
	
	const float epsilon = 0.25;

	if (ahead > epsilon)
	{
		vVel[0] = PLAYER_SIDESPEED;
	}
	else if (ahead < -epsilon)
	{
		vVel[0] = -PLAYER_SIDESPEED;
	}

	if (side <= -epsilon)
	{
		vVel[1] = -PLAYER_SIDESPEED;
	}
	else if (side >= epsilon)
	{
		vVel[1] = PLAYER_SIDESPEED;
	}
}