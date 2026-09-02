package engine

/*
Who sits where, and when a change to that is allowed to happen.

A lineup change that arrives while a wave is running cannot act: reseating means
kicking and re-adding, and a bot kicked mid-wave takes its upgrades with it. So
the change is held as a flag and the break is what spends it.
*/

// SeatingCalls are the answers.
type SeatingCalls struct {
	ReseatPending          func() bool
	SetReseatPending       func(pending bool)
	RecyclePending         func() bool
	SetRecyclePending      func(pending bool)
	RecycleDefenderBots    func() int32
	ReseatDefenderBots     func() int32
	DispatchKeyValueVec    func(entity int32, key string, value [3]float32)
	DispatchKeyValue       func(entity int32, key string, value string)
	TeamClientCount        func(team Team) int32
	PlayersChoosingClasses func() int32
	ChosenBotSeats         func() List
	BotClassesLocked       func() bool
}

var seatings SeatingCalls

// InstallSeating puts a set of answers behind them.
func InstallSeating(c SeatingCalls) func() {
	previous := seatings
	seatings = c
	return func() { seatings = previous }
}

// ReseatPending says a lineup change is waiting for the break.
//
//sp:global m_bReseatPending
func ReseatPending() bool {
	if seatings.ReseatPending == nil {
		missing("m_bReseatPending")
	}
	return seatings.ReseatPending()
}

// SetReseatPending writes it.
//
//sp:globalset m_bReseatPending
func SetReseatPending(pending bool) {
	if seatings.SetReseatPending == nil {
		missing("m_bReseatPending")
	}
	seatings.SetReseatPending(pending)
}

// RecyclePending says bots held from mid-wave are waiting to be recycled.
//
//sp:global m_bRecyclePending
func RecyclePending() bool {
	if seatings.RecyclePending == nil {
		missing("m_bRecyclePending")
	}
	return seatings.RecyclePending()
}

// SetRecyclePending writes it.
//
//sp:globalset m_bRecyclePending
func SetRecyclePending(pending bool) {
	if seatings.SetRecyclePending == nil {
		missing("m_bRecyclePending")
	}
	seatings.SetRecyclePending(pending)
}

// RecycleDefenderBots reclasses the bots in place and says how many it moved.
// Ported, roster_counts.
//
//sp:body RecycleDefenderBots
func RecycleDefenderBots() int32 {
	if seatings.RecycleDefenderBots == nil {
		missing("RecycleDefenderBots")
	}
	return seatings.RecycleDefenderBots()
}

// ReseatDefenderBots kicks and re-adds so the team matches the lineup. Still in
// tf2_defenderbots.sp.
//
//sp:plugin ReseatDefenderBots
func ReseatDefenderBots() int32 {
	if seatings.ReseatDefenderBots == nil {
		missing("ReseatDefenderBots")
	}
	return seatings.ReseatDefenderBots()
}

// DispatchKeyValueVec sets a three-float key before the spawn.
//
//sp:native DispatchKeyValueVector
func DispatchKeyValueVec(entity int32, key string, value [3]float32) {
	if seatings.DispatchKeyValueVec == nil {
		missing("DispatchKeyValueVector")
	}
	seatings.DispatchKeyValueVec(entity, key, value)
}

// DispatchKeyValue sets a string key before the spawn.
//
//sp:native DispatchKeyValue
func DispatchKeyValue(entity int32, key string, value string) {
	if seatings.DispatchKeyValue == nil {
		missing("DispatchKeyValue")
	}
	seatings.DispatchKeyValue(entity, key, value)
}

// TeamClientCount is how many players are on that team, bots included.
//
//sp:native GetTeamClientCount
func TeamClientCount(team Team) int32 {
	if seatings.TeamClientCount == nil {
		missing("GetTeamClientCount")
	}
	return seatings.TeamClientCount(team)
}

// PlayersChoosingClasses is how many people have the lineup menu open. Ported,
// teammenu.
//
//sp:body GetCountOfPlayersChoosingBotClasses
func PlayersChoosingClasses() int32 {
	if seatings.PlayersChoosingClasses == nil {
		missing("GetCountOfPlayersChoosingBotClasses")
	}
	return seatings.PlayersChoosingClasses()
}

// ChosenBotSeats is g_adtChosenBotSeats, the seat each chosen class sits in.
//
//sp:global g_adtChosenBotSeats
func ChosenBotSeats() List {
	if seatings.ChosenBotSeats == nil {
		missing("g_adtChosenBotSeats")
	}
	return seatings.ChosenBotSeats()
}

// BotClassesLocked says the lineup a player accepted is being held.
//
//sp:global g_bBotClassesLocked
func BotClassesLocked() bool {
	if seatings.BotClassesLocked == nil {
		missing("g_bBotClassesLocked")
	}
	return seatings.BotClassesLocked()
}
