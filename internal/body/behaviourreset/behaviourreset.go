/*
Package behaviourreset is the part of source/redbots3/events.sp that makes every
defender think again, one bot per tenth of a second.

Spread out rather than all at once: rethinking is a nav search, and six of them
on the frame a wave starts is a frame the server does not finish in time.
*/
package behaviourreset

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// BehaviourResetInterval is how long between one bot and the next.
//
//sp:name BEHAVIOUR_RESET_INTERVAL
const BehaviourResetInterval = 0.1

// The client the drain has reached, counted from 1.
//
//sp:name m_iBehaviourResetNext
var behaviourResetNext int32

// The timer draining them, and null when there is nothing left to drain.
//
//sp:name m_hBehaviourResetTimer
var behaviourResetTimer engine.Timer

// QueueBehaviourReset starts the drain at the first client.
//
//sp:name QueueBehaviourReset
func QueueBehaviourReset() {
	StopBehaviourReset()

	behaviourResetNext = 1
	behaviourResetTimer = engine.CreateTimer(BehaviourResetInterval, TimerResetOneBehaviour, engine.Default(), engine.TimerRepeat())
}

// StopBehaviourReset ends it. Killed by handle rather than deleted, because a
// map change closes it and leaves this one stale.
//
//sp:name StopBehaviourReset
func StopBehaviourReset() {
	if behaviourResetTimer != engine.NoTimer() {
		engine.KillTimer(behaviourResetTimer)
	}

	behaviourResetTimer = engine.NoTimer()
}

/*
TimerResetOneBehaviour rethinks one bot and comes back for the next.

Walked once, forwards, so a bot that joins mid-drain is not reset twice and none
is skipped.
*/
//
//sp:name Timer_ResetOneBehaviour
//sp:public
//
//nolint:revive // unused-parameter: the handle is the timer's own, and nothing here needs it
func TimerResetOneBehaviour(timer engine.Timer) engine.Outcome {
	for behaviourResetNext <= engine.MaxClients() {
		client := behaviourResetNext
		behaviourResetNext++

		if !engine.IsClientInGame(client) || !engine.DefenderBotFlag(client) || !engine.IsPlayerAlive(client) {
			continue
		}

		if !ShouldResetBehavior(client) {
			continue
		}

		// Rethink what we're supposed to do.
		engine.ResetIntentionInterface(client)

		return engine.PluginContinue()
	}

	behaviourResetTimer = engine.NoTimer()

	return engine.PluginStop()
}

// ShouldResetBehavior says this bot is not in the middle of something worth
// leaving alone.
//
//sp:name ShouldResetBehavior
func ShouldResetBehavior(client int32) bool {
	// Looking for sniping spots, don't disturb.
	if engine.LookupEntityActionByName(client, "SniperLurk") != engine.InvalidAction() {
		return false
	}

	// I'm healing people.
	if engine.LookupEntityActionByName(client, "Heal") != engine.InvalidAction() {
		return false
	}

	// I am building shit.
	if engine.LookupEntityActionByName(client, "DefenderEngineerIdle") != engine.InvalidAction() {
		return false
	}

	return true
}
