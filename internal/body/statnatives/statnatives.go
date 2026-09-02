/*
Package statnatives is the six natives source/tf2_defenderbots.sp exports to the
statistics plugin.

Every one of them reads one thing about one bot. The guard is the same in all
six because the caller is another plugin: a client index from outside is not to
be trusted, and a bot that has left is not ours to answer about.
*/
package statnatives

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
The guard is written out six times, once per native, rather than shared.

Sharing it reads better and costs the proof: the comparison walks the call
sequence of each body against the shipped one, and a body whose whole content is
one call to a helper has nothing left to compare. Six copies of four tests is
what the plugin wrote and what can be checked.
*/

// NativeGetPathLength is how long the bot's current route is.
//
//sp:name Native_GetPathLength
//nolint:revive // unused-parameter: SourceMod hands every native the plugin and the count
func NativeGetPathLength(plugin engine.Timer, numParams int32) engine.Cell {
	client := engine.NativeCell(1)

	if client < 1 || client > engine.MaxClients() || !engine.IsClientInGame(client) || !engine.DefenderBotFlag(client) {
		return engine.Cell(-1)
	}

	// Unguarded, as everywhere else that reads it: the path is made with the
	// bot and outlives it.
	return engine.CellOfFloat(engine.PathOf(client).Length())
}

/*
NativePathFailed says the last computation came back with nothing, which the
length cannot tell anybody.

A refused computation leaves the path object holding whatever it held before, so
GetLength keeps returning the old answer and a failing bot reads as a bot with a
perfectly good path. Measured on Decoy: the medic reported a path 10400 units
long, constant to within fifty units over eighty seconds, while the nearest
teammate stood four hundred units away. Every one of those samples was a failure
wearing the length of the last success.
*/
//
//sp:name Native_PathFailed
//
//nolint:revive // unused-parameter: SourceMod hands every native the plugin and the count
func NativePathFailed(plugin engine.Timer, numParams int32) engine.Cell {
	client := engine.NativeCell(1)

	if client < 1 || client > engine.MaxClients() || !engine.IsClientInGame(client) || !engine.DefenderBotFlag(client) {
		return engine.Cell(0)
	}

	return engine.CellOfBool(engine.PathFailedFor(client))
}

// NativePathFailures is how many times in a row it has.
//
//sp:name Native_PathFailures
//nolint:revive // unused-parameter: SourceMod hands every native the plugin and the count
func NativePathFailures(plugin engine.Timer, numParams int32) engine.Cell {
	client := engine.NativeCell(1)

	if client < 1 || client > engine.MaxClients() || !engine.IsClientInGame(client) || !engine.DefenderBotFlag(client) {
		return engine.Cell(-1)
	}

	return engine.Cell(engine.PathFailuresOf(client))
}

/*
NativeRangeRepairStalls is bolts fired at a sentry that gained nothing for three
seconds, counted rather than sampled.

The state is rare enough that a five second sampler saw it zero times in a
hundred and thirty seven engineer samples. A counter does not care how rare it
is.
*/
//
//sp:name Native_RangeRepairStalls
//
//nolint:revive // unused-parameter: SourceMod hands every native the plugin and the count
func NativeRangeRepairStalls(plugin engine.Timer, numParams int32) engine.Cell {
	client := engine.NativeCell(1)

	if client < 1 || client > engine.MaxClients() || !engine.IsClientInGame(client) || !engine.DefenderBotFlag(client) {
		return engine.Cell(-1)
	}

	return engine.Cell(engine.RangeRepairStallsOf(client))
}

/*
NativeGetAttackTarget is who this bot decided to shoot, which is the decision
rather than where the crosshair happens to be pointing.

A wave that wipes the team is usually one robot nobody chose.
*/
//
//sp:name Native_GetAttackTarget
//
//nolint:revive // unused-parameter: SourceMod hands every native the plugin and the count
func NativeGetAttackTarget(plugin engine.Timer, numParams int32) engine.Cell {
	client := engine.NativeCell(1)

	if client < 1 || client > engine.MaxClients() || !engine.IsClientInGame(client) || !engine.DefenderBotFlag(client) {
		return engine.Cell(-1)
	}

	return engine.Cell(engine.AttackTargetOf(client))
}

// NativeIsPathing says the bot is walking somewhere under the mod's own
// pathing rather than the game's.
//
//sp:name Native_IsPathing
//nolint:revive // unused-parameter: SourceMod hands every native the plugin and the count
func NativeIsPathing(plugin engine.Timer, numParams int32) engine.Cell {
	client := engine.NativeCell(1)

	if client < 1 || client > engine.MaxClients() || !engine.IsClientInGame(client) || !engine.DefenderBotFlag(client) {
		return engine.Cell(0)
	}

	return engine.CellOfBool(engine.PluginBotOf(client).Pathing())
}
