/*
Package state is the part of source/redbots3/util.sp that reads what a player or
a weapon is doing right now: charges in a bottle, whether a cloak is showing,
whether a medic beam is a person or a dispenser.

Short reads behind names, which is what makes the behaviours readable.
*/
package state

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// PowerupBottleCharges is how many uses are left in a canteen.
//
//sp:name PowerupBottle_GetNumCharges
func PowerupBottleCharges(bottle int32) int32 {
	return engine.EntProp(bottle, engine.PropSend(), "m_usNumCharges")
}

// PowerupBottleType is which canteen it is.
//
//sp:name PowerupBottle_GetType
func PowerupBottleType(bottle int32) int32 {
	return engine.EntProp(bottle, engine.PropSend(), "m_usAdvancedType")
}

// GetPowerupBottle is the canteen this player owns, and -1 for none.
//
//sp:name GetPowerupBottle
func GetPowerupBottle(client int32) int32 {
	ent := int32(-1)

	for {
		ent = engine.FindEntityByClassname(ent, "tf_powerup_bottle")

		if ent == -1 {
			break
		}

		if engine.OwnerEntity(ent) == client {
			break
		}
	}

	return ent
}

// CanWeaponAirblast says the flamethrower still has its airblast, which the
// Phlogistinator does not.
//
//sp:name CanWeaponAirblast
func CanWeaponAirblast(weapon int32) bool {
	return engine.AttribHookValueInt(0, "airblast_disabled", weapon) == 0
}

// CountEnemiesNearPosition is how many robots are standing within the radius.
//
//sp:name CountEnemiesNearPosition
func CountEnemiesNearPosition(client int32, origin [3]float32, radius float32) int32 {
	count := int32(0)
	enemyTeam := engine.PlayerEnemyTeam(client)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if !engine.IsClientInGame(i) || !engine.IsPlayerAlive(i) {
			continue
		}

		if engine.PlayerTeam(i) != enemyTeam {
			continue
		}

		if engine.VectorDistance(engine.WorldSpaceCenter(i), origin) <= radius {
			count++
		}
	}

	return count
}

// CanRevolverHeadshot says the revolver is the Ambassador.
//
//sp:name CanRevolverHeadshot
func CanRevolverHeadshot(weapon int32) bool {
	return engine.AttribHookValueInt(0, "set_weapon_mode", weapon) == 1
}

// IsPlayerMoving says he is going somewhere rather than standing.
//
//sp:name IsPlayerMoving
func IsPlayerMoving(client int32) bool {
	vec := engine.EntityOf(client).AbsVelocity()

	return !engine.IsZeroVector(vec)
}

// CanWeaponAddUberOnHit says a hit with it fills the medigun, which is what a
// Crusader's Crossbow does.
//
//sp:name CanWeaponAddUberOnHit
func CanWeaponAddUberOnHit(weapon int32) bool {
	return engine.AttribHookValueFloat(0.0, "add_onhit_ubercharge", weapon) > 0.0
}

// IsCloakedPlayerExposed says something is giving the cloak away, so shooting at
// him is not seeing through it.
//
//sp:name IsCloakedPlayerExposed
func IsCloakedPlayerExposed(client int32) bool {
	if engine.IsPlayerInCondition(client, engine.ConditionOnFire()) {
		return true
	}

	if engine.IsPlayerInCondition(client, engine.ConditionJarated()) {
		return true
	}

	if engine.IsPlayerInCondition(client, engine.ConditionCloakFlicker()) {
		return true
	}

	if engine.IsPlayerInCondition(client, engine.ConditionBleeding()) {
		return true
	}

	if engine.IsPlayerInCondition(client, engine.ConditionMilked()) {
		return true
	}

	if engine.IsPlayerInCondition(client, engine.ConditionGas()) {
		return true
	}

	return false
}

// GetHealerOfPlayer is whatever is healing him, optionally only a person.
//
//sp:name GetHealerOfPlayer
//sp:default playerOnly false
func GetHealerOfPlayer(client int32, playerOnly bool) int32 {
	for i := int32(0); i < engine.NumHealers(client); i++ {
		healer := engine.PlayerHealer(client, i)

		if healer != -1 {
			if playerOnly && !engine.IsPlayer(healer) {
				continue
			}

			return healer
		}
	}

	return -1
}

// IsHealedByObject says a dispenser rather than a medic is doing it.
//
//sp:name IsHealedByObject
func IsHealedByObject(client int32) bool {
	for i := int32(0); i < engine.NumHealers(client); i++ {
		healer := engine.PlayerHealer(client, i)

		if !engine.IsBaseObject(healer) {
			continue
		}

		return true
	}

	return false
}
