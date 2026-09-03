package engine

/*
What the medic behaviours reach.

The uber attack is a medic hitting somebody with a melee weapon to fill his own
charge, which is why so much of this is about weapon slots and charge levels.
*/

// MedicCalls are the answers.
type MedicCalls struct {
	ClientHealth                 func(client int32) int32
	EntPropFloat                 func(entity int32, propType PropType, prop string) float32
	IsRangeGreaterThanEx         func(bot Bot, position [3]float32, distance float32) bool
	IsTaunting                   func(client int32) bool
	IsPlayerMoving               func(client int32) bool
	CanWeaponAddUberOnHit        func(weapon int32) bool
	EnemyNearestToMe             func(client int32, maxDistance float32, giantsOnly bool, ignoreUber bool, stunnedOnly bool, class Class) int32
	DamageAmount                 func(d Damage) float32
	IsRangeLessThanEx            func(bot Bot, position [3]float32, distance float32) bool
	IsPathToVectorPossible       func(client int32, position [3]float32) bool
	IsPathToVectorPossibleLength func(client int32, position [3]float32) (bool, float32)
	NearestReviveMarker          func(client int32, maxDistance float32) int32
	AbsOrigin                    func(entity int32) [3]float32
}

var medics MedicCalls

// InstallMedics puts a set of answers behind them.
func InstallMedics(c MedicCalls) func() {
	previous := medics
	Fill(&c)
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
func ClientHealth(client int32) int32 { return medics.ClientHealth(client) }

// EntPropFloatOf reads a float from one of the entity's property tables.
//
//sp:native GetEntPropFloat
func EntPropFloatOf(entity int32, propType PropType, prop string) float32 {
	return medics.EntPropFloat(entity, propType, prop)
}

// IsRangeGreaterThanEx says whether the bot has strayed further than that from
// a position.
//
//sp:method IsRangeGreaterThanEx
func (b Bot) IsRangeGreaterThanEx(position [3]float32, distance float32) bool {
	return medics.IsRangeGreaterThanEx(b, position, distance)
}

// IsTaunting says whether the player is in the middle of a taunt.
//
//sp:native TF2_IsTaunting
func IsTaunting(client int32) bool { return medics.IsTaunting(client) }

// IsPlayerMoving says whether the player is going anywhere.
//
//sp:body IsPlayerMoving
func IsPlayerMoving(client int32) bool { return medics.IsPlayerMoving(client) }

// CanWeaponAddUberOnHit says whether hitting somebody with it fills the charge.
//
//sp:body CanWeaponAddUberOnHit
func CanWeaponAddUberOnHit(weapon int32) bool { return medics.CanWeaponAddUberOnHit(weapon) }

// EnemyNearestToMe is util.sp:1183, ported: internal/body/scan generates it.
//
//sp:body FindEnemyNearestToMe
func EnemyNearestToMe(client int32, maxDistance float32, giantsOnly bool, ignoreUber bool, stunnedOnly bool, class Class) int32 {
	return medics.EnemyNearestToMe(client, maxDistance, giantsOnly, ignoreUber, stunnedOnly, class)
}

// Damage is CBaseNPC's CTakeDamageInfo, what the engine hands a behaviour when
// something hits the bot.
//
//sp:tag CTakeDamageInfo
type Damage int32

// WeaponMedigunRange is WEAPON_MEDIGUN_RANGE, how far the beam reaches.
//
//sp:global WEAPON_MEDIGUN_RANGE
func WeaponMedigunRange() float32 { return 450.0 }

// DamageOf reads the damage record out of the address the engine passed.
//
//sp:native CTakeDamageInfo
func DamageOf(address int32) Damage { return Damage(address) }

// Amount is how much damage it was.
//
//sp:method GetDamage
func (d Damage) Amount() float32 { return medics.DamageAmount(d) }

// IsRangeLessThanEx says whether the bot is closer than that to a position.
//
//sp:method IsRangeLessThanEx
func (b Bot) IsRangeLessThanEx(position [3]float32, distance float32) bool {
	return medics.IsRangeLessThanEx(b, position, distance)
}

// IsPathToVectorPossible says whether the bot could actually walk there, which
// is a whole nav mesh search and not the cheap predicate it reads as.
//
//sp:body IsPathToVectorPossible
func IsPathToVectorPossible(client int32, position [3]float32) bool {
	return medics.IsPathToVectorPossible(client, position)
}

// IsPathToVectorPossibleLength is the same search, answering how long the route
// it found was. The plugin declares the length as a defaulted by-reference
// parameter, so a caller that wants it takes it as a second result.
//
//sp:body IsPathToVectorPossible
func IsPathToVectorPossibleLength(client int32, position [3]float32) (ok bool, length float32) {
	return medics.IsPathToVectorPossibleLength(client, position)
}

// NearestReviveMarker is the closest reanimator a medic could pick up.
//
//sp:body GetNearestReviveMarker
func NearestReviveMarker(client int32, maxDistance float32) int32 {
	return medics.NearestReviveMarker(client, maxDistance)
}

// AbsOriginOf is util.sp's GetAbsOrigin, ported: internal/body/scan generates it.
//
//sp:body GetAbsOrigin returns
func AbsOriginOf(entity int32) [3]float32 { return medics.AbsOrigin(entity) }

// WeaponSlotPrimary is TFWeaponSlot_Primary.
//
//sp:global TFWeaponSlot_Primary
func WeaponSlotPrimary() int32 { return 0 }
