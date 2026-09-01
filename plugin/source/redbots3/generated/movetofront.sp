BehaviorAction CTFBotMoveToFront()
{
	BehaviorAction action = ActionsManager.Create("DefenderMoveToFront");

	action.OnStart = CTFBotMoveToFront_OnStart;
	action.Update = CTFBotMoveToFront_Update;
	action.OnEnd = CTFBotMoveToFront_OnEnd;

	return action;
}

#define Go_Slots (65)

#define MOVE_TO_FRONT_ARRIVED (80.0)
#define MOVE_TO_FRONT_REACH (60.0)
#define MOVE_TO_FRONT_TRIES (3)

float m_vecGoalArea[65][3];
float m_ctMoveTimeout[65];
int m_iMoveToFrontTry[65];
bool m_bAtTheFront[65];

stock bool IsWaitingAtTheFront(int client)
{
	return m_bAtTheFront[client];
}

stock bool PickTheFront(int actor)
{
	if (Feature(FEATURE_HOLD_THE_NEST) && FightsAtRange(actor) && PickTheNest(actor))
	{
		return true;
	}
	int spawn = -1;
	for (;;)
	{
		spawn = FindEntityByClassname(spawn, "func_respawnroomvisualizer");
		if (spawn == -1)
		{
			break;
		}
		if (GetEntProp(spawn, Prop_Data, "m_iDisabled") != 0)
		{
			continue;
		}
		if (BaseEntity_GetTeamNumber(spawn) == BaseEntity_GetTeamNumber(actor))
		{
			continue;
		}
		break;
	}
	if (spawn == -1)
	{
		return false;
	}
	float flSmallestDistance = 99999.0;
	int iBestEnt = -1;
	int holo = -1;
	for (;;)
	{
		holo = FindEntityByClassname(holo, "prop_dynamic");
		if (holo == -1)
		{
			break;
		}
		char strModel[512];
		GetEntPropString(holo, Prop_Data, "m_ModelName", strModel, 512);
		if (!StrEqual(strModel, "models/props_mvm/robot_hologram.mdl"))
		{
			continue;
		}
		if ((GetEntProp(holo, Prop_Send, "m_fEffects") & 32) != 0)
		{
			continue;
		}
		float flDistance = GetVectorDistance(WorldSpaceCenter(spawn), WorldSpaceCenter(holo));
		if ((flDistance <= flSmallestDistance) && IsPathToVectorPossible(actor, WorldSpaceCenter(holo)))
		{
			iBestEnt = holo;
			flSmallestDistance = flDistance;
		}
	}
	if (iBestEnt == -1)
	{
		return false;
	}
	CNavArea area = TheNavMesh.GetNearestNavArea(WorldSpaceCenter(iBestEnt), true, 1000.0, true, true, GetClientTeam(actor));
	if (area == NULL_AREA)
	{
		return false;
	}
	CNavArea_GetRandomPoint(area, m_vecGoalArea[actor]);
	m_flRepathTime[actor] = 0.0;
	return true;
}

stock bool FightsAtRange(int actor)
{
	switch (TF2_GetPlayerClass(actor))
	{
		case TFClass_Scout, TFClass_Pyro, TFClass_Spy:
		{
			return false;
		}
	}
	return true;
}

stock bool PickTheNest(int actor)
{
	int best = -1;
	float bestRange = 0.0;
	float mine[3];
	mine = WorldSpaceCenter(actor);
	int sentry = -1;
	for (;;)
	{
		sentry = FindEntityByClassname(sentry, "obj_sentrygun");
		if (sentry == -1)
		{
			break;
		}
		if (BaseEntity_GetTeamNumber(sentry) != GetClientTeam(actor))
		{
			continue;
		}
		if ((GetEntProp(sentry, Prop_Send, "m_bPlacing") != 0) || (GetEntProp(sentry, Prop_Send, "m_bCarried") != 0))
		{
			continue;
		}
		float sentryRange = GetVectorDistance(mine, WorldSpaceCenter(sentry));
		if ((best == -1) || (sentryRange < bestRange))
		{
			best = sentry;
			bestRange = sentryRange;
		}
	}
	if (best == -1)
	{
		return false;
	}
	CNavArea area = TheNavMesh.GetNearestNavArea(WorldSpaceCenter(best), true, 1000.0, true, true, GetClientTeam(actor));
	if (area == NULL_AREA)
	{
		return false;
	}
	CNavArea_GetRandomPoint(area, m_vecGoalArea[actor]);
	m_flRepathTime[actor] = 0.0;
	return true;
}

public Action CTFBotMoveToFront_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_iMoveToFrontTry[actor] = 0;
	m_bAtTheFront[actor] = false;
	m_ctMoveTimeout[actor] = GetGameTime() + MOVE_TO_FRONT_REACH;
	RecoverDefenderFromDisconnectedSpawn(actor);
	if (!PickTheFront(actor))
	{
		SetPlayerReady(actor, true);
		return action.Done("Cannot find the start of the robots' path from wherever we are");
	}
	return action.Continue();
}

public Action CTFBotMoveToFront_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (GameRules_GetRoundState() != RoundState_BetweenRounds)
	{
		return action.Done("The wave has started");
	}
	if (CTFBotCollectMoney_IsPossible(actor))
	{
		return action.SuspendFor(CTFBotCollectMoney(), "Money on the floor");
	}
	if (m_bAtTheFront[actor])
	{
		return action.Continue();
	}
	if (GetVectorDistance(m_vecGoalArea[actor], WorldSpaceCenter(actor)) < MOVE_TO_FRONT_ARRIVED)
	{
		SetPlayerReady(actor, true);
		m_bAtTheFront[actor] = true;
		return action.Continue();
	}
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	ILocomotion myLoco = myBot.GetLocomotionInterface();
	if (myLoco.IsStuck())
	{
		myLoco.ClearStuckStatus("Wedged on the way to the front");
		m_iMoveToFrontTry[actor]++;
		if (m_iMoveToFrontTry[actor] < MOVE_TO_FRONT_TRIES)
		{
			PickTheFront(actor);
		}
	}
	if ((m_iMoveToFrontTry[actor] >= MOVE_TO_FRONT_TRIES) || (m_ctMoveTimeout[actor] < GetGameTime()))
	{
		SetPlayerReady(actor, true);
		m_bAtTheFront[actor] = true;
		if (redbots_manager_debug_actions.BoolValue)
		{
			PrintToServer("[%8.3f] CTFBotMoveToFront(#%d): giving up short of the front", GetGameTime(), actor);
		}
		return action.Continue();
	}
	if (m_flRepathTime[actor] <= GetGameTime())
	{
		m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(3.0, 4.0);
		RepathToPos(actor, myBot, m_vecGoalArea[actor]);
	}
	if (PathFailedFor(actor))
	{
		NudgeTowardsGoal(actor, myBot, m_vecGoalArea[actor]);
	}
	else
	{
		m_pPath[actor].Update(myBot);
	}
	return action.Continue();
}

public void CTFBotMoveToFront_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_vecGoalArea[actor] = NULL_VECTOR;
	m_bAtTheFront[actor] = false;
	m_iMoveToFrontTry[actor] = 0;
}

public Action Command_DumpFront(int client, int args)
{
	BombInfo_t bomb;
	bool haveBomb = GetBombInfo(bomb);
	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || !IsPlayerAlive(i) || !IsDefenderBot(i))
		{
			continue;
		}
		float mine[3];
		mine = GetAbsOrigin(i);
		char action[512];
		strcopy(action, 512, "no waiting action");
		if (ActionsManager.LookupEntityActionByName(i, "DefenderMoveToFront") != INVALID_ACTION)
		{
			Format(action, 512, "walking to the front");
			if (m_bAtTheFront[i])
			{
				Format(action, 512, "holding the front");
			}
		}
		else
			if (ActionsManager.LookupEntityActionByName(i, "DefenderEngineerIdle") != INVALID_ACTION)
			{
				Format(action, 512, "at his nest");
			}
			else
				if (ActionsManager.LookupEntityActionByName(i, "DefenderGotoUpgrade") != INVALID_ACTION)
				{
					Format(action, 512, "walking to the station");
				}
				else
					if (ActionsManager.LookupEntityActionByName(i, "DefenderUpgrade") != INVALID_ACTION)
					{
						Format(action, 512, "shopping");
					}
		float fromGoal = -1.0;
		if (!IsZeroVector(m_vecGoalArea[i]))
		{
			fromGoal = GetVectorDistance(mine, m_vecGoalArea[i]);
		}
		float fromBomb = -1.0;
		if (haveBomb)
		{
			fromBomb = GetVectorDistance(mine, bomb.vPosition);
		}
		char shopped[512];
		strcopy(shopped, 512, "has not shopped");
		if (g_bShoppedThisBreak[i])
		{
			strcopy(shopped, 512, "has shopped");
		}
		char ready[512];
		strcopy(ready, 512, "not ready");
		if (IsPlayerReady(i))
		{
			strcopy(ready, 512, "ready");
		}
		ReplyToCommand(client, "%N (%s): %s, %.0f from his goal, %.0f from the bomb, %s, %s, stuck %d times, %d dead-end paths", i, g_sRawPlayerClassNames[TF2_GetPlayerClass(i)], action, fromGoal, fromBomb, shopped, ready, StuckCountOf(i), PathFailuresOf(i));
	}
	return Plugin_Handled;
}

stock void Go_ResetMoveToFront(int client)
{
	m_vecGoalArea[client] = NULL_VECTOR;
	m_ctMoveTimeout[client] = 0.0;
}

