package tables

/*
	Engagement ranges per weapon, by item definition index

The bot's ranges come from the weapon's ID, so every minigun is one weapon and
every shotgun is another. That is the wrong grain for a server that hands out
loadouts. A Brass Beast cannot reposition once it is spun up and wants to already
be close; a Tomislav spins up fast enough to hold a lane. A Heavy who pulled the
shotgun is standing at minigun distance with a weapon that does nothing there.
The item definition index is the only thing that tells any of them apart.

Desired is the distance the bot closes to before it settles. It is what moves a
bot, so it is the one worth setting: attack.sp walks the bot in whenever the
target is further away than this.

Max is where the bot stops firing at all. It stays RangeNone, "no opinion", for
almost everything, because a bot that refuses to shoot is worse than one that
shoots for little damage. It is set only where the shot is genuinely wasted, or
where firing point blank hurts the bot itself.

Every index here is a stock TF2 item definition. A weapon absent from the table
keeps the answer its weapon ID already gave, so this table only ever narrows
behaviour that was previously flat.
*/

// RangeNone is no opinion: the caller keeps the range its weapon ID produced.
const RangeNone = 0.0

/*
SoldierRocketSettle is where a Soldier fights, when he is allowed to choose it.

Between the Beggar's six hundred and the twelve fifty the stock launcher used to
sit at. Far enough that his own blast does not reach him, near enough that a
rocket arrives before the robot it was aimed at has walked out of the splash.
*/
const SoldierRocketSettle = 750.0

/*
DemoPipeSettle is how close a Demoman gets before he settles.

Two different numbers and it took a measurement to learn which one was the
problem.

Closing in was tried first, on the reasoning that a pipe thrown from six hundred
units is half a second in the air and a walking robot has moved a hundred and
forty by the time it lands. Three fifty instead of six hundred, six waves a side
on three maps:

	settles at 600   1403, 2096, 1305 damage a wave on Coaltown, Decoy, Mannworks
	settles at 350    929, 1274, 1238

Down on all three. Walking in is time not shooting, and in this mode the robots
are walking to him anyway, so the ground he gives up to reach them is ground he
has to cross again.
*/
const DemoPipeSettle = 600.0

/*
DemoPipeMaxRange is how far out a pipe is worth throwing.

Nine hundred was the other half of a pair that won its A/B years of commits ago,
bundled with attack_strafe, and the pair was never split. Split now, over twelve
waves on two maps: holding fire past nine hundred cost 20073 damage against
22285, a hundred and forty six team deaths against a hundred and three, and a
wave.

His hit rate is better with the cap, 43 percent against 40, and it does not
matter. The shots he is not allowed to take are worth more than the ones he
lands, because a pipe that misses still lands somewhere and the ones he was
holding were never thrown at all. attack_strafe was doing the work in that pair.
*/
const DemoPipeMaxRange = 1400.0

// Tuning is one weapon's ranges, by the item definition the game gives it.
type Tuning struct {
	// ItemDef is the item definition index, which is the only thing that
	// tells two weapons of the same kind apart.
	ItemDef int32
	// Weapon is what a reader calls it.
	Weapon string
	// Desired is where the bot settles and Max where it stops firing, each
	// spelled the way the file spells it: a number, or the name of the
	// constant the rest of the mod reads.
	Desired string
	Max     string
	// Section is the heading this weapon sits under, on the first weapon of
	// each group and empty on the rest.
	Section string
	// Note is why this weapon is not the default, where somebody wrote it
	// down. Block says it was written as a block comment rather than a line.
	Note  string
	Block bool
	// Lead is a note that sits above the case rather than inside it, which
	// is where a longer argument about the weapon goes.
	Lead string
}

// Tunings are the weapons worth telling apart, in the shipped order.
var Tunings = []Tuning{
	{
		ItemDef: 45, Weapon: "Force-A-Nature",
		Desired: "180.0", Max: "600.0",
		Section: "Scatterguns. Knockback and damage both want the bot in the target's face",
	},
	{
		ItemDef: 448, Weapon: "Soda Popper",
		Desired: "250.0", Max: "650.0",
	},
	{
		ItemDef: 425, Weapon: "Family Business",
		Desired: "280.0", Max: "700.0",
		Section: "Shotguns. Past a few hundred units the pellets are worth nothing",
	},
	{
		ItemDef: 1153, Weapon: "Panic Attack",
		Desired: "250.0", Max: "650.0",
	},
	{
		ItemDef: 199, Weapon: "Shotgun",
		Desired: "280.0", Max: "700.0",
		Lead: "The stock shotgun, which was absent and took the five hundred unit default\n\nFive hundred is minigun ground. Every other shotgun in this table sits between two and\nthree hundred, because that is where the pellets are worth anything, and the one four\nclasses actually carry was the one nobody had written down.",
	},
	{
		ItemDef: 527, Weapon: "Widowmaker",
		Desired: "300.0", Max: "750.0",
	},
	{
		ItemDef: 312, Weapon: "Brass Beast",
		Desired: "350.0", Max: "RANGE_TUNING_NONE",
		Section: "Miniguns. The difference is whether the bot can afford to be caught out of position",
	},
	{
		ItemDef: 424, Weapon: "Tomislav",
		Desired: "500.0", Max: "RANGE_TUNING_NONE",
	},
	{
		ItemDef: 996, Weapon: "Loose Cannon",
		Desired: "650.0", Max: "1500.0",
		Section: "Explosives. Far enough out that the blast does not catch the bot",
	},
	{
		ItemDef: 1151, Weapon: "Iron Bomber",
		Desired: "DEMO_PIPE_SETTLE", Max: "DemoPipeMaxRange()",
	},
	{
		ItemDef: 730, Weapon: "Beggar's Bazooka",
		Desired: "600.0", Max: "1500.0",
		Note: "Shorter than a stock rocket launcher: the rockets spread as they load",
	},
	{
		ItemDef: 997, Weapon: "Rescue Ranger",
		Desired: "800.0", Max: "RANGE_TUNING_NONE",
		Section: "Weapons that want the bot to hold its distance",
	},
	{
		ItemDef: 305, Weapon: "Crusader's Crossbow",
		Desired: "900.0", Max: "RANGE_TUNING_NONE",
	},
	{
		ItemDef: 61, Weapon: "Ambassador",
		Desired: "700.0", Max: "RANGE_TUNING_NONE",
	},
	{
		ItemDef: 412, Weapon: "Overdose",
		Desired: "300.0", Max: "800.0",
		Note: "A syringe gun is a close weapon on a class that should not be there",
	},
}
