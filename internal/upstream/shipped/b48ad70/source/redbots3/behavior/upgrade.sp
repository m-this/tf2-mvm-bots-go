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
/* Upgrades that do nothing for the weapons this bot is actually holding

Reported on 1.8: Pyros buying Explode on Ignite with no Gas Passer, and airblast pushback while
carrying a Phlogistinator, which has no airblast at all. Both are the upgrade menu offering
everything the class can theoretically use rather than what this loadout can.

The menu is right to offer them. Deciding is this mod's job. */
static bool IsUpgradeWasted(int client, const char[] attribute)
{
	//Explode on Ignite is the Gas Passer's, and nothing else can be ignited into exploding
	if (StrContains(attribute, "explode_on_ignite", false) != -1
		|| StrContains(attribute, "explode on ignite", false) != -1)
	{
		int secondary = GetPlayerWeaponSlot(client, TFWeaponSlot_Secondary);

		return secondary == -1 || TF2Util_GetWeaponID(secondary) != TF_WEAPON_JAR_GAS;
	}

	if (StrContains(attribute, "airblast", false) != -1)
	{
		int primary = GetPlayerWeaponSlot(client, TFWeaponSlot_Primary);

		if (primary == -1 || TF2Util_GetWeaponID(primary) != TF_WEAPON_FLAMETHROWER)
			return true;

		//A flamethrower that cannot airblast, which is the Phlogistinator and anything like it
		return TF2Attrib_GetByName(primary, "airblast disabled") != Address_Null;
	}

	/* Destroy Projectiles is an airblast on a Pyro and a spun-up minigun on a Heavy

	Same attribute, two different things behind it, and the guides rate it for both because a
	person carrying a Phlogistinator knows they have given up the airblast. The upgrade menu does
	not, and this table did not either: the Pyro's own line ranked it at 250 while the loadout
	handed him the one flamethrower that cannot do it. */
	if (StrEqual(attribute, "attack projectiles") && TF2_GetPlayerClass(client) == TFClass_Pyro)
	{
		int primary = GetPlayerWeaponSlot(client, TFWeaponSlot_Primary);

		if (primary == -1)
			return true;

		return TF2Attrib_GetByName(primary, "airblast disabled") != Address_Null;
	}

	/* The Projectile Shield, which nothing in this mod presses

	Every guide puts one tick of it first for a Medic and they are right about a person: it is the
	strongest thing a Medic can do to a wave. It is deployed with the special attack key, and no
	behaviour here has ever pressed one, so what the rage meter fills is a button nobody uses.

	Three hundred credits for that, ranked at the top of the Medic's list, every wave. It goes back
	the moment something deploys it, and that is the TODO rather than this. */
	if (StrEqual(attribute, "generate rage on heal"))
		return !Feature(FEATURE_MEDIC_SHIELD);

	/* Afterburn, which the wiki calls useless and a bot has even less use for

	It does not scale the way direct damage does, a small robot dies before it finishes ticking,
	and a giant outlives it. */
	if (StrEqual(attribute, "weapon burn dmg increased") || StrEqual(attribute, "weapon burn time increased"))
		return true;

	return false;
}

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
	
	int priority = LoadoutUpgradePriority(client, slot, attribute);
	
	if (priority > 0)
		return priority;
	
	priority = ClassUpgradePriority(view_as<TFClassType>(info.GetInt("pclass")), slot, attribute);
	
	if (priority > 0)
		return priority;
	
	return GeneralUpgradePriority(attribute);
}

/* The upgrade that is the reason to carry this weapon at all, by item definition index

Zero when the weapon in that slot has no opinion, which is most of them: this only names the few
where the loadout, not the class, decides what to buy first */
static int LoadoutUpgradePriority(int client, int slot, const char[] attribute)
{
	/* An engineer whose gun is paid for in metal

	The metal upgrades do not hang off the gun, so the switch below cannot answer for them however
	the loadout is put together. A Widowmaker engineer without them fights the wave out of the same
	supply the sentry is repaired from, and runs out of both. Under the sentry's own fire rate,
	above everything else the class buys */
	if (TF2_GetPlayerClass(client) == TFClass_Engineer && EngineerGunSpendsMetal(client))
	{
		if (StrEqual(attribute, "maxammo metal increased")) return 310;
		if (StrEqual(attribute, "metal regen")) return 305;
	}

	if (slot < TF_LOADOUT_SLOT_PRIMARY || slot > TF_LOADOUT_SLOT_MELEE)
		return 0;

	int weapon = GetPlayerWeaponSlot(client, slot);
	
	if (weapon < 1 || !HasEntProp(weapon, Prop_Send, "m_iItemDefinitionIndex"))
		return 0;
	
	switch (GetEntProp(weapon, Prop_Send, "m_iItemDefinitionIndex"))
	{
		case 35: //Kritzkrieg: the crits are the weapon, so the meter is what matters
		{
			if (StrEqual(attribute, "ubercharge rate bonus")) return 330;
		}
		case 411: //Quick-Fix: it heals rather than saves, so it should heal faster
		{
			if (StrEqual(attribute, "healing mastery")) return 330;
			if (StrEqual(attribute, "ubercharge rate bonus")) return 300;
		}
		case 312: //Brass Beast: the damage minigun, and it cannot reposition to make up for less
		{
			if (StrEqual(attribute, "damage bonus")) return 320;
		}
		case 424: //Tomislav: it already fires fast, so damage per bullet beats more bullets
		{
			if (StrEqual(attribute, "damage bonus")) return 300;
		}
		case 752: //Hitman's Heatmaker: reach the shot sooner
		{
			if (StrEqual(attribute, "SRifle Charge rate increased")) return 300;
		}
		case 526: //Machina: every shot is a charged one, so damage rides on all of them
		{
			if (StrEqual(attribute, "damage bonus")) return 300;
		}
		case 996: //Loose Cannon: a faster cannonball is one a bot can actually land
		{
			if (StrEqual(attribute, "Projectile speed increased")) return 300;
		}
		case 997: //Rescue Ranger: every shot and every repair at range costs metal
		{
			if (StrEqual(attribute, "metal regen")) return 300;
			if (StrEqual(attribute, "maxammo metal increased")) return 290;
		}
		case 730: //Beggar's Bazooka: it fires as fast as the button is pressed, so buying that twice is buying nothing
		{
			if (StrEqual(attribute, "fire rate bonus")) return 20;
			//The clip is the burst, and the burst is the whole weapon
			if (StrEqual(attribute, "clip size upgrade atomic")) return 280;
			if (StrEqual(attribute, "clip size bonus upgrade")) return 280;
		}
		case 527: //Widowmaker: the shot is paid for in metal and paid back in damage dealt
		{
			if (StrEqual(attribute, "damage bonus")) return 300;
			if (StrEqual(attribute, "fire rate bonus")) return 250;
		}
		case 528: //Short Circuit: it eats projectiles rather than robots, and eats metal doing it
		{
			if (StrEqual(attribute, "metal regen")) return 300;
		}
		case 141: //Frontier Justice: the crits are banked by the sentry, so the clip is what holds them
		{
			if (StrEqual(attribute, "clip size upgrade atomic")) return 260;
			if (StrEqual(attribute, "clip size bonus upgrade")) return 260;
			if (StrEqual(attribute, "damage bonus")) return 250;
		}
	}
	
	return 0;
}

/* What this class contributes with, which is not always the weapon in its hands */
static int ClassUpgradePriority(TFClassType pclass, int slot, const char[] attribute)
{
	/* Blast resistance was given a floor for these two and it did not pay
	
	They explode themselves in every wave whatever the robots carry, so pricing the resistance by
	the wave looked like the wrong question for them. The resistance does work: six waves on Decoy
	took the Soldier's self damage from 3272 to 2147 and his deaths to his own rockets from three
	to one.
	
	It bought nothing with it. Damage came out flat, 14187 to 13485, the waves cleared were the
	same three either way, and the team died more, 40 against 50. The credits come out of upgrades
	that do produce damage, and on this harness a five percent swing over six waves is inside the
	noise anyway. Deleted rather than left switched off. */

	switch (pclass)
	{
		case TFClass_Engineer:
		{
			/* The gun, which is bought with what the nest leaves over

			Handled before the sentry lines below, and ranked under every one of them, because
			the general table would otherwise put "damage bonus" at 260 and buy shotgun damage
			ahead of the metal that keeps the sentry firing. A weapon that is the reason to carry
			the loadout says so in LoadoutUpgradePriority instead, which runs before this */
			if (slot == TF_LOADOUT_SLOT_PRIMARY || slot == TF_LOADOUT_SLOT_SECONDARY)
			{
				if (StrEqual(attribute, "damage bonus")) return 200;
				if (StrEqual(attribute, "fire rate bonus")) return 190;
				if (StrEqual(attribute, "clip size upgrade atomic")) return 150;
				if (StrEqual(attribute, "clip size bonus upgrade")) return 150;
				if (StrEqual(attribute, "faster reload rate")) return 140;
				if (StrEqual(attribute, "maxammo primary increased")) return 130;
				if (StrEqual(attribute, "maxammo secondary increased")) return 120;

				//Anything else on the gun is worth less than the cheapest thing the nest wants
				return 50;
			}

			/* The sentry is the damage. The shotgun only defends it

			These rankings are the mod's own and they stay: the wiki puts dispenser range first and
			sentry firing speed under what to avoid, and a play-test of this mod put the radius
			where it is for a reason the wiki has no way to know about. A bot that is hurt or low
			on ammo holds the bomb from a friendly dispenser rather than walking off to a health
			pack, so the radius is how much of the team that covers, and it was reported as
			"incredibly useful, a very good upgrade to max out early" from play here.

			Measured beats read. If the ordering is to change it should change because a run said
			so, not because a page did. */
			if (StrEqual(attribute, "engy dispenser radius increased")) return 330;
			if (StrEqual(attribute, "engy sentry fire rate increased")) return 320;
			/* A second gun, now that something puts it somewhere on purpose

			Every guide says never, and every guide is describing a person who drops one and
			forgets it. It was refused here for a while for a better reason than that: nothing in
			this mod placed it, so the game put it wherever the engineer happened to be facing,
			and what that produced was minis pointing at walls.

			behavior/engineerbuilddisposable.sp stands it beside the real one now, on ground that
			can see what the real one sees, so the upgrade buys what it is supposed to buy. */
			/* Off, this is not bought at all rather than bought late
			
			Two players have now reported the mini as the wrong purchase, and a third report was
			"3 sentries with 2 engineers, THIS IS NOT POSSIBLE" — which it is, and this line is
			how. At 310 it outranks building health at 260 and metal capacity at 210, so it is
			bought before the nest is durable. Ranking it lower would still buy it on a rich wave,
			and the objection is to buying it, so the switch refuses it outright. See mvm-8ws. */
			if (StrEqual(attribute, "engy disposable sentries"))
				return Feature(FEATURE_ENGINEER_DISPOSABLE) ? 310 : -10;
			
			if (StrEqual(attribute, "engy building health bonus")) return 260;
			if (StrEqual(attribute, "metal regen")) return 220;
			if (StrEqual(attribute, "maxammo metal increased")) return 210;
			//The Jag swings faster, and swinging is what builds and repairs the nest
			if (StrEqual(attribute, "melee attack rate bonus")) return 200;
		}
		case TFClass_Medic:
		{
			//A Medic that shoots is a Medic not healing, so its own damage comes last
			if (StrEqual(attribute, "generate rage on heal")) return 320;
			if (StrEqual(attribute, "ubercharge rate bonus")) return 300;
			if (StrEqual(attribute, "healing mastery")) return 280;
			if (StrEqual(attribute, "uber duration bonus")) return 230;
			if (StrEqual(attribute, "overheal expert")) return 210;
			if (StrEqual(attribute, "damage bonus")) return 40;
			if (StrEqual(attribute, "fire rate bonus")) return 40;
		}
		case TFClass_Sniper:
		{
			/* One shot through a line of robots, then the speed to take the next one

			"The first upgrade you should always get, regardless of context or starting credits,
			is one tick of Explosive Headshot", then reload speed, then the rest of it.

			Charge rate was ranked second here and the guides rank it nowhere: it is better to
			land repeated quick shots than to hold one full-damage shot, so damage buys more than
			charge does. */
			if (StrEqual(attribute, "explosive sniper shot")) return 330;
			if (StrEqual(attribute, "faster reload rate")) return 300;
			if (StrEqual(attribute, "SRifle Charge rate increased")) return 60;
		}
		case TFClass_Spy:
		{
			if (slot == TF_LOADOUT_SLOT_MELEE)
			{
				//A backstab through a giant's armour is the whole class in this mode
				//Full armour penetration first, then the swing speed, and the sapper long after
				if (StrEqual(attribute, "armor piercing")) return 330;
				if (StrEqual(attribute, "melee attack rate bonus")) return 280;
				if (StrEqual(attribute, "robo sapper")) return 70;
			}
		}
		case TFClass_Pyro:
		{
			/* The flames first, and as fast as the wallet allows

			"You should try to max out your primary's damage as quickly as possible." Reflecting
			what is aimed at the team comes after that, not before it. */
			if (StrEqual(attribute, "damage bonus")) return 320;
			if (StrEqual(attribute, "attack projectiles")) return 250;
			/* Afterburn is the last thing to spend a credit on, whatever the flamethrower
			It does not scale with the upgrade the way direct damage does, a robot dies before it
			finishes ticking, and a giant outlives it. That holds for the Phlogistinator too: its
			taunt fills from damage dealt, and direct flame damage is most of what it deals */
			//Afterburn is refused outright in IsUpgradeWasted
		}
		case TFClass_Soldier:
		{
			/* Reload first, and everybody who writes about this class says so

			A Soldier's damage is four rockets and then a wait, so the wait is what the credits
			buy. Rocket Specialist next and exactly one tick of it, capped in
			CTFBotPurchaseUpgrades_PurchaseUpgrade: the first removes the falloff, the rest widen
			a blast radius nobody needed. Then damage, then the rest of the general table */
			if (StrEqual(attribute, "faster reload rate")) return 310;
			if (StrEqual(attribute, "rocket specialist")) return 290;
			//"Very helpful throughout the whole game", and it goes under the damage, not over it
			if (StrEqual(attribute, "heal on kill")) return 250;
		}
		case TFClass_DemoMan:
		{
			/* Reload, then fire rate, then everything else

			Same shape as the Soldier and for the same reason: the launcher is a burst and a
			reload, so the reload is the damage. Fire rate second because a sticky trap is
			however many bombs he got down before the robots arrived.

			Projectile speed is on the list because it is cheap and it is what makes a pipe land
			on something that is walking, which is the shot a bot is worst at leading */
			if (StrEqual(attribute, "faster reload rate")) return 310;
			if (StrEqual(attribute, "fire rate bonus")) return 290;
			if (StrEqual(attribute, "Projectile speed increased")) return 200;
		}
		case TFClass_Heavy:
		{
			/* Staying alive is the first upgrade, because a dead Heavy shoots nothing

			"If you're dead, you're not shooting the robots" is the whole argument and it is the
			first line of every Heavy guide. This class had one rule before, for shooting down
			rockets, and took the general table's damage-first answer for everything else.

			Firing speed is deliberately not raised here. The second tick of it does nothing at
			all, which is a known bug rather than an opinion, so it is capped at one in
			UpgradeTierCap and left to rank where the general table puts it. */
			if (StrEqual(attribute, "heal on kill")) return 320;
			//Shooting down the rockets aimed at the team
			if (StrEqual(attribute, "attack projectiles")) return 230;
		}
		case TFClass_Scout:
		{
			//Milk marks a wave for the whole team, which is worth more than what one Scout shoots
			if (StrEqual(attribute, "applies snare effect")) return 250;
			/* Jump height is a person's tool and it stays at the bottom of the general table

			The guides want two ticks of it early on Mannhattan and Decoy, for reaching credits on
			a ledge before they expire. A person reads a ledge and jumps at it. A bot walks where
			the nav mesh says it can walk, and buying it more jump does not add a route to the
			mesh, so what the credits buy is a bot that jumps higher along the same path.

			Worth revisiting if anything here ever aims a jump at a specific piece of ground. */
			if (StrEqual(attribute, "mad milk syringes")) return 200;
			//Money is the Scout's job here and it needs the legs to do it
			if (StrEqual(attribute, "move speed bonus")) return 190;
		}
	}
	
	return 0;
}

/* What a resistance is worth, given whether the wave will actually deal that damage

Below the damage upgrades on purpose. A team that kills the wave faster takes less of everything,
and the guides all put resistances after the weapon is bought. Above the rest of the tail,
because the alternative is what this mod did before, which was never buying one. */
static int ResistancePriority(bool wanted)
{
	if (!Feature(FEATURE_WAVE_RESISTANCES))
		return 35;
	
	return wanted ? 210 : 25;
}

/* Damage first, then what keeps it firing. What a bot buys when nothing above had an opinion */
static int GeneralUpgradePriority(const char[] attribute)
{
	//--- The damage itself
	if (StrEqual(attribute, "damage bonus")) return 260;
	if (StrEqual(attribute, "fire rate bonus")) return 250;
	if (StrEqual(attribute, "melee attack rate bonus")) return 200;
	if (StrEqual(attribute, "projectile penetration")) return 190;
	if (StrEqual(attribute, "projectile penetration heavy")) return 190;
	if (StrEqual(attribute, "critboost on kill")) return 180;
	
	//--- Keeping it firing
	if (StrEqual(attribute, "clip size upgrade atomic")) return 170;
	if (StrEqual(attribute, "clip size bonus upgrade")) return 170;
	if (StrEqual(attribute, "faster reload rate")) return 160;
	if (StrEqual(attribute, "maxammo primary increased")) return 150;
	if (StrEqual(attribute, "Projectile speed increased")) return 130;
	if (StrEqual(attribute, "maxammo secondary increased")) return 120;
	
	//--- Worth having once the damage is bought
	if (StrEqual(attribute, "heal on kill")) return 110;
	if (StrEqual(attribute, "mark for death")) return 90;
	if (StrEqual(attribute, "armor piercing")) return 85;
	if (StrEqual(attribute, "attack projectiles")) return 80;
	if (StrEqual(attribute, "increase buff duration")) return 75;
	if (StrEqual(attribute, "effect bar recharge rate increased")) return 70;
	if (StrEqual(attribute, "charge recharge rate increased")) return 70;
	if (StrEqual(attribute, "generate rage on damage")) return 60;
	if (StrEqual(attribute, "bleeding duration")) return 55;
	
	/* --- Not dying, which was ranked here on a premise that is not true

	This block used to open with "a bot respawns every wave, so staying alive is what it needs
	least". Bots do not respawn every wave, and the test-bed says explosions are between forty
	five and sixty percent of every defender death on every map measured. A resistance ranked at
	thirty five is a resistance nobody ever buys.

	What the guides do about it is buy the resistance the coming wave calls for, and the wave bar
	says what is coming before it starts. So a resistance is worth a middling amount when the
	robots that deal that damage are in the wave, and very little when they are not: blast
	resistance against a wave of Scouts is three hundred credits spent on nothing. */
	if (StrEqual(attribute, "dmg taken from blast reduced"))
		return ResistancePriority(WaveHasExplosiveRobots());
	
	if (StrEqual(attribute, "dmg taken from bullets reduced"))
		return ResistancePriority(WaveHasBulletRobots());
	
	if (StrEqual(attribute, "dmg taken from fire reduced"))
		return ResistancePriority(WaveHasFireRobots());
	
	if (StrEqual(attribute, "move speed bonus")) return 45;
	if (StrEqual(attribute, "health regen")) return 40;
	if (StrEqual(attribute, "dmg taken from crit reduced")) return 30;
	if (StrEqual(attribute, "damage force reduction")) return 25;
	if (StrEqual(attribute, "increased jump height")) return 10;
	
	return UnrankedUpgradePriority();
}

/* An upgrade no table above recognised

The mod's own answer for every upgrade, kept for the ones this file does not name. It has to
stay random: a constant would tie every unknown upgrade, and a tie is broken by whichever the
game listed first, so a bot would buy the same wrong thing every wave of every mission */
static int UnrankedUpgradePriority()
{
	return GetRandomInt(50, 100);
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
#define UPGRADE_TIERS_MAX	4

/* Upgrades where the steps after the first buy nothing worth having

Rocket Specialist is the one everybody names: the first tick removes the damage falloff and slows
what it hits, and the three after it are three hundred credits each for a bigger blast radius
nobody asked for. Buying all four is the single most expensive mistake a Soldier can make here,
and the batching above would buy all four in one go.

A cap of one, not a ban: the tick itself is worth having, which is why it ranks high */
static int UpgradeTierCap(const char[] attribute)
{
	if (StrEqual(attribute, "rocket specialist"))
		return 1;

	return UPGRADE_TIERS_MAX;
}

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
