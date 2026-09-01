/* Does a sale come back when the money did not come from the game?
 *
 * tf2-archipelago hands out Cash Bundles by writing m_nCurrency straight onto a
 * player, because the currency pack the game itself uses cannot be told what it
 * is worth on this server. That money is real on the screen and absent from the
 * game's own record of the wave. Two play-test reports look like they come from
 * that gap: a refund that hands back the standard 400 after 1200 was spent, and
 * an upgrade that stays bought when it is sold again.
 *
 * The second one is the question here, and it did not need a play session. A
 * defender bot buys through FakeClientCommandKeyValues("MVM_Upgrade"), which is
 * the same command the upgrade station sends for a human, so a bot is a fair
 * subject. This drives one: buy an upgrade, sell it back, and write down whether
 * the credits returned. Once with money the game handed out, once with money
 * written on top of it the way a bundle is.
 *
 * If only the second fails, the sale is not a Valve bug and fixing the bundle
 * accounting fixes both. If both fail, the bundle is innocent. Either answer is
 * worth more than another evening of somebody playing.
 *
 * A test-bed plugin. It writes to the game on purpose, which is why it is not
 * in the statistics plugin, and it belongs on a test server and nowhere else.
 */

#include <sourcemod>
#include <tf2_stocks>
#include <tf2attributes>

#pragma semicolon 1
#pragma newdecls required

public Plugin myinfo =
{
	name = "MvM Defender Bots: refund probe",
	author = "m-this",
	description = "Buys and sells an upgrade on a bot, with and without money the game never recorded",
	version = "1.0.0",
	url = "https://github.com/m-this/tf2-mvm-bots"
};

/* How many upgrade indexes to offer before giving up on a subject
 *
 * The probe does not know which upgrades this subject's class and weapons accept, and asking the
 * item schema for that is a much larger plugin. So it offers them in order and watches the credits:
 * the game turns down one it does not want by not charging for it, which is the same test the mod's
 * own purchase path makes at upgrade.sp:1157.
 *
 * Eighty rather than a round forty, because forty was wrong. The mod's own notes in
 * source/redbots3/tf_upgrades.sp record that sm_dump_upgrades walked CMannVsMachineUpgradeManager
 * and printed sixty three, and the player-attached upgrades sit past the weapon ones. A cap under
 * the real count does not report a refusal, it reports a refusal that never asked the question.
 */
#define PROBE_UPGRADE_LIMIT	80

//Long enough for the station's trigger to touch the subject it was just teleported into
#define STATION_SETTLE		0.5

/* How many attributes to read off one entity
 *
 * An upgrade is an attribute, so whether a sale took is answerable by reading the attributes before
 * the buy and after the sale rather than by reading the balance. A player carries a handful; twenty
 * is the include's own default and there is no reason to want more.
 */
#define PROBE_ATTRIB_LIMIT	20

//Not a slot the game has, so it cannot collide with -1 meaning the player
#define PROBE_ANY_SLOT		-99

//Weapon slots to try, in the order the station lists them. -1 is the player, which is where resistances live.
static const int ProbeSlots[] = {-1, 0, 1, 2};

/* Results go to a file, not to the console
 *
 * The probe finishes on a timer, so its answer is written after the rcon command that started it
 * has already replied. srcds stops echoing its console once it takes it over, which is the same
 * reason the test-bed asks rcon whether the server is up rather than reading the log. A file is
 * the only place a result survives to be read.
 */
static char g_ResultPath[PLATFORM_MAX_PATH];

static void Result(const char[] format, any ...)
{
	char line[512];
	VFormat(line, sizeof(line), format, 2);
	PrintToServer("[refund-probe] %s", line);
	LogToFileEx(g_ResultPath, "%s", line);
}

public void OnPluginStart()
{
	BuildPath(Path_SM, g_ResultPath, sizeof(g_ResultPath), "logs/mvmbots_refund.jsonl");

	RegServerCmd("mvmbots_refund_probe", Command_Probe,
		"Buy and sell an upgrade. Arguments: credits to write on first (0 for none), host or bot, and a slot (-1 player, 0-2 weapons, any).");
}

/* The bot to experiment on
 *
 * A fake client on RED that is alive and has a class. The test-bed's own host is a fake client on
 * RED too and it is a statue with no money and no loadout, so it is skipped by name rather than by
 * guessing from its state.
 */
/* The test-bed host, which is the only body on this server holding money the game itself issued
 *
 * Every credit a defender bot has was written onto m_nCurrency from outside: the mod pays them with
 * TF2_SetCurrency, which is SetEntProp on that same property, which is what tf2-archipelago does for
 * a Cash Bundle. So a bot cannot be the control leg of this experiment. There is no game-recorded
 * money on one to compare against.
 *
 * The host joined the way a player joins and the game handed it the mission's starting currency. It
 * is a statue with no AI, which is a problem solved below rather than here.
 */
static int ProbeHost()
{
	char host[MAX_NAME_LENGTH];
	ConVar hostName = FindConVar("mvmbots_host_name");
	if (hostName != null)
		hostName.GetString(host, sizeof(host));
	if (host[0] == '\0')
		return -1;

	for (int client = 1; client <= MaxClients; client++)
	{
		if (!IsClientInGame(client) || !IsPlayerAlive(client))
			continue;
		char name[MAX_NAME_LENGTH]; GetClientName(client, name, sizeof(name));
		if (StrEqual(name, host))
			return client;
	}
	return -1;
}

/* Standing in the upgrade station, because the game checks
 *
 * MvM refuses a purchase from somebody who is not inside a func_upgradestation, and the host never
 * walks anywhere. That is why it accepted nothing at 400 credits: not a refusal worth reporting,
 * just a body in the wrong place.
 *
 * Two things here are not obvious and both cost a run to find.
 *
 * A func_upgradestation is a brush, and a brush entity keeps its origin at the world origin unless
 * the mapper moved it. Teleporting to m_vecOrigin therefore drops the subject at 0,0,0, which on
 * most maps is somewhere under the floor. The centre of its bounding box is the position that
 * means what the name suggests.
 *
 * And the game does not decide you are in the zone when you arrive. The trigger sets
 * m_bInUpgradeZone when it next touches you, which is the following tick at the earliest, so a
 * purchase sent in the same frame as the teleport is always refused. Everything after the move
 * happens on a timer for that reason.
 */
static bool StationCentre(float centre[3])
{
	int station = FindEntityByClassname(-1, "func_upgradestation");
	if (station == -1)
		return false;

	float mins[3], maxs[3];
	GetEntPropVector(station, Prop_Send, "m_vecOrigin", centre);
	GetEntPropVector(station, Prop_Send, "m_vecMins", mins);
	GetEntPropVector(station, Prop_Send, "m_vecMaxs", maxs);
	for (int i = 0; i < 3; i++)
		centre[i] += (mins[i] + maxs[i]) / 2.0;

	return true;
}

static bool InUpgradeZone(int client)
{
	return HasEntProp(client, Prop_Send, "m_bInUpgradeZone")
		&& GetEntProp(client, Prop_Send, "m_bInUpgradeZone") != 0;
}

static int ProbeSubject()
{
	/* Looked up here rather than at plugin start, because at plugin start it does not exist yet.
	 * SourceMod loads plugins in directory order, this one sorts before mvmbots_host, and a
	 * FindConVar for a convar the host plugin has not created returns null. That null read as
	 * "no host to skip", so the probe cheerfully experimented on the statue: 400 credits, no
	 * loadout, and nothing it was offered was ever going to be accepted. */
	char host[MAX_NAME_LENGTH];
	ConVar hostName = FindConVar("mvmbots_host_name");
	if (hostName != null)
		hostName.GetString(host, sizeof(host));

	for (int client = 1; client <= MaxClients; client++)
	{
		if (!IsClientInGame(client) || !IsFakeClient(client))
			continue;
		if (GetClientTeam(client) != view_as<int>(TFTeam_Red) || !IsPlayerAlive(client))
			continue;
		if (TF2_GetPlayerClass(client) == TFClass_Unknown)
			continue;

		char name[MAX_NAME_LENGTH]; GetClientName(client, name, sizeof(name));
		if (host[0] != '\0' && StrEqual(name, host))
			continue;

		return client;
	}
	return -1;
}

/* The attributes on an entity, so the same list can be taken twice and compared
 *
 * This is the measurement the balance could not make. The credits coming back says the game
 * accepted the sale; it does not say the upgrade came off. A sale that refunds and leaves the
 * attribute in place is the reported bug and passes every test of the balance.
 */
static int Attribs(int entity, int[] defs, float[] values)
{
	int count = TF2Attrib_ListDefIndices(entity, defs, PROBE_ATTRIB_LIMIT);
	if (count > PROBE_ATTRIB_LIMIT)
		count = PROBE_ATTRIB_LIMIT;

	for (int i = 0; i < count; i++)
	{
		Address attrib = TF2Attrib_GetByDefIndex(entity, defs[i]);
		values[i] = attrib == Address_Null ? 0.0 : TF2Attrib_GetValue(attrib);
	}
	return count;
}

//The value of one attribute on an entity, or zero when it is not there at all
static float AttribValue(int entity, int def)
{
	Address attrib = TF2Attrib_GetByDefIndex(entity, def);
	return attrib == Address_Null ? 0.0 : TF2Attrib_GetValue(attrib);
}

/* The attribute the buy put on, found by taking the list twice
 *
 * Naming it from the upgrade index would mean reading CMannVsMachineUpgradeManager, which is the
 * mod's gamedata and not this plugin's. Comparing before with after needs none of that: whatever
 * appeared, or changed value, is what was bought.
 */
static int BoughtAttrib(const int[] before, int nBefore, const float[] valuesBefore,
	const int[] after, int nAfter, const float[] valuesAfter, float &wanted)
{
	for (int i = 0; i < nAfter; i++)
	{
		bool seen = false;
		for (int j = 0; j < nBefore; j++)
		{
			if (before[j] != after[i])
				continue;
			seen = true;
			if (valuesBefore[j] != valuesAfter[i])
			{
				wanted = valuesAfter[i];
				return after[i];
			}
			break;
		}
		if (!seen)
		{
			wanted = valuesAfter[i];
			return after[i];
		}
	}
	return -1;
}

static int Credits(int client)
{
	return HasEntProp(client, Prop_Send, "m_nCurrency")
		? GetEntProp(client, Prop_Send, "m_nCurrency")
		: -1;
}

/* Opening and closing the session the station opens
 *
 * The game does not accept a bare MVM_Upgrade. The station opens a session first and closes it
 * after, and a purchase outside one is refused without a word: the credits simply do not move,
 * which reads exactly like an upgrade the class cannot take. That cost three runs here, with the
 * subject standing in the zone holding eighteen hundred credits and every one of eighty upgrades
 * turned down across four slots.
 *
 * The defender mod has always done this (KV_MvM_UpgradesBegin in behavior/upgrade.sp), which is
 * why its bots buy and this probe did not.
 *
 * Note the spelling. The session is MvM_UpgradesBegin and MvM_UpgradesDone, the purchase is
 * MVM_Upgrade. The game is inconsistent about the capital V and it does not forgive it.
 */
static void SendSession(int client, const char[] name)
{
	KeyValues kv = new KeyValues(name);
	FakeClientCommandKeyValues(client, kv);
	delete kv;
}

//The station's own command, which is how a bot buys and how a human buys. A negative count sells.
static void SendUpgrade(int client, int slot, int index, int count)
{
	KeyValues kv = new KeyValues("MVM_Upgrade");
	kv.JumpToKey("upgrade", true);
	kv.SetNum("itemslot", slot);
	kv.SetNum("upgrade", index);
	kv.SetNum("count", count);
	FakeClientCommandKeyValues(client, kv);
	delete kv;
}

/* An upgrade this bot will actually buy, bought
 *
 * Returns what it cost, 0 when nothing was accepted. The slot and index that worked are written
 * back so the sale can name the same one: selling a different upgrade would answer a different
 * question.
 */
static int BuyAnything(int client, int &slotOut, int &indexOut, int onlySlot)
{
	for (int s = 0; s < sizeof(ProbeSlots); s++)
	{
		// A run can name the slot it cares about. The first upgrade any subject accepts is a
		// player one, so left to itself this never tests a weapon, and a weapon upgrade lives on
		// a different entity and is refunded down a different path.
		if (onlySlot != PROBE_ANY_SLOT && ProbeSlots[s] != onlySlot)
			continue;

		for (int index = 0; index < PROBE_UPGRADE_LIMIT; index++)
		{
			int before = Credits(client);
			SendUpgrade(client, ProbeSlots[s], index, 1);
			int spent = before - Credits(client);
			if (spent > 0)
			{
				slotOut = ProbeSlots[s];
				indexOut = index;
				return spent;
			}
		}
	}
	return 0;
}

/* One probe at a time, because the subject is teleported and two at once would fight over where it
 * stands. A second call while one is in flight is refused rather than queued. */
static bool g_Running;
static int g_Bundle;
static bool g_OnHost;
static int g_OnlySlot;
static float g_Back[3];

public Action Command_Probe(int argc)
{
	if (g_Running)
	{
		PrintToServer("[refund-probe] one is already in flight. Nothing done.");
		return Plugin_Handled;
	}
	if (GameRules_GetRoundState() != RoundState_BetweenRounds)
	{
		PrintToServer("[refund-probe] not between waves, so the upgrade station is shut. Nothing done.");
		return Plugin_Handled;
	}

	char who[16] = "bot";
	if (argc >= 2)
		GetCmdArg(2, who, sizeof(who));
	g_OnHost = StrEqual(who, "host", false);

	int client = g_OnHost ? ProbeHost() : ProbeSubject();
	if (client == -1)
	{
		PrintToServer("[refund-probe] no %s alive to experiment on. Nothing done.", g_OnHost ? "host" : "defender bot");
		return Plugin_Handled;
	}
	if (Credits(client) == -1)
	{
		PrintToServer("[refund-probe] this server has no m_nCurrency. Nothing done.");
		return Plugin_Handled;
	}

	g_OnlySlot = PROBE_ANY_SLOT;
	if (argc >= 3)
	{
		char arg[8]; GetCmdArg(3, arg, sizeof(arg));
		if (!StrEqual(arg, "any", false))
			g_OnlySlot = StringToInt(arg);
	}

	g_Bundle = 0;
	if (argc >= 1)
	{
		char arg[16]; GetCmdArg(1, arg, sizeof(arg));
		g_Bundle = StringToInt(arg);
	}

	float centre[3];
	if (!StationCentre(centre))
	{
		PrintToServer("[refund-probe] no func_upgradestation on this map, so a refusal here would mean nothing. Nothing done.");
		return Plugin_Handled;
	}

	GetEntPropVector(client, Prop_Data, "m_vecAbsOrigin", g_Back);
	TeleportEntity(client, centre, NULL_VECTOR, NULL_VECTOR);
	PrintToServer("[refund-probe] %N moved to the station at %.0f %.0f %.0f, buying next tick.",
		client, centre[0], centre[1], centre[2]);

	g_Running = true;
	CreateTimer(STATION_SETTLE, Timer_BuyAndSell, GetClientUserId(client));
	return Plugin_Handled;
}

static Action Timer_BuyAndSell(Handle timer, any userid)
{
	g_Running = false;

	int client = GetClientOfUserId(userid);
	if (client <= 0 || !IsClientInGame(client) || !IsPlayerAlive(client))
	{
		Result("the subject left before it could buy. Nothing done.");
		return Plugin_Stop;
	}

	if (!InUpgradeZone(client))
	{
		TeleportEntity(client, g_Back, NULL_VECTOR, NULL_VECTOR);
		Result("%N is standing in the station and the game still says it is not in the zone. A refusal here says nothing about selling.",
			client);
		return Plugin_Stop;
	}

	int started = Credits(client);
	// The bundle, written exactly the way tf2-archipelago writes one: on top of the balance, with
	// the game's record of the wave left saying what it said before.
	if (g_Bundle > 0)
		SetEntProp(client, Prop_Send, "m_nCurrency", started + g_Bundle);
	int held = Credits(client);

	SendSession(client, "MvM_UpgradesBegin");

	// The player carries the slot -1 upgrades. A weapon upgrade lands on the weapon, so the
	// entity that is read has to be the one the upgrade was bought for, decided after the buy.
	int defsBefore[PROBE_ATTRIB_LIMIT]; float valuesBefore[PROBE_ATTRIB_LIMIT];
	int nBefore = Attribs(client, defsBefore, valuesBefore);

	int slot = 0, index = 0;
	int spent = BuyAnything(client, slot, index, g_OnlySlot);
	if (spent <= 0)
	{
		SendSession(client, "MvM_UpgradesDone");
		TeleportEntity(client, g_Back, NULL_VECTOR, NULL_VECTOR);
		/* Why it bought nothing, because the sentence on its own is not a finding.
		 *
		 * A subject with no weapons can only take the upgrades the game attaches to the player,
		 * and one that is not really wearing a class cannot take those either. Saying which slots
		 * hold a weapon turns "no upgrade accepted" from a dead end into either a fact about the
		 * sale or a fact about the subject. */
		char slots[64];
		FormatEx(slots, sizeof(slots), "%d/%d/%d",
			GetPlayerWeaponSlot(client, 0), GetPlayerWeaponSlot(client, 1), GetPlayerWeaponSlot(client, 2));
		Result("%N bought nothing in slot %d: %d credits in the zone, class %d, weapons in slots 0/1/2 are %s, and none of %d upgrades was accepted. Nothing to sell.",
			client, g_OnlySlot, held, view_as<int>(TF2_GetPlayerClass(client)), slots, PROBE_UPGRADE_LIMIT);
		return Plugin_Stop;
	}

	int afterBuy = Credits(client);

	/* Which entity the upgrade landed on
	 *
	 * A slot -1 upgrade is attached to the player. Anything else is attached to the weapon in that
	 * slot, and reading the player for it would report every sale as clean because the attribute
	 * was never there to begin with. */
	int carrier = slot == -1 ? client : GetPlayerWeaponSlot(client, slot);
	if (carrier == -1)
		carrier = client;

	int defsAfter[PROBE_ATTRIB_LIMIT]; float valuesAfter[PROBE_ATTRIB_LIMIT];
	int nAfter = Attribs(carrier, defsAfter, valuesAfter);

	float bought = 0.0;
	int attrib = carrier == client
		? BoughtAttrib(defsBefore, nBefore, valuesBefore, defsAfter, nAfter, valuesAfter, bought)
		: -1;
	// A weapon carrier has no before-list of its own, so the attribute is whatever it now holds
	// that the player does not. One entry is enough: the buy applied exactly one upgrade.
	if (carrier != client && nAfter > 0)
	{
		attrib = defsAfter[0];
		bought = valuesAfter[0];
	}

	SendUpgrade(client, slot, index, -1);
	int afterSell = Credits(client);
	int returned = afterSell - afterBuy;

	float left = attrib == -1 ? 0.0 : AttribValue(carrier, attrib);
	// The upgrade came off when the attribute is gone, or back to what it was worth before.
	bool removed = attrib == -1 ? false : (left != bought);

	SendSession(client, "MvM_UpgradesDone");
	TeleportEntity(client, g_Back, NULL_VECTOR, NULL_VECTOR);

	// One line, the same shape as the statistics plugin's, so two probes diff against each other.
	Result("{\"subject\":\"%s\",\"bundle\":%d,\"started\":%d,\"held\":%d,\"slot\":%d,\"upgrade\":%d,\"spent\":%d,\"after_buy\":%d,\"after_sell\":%d,\"returned\":%d,\"credits_back\":%s,\"attrib\":%d,\"attrib_bought\":%.3f,\"attrib_after_sell\":%.3f,\"upgrade_removed\":%s}",
		g_OnHost ? "host" : "bot", g_Bundle, started, held, slot, index, spent, afterBuy, afterSell, returned,
		returned == spent ? "true" : "false", attrib, bought, left, removed ? "true" : "false");

	if (attrib == -1)
		Result("the buy put no attribute anywhere this probe can read, so the sale cannot be judged.");
	else if (returned == spent && removed)
		Result("the sale took: the credits came back and the upgrade came off.");
	else if (returned == spent && !removed)
		Result("THE REPORTED BUG: the credits came back and the upgrade stayed on at %.3f.", left);
	else if (returned == 0 && !removed)
		Result("the sale did nothing at all: no credits back and the upgrade still on.");
	else
		Result("the sale returned %d of %d and the upgrade is %s, which is neither answer and wants looking at.",
			returned, spent, removed ? "off" : "on");

	return Plugin_Stop;
}
