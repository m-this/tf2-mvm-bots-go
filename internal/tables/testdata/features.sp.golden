/* Generated from internal/tables/feature.go. Do not edit.

The table it comes from is the only place these names are written. */

/* Ways of playing that can be switched off, so two of them can be compared

Every behaviour in this mod is an argument until somebody measures it, and measuring one means
running the same mission twice with one thing different. That was being done by building two
copies of the mod and keeping them in two directories, which is slow, easy to get wrong, and
impossible to tell apart afterwards from the results alone.

A feature is a named switch with a default. It becomes a convar, so a mission can turn it off in
one line of a config, and the set that was on is written into the wave results, so a file of
numbers says which mod produced it without anybody having to remember.

Adding one is an entry in the Go table and a call to Feature() where the behaviour lives. Removing
one is deleting both, which is the point: a switch nobody has turned off in a month is a
behaviour, and it should stop being a switch. */

enum
{
	FEATURE_THREAT_PRIORITY = 0,
	FEATURE_DISPENSER_GUARD,
	FEATURE_SPY_GLANCE,
	FEATURE_STICKY_STACK,
	FEATURE_NEST_ZONES,
	FEATURE_READY_WHEN_PREPARED,
	FEATURE_WAVE_RESISTANCES,
	FEATURE_ENGINEER_DISPOSABLE,
	FEATURE_ATTACK_STRAFE,
	FEATURE_SOLDIER_CLOSES_IN,
	FEATURE_MEDIC_POCKETS_BIGGEST,
	FEATURE_DEMO_TANK_PIPES,
	FEATURE_DEMO_STICKY_SELF_VETO,
	FEATURE_HOLD_THE_NEST,
	FEATURE_MEDIC_SHIELD,
	FEATURE_PATH_LENGTH_CAP,
	FEATURE_ENGINEER_CLIMBS,
	FEATURE_WATCH_IDLE_BOTS,
	FEATURE_WATCH_LURKING_SNIPERS,
	FEATURE_AMMO_FAILOVER,
	FEATURE_MEDIC_ANSWERS_CALL,
	FEATURE_GENERATED_THREAT_PRIORITY,
	FEATURE_ENGINEER_ENTRANCE_FIRST,
	FEATURE_COUNT
}

/* Same order as the enum above, and both are written by the generator

A name inserted in the wrong place used to rename three convars: "ammo_failover" sat at
FEATURE_WATCH_IDLE_BOTS for a release, which made sm_redbots_feature_watch_lurking_snipers drive
the idle watchdog. An A/B armed the wrong feature and read as a measurement. The constant is now
the name in capitals, so the two cannot part company. */
static const char FEATURE_NAME[FEATURE_COUNT][] =
{
	"threat_priority",
	"dispenser_guard",
	"spy_glance",
	"sticky_stack",
	"nest_zones",
	"ready_when_prepared",
	"wave_resistances",
	"engineer_disposable",
	"attack_strafe",
	"soldier_closes_in",
	"medic_pockets_biggest",
	"demo_tank_pipes",
	"demo_sticky_self_veto",
	"hold_the_nest",
	"medic_shield",
	"path_length_cap",
	"engineer_climbs",
	"watch_idle_bots",
	"watch_lurking_snipers",
	"ammo_failover",
	"medic_answers_call",
	"generated_threat_priority",
	"engineer_entrance_first"
};

static ConVar g_arrFeatureConVars[FEATURE_COUNT];
static ConVar g_cvFeaturesActive;

static ConVar MakeFeature(int id, const char[] description, bool on = true)
{
	char name[64]; Format(name, sizeof(name), "sm_redbots_feature_%s", FEATURE_NAME[id]);

	/* A feature ships on once it has been measured, and off until then

	The switch exists to turn something off and measure the difference, and that is the whole point
	of it: a behaviour that has not cleared the spread of the arm it was measured against is not
	yet a behaviour this mod claims. See the rule in docs/testbed-metrics.md. */
	ConVar cv = CreateConVar(name, on ? "1" : "0", description, FCVAR_NOTIFY);

	/* Republish whenever one of these moves, rather than only when a wave begins

	The published list used to go stale between a switch being set and the next wave, and the
	statistics plugin reads it in its own handler for that wave. Whichever of the two hooks first
	is whichever SourceMod loaded first, so an arm set by rcon after the map load, which is when
	the test-bed sets one, was recorded as not set for the whole of the first wave.

	That is worse than an empty list. An empty list says nothing; a stale one says the arm was off
	and files the wave under the other side of the comparison. */
	cv.AddChangeHook(Feature_OnChanged);

	return cv;
}

static void Feature_OnChanged(ConVar convar, const char[] before, const char[] after)
{
	PublishActiveFeatures();
}

void LoadFeatures()
{
	g_arrFeatureConVars[FEATURE_THREAT_PRIORITY] = MakeFeature(FEATURE_THREAT_PRIORITY,
		"Shoot the Medic, then the Sniper and Engineer, then giants, rather than whatever is nearest.");

	g_arrFeatureConVars[FEATURE_DISPENSER_GUARD] = MakeFeature(FEATURE_DISPENSER_GUARD,
		"A hurt or dry bot holds the bomb from a friendly dispenser instead of leaving to find a health pack.");

	g_arrFeatureConVars[FEATURE_SPY_GLANCE] = MakeFeature(FEATURE_SPY_GLANCE,
		"Bots look behind themselves while the team knows a Spy is about.");

	g_arrFeatureConVars[FEATURE_STICKY_STACK] = MakeFeature(FEATURE_STICKY_STACK,
		"Sticky traps stack on one spot for a giant rather than carpeting ground for a crowd.");

	g_arrFeatureConVars[FEATURE_NEST_ZONES] = MakeFeature(FEATURE_NEST_ZONES,
		"Engineers spread across the zones a map names, so one holds inside and one holds out.");

	g_arrFeatureConVars[FEATURE_READY_WHEN_PREPARED] = MakeFeature(FEATURE_READY_WHEN_PREPARED,
		"An engineer readies at a level three nest and a medic at a full charge, not before.");

	g_arrFeatureConVars[FEATURE_WAVE_RESISTANCES] = MakeFeature(FEATURE_WAVE_RESISTANCES,
		"Buy the resistance the coming wave's robots call for, rather than ranking resistances last.");

	/* Measured and switched off: six waves of Decoy, defender deaths per wave
	[0,3,5,7,9,10] without it against [11,12,14,15,15,17] with it. The two
	spreads do not touch, so the worst wave without the mini beat the best
	wave with it. Sentries lost doubled, 7 against 14. See mvm-8ws. */
	g_arrFeatureConVars[FEATURE_ENGINEER_DISPOSABLE] = MakeFeature(FEATURE_ENGINEER_DISPOSABLE,
		"The engineer buys the disposable sentry and stands a mini beside his nest.", false);

	g_arrFeatureConVars[FEATURE_ATTACK_STRAFE] = MakeFeature(FEATURE_ATTACK_STRAFE,
		"A bot that has arrived at its firing position keeps sidestepping instead of standing still.");

	g_arrFeatureConVars[FEATURE_SOLDIER_CLOSES_IN] = MakeFeature(FEATURE_SOLDIER_CLOSES_IN,
		"A rocket is fought at a grenade's distance rather than twelve hundred units out.");

	g_arrFeatureConVars[FEATURE_MEDIC_POCKETS_BIGGEST] = MakeFeature(FEATURE_MEDIC_POCKETS_BIGGEST,
		"The game's medic is pointed at the biggest body. Off leaves him whoever he picked.");

	g_arrFeatureConVars[FEATURE_DEMO_TANK_PIPES] = MakeFeature(FEATURE_DEMO_TANK_PIPES,
		"The demoman answers a tank with pipes. Off lets him lay stickies on the hull.");

	g_arrFeatureConVars[FEATURE_DEMO_STICKY_SELF_VETO] = MakeFeature(FEATURE_DEMO_STICKY_SELF_VETO,
		"A bomb of his own close enough to hurt him stops the detonator. Off presses it anyway.");

	g_arrFeatureConVars[FEATURE_HOLD_THE_NEST] = MakeFeature(FEATURE_HOLD_THE_NEST,
		"Wait for the wave beside the engineer's sentry instead of at the robots' gate.", false);

	g_arrFeatureConVars[FEATURE_MEDIC_SHIELD] = MakeFeature(FEATURE_MEDIC_SHIELD,
		"Let the medic put up the projectile shield, and buy the rage that fills it.");

	/* Off until a run says otherwise. An unreachable goal makes NavAreaBuildPath walk the whole
	nav mesh, and Mannhattan produced an 1833 ms frame where Decoy never passed 153. See
	mvm-cf3. */
	g_arrFeatureConVars[FEATURE_PATH_LENGTH_CAP] = MakeFeature(FEATURE_PATH_LENGTH_CAP,
		"Stop a path search that has walked far enough, instead of letting an unreachable goal cost the whole mesh.", false);

	/* Off until a Bigrock run says otherwise. The exit spot there is 70 units up a rock the
	engineer cannot walk onto, so he gives up and builds where he stands, in the bot lane.
	This is the first jump in the mod aimed at a piece of ground. See mvm-fgs. */
	g_arrFeatureConVars[FEATURE_ENGINEER_CLIMBS] = MakeFeature(FEATURE_ENGINEER_CLIMBS,
		"The engineer crouch jumps onto a teleporter spot above him, and falls back to the nest ring rather than to his feet.", false);

	/* Off until a run says otherwise. The stuck watchdog only armed for a bot that was pathing, so
	an engineer frozen with an empty action stack was invisible to it: 45 seconds at one spot on
	Mannhunt and not one stuck line. See mvm-ipf. */
	g_arrFeatureConVars[FEATURE_WATCH_IDLE_BOTS] = MakeFeature(FEATURE_WATCH_IDLE_BOTS,
		"The stuck watchdog also rescues a bot that has no behaviour at all, not only one that cannot reach its goal.", false);

	/* A sniper whose lurk cannot reach its spot asks the game for a path every update, and three
	stock snipers on Decoy killed the server: the core names CTFBotSniperLurk::Update over
	NavAreaBuildPath under the watchdog. See mvm-bj8.

	On rather than off, against the usual rule, because the fault is confirmed twice from a
	player's own server and the test-bed has never once reproduced it. Peppy's second bundle ran
	v2.33.0 with the rescue built in and his sniper still sat at one position for 577 samples: the
	switch was never set, and a fix nobody turns on is not a fix.

	It costs nothing when snipers work. It fires only on a rifle sniper who is not pathing and has
	not moved for STUCK_TIME while further than SNIPER_AT_SPOT from every spot the map offers. */
	g_arrFeatureConVars[FEATURE_WATCH_LURKING_SNIPERS] = MakeFeature(FEATURE_WATCH_LURKING_SNIPERS,
		"The stuck watchdog also rescues a sniper parked nowhere near a spot, which is the one it used to miss.");

	/* Off until a run says otherwise. The pack was validated once and then repathed to every
	second with the answer thrown away, so a route that stopped existing left the bot walking
	an empty path until the pack expired. See mvm-zx0. */
	g_arrFeatureConVars[FEATURE_AMMO_FAILOVER] = MakeFeature(FEATURE_AMMO_FAILOVER,
		"A bot whose path to the ammo it picked keeps failing takes the next pack, then gives up, rather than walking at a wall.", false);

	/* Off until a run says otherwise. Reported twice, by Cowser and by Peppy: a human presses the
	medic call and the bot medic carries on healing whichever bot it had picked, which reads as
	the medic being broken. See mvm-w9b. */
	g_arrFeatureConVars[FEATURE_MEDIC_ANSWERS_CALL] = MakeFeature(FEATURE_MEDIC_ANSWERS_CALL,
		"A player who calls for a medic takes the beam, and a player outranks a bot for it either way.", false);

	/* Off until a run says otherwise, and this one is meant to change nothing. It is the port in
	mvm-z83.6 wired up: the two are proved identical over the whole domain under SourcePawn's own
	VM, so an arm that moves is the edge filling the record wrong, not the decision. */
	g_arrFeatureConVars[FEATURE_GENERATED_THREAT_PRIORITY] = MakeFeature(FEATURE_GENERATED_THREAT_PRIORITY,
		"Rank threats with the table generated from the Go, rather than with the hand written chain.", false);

	/* Off until a run says otherwise. The entrance used to wait for the nest, so the engineer built
	it, walked to the nest, built there, and walked back to spawn for the entrance. Peppy asked
	for the entrance first and the walk is what the test-bed measures. See mvm-dh8. */
	g_arrFeatureConVars[FEATURE_ENGINEER_ENTRANCE_FIRST] = MakeFeature(FEATURE_ENGINEER_ENTRANCE_FIRST,
		"The engineer puts his teleporter entrance up in spawn before he walks out to build the nest.", false);

	/* What is on, as one string, for whoever reads the results later

	Written rather than read: nothing in the mod uses it. It exists so the statistics plugin can
	put it in the file, because a run whose settings are not recorded is a run that cannot be
	compared with anything. */
	g_cvFeaturesActive = CreateConVar("sm_redbots_features_active", "",
		"The features that are on, comma separated. Set by the mod, not by you.", FCVAR_NONE);
}

//A feature nobody switched is on, so a config that names none of them gets the mod as shipped
bool Feature(int id)
{
	if (id < 0 || id >= FEATURE_COUNT || g_arrFeatureConVars[id] == null)
		return true;

	return g_arrFeatureConVars[id].BoolValue;
}

/* Publish the set that is on

Called when a wave begins rather than once at map start: a config file executes at its own pace
and a late loaded plugin misses it entirely, so the answer earlier than this is the defaults
rather than what the server was asked for. */
void PublishActiveFeatures()
{
	if (g_cvFeaturesActive == null)
		return;

	char list[512];

	for (int i = 0; i < FEATURE_COUNT; i++)
	{
		if (!Feature(i))
			continue;

		if (list[0] != '\0')
			StrCat(list, sizeof(list), ",");

		StrCat(list, sizeof(list), FEATURE_NAME[i]);
	}

	/* The three BLU scales ride along, because they are the other thing that can
	make one run differ from another and a results file has to say so. They are
	convars rather than switches: 1.0 is off, so a scale that is set at all is
	worth recording, and there is nothing to record when none is. */
	BluAssist_Describe(list, sizeof(list));

	g_cvFeaturesActive.SetString(list);
}
