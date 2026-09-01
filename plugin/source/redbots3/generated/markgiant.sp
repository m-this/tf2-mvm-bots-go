BehaviorAction CTFBotMarkGiant()
{
	BehaviorAction action = ActionsManager.Create("DefenderMarkGiant");

	action.OnStart = CTFBotMarkGiant_OnStart;
	action.Update = CTFBotMarkGiant_Update;
	action.OnEnd = CTFBotMarkGiant_OnEnd;

	return action;
}

#define Go_Slots (65)

#define Go_fanOWar (355)

int m_iTarget[65];
float m_flNextMarkTime[65];

public Action CTFBotMarkGiant_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	int weapon = GetMarkForDeathWeapon(actor);
	if (weapon == INVALID_ENT_REFERENCE)
	{
		return action.Done("Don't have a mark-for-death weapon");
	}
	ArrayList potentialVictims = new ArrayList();
	for (int i = 1; i <= MaxClients; i++)
	{
		if (i == actor)
		{
			continue;
		}
		if (!IsClientInGame(i))
		{
			continue;
		}
		if (IsPlayerMarkable(actor, i))
		{
			potentialVictims.Push(i);
		}
	}
	if (potentialVictims.Length == 0)
	{
		m_iTarget[actor] = -1;
		delete potentialVictims;
		return action.Done("No eligible mark victims");
	}
	m_iTarget[actor] = potentialVictims.Get(GetRandomInt(0, potentialVictims.Length - 1));
	EquipWeaponSlot(actor, TFWeaponSlot_Melee);
	delete potentialVictims;
	return action.Continue();
}

public Action CTFBotMarkGiant_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (!IsValidClientIndex(m_iTarget[actor]) || !IsPlayerAlive(m_iTarget[actor]))
	{
		m_iTarget[actor] = -1;
		return action.Done("Mark target is no longer valid");
	}
	if (!IsPlayerMarkable(actor, m_iTarget[actor]))
	{
		m_iTarget[actor] = -1;
		return action.Done("Mark target is no longer markable");
	}
	float myOrigin[3];
	GetClientAbsOrigin(actor, myOrigin);
	float targetOrigin[3];
	GetClientAbsOrigin(m_iTarget[actor], targetOrigin);
	float distToTarget = GetVectorDistance(myOrigin, targetOrigin);
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	if (distToTarget < 512.0)
	{
		IVision myVision = myBot.GetVisionInterface();
		if ((myVision.GetKnownCount(TFTeam_Blue) > 1) || (myVision.GetKnown(m_iTarget[actor]) == NULL_KNOWN_ENTITY))
		{
			myVision.ForgetAllKnownEntities();
			myVision.AddKnownEntity(m_iTarget[actor]);
		}
	}
	if (m_flRepathTime[actor] <= GetGameTime())
	{
		m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(1.0, 2.0);
		RepathToTarget(actor, myBot, m_iTarget[actor]);
	}
	m_pPath[actor].Update(myBot);
	return action.Continue();
}

public void CTFBotMarkGiant_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_flNextMarkTime[actor] = GetGameTime() + 30.0;
	m_iTarget[actor] = -1;
}

stock int GetMarkForDeathWeapon(int player)
{
	for (int i = 0; i < 8; i++)
	{
		int weapon = GetPlayerWeaponSlot(player, i);
		if (!IsValidEntity(weapon))
		{
			continue;
		}
		int itemDefinitionIndex = GetEntProp(weapon, Prop_Send, "m_iItemDefinitionIndex");
		if (itemDefinitionIndex == Go_fanOWar)
		{
			return weapon;
		}
	}
	return INVALID_ENT_REFERENCE;
}

stock bool IsPlayerMarkable(int bot, int victim)
{
	if (m_flNextMarkTime[bot] < GetGameTime())
	{
		return false;
	}
	if (!IsClientInGame(victim))
	{
		return false;
	}
	if (!IsPlayerAlive(victim))
	{
		return false;
	}
	if (BaseEntity_GetTeamNumber(bot) == BaseEntity_GetTeamNumber(victim))
	{
		return false;
	}
	if (!TF2_IsMiniBoss(victim))
	{
		return false;
	}
	if (IsSentryBusterRobot(victim))
	{
		return false;
	}
	if (TF2_IsPlayerInCondition(victim, TFCond_MarkedForDeath))
	{
		return false;
	}
	if (TF2_IsInvulnerable(victim))
	{
		return false;
	}
	return true;
}

stock bool CTFBotMarkGiant_IsPossible(int actor)
{
	if (GetMarkForDeathWeapon(actor) == INVALID_ENT_REFERENCE)
	{
		return false;
	}
	bool victimExists = false;
	for (int i = 1; i <= MaxClients; i++)
	{
		if (i == actor)
		{
			continue;
		}
		if (!IsClientConnected(i))
		{
			continue;
		}
		if (IsPlayerMarkable(actor, i))
		{
			victimExists = true;
		}
	}
	return victimExists;
}

stock void Go_ResetMarkGiant(int client)
{
	m_iTarget[client] = -1;
	m_flNextMarkTime[client] = 0.0;
}

