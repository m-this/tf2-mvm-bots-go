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
func (d Damage) Amount() float32 {
	if medics.DamageAmount == nil {
		missing("CTakeDamageInfo.GetDamage")
	}
	return medics.DamageAmount(d)
}

// IsRangeLessThanEx says whether the bot is closer than that to a position.
//
//sp:method IsRangeLessThanEx
func (b Bot) IsRangeLessThanEx(position [3]float32, distance float32) bool {
	if medics.IsRangeLessThanEx == nil {
		missing("INextBot.IsRangeLessThanEx")
	}
	return medics.IsRangeLessThanEx(b, position, distance)
}

// IsPathToVectorPossible says whether the bot could actually walk there, which
// is a whole nav mesh search and not the cheap predicate it reads as.
//
//sp:plugin IsPathToVectorPossible
func IsPathToVectorPossible(client int32, position [3]float32) bool {
	if medics.IsPathToVectorPossible == nil {
		missing("IsPathToVectorPossible")
	}
	return medics.IsPathToVectorPossible(client, position)
}

// IsPathToVectorPossibleLength is the same search, answering how long the route
// it found was. The plugin declares the length as a defaulted by-reference
// parameter, so a caller that wants it takes it as a second result.
//
//sp:plugin IsPathToVectorPossible
func IsPathToVectorPossibleLength(client int32, position [3]float32) (ok bool, length float32) {
	if medics.IsPathToVectorPossibleLength == nil {
		missing("IsPathToVectorPossible")
	}
	return medics.IsPathToVectorPossibleLength(client, position)
}

// NearestReviveMarker is the closest reanimator a medic could pick up.
//
//sp:plugin GetNearestReviveMarker
func NearestReviveMarker(client int32, maxDistance float32) int32 {
	if medics.NearestReviveMarker == nil {
		missing("GetNearestReviveMarker")
	}
	return medics.NearestReviveMarker(client, maxDistance)
}

// AbsOriginOf is util.sp's GetAbsOrigin, ported: internal/body/scan generates it.
//
//sp:body GetAbsOrigin returns
func AbsOriginOf(entity int32) [3]float32 {
	if medics.AbsOrigin == nil {
		missing("GetAbsOrigin")
	}
	return medics.AbsOrigin(entity)
}

// WeaponSlotPrimary is TFWeaponSlot_Primary.
//
//sp:global TFWeaponSlot_Primary
func WeaponSlotPrimary() int32 { return 0 }
