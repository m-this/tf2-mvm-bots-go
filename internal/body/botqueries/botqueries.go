/*
Package botqueries is the query layer of source/redbots3/nextbot_behavior.sp:
the questions a behaviour asks about a bot that need no state of their own.

Every one of these was a plugin extern somewhere; each port here deletes one.
*/
package botqueries

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// GetDesiredPathLookAheadRange is how far along the path a bot of that size
// aims.
//
//sp:name GetDesiredPathLookAheadRange
func GetDesiredPathLookAheadRange(client int32) float32 {
	return engine.PathLookaheadRange().Float() * engine.ModelScale(client)
}

// IsAmmoLow says the bot is worth sending to a resupply.
//
//sp:name IsAmmoLow
func IsAmmoLow(client int32) bool {
	primary := engine.PlayerWeaponSlot(client, engine.WeaponSlotPrimary())

	if engine.IsValidEntity(primary) && !engine.HasAmmo(primary) {
		return true
	}

	myWeapon := engine.ActiveWeapon(client)

	if myWeapon != -1 && engine.WeaponID(myWeapon) != engine.WeaponWrench() {
		if !engine.IsMeleeWeapon(myWeapon) {
			flAmmoRation := float32(engine.AmmoCount(client, engine.AmmoPrimary())) / float32(engine.PlayerMaxAmmo(client, engine.AmmoPrimary()))
			return flAmmoRation < 0.2
		}

		return false
	}

	return engine.AmmoCount(client, engine.AmmoMetal()) <= 0
}

// IsAmmoFull says a resupply has nothing left to give.
//
//sp:name IsAmmoFull
func IsAmmoFull(client int32) bool {
	isPrimaryFull := engine.AmmoCount(client, engine.AmmoPrimary()) >= engine.PlayerMaxAmmo(client, engine.AmmoPrimary())
	isSecondaryFull := engine.AmmoCount(client, engine.AmmoSecondary()) >= engine.PlayerMaxAmmo(client, engine.AmmoSecondary())

	if engine.PlayerClass(client) == engine.ClassEngineer() {
		// In addition, I want some metal as well.
		return engine.AmmoCount(client, engine.AmmoMetal()) >= 200 && isPrimaryFull && isSecondaryFull
	}

	return isPrimaryFull && isSecondaryFull
}

// ResetIntentionInterface makes the bot decide again from the top.
//
//sp:name ResetIntentionInterface
func ResetIntentionInterface(botEntidx int32) {
	engine.NextBotOf(botEntidx).Intention().Reset()
}

// UpdateLookAroundForEnemies turns the bot's own looking on or off, so a
// behaviour that aims for itself is not fought by the game.
//
//sp:name UpdateLookAroundForEnemies
func UpdateLookAroundForEnemies(client int32, bVal bool) {
	engine.SetLookingAroundForEnemies(client, bVal)
}

// IsCombatWeapon says the thing in hand can hurt somebody.
//
//sp:name IsCombatWeapon
func IsCombatWeapon(client int32, weapon int32) bool {
	if !engine.IsValidEntity(weapon) {
		weapon = engine.ActiveWeapon(client)
	}

	if engine.IsValidEntity(weapon) {
		switch engine.WeaponID(weapon) {
		case engine.WeaponMedigun(), engine.WeaponPDA(), engine.WeaponPDAEngineerBuild(), engine.WeaponPDAEngineerDestroy(), engine.WeaponPDASpy(), engine.WeaponBuilder(), engine.WeaponDispenser(), engine.WeaponInvis(), engine.WeaponLunchbox(), engine.WeaponBuffItem(), engine.WeaponPumpkinBomb():
			return false
		}
	}

	return true
}

/*
GetDesiredAttackRange is the distance the bot closes to before it settles.

The Pyro closes whatever is in his hands, because the flamethrower is the only
reason he is here. The weapon is chosen by range and the range he closes to is
chosen by the weapon, and letting those two answer separately parked him between
the two distances holding the wrong gun.

The rocket's twelve fifty is how far out a rocket is worth firing, which is not
as far as it will travel: everything a defender shoots at is walking, and past
that range it has left the splash before the rocket arrives.
*/
//
//sp:name GetDesiredAttackRange
func GetDesiredAttackRange(client int32) float32 {
	weapon := engine.ActiveWeapon(client)

	if weapon < 1 {
		return 0.0
	}

	// The loadout the server handed out is more specific than the weapon's ID.
	found, tunedDesired, tunedMax := engine.TunedWeaponRanges(weapon)
	_ = tunedMax

	if found {
		return tunedDesired
	}

	weaponID := engine.WeaponID(weapon)

	if weaponID == engine.WeaponKnife() {
		return 70.0
	}

	if engine.IsMeleeWeapon(weapon) || weaponID == engine.WeaponFlamethrower() {
		return 100.0
	}

	if engine.PlayerClass(client) == engine.ClassPyro() {
		flamethrower := engine.PlayerWeaponSlot(client, engine.WeaponSlotPrimary())

		if flamethrower != -1 && engine.WeaponID(flamethrower) == engine.WeaponFlamethrower() {
			return 100.0
		}
	}

	if engine.WeaponIDIsSniperRifle(weaponID) {
		return engine.FloatMax()
	}

	if weaponID == engine.WeaponRocketLauncher() {
		if engine.Feature(engine.FeatureSoldierClosesIn()) {
			return engine.SoldierRocketSettle()
		}

		return 1250.0
	}

	// The same answer as the Iron Bomber, which is the launcher this loadout
	// actually hands out.
	if weaponID == engine.WeaponGrenadeLauncher() {
		return engine.DemoPipeSettle()
	}

	return 500.0
}

// ShouldBuybackIntoGame is the buyback decision, rolled once per death.
//
//sp:name ShouldBuybackIntoGame
func ShouldBuybackIntoGame(client int32) bool {
	// Scouts respawn very quickly.
	if engine.PlayerClass(client) == engine.ClassScout() {
		return false
	}

	// Can't afford a buyback.
	if engine.Currency(client) < engine.BuybackCostPerSecond() {
		return false
	}

	// Not opportunistic if we're about to fail.
	if IsFailureImminent(client) {
		return true
	}

	// We're being revived.
	if engine.BeingRevived(client) {
		return false
	}

	// Based on our rolled number, decide to buyback.
	return engine.BuybackNumber(client) <= engine.BuybackChance().Int()
}

// ShouldUpgradeMidRound says the bot spawned into a wave and should shop first.
//
//sp:name ShouldUpgradeMidRound
func ShouldUpgradeMidRound(client int32) bool {
	// If we were revived, we should not bother.
	if !engine.IsPointInRespawnRoom(engine.WorldSpaceCenter(client)) {
		return false
	}

	// Based on our rolled number from spawn, decide to buy upgrades now.
	return engine.BuyUpgradesNumber(client) > 0 && engine.BuyUpgradesNumber(client) <= engine.BuyUpgradesChance().Int()
}

// CanBuyUpgradesNow says shopping is affordable and not suicidal.
//
//sp:name CanBuyUpgradesNow
func CanBuyUpgradesNow(client int32) bool {
	if engine.Currency(client) < 25 {
		return false
	}

	if IsFailureImminent(client) {
		return false
	}

	return true
}

// TransientlyConsistentRandomValue is the game's own trick: a number that is
// random across bots and stable for a period, so a decision does not flicker.
//
//sp:name TransientlyConsistentRandomValue
//sp:default period 10.0
//sp:default seedValue 0
func TransientlyConsistentRandomValue(client int32, period float32, seedValue int32) float32 {
	area := engine.CombatOf(client).LastKnownArea()

	if area == engine.NoNavArea() {
		return 0.0
	}

	timeMod := engine.RoundToFloor(engine.GameTime()/period) + 1

	return engine.FloatAbs(engine.Cosine(float32(seedValue + (client * area.ID() * timeMod))))
}

// IsFailureImminent says a robot is about to pick up a bomb next to the hatch.
//
//sp:name IsFailureImminent
func IsFailureImminent(client int32) bool {
	// TODO: factor in tank closest to hatch for certain classes

	flag := engine.BombNearestToHatch()

	if flag == -1 {
		return false
	}

	bombPosition := engine.WorldSpaceCenter(flag)

	// Bomb is far and not a threat.
	if engine.VectorDistance(bombPosition, engine.BombHatchPosition()) > engine.BombHatchRangeCritical() {
		return false
	}

	closestToHatch := engine.BotNearestToBombNearestToHatch(client)

	// No robot near the bomb close to the hatch, we're probably okay for now.
	if closestToHatch == -1 {
		return false
	}

	threatOrigin := engine.Origin(closestToHatch)

	// Robot about to pick up a bomb very close to the hatch, we're in danger!
	return engine.VectorDistance(threatOrigin, bombPosition) <= 800.0
}

// GetFlameThrowerAimForTank aims a bit high: since the March 28 2018 update
// flamethrower damage is calculated on the oldest particles.
//
//sp:name GetFlameThrowerAimForTank
func GetFlameThrowerAimForTank(tank int32) (aimPos [3]float32) {
	aimPos = engine.WorldSpaceCenter(tank)
	aimPos[2] += 90.0
	return aimPos
}

/*
TeleporterWorthRiding is how much walking a teleporter has to save before it is
worth walking to.

Saying yes puts the bot on a walk to the entrance, which is back in the spawn he
is trying to leave: the walk it saves has to beat the walk it costs.
*/
//
//sp:name TELEPORTER_WORTH_RIDING
const TeleporterWorthRiding = 1500.0

// ShouldUseTeleporter says the ride beats the walk by enough to bother.
//
//sp:name ShouldUseTeleporter
func ShouldUseTeleporter(client int32) bool {
	// No bomb in play, so there is no fight to be late for and no reason to
	// leave the ground.
	found, bombinfo := engine.GetBombInfo()

	if !found {
		return false
	}

	myArea := engine.CombatOf(client).LastKnownArea()

	if myArea == engine.NoNavArea() {
		return false
	}

	bombArea := engine.NearestNavArea(bombinfo.Position, false, 10000.0, false, false, engine.TeamAny())

	if bombArea == engine.NullArea() {
		return false
	}

	return engine.TravelDistanceToBombTarget(engine.Area(myArea))+TeleporterWorthRiding < engine.TravelDistanceToBombTarget(bombArea)
}

// GetCountOfBotsWithNamedAction is how many defenders are doing that right now.
//
//sp:name GetCountOfBotsWithNamedAction
//sp:default ignore -1
func GetCountOfBotsWithNamedAction(name string, ignore int32) int32 {
	count := int32(0)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if i != ignore && engine.IsClientInGame(i) && engine.DefenderBotFlag(i) && engine.LookupEntityActionByName(i, name) != engine.InvalidAction() {
			count++
		}
	}

	return count
}

// HealerOrThreat swaps a player threat for whoever is healing it, when the bot
// can see the healer.
//
//sp:name HealerOrThreat
//sp:const threat
func HealerOrThreat(bot engine.Bot, threat engine.Known) engine.Known {
	if threat == engine.NoKnownEntity() || !engine.IsPlayer(threat.Entity()) {
		return threat
	}

	return GetHealerOfThreat(bot, threat)
}

// GetHealerOfThreat is the first visible medic on the threat, or the threat.
//
//sp:name GetHealerOfThreat
//sp:const threat
func GetHealerOfThreat(bot engine.Bot, threat engine.Known) engine.Known {
	if threat == engine.NoKnownEntity() {
		return engine.NoKnownEntity()
	}

	playerThreat := threat.Entity()

	for i := int32(0); i < engine.NumHealers(playerThreat); i++ {
		playerHealer := engine.PlayerHealer(playerThreat, i)

		if playerHealer != -1 && engine.IsPlayer(playerHealer) {
			knownHealer := bot.Vision().GetKnown(playerHealer)

			if knownHealer != engine.NoKnownEntity() && knownHealer.VisibleInFOVNow() {
				return knownHealer
			}
		}
	}

	return threat
}

// SelectCloserThreat is whichever of the two the bot could touch first.
//
//sp:name SelectCloserThreat
//sp:const threat1
//sp:const threat2
func SelectCloserThreat(bot engine.Bot, threat1 engine.Known, threat2 engine.Known) engine.Known {
	rangeSq1 := bot.RangeSquaredTo(threat1.Entity())
	rangeSq2 := bot.RangeSquaredTo(threat2.Entity())

	if rangeSq1 < rangeSq2 {
		return threat1
	}

	return threat2
}

/*
OpportunisticallyUseWeaponAbilities fires the weapon's own gimmick when the
moment fits: the Heatmaker's focus while scoped on a visible threat, the
Phlogistinator's Mmmph in reach, and the minigun's rage on a flag carrier at
the hatch.
*/
//
//sp:name OpportunisticallyUseWeaponAbilities
//sp:const threat
func OpportunisticallyUseWeaponAbilities(client int32, activeWeapon int32, bot engine.Bot, threat engine.Known) bool {
	if threat == engine.NoKnownEntity() {
		return false
	}

	if activeWeapon == -1 {
		return false
	}

	weaponID := engine.WeaponID(activeWeapon)

	// Hitmans Heatmaker.
	if weaponID == engine.WeaponSniperrifle() && engine.IsPlayerInCondition(client, engine.ConditionSlowed()) && threat.VisibleRecently() {
		if engine.RageMeter(client) >= 0.0 && !engine.IsRageDraining(client) {
			engine.ExtraButtonsOf(client).PressButtonsNow(engine.InReload())
			return true
		}
	}

	iThreat := threat.Entity()

	// Phlogistinator.
	if weaponID == engine.WeaponFlamethrower() && bot.IsRangeLessThan(iThreat, engine.FlamethrowerReachRange()) && !engine.IsCritBoosted(client) {
		if engine.RageMeter(client) >= 100.0 && !engine.IsRageDraining(client) {
			engine.PressAltFireButton(client)
			return true
		}
	}

	if weaponID == engine.WeaponMinigun() && engine.IsPlayer(iThreat) && engine.RageMeter(client) >= 100.0 {
		if engine.HasTheFlag(iThreat) {
			vThreatOrigin := engine.Origin(iThreat)

			if engine.VectorDistance(vThreatOrigin, engine.BombHatchPosition()) <= 100.0 {
				engine.PressSpecialFireButton(client)
				return true
			}
		}
	}

	return false
}

/*
The entity table's size, asked once. SourcePawn keeps it in a function-local
static; one package-level cell is the same single initialisation.
*/
//
//sp:name s_iMaxEntCount
var maxEntCount int32 = -1

/*
MonitorKnownEntities widens the game's own vision with a plain line-of-sight
check.

IVision::UpdateKnownEntities only collects entities in the bot's FOV, so a known
entity that leaves it goes obsolete after ten seconds and is dropped. This keeps
everything the bot could actually see on the list.
*/
//
//sp:name MonitorKnownEntities
func MonitorKnownEntities(client int32, vision engine.Vision) {
	if engine.NbBlind().Bool() {
		return
	}

	if maxEntCount == -1 {
		maxEntCount = engine.MaxEntities()
	}

	myTeam := engine.GetClientTeam(client)

	for i := int32(1); i <= maxEntCount; i++ {
		if !engine.IsValidEntity(i) {
			continue
		}

		if i == client {
			continue
		}

		if engine.IsPlayer(i) && !engine.IsPlayerAlive(i) {
			continue
		}

		if !engine.EntityOf(i).IsCombatCharacter() {
			continue
		}

		if engine.EntityTeamNumber(i) == myTeam {
			continue
		}

		if engine.IsLineOfFireClearEntity(client, engine.EyePosition(client), i) {
			known := vision.GetKnown(i)

			if known != engine.NoKnownEntity() {
				// We already know about this entity and we can currently
				// see it.
				known.UpdatePosition()
			} else {
				// We didn't know about it but we can see it now,
				// recognize it.
				vision.AddKnownEntity(i)
			}
		}
	}
}
