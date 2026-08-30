package engine

/*
Printing, which is variadic and is the one place the subset lets that through.

The subset has no variadic call of its own: a body cannot declare one and cannot
spread a slice into one. What it can do is call an extern that is variadic,
because the arguments are written out at the call site and the emitter puts them
where they were.

Nothing else in the subset is allowed to be variadic, and it is worth saying why
this is: the format string is a literal at every call site in the plugin, so
what is emitted is exactly what was written, and there is no place for the
argument list to be wrong that spcomp does not already check.
*/

// LogCalls are the answers. The arguments come through untouched, so a test
// that wants to read them takes them as they are.
type LogCalls struct {
	PrintToServer  func(format string, args ...any)
	PrintToChatAll func(format string, args ...any)
	LogAction      func(client int32, target int32, format string, args ...any)
}

var logs LogCalls

// InstallLogs puts a set of answers behind them.
func InstallLogs(c LogCalls) func() {
	previous := logs
	logs = c
	return func() { logs = previous }
}

// PrintToServer writes a line to the server console, which is where a run's
// running commentary goes.
//
//sp:native PrintToServer
func PrintToServer(format string, args ...any) {
	if logs.PrintToServer == nil {
		missing("PrintToServer")
	}
	logs.PrintToServer(format, args...)
}

// PrintToChatAll writes a line every player sees.
//
//sp:native PrintToChatAll
func PrintToChatAll(format string, args ...any) {
	if logs.PrintToChatAll == nil {
		missing("PrintToChatAll")
	}
	logs.PrintToChatAll(format, args...)
}

// LogAction writes a line to the game's own log, attributed to a player.
//
//sp:native LogAction
func LogAction(client int32, target int32, format string, args ...any) {
	if logs.LogAction == nil {
		missing("LogAction")
	}
	logs.LogAction(client, target, format, args...)
}

// TextCalls are the few text operations the port needs. There are few on
// purpose: 95 of the 97 string comparisons in the behaviour files compare an
// attribute name against a literal, and internal/tables already generates the
// name to id table that turns those into an integer switch. That is the rule
// SUBSET.md states and it is the right answer for almost all of them.
type TextCalls struct {
	CurrentMap    func() [64]byte
	ActionStackOf func(client int32) [512]byte
}

var texts TextCalls

// InstallTexts puts a set of answers behind them.
func InstallTexts(c TextCalls) func() {
	previous := texts
	texts = c
	return func() { texts = previous }
}

// CurrentMap is the map being played, filled into the buffer it is given.
//
//sp:native GetCurrentMap sized
func CurrentMap() (name [64]byte) {
	if texts.CurrentMap == nil {
		missing("GetCurrentMap")
	}
	return texts.CurrentMap()
}

// ActionStackOf is the behaviour stack a bot is running, as text, which is what
// the debug output prints.
//
//sp:plugin ActionStackOf sized
func ActionStackOf(client int32) (stack [512]byte) {
	if texts.ActionStackOf == nil {
		missing("ActionStackOf")
	}
	return texts.ActionStackOf(client)
}
