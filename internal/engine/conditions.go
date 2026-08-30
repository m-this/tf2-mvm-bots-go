package engine

/*
The player conditions a cloak can be given away by, and the two attribute reads
that answer what a weapon does.

Each condition is a constant the game defines, so they are globals rather than
numbers: TFCond_Jarated is what the file says and what a reader recognises.
*/

// ConditionCalls are the answers.
type ConditionCalls struct {
	AttribHookValueFloat func(value float32, name string, weapon int32) float32
	AbsVelocity          func(e Entity) [3]float32
}

var conditions ConditionCalls

// InstallConditions puts a set of answers behind them.
func InstallConditions(c ConditionCalls) func() {
	previous := conditions
	conditions = c
	return func() { conditions = previous }
}

// ConditionOnFire is TFCond_OnFire.
//
//sp:global TFCond_OnFire
func ConditionOnFire() Condition { return 22 }

// ConditionJarated is TFCond_Jarated.
//
//sp:global TFCond_Jarated
func ConditionJarated() Condition { return 23 }

// ConditionCloakFlicker is TFCond_CloakFlicker.
//
//sp:global TFCond_CloakFlicker
func ConditionCloakFlicker() Condition { return 32 }

// ConditionBleeding is TFCond_Bleeding.
//
//sp:global TFCond_Bleeding
func ConditionBleeding() Condition { return 76 }

// ConditionMilked is TFCond_Milked.
//
//sp:global TFCond_Milked
func ConditionMilked() Condition { return 84 }

// ConditionGas is TFCond_Gas.
//
//sp:global TFCond_Gas
func ConditionGas() Condition { return 127 }

// AttribHookValueFloat is a float attribute the weapon may carry, or the value
// given when it does not.
//
//sp:native TF2Attrib_HookValueFloat
func AttribHookValueFloat(value float32, name string, weapon int32) float32 {
	if conditions.AttribHookValueFloat == nil {
		missing("TF2Attrib_HookValueFloat")
	}
	return conditions.AttribHookValueFloat(value, name, weapon)
}

// AbsVelocity is how fast the entity is going, which is how a standing player is
// told from a walking one.
//
//sp:method GetAbsVelocity
func (e Entity) AbsVelocity() (velocity [3]float32) {
	if conditions.AbsVelocity == nil {
		missing("CBaseEntity.GetAbsVelocity")
	}
	return conditions.AbsVelocity(e)
}

// ConditionMeleeOnly is TFCond_MeleeOnly, which a mission can put a wave in.
//
//sp:global TFCond_MeleeOnly
func ConditionMeleeOnly() Condition { return 27 }
