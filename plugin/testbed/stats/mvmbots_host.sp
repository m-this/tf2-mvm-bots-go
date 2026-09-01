/* One fake player, so that a server with nobody on it can start a wave
 *
 * The defender mod adds its bots in response to a human pressing F4. Its ready
 * listener passes its own bots straight through, and the wave itself needs a
 * ready player before Mann vs Machine will begin one. So an empty server sits
 * in the pre-round forever: no ready, no wave, no bots, nothing to measure.
 *
 * This connects one fake client, puts it on RED, gives it a class and readies
 * it up. That is the whole job. It has no AI, it does not move, and it does not
 * shoot: it is a body holding a seat so the game will start.
 *
 * The cost is honest and worth stating: a six seat RED team with this in one of
 * the seats is five bots and a statue. `mvmbots_host_seat` decides whether that
 * statue is one of the six or an extra on top, and the statistics plugin
 * records how many of RED were bots at the start of every wave, so a file can
 * always say which it was.
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
	description = "Holds a seat on RED and readies up, so a wave can start with nobody playing",
	version = "1.0.0",
	url = "https://github.com/m-this/tf2-mvm-bots"
};

//Long enough after a map change that the game accepts a join, short enough not to waste a run
#define HOST_JOIN_DELAY		10.0

//How often to check that the host is still there and still ready
#define HOST_WATCH_INTERVAL	5.0

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

int g_iHost = -1;

public void OnPluginStart()
{
	g_cvEnabled = CreateConVar("mvmbots_host_enabled", "1",
		"Connect one fake client to hold a seat on RED and ready up.", _, true, 0.0, true, 1.0);

	RegServerCmd("mvmbots_roster", Command_Roster, "Say who is on each team, for a run to check before it believes itself.");
	g_cvName = CreateConVar("mvmbots_host_name", "testbed-host",
		"What to call it, so it can be told apart from the mod's own bots.");

	CreateTimer(HOST_WATCH_INTERVAL, Timer_WatchHost, _, TIMER_REPEAT | TIMER_FLAG_NO_MAPCHANGE);
}

public void OnMapStart()
{
	g_iHost = -1;

	CreateTimer(HOST_JOIN_DELAY, Timer_Connect, _, TIMER_FLAG_NO_MAPCHANGE);
}

static Action Timer_Connect(Handle timer)
{
	ConnectHost();

	return Plugin_Stop;
}

/* Keep it connected and keep it ready
 *
 * Ready state is cleared between waves, and a fake client can be dropped for
 * reasons nothing here controls. Both are cheap to check and neither is worth
 * an event hook */
static Action Timer_WatchHost(Handle timer)
{
	if (!g_cvEnabled.BoolValue)
		return Plugin_Continue;

	if (!IsHostConnected())
	{
		ConnectHost();

		return Plugin_Continue;
	}

	if (GameRules_GetRoundState() == RoundState_BetweenRounds && !IsPlayerReady(g_iHost))
		PressReady();

	return Plugin_Continue;
}

/* Press ready, and press it again a second later
 *
 * Both presses are needed and they do different things. Before the bots exist
 * the pair is what starts them, and the mod swallows each press rather than
 * passing it to the game. Once they are running the mod passes a press
 * straight through, and then it is the ready that starts the wave. Sending two
 * covers both without having to know which state the server is in */
static void PressReady()
{
	FakeClientCommand(g_iHost, "tournament_player_readystate 1");

	CreateTimer(HOST_READY_GAP, Timer_ReadyAgain, GetClientUserId(g_iHost), TIMER_FLAG_NO_MAPCHANGE);
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
	if (g_iHost > 0 && g_iHost <= MaxClients && IsClientInGame(g_iHost))
		return true;

	/* Adopt one that is already here rather than connecting a second
	The index is lost whenever this plugin is reloaded, and the client it named
	is not: without this, a reload leaves "testbed-host" standing next to
	"(1)testbed-host", and every reload after that adds another */
	char wanted[MAX_NAME_LENGTH]; g_cvName.GetString(wanted, sizeof(wanted));

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || !IsFakeClient(i))
			continue;

		char name[MAX_NAME_LENGTH]; GetClientName(i, name, sizeof(name));

		if (!StrEqual(name, wanted))
			continue;

		g_iHost = i;

		return true;
	}

	return false;
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
	PressReady();

	LogMessage("mvmbots_host: %N is holding a seat on RED", client);
}

static bool IsPlayerReady(int client)
{
	return GameRules_GetProp("m_bPlayerReady", 4, client) != 0;
}


/* Say who is on each team, in one line a runner can read

status cannot do this. It lists names and never says which side anybody is on,
and the robots are named by class on most maps: a runner counting "not a robot
name" as a defender read fourteen Pyros on BLU as a full RED and passed. The
game knows the answer and this asks it.
*/
static Action Command_Roster(int args)
{
	int red, blu, humans, host;

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i))
			continue;

		if (!IsFakeClient(i))
			humans++;
		else if (i == g_iHost)
			host++;

		switch (TF2_GetClientTeam(i))
		{
			case TFTeam_Red: red++;
			case TFTeam_Blue: blu++;
		}
	}

	//The host holds a RED seat and is not a defender, so it is named separately
	PrintToServer("mvmbots_roster red=%d blu=%d humans=%d host=%d", red, blu, humans, host);

	return Plugin_Handled;
}
