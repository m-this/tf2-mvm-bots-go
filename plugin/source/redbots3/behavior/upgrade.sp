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

public void CTFBotUpgrade_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	/* What is left over stays in the wallet
	
	This spent it on canteens, every session, on the reasoning that money not spent is money wasted.
	It is the other way round in this mode. Credits carry between waves and upgrades do not expire,
	so an unspent hundred is a hundred towards the four hundred upgrade that actually changes a
	wave. A canteen is used once and gone, and a bot that empties its wallet into them every wave
	never saves for anything.
	
	If a canteen is worth buying it is worth ranking against everything else that money could buy,
	which is what the upgrade path is for. Buying it with whatever happened to be left is not a
	decision, it is a leak. */
	KV_MvM_UpgradesDone(actor);
	
	/* What comes down after a shopping trip, and what is left standing

	Everything used to. That was right about a level 1, which is worth less than the level 3 the
	engineer would rebuild in the time he has, and wrong about a level 3, which is worth more than
	anything he can do with the break: taking one down means a walk, three hundred metal, and
	another go at every way placing a building can fail. Reported from play as the engineer
	destroying perfectly good buildings between waves on a path that had not changed.

	So the test is whether rebuilding could produce anything better. A finished building on ground
	the nest still occupies cannot be improved on, and stays. Anything short of finished comes
	down, and so does everything when the nest has moved, because then it is in the wrong place
	however good it is. */
	if (TF2_GetPlayerClass(actor) == TFClass_Engineer && GameRules_GetRoundState() == RoundState_BetweenRounds)
	{
		bool nestMoved = m_aNestAreaRelocate[actor] != NULL_AREA;
		
		if (nestMoved)
		{
			m_aNestArea[actor] = m_aNestAreaRelocate[actor];
			m_aNestAreaRelocate[actor] = NULL_AREA;
		}
		
		if (nestMoved || !NothingLeftToBuild(GetObjectOfType(actor, TFObject_Sentry)))
			DetonateObjectOfType(actor, TFObject_Sentry);
		
		if (nestMoved || !NothingLeftToBuild(GetObjectOfType(actor, TFObject_Dispenser)))
			DetonateObjectOfType(actor, TFObject_Dispenser);
	}
	
	if (IsPlayerAlive(actor))
	{
		//Remember this bot's upgrades
		Command_BoughtUpgrades(actor, 0);
		
		//First upgrade session upon joining, give everything as if we prepared beforehand
		//Mainly for use with MANAGER_MODE_AUTO_BOTS
		if (GameRules_GetRoundState() == RoundState_RoundRunning && g_bHasUpgraded[actor] == false)
			UpgradeMidRoundPostActivity(actor);
		
		g_bHasUpgraded[actor] = true;
		g_bShoppedThisBreak[actor] = true;
		g_iBuyUpgradesNumber[actor] = 0;
		
		TF2_SetInUpgradeZone(actor, false);
		RecoverDefenderFromDisconnectedSpawn(actor);
	}
}
void KV_MvM_UpgradesBegin(int client)
{
	m_nPurchasedUpgrades[client] = 0;

	KeyValues kv = new KeyValues("MvM_UpgradesBegin");
	FakeClientCommandKeyValues(client, kv);
	delete kv;
}
void KV_MVM_Upgrade(int client, int count, int slot, int index)
{
	KeyValues kv = new KeyValues("MVM_Upgrade");
	kv.JumpToKey("upgrade", true);
	kv.SetNum("itemslot", slot);
	kv.SetNum("upgrade", index);
	kv.SetNum("count", count);
	FakeClientCommandKeyValues(client, kv);
	delete kv;
}

void KV_MvM_UpgradesDone(int client)
{
	KeyValues kv = new KeyValues("MvM_UpgradesDone");
	kv.SetNum("num_upgrades", m_nPurchasedUpgrades[client]);
	FakeClientCommandKeyValues(client, kv);
	delete kv;
}

void UpgradeMidRoundPostActivity(int client)
{
	switch (TF2_GetPlayerClass(client))
	{
		case TFClass_Medic:
		{
			int secondary = GetPlayerWeaponSlot(client, TFWeaponSlot_Secondary);
			
			if (secondary != -1)
				SetEntPropFloat(secondary, Prop_Send, "m_flChargeLevel", 1.0);
			
			SetEntPropFloat(client, Prop_Send, "m_flRageMeter", 100.0);
		}
	}
}
