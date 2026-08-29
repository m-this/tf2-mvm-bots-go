/*
Package engine is what a generated body calls when it calls the engine.

One Go function per engine call, each carrying the directive that says how the
call is written in SourcePawn. internal/spbody reads those directives, so the
Go a body compiles against and the SourcePawn it emits come from the same
declaration and cannot drift apart.

Three kinds of call, and the directive names which:

	//sp:native NAME     a SourceMod native, called by name
	//sp:global NAME     a SourcePawn variable, not a call at all
	//sp:plugin NAME     a plugin function the port has not reached yet
	//sp:sdkcall HANDLE  SDKCall through a handle prepared at load
	//sp:address NAME    a read through a raw address

A directive may end in the flag "returns", which says the SourcePawn declaration
returns its array rather than filling a parameter with it. That is the float[]
form, and spcomp will not assign one to a sized array, so neither will this: such
a call can only be an argument to something else.

A plugin extern is temporary by construction. It names SourcePawn this
repository is going to own, and when it does the extern goes and the call
becomes an ordinary one. internal/body refuses an extern that names a function a
body already generates, so the two can never both be there.

In a Go process none of them mean anything, so the default answer to all of them
is a panic. Install puts a set of answers behind them, which is what the
differential test does: the same canned answers on both sides, and the two call
traces have to match.
*/
package engine

import "fmt"

// Class is SourceMod's TFClassType, named here so a ported signature keeps the
// tag its callers pass rather than widening to int.
//
//sp:tag TFClassType
type Class int32

// Team is SourceMod's TFTeam, for the same reason.
//
//sp:tag TFTeam
type Team int32

// Condition is SourceMod's TFCond.
//
//sp:tag TFCond
type Condition int32

// Calls is the set of answers a body gets. A nil field is a call the caller did
// not expect the body to make, and reaching it is a failed expectation rather
// than a zero value quietly standing in for one.
type Calls struct {
	IsClientInGame func(client int32) bool
	IsPlayerAlive  func(client int32) bool
	GetClientTeam  func(client int32) int32
	HasAmmo        func(weapon int32) bool
	Clip1          func(weapon int32) int32
	Origin         func(client int32) [3]float32
	MaxClients     func() int32
	VectorDistance func(a [3]float32, b [3]float32) float32

	// The plugin's own, still hand-written SourcePawn. Each one is work the
	// port has not reached, and each goes the day it does. The TF2_ names
	// beside them are not: they come from SourceMod and from the vendored
	// stocklib include, which this repository is not going to rewrite.
	IsSentryBusterRobot    func(client int32) bool
	IsInvulnerable         func(client int32) bool
	IsStealthed            func(client int32) bool
	IsCloakedPlayerExposed func(client int32) bool
	WorldSpaceCenter       func(entity int32) [3]float32
	IsMiniBoss             func(client int32) bool
	IsPlayerInCondition    func(client int32, condition Condition) bool
	PlayerClass            func(client int32) Class
	PlayerTeam             func(client int32) Team
	PlayerEnemyTeam        func(client int32) Team
}

var installed Calls

// Install puts a set of answers behind the calls and returns the undo, so a
// test restores what it found. It is not safe to call from two tests at once,
// which is why nothing here runs in parallel.
func Install(c Calls) func() {
	previous := installed
	installed = c
	return func() { installed = previous }
}

func missing(name string) {
	panic(fmt.Sprintf("engine: %s was called and no answer is installed; this call has meaning on a game server and none here", name))
}

// IsClientInGame says whether the slot holds a client that has entered the game.
//
//sp:native IsClientInGame
func IsClientInGame(client int32) bool {
	if installed.IsClientInGame == nil {
		missing("IsClientInGame")
	}
	return installed.IsClientInGame(client)
}

// IsPlayerAlive says whether the client is alive right now.
//
//sp:native IsPlayerAlive
func IsPlayerAlive(client int32) bool {
	if installed.IsPlayerAlive == nil {
		missing("IsPlayerAlive")
	}
	return installed.IsPlayerAlive(client)
}

// GetClientTeam is the team index the client is on.
//
//sp:native GetClientTeam
func GetClientTeam(client int32) int32 {
	if installed.GetClientTeam == nil {
		missing("GetClientTeam")
	}
	return installed.GetClientTeam(client)
}

// HasAmmo says whether the weapon has anything left to fire.
//
//sp:sdkcall m_hHasAmmo
func HasAmmo(weapon int32) bool {
	if installed.HasAmmo == nil {
		missing("HasAmmo")
	}
	return installed.HasAmmo(weapon)
}

// Clip1 is what the weapon holds in its first clip.
//
//sp:sdkcall m_hClip1
func Clip1(weapon int32) int32 {
	if installed.Clip1 == nil {
		missing("Clip1")
	}
	return installed.Clip1(weapon)
}

// Origin is where the client is standing.
//
//sp:native GetClientAbsOrigin
func Origin(client int32) (origin [3]float32) {
	if installed.Origin == nil {
		missing("GetClientAbsOrigin")
	}
	return installed.Origin(client)
}

// MaxClients is the highest client slot the server has.
//
//sp:global MaxClients
func MaxClients() int32 {
	if installed.MaxClients == nil {
		missing("MaxClients")
	}
	return installed.MaxClients()
}

// VectorDistance is how far apart two points are.
//
//sp:native GetVectorDistance
func VectorDistance(a [3]float32, b [3]float32) float32 {
	if installed.VectorDistance == nil {
		missing("GetVectorDistance")
	}
	return installed.VectorDistance(a, b)
}

// IsSentryBusterRobot says whether the robot is a sentry buster, which is
// usually not a threat worth counting.
//
//sp:plugin IsSentryBusterRobot
func IsSentryBusterRobot(client int32) bool {
	if installed.IsSentryBusterRobot == nil {
		missing("IsSentryBusterRobot")
	}
	return installed.IsSentryBusterRobot(client)
}

// IsInvulnerable says whether the player cannot be hurt right now.
//
//sp:native TF2_IsInvulnerable
func IsInvulnerable(client int32) bool {
	if installed.IsInvulnerable == nil {
		missing("TF2_IsInvulnerable")
	}
	return installed.IsInvulnerable(client)
}

// IsStealthed says whether the player is cloaked.
//
//sp:native TF2_IsStealthed
func IsStealthed(client int32) bool {
	if installed.IsStealthed == nil {
		missing("TF2_IsStealthed")
	}
	return installed.IsStealthed(client)
}

// IsCloakedPlayerExposed says whether a cloaked player can be seen anyway.
//
//sp:plugin IsCloakedPlayerExposed
func IsCloakedPlayerExposed(client int32) bool {
	if installed.IsCloakedPlayerExposed == nil {
		missing("IsCloakedPlayerExposed")
	}
	return installed.IsCloakedPlayerExposed(client)
}

// WorldSpaceCenter is the middle of the entity, which is what the plugin
// measures ranges from. Its SourcePawn returns the array.
//
//sp:plugin WorldSpaceCenter returns
func WorldSpaceCenter(entity int32) [3]float32 {
	if installed.WorldSpaceCenter == nil {
		missing("WorldSpaceCenter")
	}
	return installed.WorldSpaceCenter(entity)
}

// ClassUnknown is TFClass_Unknown, the class a slot has before it picks one.
//
//sp:global TFClass_Unknown
func ClassUnknown() Class { return 0 }

// ConditionDazed is TFCond_Dazed, which is what a stun leaves behind.
//
//sp:global TFCond_Dazed
func ConditionDazed() Condition { return 17 }

// IsMiniBoss says whether the robot is a giant.
//
//sp:native TF2_IsMiniBoss
func IsMiniBoss(client int32) bool {
	if installed.IsMiniBoss == nil {
		missing("TF2_IsMiniBoss")
	}
	return installed.IsMiniBoss(client)
}

// IsPlayerInCondition says whether the player carries the condition.
//
//sp:native TF2_IsPlayerInCondition
func IsPlayerInCondition(client int32, condition Condition) bool {
	if installed.IsPlayerInCondition == nil {
		missing("TF2_IsPlayerInCondition")
	}
	return installed.IsPlayerInCondition(client, condition)
}

// PlayerClass is the class the player is playing.
//
//sp:native TF2_GetPlayerClass
func PlayerClass(client int32) Class {
	if installed.PlayerClass == nil {
		missing("TF2_GetPlayerClass")
	}
	return installed.PlayerClass(client)
}

// PlayerTeam is the team the player is on, as a tag rather than an index.
//
//sp:native TF2_GetClientTeam
func PlayerTeam(client int32) Team {
	if installed.PlayerTeam == nil {
		missing("TF2_GetClientTeam")
	}
	return installed.PlayerTeam(client)
}

// PlayerEnemyTeam is the team the player is fighting.
//
//sp:plugin GetPlayerEnemyTeam
func PlayerEnemyTeam(client int32) Team {
	if installed.PlayerEnemyTeam == nil {
		missing("GetPlayerEnemyTeam")
	}
	return installed.PlayerEnemyTeam(client)
}
