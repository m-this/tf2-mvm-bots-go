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
}

var putInServers PutInServerCalls

// InstallPutInServers puts a set of answers behind them.
func InstallPutInServers(c PutInServerCalls) func() {
	previous := putInServers
	putInServers = c
	return func() { putInServers = previous }
}

// ForgetMedicCall drops a call left on the clock by whoever had the slot.
// Ported, mediccall.
//
//sp:body ForgetMedicCall
func ForgetMedicCall(client int32) {
	if putInServers.ForgetMedicCall == nil {
		missing("ForgetMedicCall")
	}
	putInServers.ForgetMedicCall(client)
}

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
func (b ButtonInput) Reset() {
	if putInServers.ResetExtraButtons == nil {
		missing("esButtonInput.Reset")
	}
	putInServers.ResetExtraButtons(int32(b))
}

// SetDeadRethinkTime writes m_flDeadRethinkTime for one client.
//
//sp:slotset m_flDeadRethinkTime
func SetDeadRethinkTime(client int32, when float32) {
	if putInServers.SetDeadRethinkTime == nil {
		missing("m_flDeadRethinkTime")
	}
	putInServers.SetDeadRethinkTime(client, when)
}

// SetBuybackNumber writes g_iBuybackNumber for one client.
//
//sp:slotset g_iBuybackNumber
func SetBuybackNumber(client int32, number int32) {
	if putInServers.SetBuybackNumber == nil {
		missing("g_iBuybackNumber")
	}
	putInServers.SetBuybackNumber(client, number)
}

// SetNextRollTime writes m_flNextRollTime for one client.
//
//sp:slotset m_flNextRollTime
func SetNextRollTime(client int32, when float32) {
	if putInServers.SetNextRollTime == nil {
		missing("m_flNextRollTime")
	}
	putInServers.SetNextRollTime(client, when)
}

// ResetCommandThrottle forgets when that client last typed. Ported, humans.
//
//sp:body Go_ResetCommandThrottle
func ResetCommandThrottle(client int32) {
	if putInServers.ResetCommandThrottle == nil {
		missing("Go_ResetCommandThrottle")
	}
	putInServers.ResetCommandThrottle(client)
}

// SetLastReadyInputTime writes m_flLastReadyInputTime for one client.
//
//sp:slotset m_flLastReadyInputTime
func SetLastReadyInputTime(client int32, when float32) {
	if putInServers.SetLastReadyInputTime == nil {
		missing("m_flLastReadyInputTime")
	}
	putInServers.SetLastReadyInputTime(client, when)
}

// ResetNextBot clears every field the behaviour side keeps for that client.
// Ported, botreset.
//
//sp:body ResetNextBot
func ResetNextBot(client int32) {
	if putInServers.ResetNextBot == nil {
		missing("ResetNextBot")
	}
	putInServers.ResetNextBot(client)
}

// TakeBotSeat hands the waiting seat to the bot that just entered. Ported,
// playerpref.
//
//sp:body TakeBotSeat
func TakeBotSeat(client int32) {
	if putInServers.TakeBotSeat == nil {
		missing("TakeBotSeat")
	}
	putInServers.TakeBotSeat(client)
}

// MakeRoomForHumanPlayer frees a defender seat for somebody who just
// connected. Ported, manage.
//
//sp:body MakeRoomForHumanPlayer
func MakeRoomForHumanPlayer(client int32) {
	if putInServers.MakeRoomForHumanPlayer == nil {
		missing("MakeRoomForHumanPlayer")
	}
	putInServers.MakeRoomForHumanPlayer(client)
}
