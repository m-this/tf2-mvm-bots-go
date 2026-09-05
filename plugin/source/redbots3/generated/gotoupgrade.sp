BehaviorAction CTFBotGotoUpgrade()
{
	BehaviorAction action = ActionsManager.Create("DefenderGotoUpgrade");

	action.OnStart = CTFBotGotoUpgrade_OnStart;
	action.Update = CTFBotGotoUpgrade_Update;
	action.OnEnd = CTFBotGotoUpgrade_OnEnd;
	action.OnNavAreaChanged = CTFBotGotoUpgrade_OnNavAreaChanged;

	return action;
}

int m_iStation[65];

// OnStart picks a station, and pretends the bot is at one when there is none it
// could reach.
public Action CTFBotGotoUpgrade_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	m_iStation[actor] = FindClosestUpgradeStation(actor);
	if ((m_iStation[actor] <= MaxClients) || !IsValidEntity(m_iStation[actor]))
	{
		// We couldn't find an upgrade station to path to, so let's just pretend we're at one
		TF2_SetInUpgradeZone(actor, true);
	}
	else
		if (GameRules_GetRoundState() == RoundState_RoundRunning)
		{
			float myOrigin[3];
			GetClientAbsOrigin(actor, myOrigin);
			// The closest station is so far away, pretend we're in it
			if (GetVectorDistance(myOrigin, WorldSpaceCenter(m_iStation[actor])) >= 1000.0)
			{
				TF2_SetInUpgradeZone(actor, true);
			}
		}
	return action.Continue();
}

// Update walks there, tracing for ground the bot can stand on in front of it.
public Action CTFBotGotoUpgrade_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (TF2_IsInUpgradeZone(actor))
	{
		return action.ChangeTo(CTFBotUpgrade(), "Reached upgrade station; buying upgrades");
	}
	int theStation = m_iStation[actor];
	// Moved from OnStart for technical reasons
	float center[3];
	bool hasGoal = GetMapUpgradeStationGoal(center);
	if (!hasGoal)
	{
		if ((theStation <= MaxClients) || !IsValidEntity(theStation))
		{
			return action.Done("No upgrade station to path to");
		}
		CNavArea area = TheNavMesh.GetNearestNavArea(WorldSpaceCenter(theStation), true, 1000.0, false, false, TEAM_ANY);
		if (area == NULL_AREA)
		{
			return action.Continue();
		}
		CNavArea_GetRandomPoint(area, center);
		center[2] += 50.0;
		TR_TraceRayFilter(center, WorldSpaceCenter(theStation), MASK_PLAYERSOLID, RayType_EndPoint, NextBotTraceFilterIgnoreActors);
		TR_GetEndPosition(center);
	}
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	if (m_flRepathTime[actor] <= GetGameTime())
	{
		m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(1.0, 2.0);
		RepathToPos(actor, myBot, center);
	}
	if (PathFailedFor(actor))
	{
		NudgeTowardsGoal(actor, myBot, center);
	}
	else
	{
		m_pPath[actor].Update(myBot);
	}
	return action.Continue();
}

// OnEnd forgets the station.
public void CTFBotGotoUpgrade_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_iStation[actor] = -1;
}

// OnNavAreaChanged bails out if the bot has wandered out of spawn mid-wave.
public Action CTFBotGotoUpgrade_OnNavAreaChanged(BehaviorAction action, int actor, CTFNavArea newArea, CTFNavArea oldArea, ActionDesiredResult result)
{
	// If we are for some reason not in our spawn room during an active game, just bail out
	if ((newArea != 0) && (GameRules_GetRoundState() == RoundState_RoundRunning))
	{
		int spawnRoomFlag = BLUE_SPAWN_ROOM;
		if (TF2_GetClientTeam(actor) == TFTeam_Red)
		{
			spawnRoomFlag = RED_SPAWN_ROOM;
		}
		if (!newArea.HasAttributeTF(spawnRoomFlag))
		{
			return action.TryDone(RESULT_IMPORTANT, "I am not in a spawn room");
		}
	}
	return action.TryContinue();
}

// FindClosestUpgradeStation picks one of the stations a bot can actually walk
// to, at random.
stock int FindClosestUpgradeStation(int actor)
{
	int stations[65];
	int stationcount = 0;
	int i = -1;
	for (;;)
	{
		i = FindEntityByClassname(i, "func_upgradestation");
		if (i == -1)
		{
			break;
		}
		if (!IsUpgradeStationEnabled(i))
		{
			continue;
		}
		CNavArea area = TheNavMesh.GetNearestNavArea(WorldSpaceCenter(i), true, 8000.0, false, false, TEAM_ANY);
		if (area == NULL_AREA)
		{
			continue;
		}
		float center[3];
		area.GetCenter(center);
		center[2] += 50.0;
		TR_TraceRay(center, WorldSpaceCenter(i), MASK_PLAYERSOLID, RayType_EndPoint);
		TR_GetEndPosition(center);
		if (!IsPathToVectorPossible(actor, center))
		{
			continue;
		}
		stations[stationcount] = i;
		stationcount++;
	}
	if (stationcount == 0)
	{
		return -1;
	}
	return stations[GetRandomInt(0, stationcount - 1)];
}

// MapUpgradeStationGoal is the hard coded spot for the six maps whose station
// the nav mesh cannot get a bot to.
//
// switch over which prefix a map name contains is not one SourcePawn can write
stock bool GetMapUpgradeStationGoal(float buffer[3])
{
	for (int i = 0; i < 3; i++)
	{
		buffer[i] = 0.0;
	}
	char mapName[512];
	GetCurrentMap(mapName, 512);
	if (StrContains(mapName, "mvm_mannworks", true) != -1)
	{
		buffer = {-643.9, -2635.2, 384.0};
		return true;
	}
	else
		if (StrContains(mapName, "mvm_teien", true) != -1)
		{
			buffer = {4613.1, -6561.9, 260.0};
			return true;
		}
		else
			if (StrContains(mapName, "mvm_sequoia", true) != -1)
			{
				buffer = {-5117.0, -377.3, 4.5};
				return true;
			}
			else
				if (StrContains(mapName, "mvm_highground", true) != -1)
				{
					buffer = {-2013.0, 4561.0, 448.0};
					return true;
				}
				else
					if (StrContains(mapName, "mvm_newnormandy", true) != -1)
					{
						buffer = {-345.0, 4178.0, 205.0};
						return true;
					}
					else
						if (StrContains(mapName, "mvm_snowfall", true) != -1)
						{
							buffer = {-26.0, 792.0, -159.0};
							return true;
						}
	return false;
}

// ResetGotoUpgrade forgets the station this bot walked to.
//
// A bot leaving takes its seat's state with it, and the next bot in that seat
// is a different bot.
stock void Go_ResetGotoUpgrade(int client)
{
	m_iStation[client] = -1;
}

