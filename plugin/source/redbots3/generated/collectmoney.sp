BehaviorAction CTFBotCollectMoney()
{
	BehaviorAction action = ActionsManager.Create("DefenderCollectMoney");

	action.OnStart = CTFBotCollectMoney_OnStart;
	action.Update = CTFBotCollectMoney_Update;
	action.OnEnd = CTFBotCollectMoney_OnEnd;

	return action;
}

#define Go_Slots (65)

#define MONEY_URGENT_TIME (15.0)
#define MONEY_URGENT_WORTH (3000.0)

#define MONEY_ASK_INTERVAL (0.3)

int m_iCurrencyPack[65];
float m_ctMoneyAsk[65];

public Action CTFBotCollectMoney_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	SelectCurrencyPack(actor);
	return action.Continue();
}

public Action CTFBotCollectMoney_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (!IsValidCurrencyPack(m_iCurrencyPack[actor]))
	{
		return action.Done("No credits to collect");
	}
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	if (m_flRepathTime[actor] <= GetGameTime())
	{
		m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(1.0, 2.0);
		RepathToPos(actor, myBot, WorldSpaceCenter(m_iCurrencyPack[actor]));
	}
	m_pPath[actor].Update(myBot);
	return action.Continue();
}

public void CTFBotCollectMoney_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_iCurrencyPack[actor] = -1;
}

stock float GetTimeUntilRemoved(int powerup)
{
	return CBaseEntity(powerup).GetNextThink("PowerupRemoveThink") - GetGameTime();
}

stock bool IsCurrencyPackClaimed(int actor, int pack)
{
	for (int i = 1; i <= MaxClients; i++)
	{
		if ((i == actor) || !IsClientInGame(i))
		{
			continue;
		}
		if (m_iCurrencyPack[i] == pack)
		{
			return true;
		}
	}
	return false;
}

stock int SelectCurrencyPack(int actor)
{
	if (IsValidCurrencyPack(m_iCurrencyPack[actor]) && (m_ctMoneyAsk[actor] > GetGameTime()))
	{
		return m_iCurrencyPack[actor];
	}
	m_ctMoneyAsk[actor] = GetGameTime() + MONEY_ASK_INTERVAL;
	int iBestPack = INVALID_ENT_REFERENCE;
	float flBestCost = -1.0;
	float myOrigin[3];
	myOrigin = GetAbsOrigin(actor);
	int x = INVALID_ENT_REFERENCE;
	for (;;)
	{
		x = FindEntityByClassname(x, "item_currency*");
		if (x == -1)
		{
			break;
		}
		bool bDistributed = GetEntProp(x, Prop_Send, "m_bDistributed") != 0;
		if (bDistributed)
		{
			continue;
		}
		if ((GetEntityFlags(x) & FL_ONGROUND) == 0)
		{
			continue;
		}
		if (IsCurrencyPackClaimed(actor, x))
		{
			continue;
		}
		float flCost = GetVectorDistance(myOrigin, WorldSpaceCenter(x));
		if (GetTimeUntilRemoved(x) < MONEY_URGENT_TIME)
		{
			flCost -= MONEY_URGENT_WORTH;
		}
		if ((flBestCost < 0.0) || (flCost < flBestCost))
		{
			flBestCost = flCost;
			iBestPack = x;
		}
	}
	m_iCurrencyPack[actor] = iBestPack;
	return iBestPack;
}

stock bool IsValidCurrencyPack(int pack)
{
	if (!IsValidEntity(pack))
	{
		return false;
	}
	char class[512];
	GetEntityClassname(pack, class, 512);
	if (StrContains(class, "item_currency", false) == -1)
	{
		return false;
	}
	return true;
}

stock bool CTFBotCollectMoney_IsPossible(int actor)
{
	if ((GameRules_GetRoundState() != RoundState_BetweenRounds) && (GetCountOfBotsWithNamedAction("DefenderCollectMoney") > 0))
	{
		return false;
	}
	if (!IsValidCurrencyPack(SelectCurrencyPack(actor)))
	{
		return false;
	}
	return true;
}

stock void Go_ResetCollectMoney(int client)
{
	m_iCurrencyPack[client] = -1;
}

