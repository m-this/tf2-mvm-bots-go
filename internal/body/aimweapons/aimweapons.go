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

// IsValidTarget says the entity is still there and is something that fights.
//
//sp:name IsValidTarget
func IsValidTarget(entity int32) bool {
	return engine.IsValidEntity(entity) && engine.EntityOf(entity).IsCombatCharacter()
}

/*
GetMaxAttackRange is how far this weapon is worth firing.

Only where firing further is genuinely wasted; most loadouts say nothing here
and get no limit at all.

The stock launcher gets the same answer the Iron Bomber does. It is absent from
the tuning table, so it fell through to no limit and threw pipes across the map
at anything it could see.
*/
//
//sp:name GetMaxAttackRange
func GetMaxAttackRange(client int32) float32 {
	myWeapon := engine.ActiveWeapon(client)

	if myWeapon == -1 {
		return 0.0
	}

	if engine.IsMeleeWeapon(myWeapon) {
		return 100.0
	}

	tuned, tunedDesired, tunedMax := engine.TunedWeaponRanges(myWeapon)
	_ = tunedDesired

	if tuned && tunedMax > engine.RangeTuningNone() {
		return tunedMax
	}

	myWeaponID := engine.WeaponID(myWeapon)

	if myWeaponID == engine.WeaponFlamethrower() {
		if engine.IsMannVsMachineMode() {
			return 350.0
		}

		return 250.0
	}

	if engine.WeaponIDIsSniperRifle(myWeaponID) {
		return engine.FloatMax()
	}

	if myWeaponID == engine.WeaponRocketLauncher() {
		return 3000.0
	}

	if myWeaponID == engine.WeaponGrenadeLauncher() {
		return engine.DemoPipeMaxRange()
	}

	return engine.FloatMax()
}

/*
ShouldAimRocketsAtFeet says the splash is worth more than the direct hit.

Aiming at the body up close was tried and cost more than the splash did. The
reasoning was sound: a quarter of the Soldier's damage goes into his own feet.
Measured over six waves on Decoy, aiming at the chest inside 350 units took his
hit rate from 60 per cent to 40 and his damage from 16890 to 10886, and the self
damage went up rather than down. The ground does not move and a robot does.

A rocket fired into a Pyro's face comes back, and a reflected rocket is the
Soldier's own damage aimed at his team. The wiki says to shoot the ground
instead, which is a shot an airblast cannot catch: worth doing even for one of
them, and even for a giant, which is why that test sits above both below it.
*/
//
//sp:name ShouldAimRocketsAtFeet
func ShouldAimRocketsAtFeet(client int32, target int32, weaponID engine.Weapon) bool {
	if weaponID == engine.WeaponDirecthit() {
		return false
	}

	if engine.IsPlayer(target) && engine.IsClientInGame(target) && engine.PlayerClass(target) == engine.ClassPyro() {
		return true
	}

	if engine.IsMiniBoss(target) {
		return false
	}

	// The rocket splash radius, which is the ground this shot would cover.
	return engine.CountEnemiesNearPosition(client, engine.AbsOriginOf(target), 146.0) > 1
}
