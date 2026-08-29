/*
Package engine is what a generated body calls when it calls the engine.

One Go function per engine call, each carrying the directive that says how the
call is written in SourcePawn. internal/spbody reads those directives, so the
Go a body compiles against and the SourcePawn it emits come from the same
declaration and cannot drift apart.

Three kinds of call, and the directive names which:

	//sp:native NAME     a SourceMod native, called by name
	//sp:sdkcall HANDLE  SDKCall through a handle prepared at load
	//sp:address NAME    a read through a raw address

In a Go process none of them mean anything, so the default answer to all of them
is a panic. Install puts a set of answers behind them, which is what the
differential test does: the same canned answers on both sides, and the two call
traces have to match.
*/
package engine

import "fmt"

// Calls is the set of answers a body gets. A nil field is a call the caller did
// not expect the body to make, and reaching it is a failed expectation rather
// than a zero value quietly standing in for one.
type Calls struct {
	IsClientInGame  func(client int32) bool
	IsPlayerAlive   func(client int32) bool
	GetClientTeam   func(client int32) int32
	HasAmmo         func(weapon int32) bool
	Clip1           func(weapon int32) int32
	IsDefenderBot   func(client int32) bool
	SetTouchCredits func(touching bool)
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

// The two below stand in for plugin globals, g_bIsDefenderBot and
// m_bTouchCredits. A generated body owns no state, so the state stays in the
// hand-written SourcePawn and is reached the same way an engine call is. When
// package-level state moves here too they become ordinary variables.

// IsDefenderBot says whether the client is one of ours.
//
//sp:native IsDefenderBot
func IsDefenderBot(client int32) bool {
	if installed.IsDefenderBot == nil {
		missing("IsDefenderBot")
	}
	return installed.IsDefenderBot(client)
}

// SetTouchCredits says a defender bot is in the middle of picking up credits.
//
//sp:native SetTouchCredits
func SetTouchCredits(touching bool) {
	if installed.SetTouchCredits == nil {
		missing("SetTouchCredits")
	}
	installed.SetTouchCredits(touching)
}
