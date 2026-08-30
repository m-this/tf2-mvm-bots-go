package engine

/*
Finding health and ammo, and the convars that say how far a bot will walk for
either.

The four tf_bot_health_ convars are the game's own, and the mod reads them
rather than inventing its own distances: a bot on ten health looks further than
one on half.
*/

// HealthCalls are the answers.
type HealthCalls struct {
	PlayerMaxHealth             func(client int32) int32
	ClampFloat                  func(value float32, low float32, high float32) float32
	IsPointInRespawnRoom        func(position [3]float32) bool
	IsAmmoLow                   func(client int32) bool
	ComputeHealthAndAmmoVectors func(actor int32, into List, maxRange float32)
	IsCarryingObject            func(client int32) bool
	IsHealedByObject            func(client int32) bool
	IsHealedByMedic             func(client int32) bool
	WeaponIDIsSniperRifle       func(id Weapon) bool
	IsCritBoosted               func(client int32) bool
	HealthCritical              func() ConVar
	HealthOK                    func() ConVar
	HealthFarRange              func() ConVar
	HealthNearRange             func() ConVar
	AsFloat                     func(cell int32) float32
}

var healths HealthCalls

// InstallHealths puts a set of answers behind them.
func InstallHealths(c HealthCalls) func() {
	previous := healths
	healths = c
	return func() { healths = previous }
}

// ConceptMedic is MP_CONCEPT_PLAYER_MEDIC, the bot calling for one.
//
//sp:global MP_CONCEPT_PLAYER_MEDIC
func ConceptMedic() int32 { return 4 }

// ConditionZoomed is TFCond_Zoomed.
//
//sp:global TFCond_Zoomed
func ConditionZoomed() Condition { return 1 }

// The four the game uses to decide how far a hurt bot will walk for health.

// HealthCriticalRatio is the health fraction below which a bot will walk as
// far as it has to.
//
//sp:global tf_bot_health_critical_ratio
func HealthCriticalRatio() ConVar { return 0 }

// HealthOKRatio is the fraction above which it will not bother.
//
//sp:global tf_bot_health_ok_ratio
func HealthOKRatio() ConVar { return 0 }

// HealthSearchFarRange is how far a bot on full health looks.
//
//sp:global tf_bot_health_search_far_range
func HealthSearchFarRange() ConVar { return 0 }

// HealthSearchNearRange is how far a nearly dead one does.
//
//sp:global tf_bot_health_search_near_range
func HealthSearchNearRange() ConVar { return 0 }

// PlayerMaxHealth is what the bot would have at full.
//
//sp:plugin TEMP_GetPlayerMaxHealth
func PlayerMaxHealth(client int32) int32 {
	if healths.PlayerMaxHealth == nil {
		missing("TEMP_GetPlayerMaxHealth")
	}
	return healths.PlayerMaxHealth(client)
}

// ClampFloat holds a value between two others.
//
//sp:plugin ClampFloat
func ClampFloat(value float32, low float32, high float32) float32 {
	if healths.ClampFloat == nil {
		missing("ClampFloat")
	}
	return healths.ClampFloat(value, low, high)
}

// ComputeHealthAndAmmoVectors fills the list with what is within range, as an
// entity and a distance per entry.
//
//sp:body ComputeHealthAndAmmoVectors
func ComputeHealthAndAmmoVectors(actor int32, into List, maxRange float32) {
	if healths.ComputeHealthAndAmmoVectors == nil {
		missing("ComputeHealthAndAmmoVectors")
	}
	healths.ComputeHealthAndAmmoVectors(actor, into, maxRange)
}

// IsCarryingObject says the engineer has a building in his hands.
//
//sp:native TF2_IsCarryingObject
func IsCarryingObject(client int32) bool {
	if healths.IsCarryingObject == nil {
		missing("TF2_IsCarryingObject")
	}
	return healths.IsCarryingObject(client)
}

// IsHealedByObject says a dispenser is doing it.
//
//sp:body IsHealedByObject
func IsHealedByObject(client int32) bool {
	if healths.IsHealedByObject == nil {
		missing("IsHealedByObject")
	}
	return healths.IsHealedByObject(client)
}

// IsHealedByMedic says a medic is doing it.
//
//sp:body IsHealedByMedic
func IsHealedByMedic(client int32) bool {
	if healths.IsHealedByMedic == nil {
		missing("IsHealedByMedic")
	}
	return healths.IsHealedByMedic(client)
}

// WeaponIDIsSniperRifle says the weapon is one of the rifles.
//
//sp:native WeaponID_IsSniperRifle
func WeaponIDIsSniperRifle(id Weapon) bool {
	if healths.WeaponIDIsSniperRifle == nil {
		missing("WeaponID_IsSniperRifle")
	}
	return healths.WeaponIDIsSniperRifle(id)
}

// IsCritBoosted says every shot is a crit, which is what makes a revolver worth
// firing at somebody with 400 health.
//
//sp:native TF2_IsCritBoosted
func IsCritBoosted(client int32) bool {
	if healths.IsCritBoosted == nil {
		missing("TF2_IsCritBoosted")
	}
	return healths.IsCritBoosted(client)
}

// AsFloat reinterprets a cell as a float, which is what view_as<float> does and
// what a list holding both has to do to get one back out.
//
//sp:native view_as<float>
func AsFloat(cell int32) float32 {
	if healths.AsFloat == nil {
		missing("view_as<float>")
	}
	return healths.AsFloat(cell)
}

// IsPointInRespawnRoom says the position is inside a spawn, where a robot
// cannot be reached and a buster cannot be walked away from.
//
//sp:native TF2Util_IsPointInRespawnRoom
func IsPointInRespawnRoom(position [3]float32) bool {
	if healths.IsPointInRespawnRoom == nil {
		missing("TF2Util_IsPointInRespawnRoom")
	}
	return healths.IsPointInRespawnRoom(position)
}

// IsAmmoLow says the bot is running out, which is one of the two reasons to
// stand at a dispenser.
//
//sp:plugin IsAmmoLow
func IsAmmoLow(client int32) bool {
	if healths.IsAmmoLow == nil {
		missing("IsAmmoLow")
	}
	return healths.IsAmmoLow(client)
}
