package engine

/*
Who sits where, and when a change to that is allowed to happen.

A lineup change that arrives while a wave is running cannot act: reseating means
kicking and re-adding, and a bot kicked mid-wave takes its upgrades with it. So
the change is held as a flag and the break is what spends it.
*/

// SeatingCalls are the answers.
type SeatingCalls struct {
	ReseatPending                    func() bool
	SetReseatPending                 func(pending bool)
	RecyclePending                   func() bool
	SetRecyclePending                func(pending bool)
	RecycleDefenderBots              func() int32
	ReseatDefenderBots               func() int32
	DispatchKeyValueVec              func(entity int32, key string, value [3]float32)
	DispatchKeyValue                 func(entity int32, key string, value string)
	TeamClientCount                  func(team Team) int32
	PlayersChoosingClasses           func() int32
	ChosenBotSeats                   func() List
	BotClassesLocked                 func() bool
	SetGameConVar                    func(name string, c ConVar)
	NewListSized                     func(blocksize int32) List
	CollectPlayerBotClassPreferences func(out List)
	CollectMissingTeamComposition    func(classes List, seats List, count int32) int32
	ChooseBotClassesFromLineupMode   func(count int32)
	RandomClassBetween               func(low Class, high Class) Class
	NoteBotSeatPending               func(seat int32)
	ClassIndexFromString             func(name Text) Class
}

var seatings SeatingCalls

// InstallSeating puts a set of answers behind them.
func InstallSeating(c SeatingCalls) func() {
	previous := seatings
	Fill(&c)
	seatings = c
	return func() { seatings = previous }
}

// ReseatPending says a lineup change is waiting for the break.
//
//sp:global m_bReseatPending
func ReseatPending() bool { return seatings.ReseatPending() }

// SetReseatPending writes it.
//
//sp:globalset m_bReseatPending
func SetReseatPending(pending bool) { seatings.SetReseatPending(pending) }

// RecyclePending says bots held from mid-wave are waiting to be recycled.
//
//sp:global m_bRecyclePending
func RecyclePending() bool { return seatings.RecyclePending() }

// SetRecyclePending writes it.
//
//sp:globalset m_bRecyclePending
func SetRecyclePending(pending bool) { seatings.SetRecyclePending(pending) }

// RecycleDefenderBots reclasses the bots in place and says how many it moved.
// Ported, roster_counts.
//
//sp:body RecycleDefenderBots
func RecycleDefenderBots() int32 { return seatings.RecycleDefenderBots() }

// ReseatDefenderBots kicks and re-adds so the team matches the lineup. Still in
// tf2_defenderbots.sp.
//
//sp:body ReseatDefenderBots
func ReseatDefenderBots() int32 { return seatings.ReseatDefenderBots() }

// DispatchKeyValueVec sets a three-float key before the spawn.
//
//sp:native DispatchKeyValueVector
func DispatchKeyValueVec(entity int32, key string, value [3]float32) {
	seatings.DispatchKeyValueVec(entity, key, value)
}

// DispatchKeyValue sets a string key before the spawn.
//
//sp:native DispatchKeyValue
func DispatchKeyValue(entity int32, key string, value string) {
	seatings.DispatchKeyValue(entity, key, value)
}

// TeamClientCount is how many players are on that team, bots included.
//
//sp:native GetTeamClientCount
func TeamClientCount(team Team) int32 { return seatings.TeamClientCount(team) }

// PlayersChoosingClasses is how many people have the lineup menu open. Ported,
// teammenu.
//
//sp:body GetCountOfPlayersChoosingBotClasses
func PlayersChoosingClasses() int32 { return seatings.PlayersChoosingClasses() }

// ChosenBotSeats is g_adtChosenBotSeats, the seat each chosen class sits in.
//
//sp:global g_adtChosenBotSeats
func ChosenBotSeats() List { return seatings.ChosenBotSeats() }

// BotClassesLocked says the lineup a player accepted is being held.
//
//sp:global g_bBotClassesLocked
func BotClassesLocked() bool { return seatings.BotClassesLocked() }

/*
The game's own convars, found once at startup.

The plugin declares each one and this writes it. A globalset per convar rather
than one call taking a name, because the name is what SourcePawn writes on the
left of the assignment and there is nothing generic to hold.
*/

// SetBlind writes nb_blind.
//
//sp:globalset nb_blind
func SetBlind(c ConVar) { seatings.set("nb_blind", c) }

// SetPathLookaheadRange writes tf_bot_path_lookahead_range.
//
//sp:globalset tf_bot_path_lookahead_range
func SetPathLookaheadRange(c ConVar) { seatings.set("tf_bot_path_lookahead_range", c) }

// SetHealthCriticalRatio writes tf_bot_health_critical_ratio.
//
//sp:globalset tf_bot_health_critical_ratio
func SetHealthCriticalRatio(c ConVar) { seatings.set("tf_bot_health_critical_ratio", c) }

// SetHealthOkRatio writes tf_bot_health_ok_ratio.
//
//sp:globalset tf_bot_health_ok_ratio
func SetHealthOkRatio(c ConVar) { seatings.set("tf_bot_health_ok_ratio", c) }

// SetAmmoSearchRange writes tf_bot_ammo_search_range.
//
//sp:globalset tf_bot_ammo_search_range
func SetAmmoSearchRange(c ConVar) { seatings.set("tf_bot_ammo_search_range", c) }

// SetHealthSearchFarRange writes tf_bot_health_search_far_range.
//
//sp:globalset tf_bot_health_search_far_range
func SetHealthSearchFarRange(c ConVar) { seatings.set("tf_bot_health_search_far_range", c) }

// SetHealthSearchNearRange writes tf_bot_health_search_near_range.
//
//sp:globalset tf_bot_health_search_near_range
func SetHealthSearchNearRange(c ConVar) { seatings.set("tf_bot_health_search_near_range", c) }

// set records one, which is all a Go process can do with it.
func (s SeatingCalls) set(name string, c ConVar) {
	if s.SetGameConVar == nil {
		missing(name)
	}
	s.SetGameConVar(name, c)
}

// ClassNameMax is TF2_CLASS_MAX_NAME_LENGTH, the block size of every list that
// holds class names.
//
//sp:global TF2_CLASS_MAX_NAME_LENGTH
func ClassNameMax() int32 { return 16 }

// NewListSized makes an ArrayList whose entries are that many cells, which is
// what a list of strings needs.
//
//sp:new ArrayList
func NewListSized(blocksize int32) List { return seatings.NewListSized(blocksize) }

// CollectPlayerBotClassPreferences fills a list with the classes the people on
// RED asked for. Ported, playerpref.
//
//sp:body CollectPlayerBotClassPreferences
func CollectPlayerBotClassPreferences(out List) { seatings.CollectPlayerBotClassPreferences(out) }

// CollectMissingTeamComposition names the seats the convar still wants filled
// and says how many. Still in tf2_defenderbots.sp.
//
//sp:body CollectMissingTeamComposition
func CollectMissingTeamComposition(classes List, seats List, count int32) int32 {
	return seatings.CollectMissingTeamComposition(classes, seats, count)
}

// ChooseBotClassesFromLineupMode fills the rest of the lineup from the mode.
// Ported, seating.
//
//sp:body ChooseBotClassesFromLineupMode
func ChooseBotClassesFromLineupMode(count int32) { seatings.ChooseBotClassesFromLineupMode(count) }

// RandomClassBetween is GetRandomInt over two class values, which is how a
// random lineup picks one.
//
//sp:native GetRandomInt
func RandomClassBetween(low Class, high Class) Class { return seatings.RandomClassBetween(low, high) }

// NoteBotSeatPending remembers a seat asked for, waiting for the bot the server
// has not created yet. Ported, playerpref.
//
//sp:body NoteBotSeatPending
func NoteBotSeatPending(seat int32) { seatings.NoteBotSeatPending(seat) }

// ClassIndexFromString is the class a name means, and TFClass_Unknown for one
// that means none.
//
//sp:native TF2_GetClassIndexFromString
func ClassIndexFromString(name Text) Class { return seatings.ClassIndexFromString(name) }
