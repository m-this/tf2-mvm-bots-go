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
	Continue       func() Outcome
	Done           func(reason string) Outcome
	ChangeTo       func(next Behaviour, reason string) Outcome
	ActionName     func(a Behaviour) Text
	IterateActions func(client int32, visit func(action Behaviour))
	AppendText     func(from string) Text
	AppendTextFrom func(from Text) Text
	SuspendFor     func(next Behaviour, reason string) Outcome
	Actor          func() int32
	TryToSustain   func() Outcome
	TryChangeTo    func(next Behaviour, priority int32, reason string) Outcome
	TryContinue    func() Outcome
	TryDone        func(priority int32, reason string) Outcome
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

// ThisAction is the action the engine handed the callback, for a helper that
// takes it rather than being one.
//
//sp:global action
func ThisAction() Behaviour { return 0 }

// EndWith is Done written on an action a helper was given, which is the same
// call with the receiver spelled out.
//
//sp:method Done
func (a Behaviour) EndWith(reason string) Outcome {
	if actions.Done == nil {
		missing("action.Done")
	}
	return actions.Done(reason)
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

// TryToSustain asks to keep going, which the engine may refuse.
//
//sp:native action.TryToSustain
func TryToSustain() Outcome {
	if actions.TryToSustain == nil {
		missing("action.TryToSustain")
	}
	return actions.TryToSustain()
}

// TryChangeTo asks to become another behaviour, at a priority the engine
// weighs against whatever else wants to run.
//
//sp:native action.TryChangeTo
func TryChangeTo(next Behaviour, priority int32, reason string) Outcome {
	if actions.TryChangeTo == nil {
		missing("action.TryChangeTo")
	}
	return actions.TryChangeTo(next, priority, reason)
}

// ResultCritical is RESULT_CRITICAL, which is a change the engine should not
// weigh against anything.
//
//sp:global RESULT_CRITICAL
func ResultCritical() int32 { return 3 }

// TryContinue asks to keep going, at the engine's discretion.
//
//sp:native action.TryContinue
func TryContinue() Outcome {
	if actions.TryContinue == nil {
		missing("action.TryContinue")
	}
	return actions.TryContinue()
}

// TryDone asks to stop, at a priority the engine weighs.
//
//sp:native action.TryDone
func TryDone(priority int32, reason string) Outcome {
	if actions.TryDone == nil {
		missing("action.TryDone")
	}
	return actions.TryDone(priority, reason)
}

// ResultImportant is RESULT_IMPORTANT.
//
//sp:global RESULT_IMPORTANT
func ResultImportant() int32 { return 2 }

// ActionName is what the behaviour calls itself, filled into a buffer.
//
//sp:method GetName sized
func (a Behaviour) ActionName() (name Text) {
	if actions.ActionName == nil {
		missing("BehaviorAction.GetName")
	}
	return actions.ActionName(a)
}

// IterateActions walks a bot's action stack, calling the visitor for each,
// taking it by name.
//
//sp:native ActionsManager.Iterator
//nolint:revive // unused-parameter: the visitor is a name the emitter writes, not something the Go calls
func IterateActions(client int32, visit func(action Behaviour)) {
	if actions.IterateActions == nil {
		missing("ActionsManager.Iterator")
	}
	actions.IterateActions(client, visit)
}

// AppendText adds to the end of a buffer, which is how the stack is built up.
//
//sp:native StrCat fills
func AppendText(from string) (into Text) {
	if actions.AppendText == nil {
		missing("StrCat")
	}
	return actions.AppendText(from)
}

// AppendTextFrom is StrCat where what is added is another buffer.
//
//sp:native StrCat fills
func AppendTextFrom(from Text) (into Text) {
	if actions.AppendTextFrom == nil {
		missing("StrCat")
	}
	return actions.AppendTextFrom(from)
}
