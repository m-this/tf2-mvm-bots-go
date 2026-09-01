/* When a medic fires the charge it spent the wave building

Nothing in this mod used to decide this. The medic ran Valve's own healing behaviour, and Valve's
rule is a panic rule: the charge goes off when the medic or the patient is about to die. That is
a defensible answer for a stock medigun and the wrong answer for every other one. A play-test put
it plainly: the Kritzkrieg pops with the medic already dying, the patient gets crits for the half
second he outlives him, and the weapon may as well not be there.

So each medigun is asked its own question, because each one is carried for a different reason:

  stock         invulnerability, spent on the fight that would otherwise kill the patient
  Kritzkrieg    damage, spent on a crowd the patient is already shooting at
  Quick-Fix     a heal rate, spent as soon as the patient is losing health faster than it heals
  Vaccinator    a bubble, spent often, because it charges four times as fast as it is spent

None of this suppresses Valve's rule. It cannot: the deploy is the game's, and all a plugin can
do is press the button earlier than the game would have. Earlier is the entire fix. */

//How far from the patient still counts as the fight the patient is in
#define UBER_FIGHT_RANGE	1000.0

//A crowd worth critting. Two robots and a giant is a wave; one robot is a straggler
#define UBER_CRITBOOST_ENEMIES	3

//Long enough after the patient last fired to know he is in a fight rather than walking to one
#define UBER_PATIENT_FIRING_TIME	2.0

//Health left before the charge is what keeps the patient alive
#define UBER_PANIC_HEALTH_RATIO		0.5
#define UBER_MEGAHEAL_HEALTH_RATIO	0.7

//The Vaccinator spends a quarter of its meter, so a quarter is a full charge as far as this goes
#define UBER_RESIST_CHARGE	0.25

/* Whether to fire the charge now

Every branch wants a patient. A medic with nobody to heal has nothing to spend a charge on, and
the one who should be saving himself is running, not ubering into an empty corridor */
bool ShouldDeployUber(int client, int medigun, int patient)
{
	if (medigun == -1 || TF2Util_GetWeaponID(medigun) != TF_WEAPON_MEDIGUN)
		return false;

	//Already spending it
	if (GetEntProp(medigun, Prop_Send, "m_bChargeRelease"))
		return false;

	if (!IsValidClientIndex(patient) || !IsPlayerAlive(patient))
		return false;

	int type = GetMedigunType(medigun);
	float charge = GetEntPropFloat(medigun, Prop_Send, "m_flChargeLevel");

	if (charge < (type == MEDIGUN_RESIST ? UBER_RESIST_CHARGE : 1.0))
		return false;

	//A charge spent on a patient the medic is not connected to is a charge spent on the medic alone
	if (GetEntPropEnt(medigun, Prop_Send, "m_hHealingTarget") != patient)
		return false;

	float patientOrigin[3]; patientOrigin = WorldSpaceCenter(patient);
	int enemies = CountEnemiesNearPosition(client, patientOrigin, UBER_FIGHT_RANGE);

	//Nothing to spend it on, whatever the medigun is
	if (enemies < 1)
		return false;

	switch (type)
	{
		case MEDIGUN_CRITBOOST:
		{
			/* Crits are damage the patient has to deliver himself, so the patient has to be
			shooting. A giant counts for the crowd on its own: it is what the crits are for */
			if (GetTimeSinceWeaponFired(patient) > UBER_PATIENT_FIRING_TIME)
				return false;

			if (FindEnemyNearestToMe(patient, UBER_FIGHT_RANGE, true) != -1)
				return true;

			return enemies >= UBER_CRITBOOST_ENEMIES;
		}
		case MEDIGUN_MEGAHEAL:
		{
			//It heals rather than saves, so it is spent on damage taken rather than on death
			return HealthRatio(patient) < UBER_MEGAHEAL_HEALTH_RATIO || HealthRatio(client) < UBER_MEGAHEAL_HEALTH_RATIO;
		}
		case MEDIGUN_RESIST:
		{
			//A quarter of a meter is cheap enough to spend on anybody taking fire
			return HealthRatio(patient) < 1.0 || HealthRatio(client) < 1.0;
		}
	}

	/* Stock, and the one case where the game's own rule is nearly right. It is kept, and moved
	off the floor: waiting for the last of the patient's health spends the charge on the retreat
	rather than on the fight it was built for */
	return HealthRatio(patient) < UBER_PANIC_HEALTH_RATIO || HealthRatio(client) < UBER_PANIC_HEALTH_RATIO;
}

/* The projectile shield, which is the strongest thing a medic does to a wave
 *
 * Every guide puts a tick of rage first for a medic, and nothing here had ever pressed the button
 * it fills. Bavarian Botbash wave 1 kills this team with projectiles: 73 per cent of the deaths are
 * explosions and 62 per cent of them are robot Soldiers, which is what a shield is for.
 *
 * Up when it is full and there is something shooting. It drains on its own, so there is nothing to
 * decide about taking it down, and holding a full meter through a wave is the same waste as never
 * buying the rage in the first place. */
#define SHIELD_FIGHT_RANGE	1200.0

void MedicProjectileShield(int actor, int patient)
{
	if (!Feature(FEATURE_MEDIC_SHIELD))
		return;

	if (TF2_GetRageMeter(actor) < 100.0 || TF2_IsRageDraining(actor))
		return;

	//Somewhere worth putting it: the fight the patient is in, or the one the medic is in himself
	float where[3]; where = WorldSpaceCenter(actor);

	if (IsValidClientIndex(patient) && IsPlayerAlive(patient))
		where = WorldSpaceCenter(patient);

	if (CountEnemiesNearPosition(actor, where, SHIELD_FIGHT_RANGE) < 1)
		return;

	/* Said out loud, because a behaviour nobody can see fire is a behaviour nobody can measure
	
	The first arm of this could not be read: every number sat inside the baseline's spread, which
	means either the shield does nothing or it never went up, and there was no way to tell those
	apart. */
	LogMessage("Shield: %N puts it up, rage %.0f", actor, TF2_GetRageMeter(actor));

	VS_PressSpecialFireButton(actor);
}

float HealthRatio(int client)
{
	int maxHealth = TEMP_GetPlayerMaxHealth(client);

	if (maxHealth <= 0)
		return 1.0;

	return float(GetClientHealth(client)) / float(maxHealth);
}
