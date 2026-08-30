package upgrade

import "github.com/m-this/tf2-mvm-bots-go/internal/tables"

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
	Attr  tables.Attribute
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
		rule("ubercharge rate bonus", 330),
	},
	// Quick-Fix: it heals rather than saves, so it should heal faster
	411: {
		rule("healing mastery", 330),
		rule("ubercharge rate bonus", 300),
	},
	// Brass Beast: the damage minigun, and it cannot reposition to make up for less
	312: {
		rule("damage bonus", 320),
	},
	// Tomislav: it already fires fast, so damage per bullet beats more bullets
	424: {
		rule("damage bonus", 300),
	},
	// Hitman's Heatmaker: reach the shot sooner
	752: {
		rule("SRifle Charge rate increased", 300),
	},
	// Machina: every shot is a charged one, so damage rides on all of them
	526: {
		rule("damage bonus", 300),
	},
	// Loose Cannon: a faster cannonball is one a bot can actually land
	996: {
		rule("Projectile speed increased", 300),
	},
	// Rescue Ranger: every shot and every repair at range costs metal
	997: {
		rule("metal regen", 300),
		rule("maxammo metal increased", 290),
	},
	// Beggar's Bazooka: it fires as fast as the button is pressed, so buying that twice is buying nothing
	730: {
		rule("fire rate bonus", 20),
		rule("clip size upgrade atomic", 280),
		rule("clip size bonus upgrade", 280),
	},
	// Widowmaker: the shot is paid for in metal and paid back in damage dealt
	527: {
		rule("damage bonus", 300),
		rule("fire rate bonus", 250),
	},
	// Short Circuit: it eats projectiles rather than robots, and eats metal doing it
	528: {
		rule("metal regen", 300),
	},
	// Frontier Justice: the crits are banked by the sentry, so the clip is what holds them
	141: {
		rule("clip size upgrade atomic", 260),
		rule("clip size bonus upgrade", 260),
		rule("damage bonus", 250),
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
	rule("maxammo metal increased", 310),
	rule("metal regen", 305),
}

/*
ClassRules is what one class contributes with, which is not always the weapon in
its hands.

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
			rule("damage bonus", 200),
			rule("fire rate bonus", 190),
			rule("clip size upgrade atomic", 150),
			rule("clip size bonus upgrade", 150),
			rule("faster reload rate", 140),
			rule("maxammo primary increased", 130),
			rule("maxammo secondary increased", 120),
		},
		GunFallthrough: 50,
		Rest: []Rule{
			rule("engy dispenser radius increased", 330),
			rule("engy sentry fire rate increased", 320),
			rule("engy building health bonus", 260),
			rule("metal regen", 220),
			rule("maxammo metal increased", 210),
			rule("melee attack rate bonus", 200),
			conditional("engy disposable sentries", 310, EngineerDisposable, -10),
		},
	},
	KlassMedic: {
		Rest: []Rule{
			rule("generate rage on heal", 320),
			rule("ubercharge rate bonus", 300),
			rule("healing mastery", 280),
			rule("uber duration bonus", 230),
			rule("overheal expert", 210),
			rule("damage bonus", 40),
			rule("fire rate bonus", 40),
		},
	},
	KlassSniper: {
		Rest: []Rule{
			rule("explosive sniper shot", 330),
			rule("faster reload rate", 300),
			rule("SRifle Charge rate increased", 60),
		},
	},
	KlassSpy: {
		MeleeOnly: true,
		Rest: []Rule{
			rule("armor piercing", 330),
			rule("melee attack rate bonus", 280),
			rule("robo sapper", 70),
		},
	},
	KlassPyro: {
		Rest: []Rule{
			rule("damage bonus", 320),
			rule("attack projectiles", 250),
		},
	},
	KlassSoldier: {
		Rest: []Rule{
			rule("faster reload rate", 310),
			rule("rocket specialist", 290),
			rule("heal on kill", 250),
		},
	},
	KlassDemoMan: {
		Rest: []Rule{
			rule("faster reload rate", 310),
			rule("fire rate bonus", 290),
			rule("Projectile speed increased", 200),
		},
	},
	KlassHeavy: {
		Rest: []Rule{
			rule("heal on kill", 320),
			rule("attack projectiles", 230),
		},
	},
	KlassScout: {
		Rest: []Rule{
			rule("applies snare effect", 250),
			rule("mad milk syringes", 200),
			rule("move speed bonus", 190),
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
	rule("damage bonus", 260),
	rule("fire rate bonus", 250),
	rule("melee attack rate bonus", 200),
	rule("projectile penetration", 190),
	rule("projectile penetration heavy", 190),
	rule("critboost on kill", 180),
	rule("clip size upgrade atomic", 170),
	rule("clip size bonus upgrade", 170),
	rule("faster reload rate", 160),
	rule("maxammo primary increased", 150),
	rule("Projectile speed increased", 130),
	rule("maxammo secondary increased", 120),
	rule("heal on kill", 110),
	rule("mark for death", 90),
	rule("armor piercing", 85),
	rule("attack projectiles", 80),
	rule("increase buff duration", 75),
	rule("effect bar recharge rate increased", 70),
	rule("charge recharge rate increased", 70),
	rule("generate rage on damage", 60),
	rule("bleeding duration", 55),
	rule("move speed bonus", 45),
	rule("health regen", 40),
	rule("dmg taken from crit reduced", 30),
	rule("damage force reduction", 25),
	rule("increased jump height", 10),
	conditional("dmg taken from blast reduced", 210, WaveBlast, 25),
	conditional("dmg taken from bullets reduced", 210, WaveBullet, 25),
	conditional("dmg taken from fire reduced", 210, WaveFire, 25),
}

/*
rule is one plain score, by the schema name the game gives the attribute.

Written as a name rather than as a generated constant because the generator
itself reads this table: importing what cmd/gen emits would mean cmd/gen could
not be built without its own output. The name is checked at init, so a typo is a
panic on the first run rather than an upgrade that silently never ranks.
*/
func rule(name string, score Score) Rule {
	return Rule{Attr: mustAttr(name), Score: score}
}

// conditional is a rule whose score depends on something the caller answers.
func conditional(name string, score Score, when When, otherwise Score) Rule {
	return Rule{Attr: mustAttr(name), Score: score, When: when, Otherwise: otherwise}
}

func mustAttr(name string) tables.Attribute {
	for _, a := range tables.Attributes {
		if a.Name == name {
			return a
		}
	}
	panic("upgrade: the ranking names " + name + " and internal/tables does not")
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

func (b Bot) lookup(rules []Rule, want tables.Attribute) (Score, bool) {
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
func (b Bot) Priority(slot Slot, a tables.Attribute) (Score, bool) {
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
func (b Bot) classPriority(slot Slot, a tables.Attribute) (Score, bool) {
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
