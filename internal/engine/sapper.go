package engine

/*
What the spy needs to sap a player rather than a building.

Same pattern as everything else here: SourceMod's and the vendored stocklib's
are natives, the plugin's are //sp:plugin and go when the port reaches them, and
the four scans are //sp:body because internal/body/scan already generates them.
*/

// SapperCalls are the answers.
type SapperCalls struct {
	IsValidClientIndex func(client int32) bool
	SubtractVectors    func(a [3]float32, b [3]float32) [3]float32
	VectorLength       func(v [3]float32) float32
	CanWeaponAttack    func(weapon int32) bool
	AttribByName       func(entity int32, name string) int32
	AmmoCount          func(client int32, ammo int32) int32
	RemoveAmmo         func(client int32, count int32, ammo int32)
	SetEntPropFloat    func(entity int32, propType PropType, prop string, value float32)
	SpawnSapper        func(owner int32, entity int32, weapon int32) int32

	NearestSappablePlayer               func(client int32, maxDistance float32, giantsOnly bool, class Class, speedCheck float32) int32
	FarthestSappablePlayer              func(client int32, maxDistance float32, giantsOnly bool, class Class, speedCheck float32) int32
	NearestSappablePlayerHealingSomeone func(client int32, maxDistance float32, giantsOnly bool, class Class, speedCheck float32) int32
	NearestEnemyCount                   func(client int32, maxDistance float32, ignoreUber bool) int32
	PlayerSappable                      func(client int32) bool
	PlayerEnemyTeam                     func(client int32) Team
}

var sappers SapperCalls

// InstallSappers puts a set of answers behind them.
func InstallSappers(c SapperCalls) func() {
	previous := sappers
	sappers = c
	return func() { sappers = previous }
}

// SapperPlayerBuildOnRange is how close the spy has to be to sap a player.
//
//sp:global SAPPER_PLAYER_BUILD_ON_RANGE
func SapperPlayerBuildOnRange() float32 { return 160.0 }

// SapperRechargeTime is how long the robo sapper takes to come back.
//
//sp:global SAPPER_RECHARGE_TIME
func SapperRechargeTime() float32 { return 15.0 }

// AmmoGrenades2 is TF_AMMO_GRENADES2, which is what the builder spends.
//
//sp:global TF_AMMO_GRENADES2
func AmmoGrenades2() int32 { return 6 }

// WeaponBuilder is TF_WEAPON_BUILDER.
//
//sp:global TF_WEAPON_BUILDER
func WeaponBuilder() Weapon { return 26 }

// AddressNull is Address_Null, which is what an absent attribute reads as.
//
//sp:global Address_Null
func AddressNull() int32 { return 0 }

// ClassMedic is TFClass_Medic.
//
//sp:global TFClass_Medic
func ClassMedic() Class { return 5 }

// IsValidClientIndex says whether the number could be a client at all.
//
//sp:plugin IsValidClientIndex
func IsValidClientIndex(client int32) bool {
	if sappers.IsValidClientIndex == nil {
		missing("IsValidClientIndex")
	}
	return sappers.IsValidClientIndex(client)
}

// SubtractVectors is a - b, filling the array it is given.
//
//sp:native SubtractVectors
func SubtractVectors(a [3]float32, b [3]float32) (difference [3]float32) {
	if sappers.SubtractVectors == nil {
		missing("SubtractVectors")
	}
	return sappers.SubtractVectors(a, b)
}

// VectorLength is how long the vector is.
//
//sp:native GetVectorLength
func VectorLength(v [3]float32) float32 {
	if sappers.VectorLength == nil {
		missing("GetVectorLength")
	}
	return sappers.VectorLength(v)
}

// CanWeaponAttack says whether the weapon is off cooldown.
//
//sp:native TF2Util_CanWeaponAttack
func CanWeaponAttack(weapon int32) bool {
	if sappers.CanWeaponAttack == nil {
		missing("TF2Util_CanWeaponAttack")
	}
	return sappers.CanWeaponAttack(weapon)
}

// AttribByName is the attribute's address, and Address_Null when it has none.
//
//sp:native TF2Attrib_GetByName
func AttribByName(entity int32, name string) int32 {
	if sappers.AttribByName == nil {
		missing("TF2Attrib_GetByName")
	}
	return sappers.AttribByName(entity, name)
}

// AmmoCount is how much of that ammo the player is carrying.
//
//sp:native BaseCombatCharacter_GetAmmoCount
func AmmoCount(client int32, ammo int32) int32 {
	if sappers.AmmoCount == nil {
		missing("BaseCombatCharacter_GetAmmoCount")
	}
	return sappers.AmmoCount(client, ammo)
}

// RemoveAmmo takes some away.
//
//sp:native BaseCombatCharacter_RemoveAmmo
func RemoveAmmo(client int32, count int32, ammo int32) {
	if sappers.RemoveAmmo == nil {
		missing("BaseCombatCharacter_RemoveAmmo")
	}
	sappers.RemoveAmmo(client, count, ammo)
}

// SetEntPropFloat writes a float into one of the entity's property tables.
//
//sp:native SetEntPropFloat
func SetEntPropFloat(entity int32, propType PropType, prop string, value float32) {
	if sappers.SetEntPropFloat == nil {
		missing("SetEntPropFloat")
	}
	sappers.SetEntPropFloat(entity, propType, prop, value)
}

// SpawnSapper puts one on the entity, which takes five steps the plugin knows
// about and this port has not reached.
//
//sp:plugin SpawnSapper
func SpawnSapper(owner int32, entity int32, weapon int32) int32 {
	if sappers.SpawnSapper == nil {
		missing("SpawnSapper")
	}
	return sappers.SpawnSapper(owner, entity, weapon)
}

// The four scans, all of them ported: internal/body/scan generates each.

// NearestSappablePlayer is util.sp:1451.
//
//sp:body GetNearestSappablePlayer
func NearestSappablePlayer(client int32, maxDistance float32, giantsOnly bool, class Class, speedCheck float32) int32 {
	if sappers.NearestSappablePlayer == nil {
		missing("GetNearestSappablePlayer")
	}
	return sappers.NearestSappablePlayer(client, maxDistance, giantsOnly, class, speedCheck)
}

// FarthestSappablePlayer is util.sp:1501.
//
//sp:body GetFarthestSappablePlayer
func FarthestSappablePlayer(client int32, maxDistance float32, giantsOnly bool, class Class, speedCheck float32) int32 {
	if sappers.FarthestSappablePlayer == nil {
		missing("GetFarthestSappablePlayer")
	}
	return sappers.FarthestSappablePlayer(client, maxDistance, giantsOnly, class, speedCheck)
}

// NearestSappablePlayerHealingSomeone is util.sp:1638.
//
//sp:body GetNearestSappablePlayerHealingSomeone
func NearestSappablePlayerHealingSomeone(client int32, maxDistance float32, giantsOnly bool, class Class, speedCheck float32) int32 {
	if sappers.NearestSappablePlayerHealingSomeone == nil {
		missing("GetNearestSappablePlayerHealingSomeone")
	}
	return sappers.NearestSappablePlayerHealingSomeone(client, maxDistance, giantsOnly, class, speedCheck)
}

// NearestEnemyCount is util.sp:1398.
//
//sp:body GetNearestEnemyCount
func NearestEnemyCount(client int32, maxDistance float32, ignoreUber bool) int32 {
	if sappers.NearestEnemyCount == nil {
		missing("GetNearestEnemyCount")
	}
	return sappers.NearestEnemyCount(client, maxDistance, ignoreUber)
}

// PlayerSappable is util.sp:1437, ported: internal/body/scan generates it.
//
//sp:body IsPlayerSappable
func PlayerSappable(client int32) bool {
	if sappers.PlayerSappable == nil {
		missing("IsPlayerSappable")
	}
	return sappers.PlayerSappable(client)
}

// PlayerEnemyTeam is util.sp:1044, ported: internal/body/scan generates it.
//
//sp:body GetPlayerEnemyTeam
func PlayerEnemyTeam(client int32) Team {
	if sappers.PlayerEnemyTeam == nil {
		missing("GetPlayerEnemyTeam")
	}
	return sappers.PlayerEnemyTeam(client)
}
