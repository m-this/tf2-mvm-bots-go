int m_iCurrencyPack[MAXPLAYERS + 1];

BehaviorAction CTFBotCollectMoney()
{
	BehaviorAction action = ActionsManager.Create("DefenderCollectMoney");
	
	action.OnStart = CTFBotCollectMoney_OnStart;
	action.Update = CTFBotCollectMoney_Update;
	action.OnEnd = CTFBotCollectMoney_OnEnd;
	
	return action;
}

public Action CTFBotCollectMoney_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	
	SelectCurrencyPack(actor);
	
	return action.Continue();
}

public Action CTFBotCollectMoney_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	//TODO: if we're not a scout, see if we should attack instead if we have an active threat
	
	if (!IsValidCurrencyPack(m_iCurrencyPack[actor])) 
		return action.Done("No credits to collect");
	
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

float GetTimeUntilRemoved(int powerup)
{
	return CBaseEntity(powerup).GetNextThink("PowerupRemoveThink") - GetGameTime();
}

/* How long a pack has left before it is worth crossing the map for, and what that is worth

Nearest first, because at the end of a wave the money is in a heap where the last robot died and
the whole point is to clear the heap. It used to be soonest-to-vanish first, and a pack with more
than thirty seconds left was not a candidate at all: freshly dropped cash has its whole lifetime
in front of it, so nothing on the floor qualified until the last thirty seconds of it, by which
time everybody was stood at the front. Big stacks walked past, reported from play.

One about to go still jumps the queue, priced as a discount on the walk rather than as a rule of
its own, so the bot nearest to it is still the one that goes. */
#define MONEY_URGENT_TIME	15.0
#define MONEY_URGENT_WORTH	3000.0

//Bounded, because this is asked every frame by the gate below and a heap of cash is a heap of entities
#define MONEY_ASK_INTERVAL	0.3

static float m_ctMoneyAsk[MAXPLAYERS + 1];

//Whoever else is already walking at this one, so a heap is shared out instead of raced for
static bool IsCurrencyPackClaimed(int actor, int pack)
{
	for (int i = 1; i <= MaxClients; i++)
	{
		if (i == actor || !IsClientInGame(i))
			continue;
		
		if (m_iCurrencyPack[i] == pack)
			return true;
	}
	
	return false;
}

int SelectCurrencyPack(int actor)
{
	//The held pack is re-asked on its own interval; losing it is what forces a fresh look
	if (IsValidCurrencyPack(m_iCurrencyPack[actor]) && m_ctMoneyAsk[actor] > GetGameTime())
		return m_iCurrencyPack[actor];
	
	m_ctMoneyAsk[actor] = GetGameTime() + MONEY_ASK_INTERVAL;
	
	int iBestPack = INVALID_ENT_REFERENCE;
	float flBestCost = -1.0;
	
	float myOrigin[3]; myOrigin = GetAbsOrigin(actor);
	
	int x = INVALID_ENT_REFERENCE; 
	while ((x = FindEntityByClassname(x, "item_currency*")) != -1)
	{
		bool bDistributed = !!GetEntProp(x, Prop_Send, "m_bDistributed");
		
		if (bDistributed)
			continue;
		
		if (!(GetEntityFlags(x) & FL_ONGROUND))
			continue;
		
		if (IsCurrencyPackClaimed(actor, x))
			continue;
		
		float flCost = GetVectorDistance(myOrigin, WorldSpaceCenter(x));
		
		if (GetTimeUntilRemoved(x) < MONEY_URGENT_TIME)
			flCost -= MONEY_URGENT_WORTH;
		
		if (flBestCost < 0.0 || flCost < flBestCost)
		{
			flBestCost = flCost;
			iBestPack = x;
		}
	}

	m_iCurrencyPack[actor] = iBestPack;
	return iBestPack;
}

bool IsValidCurrencyPack(int pack)
{
	if (!IsValidEntity(pack))
		return false;

	char class[64]; GetEntityClassname(pack, class, sizeof(class));
	
	if (StrContains(class, "item_currency", false) == -1)
		return false;
	
	return true;
}

bool CTFBotCollectMoney_IsPossible(int actor)
{	
	/* One of them in a wave, all of them in the break
	
	Mid-wave the money is a distraction from the robots walking a bomb up the map, so one goes and
	the rest keep shooting. Between waves there is nothing else to do with the time, and one bot
	clearing a heap on his own does not finish before the break does. */
	if (GameRules_GetRoundState() != RoundState_BetweenRounds
		&& GetCountOfBotsWithNamedAction("DefenderCollectMoney") > 0)
		return false;
	
	if (!IsValidCurrencyPack(SelectCurrencyPack(actor)))
		return false;
	
	return true;
}