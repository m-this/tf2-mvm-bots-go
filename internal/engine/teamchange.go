package engine

/*
Somebody moved between teams, and the ready panel has to be told.

Mann vs Machine starts the wave when every member of the defending team has
pressed ready. A team that is entirely ready has no room for another bot to walk
in, so one member is unreadied to hold the door open.
*/

// TeamChangeCalls are the answers.
type TeamChangeCalls struct {
	VoteInProgress   func() bool
	CancelVote       func()
	PrintToChatTeam  func(team int32, format string, args []any)
	BotSummoner      func() int32
	SetAllowBotRedo  func(allow bool)
	CreateTimerVoid  func(interval float32, data int32, flags int32) Timer
	ExplodeClassList func(text Text, split string, out [9]Text, maxStrings int32, maxStringLength int32) int32
	ExplodeSeatList  func(text Text, split string, out [65]Text, maxStrings int32, maxStringLength int32) int32
}

var teamChanges TeamChangeCalls

// InstallTeamChanges puts a set of answers behind them.
func InstallTeamChanges(c TeamChangeCalls) func() {
	previous := teamChanges
	Fill(&c)
	teamChanges = c
	return func() { teamChanges = previous }
}

// VoteInProgress says a vote is running, which is the only time cancelling one
// means anything.
//
//sp:native IsVoteInProgress
func VoteInProgress() bool { return teamChanges.VoteInProgress() }

// CancelVote stops the running vote.
//
//sp:native CancelVote
func CancelVote() { teamChanges.CancelVote() }

// PrintToChatTeam writes one line to everybody on that team. Ported, chat.
//
// The team is an int and not a TFTeam, which is what util.sp declared and what
// the body generates. The extern said TFTeam for a while and nothing noticed:
// both are one cell.
//
//sp:body PrintToChatTeam
func PrintToChatTeam(team int32, format string, args ...any) {
	teamChanges.PrintToChatTeam(team, format, args)
}

// BotSummoner is g_iUIDBotSummoner, the userid of whoever called the bots in.
//
//sp:global g_iUIDBotSummoner
func BotSummoner() int32 { return teamChanges.BotSummoner() }

// SetAllowBotRedo writes g_bAllowBotTeamRedo: RED may pick its lineup again.
//
//sp:globalset g_bAllowBotTeamRedo
func SetAllowBotRedo(allow bool) { teamChanges.SetAllowBotRedo(allow) }

/*
CreateTimerVoid is CreateTimer with a callback that returns nothing.

A third declaration of the same native: the callback's signature is the thing
worth checking, and a timer whose callback is void is not one whose callback
returns an Action.

//sp:native CreateTimer
*/
//
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func CreateTimerVoid(interval float32, callback func(timer Timer, data int32), data int32, flags int32) Timer {
	return teamChanges.CreateTimerVoid(interval, data, flags)
}

/*
ExplodeClassList splits a comma-separated list into one buffer per class.

Two of these rather than one, because a Go array length is part of its type and
the plugin explodes into two different widths: nine, one per class, and one per
seat. Neither can be written generically and neither is worth losing the length
off.

//sp:native ExplodeString
*/
func ExplodeClassList(text Text, split string, out [9]Text, maxStrings int32, maxStringLength int32) int32 {
	return teamChanges.ExplodeClassList(text, split, out, maxStrings, maxStringLength)
}

// ExplodeSeatList is the same into one buffer per seat.
//
//sp:native ExplodeString
func ExplodeSeatList(text Text, split string, out [65]Text, maxStrings int32, maxStringLength int32) int32 {
	return teamChanges.ExplodeSeatList(text, split, out, maxStrings, maxStringLength)
}
