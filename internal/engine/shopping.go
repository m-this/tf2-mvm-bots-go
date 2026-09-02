package engine

/*
The shopping trip: what a bot may still build, what it has already spent, and
how fast it presses the buy button.
*/

// ShoppingCalls are the answers.
type ShoppingCalls struct {
	RoundToNearest         func(value float32) int32
	SessionWallet          func(client int32) int32
	SpentOnUpgrade         func(client int32, index int32) int32
	WaveHasExplosiveRobots func() bool
	WaveHasBulletRobots    func() bool
	WaveHasFireRobots      func() bool
}

var shopping ShoppingCalls

// InstallShopping puts a set of answers behind them.
func InstallShopping(c ShoppingCalls) func() {
	previous := shopping
	shopping = c
	return func() { shopping = previous }
}

// MaxUpgrades is MAX_UPGRADES, how many attributes the station offers.
//
//sp:global MAX_UPGRADES
func MaxUpgrades() int32 { return 128 }

// RoundToNearest is SourcePawn's rounding, which Go has no operator for.
//
//sp:native RoundToNearest
func RoundToNearest(value float32) int32 {
	if shopping.RoundToNearest == nil {
		missing("RoundToNearest")
	}
	return shopping.RoundToNearest(value)
}

// UpgradeInterval is redbots_manager_bot_upgrade_interval: a server owner's
// override on how fast a bot shops, negative when they have not set one.
//
//sp:global redbots_manager_bot_upgrade_interval
func UpgradeInterval() ConVar { return 0 }

// WaveHasExplosiveRobots says the coming wave deals blast damage, which is what
// prices a blast resistance. Ported, mission.
//
//sp:body WaveHasExplosiveRobots
func WaveHasExplosiveRobots() bool {
	if shopping.WaveHasExplosiveRobots == nil {
		missing("WaveHasExplosiveRobots")
	}
	return shopping.WaveHasExplosiveRobots()
}

// WaveHasBulletRobots is the same for bullets. Ported, mission.
//
//sp:body WaveHasBulletRobots
func WaveHasBulletRobots() bool {
	if shopping.WaveHasBulletRobots == nil {
		missing("WaveHasBulletRobots")
	}
	return shopping.WaveHasBulletRobots()
}

// WaveHasFireRobots is the same for fire. Ported, mission.
//
//sp:body WaveHasFireRobots
func WaveHasFireRobots() bool {
	if shopping.WaveHasFireRobots == nil {
		missing("WaveHasFireRobots")
	}
	return shopping.WaveHasFireRobots()
}
