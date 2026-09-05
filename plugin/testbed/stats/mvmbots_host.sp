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
	version = "1.1.0",
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

ConVar g_cvEnabled;
ConVar g_cvName;
ConVar g_cvPuppets;
ConVar g_cvPuppetName;
ConVar g_cvPuppetClass;

int g_iHost = -1;
int g_arrPuppets[MAX_PUPPETS];

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

	RegServerCmd("mvmbots_puppet_call", Command_PuppetCall,
		"Press MEDIC!, as a player's key does. Takes a puppet index, or nothing for all of them.");
	RegServerCmd("mvmbots_puppet_status", Command_PuppetStatus,
		"Say what each puppet is doing and who is healing it, for a run to read without the results file.");

	CreateTimer(HOST_WATCH_INTERVAL, Timer_WatchHost, _, TIMER_REPEAT | TIMER_FLAG_NO_MAPCHANGE);
}

public void OnMapStart()
{
	g_iHost = -1;

	for (int n = 0; n < MAX_PUPPETS; n++)
		g_arrPuppets[n] = -1;

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

	if (GameRules_GetRoundState() == RoundState_BetweenRounds && !IsPlayerReady(g_iHost))
		PressReady(g_iHost);

	return Plugin_Continue;
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

		if (GameRules_GetRoundState() == RoundState_BetweenRounds && !IsPlayerReady(g_arrPuppets[n]))
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
	for (int n = 0; n < MAX_PUPPETS; n++)
	{
		if (g_arrPuppets[n] == client)
			return true;
	}

	return false;
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

		PrintToServer("mvmbots_puppet %d name=%N class=%s alive=%d hp=%d healer=%s",
			n + 1, client, ClassNameOf(client), alive ? 1 : 0,
			alive ? GetClientHealth(client) : 0, healer);
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
