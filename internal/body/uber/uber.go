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

/*
What a fight the patient can lose looks like before he reads as half dead.

Health first, because it is what says the robots in front of him are hitting him
rather than being walked past. The threshold is his maximum and not a fraction of
it: a man under a beam is overhealed, so any health at all missing means the
damage is outrunning the 24 to 72 a second the medigun puts back. Measured over
127 heavy samples on Decoy, 80 per cent sat at or above maximum health and only 3
per cent fell in the band between 85 per cent and whole, which is why asking for
a fraction asked for almost nothing.

Then either a giant, which takes a patient from whole to dead while he is still
reading as healthy, or a crowd large enough to add up to the same thing. The
crowd is one robot wider than the Kritzkrieg's: crits are spent on a target the
patient is already shooting, and invulnerability is spent on damage that would
otherwise land.
*/
const (
	//sp:name UBER_PRESSED_HEALTH_RATIO
	pressedHealthRatio = 1.0
	//sp:name UBER_PRESSED_ENEMIES
	pressedEnemies = 4
)

// A giant this close to the patient is committed to him rather than walking past
//
//sp:name UBER_GIANT_RANGE
const giantRange = 500.0

// The Vaccinator spends a quarter of its meter, so a quarter is a full charge as far as this goes
//
//sp:name UBER_RESIST_CHARGE
const resistCharge = 0.25

/*
IsChargeReleasing says the medigun is spending its charge right now.

Asked in four places and worth one name: the deploy must not press a button that
is already down, the ranking must not move the beam off the man the charge is
going into, the revive must not take the medic away mid-charge, and a sample in a
results file is worth nothing without it.
*/
//
//sp:name IsChargeReleasing
func IsChargeReleasing(medigun int32) bool {
	if medigun == -1 || !engine.HasEntProp(medigun, engine.PropSend(), "m_bChargeRelease") {
		return false
	}

	return engine.EntProp(medigun, engine.PropSend(), "m_bChargeRelease") != 0
}

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
	if IsChargeReleasing(medigun) {
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

	/* The caller's precondition, asserted rather than trusted

	A charge spent on a patient the medic is not connected to is a charge spent on the medic
	alone. The shipped caller reads the patient off this same field, so this never fires there;
	it is what stops a second caller reintroducing the bug by handing over the man the medic is
	walking towards instead of the man the beam is on. */
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
	if HealthRatio(patient) < panicHealthRatio || HealthRatio(client) < panicHealthRatio {
		return true
	}

	return IsPatientUnderPressure(client, patient, enemies)
}

/*
IsPatientUnderPressure is the fight that kills a patient without ever showing him
at half health.

Half health is a threshold, and a giant does not respect one: it takes a patient
from whole to dead between two thinks, so the charge is still full when the body
lands. Measured over every results file here, 74 of 155 waves deployed no charge
at all and 173 defenders died with a full one behind them.

A giant closing on the patient goes first and is not asked about health at all,
because health is the one thing a medic is already fixing. The beam puts back 24
to 72 a second and overheals on top, so the patient reads whole right up to the
point a giant removes him: over 127 heavy samples he sat at or above his maximum
in 80 per cent of them. Gating on health behind a live beam is gating on a state
the beam exists to prevent, and it measured as exactly that. With the gate in
front, medic_ubers_early was reached 24 times in a wave and still deployed one
charge, the same as the arm without it.

The crowd keeps the health test, because a crowd is not a deadline. Robots near
an overhealed patient are robots he is walking past, and a charge spent on that
is the charge missing from the fight after it.
*/
//
//sp:name IsPatientUnderPressure
func IsPatientUnderPressure(client int32, patient int32, enemies int32) bool {
	if !engine.Feature(engine.FeatureMedicUbersEarly()) {
		return false
	}

	if engine.EnemyNearestToMe(patient, giantRange, true, false, false, engine.ClassUnknown()) != -1 {
		return true
	}

	if HealthRatio(patient) >= pressedHealthRatio && HealthRatio(client) >= pressedHealthRatio {
		return false
	}

	return enemies >= pressedEnemies
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
