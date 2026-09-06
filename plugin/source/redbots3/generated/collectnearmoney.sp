BehaviorAction CTFBotCollectNearMoney()
{
	BehaviorAction action = ActionsManager.Create("DefenderCollectNearMoney");

	action.OnStart = CTFBotCollectNearMoney_OnStart;
	action.Update = CTFBotCollectNearMoney_Update;
	action.OnEnd = CTFBotCollectNearMoney_OnEnd;

	return action;
}

// OnStart aims the path. The pack was picked before the action started.
public Action CTFBotCollectNearMoney_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	// NOTE: we pick a money pack before entering this action
	return action.Continue();
}

// Update walks to the pack, and gives up the moment anything is worth fighting.
public Action CTFBotCollectNearMoney_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (!IsValidCurrencyPack(m_iCurrencyPack[actor]))
	{
		return action.Done("No money");
	}
	INextBot myBot = CBaseNPC_GetNextBotOfEntity(actor);
	CKnownEntity threat = myBot.GetVisionInterface().GetPrimaryKnownThreat(false);
	if (threat != 0)
	{
		return action.Done("I see a threat");
	}
	if (m_flRepathTime[actor] <= GetGameTime())
	{
		m_flRepathTime[actor] = GetGameTime() + GetRandomFloat(0.3, 1.0);
		RepathToPos(actor, myBot, WorldSpaceCenter(m_iCurrencyPack[actor]));
	}
	m_pPath[actor].Update(myBot);
	return action.Continue();
}

// OnEnd forgets the pack, so the next behaviour does not inherit it.
public void CTFBotCollectNearMoney_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_iCurrencyPack[actor] = -1;
}

// SelectTarget picks a pack, and refuses to while there is a threat about.
stock bool CTFBotCollectNearMoney_SelectTarget(int client)
{
	CKnownEntity threat = CBaseNPC_GetNextBotOfEntity(client).GetVisionInterface().GetPrimaryKnownThreat(false);
	if (threat != 0)
	{
		return false;
	}
	m_iCurrencyPack[client] = GetNearestCurrencyPack(client, 6000.0);
	return m_iCurrencyPack[client] != -1;
}

