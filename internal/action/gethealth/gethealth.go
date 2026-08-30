/*
Package gethealth is source/redbots3/behavior/gethealth.sp.

Walking to health when there is any worth walking to. Seventeenth behaviour
across.

//sp:action DefenderGetHealth CTFBotGetHealth
*/
package gethealth

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

//sp:name m_iHealthPack
var healthPack [Slots]int32

/*
How long an answer about health is kept for.

The same reasoning as the ammo the tactical monitor asks about on the same
frame, and the same cost: a nav mesh search per candidate, for a bot that is
below the health ratio and has nothing reachable, sixty-six times a second. Half
a second is a bot walking about a hundred and fifty units, and nothing that
matters appears inside that.
*/
//
//sp:name HEALTH_ASK_INTERVAL
const askInterval = 0.5

var (
	//sp:name m_ctHealthAsk
	healthAsk [Slots]float32
	//sp:name m_bHealthPossible
	healthPossible [Slots]bool
)

/*
The range computation is written out twice, once here and once in IsPossible,
because the shipped file writes it twice. Lifting it into a helper changes the
sequence of calls each of them makes, and the port is compared on that sequence:
a tidy that rides along with a port cannot be told from one that does not.
*/

// OnStart picks the nearest health of what is in range.
func OnStart(actor int32) engine.Outcome {
	healthRatio := float32(engine.ClientHealth(actor)) / float32(engine.PlayerMaxHealth(actor))
	ratio := engine.ClampFloat((healthRatio-engine.HealthCriticalRatio().Float())/(engine.HealthOKRatio().Float()-engine.HealthCriticalRatio().Float()), 0.0, 1.0)

	farRange := engine.HealthSearchFarRange().Float()
	maxRange := ratio * (engine.HealthSearchNearRange().Float() - farRange)
	maxRange += farRange

	ammo := engine.NewBlocks(2)
	defer ammo.Close()

	engine.ComputeHealthAndAmmoVectors(actor, ammo, maxRange)

	if ammo.Length() <= 0 {
		return engine.Done("No health")
	}

	flSmallestDistance := float32(99999.0)

	for i := int32(0); i < ammo.Length(); i++ {
		entity := ammo.GetAt(i, 0)

		if !IsValidHealth(entity) {
			continue
		}

		flDistance := engine.AsFloat(ammo.GetAt(i, 1))

		if flDistance <= flSmallestDistance {
			healthPack[actor] = entity
			flSmallestDistance = flDistance
		}
	}

	if healthPack[actor] != -1 {
		if engine.PlayerClass(actor) == engine.ClassEngineer() {
			engine.UpdateLookAroundForEnemies(actor, true)
		}

		engine.SpeakConceptIfAllowed(actor, engine.ConceptMedic())
		return engine.Continue()
	}

	return engine.Done("Could not find health")
}

// Update walks to it, and stands still while a dispenser is doing the work.
func Update(actor int32) engine.Outcome {
	if !IsValidHealth(healthPack[actor]) {
		return engine.Done("Health is not valid")
	}

	if engine.IsHealedByMedic(actor) {
		return engine.Done("A medic heals me")
	}

	if engine.ClientHealth(actor) >= engine.PlayerMaxHealth(actor) {
		return engine.Done("I am healed")
	}

	if engine.IsCarryingObject(actor) {
		// Drop our building or we cant defend ourselves
		engine.PressFireButton(actor)
	}

	myBot := engine.NextBotOf(actor)

	if engine.IsHealedByObject(actor) {
		myWeapon := engine.ActiveWeapon(actor)

		if myWeapon != -1 && engine.WeaponIDIsSniperRifle(engine.WeaponID(myWeapon)) && !engine.IsPlayerInCondition(actor, engine.ConditionZoomed()) {
			// Aim while healed by dispenser
			engine.PressAltFireButton(actor)
		}
	} else {
		// Path if not currently healed by dispenser
		if engine.RepathTime(actor) <= engine.GameTime() {
			engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(0.9, 1.0))
			engine.RepathToPos(actor, myBot, engine.WorldSpaceCenter(healthPack[actor]))
		}

		engine.PathOf(actor).Update(myBot)
	}

	threat := myBot.Vision().PrimaryKnownThreat(false)

	if threat != 0 {
		engine.EquipBestWeaponForThreat(actor, threat)
	}

	return engine.Continue()
}

// OnEnd forgets the pack.
func OnEnd(actor int32) {
	healthPack[actor] = -1
}

// ShouldHurry says yes: a bot walking for health is not sightseeing.
//
//nolint:revive // unused-parameter: the signature is the engine's, not ours
func ShouldHurry(nextbot engine.Bot) (changed engine.Outcome, result engine.Answer) {
	return engine.Changed(), engine.AnswerYes()
}

// ShouldAttack keeps a hurt spy from picking a fight it cannot win.
//
//nolint:revive // unused-parameter: the signature is the engine's, not ours
func ShouldAttack(nextbot engine.Bot, knownEntity engine.Known) (changed engine.Outcome, result engine.Answer) {
	me := engine.Actor()

	if engine.PlayerClass(me) == engine.ClassSpy() {
		iThreat := knownEntity.Entity()

		if engine.IsPlayer(iThreat) && engine.ClientHealth(iThreat) > 360 && !engine.IsCritBoosted(me) {
			// Don't attack if we can't possibly kill them with our revolver (360 from 6 shots with max damage)
			return engine.Changed(), engine.AnswerNo()
		} else if engine.NearestEnemyCount(me, 1000.0, false) > 1 {
			// There's too many enemies nearby, it'd be better to redisguise so they'll forget about us
			return engine.Changed(), engine.AnswerNo()
		}
	}

	return engine.Changed(), engine.AnswerUndefined()
}

// IsValidHealth says the entity is health the bot could still take.
//
//sp:name IsValidHealth
func IsValidHealth(pack int32) bool {
	if !engine.IsValidEntity(pack) {
		return false
	}

	if !engine.HasEntProp(pack, engine.PropSend(), "m_fEffects") {
		return false
	}

	// It has been taken.
	if engine.EntProp(pack, engine.PropSend(), "m_fEffects") != 0 {
		return false
	}

	class := engine.EntityClassname(pack)

	if engine.StrContains(class, "item_health", false) == -1 &&
		engine.StrContains(class, "obj_dispenser", false) == -1 &&
		engine.StrContains(class, "func_regen", false) == -1 {
		return false
	}

	if engine.StrContains(class, "obj_dispenser", false) != -1 && engine.HasSapper(pack) {
		return false
	}

	return true
}

// IsPossible says whether there is health worth walking to, kept for a moment
// after it is worked out.
//
//sp:name CTFBotGetHealth_IsPossible
func IsPossible(actor int32) bool {
	if engine.IsHealedByMedic(actor) || engine.IsInvulnerable(actor) {
		return false
	}

	healthRatio := float32(engine.ClientHealth(actor)) / float32(engine.PlayerMaxHealth(actor))
	ratio := engine.ClampFloat((healthRatio-engine.HealthCriticalRatio().Float())/(engine.HealthOKRatio().Float()-engine.HealthCriticalRatio().Float()), 0.0, 1.0)

	farRange := engine.HealthSearchFarRange().Float()
	maxRange := ratio * (engine.HealthSearchNearRange().Float() - farRange)
	maxRange += farRange

	// Skip lag.
	if healthPack[actor] != -1 && IsValidHealth(healthPack[actor]) {
		return true
	}

	if healthAsk[actor] > engine.GameTime() {
		return healthPossible[actor]
	}

	healthAsk[actor] = engine.GameTime() + askInterval

	if engine.DebugActions().Bool() {
		engine.PrintToServer("ratio %f max_range %f", ratio, maxRange)
	}

	ammo := engine.NewBlocks(2)
	defer ammo.Close()

	engine.ComputeHealthAndAmmoVectors(actor, ammo, maxRange)

	bPossible := false

	for i := int32(0); i < ammo.Length(); i++ {
		if !IsValidHealth(ammo.GetAt(i, 0)) {
			continue
		}

		bPossible = true
		break
	}

	healthPossible[actor] = bPossible

	return bPossible
}
