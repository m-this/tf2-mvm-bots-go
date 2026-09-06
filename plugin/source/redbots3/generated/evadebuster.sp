BehaviorAction CTFBotEvadeBuster()
{
	BehaviorAction action = ActionsManager.Create("DefenderEvadeBuster");

	action.OnStart = CTFBotEvadeBuster_OnStart;
	action.Update = CTFBotEvadeBuster_Update;

	return action;
}

#define BUSTER_ESCAPE_SEARCH_RANGE (1500.0)

#define BUSTER_EVADE_MAX_TIME (8.0)

#define Go_maxAreas (256)

float m_ctEvadeBusterGiveUp[65];

// OnStart starts the clock and tells the team what is coming.
public Action CTFBotEvadeBuster_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	m_ctEvadeBusterGiveUp[actor] = GetGameTime() + BUSTER_EVADE_MAX_TIME;
	BaseMultiplayerPlayer_SpeakConceptIfAllowed(actor, MP_CONCEPT_PLAYER_INCOMING);
	return action.Continue();
}

// Update runs, until the clock runs out or there is nothing to run from.
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

// FindEscape is the ground furthest from the blast, of what the bot can see a way
// to from here.
//
// Furthest rather than first past a threshold, which is what this used to take: a
// buster standing in a corridor makes most of the areas within a radius worse than
// the one the bot is on, and the first one the collector happens to hand back is as
// likely to be the far side of the buster as the near side of the exit.
//
// No path is computed per candidate. One path query per area, at four a second,
// for every bot near a buster, costs more than picking a spot the bot cannot quite
// reach and being handed the next one a tenth of a second later.
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
	// The ground the bot is standing on, so that a bot with nowhere better still has an answer
	float bestDistance = GetVectorDistance(myOrigin, busterOrigin);
	int count = hAreas.Count();
	// Every wave has one buster and every bot near it runs this. The count is the map's, so cap it
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

// Threat is the buster this bot has to get away from, or -1.
//
// A buster that has started its detonation is a threat at blast range whatever it
// is doing. One that has not is a threat only when it is close enough that it
// could arrive before the bot is gone, which is what keeps a team from spending
// the wave backing away from a robot walking the length of the map.
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

// IsPossible says whether running is worth doing.
stock bool CTFBotEvadeBuster_IsPossible(int client)
{
	if (!IsPlayerAlive(client))
	{
		return false;
	}
	// A bot at the upgrade station is between waves and there is no buster walking towards it.
	// Leaving the station mid-purchase is also how a bot ends up owing the wave a ready-up
	if (TF2_IsInUpgradeZone(client))
	{
		return false;
	}
	return CTFBotEvadeBuster_Threat(client) != -1;
}

