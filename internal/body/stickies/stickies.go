/*
Package stickies is source/redbots3/demoman_stickies.sp.

The Demoman's stickybomb launcher, which used to be a weapon he carried and never
used.

Nothing in this repository detonated a stickybomb. There was no read of the bombs
in play and no alt-fire anywhere, so a bot holding the stock launcher fired eight
bombs that sat on the floor until they faded.

What is here: the bot fires stickies at what it is already fighting and blows them
when the blast pays. It is a sticky launcher used as a direct weapon, which is what
a bot can be trusted with.

What is not here: the trap. A human lays eight bombs on the ground the wave has to
walk over, backs off, and takes a giant apart with one press. That wants a bot that
picks the ground, waits on robots it cannot see yet, and gives up on a deadline,
and it is a bigger piece of work than this.
*/
package stickies

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// The stickybomb blast, which is the ground one bomb covers.
//
//sp:name STICKY_BLAST_RANGE
const blastRange = 146.0

// A cluster worth pressing for: two robots in the blast, or two bombs each with
// somebody on them.
const (
	//sp:name STICKY_DETONATE_ENEMIES
	detonateEnemies = 2
	//sp:name STICKY_DETONATE_BOMBS
	detonateBombs = 2
)

// How close to a tank a bomb still hurts it.
//
//sp:name STICKY_TANK_RANGE
const tankRange = 300.0

// A bomb this close to the bot is a bomb that takes the bot with it.
//
//sp:name STICKY_SELF_SAFE_RANGE
const selfSafeRange = 200.0

// The launcher holds eight, so nothing walks further than that.
//
//sp:name STICKY_MAX_BOMBS
const maxBombs = 8

/*
ShouldDetonateStickies is whether to press the detonator, and it is only ever
pressed for damage.

The cluster is whatever bombs happen to be on the ground, not a trap that was laid,
so the question is the same one asked of a rocket at somebody's feet: is there more
than one robot standing in the blast, or one robot big enough that the blast is
worth it on its own.
*/
//
//sp:name ShouldDetonateStickies
func ShouldDetonateStickies(client int32) bool {
	if engine.PlayerClass(client) != engine.ClassDemoMan() {
		return false
	}

	launcher := engine.PlayerWeaponSlot(client, engine.WeaponSlotSecondary())

	if launcher == -1 || engine.WeaponID(launcher) != engine.WeaponPipebombLauncher() {
		return false
	}

	// Nothing to blow up
	if engine.EntProp(launcher, engine.PropSend(), "m_iPipebombCount") <= 0 {
		return false
	}

	myOrigin := engine.Origin(client)
	enemyTeam := engine.PlayerEnemyTeam(client)

	examined := int32(0)
	sticky := int32(-1)

	/* Counted across the whole cluster rather than answered by the first bomb that qualifies

	Alt-fire blows all of them, so the question is what the cluster catches, not what one bomb
	catches. Asking it a bomb at a time meant two robots on two different bombs read as two bombs
	with one robot each and the button was never pressed. */
	caughtTotal := int32(0)
	bombsWithEnemies := int32(0)
	worthItAlone := false

	for {
		sticky = engine.FindEntityByClassname(sticky, "tf_projectile_pipe_remote")

		if sticky == -1 {
			break
		}

		// Somebody else's bombs, and blowing those up is not a button this bot has
		if engine.OwnerEntity(sticky) != client {
			continue
		}

		// The count above is the bot's own, so this is the same bound read from the other side
		examined++

		if examined > maxBombs {
			break
		}

		stickyOrigin := engine.AbsOriginOf(sticky)

		/* One bomb of his own on top of him and the button is not worth pressing at all

		This used to skip the bomb and carry on, which reads as a safety rule and is not one. The
		detonator is one button for every bomb he owns: skipping a close one only stops it counting
		towards whether to press, it does not stop it going off when he does. So a Demoman with six
		on a tank hull and two down the corridor scored the two, pressed, and took all eight.

		He is the worst self-harmer on the team by an order of magnitude and this is the mechanism.
		Vetoing outright rather than pricing it: the cluster he gives up is one press, the health he
		gives up is the rest of the wave. */
		if engine.VectorDistance(myOrigin, stickyOrigin) < selfSafeRange {
			if engine.Feature(engine.FeatureDemoStickySelfVeto()) {
				return false
			}

			continue
		}

		caught := int32(0)

		for i := int32(1); i <= engine.MaxClients(); i++ {
			if !engine.IsClientInGame(i) || !engine.IsPlayerAlive(i) {
				continue
			}

			if engine.PlayerTeam(i) != enemyTeam {
				continue
			}

			if engine.VectorDistance(engine.WorldSpaceCenter(i), stickyOrigin) > blastRange {
				continue
			}

			caught++

			/* A giant, the bomb carrier, or a Medic is worth the cluster by itself

			The Medic is the addition and it is the whole job on a wave that has them: a giant
			with one attached cannot be killed by anybody until the Medic is, and a Demoman is
			one of the two classes that can reach it. */
			if engine.IsMiniBoss(i) || engine.HasTheFlag(i) || engine.PlayerClass(i) == engine.ClassMedic() {
				worthItAlone = true
			}
		}

		if caught > 0 {
			bombsWithEnemies++
		}

		caughtTotal += caught

		/* A tank is not a player, so none of the counting above sees one
		Without this a bot puts a clip into the hull and never presses the button, which is the
		same weapon doing nothing that this file exists to fix */
		if IsStickyOnTank(stickyOrigin) {
			worthItAlone = true
		}
	}

	return worthItAlone ||
		caughtTotal >= detonateEnemies ||
		bombsWithEnemies >= detonateBombs
}

// IsStickyOnTank says a bomb at this position is stuck to a tank, or close enough
// to hurt one.
//
//sp:name IsStickyOnTank
func IsStickyOnTank(stickyOrigin [3]float32) bool {
	tank := int32(-1)

	for {
		tank = engine.FindEntityByClassname(tank, "tank_boss")

		if tank == -1 {
			break
		}

		if !engine.IsBaseBoss(tank) {
			continue
		}

		if engine.VectorDistance(engine.WorldSpaceCenter(tank), stickyOrigin) <= tankRange {
			return true
		}
	}

	return false
}

/*
ShouldUseStickyLauncher is whether this Demoman should be holding the sticky
launcher rather than the pipes.

Both are the same arc and the same splash, so this is not about which does more
damage. It is about which one lands. A pipe has to be timed onto a moving robot; a
sticky sticks where it hits and waits for the bot to decide, which is a decision a
bot makes better than a lead.

That reasoning is why the launcher was tried as the default weapon, and it was
measured and it was wrong. Six waves of Coaltown either way, one build, one switch
between them:

	pipes first    1821 damage a wave, 27 kills, five waves of six cleared
	stickies first  880 damage a wave, 11 kills, four waves of six cleared

Half the damage. The hole in the argument is that the bot fires at where a robot is
rather than where it is going, and a sticky thrown at a walking robot lands behind
it and catches nobody. The clip and the reload are spent for nothing, where a pipe
at least does its damage when it connects. Sticky spam is a human laying bombs on
ground the robots have not reached yet, and none of that is what this does.

So: stickies at the things worth a cluster, pipes at everything else. Close in it is
pipes whatever the target, because a sticky under the bot's own feet is a bot
blowing itself up.
*/
//
//sp:name ShouldUseStickyLauncher
func ShouldUseStickyLauncher(client int32, launcher int32, threat int32, threatRange float32) bool {
	if launcher == -1 || engine.WeaponID(launcher) != engine.WeaponPipebombLauncher() {
		return false
	}

	if !engine.HasAmmo(launcher) {
		return false
	}

	if threatRange < selfSafeRange {
		return false
	}

	// Out past this the arc is guesswork the bot does not charge the shot for
	if threatRange > 1200.0 {
		return false
	}

	if !engine.IsPlayer(threat) {
		return false
	}

	if engine.IsMiniBoss(threat) || engine.HasTheFlag(threat) {
		return true
	}

	// A Medic is the one robot the rest of the team cannot finish around, so it is worth the switch
	if engine.PlayerClass(threat) == engine.ClassMedic() {
		return true
	}

	// A crowd, counted where it stands rather than where the bombs would land
	return engine.CountEnemiesNearPosition(client, engine.WorldSpaceCenter(threat), blastRange) >= detonateEnemies
}
