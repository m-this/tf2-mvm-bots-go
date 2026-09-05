// Package tables holds the facts that used to be written down twice.
//
// A feature was an enum entry in one place and a string in a parallel array in
// another, and the compiler only ever counted them. Here each fact is one entry
// in one slice, and both sides of it are derived rather than transcribed: the
// enum constant comes from the name, so a name in the wrong place moves its own
// constant with it.
package tables

import "strings"

// Feature is one switchable way of playing.
//
// Name is the whole of the identity. The enum constant and the convar name are
// computed from it, which is the property the parallel array did not have.
type Feature struct {
	Name        string
	Description string
	On          bool

	// Note is the comment written above the CreateConVar call, usually the run
	// that decided the default. Kept here so the table carries the reason with
	// the switch rather than leaving it behind in the generated file.
	Note string
}

// Enum is the SourcePawn constant for this feature.
func (f Feature) Enum() string { return "FEATURE_" + strings.ToUpper(f.Name) }

// ConVar is the console variable a config sets to turn this feature off.
func (f Feature) ConVar() string { return "sm_redbots_feature_" + f.Name }

// Default is the value CreateConVar is given.
//
// A feature ships on once it has been measured and off until then: a behaviour
// that has not cleared the spread of the arm it was measured against is not a
// behaviour this mod claims.
func (f Feature) Default() string {
	if f.On {
		return "1"
	}
	return "0"
}

// FeaturesActiveConVar is written by the mod and read by the statistics plugin,
// so a file of numbers says which mod produced it.
const FeaturesActiveConVar = "sm_redbots_features_active"

// Features is the table, in enum order. Appending is safe; reordering renames
// nothing, because every name in the generated output comes from Name.
var Features = []Feature{
	{
		Name:        "threat_priority",
		Description: "Shoot the Medic, then the Sniper and Engineer, then giants, rather than whatever is nearest.",
		On:          true,
	},
	{
		Name:        "dispenser_guard",
		Description: "A hurt or dry bot holds the bomb from a friendly dispenser instead of leaving to find a health pack.",
		On:          true,
	},
	{
		Name:        "spy_glance",
		Description: "Bots look behind themselves while the team knows a Spy is about.",
		On:          true,
	},
	{
		Name:        "sticky_stack",
		Description: "Sticky traps stack on one spot for a giant rather than carpeting ground for a crowd.",
		On:          true,
	},
	{
		Name:        "nest_zones",
		Description: "Engineers spread across the zones a map names, so one holds inside and one holds out.",
		On:          true,
	},
	{
		Name:        "ready_when_prepared",
		Description: "An engineer readies at a level three nest and a medic at a full charge, not before.",
		On:          true,
	},
	{
		Name:        "wave_resistances",
		Description: "Buy the resistance the coming wave's robots call for, rather than ranking resistances last.",
		On:          true,
	},
	{
		Name:        "engineer_disposable",
		Description: "The engineer buys the disposable sentry and stands a mini beside his nest.",
		On:          false,
		Note: "Measured and switched off: six waves of Decoy, defender deaths per wave\n" +
			"[0,3,5,7,9,10] without it against [11,12,14,15,15,17] with it. The two\n" +
			"spreads do not touch, so the worst wave without the mini beat the best\n" +
			"wave with it. Sentries lost doubled, 7 against 14. See mvm-8ws.",
	},
	{
		Name:        "attack_strafe",
		Description: "A bot that has arrived at its firing position keeps sidestepping instead of standing still.",
		On:          true,
	},
	{
		Name:        "soldier_closes_in",
		Description: "A rocket is fought at a grenade's distance rather than twelve hundred units out.",
		On:          true,
	},
	{
		Name:        "medic_pockets_biggest",
		Description: "The game's medic is pointed at the biggest body. Off leaves him whoever he picked.",
		On:          true,
	},
	{
		Name:        "demo_tank_pipes",
		Description: "The demoman answers a tank with pipes. Off lets him lay stickies on the hull.",
		On:          true,
	},
	{
		Name:        "demo_sticky_self_veto",
		Description: "A bomb of his own close enough to hurt him stops the detonator. Off presses it anyway.",
		On:          true,
	},
	{
		Name:        "hold_the_nest",
		Description: "Wait for the wave beside the engineer's sentry instead of at the robots' gate.",
		On:          false,
	},
	{
		Name:        "medic_shield",
		Description: "Let the medic put up the projectile shield, and buy the rage that fills it.",
		On:          true,
	},
	{
		Name:        "path_length_cap",
		Description: "Stop a path search that has walked far enough, instead of letting an unreachable goal cost the whole mesh.",
		On:          false,
		Note: "Off until a run says otherwise. An unreachable goal makes NavAreaBuildPath walk the whole\n" +
			"nav mesh, and Mannhattan produced an 1833 ms frame where Decoy never passed 153. See\n" +
			"mvm-cf3.",
	},
	{
		Name:        "engineer_climbs",
		Description: "The engineer crouch jumps onto a teleporter spot above him, and falls back to the nest ring rather than to his feet.",
		On:          false,
		Note: "Off until a Bigrock run says otherwise. The exit spot there is 70 units up a rock the\n" +
			"engineer cannot walk onto, so he gives up and builds where he stands, in the bot lane.\n" +
			"This is the first jump in the mod aimed at a piece of ground. See mvm-fgs.",
	},
	{
		Name:        "watch_idle_bots",
		Description: "The stuck watchdog also rescues a bot that has no behaviour at all, not only one that cannot reach its goal.",
		On:          false,
		Note: "Off until a run says otherwise. The stuck watchdog only armed for a bot that was pathing, so\n" +
			"an engineer frozen with an empty action stack was invisible to it: 45 seconds at one spot on\n" +
			"Mannhunt and not one stuck line. See mvm-ipf.",
	},
	{
		Name:        "watch_lurking_snipers",
		Description: "The stuck watchdog also rescues a sniper parked nowhere near a spot, which is the one it used to miss.",
		On:          true,
		Note: "A sniper whose lurk cannot reach its spot asks the game for a path every update, and three\n" +
			"stock snipers on Decoy killed the server: the core names CTFBotSniperLurk::Update over\n" +
			"NavAreaBuildPath under the watchdog. See mvm-bj8.\n" +
			"\n" +
			"On rather than off, against the usual rule, because the fault is confirmed twice from a\n" +
			"player's own server and the test-bed has never once reproduced it. Peppy's second bundle ran\n" +
			"v2.33.0 with the rescue built in and his sniper still sat at one position for 577 samples: the\n" +
			"switch was never set, and a fix nobody turns on is not a fix.\n" +
			"\n" +
			"It costs nothing when snipers work. It fires only on a rifle sniper who is not pathing and has\n" +
			"not moved for STUCK_TIME while further than SNIPER_AT_SPOT from every spot the map offers.",
	},
	{
		Name:        "ammo_failover",
		Description: "A bot whose path to the ammo it picked keeps failing takes the next pack, then gives up, rather than walking at a wall.",
		On:          false,
		Note: "Off until a run says otherwise. The pack was validated once and then repathed to every\n" +
			"second with the answer thrown away, so a route that stopped existing left the bot walking\n" +
			"an empty path until the pack expired. See mvm-zx0.",
	},
	{
		Name:        "medic_answers_call",
		Description: "A player who calls for a medic takes the beam, and a player outranks a bot for it either way.",
		On:          true,
		Note: "On, measured 2026-09-05 with a calling puppet on Decoy, three attempts each of two waves: the\n" +
			"caller held the beam 26%, 0% and 12% of the time with this on and 2% at most with it\n" +
			"off; the beam was on somebody 30 to 44% of the time either way and every wave cleared in both\n" +
			"arms. Reported twice, by Cowser and by Peppy: a human presses the medic call and the bot medic\n" +
			"carries on healing whichever bot it had picked. See mvm-w9b.",
	},
	{
		Name:        "generated_threat_priority",
		Description: "Rank threats with the table generated from the Go, rather than with the hand written chain.",
		On:          false,
		Note: "Off until a run says otherwise, and this one is meant to change nothing. It is the port in\n" +
			"mvm-z83.6 wired up: the two are proved identical over the whole domain under SourcePawn's own\n" +
			"VM, so an arm that moves is the edge filling the record wrong, not the decision.",
	},
	{
		Name:        "engineer_entrance_first",
		Description: "The engineer puts his teleporter entrance up in spawn before he walks out to build the nest.",
		On:          false,
		Note: "Off: measured and not clearly better. Three attempts each of two waves of Decoy, 2026-09-05:\n" +
			"the entrance goes up at 18s of the break against 37s, the exit at 33s against 65s, the sentry at\n" +
			"17s against 7s, and the walk is the same 15600 units. Peppy asked for the entrance first, and it\n" +
			"is; whether ten seconds off the sentry is worth thirty off the exit is a call for a longer run.\n" +
			"See mvm-dh8.",
	},
	{
		Name:        "bot_test_by_nextbot",
		Description: "A seat is one of the game's own bots when it has a nextbot, rather than whenever it is a fake client.",
		On:          false,
		Note: "Off until a run says otherwise. IsTFBotPlayer was IsFakeClient, so every body the test-bed\n" +
			"seats read as a bot: medic_answers_call was a no-op there by construction and no puppet\n" +
			"could stand in for a player. Defenders come from tf_bot_add and robots from the popfile, so\n" +
			"both have a nextbot and a CreateFakeClient body has none. It is read in five places, so it\n" +
			"moves more than the medic. See mvm-z83.93.",
	},
	{
		Name:        "engineer_setup_phase",
		Description: "Between waves the engineer claims his spots, jumps to each of them, and what he builds comes up finished.",
		On:          true,
		Note: "On by Mathis's decision, with the trade on the record. Measured 2026-09-05 on Decoy, two\n" +
			"engineers, three attempts each of two waves: the entrance goes up at 21 to 30s of the break\n" +
			"against 37 to 45s, the exit at 26 to 38s against 50 to 58s, and the walk drops from 13810\n" +
			"and 15544 units to 12877 and 12404. Six of six waves cleared in both arms. Against that the\n" +
			"sentry is at level three in 76% of samples rather than 85%: the wrench is what applies a\n" +
			"level and an engineer who jumps has less time swinging. See mvm-dn5, and mvm-9nu for\n" +
			"upgrading a building outright, which is what removes the trade.",
	},
}
