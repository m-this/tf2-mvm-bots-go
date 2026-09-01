package engine

/*
Counting the team.

Who is on RED, who among them is a person, and who among the people has said
they are ready. Every one of these walks the client slots, and several of them
are asked every think, which is why the readiness pair is cached for a frame.
*/

// RosterCountCalls are the answers.
type RosterCountCalls struct {
	KickClient            func(client int32, reason string)
	ChoosingBotClasses    func(client int32) bool
	LastCommandTime       func(client int32) float32
	SetLastCommandTime    func(client int32, when float32)
	HumansOnRed           func() bool
	SetHumansOnRed        func(on bool)
	AnyHumanReady         func() bool
	SetAnyHumanReady      func(ready bool)
	HumanReadinessTime    func() float32
	SetHumanReadinessTime func(when float32)
}

var rosterCounts RosterCountCalls

// InstallRosterCounts puts a set of answers behind them.
func InstallRosterCounts(c RosterCountCalls) func() {
	previous := rosterCounts
	rosterCounts = c
	return func() { rosterCounts = previous }
}

// TFBotIdentityName is the name every bot this mod makes carries, which is how
// one is recognised when the flag has not been set yet.
//
//sp:global TFBOT_IDENTITY_NAME
func TFBotIdentityName() string { return "" }

// CommandMaxRate is COMMAND_MAX_RATE, the fastest a bot may type.
//
//sp:global COMMAND_MAX_RATE
func CommandMaxRate() float32 { return 0 }

// KickClient removes a player, saying why.
//
//sp:native KickClient
func KickClient(client int32, reason string) {
	if rosterCounts.KickClient == nil {
		missing("KickClient")
	}
	rosterCounts.KickClient(client, reason)
}

// ChoosingBotClasses says this player has the lineup menu open.
//
//sp:slot g_bChoosingBotClasses
func ChoosingBotClasses(client int32) bool {
	if rosterCounts.ChoosingBotClasses == nil {
		missing("g_bChoosingBotClasses")
	}
	return rosterCounts.ChoosingBotClasses(client)
}
