/*
Package buildings is the part of source/redbots3/util.sp that answers what an
engineer has standing, and takes one down.

Every build behaviour asks these five questions and they were the last of the
plugin's own that the ported ones reached through an extern. They are here now, so
the answer and the callers are the same code.
*/
package buildings

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// More than an engineer is supposed to own, which is the point: he is not
// supposed to and he does.
//
//sp:name MAX_PLAYER_OBJECTS
const maxPlayerObjects = 8

/*
IsBuilderSetTo is whether the toolbox in his hands is set to build the thing this
action came here to build.

Every build action used to ask only whether he was holding the toolbox at all, and
the toolbox remembers what it was last told to make. So an engineer walking from one
build straight into the next never re-issued the command, and pressed fire on a
toolbox still set to the last job.

Measured on Coaltown: he finishes the dispenser at his nest, walks to the spawn to
put down the teleporter entrance, and the entrance never happens because the toolbox
is still set to dispenser. What goes down at the spawn is a second dispenser, which
is both the "dispenser right beside the teleporter entrance" and the "two dispensers
for one engineer" from play.
*/
//
//sp:name IsBuilderSetTo
//sp:default mode TFObjectMode_None
func IsBuilderSetTo(client int32, objectType engine.Object, mode engine.ObjectMode) bool {
	weapon := engine.ActiveWeapon(client)

	if weapon < 1 || engine.WeaponID(weapon) != engine.WeaponBuilder() {
		return false
	}

	if engine.EntProp(weapon, engine.PropSend(), "m_iObjectType") != int32(objectType) {
		return false
	}

	// Only the teleporter has two of them, and putting an entrance down for an exit is the same bug
	if objectType == engine.ObjectTeleporter() && engine.EntProp(weapon, engine.PropSend(), "m_iObjectMode") != int32(mode) {
		return false
	}

	return true
}

/*
PlayerObjectCount is how many buildings this player owns, and none for a player who
has left.

TF2Util_GetPlayerObjectCount throws on a client that is not in game, and a thrown
native takes the whole callback with it. An action's OnEnd is the one place that
reliably asks about a bot after he has gone: the seat refill kicks bots between
waves, which ends their actions, and k-kaneta's log of 2026-08-27 has the trace
twice on Mannworks with the engineer's sentry action named in it.

Zero rather than a refusal, because every caller loops over the answer and a player
with no buildings is the truth about a player who is not there.
*/
//
//sp:name PlayerObjectCount
func PlayerObjectCount(client int32) int32 {
	if client <= 0 || client > engine.MaxClients() || !engine.IsClientInGame(client) {
		return 0
	}

	return engine.PlayerObjectCountRaw(client)
}

/*
DetonateObjectOfType takes down every building of the type, not the first one found.

An engineer is not meant to be able to hold two dispensers, and one was measured
holding two on Coaltown: the working one at his nest, and a second at the spawn a
teleporter's width from his entrance. Taking down "the" dispenser between waves took
down whichever came first in his object list, so the other one outlived it, and then
outlived every wave after that. The nest was rebuilt each break and the stray never
was. Reported as two dispensers for one engineer.

Collected before any of them is detonated, because detonating edits the list being
walked.
*/
//
//sp:name DetonateObjectOfType
//sp:default mode TFObjectMode_None
//sp:default ignoreSapperState false
func DetonateObjectOfType(client int32, objectType engine.Object, mode engine.ObjectMode, ignoreSapperState bool) {
	var found [maxPlayerObjects]int32

	count := int32(0)

	numObjects := PlayerObjectCount(client)

	for i := int32(0); i < numObjects && count < maxPlayerObjects; i++ {
		obj := engine.PlayerObject(client, i)

		if engine.ObjectType(obj) != objectType {
			continue
		}

		if objectType == engine.ObjectTeleporter() && engine.ModeOf(obj) != mode {
			continue
		}

		if engine.IsDisposableBuilding(obj) {
			continue
		}

		if !ignoreSapperState && (engine.HasSapper(obj) || engine.IsPlasmaDisabled(obj)) {
			continue
		}

		found[count] = engine.EntIndexToEntRef(obj)
		count++
	}

	for i := int32(0); i < count; i++ {
		obj := engine.EntRefToEntIndex(found[i])

		if obj == engine.InvalidEntReference() || !engine.IsValidEntity(obj) {
			continue
		}

		event := engine.CreateEvent("object_removed")

		if event != engine.NoEvent() {
			event.SetEventInt("userid", engine.ClientUserID(client))
			event.SetEventInt("objecttype", int32(objectType))
			event.SetEventInt("index", obj)
			event.Fire()
		}

		engine.DetonateObject(obj)
	}
}

/*
HasObjectOfType is a building of this type he owns, counting the one in his hands.

The game takes a building out of the player's object list the moment he picks it up,
so every question the mod asks about what an engineer has answered no while he was
carrying it. What follows from that is a second one: the dispenser gate sees none,
sends him to build, and when he finally puts the carried one down there are two
dispensers and one engineer. Reported from play with a photograph.

The carried one is his by any reading of the question, so it is counted here rather
than at each of the twenty call sites that ask.
*/
//
//sp:name HasObjectOfType
//sp:default mode TFObjectMode_None
func HasObjectOfType(client int32, objectType engine.Object, mode engine.ObjectMode) int32 {
	standing := GetObjectOfType(client, objectType, mode)

	if standing != engine.InvalidEntReference() {
		return standing
	}

	carried := engine.CarriedObject(client)

	if carried == -1 || !engine.IsValidEntity(carried) {
		return engine.InvalidEntReference()
	}

	if engine.ObjectType(carried) != objectType {
		return engine.InvalidEntReference()
	}

	if objectType == engine.ObjectTeleporter() && engine.ModeOf(carried) != mode {
		return engine.InvalidEntReference()
	}

	return carried
}

// GetObjectOfType is the first building of that type standing, walking past the
// disposable ones on purpose.
//
//sp:name GetObjectOfType
//sp:default mode TFObjectMode_None
func GetObjectOfType(client int32, objectType engine.Object, mode engine.ObjectMode) int32 {
	numObjects := PlayerObjectCount(client)

	for i := int32(0); i < numObjects; i++ {
		obj := engine.PlayerObject(client, i)

		if engine.ObjectType(obj) != objectType {
			continue
		}

		if objectType == engine.ObjectTeleporter() && engine.ModeOf(obj) != mode {
			continue
		}

		if engine.IsDisposableBuilding(obj) {
			continue
		}

		return obj
	}

	return -1
}

/*
LogBuildFailure says why a build ended without a building, out loud.

A nest that is standing for two fifths of a wave is the engineer's whole problem,
and it was invisible: every build action has half a dozen ways to end and none of
them left a trace, so "he never built one" and "he built three and lost three" and
"he gave up after twelve seconds" all looked the same from a results file.

Printed rather than counted, because the interesting thing is the sequence: which
reason, in which order, at what point in the wave.
*/
//
//sp:name LogBuildFailure
func LogBuildFailure(actor int32, what string, why string) {
	if actor < 1 || actor > engine.MaxClients() || !engine.IsClientInGame(actor) {
		return
	}

	engine.PrintToServer("[defenderbots] %s failed for %N at %.1f: %s", what, actor, engine.GameTime(), why)

	// The console is a stream nobody can count per run; the log is a file with the run in it
	engine.LogMessage("Build: %s for %N at %.1f: %s", what, actor, engine.GameTime(), why)
}
