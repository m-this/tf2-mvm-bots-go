/*
Package humans is the part of source/tf2_defenderbots.sp that asks about the
people: whether any are on RED, whether one of them has pressed ready, and how
fast a bot may type on their behalf.
*/
package humans

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

/*
The readiness answer, kept for a frame.

Read once per bot per frame, answered once per frame: the roster cannot change
between two bots of the same tick, and a walk of every client slot inside
something that reads like a cheap question is how four of this mod's per-frame
costs got there.
*/

//sp:name m_flHumanReadinessTime
var humanReadinessTime float32 = -1.0

//sp:name m_bHumansOnRed
var humansOnRed bool

//sp:name m_bAnyHumanReadyOnRed
var anyHumanReadyOnRed bool

// RefreshHumanReadiness walks the slots once a frame and remembers what it saw.
//
//sp:name RefreshHumanReadiness
func RefreshHumanReadiness() {
	if humanReadinessTime == engine.GameTime() {
		return
	}

	humanReadinessTime = engine.GameTime()
	humansOnRed = false
	anyHumanReadyOnRed = false

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if !engine.IsClientInGame(i) || engine.IsFakeClient(i) || engine.ClientTeam(i) != engine.TeamRed() {
			continue
		}

		humansOnRed = true

		if engine.IsPlayerReady(i) {
			anyHumanReadyOnRed = true
			break
		}
	}
}

// AnyHumanOnRed says a person is on the defending team.
//
//sp:name AnyHumanOnRed
func AnyHumanOnRed() bool {
	RefreshHumanReadiness()

	return humansOnRed
}

// AnyHumanReadyOnRed says one of them has pressed ready.
//
//sp:name AnyHumanReadyOnRed
func AnyHumanReadyOnRed() bool {
	RefreshHumanReadiness()

	return anyHumanReadyOnRed
}

/*
The command throttle.

A bot that types every frame is a bot the server rate-limits into silence, so
every console command a bot sends goes through one of these two, and a seat
starts with its clock at now.
*/

//sp:name m_flLastCommandTime
var lastCommandTime [slots.Count]float32

// ResetCommandThrottle starts a seat's clock, which the plugin does as a player
// is put in the server.
func ResetCommandThrottle(client int32) {
	lastCommandTime[client] = engine.GameTime()
}

// FakeClientCommandThrottled sends the command unless one went out too
// recently, and says whether it went.
//
//sp:name FakeClientCommandThrottled
func FakeClientCommandThrottled(client int32, command string) bool {
	if lastCommandTime[client] > engine.GameTime() {
		return false
	}

	engine.FakeClientCommandText(client, command)

	lastCommandTime[client] = engine.GameTime() + 0.4

	return true
}

// ShouldProcessCommand is the same gate for a command the plugin runs itself
// rather than sends.
//
//sp:name ShouldProcessCommand
func ShouldProcessCommand(client int32) bool {
	if lastCommandTime[client] > engine.GameTime() {
		return false
	}

	lastCommandTime[client] = engine.GameTime() + engine.CommandMaxRate()
	return true
}
