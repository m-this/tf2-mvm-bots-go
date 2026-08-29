package actionsel

import "fmt"

// Predicate is one question the decision can ask about a bot.
//
// It is an identity rather than a struct field because three of the answers
// cost something: CTFBotAttackTank_SelectTarget writes m_iTankTarget,
// CTFBotCollectNearMoney_SelectTarget writes m_iCurrencyPack, and
// CTFBotDefenderAttack_SelectTarget consumes randomness. Asking one the shipped
// chain would not have asked is a behaviour change, so the decision asks by
// name, one at a time, and something else decides what answering costs.
type Predicate int

// Every question the decision can ask, in the order the ids run in: the id is
// what the table hands the plugin, so the order is part of the emitted table.
const (
	MoneyToCollect Predicate = iota
	InUpgradeZone
	ShoppedThisBreak
	MovingToFront
	UpgradesEnabled
	HasUpgraded
	UpgradeMidRound
	HasSniperRifle
	SniperStalled
	AttackTargetFound
	TankTargetFound
	GiantToMark
	NearbyMoney
	StickyTrapPossible
	numPredicates
)

// call is the SourcePawn each predicate stands for. The generated edge emits
// these verbatim, so a rename here is a rename in the plugin.
var call = map[Predicate]string{
	MoneyToCollect:     "CTFBotCollectMoney_IsPossible(client)",
	InUpgradeZone:      "TF2_IsInUpgradeZone(client)",
	ShoppedThisBreak:   "g_bShoppedThisBreak[client]",
	MovingToFront:      `ActionsManager.LookupEntityActionByName(client, "DefenderMoveToFront") != INVALID_ACTION`,
	UpgradesEnabled:    "redbots_manager_bot_use_upgrades.BoolValue",
	HasUpgraded:        "g_bHasUpgraded[client]",
	UpgradeMidRound:    "ShouldUpgradeMidRound(client)",
	HasSniperRifle:     "HasSniperRifle(client)",
	SniperStalled:      "IsSniperStalled(client)",
	AttackTargetFound:  "CTFBotDefenderAttack_SelectTarget(client)",
	TankTargetFound:    "CTFBotAttackTank_SelectTarget(client)",
	GiantToMark:        "CTFBotMarkGiant_IsPossible(client)",
	NearbyMoney:        "CTFBotCollectNearMoney_SelectTarget(client)",
	StickyTrapPossible: "CTFBotStickyTrap_IsPossible(client)",
}

// name is what the predicate is called in Go and in a failure message.
var predicateName = map[Predicate]string{
	MoneyToCollect: "MoneyToCollect", InUpgradeZone: "InUpgradeZone",
	ShoppedThisBreak: "ShoppedThisBreak", MovingToFront: "MovingToFront",
	UpgradesEnabled: "UpgradesEnabled", HasUpgraded: "HasUpgraded",
	UpgradeMidRound: "UpgradeMidRound", HasSniperRifle: "HasSniperRifle",
	SniperStalled: "SniperStalled", AttackTargetFound: "AttackTargetFound",
	TankTargetFound: "TankTargetFound", GiantToMark: "GiantToMark",
	NearbyMoney: "NearbyMoney", StickyTrapPossible: "StickyTrapPossible",
}

func (p Predicate) String() string {
	if s, ok := predicateName[p]; ok {
		return s
	}
	return fmt.Sprintf("Predicate(%d)", int(p))
}

// Call is the SourcePawn expression that answers p.
func (p Predicate) Call() string { return call[p] }

// Predicates is every question, in declaration order.
func Predicates() []Predicate {
	all := make([]Predicate, 0, numPredicates)
	for p := Predicate(0); p < numPredicates; p++ {
		all = append(all, p)
	}
	return all
}

// Facts answers questions about one bot. The plugin's implementation calls the
// engine, the exhaustive tests answer from a bit set, and the table builder
// records what was asked and in what order.
type Facts interface {
	Ask(Predicate) bool
}

/*
	Flags answers from a fixed set of answers

A struct rather than a bit set because it is what the tests read and write, and
a named field is worth more there than a compact one. FromBits enumerates it,
which is the only place the packing matters.
*/
type Flags struct {
	MoneyToCollect     bool
	InUpgradeZone      bool
	ShoppedThisBreak   bool
	MovingToFront      bool
	UpgradesEnabled    bool
	HasUpgraded        bool
	UpgradeMidRound    bool
	HasSniperRifle     bool
	SniperStalled      bool
	AttackTargetFound  bool
	TankTargetFound    bool
	GiantToMark        bool
	NearbyMoney        bool
	StickyTrapPossible bool
}

// field points at each answer, so Ask and FromBits share one ordering and
// cannot disagree about which bit is which question.
func (f *Flags) field(p Predicate) *bool {
	switch p {
	case MoneyToCollect:
		return &f.MoneyToCollect
	case InUpgradeZone:
		return &f.InUpgradeZone
	case ShoppedThisBreak:
		return &f.ShoppedThisBreak
	case MovingToFront:
		return &f.MovingToFront
	case UpgradesEnabled:
		return &f.UpgradesEnabled
	case HasUpgraded:
		return &f.HasUpgraded
	case UpgradeMidRound:
		return &f.UpgradeMidRound
	case HasSniperRifle:
		return &f.HasSniperRifle
	case SniperStalled:
		return &f.SniperStalled
	case AttackTargetFound:
		return &f.AttackTargetFound
	case TankTargetFound:
		return &f.TankTargetFound
	case GiantToMark:
		return &f.GiantToMark
	case NearbyMoney:
		return &f.NearbyMoney
	case StickyTrapPossible:
		return &f.StickyTrapPossible
	}
	panic(fmt.Sprintf("actionsel: no field for %v", p))
}

// Ask answers from the struct, so every question is free and none is refused.
func (f Flags) Ask(p Predicate) bool { return *f.field(p) }

// FromBits is the answer set numbered n, which is how exhaustiveness
// enumerates the domain.
func FromBits(n int) Flags {
	var f Flags
	for p := Predicate(0); p < numPredicates; p++ {
		*f.field(p) = n&(1<<uint(p)) != 0
	}
	return f
}

// AllFlags is how many answer sets there are.
const AllFlags = 1 << numPredicates

/*
	Reachable refuses the combinations the engine cannot produce

Exhaustiveness is only worth asserting over the domain that exists. Both rifle
and stall are sniper state: m_bSniperStalled is only ever set by the sniper
stall rescue, and HasSniperRifle reads the primary slot of a class that has one.
*/
func Reachable(class Class, f Flags) bool {
	if class == ClassSniper {
		return true
	}
	return !f.HasSniperRifle && !f.SniperStalled
}
