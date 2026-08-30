/*
Package bluassist is source/redbots3/blu_assist.sp.

# Bend a mission when few people turned up, by taking the robots down

Valve tunes every wave for six defenders. Fewer than six is a harder mission than
the one the map was built for, and the answer this mod has had until now is to
fill the empty seats with bots and hope their AI is good enough. This is the other
lever: leave the seats empty and make the robots weaker instead.

The convar is the scale at one human, and the scale rises to 1.0 at a full team.
So 0.7 means a lone player fights robots at seven tenths, three players at about
six sevenths, and six players fight the mission Valve wrote. Set to 1.0 the lever
is off, which is the default and what every existing server keeps.

The convar is the switch, so the two arms of an A/B are the same build with one
number different. There is no feature flag beside it: 1.0 already means off, and a
second switch that also has to be on would be two ways to say one thing. What is
set gets written into the run's results, so a file of numbers says which arm
produced it.

See docs/testbed-metrics.md: none of this ships on until a run says it helped.
*/
package bluassist

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// The team size the scale reaches 1.0 at, which is what Valve tunes a wave for.
//
//sp:name BLU_ASSIST_FULL_TEAM
const fullTeam = 6.0

//sp:name m_iBluAssistSeen
var seen int32

//sp:name redbots_manager_blu_health_scale
var healthScaleConVar engine.ConVar

// Init makes the convar and forgets the last mission's count.
//
//sp:name BluAssist_Init
func Init() {
	seen = 0

	healthScaleConVar = engine.CreateAssistConVar("sm_redbots_manager_blu_health_scale", "1.0",
		"What a robot's health is multiplied by when one human is on RED. Rises to 1.0 at six. 1.0 is off.",
		engine.FCVarNotify(), true, 0.1, true, 1.0)
}

// HumansOnRed is how many people who are not bots are playing on RED right now.
//
//sp:name BluAssist_HumansOnRed
func HumansOnRed() int32 {
	count := int32(0)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if engine.IsClientInGame(i) && !engine.IsFakeClient(i) && !engine.IsClientSourceTV(i) && engine.PlayerTeam(i) == engine.TeamRed() {
			count++
		}
	}

	return count
}

/*
Scale is the multiplier one of the three levers is at, for the people currently on RED.

Straight line from the convar at one human to 1.0 at six, so the assist fades as
the team fills rather than switching off at some threshold nobody can feel. An
empty RED is treated as one player: a server between rounds is not a reason to make
the wave hardest.

Returns 1.0 when the convar is 1.0, which is the whole of the switch being off.
*/
//
//sp:name BluAssistScale
func Scale(convar engine.ConVar) float32 {
	atOne := convar.Float()

	if atOne >= 1.0 {
		return 1.0
	}

	humans := float32(HumansOnRed())

	if humans < 1.0 {
		humans = 1.0
	}

	if humans >= fullTeam {
		return 1.0
	}

	return atOne + (1.0-atOne)*((humans-1.0)/(fullTeam-1.0))
}

// HealthScale is what a robot's health is multiplied by.
//
//sp:name BluAssist_HealthScale
func HealthScale() float32 {
	return Scale(healthScaleConVar)
}

/*
Describe adds what the levers are set to, for the line that says what was
different about this run.

Nothing is added when they are off, which is every run until somebody sets one, so
the string reads exactly as it did before this existed.

//sp:name BluAssist_Describe
//sp:length buffer maxlength
*/
//
//nolint:revive,ineffassign,staticcheck,wastedassign // the write is the point: SourcePawn passes the buffer by reference and //sp:length carries its size
func Describe(buffer engine.Text, maxlength int32) {
	if healthScaleConVar == engine.NoConVar() || healthScaleConVar.Float() >= 1.0 {
		return
	}

	buffer = engine.Format("blu_health=%.2f", healthScaleConVar.Float())
}

/*
OnRobotSpawn scales a robot's health as it spawns.

Applied a frame after player_spawn. The popfile finishes building the robot inside
the spawn frame: it gives the template's items, fires post_inventory_application,
then writes the health and the attributes the mission wants. Anything written from
the event itself is overwritten by that, which is why health did nothing while
damage, scaled on the hit rather than on the robot, did.

A giant scales the same way a common does. The alternative is a lever per robot
size, which is more numbers to measure before anybody knows whether the first one
matters.
*/
//
//sp:name BluAssist_OnRobotSpawn
func OnRobotSpawn(client int32) {
	if HealthScale() >= 1.0 {
		return
	}

	engine.ApplyNextFrame(ApplyToRobot, engine.ClientUserID(client))
}

/*
ApplyToRobot writes the health the lever asks for.

Health goes through "max health additive bonus" as well as m_iMaxHealth, because
TF2 recomputes the maximum from the attributes and would otherwise put the
popfile's number back. The delta is added to what the popfile already set, so a
giant stays a giant, scaled.
*/
//
//sp:name BluAssistApplyToRobot
//sp:public
func ApplyToRobot(userid int32) {
	client := engine.ClientOfUserID(userid)

	if client <= 0 || !engine.IsClientInGame(client) || !engine.IsPlayerAlive(client) {
		return
	}

	if engine.PlayerTeam(client) != engine.TeamBlue() || !engine.IsFakeClient(client) {
		return
	}

	health := HealthScale()

	if health < 1.0 {
		maxHealth := engine.EntProp(client, engine.PropData(), "m_iMaxHealth")
		wanted := engine.RoundToCeil(float32(maxHealth) * health)

		if wanted < 1 {
			wanted = 1
		}

		BendAttrib(client, "max health additive bonus", 0.0, float32(wanted-maxHealth), 1.0)

		engine.SetEntPropData(client, engine.PropData(), "m_iMaxHealth", wanted)
		engine.SetEntityHealth(client, wanted)
	}

	Say(client, health)
}

/*
How often a bend is written down: one robot in every BLU_ASSIST_SAMPLE.

The levers are read off wave outcomes otherwise, and a wave outcome cannot tell a
lever that did nothing from a lever that did something too small to win with. This
says what the robot was worth before and after, which is the question.

Sampled rather than every robot: a wave brings them by the hundred and a line per
robot is a line nobody reads.
*/
//
//sp:name BLU_ASSIST_SAMPLE
const sample = 25

// Say writes down what a bend actually did to one robot.
//
//sp:name BluAssistSay
func Say(client int32, health float32) {
	if health >= 1.0 {
		return
	}

	seen++

	if seen%sample != 0 {
		return
	}

	engine.LogMessage("BluAssist: robot %d of the wave, health %d of %d, wanted x%.2f",
		seen, engine.ClientHealth(client), engine.EntProp(client, engine.PropData(), "m_iMaxHealth"), health)
}

/*
BendAttrib bends an attribute the popfile may already have set, from its neutral
value when it has not.

Both bends compose with the mission rather than replacing it: the delta for a
health bonus that is additive, the scale for a speed bonus that is a multiplier.
*/
//
//sp:name BluAssistBendAttrib
//sp:default scale 1.0
func BendAttrib(client int32, name string, neutral float32, delta float32, scale float32) {
	attrib := engine.AttribByName(client, name)

	current := neutral

	if attrib != engine.AddressNull() {
		current = engine.AttribValue(attrib)
	}

	engine.SetAttribByName(client, name, current*scale+delta)
}
