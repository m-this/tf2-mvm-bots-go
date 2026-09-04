/*
Package bluassist is source/redbots3/blu_assist.sp.

# Bend a mission by what the robots are worth

Valve tunes every wave for six defenders. Fewer than six is a harder mission than
the one the map was built for, and the answer this mod has had until now is to
fill the empty seats with bots and hope their AI is good enough. This is the other
lever: leave the seats empty and change what the robots are worth instead.

The convar is a straight multiplier on every BLU robot's maximum health. 0.5 is a
mission at half, 2.0 is one at double, and 1.0 is the lever off, which is the
default and what every existing server keeps. It reaches above 1.0 because a team
that finds the mission easy is as real as one that finds it hard, and a lever with
only one direction cannot say so.

The convar is the switch, so the two arms of an A/B are the same build with one
number different. There is no feature flag beside it: 1.0 already means off, and a
second switch that also has to be on would be two ways to say one thing. What is
set gets written into the run's results, so a file of numbers says which arm
produced it.

See docs/testbed-metrics.md: none of this ships on until a run says it helped.
*/
package bluassist

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

/*
How close to 1.0 counts as off.

The lever is a float a server operator types, and 1.0 is the only value that means
"change nothing". Comparing a float for equality would make 0.999999 a mission
nobody asked to bend, so the test has a width.
*/
//
//sp:name BLU_ASSIST_EPSILON
const epsilon = 0.0001

// The delay a robot is left to finish spawning before its health is written.
//
//sp:name BLU_ASSIST_SETTLE
const settle = 0.10

//sp:name m_iBluAssistSeen
var seen int32

// The maximum this robot is meant to have, which is what the hook answers with.
// Zero is a slot the lever is not holding, so the hook says nothing about it.
//
//sp:name m_iBluAssistMaxHealth
//sp:keep only ever read through the hook, and BluAssist_OnRobotSpawn takes the hook off every spawn before deciding whether this robot wants it back
var wantedMaxHealth [slots.Count]int32

// What the popfile had given the robot before the lever touched it, kept for the
// line that says what the lever did.
//
//sp:name m_iBluAssistWasHealth
//sp:keep read only beside the entry above, which the same spawn clears
var healthBefore [slots.Count]int32

//sp:name redbots_manager_blu_health_scale
var healthScaleConVar engine.ConVar

//sp:name redbots_manager_blu_health_debug
var healthDebugConVar engine.ConVar

// Init makes the convars and forgets the last mission's count.
//
//sp:name BluAssist_Init
func Init() {
	seen = 0

	healthScaleConVar = engine.CreateAssistConVar("sm_redbots_manager_blu_health_scale", "1.0",
		"What every BLU robot's maximum health is multiplied by. 1.0 is off.",
		engine.FCVarNotify(), true, 0.1, true, 10.0)

	healthDebugConVar = engine.CreateAssistConVar("sm_redbots_manager_blu_health_debug", "0",
		"Log the original, wanted and observed health of every robot the lever bends, rather than one in BLU_ASSIST_SAMPLE.",
		engine.FCVarNone(), true, 0.0, true, 1.0)
}

// HealthScale is what a robot's maximum health is multiplied by.
//
//sp:name BluAssist_HealthScale
func HealthScale() float32 {
	return healthScaleConVar.Float()
}

// Off says the scale changes nothing, which is the lever's default and the state
// every path here returns early on.
//
//sp:name BluAssistOff
func Off(scale float32) bool {
	return engine.FloatAbs(scale-1.0) < epsilon
}

/*
GetMaxHealth answers the game's own question about a robot's maximum.

TF2 recomputes the maximum from the class and the attributes whenever it likes, so
a number written into m_iMaxHealth does not stay written: the first version of this
bent an attribute, the second wrote the property, and a robot still spawned with
what the popfile gave it. This is the game asking, which is the one answer it does
not go back over.
*/
//
//sp:name BluAssistGetMaxHealth
//sp:byref maxHealth
//
//nolint:revive,ineffassign,staticcheck,wastedassign // the write is the point: SourcePawn passes maxHealth by reference and the hook's answer is what it leaves there
func GetMaxHealth(entity int32, maxHealth int32) engine.Outcome {
	if wantedMaxHealth[entity] <= 0 {
		return engine.PluginContinue()
	}

	maxHealth = wantedMaxHealth[entity]

	return engine.PluginChanged()
}

/*
Describe adds what the lever is set to, for the line that says what was different
about this run.

Nothing is added when it is off, which is every run until somebody sets it, so the
string reads exactly as it did before this existed.

//sp:name BluAssist_Describe
//sp:length buffer maxlength
*/
//
//nolint:revive,ineffassign,staticcheck,wastedassign // the write is the point: SourcePawn passes the buffer by reference and //sp:length carries its size
func Describe(buffer engine.Text, maxlength int32) {
	if healthScaleConVar == engine.NoConVar() || Off(healthScaleConVar.Float()) {
		return
	}

	buffer = engine.Format("blu_health=%.2f", healthScaleConVar.Float())
}

/*
OnRobotSpawn takes the last robot's answer off this slot and asks for the new one.

The unhook and the cleared slot come first, whatever the lever is set to: a slot is
reused by whoever spawns into it next, and a maximum left behind from a robot would
be answered for a human.
*/
//
//sp:name BluAssist_OnRobotSpawn
func OnRobotSpawn(client int32) {
	engine.UnhookMaxHealth(client)

	wantedMaxHealth[client] = 0
	healthBefore[client] = 0

	if Off(HealthScale()) {
		return
	}

	engine.CreateRobotTimer(settle, ApplyToRobot, engine.ClientUserID(client), engine.TimerNoMapChange())
}

/*
ApplyToRobot writes the health the lever asks for, once the popfile has finished.

The health it has now is the floor: a giant whose maximum has not been written yet
reads back the class default, and scaling that would take a 3000 health giant down
to a heavy's worth of it. Whatever the game already gave it is the number the
mission meant.
*/
//
//sp:name BluAssistApplyToRobot
//
//nolint:revive // unused-parameter: the handle is the timer's own, and the userid is what survives the robot leaving
func ApplyToRobot(timer engine.Timer, userid int32) engine.Outcome {
	client := engine.ClientOfUserID(userid)

	if client <= 0 || !engine.IsClientInGame(client) || !engine.IsPlayerAlive(client) {
		return engine.PluginStop()
	}

	if engine.PlayerTeam(client) != engine.TeamBlue() || !engine.IsFakeClient(client) {
		return engine.PluginStop()
	}

	scale := HealthScale()

	if Off(scale) {
		return engine.PluginStop()
	}

	maxHealth := engine.PlayerMaxHealth(client)

	if maxHealth < engine.ClientHealth(client) {
		maxHealth = engine.ClientHealth(client)
	}

	wanted := engine.RoundToCeil(float32(maxHealth) * scale)

	if wanted < 1 {
		wanted = 1
	}

	healthBefore[client] = maxHealth
	wantedMaxHealth[client] = wanted

	engine.HookMaxHealth(client)
	engine.SetEntPropData(client, engine.PropData(), "m_iMaxHealth", wanted)
	engine.SetEntityHealth(client, wanted)

	engine.CreateRobotTimer(settle, VerifyRobot, userid, engine.TimerNoMapChange())

	return engine.PluginStop()
}

/*
VerifyRobot reads the robot back once the game has had the same delay again.

Written because two versions of this reported success and changed nothing: the
lever was on, the log said it had been applied, and the robots came at the health
the popfile gave them. What is worth writing down is what the game says the robot
is worth now, not what this asked for.
*/
//
//sp:name BluAssistVerifyRobot
//
//nolint:revive // unused-parameter: the handle is the timer's own
func VerifyRobot(timer engine.Timer, userid int32) engine.Outcome {
	client := engine.ClientOfUserID(userid)

	if client <= 0 || !engine.IsClientInGame(client) || !engine.IsPlayerAlive(client) {
		return engine.PluginStop()
	}

	if engine.PlayerTeam(client) != engine.TeamBlue() || !engine.IsFakeClient(client) {
		return engine.PluginStop()
	}

	Say(client, engine.PlayerMaxHealth(client))

	return engine.PluginStop()
}

/*
How often a bend is written down: one robot in every BLU_ASSIST_SAMPLE.

The lever is read off wave outcomes otherwise, and a wave outcome cannot tell a
lever that did nothing from a lever that did something too small to win with. This
says what the robot was worth before and after, which is the question.

Sampled rather than every robot: a wave brings them by the hundred and a line per
robot is a line nobody reads. A robot the game disagrees about is written down
whatever the sample says, because one of those is the whole bug this lever has had
twice.
*/
//
//sp:name BLU_ASSIST_SAMPLE
const sample = 25

// Say writes down what a bend actually did to one robot.
//
//sp:name BluAssistSay
func Say(client int32, observed int32) {
	wanted := wantedMaxHealth[client]

	if wanted <= 0 {
		return
	}

	seen++

	mismatch := observed != wanted || engine.ClientHealth(client) != wanted

	if !mismatch && !healthDebugConVar.Bool() && seen%sample != 0 {
		return
	}

	if mismatch {
		engine.LogMessage("BluAssist: robot %d of the wave, was %d, wanted %d (x%.2f), game says %d and health %d: MISMATCH",
			seen, healthBefore[client], wanted, HealthScale(), observed, engine.ClientHealth(client))

		return
	}

	engine.LogMessage("BluAssist: robot %d of the wave, was %d, wanted %d (x%.2f), game says %d and health %d",
		seen, healthBefore[client], wanted, HealthScale(), observed, engine.ClientHealth(client))
}
