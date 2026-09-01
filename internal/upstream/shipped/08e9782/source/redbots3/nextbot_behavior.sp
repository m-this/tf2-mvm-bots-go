//CTFBotMedicHeal::m_patient
#define ACTION_HEAL_PATIENT_OFFSET	0x4850

#define FLAMETHROWER_REACH_RANGE	350.0
#define FLAMEBALL_REACH_RANGE	526.0

PathFollower m_pPath[MAXPLAYERS + 1];
ChasePath m_pChasePath[MAXPLAYERS + 1];
float m_flRepathTime[MAXPLAYERS + 1];
static float m_flNextJumpTime[MAXPLAYERS + 1];
static float m_flScoutDoubleJumpTime[MAXPLAYERS + 1];
static int m_iScoutDoubleJumpSide[MAXPLAYERS + 1];

/* The bottle this bot is wearing, kept rather than found again every frame

Finding it walks the entity list looking for a tf_powerup_bottle, and this runs on the player
command, which is every frame for every bot. The bottle is a wearable: it appears when the bot
spawns and does not move afterwards, so it is worth exactly one lookup a life.

The second was worse. This used to be a cached canteen type, written by the purchase code, and the
purchase code is gone: nothing wrote it any more, so the switch below always read "no bottle" and
a bot handed a canteen would never have drunk it. The type comes off the bottle now, which is
where it was always true. */
static int m_hPowerupBottle[MAXPLAYERS + 1] = {INVALID_ENT_REFERENCE, ...};
static float m_ctPowerupBottleLook[MAXPLAYERS + 1];

static int PowerupBottleOf(int client)
{
	int bottle = EntRefToEntIndex(m_hPowerupBottle[client]);
	
	if (bottle != INVALID_ENT_REFERENCE)
		return bottle;
	
	//A bot with no bottle is the normal case now, and it should not cost an entity walk a frame
	if (m_ctPowerupBottleLook[client] > GetGameTime())
		return -1;
	
	m_ctPowerupBottleLook[client] = GetGameTime() + 1.0;
	
	bottle = GetPowerupBottle(client);
	
	if (bottle != -1)
		m_hPowerupBottle[client] = EntIndexToEntRef(bottle);
	
	return bottle;
}
static float m_flNextBottleUseTime[MAXPLAYERS + 1];

#if defined EXTRA_PLUGINBOT
//Replicate behavior of PathFollower's PluginBot
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
#endif

/* What to do when the nav mesh will not give a path to somewhere the bot has to be

Every behaviour in this mod that walks anywhere sets a goal, sets bPathing, and trusts that this
function gets the bot there. Nothing ever checked whether a path came back. ComputeToTarget returns
a bool and it was discarded, so a failed computation left an empty path, Update walked the bot
along nothing, and the behaviour above went on believing it was travelling.

Measured on Coaltown: a medic with a live patient two thousand units away, "walking", path 0 long,
in the same spot for thirty five seconds while a demoman on half health fought without him. That is
the medic stuck in the middle of the map, reported four times and blamed on four different things,
this one included.

The mesh usually refuses from one particular piece of ground rather than for the whole journey, so
the answer is to get off that ground. A step in the goal's direction, guarded the same way the
attack strafe guards one, and the next computation is made from somewhere else. Counted, because a
bot doing this often is a bot the map's nav mesh has a hole in. */
#define PATH_NUDGE_STEP		120.0
#define PATH_RETRY_INTERVAL	0.5

/* How many paths the whole team may compute in one frame
 *
 * NavAreaBuildPath is a search over the map's nav areas and it is what the watchdog has caught the
 * server inside three times now, most recently with nest relocation on: the symbolised frame was
 * CNavArea::GetZ under ComputePortal under NavAreaBuildPath, reached from a plugin's ComputeToPos
 * inside the per-frame PlayerRunCmd forward. An unreachable goal makes that search walk the whole
 * mesh, and six bots asking in the same frame multiplies it.
 *
 * Two a frame at 66 ticks is a hundred and thirty a second, which is far more than the 0.2 second
 * refresh below ever wants, so nobody waits for a path in practice. What it removes is the frame
 * where everybody asks at once. */
#define PATHS_PER_FRAME	2

static int m_iPathBudgetTick;
static int m_iPathsThisTick;

/* Whether there is room to compute a path this frame. Only the per-frame refresh asks: a behaviour
that computes once when it starts has nothing to retry with, so it is never refused. */
static bool TakePathBudget()
{
	int tick = GetGameTickCount();
	
	if (tick != m_iPathBudgetTick)
	{
		m_iPathBudgetTick = tick;
		m_iPathsThisTick = 0;
	}
	
	if (m_iPathsThisTick >= PATHS_PER_FRAME)
		return false;
	
	m_iPathsThisTick++;
	
	return true;
}

static bool m_bPathFailed[MAXPLAYERS + 1];
static int m_iPathFailures[MAXPLAYERS + 1];

//A valid path can still drive a bot into a corner without getting it out of spawn.
#define SPAWN_EXIT_WATCH_INTERVAL  1.0
#define SPAWN_EXIT_STALL_TIME      6.0
#define SPAWN_EXIT_PROGRESS        96.0

static float m_vecSpawnExitProgress[MAXPLAYERS + 1][3];
static float m_flSpawnExitProgressAt[MAXPLAYERS + 1];
static float m_flSpawnExitStartedAt[MAXPLAYERS + 1];
static float m_flSpawnExitWatchAt[MAXPLAYERS + 1];

void ResetSpawnExitWatch(int client)
{
	m_vecSpawnExitProgress[client] = NULL_VECTOR;
	m_flSpawnExitProgressAt[client] = 0.0;
	m_flSpawnExitStartedAt[client] = 0.0;
	m_flSpawnExitWatchAt[client] = 0.0;
}

static float DistanceFromPointToBounds(const float point[3], const float mins[3], const float maxs[3])
{
	float squared;

	for (int axis = 0; axis < 3; axis++)
	{
		float outside;

		if (point[axis] < mins[axis])
			outside = mins[axis] - point[axis];
		else if (point[axis] > maxs[axis])
			outside = point[axis] - maxs[axis];

		squared += outside * outside;
	}

	return SquareRoot(squared);
}

static float DistanceToClosestDefenderSpawn(int client)
{
	float point[3]; point = WorldSpaceCenter(client);
	float closest = -1.0;
	int room = -1;

	while ((room = FindEntityByClassname(room, "func_respawnroom")) != -1)
	{
		int team = BaseEntity_GetTeamNumber(room);

		if (team != 0 && team != GetClientTeam(client))
			continue;

		if (HasEntProp(room, Prop_Data, "m_bDisabled") && GetEntProp(room, Prop_Data, "m_bDisabled"))
			continue;

		float origin[3]; origin = GetAbsOrigin(room);
		float mins[3]; GetEntPropVector(room, Prop_Data, "m_vecMins", mins);
		float maxs[3]; GetEntPropVector(room, Prop_Data, "m_vecMaxs", maxs);
		AddVectors(mins, origin, mins);
		AddVectors(maxs, origin, maxs);

		float distance = DistanceFromPointToBounds(point, mins, maxs);

		if (closest < 0.0 || distance < closest)
			closest = distance;
	}

	return closest;
}

static bool IsInOrNearDefenderSpawn(int client)
{
	if (TF2Util_IsPointInRespawnRoom(WorldSpaceCenter(client), client))
		return true;

	float distance = DistanceToClosestDefenderSpawn(client);
	return distance >= 0.0 && distance <= redbots_manager_spawn_nav_recovery_radius.FloatValue;
}

static bool ShouldWatchDefenderSpawnExit(int client)
{
	RoundState state = GameRules_GetRoundState();
	return state == RoundState_RoundRunning ? g_bHasUpgraded[client] :
		state == RoundState_BetweenRounds && (g_bShoppedThisBreak[client] || !redbots_manager_bot_use_upgrades.BoolValue);
}

static CNavArea FindNearestRecoveryAreaByClassname(int client, const char[] classname)
{
	float clientPosition[3]; clientPosition = WorldSpaceCenter(client);
	float closest = 999999.0;
	CNavArea best = NULL_AREA;
	int entity = -1;

	while ((entity = FindEntityByClassname(entity, classname)) != -1)
	{
		float position[3]; position = WorldSpaceCenter(entity);
		CNavArea area = TheNavMesh.GetNearestNavArea(position, true, 1500.0, true, true, TEAM_ANY);

		if (area == NULL_AREA)
			continue;

		float distance = GetVectorDistance(clientPosition, position);

		if (distance < closest)
		{
			closest = distance;
			best = area;
		}
	}

	return best;
}

int PathFailuresOf(int client)
{
	return m_iPathFailures[client];
}

//Whether the computation that produced the path this bot is holding actually succeeded
bool PathFailedFor(int client)
{
	return m_bPathFailed[client];
}

void NudgeTowardsGoal(int client, INextBot myBot, const float goal[3])
{
	ILocomotion myLoco = myBot.GetLocomotionInterface();
	
	if (!myLoco.IsOnGround())
		return;
	
	float myOrigin[3]; myOrigin = GetAbsOrigin(client);
	float towards[3]; SubtractVectors(goal, myOrigin, towards);
	
	towards[2] = 0.0;
	
	if (NormalizeVector(towards, towards) < 1.0)
		return;
	
	float step[3];
	step[0] = myOrigin[0] + towards[0] * PATH_NUDGE_STEP;
	step[1] = myOrigin[1] + towards[1] * PATH_NUDGE_STEP;
	step[2] = myOrigin[2];
	
	if (!myLoco.IsPotentiallyTraversable(myOrigin, step, IMMEDIATELY) || myLoco.HasPotentialGap(myOrigin, step))
		return;
	
	myLoco.Approach(step);
}


/* Computing a path and walking it, with the refusal noticed
 *
 * Every behaviour in this mod computes into m_pPath and then calls Update on it, and every one of
 * them discarded the bool the computation returns. An empty path walks the bot nowhere while the
 * behaviour above believes it is travelling.
 *
 * It was fixed in PluginBot_SimulateFrame and nowhere else, which left twenty other call sites
 * with it, including the one every fighting class uses. Measured on Coaltown: the Demoman's median
 * distance to the nearest robot is 1044 units while his attack action is trying to close him to
 * 600, and he is the second lowest scoring seat on the team.
 */
void RepathToTarget(int actor, INextBot myBot, int target)
{
	float began = GetEngineTime();
	bool built = m_pPath[actor].ComputeToTarget(myBot, target, PathLengthCap());

	SayIfSlow(actor, began, "to a target");
	NotePathResult(actor, built);
}

/* Write down a path search that cost real time

The crash in the cores is a frame the watchdog killed, and the story is that a bot asking for an
impossible path walks the whole mesh to find that out. Nothing has ever measured what one of these
costs, so the story has never been checked: about thirty five attempts reproduced the wedge and not
one long frame. This says what a search actually takes, which is the number the story rests on. */
#define PATH_SLOW_MS	20.0

static void SayIfSlow(int actor, float began, const char[] what)
{
	float ms = (GetEngineTime() - began) * 1000.0;

	if (ms < PATH_SLOW_MS)
		return;

	LogMessage("Path: %N spent %.0fms searching %s", actor, ms, what);
}

void RepathToPos(int actor, INextBot myBot, const float goal[3])
{
	/* The crash condition, on demand: a goal with no path to it

	A wedged bot is not expensive on its own. What the cores show is a bot asking for a path that
	cannot exist, which sends NavAreaBuildPath across the whole mesh for an answer it never finds,
	every time the behaviour resets. Pinning a bot somewhere he can still path from reproduces the
	wedge and none of the cost, which is why six attempts under load produced no long frame at all.

	So the injector can send the held bot at a point off the mesh instead. */
	float unreachable[3];

	if (DebugFaults_UnreachableGoal(actor, unreachable))
	{
		float began = GetEngineTime();
		bool built = m_pPath[actor].ComputeToPos(myBot, unreachable, PathLengthCap());

		SayIfSlow(actor, began, "a goal with no path");
		NotePathResult(actor, built);
		return;
	}

	float began = GetEngineTime();
	bool built = m_pPath[actor].ComputeToPos(myBot, goal, PathLengthCap());

	SayIfSlow(actor, began, "to a position");
	NotePathResult(actor, built);
}

/* How far a path search may walk before it gives up, and 0.0 for no limit

NavAreaBuildPath searches until it reaches the goal or runs out of mesh, so a goal it cannot reach
costs the whole map every time it is asked. Six bots asking in one frame is the 1833 ms frame
Mannhattan produced and Decoy never did.

The number is generous on purpose. It is not a leash on where a bot may go: it is the point past
which the search has plainly failed, and every real route on these maps is far inside it. */
#define PATH_LENGTH_CAP	6000.0

static float PathLengthCap()
{
	return Feature(FEATURE_PATH_LENGTH_CAP) ? PATH_LENGTH_CAP : 0.0;
}

static void NotePathResult(int actor, bool built)
{
	bool failed = !built || m_pPath[actor].GetLength() <= 0.0;
	
	if (failed && !m_bPathFailed[actor])
		m_iPathFailures[actor]++;
	
	m_bPathFailed[actor] = failed;
}

static CNavArea FindSpawnRecoveryArea(int client, char[] source, int sourceLength)
{
	int anchor = GetCapturableAreaTrigger(GetPlayerEnemyTeam(client));

	if (anchor != -1)
	{
		CNavArea area = TheNavMesh.GetNearestNavArea(WorldSpaceCenter(anchor), true, 1500.0, true, true, TEAM_ANY);

		if (area != NULL_AREA)
		{
			strcopy(source, sourceLength, "capture trigger");
			return area;
		}
	}

	CNavArea best = FindNearestRecoveryAreaByClassname(client, "func_capturezone");

	if (best != NULL_AREA)
	{
		strcopy(source, sourceLength, "func_capturezone");
		return best;
	}

	best = FindNearestRecoveryAreaByClassname(client, "team_control_point");

	if (best != NULL_AREA)
	{
		strcopy(source, sourceLength, "control point");
		return best;
	}

	float shortestBombTravel = 999999.0;

	for (int i = 0; i < TheNavAreas.Count; i++)
	{
		CTFNavArea area = view_as<CTFNavArea>(TheNavAreas.Get(i));

		if (area == NULL_AREA || area.HasAttributeTF(RED_SPAWN_ROOM) || area.HasAttributeTF(BLUE_SPAWN_ROOM))
			continue;

		float travel = GetTravelDistanceToBombTarget(area);

		if (travel >= 0.0 && travel < shortestBombTravel)
		{
			shortestBombTravel = travel;
			best = area;
		}
	}

	if (best != NULL_AREA)
		strcopy(source, sourceLength, "bomb-target NAV");

	return best;
}

//Move to walkable NAV near the final objective, then let normal class behavior resume.
static bool MoveDefenderFromSpawnToBattlefield(int client, const char[] reason)
{
	if (client < 1 || client > MaxClients || !IsClientInGame(client) || !IsPlayerAlive(client) || !g_bIsDefenderBot[client])
		return false;

	if (!IsInOrNearDefenderSpawn(client))
		return false;

	char anchorSource[32];
	CNavArea area = FindSpawnRecoveryArea(client, anchorSource, sizeof(anchorSource));

	if (area == NULL_AREA)
		return false;

	float destination[3]; CNavArea_GetRandomPoint(area, destination);
	destination[2] += 10.0;
	float stopped[3];

	TeleportEntity(client, destination, NULL_VECTOR, stopped);
	CBaseCombatCharacter(client).UpdateLastKnownArea();
	m_flRepathTime[client] = 0.0;
	ResetSpawnExitWatch(client);

	LogMessage("SpawnNavRecovery: %N %s; moved them using %s", client, reason, anchorSource);
	return true;
}

bool RecoverDefenderFromDisconnectedSpawn(int client)
{
	if (client < 1 || client > MaxClients || !IsClientInGame(client) || !IsPlayerAlive(client) || !g_bIsDefenderBot[client])
		return false;

	if (!IsInOrNearDefenderSpawn(client))
		return false;

	char anchorSource[32];
	CNavArea area = FindSpawnRecoveryArea(client, anchorSource, sizeof(anchorSource));

	if (area == NULL_AREA)
		return false;

	float anchor[3]; area.GetCenter(anchor);

	if (IsPathToVectorPossible(client, anchor))
		return false;

	return MoveDefenderFromSpawnToBattlefield(client, "has no route out of spawn");
}

void WatchDefenderSpawnExit(int client)
{
	if (!redbots_manager_spawn_nav_recovery.BoolValue || !ShouldWatchDefenderSpawnExit(client) ||
		TF2_IsInUpgradeZone(client) || !IsInOrNearDefenderSpawn(client))
	{
		ResetSpawnExitWatch(client);
		return;
	}

	float now = GetGameTime();

	if (m_flSpawnExitWatchAt[client] > now)
		return;

	m_flSpawnExitWatchAt[client] = now + SPAWN_EXIT_WATCH_INTERVAL;

	if (m_flSpawnExitStartedAt[client] == 0.0)
		m_flSpawnExitStartedAt[client] = now;
	else if (now - m_flSpawnExitStartedAt[client] >= redbots_manager_spawn_nav_recovery_time.FloatValue)
	{
		MoveDefenderFromSpawnToBattlefield(client, "exceeded the configured time for leaving spawn");
		return;
	}

	float here[3]; here = GetAbsOrigin(client);
	float flatHere[3]; flatHere = here;
	float flatPrevious[3]; flatPrevious = m_vecSpawnExitProgress[client];
	flatHere[2] = 0.0;
	flatPrevious[2] = 0.0;

	if (m_flSpawnExitProgressAt[client] == 0.0 || GetVectorDistance(flatHere, flatPrevious) >= SPAWN_EXIT_PROGRESS)
	{
		m_vecSpawnExitProgress[client] = here;
		m_flSpawnExitProgressAt[client] = now;
		return;
	}

	if (now - m_flSpawnExitProgressAt[client] < SPAWN_EXIT_STALL_TIME)
		return;

	MoveDefenderFromSpawnToBattlefield(client, "made no spawn-exit progress for six seconds");
}

public Action Command_DumpSpawnNav(int client, int args)
{
	ReplyToCommand(client, "Spawn NAV recovery: enabled %d, radius %.0f, max time %.1f", redbots_manager_spawn_nav_recovery.BoolValue,
		redbots_manager_spawn_nav_recovery_radius.FloatValue, redbots_manager_spawn_nav_recovery_time.FloatValue);

	for (int bot = 1; bot <= MaxClients; bot++)
	{
		if (!IsClientInGame(bot) || !IsPlayerAlive(bot) || !g_bIsDefenderBot[bot])
			continue;

		bool strict = TF2Util_IsPointInRespawnRoom(WorldSpaceCenter(bot), bot);
		float distance = DistanceToClosestDefenderSpawn(bot);
		bool near = strict || distance >= 0.0 && distance <= redbots_manager_spawn_nav_recovery_radius.FloatValue;
		float now = GetGameTime();
		float watched = m_flSpawnExitStartedAt[bot] > 0.0 ? now - m_flSpawnExitStartedAt[bot] : 0.0;
		float stalled = m_flSpawnExitProgressAt[bot] > 0.0 ? now - m_flSpawnExitProgressAt[bot] : 0.0;
		float moved = m_flSpawnExitProgressAt[bot] > 0.0 ? GetVectorDistance(GetAbsOrigin(bot), m_vecSpawnExitProgress[bot]) : 0.0;
		char anchorSource[32];
		bool anchorNav = FindSpawnRecoveryArea(bot, anchorSource, sizeof(anchorSource)) != NULL_AREA;

		ReplyToCommand(client, "%N: strict %d, spawn distance %.0f, near %d, eligible %d, upgrade zone %d, watched %.1fs, stalled %.1fs, moved %.0f, anchor %s, anchor NAV %d",
			bot, strict, distance, near, ShouldWatchDefenderSpawnExit(bot), TF2_IsInUpgradeZone(bot), watched, stalled, moved, anchorSource, anchorNav);
	}

	return Plugin_Handled;
}

public Action Command_RecoverSpawnBots(int client, int args)
{
	int recovered;

	for (int bot = 1; bot <= MaxClients; bot++)
	{
		if (!IsClientInGame(bot) || !IsPlayerAlive(bot) || !g_bIsDefenderBot[bot] || !IsInOrNearDefenderSpawn(bot))
			continue;

		if (MoveDefenderFromSpawnToBattlefield(bot, "was manually recovered by an admin"))
			recovered++;
	}

	ReplyToCommand(client, "Recovered %d defender bot(s) from the configured spawn radius.", recovered);
	return Plugin_Handled;
}

#include "generated/attack.sp"
#include "generated/markgiant.sp"
#include "generated/collectmoney.sp"
#include "generated/gotoupgrade.sp"
#include "generated/attributes.sp"
#include "generated/upgrade_rank.sp"
#include "generated/upgrade_rules.sp"
#include "behavior/upgrade.sp"
#include "generated/getammo.sp"
#include "generated/movetofront.sp"
#include "generated/gethealth.sp"
#include "generated/engineeridle.sp"
#include "generated/engineerbuildsentrygun.sp"
#include "generated/engineerbuilddispenser.sp"
#include "generated/engineerbuildteleporter.sp"
#include "generated/engineerbuilddisposable.sp"
#include "generated/spycheck.sp"
#include "generated/stickytrap.sp"
#include "generated/spylurk.sp"
#include "generated/spysap.sp"
#include "generated/spysapplayer.sp"
#include "generated/medicrevive.sp"
#include "generated/medic.sp"
#include "generated/attackforuber.sp"
#include "generated/evadebuster.sp"
#include "generated/campbomb.sp"
#include "generated/attacktank.sp"
#include "generated/destroyteleporter.sp"
#include "generated/guardpoint.sp"
#include "generated/collectnearmoney.sp"

void InitNextBotPathing()
{
	for (int i = 1; i <= MaxClients; i++)
	{
		m_pPath[i] = PathFollower(_, Path_FilterIgnoreActors, Path_FilterOnlyActors);
		m_pChasePath[i] = ChasePath(LEAD_SUBJECT, _, Path_FilterIgnoreActors, Path_FilterOnlyActors);
	}
}

void ResetNextBot(int client)
{
	m_flRepathTime[client] = 0.0;
	m_flNextJumpTime[client] = 0.0;
	ResetSpyCheck(client);
	ResetStickyTrap(client);
	
	m_hPowerupBottle[client] = INVALID_ENT_REFERENCE;
	m_ctPowerupBottleLook[client] = 0.0;
	m_flNextBottleUseTime[client] = 0.0;
	
	m_iAttackTarget[client] = -1;
	m_iTarget[client] = -1;
	m_flNextMarkTime[client] = 0.0;
	m_iCurrencyPack[client] = -1;
	m_iStation[client] = -1;
	m_flNextUpgrade[client] = 0.0;
	m_nPurchasedUpgrades[client] = 0;
	m_flUpgradingTime[client] = 0.0;
	m_iAmmoPack[client] = -1;
	m_vecGoalArea[client] = NULL_VECTOR;
	m_ctMoveTimeout[client] = 0.0;
	m_iHealthPack[client] = -1;
	//NOTE: engineer-specific behavior stuff is reset in the action itself
	m_iSapTarget[client] = -1;
	m_iPlayerSapTarget[client] = -1;
	m_vecStartArea[client] = NULL_VECTOR;
	m_iTankTarget[client] = -1;
	m_iTeleporterTarget[client] = -1;
	m_vecPointDefendArea[client] = NULL_VECTOR;
	
#if defined EXTRA_PLUGINBOT
	g_arrPluginBot[client].Reset();
#endif
}

#if defined EXTRA_PLUGINBOT
void PluginBot_SimulateFrame(int client)
{
	//SimulateFrame > PFContext::RecalculatePath
	//This is used whenever we want to path somewhere constantly
	if (g_arrPluginBot[client].bPathing)
	{
		if (TF2_GetPlayerClass(client) == TFClass_Engineer)
		{
			//Dumb hack for engineer so pathing does not conflict
			if (ActionsManager.LookupEntityActionByName(client, "DefenderGetAmmo") != INVALID_ACTION || ActionsManager.LookupEntityActionByName(client, "DefenderGetHealth") != INVALID_ACTION)
				return;
		}
		
		bool shouldPathToVec = g_arrPluginBot[client].HasPathGoalVector();
		bool shouldPathToEntity = g_arrPluginBot[client].HasPathGoalEntity();
		
		if (shouldPathToVec || shouldPathToEntity)
		{
			INextBot myBot = CBaseNPC_GetNextBotOfEntity(client);
			
			float goal[3];
			
			if (shouldPathToVec)
				goal = g_arrPluginBot[client].vecPathGoal;
			else
				goal = GetAbsOrigin(g_arrPluginBot[client].iPathGoalEntity);
			
			if (m_flRepathTime[client] <= GetGameTime() && TakePathBudget())
			{
				CBaseCombatCharacter(client).UpdateLastKnownArea();
				
				bool built;
				
				if (shouldPathToVec)
					built = m_pPath[client].ComputeToPos(myBot, g_arrPluginBot[client].vecPathGoal, PathLengthCap());
				else
					built = m_pPath[client].ComputeToTarget(myBot, g_arrPluginBot[client].iPathGoalEntity, PathLengthCap());
				
				//An empty path is a failure the same as a refusal, and it is the shape seen in play
				bool failed = !built || m_pPath[client].GetLength() <= 0.0;
				
				if (failed && !m_bPathFailed[client])
					m_iPathFailures[client]++;
				
				m_bPathFailed[client] = failed;
				
				//Retrying a refusal on the walking interval is most of a frame's path work for nothing
				m_flRepathTime[client] = GetGameTime() + (failed ? PATH_RETRY_INTERVAL : 0.2);
			}
			
			//I don't see a reason to use UpdateLastKnownArea again
			
			if (m_bPathFailed[client])
				NudgeTowardsGoal(client, myBot, goal);
			else
				m_pPath[client].Update(myBot);
		}
	}
}
#endif

//Ends the behaviour of the bot the empty-stack injector holds, and does nothing otherwise
public Action CTFBotMainAction_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (g_bIsDefenderBot[actor] && DebugFaults_ShouldEmpty(actor))
		return action.Done("DebugFaults: emptying the stack");

	return Plugin_Continue;
}

public void OnActionCreated(BehaviorAction action, int actor, const char[] name)
{
	//TFBots are players, ignore all other nextbots
	if (actor <= MaxClients)
	{
		if (StrEqual(name, "MainAction"))
		{
			action.SelectTargetPoint = CTFBotMainAction_SelectTargetPoint;
			action.ShouldAttack = CTFBotMainAction_ShouldAttack;
			action.Update = CTFBotMainAction_Update;
		}
		else if (StrEqual(name, "TacticalMonitor"))
		{
			action.Update = CTFBotTacticalMonitor_Update;
			
			/* NOTE: I've noticed this seems to be very inconsistent at the MainAction level and it also seems to behave differently on windows vs linux
			Let's just override it at the TacticalMonitor level, though this one doesn't actually have a function for it in its class
			But since all nextbot callbacks are virtual i think this should work fine */
			action.SelectMoreDangerousThreat = CTFBotMainAction_SelectMoreDangerousThreat;
		}
		else if (StrEqual(name, "ScenarioMonitor"))
		{
			action.Update = CTFBotScenarioMonitor_Update;
		}
		else if (StrEqual(name, "Heal"))
		{
			action.UpdatePost = CTFBotMedicHeal_UpdatePost;
		}
		else if (StrEqual(name, "FetchFlag"))
		{
			action.OnStart = CTFBotFetchFlag_OnStart;
		}
		else if (StrEqual(name, "MvMEngineerIdle"))
		{
			action.OnStart = CTFBotMvMEngineerIdle_OnStart;
		}
		else if (StrEqual(name, "SniperLurk"))
		{
			action.Update = CTFBotSniperLurk_Update;
			action.SelectMoreDangerousThreat = CTFBotSniperLurk_SelectMoreDangerousThreat;
		}
		else if (StrEqual(name, "SpyLeaveSpawnRoom"))
		{
			action.OnStart = CTFBotSpyLeaveSpawnRoom_OnStart;
		}
	}
}

/* What a robot is worth killing first

The ranges, the enum and the generated table are in generated/threat_priority.sp, written from
internal/threat in tf2-mvm-bots-go. The chain below is the one that shipped, kept beside it so the
two can be played against each other: see FEATURE_GENERATED_THREAT_PRIORITY. */

static int ThreatPriority(int threat, float rangeSq)
{
	if (rangeSq < THREAT_URGENT_RANGE * THREAT_URGENT_RANGE)
		return THREAT_PRIORITY_URGENT;
	
	//Too far to be worth walking the aim across the map for
	if (rangeSq > THREAT_PRIORITY_RANGE * THREAT_PRIORITY_RANGE)
		return THREAT_PRIORITY_NONE;
	
	if (!BaseEntity_IsPlayer(threat) || !IsClientInGame(threat))
		return THREAT_PRIORITY_NONE;
	
	switch (TF2_GetPlayerClass(threat))
	{
		//A giant with a Medic on it is not killable until the Medic is dead
		case TFClass_Medic:
			return THREAT_PRIORITY_MEDIC;
		
		//The two the rest of the team cannot get to: one sits out of reach, the other builds
		case TFClass_Sniper, TFClass_Engineer:
			return THREAT_PRIORITY_SUPPORT;
	}
	
	bool giant = TF2_IsMiniBoss(threat);
	bool carrier = TF2_HasTheFlag(threat);
	
	//Carrying the bomb halves a robot's speed, except a giant's, so that one is still running
	if (giant && carrier)
		return THREAT_PRIORITY_GIANT_BOMB;
	
	if (giant)
		return THREAT_PRIORITY_GIANT;
	
	if (carrier)
		return THREAT_PRIORITY_BOMB;
	
	return THREAT_PRIORITY_NONE;
}

/* The same question, asked of the generated table

The record is what the move in mvm-z83.6 was for: the decision takes what is known about a threat
rather than an entity index, so something that occupies no player slot can still be ranked. Every
threat scan in this mod walks player slots and a tank occupies none, which is mvm-ds3, and this
does not fix it. It makes fixing it possible.

Every field after isPlayer is filled behind it, not beside it, and the first version of this was
not. The chain above reads the class, the miniboss flag and the carrier flag only after its player
test has passed, and all three throw when asked about something that is not a player: measured,
TF2_HasTheFlag threw 3933 times over four waves on tank_boss and obj_attachment_sapper, and each
one aborted the whole threat choice for that tick.

The decision reads none of them for a non-player, so filling them false costs nothing. That is
asserted in internal/threat rather than assumed here. See mvm-z83.46. */
static int ThreatPriorityGenerated(int threat, float rangeSq)
{
	bool isPlayer = BaseEntity_IsPlayer(threat);
	bool inGame = isPlayer && IsClientInGame(threat);
	
	if (!inGame)
		return ThreatPriorityOf(rangeSq, isPlayer, false, TFClass_Unknown, false, false);
	
	return ThreatPriorityOf(rangeSq, isPlayer, true, TF2_GetPlayerClass(threat),
		TF2_IsMiniBoss(threat), TF2_HasTheFlag(threat));
}

/* Where the generated answer and the shipped chain part company, while the port is measured

The differential test proves the decision and the table agree on every combination it can be asked
about. It cannot prove the edge fills the record the way the chain reads it, because it drives both
sides from the same record. Only a running game can answer that.

Scaffolding, to be deleted with the measurement. It runs on the armed side only, so the other arm
pays nothing, and it stops writing after twenty lines because a disagreement that happens at all is
the finding and a log full of them is not more of one. */
static int g_iThreatSplits;
static int g_iThreatCompared;

/* Say how much was compared, not only what disagreed

Zero disagreements and never having run look identical in a log that only writes on a
disagreement, and reading the first as the second is the fault mvm-z83.23 is about. */
void ThreatPortAudit_Report()
{
	if (g_iThreatCompared == 0)
		return;
	
	LogMessage("threat audit: %d compared, %d disagreed", g_iThreatCompared, g_iThreatSplits);
	
	g_iThreatCompared = 0;
}

static void ThreatPortAudit(int threat, float rangeSq)
{
	g_iThreatCompared++;
	
	if (g_iThreatSplits >= 20)
		return;
	
	int shipped = ThreatPriority(threat, rangeSq);
	int fromTable = ThreatPriorityGenerated(threat, rangeSq);
	
	if (shipped == fromTable)
		return;
	
	g_iThreatSplits++;
	
	char classname[64];
	GetEntityClassname(threat, classname, sizeof(classname));
	
	LogMessage("threat audit: entity %d/%s rangeSq %.0f, chain says %d, table says %d",
		threat, classname, rangeSq, shipped, fromTable);
}

public Action CTFBotMainAction_SelectMoreDangerousThreat(BehaviorAction action, INextBot nextbot, int entity, CKnownEntity threat1, CKnownEntity threat2, CKnownEntity& knownEntity)
{
	int me = action.Actor;
	
	if (g_bIsDefenderBot[me] == false)
		return Plugin_Continue;
	
	int myWeapon = BaseCombatCharacter_GetActiveWeapon(me);
	
	if (myWeapon != -1 && (TF2Util_GetWeaponID(myWeapon) == TF_WEAPON_FLAMETHROWER || IsMeleeWeapon(myWeapon)))
	{
		//Always target the closest one to us with these weapons
		knownEntity = HealerOrThreat(nextbot, SelectCloserThreat(nextbot, threat1, threat2));
		return Plugin_Changed;
	}
	
	int iThreat1 = threat1.GetEntity();
	int iThreat2 = threat2.GetEntity();
	
	//If we can only see one threat, then it's our best target
	int oneVisible = FindOnlyOneVisibleEntity(me, iThreat1, iThreat2);
	
	if (oneVisible == iThreat1)
	{
		knownEntity = HealerOrThreat(nextbot, threat1);
		return Plugin_Changed;
	}
	
	if (oneVisible == iThreat2)
	{
		knownEntity = HealerOrThreat(nextbot, threat2);
		return Plugin_Changed;
	}
	
	if (myWeapon != -1 && TF2Util_GetWeaponID(myWeapon) == TF_WEAPON_MINIGUN)
	{
		if (TF2_IsRageDraining(me))
		{
			//When using knockback rage, focus only on particular threats
			if (BaseEntity_IsPlayer(iThreat1) && (TF2_HasTheFlag(iThreat1) || TF2_IsMiniBoss(iThreat1)))
			{
				knownEntity = threat1;
				return Plugin_Changed;
			}
			
			if (BaseEntity_IsPlayer(iThreat2) && (TF2_HasTheFlag(iThreat2) || TF2_IsMiniBoss(iThreat2)))
			{
				knownEntity = threat2;
				return Plugin_Changed;
			}
		}
		
		//Minigun deals 75% less damage against tanks so prioritize them least
		if (IsBaseBoss(iThreat1) && !IsBaseBoss(iThreat2))
		{
			knownEntity = threat2;
			return Plugin_Changed;
		}
		
		if (!IsBaseBoss(iThreat1) && IsBaseBoss(iThreat2))
		{
			knownEntity = threat1;
			return Plugin_Changed;
		}
	}
	
	float rangeSq1 = nextbot.GetRangeSquaredTo(iThreat1);
	float rangeSq2 = nextbot.GetRangeSquaredTo(iThreat2);
	
	/* Shoot the robot that is worth shooting, not the one that happens to be nearest

	Every guide written about this mode says the same order, and none of it was here: the Medic
	first because a giant being healed cannot be killed at all, then the Sniper and the Engineer
	because they are the two the rest of the team cannot reach, then giants, then whoever is
	holding the bomb. A robot close enough to be killing the bot outranks all of it, because a
	priority target is worth nothing to a corpse */
	bool generated = Feature(FEATURE_GENERATED_THREAT_PRIORITY);
	
	int priority1 = generated ? ThreatPriorityGenerated(iThreat1, rangeSq1) : ThreatPriority(iThreat1, rangeSq1);
	int priority2 = generated ? ThreatPriorityGenerated(iThreat2, rangeSq2) : ThreatPriority(iThreat2, rangeSq2);
	
	if (generated)
	{
		ThreatPortAudit(iThreat1, rangeSq1);
		ThreatPortAudit(iThreat2, rangeSq2);
	}
	
	if (Feature(FEATURE_THREAT_PRIORITY) && priority1 != priority2)
	{
		knownEntity = priority1 > priority2 ? threat1 : threat2;
	}
	//Target the closest visible
	else if (rangeSq1 < rangeSq2)
	{
		knownEntity = threat1;
	}
	else
	{
		knownEntity = threat2;
	}
	
	//Target the healer
	knownEntity = HealerOrThreat(nextbot, knownEntity);
	
	return Plugin_Changed;
}

public Action CTFBotMainAction_SelectTargetPoint(BehaviorAction action, INextBot nextbot, int entity, float vec[3])
{
	int me = action.Actor;
	
	if (g_bIsDefenderBot[me] == false)
		return Plugin_Continue;
	
	int myWeapon = BaseCombatCharacter_GetActiveWeapon(me);
	
	if (myWeapon != -1)
	{
		switch (TF2Util_GetWeaponID(myWeapon))
		{
			case TF_WEAPON_GRENADELAUNCHER, TF_WEAPON_PIPEBOMBLAUNCHER:
			{
				//TFBots can't compensate their arc if projectile speed differs, so we do our own calculation here
				float target_point[3];
				
				target_point = WorldSpaceCenter(entity);
				float vecTarget[3], vecActor[3];
				vecTarget = GetAbsOrigin(entity);
				GetClientAbsOrigin(me, vecActor);
				
				float distance = GetVectorDistance(vecTarget, vecActor);
				
				if (distance > 150.0)
				{
					distance = distance / GetProjectileSpeed(myWeapon);
					
					float absVelocity[3]; CBaseEntity(entity).GetAbsVelocity(absVelocity);
					
					target_point[0] = vecTarget[0] + absVelocity[0] * distance;
					target_point[1] = vecTarget[1] + absVelocity[1] * distance;
					target_point[2] = vecTarget[2] + absVelocity[2] * distance;
				}
				else
				{
					target_point = WorldSpaceCenter(entity);
				}
				
				float vecToTarget[3]; SubtractVectors(target_point, vecActor, vecToTarget);
				
				float a5 = NormalizeVector(vecToTarget, vecToTarget);
				
				float ballisticElevation = 0.0125 * a5;
				
				if (ballisticElevation > 45.0)
					ballisticElevation = 45.0;
				
				float elevation = ballisticElevation * (FLOAT_PI / 180.0);
				float sineValue = Sine(elevation);
				float cosineValue = Cosine(elevation);
				
				if (cosineValue != 0.0)
					target_point[2] += (sineValue * a5) / cosineValue;
				
				vec = target_point;
				
				return Plugin_Changed;
			}
			case TF_WEAPON_PARTICLE_CANNON:
			{
				//TFBots won't do projectile prediciton with cow mangler 5000 since it's left out of the code, so we'll do it ourselves
				float target_point[3];
				
				float vecTarget[3], vecActor[3];
				vecTarget = GetAbsOrigin(entity);
				vecActor = GetAbsOrigin(me);
				
				float distance = GetVectorDistance(vecTarget, vecActor);
				
				if (distance > 150.0)
				{
					distance = distance * 0.00090909092;
					
					float absVelocity[3]; CBaseEntity(entity).GetAbsVelocity(absVelocity);
					
					target_point[0] = vecTarget[0] + absVelocity[0] * distance;
					target_point[1] = vecTarget[1] + absVelocity[1] * distance;
					target_point[2] = vecTarget[2] + absVelocity[2] * distance;
					
					if (!IsLineOfFireClearPosition(me, GetEyePosition(me), target_point))
					{
						vecTarget = WorldSpaceCenter(entity);
						
						target_point[0] = vecTarget[0] + absVelocity[0] * distance;
						target_point[1] = vecTarget[1] + absVelocity[1] * distance;
						target_point[2] = vecTarget[2] + absVelocity[2] * distance;
					}
				}
				else
				{
					target_point = WorldSpaceCenter(entity);
				}
				
				vec = target_point;
				
				return Plugin_Changed;
			}
			case TF_WEAPON_ROCKETLAUNCHER:
			{
				/* Splash, when there is a crowd standing in it

				ShouldAimRocketsAtFeet was written for the aiming code behind IDLEBOT_AIMING, which
				is not compiled, so nothing ever asked it anything. The live aiming is Valve's, and
				Valve's aims at the middle of the robot, which is the right answer for one robot and
				the wrong one for the line of them walking a choke */
				if (BaseEntity_IsPlayer(entity) && ShouldAimRocketsAtFeet(me, entity, TF_WEAPON_ROCKETLAUNCHER))
				{
					vec = GetAbsOrigin(entity);

					return Plugin_Changed;
				}
			}
			case TF_WEAPON_SNIPERRIFLE, TF_WEAPON_SNIPERRIFLE_DECAP, TF_WEAPON_SNIPERRIFLE_CLASSIC:
			{
				//For sniper rifles, try to lookup their head bone to aim at
				int bone = LookupBone(entity, "bip_head");
				
				if (bone != -1)
				{
					float vEmpty[3];
					GetBonePosition(entity, bone, vec, vEmpty);
					vec[2] += 3.0;
					
					return Plugin_Changed;
				}
				
				//For sniper rifles, TFBots always try to aim at the entity's eye position on harder difficulties
			}
			case TF_WEAPON_REVOLVER:
			{
				//Try to aim for the head with ambassador
				if (CanRevolverHeadshot(myWeapon))
				{
					int bone = LookupBone(entity, "bip_head");
					
					if (bone != -1)
					{
						float vEmpty[3];
						GetBonePosition(entity, bone, vec, vEmpty);
						vec[2] += 3.0;
						
						return Plugin_Changed;
					}
					
					vec = GetEyePosition(entity);
					
					return Plugin_Changed;
				}
			}
			case TF_WEAPON_FLAMETHROWER:
			{
				/* Aim above a tank rather than at it

				Flames rise, so a Pyro standing at the treads and pointing straight ahead puts
				half of every puff into the ground. The wiki's advice is to aim at the top of it,
				which is what this offset is, and a Pyro is the tank damage in most lineups */
				if (IsBaseBoss(entity))
				{
					GetFlameThrowerAimForTank(entity, vec);
					
					return Plugin_Changed;
				}
			}
		}
	}
	
	//Let the game do its default aiming
	return Plugin_Continue;
}

static Action CTFBotMainAction_ShouldAttack(BehaviorAction action, INextBot nextbot, CKnownEntity knownEntity, QueryResultType& result)
{
	int me = action.Actor;
	
	if (g_bIsDefenderBot[me] == false)
		return Plugin_Continue;
	
	//Always attack even in spawn room because we are not the invaders
	result = ANSWER_YES;
	return Plugin_Changed;
}

/* A bot is ready when it has done the thing its seat exists for

An engineer pressed ready the moment a sentry entity existed, which is a level one still being
hammered together, and the wave started in front of it. A level three sentry and a level three
dispenser are what an engineer's seat is for; the teleporter can be built in his own time.

The medic used to be held here too, until his charge was full. It is off again: a charge builds
into whoever he is beaming, so the wait is however long it takes him to find somebody and stay
next to them, which was minutes of everybody else standing about, and a medic who wandered off
looking for a patient held the wave from wherever he ended up. He is ready when he spawns.

Several places set a bot ready and gating each of them would be four chances to miss one, so this
takes the ready away again while the nest is unfinished, every frame, wherever it came from.

The grace is the important part. A bot that cannot finish, because a buster took the sentry or
the metal ran out, must not hold the wave forever: past it it is ready whatever it has. */
#define READY_GRACE	90.0
#define BUILDING_MAX_LEVEL		3

static float m_ctReadyDeadline[MAXPLAYERS + 1];

static bool IsBuildingFinished(int building)
{
	if (building == INVALID_ENT_REFERENCE)
		return false;

	if (TF2_IsBuilding(building))
		return false;

	return GetEntProp(building, Prop_Send, "m_iUpgradeLevel") >= BUILDING_MAX_LEVEL;
}

bool IsEngineerNestFinished(int client)
{
	return IsBuildingFinished(GetObjectOfType(client, TFObject_Sentry))
		&& IsBuildingFinished(GetObjectOfType(client, TFObject_Dispenser));
}

//Whether this bot has done the thing its seat exists for, before the wave starts
static bool IsDefenderPrepared(int client)
{
	//Credits in a pocket are worth nothing, and the whole break exists for spending them
	if (redbots_manager_bot_use_upgrades.BoolValue && !g_bShoppedThisBreak[client])
		return false;

	/* Whoever walks to the front is prepared once he is stood there

	Without this the last bot to finish shopping starts the wave, and the walk to the front is
	whatever fits in the time nobody is waiting for: measured on Coaltown, where the front is five
	thousand units from the station, that was never the whole walk. Nobody ever arrived. */
	if (ShouldTakeUpPosition(client))
		return IsWaitingAtTheFront(client);

	if (TF2_GetPlayerClass(client) != TFClass_Engineer)
		return true;

	if (!IsEngineerNestFinished(client))
		return false;

	/* The teleporter too, but only while nobody is being made to wait for it

	The nest finishing is what lets the wave start, and the engineer's teleporter window is
	whatever is left of the between-rounds time after it. On a team of nothing but bots that is
	nothing at all: the last nest finishes, everybody is ready, the wave starts, and the build
	action gives up on its first update with "Wave started". No engineer had ever finished one.

	Not with a player on the server. Somebody who has finished shopping should not be held at the
	ready screen for a building the bots want, and their shopping is already the time the engineer
	needs. */
	if (GetRealPlayerCount() > 0)
		return true;

	return !ShouldBuildTeleporter(client);
}

/* Readying a bot, and ending whatever is stopping it saying so

A taunt holds the ready. The command goes out every frame while the flag disagrees, so a short
taunt only delays it, but a looping taunt never ends on its own and the wave waits on a bot doing
a dance. Between rounds a taunt is worth nothing to anybody, so it loses to the ready rather than
the other way round. */
static void ReadyDefender(int actor, bool state)
{
	if (state && TF2_IsPlayerInCondition(actor, TFCond_Taunting))
		TF2_RemoveCondition(actor, TFCond_Taunting);
	
	SetPlayerReady(actor, state);
}

static void UpdateDefenderReadiness(int actor)
{
	if (!Feature(FEATURE_READY_WHEN_PREPARED))
		return;
	
	if (GameRules_GetRoundState() != RoundState_BetweenRounds)
	{
		m_ctReadyDeadline[actor] = 0.0;
		return;
	}

	/* With a person on the team, the bots are a mirror of what the people have said

	Everything below this exists for a team of nothing but bots, where somebody has to decide the
	nest is finished before the wave starts. With a player on RED it is his call and only his: he
	presses F4 when he has finished shopping, and a bot holding the wave for its own reasons is a
	player staring at "Waiting for team to organize" with no way to find out which bot, or why, or
	how long for. Reported from play, with two engineers still building.

	One person saying ready readies the bots, and the last person taking it back takes theirs back
	too, so somebody who changes his mind gets his upgrade time and does not have to fight six bots
	for it. */
	if (AnyHumanOnRed())
	{
		m_ctReadyDeadline[actor] = 0.0;
		
		ReadyDefender(actor, AnyHumanReadyOnRed());
		
		return;
	}

	if (m_ctReadyDeadline[actor] <= 0.0)
		m_ctReadyDeadline[actor] = GetGameTime() + READY_GRACE;

	/* Past the grace he says he is ready, rather than merely stopping being made unready
	
	Nothing else says it for him. A bot readies when it leaves the upgrade station or moves to the
	front, and an engineer whose nest will not finish does neither: he is still trying to build.
	Letting go of the ready was not the same as pressing it, so the wave waited for him for as
	long as he kept trying, which is the whole round. */
	if (GetGameTime() > m_ctReadyDeadline[actor])
	{
		if (!IsPlayerReady(actor))
			ReadyDefender(actor, true);
		
		return;
	}

	if (IsDefenderPrepared(actor))
		return;

	ReadyDefender(actor, false);
}

/* Whether leaving the fight to find a pack is worth what the walk costs the team

For everybody it is. For a medic it almost never is: he heals himself, three health a second and
six once he has been out of it a while, so the pack buys him what standing still would have bought
him anyway. What it costs is the medigun, for the length of a trip that the search range prices at
up to two thousand units.

Coaltown is why this is written down. The health pack there is in the house in the middle of the
map, so a medic who took eighty percent of a rocket left the front line and walked to the exact
spot he has now been reported standing in three times. Ammo goes with it: the medigun does not use
any and the syringe gun is what he holds when there is nobody to heal.

Below the critical ratio he goes anyway. A medic who dies takes the medigun with him for the rest
of the wave, which is worse than any trip. */
static bool ShouldLeaveToBePatchedUp(int client, float healthRatio)
{
	if (TF2_GetPlayerClass(client) != TFClass_Medic)
		return true;

	return healthRatio < tf_bot_health_critical_ratio.FloatValue;
}

/* A bot that is trying to walk somewhere and not getting anywhere

Every one of this mod's reported faults arrives looking the same. A build spot computed in mid-air,
a toolbox still set to the last building, two rules dragging a medic between two patients, a filter
that excluded every coin on the floor: five different causes, and from inside the game all five are
a bot standing still. Standing still is silent, so each of them was found by somebody playing and
noticing, one at a time, and the first guess at the cause was wrong about as often as it was right.

This does not fix any of them. It makes them loud. The one thing true of all of them is that the
bot wanted to be somewhere and stopped getting closer to it, so that is what is measured: past the
deadline his behaviour is thrown away and rebuilt, which is what the wave-start reset already does
to every bot, and the fact is printed so a test-bed run counts them instead of a player noticing.

Only while he is asking to go somewhere. An engineer stood at a finished nest and a sniper on his
perch are both motionless on purpose and neither is stuck.

Deferred by a frame, because throwing away the action stack from inside an action's own update is
freeing the thing that is running. */
#define STUCK_RADIUS	72.0
#define STUCK_TIME		12.0

static float m_vStuckOrigin[MAXPLAYERS + 1][3];
static float m_ctStuckDeadline[MAXPLAYERS + 1];
static int m_iStuckCount[MAXPLAYERS + 1];

//Where a bot kept getting stuck, and how many times running, so a wedge is told from a slow walk
static float m_vStuckWedge[MAXPLAYERS + 1][3];
static int m_iStuckWedgeCount[MAXPLAYERS + 1];

//Stucks at one spot before the bot is moved off it, and how far to look for somewhere it can stand
#define STUCK_WEDGE_GIVEUP	3

#define STUCK_WEDGE_SEARCH	400.0

int StuckCountOf(int client)
{
	return m_iStuckCount[client];
}

static void Frame_UnstickDefender(any client)
{
	if (!IsClientInGame(client) || !g_bIsDefenderBot[client] || !IsPlayerAlive(client))
		return;
	
	ResetIntentionInterface(client);
}

/* Whether this is a sniper who is nowhere near a spot and not on his way to one

Near a spot means he arrived and standing still is his job. Far from every one of them means he is
not doing the thing his seat exists for.

This used to require SniperLurk on his stack, which is exactly backwards for the fault Peppy
reported. His logs name it: bot 25 sat at 706 -2229 480 for 1097 samples, one position, stack
"MainAction < TacticalMonitor < ScenarioMonitor" and no SniperLurk at all, while bot 24 held the
lurk and moved through 234 positions. The stuck one has a stack, so the idle watchdog's empty-stack
test never fires either, and he is not pathing, so nothing else arms. He was invisible to every
check the watchdog had.

So the lurk is not required, in either direction: a rifle sniper parked far from every spot is the
fault whether ScenarioMonitor gave him a lurk that cannot finish or never gave him one. The reset
is what both need, and it is the same restart a custom primary causes by accident. See mvm-bj8. */
#define SNIPER_AT_SPOT	400.0

//How long a rifle sniper may go without either a lurk or a spot before he is restarted
#define SNIPER_STALL_TIME	20.0

static float m_ctSniperStallDeadline[MAXPLAYERS + 1];

/* Set once the stall is called, and never cleared while he is on this team

Resetting his intention is not enough on its own: Peppy watched the rescue fire on a sniper nobody
had walked into, and he stayed in spawn. So the mark outlives the reset and the break reads it, which
is the only place a bot can be handed an action. */
static bool m_bSniperStalled[MAXPLAYERS + 1];

//Whether this sniper has been caught stalling and should be sent to the front like every other class
bool IsSniperStalled(int client)
{
	return m_bSniperStalled[client];
}

void ClearSniperStall(int client)
{
	m_bSniperStalled[client] = false;
	m_ctSniperStallDeadline[client] = 0.0;
}

static bool IsLurkingNowhere(int actor, const char[] actions, const float here[3])
{
	if (TF2_GetPlayerClass(actor) != TFClass_Sniper || !HasSniperRifle(actor))
		return false;

	//A spot he is walking to is a spot he has not reached, and the walk is not the fault
	if (g_arrPluginBot[actor].bPathing)
		return false;

	//A lurk on the stack is the game doing its job, however far off he still is
	if (StrContains(actions, "SniperLurk") != -1)
		return false;

	ArrayList spots = g_arrMapConfig.adtSniperSpot;

	for (int i = 0; i < spots.Length; i++)
	{
		float spot[3]; spots.GetArray(i, spot);

		if (GetVectorDistance(here, spot) <= SNIPER_AT_SPOT)
			return false;
	}

	return true;
}

static void UpdateStuckWatchdog(int actor)
{
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	ILocomotion myLoco = myBot.GetLocomotionInterface();
	
	/* A bot with nothing on its stack is stuck too, and the watchdog used to miss it

	Measured on Mannhunt with v2.21.3: the engineer sat at 1014 885 274 for 45 seconds with no
	behaviour in 69 per cent of samples, and not one stuck line was written. He was not pathing, so
	wantsToBeElsewhere was false and the watchdog never armed. A bot that has stopped asking to go
	anywhere is exactly the one nobody is going to rescue, which is the opposite of what a watchdog
	is for.

	The reset below is what an empty stack needs anyway: ResetIntentionInterface is what gives a bot
	its behaviour back. */
	char actions[512]; ActionStackOf(actor, actions, sizeof(actions));

	bool noBehaviour = Feature(FEATURE_WATCH_IDLE_BOTS) && actions[0] == '\0';

	float here[3]; here = GetAbsOrigin(actor);

	/* A sniper spinning inside the game's own lurk is the one the watchdog missed

	bPathing is the mod's flag, set by the mod's own repaths, so a bot held inside a behaviour the
	game owns never sets it. CTFBotSniperLurk::Update calls Path::Compute directly, and when the
	spot is out of reach it does that every update: three stock snipers on Decoy killed the server
	that way, and the core names every frame of it. See mvm-bj8.

	Standing still is normal for a sniper who arrived, so it only counts against one who is nowhere
	near a spot the map offers. */
	/* A sniper's stall is timed on its own, because the watchdog's timer can be reset by a shove

	The watchdog arms only when a bot has not moved further than STUCK_RADIUS for STUCK_TIME, and
	teammates walking through a parked sniper push him further than that. Peppy watched exactly this:
	one stuck line, never a second, and the sniper still in spawn the whole time. The rescue read as
	holding when what it really did was get its timer reset by a shove.

	So this is timed from the last moment he was doing his job, not from the last moment he moved.
	Reset only when he reaches a spot or gets a lurk, and rescued when neither has happened for
	SNIPER_STALL_TIME. See mvm-bj8. */
	if (!IsLurkingNowhere(actor, actions, here))
	{
		m_ctSniperStallDeadline[actor] = GetGameTime() + SNIPER_STALL_TIME;
	}
	else if (Feature(FEATURE_WATCH_LURKING_SNIPERS) && GetGameTime() >= m_ctSniperStallDeadline[actor])
	{
		m_ctSniperStallDeadline[actor] = GetGameTime() + SNIPER_STALL_TIME;
		m_bSniperStalled[actor] = true;
		m_iStuckCount[actor]++;

		LogMessage("Stalled: %N (sniper) at %.0f %.0f %.0f for %.0fs, stall #%d, %s",
			actor, here[0], here[1], here[2], SNIPER_STALL_TIME, m_iStuckCount[actor],
			actions[0] == '\0' ? "no behaviour" : actions);

		RequestFrame(Frame_UnstickDefender, actor);

		return;
	}

	bool lurkingNowhere = false;

	
	bool wantsToBeElsewhere = g_arrPluginBot[actor].bPathing || myLoco.IsStuck() || noBehaviour || lurkingNowhere;

	if (!wantsToBeElsewhere || GetVectorDistance(here, m_vStuckOrigin[actor]) > STUCK_RADIUS)
	{
		m_vStuckOrigin[actor] = here;
		m_ctStuckDeadline[actor] = GetGameTime() + STUCK_TIME;
		
		return;
	}
	
	if (GetGameTime() < m_ctStuckDeadline[actor])
		return;
	
	/* Stuck again without having moved, which is a bot wedged rather than a bot walking slowly

	Measured on Mannworks with Mean Machines: the same engineer at 1014 885 274, eleven times, and
	the server killed by the watchdog on the same spot. Resetting his behaviour does not move him,
	so he comes back to the same wedge and asks for another path, which is the frame that grows.

	Counting here rather than inside the action he happens to be running. Anything armed in an
	action is armed again when the reset restarts it: two attempts at this fix were defeated that
	way, and the nest the engineer is aiming at churns underneath it. The watchdog is the one place
	the reset cannot reach. */
	if (GetVectorDistance(here, m_vStuckWedge[actor]) <= STUCK_RADIUS)
	{
		m_iStuckWedgeCount[actor]++;
	}
	else
	{
		m_vStuckWedge[actor] = here;
		m_iStuckWedgeCount[actor] = 1;
	}

	m_vStuckOrigin[actor] = here;
	m_ctStuckDeadline[actor] = GetGameTime() + STUCK_TIME;
	m_iStuckCount[actor]++;
	
	myLoco.ClearStuckStatus("Watchdog");
	g_arrPluginBot[actor].bPathing = false;
	
	PrintToServer("[defenderbots] stuck: %N (%s) at %.0f %.0f %.0f for %.0fs, stuck #%d, wedge #%d, %s",
		actor, g_sRawPlayerClassNames[TF2_GetPlayerClass(actor)],
		here[0], here[1], here[2], STUCK_TIME, m_iStuckCount[actor],
		m_iStuckWedgeCount[actor], actions[0] == '\0' ? "no behaviour" : actions);

	//The same line in the file, so a run can be counted rather than watched
	LogMessage("Stuck: %N (%s) at %.0f %.0f %.0f, stuck #%d, wedge #%d, %s",
		actor, g_sRawPlayerClassNames[TF2_GetPlayerClass(actor)],
		here[0], here[1], here[2], m_iStuckCount[actor], m_iStuckWedgeCount[actor],
		actions[0] == '\0' ? "no behaviour" : actions);
	
	if (m_iStuckWedgeCount[actor] >= STUCK_WEDGE_GIVEUP && MoveWedgedDefender(actor))
		return;

	/* A bot that cannot be moved is the one that kills the server

	The crash in k-kaneta's cores is not the wedge. It is what the wedge costs: a bot that cannot
	reach its goal asks for a path again every time it is reset, NavAreaBuildPath walks the mesh for
	an answer that does not exist, and the watchdog kills the server on the frame that takes.
	Twelve forced wedges produced no successful recovery at all, so a bot the recovery cannot help
	is not rare.

	Kicking is the last resort and it is the right one: the seat is refilled by the top-up path a
	moment later, with a bot that can walk. Leaving him costs the mission. His buildings go with him
	because a sentry with no owner is somebody else's problem. */

	RequestFrame(Frame_UnstickDefender, actor);
}

/* Somewhere out of the wedge, and false when nothing near him is far enough

A random point in his own area is what this used to take, and on Mannhunt that was the bug: he is
standing on valid nav and wedged in the geometry above it, so the nearest area is the one under his
feet and a point inside it lands back on him. Six give-ups in a row at 1014 885 274 and not one
move, which is mvm-ipf.

So his own area is tried first and only accepted when the point clears STUCK_RADIUS, then the areas
touching it. Bounded twice over: the four directions, and MOVE_WEDGED_TRIES points per area. */
#define MOVE_WEDGED_TRIES	8

static bool WedgeEscapePoint(CNavArea area, const float here[3], float destination[3])
{
	if (AreaEscapePoint(area, here, destination))
		return true;

	for (NavDirType dir = NORTH; dir < NUM_DIRECTIONS; dir++)
	{
		int count = area.GetAdjacentCount(dir);

		for (int i = 0; i < count; i++)
		{
			CNavArea next = area.GetAdjacentArea(dir, i);

			if (next != NULL_AREA && AreaEscapePoint(next, here, destination))
				return true;
		}
	}

	return false;
}

//A point in one area that is far enough from where he is wedged, tried a fixed number of times
static bool AreaEscapePoint(CNavArea area, const float here[3], float destination[3])
{
	for (int attempt = 0; attempt < MOVE_WEDGED_TRIES; attempt++)
	{
		float point[3]; CNavArea_GetRandomPoint(area, point);
		point[2] += 10.0;

		if (GetVectorDistance(here, point) > STUCK_RADIUS)
		{
			destination = point;
			return true;
		}
	}

	return false;
}

/* Put a wedged bot somewhere it can walk, when resetting it has stopped working

The same shape as the spawn recovery below and for the same reason, one step further along: that
one moves a bot that never left spawn, and this one moves a bot that left and then stopped. Both
beat leaving it where it is, because a bot that cannot move asks for a path every frame and the
watchdog kills the server on the answer.

Nearby rather than at the objective. He was going somewhere and the wedge is what stopped him, so
the least surprising thing is the same place minus the geometry he is inside. */
static bool MoveWedgedDefender(int client)
{
	float here[3]; here = GetAbsOrigin(client);

	CNavArea area = TheNavMesh.GetNearestNavArea(here, true, STUCK_WEDGE_SEARCH, false, true, TEAM_ANY);

	if (area == NULL_AREA)
	{
		LogMessage("Stuck: %N is wedged at %.0f %.0f %.0f with no nav area within %.0f, so nothing can be done",
			client, here[0], here[1], here[2], STUCK_WEDGE_SEARCH);
		return false;
	}

	float destination[3];

	if (DebugFaults_OldWedgeRecovery())
	{
		//The pre-2.21.3 behaviour, kept only so a run can measure what replacing it was worth
		CNavArea_GetRandomPoint(area, destination);
		destination[2] += 10.0;

		if (GetVectorDistance(here, destination) <= STUCK_RADIUS)
			return false;
	}
	else if (!WedgeEscapePoint(area, here, destination))
	{
		LogMessage("Stuck: %N is wedged at %.0f %.0f %.0f and every point in its area and the ones touching it is too close",
			client, here[0], here[1], here[2]);
		return false;
	}

	float stopped[3];
	TeleportEntity(client, destination, NULL_VECTOR, stopped);
	CBaseCombatCharacter(client).UpdateLastKnownArea();

	m_flRepathTime[client] = 0.0;
	m_iStuckWedgeCount[client] = 0;

	LogMessage("Stuck: %N was wedged at %.0f %.0f %.0f, moved to %.0f %.0f %.0f",
		client, here[0], here[1], here[2], destination[0], destination[1], destination[2]);

	return true;
}

public Action CTFBotTacticalMonitor_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (g_bIsDefenderBot[actor] == false)
		return Plugin_Continue;
	
	/* Before the returns below, not after them
	
	UpdateDefenderReadiness says of itself that it takes the ready away "every frame, wherever it
	came from", and the upgrade-zone branch below returns before it ever ran. A bot shopping at
	the station therefore had no readiness of its own: only the upgrade action's own exits set it,
	and the human-mirror never applied. Reported as an engineer beside spawn who never pressed F4
	while a player on RED had. The station is next to spawn on Decoy, which is where that bot was
	standing. */
	UpdateDefenderReadiness(actor);
	
	if (TF2_IsInUpgradeZone(actor) && ActionsManager.LookupEntityActionByName(actor, "DefenderUpgrade") != INVALID_ACTION)
	{
		TFClassType iClass = TF2_GetPlayerClass(actor);
		
		if (iClass == TFClass_DemoMan || iClass == TFClass_Scout)
		{
			CountdownTimer pOpportunisticTimer = CountdownTimer(GetOpportunisticTimer(actor));
			
			if (pOpportunisticTimer.Address)
			{
				//We don't do any of these things while upgrading
				pOpportunisticTimer.Start(interval);
			}
		}
		
		return Plugin_Continue;
	}
	
	if (!ShouldUseTeleporter(actor))
	{
		CountdownTimer pFindTeleporterTimer = CountdownTimer(action + 0x70);
		
		if (pFindTeleporterTimer.Address)
		{
			//Don't look for any nearby teleporters to use
			//This forces CTFBotTacticalMonitor::FindNearbyTeleporter to return NULL
			pFindTeleporterTimer.Start(interval);
		}
	}
	
	UpdateStuckWatchdog(actor);

	if (GameRules_GetRoundState() == RoundState_RoundRunning)
	{
		/* Nothing else matters while a buster is on top of the bot
		Above health and ammo on purpose: a bot walking to a health pack through the blast is a
		bot that arrives dead */
		if (CTFBotEvadeBuster_IsPossible(actor))
			return action.SuspendFor(CTFBotEvadeBuster(), "Sentry buster");

		UpdateScoutCombatJump(actor);

		/* The bombs already on the ground, blown when the blast pays
		Pressed here rather than anywhere in the attack behaviour, because it holds whatever the
		bot is doing: a Demoman walking to the next fight is still standing over his last one */
		if (ShouldDetonateStickies(actor))
			VS_PressAltFireButton(actor);

		/* Spies, in two pieces: what the bot can honestly say it has seen of one, and whether
		that is enough for it to go and frisk the teammate who was not there a moment ago */
		UpdateSpyIntel(actor);

		if (CTFBotSpyCheck_IsPossible(actor))
			return action.SuspendFor(CTFBotSpyCheck(), "Spy check");

		bool low_health = false;

		float health_ratio = HealthRatio(actor);
		
		if ((GetTimeSinceWeaponFired(actor) > 2.0 || TF2_GetPlayerClass(actor) == TFClass_Sniper) && health_ratio < tf_bot_health_critical_ratio.FloatValue)
			low_health = true;
		else if (health_ratio < tf_bot_health_ok_ratio.FloatValue)
			low_health = true;
		
		if (low_health && ShouldLeaveToBePatchedUp(actor, health_ratio) && CTFBotGetHealth_IsPossible(actor))
			return action.SuspendFor(CTFBotGetHealth(), "Getting health");
		else
		{
			int primary = GetPlayerWeaponSlot(actor, TFWeaponSlot_Primary);
			
			if (primary != -1 && TF2Util_GetWeaponID(primary) == TF_WEAPON_FLAMETHROWER && (TF2_IsCritBoosted(actor) || TF2_IsPlayerInCondition(actor, TFCond_CritMmmph)))
			{
				//Don't bother going for ammo while using crits unless our weapon has completely run out
				if (!HasAmmo(primary) && CTFBotGetAmmo_IsPossible(actor))
					return action.SuspendFor(CTFBotGetAmmo(), "Get ammo for crit");
			}
			else if (IsAmmoLow(actor) && ShouldLeaveToBePatchedUp(actor, health_ratio) && CTFBotGetAmmo_IsPossible(actor))
			{
				//Go for ammo when we're low and nearby packs are available
				return action.SuspendFor(CTFBotGetAmmo(), "Getting ammo");
			}
		}
	}
	
	return Plugin_Continue;
}

/* A Scout that keeps both feet on the ground

Nothing in this mod ever pressed IN_JUMP. Not in a fight, not to cross a gap, nowhere: a
play-test called the Scouts too easy to kill and that is the whole of the reason. A robot leads
a target moving in two dimensions perfectly well. The third one is most of what keeps a Scout
alive, and it costs him nothing: a scattergun is as accurate in the air as on the ground.

Only a Scout. Every other class is slower in the air than on it, and a Heavy who leaves the
ground has traded his aim for a hop */

//Close enough that the robot shooting back cannot miss unless it is made to
#define SCOUT_JUMP_THREAT_RANGE	900.0

//Slow enough to be standing still, whatever the bot thinks it is doing
#define SCOUT_JUMP_MIN_SPEED	100.0

/* The second jump, and why it is not every time

A Scout that always double jumps is as easy to lead as one that never does: the second jump lands
on the same beat every time. Seven times in ten is often enough to be the thing an aim expects and
irregular enough that expecting it is wrong.

The second jump goes the other way. Jumping twice in one direction is one long arc and a shooter
tracks it; jumping left and then right is two arcs with a corner in the middle, and the corner is
what a robot's aim cannot follow. Reported after the 1.3 play-test: he only ever single jumps */
#define SCOUT_DOUBLE_JUMP_CHANCE	70
#define SCOUT_DOUBLE_JUMP_DELAY		0.22
#define SCOUT_JUMP_STRAFE_TIME		0.35

static void UpdateScoutCombatJump(int client)
{
	if (TF2_GetPlayerClass(client) != TFClass_Scout)
		return;

	/* The second half of a double jump, which by definition happens off the ground, so this is
	before every check that wants the bot standing on something */
	if (m_flScoutDoubleJumpTime[client] > 0.0)
	{
		if (GetGameTime() < m_flScoutDoubleJumpTime[client])
			return;

		m_flScoutDoubleJumpTime[client] = 0.0;

		//Landed early. The air jump is gone and pressing it again only queues the next ground one
		if (GetEntityFlags(client) & FL_ONGROUND)
			return;

		g_arrExtraButtons[client].PressButtons(IN_JUMP | m_iScoutDoubleJumpSide[client], SCOUT_JUMP_STRAFE_TIME);

		return;
	}

	if (m_flNextJumpTime[client] > GetGameTime())
		return;

	//Already in the air, or held down by something that a jump will not get it out of
	if (!(GetEntityFlags(client) & FL_ONGROUND))
		return;

	if (TF2_IsPlayerInCondition(client, TFCond_Dazed) || TF2_IsPlayerInCondition(client, TFCond_Slowed))
		return;

	/* Standing still. A jump in place lands where it started and is a worse target for the second
	it is in the air, so the dodge is only a dodge while the bot is going somewhere */
	float velocity[3]; GetEntPropVector(client, Prop_Data, "m_vecVelocity", velocity);

	if (velocity[0] * velocity[0] + velocity[1] * velocity[1] < SCOUT_JUMP_MIN_SPEED * SCOUT_JUMP_MIN_SPEED)
		return;

	INextBot myBot = CBaseNPC_GetNextBotOfEntity(client);
	CKnownEntity threat = myBot.GetVisionInterface().GetPrimaryKnownThreat(false);

	if (threat == NULL_KNOWN_ENTITY || !threat.IsVisibleRecently())
		return;

	float threatOrigin[3]; threat.GetLastKnownPosition(threatOrigin);

	if (myBot.IsRangeGreaterThanEx(threatOrigin, SCOUT_JUMP_THREAT_RANGE))
		return;

	//Irregular on purpose: a jump on a fixed beat is as easy to lead as no jump at all
	m_flNextJumpTime[client] = GetGameTime() + GetRandomFloat(0.5, 1.2);

	int side = GetRandomInt(0, 1) ? IN_MOVELEFT : IN_MOVERIGHT;

	g_arrExtraButtons[client].PressButtons(IN_JUMP | side, SCOUT_JUMP_STRAFE_TIME);

	if (GetRandomInt(1, 100) > SCOUT_DOUBLE_JUMP_CHANCE)
		return;

	m_flScoutDoubleJumpTime[client] = GetGameTime() + SCOUT_DOUBLE_JUMP_DELAY;
	m_iScoutDoubleJumpSide[client] = (side == IN_MOVELEFT) ? IN_MOVERIGHT : IN_MOVELEFT;
}

public Action CTFBotScenarioMonitor_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (g_bIsDefenderBot[actor] == false)
		return Plugin_Continue;
	
	//Suspend for the action we desire
	//Once it has ended, we will return here and suspend for another one
	return GetDesiredBotAction(actor, action);
}

/* The charge and the resistance, whoever is doing the healing

Written for the game's heal action and called from the mod's own as well, because suspending an
action stops its update running and these two are not the part worth reimplementing.

The charge is pressed rather than set: the deploy belongs to the game, and this only ever asks
for it sooner than the game's own dying-patient rule would have. */
//How much more health another body needs before it is worth breaking a beam for
#define MEDIC_PATIENT_MARGIN	25

/* How long a medic call is worth answering for

The game gives one event and no state: nothing says the call is still wanted, and nothing says it
was met. So the call is a countdown rather than a flag, and it is short. Long enough to cross the
room he is being called across, short enough that a player who called once during a wave is not
still holding the beam a minute later.

Pressed again, it starts again, which is what a player who is still hurt does anyway. */
#define MEDIC_CALL_ANSWER_TIME	10.0

static float m_ctMedicCalled[MAXPLAYERS + 1];

//Written from the voice command hook in events.sp, read by the patient ranking below
void NoteMedicCall(int client)
{
	m_ctMedicCalled[client] = GetGameTime() + MEDIC_CALL_ANSWER_TIME;
}

void ForgetMedicCall(int client)
{
	m_ctMedicCalled[client] = 0.0;
}

static bool IsCallingForMedic(int client)
{
	return m_ctMedicCalled[client] > GetGameTime();
}

/* Which teammate a medigun is worth the most on
 *
 * A medigun is worth what the body in front of it is worth, so it belongs on the biggest one: the
 * Heavy, and failing that whoever has the most health to work with. Maximum health rather than a
 * class table, because that follows the health upgrades the team buys without anybody keeping a
 * list up to date.
 *
 * Where anybody is standing is deliberately not in this. The last version of this ranking had a
 * "nearby wins outright" bucket and that bucket was a fixed point: the medic stood next to whoever
 * he had, so whoever he had stayed the nearest, so he never moved. The walking is the game's job
 * again and the game is good at it. This only has to answer who.
 */
static int BiggestBody(int medic, int current = -1)
{
	int best = -1;
	int bestHealth = 0;
	bool bestIsHeavy = false;
	bool bestIsPlayer = false;
	bool bestIsCalling = false;
	int currentHealth = 0;
	bool currentIsHeavy = false;
	bool currentIsPlayer = false;
	bool currentIsCalling = false;
	bool currentStands = false;

	for (int i = 1; i <= MaxClients; i++)
	{
		if (i == medic || !IsClientInGame(i) || !IsPlayerAlive(i))
			continue;

		if (GetClientTeam(i) != GetClientTeam(medic))
			continue;

		//A medic healing a medic is two classes doing nothing
		if (TF2_GetPlayerClass(i) == TFClass_Medic)
			continue;

		bool isHeavy = TF2_GetPlayerClass(i) == TFClass_Heavy;
		int health = TF2Util_GetEntityMaxHealth(i);

		/* A player outranks every body, and a player who called outranks a player who did not

		Reported twice, by Cowser and by Peppy: pressing the call did nothing visible, because the
		ranking below only ever asked how big somebody was and a bot Heavy is bigger than any human.
		Ranked rather than special-cased, so the tie-break at the bottom still applies and the beam
		does not flicker between two players who both called. See mvm-w9b. */
		bool isPlayer = Feature(FEATURE_MEDIC_ANSWERS_CALL) && !IsTFBotPlayer(i);
		bool isCalling = isPlayer && IsCallingForMedic(i);

		/* Said as a class rather than left to the arithmetic
		
		Maximum health picks the Heavy out most of the time, and most of the time is the problem: a
		Soldier who has bought health and a Heavy who has not are a coin toss, and the beam moving
		off the Heavy for that is the beam on the wrong man. */
		if (i == current)
		{
			currentStands = true;
			currentHealth = health;
			currentIsHeavy = isHeavy;
			currentIsPlayer = isPlayer;
			currentIsCalling = isCalling;
		}

		bool better = best <= 0
			|| (isCalling && !bestIsCalling)
			|| (isCalling == bestIsCalling && isPlayer && !bestIsPlayer)
			|| (isCalling == bestIsCalling && isPlayer == bestIsPlayer && isHeavy && !bestIsHeavy)
			|| (isCalling == bestIsCalling && isPlayer == bestIsPlayer && isHeavy == bestIsHeavy && health > bestHealth);

		if (better)
		{
			best = i;
			bestHealth = health;
			bestIsHeavy = isHeavy;
			bestIsPlayer = isPlayer;
			bestIsCalling = isCalling;
		}
	}

	/* A patient he already has keeps the beam unless somebody is plainly worth more
	
	Maximum health follows the upgrades the team buys, and mid wave several bodies sit within a few
	points of each other, so the winner of this ranking flips between them every time it is asked.
	Measured on Bavarian Botbash wave 3: the beam was on somebody in 34 of 125 samples, and the
	patient changed almost every ask. A switch costs the walk to the new one and the healing that
	is not happening during it, so a tie keeps the man he has. */
	if (currentStands && best > 0 && best != current)
	{
		/* Both halves of the ask break the beam, and neither can make it flicker

		A call is worth the walk and the healing lost on the way, and a player is worth it over a
		bot whether he called or not, which is the whole of what Cowser and Peppy asked for.

		Neither can churn the way the margin below exists to stop. That churn comes of ranking on
		maximum health, which several bodies sit within a few points of, so the winner flips every
		ask. Whether somebody is a player does not flip, and a call runs down a clock and does not
		come back on its own. Once the beam has moved for either, the reason it moved is still true
		the next time this is asked. */
		bool answersCall = bestIsCalling && !currentIsCalling;
		bool betterSeat = bestIsCalling == currentIsCalling && bestIsPlayer && !currentIsPlayer;
		bool betterClass = bestIsCalling == currentIsCalling && bestIsPlayer == currentIsPlayer
			&& bestIsHeavy && !currentIsHeavy;
		bool betterBody = bestIsCalling == currentIsCalling && bestIsPlayer == currentIsPlayer
			&& bestIsHeavy == currentIsHeavy && bestHealth > currentHealth + MEDIC_PATIENT_MARGIN;

		if (!answersCall && !betterSeat && !betterClass && !betterBody)
			return current;
	}

	return best;
}


/* How often the game is told who its patient should be
 *
 * Every frame would be arguing with the action rather than nudging it, and the beam does not need
 * an answer more often than the team changes shape.
 */
#define MEDIC_PATIENT_INTERVAL	2.0

static float m_ctNextPatientNudge[MAXPLAYERS + 1];

/* Point the game's own heal action at the biggest body, from inside the action's own callback
 *
 * The mod used to answer this by replacing the whole action, which cost the medic his walking and
 * most of his output. The action is left alone now and only its patient is written.
 *
 * An earlier attempt at writing this field segfaulted the server, and this is deliberately narrower
 * than that one. It runs only from CTFBotMedicHeal_UpdatePost, so the action being written is the
 * action the game is running and not one looked up from somewhere else; the same offset is read in
 * the same callback every frame and has never faulted. The value is a checked, living, same team,
 * non medic client, and it is only written when it differs from what is already there.
 */
static void PointMedicAtBiggestBody(BehaviorAction action, int actor)
{
	if (m_ctNextPatientNudge[actor] > GetGameTime())
		return;

	m_ctNextPatientNudge[actor] = GetGameTime() + MEDIC_PATIENT_INTERVAL;

	int have = action.GetHandleEntity(ACTION_HEAL_PATIENT_OFFSET);
	int want = BiggestBody(actor, have);

	if (want <= 0)
		return;

	if (have == want)
		return;

	action.SetHandleEntity(ACTION_HEAL_PATIENT_OFFSET, want);
}

void MedicUberAndResist(int actor, int medigun, int patient)
{
	MedicProjectileShield(actor, patient);

	if (ShouldDeployUber(actor, medigun, patient))
		VS_PressAltFireButton(actor);
	
	if (patient <= 0 || GetMedigunType(medigun) != MEDIGUN_RESIST)
		return;
	
	int iResistType = GetResistType(medigun);
	int iLastDmgType = GetLastDamageType(patient);
	
	if (iLastDmgType & DMG_BULLET && iResistType != MEDIGUN_BULLET_RESIST)
		g_arrExtraButtons[actor].PressButtons(IN_RELOAD);
	else if (iLastDmgType & DMG_BLAST && iResistType != MEDIGUN_BLAST_RESIST)
		g_arrExtraButtons[actor].PressButtons(IN_RELOAD);
	else if (iLastDmgType & DMG_BURN && iResistType != MEDIGUN_FIRE_RESIST)
		g_arrExtraButtons[actor].PressButtons(IN_RELOAD);
}

public Action CTFBotMedicHeal_UpdatePost(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (g_bIsDefenderBot[actor] == false)
		return Plugin_Continue;
	
	if (result.type == CHANGE_TO)
	{
		//In mvm mode, medic bots will go for the flag when there's no patient available
		//Let's be smarter about it instead
		
		BehaviorAction resultingAction = result.action;
		char name[ACTION_NAME_LENGTH]; resultingAction.GetName(name);
		
		/* Nobody to heal, so he holds the hatch like the rest of them
		
		The game's answer here is to go and fetch the bomb, and the answer this had for that was to
		go and fight whatever is on it, which is the same walk into the middle of the map by a
		different name. Everything the team is defending comes to the hatch eventually. */
		if (StrEqual(name, "FetchFlag"))
			return action.SuspendFor(CTFBotGuardPoint(), "Nothing to heal, so hold the hatch");
	}
	
	int secondary = GetPlayerWeaponSlot(actor, TFWeaponSlot_Secondary);
	
	if (secondary == -1)
		return action.SuspendFor(CTFBotDefenderAttack(), "No medigun");
	
	if (CTFBotAttackUber_IsPossible(actor, secondary))
		return action.SuspendFor(CTFBotAttackUber(), "Seek uber");
	
	if (CTFBotMedicRevive_IsPossible(actor))
		return action.SuspendFor(CTFBotMedicRevive(), "Revive teammate");
	
	/* Only his own shopping comes before healing

	This used to be the whole break, so a medic spent the upgrade period walking after whoever he
	had picked: to the station, out of it, across the map, wherever that man went. Then it was the
	other extreme, and he walked to the front and stood there with a medigun and nobody in front of
	it until the wave started.

	Buying his upgrades is the one thing nobody else can do for him. After that the man he beams is
	walking to the front regardless, so following him is both the healing and the walk, and the
	team starts the wave with the overheal it spent the break earning. */
	if (GameRules_GetRoundState() == RoundState_BetweenRounds && !g_bShoppedThisBreak[actor])
		return Plugin_Continue;

	if (Feature(FEATURE_MEDIC_POCKETS_BIGGEST))
		PointMedicAtBiggestBody(action, actor);

	int myWeapon = BaseCombatCharacter_GetActiveWeapon(actor);

	if (myWeapon != -1 && TF2Util_GetWeaponID(myWeapon) == TF_WEAPON_MEDIGUN)
		MedicUberAndResist(actor, myWeapon, action.GetHandleEntity(ACTION_HEAL_PATIENT_OFFSET));
	
	return Plugin_Continue;
}

public Action CTFBotFetchFlag_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	if (g_bIsDefenderBot[actor] == false)
		return Plugin_Continue;
	
	return action.Done();
}

public Action CTFBotMvMEngineerIdle_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	if (g_bIsDefenderBot[actor] == false)
		return Plugin_Continue;
	
	return action.Done();
}

public Action CTFBotSniperLurk_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (g_bIsDefenderBot[actor] == false)
		return Plugin_Continue;
	
	if (!CanUsePrimayWeapon(actor))
	{
		//Where did my gun go?
		return action.SuspendFor(CTFBotDefenderAttack(), "Lost my rifle");
	}
	
	return Plugin_Continue;
}

public Action CTFBotSniperLurk_SelectMoreDangerousThreat(BehaviorAction action, INextBot nextbot, int entity, CKnownEntity threat1, CKnownEntity threat2, CKnownEntity& knownEntity)
{
	int me = action.Actor;
	
	if (g_bIsDefenderBot[me] == false)
		return Plugin_Continue;
	
	//Return NULL so the normal threat targetting happens
	knownEntity = NULL_KNOWN_ENTITY;
	
	int iThreat1 = threat1.GetEntity();
	
	if (BaseEntity_IsPlayer(iThreat1) && IsLineOfFireClearEntity(me, GetEyePosition(me), iThreat1))
	{
		int enemyWeapon = BaseCombatCharacter_GetActiveWeapon(iThreat1);
		
		if (enemyWeapon != -1)
		{
			int enemyWepID = TF2Util_GetWeaponID(enemyWeapon);
			
			if (WeaponID_IsSniperRifle(enemyWepID))
			{
				//This sniper ain't gonna snipe me
				knownEntity = threat1;
				return Plugin_Changed;
			}
			else if (enemyWepID == TF_WEAPON_MEDIGUN)
			{
				if (GetEntPropEnt(enemyWeapon, Prop_Send, "m_hHealingTarget") != -1 || GetEntPropFloat(enemyWeapon, Prop_Send, "m_flChargeLevel") >= 1.0)
				{
					//Healers should die first, ideally before they pop
					knownEntity = threat1;
					return Plugin_Changed;
				}
			}
		}
	}
	
	int iThreat2 = threat2.GetEntity();
	
	if (BaseEntity_IsPlayer(iThreat2) && IsLineOfFireClearEntity(me, GetEyePosition(me), iThreat2))
	{
		int enemyWeapon = BaseCombatCharacter_GetActiveWeapon(iThreat2);
		
		if (enemyWeapon != -1)
		{
			int enemyWepID = TF2Util_GetWeaponID(enemyWeapon);
			
			if (WeaponID_IsSniperRifle(enemyWepID))
			{
				knownEntity = threat2;
				return Plugin_Changed;
			}
			else if (enemyWepID == TF_WEAPON_MEDIGUN)
			{
				if (GetEntPropEnt(enemyWeapon, Prop_Send, "m_hHealingTarget") != -1 || GetEntPropFloat(enemyWeapon, Prop_Send, "m_flChargeLevel") >= 1.0)
				{
					knownEntity = threat2;
					return Plugin_Changed;
				}
			}
		}
	}
	
	return Plugin_Changed;
}

public Action CTFBotSpyLeaveSpawnRoom_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	if (g_bIsDefenderBot[actor] == false)
		return Plugin_Continue;
	
	return action.Done();
}

Action GetDesiredBotAction(int client, BehaviorAction action)
{
	RoundState state = GameRules_GetRoundState();
	if (state == RoundState_RoundRunning && g_bHasUpgraded[client])
		RecoverDefenderFromDisconnectedSpawn(client);
	
	if (state == RoundState_BetweenRounds)
	{
		if (CTFBotCollectMoney_IsPossible(client))
		{
			//Collect any leftover money that my team didn't collect
			return action.SuspendFor(CTFBotCollectMoney(), "Is possible");
		}
		else if (!TF2_IsInUpgradeZone(client) && !g_bShoppedThisBreak[client] && ActionsManager.LookupEntityActionByName(client, "DefenderMoveToFront") == INVALID_ACTION)
		{
			if (redbots_manager_bot_use_upgrades.BoolValue)
			{
				return action.SuspendFor(CTFBotGotoUpgrade(), "!IsInUpgradeZone && RoundState_BetweenRounds");
			}
			else
			{
				SetPlayerReady(client, true);
				return action.SuspendFor(CTFBotMoveToFront(), "Skip upgrading");
			}
		}
		else if ((ShouldTakeUpPosition(client) || IsSniperStalled(client)) && ActionsManager.LookupEntityActionByName(client, "DefenderMoveToFront") == INVALID_ACTION)
		{
			/* Shopping is finished, so go and stand where the robots come out

			Without this the break has nothing left to say to a bot that has bought its upgrades,
			and a behaviour that says nothing hands the bot back to the game, whose answer for a
			defender with no mission is to roam. Reported as the Heavy, the Medic and the Pyro
			wandering off before the wave, and found inside the middle house on Coaltown. */
			return action.SuspendFor(CTFBotMoveToFront(), "Shopping is done, so take up a position");
		}
	}
	else if (state == RoundState_RoundRunning)
	{
		if (redbots_manager_bot_use_upgrades.BoolValue && (g_bHasUpgraded[client] == false || ShouldUpgradeMidRound(client)) && !TF2_IsInUpgradeZone(client))
		{
			//We probably just joined in the middle of an active game, or we want to buy upgrades again right now
			g_iBuyUpgradesNumber[client] = 0;
			
			return action.SuspendFor(CTFBotGotoUpgrade(), "Buy upgrades now");
		}
		
		//NOTE: Health and ammo is moved to CTFBotTacticalMonitor_Update as it takes precedence over ScenarioMonitor
		
		switch (TF2_GetPlayerClass(client))
		{
			case TFClass_Medic:
			{
				//Medics automatically start healing
				return Plugin_Continue;
			}
			case TFClass_Scout:
			{
				if (CTFBotCollectMoney_IsPossible(client))
					return action.SuspendFor(CTFBotCollectMoney(), "Collecting money");
				else if (CTFBotMarkGiant_IsPossible(client))
					return action.SuspendFor(CTFBotMarkGiant(), "Marking giant");
				else if (CTFBotAttackTank_SelectTarget(client))
					return action.SuspendFor(CTFBotAttackTank(), "Scout: Attacking tank");
				else if (CTFBotDefenderAttack_SelectTarget(client))
					return action.SuspendFor(CTFBotDefenderAttack(), "Scout: Attacking robots");
			}
			case TFClass_Sniper:
			{
				/* A rifle sniper is given nothing here because Timer_PlayerSpawn is meant to have
				   given him his sniping behaviour already. When that did not happen he has nothing
				   for the whole wave, which is the running half of mvm-bj8: Peppy's sniper wandered
				   the map with an empty ScenarioMonitor and never fired a shot, stall #4, #5, #6.

				   So a sniper caught stalling fights like one who never had a rifle. */
				if (HasSniperRifle(client) && !IsSniperStalled(client))
				{
					//NOTE: we set the sniping behavior manually in Timer_PlayerSpawn
					return Plugin_Continue;
				}
				else
				{
					return action.SuspendFor(CTFBotDefenderAttack(), "Sniper Attacking robots");
				}
			}
			case TFClass_Engineer:
			{
				return action.SuspendFor(CTFBotMvMEngineerIdle(), "Engineer Start building");
			}
			case TFClass_Spy:
			{
				return action.SuspendFor(CTFBotSpyLurkMvM(), "Spy do be lurking");
			}
			case TFClass_Heavy:
			{
				if (CTFBotDefenderAttack_SelectTarget(client))
					return action.SuspendFor(CTFBotDefenderAttack(), "CTFBotAttack_IsPossible");
				else if (CTFBotAttackTank_SelectTarget(client))
					return action.SuspendFor(CTFBotAttackTank(), "Attacking tank");
				else if (CTFBotCollectNearMoney_SelectTarget(client))
					return action.SuspendFor(CTFBotCollectNearMoney(), "Nearby money");
			}
			case TFClass_DemoMan:
			{
				if (CTFBotAttackTank_SelectTarget(client))
					return action.SuspendFor(CTFBotAttackTank(), "Attacking tank");
				else if (CTFBotDefenderAttack_SelectTarget(client))
					return action.SuspendFor(CTFBotDefenderAttack(), "CTFBotAttack_IsPossible");
				else if (CTFBotStickyTrap_IsPossible(client))
					return action.SuspendFor(CTFBotStickyTrap(), "Nothing to fight, so lay a trap");
				else if (CTFBotCollectNearMoney_SelectTarget(client))
					return action.SuspendFor(CTFBotCollectNearMoney(), "Nearby money");
			}
			case TFClass_Soldier, TFClass_Pyro:
			{
				if (CTFBotAttackTank_SelectTarget(client))
					return action.SuspendFor(CTFBotAttackTank(), "Attacking tank");
				else if (CTFBotDefenderAttack_SelectTarget(client))
					return action.SuspendFor(CTFBotDefenderAttack(), "CTFBotAttack_IsPossible");
				else if (CTFBotCollectNearMoney_SelectTarget(client))
					return action.SuspendFor(CTFBotCollectNearMoney(), "Nearby money");
			}
		}
		
		/* Nothing to shoot, no tank, no money on the floor: go and hold the hatch

		Every branch above needs something to already be happening, and when none of them does the
		bot was handed back to the game with no instruction at all. What that looks like from
		inside the game is a Pyro standing next to the spawn door, not moving, for as long as no
		robot walks into his view. Reported exactly that way, and it is worst right after a
		respawn, because that is when a defender is furthest from anything worth doing.

		CTFBotGuardPoint has been in this repository the whole time and nothing has ever called it.
		It walks to the ground around the hatch and stays there, which is the answer to "there is
		nothing to do yet" on a map where everything eventually comes to the hatch, and it hands
		the bot back the moment there is something to fight. */
		if (CanGuardTheHatch(client))
			return action.SuspendFor(CTFBotGuardPoint(), "Nothing to do, so hold the hatch");
	}
	
	return Plugin_Continue;
}

/* Whether this bot has nowhere of its own to wait out the break

The Engineer has a nest to build, the Spy has somewhere to lurk and the Sniper has a perch, and
all three get there under their own behaviour.

So does the Medic, once he has done his shopping: his place is beside the man he is beaming, and
that man is walking to the front anyway. Sending him there himself made him stand on the front
line with a medigun and nobody in front of it, which is a medic doing nothing for the length of the
break when he could be handing out overheal the whole way there. */
/* Whether the break ends with this one walking to where the robots come out
 *
 * The three that say no have somewhere else to be: the engineer has his nest, the spy is lurking
 * and the sniper with a rifle has his spot. The medic used to be a fourth, on the reasoning that
 * he follows a patient, and between rounds he has no patient to follow: he heals nobody, so
 * nothing suspends for him and he stood where the last wave left him for the whole break. His
 * patient is walking to the front, so that is where he goes. */
static bool ShouldTakeUpPosition(int client)
{
	switch (TF2_GetPlayerClass(client))
	{
		case TFClass_Engineer, TFClass_Spy:
			return false;
		
		case TFClass_Sniper:
			return !HasSniperRifle(client);
	}

	return true;
}

/* Whether this bot is one of the ones that should fall back to holding the hatch

The Engineer has his nest, the Spy is lurking, the Sniper has his spot and the Medic follows
somebody: all four already have somewhere to be when there is nothing to shoot. The Scout is
collecting money, which is his job and takes him wherever the money is. What is left is the three
that fight, and standing still is what they were doing instead. */
static bool CanGuardTheHatch(int client)
{
	switch (TF2_GetPlayerClass(client))
	{
		case TFClass_Soldier, TFClass_Pyro, TFClass_DemoMan, TFClass_Heavy:
			return true;
	}

	return false;
}

Action GetUpgradePostAction(int client, BehaviorAction action)
{
	if (GameRules_GetRoundState() == RoundState_BetweenRounds)
	{
		if (TF2_GetPlayerClass(client) == TFClass_Engineer)
			return action.ChangeTo(CTFBotMvMEngineerIdle(), "Start building");
		else if (TF2_GetPlayerClass(client) == TFClass_Medic)
			return action.Done("Start heal mission");
		else if (TF2_GetPlayerClass(client) == TFClass_Spy)
			return action.ChangeTo(CTFBotSpyLurkMvM(), "Start spy lurking");
		else if (HasSniperRifle(client))
			return action.Done("Start lurking");
		else
			return action.ChangeTo(CTFBotMoveToFront(), "Finished upgrading; Move to front and press F4");
	}
	
	/* The round's probably already running
	CTFBotScenarioMonitor_Update will assign the appropriate task */
	return action.Done("I finished upgrading");
}

public bool NextBotTraceFilterIgnoreActors(int entity, int contentsMask, any iExclude)
{
	char class[64]; GetEntityClassname(entity, class, sizeof(class));
	
	if (StrEqual(class, "entity_medigun_shield"))
		return false;
	else if (StrEqual(class, "func_respawnroomvisualizer"))
		return false;
	else if (StrContains(class, "tf_projectile_", false) != -1)
		return false;
	else if (StrContains(class, "obj_", false) != -1)
		return false;
	else if (StrEqual(class, "entity_revive_marker"))
		return false;
	else if (StrEqual(class, "tank_boss"))
		return false;
	else if (StrEqual(class, "func_forcefield"))
		return false;
	
	return !CBaseEntity(entity).IsCombatCharacter();
}

float GetDesiredPathLookAheadRange(int client)
{
	return tf_bot_path_lookahead_range.FloatValue * BaseAnimating_GetModelScale(client);
}

bool IsPathToVectorPossible(int bot_entidx, const float vec[3], float &length = -1.0)
{
	CBaseCombatCharacter(bot_entidx).UpdateLastKnownArea();
	
	PathFollower temp_path = PathFollower(_, Path_FilterIgnoreActors, Path_FilterOnlyActors);
	
	bool success = temp_path.ComputeToPos(CBaseNPC_GetNextBotOfEntity(bot_entidx), vec);
	
	length = temp_path.GetLength();
	
	temp_path.Destroy();
	
	return success;
}


bool IsAmmoLow(int client)
{
	int primary = GetPlayerWeaponSlot(client, TFWeaponSlot_Primary);

	if (IsValidEntity(primary) && !HasAmmo(primary))
		return true;
	
	int myWeapon = BaseCombatCharacter_GetActiveWeapon(client);
	
	if (myWeapon != -1 && TF2Util_GetWeaponID(myWeapon) != TF_WEAPON_WRENCH)
	{
		if (!IsMeleeWeapon(myWeapon))
		{
			float flAmmoRation = float(BaseCombatCharacter_GetAmmoCount(client, TF_AMMO_PRIMARY)) / float(TF2Util_GetPlayerMaxAmmo(client, TF_AMMO_PRIMARY));
			return flAmmoRation < 0.2;
		}
		
		return false;
	}
	
	return BaseCombatCharacter_GetAmmoCount(client, TF_AMMO_METAL) <= 0;
}

bool IsAmmoFull(int client)
{
	bool isPrimaryFull = BaseCombatCharacter_GetAmmoCount(client, TF_AMMO_PRIMARY) >= TF2Util_GetPlayerMaxAmmo(client, TF_AMMO_PRIMARY);
	bool isSecondaryFull = BaseCombatCharacter_GetAmmoCount(client, TF_AMMO_SECONDARY) >= TF2Util_GetPlayerMaxAmmo(client, TF_AMMO_SECONDARY);
	
	if (TF2_GetPlayerClass(client) == TFClass_Engineer)
	{
		//In addition, I want some metal as well
		return BaseCombatCharacter_GetAmmoCount(client, TF_AMMO_METAL) >= 200 && isPrimaryFull && isSecondaryFull;
	}
	
	return isPrimaryFull && isSecondaryFull;
}

void ResetIntentionInterface(int bot_entidx)
{
	CBaseNPC_GetNextBotOfEntity(bot_entidx).GetIntentionInterface().Reset();
}

void UpdateLookAroundForEnemies(int client, bool bVal)
{
	SetLookingAroundForEnemies(client, bVal);
}

bool IsCombatWeapon(int client, int weapon)
{
	if (!IsValidEntity(weapon))
		weapon = BaseCombatCharacter_GetActiveWeapon(client);
	
	if (IsValidEntity(weapon))
	{
		switch (TF2Util_GetWeaponID(weapon))
		{
			case TF_WEAPON_MEDIGUN, TF_WEAPON_PDA, TF_WEAPON_PDA_ENGINEER_BUILD, TF_WEAPON_PDA_ENGINEER_DESTROY, TF_WEAPON_PDA_SPY, TF_WEAPON_BUILDER, TF_WEAPON_DISPENSER, TF_WEAPON_INVIS, TF_WEAPON_LUNCHBOX, TF_WEAPON_BUFF_ITEM, TF_WEAPON_PUMPKIN_BOMB:
			{
				return false;
			}
		}
    }
	
	return true;
}

float GetDesiredAttackRange(int client)
{
	int weapon = BaseCombatCharacter_GetActiveWeapon(client);
	
	if (weapon < 1)
		return 0.0;
	
	//The loadout the server handed out is more specific than the weapon's ID
	float tunedDesired, tunedMax;
	
	if (GetTunedWeaponRanges(weapon, tunedDesired, tunedMax))
		return tunedDesired;
	
	int weaponID = TF2Util_GetWeaponID(weapon);
	
	if (weaponID == TF_WEAPON_KNIFE)
		return 70.0;
	
	if (IsMeleeWeapon(weapon) || weaponID == TF_WEAPON_FLAMETHROWER)
		return 100.0;
	
	/* A Pyro closes whatever is in his hands, because the flamethrower is the only reason he is here
	
	Reported from play: the Pyro stands a long way back and never gets to use the Phlogistinator.
	The weapon is chosen by range and the range he closes to is chosen by the weapon, and those two
	disagreed. Past seven hundred and fifty units he pulls the shotgun; holding the shotgun he
	settles at shotgun range; and at shotgun range he is inside seven hundred and fifty again, so
	he swaps back, walks in, swaps out. What that produces is a Pyro parked between the two
	distances holding the wrong gun.
	
	So the distance he closes to is the flamethrower's, always. The shotgun is what he shoots with
	on the way in, not a reason to stop there. */
	if (TF2_GetPlayerClass(client) == TFClass_Pyro)
	{
		int flamethrower = GetPlayerWeaponSlot(client, TFWeaponSlot_Primary);
		
		if (flamethrower != -1 && TF2Util_GetWeaponID(flamethrower) == TF_WEAPON_FLAMETHROWER)
			return 100.0;
	}
	
	if (WeaponID_IsSniperRifle(weaponID))
		return FLT_MAX;
	
	/* How far out a rocket is worth firing, which is not as far as it will travel
	
	Twelve hundred and fifty units is over a second of flight, and everything a defender shoots at
	is walking. The blast covers a hundred and forty six of them, so a robot at that range has left
	the splash before the rocket arrives unless it is walking straight at him.
	
	The Demoman fights the same shape of weapon at six hundred and out-damages the Soldier by half
	again, on a weapon with a slower projectile and an arc on top. That is the comparison this is
	drawn from. */
	if (weaponID == TF_WEAPON_ROCKETLAUNCHER)
		return Feature(FEATURE_SOLDIER_CLOSES_IN) ? SOLDIER_ROCKET_SETTLE : 1250.0;
	
	//The same answer as the Iron Bomber, which is the launcher this loadout actually hands out
	if (weaponID == TF_WEAPON_GRENADELAUNCHER)
		return DEMO_PIPE_SETTLE;
	
	return 500.0;
}

//Extension of the original function
bool OpportunisticallyUseWeaponAbilities(int client, int activeWeapon, INextBot bot, const CKnownEntity threat)
{
	if (threat == NULL_KNOWN_ENTITY)
		return false;
	
	if (activeWeapon == -1)
		return false;
	
	int weaponID = TF2Util_GetWeaponID(activeWeapon);
	
	//Hitmans Heatmaker
	if (weaponID == TF_WEAPON_SNIPERRIFLE && TF2_IsPlayerInCondition(client, TFCond_Slowed) && threat.IsVisibleRecently())
	{
		if (TF2_GetRageMeter(client) >= 0.0 && !TF2_IsRageDraining(client))
		{
			g_arrExtraButtons[client].PressButtons(IN_RELOAD);
			return true;
		}
	}
	
	int iThreat = threat.GetEntity();
	
	//Phlogistinator
	if (weaponID == TF_WEAPON_FLAMETHROWER && bot.IsRangeLessThan(iThreat, FLAMETHROWER_REACH_RANGE) && !TF2_IsCritBoosted(client))
	{
		if (TF2_GetRageMeter(client) >= 100.0 && !TF2_IsRageDraining(client))
		{
			VS_PressAltFireButton(client);
			return true;
		}
	}
	
	if (weaponID == TF_WEAPON_MINIGUN && BaseEntity_IsPlayer(iThreat) && TF2_GetRageMeter(client) >= 100.0)
	{
		if (TF2_HasTheFlag(iThreat))
		{
			float vThreatOrigin[3]; GetClientAbsOrigin(iThreat, vThreatOrigin);
			
			if (GetVectorDistance(vThreatOrigin, GetBombHatchPosition()) <= 100.0)
			{
				VS_PressSpecialFireButton(client);
				return true;
			}
		}
	}
	
	return false;
}

bool OpportunisticallyUsePowerupBottle(int client, int activeWeapon, INextBot bot, const CKnownEntity threat)
{
	if (m_flNextBottleUseTime[client] > GetGameTime())
		return false;
	
	int bottle = PowerupBottleOf(client);
	
	if (bottle == -1)
		return false;
	
	if (PowerupBottle_GetNumCharges(bottle) < 1)
		return false;
	
	switch (PowerupBottle_GetType(bottle))
	{
		case POWERUP_BOTTLE_CRITBOOST:
		{
			//Can't do anything useful without a weapon
			if (activeWeapon == -1)
				return false;
			
			//No threat to tactually use it against
			if (threat == NULL_KNOWN_ENTITY)
				return false;
			
			//Medic would rather share this than use it for himself
			if (TF2_GetPlayerClass(client) == TFClass_Medic)
				return false;
			
			//Already have crits
			if (TF2_IsCritBoosted(client) || TF2_IsPlayerInCondition(client, TFCond_CritMmmph))
				return false;
			
			int iThreat = threat.GetEntity();
			
			if (!IsLineOfFireClearEntity(client, GetEyePosition(client), iThreat))
				return false;
			
			int weaponID = TF2Util_GetWeaponID(activeWeapon);
			
			if (weaponID == TF_WEAPON_FLAMETHROWER && bot.IsRangeGreaterThan(iThreat, FLAMETHROWER_REACH_RANGE))
				return false;
			
			if (weaponID == TF_WEAPON_FLAME_BALL && bot.IsRangeGreaterThan(iThreat, FLAMEBALL_REACH_RANGE))
				return false;
			
			if (IsMeleeWeapon(activeWeapon) && bot.IsRangeGreaterThan(iThreat, 100.0))
				return false;
			
			if (BaseEntity_IsPlayer(iThreat))
			{
				/* So basically here we determine based on a few factors
				if our threat is giant and has a lot of health, they're probably a boss
				if we're close to failing and they have a lot of health left, we want to kill them fast
				i really want this to be done better, but we probably need people that actually know what the optimal use of this canteen is */
				if ((TF2_IsMiniBoss(iThreat) && GetClientHealth(iThreat) > 5000) || (IsFailureImminent(client) && GetClientHealth(iThreat) > 2000))
				{
					UseActionSlotItem(client);
					return true;
				}
			}
			else if (IsBaseBoss(iThreat) && BaseEntity_GetHealth(iThreat) > 1000)
			{
				//Crit against the tank
				UseActionSlotItem(client);
				return true;
			}
		}
		case POWERUP_BOTTLE_UBERCHARGE:
		{
			//I'm invincible already
			if (TF2_IsInvulnerable(client))
				return false;
			
			//Only when there's a threat nearby, otherwise we could just go heal ourselves
			if (!threat || !threat.IsVisibleRecently())
				return false;
			
			float healthRatio = float(GetClientHealth(client)) / float(TEMP_GetPlayerMaxHealth(client));
			
			if (healthRatio < tf_bot_health_critical_ratio.FloatValue)
			{
				//I'm about to die
				UseActionSlotItem(client);
				m_flNextBottleUseTime[client] = GetGameTime() + GetRandomFloat(10.0, 30.0);
				return true;
			}
			
			if (TF2_IsPlayerInCondition(client, TFCond_Gas))
			{
				//This gas might be explosive
				UseActionSlotItem(client);
				m_flNextBottleUseTime[client] = GetGameTime() + GetRandomFloat(20.0, 30.0);
				return true;
			}
		}
		case POWERUP_BOTTLE_RECALL:
		{
			//TODO: medic can't share this, but he could use it for himself in an attempt to defend the hatch
			if (TF2_GetPlayerClass(client) == TFClass_Medic)
				return false;
			
			//TODO: engineer should probably only uses this if his sentry was destroyed
			if (TF2_GetPlayerClass(client) == TFClass_Engineer)
				return false;
			
			//We're busy going for the tank
			if (ActionsManager.LookupEntityActionByName(client, "DefenderAttackTank") != INVALID_ACTION)
				return false;
			
			float myPosition[3]; myPosition = WorldSpaceCenter(client);
			
			//I'm already in my spawn room
			if (TF2Util_IsPointInRespawnRoom(myPosition, client, true))
				return false;
			
			float hatchPosition[3]; hatchPosition = GetBombHatchPosition();
			
			//We're already close enough to the hatch
			if (GetVectorDistance(myPosition, hatchPosition) <= 1000.0)
				return false;
			
			int flag = FindBombNearestToHatch();
			
			//No bomb active
			if (flag == -1)
				return false;
			
			float bombPosition[3]; bombPosition = WorldSpaceCenter(flag);
			
			//Bomb is far and not a threat
			if (GetVectorDistance(bombPosition, hatchPosition) > BOMB_HATCH_RANGE_CRITICAL)
				return false;
			
			int closestToHatch = FindBotNearestToBombNearestToHatch(client);
			
			//No robot near the bomb close to the hatch
			if (closestToHatch == -1)
				return false;
			
			float threatPosition[3]; GetClientAbsOrigin(closestToHatch, threatPosition);
			
			//Nearest robot isn't that close to the bomb
			if (GetVectorDistance(threatPosition, bombPosition) > 800.0)
				return false;
			
			//We are already close enough to deal with it
			if (GetVectorDistance(myPosition, threatPosition) <= 500.0)
				return false;
			
			UseActionSlotItem(client);
			return true;
		}
		case POWERUP_BOTTLE_REFILL_AMMO:
		{
			int primary = GetPlayerWeaponSlot(client, TFWeaponSlot_Primary);
			
			if (primary != -1 && !HasAmmo(primary))
			{
				//I got no ammo
				UseActionSlotItem(client);
				return true;
			}
		}
		case POWERUP_BOTTLE_BUILDINGS_INSTANT_UPGRADE:
		{
			//TODO
		}
	}
	
	return false;
}

/* Whether a weapon that fires off a meter has enough of it to fire, which HasAmmo cannot say

HasAmmo is CBaseCombatWeapon::HasAmmo, and a weapon carrying a meter instead of a clip has no ammo
to run out of, so it answers yes for ever. "Always throw gas whenever HasAmmo" therefore reads as
"always hold the jar": the Pyro equipped one it could not throw, on every tick, and never picked
the flamethrower back up. Four runs in six of Decoy ended on the watchdog line because of it, and
the loadout has carried a shotgun in that slot ever since to keep away from it.

The two kinds of meter are kept in different places, which is why this is not one line. Jarate,
Mad Milk and the Cleaver refill on a clock the weapon itself carries. The Gas Passer refills out
of damage dealt, and the game keeps that on the player, per loadout slot, rather than on the
weapon.

Anything else is asked the question it was always asked. */
static bool IsThrowableReady(int client, int weapon)
{
	switch (TF2Util_GetWeaponID(weapon))
	{
		case TF_WEAPON_JAR_GAS:
			return GetEntPropFloat(client, Prop_Send, "m_flItemChargeMeter", TF_LOADOUT_SLOT_SECONDARY) >= 100.0;
		
		case TF_WEAPON_JAR, TF_WEAPON_JAR_MILK, TF_WEAPON_CLEAVER:
			return GetEntPropFloat(weapon, Prop_Send, "m_flEffectBarRegenTime") <= GetGameTime();
	}
	
	return HasAmmo(weapon);
}

void EquipBestWeaponForThreat(int client, const CKnownEntity threat)
{
	//Don't care about any weapon restrictions here
	
	int primary = GetPlayerWeaponSlot(client, TFWeaponSlot_Primary);
	
	if (!IsCombatWeapon(client, primary))
		primary = -1;
	
	int secondary = GetPlayerWeaponSlot(client, TFWeaponSlot_Secondary);
	
	if (!IsCombatWeapon(client, secondary))
		secondary = -1;
	
	//Don't care about mvm-specific rules here
	
	int melee = GetPlayerWeaponSlot(client, TFWeaponSlot_Melee);
	
	if (!IsCombatWeapon(client, melee))
		melee = -1;
	
	int gun = -1;
	
	if (primary != -1)
		gun = primary;
	else if (secondary != -1)
		gun = secondary;
	else
		gun = melee;
	
	if (threat == NULL_KNOWN_ENTITY || !threat.WasEverVisible() || threat.GetTimeSinceLastSeen() > 5.0)
	{
		if (gun != -1)
			TF2Util_SetPlayerActiveWeapon(client, gun);
		
		return;
	}
	
	if (BaseCombatCharacter_GetAmmoCount(client, TF_AMMO_PRIMARY) <= 0)
		primary = -1;
	
	/* TFWeaponSlot_Secondary is 1 and TF_AMMO_SECONDARY is 2, so this read primary ammo and
	retired the secondary along with the primary. A Heavy whose minigun ran dry was left with
	no shotgun, an Engineer with no pistol, a Sniper with no SMG */
	if (BaseCombatCharacter_GetAmmoCount(client, TF_AMMO_SECONDARY) <= 0)
		secondary = -1;
	
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(client);
	int threatEnt = threat.GetEntity();
	
	switch (TF2_GetPlayerClass(client))
	{
		case TFClass_DemoMan:
		{
			/* The stickybomb launcher, which this switch used to pass over in silence
			A Demoman was listed with the classes that only ever want their primary, so the
			launcher came out when the pipes ran dry and at no other time */
			float threatOrigin[3]; threat.GetLastKnownPosition(threatOrigin);
			float myOrigin[3]; GetClientAbsOrigin(client, myOrigin);
			float threatRange = GetVectorDistance(myOrigin, threatOrigin);

			bool wantSticky = secondary != -1 && ShouldUseStickyLauncher(client, secondary, threatEnt, threatRange);

			/* An empty launcher is not the weapon that lands, whatever the rule above says
			
			It matters more now the launcher is what he reaches for by default: holding eight
			spent bombs through the reload is a second and a half of nothing with a loaded grenade
			launcher in the other hand. */
			if (wantSticky && Clip1(secondary) > 0)
				gun = secondary;
			else if (gun != -1 && !Clip1(gun) && secondary != -1 && Clip1(secondary))
				gun = secondary;
		}
		case TFClass_Heavy, TFClass_Spy, TFClass_Engineer:
		{
			//Uses primary
		}
		case TFClass_Medic:
		{
			/* The medigun is the weapon, and the syringe gun is what he holds when there is
			nobody to point it at

			A medic is a class whose damage is somebody else's. Every visible robot used to pull
			him onto his primary, which drops the beam, stops the heal and stops the charge
			building, in exchange for a syringe gun nobody is frightened of. Reported after the
			1.3 play-test: "the Medics always keep using their Syringe Guns" */
			if (secondary != -1 && MedicHasPatient(client, secondary))
				gun = secondary;
		}
		case TFClass_Scout:
		{
			if (secondary != -1)
			{
				int weaponID = TF2Util_GetWeaponID(secondary);
				
				if ((weaponID == TF_WEAPON_JAR_MILK || weaponID == TF_WEAPON_CLEAVER) && IsThrowableReady(client, secondary) && BaseEntity_IsPlayer(threatEnt) && !TF2_IsInvulnerable(threatEnt))
				{
					//Always throw milk at them if we can
					gun = secondary;
				}
				else if (gun != -1 && !Clip1(gun))
				{
					gun = secondary;
				}
			}
		}
		case TFClass_Soldier:
		{
			/* Handing him the shotgun inside his own blast was tried, with the aim change in
			botaim, and the pair lost: damage 16890 to 10886 over six waves on Decoy. A rocket
			that hurts him also kills what is standing on him, and a shotgun does not. */
			if (gun != -1 && !Clip1(gun))
			{
				/* NOTE: we do not want to switch off the rocket launcher against uber threats or else we will conflctingly ignore them
				on and off due to the detour callback that we do at DHookCallback_IsIgnored_Pre */
				if (secondary != -1 && Clip1(secondary) && (!BaseEntity_IsPlayer(threatEnt) || !TF2_IsInvulnerable(threatEnt)))
				{
					const float closeSoldierRange = 500.0;
					
					float lastKnownPos[3]; threat.GetLastKnownPosition(lastKnownPos);
					
					if (myBot.IsRangeLessThanEx(lastKnownPos, closeSoldierRange))
						gun = secondary;
				}
			}
		}
		case TFClass_Sniper:
		{
			if (secondary != -1 && TF2Util_GetWeaponID(secondary) == TF_WEAPON_JAR && IsThrowableReady(client, secondary) && BaseEntity_IsPlayer(threatEnt) && !TF2_IsInvulnerable(threatEnt))
			{
				//Always throw pee at them if we can
				gun = secondary;
			}
			else if (primary != -1 && TF2Util_GetWeaponID(primary) == TF_WEAPON_COMPOUND_BOW)
			{
				//Always use the bow, unless it has no ammo
				gun = primary;
			}
			else
			{
				const float closeSniperRange = 750.0;
				
				float lastKnownPos[3]; threat.GetLastKnownPosition(lastKnownPos);
				
				if (secondary != -1 && myBot.IsRangeLessThanEx(lastKnownPos, closeSniperRange))
					gun = secondary;
			}
		}
		case TFClass_Pyro:
		{
			if (secondary != -1 && TF2Util_GetWeaponID(secondary) == TF_WEAPON_JAR_GAS && IsThrowableReady(client, secondary) && BaseEntity_IsPlayer(threatEnt) && !TF2_IsInvulnerable(threatEnt))
			{
				//Always throw gas
				gun = secondary;
			}
			else
			{
				const float flameRange = 750.0;
				
				float lastKnownPos[3]; threat.GetLastKnownPosition(lastKnownPos);
				
				if (secondary != -1 && myBot.IsRangeGreaterThanEx(lastKnownPos, flameRange))
					gun = secondary;
				
				if (BaseEntity_IsPlayer(threatEnt))
				{
					TFClassType threatClass = TF2_GetPlayerClass(threatEnt);
					
					if (threatClass == TFClass_Soldier || threatClass == TFClass_DemoMan)
						gun = primary;
				}
			}
		}
	}
	
	/* Whatever the rules above picked, never walk at a robot holding something that cannot
	fire. The per class cases only ever choose between weapons; this is the one place that
	asks whether the choice can still shoot. Melee always can */
	if (gun != -1 && !IsMeleeWeapon(gun) && !HasAmmo(gun))
	{
		if (primary != -1 && HasAmmo(primary))
			gun = primary;
		else if (secondary != -1 && HasAmmo(secondary))
			gun = secondary;
		else if (melee != -1)
			gun = melee;
	}
	
	if (gun != -1)
		TF2Util_SetPlayerActiveWeapon(client, gun);
}

/* The Medic behind whatever we just picked, when there is one

Shooting the patient of a Quick-Fix Medic is shooting through the heal rate, which is the whole
problem with that pair. Every path out of the threat selection goes through here now: it used to
sit on the last one only, so a bot that could see the giant and not the Medic locked onto the
giant and stayed there. Reported on Bavarian Botbash */
static CKnownEntity HealerOrThreat(INextBot bot, const CKnownEntity threat)
{
	if (!threat || !BaseEntity_IsPlayer(threat.GetEntity()))
		return threat;
	
	return GetHealerOfThreat(bot, threat);
}

CKnownEntity GetHealerOfThreat(INextBot bot, const CKnownEntity threat)
{
	if (!threat)
		return NULL_KNOWN_ENTITY;
	
	int playerThreat = threat.GetEntity();
	
	for (int i = 0; i < TF2_GetNumHealers(playerThreat); i++)
	{
		int playerHealer = TF2Util_GetPlayerHealer(playerThreat, i);
		
		if (playerHealer != -1 && BaseEntity_IsPlayer(playerHealer))
		{
			CKnownEntity knownHealer = bot.GetVisionInterface().GetKnown(playerHealer);
			
			if (knownHealer && knownHealer.IsVisibleInFOVNow())
				return knownHealer;
		}
	}
	
	return threat;
}

CKnownEntity SelectCloserThreat(INextBot bot, const CKnownEntity threat1, const CKnownEntity threat2)
{
	float rangeSq1 = bot.GetRangeSquaredTo(threat1.GetEntity());
	float rangeSq2 = bot.GetRangeSquaredTo(threat2.GetEntity());
	
	if (rangeSq1 < rangeSq2)
		return threat1;
	
	return threat2;
}

void MonitorKnownEntities(int client, IVision vision)
{
	if (nb_blind.BoolValue)
		return;
	
	static int maxEntCount = -1;
	
	if (maxEntCount == -1)
		maxEntCount = GetMaxEntities();
	
	int myTeam = GetClientTeam(client);
	
	for (int i = 1; i <= maxEntCount; i++)
	{
		if (!IsValidEntity(i))
			continue;
		
		if (i == client)
			continue;
		
		if (BaseEntity_IsPlayer(i) && !IsPlayerAlive(i))
			continue;
		
		if (CBaseEntity(i).IsCombatCharacter() == false)
			continue;
		
		if (BaseEntity_GetTeamNumber(i) == myTeam)
			continue;
		
		/* IVision::UpdateKnownEntities runs its own check for collecting potentially visible entities
		However it only seems to check for them only regarding the bot's FOV
		When the known entity leaves the bot's FOV, it would eventually become obsolete after 10 seconds
		And when it becomes obsolete, it gets removed from the list of known entities
		So here we are basically expanding the functionality using our own line-of-sight of check */
		if (IsLineOfFireClearEntity(client, GetEyePosition(client), i))
		{
			CKnownEntity known = vision.GetKnown(i);
			
			if (known)
			{
				//We already know about this entity and we can currently see it
				known.UpdatePosition();
			}
			else
			{
				//We didn't know about it but we can see it now, recognize it
				vision.AddKnownEntity(i);
			}
		}
	}
}

int GetCountOfBotsWithNamedAction(const char[] name, int ignore = -1)
{
	int count = 0;
	
	for (int i = 1; i <= MaxClients; i++)
		if (i != ignore && IsClientInGame(i) && g_bIsDefenderBot[i] && ActionsManager.LookupEntityActionByName(i, name) != INVALID_ACTION)
			count++;
	
	return count;
}

void UtilizeCompressionBlast(int client, INextBot bot, const CKnownEntity threat, int enhancedStage = 0)
{
	if (threat == NULL_KNOWN_ENTITY)
		return;
	
	if (redbots_manager_bot_reflect_skill.IntValue < 1)
		return;
	
	int iThreat = threat.GetEntity();
	
	if (BaseEntity_IsPlayer(iThreat))
	{
		float threatOrigin[3]; GetClientAbsOrigin(iThreat, threatOrigin);
		
		//Make sure we're close enough to actually airblast them
		if (bot.IsRangeLessThanEx(threatOrigin, 250.0))
		{
			if (TF2_IsInvulnerable(iThreat))
			{
				//Shove ubers away from us
				g_arrExtraButtons[client].ReleaseButtons(IN_ATTACK);
				VS_PressAltFireButton(client);
				return;
			}
			
			if (TF2_IsPlayerInCondition(iThreat, TFCond_Charging))
			{
				//Shove chargers away from us
				g_arrExtraButtons[client].ReleaseButtons(IN_ATTACK);
				VS_PressAltFireButton(client);
				return;
			}
			
			if (TF2_HasTheFlag(iThreat) && GetVectorDistance(threatOrigin, GetBombHatchPosition()) <= 100.0)
			{
				//Shove the bomb carrier off the hatch
				g_arrExtraButtons[client].ReleaseButtons(IN_ATTACK);
				VS_PressAltFireButton(client);
				return;
			}
		}
	}
	
	if (redbots_manager_bot_reflect_skill.IntValue < 2)
		return;
	
	if (redbots_manager_bot_reflect_chance.FloatValue < 100.0 && TransientlyConsistentRandomValue(client, 1.0) > redbots_manager_bot_reflect_chance.FloatValue / 100.0)
		return;
	
	//Enhanced projectile airblast
	int myTeam = GetClientTeam(client);
	float myEyePos[3]; GetClientEyePosition(client, myEyePos);
	int ent = -1;
	
	while ((ent = FindEntityByClassname(ent, "tf_projectile_*")) != -1)
	{
		if (BaseEntity_GetTeamNumber(ent) == myTeam)
			continue;
		
		if (!CanBeReflected(ent))
			continue;
		
		float origin[3]; BaseEntity_GetLocalOrigin(ent, origin);
		float vec[3]; MakeVectorFromPoints(origin, myEyePos, vec);
		
		//Airblast the projectile if we are actually facing towards it
		if (GetVectorLength(vec) < 150.0)
		{
			g_arrExtraButtons[client].ReleaseButtons(IN_ATTACK);
			VS_PressAltFireButton(client);
			return;
		}
	}
}

bool ShouldBuybackIntoGame(int client)
{
	//Scouts respawn very quickly
	if (TF2_GetPlayerClass(client) == TFClass_Scout)
		return false;
	
	//Can't afford a buyback
	if (TF2_GetCurrency(client) < MVM_BUYBACK_COST_PER_SEC)
		return false;
	
	//Not opportunistic if we're about to fail
	if (IsFailureImminent(client))
		return true;
	
	//We're being revived
	if (g_bIsBeingRevived[client])
		return false;
	
	//Based on our rolled number, decide to buyback
	return g_iBuybackNumber[client] <= redbots_manager_bot_buyback_chance.IntValue;
}

bool ShouldUpgradeMidRound(int client)
{
	//If we were revived, we should not bother
	if (!TF2Util_IsPointInRespawnRoom(WorldSpaceCenter(client), client))
		return false;
	
	//Based on our rolled number from spawn, decide to buy upgrades now
	return g_iBuyUpgradesNumber[client] > 0 && g_iBuyUpgradesNumber[client] <= redbots_manager_bot_buy_upgrades_chance.IntValue;
}

bool CanBuyUpgradesNow(int client)
{
	if (TF2_GetCurrency(client) < 25)
		return false;
	
	if (IsFailureImminent(client))
		return false;
	
	return true;
}

float TransientlyConsistentRandomValue(int client, float period = 10.0, int seedValue = 0)
{
	CNavArea area = CBaseCombatCharacter(client).GetLastKnownArea();
	
	if (!area)
		return 0.0;
	
	int timeMod = RoundToFloor(GetGameTime() / period) + 1;
	
	return FloatAbs(Cosine(float(seedValue + (client * area.GetID() * timeMod))));
}

bool IsFailureImminent(int client)
{
	//TODO: factor in tank closest to hatch for certain classes
	
	int flag = FindBombNearestToHatch();
	
	if (flag == -1)
		return false;
	
	float bombPosition[3]; bombPosition = WorldSpaceCenter(flag);
	
	//Bomb is far and not a threat
	if (GetVectorDistance(bombPosition, GetBombHatchPosition()) > BOMB_HATCH_RANGE_CRITICAL)
		return false;
	
	int closestToHatch = FindBotNearestToBombNearestToHatch(client);
	
	//No robot near the bomb close to the hatch, we're probably okay for now
	if (closestToHatch == -1)
		return false;
	
	float threatOrigin[3]; GetClientAbsOrigin(closestToHatch, threatOrigin);
	
	//Robot about to pick up a bomb very close to the hatch, we're in danger!
	return GetVectorDistance(threatOrigin, bombPosition) <= 800.0;
}

//Since March 28 2018 update, flamethrower damage is calculated based on the oldest particles
//Aim a bit higher on the tank for the highest damage output
void GetFlameThrowerAimForTank(int tank, float aimPos[3])
{
	aimPos = WorldSpaceCenter(tank);
	aimPos[2] += 90.0;
}

/* Whether a teleporter is worth walking to

It used to answer yes to everything, in both branches, so the caller that exists to stop a bot
looking for one never stopped anything. A play-test watched the result and drew the obvious
conclusion, that engineers are better off not building teleporters at all: a defender who wants
to ride one has to walk back to the entrance first, and the entrance is at the spawn the fight is
being pushed towards. So the bots walked away from the hatch, into the teleporter, and out of it
roughly where they had started, having given the wave the seconds it takes to do all that.

A ride is worth it when it saves a walk the bot would otherwise make: the fight has to be far
enough up the path that going back to spawn and coming out forward is still ahead. When the bomb
is on the hatch, nothing is forward of anything, and the answer is no.

This says nothing about which teleporter, because there is nothing here to say it with. It is the
gate on looking at all, which is where the cost is */

//Far enough up the path that the walk back to the entrance is bought back by the ride
#define TELEPORTER_WORTH_RIDING	1500.0

/* Sending an engineer with no sentry to his own teleporter was tried and lost
 *
 * The measurement that suggested it is real: the team has no sentry for well over half of a
 * Coaltown wave, and more than half of that is the engineer alive and walking back from a spawn
 * three and a half thousand units from his nest. He builds a level three teleporter for exactly
 * that journey.
 *
 * Twelve waves on two maps, engineer_rides_home: sentry uptime fell from 69 percent to 54, the
 * worst gap grew from 95 seconds to 125, and the samples of him walking with no sentry went from
 * 20 to 58. Saying yes here does not put him on the teleporter, it puts him on a walk to the
 * entrance, which is back in the spawn he is trying to leave. The walk it saves is shorter than
 * the walk it costs, and the switch is gone rather than turned off.
 */
static bool ShouldUseTeleporter(int client)
{
	BombInfo_t bombinfo;

	//No bomb in play, so there is no fight to be late for and no reason to leave the ground
	if (!GetBombInfo(bombinfo))
		return false;

	CTFNavArea myArea = view_as<CTFNavArea>(CBaseCombatCharacter(client).GetLastKnownArea());

	if (myArea == NULL_AREA)
		return false;

	CTFNavArea bombArea = view_as<CTFNavArea>(TheNavMesh.GetNearestNavArea(bombinfo.vPosition));

	if (bombArea == NULL_AREA)
		return false;

	return GetTravelDistanceToBombTarget(myArea) + TELEPORTER_WORTH_RIDING < GetTravelDistanceToBombTarget(bombArea);
}
