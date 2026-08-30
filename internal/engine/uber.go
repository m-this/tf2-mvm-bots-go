package engine

// UberCalls are the answers for what a medic does with a full meter.
type UberCalls struct {
	MedigunType              func(weapon int32) int32
	CountEnemiesNearPosition func(client int32, origin [3]float32, radius float32) int32
	RageMeter                func(client int32) float32
	IsRageDraining           func(client int32) bool
	PressSpecialFireButton   func(client int32)
	LogMessage               func(format string, args ...any)
}

var ubers UberCalls

// InstallUbers puts a set of answers behind them.
func InstallUbers(c UberCalls) func() {
	previous := ubers
	ubers = c
	return func() { ubers = previous }
}

// MedigunCritboost is MEDIGUN_CRITBOOST, the Kritzkrieg.
//
//sp:global MEDIGUN_CRITBOOST
func MedigunCritboost() int32 { return 1 }

// MedigunMegaheal is MEDIGUN_MEGAHEAL, the Quick-Fix.
//
//sp:global MEDIGUN_MEGAHEAL
func MedigunMegaheal() int32 { return 2 }

// MedigunResist is MEDIGUN_RESIST, the Vaccinator.
//
//sp:global MEDIGUN_RESIST
func MedigunResist() int32 { return 3 }

// FeatureMedicShield is the switch on a medic putting the projectile shield up.
//
//sp:global FEATURE_MEDIC_SHIELD
func FeatureMedicShield() int32 { return 14 }

// MedigunType is which of the four this medigun is.
//
//sp:body GetMedigunType
func MedigunType(weapon int32) int32 {
	if ubers.MedigunType == nil {
		missing("GetMedigunType")
	}
	return ubers.MedigunType(weapon)
}

// CountEnemiesNearPosition is how many robots are in the fight at that spot.
//
//sp:body CountEnemiesNearPosition
func CountEnemiesNearPosition(client int32, origin [3]float32, radius float32) int32 {
	if ubers.CountEnemiesNearPosition == nil {
		missing("CountEnemiesNearPosition")
	}
	return ubers.CountEnemiesNearPosition(client, origin, radius)
}

// RageMeter is how full the banner or the shield is.
//
//sp:plugin TF2_GetRageMeter
func RageMeter(client int32) float32 {
	if ubers.RageMeter == nil {
		missing("TF2_GetRageMeter")
	}
	return ubers.RageMeter(client)
}

// IsRageDraining says it is already being spent.
//
//sp:plugin TF2_IsRageDraining
func IsRageDraining(client int32) bool {
	if ubers.IsRageDraining == nil {
		missing("TF2_IsRageDraining")
	}
	return ubers.IsRageDraining(client)
}

// PressSpecialFireButton is the third button, which is what puts a shield up.
//
//sp:plugin VS_PressSpecialFireButton
func PressSpecialFireButton(client int32) {
	if ubers.PressSpecialFireButton == nil {
		missing("VS_PressSpecialFireButton")
	}
	ubers.PressSpecialFireButton(client)
}

// LogMessage writes a line to the plugin log, which is where a measured
// behaviour says it fired.
//
//sp:native LogMessage
func LogMessage(format string, args ...any) {
	if ubers.LogMessage == nil {
		missing("LogMessage")
	}
	ubers.LogMessage(format, args...)
}

// WeaponJarGas is TF_WEAPON_JAR_GAS, the Gas Passer.
//
//sp:global TF_WEAPON_JAR_GAS
func WeaponJarGas() Weapon { return 74 }
