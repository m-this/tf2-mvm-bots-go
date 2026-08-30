/*
Package loadout is the part of source/redbots3/util.sp that asks what this bot is
carrying.

An item definition index rather than a weapon id: the game gives the Gunslinger and
the stock wrench the same TF_WEAPON_WRENCH, and it is the item that decides whether
this engineer holds a level three or spends mini sentries.
*/
package loadout

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// The item definitions the mod tells apart.
const (
	//sp:name TF_ITEMDEF_GUNSLINGER
	gunslinger = 142
	// Declared by the shipped file and read by nothing, here or anywhere
	// else. Dropping it would be a tidy riding along with a port.
	//
	//sp:name TF_ITEMDEF_EUREKA_EFFECT
	//
	//nolint:unused // the shipped file declares it unused, and the port keeps what ships
	eurekaEffect = 589
	//sp:name TF_ITEMDEF_RESCUE_RANGER
	rescueRanger = 997
	//sp:name TF_ITEMDEF_WRANGLER
	//
	//nolint:unused // the shipped file declares it unused, and the port keeps what ships
	wrangler = 140
	//sp:name TF_ITEMDEF_WIDOWMAKER
	widowmaker = 527
	//sp:name TF_ITEMDEF_SHORT_CIRCUIT
	shortCircuit = 528
)

// CanUsePrimayWeapon says he has a primary and the mission lets him use it. The
// name is the shipped one, typo included.
//
//sp:name CanUsePrimayWeapon
func CanUsePrimayWeapon(client int32) bool {
	if engine.IsPlayerInCondition(client, engine.ConditionMeleeOnly()) {
		return false
	}

	weapon := engine.PlayerWeaponSlot(client, engine.WeaponSlotPrimary())

	return weapon != -1
}

// GetLoadoutSlotItemDefinitionIndex is which item is in that slot, and -1 for an
// empty slot or one holding something with no definition.
//
//sp:name GetLoadoutSlotItemDefinitionIndex
func GetLoadoutSlotItemDefinitionIndex(client int32, slot int32) int32 {
	weapon := engine.PlayerWeaponSlot(client, slot)

	if weapon < 1 || !engine.HasEntProp(weapon, engine.PropSend(), "m_iItemDefinitionIndex") {
		return -1
	}

	return engine.EntProp(weapon, engine.PropSend(), "m_iItemDefinitionIndex")
}

// IsGunslingerEquipped says this engineer builds minis rather than level threes.
//
//sp:name TF2_IsGunslingerEquipped
func IsGunslingerEquipped(client int32) bool {
	return GetLoadoutSlotItemDefinitionIndex(client, engine.WeaponSlotMelee()) == gunslinger
}

// IsRescueRangerEquipped says he can repair from behind cover.
//
//sp:name TF2_IsRescueRangerEquipped
func IsRescueRangerEquipped(client int32) bool {
	return GetLoadoutSlotItemDefinitionIndex(client, engine.WeaponSlotPrimary()) == rescueRanger
}

// EngineerGunSpendsMetal says his gun is paid for out of the same supply the
// sentry is repaired from.
//
//sp:name EngineerGunSpendsMetal
func EngineerGunSpendsMetal(client int32) bool {
	if engine.PlayerClass(client) != engine.ClassEngineer() {
		return false
	}

	switch GetLoadoutSlotItemDefinitionIndex(client, engine.WeaponSlotPrimary()) {
	case widowmaker, rescueRanger:
		return true
	}

	return GetLoadoutSlotItemDefinitionIndex(client, engine.WeaponSlotSecondary()) == shortCircuit
}
