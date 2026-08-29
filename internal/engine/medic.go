package engine

/*
What the medic behaviours reach.

The uber attack is a medic hitting somebody with a melee weapon to fill his own
charge, which is why so much of this is about weapon slots and charge levels.
*/

// MedicCalls are the answers.
type MedicCalls struct {
	ClientHealth          func(client int32) int32
	EntPropFloat          func(entity int32, propType PropType, prop string) float32
	IsRangeGreaterThanEx  func(bot Bot, position [3]float32, distance float32) bool
	IsTaunting            func(client int32) bool
	IsPlayerMoving        func(client int32) bool
	CanWeaponAddUberOnHit func(weapon int32) bool
	EnemyNearestToMe      func(client int32, maxDistance float32, giantsOnly bool, ignoreUber bool, stunnedOnly bool, class Class) int32
}

var medics MedicCalls

// InstallMedics puts a set of answers behind them.
func InstallMedics(c MedicCalls) func() {
	previous := medics
	medics = c
	return func() { medics = previous }
}

// WeaponSlotMelee is TFWeaponSlot_Melee.
//
//sp:global TFWeaponSlot_Melee
func WeaponSlotMelee() int32 { return 2 }

// MeleeAttackRange is TFBOT_MELEE_ATTACK_RANGE, how close a swing lands.
//
//sp:global TFBOT_MELEE_ATTACK_RANGE
func MeleeAttackRange() float32 { return 100.0 }

// NullVector is NULL_VECTOR, which is what a forgotten position is set to.
//
//sp:global NULL_VECTOR
func NullVector() [3]float32 { return [3]float32{} }

// ClientHealth is what the player has left.
//
//sp:native GetClientHealth
func ClientHealth(client int32) int32 {
	if medics.ClientHealth == nil {
		missing("GetClientHealth")
	}
	return medics.ClientHealth(client)
}

// EntPropFloatOf reads a float from one of the entity's property tables.
//
//sp:native GetEntPropFloat
func EntPropFloatOf(entity int32, propType PropType, prop string) float32 {
	if medics.EntPropFloat == nil {
		missing("GetEntPropFloat")
	}
	return medics.EntPropFloat(entity, propType, prop)
}

// IsRangeGreaterThanEx says whether the bot has strayed further than that from
// a position.
//
//sp:method IsRangeGreaterThanEx
func (b Bot) IsRangeGreaterThanEx(position [3]float32, distance float32) bool {
	if medics.IsRangeGreaterThanEx == nil {
		missing("INextBot.IsRangeGreaterThanEx")
	}
	return medics.IsRangeGreaterThanEx(b, position, distance)
}

// IsTaunting says whether the player is in the middle of a taunt.
//
//sp:native TF2_IsTaunting
func IsTaunting(client int32) bool {
	if medics.IsTaunting == nil {
		missing("TF2_IsTaunting")
	}
	return medics.IsTaunting(client)
}

// IsPlayerMoving says whether the player is going anywhere.
//
//sp:plugin IsPlayerMoving
func IsPlayerMoving(client int32) bool {
	if medics.IsPlayerMoving == nil {
		missing("IsPlayerMoving")
	}
	return medics.IsPlayerMoving(client)
}

// CanWeaponAddUberOnHit says whether hitting somebody with it fills the charge.
//
//sp:plugin CanWeaponAddUberOnHit
func CanWeaponAddUberOnHit(weapon int32) bool {
	if medics.CanWeaponAddUberOnHit == nil {
		missing("CanWeaponAddUberOnHit")
	}
	return medics.CanWeaponAddUberOnHit(weapon)
}

// EnemyNearestToMe is util.sp:1183, ported: internal/body/scan generates it.
//
//sp:body FindEnemyNearestToMe
func EnemyNearestToMe(client int32, maxDistance float32, giantsOnly bool, ignoreUber bool, stunnedOnly bool, class Class) int32 {
	if medics.EnemyNearestToMe == nil {
		missing("FindEnemyNearestToMe")
	}
	return medics.EnemyNearestToMe(client, maxDistance, giantsOnly, ignoreUber, stunnedOnly, class)
}
