/* The Demoman's stickybomb launcher, which used to be a weapon he carried and never used

Nothing in this repository detonated a stickybomb. There was no read of the bombs in play and no
alt-fire anywhere, so a bot holding the stock launcher fired eight bombs that sat on the floor
until they faded. The weapon selection did not help either: the Demoman was listed under "uses
primary" with no branch at all, so the launcher was only ever in his hands when the grenade
launcher had run dry.

Stickies are the largest damage a Demoman has in this mode, so this is the largest hole the
play-test found, and it is worth being plain about what is fixed here and what is not.

What is here: the bot fires stickies at what it is already fighting and blows them when the blast
pays. It is a sticky launcher used as a direct weapon, which is what a bot can be trusted with.

What is not here: the trap. A human lays eight bombs on the ground the wave has to walk over,
backs off, and takes a giant apart with one press. That wants a bot that picks the ground, waits
on robots it cannot see yet, and gives up on a deadline, and it is a bigger piece of work than
this. The TODO for it stands. */

//The stickybomb blast, which is the ground one bomb covers
#define STICKY_BLAST_RANGE	146.0

//A crowd worth spending the cluster on. One robot is not worth eight bombs and a reload
#define STICKY_DETONATE_ENEMIES	2

/* Bombs with somebody standing on them, which is the other way the cluster is worth pressing

Counting enemies alone misses the shape a sticky Demoman actually produces: two robots walking a
corridor, each on a different bomb. Neither bomb has a crowd on it and the pair is worth the same
as a crowd on one. */
#define STICKY_DETONATE_BOMBS	2

//A tank is a hull rather than a point, so a bomb stuck to it is further from its middle than the blast
#define STICKY_TANK_RANGE	300.0

/* Close enough that the bot blows itself up with them

A Demoman survives his own stickies at full health and does not at half, and either way the
knockback throws him off the ground he was holding. Beyond this he is only ever hurt by the
splash he was aiming for anyway */
#define STICKY_SELF_SAFE_RANGE	200.0

//The launcher holds eight, so nothing that reads them needs to look at more than that
#define STICKY_MAX_BOMBS	8

/* Whether to press the detonator, and it is only ever pressed for damage

The cluster is whatever bombs happen to be on the ground, not a trap that was laid, so the
question is the same one asked of a rocket at somebody's feet: is there more than one robot
standing in the blast, or one robot big enough that the blast is worth it on its own */
bool ShouldDetonateStickies(int client)
{
	if (TF2_GetPlayerClass(client) != TFClass_DemoMan)
		return false;

	int launcher = GetPlayerWeaponSlot(client, TFWeaponSlot_Secondary);

	if (launcher == -1 || TF2Util_GetWeaponID(launcher) != TF_WEAPON_PIPEBOMBLAUNCHER)
		return false;

	//Nothing to blow up
	if (GetEntProp(launcher, Prop_Send, "m_iPipebombCount") <= 0)
		return false;

	float myOrigin[3]; GetClientAbsOrigin(client, myOrigin);
	TFTeam enemyTeam = GetPlayerEnemyTeam(client);

	int examined = 0;
	int sticky = -1;

	/* Counted across the whole cluster rather than answered by the first bomb that qualifies
	
	Alt-fire blows all of them, so the question is what the cluster catches, not what one bomb
	catches. Asking it a bomb at a time meant two robots on two different bombs read as two bombs
	with one robot each and the button was never pressed. */
	int caughtTotal = 0;
	int bombsWithEnemies = 0;
	bool worthItAlone = false;

	while ((sticky = FindEntityByClassname(sticky, "tf_projectile_pipe_remote")) != -1)
	{
		//Somebody else's bombs, and blowing those up is not a button this bot has
		if (BaseEntity_GetOwnerEntity(sticky) != client)
			continue;

		//The count above is the bot's own, so this is the same bound read from the other side
		if (++examined > STICKY_MAX_BOMBS)
			break;

		float stickyOrigin[3]; stickyOrigin = GetAbsOrigin(sticky);

		/* One bomb of his own on top of him and the button is not worth pressing at all
		
		This used to skip the bomb and carry on, which reads as a safety rule and is not one. The
		detonator is one button for every bomb he owns: skipping a close one only stops it counting
		towards whether to press, it does not stop it going off when he does. So a Demoman with six
		on a tank hull and two down the corridor scored the two, pressed, and took all eight.
		
		He is the worst self-harmer on the team by an order of magnitude and this is the mechanism.
		Vetoing outright rather than pricing it: the cluster he gives up is one press, the health he
		gives up is the rest of the wave. */
		if (GetVectorDistance(myOrigin, stickyOrigin) < STICKY_SELF_SAFE_RANGE)
		{
			if (Feature(FEATURE_DEMO_STICKY_SELF_VETO))
				return false;
			
			continue;
		}

		int caught = 0;

		for (int i = 1; i <= MaxClients; i++)
		{
			if (!IsClientInGame(i) || !IsPlayerAlive(i))
				continue;

			if (TF2_GetClientTeam(i) != enemyTeam)
				continue;

			if (GetVectorDistance(WorldSpaceCenter(i), stickyOrigin) > STICKY_BLAST_RANGE)
				continue;

			caught++;

			/* A giant, the bomb carrier, or a Medic is worth the cluster by itself
			
			The Medic is the addition and it is the whole job on a wave that has them: a giant
			with one attached cannot be killed by anybody until the Medic is, and a Demoman is
			one of the two classes that can reach it. */
			if (TF2_IsMiniBoss(i) || TF2_HasTheFlag(i) || TF2_GetPlayerClass(i) == TFClass_Medic)
				worthItAlone = true;
		}

		if (caught > 0)
			bombsWithEnemies++;

		caughtTotal += caught;

		/* A tank is not a player, so none of the counting above sees one
		Without this a bot puts a clip into the hull and never presses the button, which is the
		same weapon doing nothing that this file exists to fix */
		if (IsStickyOnTank(stickyOrigin))
			worthItAlone = true;
	}

	return worthItAlone
		|| caughtTotal >= STICKY_DETONATE_ENEMIES
		|| bombsWithEnemies >= STICKY_DETONATE_BOMBS;
}

//Whether a bomb at this position is stuck to a tank, or close enough to hurt one
static bool IsStickyOnTank(const float stickyOrigin[3])
{
	int tank = -1;

	while ((tank = FindEntityByClassname(tank, "tank_boss")) != -1)
	{
		if (!IsBaseBoss(tank))
			continue;

		if (GetVectorDistance(WorldSpaceCenter(tank), stickyOrigin) <= STICKY_TANK_RANGE)
			return true;
	}

	return false;
}

/* Whether this Demoman should be holding the sticky launcher rather than the pipes

Both are the same arc and the same splash, so this is not about which does more damage. It is
about which one lands. A pipe has to be timed onto a moving robot; a sticky sticks where it hits
and waits for the bot to decide, which is a decision a bot makes better than a lead.

That reasoning is why the launcher was tried as the default weapon, and it was measured and it was
wrong. Six waves of Coaltown either way, one build, one switch between them:

  pipes first    1821 damage a wave, 27 kills, five waves of six cleared
  stickies first  880 damage a wave, 11 kills, four waves of six cleared

Half the damage. The hole in the argument is that the bot fires at where a robot is rather than
where it is going, and a sticky thrown at a walking robot lands behind it and catches nobody. The
clip and the reload are spent for nothing, where a pipe at least does its damage when it connects.
Sticky spam is a human laying bombs on ground the robots have not reached yet, and none of that is
what this does.

So: stickies at the things worth a cluster, pipes at everything else. Close in it is pipes whatever
the target, because a sticky under the bot's own feet is a bot blowing itself up. The trap is the
part that would actually pay, and it is TODO item 6a rather than a switch on this function. */
bool ShouldUseStickyLauncher(int client, int launcher, int threat, float range)
{
	if (launcher == -1 || TF2Util_GetWeaponID(launcher) != TF_WEAPON_PIPEBOMBLAUNCHER)
		return false;

	if (!HasAmmo(launcher))
		return false;

	if (range < STICKY_SELF_SAFE_RANGE)
		return false;

	//Out past this the arc is guesswork the bot does not charge the shot for
	if (range > 1200.0)
		return false;

	if (!BaseEntity_IsPlayer(threat))
		return false;

	if (TF2_IsMiniBoss(threat) || TF2_HasTheFlag(threat))
		return true;

	//A Medic is the one robot the rest of the team cannot finish around, so it is worth the switch
	if (TF2_GetPlayerClass(threat) == TFClass_Medic)
		return true;

	//A crowd, counted where it stands rather than where the bombs would land
	return CountEnemiesNearPosition(client, WorldSpaceCenter(threat), STICKY_BLAST_RANGE) >= STICKY_DETONATE_ENEMIES;
}
