/*
Package attacktank is source/redbots3/behavior/attacktank.sp.

Shooting the tank, and choosing what to shoot it with.

//sp:action DefenderAttackTank CTFBotAttackTank
*/
package attacktank

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

const (
	//sp:name TANK_ATTACK_RANGE_MELEE
	rangeMelee = 1.0
	//sp:name TANK_ATTACK_RANGE_SPLASH
	rangeSplash = 400.0
	//sp:name TANK_ATTACK_RANGE_DEFAULT
	rangeDefault = 100.0
)

/*
How close a man carrying a blast weapon may get to the side of a tank

A rocket's own blast radius is a hundred and forty six units and it goes off on
whatever it hits, which against a tank is the hull rather than the point the range
was measured to. Two hundred and fifty leaves a rocket's worth of margin on top of
that for a tank that is still driving at him.
*/
//
//sp:name TANK_BLAST_SAFE_RANGE
const blastSafeRange = 250.0

//sp:name m_iTankTarget
var tankTarget [Slots]int32

// OnStart only sets the look-ahead: the target was chosen before the action
// started.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	// NOTE: CTFBotAttackTank_SelectTarget chooses a tank threat beforehand

	return engine.Continue()
}

// Update closes to the range the weapon wants, and backs off a hull it would
// blow itself up on.
func Update(actor int32) engine.Outcome {
	if !engine.IsValidEntity(tankTarget[actor]) {
		if !SelectTarget(actor) {
			return engine.Done("No valid target")
		}
	}

	switch engine.PlayerClass(actor) {
	case engine.ClassScout():
		// We still prefer money
		if engine.CollectMoneyIsPossible(actor) {
			return engine.ChangeTo(engine.CollectMoney(), "Get credits")
		}
	case engine.ClassHeavyweapons(), engine.ClassSniper():
		// We're more useful against the robots than the tank
		if engine.DefenderAttackSelectTarget(actor) {
			return engine.ChangeTo(engine.DefenderAttack(), "Robot priority")
		}
	}

	EquipBestWeapon(actor)

	myEyePos := engine.ClientEyePosition(actor)
	targetOrigin := engine.WorldSpaceCenter(tankTarget[actor])
	distToTank := engine.VectorDistance(myEyePos, targetOrigin)

	myBot := engine.NextBotOf(actor)

	attackRange := IdealAttackRange(actor)

	/* Backing off a tank he is already inside the blast radius of

	Every range here is measured to the middle of the tank, and a tank is a large box: the hull he
	actually detonates a rocket against is half a tank nearer than that. So a standoff that reads
	as safe from the centre is not one from the front, and what that produced is soldiers killing
	themselves on tanks, reported from play.

	Measured off the collision box rather than by making the centre distance bigger, because the
	box is the thing the rocket hits and it is the same answer whichever end of the tank he is
	standing at. */
	if IsBlastWeapon(engine.ActiveWeapon(actor)) &&
		RangeToHull(myEyePos, tankTarget[actor]) < blastSafeRange {
		away := engine.SubtractVectors(myEyePos, targetOrigin)

		away[2] = 0.0

		length, away := engine.NormalizeVector(away)

		if length > 0.0 {
			away = engine.ScaleVector(away, blastSafeRange)
			away = engine.AddVectors(myEyePos, away)

			myBot.Locomotion().ApproachWeighted(away, 1.0)
		}

		return engine.Continue()
	}

	if distToTank > attackRange || !engine.IsLineOfFireClearEntity(actor, myEyePos, tankTarget[actor]) {
		if engine.RepathTime(actor) <= engine.GameTime() {
			engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(0.5, 1.0))
			// Its own arguments: a tank is a moving hull, and the goal is wanted even when the path fails
			engine.PathOf(actor).ComputeToPos(myBot, engine.AbsOriginOf(tankTarget[actor]), 0.0, true)
		}

		engine.PathOf(actor).Update(myBot)
	}

	return engine.Continue()
}

// SelectMoreDangerousThreat keeps the tank the threat unless something nearer is
// shooting at him.
//
//nolint:revive // unused-parameter: the signature is the engine's, not ours
func SelectMoreDangerousThreat(nextbot engine.Bot, entity int32, threat1 engine.Known, threat2 engine.Known) (changed engine.Outcome, knownEntity engine.Known) {
	iThreat1 := threat1.Entity()
	iThreat2 := threat2.Entity()

	me := engine.Actor()
	myWeapon := engine.ActiveWeapon(me)

	if myWeapon != -1 && engine.IsMeleeWeapon(myWeapon) {
		// Close range weapons only target the closest threat
		return engine.Changed(), engine.SelectCloserThreat(nextbot, threat1, threat2)
	}

	// Nearby enemies might try to kill us
	notSafeRange := engine.FlamethrowerReachRange()

	if engine.IsPlayer(iThreat1) {
		if nextbot.IsRangeLessThan(iThreat1, notSafeRange) {
			return engine.Changed(), threat1
		}
	}

	if engine.IsPlayer(iThreat2) {
		if nextbot.IsRangeLessThan(iThreat2, notSafeRange) {
			return engine.Changed(), threat2
		}
	}

	// Our most dangerous threat should be the tank
	if iThreat1 == tankTarget[me] {
		return engine.Changed(), threat1
	}

	if iThreat2 == tankTarget[me] {
		return engine.Changed(), threat2
	}

	// We probably can't see it right now
	return engine.Changed(), engine.NoKnownEntity()
}

// SelectTarget picks a tank, unless enough bots are already on one.
//
//sp:name CTFBotAttackTank_SelectTarget
func SelectTarget(actor int32) bool {
	if engine.CountOfBotsWithNamedActionExcept("DefenderAttackTank", actor) >= engine.MaxTankAttackers().Int() {
		return false
	}

	tankTarget[actor] = TankToTarget(actor, 999999.0)

	return tankTarget[actor] != -1
}

// TankToTarget is the nearest tank on the other team.
//
//sp:name GetTankToTarget
//sp:default maxDistance 999999.0
//nolint:misspell // the TODO below is the shipped comment, spelling and all
func TankToTarget(actor int32, maxDistance float32) int32 {
	// TODO: We should be targetting the closest tank that has the farthest progress
	// to the hatch instead of going for the closest one to us

	origin := engine.Origin(actor)
	myTeam := engine.GetClientTeam(actor)
	primary := engine.PlayerWeaponSlot(actor, engine.WeaponSlotPrimary())
	// int rather than the weapon tag: TF2Util_GetWeaponID answers a plain
	// cell here and the plugin's includes have no TFWeaponType to hold it in.
	primaryID := int32(-1)

	if primary != -1 {
		primaryID = int32(engine.WeaponID(primary))
	}

	bestDistance := float32(999999.0)
	bestEntity := int32(-1)

	ent := int32(-1)

	for {
		ent = engine.FindEntityByClassname(ent, "tank_boss")

		if ent == -1 {
			break
		}

		// Ignore tanks on our team
		if myTeam == engine.EntityTeamNumber(ent) {
			continue
		}

		if primaryID == int32(engine.WeaponFlamethrower()) {
			// Somehow this tank is in the air, we can't reach it with this weapon
			if engine.EntityFlags(ent)&engine.FlagOnGround() == 0 {
				continue
			}
		}

		distance := engine.VectorDistance(origin, engine.WorldSpaceCenter(ent))

		if distance <= bestDistance && distance <= maxDistance {
			bestDistance = distance
			bestEntity = ent
		}
	}

	return bestEntity
}

// IdealAttackRange is how close the weapon in his hands wants to be.
//
//sp:name GetIdealTankAttackRange
func IdealAttackRange(client int32) float32 {
	weapon := engine.ActiveWeapon(client)

	if weapon > 0 {
		if engine.IsMeleeWeapon(weapon) {
			// TODO: factor in other factors for melee
			// GetSwingRange
			// melee_bounds_multiplier
			// melee_range_multiplier

			return rangeMelee
		}

		switch engine.WeaponID(weapon) {
		case engine.WeaponRocketlauncher(), engine.WeaponGrenadelauncher(), engine.WeaponFlaregun(), engine.WeaponDirecthit(), engine.WeaponParticleCannon(), engine.WeaponCannon():
			return rangeSplash
		}
	}

	return rangeDefault
}

// IsBlastWeapon says whether firing this at something touching the man would hurt
// him as well as it.
//
//sp:name IsBlastWeapon
func IsBlastWeapon(weapon int32) bool {
	if weapon < 1 {
		return false
	}

	switch engine.WeaponID(weapon) {
	/* The stickybomb launcher is the one this list was missing, and it is the one that mattered

	Its only caller is the tank standoff, which exists because soldiers were killing themselves
	on hulls. The Demoman was never covered by it: he scored the sticky launcher highest of
	anything for a tank, walked to the hull because a weapon that is not a blast weapon needs no
	standoff, laid eight bombs on it and pressed the button. He is the worst self-harmer on the
	team by an order of magnitude and this is half of why. Not a switch: a bomb that does splash
	damage is a blast weapon, and saying otherwise was simply wrong. */
	case engine.WeaponRocketlauncher(), engine.WeaponGrenadelauncher(), engine.WeaponDirecthit(),
		engine.WeaponParticleCannon(), engine.WeaponCannon(), engine.WeaponPipebombLauncher():
		return true
	}

	return false
}

/*
RangeToHull is how far the nearest corner of the tank is, rather than how far its
middle is.

The collision box is what a rocket goes off against. Yaw is ignored: the box is
axis aligned and a tank driving a diagonal is a few units of error in something
that already carries a rocket of margin.
*/
//
//sp:name RangeToTankHull
func RangeToHull(from [3]float32, tank int32) float32 {
	origin := engine.AbsOriginOf(tank)

	if !engine.HasEntProp(tank, engine.PropSend(), "m_vecMins") || !engine.HasEntProp(tank, engine.PropSend(), "m_vecMaxs") {
		return engine.VectorDistance(from, engine.WorldSpaceCenter(tank))
	}

	mins := engine.EntPropVector(tank, engine.PropSend(), "m_vecMins")
	maxs := engine.EntPropVector(tank, engine.PropSend(), "m_vecMaxs")

	var closest [3]float32

	for axis := int32(0); axis < 3; axis++ {
		closest[axis] = engine.ClampFloat(from[axis], origin[axis]+mins[axis], origin[axis]+maxs[axis])
	}

	return engine.VectorDistance(from, closest)
}

// EquipBestWeapon uses a score-based system to determine what weapon the bot
// should be using against a tank boss.
//
//sp:name EquipBestTankWeapon
func EquipBestWeapon(client int32) {
	bestWeapon := int32(-1)
	bestScore := int32(0)

	for slot := engine.WeaponSlotPrimary(); slot <= engine.WeaponSlotMelee(); slot++ {
		weapon := engine.PlayerWeaponSlot(client, slot)

		if weapon == -1 {
			continue
		}

		score := int32(0)

		switch engine.PlayerClass(client) {
		case engine.ClassScout():
			score = EvalScout(slot, weapon)
		case engine.ClassSniper():
			score = EvalSniper(slot, weapon)
		case engine.ClassSoldier():
			score = EvalSoldier(slot, weapon)
		case engine.ClassDemoMan():
			score = EvalDemo(slot, weapon)
		case engine.ClassMedic():
			score = EvalMedic(slot, weapon)
		case engine.ClassHeavyweapons():
			score = EvalHeavy(slot, weapon)
		case engine.ClassPyro():
			score = EvalPyro(slot, weapon)
		case engine.ClassSpy():
			score = EvalSpy(slot, weapon)
		case engine.ClassEngineer():
			score = EvalEngie(slot, weapon)
		}

		if bestWeapon == -1 || score > bestScore {
			bestWeapon = weapon
			bestScore = score
		}
	}

	if bestWeapon == -1 {
		engine.LogError("EquipBestTankWeapon: no valid weapons!")
		return
	}

	engine.SetPlayerActiveWeapon(client, bestWeapon)
}

// EvalScout scores a weapon for the scout against a tank.
//
//sp:name EvalTankWeapon_Scout
func EvalScout(slot int32, weapon int32) int32 {
	switch engine.WeaponID(weapon) {
	case engine.WeaponScattergun(), engine.WeaponPepBrawlerBlaster(), engine.WeaponSodaPopper(), engine.WeaponHandgunScoutPrimary():
		return 100
	case engine.WeaponBat(), engine.WeaponBatFish(), engine.WeaponBatWood():
		return 80
	case engine.WeaponPistol(), engine.WeaponPistolScout(), engine.WeaponHandgunScoutSec():
		return 60
	case engine.WeaponBatGiftwrap():
		return 20
	case engine.WeaponCleaver(), engine.WeaponLunchbox(), engine.WeaponJar(), engine.WeaponJarMilk():
		return 0
	}

	switch slot {
	case engine.WeaponSlotPrimary():
		return 100
	case engine.WeaponSlotSecondary():
		return 60
	case engine.WeaponSlotMelee():
		return 80
	default:
		return 10
	}
}

// EvalSoldier scores a weapon for the soldier against a tank.
//
//sp:name EvalTankWeapon_Soldier
func EvalSoldier(slot int32, weapon int32) int32 {
	switch engine.WeaponID(weapon) {
	case engine.WeaponRocketlauncher(), engine.WeaponParticleCannon(), engine.WeaponDirecthit():
		return 100
	case engine.WeaponShovel(), engine.WeaponBottle(), engine.WeaponSword():
		return 80
	case engine.WeaponShotgunPrimary(), engine.WeaponShotgunSoldier(), engine.WeaponShotgunHwg(), engine.WeaponShotgunPyro(), engine.WeaponRaygun():
		return 60
	case engine.WeaponBuffItem(), engine.WeaponParachute():
		return 0
	}

	switch slot {
	case engine.WeaponSlotPrimary():
		return 100
	case engine.WeaponSlotSecondary():
		return 60
	case engine.WeaponSlotMelee():
		return 80
	default:
		return 10
	}
}

// EvalPyro scores a weapon for the pyro against a tank.
//
//sp:name EvalTankWeapon_Pyro
func EvalPyro(slot int32, weapon int32) int32 {
	switch engine.WeaponID(weapon) {
	case engine.WeaponFlamethrower(), engine.WeaponFlamethrowerRocket():
		return 100
	case engine.WeaponFireaxe():
		return 80
	case engine.WeaponShotgunPrimary(), engine.WeaponShotgunSoldier(), engine.WeaponShotgunHwg(), engine.WeaponShotgunPyro():
		return 60
	case engine.WeaponFlaregun(), engine.WeaponRaygunRevenge():
		return 20
	}

	switch slot {
	case engine.WeaponSlotPrimary():
		return 100
	case engine.WeaponSlotSecondary():
		return 20
	case engine.WeaponSlotMelee():
		return 80
	default:
		return 10
	}
}

// EvalDemo scores a weapon for the demo against a tank.
//
//sp:name EvalTankWeapon_Demo
func EvalDemo(slot int32, weapon int32) int32 {
	switch engine.WeaponID(weapon) {
	case engine.WeaponGrenadelauncher(), engine.WeaponCannon():
		return 100
	case engine.WeaponPipebombLauncher():
		if engine.Feature(engine.FeatureDemoTankPipes()) {
			return 0
		}

		return 110
	case engine.WeaponBottle(), engine.WeaponShovel(), engine.WeaponSword(), engine.WeaponStickbomb():
		return 80
	case engine.WeaponBuffItem(), engine.WeaponParachute(), engine.WeaponStickyBallLauncher():
		return 0
	}

	switch slot {
	case engine.WeaponSlotPrimary():
		return 100
	case engine.WeaponSlotSecondary():
		return 0
	case engine.WeaponSlotMelee():
		return 80
	default:
		return 10
	}
}

// EvalHeavy scores a weapon for the heavy against a tank.
//
//sp:name EvalTankWeapon_Heavy
func EvalHeavy(slot int32, weapon int32) int32 {
	switch engine.WeaponID(weapon) {
	case engine.WeaponMinigun():
		return 100
	case engine.WeaponFists(), engine.WeaponFireaxe():
		return 80
	case engine.WeaponShotgunPrimary(), engine.WeaponShotgunSoldier(), engine.WeaponShotgunHwg(), engine.WeaponShotgunPyro():
		return 60
	case engine.WeaponLunchbox():
		return 0
	}

	switch slot {
	case engine.WeaponSlotPrimary():
		return 100
	case engine.WeaponSlotSecondary():
		return 0
	case engine.WeaponSlotMelee():
		return 60
	default:
		return 10
	}
}

// EvalEngie scores a weapon for the engie against a tank.
//
//sp:name EvalTankWeapon_Engie
func EvalEngie(slot int32, weapon int32) int32 {
	switch engine.WeaponID(weapon) {
	case engine.WeaponWrench(), engine.WeaponMechanicalArm():
		return 100
	case engine.WeaponShotgunPrimary(), engine.WeaponShotgunSoldier(), engine.WeaponShotgunHwg(), engine.WeaponShotgunPyro(), engine.WeaponSentryRevenge(), engine.WeaponDrgPomson():
		return 80
	case engine.WeaponShotgunBuildingRescue():
		return 60
	case engine.WeaponPistol(), engine.WeaponPistolScout(), engine.WeaponRevolver():
		return 40
	case engine.WeaponPda(), engine.WeaponPdaEngineerBuild(), engine.WeaponPdaEngineerDestroy(), engine.WeaponPdaSpy(), engine.WeaponPdaSpyBuild(), engine.WeaponBuilder(), engine.WeaponLaserPointer(), engine.WeaponDispenser(), engine.WeaponDispenserGun():
		return 0
	}

	switch slot {
	case engine.WeaponSlotPrimary():
		return 80
	case engine.WeaponSlotSecondary():
		return 0
	case engine.WeaponSlotMelee():
		return 100
	default:
		return 10
	}
}

// EvalMedic scores a weapon for the medic against a tank.
//
//sp:name EvalTankWeapon_Medic
func EvalMedic(slot int32, weapon int32) int32 {
	switch engine.WeaponID(weapon) {
	case engine.WeaponBonesaw(), engine.WeaponHarvesterSaw():
		return 100
	case engine.WeaponSyringegunMedic(), engine.WeaponNailgun():
		return 80
	case engine.WeaponCrossbow():
		return 60
	case engine.WeaponMedigun():
		return 0
	}

	switch slot {
	case engine.WeaponSlotPrimary():
		return 80
	case engine.WeaponSlotSecondary():
		return 0
	case engine.WeaponSlotMelee():
		return 100
	default:
		return 10
	}
}

// EvalSniper scores a weapon for the sniper against a tank.
//
//sp:name EvalTankWeapon_Sniper
func EvalSniper(slot int32, weapon int32) int32 {
	switch engine.WeaponID(weapon) {
	case engine.WeaponSniperrifle(), engine.WeaponSniperrifleDecap(), engine.WeaponSniperrifleClassic():
		return 100
	case engine.WeaponCompoundBow():
		return 80
	case engine.WeaponClub():
		return 60
	case engine.WeaponChargedSmg(), engine.WeaponSmg():
		return 40
	case engine.WeaponJar(), engine.WeaponJarMilk():
		return 0
	}

	switch slot {
	case engine.WeaponSlotPrimary():
		return 100
	case engine.WeaponSlotSecondary():
		return 40
	case engine.WeaponSlotMelee():
		return 60
	default:
		return 10
	}
}

// EvalSpy scores a weapon for the spy against a tank.
//
//sp:name EvalTankWeapon_Spy
func EvalSpy(slot int32, weapon int32) int32 {
	switch engine.WeaponID(weapon) {
	case engine.WeaponRevolver():
		return 100
	case engine.WeaponKnife():
		return 80
	case engine.WeaponPda(), engine.WeaponPdaEngineerBuild(), engine.WeaponPdaEngineerDestroy(), engine.WeaponPdaSpy(), engine.WeaponPdaSpyBuild(), engine.WeaponBuilder(), engine.WeaponInvis():
		return 0
	}

	switch slot {
	case engine.WeaponSlotPrimary():
		return 100
	case engine.WeaponSlotSecondary():
		return 0
	case engine.WeaponSlotMelee():
		return 80
	default:
		return 10
	}
}

// ResetAttackTank forgets the tank this bot was shooting.
//
// A bot leaving takes its seat's state with it, and the next bot in that seat
// is a different bot.
func ResetAttackTank(client int32) {
	tankTarget[client] = -1
}
