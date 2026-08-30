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
	SetShoppedThisBreak     func(client int32, shopped bool)
	SetBeingRevived         func(client int32, reviving bool)
	EventInt                func(e Event, key string) int32
	EventBool               func(e Event, key string) bool
	ResetSpyIntel           func()
	SetupSniperSpotHints    func()
	NestRelocationResetAll  func()
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

// SetShoppedThisBreak writes the shopping flag, which a break clears for
// everybody.
//
//sp:slotset g_bShoppedThisBreak
func SetShoppedThisBreak(client int32, shopped bool) {
	if registrations.SetShoppedThisBreak == nil {
		missing("g_bShoppedThisBreak")
	}
	registrations.SetShoppedThisBreak(client, shopped)
}

// SetBeingRevived says somebody has a revive marker up over this player.
//
//sp:slotset g_bIsBeingRevived
func SetBeingRevived(client int32, reviving bool) {
	if registrations.SetBeingRevived == nil {
		missing("g_bIsBeingRevived")
	}
	registrations.SetBeingRevived(client, reviving)
}

// SpyKilled is g_bSpyKilled, set while a defender spy's death is being handled.
//
//sp:global g_bSpyKilled
func SpyKilled() bool { return false }

// EventInt reads a number off a game event.
//
//sp:method GetInt
func (e Event) EventInt(key string) int32 {
	if registrations.EventInt == nil {
		missing("Event.GetInt")
	}
	return registrations.EventInt(e, key)
}

// EventBool reads a flag off one.
//
//sp:method GetBool
func (e Event) EventBool(key string) bool {
	if registrations.EventBool == nil {
		missing("Event.GetBool")
	}
	return registrations.EventBool(e, key)
}

// ResetSpyIntel forgets what the bots think they know about spies. Ported,
// spycheck.
//
//sp:body ResetSpyIntel
func ResetSpyIntel() {
	if registrations.ResetSpyIntel == nil {
		missing("ResetSpyIntel")
	}
	registrations.ResetSpyIntel()
}

// SetupSniperSpotHints reads the map's sniping spots again.
//
//sp:plugin SetupSniperSpotHints
func SetupSniperSpotHints() {
	if registrations.SetupSniperSpotHints == nil {
		missing("SetupSniperSpotHints")
	}
	registrations.SetupSniperSpotHints()
}

// NestRelocationResetAll forgets every engineer's relocation state. Ported,
// engineeridle.
//
//sp:body EngineerNestRelocation_ResetAll
func NestRelocationResetAll() {
	if registrations.NestRelocationResetAll == nil {
		missing("EngineerNestRelocation_ResetAll")
	}
	registrations.NestRelocationResetAll()
}
