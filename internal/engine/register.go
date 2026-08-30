package engine

/*
Registering a callback that is not a timer: a console command, a game event.

CreateTimer already takes its callback by name, and this is the same shape for
the other two. The function is a value in the one way the subset allows: named,
as an argument to an extern that declares it takes one. A name in a string would
throw away the one thing worth checking, which is that the callback has the
signature the registration expects.
*/

// RegisterCalls are the answers.
type RegisterCalls struct {
	RegConsoleCmd           func(name string)
	HookEvent               func(name string)
	CreateTimerWith         func(interval float32, data int32, flags int32) Timer
	ResetIntentionInterface func(client int32)
}

var registrations RegisterCalls

// InstallRegistrations puts a set of answers behind them.
func InstallRegistrations(c RegisterCalls) func() {
	previous := registrations
	registrations = c
	return func() { registrations = previous }
}

// RegConsoleCmd puts a command on the console, taking its callback by name.
//
//sp:native RegConsoleCmd
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func RegConsoleCmd(name string, callback func(client int32, args int32) Outcome) {
	if registrations.RegConsoleCmd == nil {
		missing("RegConsoleCmd")
	}
	registrations.RegConsoleCmd(name)
}

// HookEvent listens for a game event, taking its callback by name.
//
//sp:native HookEvent
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func HookEvent(name string, callback func(event Event, name string, dontBroadcast bool) Outcome) {
	if registrations.HookEvent == nil {
		missing("HookEvent")
	}
	registrations.HookEvent(name)
}

/*
CreateTimerWith is CreateTimer with the cell a timer carries to its callback.

The same native as the one in engineer.go, declared again because the callback
takes the cell: a timer that carries a userid and one that carries nothing have
different callbacks, and the signature is the thing worth checking.
*/
//
//sp:native CreateTimer
//
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func CreateTimerWith(interval float32, callback func(timer Timer, data int32) Outcome, data int32, flags int32) Timer {
	if registrations.CreateTimerWith == nil {
		missing("CreateTimer")
	}
	return registrations.CreateTimerWith(interval, data, flags)
}

// ResetIntentionInterface makes the bot throw away what it was doing and decide
// again from the top.
//
//sp:plugin ResetIntentionInterface
func ResetIntentionInterface(client int32) {
	if registrations.ResetIntentionInterface == nil {
		missing("ResetIntentionInterface")
	}
	registrations.ResetIntentionInterface(client)
}
