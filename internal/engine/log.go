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
	ReplyToCommand func(client int32, format string, args ...any)
	HasEntProp     func(entity int32, propType PropType, prop string) bool
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
	CurrentMap    func() Text
	ActionStackOf func(client int32) Text
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
func CurrentMap() (name Text) {
	if texts.CurrentMap == nil {
		missing("GetCurrentMap")
	}
	return texts.CurrentMap()
}

// ActionStackOf is the behaviour stack a bot is running, as text, which is what
// the debug output prints.
//
//sp:body ActionStackOf sized
func ActionStackOf(client int32) (stack Text) {
	if texts.ActionStackOf == nil {
		missing("ActionStackOf")
	}
	return texts.ActionStackOf(client)
}

/*
Text is a string buffer, and there is one size of it.

SourcePawn buffers are declared with a length and the plugin uses several: 64
for a classname, PLATFORM_MAX_PATH for a map or a model, 512 for a behaviour
stack. Go needs a type per size, because an array length is not something a
function can be generic over, and one extern per size would be a set of
near-identical declarations nobody would keep in step.

So the port has one text size, and it is the largest the plugin uses: 512, which
is the behaviour stack the debug output prints. A generated buffer where the
plugin declared 64 is 512 bytes of stack instead of 64, on a frame that already
has a nav mesh search in it. That is the known normalisation, and it is a size
rather than a behaviour: what is written into it and compared out of it is the
same either way.
*/
//
//sp:tag char
type Text [512]byte

// TextOps are the answers for the operations on it.
type TextOps struct {
	StrEqual        func(a Text, b string) bool
	StrContains     func(haystack Text, needle string, caseSensitive bool) int32
	EntityClassname func(entity int32) Text
	CopyText        func(from string) Text
	CopyTextInto    func(from Text) Text
	StrEqualFolded  func(a Text, b string, caseSensitive bool) bool
}

var textOps TextOps

// InstallTextOps puts a set of answers behind them.
func InstallTextOps(c TextOps) func() {
	previous := textOps
	textOps = c
	return func() { textOps = previous }
}

/*
CopyText is strcopy into a buffer this port owns.

The destination is the assignment's left side and its length comes from the
declaration, which is what "fills" means: strcopy(buffer, sizeof buffer, from).
A buffer somebody else declared cannot be written this way, because the length
the caller passed is not something the generated code can see.

//sp:native strcopy fills
*/
func CopyText(from string) (into Text) {
	if textOps.CopyText == nil {
		missing("strcopy")
	}
	return textOps.CopyText(from)
}

// ChooseText is SourcePawn's ?: where one side is a buffer and the other a
// literal, which is the shape [[Choose]] cannot take.
//
//sp:choice ?:
func ChooseText(cond bool, yes string, no Text) Text {
	if cond {
		return CopyText(yes)
	}
	return no
}

// ChooseFloat is SourcePawn's ?: over two numbers, which the charge test writes
// inline rather than as a branch.
//
//sp:choice ?:
func ChooseFloat(cond bool, yes float32, no float32) float32 {
	if cond {
		return yes
	}
	return no
}

// ChooseInt is SourcePawn's ?: over two numbers.
//
//sp:choice ?:
func ChooseInt(cond bool, yes int32, no int32) int32 {
	if cond {
		return yes
	}
	return no
}

// CopyTextInto is strcopy into a buffer whose length the caller passed, which
// is what a function handed a buffer has to write with.
//
//sp:native strcopy fills
func CopyTextInto(from Text) (into Text) {
	if textOps.CopyTextInto == nil {
		missing("strcopy")
	}
	return textOps.CopyTextInto(from)
}

// StrEqualFolded compares a buffer against a literal, case sensitively or not.
//
//sp:native StrEqual
func StrEqualFolded(a Text, b string, caseSensitive bool) bool {
	if textOps.StrEqualFolded == nil {
		missing("StrEqual")
	}
	return textOps.StrEqualFolded(a, b, caseSensitive)
}

// StrEqual says whether the buffer holds exactly that.
//
//sp:native StrEqual
func StrEqual(a Text, b string) bool {
	if textOps.StrEqual == nil {
		missing("StrEqual")
	}
	return textOps.StrEqual(a, b)
}

// StrContains is where the needle starts, and -1 when it is not there.
//
//sp:native StrContains
func StrContains(haystack Text, needle string, caseSensitive bool) int32 {
	if textOps.StrContains == nil {
		missing("StrContains")
	}
	return textOps.StrContains(haystack, needle, caseSensitive)
}

// EntityClassname is what the entity is, filled into the buffer it is given.
//
//sp:native GetEntityClassname sized
func EntityClassname(entity int32) (class Text) {
	if textOps.EntityClassname == nil {
		missing("GetEntityClassname")
	}
	return textOps.EntityClassname(entity)
}

// ReplyToCommand answers whoever typed the command, in the console or the chat
// depending on where they typed it.
//
//sp:native ReplyToCommand
func ReplyToCommand(client int32, format string, args ...any) {
	if logs.ReplyToCommand == nil {
		missing("ReplyToCommand")
	}
	logs.ReplyToCommand(client, format, args...)
}

// HasEntProp says the entity has that property at all, which a medigun with no
// healing target still does and a weapon that is not a medigun does not.
//
//sp:native HasEntProp
func HasEntProp(entity int32, propType PropType, prop string) bool {
	if logs.HasEntProp == nil {
		missing("HasEntProp")
	}
	return logs.HasEntProp(entity, propType, prop)
}

// PluginHandled is Plugin_Handled, which a command callback returns to say it
// dealt with the command.
//
//sp:global Plugin_Handled
func PluginHandled() Outcome { return 3 }
