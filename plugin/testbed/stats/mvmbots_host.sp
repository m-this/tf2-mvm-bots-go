/* The seats the test-bed puts on RED, and nobody else does
 *
 * Two kinds, and they are different jobs.
 *
 * The host is one fake player so that a server with nobody on it can start a
 * wave. The defender mod adds its bots in response to a human pressing F4, its
 * ready listener passes its own bots straight through, and the wave itself
 * needs a ready player before Mann vs Machine will begin one. So an empty
 * server sits in the pre-round forever: no ready, no wave, no bots, nothing to
 * measure. The host has no AI, it does not move and it does not shoot: it is a
 * body holding a seat so the game will start.
 *
 * A puppet is the opposite. It stands where a player stands, and a run drives
 * it: `mvmbots_puppet_call` presses MEDIC!, the same voicemenu command a
 * player's key sends, so a fault that needs a person on RED can be measured
 * without one. It is off by default and a run asks for it. See mvm-n4s.
 *
 * A puppet is only a player if the mod agrees it is one, and IsTFBotPlayer is
 * IsFakeClient, which every seat here trips. The mod answers that question by
 * the nextbot instead under sm_redbots_feature_bot_test_by_nextbot, and a run
 * measuring a player has to turn that on. Without it a puppet is a statue with
 * a name.
 *
 * The cost is honest and worth stating: a six seat RED team with the host in
 * one of the seats is five bots and a statue, and every puppet takes another.
 * mvmbots_roster names all three kinds separately, so a run can always say
 * which of RED were the mod's.
 *
 * A test-bed plugin. It belongs on a test server and nowhere else.
 */

#include <sourcemod>
#include <tf2_stocks>

#pragma semicolon 1
#pragma newdecls required

public Plugin myinfo =
{
	name = "MvM Defender Bots: test-bed host",
	author = "m-this",
	description = "Holds the seats a run puts on RED: the host that readies up, and the puppets that stand in for players",
	version = "1.3.0",
	url = "https://github.com/m-this/tf2-mvm-bots"
};

//Long enough after a map change that the game accepts a join, short enough not to waste a run
#define HOST_JOIN_DELAY		10.0

//How often to check that the host is still there and still ready
#define HOST_WATCH_INTERVAL	5.0

/* How many puppets a run may seat
 *
 * RED is six seats and the host already holds one, so four is past anything a
 * run can ask for and still have defenders to measure against. The bound is
 * here because every loop needs one, not because four is a target. */
#define MAX_PUPPETS		4

/* The gap between the two ready presses that start the bots
 *
 * The mod asks for the button twice: the first press answers "Press ready
 * again to start the bots" and does nothing else, and the second one, inside
 * three seconds, is what actually spawns them. It also rate limits a client's
 * commands to one every 0.3 seconds, so the gap has to clear that and stay
 * well inside the three. */
#define HOST_READY_GAP		1.0

/* How long to wait for the station's trigger to notice the puppet
 *
 * TeleportEntity touches the triggers it lands in and a fake client is
 * simulated like any other player, so m_bInUpgradeZone is usually set on the
 * frame of the teleport. The poll is here because "usually" is not something to
 * buy upgrades on: an odd brush, or a station this landed outside of, has to be
 * refused by name rather than reported as in_zone=0 next to a purchase the game
 * threw away in silence. Two seconds is past any landing that was going to
 * work. */
#define SHOP_TOUCH_INTERVAL	0.1
#define SHOP_TOUCH_TRIES	20

/* The most steps of one upgrade a single call may buy
 *
 * The game takes a count, applies what it can and refuses the rest, so a big
 * number is not dangerous, only unanswerable: nothing in the reply says which
 * of ten thousand steps were taken. No attribute in mvm_upgrades.txt has more
 * tiers than this. */
#define SHOP_COUNT_MAX		10

//The itemslot range the game's own menu sends: -1 for the player, 0 to 5 for a weapon
#define SHOP_SLOT_MIN		-1
#define SHOP_SLOT_MAX		5

//More func_upgradestation than any map ships, so the walk over them has a bound
#define SHOP_STATIONS_MAX	16

//A player's top run speed. A held speed past it is clamped by the game anyway, so it is refused here
#define INPUT_SPEED_MAX		450.0

//The button names mvmbots_puppet_input takes, a comma list with no more entries than there are names
#define INPUT_NAMES_MAX		8

ConVar g_cvEnabled;
ConVar g_cvName;
ConVar g_cvPuppets;
ConVar g_cvPuppetName;
ConVar g_cvPuppetClass;
ConVar g_cvReadyDelay;

int g_iHost = -1;
int g_arrPuppets[MAX_PUPPETS];

/* A trip to the upgrade station, while one is in flight
 *
 * Per puppet, because the command is per puppet and so is the timer that
 * finishes it. home is where the puppet stood before it was moved: one left
 * standing in the station is one out of the medic's range for the rest of the
 * wave, which quietly spoils every other measurement the run was making. */
enum struct ShopTrip
{
	bool busy;
	int slot;
	int upgrade;
	int count;
	int stations;
	int tries;
	float home[3];
}

ShopTrip g_arrTrips[MAX_PUPPETS];

/* What a run is holding down on each puppet
 *
 * Applied every tick in OnPlayerRunCmd, which is the usercmd a real client
 * sends: the buttons, the walk speed on two axes, and where it looks. Held
 * rather than pulsed, because a fake client sends nothing of its own and a
 * command that only lasted one tick would be over before rcon answered. A run
 * lets go with mvmbots_puppet_input <n> none 0 0. */
enum struct PuppetInput
{
	int buttons;
	float walk;
	float strafe;
	bool steering;
	float angles[3];

	//A loadout slot to select on the next tick, then -1: a weaponselect is sent once, not held
	int select;
}

PuppetInput g_arrInputs[MAX_PUPPETS];

//When the last round ended, in game time. The ready delay counts from here.
float g_flRoundOver;

public void OnPluginStart()
{
	g_cvEnabled = CreateConVar("mvmbots_host_enabled", "1",
		"Connect one fake client to hold a seat on RED and ready up.", _, true, 0.0, true, 1.0);

	RegServerCmd("mvmbots_roster", Command_Roster, "Say who is on each team, for a run to check before it believes itself.");
	g_cvName = CreateConVar("mvmbots_host_name", "testbed-host",
		"What to call it, so it can be told apart from the mod's own bots.");

	g_cvPuppets = CreateConVar("mvmbots_puppet_count", "0",
		"How many puppets to seat on RED. A puppet stands where a player stands and does what a run tells it.",
		_, true, 0.0, true, float(MAX_PUPPETS));

	/* A puppet is only a player to the mod once the nextbot test is on, so
	   the switch is set here rather than left to whoever wrote the run.
	   A run that seats a puppet and forgets it measures the mod ignoring a
	   fake client, which is what it already did before there were puppets */
	g_cvPuppets.AddChangeHook(OnPuppetCountChanged);

	g_cvPuppetName = CreateConVar("mvmbots_puppet_name", "testbed-player",
		"What to call them. The index is appended, so the results file names each one.");

	/* Scout, because the medic ranking is on maximum health

	A Heavy puppet wins BiggestBody on its body alone, so a beam on it says
	nothing about whether the call was answered. A Scout is the smallest body
	on the team and only takes the beam by being a player or by calling, which
	is the two halves of mvm-w9b and nothing else. */
	g_cvPuppetClass = CreateConVar("mvmbots_puppet_class", "scout",
		"The class they join as. Scout by default: the smallest body, so it only takes the medic beam for being a player.");

	/* A team that just lost sits in the ready-up before it readies again, and
	   that is where a player saves a new lineup. The host readying at once
	   left no such window here, so a fault that lives in it could not be
	   played: mvm-tcc. */
	g_cvReadyDelay = CreateConVar("mvmbots_host_ready_delay", "0",
		"Seconds the host and the puppets wait after a round ends before they ready up again. Nought readies at once.",
		_, true, 0.0, true, 600.0);

	RegServerCmd("mvmbots_puppet_call", Command_PuppetCall,
		"Press MEDIC!, as a player's key does. Takes a puppet index, or nothing for all of them.");
	RegServerCmd("mvmbots_puppet_status", Command_PuppetStatus,
		"Say what each puppet is doing and who is healing it, for a run to read without the results file.");
	RegServerCmd("mvmbots_puppet_shop", Command_PuppetShop,
		"Walk a puppet into the upgrade station, buy, and walk it back. Args: puppet [itemslot upgrade count].");
	RegServerCmd("mvmbots_puppet_input", Command_PuppetInput,
		"Hold buttons and a walk speed on a puppet every tick. Args: puppet buttons|none forward side.");
	RegServerCmd("mvmbots_puppet_look", Command_PuppetLook,
		"Point a puppet's view and keep it there, or let go. Args: puppet pitch yaw, or puppet free.");
	RegServerCmd("mvmbots_puppet_cmd", Command_PuppetCmd,
		"Run a client command as a puppet, the way its console would. Args: puppet command...");
	RegServerCmd("mvmbots_puppet_slot", Command_PuppetSlot,
		"Switch a puppet to the weapon in a loadout slot, 0 primary to 5. Args: puppet slot.");
	RegServerCmd("mvmbots_puppet_teleport", Command_PuppetTeleport,
		"Put a puppet at a point. Args: puppet x y z.");
	RegServerCmd("mvmbots_puppet_station", Command_PuppetStation,
		"Say where the upgrade station nearest a puppet is. Args: puppet.");

	//Zero is a slot, so a fresh array would select the primary on the first tick
	for (int n = 0; n < MAX_PUPPETS; n++)
		ReleaseInput(n);

	CreateTimer(HOST_WATCH_INTERVAL, Timer_WatchHost, _, TIMER_REPEAT | TIMER_FLAG_NO_MAPCHANGE);

	HookEvent("mvm_wave_failed", Event_RoundOver, EventHookMode_Post);
	HookEvent("teamplay_round_start", Event_RoundOver, EventHookMode_Post);
}

/* A lost wave, or a restart, leaves the host readied and the bots gone.
 *
 * The game kicks the defenders when it restarts a wave, and in READY_BOTS mode
 * the mod adds them again only on a ready press. The host still counted as
 * ready, so the watch above never pressed, the restarted wave began with RED
 * empty, and the next attempt refused with "RED reached 0 defenders". Taking
 * the ready off is what makes the watch press it again. */
static void Event_RoundOver(Event event, const char[] name, bool dontBroadcast)
{
	g_flRoundOver = GetGameTime();

	DropReady(g_iHost);

	//The puppets sit in the same seats and go stale the same way
	for (int n = 0; n < MAX_PUPPETS; n++)
		DropReady(g_arrPuppets[n]);
}

static void DropReady(int client)
{
	if (client > 0 && client <= MaxClients && IsClientInGame(client))
		FakeClientCommand(client, "tournament_player_readystate 0");
}

public void OnMapStart()
{
	g_iHost = -1;

	for (int n = 0; n < MAX_PUPPETS; n++)
	{
		g_arrPuppets[n] = -1;

		//A map change kills the trip's timer, and a trip left busy refuses every later one
		g_arrTrips[n].busy = false;

		ReleaseInput(n);
	}

	CreateTimer(HOST_JOIN_DELAY, Timer_Connect, _, TIMER_FLAG_NO_MAPCHANGE);
}

static Action Timer_Connect(Handle timer)
{
	ConnectHost();
	ConnectPuppets();

	return Plugin_Stop;
}

/* Say so when a run asks for puppets without the switch that makes them players
 *
 * Refusing is not this plugin's call: a run may want a body on RED for some
 * other reason. Silence is, though, because a puppet the mod reads as a bot
 * produces a full results file that answers a different question */
static void OnPuppetCountChanged(ConVar convar, const char[] before, const char[] after)
{
	if (StringToInt(after) < 1)
		return;

	ConVar test = FindConVar("sm_redbots_feature_bot_test_by_nextbot");

	if (test == null)
		LogMessage("mvmbots_host: the mod has no bot_test_by_nextbot switch, so a puppet is a bot to it");
	else if (!test.BoolValue)
		LogMessage("mvmbots_host: sm_redbots_feature_bot_test_by_nextbot is off, so a puppet is a bot to the mod");
}

/* Keep them connected and keep them ready
 *
 * Ready state is cleared between waves, and a fake client can be dropped for
 * reasons nothing here controls. Both are cheap to check and neither is worth
 * an event hook */
static Action Timer_WatchHost(Handle timer)
{
	WatchPuppets();

	if (!g_cvEnabled.BoolValue)
		return Plugin_Continue;

	if (!IsHostConnected())
	{
		ConnectHost();

		return Plugin_Continue;
	}

	if (ReadyIsDue() && !IsPlayerReady(g_iHost))
		PressReady(g_iHost);

	return Plugin_Continue;
}

//Between rounds, and the delay a run asked for has passed since the last one ended
static bool ReadyIsDue()
{
	if (GameRules_GetRoundState() != RoundState_BetweenRounds)
		return false;

	return GetGameTime() - g_flRoundOver >= g_cvReadyDelay.FloatValue;
}

/* Press ready, and press it again a second later
 *
 * Both presses are needed and they do different things. Before the bots exist
 * the pair is what starts them, and the mod swallows each press rather than
 * passing it to the game. Once they are running the mod passes a press
 * straight through, and then it is the ready that starts the wave. Sending two
 * covers both without having to know which state the server is in */
static void PressReady(int client)
{
	FakeClientCommand(client, "tournament_player_readystate 1");

	CreateTimer(HOST_READY_GAP, Timer_ReadyAgain, GetClientUserId(client), TIMER_FLAG_NO_MAPCHANGE);
}

static Action Timer_ReadyAgain(Handle timer, int userid)
{
	int client = GetClientOfUserId(userid);

	if (client > 0 && IsClientInGame(client))
		FakeClientCommand(client, "tournament_player_readystate 1");

	return Plugin_Stop;
}

static bool IsHostConnected()
{
	char wanted[MAX_NAME_LENGTH]; g_cvName.GetString(wanted, sizeof(wanted));

	g_iHost = SeatNamed(g_iHost, wanted);

	return g_iHost > 0;
}

/* The seat that answers to this name, and -1 when there is none
 *
 * Adopting one that is already here rather than connecting a second. The index
 * is lost whenever this plugin is reloaded and the client it named is not:
 * without this, a reload leaves "testbed-host" standing next to
 * "(1)testbed-host", and every reload after that adds another */
static int SeatNamed(int known, const char[] wanted)
{
	if (known > 0 && known <= MaxClients && IsClientInGame(known))
		return known;

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || !IsFakeClient(i))
			continue;

		char name[MAX_NAME_LENGTH]; GetClientName(i, name, sizeof(name));

		if (StrEqual(name, wanted))
			return i;
	}

	return -1;
}

static void ConnectHost()
{
	if (!g_cvEnabled.BoolValue || IsHostConnected())
		return;

	char name[MAX_NAME_LENGTH]; g_cvName.GetString(name, sizeof(name));

	int client = CreateFakeClient(name);

	if (client == 0)
	{
		LogError("mvmbots_host: the server refused a fake client, so no wave will start");
		return;
	}

	g_iHost = client;

	/* Medic, because that is the one class the mod's own medic refuses to heal
	
	The seat holder was a Scout, and a full health Scout standing in the RED
	spawn is the best patient the mod's medic can see: PreferredPatient ranks a
	teammate within twelve hundred units above one anywhere else, and the medic
	spawns in the same room. So he latched onto the statue on the first frame of
	every wave and never left, and the trace says so plainly, "beam on
	testbed-host" for twenty five samples running.
	
	That is not a small measurement error. Four medic experiments were run
	against it and all four lost: medic_nearest, medic_leaves_spawn twice, and
	taking away the walk. Each of them was really being asked whether it could
	beat pocketing an immobile fake client, which is a question with no useful
	answer. They were deleted on the strength of it.
	
	PreferredPatient skips medics outright, so a medic in the seat is invisible
	to the thing under test. He still holds the seat, still readies up, and
	still does nothing, which is the whole of the job. */
	ChangeClientTeam(client, view_as<int>(TFTeam_Red));
	FakeClientCommand(client, "joinclass medic");

	//The ready is what the mod's listener is waiting for, and what starts the wave
	PressReady(client);

	LogMessage("mvmbots_host: %N is holding a seat on RED", client);
}

static bool IsPlayerReady(int client)
{
	return GameRules_GetProp("m_bPlayerReady", 4, client) != 0;
}

/* The puppets: bodies a run drives, standing where a player stands
 *
 * Seated the same way the host is and kept the same way, because the failure
 * modes are the same ones: a fake client can be dropped, and a plugin reload
 * loses the index but not the client.
 *
 * They ready up like the host. A puppet is a body on RED and MvM counts it, so
 * one that never presses ready is one more seat the wave waits on. */
static void ConnectPuppets()
{
	int wanted = g_cvPuppets.IntValue;

	for (int n = 0; n < wanted; n++)
		ConnectPuppet(n);
}

static void WatchPuppets()
{
	int wanted = g_cvPuppets.IntValue;

	for (int n = 0; n < MAX_PUPPETS; n++)
	{
		/* A count a run lowered mid-session leaves the extra ones seated.
		   Kicking them would be a second way for a seat to disappear and the
		   watchdog reads an empty RED as a fault, so they stay until the map
		   changes and mvmbots_roster keeps naming them */
		if (n >= wanted)
			continue;

		if (!IsPuppetConnected(n))
		{
			ConnectPuppet(n);

			continue;
		}

		if (ReadyIsDue() && !IsPlayerReady(g_arrPuppets[n]))
			PressReady(g_arrPuppets[n]);
	}
}

static bool IsPuppetConnected(int n)
{
	char wanted[MAX_NAME_LENGTH]; PuppetName(n, wanted, sizeof(wanted));

	g_arrPuppets[n] = SeatNamed(g_arrPuppets[n], wanted);

	return g_arrPuppets[n] > 0;
}

static void ConnectPuppet(int n)
{
	if (IsPuppetConnected(n))
		return;

	char name[MAX_NAME_LENGTH]; PuppetName(n, name, sizeof(name));

	int client = CreateFakeClient(name);

	if (client == 0)
	{
		LogError("mvmbots_host: the server refused a fake client, so puppet %d is not on RED", n + 1);
		return;
	}

	g_arrPuppets[n] = client;

	//A new body in the seat does not inherit what the last one was told to hold
	ReleaseInput(n);

	char class[32]; g_cvPuppetClass.GetString(class, sizeof(class));

	ChangeClientTeam(client, view_as<int>(TFTeam_Red));
	FakeClientCommand(client, "joinclass %s", class);

	PressReady(client);

	LogMessage("mvmbots_host: %N is standing in for a player on RED as a %s", client, class);
}

//The index is in the name, so a results file naming a patient says which puppet it was
static void PuppetName(int n, char[] buffer, int length)
{
	char base[MAX_NAME_LENGTH]; g_cvPuppetName.GetString(base, sizeof(base));

	FormatEx(buffer, length, "%s-%d", base, n + 1);
}

static bool IsPuppet(int client)
{
	return PuppetIndex(client) != -1;
}

static int PuppetIndex(int client)
{
	for (int n = 0; n < MAX_PUPPETS; n++)
	{
		if (g_arrPuppets[n] == client)
			return n;
	}

	return -1;
}

/* Press MEDIC!, the way a player's key does
 *
 * voicemenu 0 0 and not an event: the game fires no player_calls_for_medic, so
 * the mod listens for the command, and a FakeClientCommand reaches that
 * listener by the same route a person's keypress does. Nothing here knows what
 * the mod does with it, which is the point. */
static Action Command_PuppetCall(int args)
{
	int only = -1;

	if (args >= 1)
	{
		char arg[8]; GetCmdArg(1, arg, sizeof(arg));
		only = StringToInt(arg);

		if (only < 1 || only > MAX_PUPPETS)
		{
			PrintToServer("mvmbots_puppet_call: %d is not a puppet, they are 1 to %d", only, MAX_PUPPETS);

			return Plugin_Handled;
		}
	}

	int called;

	for (int n = 0; n < MAX_PUPPETS; n++)
	{
		if (only > 0 && n != only - 1)
			continue;

		if (!IsPuppetConnected(n) || !IsPlayerAlive(g_arrPuppets[n]))
			continue;

		FakeClientCommand(g_arrPuppets[n], "voicemenu 0 0");
		called++;
	}

	PrintToServer("mvmbots_puppet_call called=%d", called);

	return Plugin_Handled;
}

/* Buy one upgrade, the way a player's client does
 *
 * The client sends MvM_UpgradesBegin when its menu opens, one MVM_Upgrade per
 * click and MvM_UpgradesDone when it closes, and the game only takes the middle
 * one from a player standing in a station. So the puppet is walked into the
 * nearest station, the three go once the trigger has it, and it is put back
 * where it stood.
 *
 * Nothing the game sends back says whether the purchase took. The credits
 * moving is the only proof there is, so that is what this checks, the way the
 * mod's own shopping action does.
 *
 * A refusal on SourceMod at or before git7253 is not this command: the
 * 2026-10-02 game update changed the KeyValues layout, those builds write the
 * old one, and every MVM_Upgrade is thrown away silently. git7255 reads the new
 * layout and buys. Check the SourceMod build before changing anything here. */
static Action Command_PuppetShop(int args)
{
	if (args != 1 && args != 4)
	{
		PrintToServer("mvmbots_puppet_shop: puppet [itemslot upgrade count]");

		return Plugin_Handled;
	}

	int only;

	if (!ArgInt(1, only) || only < 1 || only > MAX_PUPPETS)
	{
		PrintToServer("mvmbots_puppet_shop: argument 1 is a puppet, they are 1 to %d", MAX_PUPPETS);

		return Plugin_Handled;
	}

	int n = only - 1;

	if (!IsPuppetConnected(n) || !IsPlayerAlive(g_arrPuppets[n]))
	{
		PrintToServer("mvmbots_puppet_shop: puppet %d is not seated and alive", only);

		return Plugin_Handled;
	}

	//One trip at a time, or the second overwrites the spot the first has to put it back in
	if (g_arrTrips[n].busy)
	{
		PrintToServer("mvmbots_puppet_shop: puppet %d is still at the station", only);

		return Plugin_Handled;
	}

	int slot = -1, upgrade = -1, count = 0;

	if (args == 4 && !ShopArgs(slot, upgrade, count))
		return Plugin_Handled;

	int client = g_arrPuppets[n];
	int stations;
	int station = NearestStation(client, stations);

	if (station == -1)
	{
		PrintToServer("mvmbots_puppet_shop: this map has no func_upgradestation a puppet could stand in");

		return Plugin_Handled;
	}

	/* Said rather than refused: shopping during a wave is a thing a player
	   does, and it is also the puppet leaving the fight for a second, which
	   is worth having in the log beside whatever the run measured */
	if (GameRules_GetRoundState() == RoundState_RoundRunning)
		LogMessage("mvmbots_puppet_shop: the wave is running, so %N leaves it for the length of the trip", client);

	g_arrTrips[n].busy = true;
	g_arrTrips[n].slot = slot;
	g_arrTrips[n].upgrade = upgrade;
	g_arrTrips[n].count = count;
	g_arrTrips[n].stations = stations;
	g_arrTrips[n].tries = 0;
	GetClientAbsOrigin(client, g_arrTrips[n].home);

	float centre[3]; WorldSpaceCentre(station, centre);
	TeleportEntity(client, centre, NULL_VECTOR, {0.0, 0.0, 0.0});

	CreateTimer(SHOP_TOUCH_INTERVAL, Timer_PuppetShop, n, TIMER_REPEAT | TIMER_FLAG_NO_MAPCHANGE);

	return Plugin_Handled;
}

/* The three numbers an upgrade takes, refused here rather than passed on
 *
 * A bad itemslot or a bad row is answered by the game doing nothing, so a typo
 * and a working command read the same in the log. count is bounded because
 * nothing on the way in bounds it. */
static bool ShopArgs(int &slot, int &upgrade, int &count)
{
	if (!ArgInt(2, slot) || slot < SHOP_SLOT_MIN || slot > SHOP_SLOT_MAX)
	{
		PrintToServer("mvmbots_puppet_shop: itemslot is %d to %d, -1 being the player's own upgrades",
			SHOP_SLOT_MIN, SHOP_SLOT_MAX);

		return false;
	}

	if (!ArgInt(3, upgrade) || upgrade < 0)
	{
		PrintToServer("mvmbots_puppet_shop: upgrade is a row of mvm_upgrades.txt, counted from nought");

		return false;
	}

	if (!ArgInt(4, count) || count < 1 || count > SHOP_COUNT_MAX)
	{
		PrintToServer("mvmbots_puppet_shop: count is 1 to %d", SHOP_COUNT_MAX);

		return false;
	}

	return true;
}

//The whole argument or nothing: StringToInt reads 0 out of a word, and 0 is a row and a slot
static bool ArgInt(int argnum, int &value)
{
	char arg[16]; int length = GetCmdArg(argnum, arg, sizeof(arg));

	return length > 0 && StringToIntEx(arg, value) == length;
}

/* The station nearest the puppet, and how many this map has
 *
 * Nearest, because a map with several puts them in different spawn rooms and
 * the far one is a teleport through a wall. The count is reported rather than
 * acted on: when the trip fails it is the first thing somebody needs to know.
 *
 * A station the map has switched off is skipped where the field is there to
 * read. The mod reads a CUpgrades field of its own through a hand-counted
 * offset; this does not, because an offset in a test-bed plugin rots without
 * saying so, and the touch check in the timer is what actually proves the
 * station took the puppet. */
static int NearestStation(int client, int &count)
{
	float here[3]; GetClientAbsOrigin(client, here);

	int nearest = -1;
	float nearestDistance = 0.0;
	int station = -1;

	count = 0;

	for (int seen = 0; seen < SHOP_STATIONS_MAX; seen++)
	{
		station = FindEntityByClassname(station, "func_upgradestation");

		if (station == -1)
			break;

		count++;

		if (HasEntProp(station, Prop_Data, "m_bDisabled") && GetEntProp(station, Prop_Data, "m_bDisabled") != 0)
			continue;

		float centre[3]; WorldSpaceCentre(station, centre);
		float distance = GetVectorDistance(here, centre, true);

		if (nearest == -1 || distance < nearestDistance)
		{
			nearest = station;
			nearestDistance = distance;
		}
	}

	return nearest;
}

/* The middle of an entity's box, in world coordinates
 *
 * m_vecMins and m_vecMaxs are the entity's own box and not world coordinates,
 * even on a brush, so the middle is the origin plus half of them. The same
 * arithmetic as the mod's WorldSpaceCenter, and the reason the z is not nudged:
 * a trigger brush reaches below the floor it sits on as often as not, so feet
 * placed just above its bottom are feet inside the floor. Dropped in at the
 * middle the hull is inside the brush on the frame it arrives, which is all the
 * touch needs, and gravity does the rest. */
static void WorldSpaceCentre(int entity, float buffer[3])
{
	float origin[3], mins[3], maxs[3];
	GetEntPropVector(entity, Prop_Data, "m_vecAbsOrigin", origin);
	GetEntPropVector(entity, Prop_Data, "m_vecMins", mins);
	GetEntPropVector(entity, Prop_Data, "m_vecMaxs", maxs);

	for (int axis = 0; axis < 3; axis++)
		buffer[axis] = origin[axis] + (mins[axis] + maxs[axis]) * 0.5;
}

/* Finish the trip once the station's trigger has the puppet
 *
 * m_bInUpgradeZone is the whole precondition and the first version of this only
 * logged it. A purchase sent without it is dropped by the game without a word,
 * which reads in the results exactly like the mod refusing one. Failing by name
 * is the point of the poll. */
static Action Timer_PuppetShop(Handle timer, int n)
{
	if (!IsPuppetConnected(n) || !IsPlayerAlive(g_arrPuppets[n]))
	{
		LogError("mvmbots_puppet_shop: puppet %d left or died at the station, so it is not going back", n + 1);

		g_arrTrips[n].busy = false;

		return Plugin_Stop;
	}

	int client = g_arrPuppets[n];

	g_arrTrips[n].tries++;

	bool inZone = GetEntProp(client, Prop_Send, "m_bInUpgradeZone") != 0;

	if (!inZone && g_arrTrips[n].tries < SHOP_TOUCH_TRIES)
		return Plugin_Continue;

	int before = GetEntProp(client, Prop_Send, "m_nCurrency");
	int spent = inZone ? Shop(client, g_arrTrips[n].slot, g_arrTrips[n].upgrade, g_arrTrips[n].count) : 0;

	PrintToServer("mvmbots_puppet_shop name=%N stations=%d tries=%d in_zone=%d currency=%d->%d",
		client, g_arrTrips[n].stations, g_arrTrips[n].tries, inZone ? 1 : 0, before, before - spent);

	if (!inZone)
		LogError("mvmbots_puppet_shop: %N never touched a station in %d tries, of %d on this map, so nothing was bought",
			client, g_arrTrips[n].tries, g_arrTrips[n].stations);
	else if (g_arrTrips[n].upgrade >= 0 && spent < 1)
		LogError("mvmbots_puppet_shop: the game turned down row %d in slot %d for %N, who holds %d credits",
			g_arrTrips[n].upgrade, g_arrTrips[n].slot, client, before);

	TeleportEntity(client, g_arrTrips[n].home, NULL_VECTOR, {0.0, 0.0, 0.0});

	g_arrTrips[n].busy = false;

	return Plugin_Stop;
}

/* The key values the client sends, and the credits they cost
 *
 * Begin and Done are a pair a client never leaves half open: the server holds a
 * per player flag from the one to the other. The mod's own shopping action
 * sends Done the same way and threw "client is not connected" out of it once,
 * mvm-9sw, so the caller checks the seat first and all three go in this frame.
 *
 * No Rewind between the subkey and the send, matching the mod: the native sends
 * the KeyValues it was handed, not wherever the cursor was left, so a rewind
 * only looks like it is doing something.
 *
 * num_upgrades is read for the announcement and nothing else, and how many
 * steps the game applied is not knowable from here, so a purchase that moved
 * the credits counts as one. */
static int Shop(int client, int slot, int upgrade, int count)
{
	KeyValues begin = new KeyValues("MvM_UpgradesBegin");
	FakeClientCommandKeyValues(client, begin);
	delete begin;

	int before = GetEntProp(client, Prop_Send, "m_nCurrency");

	if (upgrade >= 0)
	{
		KeyValues kv = new KeyValues("MVM_Upgrade");
		kv.JumpToKey("upgrade", true);
		kv.SetNum("itemslot", slot);
		kv.SetNum("upgrade", upgrade);
		kv.SetNum("count", count);
		FakeClientCommandKeyValues(client, kv);
		delete kv;
	}

	int spent = before - GetEntProp(client, Prop_Send, "m_nCurrency");

	KeyValues done = new KeyValues("MvM_UpgradesDone");
	done.SetNum("num_upgrades", spent > 0 ? 1 : 0);
	FakeClientCommandKeyValues(client, done);
	delete done;

	return spent;
}

/* What each puppet is and who is healing it, in lines a run can read
 *
 * The results file already carries the answer, in the healing field of every
 * medic sample, but that arrives at the end of an attempt. This says it while
 * the wave is running, which is the difference between watching a call being
 * answered and reading about it half an hour later. */
static Action Command_PuppetStatus(int args)
{
	for (int n = 0; n < MAX_PUPPETS; n++)
	{
		if (!IsPuppetConnected(n))
			continue;

		int client = g_arrPuppets[n];
		bool alive = IsPlayerAlive(client);

		char healer[MAX_NAME_LENGTH] = "";

		if (alive)
			HealerOf(client, healer, sizeof(healer));

		float origin[3]; GetClientAbsOrigin(client, origin);
		float eyes[3]; GetClientEyeAngles(client, eyes);
		char weapon[48] = "none";
		int clip = -1;
		int active = alive ? GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon") : -1;

		if (active > MaxClients && IsValidEntity(active))
		{
			GetEntityClassname(active, weapon, sizeof(weapon));
			clip = GetEntProp(active, Prop_Send, "m_iClip1");
		}

		int held = g_arrInputs[n].buttons;
		float walk = g_arrInputs[n].walk, strafe = g_arrInputs[n].strafe;

		/* healer stays where it was in the line, empty or not, so the fields
		   after it are named and a reader never counts its way to them */
		PrintToServer("mvmbots_puppet %d name=%N class=%s alive=%d hp=%d healer=%s pos=%.0f,%.0f,%.0f pitch=%.0f yaw=%.0f currency=%d in_zone=%d buttons=%d forward=%.0f side=%.0f weapon=%s clip=%d",
			n + 1, client, ClassNameOf(client), alive ? 1 : 0,
			alive ? GetClientHealth(client) : 0, healer,
			origin[0], origin[1], origin[2], eyes[0], eyes[1],
			GetEntProp(client, Prop_Send, "m_nCurrency"),
			GetEntProp(client, Prop_Send, "m_bInUpgradeZone"),
			held, walk, strafe, weapon, clip);
	}

	return Plugin_Handled;
}

/* Who has a medigun on this one, by name
 *
 * Asked of the healers rather than of the patient: m_hHealingTarget is on the
 * medigun, and there is no property on a player saying who chose him. One pass
 * over the seats is cheap and this runs when a run asks, not per frame. */
static void HealerOf(int patient, char[] buffer, int length)
{
	buffer[0] = '\0';

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || !IsPlayerAlive(i) || TF2_GetPlayerClass(i) != TFClass_Medic)
			continue;

		int medigun = GetPlayerWeaponSlot(i, 1);

		if (medigun == -1 || !HasEntProp(medigun, Prop_Send, "m_hHealingTarget"))
			continue;

		if (GetEntPropEnt(medigun, Prop_Send, "m_hHealingTarget") != patient)
			continue;

		GetClientName(i, buffer, length);

		return;
	}
}

static char[] ClassNameOf(int client)
{
	char name[16] = "none";

	switch (TF2_GetPlayerClass(client))
	{
		case TFClass_Scout: name = "scout";
		case TFClass_Soldier: name = "soldier";
		case TFClass_Pyro: name = "pyro";
		case TFClass_DemoMan: name = "demoman";
		case TFClass_Heavy: name = "heavy";
		case TFClass_Engineer: name = "engineer";
		case TFClass_Medic: name = "medic";
		case TFClass_Sniper: name = "sniper";
		case TFClass_Spy: name = "spy";
	}

	return name;
}


/* Say who is on each team, in one line a runner can read

status cannot do this. It lists names and never says which side anybody is on,
and the robots are named by class on most maps: a runner counting "not a robot
name" as a defender read fourteen Pyros on BLU as a full RED and passed. The
game knows the answer and this asks it.
*/
static Action Command_Roster(int args)
{
	int red, blu, humans, host, puppets;

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i))
			continue;

		if (!IsFakeClient(i))
			humans++;
		else if (IsPuppet(i))
			puppets++;
		else if (i == g_iHost)
			host++;

		switch (TF2_GetClientTeam(i))
		{
			case TFTeam_Red: red++;
			case TFTeam_Blue: blu++;
		}
	}

	//The host and the puppets hold RED seats and neither is a defender, so both are named separately
	PrintToServer("mvmbots_roster red=%d blu=%d humans=%d host=%d puppets=%d", red, blu, humans, host, puppets);

	return Plugin_Handled;
}

/* The usercmd a run is holding, put on the puppet every tick
 *
 * A fake client's command is empty unless something fills it, and this is the
 * one place that does. The movement bits go with the speeds because the
 * animation and a few game checks read the buttons, not the velocity.
 *
 * The angles go in the command and through TeleportEntity both: the command is
 * what the game simulates the tick with, and a fake client's eye angles are not
 * always taken from it, which is the difference between aiming and looking. */
public Action OnPlayerRunCmd(int client, int &buttons, int &impulse, float vel[3], float angles[3], int &weapon, int &subtype, int &cmdnum, int &tickcount, int &seed, int mouse[2])
{
	int n = PuppetIndex(client);

	if (n == -1 || !IsClientInGame(client) || !IsPlayerAlive(client))
		return Plugin_Continue;

	buttons |= g_arrInputs[n].buttons;
	vel[0] = g_arrInputs[n].walk;
	vel[1] = g_arrInputs[n].strafe;

	if (vel[0] > 0.0)
		buttons |= IN_FORWARD;
	else if (vel[0] < 0.0)
		buttons |= IN_BACK;

	if (vel[1] > 0.0)
		buttons |= IN_MOVERIGHT;
	else if (vel[1] < 0.0)
		buttons |= IN_MOVELEFT;

	/* slot1 to slot6 never reach the server: the client's HUD turns them into
	   the weapon field of its next usercmd, and this is that field */
	if (g_arrInputs[n].select != -1)
	{
		int chosen = GetPlayerWeaponSlot(client, g_arrInputs[n].select);

		if (chosen != -1)
			weapon = chosen;

		g_arrInputs[n].select = -1;
	}

	if (g_arrInputs[n].steering)
	{
		for (int axis = 0; axis < 3; axis++)
			angles[axis] = g_arrInputs[n].angles[axis];

		TeleportEntity(client, NULL_VECTOR, angles, NULL_VECTOR);
	}

	return Plugin_Changed;
}

static void ReleaseInput(int n)
{
	g_arrInputs[n].buttons = 0;
	g_arrInputs[n].walk = 0.0;
	g_arrInputs[n].strafe = 0.0;
	g_arrInputs[n].steering = false;
	g_arrInputs[n].select = -1;
}

/* The puppet named by argument 1, seated, or -1 having said why not
 *
 * Alive is not asked: holding input on a dead puppet is how a run says what it
 * does the moment it respawns. */
static int PuppetArg(const char[] command)
{
	int only;

	if (!ArgInt(1, only) || only < 1 || only > MAX_PUPPETS)
	{
		PrintToServer("%s: argument 1 is a puppet, they are 1 to %d", command, MAX_PUPPETS);

		return -1;
	}

	if (!IsPuppetConnected(only - 1))
	{
		PrintToServer("%s: puppet %d is not seated", command, only);

		return -1;
	}

	return only - 1;
}

static bool ArgFloat(int argnum, float &value)
{
	char arg[32]; int length = GetCmdArg(argnum, arg, sizeof(arg));

	return length > 0 && StringToFloatEx(arg, value) == length;
}

/* Hold buttons and a walk speed until told otherwise
 *
 * Buttons by name rather than by bit, so a script reads as what it does and a
 * typo is refused instead of pressing whatever bit it happened to spell. */
static Action Command_PuppetInput(int args)
{
	if (args != 4)
	{
		PrintToServer("mvmbots_puppet_input: puppet buttons|none forward side, buttons a comma list of attack,attack2,attack3,jump,duck,use,reload");

		return Plugin_Handled;
	}

	int n = PuppetArg("mvmbots_puppet_input");

	if (n == -1)
		return Plugin_Handled;

	char list[96]; GetCmdArg(2, list, sizeof(list));
	int buttons;

	if (!ParseButtons(list, buttons))
	{
		PrintToServer("mvmbots_puppet_input: %s is not a list of attack,attack2,attack3,jump,duck,use,reload, or none", list);

		return Plugin_Handled;
	}

	float walk, strafe;

	if (!ArgFloat(3, walk) || !ArgFloat(4, strafe)
		|| FloatAbs(walk) > INPUT_SPEED_MAX || FloatAbs(strafe) > INPUT_SPEED_MAX)
	{
		PrintToServer("mvmbots_puppet_input: forward and side are speeds from -%.0f to %.0f", INPUT_SPEED_MAX, INPUT_SPEED_MAX);

		return Plugin_Handled;
	}

	g_arrInputs[n].buttons = buttons;
	g_arrInputs[n].walk = walk;
	g_arrInputs[n].strafe = strafe;

	PrintToServer("mvmbots_puppet_input puppet=%d buttons=%d forward=%.0f side=%.0f", n + 1, buttons, walk, strafe);

	return Plugin_Handled;
}

static bool ParseButtons(const char[] list, int &buttons)
{
	buttons = 0;

	if (StrEqual(list, "none"))
		return true;

	char names[INPUT_NAMES_MAX][16];
	int count = ExplodeString(list, ",", names, sizeof(names), sizeof(names[]));

	for (int i = 0; i < count; i++)
	{
		int bit = ButtonNamed(names[i]);

		if (bit == 0)
			return false;

		buttons |= bit;
	}

	return count > 0;
}

static int ButtonNamed(const char[] name)
{
	if (StrEqual(name, "attack"))
		return IN_ATTACK;
	if (StrEqual(name, "attack2"))
		return IN_ATTACK2;
	if (StrEqual(name, "attack3"))
		return IN_ATTACK3;
	if (StrEqual(name, "jump"))
		return IN_JUMP;
	if (StrEqual(name, "duck"))
		return IN_DUCK;
	if (StrEqual(name, "use"))
		return IN_USE;
	if (StrEqual(name, "reload"))
		return IN_RELOAD;

	return 0;
}

static Action Command_PuppetLook(int args)
{
	int n = args >= 1 ? PuppetArg("mvmbots_puppet_look") : -1;

	if (n == -1)
	{
		if (args < 1)
			PrintToServer("mvmbots_puppet_look: puppet pitch yaw, or puppet free");

		return Plugin_Handled;
	}

	char arg[16]; GetCmdArg(2, arg, sizeof(arg));

	if (args == 2 && StrEqual(arg, "free"))
	{
		g_arrInputs[n].steering = false;
		PrintToServer("mvmbots_puppet_look puppet=%d free", n + 1);

		return Plugin_Handled;
	}

	float pitch, yaw;

	if (args != 3 || !ArgFloat(2, pitch) || !ArgFloat(3, yaw) || FloatAbs(pitch) > 89.0 || FloatAbs(yaw) > 360.0)
	{
		PrintToServer("mvmbots_puppet_look: pitch is -89 to 89 and yaw -360 to 360, in degrees");

		return Plugin_Handled;
	}

	g_arrInputs[n].steering = true;
	g_arrInputs[n].angles[0] = pitch;
	g_arrInputs[n].angles[1] = yaw;
	g_arrInputs[n].angles[2] = 0.0;

	PrintToServer("mvmbots_puppet_look puppet=%d pitch=%.1f yaw=%.1f", n + 1, pitch, yaw);

	return Plugin_Handled;
}

/* A line typed into the puppet's console
 *
 * joinclass, voicemenu, build, taunt, say: what a player's binds send to the
 * server goes through here, by the same FakeClientCommand route
 * mvmbots_puppet_call already uses. slot1 to slot6 are the exception, being the
 * client's own: mvmbots_puppet_slot does what they turn into. What a client builds itself and sends as
 * key values does not: an engine-built MvM_UpgradesBegin or +inspect_server is
 * out of a puppet's reach, which is the limit to state wherever a result rests
 * on it. */
static Action Command_PuppetCmd(int args)
{
	if (args < 2)
	{
		PrintToServer("mvmbots_puppet_cmd: puppet command...");

		return Plugin_Handled;
	}

	int n = PuppetArg("mvmbots_puppet_cmd");

	if (n == -1)
		return Plugin_Handled;

	char line[256]; GetCmdArgString(line, sizeof(line));
	char first[16]; int skip = BreakString(line, first, sizeof(first));

	if (skip == -1)
		return Plugin_Handled;

	FakeClientCommand(g_arrPuppets[n], "%s", line[skip]);

	PrintToServer("mvmbots_puppet_cmd puppet=%d ran=%s", n + 1, line[skip]);

	return Plugin_Handled;
}

static Action Command_PuppetSlot(int args)
{
	int n = args == 2 ? PuppetArg("mvmbots_puppet_slot") : -1;
	int slot;

	if (n == -1 || !ArgInt(2, slot) || slot < 0 || slot > SHOP_SLOT_MAX)
	{
		PrintToServer("mvmbots_puppet_slot: puppet slot, slot 0 primary to %d", SHOP_SLOT_MAX);

		return Plugin_Handled;
	}

	if (!IsPlayerAlive(g_arrPuppets[n]) || GetPlayerWeaponSlot(g_arrPuppets[n], slot) == -1)
	{
		PrintToServer("mvmbots_puppet_slot: puppet %d holds nothing in slot %d", n + 1, slot);

		return Plugin_Handled;
	}

	g_arrInputs[n].select = slot;

	PrintToServer("mvmbots_puppet_slot puppet=%d slot=%d", n + 1, slot);

	return Plugin_Handled;
}

static Action Command_PuppetTeleport(int args)
{
	if (args != 4)
	{
		PrintToServer("mvmbots_puppet_teleport: puppet x y z");

		return Plugin_Handled;
	}

	int n = PuppetArg("mvmbots_puppet_teleport");

	if (n == -1)
		return Plugin_Handled;

	float point[3];

	if (!ArgFloat(2, point[0]) || !ArgFloat(3, point[1]) || !ArgFloat(4, point[2]))
	{
		PrintToServer("mvmbots_puppet_teleport: x y z are numbers");

		return Plugin_Handled;
	}

	TeleportEntity(g_arrPuppets[n], point, NULL_VECTOR, {0.0, 0.0, 0.0});

	PrintToServer("mvmbots_puppet_teleport puppet=%d pos=%.0f,%.0f,%.0f", n + 1, point[0], point[1], point[2]);

	return Plugin_Handled;
}

//Where to walk to, for a harness that steers there itself instead of teleporting
static Action Command_PuppetStation(int args)
{
	int n = args == 1 ? PuppetArg("mvmbots_puppet_station") : -1;

	if (n == -1)
	{
		if (args != 1)
			PrintToServer("mvmbots_puppet_station: puppet");

		return Plugin_Handled;
	}

	int stations;
	int station = NearestStation(g_arrPuppets[n], stations);

	if (station == -1)
	{
		PrintToServer("mvmbots_puppet_station puppet=%d stations=%d none", n + 1, stations);

		return Plugin_Handled;
	}

	float centre[3]; WorldSpaceCentre(station, centre);

	PrintToServer("mvmbots_puppet_station puppet=%d stations=%d pos=%.0f,%.0f,%.0f",
		n + 1, stations, centre[0], centre[1], centre[2]);

	return Plugin_Handled;
}
