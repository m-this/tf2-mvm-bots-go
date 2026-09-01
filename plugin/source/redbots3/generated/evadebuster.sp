BehaviorAction CTFBotEvadeBuster()
{
	BehaviorAction action = ActionsManager.Create("DefenderEvadeBuster");

	action.OnStart = CTFBotEvadeBuster_OnStart;
	action.Update = CTFBotEvadeBuster_Update;

	return action;
}

#define Go_Slots (65)

#define BUSTER_ESCAPE_SEARCH_RANGE (1500.0)

#define BUSTER_EVADE_MAX_TIME (8.0)

#define Go_maxAreas (256)

float m_ctEvadeBusterGiveUp[65];

public Action CTFBotEvadeBuster_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	m_ctEvadeBusterGiveUp[actor] = GetGameTime() + BUSTER_EVADE_MAX_TIME;
	BaseMultiplayerPlayer_SpeakConceptIfAllowed(actor, MP_CONCEPT_PLAYER_INCOMING);
	return action.Continue();
}

public Action CTFBotEvadeBuster_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (m_ctEvadeBusterGiveUp[actor] < GetGameTime())
	{
		return action.Done("Ran from the buster for long enough");
	}
	int buster = CTFBotEvadeBuster_Threat(actor);
	if (buster == -1)
	{
		return action.Done("No buster to run from");
	}
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	float busterOrigin[3];
	busterOrigin = WorldSpaceCenter(buster);
	float escape[3];
	bool found = CTFBotEvadeBuster_FindEscape(actor, busterOrigin, escape);
	if (!found)
	{
		return action.Done("Nowhere to run");
	}
	if (m_flRepathTime[actor] <= GetGameTime())
	{
		m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(0.3, 0.4);
		RepathToPos(actor, myBot, escape);
	}
	m_pPath[actor].Update(myBot);
	return action.Continue();
}

stock bool CTFBotEvadeBuster_FindEscape(int actor, float busterOrigin[3], float escape[3])
{
	bool found;
	for (int i = 0; i < 3; i++)
	{
		escape[i] = 0.0;
	}
	float myOrigin[3];
	GetClientAbsOrigin(actor, myOrigin);
	AreasCollector hAreas = TheNavMesh.CollectAreasInRadius(myOrigin, BUSTER_ESCAPE_SEARCH_RANGE);
	float bestDistance = GetVectorDistance(myOrigin, busterOrigin);
	int count = hAreas.Count();
	if (count > Go_maxAreas)
	{
		count = Go_maxAreas;
	}
	for (int i = 0; i < count; i++)
	{
		CNavArea area = hAreas.Get(i);
		float center[3];
		area.GetCenter(center);
		float distance = GetVectorDistance(center, busterOrigin);
		if (distance <= bestDistance)
		{
			continue;
		}
		bestDistance = distance;
		escape = center;
		found = true;
	}
	delete hAreas;
	return found;
}

stock int CTFBotEvadeBuster_Threat(int client)
{
	float myOrigin[3];
	GetClientAbsOrigin(client, myOrigin);
	TFTeam enemyTeam = GetPlayerEnemyTeam(client);
	if (IsValidClientIndex(g_iDetonatingPlayer) && IsPlayerAlive(g_iDetonatingPlayer) && (TF2_GetClientTeam(g_iDetonatingPlayer) == enemyTeam))
	{
		float theirOrigin[3];
		GetClientAbsOrigin(g_iDetonatingPlayer, theirOrigin);
		if (GetVectorDistance(myOrigin, theirOrigin) <= (BUSTER_BLAST_RANGE * 2.0))
		{
			return g_iDetonatingPlayer;
		}
	}
	return FindSentryBusterNear(myOrigin, enemyTeam, BUSTER_FLEE_RANGE);
}

stock bool CTFBotEvadeBuster_IsPossible(int client)
{
	if (!IsPlayerAlive(client))
	{
		return false;
	}
	if (TF2_IsInUpgradeZone(client))
	{
		return false;
	}
	return CTFBotEvadeBuster_Threat(client) != -1;
}

