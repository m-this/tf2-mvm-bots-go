BehaviorAction CTFBotCollectMoney()
{
	BehaviorAction action = ActionsManager.Create("DefenderCollectMoney");

	action.OnStart = CTFBotCollectMoney_OnStart;
	action.Update = CTFBotCollectMoney_Update;
	action.OnEnd = CTFBotCollectMoney_OnEnd;

	return action;
}

#define MONEY_URGENT_TIME (15.0)
#define MONEY_URGENT_WORTH (3000.0)

#define MONEY_ASK_INTERVAL (0.3)

int m_iCurrencyPack[65];
float m_ctMoneyAsk[65];

// OnStart aims the path and picks a pack.
public Action CTFBotCollectMoney_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	SelectCurrencyPack(actor);
	return action.Continue();
}

// Update walks to it.
public Action CTFBotCollectMoney_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	// TODO: if we're not a scout, see if we should attack instead if we have an active threat
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

// OnEnd forgets the pack.
public void CTFBotCollectMoney_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_iCurrencyPack[actor] = -1;
}

// TimeUntilRemoved is how long the pack has left.
stock float GetTimeUntilRemoved(int powerup)
{
	return CBaseEntity(powerup).GetNextThink("PowerupRemoveThink") - GetGameTime();
}

// IsCurrencyPackClaimed says whoever else is already walking at this one, so a
// heap is shared out instead of raced for.
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

// SelectCurrencyPack picks the cheapest pack to walk to, with a discount for
// one about to vanish.
stock int SelectCurrencyPack(int actor)
{
	// The held pack is re-asked on its own interval; losing it is what forces a fresh look
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

// IsValidCurrencyPack says the entity is still a money pack.
//
// The last two lines could be one return of a comparison. They are two because
// the shipped file is two, and a port that tidies as it goes cannot be diffed
// against what it replaces.
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

// IsPossible says whether collecting is worth doing.
stock bool CTFBotCollectMoney_IsPossible(int actor)
{
	// One of them in a wave, all of them in the break
	//
	// Mid-wave the money is a distraction from the robots walking a bomb up the map, so one goes and
	// the rest keep shooting. Between waves there is nothing else to do with the time, and one bot
	// clearing a heap on his own does not finish before the break does.
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

// ResetCollectMoney forgets the credit this bot was walking to.
//
// A bot leaving takes its seat's state with it, and the next bot in that seat
// is a different bot.
stock void Go_ResetCollectMoney(int client)
{
	m_iCurrencyPack[client] = -1;
}

