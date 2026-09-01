BehaviorAction CTFBotSpyCheck()
{
	BehaviorAction action = ActionsManager.Create("DefenderSpyCheck");

	action.OnStart = CTFBotSpyCheck_OnStart;
	action.Update = CTFBotSpyCheck_Update;

	return action;
}

#define Go_Slots (65)

#define SPY_PARANOIA_SPEED (320.0)

#define SPY_PARANOIA_RANGE_MAX (2000.0)

#define SPY_PARANOIA_MEMORY (20.0)

#define SPY_CHECK_MIN_TIME (2.0)
#define SPY_CHECK_MAX_TIME (5.0)

#define SPY_CHECK_REACH (80.0)

#define SPY_CHECK_LOOK_INTERVAL (0.1)

#define SPY_BEHIND_RANGE (400.0)

#define SPY_BEHIND_TIME (0.2)

#define SPY_GLANCE_INTERVAL_MIN (1.6)
#define SPY_GLANCE_INTERVAL_MAX (3.2)
#define SPY_GLANCE_TIME (0.35)
#define SPY_GLANCE_RANGE (220.0)

float g_flLastSpySeenTime;
float g_vLastSpySeen[3];
float m_ctSpyCheckEnd[65];
float m_ctSpyCheckNextLook[65];
int m_iSpyCheckSuspect[65];
bool m_bSpyCheckHit[65];
bool m_bSpyCheckSeen[65][65];
float m_ctSpyBehindSince[65];
float m_ctNextSpyGlance[65];

stock void NoteSpySighting(float origin[3])
{
	g_flLastSpySeenTime = GetGameTime();
	g_vLastSpySeen = origin;
}

stock void ResetSpyIntel()
{
	g_flLastSpySeenTime = 0.0;
	g_vLastSpySeen = NULL_VECTOR;
}

stock bool IsInSpyParanoiaRange(int client)
{
	if (g_flLastSpySeenTime <= 0.0)
	{
		return false;
	}
	float elapsed = GetGameTime() - g_flLastSpySeenTime;
	if (elapsed > SPY_PARANOIA_MEMORY)
	{
		return false;
	}
	float reach = MinFloat(elapsed * SPY_PARANOIA_SPEED, SPY_PARANOIA_RANGE_MAX);
	float myOrigin[3];
	GetClientAbsOrigin(client, myOrigin);
	return GetVectorDistance(myOrigin, g_vLastSpySeen) <= reach;
}

public Action CTFBotSpyCheck_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	m_ctSpyCheckEnd[actor] = GetGameTime() + GetRandomFloat(SPY_CHECK_MIN_TIME, SPY_CHECK_MAX_TIME);
	m_ctSpyCheckNextLook[actor] = 0.0;
	m_iSpyCheckSuspect[actor] = -1;
	m_bSpyCheckHit[actor] = false;
	SnapshotVisibleTeammates(actor);
	BaseMultiplayerPlayer_SpeakConceptIfAllowed(actor, MP_CONCEPT_PLAYER_CLOAKEDSPY);
	return action.Continue();
}

public Action CTFBotSpyCheck_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (m_ctSpyCheckEnd[actor] < GetGameTime())
	{
		return action.Done("Checked for long enough");
	}
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	CKnownEntity threat = myBot.GetVisionInterface().GetPrimaryKnownThreat(true);
	if (threat != 0)
	{
		return action.Done("Something real to shoot at");
	}
	int suspect = m_iSpyCheckSuspect[actor];
	if (IsValidClientIndex(suspect) && !IsPlayerAlive(suspect))
	{
		suspect = -1;
	}
	if (suspect == -1)
	{
		if (m_ctSpyCheckNextLook[actor] < GetGameTime())
		{
			m_ctSpyCheckNextLook[actor] = GetGameTime() + SPY_CHECK_LOOK_INTERVAL;
			suspect = FindTeammateWhoWasNotThere(actor);
			if (suspect != -1)
			{
				m_ctSpyCheckEnd[actor] = GetGameTime() + GetRandomFloat(SPY_CHECK_MIN_TIME, SPY_CHECK_MAX_TIME);
			}
		}
		m_iSpyCheckSuspect[actor] = suspect;
	}
	if (suspect == -1)
	{
		return action.Continue();
	}
	if (GetTimeSinceWeaponFired(suspect) < 1.0)
	{
		m_iSpyCheckSuspect[actor] = -1;
		return action.Done("The suspect is fighting");
	}
	IBody myBody = myBot.GetBodyInterface();
	AimHeadTowards(myBody, WorldSpaceCenter(suspect), CRITICAL, 0.5, Address_Null, "Spy check");
	float spyRange = GetVectorDistance(GetAbsOrigin(actor), GetAbsOrigin(suspect));
	if (spyRange > SPY_CHECK_REACH)
	{
		if (m_flRepathTime[actor] <= GetGameTime())
		{
			m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(0.3, 0.4);
			RepathToTarget(actor, myBot, suspect);
		}
		m_pPath[actor].Update(myBot);
	}
	if (myBody.IsHeadAimingOnTarget())
	{
		VS_PressFireButton(actor);
	}
	if (spyRange < SPY_CHECK_REACH)
	{
		if (!m_bSpyCheckHit[actor])
		{
			m_bSpyCheckHit[actor] = true;
			m_ctSpyCheckEnd[actor] = GetGameTime() + GetRandomFloat(0.5, 1.5);
		}
	}
	return action.Continue();
}

stock void SnapshotVisibleTeammates(int actor)
{
	IVision myVision = CBaseNPC_GetNextBotOfEntity(actor).GetVisionInterface();
	TFTeam myTeam = TF2_GetClientTeam(actor);
	for (int i = 1; i <= MaxClients; i++)
	{
		m_bSpyCheckSeen[actor][i] = IsClientInGame(i) && (i != actor) && IsPlayerAlive(i) && (TF2_GetClientTeam(i) == myTeam) && myVision.IsAbleToSeeTarget(i, USE_FOV);
	}
}

stock int FindTeammateWhoWasNotThere(int actor)
{
	IVision myVision = CBaseNPC_GetNextBotOfEntity(actor).GetVisionInterface();
	TFTeam myTeam = TF2_GetClientTeam(actor);
	for (int i = 1; i <= MaxClients; i++)
	{
		if ((i == actor) || !IsClientInGame(i) || !IsPlayerAlive(i))
		{
			continue;
		}
		if (TF2_GetClientTeam(i) != myTeam)
		{
			continue;
		}
		if (!IsFakeClient(i))
		{
			m_bSpyCheckSeen[actor][i] = true;
			continue;
		}
		if (m_bSpyCheckSeen[actor][i])
		{
			continue;
		}
		if (!myVision.IsAbleToSeeTarget(i, USE_FOV))
		{
			continue;
		}
		if (TF2_HasTheFlag(i))
		{
			m_bSpyCheckSeen[actor][i] = true;
			continue;
		}
		m_bSpyCheckSeen[actor][i] = true;
		return i;
	}
	return -1;
}

stock void UpdateSpyGlance(int client)
{
	if (!Feature(FEATURE_SPY_GLANCE) || !IsInSpyParanoiaRange(client))
	{
		m_ctNextSpyGlance[client] = 0.0;
		return;
	}
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(client);
	if (myBot.GetVisionInterface().GetPrimaryKnownThreat(true) != 0)
	{
		return;
	}
	if (m_ctNextSpyGlance[client] > GetGameTime())
	{
		return;
	}
	m_ctNextSpyGlance[client] = GetGameTime() + GetRandomFloat(SPY_GLANCE_INTERVAL_MIN, SPY_GLANCE_INTERVAL_MAX);
	float myAngles[3];
	GetClientEyeAngles(client, myAngles);
	float myForward[3];
	GetAngleVectors(myAngles, myForward, NULL_VECTOR, NULL_VECTOR);
	float behind[3];
	behind = GetEyePosition(client);
	behind[0] -= myForward[0] * SPY_GLANCE_RANGE;
	behind[1] -= myForward[1] * SPY_GLANCE_RANGE;
	behind[2] -= myForward[2] * SPY_GLANCE_RANGE;
	AimHeadTowards(myBot.GetBodyInterface(), behind, IMPORTANT, SPY_GLANCE_TIME, Address_Null, "Checking behind me");
}

stock void UpdateSpyIntel(int client)
{
	UpdateSpyGlance(client);
	IVision myVision = CBaseNPC_GetNextBotOfEntity(client).GetVisionInterface();
	TFTeam enemyTeam = GetPlayerEnemyTeam(client);
	float myOrigin[3];
	GetClientAbsOrigin(client, myOrigin);
	float myAngles[3];
	GetClientEyeAngles(client, myAngles);
	float myForward[3];
	GetAngleVectors(myAngles, myForward, NULL_VECTOR, NULL_VECTOR);
	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || !IsPlayerAlive(i))
		{
			continue;
		}
		if ((TF2_GetClientTeam(i) != enemyTeam) || (TF2_GetPlayerClass(i) != TFClass_Spy))
		{
			continue;
		}
		if (TF2_IsStealthed(i) && !IsCloakedPlayerExposed(i))
		{
			continue;
		}
		float theirOrigin[3];
		GetClientAbsOrigin(i, theirOrigin);
		if (GetVectorDistance(myOrigin, theirOrigin) <= SPY_BEHIND_RANGE)
		{
			float toThem[3];
			SubtractVectors(theirOrigin, myOrigin, toThem);
			NormalizeVector(toThem, toThem);
			if (GetVectorDotProduct(myForward, toThem) < 0.0)
			{
				if (m_ctSpyBehindSince[client] <= 0.0)
				{
					m_ctSpyBehindSince[client] = GetGameTime();
				}
				else
					if ((GetGameTime() - m_ctSpyBehindSince[client]) >= SPY_BEHIND_TIME)
					{
						m_ctSpyBehindSince[client] = 0.0;
						myVision.AddKnownEntity(i);
						NoteSpySighting(theirOrigin);
					}
				continue;
			}
		}
		if (!TF2_IsPlayerInCondition(i, TFCond_Disguised) && myVision.IsAbleToSeeTarget(i, USE_FOV))
		{
			NoteSpySighting(theirOrigin);
		}
	}
}

stock void ResetSpyCheck(int client)
{
	m_ctSpyBehindSince[client] = 0.0;
	m_ctSpyCheckEnd[client] = 0.0;
	m_iSpyCheckSuspect[client] = -1;
}

stock bool CTFBotSpyCheck_IsPossible(int client)
{
	if (!IsPlayerAlive(client) || TF2_IsInUpgradeZone(client))
	{
		return false;
	}
	if (CBaseNPC_GetNextBotOfEntity(client).GetVisionInterface().GetPrimaryKnownThreat(true) != 0)
	{
		return false;
	}
	if (TF2_GetPlayerClass(client) == TFClass_Engineer)
	{
		return false;
	}
	return IsInSpyParanoiaRange(client);
}

