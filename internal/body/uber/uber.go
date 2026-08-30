/*
Package uber is source/redbots3/medic_uber.sp.

# When a medic fires the charge it spent the wave building

Nothing in this mod used to decide this. The medic ran Valve's own healing
behaviour, and Valve's rule is a panic rule: the charge goes off when the medic or
the patient is about to die. That is a defensible answer for a stock medigun and
the wrong answer for every other one. A play-test put it plainly: the Kritzkrieg
pops with the medic already dying, the patient gets crits for the half second he
outlives him, and the weapon may as well not be there.

So each medigun is asked its own question, because each one is carried for a
different reason:

	stock         invulnerability, spent on the fight that would otherwise kill the patient
	Kritzkrieg    damage, spent on a crowd the patient is already shooting at
	Quick-Fix     a heal rate, spent as soon as the patient is losing health faster than it heals
	Vaccinator    a bubble, spent often, because it charges four times as fast as it is spent

None of this suppresses Valve's rule. It cannot: the deploy is the game's, and all
a plugin can do is press the button earlier than the game would have. Earlier is
the entire fix.
*/
package uber

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// How far from the patient still counts as the fight the patient is in
//
//sp:name UBER_FIGHT_RANGE
const fightRange = 1000.0

// A crowd worth critting. Two robots and a giant is a wave; one robot is a straggler
//
//sp:name UBER_CRITBOOST_ENEMIES
const critboostEnemies = 3

// Long enough after the patient last fired to know he is in a fight rather than walking to one
//
//sp:name UBER_PATIENT_FIRING_TIME
const patientFiringTime = 2.0

// Health left before the charge is what keeps the patient alive
const (
	//sp:name UBER_PANIC_HEALTH_RATIO
	panicHealthRatio = 0.5
	//sp:name UBER_MEGAHEAL_HEALTH_RATIO
	megahealHealthRatio = 0.7
)

// The Vaccinator spends a quarter of its meter, so a quarter is a full charge as far as this goes
//
//sp:name UBER_RESIST_CHARGE
const resistCharge = 0.25

/*
ShouldDeployUber is whether to fire the charge now.

Every branch wants a patient. A medic with nobody to heal has nothing to spend a
charge on, and the one who should be saving himself is running, not ubering into an
empty corridor.
*/
//
//sp:name ShouldDeployUber
func ShouldDeployUber(client int32, medigun int32, patient int32) bool {
	if medigun == -1 || engine.WeaponID(medigun) != engine.WeaponMedigun() {
		return false
	}

	// Already spending it
	if engine.EntProp(medigun, engine.PropSend(), "m_bChargeRelease") != 0 {
		return false
	}

	if !engine.IsValidClientIndex(patient) || !engine.IsPlayerAlive(patient) {
		return false
	}

	medigunType := engine.MedigunType(medigun)
	charge := engine.EntPropFloat(medigun, engine.PropSend(), "m_flChargeLevel")

	if charge < engine.ChooseFloat(medigunType == engine.MedigunResist(), resistCharge, 1.0) {
		return false
	}

	// A charge spent on a patient the medic is not connected to is a charge spent on the medic alone
	if engine.EntPropEnt(medigun, engine.PropSend(), "m_hHealingTarget") != patient {
		return false
	}

	patientOrigin := engine.WorldSpaceCenter(patient)
	enemies := engine.CountEnemiesNearPosition(client, patientOrigin, fightRange)

	// Nothing to spend it on, whatever the medigun is
	if enemies < 1 {
		return false
	}

	switch medigunType {
	case engine.MedigunCritboost():
		/* Crits are damage the patient has to deliver himself, so the patient has to be
		shooting. A giant counts for the crowd on its own: it is what the crits are for */
		if engine.TimeSinceWeaponFired(patient) > patientFiringTime {
			return false
		}

		if engine.EnemyNearestToMe(patient, fightRange, true, false, false, engine.ClassUnknown()) != -1 {
			return true
		}

		return enemies >= critboostEnemies
	case engine.MedigunMegaheal():
		// It heals rather than saves, so it is spent on damage taken rather than on death
		return HealthRatio(patient) < megahealHealthRatio || HealthRatio(client) < megahealHealthRatio
	case engine.MedigunResist():
		// A quarter of a meter is cheap enough to spend on anybody taking fire
		return HealthRatio(patient) < 1.0 || HealthRatio(client) < 1.0
	}

	/* Stock, and the one case where the game's own rule is nearly right. It is kept, and moved
	off the floor: waiting for the last of the patient's health spends the charge on the retreat
	rather than on the fight it was built for */
	return HealthRatio(patient) < panicHealthRatio || HealthRatio(client) < panicHealthRatio
}

/*
How far the fight reaches for the shield.

The projectile shield is the strongest thing a medic does to a wave. Every guide
puts a tick of rage first for a medic, and nothing here had ever pressed the button
it fills. Bavarian Botbash wave 1 kills this team with projectiles: 73 per cent of
the deaths are explosions and 62 per cent of them are robot Soldiers, which is what
a shield is for.

Up when it is full and there is something shooting. It drains on its own, so there
is nothing to decide about taking it down, and holding a full meter through a wave
is the same waste as never buying the rage in the first place.
*/
//
//sp:name SHIELD_FIGHT_RANGE
const shieldFightRange = 1200.0

// MedicProjectileShield puts it up when it is full and there is something to put
// it in front of.
//
//sp:name MedicProjectileShield
func MedicProjectileShield(actor int32, patient int32) {
	if !engine.Feature(engine.FeatureMedicShield()) {
		return
	}

	if engine.RageMeter(actor) < 100.0 || engine.IsRageDraining(actor) {
		return
	}

	// Somewhere worth putting it: the fight the patient is in, or the one the medic is in himself
	where := engine.WorldSpaceCenter(actor)

	if engine.IsValidClientIndex(patient) && engine.IsPlayerAlive(patient) {
		where = engine.WorldSpaceCenter(patient)
	}

	if engine.CountEnemiesNearPosition(actor, where, shieldFightRange) < 1 {
		return
	}

	/* Said out loud, because a behaviour nobody can see fire is a behaviour nobody can measure

	The first arm of this could not be read: every number sat inside the baseline's spread, which
	means either the shield does nothing or it never went up, and there was no way to tell those
	apart. */
	engine.LogMessage("Shield: %N puts it up, rage %.0f", actor, engine.RageMeter(actor))

	engine.PressSpecialFireButton(actor)
}

// HealthRatio is how much of his health the client has left.
//
//sp:name HealthRatio
func HealthRatio(client int32) float32 {
	maxHealth := engine.PlayerMaxHealth(client)

	if maxHealth <= 0 {
		return 1.0
	}

	return float32(engine.ClientHealth(client)) / float32(maxHealth)
}
