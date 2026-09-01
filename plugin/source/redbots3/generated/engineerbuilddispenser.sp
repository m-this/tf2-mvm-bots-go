BehaviorAction CTFBotMvMEngineerBuildDispenser()
{
	BehaviorAction action = ActionsManager.Create("DefenderBuildDispenser");

	action.OnStart = CTFBotMvMEngineerBuildDispenser_OnStart;
	action.Update = CTFBotMvMEngineerBuildDispenser_Update;
	action.OnEnd = CTFBotMvMEngineerBuildDispenser_OnEnd;

	return action;
}

#define Go_Slots (65)

#define DISPENSER_SPOT_TAKEN_RANGE (150.0)

#define DISPENSER_SETTLE_RANGE (200.0)

#define DISPENSER_BUILD_REACH (90.0)
#define DISPENSER_TRY_POINTS (8)
#define DISPENSER_TRY_TIME (2.0)

#define DISPENSER_PRESS_SETTLE (0.3)

#define DISPENSER_BUILD_TIME (45.0)

float m_ctDispenserReachDeadline[65];
float m_ctDispenserGiveUpTime[65];
float m_ctDispenserTryDeadline[65];
float m_ctDispenserPressed[65];
int m_iDispenserTry[65];
float m_vDispenserSpot[65][3];
float m_vDispenserStand[65][3];

public Action CTFBotMvMEngineerBuildDispenser_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	UpdateLookAroundForEnemies(actor, true);
	m_ctDispenserGiveUpTime[actor] = GetGameTime() + DISPENSER_BUILD_TIME;
	m_ctDispenserTryDeadline[actor] = GetGameTime() + DISPENSER_TRY_TIME;
	m_ctDispenserPressed[actor] = 0.0;
	m_iDispenserTry[actor] = 0;
	float spot[3];
	bool configured = ConfiguredDispenserSpot(actor, spot);
	m_vDispenserSpot[actor] = spot;
	if (!configured)
	{
		if (m_aNestArea[actor] != NULL_AREA)
		{
			CNavArea_GetRandomPoint(m_aNestArea[actor], m_vDispenserSpot[actor]);
		}
		else
		{
			m_vDispenserSpot[actor] = GetAbsOrigin(actor);
		}
	}
	float stand[3];
	bool ok = DispenserStandPoint(actor, m_iDispenserTry[actor], stand);
	m_vDispenserStand[actor] = stand;
	if (!ok)
	{
		NextDispenserStandPoint(actor);
	}
	m_ctDispenserReachDeadline[actor] = GetGameTime() + BuildReachTime(GetAbsOrigin(actor), m_vDispenserStand[actor]);
	return action.Continue();
}

public Action CTFBotMvMEngineerBuildDispenser_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (m_aNestArea[actor] == NULL_AREA)
	{
		LogBuildFailure(actor, "dispenser", "no nest area");
		return action.Done("No hint entity");
	}
	int sentry = GetObjectOfType(actor, TFObject_Sentry);
	if (sentry == INVALID_ENT_REFERENCE)
	{
		LogBuildFailure(actor, "dispenser", "no sentry to feed");
		return action.Done("No sentry");
	}
	if (!IsSentrySafe(sentry))
	{
		LogBuildFailure(actor, "dispenser", "sentry under fire");
		return action.Done("Sentry not safe");
	}
	if (CTFBotMvMEngineerIdle_ShouldAdvanceNestSpot(actor))
	{
		LogBuildFailure(actor, "dispenser", "told to advance the nest");
		return action.Done("Need to advance nest");
	}
	if (GetGameTime() > m_ctDispenserGiveUpTime[actor])
	{
		LogBuildFailure(actor, "dispenser", "ran out of time to place it");
		return action.Done("Nowhere to put a dispenser");
	}
	float spot[3];
	spot = m_vDispenserSpot[actor];
	float stand[3];
	stand = m_vDispenserStand[actor];
	bool outOfTime = (m_ctDispenserReachDeadline[actor] > 0.0) && (GetGameTime() > m_ctDispenserReachDeadline[actor]) && (GetVectorDistance(GetAbsOrigin(actor), spot) < DISPENSER_SETTLE_RANGE);
	if (outOfTime)
	{
		stand = GetAbsOrigin(actor);
	}
	if ((m_ctDispenserReachDeadline[actor] > 0.0) && (GetGameTime() > m_ctDispenserReachDeadline[actor]) && (GetVectorDistance(GetAbsOrigin(actor), spot) >= DISPENSER_SETTLE_RANGE))
	{
		LogBuildFailure(actor, "dispenser", "could not reach the spot, gave it up");
		return action.Done("Cannot reach the dispenser spot");
	}
	float rangeToStand = GetVectorDistance(GetAbsOrigin(actor), stand);
	INextBot myNextbot = CBaseNPC_GetNextBotOfEntity(actor);
	IBody myBody = myNextbot.GetBodyInterface();
	if (rangeToStand < 200.0)
	{
		if (!IsBuilderSetTo(actor, TFObject_Dispenser))
		{
			FakeClientCommandThrottled(actor, "build 0");
		}
		AimHeadTowards(myBody, spot, MANDATORY, 0.1, Address_Null, "Placing dispenser");
	}
	if (rangeToStand > 70.0)
	{
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
		if (!IsPlacementOK(objBeingBuilt) && !outOfTime && myBody.IsHeadAimingOnTarget() && (GetGameTime() > m_ctDispenserTryDeadline[actor]))
		{
			NextDispenserStandPoint(actor);
			return action.Continue();
		}
	}
	int dispenser = GetObjectOfType(actor, TFObject_Dispenser);
	if (dispenser != INVALID_ENT_REFERENCE)
	{
		SetPlayerReady(actor, true);
		return action.Done("Built a dispenser");
	}
	if (GetGameTime() < m_ctDispenserPressed[actor])
	{
		return action.Continue();
	}
	m_ctDispenserPressed[actor] = GetGameTime() + DISPENSER_PRESS_SETTLE;
	VS_PressFireButton(actor);
	return action.Continue();
}

stock bool DispenserStandPoint(int actor, int attempt, float stand[3])
{
	bool ok;
	for (int i = 0; i < 3; i++)
	{
		stand[i] = 0.0;
	}
	ok = BuildStandPoint(m_vDispenserSpot[actor], GetAbsOrigin(actor), attempt, DISPENSER_TRY_POINTS, DISPENSER_BUILD_REACH, stand);
	return ok;
}

stock void NextDispenserStandPoint(int actor)
{
	for (m_iDispenserTry[actor]++; m_iDispenserTry[actor] < DISPENSER_TRY_POINTS; m_iDispenserTry[actor]++)
	{
		float stand[3];
		bool ok = DispenserStandPoint(actor, m_iDispenserTry[actor], stand);
		m_vDispenserStand[actor] = stand;
		if (!ok)
		{
			continue;
		}
		m_ctDispenserTryDeadline[actor] = GetGameTime() + DISPENSER_TRY_TIME;
		return;
	}
	m_ctDispenserReachDeadline[actor] = GetGameTime();
}

public void CTFBotMvMEngineerBuildDispenser_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	UpdateLookAroundForEnemies(actor, true);
}

stock bool ConfiguredDispenserSpot(int actor, float spot[3])
{
	bool found;
	for (int i = 0; i < 3; i++)
	{
		spot[i] = 0.0;
	}
	ArrayList spots = g_arrMapConfig.adtDispenserLocation;
	if (spots.Length == 0)
	{
		return false;
	}
	float nest[3];
	NestBuildPosition(m_aNestArea[actor], nest);
	char myZone[512];
	NestZoneOf(m_aNestArea[actor], myZone, 512);
	ArrayList free = new ArrayList(3);
	ArrayList refused = new ArrayList(3);
	CollectDispenserSpots(actor, myZone, free, refused);
	if ((free.Length == 0) && (myZone[0] != 0))
	{
		CollectDispenserSpots(actor, "", free, refused);
	}
	found = NearestConfiguredSpot(free, nest, spot);
	if (!found)
	{
		found = NearestConfiguredSpot(refused, nest, spot);
	}
	if (redbots_manager_debug.BoolValue)
	{
		if (found)
		{
			PrintToServer("ConfiguredDispenserSpot: %N takes the named spot %.0f %.0f %.0f", actor, spot[0], spot[1], spot[2]);
		}
		else
		{
			PrintToServer("ConfiguredDispenserSpot: %N has no named spot for the nest at %.0f %.0f %.0f", actor, nest[0], nest[1], nest[2]);
		}
	}
	delete refused;
	delete free;
	return found;
}

stock void CollectDispenserSpots(int actor, const char[] wanted, ArrayList free, ArrayList refused)
{
	ArrayList spots = g_arrMapConfig.adtDispenserLocation;
	ArrayList zones = g_arrMapConfig.adtDispenserZone;
	for (int i = 0; i < spots.Length; i++)
	{
		char zone[512];
		if (i < zones.Length)
		{
			zones.GetString(i, zone, 512);
		}
		if (!StrEqual(zone, wanted))
		{
			continue;
		}
		float candidate[3];
		spots.GetArray(i, candidate);
		if (IsDispenserSpotTaken(actor, candidate))
		{
			continue;
		}
		if (IsPathToVectorPossible(actor, candidate))
		{
			free.PushArray(candidate);
		}
		else
		{
			refused.PushArray(candidate);
		}
	}
}

stock bool IsDispenserSpotTaken(int actor, float spot[3])
{
	for (int i = 1; i <= MaxClients; i++)
	{
		if ((i == actor) || !IsClientInGame(i))
		{
			continue;
		}
		int dispenser = GetObjectOfType(i, TFObject_Dispenser);
		if (dispenser == INVALID_ENT_REFERENCE)
		{
			continue;
		}
		if (GetVectorDistance(spot, GetAbsOrigin(dispenser)) < DISPENSER_SPOT_TAKEN_RANGE)
		{
			return true;
		}
	}
	return false;
}

