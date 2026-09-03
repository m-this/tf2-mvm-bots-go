package engine

/*
A client slot being filled, and everything that has to be forgotten about
whoever had it last.

A slot is reused. Anything left on it from the previous occupant is not this
player's, so each of these is a write of the value the field starts at.
*/

// PutInServerCalls are the answers.
type PutInServerCalls struct {
	SetHasUpgraded         func(client int32, has bool)
	SetShoppedThisBreak    func(client int32, shopped bool)
	ForgetMedicCall        func(client int32)
	ResetExtraButtons      func(client int32)
	SetDeadRethinkTime     func(client int32, when float32)
	SetBuybackNumber       func(client int32, number int32)
	SetNextRollTime        func(client int32, when float32)
	ResetCommandThrottle   func(client int32)
	SetLastReadyInputTime  func(client int32, when float32)
	SetHasBoughtUpgrades   func(client int32, has bool)
	ResetNextBot           func(client int32)
	TakeBotSeat            func(client int32)
	MakeRoomForHumanPlayer func(client int32)
	DefenderBotCount       func(team Team) int32
	ShouldProcessCommand   func(client int32) bool
	LastReadyInputTime     func(client int32) float32
	MissionDifficultyNow   func() MissionDifficulty
	Press                  func(b int32) int32
	SetPress               func(b int32, buttons int32)
	PressTime              func(b int32) float32
	Release                func(b int32) int32
	SetRelease             func(b int32, buttons int32)
	ReleaseTime            func(b int32) float32
	KeySpeed               func(b int32) float32
}

var putInServers PutInServerCalls

// InstallPutInServers puts a set of answers behind them.
func InstallPutInServers(c PutInServerCalls) func() {
	previous := putInServers
	Fill(&c)
	putInServers = c
	return func() { putInServers = previous }
}

// ForgetMedicCall drops a call left on the clock by whoever had the slot.
// Ported, mediccall.
//
//sp:body ForgetMedicCall
func ForgetMedicCall(client int32) { putInServers.ForgetMedicCall(client) }

// ButtonInput is esButtonInput, the extra button state kept per client. An
// enum struct with a method on it, which the generator has no form for, so the
// plugin keeps it and this reaches it.
//
//sp:tag esButtonInput
type ButtonInput int32

// ExtraButtons is g_arrExtraButtons[client].
//
//sp:slot g_arrExtraButtons
//nolint:revive // unused-parameter: the index is what the emitter writes between the brackets, and nothing here reads it
func ExtraButtons(client int32) ButtonInput { return 0 }

// Reset puts every field of it back to its starting value.
//
//sp:method Reset
func (b ButtonInput) Reset() { putInServers.ResetExtraButtons(int32(b)) }

// SetDeadRethinkTime writes m_flDeadRethinkTime for one client.
//
//sp:slotset m_flDeadRethinkTime
func SetDeadRethinkTime(client int32, when float32) { putInServers.SetDeadRethinkTime(client, when) }

// SetBuybackNumber writes g_iBuybackNumber for one client.
//
//sp:slotset g_iBuybackNumber
func SetBuybackNumber(client int32, number int32) { putInServers.SetBuybackNumber(client, number) }

// SetNextRollTime writes m_flNextRollTime for one client.
//
//sp:slotset m_flNextRollTime
func SetNextRollTime(client int32, when float32) { putInServers.SetNextRollTime(client, when) }

// ResetCommandThrottle forgets when that client last typed. Ported, humans.
//
//sp:body Go_ResetCommandThrottle
func ResetCommandThrottle(client int32) { putInServers.ResetCommandThrottle(client) }

// SetLastReadyInputTime writes m_flLastReadyInputTime for one client.
//
//sp:slotset m_flLastReadyInputTime
func SetLastReadyInputTime(client int32, when float32) {
	putInServers.SetLastReadyInputTime(client, when)
}

// ResetNextBot clears every field the behaviour side keeps for that client.
// Ported, botreset.
//
//sp:body ResetNextBot
func ResetNextBot(client int32) { putInServers.ResetNextBot(client) }

// TakeBotSeat hands the waiting seat to the bot that just entered. Ported,
// playerpref.
//
//sp:body TakeBotSeat
func TakeBotSeat(client int32) { putInServers.TakeBotSeat(client) }

// MakeRoomForHumanPlayer frees a defender seat for somebody who just
// connected. Ported, manage.
//
//sp:body MakeRoomForHumanPlayer
func MakeRoomForHumanPlayer(client int32) { putInServers.MakeRoomForHumanPlayer(client) }

/*
The tournament ready panel's listener reaches these.

Pressing ready is the one input the mod intercepts, so the listener asks about
the mission, the team and the lineup before it lets one through.
*/

// DefenderBotCount is how many of ours are on that team, humans excluded.
// Ported, roster_counts.
//
//sp:body GetDefenderBotCount
func DefenderBotCount(team Team) int32 { return putInServers.DefenderBotCount(team) }

// MinPlayers is redbots_manager_min_players, the floor a server owner set, and
// -1 for no floor at all.
//
//sp:global redbots_manager_min_players
func MinPlayers() ConVar { return 0 }

// ShouldProcessCommand throttles how fast one client may type. Ported, humans.
//
//sp:body ShouldProcessCommand
func ShouldProcessCommand(client int32) bool { return putInServers.ShouldProcessCommand(client) }

// LastReadyInputTime reads m_flLastReadyInputTime for one client.
//
//sp:slot m_flLastReadyInputTime
func LastReadyInputTime(client int32) float32 { return putInServers.LastReadyInputTime(client) }

// MissionDifficultyNow is how hard the popfile being played is. Ported,
// mapconfig.
//
//sp:body GetMissionDifficulty
func MissionDifficultyNow() MissionDifficulty { return putInServers.MissionDifficultyNow() }

// Press is the buttons being held down for this client.
//
//sp:property iPress
func (b ButtonInput) Press() int32 { return putInServers.Press(int32(b)) }

// SetPress writes them.
//
//sp:propertyset iPress
func (b ButtonInput) SetPress(buttons int32) { putInServers.SetPress(int32(b), buttons) }

// PressTime is when the held buttons may be let go of.
//
//sp:property flPressTime
func (b ButtonInput) PressTime() float32 { return putInServers.PressTime(int32(b)) }

// Release is the buttons being held off for this client.
//
//sp:property iRelease
func (b ButtonInput) Release() int32 { return putInServers.Release(int32(b)) }

// SetRelease writes them.
//
//sp:propertyset iRelease
func (b ButtonInput) SetRelease(buttons int32) { putInServers.SetRelease(int32(b), buttons) }

// ReleaseTime is when the held-off buttons may be pressed again.
//
//sp:property flReleaseTime
func (b ButtonInput) ReleaseTime() float32 { return putInServers.ReleaseTime(int32(b)) }

// KeySpeed is how fast a held turn key moves the view.
//
//sp:property flKeySpeed
func (b ButtonInput) KeySpeed() float32 { return putInServers.KeySpeed(int32(b)) }
