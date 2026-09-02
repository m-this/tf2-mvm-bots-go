/*
Package aimweapons is what the aiming needs to know about a weapon, out of
source/redbots3/botaim.sp: whether it hits instantly, whether it keeps firing
while the button is held, and whether what it fires goes off on contact.

Three questions, three lists, and every one of them decides how the bot leads
its target.
*/
package aimweapons

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
IsHitScanWeapon says the shot lands the moment it is fired.

Everything else arcs or flies, and a bot aiming one of those has to lead. The
sentry's own weapons are in the list because a bot standing behind one is asked
the same question about it.
*/
//
//sp:name IsHitScanWeapon
func IsHitScanWeapon(weapon int32) bool {
	if engine.IsValidEntity(weapon) {
		switch engine.WeaponID(weapon) {
		case engine.WeaponShotgunPrimary(), engine.WeaponShotgunSoldier(), engine.WeaponShotgunHwg(), engine.WeaponShotgunPyro(), engine.WeaponScattergun(), engine.WeaponSniperrifle(), engine.WeaponMinigun(),
			engine.WeaponSmg(), engine.WeaponChargedSmg(), engine.WeaponPistol(), engine.WeaponPistolScout(), engine.WeaponRevolver(), engine.WeaponSentryBullet(), engine.WeaponSentryRocket(), engine.WeaponSentryRevenge(),
			engine.WeaponHandgunScoutPrimary(), engine.WeaponHandgunScoutSec(), engine.WeaponSodaPopper(), engine.WeaponSniperrifleDecap(), engine.WeaponPepBrawlerBlaster(), engine.WeaponSniperrifleClassic():
			return true
		}
	}

	return false
}

/*
IsContinuousFireWeapon says holding the button keeps it firing.

Written as the exceptions rather than the list: most weapons do, and the ones
that do not are the ones that cost a shot to press.
*/
//
//sp:name IsContinuousFireWeapon
func IsContinuousFireWeapon(client int32, weapon int32) bool {
	if !engine.IsCombatWeapon(client, weapon) {
		return false
	}

	if engine.IsValidEntity(weapon) {
		switch engine.WeaponID(weapon) {
		case engine.WeaponRocketLauncher(), engine.WeaponDirecthit(), engine.WeaponGrenadeLauncher(), engine.WeaponPipebombLauncher(), engine.WeaponPistol(), engine.WeaponPistolScout(), engine.WeaponFlaregun(),
			engine.WeaponJar(), engine.WeaponCompoundBow():
			return false
		}
	}

	return true
}

/*
IsExplosiveProjectileWeapon says what it fires goes off where it lands, so
firing it at something the bot is standing against hurts the bot.

The cannon belongs here as much as any of them: without it the bot skips the
proximity check and fires a grenade into the robot it is standing against.
*/
//
//sp:name IsExplosiveProjectileWeapon
func IsExplosiveProjectileWeapon(weapon int32) bool {
	if engine.IsValidEntity(weapon) {
		switch engine.WeaponID(weapon) {
		case engine.WeaponRocketLauncher(), engine.WeaponDirecthit(), engine.WeaponGrenadeLauncher(), engine.WeaponPipebombLauncher(), engine.WeaponJar(),
			engine.WeaponCannon():
			return true
		}
	}

	return false
}

// IsPipeLauncher says the weapon lobs, which is the arc the ballistic lead is
// computed for.
//
//sp:name IsPipeLauncher
func IsPipeLauncher(weaponID engine.Weapon) bool {
	switch weaponID {
	case engine.WeaponGrenadeLauncher(), engine.WeaponPipebombLauncher(), engine.WeaponCannon():
		return true
	}

	return false
}
