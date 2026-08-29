package engine

/*
What a callback answers the engine with.

Written on the action the engine handed in, so these emit as action.Continue()
and action.Done("No money"). The action itself is never named in a body: it is
the first parameter of every callback and there is nothing else a body could do
with it.

The reason strings reach the game's own debug output and the test-bed's
telemetry, so two call sites with the same outcome and different reasons are two
different things and both are kept, byte for byte.
*/

// Outcome is what a behaviour callback returns, which SourceMod calls Action.
//
//sp:tag Action
type Outcome int32

// Behaviour is a BehaviorAction, the thing a constructor hands back and a
// callback can change to.
//
//sp:tag BehaviorAction
type Behaviour int32

// ActionCalls are the four answers, kept apart for the same reason BotCalls is.
type ActionCalls struct {
	Continue   func() Outcome
	Done       func(reason string) Outcome
	ChangeTo   func(next Behaviour, reason string) Outcome
	SuspendFor func(next Behaviour, reason string) Outcome
	Actor      func() int32
}

var actions ActionCalls

// InstallActions puts a set of answers behind the four and returns the undo.
func InstallActions(c ActionCalls) func() {
	previous := actions
	actions = c
	return func() { actions = previous }
}

// Continue keeps the behaviour running.
//
//sp:native action.Continue
func Continue() Outcome {
	if actions.Continue == nil {
		missing("action.Continue")
	}
	return actions.Continue()
}

// Done ends the behaviour and says why.
//
//sp:native action.Done
func Done(reason string) Outcome {
	if actions.Done == nil {
		missing("action.Done")
	}
	return actions.Done(reason)
}

// ChangeTo replaces this behaviour with another.
//
//sp:native action.ChangeTo
func ChangeTo(next Behaviour, reason string) Outcome {
	if actions.ChangeTo == nil {
		missing("action.ChangeTo")
	}
	return actions.ChangeTo(next, reason)
}

// SuspendFor runs another behaviour and comes back to this one.
//
//sp:native action.SuspendFor
func SuspendFor(next Behaviour, reason string) Outcome {
	if actions.SuspendFor == nil {
		missing("action.SuspendFor")
	}
	return actions.SuspendFor(next, reason)
}
