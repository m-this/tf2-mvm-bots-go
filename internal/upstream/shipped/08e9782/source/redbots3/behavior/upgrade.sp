#define BUY_UPGRADES_MAX_TIME	30.0
#define BUY_UPGRADES_FAST_MAX_TIME	3.0

/* A building rebuilding cannot improve on: finished being built, and at its top level

Its own rather than nextbot_behavior's IsBuildingFinished, which this file is included ahead of. */
static bool NothingLeftToBuild(int building)
{
	if (building == INVALID_ENT_REFERENCE || !IsValidEntity(building))
		return false;
	
	if (TF2_IsBuilding(building))
		return false;
	
	return TF2_IsMiniBuilding(building) || TF2_GetUpgradeLevel(building) >= 3;
}

static int MAX_INT = 99999999;
static int MIN_INT = -99999999;

JSONArray CTFPlayerUpgrades[MAXPLAYERS + 1];
float m_flNextUpgrade[MAXPLAYERS + 1];
int m_nPurchasedUpgrades[MAXPLAYERS + 1];
float m_flUpgradingTime[MAXPLAYERS + 1];

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
#define UPGRADE_ATTRIBUTE_SHARE	0.5

static int m_iSessionWallet[MAXPLAYERS + 1];
static int m_iSpentOnUpgrade[MAXPLAYERS + 1][MAX_UPGRADES];

/* What the game refused this trip, so the next choice is a different one
 *
 * A refusal used to end the whole shopping trip, on the reasoning that the ranking would pick the
 * same line again and be refused until the window ran out. It would, and the answer is to remember
 * the refusal rather than to stop shopping: ten of forty five trips measured on Bavarian Botbash
 * ended this way, with money still in the wallet and a list still worth walking. */
static bool m_bRefusedUpgrade[MAXPLAYERS + 1][MAX_UPGRADES];

//What one more step of this upgrade would take the bot past, or false when it would not
static bool WithinAttributeShare(int client, int index, int cost)
{
	if (index < 0 || index >= MAX_UPGRADES)
		return true;
	
	int allowed = RoundToNearest(float(m_iSessionWallet[client]) * UPGRADE_ATTRIBUTE_SHARE);
	
	if (allowed < cost * 2)
		allowed = cost * 2;
	
	return m_iSpentOnUpgrade[client][index] + cost <= allowed;
}

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

/* Why a shopping trip ended, and what was still in the wallet when it did
 *
 * The test-bed says the team finishes a wave holding a median of 3328 credits it never spent, and
 * a trip can end three ways: the window ran out, the game refused a purchase, or the ranking ran
 * out of things it was willing to buy. Those are three different faults and nothing said which. */
static void LogUpgradeSessionEnd(int actor, const char[] why)
{
	/* And what the wave looked like while he was spending
	
	A resistance is ranked at 210 when the coming wave deals that damage and 25 when it does not,
	so the three answers below decide whether a resistance was ever worth buying on this trip.
	Reported from play as "the bots really need to buy resistance upgrades", and the ranking was
	already there and already ahead of most of the table: what nobody could see was whether
	WaveHasClassIcon had anything to read. tf_objective_resource carries the wave bar, and a trip
	that shops before the game has filled it in sees an empty wave and prices every resistance at
	nothing. */
	LogMessage("Shopping: %N stopped, %s, %d credits left, wave deals blast=%d bullet=%d fire=%d",
		actor, why, TF2_GetCurrency(actor),
		WaveHasExplosiveRobots(), WaveHasBulletRobots(), WaveHasFireRobots());
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
		
		JSONObject info = CTFBotPurchaseUpgrades_ChooseUpgrade(actor);
		
		if (info != null) 
		{
			bool purchased = CTFBotPurchaseUpgrades_PurchaseUpgrade(actor, info);
			
			if (redbots_manager_debug_actions.BoolValue)
				PrintToChatAll("Currenct left for %N: %d", actor, TF2_GetCurrency(actor));
			
			/* The game refused what we asked for
			Nothing about the next interval would differ, so the same upgrade would be picked and
			refused until the window runs out, with the wave waiting on a bot that cannot spend */
			if (!purchased)
			{
				//Remembered rather than given up on: the next interval picks the next thing down
				int refused = info.GetInt("index");
				
				if (refused >= 0 && refused < MAX_UPGRADES)
					m_bRefusedUpgrade[actor][refused] = true;
				
				LogUpgradeSessionEnd(actor, "the game refused one, trying the next");
			}
		}
		else 
		{
			SetPlayerReady(actor, true);
			
			LogUpgradeSessionEnd(actor, "nothing left worth buying");
			
			delete info;
			
			return GetUpgradePostAction(actor, action);
		}
		
		delete info;
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

void CollectUpgrades(int client)
{
	if (CTFPlayerUpgrades[client] != null)
		delete CTFPlayerUpgrades[client];
		
	CTFPlayerUpgrades[client] = new JSONArray();
	
	ArrayList iArraySlots = new ArrayList();
	
	iArraySlots.Push(-1); //Always buy player upgrades
	
	bool bDemoKnight = GetPlayerWeaponSlot(client, TFWeaponSlot_Primary) == -1;
	bool bEngineer = TF2_GetPlayerClass(client) == TFClass_Engineer;
	
	if (bEngineer)
	{
		iArraySlots.Push(TF_LOADOUT_SLOT_MELEE);
		iArraySlots.Push(TF_LOADOUT_SLOT_BUILDING);
		iArraySlots.Push(TF_LOADOUT_SLOT_PDA);

		/* The gun the engineer defends the nest with, which used to buy nothing at all
		A Widowmaker engineer spends metal the sentry needs on every shot and got no damage back
		for it. A Rescue Ranger engineer had a rule written for it in LoadoutUpgradePriority that
		nothing could ever reach. Both are the weapon the loadout was chosen for.

		The sentry still outranks all of it: everything these two slots can buy is ranked below
		the sentry lines in ClassUpgradePriority, so the shotgun is bought with what is left */
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
		else if (TF2_GetPlayerClass(client) == TFClass_Medic)
		{
			//Buy upgrades for our medigun
			iArraySlots.Push(TF_LOADOUT_SLOT_SECONDARY);
		}
		else if (TF2_GetPlayerClass(client) == TFClass_Spy)
		{
			//Buy upgrades for our sapper and knife
			iArraySlots.Push(TF_LOADOUT_SLOT_BUILDING);
			iArraySlots.Push(TF_LOADOUT_SLOT_MELEE);
		}

		//Demoknight doesn't buy primary weapon upgrades.
		iArraySlots.Push(bDemoKnight ? TF_LOADOUT_SLOT_MELEE : TF_LOADOUT_SLOT_PRIMARY);
		
		if (TF2_IsShieldEquipped(client))
		{
			iArraySlots.Push(TF_LOADOUT_SLOT_SECONDARY);
		}
		else
		{
			int secondary = GetPlayerWeaponSlot(client, TFWeaponSlot_Secondary);
			int weaponID = secondary != -1 ? TF2Util_GetWeaponID(secondary) : -1;
			
			switch (weaponID)
			{
				case TF_WEAPON_JAR, TF_WEAPON_JAR_MILK, TF_WEAPON_BUFF_ITEM, TF_WEAPON_JAR_GAS:
				{
					//Secondary items that have some use
					iArraySlots.Push(TF_LOADOUT_SLOT_SECONDARY);
				}
				case TF_WEAPON_PIPEBOMBLAUNCHER:
				{
					//If we don't have an actual primary, then we rely on our secondary
					if (bDemoKnight)
						iArraySlots.Push(TF_LOADOUT_SLOT_SECONDARY);
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
			CMannVsMachineUpgrades upgrades = CMannVsMachineUpgradeManager().GetUpgradeByIndex(index);
			
			if (upgrades.m_iUIGroup() == UIGROUP_UPGRADE_ATTACHED_TO_PLAYER && slot != -1) 
				continue;
			
			/* Canteens are not bought at all
			The player slot takes every upgrade the game does not attach to a weapon, which sweeps
			up the powerup bottle charges too. The game refuses those on slot -1, the bot pays
			nothing, and the next interval picks the same charge again for as long as the upgrade
			window lasts. Nothing buys them now: see CTFBotUpgrade_OnEnd for why the leftovers stay
			in the wallet instead. */
			if (upgrades.m_iUIGroup() == UIGROUP_POWERUPBOTTLE)
				continue;
			
			CEconItemAttributeDefinition attr = CEIAD_GetAttributeDefinitionByName(upgrades.m_szAttribute());
			if (attr.Address == Address_Null)
				continue;
			
			if (!CanUpgradeWithAttrib(client, slot, attr.GetIndex(), upgrades.Address))
				continue;
			
			JSONObject UpgradeInfo = new JSONObject();
			UpgradeInfo.SetInt("pclass", view_as<int>(TF2_GetPlayerClass(client)));
			UpgradeInfo.SetInt("slot", slot);
			UpgradeInfo.SetInt("index", index);
			UpgradeInfo.SetInt("random", GetRandomInt(MIN_INT, MAX_INT));
			UpgradeInfo.SetInt("priority", GetUpgradePriority(client, UpgradeInfo));
			
			CTFPlayerUpgrades[client].Push(UpgradeInfo);
			
			delete UpgradeInfo;
		}
	}
	
	delete iArraySlots;
	
	SortUpgradesByPriority(client);
	
	if (redbots_manager_debug_actions.BoolValue)
	{
		PrintToServer("\nPreferred upgrades for #%d \"%N\"\n", client, client);
		PrintToServer("%3s %4s %4s %5s %-64s\n", "#", "SLOT", "COST", "INDEX", "ATTRIBUTE");
		
		for (int i = 0; i < CTFPlayerUpgrades[client].Length; i++)
		{
			JSONObject info = view_as<JSONObject>(CTFPlayerUpgrades[client].Get(i));
			
			CMannVsMachineUpgradeManager manager = CMannVsMachineUpgradeManager();
			int cost = GetCostForUpgrade(manager.GetUpgradeByIndex(info.GetInt("index")).Address, info.GetInt("slot"), info.GetInt("pclass"), client);
			
			PrintToServer("%3d %4d %4d %5d %-64s", i, info.GetInt("slot"), cost, info.GetInt("index"), manager.GetUpgradeByIndex(info.GetInt("index")).m_szAttribute());
			
			delete info;
		}
	}
}

/* Highest priority first, in one pass over the list rather than one pass per entry

This was a selection sort that walked the whole remaining list to find the next highest, and every
comparison in it read an element out of a JSONArray, which is a handle allocated and freed. For a
shopping list of a hundred and fifty that is over twenty thousand handles, per bot.

It was not once a wave either. CollectUpgrades ends in this sort and was being called again for
every single purchase, so six bots at the station between rounds were doing it several times a
second between them. That is the shape of the frame the watchdog kills.

Priorities come out into a plain list, that gets sorted, and the entries are moved once into the
order it gives. Two handles an entry instead of two per comparison. */
static void SortUpgradesByPriority(int client)
{
	JSONArray unsorted = CTFPlayerUpgrades[client];
	int count = unsorted.Length;
	
	ArrayList order = new ArrayList(2);
	
	for (int i = 0; i < count; i++)
	{
		JSONObject info = view_as<JSONObject>(unsorted.Get(i));
		
		int at = order.Push(info.GetInt("priority"));
		order.Set(at, i, 1);
		
		delete info;
	}
	
	order.SortCustom(SortUpgradesHighestFirst);
	
	JSONArray sorted = new JSONArray();
	
	for (int i = 0; i < count; i++)
	{
		JSONObject info = view_as<JSONObject>(unsorted.Get(order.Get(i, 1)));
		
		sorted.Push(info);
		
		delete info;
	}
	
	delete order;
	delete unsorted;
	
	CTFPlayerUpgrades[client] = sorted;
}

static int SortUpgradesHighestFirst(int index1, int index2, Handle array, Handle hndl)
{
	ArrayList list = view_as<ArrayList>(array);
	
	int first = list.Get(index1, 0);
	int second = list.Get(index2, 0);
	
	if (first > second)
		return -1;
	
	return first < second ? 1 : 0;
}

/* What a bot buys at the upgrade station, highest number first

Every upgrade it can afford is sorted by this and the top one is bought, so this function is the
whole of a bot's shopping. It used to return GetRandomInt(50, 100) for everything except a Spy's
knife, which is why a Heavy would buy jump height while its minigun stayed stock.

The bands:

  300+  the upgrade that is the reason to carry this exact weapon
  200+  the class's own damage, which for an Engineer is the sentry and for a Medic the medigun
  100+  keeping that damage going: clip, ammo, reload
   50+  worth having once the damage is bought
    0+  staying alive, which a bot that respawns every wave needs least
  -10   canteens, which a bot never learns to use well

Nothing here caps anything. CanUpgradeWithAttrib already refuses an upgrade at its ceiling, so a
maxed damage bonus falls through to the next line on its own.

The attribute strings are the ones in scripts/items/mvm_upgrades.txt. An upgrade this table has
not met is ranked at random between 50 and 100, which is what the mod did with every upgrade
before this existed.

That fallback is the point. A Team Fortress 2 update that renames an attribute, or a mission that
adds one, would otherwise give every upgrade the same number: the sort would then be stable rather
than sensible, and every bot would buy the same wrong thing in the same order forever. Ranking the
unknown at random degrades to the old behaviour one upgrade at a time instead, and leaves the ones
this table does know still ahead of the resistances */

int GetUpgradePriority(int client, JSONObject info)
{
	int slot = info.GetInt("slot");
	
	//A canteen is worth less to a bot than anything it can shoot with
	if (slot == TF_LOADOUT_SLOT_ACTION)
		return -10;
	
	CMannVsMachineUpgrades upgrade = CMannVsMachineUpgradeManager().GetUpgradeByIndex(info.GetInt("index"));
	
	//Nothing to rank it on, so rank it the way the mod used to rank everything
	if (upgrade.Address == Address_Null)
		return UnrankedUpgradePriority();
	
	char attribute[MAX_ATTRIBUTE_DESCRIPTION_LENGTH]; attribute = upgrade.m_szAttribute();
	
	if (attribute[0] == '\0')
		return UnrankedUpgradePriority();
	
	//An upgrade to something the bot is not carrying is credits set on fire
	if (IsUpgradeWasted(client, attribute))
		return -10;
	
	/* The name becomes a number once, here, and the three tables switch on it
	
	generated/upgrade_rank.sp is written from internal/upgrade/table.go, which holds the scores
	this used to compare ninety-four strings to reach. ATTRIBUTE_NONE is what a name the table
	does not rank becomes, and every switch below falls through it to the general table, which is
	what the comparison chain did with a name it did not recognise. */
	int id = AttributeID(attribute);
	
	int priority = 0;
	
	//The metal upgrades do not hang off the gun, so they are asked before the slot is
	if (TF2_GetPlayerClass(client) == TFClass_Engineer && EngineerGunSpendsMetal(client))
		priority = UpgradeRankEngineerMetal(id);
	
	if (priority <= 0 && slot >= TF_LOADOUT_SLOT_PRIMARY && slot <= TF_LOADOUT_SLOT_MELEE)
	{
		int weapon = GetPlayerWeaponSlot(client, slot);
		
		if (weapon > 0 && HasEntProp(weapon, Prop_Send, "m_iItemDefinitionIndex"))
			priority = UpgradeRankLoadout(GetEntProp(weapon, Prop_Send, "m_iItemDefinitionIndex"), id);
	}
	
	if (priority > 0)
		return priority;
	
	priority = UpgradeRankClass(view_as<TFClassType>(info.GetInt("pclass")), slot, id);
	
	if (priority > 0)
		return priority;
	
	return UpgradeRankGeneral(id);
}






void KV_MvM_UpgradesBegin(int client)
{
	m_nPurchasedUpgrades[client] = 0;

	KeyValues kv = new KeyValues("MvM_UpgradesBegin");
	FakeClientCommandKeyValues(client, kv);
	delete kv;
}

float GetUpgradeInterval()
{
	float customInterval = redbots_manager_bot_upgrade_interval.FloatValue;
	
	if (customInterval >= 0.0)
		return customInterval;
	
	//Upgrading during an active round, buy upgrades fast
	if (GameRules_GetRoundState() == RoundState_RoundRunning)
		return GetRandomFloat(0.1, 0.75);
	
	const float interval = 1.25;
	const float variance = 0.3;
	
	return GetRandomFloat(interval - variance, interval + variance);
}

JSONObject CTFBotPurchaseUpgrades_ChooseUpgrade(int actor)
{
	int currency = TF2_GetCurrency(actor);
	
	/* The list is built once a session, not once a purchase
	
	This rebuilt and re-sorted the whole shopping list every time a bot bought anything, which is
	every 0.1 to 1.25 seconds each, and six bots at the station between rounds did it several
	times a second between them.
	
	It bought nothing. Everything the rebuild filters on, the walk below re-asks per entry anyway:
	whether the attribute can still be upgraded, what it costs now, whether the wallet share is
	spent, what the priority is. A rebuild can only ever remove entries, never add one, because
	buying an upgrade does not make another available. So the only thing it changed was how long
	the walk is, and the walk skips the same entries either way. */
	if (CTFPlayerUpgrades[actor] == null || CTFPlayerUpgrades[actor].Length == 0)
		CollectUpgrades(actor);
	
	for (int i = 0; i < CTFPlayerUpgrades[actor].Length; i++) 
	{
		JSONObject info = view_as<JSONObject>(CTFPlayerUpgrades[actor].Get(i));
		
		CMannVsMachineUpgrades upgrades = CMannVsMachineUpgradeManager().GetUpgradeByIndex(info.GetInt("index"));
		if (upgrades.Address == Address_Null)
		{
			if (redbots_manager_debug_actions.BoolValue)
				PrintToServer("CMannVsMachineUpgrades is NULL");
			
			delete info;
			return null;
		}
		
		char attrib[MAX_ATTRIBUTE_DESCRIPTION_LENGTH]; attrib = upgrades.m_szAttribute();
		CEconItemAttributeDefinition attr = CEIAD_GetAttributeDefinitionByName(attrib);
		if (attr.Address == Address_Null)
			continue;
		
		//Already refused this trip, so asking again spends the window on the same no
		if (m_bRefusedUpgrade[actor][info.GetInt("index")])
		{
			delete info;
			continue;
		}
		
		int iAttribIndex = attr.GetIndex();
		if (!CanUpgradeWithAttrib(actor, info.GetInt("slot"), iAttribIndex, upgrades.Address))
		{
			delete info;
			continue;
		}
		
		int iCost = GetCostForUpgrade(upgrades.Address, info.GetInt("slot"), info.GetInt("pclass"), actor);
		
		//This one has had its share of the wallet already
		if (!WithinAttributeShare(actor, info.GetInt("index"), iCost))
		{
			delete info;
			continue;
		}
		
		if (iCost > currency)
		{
			delete info;
			continue;
		}
	
		/* A negative priority is a refusal, not a low bid

		It used to be only a bid, so once everything worth having was maxed or unaffordable the
		bot worked down the list and bought whatever was left. Reported as Pyros buying Airblast
		Pushback Scale, which is in the canteen slot and was ranked at minus ten for exactly that
		reason. Ranking it last is not the same as never buying it. */
		if (GetUpgradePriority(actor, info) < 0)
		{
			delete info;
			continue;
		}
		
		int tier = GetUpgradeTier(info.GetInt("index"));
		if (tier != 0) 
		{
			if (!IsUpgradeTierEnabled(actor, info.GetInt("slot"), tier))
			{
				delete info;
				continue;
			}
		}
		
		return info;
	}
	
	return null;
}

/* Every tier of one upgrade the bot can afford, in one purchase

The game takes a count and applies it, refusing each step it cannot. Buying one step per interval
instead is what a play-test heard as a bot announcing the same upgrade over and over, because
each step is announced and four steps of one attribute all read the same. It is also most of what
made the chat unreadable while six bots shopped.

Nothing about what gets bought changes. The list is a strict priority and the top of it stays the
top until it maxes out, so the steps bought here in one go are the ones the next four intervals
would have bought anyway. The bot just finishes sooner and says so once */

//No stock upgrade has more steps than this, and asking for steps that do not exist buys nothing


bool CTFBotPurchaseUpgrades_PurchaseUpgrade(int actor, JSONObject info)
{
	int slot = info.GetInt("slot");
	int index = info.GetInt("index");
	int cost = GetCostForUpgrade(CMannVsMachineUpgradeManager().GetUpgradeByIndex(index).Address, slot, info.GetInt("pclass"), actor);
	int currencyBefore = TF2_GetCurrency(actor);

	int count = 1;

	if (cost > 0)
	{
		CMannVsMachineUpgrades upgrade = CMannVsMachineUpgradeManager().GetUpgradeByIndex(index);

		int tiers = UPGRADE_TIERS_MAX;

		if (upgrade.Address != Address_Null)
		{
			char attribute[MAX_ATTRIBUTE_DESCRIPTION_LENGTH]; attribute = upgrade.m_szAttribute();
			tiers = UpgradeTierCap(attribute);
		}

		count = currencyBefore / cost;

		if (count > tiers)
			count = tiers;

		if (count < 1)
			count = 1;
	}

	KV_MVM_Upgrade(actor, count, slot, index);

	int spent = currencyBefore - TF2_GetCurrency(actor);

	//The credits never moved, which is the game turning the purchase down
	if (cost > 0 && spent <= 0)
		return false;

	//An upgrade that costs nothing cannot be counted this way, so it counts as one
	m_nPurchasedUpgrades[actor] += cost > 0 ? spent / cost : 1;
	
	if (index >= 0 && index < MAX_UPGRADES)
		m_iSpentOnUpgrade[actor][index] += spent;

	return true;
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
