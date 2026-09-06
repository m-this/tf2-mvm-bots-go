BehaviorAction CTFBotMoveToFront()
{
	BehaviorAction action = ActionsManager.Create("DefenderMoveToFront");

	action.OnStart = CTFBotMoveToFront_OnStart;
	action.Update = CTFBotMoveToFront_Update;
	action.OnEnd = CTFBotMoveToFront_OnEnd;

	return action;
}

#define MOVE_TO_FRONT_ARRIVED (80.0)
#define MOVE_TO_FRONT_REACH (60.0)
#define MOVE_TO_FRONT_TRIES (3)

float m_vecGoalArea[65][3];
float m_ctMoveTimeout[65];
int m_iMoveToFrontTry[65];
bool m_bAtTheFront[65];

// IsWaitingAtTheFront says whether this bot has finished taking up its position
// for the coming wave.
//
// Standing where he meant to stand and giving up short of it are the same answer
// here: both mean he has stopped walking and is not going to move again before the
// wave.
stock bool IsWaitingAtTheFront(int client)
{
	return m_bAtTheFront[client];
}

// PickTheFront is where the robots come out, which is where the team should be
// waiting for them.
//
// The holograms are the markers the game puts at the robot spawns, so the one
// nearest the enemy spawn room is the start of the bomb's path. Standing on the
// ground beside it is the difference between opening fire as the gate drops and
// meeting the wave halfway up the map.
stock bool PickTheFront(int actor)
{
	// The classes that shoot from a distance wait at the nest, the rest at the gate
	//
	// The gate is where the robots come out, and standing on it is how a defender meets a giant with
	// nothing behind him. Waiting beside the sentry instead starts the wave with a sentry, a dispenser
	// and the rest of the team in reach, and it is worth nothing to a Scout who has money to collect
	// or a Pyro who has to be within a few metres to do anything at all.
	//
	// Holding the nest with the whole team was measured first and could not be told apart from the
	// gate: four waves an arm, and the difference sat inside each arm's own spread.
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
	// A new goal is worth a path this frame rather than at the end of the old one's interval
	m_flRepathTime[actor] = 0.0;
	return true;
}

// FightsAtRange says whether this one does its damage from where the nest is, or
// has to walk into the wave.
//
// Asked for from play: the classes that fight at range belong around the
// engineer's nest, and the ones that have to close belong at the gate. The Scout
// collects money and the Pyro and the Spy work at arm's length, so all three are
// wasted standing behind a sentry. Everybody else shoots across the same ground the
// sentry covers.
//
// This replaced holding the nest with the whole team, which was measured and could
// not be told apart from the gate at four waves an arm.
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

// PickTheNest is ground beside a teammate's sentry, or false when the team has
// none up yet.
//
// The nearest one, because two engineers are two nests and the one to stand at is
// the one on the way to where this bot already is.
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

// OnStart picks the front and gives up at once when there is none to pick.
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

// Update walks there, and stops when the wave starts rather than when it
// arrives.
public Action CTFBotMoveToFront_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	// The wave is what ends this, not arriving
	//
	// Arriving used to end it, and what happened next was nothing at all: the between-rounds branch
	// of GetDesiredBotAction had no answer for a bot that had already shopped, so the game got the
	// bot back and roamed it around the map. Reported as the Heavy, the Medic and the Pyro wandering
	// off before the wave and turning up inside the middle house on Coaltown.
	if (GameRules_GetRoundState() != RoundState_BetweenRounds)
	{
		return action.Done("The wave has started");
	}
	// Credits on the floor are still worth the walk while we wait
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
	// Walking into the corner of a building is what spends an attempt
	//
	// The locomotion already knows the difference between walking and walking on the spot, and
	// nothing outside the engineer has ever asked it. A fresh random point in the same area is a
	// different approach to the same place, and three of them is a bound rather than a bot that
	// repaths for ever.
	//
	// Out of attempts, or out of clock, he stands where he is: short of the front is a bot in the
	// wrong place, and handed back to the game is a bot in the middle house.
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

// OnEnd forgets the goal.
public void CTFBotMoveToFront_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_vecGoalArea[actor] = NULL_VECTOR;
	m_bAtTheFront[actor] = false;
	m_iMoveToFrontTry[actor] = 0;
}

// DumpFront prints where each bot is and what it is waiting on.
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
		// four lookups by name, and SourcePawn has no switch over one
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

// ResetMoveToFront forgets where this bot was walking, and how long it had.
//
// A bot leaving takes its seat's state with it, and the next bot in that seat
// is a different bot.
stock void Go_ResetMoveToFront(int client)
{
	m_vecGoalArea[client] = NULL_VECTOR;
	m_ctMoveTimeout[client] = 0.0;
}

