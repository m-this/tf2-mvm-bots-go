/*
Package movetofront is source/redbots3/behavior/movetofront.sp.

Taking up a position before the wave. Eighteenth behaviour across.

How close counts as being there, how long the walk gets, and how often it may
start over: the front is the far end of the map, five thousand units from the
upgrade station on Coaltown, which is twenty-odd seconds of walking on top of the
shopping. So the clock is generous and only being wedged spends an attempt. A
clock that spent one would turn every bot that merely has a long way to go into a
bot standing still halfway there, which is what it did when this was first
written.

//sp:action DefenderMoveToFront CTFBotMoveToFront
*/
package movetofront

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

const (
	//sp:name MOVE_TO_FRONT_ARRIVED
	arrived = 80.0
	//sp:name MOVE_TO_FRONT_REACH
	reach = 60.0
	//sp:name MOVE_TO_FRONT_TRIES
	tries = 3
)

var (
	//sp:name m_vecGoalArea
	goalArea [Slots][3]float32
	//sp:name m_ctMoveTimeout
	moveTimeout [Slots]float32
	//sp:name m_iMoveToFrontTry
	moveToFrontTry [Slots]int32
	//sp:name m_bAtTheFront
	atTheFront [Slots]bool
)

/*
IsWaitingAtTheFront says whether this bot has finished taking up its position
for the coming wave.

Standing where he meant to stand and giving up short of it are the same answer
here: both mean he has stopped walking and is not going to move again before the
wave.
*/
//
//sp:name IsWaitingAtTheFront
func IsWaitingAtTheFront(client int32) bool {
	return atTheFront[client]
}

/*
PickTheFront is where the robots come out, which is where the team should be
waiting for them.

The holograms are the markers the game puts at the robot spawns, so the one
nearest the enemy spawn room is the start of the bomb's path. Standing on the
ground beside it is the difference between opening fire as the gate drops and
meeting the wave halfway up the map.
*/
//
//sp:name PickTheFront
func PickTheFront(actor int32) bool {
	/* The classes that shoot from a distance wait at the nest, the rest at the gate

	The gate is where the robots come out, and standing on it is how a defender meets a giant with
	nothing behind him. Waiting beside the sentry instead starts the wave with a sentry, a dispenser
	and the rest of the team in reach, and it is worth nothing to a Scout who has money to collect
	or a Pyro who has to be within a few metres to do anything at all.

	Holding the nest with the whole team was measured first and could not be told apart from the
	gate: four waves an arm, and the difference sat inside each arm's own spread. */
	if engine.Feature(engine.FeatureHoldTheNest()) && FightsAtRange(actor) && PickTheNest(actor) {
		return true
	}

	spawn := int32(-1)

	for {
		spawn = engine.FindEntityByClassname(spawn, "func_respawnroomvisualizer")
		if spawn == -1 {
			break
		}
		if engine.EntProp(spawn, engine.PropData(), "m_iDisabled") != 0 {
			continue
		}

		if engine.EntityTeamNumber(spawn) == engine.EntityTeamNumber(actor) {
			continue
		}

		break
	}

	if spawn == -1 {
		return false
	}

	flSmallestDistance := float32(99999.0)
	iBestEnt := int32(-1)

	holo := int32(-1)

	for {
		holo = engine.FindEntityByClassname(holo, "prop_dynamic")
		if holo == -1 {
			break
		}
		strModel := engine.EntPropString(holo, engine.PropData(), "m_ModelName")

		if !engine.StrEqual(strModel, "models/props_mvm/robot_hologram.mdl") {
			continue
		}

		if engine.EntProp(holo, engine.PropSend(), "m_fEffects")&32 != 0 {
			continue
		}

		flDistance := engine.VectorDistance(engine.WorldSpaceCenter(spawn), engine.WorldSpaceCenter(holo))

		if flDistance <= flSmallestDistance && engine.IsPathToVectorPossible(actor, engine.WorldSpaceCenter(holo)) {
			iBestEnt = holo
			flSmallestDistance = flDistance
		}
	}

	if iBestEnt == -1 {
		return false
	}

	area := engine.NearestNavArea(engine.WorldSpaceCenter(iBestEnt), true, 1000.0, true, true, engine.GetClientTeam(actor))

	if area == engine.NullArea() {
		return false
	}

	goalArea[actor] = engine.RandomPointIn(area)

	// A new goal is worth a path this frame rather than at the end of the old one's interval
	engine.SetRepathTime(actor, 0.0)

	return true
}

/*
FightsAtRange says whether this one does its damage from where the nest is, or
has to walk into the wave.

Asked for from play: the classes that fight at range belong around the
engineer's nest, and the ones that have to close belong at the gate. The Scout
collects money and the Pyro and the Spy work at arm's length, so all three are
wasted standing behind a sentry. Everybody else shoots across the same ground the
sentry covers.

This replaced holding the nest with the whole team, which was measured and could
not be told apart from the gate at four waves an arm.
*/
//
//sp:name FightsAtRange
func FightsAtRange(actor int32) bool {
	switch engine.PlayerClass(actor) {
	case engine.ClassScout(), engine.ClassPyro(), engine.ClassSpy():
		return false
	}

	return true
}

/*
PickTheNest is ground beside a teammate's sentry, or false when the team has
none up yet.

The nearest one, because two engineers are two nests and the one to stand at is
the one on the way to where this bot already is.
*/
//
//sp:name PickTheNest
func PickTheNest(actor int32) bool {
	best := int32(-1)
	bestRange := float32(0.0)
	mine := engine.WorldSpaceCenter(actor)

	sentry := int32(-1)

	for {
		sentry = engine.FindEntityByClassname(sentry, "obj_sentrygun")
		if sentry == -1 {
			break
		}
		if engine.EntityTeamNumber(sentry) != engine.GetClientTeam(actor) {
			continue
		}

		if engine.EntProp(sentry, engine.PropSend(), "m_bPlacing") != 0 || engine.EntProp(sentry, engine.PropSend(), "m_bCarried") != 0 {
			continue
		}

		sentryRange := engine.VectorDistance(mine, engine.WorldSpaceCenter(sentry))

		if best == -1 || sentryRange < bestRange {
			best = sentry
			bestRange = sentryRange
		}
	}

	if best == -1 {
		return false
	}

	area := engine.NearestNavArea(engine.WorldSpaceCenter(best), true, 1000.0, true, true, engine.GetClientTeam(actor))

	if area == engine.NullArea() {
		return false
	}

	goalArea[actor] = engine.RandomPointIn(area)

	engine.SetRepathTime(actor, 0.0)

	return true
}

// OnStart picks the front and gives up at once when there is none to pick.
func OnStart(actor int32) engine.Outcome {
	moveToFrontTry[actor] = 0
	atTheFront[actor] = false
	moveTimeout[actor] = engine.GameTime() + reach
	engine.RecoverDefenderFromDisconnectedSpawn(actor)

	if !PickTheFront(actor) {
		engine.SetPlayerReady(actor, true)
		return engine.Done("Cannot find the start of the robots' path from wherever we are")
	}

	return engine.Continue()
}

// Update walks there, and stops when the wave starts rather than when it
// arrives.
func Update(actor int32) engine.Outcome {
	/* The wave is what ends this, not arriving

	Arriving used to end it, and what happened next was nothing at all: the between-rounds branch
	of GetDesiredBotAction had no answer for a bot that had already shopped, so the game got the
	bot back and roamed it around the map. Reported as the Heavy, the Medic and the Pyro wandering
	off before the wave and turning up inside the middle house on Coaltown. */
	if engine.RoundState() != engine.RoundStateBetweenRounds() {
		return engine.Done("The wave has started")
	}

	// Credits on the floor are still worth the walk while we wait
	if engine.CollectMoneyIsPossible(actor) {
		return engine.SuspendFor(engine.CollectMoney(), "Money on the floor")
	}

	if atTheFront[actor] {
		return engine.Continue()
	}

	if engine.VectorDistance(goalArea[actor], engine.WorldSpaceCenter(actor)) < arrived {
		engine.SetPlayerReady(actor, true)
		atTheFront[actor] = true

		return engine.Continue()
	}

	myBot := engine.NextBotOf(actor)
	myLoco := myBot.Locomotion()

	/* Walking into the corner of a building is what spends an attempt

	The locomotion already knows the difference between walking and walking on the spot, and
	nothing outside the engineer has ever asked it. A fresh random point in the same area is a
	different approach to the same place, and three of them is a bound rather than a bot that
	repaths for ever.

	Out of attempts, or out of clock, he stands where he is: short of the front is a bot in the
	wrong place, and handed back to the game is a bot in the middle house. */
	if myLoco.IsStuck() {
		myLoco.ClearStuckStatus("Wedged on the way to the front")

		moveToFrontTry[actor]++

		if moveToFrontTry[actor] < tries {
			PickTheFront(actor)
		}
	}

	if moveToFrontTry[actor] >= tries || moveTimeout[actor] < engine.GameTime() {
		engine.SetPlayerReady(actor, true)
		atTheFront[actor] = true

		if engine.DebugActions().Bool() {
			engine.PrintToServer("[%8.3f] CTFBotMoveToFront(#%d): giving up short of the front", engine.GameTime(), actor)
		}

		return engine.Continue()
	}

	if engine.RepathTime(actor) <= engine.GameTime() {
		engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(3.0, 4.0))
		engine.RepathToPos(actor, myBot, goalArea[actor])
	}

	if engine.PathFailedFor(actor) {
		engine.NudgeTowardsGoal(actor, myBot, goalArea[actor])
	} else {
		engine.PathOf(actor).Update(myBot)
	}

	return engine.Continue()
}

// OnEnd forgets the goal.
func OnEnd(actor int32) {
	goalArea[actor] = engine.NullVector()
	atTheFront[actor] = false
	moveToFrontTry[actor] = 0
}

// DumpFront prints where each bot is and what it is waiting on.
//
//sp:public
//sp:name Command_DumpFront
func DumpFront(client int32, args int32) engine.Outcome { //nolint:revive // unused-parameter: the signature is SourceMod's
	haveBomb, bomb := engine.GetBombInfo()

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if !engine.IsClientInGame(i) || !engine.IsPlayerAlive(i) || !engine.IsDefenderBot(i) {
			continue
		}

		mine := engine.AbsOriginOf(i)

		action := engine.TextFrom("no waiting action")

		//nolint:gocritic // ifElseChain: the shipped file is a chain of
		// four lookups by name, and SourcePawn has no switch over one
		if engine.LookupEntityActionByName(i, "DefenderMoveToFront") != engine.InvalidAction() {
			action = engine.Format("walking to the front")

			if atTheFront[i] {
				action = engine.Format("holding the front")
			}
		} else if engine.LookupEntityActionByName(i, "DefenderEngineerIdle") != engine.InvalidAction() {
			action = engine.Format("at his nest")
		} else if engine.LookupEntityActionByName(i, "DefenderGotoUpgrade") != engine.InvalidAction() {
			action = engine.Format("walking to the station")
		} else if engine.LookupEntityActionByName(i, "DefenderUpgrade") != engine.InvalidAction() {
			action = engine.Format("shopping")
		}

		fromGoal := float32(-1.0)

		if !engine.IsZeroVector(goalArea[i]) {
			fromGoal = engine.VectorDistance(mine, goalArea[i])
		}

		fromBomb := float32(-1.0)

		if haveBomb {
			fromBomb = engine.VectorDistance(mine, bomb.Position)
		}

		shopped := engine.TextFrom("has not shopped")

		if engine.ShoppedThisBreak(i) {
			shopped = engine.TextFrom("has shopped")
		}

		ready := engine.TextFrom("not ready")

		if engine.IsPlayerReady(i) {
			ready = engine.TextFrom("ready")
		}

		engine.ReplyToCommand(client, "%N (%s): %s, %.0f from his goal, %.0f from the bomb, %s, %s, stuck %d times, %d dead-end paths",
			i, engine.RawPlayerClassName(engine.PlayerClass(i)), action,
			fromGoal, fromBomb, shopped, ready,
			engine.StuckCountOf(i), engine.PathFailuresOf(i))
	}

	return engine.PluginHandled()
}
