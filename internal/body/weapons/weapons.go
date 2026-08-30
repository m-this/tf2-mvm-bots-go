/*
Package weapons is the part of source/redbots3/util.sp that answers what is in a
bot's hands and what it is doing with it.

Five short questions every behaviour asks, each one an entity property read behind
a name, and each one an extern the ported behaviours reached for.
*/
package weapons

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// EquipWeaponSlot puts the weapon in that slot in his hands, and answers false
// when the slot is empty.
//
//sp:name EquipWeaponSlot
func EquipWeaponSlot(client int32, slot int32) bool {
	weapon := engine.PlayerWeaponSlot(client, slot)

	if weapon != -1 {
		return engine.SetActiveWeapon(client, weapon)
	}

	return false
}

// GetTimeSinceWeaponFired is how long ago he last pulled the trigger, and a very
// long time when he has not or is holding nothing.
//
//sp:name GetTimeSinceWeaponFired
func GetTimeSinceWeaponFired(client int32) float32 {
	weapon := engine.ActiveWeapon(client)

	if weapon == -1 {
		return 9999.0
	}

	lastFireTime := engine.EntPropFloat(weapon, engine.PropSend(), "m_flLastFireTime")

	if lastFireTime <= 0.0 {
		return 9999.0
	}

	return engine.GameTime() - lastFireTime
}

// GetMedigunType is which of the four mediguns this is, which the game keeps as a
// weapon mode rather than as an item.
//
//sp:name GetMedigunType
func GetMedigunType(weapon int32) int32 {
	return engine.AttribHookValueInt(0, "set_weapon_mode", weapon)
}

// GetResistType is which bubble a Vaccinator is on.
//
//sp:name GetResistType
func GetResistType(weapon int32) int32 {
	return engine.EntProp(weapon, engine.PropSend(), "m_nChargeResistType")
}

// HasSniperRifle says the primary is one, whichever rifle it is.
//
//sp:name HasSniperRifle
func HasSniperRifle(client int32) bool {
	weapon := engine.PlayerWeaponSlot(client, engine.WeaponSlotPrimary())

	if weapon == -1 {
		return false
	}

	return engine.WeaponIDIsSniperRifle(engine.WeaponID(weapon))
}
