package engine

/*
The shopping trip: what a bot may still build, what it has already spent, and
how fast it presses the buy button.
*/

// ShoppingCalls are the answers.
type ShoppingCalls struct {
	NativeCell                 func(position int32) int32
	RangeRepairStallsOf        func(client int32) int32
	GiveItemToPlayerNamed      func(client int32, classname Text, itemDefIndex int32, level int32, quality int32) int32
	RemoveWeaponSlot           func(client int32, slot int32)
	TranslateWeaponEntForClass func(classname Text, maxlen int32, playerClass Class)
	IsShieldEquipped           func(client int32) bool
	GivePlayerAmmo             func(client int32, amount int32, ammoType int32, suppressSound bool) int32
	SetMaxHealth               func(entity int32, health int32)
	PostInventoryApplication   func(client int32)
	RoundToNearest             func(value float32) int32
	SessionWallet              func(client int32) int32
	SpentOnUpgrade             func(client int32, index int32) int32
	WaveHasExplosiveRobots     func() bool
	WaveHasBulletRobots        func() bool
	WaveHasFireRobots          func() bool
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

/*
Handing a bot its loadout.

Every one of these is part of putting a weapon in a bot's hands after the game
has already given it the stock one, which is why they all run from one timer a
tenth of a second after the spawn.
*/

// RemoveWeaponSlot takes whatever the game put in that slot back out.
//
//sp:native TF2_RemoveWeaponSlot
func RemoveWeaponSlot(client int32, slot int32) {
	if shopping.RemoveWeaponSlot == nil {
		missing("TF2_RemoveWeaponSlot")
	}
	shopping.RemoveWeaponSlot(client, slot)
}

// TranslateWeaponEntForClass rewrites a classname into the one that class
// actually spawns, in place: the schema files several weapons under a name no
// class uses.
//
//sp:native TF2Econ_TranslateWeaponEntForClass
//nolint:revive // unused-parameter: the buffer is rewritten in place, which is the whole call
func TranslateWeaponEntForClass(classname Text, maxlen int32, playerClass Class) {
	if shopping.TranslateWeaponEntForClass == nil {
		missing("TF2Econ_TranslateWeaponEntForClass")
	}
	shopping.TranslateWeaponEntForClass(classname, maxlen, playerClass)
}

// IsShieldEquipped says the demoman is carrying a shield, which occupies the
// secondary slot and must not be replaced.
//
//sp:native TF2_IsShieldEquipped
func IsShieldEquipped(client int32) bool {
	if shopping.IsShieldEquipped == nil {
		missing("TF2_IsShieldEquipped")
	}
	return shopping.IsShieldEquipped(client)
}

// GivePlayerAmmo fills one ammo type.
//
//sp:native GivePlayerAmmo
func GivePlayerAmmo(client int32, amount int32, ammoType int32, suppressSound bool) int32 {
	if shopping.GivePlayerAmmo == nil {
		missing("GivePlayerAmmo")
	}
	return shopping.GivePlayerAmmo(client, amount, ammoType, suppressSound)
}

// SetMaxHealth writes the entity's maximum, which a weapon's attribute can
// have moved.
//
//sp:native BaseEntity_SetMaxHealth
func SetMaxHealth(entity int32, health int32) {
	if shopping.SetMaxHealth == nil {
		missing("BaseEntity_SetMaxHealth")
	}
	shopping.SetMaxHealth(entity, health)
}

// PostInventoryApplication tells the game the loadout is settled, which is what
// makes the new weapons real to it.
//
//sp:native PostInventoryApplication
func PostInventoryApplication(client int32) {
	if shopping.PostInventoryApplication == nil {
		missing("PostInventoryApplication")
	}
	shopping.PostInventoryApplication(client)
}

// WeaponSlotBuilding is TFWeaponSlot_Building, the spy's sapper and the
// engineer's PDA.
//
//sp:global TFWeaponSlot_Building
func WeaponSlotBuilding() int32 { return 3 }

// AmmoTypeCount is TF_AMMO_COUNT, one past the last ammo type, which is what
// the refill loop stops at. Not AmmoCount, which is how much of one a player
// is carrying.
//
//sp:global TF_AMMO_COUNT
func AmmoTypeCount() int32 { return 0 }

// GiveItemToPlayerNamed hands over an item whose classname came out of the
// schema rather than being written out. Ported, econitem.
//
//sp:body GiveItemToPlayer
func GiveItemToPlayerNamed(client int32, classname Text, itemDefIndex int32, level int32, quality int32) int32 {
	if shopping.GiveItemToPlayerNamed == nil {
		missing("GiveItemToPlayer")
	}
	return shopping.GiveItemToPlayerNamed(client, classname, itemDefIndex, level, quality)
}

/*
The stats plugin's side of the boundary.

Each of the six natives the mod exports reads one thing about one bot and hands
it over. GetNativeCell is how a native's argument arrives.
*/

// NativeCell is the argument at that position, which is how SourceMod passes
// one into a native.
//
//sp:native GetNativeCell
func NativeCell(position int32) int32 {
	if shopping.NativeCell == nil {
		missing("GetNativeCell")
	}
	return shopping.NativeCell(position)
}

// RangeRepairStallsOf is how often this engineer fired bolts at a sentry that
// gained nothing. Ported, engineeridle.
//
//sp:body RangeRepairStallsOf
func RangeRepairStallsOf(client int32) int32 {
	if shopping.RangeRepairStallsOf == nil {
		missing("RangeRepairStallsOf")
	}
	return shopping.RangeRepairStallsOf(client)
}
