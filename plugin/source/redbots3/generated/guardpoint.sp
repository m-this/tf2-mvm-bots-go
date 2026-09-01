BehaviorAction CTFBotGuardPoint()
{
	BehaviorAction action = ActionsManager.Create("DefenderGuardPoint");

	action.OnStart = CTFBotGuardPoint_OnStart;
	action.Update = CTFBotGuardPoint_Update;
	action.OnEnd = CTFBotGuardPoint_OnEnd;
	action.OnTerritoryContested = CTFBotGuardPoint_OnTerritoryContested;
	action.OnTerritoryLost = CTFBotGuardPoint_OnTerritoryLost;

	return action;
}

#define Go_Slots (65)

float m_vecPointDefendArea[65][3];

public Action CTFBotGuardPoint_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	int point = GetCapturableAreaTrigger(GetPlayerEnemyTeam(actor));
	if (point == -1)
	{
		return action.ChangeTo(CTFBotDefenderAttack(), "No point found");
	}
	AreasCollector hAreas = TheNavMesh.CollectAreasInRadius(GetAbsOrigin(point), 300.0);
	for (int i = 0; i < hAreas.Count(); i++)
	{
		CTFNavArea area = hAreas.Get(i);
		if (area.HasAttributeTF(RED_SPAWN_ROOM) || area.HasAttributeTF(BLUE_SPAWN_ROOM))
		{
			continue;
		}
		float center[3];
		area.GetCenter(center);
		if (!IsPathToVectorPossible(actor, center))
		{
			continue;
		}
		m_vecPointDefendArea[actor] = center;
		break;
	}
	if (IsZeroVector(m_vecPointDefendArea[actor]))
	{
		delete hAreas;
		return action.ChangeTo(CTFBotDefenderAttack(), "NULL defense area");
	}
	BaseMultiplayerPlayer_SpeakConceptIfAllowed(actor, MP_CONCEPT_PLAYER_HELP);
	delete hAreas;
	return action.Continue();
}

public Action CTFBotGuardPoint_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	switch (TF2_GetPlayerClass(actor))
	{
		case TFClass_Soldier, TFClass_Pyro, TFClass_DemoMan:
		{
			if (CTFBotAttackTank_SelectTarget(actor))
			{
				return action.ChangeTo(CTFBotAttackTank(), "Tank priority");
			}
		}
	}
	if (CTFBotDefenderAttack_SelectTarget(actor))
	{
		return action.ChangeTo(CTFBotDefenderAttack(), "Something to fight");
	}
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	CKnownEntity threat = myBot.GetVisionInterface().GetPrimaryKnownThreat(false);
	if (threat != 0)
	{
		EquipBestWeaponForThreat(actor, threat);
	}
	int myWeapon = BaseCombatCharacter_GetActiveWeapon(actor);
	if ((myWeapon != -1) && ((TF2Util_GetWeaponID(myWeapon) == TF_WEAPON_FLAMETHROWER) || IsMeleeWeapon(myWeapon)))
	{
		int nearest = GetEnemyPlayerNearestToPosition(actor, m_vecPointDefendArea[actor], 1000.0);
		if (nearest != -1)
		{
			if (m_flRepathTime[actor] <= GetGameTime())
			{
				m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(0.5, 1.0);
				RepathToTarget(actor, myBot, nearest);
			}
			m_pPath[actor].Update(myBot);
			return action.Continue();
		}
	}
	if (myBot.IsRangeGreaterThanEx(m_vecPointDefendArea[actor], 200.0))
	{
		if (m_flRepathTime[actor] <= GetGameTime())
		{
			m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(1.0, 2.0);
			RepathToPos(actor, myBot, m_vecPointDefendArea[actor]);
		}
		m_pPath[actor].Update(myBot);
	}
	return action.Continue();
}

public void CTFBotGuardPoint_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_vecPointDefendArea[actor] = NULL_VECTOR;
}

public Action CTFBotGuardPoint_OnTerritoryContested(BehaviorAction action, int actor, int territory)
{
	if (redbots_manager_debug_actions.BoolValue)
	{
		PrintToChatAll("[OnTerritoryContested] Losing CP %d", GetControlPointByID(territory));
	}
	return action.TryToSustain();
}

public Action CTFBotGuardPoint_OnTerritoryLost(BehaviorAction action, int actor, int territory)
{
	if (redbots_manager_debug_actions.BoolValue)
	{
		PrintToChatAll("[OnTerritoryLost] Lost CP %d!", GetControlPointByID(territory));
	}
	return action.TryChangeTo(CTFBotDefenderAttack(), RESULT_CRITICAL, "Point lost");
}

stock bool CTFBotGuardPoint_IsPossible(int client)
{
	if (TF2_GetPlayerClass(client) == TFClass_Scout)
	{
		return false;
	}
	if (GetCountOfBotsWithNamedAction("DefenderGuardPoint") > 0)
	{
		return false;
	}
	if (GetCapturableAreaTrigger(GetPlayerEnemyTeam(client)) == -1)
	{
		return false;
	}
	if (IsFailureImminent(client))
	{
		return false;
	}
	return true;
}

stock void Go_ResetGuardPoint(int client)
{
	m_vecPointDefendArea[client] = NULL_VECTOR;
}

