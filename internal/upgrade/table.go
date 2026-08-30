package upgrade

import "github.com/m-this/tf2-mvm-bots-go/gen/go/attr"

/*
What a bot buys at the upgrade station, as a table.

The shipped ranking is three lookups written as ninety-four string comparisons,
and a comparison chain is a table that cannot be read, sorted or tested. It is
one here: the attribute is an id by the time it arrives, so each layer is a list
of (attribute, score) and the lookup is the same three questions in the same
order.

The order is the whole of the semantics and it does not change: the weapon that
is the reason to carry the loadout, then what the class contributes with, then
the general table, and an upgrade nothing recognises is ranked at random the way
the mod always did.

The scores are the shipped ones, and upstream_test.go re-reads them out of
upgrade.sp at the pinned revision and fails if a single one differs.
*/

// Score is what one upgrade is worth to one bot. Higher is bought first.
type Score int32

/*
When is what a score depends on beyond the attribute itself.

Three of the rules are not constants: the disposable sentry is behind a feature
switch, and the three resistances are worth something only when the coming wave
deals that damage. Everything else is Always, and the emitter writes a plain
number for it.
*/
type When int32

// The conditions a rule can carry.
const (
	// Always is a plain score, which is all but four of the rules.
	Always When = iota
	// EngineerDisposable is FEATURE_ENGINEER_DISPOSABLE: on, the mini is
	// worth buying; off, it is refused outright.
	EngineerDisposable
	// WaveBlast, WaveBullet and WaveFire are the three resistances, worth
	// having when the wave bar says that damage is coming.
	WaveBlast
	WaveBullet
	WaveFire
)

// Rule is one attribute and what it is worth.
type Rule struct {
	Attr  attr.Attribute
	Score Score
	// When is Always unless the score depends on something the caller has
	// to answer.
	When When
	// Otherwise is the score when the condition does not hold, and is read
	// only when When is not Always.
	Otherwise Score
}

/*
Loadout is the upgrade that is the reason to carry this exact weapon, by item
definition index.

Only the few where the loadout rather than the class decides what to buy first.
Everything else falls through to the class table below.
*/
var Loadout = map[int32][]Rule{
	// Kritzkrieg: the crits are the weapon, so the meter is what matters
	35: {
		{Attr: attr.AttrUberchargeRateBonus, Score: 330},
	},
	// Quick-Fix: it heals rather than saves, so it should heal faster
	411: {
		{Attr: attr.AttrHealingMastery, Score: 330},
		{Attr: attr.AttrUberchargeRateBonus, Score: 300},
	},
	// Brass Beast: the damage minigun, and it cannot reposition to make up for less
	312: {
		{Attr: attr.AttrDamageBonus, Score: 320},
	},
	// Tomislav: it already fires fast, so damage per bullet beats more bullets
	424: {
		{Attr: attr.AttrDamageBonus, Score: 300},
	},
	// Hitman's Heatmaker: reach the shot sooner
	752: {
		{Attr: attr.AttrSRifleChargeRateIncreased, Score: 300},
	},
	// Machina: every shot is a charged one, so damage rides on all of them
	526: {
		{Attr: attr.AttrDamageBonus, Score: 300},
	},
	// Loose Cannon: a faster cannonball is one a bot can actually land
	996: {
		{Attr: attr.AttrProjectileSpeedIncreased, Score: 300},
	},
	// Rescue Ranger: every shot and every repair at range costs metal
	997: {
		{Attr: attr.AttrMetalRegen, Score: 300},
		{Attr: attr.AttrMaxammoMetalIncreased, Score: 290},
	},
	// Beggar's Bazooka: it fires as fast as the button is pressed, so buying that twice is buying nothing
	730: {
		{Attr: attr.AttrFireRateBonus, Score: 20},
		{Attr: attr.AttrClipSizeUpgradeAtomic, Score: 280},
		{Attr: attr.AttrClipSizeBonusUpgrade, Score: 280},
	},
	// Widowmaker: the shot is paid for in metal and paid back in damage dealt
	527: {
		{Attr: attr.AttrDamageBonus, Score: 300},
		{Attr: attr.AttrFireRateBonus, Score: 250},
	},
	// Short Circuit: it eats projectiles rather than robots, and eats metal doing it
	528: {
		{Attr: attr.AttrMetalRegen, Score: 300},
	},
	// Frontier Justice: the crits are banked by the sentry, so the clip is what holds them
	141: {
		{Attr: attr.AttrClipSizeUpgradeAtomic, Score: 260},
		{Attr: attr.AttrClipSizeBonusUpgrade, Score: 260},
		{Attr: attr.AttrDamageBonus, Score: 250},
	},
}

/*
EngineerMetal is the engineer whose gun is paid for in metal.

The metal upgrades do not hang off the gun, so the loadout table cannot answer
for them however the loadout is put together. A Widowmaker engineer without them
fights the wave out of the same supply the sentry is repaired from, and runs out
of both. Under the sentry's own fire rate, above everything else the class buys.
*/
var EngineerMetal = []Rule{
	{Attr: attr.AttrMaxammoMetalIncreased, Score: 310},
	{Attr: attr.AttrMetalRegen, Score: 305},
}

/*
Class is what this class contributes with, which is not always the weapon in its
hands.

Two of these are restricted to a slot, and the restriction is part of the answer
rather than a detail of it: the engineer's gun rules apply to the gun slots and
the sentry rules to everything else, and the spy's knife rules only to the knife.
*/
type ClassRules struct {
	// GunSlots says the rules above the split apply to the primary and
	// secondary slots only, and the ones below to any other slot.
	Split bool
	// Gun are the rules for the primary and secondary slots.
	Gun []Rule
	// GunFallthrough is what anything else on the gun is worth: less than
	// the cheapest thing the nest wants.
	GunFallthrough Score
	// Rest are the rules for every other slot, or for every slot when
	// Split is false.
	Rest []Rule
	// MeleeOnly says Rest applies to the melee slot and nothing else, which
	// is the spy's knife.
	MeleeOnly bool
}

// Class is the per-class table, in the shipped order.
var Class = map[Klass]ClassRules{
	KlassEngineer: {
		Split: true,
		Gun: []Rule{
			{Attr: attr.AttrDamageBonus, Score: 200},
			{Attr: attr.AttrFireRateBonus, Score: 190},
			{Attr: attr.AttrClipSizeUpgradeAtomic, Score: 150},
			{Attr: attr.AttrClipSizeBonusUpgrade, Score: 150},
			{Attr: attr.AttrFasterReloadRate, Score: 140},
			{Attr: attr.AttrMaxammoPrimaryIncreased, Score: 130},
			{Attr: attr.AttrMaxammoSecondaryIncreased, Score: 120},
		},
		GunFallthrough: 50,
		Rest: []Rule{
			{Attr: attr.AttrEngyDispenserRadiusIncreased, Score: 330},
			{Attr: attr.AttrEngySentryFireRateIncreased, Score: 320},
			{Attr: attr.AttrEngyBuildingHealthBonus, Score: 260},
			{Attr: attr.AttrMetalRegen, Score: 220},
			{Attr: attr.AttrMaxammoMetalIncreased, Score: 210},
			{Attr: attr.AttrMeleeAttackRateBonus, Score: 200},
			{Attr: attr.AttrEngyDisposableSentries, Score: 310, When: EngineerDisposable, Otherwise: -10},
		},
	},
	KlassMedic: {
		Rest: []Rule{
			{Attr: attr.AttrGenerateRageOnHeal, Score: 320},
			{Attr: attr.AttrUberchargeRateBonus, Score: 300},
			{Attr: attr.AttrHealingMastery, Score: 280},
			{Attr: attr.AttrUberDurationBonus, Score: 230},
			{Attr: attr.AttrOverhealExpert, Score: 210},
			{Attr: attr.AttrDamageBonus, Score: 40},
			{Attr: attr.AttrFireRateBonus, Score: 40},
		},
	},
	KlassSniper: {
		Rest: []Rule{
			{Attr: attr.AttrExplosiveSniperShot, Score: 330},
			{Attr: attr.AttrFasterReloadRate, Score: 300},
			{Attr: attr.AttrSRifleChargeRateIncreased, Score: 60},
		},
	},
	KlassSpy: {
		MeleeOnly: true,
		Rest: []Rule{
			{Attr: attr.AttrArmorPiercing, Score: 330},
			{Attr: attr.AttrMeleeAttackRateBonus, Score: 280},
			{Attr: attr.AttrRoboSapper, Score: 70},
		},
	},
	KlassPyro: {
		Rest: []Rule{
			{Attr: attr.AttrDamageBonus, Score: 320},
			{Attr: attr.AttrAttackProjectiles, Score: 250},
		},
	},
	KlassSoldier: {
		Rest: []Rule{
			{Attr: attr.AttrFasterReloadRate, Score: 310},
			{Attr: attr.AttrRocketSpecialist, Score: 290},
			{Attr: attr.AttrHealOnKill, Score: 250},
		},
	},
	KlassDemoMan: {
		Rest: []Rule{
			{Attr: attr.AttrFasterReloadRate, Score: 310},
			{Attr: attr.AttrFireRateBonus, Score: 290},
			{Attr: attr.AttrProjectileSpeedIncreased, Score: 200},
		},
	},
	KlassHeavy: {
		Rest: []Rule{
			{Attr: attr.AttrHealOnKill, Score: 320},
			{Attr: attr.AttrAttackProjectiles, Score: 230},
		},
	},
	KlassScout: {
		Rest: []Rule{
			{Attr: attr.AttrAppliesSnareEffect, Score: 250},
			{Attr: attr.AttrMadMilkSyringes, Score: 200},
			{Attr: attr.AttrMoveSpeedBonus, Score: 190},
		},
	},
}

/*
General is damage first, then what keeps it firing: what a bot buys when nothing
above had an opinion.

The three resistances sit in the middle of it and are the only rules here that
are not constants. A resistance is worth a middling amount when the robots that
deal that damage are in the wave, and very little when they are not: blast
resistance against a wave of Scouts is three hundred credits spent on nothing.
*/
var General = []Rule{
	{Attr: attr.AttrDamageBonus, Score: 260},
	{Attr: attr.AttrFireRateBonus, Score: 250},
	{Attr: attr.AttrMeleeAttackRateBonus, Score: 200},
	{Attr: attr.AttrProjectilePenetration, Score: 190},
	{Attr: attr.AttrProjectilePenetrationHeavy, Score: 190},
	{Attr: attr.AttrCritboostOnKill, Score: 180},
	{Attr: attr.AttrClipSizeUpgradeAtomic, Score: 170},
	{Attr: attr.AttrClipSizeBonusUpgrade, Score: 170},
	{Attr: attr.AttrFasterReloadRate, Score: 160},
	{Attr: attr.AttrMaxammoPrimaryIncreased, Score: 150},
	{Attr: attr.AttrProjectileSpeedIncreased, Score: 130},
	{Attr: attr.AttrMaxammoSecondaryIncreased, Score: 120},
	{Attr: attr.AttrHealOnKill, Score: 110},
	{Attr: attr.AttrMarkForDeath, Score: 90},
	{Attr: attr.AttrArmorPiercing, Score: 85},
	{Attr: attr.AttrAttackProjectiles, Score: 80},
	{Attr: attr.AttrIncreaseBuffDuration, Score: 75},
	{Attr: attr.AttrEffectBarRechargeRateIncreased, Score: 70},
	{Attr: attr.AttrChargeRechargeRateIncreased, Score: 70},
	{Attr: attr.AttrGenerateRageOnDamage, Score: 60},
	{Attr: attr.AttrBleedingDuration, Score: 55},
	{Attr: attr.AttrMoveSpeedBonus, Score: 45},
	{Attr: attr.AttrHealthRegen, Score: 40},
	{Attr: attr.AttrDmgTakenFromCritReduced, Score: 30},
	{Attr: attr.AttrDamageForceReduction, Score: 25},
	{Attr: attr.AttrIncreasedJumpHeight, Score: 10},
	{Attr: attr.AttrDmgTakenFromBlastReduced, Score: 210, When: WaveBlast, Otherwise: 25},
	{Attr: attr.AttrDmgTakenFromBulletsReduced, Score: 210, When: WaveBullet, Otherwise: 25},
	{Attr: attr.AttrDmgTakenFromFireReduced, Score: 210, When: WaveFire, Otherwise: 25},
}

// Klass is the class the ranking asks about, as the id the SourcePawn side
// already uses.
type Klass int32

// The nine, in the game's own order.
const (
	KlassUnknown Klass = iota
	KlassScout
	KlassSniper
	KlassSoldier
	KlassDemoMan
	KlassMedic
	KlassHeavy
	KlassPyro
	KlassSpy
	KlassEngineer
)

// Slot is a loadout slot, as the ranking sees it.
type Slot int32

// The slots the ranking distinguishes. Action is the canteen and Player is the
// slot the game hangs everything not attached to a weapon off.
const (
	SlotPlayer  Slot = -1
	SlotPrimary Slot = iota - 1
	SlotSecondary
	SlotMelee
	SlotUtility
	SlotBuilding
	SlotPDA
	SlotPDA2
	SlotAction
)

// Wave is what the coming wave deals, which is what a resistance is priced
// against.
type Wave struct {
	Blast, Bullet, Fire bool
}

// Bot is everything about one shopper the ranking reads.
type Bot struct {
	Klass Klass
	// ItemDef is the item definition index of the weapon in the slot being
	// ranked, or 0 for a slot with no weapon in it.
	ItemDef int32
	// GunSpendsMetal is an engineer carrying a gun that costs metal to
	// fire, which is the Widowmaker, the Short Circuit and the Rescue
	// Ranger.
	GunSpendsMetal bool
	// Disposable is FEATURE_ENGINEER_DISPOSABLE.
	Disposable bool
	Wave       Wave
}

// scoreOf reads one rule, answering the condition the caller carries.
func (b Bot) scoreOf(r Rule) Score {
	switch r.When {
	case Always:
		return r.Score
	case EngineerDisposable:
		if b.Disposable {
			return r.Score
		}
	case WaveBlast:
		if b.Wave.Blast {
			return r.Score
		}
	case WaveBullet:
		if b.Wave.Bullet {
			return r.Score
		}
	case WaveFire:
		if b.Wave.Fire {
			return r.Score
		}
	}
	return r.Otherwise
}

func (b Bot) lookup(rules []Rule, want attr.Attribute) (Score, bool) {
	for _, r := range rules {
		if r.Attr == want {
			return b.scoreOf(r), true
		}
	}
	return 0, false
}

/*
Priority is what this upgrade is worth to this bot, and false when nothing here
has an opinion: the caller ranks those at random, which is what the mod did with
every upgrade before this table existed.

The three layers in the shipped order. A layer that answers wins, and a layer
that answers zero is a layer with no opinion rather than one saying the upgrade
is worthless, which is why the shipped code tests `> 0` between them.
*/
func (b Bot) Priority(slot Slot, a attr.Attribute) (Score, bool) {
	// The metal upgrades do not hang off the gun, so they are asked before
	// the slot is looked at at all.
	if b.Klass == KlassEngineer && b.GunSpendsMetal {
		if score, ok := b.lookup(EngineerMetal, a); ok && score > 0 {
			return score, true
		}
	}

	if slot >= SlotPrimary && slot <= SlotMelee {
		if score, ok := b.lookup(Loadout[b.ItemDef], a); ok && score > 0 {
			return score, true
		}
	}

	if score, ok := b.classPriority(slot, a); ok && score > 0 {
		return score, true
	}

	return b.lookup(General, a)
}

// classPriority is the middle layer, including the two slot restrictions that
// are part of the answer rather than a detail of it.
func (b Bot) classPriority(slot Slot, a attr.Attribute) (Score, bool) {
	rules, known := Class[b.Klass]
	if !known {
		return 0, false
	}

	if rules.Split && (slot == SlotPrimary || slot == SlotSecondary) {
		if score, ok := b.lookup(rules.Gun, a); ok {
			return score, true
		}
		// Anything else on the gun is worth less than the cheapest thing
		// the nest wants.
		return rules.GunFallthrough, true
	}

	if rules.MeleeOnly && slot != SlotMelee {
		return 0, false
	}

	return b.lookup(rules.Rest, a)
}
