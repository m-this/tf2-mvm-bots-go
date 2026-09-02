/*
Package shared is what source/redbots3/util.sp had left: the constants and the
enums several generated files read, and nothing else.

Every one of them is read only by generated SourcePawn now, so the plugin's own
copy was a declaration nobody used. They live here and are emitted once, ahead
of everything that reads them.
Nothing here is read from Go: every declaration exists to be emitted once, ahead
of the generated files that read it. The linter is told so at each one rather
than at the package, so a declaration that stops being read is still noticed.
*/
package shared

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// SentryMaxRange is how far a sentry shoots, which is the ground a nest covers.
//
//sp:name SENTRY_MAX_RANGE
const SentryMaxRange = 1100.0

// WeaponMedigunRange is how far the beam reaches.
//
//sp:name WEAPON_MEDIGUN_RANGE
const WeaponMedigunRange = 450.0

// SapperRechargeTime is how long a spy waits between sappers.
//
//sp:name SAPPER_RECHARGE_TIME
const SapperRechargeTime = 15.0

// SapperPlayerBuildOnRange is how close a spy has to be to sap what a player
// built.
//
//sp:name SAPPER_PLAYER_BUILD_ON_RANGE
const SapperPlayerBuildOnRange = 160.0

// PlayerSidespeed is how fast a player strafes, which the sidestep is computed
// from.
//
//sp:name PLAYER_SIDESPEED
const PlayerSidespeed = 450.0

// TFBotMeleeAttackRange is how close a bot swings from.
//
//sp:name TFBOT_MELEE_ATTACK_RANGE
const TFBotMeleeAttackRange = 250.0

// TFBotStepHeight is how high a bot walks up without jumping, which decides
// whether a build spot is reachable.
//
//sp:name TFBOT_STEP_HEIGHT
const TFBotStepHeight = 18.0

// SniperReactionTime is how long a bot sniper takes to fire once it is on
// target.
//
//sp:name SNIPER_REACTION_TIME
const SniperReactionTime = 0.5

// The three resistances the vaccinator cycles, in the game's order.
const (
	//sp:name MEDIGUN_BULLET_RESIST
	ResistBullet = 0
	//sp:name MEDIGUN_BLAST_RESIST
	ResistBlast = 1
	//sp:name MEDIGUN_FIRE_RESIST
	ResistFire = 2
	//sp:name MEDIGUN_NUM_RESISTS
	ResistCount = 3
)

/*
The four mediguns, with the game's own names: 1 is the Kritzkrieg, not an uber.
*/
const (
	//sp:name MEDIGUN_STANDARD
	MedigunStandard = 0
	//sp:name MEDIGUN_CRITBOOST
	MedigunCritBoost = 1
	//sp:name MEDIGUN_MEGAHEAL
	MedigunMegaheal = 2
	//sp:name MEDIGUN_RESIST
	MedigunResist = 3
)

// BombInfo is where the bomb is and how far in front of it the fight should be.
//
//sp:name BombInfo_t
type BombInfo struct {
	Position       [3]float32 `sp:"vPosition"`
	MinBattleFront float32    `sp:"flMinBattleFront"`
	MaxBattleFront float32    `sp:"flMaxBattleFront"`
}

/*
The schema's slot numbering, which is not the weapon slot numbering the game
uses at runtime.
*/
const (
	//sp:name TF_LOADOUT_SLOT_PRIMARY
	LoadoutPrimary = 0
	//sp:name TF_LOADOUT_SLOT_SECONDARY
	LoadoutSecondary = 1
	//sp:name TF_LOADOUT_SLOT_MELEE
	LoadoutMelee = 2
	//sp:name TF_LOADOUT_SLOT_UTILITY
	LoadoutUtility = 3
	//sp:name TF_LOADOUT_SLOT_BUILDING
	LoadoutBuilding = 4
	//sp:name TF_LOADOUT_SLOT_PDA
	LoadoutPDA = 5
	//sp:name TF_LOADOUT_SLOT_PDA2
	LoadoutPDA2 = 6
	//sp:name TF_LOADOUT_SLOT_HEAD
	LoadoutHead = 7
	//sp:name TF_LOADOUT_SLOT_MISC
	LoadoutMisc = 8
	//sp:name TF_LOADOUT_SLOT_ACTION
	LoadoutAction = 9
	//sp:name TF_LOADOUT_SLOT_MISC2
	LoadoutMisc2 = 10
	//sp:name TF_LOADOUT_SLOT_TAUNT
	LoadoutTaunt = 11
)

// BusterBlastRange is what the explosion reaches. Valve's own is smaller, and a
// bot that stops running early is dead.
//
//sp:name BUSTER_BLAST_RANGE
const BusterBlastRange = 400.0

// BusterFleeRange is how close a live buster has to be before a bot drops what
// it is doing and runs.
//
//sp:name BUSTER_FLEE_RANGE
const BusterFleeRange = 700.0

/*
BusterHaulRange is how close a buster has to be before an engineer picks the
sentry up.

A buster walks faster than an engineer carries. Further out than this and the
engineer would put the sentry down and pick it up again for every robot that
walks past the nest.
*/
//
//sp:name BUSTER_HAUL_RANGE
const BusterHaulRange = 1800.0

// BuildWalkSpeed is how fast an engineer carrying a building moves, which the
// walk time is estimated from.
//
//sp:name BUILD_WALK_SPEED
const BuildWalkSpeed = 180.0

// BuildWalkTimeMin and BuildWalkTimeMax bound that estimate, because a route
// the mesh reports as very short or very long is usually the mesh being odd
// rather than the walk being odd.
//
//sp:name BUILD_WALK_TIME_MIN
const BuildWalkTimeMin = 12.0

// BuildWalkTimeMax is the upper end of that bound.
//
//sp:name BUILD_WALK_TIME_MAX
const BuildWalkTimeMax = 40.0

// MissionDifficulty is eMissionDifficulty, which of the five a popfile says it
// is.
//
//sp:name eMissionDifficulty
type MissionDifficulty int32

// The mission difficulties, in the order the file names are filed under.
const (
	//sp:name MISSION_UNKNOWN
	MissionUnknown MissionDifficulty = 0
	//sp:name MISSION_NORMAL
	MissionNormal MissionDifficulty = 1
	//sp:name MISSION_INTERMEDIATE
	MissionIntermediate MissionDifficulty = 2
	//sp:name MISSION_ADVANCED
	MissionAdvanced MissionDifficulty = 3
	//sp:name MISSION_EXPERT
	MissionExpert MissionDifficulty = 4
	//sp:name MISSION_NIGHTMARE
	MissionNightmare MissionDifficulty = 5
	//sp:name MISSION_MAX_COUNT
	MissionMaxCount MissionDifficulty = 6
)

// The mission file each difficulty is read from. Index for index with the
// difficulties above.
//
//sp:name g_sMissionDifficultyFilePaths
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var missionDifficultyFilePaths = [6]string{
	"",
	"configs/defenderbots/mission/mission_normal.txt",
	"configs/defenderbots/mission/mission_intermediate.txt",
	"configs/defenderbots/mission/mission_advanced.txt",
	"configs/defenderbots/mission/mission_expert.txt",
	"configs/defenderbots/mission/mission_nightmare.txt",
}

/*
The game's own spelling of every class, indexed by TFClassType.

Index 0 is TFClass_Unknown, and the two at the end are what the game calls a
slot with nothing in it and a slot set to random.
*/
//
//sp:name g_sRawPlayerClassNames
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var rawPlayerClassNames = [13]string{
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
	"random",
}

/*
The ground this engineer nests on, and the ground he should move to.

The bomb does not take the same route for a whole mission. Mannhattan opens and
closes gates, maps drop barricades, and which way the robots come changes from
one wave to the next, so ground that covered the approach in wave one can be
facing a wall in wave three. The second is written once per between-waves period
and read by the engineer behaviours, which own the buildings standing on the old
ground.
*/

//sp:name m_aNestArea
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var nestArea = [65]engine.Area{engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea()}

//sp:name m_aNestAreaRelocate
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var nestAreaRelocate = [65]engine.Area{engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea(), engine.NullArea()}
