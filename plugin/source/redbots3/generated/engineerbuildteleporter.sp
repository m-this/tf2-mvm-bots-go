BehaviorAction CTFBotMvMEngineerBuildTeleporter()
{
	BehaviorAction action = ActionsManager.Create("DefenderBuildTeleporter");

	action.OnStart = CTFBotMvMEngineerBuildTeleporter_OnStart;
	action.Update = CTFBotMvMEngineerBuildTeleporter_Update;
	action.OnEnd = CTFBotMvMEngineerBuildTeleporter_OnEnd;

	return action;
}

#define Go_Slots (65)

#define TELEPORTER_BUILD_MAX_TIME (40.0)

#define TELEPORTER_EXIT_REACH_TIME (12.0)

#define TELEPORTER_CLIMB_RISE_MIN (24.0)
#define TELEPORTER_CLIMB_RISE_MAX (72.0)
#define TELEPORTER_CLIMB_RANGE (140.0)
#define TELEPORTER_CLIMB_INTERVAL (0.7)
#define TELEPORTER_CLIMB_HOLD (0.3)
#define TELEPORTER_CLIMB_LIMIT (6)

#define TELEPORTER_BUILD_REACH (90.0)

#define TELEPORTER_SPAWN_OFFSET (200.0)
#define TELEPORTER_SPAWN_STEP (150.0)

#define TELEPORTER_EXIT_RADIUS (150.0)

#define TELEPORTER_EXIT_RINGS (2)

#define TELEPORTER_TRY_POINTS (8)
#define TELEPORTER_TRY_TIME (1.5)

#define TELEPORTER_EXIT_TAKEN_RANGE (200.0)

float m_ctTeleporterGiveUp[65];
float m_ctTeleporterReachDeadline[65];
float m_ctTeleporterTryDeadline[65];
float m_ctTeleporterClimb[65];
int m_iTeleporterClimbs[65];
int m_iTeleporterTry[65];
TFObjectMode m_nTeleporterMode[65];
float m_vTeleporterSpot[65][3];
float m_vTeleporterStand[65][3];
float m_vTeleporterSpawn[65][3];
float m_vTeleporterNest[65][3];
float m_vTeleporterRouteSpot[65][8][3];
float m_vTeleporterRouteStand[65][8][3];
int m_iTeleporterRoutePoints[65];
bool m_bTeleporterNamedSpot[65];
bool m_bTeleporterGaveUp[65];
char m_sTeleporterLastResult[65][512];

public Action CTFBotMvMEngineerBuildTeleporter_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_ctTeleporterGiveUp[actor] = GetGameTime() + TELEPORTER_BUILD_MAX_TIME;
	m_ctTeleporterReachDeadline[actor] = GetGameTime() + TELEPORTER_EXIT_REACH_TIME;
	m_ctTeleporterTryDeadline[actor] = GetGameTime() + TELEPORTER_TRY_TIME;
	m_ctTeleporterClimb[actor] = 0.0;
	m_iTeleporterClimbs[actor] = 0;
	m_iTeleporterTry[actor] = 0;
	m_iTeleporterRoutePoints[actor] = 0;
	if ((m_nTeleporterMode[actor] == TFObjectMode_Entrance) && !m_bTeleporterNamedSpot[actor])
	{
		m_iTeleporterRoutePoints[actor] = SpawnRoutePoints(actor, m_vTeleporterSpawn[actor], TELEPORTER_SPAWN_OFFSET, TELEPORTER_SPAWN_STEP, TELEPORTER_BUILD_REACH, m_vTeleporterRouteSpot[actor], m_vTeleporterRouteStand[actor], TELEPORTER_TRY_POINTS);
	}
	if (!TeleporterStandPoint(actor))
	{
		m_bTeleporterGaveUp[actor] = true;
		return TeleporterDone(action, actor, "No route out of spawn to walk");
	}
	UpdateLookAroundForEnemies(actor, true);
	return action.Continue();
}

public Action CTFBotMvMEngineerBuildTeleporter_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (GameRules_GetRoundState() != RoundState_BetweenRounds)
	{
		return TeleporterDone(action, actor, "Wave started");
	}
	if (GetObjectOfType(actor, TFObject_Sentry) == INVALID_ENT_REFERENCE)
	{
		return TeleporterDone(action, actor, "No sentry to leave behind");
	}
	if (GetObjectOfType(actor, TFObject_Teleporter, m_nTeleporterMode[actor]) != INVALID_ENT_REFERENCE)
	{
		g_arrPluginBot[actor].bPathing = false;
		return TeleporterDone(action, actor, "Built one");
	}
	if (m_ctTeleporterGiveUp[actor] < GetGameTime())
	{
		m_bTeleporterGaveUp[actor] = true;
		return TeleporterDone(action, actor, "Ran out of time");
	}
	if (Feature(FEATURE_ENGINEER_CLIMBS) && (m_nTeleporterMode[actor] == TFObjectMode_Exit) && (GetGameTime() > m_ctTeleporterReachDeadline[actor]))
	{
		TeleporterFallBackToNest(actor);
	}
	float spot[3];
	spot = m_vTeleporterSpot[actor];
	bool outOfTime = (m_nTeleporterMode[actor] == TFObjectMode_Exit) && (GetGameTime() > m_ctTeleporterReachDeadline[actor]);
	INextBot myNextbot = CBaseNPC_GetNextBotOfEntity(actor);
	IBody myBody = myNextbot.GetBodyInterface();
	if (redbots_manager_debug_actions.BoolValue && Feature(FEATURE_ENGINEER_CLIMBS) && (outOfTime || !m_bTeleporterNamedSpot[actor]))
	{
		PrintToServer("[teleclimb] %N not asked: out of time %d, named spot %d", actor, outOfTime, m_bTeleporterNamedSpot[actor]);
	}
	if (Feature(FEATURE_ENGINEER_CLIMBS) && !outOfTime && m_bTeleporterNamedSpot[actor] && TeleporterClimbToSpot(actor, myBody, spot))
	{
		g_arrPluginBot[actor].bPathing = false;
		return action.Continue();
	}
	float stand[3];
	stand = m_vTeleporterStand[actor];
	if (outOfTime)
	{
		stand = GetAbsOrigin(actor);
	}
	float teleporterRange = GetVectorDistance(GetAbsOrigin(actor), stand);
	if (teleporterRange < 200.0)
	{
		if (!IsBuilderSetTo(actor, TFObject_Teleporter, m_nTeleporterMode[actor]))
		{
			FakeClientCommandThrottled(actor, (m_nTeleporterMode[actor] == TFObjectMode_Entrance ? "build 1 0" : "build 1 1"));
		}
		AimHeadTowards(myBody, spot, MANDATORY, 0.1, Address_Null, "Placing teleporter");
	}
	if (teleporterRange > 70.0)
	{
		m_ctTeleporterTryDeadline[actor] = GetGameTime() + TELEPORTER_TRY_TIME;
		g_arrPluginBot[actor].SetPathGoalVector(stand);
		g_arrPluginBot[actor].bPathing = true;
		return action.Continue();
	}
	g_arrPluginBot[actor].bPathing = false;
	int myWeapon = BaseCombatCharacter_GetActiveWeapon(actor);
	if ((myWeapon != -1) && (TF2Util_GetWeaponID(myWeapon) == TF_WEAPON_BUILDER))
	{
		int objBeingBuilt = GetEntPropEnt(myWeapon, Prop_Send, "m_hObjectBeingBuilt");
		if (objBeingBuilt == -1)
		{
			return action.Continue();
		}
		if (!IsPlacementOK(objBeingBuilt) && !outOfTime && myBody.IsHeadAimingOnTarget() && (GetGameTime() > m_ctTeleporterTryDeadline[actor]))
		{
			m_iTeleporterTry[actor]++;
			if ((m_iTeleporterTry[actor] >= TeleporterTryLimit(actor)) || !TeleporterStandPoint(actor))
			{
				if (m_nTeleporterMode[actor] != TFObjectMode_Exit)
				{
					m_bTeleporterGaveUp[actor] = true;
					return TeleporterDone(action, actor, "Nowhere out of spawn takes one");
				}
				if (!Feature(FEATURE_ENGINEER_CLIMBS) || !TeleporterFallBackToNest(actor))
				{
					m_ctTeleporterReachDeadline[actor] = GetGameTime();
				}
				return action.Continue();
			}
			m_ctTeleporterReachDeadline[actor] = GetGameTime() + TELEPORTER_EXIT_REACH_TIME;
			return action.Continue();
		}
	}
	VS_PressFireButton(actor);
	return action.Continue();
}

stock int TeleporterTryLimit(int actor)
{
	if (!m_bTeleporterNamedSpot[actor] && (m_nTeleporterMode[actor] == TFObjectMode_Exit))
	{
		return 16;
	}
	return TELEPORTER_TRY_POINTS;
}

stock void SayClimb(int actor, const char[] why, float rise, float flat)
{
	if (!redbots_manager_debug_actions.BoolValue)
	{
		return;
	}
	PrintToServer("[teleclimb] %N %s, rise %.0f of %.0f to %.0f, out %.0f of %.0f, climb %d of %d", actor, why, rise, TELEPORTER_CLIMB_RISE_MIN, TELEPORTER_CLIMB_RISE_MAX, flat, TELEPORTER_CLIMB_RANGE, m_iTeleporterClimbs[actor], TELEPORTER_CLIMB_LIMIT);
}

stock bool TeleporterClimbToSpot(int actor, IBody myBody, float spot[3])
{
	float origin[3];
	origin = GetAbsOrigin(actor);
	float rise = spot[2] - origin[2];
	float reach[3];
	SubtractVectors(spot, origin, reach);
	reach[2] = 0.0;
	float out = GetVectorLength(reach);
	if (rise < TELEPORTER_CLIMB_RISE_MIN)
	{
		SayClimb(actor, "nothing to climb", rise, out);
		if (m_iTeleporterClimbs[actor] > 0)
		{
			m_iTeleporterClimbs[actor] = 0;
			m_vTeleporterStand[actor] = origin;
		}
		return false;
	}
	if ((rise > TELEPORTER_CLIMB_RISE_MAX) || (m_iTeleporterClimbs[actor] >= TELEPORTER_CLIMB_LIMIT))
	{
		SayClimb(actor, (rise > TELEPORTER_CLIMB_RISE_MAX ? "too high to climb" : "out of climbs"), rise, out);
		return false;
	}
	if (out > TELEPORTER_CLIMB_RANGE)
	{
		SayClimb(actor, "too far out to climb", rise, out);
		return false;
	}
	SayClimb(actor, "climbing", rise, out);
	AimHeadTowards(myBody, spot, MANDATORY, 0.2, Address_Null, "Climbing to the teleporter spot");
	if (m_ctTeleporterClimb[actor] > GetGameTime())
	{
		return true;
	}
	m_ctTeleporterClimb[actor] = GetGameTime() + TELEPORTER_CLIMB_INTERVAL;
	m_iTeleporterClimbs[actor]++;
	g_arrExtraButtons[actor].PressButtons(IN_FORWARD | IN_JUMP | IN_DUCK, TELEPORTER_CLIMB_HOLD);
	return true;
}

stock bool TeleporterFallBackToNest(int actor)
{
	if (!m_bTeleporterNamedSpot[actor] || (m_nTeleporterMode[actor] != TFObjectMode_Exit))
	{
		return false;
	}
	m_bTeleporterNamedSpot[actor] = false;
	m_iTeleporterTry[actor] = 0;
	m_iTeleporterClimbs[actor] = 0;
	m_ctTeleporterReachDeadline[actor] = GetGameTime() + TELEPORTER_EXIT_REACH_TIME;
	return TeleporterStandPoint(actor);
}

stock bool TeleporterStandPoint(int actor)
{
	int attempt = m_iTeleporterTry[actor];
	if (m_bTeleporterNamedSpot[actor])
	{
		float stand[3];
		BuildStandPoint(m_vTeleporterSpot[actor], GetAbsOrigin(actor), attempt, TELEPORTER_TRY_POINTS, TELEPORTER_BUILD_REACH, stand);
		m_vTeleporterStand[actor] = stand;
		return true;
	}
	if (m_nTeleporterMode[actor] == TFObjectMode_Exit)
	{
		float nest[3];
		nest = m_vTeleporterNest[actor];
		float radius = 150.0;
		if (attempt < TELEPORTER_TRY_POINTS)
		{
			radius = BUSTER_BLAST_RANGE + 100.0;
		}
		int angle = attempt % TELEPORTER_TRY_POINTS;
		float spot[3];
		BuildStandPoint(nest, GetAbsOrigin(actor), angle, TELEPORTER_TRY_POINTS, radius, spot);
		m_vTeleporterSpot[actor] = spot;
		float stand[3];
		BuildStandPoint(nest, GetAbsOrigin(actor), angle, TELEPORTER_TRY_POINTS, radius - TELEPORTER_BUILD_REACH, stand);
		m_vTeleporterStand[actor] = stand;
		return true;
	}
	if (attempt >= m_iTeleporterRoutePoints[actor])
	{
		return false;
	}
	m_vTeleporterSpot[actor] = m_vTeleporterRouteSpot[actor][attempt];
	m_vTeleporterStand[actor] = m_vTeleporterRouteStand[actor][attempt];
	return true;
}

stock Action TeleporterDone(BehaviorAction action, int actor, const char[] reason)
{
	strcopy(m_sTeleporterLastResult[actor], 512, reason);
	return action.Done(reason);
}

public void CTFBotMvMEngineerBuildTeleporter_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	g_arrPluginBot[actor].bPathing = false;
	UpdateLookAroundForEnemies(actor, true);
}

stock void EngineerTeleporter_LastResult(int actor, char[] buffer, int maxlength)
{
	strcopy(buffer, maxlength, (m_sTeleporterLastResult[actor][0] == 0 ? "nothing yet" : m_sTeleporterLastResult[actor]));
}

stock bool EngineerTeleporter_HasGivenUp(int actor)
{
	return m_bTeleporterGaveUp[actor];
}

stock TFObjectMode EngineerTeleporter_Mode(int actor)
{
	return m_nTeleporterMode[actor];
}

stock void EngineerTeleporter_Spot(int actor, float spot[3])
{
	for (int i = 0; i < 3; i++)
	{
		spot[i] = 0.0;
	}
	spot = m_vTeleporterSpot[actor];
	return;
}

stock void EngineerTeleporter_ForgetGivingUp()
{
	for (int i = 1; i <= MaxClients; i++)
	{
		m_bTeleporterGaveUp[i] = false;
	}
}

stock bool ShouldBuildTeleporter(int actor)
{
	if (GameRules_GetRoundState() != RoundState_BetweenRounds)
	{
		return false;
	}
	if (m_bTeleporterGaveUp[actor])
	{
		return false;
	}
	if (HasObjectOfType(actor, TFObject_Sentry, TFObjectMode_None) == INVALID_ENT_REFERENCE)
	{
		return false;
	}
	if (HasObjectOfType(actor, TFObject_Dispenser, TFObjectMode_None) == INVALID_ENT_REFERENCE)
	{
		return false;
	}
	if (m_aNestArea[actor] == NULL_AREA)
	{
		return false;
	}
	NestBuildPosition(m_aNestArea[actor], m_vTeleporterNest[actor]);
	if (GetObjectOfType(actor, TFObject_Teleporter, TFObjectMode_Entrance) == INVALID_ENT_REFERENCE)
	{
		m_nTeleporterMode[actor] = TFObjectMode_Entrance;
		float spot[3];
		bool named = NearestConfiguredSpot(g_arrMapConfig.adtTeleporterEntranceLocation, GetAbsOrigin(actor), spot);
		m_bTeleporterNamedSpot[actor] = named;
		m_vTeleporterSpot[actor] = spot;
		if (m_bTeleporterNamedSpot[actor])
		{
			return true;
		}
		float spawn[3];
		bool ok = NearestSpawnPoint(actor, spawn);
		m_vTeleporterSpawn[actor] = spawn;
		return ok;
	}
	if (GetObjectOfType(actor, TFObject_Teleporter, TFObjectMode_Exit) == INVALID_ENT_REFERENCE)
	{
		m_nTeleporterMode[actor] = TFObjectMode_Exit;
		float spot[3];
		bool named = NearestFreeExitSpot(actor, m_vTeleporterNest[actor], spot);
		m_bTeleporterNamedSpot[actor] = named;
		m_vTeleporterSpot[actor] = spot;
		return true;
	}
	return false;
}

stock bool NearestFreeExitSpot(int actor, float nest[3], float spot[3])
{
	bool found;
	for (int i = 0; i < 3; i++)
	{
		spot[i] = 0.0;
	}
	ArrayList spots = g_arrMapConfig.adtTeleporterExitLocation;
	if (spots.Length == 0)
	{
		return false;
	}
	ArrayList free = new ArrayList(3);
	for (int i = 0; i < spots.Length; i++)
	{
		float candidate[3];
		spots.GetArray(i, candidate);
		if (!IsExitSpotTaken(actor, candidate))
		{
			free.PushArray(candidate);
		}
	}
	found = NearestConfiguredSpot(free, nest, spot);
	delete free;
	return found;
}

stock bool IsExitSpotTaken(int actor, float spot[3])
{
	for (int i = 1; i <= MaxClients; i++)
	{
		if ((i == actor) || !IsClientInGame(i))
		{
			continue;
		}
		int exitTele = GetObjectOfType(i, TFObject_Teleporter, TFObjectMode_Exit);
		if (exitTele == INVALID_ENT_REFERENCE)
		{
			continue;
		}
		if (GetVectorDistance(spot, GetAbsOrigin(exitTele)) < TELEPORTER_EXIT_TAKEN_RANGE)
		{
			return true;
		}
	}
	return false;
}

stock bool NearestConfiguredSpot(ArrayList spots, float from[3], float spot[3])
{
	for (int i = 0; i < 3; i++)
	{
		spot[i] = 0.0;
	}
	float nearest = -1.0;
	for (int i = 0; i < spots.Length; i++)
	{
		float candidate[3];
		spots.GetArray(i, candidate);
		float distance = GetVectorDistance(from, candidate);
		if ((nearest < 0.0) || (distance < nearest))
		{
			nearest = distance;
			spot = candidate;
		}
	}
	return nearest >= 0.0;
}

