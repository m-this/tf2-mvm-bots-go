/*
Package spysap is source/redbots3/behavior/spysap.sp, ported.

The spy walks to the nearest enemy building it can sap, uncloaks when it gets
there, and plants the sapper. It is the first behaviour across, and it was
chosen because it has all five callbacks: the shape is the point, and a
behaviour with two of them would prove less of it.

What did not come with it, and why, is at the bottom.

//sp:action DefenderSpySap CTFBotSpySap
*/
package spysap

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// sapRange is how close the spy has to be. The shipped code declares it inside
// Update; it is here because the subset has no function-local const of its own
// to emit and this reads the same.
const sapRange = 40.0

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

// sapTarget is the building each spy is going for. It was declared at the top
// of spysap.sp and is the action's own state, so it comes across with it.
//
//sp:name m_iSapTarget
var sapTarget [Slots]int32

// OnStart aims the path and stops the bot looking around for itself.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	engine.UpdateLookAroundForEnemies(actor, false)

	return engine.Continue()
}

// Update walks to the target and saps it.
func Update(actor int32) engine.Outcome {
	if !engine.IsValidEntity(sapTarget[actor]) || !engine.IsBaseObject(sapTarget[actor]) || engine.HasSapper(sapTarget[actor]) {
		if !SelectTarget(actor) {
			return engine.Done("No sap target")
		}
	}

	myBot := engine.NextBotOf(actor)

	if myBot.IsRangeLessThan(sapTarget[actor], 2.0*sapRange) {
		mySapper := engine.PlayerWeaponSlot(actor, engine.WeaponSlotSecondary())

		if mySapper != -1 {
			engine.SetPlayerActiveWeapon(actor, mySapper)
		}

		if engine.IsStealthed(actor) || engine.IsFeignDeathReady(actor) {
			engine.PressAltFireButton(actor)
		}

		engine.SnapViewToPosition(actor, engine.WorldSpaceCenter(sapTarget[actor]))
		engine.PressFireButton(actor)
	}

	if engine.RepathTime(actor) <= engine.GameTime() {
		engine.CombatOf(sapTarget[actor]).UpdateLastKnownArea()

		engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(1.0, 2.0))
		engine.RepathToTarget(actor, myBot, sapTarget[actor])
	}

	engine.PathOf(actor).Update(myBot)

	return engine.Continue()
}

// OnEnd gives the looking back to the game.
func OnEnd(actor int32) {
	engine.UpdateLookAroundForEnemies(actor, true)
}

// OnSuspend gives it back while something else runs.
func OnSuspend(actor int32) engine.Outcome {
	engine.UpdateLookAroundForEnemies(actor, true)

	return engine.Continue()
}

// OnResume takes it again.
func OnResume(actor int32) engine.Outcome {
	engine.UpdateLookAroundForEnemies(actor, false)

	return engine.Continue()
}

// SelectTarget picks the nearest building worth sapping.
//
//sp:name CTFBotSpySap_SelectTarget
func SelectTarget(actor int32) bool {
	sapTarget[actor] = engine.NearestSappableObject(actor, 2000.0)

	return sapTarget[actor] != -1
}

/*
The two queries.

The engine asks rather than tells, so the answer goes back through a
by-reference parameter and the return says only that the behaviour had one. In
Go that is a second result, which becomes that parameter.

A spy on his way to a sapper does not stop to shoot, and a building he is about
to sap is not in his way.
*/

// ShouldAttack says no: the spy is going for the sapper, not the fight. It does
// not read who is being asked about, and the parameters stay because the engine
// passes them and the emitted declaration is the engine's.
//
//nolint:revive // unused-parameter: the signature is the engine's, not ours
func ShouldAttack(nextbot engine.Bot, knownEntity engine.Known) (changed engine.Outcome, result engine.Answer) {
	return engine.Changed(), engine.AnswerNo()
}

// IsHindrance says the target is not in the way, once the spy is close to it.
//
//nolint:revive // unused-parameter: the signature is the engine's, not ours
func IsHindrance(nextbot engine.Bot, entity int32) (changed engine.Outcome, result engine.Answer) {
	me := engine.Actor()

	if sapTarget[me] != -1 && nextbot.IsRangeLessThan(sapTarget[me], 300.0) {
		return engine.Changed(), engine.AnswerNo()
	}

	return engine.Changed(), engine.AnswerUndefined()
}
