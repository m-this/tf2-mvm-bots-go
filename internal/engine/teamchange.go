package engine

/*
Somebody moved between teams, and the ready panel has to be told.

Mann vs Machine starts the wave when every member of the defending team has
pressed ready. A team that is entirely ready has no room for another bot to walk
in, so one member is unreadied to hold the door open.
*/

// TeamChangeCalls are the answers.
type TeamChangeCalls struct {
	VoteInProgress  func() bool
	CancelVote      func()
	PrintToChatTeam func(team Team, format string, args []any)
	BotSummoner     func() int32
	SetAllowBotRedo func(allow bool)
	CreateTimerVoid func(interval float32, data int32, flags int32) Timer
}

var teamChanges TeamChangeCalls

// InstallTeamChanges puts a set of answers behind them.
func InstallTeamChanges(c TeamChangeCalls) func() {
	previous := teamChanges
	teamChanges = c
	return func() { teamChanges = previous }
}

// VoteInProgress says a vote is running, which is the only time cancelling one
// means anything.
//
//sp:native IsVoteInProgress
func VoteInProgress() bool {
	if teamChanges.VoteInProgress == nil {
		missing("IsVoteInProgress")
	}
	return teamChanges.VoteInProgress()
}

// CancelVote stops the running vote.
//
//sp:native CancelVote
func CancelVote() {
	if teamChanges.CancelVote == nil {
		missing("CancelVote")
	}
	teamChanges.CancelVote()
}

// PrintToChatTeam writes one line to everybody on that team. Ported, chat.
//
//sp:body PrintToChatTeam
func PrintToChatTeam(team Team, format string, args ...any) {
	if teamChanges.PrintToChatTeam == nil {
		missing("PrintToChatTeam")
	}
	teamChanges.PrintToChatTeam(team, format, args)
}

// BotSummoner is g_iUIDBotSummoner, the userid of whoever called the bots in.
//
//sp:global g_iUIDBotSummoner
func BotSummoner() int32 {
	if teamChanges.BotSummoner == nil {
		missing("g_iUIDBotSummoner")
	}
	return teamChanges.BotSummoner()
}

// SetAllowBotRedo writes g_bAllowBotTeamRedo: RED may pick its lineup again.
//
//sp:globalset g_bAllowBotTeamRedo
func SetAllowBotRedo(allow bool) {
	if teamChanges.SetAllowBotRedo == nil {
		missing("g_bAllowBotTeamRedo")
	}
	teamChanges.SetAllowBotRedo(allow)
}

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
	if teamChanges.CreateTimerVoid == nil {
		missing("CreateTimer")
	}
	return teamChanges.CreateTimerVoid(interval, data, flags)
}
