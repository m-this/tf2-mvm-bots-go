/*
Package stuckwatch is the watchdog of source/redbots3/nextbot_behavior.sp: the
one place a wedged bot, an idle bot and a stalled sniper are all seen from,
because it is the one place a behaviour reset cannot reach.

Anything armed inside an action is armed again when the reset restarts it; two
attempts at the wedge fix were defeated that way.
*/
package stuckwatch

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

// StuckRadius is how far a bot must move for the watchdog to call it moving.
//
//sp:name STUCK_RADIUS
const StuckRadius = 72.0

// StuckTime is how long inside that radius arms it.
//
//sp:name STUCK_TIME
const StuckTime = 12.0

// StuckWedgeGiveup is stucks at one spot before the bot is moved off it.
//
//sp:name STUCK_WEDGE_GIVEUP
const StuckWedgeGiveup = 3

// StuckWedgeSearch is how far to look for somewhere it can stand.
//
//sp:name STUCK_WEDGE_SEARCH
const StuckWedgeSearch = 400.0

// SniperAtSpot is near enough to a spot that standing still is his job.
//
//sp:name SNIPER_AT_SPOT
const SniperAtSpot = 400.0

// SniperStallTime is how long a rifle sniper may go without either a lurk or a
// spot before he is restarted.
//
//sp:name SNIPER_STALL_TIME
const SniperStallTime = 20.0

//sp:name m_vStuckOrigin
var stuckOrigin [slots.Count][3]float32

//sp:name m_ctStuckDeadline
var stuckDeadline [slots.Count]float32

//sp:name m_iStuckCount
var stuckCount [slots.Count]int32

// Where a bot kept getting stuck, and how many times running, so a wedge is
// told from a slow walk.
//
//sp:name m_vStuckWedge
var stuckWedge [slots.Count][3]float32

//sp:name m_iStuckWedgeCount
var stuckWedgeCount [slots.Count]int32

//sp:name m_ctSniperStallDeadline
var sniperStallDeadline [slots.Count]float32

/*
Set once the stall is called, and never cleared while he is on this team.

Resetting his intention is not enough on its own: the rescue fired on a sniper
nobody had walked into, and he stayed in spawn. So the mark outlives the reset
and the break reads it, which is the only place a bot can be handed an action.
*/
//
//sp:name m_bSniperStalled
var sniperStalled [slots.Count]bool

// StuckCountOf is how many times the watchdog has caught this bot.
//
//sp:name StuckCountOf
func StuckCountOf(client int32) int32 {
	return stuckCount[client]
}

// FrameUnstickDefender is the rescue, a frame later than the catch.
//
//sp:name Frame_UnstickDefender
//sp:public
func FrameUnstickDefender(client engine.Cell) {
	if !engine.IsClientInGame(int32(client)) || !engine.DefenderBotFlag(int32(client)) || !engine.IsPlayerAlive(int32(client)) {
		return
	}

	engine.ResetIntentionInterface(int32(client))
}

// IsSniperStalled says this sniper has been caught stalling and should be sent
// to the front like every other class.
//
//sp:name IsSniperStalled
func IsSniperStalled(client int32) bool {
	return sniperStalled[client]
}

// ClearSniperStall is the break reading the mark: the bot is being handed an
// action, so the stall is over.
//
//sp:name ClearSniperStall
func ClearSniperStall(client int32) {
	sniperStalled[client] = false
	sniperStallDeadline[client] = 0.0
}

/*
IsLurkingNowhere is a sniper who is nowhere near a spot and not on his way to
one.

The lurk is not required, in either direction: a rifle sniper parked far from
every spot is the fault whether ScenarioMonitor gave him a lurk that cannot
finish or never gave him one. The reset is what both need. See mvm-bj8.
*/
//
//sp:name IsLurkingNowhere
//sp:const here
func IsLurkingNowhere(actor int32, actions engine.Text, here [3]float32) bool {
	if engine.PlayerClass(actor) != engine.ClassSniper() || !engine.HasSniperRifle(actor) {
		return false
	}

	// A spot he is walking to is a spot he has not reached, and the walk is
	// not the fault.
	if engine.PluginBotOf(actor).Pathing() {
		return false
	}

	// A lurk on the stack is the game doing its job, however far off he still
	// is.
	if engine.StrContains(actions, "SniperLurk", true) != -1 {
		return false
	}

	spots := engine.SniperSpots()

	for i := int32(0); i < spots.Length(); i++ {
		spot := spots.GetArray(i)

		if engine.VectorDistance(here, spot) <= SniperAtSpot {
			return false
		}
	}

	return true
}

/*
UpdateStuckWatchdog is the whole watch, run per bot per think.

A bot with nothing on its stack is stuck too: he is not pathing, so nothing else
arms, and a bot that has stopped asking to go anywhere is exactly the one nobody
is going to rescue.

A sniper's stall is timed on its own, because the watchdog's timer can be reset
by a shove: teammates walking through a parked sniper push him further than
STUCK_RADIUS, so this is timed from the last moment he was doing his job rather
than the last moment he moved.

Stuck again without having moved is a bot wedged rather than a bot walking
slowly: resetting his behaviour does not move him, so he comes back to the same
wedge and asks for another path, which is the frame that grows. Past the giveup
he is teleported; a bot that cannot be moved is the one that kills the server.
*/
//
//sp:name UpdateStuckWatchdog
func UpdateStuckWatchdog(actor int32) {
	myBot := engine.NextBotOf(actor)
	myLoco := myBot.Locomotion()

	actions := engine.ActionStackOf(actor)

	noBehaviour := engine.Feature(engine.FeatureWatchIdleBots()) && actions[0] == 0

	here := engine.AbsOriginOf(actor)

	if !IsLurkingNowhere(actor, actions, here) {
		sniperStallDeadline[actor] = engine.GameTime() + SniperStallTime
	} else if engine.Feature(engine.FeatureWatchLurkingSnipers()) && engine.GameTime() >= sniperStallDeadline[actor] {
		sniperStallDeadline[actor] = engine.GameTime() + SniperStallTime
		sniperStalled[actor] = true
		stuckCount[actor]++

		engine.LogMessage("Stalled: %N (sniper) at %.0f %.0f %.0f for %.0fs, stall #%d, %s",
			actor, here[0], here[1], here[2], SniperStallTime, stuckCount[actor],
			engine.ChooseText(actions[0] == 0, "no behaviour", actions))

		engine.ApplyNextFrameCell(FrameUnstickDefender, actor)

		return
	}

	lurkingNowhere := false

	wantsToBeElsewhere := engine.PluginBotOf(actor).Pathing() || myLoco.IsStuck() || noBehaviour || lurkingNowhere

	if !wantsToBeElsewhere || engine.VectorDistance(here, stuckOrigin[actor]) > StuckRadius {
		stuckOrigin[actor] = here
		stuckDeadline[actor] = engine.GameTime() + StuckTime

		return
	}

	if engine.GameTime() < stuckDeadline[actor] {
		return
	}

	if engine.VectorDistance(here, stuckWedge[actor]) <= StuckRadius {
		stuckWedgeCount[actor]++
	} else {
		stuckWedge[actor] = here
		stuckWedgeCount[actor] = 1
	}

	stuckOrigin[actor] = here
	stuckDeadline[actor] = engine.GameTime() + StuckTime
	stuckCount[actor]++

	myLoco.ClearStuckStatus("Watchdog")
	engine.PluginBotOf(actor).SetPathing(false)

	engine.PrintToServer("[defenderbots] stuck: %N (%s) at %.0f %.0f %.0f for %.0fs, stuck #%d, wedge #%d, %s",
		actor, engine.RawClassName(engine.PlayerClass(actor)),
		here[0], here[1], here[2], StuckTime, stuckCount[actor], stuckWedgeCount[actor],
		engine.ChooseText(actions[0] == 0, "no behaviour", actions))

	// The same line in the file, so a run can be counted rather than watched.
	engine.LogMessage("Stuck: %N (%s) at %.0f %.0f %.0f, stuck #%d, wedge #%d, %s",
		actor, engine.RawClassName(engine.PlayerClass(actor)),
		here[0], here[1], here[2], stuckCount[actor], stuckWedgeCount[actor],
		engine.ChooseText(actions[0] == 0, "no behaviour", actions))

	if stuckWedgeCount[actor] >= StuckWedgeGiveup && engine.MoveWedgedDefender(actor) {
		return
	}

	engine.ApplyNextFrameCell(FrameUnstickDefender, actor)
}

// MoveWedgedTries is how many random points are tried per area before giving
// up on it.
//
//sp:name MOVE_WEDGED_TRIES
const MoveWedgedTries = 8

// AreaEscapePoint is a point in the area far enough from the wedge to be worth
// standing on.
//
//sp:name AreaEscapePoint
//sp:const here
func AreaEscapePoint(area engine.Area, here [3]float32) (found bool, destination [3]float32) {
	for attempt := int32(0); attempt < MoveWedgedTries; attempt++ {
		point := engine.RandomPointIn(area)
		point[2] += 10.0

		if engine.VectorDistance(here, point) > StuckRadius {
			destination = point
			return true, destination
		}
	}

	return false, destination
}

// WedgeEscapePoint tries the wedge's own area first and then everything
// touching it.
//
//sp:name WedgeEscapePoint
//sp:const here
func WedgeEscapePoint(area engine.Area, here [3]float32) (found bool, destination [3]float32) {
	found, destination = AreaEscapePoint(area, here)

	if found {
		return true, destination
	}

	for dir := engine.DirectionNorth(); dir < engine.DirectionCount(); dir++ {
		count := area.AdjacentCount(dir)

		for i := int32(0); i < count; i++ {
			next := area.AdjacentArea(dir, i)

			if next != engine.NullArea() {
				found, destination = AreaEscapePoint(next, here)

				if found {
					return true, destination
				}
			}
		}
	}

	return false, destination
}

/*
MoveWedgedDefender teleports a bot off ground it cannot leave on its own, to a
point in its area or a touching one that is far enough away to be different
ground.
*/
//
//sp:name MoveWedgedDefender
func MoveWedgedDefender(client int32) bool {
	here := engine.AbsOriginOf(client)

	area := engine.NearestNavArea(here, true, StuckWedgeSearch, false, true, engine.TeamAny())

	if area == engine.NullArea() {
		engine.LogMessage("Stuck: %N is wedged at %.0f %.0f %.0f with no nav area within %.0f, so nothing can be done",
			client, here[0], here[1], here[2], StuckWedgeSearch)
		return false
	}

	var destination [3]float32

	if engine.OldWedgeRecovery() {
		// The pre-2.21.3 behaviour, kept only so a run can measure what
		// replacing it was worth.
		destination = engine.RandomPointIn(area)
		destination[2] += 10.0

		if engine.VectorDistance(here, destination) <= StuckRadius {
			return false
		}
	} else {
		found, escape := WedgeEscapePoint(area, here)

		if !found {
			engine.LogMessage("Stuck: %N is wedged at %.0f %.0f %.0f and every point in its area and the ones touching it is too close",
				client, here[0], here[1], here[2])
			return false
		}

		destination = escape
	}

	var stopped [3]float32
	engine.TeleportEntity(client, destination, engine.NullVector(), stopped)
	engine.CombatOf(client).UpdateLastKnownArea()

	engine.SetRepathTime(client, 0.0)
	stuckWedgeCount[client] = 0

	engine.LogMessage("Stuck: %N was wedged at %.0f %.0f %.0f, moved to %.0f %.0f %.0f",
		client, here[0], here[1], here[2], destination[0], destination[1], destination[2])

	return true
}
