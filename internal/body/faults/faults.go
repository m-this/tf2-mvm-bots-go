/*
Package faults is source/redbots3/debug_faults.sp.

Make a fault happen on purpose, so a fix for it can be measured.

Three engineer fixes have shipped against faults the test-bed will not produce:
the setup freeze, the refused path to a metal pack and the wedge. Each was
measured against a condition that did not happen, so each arm ran the same code
and the run said nothing. See mvm-0lo.

These convars force the condition instead of waiting for it. They are for a run,
not for a server: each is off at zero and does nothing until somebody sets it.

Wedging is done by putting the bot back where it was every frame rather than by
freezing it. A bot held in place still asks for paths, still runs its actions and
still fails to arrive, which is what the real wedge looks like from the watchdog's
side. Freezing the entity would be a different bug.
*/
package faults

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

// Far enough that only a teleport explains it. A bot shuffling inside the hold is still held.
//
//sp:name DEBUG_WEDGE_ESCAPED
const wedgeEscaped = 120.0

var (
	//sp:name redbots_debug_wedge_seconds
	wedgeSeconds engine.ConVar
	//sp:name redbots_debug_wedge_class
	wedgeClass engine.ConVar
	//sp:name redbots_debug_refuse_ammo_paths
	refuseAmmoPaths engine.ConVar
	//sp:name redbots_debug_old_wedge_recovery
	oldWedgeRecovery engine.ConVar
	//sp:name redbots_debug_unreachable_goal
	unreachableGoal engine.ConVar
	//sp:name redbots_debug_trace_snipers
	traceSnipers engine.ConVar
	//sp:name redbots_debug_empty_stack
	emptyStack engine.ConVar
)

// The bot being held, and until when. One at a time: two wedged bots is a different test.
var (
	//sp:name m_iWedgedBot
	wedgedBot int32 = -1
	//sp:name m_flWedgedUntil
	wedgedUntil float32
	//sp:name m_vWedgedAt
	wedgedAt [3]float32
)

// Init makes the convars, the command and the trace timer.
//
//sp:name DebugFaults_Init
func Init() {
	wedgeSeconds = engine.CreateAssistConVar("sm_redbots_debug_wedge_seconds", "0",
		"Hold one defender in place for this many seconds after a wave starts, to exercise the stuck watchdog. 0 is off.",
		engine.FCVarNotify(), true, 0.0, true, 300.0)

	wedgeClass = engine.CreatePlainConVar("sm_redbots_debug_wedge_class", "engineer",
		"Which class to hold when sm_redbots_debug_wedge_seconds is on.", engine.FCVarNotify())

	refuseAmmoPaths = engine.CreateAssistConVar("sm_redbots_debug_refuse_ammo_paths", "0",
		"Refuse this many path answers to a metal pack per bot, to exercise the ammo failover. 0 is off.",
		engine.FCVarNotify(), true, 0.0, true, 20.0)

	unreachableGoal = engine.CreateAssistConVar("sm_redbots_debug_unreachable_goal", "0",
		"Send the held bot at a point off the nav mesh, so every path search walks the whole thing and finds nothing. 0 is off.",
		engine.FCVarNotify(), true, 0.0, true, 1.0)

	oldWedgeRecovery = engine.CreateAssistConVar("sm_redbots_debug_old_wedge_recovery", "0",
		"Use the pre-2.21.3 wedge recovery, which only ever tried the area the bot stands in. For measuring what that fix is worth.",
		engine.FCVarNotify(), true, 0.0, true, 1.0)

	emptyStack = engine.CreateAssistConVar("sm_redbots_debug_empty_stack", "0",
		"Leave one defender with no behaviour at all for this many seconds after a wave starts, to exercise the idle watchdog. 0 is off.",
		engine.FCVarNotify(), true, 0.0, true, 300.0)

	traceSnipers = engine.CreateAssistConVar("sm_redbots_debug_trace_snipers", "0",
		"Write every sniper's action stack and position to the console each tenth of a second, to read back after a watchdog trip. 0 is off.",
		engine.FCVarNotify(), true, 0.0, true, 1.0)

	engine.RegServerCmd("sm_redbots_debug_sniper_spots", CommandSniperSpots)

	engine.CreateTimer(0.1, TraceSnipers, engine.Default(), engine.TimerRepeat()|engine.TimerNoMapChange())
}

/*
TraceSnipers says what each sniper was doing, tick by tick, so the frame that
hangs can be read back.

Three fixes for the sniper crash were written from the core alone and all three
failed a measurement, the last one still tripping the watchdog with
CTFBotSniperLurk refused outright. So the search that runs away is not the one the
core's top frame suggested. The last line printed before WatchDog! names the action
that was actually running. See mvm-bj8.
*/
//
//sp:name Timer_TraceSnipers
//sp:public
//
//nolint:revive // unused-parameter: the signature is the timer's, not ours
func TraceSnipers(timer engine.Timer) engine.Outcome {
	if traceSnipers == engine.NoConVar() || !traceSnipers.Bool() {
		return engine.PluginContinue()
	}

	for client := int32(1); client <= engine.MaxClients(); client++ {
		if !engine.IsClientInGame(client) || !engine.IsPlayerAlive(client) {
			continue
		}

		if !engine.DefenderBotFlag(client) || engine.PlayerClass(client) != engine.ClassSniper() {
			continue
		}

		actions := engine.ActionStackOf(client)
		here := engine.AbsOriginOf(client)

		engine.PrintToServer("[snipertrace] %.2f bot %d at %.0f %.0f %.0f doing %s",
			engine.GameTime(), client, here[0], here[1], here[2], actions)
	}

	return engine.PluginContinue()
}

// How many refusals are still owed to this bot, counted down as they are handed out
//
//sp:name m_iAmmoRefusalsLeft
var ammoRefusalsLeft [slots.Count]int32

// OnAmmoWalkStart starts the count again, or one bot spends the whole wave refused.
//
//sp:name DebugFaults_OnAmmoWalkStart
func OnAmmoWalkStart(client int32) {
	ammoRefusalsLeft[client] = refuseAmmoPaths.Int()
}

/*
RefuseAmmoPath is whether this path answer should be a refusal.

Sits in front of PathFailedFor rather than replacing it, so a route that really
failed still reads as failed once the owed refusals run out.
*/
//
//sp:name DebugFaults_RefuseAmmoPath
func RefuseAmmoPath(client int32) bool {
	if ammoRefusalsLeft[client] <= 0 {
		return false
	}

	ammoRefusalsLeft[client]--

	return true
}

// OnWaveStart picks somebody to hold. Nothing happens while the convar is zero.
//
//sp:name DebugFaults_OnWaveStart
func OnWaveStart() {
	wedgedBot = -1

	seconds := wedgeSeconds.Float()

	if seconds <= 0.0 {
		return
	}

	wanted := wedgeClass.StringValue()

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if !engine.IsClientInGame(i) || !engine.IsDefenderBot(i) || !engine.IsPlayerAlive(i) {
			continue
		}

		if !engine.StrEqualFold(engine.RawClassName(engine.PlayerClass(i)), wanted, false) {
			continue
		}

		wedgedBot = i
		wedgedUntil = engine.GameTime() + seconds
		wedgedAt = engine.AbsOriginOf(i)

		engine.LogMessage("DebugFaults: holding %N (%s) at %.0f %.0f %.0f for %.0fs",
			i, wanted, wedgedAt[0], wedgedAt[1], wedgedAt[2], seconds)

		return
	}

	engine.LogMessage("DebugFaults: no %s on RED to hold", wanted)
}

/*
OnGameFrame puts the held bot back, once a frame.

Stops on its own at the deadline, and stops early if the bot died, left, or was
moved a long way: a teleport out of the hold is the watchdog's recovery working,
and continuing to drag him back would be measuring this file rather than the fix.
*/
//
//sp:name DebugFaults_OnGameFrame
func OnGameFrame() {
	if wedgedBot <= 0 {
		return
	}

	if engine.GameTime() > wedgedUntil || !engine.IsClientInGame(wedgedBot) || !engine.IsPlayerAlive(wedgedBot) {
		wedgedBot = -1
		return
	}

	here := engine.AbsOriginOf(wedgedBot)

	if engine.VectorDistance(here, wedgedAt) > wedgeEscaped {
		engine.LogMessage("DebugFaults: %N left the hold, %.0f units away, so something moved him",
			wedgedBot, engine.VectorDistance(here, wedgedAt))

		wedgedBot = -1

		return
	}

	var still [3]float32

	engine.TeleportEntity(wedgedBot, wedgedAt, engine.NullVector(), still)
}

/*
Leave one bot with no behaviour at all, which the wedge injector cannot do.

A held bot still has behaviour, it just cannot get anywhere, so the wedge exercises
the pathing half of the stuck watchdog and never the idle half. k-kaneta's frozen
Decoy engineer is the other shape: a bot that has stopped asking to go anywhere.
Six Decoy runs put engineer idle at nought per cent and the idle watchdog wrote no
line in either arm, so FEATURE_WATCH_IDLE_BOTS is neither confirmed nor refuted.
This is the fault it needs. See mvm-6rt and mvm-0lo.

Ending MainAction from its own Update rather than reaching into the stack from
outside. Done returns a result the behaviour is asking for, so it only means
anything returned from a callback, and MainAction is the one every bot has. Ending
it every update is what makes the freeze last: ending it once produces a bot that
is idle for a frame and then carries on.
*/
var (
	//sp:name m_iEmptiedBot
	emptiedBot int32 = -1
	//sp:name m_flEmptiedUntil
	emptiedUntil float32
	//sp:name m_flEmptiedSaid
	emptiedSaid float32
)

// OnWaveStartEmpty picks somebody to empty.
//
//sp:name DebugFaults_OnWaveStartEmpty
func OnWaveStartEmpty() {
	emptiedBot = -1

	seconds := emptyStack.Float()

	if seconds <= 0.0 {
		return
	}

	wanted := wedgeClass.StringValue()

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if !engine.IsClientInGame(i) || !engine.IsDefenderBot(i) || !engine.IsPlayerAlive(i) {
			continue
		}

		if !engine.StrEqualFold(engine.RawClassName(engine.PlayerClass(i)), wanted, false) {
			continue
		}

		emptiedBot = i
		emptiedUntil = engine.GameTime() + seconds

		engine.LogMessage("DebugFaults: emptying %N (%s) for %.0fs", i, wanted, seconds)

		return
	}

	engine.LogMessage("DebugFaults: no %s on RED to empty", wanted)
}

// ShouldEmpty is whether this bot's behaviour should be ended now, checked from
// MainAction's update.
//
//sp:name DebugFaults_ShouldEmpty
func ShouldEmpty(client int32) bool {
	if emptiedBot != client {
		return false
	}

	if engine.GameTime() <= emptiedUntil && engine.IsPlayerAlive(client) {
		/* Say what the stack looks like while it is being emptied

		Ending MainAction lets the intention build it again, so the stack a watchdog sampling once a
		second sees may never be the empty one. Without this line, a rescue that does not fire and a
		fault that is not there read the same. */
		if emptiedSaid <= engine.GameTime() {
			emptiedSaid = engine.GameTime() + 1.0

			actions := engine.ActionStackOf(client)

			engine.LogMessage("DebugFaults: %N holds \"%s\"", client, actions)
		}

		return true
	}

	engine.LogMessage("DebugFaults: done emptying %N", client)

	emptiedBot = -1

	return false
}

/*
OldWedgeRecovery is whether to use the recovery as it was before v2.21.3.

The old one asked TheNavMesh for the nearest area and took a random point in it.
For a bot wedged in geometry while standing on valid nav that area is the one under
his feet, so the point landed back on him and the move was thrown away. That is the
defect, and this convar is how the arms of an A/B differ: measuring what a fix is
worth needs the fault available, not only its absence.
*/
//
//sp:name DebugFaults_OldWedgeRecovery
func OldWedgeRecovery() bool {
	return oldWedgeRecovery != engine.NoConVar() && oldWedgeRecovery.Bool()
}

/*
UnreachableGoal is where to send the held bot so that no path exists.

Far above the map, which is off every nav area there is. The search has to walk the
mesh to establish that, which is the frame the watchdog kills the server on, and the
whole point of this convar is to make that frame happen rather than wait for a map
to arrange it.
*/
//
//sp:name DebugFaults_UnreachableGoal
func UnreachableGoal(client int32) (found bool, goal [3]float32) {
	if unreachableGoal == engine.NoConVar() || !unreachableGoal.Bool() {
		return false, goal
	}

	if client != wedgedBot {
		return false, goal
	}

	goal = wedgedAt
	goal[2] += 16384.0

	return true, goal
}

/*
ReportSniperSpots says whether each sniper spot can be walked to, from each sniper
standing now.

The stock sniper is handed to the game's CTFBotSniperLurk, which computes its own
path and hangs the frame the watchdog kills the server on. Whether the spots are
reachable at all decides the fix: an unreachable spot means filtering them before
committing, a reachable one means the lurk never starts. See mvm-bj8.
*/
//
//sp:name DebugFaults_ReportSniperSpots
func ReportSniperSpots() {
	spots := engine.SniperSpots()

	engine.PrintToServer("[sniperspots] %d configured", spots.Length())

	for client := int32(1); client <= engine.MaxClients(); client++ {
		if !engine.IsClientInGame(client) || !engine.IsPlayerAlive(client) {
			continue
		}

		if !engine.DefenderBotFlag(client) || engine.PlayerClass(client) != engine.ClassSniper() {
			continue
		}

		here := engine.AbsOriginOf(client)

		for i := int32(0); i < spots.Length(); i++ {
			spot := spots.GetArray(i)

			engine.PrintToServer("[sniperspots] bot %d spot %d away %.0f reachable %d rifle %d",
				client, i, engine.VectorDistance(here, spot),
				engine.IsPathToVectorPossible(client, spot), engine.HasSniperRifle(client))
		}
	}
}

// CommandSniperSpots is the console command that asks for it.
//
//sp:name Command_SniperSpots
//sp:public
//nolint:revive // unused-parameter: the signature is SourceMod's, not ours
func CommandSniperSpots(args int32) engine.Outcome {
	ReportSniperSpots()

	return engine.PluginHandled()
}
