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
