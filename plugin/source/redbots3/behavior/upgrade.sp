#define BUY_UPGRADES_FAST_MAX_TIME	3.0

BehaviorAction CTFBotUpgrade()
{
	BehaviorAction action = ActionsManager.Create("DefenderUpgrade");
	
	action.OnStart = CTFBotUpgrade_OnStart;
	action.Update = CTFBotUpgrade_Update;
	action.OnEnd = CTFBotUpgrade_OnEnd;
	
	return action;
}

/* How much of one shopping trip a single attribute may take

A play-test bundle came back with a medic who bought ubercharge rate twenty five times across a
mission, ten of them inside thirty seconds, and nothing else all run: no health, no resistance,
no other line on the medigun. Whether the game is failing to refuse the tiers or the ranking is
never getting past the first line, a bot that spends a whole wallet on one attribute has bought
the wrong thing, and this stops it either way.

Half, and never less than two steps of whatever it is: a small wallet that could only afford one
thing would otherwise buy nothing at all. */

/* What the game refused this trip, so the next choice is a different one
 *
 * A refusal used to end the whole shopping trip, on the reasoning that the ranking would pick the
 * same line again and be refused until the window ran out. It would, and the answer is to remember
 * the refusal rather than to stop shopping: ten of forty five trips measured on Bavarian Botbash
 * ended this way, with money still in the wallet and a list still worth walking. */
public Action CTFBotUpgrade_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	
	//The wallet this trip is measured against, and nothing spent out of it yet
	m_iSessionWallet[actor] = TF2_GetCurrency(actor);

	for (int i = 0; i < MAX_UPGRADES; i++)
		m_bRefusedUpgrade[actor][i] = false;
	
	for (int index = 0; index < MAX_UPGRADES; index++)
		m_iSpentOnUpgrade[actor][index] = 0;
	
	if (!TF2_IsInUpgradeZone(actor)) 
		return action.ChangeTo(CTFBotGotoUpgrade(), "Not standing at an upgrade station!");
	
	CollectUpgrades(actor);
	
	KV_MvM_UpgradesBegin(actor);
	
	m_flNextUpgrade[actor] = GetGameTime() + GetUpgradeInterval();
	
	bool isRoundActive = GameRules_GetRoundState() == RoundState_RoundRunning;
	
	//How long should it take us to buy upgrades?
	if (g_bHasUpgraded[actor] == false && isRoundActive)
	{
		//We probably just joined during an active game
		m_flUpgradingTime[actor] = GetGameTime() + 15.0;
	}
	else
	{
		//spend less time upgrading during the round, normal otherwise
		m_flUpgradingTime[actor] = GetGameTime() + (isRoundActive ? BUY_UPGRADES_FAST_MAX_TIME : BUY_UPGRADES_MAX_TIME);
	}
	
	return action.Continue();
}
public Action CTFBotUpgrade_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (!TF2_IsInUpgradeZone(actor)) 
		return action.ChangeTo(CTFBotGotoUpgrade(), "Not standing at an upgrade station!");
	
	if (m_flUpgradingTime[actor] <= GetGameTime())
	{
		//It shouldn't take us this long to upgrade...
		
		SetPlayerReady(actor, true);
		
		LogUpgradeSessionEnd(actor, "the window ran out");
		
		return GetUpgradePostAction(actor, action);
	}
	
	float flNextTime = m_flNextUpgrade[actor] - GetGameTime();
	
	if (flNextTime <= 0.0)
	{
		m_flNextUpgrade[actor] = GetGameTime() + GetUpgradeInterval();
		
		int row = CTFBotPurchaseUpgrades_ChooseUpgrade(actor);
		
		if (row != -1) 
		{
			bool purchased = CTFBotPurchaseUpgrades_PurchaseUpgrade(actor, row);
			
			if (redbots_manager_debug_actions.BoolValue)
				PrintToChatAll("Currenct left for %N: %d", actor, TF2_GetCurrency(actor));
			
			/* The game refused what we asked for
			Nothing about the next interval would differ, so the same upgrade would be picked and
			refused until the window runs out, with the wave waiting on a bot that cannot spend */
			if (!purchased)
			{
				//Remembered rather than given up on: the next interval picks the next thing down
				int refused = Go_RowIndexOf(actor, row);
				
				if (refused >= 0 && refused < MAX_UPGRADES)
					Go_SetRefusedUpgrade(actor, refused);
				
				LogUpgradeSessionEnd(actor, "the game refused one, trying the next");
			}
		}
		else 
		{
			SetPlayerReady(actor, true);
			
			LogUpgradeSessionEnd(actor, "nothing left worth buying");
			
			return GetUpgradePostAction(actor, action);
		}
	}
	
	if (TF2_GetPlayerClass(actor) == TFClass_Medic)
	{
		int secondary = GetPlayerWeaponSlot(actor, TFWeaponSlot_Secondary);
		
		if (secondary != -1 && TF2Util_GetWeaponID(secondary) == TF_WEAPON_MEDIGUN)
		{
			int teammate = GerNearestTeammate(actor, WEAPON_MEDIGUN_RANGE);
			
			if (teammate != -1)
			{
				//Heal a nearby teammate so we build up uber
				TF2Util_SetPlayerActiveWeapon(actor, secondary);
				SnapViewToPosition(actor, WorldSpaceCenter(teammate));
				VS_PressFireButton(actor);
			}
		}
	}
	
	return action.Continue();
}
