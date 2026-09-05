BehaviorAction CTFBotUpgrade()
{
	BehaviorAction action = ActionsManager.Create("DefenderUpgrade");

	action.OnStart = CTFBotUpgrade_OnStart;
	action.Update = CTFBotUpgrade_Update;
	action.OnEnd = CTFBotUpgrade_OnEnd;

	return action;
}

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

// NothingLeftToBuild says the engineer has nothing further to do to this
// building.
//
// A building still going up is not finished, and neither is one below level three.
// A mini is: it cannot be upgraded at all, which is the whole point of it.
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

// WithinAttributeShare says this purchase leaves the trip balanced.
//
// Without it a bot spends its whole wallet on the first attribute the ranking
// likes, which on a good roll is one very sharp weapon and nothing else. The floor
// of twice the cost is there so a cheap attribute is never refused outright: half
// of a small wallet can be less than one step.
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

// LogUpgradeSessionEnd says why the trip stopped, and what the wave looked like
// while it was spending.
//
// A resistance is ranked at 210 when the coming wave deals that damage and 25 when
// it does not, so the three answers decide whether a resistance was ever worth
// buying on this trip. Reported from play as "the bots really need to buy
// resistance upgrades", and the ranking was already there and already ahead of
// most of the table: what nobody could see was whether WaveHasClassIcon had
// anything to read. tf_objective_resource carries the wave bar, and a trip that
// shops before the game has filled it in sees an empty wave and prices every
// resistance at nothing.
stock void LogUpgradeSessionEnd(int actor, const char[] why)
{
	LogMessage("Shopping: %N stopped, %s, %d credits left, wave deals blast=%d bullet=%d fire=%d", actor, why, TF2_GetCurrency(actor), WaveHasExplosiveRobots(), WaveHasBulletRobots(), WaveHasFireRobots());
}

// GetUpgradeInterval is how long a bot waits between purchases.
//
// Fast during a round, because a bot shopping mid-wave is a bot not fighting, and
// unhurried between them. The spread is so six bots at one station do not press
// the button on the same frame.
stock float GetUpgradeInterval()
{
	float customInterval = redbots_manager_bot_upgrade_interval.FloatValue;
	if (customInterval >= 0.0)
	{
		return customInterval;
	}
	// Upgrading during an active round, buy upgrades fast.
	if (GameRules_GetRoundState() == RoundState_RoundRunning)
	{
		return GetRandomFloat(0.1, 0.75);
	}
	float interval = 1.25;
	float variance = 0.3;
	return GetRandomFloat(0.95, 1.55);
}

// GetUpgradePriority is what one candidate is worth to this bot.
//
// The name becomes a number once, here, and the three tables switch on it.
// generated/upgrade_rank.sp is written from internal/upgrade/table.go, which holds
// the scores this used to compare ninety-four strings to reach. ATTRIBUTE_NONE is
// what a name the table does not rank becomes, and every table below falls through
// it to the general one, which is what the comparison chain did with a name it did
// not recognise.
stock int GetUpgradePriority(int client, int slot, int index, TFClassType pclass)
{
	// A canteen is worth less to a bot than anything it can shoot with.
	if (slot == TF_LOADOUT_SLOT_ACTION)
	{
		return -10;
	}
	Address upgrade = UpgradeAddressByIndex(index);
	// Nothing to rank it on, so rank it the way the mod used to rank
	// everything.
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
	// An upgrade to something the bot is not carrying is credits set on fire.
	if (IsUpgradeWasted(client, attribute))
	{
		return -10;
	}
	int id = AttributeID(attribute);
	int priority = 0;
	// The metal upgrades do not hang off the gun, so they are asked before
	// the slot is.
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

// SortUpgradesHighestFirst orders the candidates, dearest first.
//
// Equal priorities fall back to the random cell each candidate drew, so two
// upgrades worth the same are not always bought in table order.
stock int SortUpgradesHighestFirst(int index1, int index2, Handle array, Handle hndl)
{
	ArrayList list = view_as<ArrayList>(array);
	int first = list.Get(index1, Go_rowPriority);
	int second = list.Get(index2, Go_rowPriority);
	if (first == second)
	{
		first = list.Get(index1, Go_rowRandom);
		second = list.Get(index2, Go_rowRandom);
	}
	if (first > second)
	{
		return -1;
	}
	return (first < second ? 1 : 0);
}

// CollectUpgrades builds the list of everything this bot could buy, best first.
//
// The slots are chosen by class first, because most of what the station offers is
// worth nothing to most classes. The engineer gets the most: his melee, his PDAs,
// and both guns, because a Widowmaker spends the metal the sentry needs and a
// Rescue Ranger had a rule nothing could reach. The sentry still outranks all of
// it, so the gun is bought with what is left.
stock void CollectUpgrades(int client)
{
	if (CTFPlayerUpgrades[client] != null)
	{
		CTFPlayerUpgrades[client].Close();
	}
	CTFPlayerUpgrades[client] = new ArrayList(Go_rowCells);
	ArrayList iArraySlots = new ArrayList();
	// Always buy player upgrades.
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
				// Buy upgrades for our medigun.
				iArraySlots.Push(TF_LOADOUT_SLOT_SECONDARY);
			}
			else
				if (TF2_GetPlayerClass(client) == TFClass_Spy)
				{
					// Buy upgrades for our sapper and knife.
					iArraySlots.Push(TF_LOADOUT_SLOT_BUILDING);
					iArraySlots.Push(TF_LOADOUT_SLOT_MELEE);
				}
		// A demoknight does not buy primary weapon upgrades.
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
					// Secondary items that have some use.
					iArraySlots.Push(TF_LOADOUT_SLOT_SECONDARY);
				}
				case TF_WEAPON_PIPEBOMBLAUNCHER:
				{
					// With no actual primary, the secondary is what it relies on.
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
			//  Canteens are not bought at all
			//
			// 			The player slot takes every upgrade the game does not attach to a
			// 			weapon, which sweeps up the powerup bottle charges too. The game
			// 			refuses those on slot -1, the bot pays nothing, and the next
			// 			interval picks the same charge again for as long as the upgrade
			// 			window lasts. See CTFBotUpgrade_OnEnd for why the leftovers stay
			// 			in the wallet instead.
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

// ChooseUpgrade is the best thing this bot can still afford, as a row index, and
// -1 when there is nothing left worth buying.
//
// The list is built once a session, not once a purchase. It used to be rebuilt and
// re-sorted every time a bot bought anything, which is every 0.1 to 1.25 seconds
// each, and six bots at the station did it several times a second between them. It
// bought nothing: everything the rebuild filtered on, the walk below re-asks per
// entry anyway, and a rebuild can only ever remove entries because buying an
// upgrade does not make another available.
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
		// Already refused this trip, so asking again spends the window on the
		// same no.
		if (m_bRefusedUpgrade[actor][index])
		{
			continue;
		}
		if (!CanUpgradeWithAttrib(actor, slot, AttributeDefinitionIndexOf(attr), upgrade))
		{
			continue;
		}
		int iCost = GetCostForUpgrade(upgrade, slot, pclass, actor);
		// This one has had its share of the wallet already.
		if (!WithinAttributeShare(actor, index, iCost))
		{
			continue;
		}
		if (iCost > currency)
		{
			continue;
		}
		//  A negative priority is a refusal, not a low bid
		//
		// 		It used to be only a bid, so once everything worth having was maxed or
		// 		unaffordable the bot worked down the list and bought whatever was left.
		// 		Reported as Pyros buying Airblast Pushback Scale, which is in the
		// 		canteen slot and was ranked at minus ten for exactly that reason.
		// 		Ranking it last is not the same as never buying it.
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

// PurchaseUpgrade buys every tier of one upgrade the bot can afford, in one go.
//
// The game takes a count and applies it, refusing each step it cannot. Buying one
// step per interval instead is what a play-test heard as a bot announcing the same
// upgrade over and over, because each step is announced and four steps of one
// attribute all read the same.
//
// Nothing about what gets bought changes. The list is a strict priority and the top
// of it stays the top until it maxes out, so the steps bought here in one go are
// the ones the next four intervals would have bought anyway.
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
	// The credits never moved, which is the game turning the purchase down.
	if ((cost > 0) && (spent <= 0))
	{
		return false;
	}
	// An upgrade that costs nothing cannot be counted this way, so it counts
	// as one.
	m_nPurchasedUpgrades[actor] += (cost > 0 ? spent / cost : 1);
	if ((index >= 0) && (index < MAX_UPGRADES))
	{
		m_iSpentOnUpgrade[actor][index] += spent;
	}
	return true;
}

// RowIndexOf is the upgrade index a chosen row names, which the caller needs to
// remember a refusal by.
stock int Go_RowIndexOf(int actor, int row)
{
	return CTFPlayerUpgrades[actor].Get(row, Go_rowIndex);
}

// SetRefusedUpgrade remembers that the game turned one down.
stock void Go_SetRefusedUpgrade(int actor, int index)
{
	m_bRefusedUpgrade[actor][index] = true;
}

// KVUpgradesBegin opens the session.
stock void KV_MvM_UpgradesBegin(int client)
{
	m_nPurchasedUpgrades[client] = 0;
	KeyValues kv = new KeyValues("MvM_UpgradesBegin");
	FakeClientCommandKeyValues(client, kv);
	delete kv;
}

// KVUpgrade buys count steps of one upgrade in one slot.
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

// KVUpgradesDone closes the session, saying how many steps were taken.
stock void KV_MvM_UpgradesDone(int client)
{
	KeyValues kv = new KeyValues("MvM_UpgradesDone");
	kv.SetNum("num_upgrades", m_nPurchasedUpgrades[client]);
	FakeClientCommandKeyValues(client, kv);
	delete kv;
}

// MidRoundPostActivity gives a bot that joined mid-round what it would have
// had if it had shopped with everybody else.
//
// Only the medic, and only the two things a wave in progress has already given the
// others: a full charge and a full rage meter. Mainly for the auto mode, where bots
// arrive when the wave begins rather than before it.
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

// OnEnd closes the shopping trip.
//
// What is left over stays in the wallet. This spent it on canteens, every session,
// on the reasoning that money not spent is money wasted. It is the other way round
// in this mode: credits carry between waves and upgrades do not expire, so an
// unspent hundred is a hundred towards the four hundred upgrade that actually
// changes a wave. A canteen is used once and gone.
//
// What comes down after a trip is whatever rebuilding could improve on. Everything
// used to, which was right about a level 1 and wrong about a level 3: taking one
// down means a walk, three hundred metal, and another go at every way placing a
// building can fail. Reported from play as the engineer destroying perfectly good
// buildings between waves on a path that had not changed. A finished building on
// ground the nest still occupies stays; anything short of finished comes down, and
// so does everything when the nest has moved, because then it is in the wrong
// place however good it is.
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
		// Remember this bot's upgrades.
		Command_BoughtUpgrades(actor, 0);
		// The first session after joining gives everything as if the bot had
		// prepared beforehand, which is what the auto mode needs.
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

// OnStart opens the shopping trip.
//
// The wallet is measured once, here, and every attribute's share is taken against
// that rather than against whatever is left: a play-test bundle came back with a
// medic who bought ubercharge rate twenty five times across a mission, ten of them
// inside thirty seconds, and nothing else all run.
//
// The refusals are cleared too. A refusal used to end the whole trip, on the
// reasoning that the ranking would pick the same line again and be refused until
// the window ran out. It would, and the answer is to remember the refusal rather
// than to stop shopping: ten of forty five trips measured on Bavarian Botbash
// ended that way, with money still in the wallet and a list still worth walking.
public Action CTFBotUpgrade_OnStart(BehaviorAction action, int actor, BehaviorAction priorAction, ActionResult result)
{
	m_pPath[actor].SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor));
	// The wallet this trip is measured against, and nothing spent out of it
	// yet.
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
	// How long should it take us to buy upgrades?
	if (!g_bHasUpgraded[actor] && isRoundActive)
	{
		// We probably just joined during an active game.
		m_flUpgradingTime[actor] = GetGameTime() + 15.0;
	}
	else
	{
		// Spend less time upgrading during the round, normal otherwise.
		m_flUpgradingTime[actor] = GetGameTime() + (isRoundActive ? BUY_UPGRADES_FAST_MAX_TIME : BUY_UPGRADES_MAX_TIME);
	}
	return action.Continue();
}

// Update buys one thing per interval, and heals while it waits.
//
// The medic keeps his beam on somebody through the whole trip: a charge builds into
// whoever he is beaming, so the break is worth an uber if he spends it next to
// anybody at all.
public Action CTFBotUpgrade_Update(BehaviorAction action, int actor, float interval, ActionResult result)
{
	if (!TF2_IsInUpgradeZone(actor))
	{
		return action.ChangeTo(CTFBotGotoUpgrade(), "Not standing at an upgrade station!");
	}
	if (m_flUpgradingTime[actor] <= GetGameTime())
	{
		// It should not take this long to upgrade.
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
			//  The game refused what we asked for
			//
			// 			Nothing about the next interval would differ, so the same upgrade
			// 			would be picked and refused until the window runs out, with the
			// 			wave waiting on a bot that cannot spend. Remembered rather than
			// 			given up on: the next interval picks the next thing down.
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
				// Heal a nearby teammate so we build up uber.
				TF2Util_SetPlayerActiveWeapon(actor, secondary);
				SnapViewToPosition(actor, WorldSpaceCenter(teammate));
				VS_PressFireButton(actor);
			}
		}
	}
	return action.Continue();
}

