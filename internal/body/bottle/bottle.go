/*
Package bottle is the powerup canteen out of
source/redbots3/nextbot_behavior.sp: finding the one the bot wears, and the
decision to drink it.
*/
package bottle

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

/*
The bottle this bot is wearing, kept rather than found again every frame.

Finding it walks the entity list looking for a tf_powerup_bottle, and this runs
on the player command, which is every frame for every bot. The bottle is a
wearable: it appears when the bot spawns and does not move afterwards, so it is
worth exactly one lookup a life.

The second was worse. This used to be a cached canteen type, written by the
purchase code, and the purchase code is gone: nothing wrote it any more, so the
switch below always read "no bottle" and a bot handed a canteen would never have
drunk it. The type comes off the bottle now, which is where it was always true.
*/
//
//sp:name m_hPowerupBottle
var powerupBottle = [Slots]int32{engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference(), engine.InvalidEntReference()}

//sp:name m_ctPowerupBottleLook
var powerupBottleLook [Slots]float32

//sp:name m_flNextBottleUseTime
var nextBottleUseTime [Slots]float32

// PowerupBottleOf is the canteen, at the cost of one entity walk a second at
// most.
//
//sp:name PowerupBottleOf
func PowerupBottleOf(client int32) int32 {
	bottle := engine.EntRefToEntIndex(powerupBottle[client])

	if bottle != engine.InvalidEntReference() {
		return bottle
	}

	// A bot with no bottle is the normal case now, and it should not cost an
	// entity walk a frame.
	if powerupBottleLook[client] > engine.GameTime() {
		return -1
	}

	powerupBottleLook[client] = engine.GameTime() + 1.0

	bottle = engine.FindPowerupBottle(client)

	if bottle != -1 {
		powerupBottle[client] = engine.EntIndexToEntRef(bottle)
	}

	return bottle
}

/*
OpportunisticallyUsePowerupBottle drinks the canteen when the moment fits what
it does: crits want a threat in reach of the weapon in hand, an uber wants the
bot about to die in front of somebody, a recall wants a bomb at the hatch the
bot cannot reach in time, and ammo wants an empty primary.
*/
//
//sp:name OpportunisticallyUsePowerupBottle
//sp:const threat
func OpportunisticallyUsePowerupBottle(client int32, activeWeapon int32, bot engine.Bot, threat engine.Known) bool {
	if nextBottleUseTime[client] > engine.GameTime() {
		return false
	}

	bottle := PowerupBottleOf(client)

	if bottle == -1 {
		return false
	}

	if engine.PowerupBottleCharges(bottle) < 1 {
		return false
	}

	switch engine.PowerupBottleKind(bottle) {
	case engine.BottleCritBoost():
		// Can't do anything useful without a weapon.
		if activeWeapon == -1 {
			return false
		}

		// No threat to actually use it against.
		if threat == engine.NoKnownEntity() {
			return false
		}

		// Medic would rather share this than use it for himself.
		if engine.PlayerClass(client) == engine.ClassMedic() {
			return false
		}

		// Already have crits.
		if engine.IsCritBoosted(client) || engine.IsPlayerInCondition(client, engine.ConditionCritMmmph()) {
			return false
		}

		iThreat := threat.Entity()

		if !engine.IsLineOfFireClearEntity(client, engine.EyePosition(client), iThreat) {
			return false
		}

		weaponID := engine.WeaponID(activeWeapon)

		if weaponID == engine.WeaponFlamethrower() && bot.IsRangeGreaterThanEntity(iThreat, engine.FlamethrowerReachRange()) {
			return false
		}

		if weaponID == engine.WeaponFlameBall() && bot.IsRangeGreaterThanEntity(iThreat, engine.FlameballReachRange()) {
			return false
		}

		if engine.IsMeleeWeapon(activeWeapon) && bot.IsRangeGreaterThanEntity(iThreat, 100.0) {
			return false
		}

		if engine.IsPlayer(iThreat) {
			/* A giant with a lot of health is probably a boss, and a boss
			near a failing wave wants killing fast. This wants doing better
			by somebody who knows what the optimal use of this canteen is. */
			if (engine.IsMiniBoss(iThreat) && engine.ClientHealth(iThreat) > 5000) || (engine.IsFailureImminent(client) && engine.ClientHealth(iThreat) > 2000) {
				engine.UseActionSlot(client)
				return true
			}
		} else if engine.IsBaseBoss(iThreat) && engine.EntityHealth(iThreat) > 1000 {
			// Crit against the tank.
			engine.UseActionSlot(client)
			return true
		}
	case engine.BottleUberCharge():
		// I'm invincible already.
		if engine.IsInvulnerable(client) {
			return false
		}

		// Only when there's a threat nearby, otherwise we could just go heal
		// ourselves.
		if threat == engine.NoKnownEntity() || !threat.VisibleRecently() {
			return false
		}

		healthRatio := float32(engine.ClientHealth(client)) / float32(engine.PlayerMaxHealth(client))

		if healthRatio < engine.HealthCriticalRatio().Float() {
			// I'm about to die.
			engine.UseActionSlot(client)
			nextBottleUseTime[client] = engine.GameTime() + engine.RandomFloat(10.0, 30.0)
			return true
		}

		if engine.IsPlayerInCondition(client, engine.ConditionGas()) {
			// This gas might be explosive.
			engine.UseActionSlot(client)
			nextBottleUseTime[client] = engine.GameTime() + engine.RandomFloat(20.0, 30.0)
			return true
		}
	case engine.BottleRecall():
		// The medic can't share this, and the engineer should probably only
		// use it if his sentry was destroyed; neither is written yet.
		if engine.PlayerClass(client) == engine.ClassMedic() {
			return false
		}

		if engine.PlayerClass(client) == engine.ClassEngineer() {
			return false
		}

		// We're busy going for the tank.
		if engine.LookupEntityActionByName(client, "DefenderAttackTank") != engine.InvalidAction() {
			return false
		}

		myPosition := engine.WorldSpaceCenter(client)

		// I'm already in my spawn room.
		if engine.IsPointInRespawnRoomStrict(myPosition, client) {
			return false
		}

		hatchPosition := engine.BombHatchPosition()

		// We're already close enough to the hatch.
		if engine.VectorDistance(myPosition, hatchPosition) <= 1000.0 {
			return false
		}

		flag := engine.BombNearestToHatch()

		// No bomb active.
		if flag == -1 {
			return false
		}

		bombPosition := engine.WorldSpaceCenter(flag)

		// Bomb is far and not a threat.
		if engine.VectorDistance(bombPosition, hatchPosition) > engine.BombHatchRangeCritical() {
			return false
		}

		closestToHatch := engine.BotNearestToBombNearestToHatch(client)

		// No robot near the bomb close to the hatch.
		if closestToHatch == -1 {
			return false
		}

		threatPosition := engine.Origin(closestToHatch)

		// Nearest robot isn't that close to the bomb.
		if engine.VectorDistance(threatPosition, bombPosition) > 800.0 {
			return false
		}

		// We are already close enough to deal with it.
		if engine.VectorDistance(myPosition, threatPosition) <= 500.0 {
			return false
		}

		engine.UseActionSlot(client)
		return true
	case engine.BottleRefillAmmo():
		primary := engine.PlayerWeaponSlot(client, engine.WeaponSlotPrimary())

		if primary != -1 && !engine.HasAmmo(primary) {
			// I got no ammo.
			engine.UseActionSlot(client)
			return true
		}
	case engine.BottleBuildingsInstantUpgrade():
		// TODO
	}

	return false
}
