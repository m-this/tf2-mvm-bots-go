package engine

/*
What the query layer of nextbot_behavior.sp reaches: ammo, money, the buyback
dice, and the intention interface a rethink goes through.
*/

// QueryCalls are the answers.
type QueryCalls struct {
	PlayerMaxAmmo              func(client int32, ammo int32) int32
	Currency                   func(client int32) int32
	BuybackNumber              func(client int32) int32
	BuyUpgradesNumber          func(client int32) int32
	BeingRevived               func(client int32) bool
	AreaID                     func(n NavArea) int32
	SetLookingAroundForEnemies func(client int32, allow bool)
	Intention                  func(b Bot) Intention
	IntentionReset             func(i Intention)
	TunedWeaponRanges          func(weapon int32) (bool, float32, float32)
	VisibleInFOVNow            func(k Known) bool
	RangeSquaredTo             func(b Bot, entity int32) float32
	RemoveCondition            func(client int32, condition Condition)
	ShouldTakeUpPosition       func(client int32) bool
	IsWaitingAtTheFront        func(client int32) bool
	RealPlayerCount            func() int32
	AnyHumanOnRed              func() bool
	AnyHumanReadyOnRed         func() bool
	IsTFBotPlayer              func(client int32) bool
	VisibleRecently            func(k Known) bool
	LastKnownPosition          func(k Known) [3]float32
	IsRangeGreaterThanEntity   func(b Bot, entity int32, distance float32) bool
	UseActionSlot              func(client int32)
	PowerupBottleCharges       func(bottle int32) int32
	PowerupBottleKind          func(bottle int32) int32
	FindPowerupBottle          func(client int32) int32
	IsPointInRespawnRoomStrict func(position [3]float32, client int32) bool
}

var queries QueryCalls

// InstallQueries puts a set of answers behind them.
func InstallQueries(c QueryCalls) func() {
	previous := queries
	queries = c
	return func() { queries = previous }
}

// PlayerMaxAmmo is how much of that ammo the player can carry.
//
//sp:native TF2Util_GetPlayerMaxAmmo
func PlayerMaxAmmo(client int32, ammo int32) int32 {
	if queries.PlayerMaxAmmo == nil {
		missing("TF2Util_GetPlayerMaxAmmo")
	}
	return queries.PlayerMaxAmmo(client, ammo)
}

// AmmoPrimary is TF_AMMO_PRIMARY.
//
//sp:global TF_AMMO_PRIMARY
func AmmoPrimary() int32 { return 1 }

// AmmoSecondary is TF_AMMO_SECONDARY.
//
//sp:global TF_AMMO_SECONDARY
func AmmoSecondary() int32 { return 2 }

// AmmoMetal is TF_AMMO_METAL.
//
//sp:global TF_AMMO_METAL
func AmmoMetal() int32 { return 3 }

// Currency is what the player holds right now.
//
//sp:native TF2_GetCurrency
func Currency(client int32) int32 {
	if queries.Currency == nil {
		missing("TF2_GetCurrency")
	}
	return queries.Currency(client)
}

// BuybackCostPerSecond is MVM_BUYBACK_COST_PER_SEC, what a second of respawn
// time costs to skip.
//
//sp:global MVM_BUYBACK_COST_PER_SEC
func BuybackCostPerSecond() int32 { return 5 }

// BuybackNumber is the die this bot rolled for buying back, set on spawn.
//
//sp:slot g_iBuybackNumber
func BuybackNumber(client int32) int32 {
	if queries.BuybackNumber == nil {
		missing("g_iBuybackNumber")
	}
	return queries.BuybackNumber(client)
}

// BuyUpgradesNumber is the die for shopping mid-round, set on spawn.
//
//sp:slot g_iBuyUpgradesNumber
func BuyUpgradesNumber(client int32) int32 {
	if queries.BuyUpgradesNumber == nil {
		missing("g_iBuyUpgradesNumber")
	}
	return queries.BuyUpgradesNumber(client)
}

// BeingRevived says somebody has a revive marker up over this player.
//
//sp:slot g_bIsBeingRevived
func BeingRevived(client int32) bool {
	if queries.BeingRevived == nil {
		missing("g_bIsBeingRevived")
	}
	return queries.BeingRevived(client)
}

// BuybackChance is redbots_manager_bot_buyback_chance.
//
//sp:global redbots_manager_bot_buyback_chance
func BuybackChance() ConVar { return 0 }

// BuyUpgradesChance is redbots_manager_bot_buy_upgrades_chance.
//
//sp:global redbots_manager_bot_buy_upgrades_chance
func BuyUpgradesChance() ConVar { return 0 }

// PathLookaheadRange is tf_bot_path_lookahead_range, the game's own convar,
// found and kept by the plugin.
//
//sp:global tf_bot_path_lookahead_range
func PathLookaheadRange() ConVar { return 0 }

// ID is the nav area's own number, stable across a session.
//
//sp:method GetID
func (n NavArea) ID() int32 {
	if queries.AreaID == nil {
		missing("CNavArea.GetID")
	}
	return queries.AreaID(n)
}

// NoNavArea is what GetLastKnownArea answers for a bot that has never stood on
// the mesh.
//
//sp:global NULL_AREA
func NoNavArea() NavArea { return 0 }

// BombHatchRangeCritical is BOMB_HATCH_RANGE_CRITICAL, declared by the
// generated campbomb file.
//
//sp:global BOMB_HATCH_RANGE_CRITICAL
func BombHatchRangeCritical() float32 { return 1000.0 }

// SetLookingAroundForEnemies turns the game's own scanning on or off.
//
//sp:plugin SetLookingAroundForEnemies
func SetLookingAroundForEnemies(client int32, allow bool) {
	if queries.SetLookingAroundForEnemies == nil {
		missing("SetLookingAroundForEnemies")
	}
	queries.SetLookingAroundForEnemies(client, allow)
}

// Intention is IIntention, the interface a bot's behaviour hangs off.
//
//sp:tag IIntention
type Intention int32

// Intention is the bot's.
//
//sp:method GetIntentionInterface
func (b Bot) Intention() Intention {
	if queries.Intention == nil {
		missing("INextBot.GetIntentionInterface")
	}
	return queries.Intention(b)
}

// Reset throws the behaviour away so the next update rebuilds it.
//
//sp:method Reset
func (i Intention) Reset() {
	if queries.IntentionReset == nil {
		missing("IIntention.Reset")
	}
	queries.IntentionReset(i)
}

// TunedWeaponRanges is the range table the loadout tuning emits. Ported,
// internal/tables.
//
//sp:body GetTunedWeaponRanges
func TunedWeaponRanges(weapon int32) (found bool, desired float32, maxRange float32) {
	if queries.TunedWeaponRanges == nil {
		missing("GetTunedWeaponRanges")
	}
	return queries.TunedWeaponRanges(weapon)
}

// FeatureSoldierClosesIn is FEATURE_SOLDIER_CLOSES_IN.
//
//sp:global FEATURE_SOLDIER_CLOSES_IN
func FeatureSoldierClosesIn() int32 { return 21 }

// SoldierRocketSettle is the distance the soldier settles at when the feature
// is on. Ported, internal/tables.
//
//sp:global SOLDIER_ROCKET_SETTLE
func SoldierRocketSettle() float32 { return 0 }

// DemoPipeSettle is the demoman's, always on.
//
//sp:global DEMO_PIPE_SETTLE
func DemoPipeSettle() float32 { return 0 }

// FloatMax is FLT_MAX, the range that means never stop closing.
//
//sp:global FLT_MAX
func FloatMax() float32 { return 3.4e38 }

// WeaponRocketLauncher is TF_WEAPON_ROCKETLAUNCHER.
//
//sp:global TF_WEAPON_ROCKETLAUNCHER
func WeaponRocketLauncher() Weapon { return 22 }

// WeaponGrenadeLauncher is TF_WEAPON_GRENADELAUNCHER.
//
//sp:global TF_WEAPON_GRENADELAUNCHER
func WeaponGrenadeLauncher() Weapon { return 23 }

// WeaponPDA is TF_WEAPON_PDA.
//
//sp:global TF_WEAPON_PDA
func WeaponPDA() Weapon { return 45 }

// WeaponPDAEngineerBuild is TF_WEAPON_PDA_ENGINEER_BUILD.
//
//sp:global TF_WEAPON_PDA_ENGINEER_BUILD
func WeaponPDAEngineerBuild() Weapon { return 46 }

// WeaponPDAEngineerDestroy is TF_WEAPON_PDA_ENGINEER_DESTROY.
//
//sp:global TF_WEAPON_PDA_ENGINEER_DESTROY
func WeaponPDAEngineerDestroy() Weapon { return 47 }

// WeaponPDASpy is TF_WEAPON_PDA_SPY.
//
//sp:global TF_WEAPON_PDA_SPY
func WeaponPDASpy() Weapon { return 48 }

// WeaponPumpkinBomb is TF_WEAPON_PUMPKIN_BOMB.
//
//sp:global TF_WEAPON_PUMPKIN_BOMB
func WeaponPumpkinBomb() Weapon { return 63 }

// VisibleInFOVNow says the bot can see it this frame, in its cone.
//
//sp:method IsVisibleInFOVNow
func (k Known) VisibleInFOVNow() bool {
	if queries.VisibleInFOVNow == nil {
		missing("CKnownEntity.IsVisibleInFOVNow")
	}
	return queries.VisibleInFOVNow(k)
}

// RangeSquaredTo is the bot's distance to the entity, squared, which is the
// form the game compares in.
//
//sp:method GetRangeSquaredTo
func (b Bot) RangeSquaredTo(entity int32) float32 {
	if queries.RangeSquaredTo == nil {
		missing("INextBot.GetRangeSquaredTo")
	}
	return queries.RangeSquaredTo(b, entity)
}

// ConditionTaunting is TFCond_Taunting.
//
//sp:global TFCond_Taunting
func ConditionTaunting() Condition { return 7 }

// RemoveCondition takes one off.
//
//sp:native TF2_RemoveCondition
func RemoveCondition(client int32, condition Condition) {
	if queries.RemoveCondition == nil {
		missing("TF2_RemoveCondition")
	}
	queries.RemoveCondition(client, condition)
}

// UseUpgrades is redbots_manager_bot_use_upgrades, whether the bots shop at
// all.
//
//sp:global redbots_manager_bot_use_upgrades
func UseUpgrades() ConVar { return 0 }

// FeatureReadyWhenPrepared is FEATURE_READY_WHEN_PREPARED.
//
//sp:global FEATURE_READY_WHEN_PREPARED
func FeatureReadyWhenPrepared() int32 { return 5 }

// ShouldTakeUpPosition says this bot's class walks to the front before the
// wave rather than waiting where it shopped.
//
//sp:plugin ShouldTakeUpPosition
func ShouldTakeUpPosition(client int32) bool {
	if queries.ShouldTakeUpPosition == nil {
		missing("ShouldTakeUpPosition")
	}
	return queries.ShouldTakeUpPosition(client)
}

// IsWaitingAtTheFront says he got there. Ported, movetofront.
//
//sp:body IsWaitingAtTheFront
func IsWaitingAtTheFront(client int32) bool {
	if queries.IsWaitingAtTheFront == nil {
		missing("IsWaitingAtTheFront")
	}
	return queries.IsWaitingAtTheFront(client)
}

// RealPlayerCount is how many humans are on the server.
//
//sp:plugin GetRealPlayerCount
func RealPlayerCount() int32 {
	if queries.RealPlayerCount == nil {
		missing("GetRealPlayerCount")
	}
	return queries.RealPlayerCount()
}

// AnyHumanOnRed says the readiness is a person's call, not the bots'.
//
//sp:plugin AnyHumanOnRed
func AnyHumanOnRed() bool {
	if queries.AnyHumanOnRed == nil {
		missing("AnyHumanOnRed")
	}
	return queries.AnyHumanOnRed()
}

// AnyHumanReadyOnRed is what that person has said so far.
//
//sp:plugin AnyHumanReadyOnRed
func AnyHumanReadyOnRed() bool {
	if queries.AnyHumanReadyOnRed == nil {
		missing("AnyHumanReadyOnRed")
	}
	return queries.AnyHumanReadyOnRed()
}

// FeatureMedicAnswersCall is FEATURE_MEDIC_ANSWERS_CALL.
//
//sp:global FEATURE_MEDIC_ANSWERS_CALL
func FeatureMedicAnswersCall() int32 { return 20 }

// IsTFBotPlayer says the slot is one of the game's own bots. Ported, mission.
//
//sp:body IsTFBotPlayer
func IsTFBotPlayer(client int32) bool {
	if queries.IsTFBotPlayer == nil {
		missing("IsTFBotPlayer")
	}
	return queries.IsTFBotPlayer(client)
}

// ConditionSlowed is TFCond_Slowed.
//
//sp:global TFCond_Slowed
func ConditionSlowed() Condition { return 0 }

// VisibleRecently says the bot saw it within the vision system's memory.
//
//sp:method IsVisibleRecently
func (k Known) VisibleRecently() bool {
	if queries.VisibleRecently == nil {
		missing("CKnownEntity.IsVisibleRecently")
	}
	return queries.VisibleRecently(k)
}

// LastKnownPosition is where the bot last saw it.
//
//sp:method GetLastKnownPosition
func (k Known) LastKnownPosition() (position [3]float32) {
	if queries.LastKnownPosition == nil {
		missing("CKnownEntity.GetLastKnownPosition")
	}
	return queries.LastKnownPosition(k)
}

// ConditionCritMmmph is TFCond_CritMmmph, the Phlogistinator's crit boost.
//
//sp:global TFCond_CritMmmph
func ConditionCritMmmph() Condition { return 44 }

// WeaponFlameBall is TF_WEAPON_FLAME_BALL, the Dragon's Fury.
//
//sp:global TF_WEAPON_FLAME_BALL
func WeaponFlameBall() Weapon { return 109 }

// FlameballReachRange is FLAMEBALL_REACH_RANGE, the Dragon's Fury's longer
// reach.
//
//sp:global FLAMEBALL_REACH_RANGE
func FlameballReachRange() float32 { return 526.0 }

// IsRangeGreaterThanEntity is the entity form of the range question.
//
//sp:method IsRangeGreaterThan
func (b Bot) IsRangeGreaterThanEntity(entity int32, distance float32) bool {
	if queries.IsRangeGreaterThanEntity == nil {
		missing("INextBot.IsRangeGreaterThan")
	}
	return queries.IsRangeGreaterThanEntity(b, entity, distance)
}

// UseActionSlot drinks whatever is in the action slot. Ported, stocks.
//
//sp:body UseActionSlotItem
func UseActionSlot(client int32) {
	if queries.UseActionSlot == nil {
		missing("UseActionSlotItem")
	}
	queries.UseActionSlot(client)
}

// PowerupBottleCharges is how many drinks are left. Ported, state.
//
//sp:body PowerupBottle_GetNumCharges
func PowerupBottleCharges(bottle int32) int32 {
	if queries.PowerupBottleCharges == nil {
		missing("PowerupBottle_GetNumCharges")
	}
	return queries.PowerupBottleCharges(bottle)
}

// PowerupBottleKind is what the canteen does. Ported, state.
//
//sp:body PowerupBottle_GetType
func PowerupBottleKind(bottle int32) int32 {
	if queries.PowerupBottleKind == nil {
		missing("PowerupBottle_GetType")
	}
	return queries.PowerupBottleKind(bottle)
}

// FindPowerupBottle walks the wearables for the canteen. Ported, state.
//
//sp:body GetPowerupBottle
func FindPowerupBottle(client int32) int32 {
	if queries.FindPowerupBottle == nil {
		missing("GetPowerupBottle")
	}
	return queries.FindPowerupBottle(client)
}

// The canteen kinds, the schema's own order.
const (
	//nolint:unused // the port reaches the rest of the switch as it needs them
	powerupBottleNone = iota
	bottleCritBoost
	bottleUberCharge
	bottleRecall
	bottleRefillAmmo
	bottleBuildingsInstantUpgrade
)

// BottleCritBoost is POWERUP_BOTTLE_CRITBOOST.
//
//sp:global POWERUP_BOTTLE_CRITBOOST
func BottleCritBoost() int32 { return bottleCritBoost }

// BottleUberCharge is POWERUP_BOTTLE_UBERCHARGE.
//
//sp:global POWERUP_BOTTLE_UBERCHARGE
func BottleUberCharge() int32 { return bottleUberCharge }

// BottleRecall is POWERUP_BOTTLE_RECALL.
//
//sp:global POWERUP_BOTTLE_RECALL
func BottleRecall() int32 { return bottleRecall }

// BottleRefillAmmo is POWERUP_BOTTLE_REFILL_AMMO.
//
//sp:global POWERUP_BOTTLE_REFILL_AMMO
func BottleRefillAmmo() int32 { return bottleRefillAmmo }

// BottleBuildingsInstantUpgrade is POWERUP_BOTTLE_BUILDINGS_INSTANT_UPGRADE.
//
//sp:global POWERUP_BOTTLE_BUILDINGS_INSTANT_UPGRADE
func BottleBuildingsInstantUpgrade() int32 { return bottleBuildingsInstantUpgrade }

// IsPointInRespawnRoomStrict is the three-argument form: the room must be the
// player's own team's.
//
//sp:native TF2Util_IsPointInRespawnRoom after true
func IsPointInRespawnRoomStrict(position [3]float32, client int32) bool {
	if queries.IsPointInRespawnRoomStrict == nil {
		missing("TF2Util_IsPointInRespawnRoom")
	}
	return queries.IsPointInRespawnRoomStrict(position, client)
}
