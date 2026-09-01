#include <stocklib_officerspy/tf/tf_bot>
#include <stocklib_officerspy/tf/tf_player>
#include <stocklib_officerspy/tf/tf_obj>
#include <stocklib_officerspy/tf/tf_objective_resource>
#include <stocklib_officerspy/tf/stocklib_extra_vscript>
#include <stocklib_officerspy/econ_item_view>
#include <stocklib_officerspy/tf/tf_weaponbase>
#include <stocklib_officerspy/tf/entity_capture_flag>
#include <stocklib_officerspy/shared/util_shared>
#include <stocklib_officerspy/mathlib/vector>

#define SENTRY_MAX_RANGE 1100.0

//WeaponData > Range in file tf_weapon_medigun.txt
#define WEAPON_MEDIGUN_RANGE	450.0

//CTFWeaponBuilder::InternalGetEffectBarRechargeTime
#define SAPPER_RECHARGE_TIME	15.0

//Raw value found in CBaseObject::FindBuildPointOnPlayer
#define SAPPER_PLAYER_BUILD_ON_RANGE	160.0

//ConVar cl_sidespeed
#define PLAYER_SIDESPEED	450.0

//Raw value found in CTFBotMainAction::FireWeaponAtEnemy
#define TFBOT_MELEE_ATTACK_RANGE	250.0

//PlayerLocomotion::GetStepHeight
#define TFBOT_STEP_HEIGHT	18.0

#define SNIPER_REACTION_TIME	0.5

enum //medigun_resist_types_t
{
	MEDIGUN_BULLET_RESIST = 0,
	MEDIGUN_BLAST_RESIST,
	MEDIGUN_FIRE_RESIST,
	MEDIGUN_NUM_RESISTS
}

//The game's own medigun_weapontypes_t, and its own names: 1 is the Kritzkrieg, not an uber
enum //medigun_weapontypes_t
{
	MEDIGUN_STANDARD = 0,
	MEDIGUN_CRITBOOST,
	MEDIGUN_MEGAHEAL,
	MEDIGUN_RESIST
}

enum struct BombInfo_t
{
	float vPosition[3];
	float flMinBattleFront;
	float flMaxBattleFront
}

enum
{
	TF_LOADOUT_SLOT_PRIMARY   =  0,
	TF_LOADOUT_SLOT_SECONDARY =  1,
	TF_LOADOUT_SLOT_MELEE     =  2,
	TF_LOADOUT_SLOT_UTILITY   =  3,
	TF_LOADOUT_SLOT_BUILDING  =  4,
	TF_LOADOUT_SLOT_PDA       =  5,
	TF_LOADOUT_SLOT_PDA2      =  6,
	TF_LOADOUT_SLOT_HEAD      =  7,
	TF_LOADOUT_SLOT_MISC      =  8,
	TF_LOADOUT_SLOT_ACTION    =  9,
	TF_LOADOUT_SLOT_MISC2     = 10,
	TF_LOADOUT_SLOT_TAUNT     = 11,
	TF_LOADOUT_SLOT_TAUNT2    = 12,
	TF_LOADOUT_SLOT_TAUNT3    = 13,
	TF_LOADOUT_SLOT_TAUNT4    = 14,
	TF_LOADOUT_SLOT_TAUNT5    = 15,
	TF_LOADOUT_SLOT_TAUNT6    = 16,
	TF_LOADOUT_SLOT_TAUNT7    = 17,
	TF_LOADOUT_SLOT_TAUNT8    = 18,
}

enum eMissionDifficulty
{
	MISSION_UNKNOWN = 0,
	MISSION_NORMAL,
	MISSION_INTERMEDIATE,
	MISSION_ADVANCED,
	MISSION_EXPERT,
	MISSION_NIGHTMARE,
	MISSION_MAX_COUNT
}

char g_sPlayerUseMyNameResponse[][] =
{
	"You're very funny for using my name.",
	"You totally stole my name."
};

//NOTE: Make sure this matches with the eMissionDifficulty enum size
char g_sMissionDifficultyFilePaths[][] =
{
	"",
	"configs/defenderbots/mission/mission_normal.txt",
	"configs/defenderbots/mission/mission_intermediate.txt",
	"configs/defenderbots/mission/mission_advanced.txt",
	"configs/defenderbots/mission/mission_expert.txt",
	"configs/defenderbots/mission/mission_nightmare.txt"
};

char g_sBotTeamCompositions[][][] =
{
	{"scout", "soldier", "demoman", "heavyweapons", "engineer", "medic"},
	{"scout", "heavyweapons", "heavyweapons", "heavyweapons", "engineer", "sniper"},
	{"scout", "heavyweapons", "heavyweapons", "pyro", "engineer", "demoman"}
};

char g_sRawPlayerClassNames[][] =
{
	"undefined",
	"scout",
	"sniper",
	"soldier",
	"demoman",
	"medic",
	"heavyweapons",
	"pyro",
	"spy",
	"engineer",
	"civilian",
	"",
	"random"
};

//What the explosion reaches. Valve's own is smaller, and a bot that stops running early is dead
#define BUSTER_BLAST_RANGE	400.0

//How close a live buster has to be before a bot drops what it is doing and runs
#define BUSTER_FLEE_RANGE	700.0

/* How far away a buster has to be for the engineer to still have time to move the sentry

A buster walks faster than an engineer carries. Further out than this and the engineer would put
the sentry down and pick it up again for every robot that walks past the nest */
#define BUSTER_HAUL_RANGE	1800.0

/* The nearest live sentry buster to a point, or -1 for none

A buster is a mission robot, so there is at most one of them worth caring about at a time, but
nothing in the game promises that: the loop reads them all and keeps the closest.

Busters still in the spawn room are skipped. One walks the whole length of the map to reach a
sentry, and a team that starts running when it leaves the door spends the wave running */

/* Where a man stands to put a building on a spot, on the side the attempt asks for

A building goes down in front of him and never under him, so the place to stand is a build's
reach short of the spot with the spot in front of him. Attempt zero is the side he is coming
from, which costs him no walking at all; each one after it is a step round the spot, so a spot
with a wall on one side is reached from another.

Shared because the dispenser, the teleporter and the sentry all need it, and they all learned it
separately before anybody wrote it down once. */


/* Every point on the way out of spawn an attempt might use, and where to stand to build on each

The old answer was the spawn point plus the direction to the nest times a distance, and a
direction is not a route: on a map with a wall between the two, the first attempt is inside the
wall and every attempt after it is further inside. Nothing was ever placed and nothing could be.

So it reads the nav mesh's own route instead, computed from the engineer to the spawn point and
measured back from the spawn end. The first point is a little way out of the door, and each one
after it is a step further along the way a player actually walks.

All of them at once, and not one per attempt, and that is the part that was wrong when this was
first written. The route is computed from wherever the engineer is standing. Once he has walked
out to the first point, the route left between him and spawn is shorter than the second point is
from spawn, so the second ask fails and he gives up having tried exactly one place. Sampling here,
while he is still at his nest, reads the route from the far end of the whole journey, and the
world coordinates it hands back stay true however far he walks afterwards.

Returns how many points the route was long enough for, which is none when there is no route at all
and none when the engineer is already inside the spawn room. */

//Where each engineer holds, read by the nest scoring below and written by the engineer behaviours
CNavArea m_aNestArea[MAXPLAYERS + 1] = {NULL_AREA, ...};

/* Where an engineer's nest is moving to once the wave is over, NULL_AREA when it is staying put

The bomb does not take the same route for a whole mission. Mannhattan opens and closes gates,
maps drop barricades, and which way the robots come changes from one wave to the next, so ground
that covered the approach in wave one can be facing a wall in wave three. Written once per
between-waves period and read by the engineer behaviours, which are the ones that own the
buildings standing on the old ground */
CNavArea m_aNestAreaRelocate[MAXPLAYERS + 1] = {NULL_AREA, ...};

/* The engineer items a bot has to play differently to play at all

An item definition index rather than a weapon id: the game gives the Gunslinger and the stock
wrench the same TF_WEAPON_WRENCH, and it is the item that decides whether this engineer holds a
level three or spends mini sentries */

//A nest on top of the hatch has nothing in front of it to shoot at

//Two nests closer together than this cover the same ground twice and die to the same blast

//How far from the sentry to look for ground to move it to, away from a buster

/* How close to the bomb a nest is allowed to be, as a fraction of the sentry's range

A third of it. Closer than that and the sentry spends none of its range: the robots are already
on top of it when it opens fire, the giant that walks in melees it, and the engineer holding it
is standing in the fight rather than behind it */

/* How many pieces of the approach to sample, and what seeing all of them is worth

Bounded because the term is computed for every candidate area: a map hands out as many areas
within a sentry's range of the bomb as its mesh happens to have, and a score is not worth an
unbounded loop. Two dozen spread across the ground is enough to tell a ledge over the choke from
a corner behind a wall */

/* How good a nest this area is, higher being better

Every candidate handed to this has already passed the tests that make it a nest at all: on the
bomb path, out of the spawn rooms, not on top of the hatch, within nesting depth. What is left is
a matter of degree, and picking the best of them beats the random pick this replaces, which was
free to put the sentry in a corridor behind the fight while a ledge over the choke went unused.

  range     a sentry wants the robots inside its range and not on top of it, so the ideal is most
            of the way out: close in, it is meleed by the first giant to arrive
  height    ground above the path is worth holding: the sentry shoots down onto the robots and the
            splash that answers it mostly does not reach back up
  room      a wide area fits the sentry, the dispenser and the engineer between them
  spacing   two engineers nesting on the same ledge is one blast killing both nests

An engineer carrying a Gunslinger scores the opposite way on range and height. The mini sentry is
built in two seconds and dies in one, so it is spent rather than held: it wants to be near the
robots where it is worth the metal, not on a ledge where it plinks */

/* Does the wave being fought, or the one about to be, contain a tank?

The rest of the mod finds a tank by looking for a live tank_boss, which is the right answer for
shooting at one and the wrong answer for building. An engineer picks its nest and builds during
the between-waves period, when no tank exists yet, so asking the world is always a no.

m_iszMannVsMachineWaveClassNames is the row of class icons the wave bar draws. The game fills it
in before the wave starts, and a wave with a tank in it carries the "tank" icon */

/* The nest spots the map itself carries, and where they come from

The TODO in this repository said the compiled map does not hold this data and that somebody has
to stand on the ground and decide. That is true of three of the seven official maps. It is not
true of the other four, and it is not true of most community maps built after them.

Decoy, Bigrock, Rottenburg and Mannhattan ship thirteen to twenty-seven bot_hint_sentrygun and
bot_hint_engineer_nest entities each. They are there for the engineer robots, so they are BLU's
and they face the wrong way, and neither of those things matters. A sentry shoots in a circle,
and the ground under a spot Valve chose for one team to hold a corridor from is the same ground
the other team holds the same corridor from. What the entity is really saying is that a level
three fits here, that it has a line down the lane, and that a mapper put it there on purpose.

They are read at runtime rather than copied into the configuration files, so a community map that
built on the same prefabs is covered without anybody authoring anything.

None of it is trusted blindly. Every spot goes through the same nest score as everything else, so
one deep in the robots' half loses to the nav mesh reasoning below on distance to the bomb. A
hand written EngineerNest block still outranks all of it: somebody stood there. */

/* How long an engineer's walk to a build spot is allowed to take

A flat clock is a clock that is right for one distance. Twelve seconds is generous inside a nest
and nowhere near enough to come back from the upgrade station, and what expiring meant was building
where he stood: a sentry six hundred units from its nest on Decoy, a dispenser beside the
teleporter entrance instead of on the ground the map names for it.

So the walk is priced by how far it is, floored so a short one still has room to go round the spot,
and capped so nothing waits on an engineer for ever. */
#define BUILD_WALK_SPEED	180.0
#define BUILD_WALK_TIME_MIN	12.0
#define BUILD_WALK_TIME_MAX	40.0


