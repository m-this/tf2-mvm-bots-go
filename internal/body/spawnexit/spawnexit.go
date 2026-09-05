/*
Package spawnexit is the spawn-exit watch of
source/redbots3/nextbot_behavior.sp: a valid path can still drive a bot into a
corner without getting it out of spawn, so progress out of it is watched, and a
bot making none is moved to walkable ground near the objective.
*/
package spawnexit

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

// SpawnExitWatchInterval is how often the watch looks.
//
//sp:name SPAWN_EXIT_WATCH_INTERVAL
const SpawnExitWatchInterval = 1.0

// SpawnExitStallTime is how long without progress calls the walk stalled.
//
//sp:name SPAWN_EXIT_STALL_TIME
const SpawnExitStallTime = 6.0

// SpawnExitProgress is how much flat movement counts as progress.
//
//sp:name SPAWN_EXIT_PROGRESS
const SpawnExitProgress = 96.0

// SpawnRecoveryTries is how many points of the recovery area are tried for one
// the bot can stand at before the recovery is given up for this look.
//
//sp:name SPAWN_RECOVERY_TRIES
const SpawnRecoveryTries = 8

//sp:name m_vecSpawnExitProgress
var spawnExitProgress [slots.Count][3]float32

//sp:name m_flSpawnExitProgressAt
var spawnExitProgressAt [slots.Count]float32

//sp:name m_flSpawnExitStartedAt
var spawnExitStartedAt [slots.Count]float32

//sp:name m_flSpawnExitWatchAt
var spawnExitWatchAt [slots.Count]float32

// ResetSpawnExitWatch forgets the walk, which a successful exit is.
//
//sp:name ResetSpawnExitWatch
func ResetSpawnExitWatch(client int32) {
	spawnExitProgress[client] = engine.NullVector()
	spawnExitProgressAt[client] = 0.0
	spawnExitStartedAt[client] = 0.0
	spawnExitWatchAt[client] = 0.0
}

// DistanceFromPointToBounds is how far outside the box the point is, and zero
// inside it.
//
//sp:name DistanceFromPointToBounds
//sp:const point
//sp:const mins
//sp:const maxs
func DistanceFromPointToBounds(point [3]float32, mins [3]float32, maxs [3]float32) float32 {
	var squared float32

	for axis := int32(0); axis < 3; axis++ {
		var outside float32

		if point[axis] < mins[axis] {
			outside = mins[axis] - point[axis]
		} else if point[axis] > maxs[axis] {
			outside = point[axis] - maxs[axis]
		}

		squared += outside * outside
	}

	return engine.SquareRoot(squared)
}

// DistanceToClosestDefenderSpawn walks the respawn rooms and measures to the
// nearest live one on the bot's team.
//
//sp:name DistanceToClosestDefenderSpawn
func DistanceToClosestDefenderSpawn(client int32) float32 {
	point := engine.WorldSpaceCenter(client)
	closest := float32(-1.0)
	room := int32(-1)

	for {
		room = engine.FindEntityByClassname(room, "func_respawnroom")

		if room == -1 {
			break
		}

		team := engine.EntityTeamNumber(room)

		if team != 0 && team != engine.GetClientTeam(client) {
			continue
		}

		if engine.HasEntProp(room, engine.PropData(), "m_bDisabled") && engine.EntProp(room, engine.PropData(), "m_bDisabled") != 0 {
			continue
		}

		origin := engine.AbsOriginOf(room)
		mins := engine.EntPropVector(room, engine.PropData(), "m_vecMins")
		maxs := engine.EntPropVector(room, engine.PropData(), "m_vecMaxs")
		mins = engine.AddVectors(mins, origin)
		maxs = engine.AddVectors(maxs, origin)

		distance := DistanceFromPointToBounds(point, mins, maxs)

		if closest < 0.0 || distance < closest {
			closest = distance
		}
	}

	return closest
}

// IsInOrNearDefenderSpawn says the bot is in a spawn or within the configured
// radius of one.
//
//sp:name IsInOrNearDefenderSpawn
func IsInOrNearDefenderSpawn(client int32) bool {
	if engine.IsPointInRespawnRoom(engine.WorldSpaceCenter(client)) {
		return true
	}

	distance := DistanceToClosestDefenderSpawn(client)
	return distance >= 0.0 && distance <= engine.SpawnNavRecoveryRadius().Float()
}

// ShouldWatchDefenderSpawnExit says the bot has finished the thing spawn is
// for, so lingering is a fault and not a shopping trip.
//
//sp:name ShouldWatchDefenderSpawnExit
func ShouldWatchDefenderSpawnExit(client int32) bool {
	state := engine.RoundState()

	if state == engine.RoundStateRunning() {
		return engine.HasUpgraded(client)
	}

	return state == engine.RoundStateBetweenRounds() && (engine.ShoppedThisBreak(client) || !engine.UseUpgrades().Bool())
}

// FindNearestRecoveryAreaByClassname is the nav under the closest entity of
// that class.
//
//sp:name FindNearestRecoveryAreaByClassname
func FindNearestRecoveryAreaByClassname(client int32, classname string) engine.Area {
	clientPosition := engine.WorldSpaceCenter(client)
	closest := float32(999999.0)
	best := engine.NullArea()
	entity := int32(-1)

	for {
		entity = engine.FindEntityByClassname(entity, classname)

		if entity == -1 {
			break
		}

		position := engine.WorldSpaceCenter(entity)
		area := engine.NearestNavArea(position, true, 1500.0, true, true, engine.TeamAny())

		if area == engine.NullArea() {
			continue
		}

		distance := engine.VectorDistance(clientPosition, position)

		if distance < closest {
			closest = distance
			best = area
		}
	}

	return best
}

/*
FindSpawnRecoveryArea is ground worth standing on, tried in order of how much it
says about the map: the capture trigger, a capture zone, a control point, and
failing all three the area with the shortest travel to the bomb target.
*/
//
//sp:name FindSpawnRecoveryArea
//sp:length source sourceLength
//
//nolint:revive,ineffassign,staticcheck,wastedassign // the write is the point: SourcePawn passes the buffer by reference and //sp:length carries its size
func FindSpawnRecoveryArea(client int32, source engine.Text, sourceLength int32) engine.Area {
	anchor := engine.CapturableAreaTrigger(engine.PlayerEnemyTeam(client))

	if anchor != -1 {
		area := engine.NearestNavArea(engine.WorldSpaceCenter(anchor), true, 1500.0, true, true, engine.TeamAny())

		if area != engine.NullArea() {
			source = engine.CopyText("capture trigger")
			return area
		}
	}

	best := FindNearestRecoveryAreaByClassname(client, "func_capturezone")

	if best != engine.NullArea() {
		source = engine.CopyText("func_capturezone")
		return best
	}

	best = FindNearestRecoveryAreaByClassname(client, "team_control_point")

	if best != engine.NullArea() {
		source = engine.CopyText("control point")
		return best
	}

	shortestBombTravel := float32(999999.0)

	for i := int32(0); i < engine.NavAreaCount(); i++ {
		area := engine.AllNavAreas().NavAreaAt(i)

		if area == engine.NoNavArea() || area.HasAttributeTF(engine.RedSpawnRoom()) || area.HasAttributeTF(engine.BlueSpawnRoom()) {
			continue
		}

		travel := engine.TravelDistanceToBombTarget(area)

		if travel >= 0.0 && travel < shortestBombTravel {
			shortestBombTravel = travel
			best = engine.Area(area)
		}
	}

	if best != engine.NullArea() {
		source = engine.CopyText("bomb-target NAV")
	}

	return best
}

// IsRoomToStand says a standing player's box fits at the point, against
// everything that blocks a player. A nav area's random point can sit inside a
// prop or under a ledge, and a bot put there never finishes its next path search.
//
//sp:name IsRoomToStand
func IsRoomToStand(point [3]float32) bool {
	var mins [3]float32
	var maxs [3]float32
	mins[0] = -24.0
	mins[1] = -24.0
	mins[2] = 0.0
	maxs[0] = 24.0
	maxs[1] = 24.0
	maxs[2] = 82.0

	engine.TraceHull(point, point, mins, maxs, engine.MaskPlayerSolid())
	return !engine.DidHit()
}

// RecoveryDestination is a point of the area with room to stand, if a few draws
// find one.
//
//sp:name RecoveryDestination
func RecoveryDestination(area engine.Area) (found bool, destination [3]float32) {
	for attempt := int32(0); attempt < SpawnRecoveryTries; attempt++ {
		destination = engine.RandomPointIn(area)
		destination[2] += 10.0

		if IsRoomToStand(destination) {
			return true, destination
		}
	}

	return false, destination
}

// MoveDefenderFromSpawnToBattlefield moves the bot to walkable NAV near the
// final objective, then lets normal class behaviour resume.
//
//sp:name MoveDefenderFromSpawnToBattlefield
func MoveDefenderFromSpawnToBattlefield(client int32, reason string) bool {
	if client < 1 || client > engine.MaxClients() || !engine.IsClientInGame(client) || !engine.IsPlayerAlive(client) || !engine.DefenderBotFlag(client) {
		return false
	}

	if !IsInOrNearDefenderSpawn(client) {
		return false
	}

	var anchorSource engine.Text
	area := FindSpawnRecoveryArea(client, anchorSource, 32)

	if area == engine.NullArea() {
		return false
	}

	found, destination := RecoveryDestination(area)

	if !found {
		engine.LogMessage("SpawnNavRecovery: %N %s; no room to stand in the %s area, so they stay", client, reason, anchorSource)
		return false
	}

	var stopped [3]float32

	engine.TeleportEntity(client, destination, engine.NullVector(), stopped)
	engine.CombatOf(client).UpdateLastKnownArea()
	engine.SetRepathTime(client, engine.GameTime()+engine.RandomFloat(0.5, 1.0))
	ResetSpawnExitWatch(client)

	engine.LogMessage("SpawnNavRecovery: %N %s; moved them using %s", client, reason, anchorSource)
	return true
}

// RecoverDefenderFromDisconnectedSpawn moves a bot whose spawn has no route
// out at all, which a mission with gates produces on purpose.
//
//sp:name RecoverDefenderFromDisconnectedSpawn
func RecoverDefenderFromDisconnectedSpawn(client int32) bool {
	if client < 1 || client > engine.MaxClients() || !engine.IsClientInGame(client) || !engine.IsPlayerAlive(client) || !engine.DefenderBotFlag(client) {
		return false
	}

	if !IsInOrNearDefenderSpawn(client) {
		return false
	}

	var anchorSource engine.Text
	area := FindSpawnRecoveryArea(client, anchorSource, 32)

	if area == engine.NullArea() {
		return false
	}

	anchor := area.Center()

	if engine.IsPathToVectorPossible(client, anchor) {
		return false
	}

	return MoveDefenderFromSpawnToBattlefield(client, "has no route out of spawn")
}

/*
WatchDefenderSpawnExit is the watch itself, once a second per bot.

Two clocks: the whole stay, against the configured recovery time, and the walk,
against six seconds without ninety six units of flat progress. Height is left
out of the progress so a bot riding a lift or bobbing on a ramp does not read as
walking.
*/
//
//sp:name WatchDefenderSpawnExit
func WatchDefenderSpawnExit(client int32) {
	if !engine.SpawnNavRecovery().Bool() || !ShouldWatchDefenderSpawnExit(client) ||
		engine.IsInUpgradeZone(client) || !IsInOrNearDefenderSpawn(client) {
		ResetSpawnExitWatch(client)
		return
	}

	now := engine.GameTime()

	if spawnExitWatchAt[client] > now {
		return
	}

	spawnExitWatchAt[client] = now + SpawnExitWatchInterval

	if spawnExitStartedAt[client] == 0.0 {
		spawnExitStartedAt[client] = now
	} else if now-spawnExitStartedAt[client] >= engine.SpawnNavRecoveryTime().Float() {
		MoveDefenderFromSpawnToBattlefield(client, "exceeded the configured time for leaving spawn")
		return
	}

	here := engine.AbsOriginOf(client)
	flatHere := here
	flatPrevious := spawnExitProgress[client]
	flatHere[2] = 0.0
	flatPrevious[2] = 0.0

	if spawnExitProgressAt[client] == 0.0 || engine.VectorDistance(flatHere, flatPrevious) >= SpawnExitProgress {
		spawnExitProgress[client] = here
		spawnExitProgressAt[client] = now
		return
	}

	if now-spawnExitProgressAt[client] < SpawnExitStallTime {
		return
	}

	MoveDefenderFromSpawnToBattlefield(client, "made no spawn-exit progress for six seconds")
}

/*
CommandDumpSpawnNav says, per bot, everything the watch above is looking at.

Four runs were spent guessing why a bot did or did not get recovered; this
prints the answer instead. mvm-qhi is why it exists.
*/
//
//sp:name Command_DumpSpawnNav
//sp:public
//
//nolint:revive // unused-parameter: the argument count is the console's, and this command takes none
func CommandDumpSpawnNav(client int32, args int32) engine.Outcome {
	engine.ReplyToCommand(client, "Spawn NAV recovery: enabled %d, radius %.0f, max time %.1f", engine.SpawnNavRecovery().Bool(),
		engine.SpawnNavRecoveryRadius().Float(), engine.SpawnNavRecoveryTime().Float())

	for bot := int32(1); bot <= engine.MaxClients(); bot++ {
		if !engine.IsClientInGame(bot) || !engine.IsPlayerAlive(bot) || !engine.DefenderBotFlag(bot) {
			continue
		}

		strict := engine.IsPointInRespawnRoomStrict(engine.WorldSpaceCenter(bot), bot)
		distance := DistanceToClosestDefenderSpawn(bot)
		near := strict || distance >= 0.0 && distance <= engine.SpawnNavRecoveryRadius().Float()
		now := engine.GameTime()
		watched := engine.ChooseFloat(spawnExitStartedAt[bot] > 0.0, now-spawnExitStartedAt[bot], 0.0)
		stalled := engine.ChooseFloat(spawnExitProgressAt[bot] > 0.0, now-spawnExitProgressAt[bot], 0.0)
		moved := engine.ChooseFloat(spawnExitProgressAt[bot] > 0.0, engine.VectorDistance(engine.AbsOriginOf(bot), spawnExitProgress[bot]), 0.0)

		var anchorSource engine.Text
		anchorNav := FindSpawnRecoveryArea(bot, anchorSource, 32) != engine.NullArea()

		engine.ReplyToCommand(client, "%N: strict %d, spawn distance %.0f, near %d, eligible %d, upgrade zone %d, watched %.1fs, stalled %.1fs, moved %.0f, anchor %s, anchor NAV %d",
			bot, strict, distance, near, ShouldWatchDefenderSpawnExit(bot), engine.IsInUpgradeZone(bot), watched, stalled, moved, anchorSource, anchorNav)
	}

	return engine.PluginHandled()
}

// CommandRecoverSpawnBots moves every stuck defender at once, for an admin who
// can see the problem and does not want to wait for the watch.
//
//sp:name Command_RecoverSpawnBots
//sp:public
//nolint:revive // unused-parameter: the argument count is the console's, and this command takes none
func CommandRecoverSpawnBots(client int32, args int32) engine.Outcome {
	var recovered int32

	for bot := int32(1); bot <= engine.MaxClients(); bot++ {
		if !engine.IsClientInGame(bot) || !engine.IsPlayerAlive(bot) || !engine.DefenderBotFlag(bot) || !IsInOrNearDefenderSpawn(bot) {
			continue
		}

		if MoveDefenderFromSpawnToBattlefield(bot, "was manually recovered by an admin") {
			recovered++
		}
	}

	engine.ReplyToCommand(client, "Recovered %d defender bot(s) from the configured spawn radius.", recovered)
	return engine.PluginHandled()
}
