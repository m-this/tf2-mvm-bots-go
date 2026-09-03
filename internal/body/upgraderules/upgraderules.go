/*
Package upgraderules is the two questions the upgrade ranking asks that are not
scores: whether an upgrade does anything for the weapons this bot is holding, and
how many ticks of one it is allowed to buy.

The scores themselves are a table in internal/upgrade. These are here because
both read the loadout, which is a question about the bot rather than about the
attribute.
*/
package upgraderules

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
IsUpgradeWasted is an upgrade that does nothing for the weapons this bot is
actually holding.

Reported on 1.8: Pyros buying Explode on Ignite with no Gas Passer, and airblast
pushback while carrying a Phlogistinator, which has no airblast at all. Both are
the upgrade menu offering everything the class can theoretically use rather than
what this loadout can.

The menu is right to offer them. Deciding is this mod's job.
*/
//
//sp:name IsUpgradeWasted
func IsUpgradeWasted(client int32, attribute engine.Text) bool {
	// Explode on Ignite is the Gas Passer's, and nothing else can be ignited into exploding
	if engine.StrContains(attribute, "explode_on_ignite", false) != -1 ||
		engine.StrContains(attribute, "explode on ignite", false) != -1 {
		secondary := engine.PlayerWeaponSlot(client, engine.WeaponSlotSecondary())

		return secondary == -1 || engine.WeaponID(secondary) != engine.WeaponJarGas()
	}

	if engine.StrContains(attribute, "airblast", false) != -1 {
		primary := engine.PlayerWeaponSlot(client, engine.WeaponSlotPrimary())

		if primary == -1 || engine.WeaponID(primary) != engine.WeaponFlamethrower() {
			return true
		}

		// A flamethrower that cannot airblast, which is the Phlogistinator and anything like it
		return engine.AttribByName(primary, "airblast disabled") != engine.NoAddress()
	}

	/* Destroy Projectiles is an airblast on a Pyro and a spun-up minigun on a Heavy

	Same attribute, two different things behind it, and the guides rate it for both because a
	person carrying a Phlogistinator knows they have given up the airblast. The upgrade menu does
	not, and this table did not either: the Pyro's own line ranked it at 250 while the loadout
	handed him the one flamethrower that cannot do it. */
	if engine.StrEqual(attribute, "attack projectiles") && engine.PlayerClass(client) == engine.ClassPyro() {
		primary := engine.PlayerWeaponSlot(client, engine.WeaponSlotPrimary())

		if primary == -1 {
			return true
		}

		return engine.AttribByName(primary, "airblast disabled") != engine.NoAddress()
	}

	/* The Projectile Shield, which nothing in this mod presses

	Every guide puts one tick of it first for a Medic and they are right about a person: it is the
	strongest thing a Medic can do to a wave. It is deployed with the special attack key, and no
	behaviour here has ever pressed one, so what the rage meter fills is a button nobody uses.

	Three hundred credits for that, ranked at the top of the Medic's list, every wave. It goes back
	the moment something deploys it, and that is the TODO rather than this. */
	if engine.StrEqual(attribute, "generate rage on heal") {
		return !engine.Feature(engine.FeatureMedicShield())
	}

	/* Afterburn, which the wiki calls useless and a bot has even less use for

	It does not scale the way direct damage does, a small robot dies before it finishes ticking,
	and a giant outlives it. */
	if engine.StrEqual(attribute, "weapon burn dmg increased") || engine.StrEqual(attribute, "weapon burn time increased") {
		return true
	}

	return false
}

// How many ticks of one upgrade a bot may buy, which is the game's own limit for
// everything but one.
//
//sp:name UPGRADE_TIERS_MAX
const tiersMax = 4

/*
UpgradeTierCap is how many ticks of this upgrade are worth buying.

Rocket Specialist is capped at one: the first tick removes the falloff and the
rest widen a blast radius nobody needed.
*/
//
//sp:name UpgradeTierCap
func UpgradeTierCap(attribute engine.Text) int32 {
	if engine.StrEqual(attribute, "rocket specialist") {
		return 1
	}

	return tiersMax
}
