/*
Package attack is source/redbots3/behavior/attack.sp.

Fighting: pick a robot, keep it, walk at it, shoot it. Sixteenth behaviour
across, and the busiest hand-off in the mod: four of the behaviours it can turn
into are generated already, so all four are body externs.

//sp:action DefenderAttack CTFBotDefenderAttack static
*/
package attack

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/action/campbomb"
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

//sp:name m_iAttackTarget
var attackTarget [slots.Count]int32

//sp:name m_flRevalidateTarget
var revalidateTarget [slots.Count]float32

/*
Sidestepping while it shoots, because a bot that has arrived stops moving
entirely.

The path below runs while the target is too far off or behind cover, and nothing
runs once it is neither. So the bot walks up, plants its feet, and stands still
for the rest of the fight, which is the one thing every guide about this game
tells a person not to do. A stationary target is what a robot's aim was written
for, and the projectile classes among them do not miss one.

Sidestepping costs nothing in accuracy here. Projectiles do not inherit the
shooter's velocity, and the head is aimed by different code from the code that
moves the feet, so a bot that strafes shoots exactly as well as one that stands.

Approach and not a path: this is a step to one side, not a journey, and computing
a path for every flip of it would be a nav mesh search several times a second per
bot, for a distance the bot covers in a quarter of one.

The step is tested before it is taken. Locomotion stops itself walking into a
wall; it will happily walk off a ledge, and a Demoman who sidesteps into the pit
on Rottenburg has solved the wrong problem.
*/
const (
	//sp:name ATTACK_STRAFE_REACH
	strafeReach = 130.0
	//sp:name ATTACK_STRAFE_FLIP_MIN
	strafeFlipMin = 0.5
	//sp:name ATTACK_STRAFE_FLIP_MAX
	strafeFlipMax = 1.1
)

var (
	//sp:name m_ctAttackStrafeFlip
	strafeFlip [slots.Count]float32
	//sp:name m_bAttackStrafeRight
	strafeRight [slots.Count]bool
)

// OnStart aims the path. The target is usually chosen before this action starts.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	// NOTE: the attack target is usually chosen before we enter this action with CTFBotDefenderAttack_SelectTarget

	revalidateTarget[actor] = engine.GameTime() + 3.0

	return engine.Continue()
}

// Update keeps the target honest, hands off to whatever outranks fighting, and
// otherwise walks at the robot and shoots it.
func Update(actor int32) engine.Outcome {
	if engine.PlayerClass(actor) == engine.ClassSniper() && engine.TFBotMission(actor) == engine.MissionSniper() {
		if engine.CanUsePrimaryWeapon(actor) {
			// We can snipe again
			return engine.Done("I have gun")
		}
	}

	if campbomb.IsPossible(actor) {
		return engine.ChangeTo(engine.CampBomb(), "Camp bomb")
	}

	if engine.GuardPointIsPossible(actor) {
		return engine.ChangeTo(engine.GuardPoint(), "Defending a point")
	}

	if engine.DestroyTeleporterSelectTarget(actor) {
		return engine.SuspendFor(engine.DestroyTeleporter(), "Get teleporter")
	}

	if !engine.IsValidClientIndex(attackTarget[actor]) ||
		!engine.IsPlayerAlive(attackTarget[actor]) ||
		engine.PlayerTeam(attackTarget[actor]) != engine.PlayerEnemyTeam(actor) {
		if !SelectTarget(actor, false) {
			return engine.Done("Target is not valid")
		}
	}

	if revalidateTarget[actor] <= engine.GameTime() {
		revalidateTarget[actor] = engine.GameTime() + 2.0

		// Need new target.
		if !TargetEntityReachable(actor, attackTarget[actor]) {
			if !SelectTarget(actor, false) {
				return engine.Done("Unreachable target")
			}
		}
	}

	switch engine.PlayerClass(actor) {
	case engine.ClassScout():
		// Scouts primarily prefer to get money
		if engine.CollectMoneyIsPossible(actor) {
			return engine.ChangeTo(engine.CollectMoney(), "Collectinh money")
		}
	case engine.ClassSoldier(), engine.ClassPyro(), engine.ClassDemoMan():
		// These classes prefer priortizing the tank more than anything
		if engine.AttackTankSelectTarget(actor) {
			return engine.ChangeTo(engine.AttackTank(), "Changing threat to tank")
		}
	case engine.ClassMedic():
		// Make sure we have our medigun before we even think about leaving this action
		secondary := engine.PlayerWeaponSlot(actor, engine.WeaponSlotSecondary())

		if secondary != -1 {
			for i := int32(1); i <= engine.MaxClients(); i++ {
				if engine.IsClientInGame(i) && engine.GetClientTeam(i) == engine.GetClientTeam(actor) && engine.IsPlayerAlive(i) {
					class := engine.PlayerClass(i)

					if class != engine.ClassMedic() && class != engine.ClassSniper() && class != engine.ClassEngineer() && class != engine.ClassSpy() {
						// We have someone we'd prefer to heal
						return engine.Done("I have patient")
					}
				}
			}
		}
	}

	// TODO: Other classes should go for money, but only when there isn't a threat around

	SelectTarget(actor, true)

	myBot := engine.NextBotOf(actor)
	targetOrigin := engine.Origin(attackTarget[actor])
	myEyePos := engine.ClientEyePosition(actor)

	// Path if out of range or cannot see target
	if myBot.IsRangeGreaterThanEx(targetOrigin, engine.DesiredAttackRange(actor)) || !engine.IsLineOfFireClearPosition(actor, myEyePos, targetOrigin) {
		if engine.RepathTime(actor) <= engine.GameTime() {
			engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(0.3, 0.4))
			engine.RepathToTarget(actor, myBot, attackTarget[actor])
		}

		/* Walked, and not stepped toward when the mesh refuses: measured, and a fighter is not a medic

		The same nudge in PluginBot_SimulateFrame took the medic from four percent of a wave with
		his beam connected to thirty, because a path that fails on the way to a teammate is a bot
		standing still for nothing. Here it was worth twenty three percent more damage out of the
		Soldier and the Demoman over twelve waves, and cost the team thirty seven percent more
		deaths, flat total damage and a wave.

		Reaching a friend is safe and reaching a robot is not, and where the mesh will not path is
		often ground worth not standing on. */
		engine.PathOf(actor).Update(myBot)
	} else if engine.Feature(engine.FeatureAttackStrafe()) {
		StrafeWhileFighting(actor, myBot, targetOrigin)
	}

	myVision := myBot.Vision()
	threat := myVision.PrimaryKnownThreat(false)

	if threat != 0 {
		// We have a threat, prepare to fight it
		engine.EquipBestWeaponForThreat(actor, threat)
	}

	return engine.Continue()
}

// StrafeWhileFighting takes one step to the side, tested before it is taken.
//
//sp:name StrafeWhileFighting
func StrafeWhileFighting(actor int32, myBot engine.Bot, targetOrigin [3]float32) {
	// A Sniper is aiming down a scope and wants his feet exactly where they are
	if engine.PlayerClass(actor) == engine.ClassSniper() {
		return
	}

	myLoco := myBot.Locomotion()

	if !myLoco.IsOnGround() || myLoco.IsClimbingOrJumping() {
		return
	}

	if strafeFlip[actor] < engine.GameTime() {
		strafeFlip[actor] = engine.GameTime() + engine.RandomFloat(strafeFlipMin, strafeFlipMax)
		strafeRight[actor] = !strafeRight[actor]
	}

	myOrigin := engine.AbsOriginOf(actor)
	toTarget := engine.SubtractVectors(targetOrigin, myOrigin)

	toTarget[2] = 0.0

	length, toTarget := engine.NormalizeVector(toTarget)

	if length < 1.0 {
		return
	}

	// Square to the way it is facing, in the plane it walks on
	var side [3]float32

	side[0] = toTarget[1]
	side[1] = -toTarget[0]

	if strafeRight[actor] {
		side[0] = -toTarget[1]
		side[1] = toTarget[0]
	}

	side[2] = 0.0

	var step [3]float32

	for axis := range step {
		step[axis] = myOrigin[axis] + side[axis]*strafeReach
	}

	// Turn round early rather than walk into whatever is there, or off it
	if !myLoco.IsPotentiallyTraversable(myOrigin, step, engine.Immediately()) || myLoco.HasPotentialGap(myOrigin, step) {
		strafeFlip[actor] = engine.GameTime()

		return
	}

	myLoco.Approach(step)
}

// SelectTarget picks the robot worth shooting: the one nearest the bomb, then
// whoever is healing it.
//
//sp:default bBombCarrierOnly false
//sp:name CTFBotDefenderAttack_SelectTarget
func SelectTarget(actor int32, bBombCarrierOnly bool) bool {
	// Always go after the bot closest to the bomb, if possible
	target := engine.BotNearestToBombNearestToHatch(actor)

	// No bomb in play, just find random target
	if !bBombCarrierOnly && target == -1 {
		target = engine.SelectRandomReachableEnemy(actor)
	}

	// Found a valid target, update
	if target != -1 {
		// Go after the healer first
		healer := engine.HealerOfPlayer(target, true)

		if healer != -1 {
			target = healer
		}

		attackTarget[actor] = target
		return true
	}

	return false
}

// TargetEntityReachable says the bot could actually get to him, which mostly
// means he is not standing in his own spawn.
//
//sp:name IsTargetEntityReachable
func TargetEntityReachable(client int32, target int32) bool {
	area := engine.CombatOf(target).LastKnownArea()

	if area == engine.NavArea(engine.NullArea()) {
		return false
	}

	if (engine.PlayerTeam(client) == engine.TeamRed() && area.HasAttributeTF(engine.BlueSpawnRoom())) ||
		(engine.PlayerTeam(client) == engine.TeamBlue() && area.HasAttributeTF(engine.RedSpawnRoom())) {
		// Usually cannot enter enemy spawns
		return false
	}

	return true
}

// ResetAttack forgets the robot this bot was shooting.
//
// A bot leaving takes its seat's state with it, and the next bot in that seat
// is a different bot.
func ResetAttack(client int32) {
	attackTarget[client] = -1
}
