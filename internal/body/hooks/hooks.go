/*
Package hooks is the game-facing edge of source/redbots3/nextbot_behavior.sp:
the callbacks the actions extension hands to the game's own behaviours.

Four of them exist only to refuse: a defender bot has no business fetching the
flag, idling as a robot engineer, or leaving the spawn room like a robot spy, so
the action ends the moment it starts. Everything else in the file decides; these
five say no and get out of the way.
*/
package hooks

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
MainActionUpdate is the top of every bot's stack, and the only thing the mod
wants from it is the fault injector's hook.

Emptying a stack on purpose is how the idle watchdog gets tested: a bot with no
behaviour is exactly the one nothing else notices.
*/
//
//sp:name CTFBotMainAction_Update
//sp:public
//
//nolint:revive // unused-parameter: the interval and the result are the game's, and this reads neither
func MainActionUpdate(action engine.Behaviour, actor int32, interval float32, result engine.ActionResult) engine.Outcome {
	if engine.DefenderBotFlag(actor) && engine.ShouldEmptyStack(actor) {
		return action.EndWith("DebugFaults: emptying the stack")
	}

	return engine.PluginContinue()
}

// FetchFlagOnStart refuses: a defender has no bomb to fetch.
//
//sp:name CTFBotFetchFlag_OnStart
//sp:public
//nolint:revive // unused-parameter: the prior action and the result are the game's
func FetchFlagOnStart(action engine.Behaviour, actor int32, priorAction engine.Behaviour, result engine.ActionResult) engine.Outcome {
	if !engine.DefenderBotFlag(actor) {
		return engine.PluginContinue()
	}

	return action.End()
}

// MvMEngineerIdleOnStart refuses: our engineer has his own idle.
//
//sp:name CTFBotMvMEngineerIdle_OnStart
//sp:public
//nolint:revive // unused-parameter: the prior action and the result are the game's
func MvMEngineerIdleOnStart(action engine.Behaviour, actor int32, priorAction engine.Behaviour, result engine.ActionResult) engine.Outcome {
	if !engine.DefenderBotFlag(actor) {
		return engine.PluginContinue()
	}

	return action.End()
}

// SpyLeaveSpawnRoomOnStart refuses: our spy leaves under his own lurk.
//
//sp:name CTFBotSpyLeaveSpawnRoom_OnStart
//sp:public
//nolint:revive // unused-parameter: the prior action and the result are the game's
func SpyLeaveSpawnRoomOnStart(action engine.Behaviour, actor int32, priorAction engine.Behaviour, result engine.ActionResult) engine.Outcome {
	if !engine.DefenderBotFlag(actor) {
		return engine.PluginContinue()
	}

	return action.End()
}

/*
SniperLurkUpdate keeps the game's own lurk unless the rifle is gone.

A sniper holding something that is not a rifle is standing at a perch he cannot
use, which is half of mvm-bj8: he fights like anybody else instead.
*/
//
//sp:name CTFBotSniperLurk_Update
//sp:public
//
//nolint:revive // unused-parameter: the interval and the result are the game's
func SniperLurkUpdate(action engine.Behaviour, actor int32, interval float32, result engine.ActionResult) engine.Outcome {
	if !engine.DefenderBotFlag(actor) {
		return engine.PluginContinue()
	}

	if !engine.CanUsePrimaryWeapon(actor) {
		// Where did my gun go?
		return engine.SuspendFor(engine.DefenderAttack(), "Lost my rifle")
	}

	return engine.PluginContinue()
}

/*
ScenarioMonitorUpdate is where a defender is handed its next behaviour.

Suspend for the action we desire; once it has ended we come back here and
suspend for another one.
*/
//
//sp:name CTFBotScenarioMonitor_Update
//sp:public
//
//nolint:revive // unused-parameter: the interval and the result are the game's
func ScenarioMonitorUpdate(action engine.Behaviour, actor int32, interval float32, result engine.ActionResult) engine.Outcome {
	if !engine.DefenderBotFlag(actor) {
		return engine.PluginContinue()
	}

	return engine.DesiredBotAction(actor, action)
}

/*
MedicHealUpdatePost runs after the game's own heal think rather than instead of
it, so the medic keeps his walking and his output.

The game's answer to having nobody to heal is to fetch the bomb, and the answer
this mod had for that was to fight whatever is on it, which is the same walk
into the middle of the map by a different name. Everything the team is defending
comes to the hatch eventually.

Only his own shopping comes before healing. This was once the whole break, so a
medic spent the upgrade period walking after whoever he had picked; then it was
the other extreme and he stood at the front with nobody in front of the beam.
Buying his upgrades is the one thing nobody else can do for him, and after that
the man he beams is walking to the front regardless.
*/
//
//sp:name CTFBotMedicHeal_UpdatePost
//sp:public
//
//nolint:revive // unused-parameter: the interval is the game's
func MedicHealUpdatePost(action engine.Behaviour, actor int32, interval float32, result engine.ActionResult) engine.Outcome {
	if !engine.DefenderBotFlag(actor) {
		return engine.PluginContinue()
	}

	if result.ResultType() == engine.ChangeToResult() {
		// In mvm mode, medic bots will go for the flag when there's no
		// patient available. Let's be smarter about it instead.
		resultingAction := result.ResultAction()
		name := resultingAction.ActionName()

		if engine.StrEqual(name, "FetchFlag") {
			return engine.SuspendFor(engine.GuardPoint(), "Nothing to heal, so hold the hatch")
		}
	}

	secondary := engine.PlayerWeaponSlot(actor, engine.WeaponSlotSecondary())

	if secondary == -1 {
		return engine.SuspendFor(engine.DefenderAttack(), "No medigun")
	}

	if engine.AttackUberIsPossible(actor, secondary) {
		return engine.SuspendFor(engine.AttackUber(), "Seek uber")
	}

	if engine.MedicReviveIsPossible(actor) {
		return engine.SuspendFor(engine.MedicRevive(), "Revive teammate")
	}

	if engine.RoundState() == engine.RoundStateBetweenRounds() && !engine.ShoppedThisBreak(actor) {
		return engine.PluginContinue()
	}

	if engine.Feature(engine.FeatureMedicPocketsBiggest()) {
		engine.PointMedicAtBiggestBodyNow(action, actor)
	}

	myWeapon := engine.ActiveWeapon(actor)

	if myWeapon != -1 && engine.WeaponID(myWeapon) == engine.WeaponMedigun() {
		engine.MedicUberAndResistNow(actor, myWeapon, action.HandleEntity(engine.ActionHealPatientOffset()))
	}

	return engine.PluginContinue()
}
