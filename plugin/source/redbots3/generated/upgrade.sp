BehaviorAction CTFBotUpgrade()
{
	BehaviorAction action = ActionsManager.Create("DefenderUpgrade");

	action.OnStart = CTFBotUpgrade_OnStart;
	action.Update = CTFBotUpgrade_Update;
	action.OnEnd = CTFBotUpgrade_OnEnd;

	return action;
}

#define Go_Slots (65)

#define UPGRADE_ATTRIBUTE_SHARE (0.5)

#define MAX_INT (99999999)

#define MIN_INT (-99999999)

#define Go_rowClass (0)
#define Go_rowSlot (1)
#define Go_rowIndex (2)
#define Go_rowRandom (3)
#define Go_rowPriority (4)
#define Go_rowCells (5)

#define BUY_UPGRADES_FAST_MAX_TIME (3.0)

int m_iSessionWallet[65];
int m_iSpentOnUpgrade[65][128];
ArrayList CTFPlayerUpgrades[65];
float m_flNextUpgrade[65];
int m_nPurchasedUpgrades[65];
float m_flUpgradingTime[65];
bool m_bRefusedUpgrade[65][128];

stock bool NothingLeftToBuild(int building)
{
	if ((building == INVALID_ENT_REFERENCE) || !IsValidEntity(building))
	{
		return false;
	}
	if (TF2_IsBuilding(building))
	{
		return false;
	}
	return TF2_IsMiniBuilding(building) || (TF2_GetUpgradeLevel(building) >= 3);
}

stock bool WithinAttributeShare(int client, int index, int cost)
{
	if ((index < 0) || (index >= MAX_UPGRADES))
	{
		return true;
	}
	int allowed = RoundToNearest(float(m_iSessionWallet[client]) * UPGRADE_ATTRIBUTE_SHARE);
	if (allowed < (cost * 2))
	{
		allowed = cost * 2;
	}
	return (m_iSpentOnUpgrade[client][index] + cost) <= allowed;
}

stock void LogUpgradeSessionEnd(int actor, const char[] why)
{
	LogMessage("Shopping: %N stopped, %s, %d credits left, wave deals blast=%d bullet=%d fire=%d", actor, why, TF2_GetCurrency(actor), WaveHasExplosiveRobots(), WaveHasBulletRobots(), WaveHasFireRobots());
}

stock float GetUpgradeInterval()
{
	float customInterval = redbots_manager_bot_upgrade_interval.FloatValue;
	if (customInterval >= 0.0)
	{
		return customInterval;
	}
	if (GameRules_GetRoundState() == RoundState_RoundRunning)
	{
		return GetRandomFloat(0.1, 0.75);
	}
	float interval = 1.25;
	float variance = 0.3;
	return GetRandomFloat(0.95, 1.55);
}

stock int GetUpgradePriority(int client, int slot, int index, TFClassType pclass)
{
	if (slot == TF_LOADOUT_SLOT_ACTION)
	{
		return -10;
	}
	Address upgrade = UpgradeAddressByIndex(index);
	if (upgrade == Address_Null)
	{
		return UnrankedUpgradePriority();
	}
	char attribute[512];
	attribute = UpgradeAttributeOf(upgrade);
	if (attribute[0] == 0)
	{
		return UnrankedUpgradePriority();
	}
	if (IsUpgradeWasted(client, attribute))
	{
		return -10;
	}
	int id = AttributeID(attribute);
	int priority = 0;
	if ((TF2_GetPlayerClass(client) == TFClass_Engineer) && EngineerGunSpendsMetal(client))
	{
		priority = UpgradeRankEngineerMetal(id);
	}
	if ((priority <= 0) && (slot >= TF_LOADOUT_SLOT_PRIMARY) && (slot <= TF_LOADOUT_SLOT_MELEE))
	{
		int weapon = GetPlayerWeaponSlot(client, slot);
		if ((weapon > 0) && HasEntProp(weapon, Prop_Send, "m_iItemDefinitionIndex"))
		{
			priority = UpgradeRankLoadout(GetEntProp(weapon, Prop_Send, "m_iItemDefinitionIndex"), id);
		}
	}
	if (priority > 0)
	{
		return priority;
	}
	priority = UpgradeRankClass(pclass, slot, id);
	if (priority > 0)
	{
		return priority;
	}
	return UpgradeRankGeneral(id);
}

stock int SortUpgradesHighestFirst(int index1, int index2, Handle array, Handle hndl)
{
	ArrayList list = view_as<ArrayList>(array);
	int first = list.Get(index1, Go_rowPriority);
	int second = list.Get(index2, Go_rowPriority);
	if (first > second)
	{
		return -1;
	}
	return (first < second ? 1 : 0);
}

stock void CollectUpgrades(int client)
{
	if (CTFPlayerUpgrades[client] != null)
	{
		CTFPlayerUpgrades[client].Close();
	}
	CTFPlayerUpgrades[client] = new ArrayList(Go_rowCells);
	ArrayList iArraySlots = new ArrayList();
	iArraySlots.Push(-1);
	bool bDemoKnight = GetPlayerWeaponSlot(client, TFWeaponSlot_Primary) == -1;
	bool bEngineer = TF2_GetPlayerClass(client) == TFClass_Engineer;
	if (bEngineer)
	{
		iArraySlots.Push(TF_LOADOUT_SLOT_MELEE);
		iArraySlots.Push(TF_LOADOUT_SLOT_BUILDING);
		iArraySlots.Push(TF_LOADOUT_SLOT_PDA);
		iArraySlots.Push(TF_LOADOUT_SLOT_PRIMARY);
		iArraySlots.Push(TF_LOADOUT_SLOT_SECONDARY);
	}
	else
	{
		if (TF2_GetPlayerClass(client) == TFClass_Sniper)
		{
			iArraySlots.Push(TF_LOADOUT_SLOT_PRIMARY);
			iArraySlots.Push(TF_LOADOUT_SLOT_MELEE);
		}
		else
			if (TF2_GetPlayerClass(client) == TFClass_Medic)
			{
				iArraySlots.Push(TF_LOADOUT_SLOT_SECONDARY);
			}
			else
				if (TF2_GetPlayerClass(client) == TFClass_Spy)
				{
					iArraySlots.Push(TF_LOADOUT_SLOT_BUILDING);
					iArraySlots.Push(TF_LOADOUT_SLOT_MELEE);
				}
		iArraySlots.Push((bDemoKnight ? TF_LOADOUT_SLOT_MELEE : TF_LOADOUT_SLOT_PRIMARY));
		if (TF2_IsShieldEquipped(client))
		{
			iArraySlots.Push(TF_LOADOUT_SLOT_SECONDARY);
		}
		else
		{
			int secondary = GetPlayerWeaponSlot(client, TFWeaponSlot_Secondary);
			int weaponID = (secondary != -1 ? TF2Util_GetWeaponID(secondary) : -1);
			switch (weaponID)
			{
				case TF_WEAPON_JAR, TF_WEAPON_JAR_MILK, TF_WEAPON_BUFF_ITEM, TF_WEAPON_JAR_GAS:
				{
					iArraySlots.Push(TF_LOADOUT_SLOT_SECONDARY);
				}
				case TF_WEAPON_PIPEBOMBLAUNCHER:
				{
					if (bDemoKnight)
					{
						iArraySlots.Push(TF_LOADOUT_SLOT_SECONDARY);
					}
				}
			}
		}
	}
	for (int i = 0; i < iArraySlots.Length; i++)
	{
		int slot = iArraySlots.Get(i);
		int upgradeCount = UpgradeCount();
		for (int index = 0; index < upgradeCount; index++)
		{
			Address upgrade = UpgradeAddressByIndex(index);
			if ((UpgradeUIGroupOf(upgrade) == UIGROUP_UPGRADE_ATTACHED_TO_PLAYER) && (slot != -1))
			{
				continue;
			}
			if (UpgradeUIGroupOf(upgrade) == UIGROUP_POWERUPBOTTLE)
			{
				continue;
			}
			Address attr = CEIAD_GetAttributeDefinitionByName(UpgradeAttributeOf(upgrade));
			if (attr == Address_Null)
			{
				continue;
			}
			if (!CanUpgradeWithAttrib(client, slot, AttributeDefinitionIndexOf(attr), upgrade))
			{
				continue;
			}
			TFClassType pclass = TF2_GetPlayerClass(client);
			int row = CTFPlayerUpgrades[client].Push(view_as<int>(pclass));
			CTFPlayerUpgrades[client].Set(row, slot, Go_rowSlot);
			CTFPlayerUpgrades[client].Set(row, index, Go_rowIndex);
			CTFPlayerUpgrades[client].Set(row, GetRandomInt(MIN_INT, MAX_INT), Go_rowRandom);
			CTFPlayerUpgrades[client].Set(row, GetUpgradePriority(client, slot, index, pclass), Go_rowPriority);
		}
	}
	CTFPlayerUpgrades[client].SortCustom(SortUpgradesHighestFirst);
	if (redbots_manager_debug_actions.BoolValue)
	{
		PrintToServer("\nPreferred upgrades for #%d \"%N\"\n", client, client);
		PrintToServer("%3s %4s %4s %5s %-64s\n", "#", "SLOT", "COST", "INDEX", "ATTRIBUTE");
		for (int i = 0; i < CTFPlayerUpgrades[client].Length; i++)
		{
			int index = CTFPlayerUpgrades[client].Get(i, Go_rowIndex);
			int slot = CTFPlayerUpgrades[client].Get(i, Go_rowSlot);
			int pclass = CTFPlayerUpgrades[client].Get(i, Go_rowClass);
			int cost = GetCostForUpgrade(UpgradeAddressByIndex(index), slot, pclass, client);
			PrintToServer("%3d %4d %4d %5d %-64s", i, slot, cost, index, UpgradeAttributeOf(UpgradeAddressByIndex(index)));
		}
	}
	delete iArraySlots;
}

stock int CTFBotPurchaseUpgrades_ChooseUpgrade(int actor)
{
	int currency = TF2_GetCurrency(actor);
	if ((CTFPlayerUpgrades[actor] == null) || (CTFPlayerUpgrades[actor].Length == 0))
	{
		CollectUpgrades(actor);
	}
	for (int i = 0; i < CTFPlayerUpgrades[actor].Length; i++)
	{
		int index = CTFPlayerUpgrades[actor].Get(i, Go_rowIndex);
		int slot = CTFPlayerUpgrades[actor].Get(i, Go_rowSlot);
		int pclass = CTFPlayerUpgrades[actor].Get(i, Go_rowClass);
		Address upgrade = UpgradeAddressByIndex(index);
		if (upgrade == Address_Null)
		{
			if (redbots_manager_debug_actions.BoolValue)
			{
				PrintToServer("CMannVsMachineUpgrades is NULL");
			}
			return -1;
		}
		Address attr = CEIAD_GetAttributeDefinitionByName(UpgradeAttributeOf(upgrade));
		if (attr == Address_Null)
		{
			continue;
		}
		if (m_bRefusedUpgrade[actor][index])
		{
			continue;
		}
		if (!CanUpgradeWithAttrib(actor, slot, AttributeDefinitionIndexOf(attr), upgrade))
		{
			continue;
		}
		int iCost = GetCostForUpgrade(upgrade, slot, pclass, actor);
		if (!WithinAttributeShare(actor, index, iCost))
		{
			continue;
		}
		if (iCost > currency)
		{
			continue;
		}
		if (GetUpgradePriority(actor, slot, index, view_as<TFClassType>(pclass)) < 0)
		{
			continue;
		}
		int tier = GetUpgradeTier(index);
		if (tier != 0)
		{
			if (!IsUpgradeTierEnabled(actor, slot, tier))
			{
				continue;
			}
		}
		return i;
	}
	return -1;
}

stock bool CTFBotPurchaseUpgrades_PurchaseUpgrade(int actor, int row)
{
	int slot = CTFPlayerUpgrades[actor].Get(row, Go_rowSlot);
	int index = CTFPlayerUpgrades[actor].Get(row, Go_rowIndex);
	int pclass = CTFPlayerUpgrades[actor].Get(row, Go_rowClass);
	int cost = GetCostForUpgrade(UpgradeAddressByIndex(index), slot, pclass, actor);
	int currencyBefore = TF2_GetCurrency(actor);
	int count = 1;
	if (cost > 0)
	{
		Address upgrade = UpgradeAddressByIndex(index);
		int tiers = UPGRADE_TIERS_MAX;
		if (upgrade != Address_Null)
		{
			tiers = UpgradeTierCap(UpgradeAttributeOf(upgrade));
		}
		count = currencyBefore / cost;
		if (count > tiers)
		{
			count = tiers;
		}
		if (count < 1)
		{
			count = 1;
		}
	}
	KV_MVM_Upgrade(actor, count, slot, index);
	int spent = currencyBefore - TF2_GetCurrency(actor);
	if ((cost > 0) && (spent <= 0))
	{
		return false;
	}
	m_nPurchasedUpgrades[actor] += (cost > 0 ? spent / cost : 1);
	if ((index >= 0) && (index < MAX_UPGRADES))
	{
		m_iSpentOnUpgrade[actor][index] += spent;
	}
	return true;
}

stock int Go_RowIndexOf(int actor, int row)
{
	return CTFPlayerUpgrades[actor].Get(row, Go_rowIndex);
}

stock void Go_SetRefusedUpgrade(int actor, int index)
{
	m_bRefusedUpgrade[actor][index] = true;
}

stock void KV_MvM_UpgradesBegin(int client)
{
	m_nPurchasedUpgrades[client] = 0;
	KeyValues kv = new KeyValues("MvM_UpgradesBegin");
	FakeClientCommandKeyValues(client, kv);
	delete kv;
}

stock void KV_MVM_Upgrade(int client, int count, int slot, int index)
{
	KeyValues kv = new KeyValues("MVM_Upgrade");
	kv.JumpToKey("upgrade", true);
	kv.SetNum("itemslot", slot);
	kv.SetNum("upgrade", index);
	kv.SetNum("count", count);
	FakeClientCommandKeyValues(client, kv);
	delete kv;
}

stock void KV_MvM_UpgradesDone(int client)
{
	KeyValues kv = new KeyValues("MvM_UpgradesDone");
	kv.SetNum("num_upgrades", m_nPurchasedUpgrades[client]);
	FakeClientCommandKeyValues(client, kv);
	delete kv;
}

stock void UpgradeMidRoundPostActivity(int client)
{
	switch (TF2_GetPlayerClass(client))
	{
		case TFClass_Medic:
		{
			int secondary = GetPlayerWeaponSlot(client, TFWeaponSlot_Secondary);
			if (secondary != -1)
			{
				SetEntPropFloat(secondary, Prop_Send, "m_flChargeLevel", 1.0);
			}
			SetEntPropFloat(client, Prop_Send, "m_flRageMeter", 100.0);
		}
	}
}

public void CTFBotUpgrade_OnEnd(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	KV_MvM_UpgradesDone(actor);
	if ((TF2_GetPlayerClass(actor) == TFClass_Engineer) && (GameRules_GetRoundState() == RoundState_BetweenRounds))
	{
		bool nestMoved = m_aNestAreaRelocate[actor] != NULL_AREA;
		if (nestMoved)
		{
			m_aNestArea[actor] = m_aNestAreaRelocate[actor];
			m_aNestAreaRelocate[actor] = NULL_AREA;
		}
		if (nestMoved || !NothingLeftToBuild(GetObjectOfType(actor, TFObject_Sentry)))
		{
			DetonateObjectOfType(actor, TFObject_Sentry);
		}
		if (nestMoved || !NothingLeftToBuild(GetObjectOfType(actor, TFObject_Dispenser)))
		{
			DetonateObjectOfType(actor, TFObject_Dispenser);
		}
	}
	if (IsPlayerAlive(actor))
	{
		Command_BoughtUpgrades(actor, 0);
		if ((GameRules_GetRoundState() == RoundState_RoundRunning) && !g_bHasUpgraded[actor])
		{
			UpgradeMidRoundPostActivity(actor);
		}
		g_bHasUpgraded[actor] = true;
		g_bShoppedThisBreak[actor] = true;
		g_iBuyUpgradesNumber[actor] = 0;
		TF2_SetInUpgradeZone(actor, false);
		RecoverDefenderFromDisconnectedSpawn(actor);
	}
}

public Action CTFBotUpgrade_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	m_iSessionWallet[actor] = TF2_GetCurrency(actor);
	for (int i = 0; i < MAX_UPGRADES; i++)
	{
		m_bRefusedUpgrade[actor][i] = false;
	}
	for (int index = 0; index < MAX_UPGRADES; index++)
	{
		m_iSpentOnUpgrade[actor][index] = 0;
	}
	if (!TF2_IsInUpgradeZone(actor))
	{
		return action.ChangeTo(CTFBotGotoUpgrade(), "Not standing at an upgrade station!");
	}
	CollectUpgrades(actor);
	KV_MvM_UpgradesBegin(actor);
	m_flNextUpgrade[actor] = GetGameTime() + GetUpgradeInterval();
	bool isRoundActive = GameRules_GetRoundState() == RoundState_RoundRunning;
	if (!g_bHasUpgraded[actor] && isRoundActive)
	{
		m_flUpgradingTime[actor] = GetGameTime() + 15.0;
	}
	else
	{
		m_flUpgradingTime[actor] = GetGameTime() + (isRoundActive ? BUY_UPGRADES_FAST_MAX_TIME : BUY_UPGRADES_MAX_TIME);
	}
	return action.Continue();
}

public Action CTFBotUpgrade_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (!TF2_IsInUpgradeZone(actor))
	{
		return action.ChangeTo(CTFBotGotoUpgrade(), "Not standing at an upgrade station!");
	}
	if (m_flUpgradingTime[actor] <= GetGameTime())
	{
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
			{
				PrintToChatAll("Currenct left for %N: %d", actor, TF2_GetCurrency(actor));
			}
			if (!purchased)
			{
				int refused = Go_RowIndexOf(actor, row);
				if ((refused >= 0) && (refused < MAX_UPGRADES))
				{
					Go_SetRefusedUpgrade(actor, refused);
				}
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
		if ((secondary != -1) && (TF2Util_GetWeaponID(secondary) == TF_WEAPON_MEDIGUN))
		{
			int teammate = GerNearestTeammate(actor, WEAPON_MEDIGUN_RANGE);
			if (teammate != -1)
			{
				TF2Util_SetPlayerActiveWeapon(actor, secondary);
				SnapViewToPosition(actor, WorldSpaceCenter(teammate));
				VS_PressFireButton(actor);
			}
		}
	}
	return action.Continue();
}

