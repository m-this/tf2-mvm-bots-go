package spgen

import "github.com/m-this/tf2-mvm-bots-go/internal/actionsel"

// The edge is the one place a decision id is paired with what the plugin does
// about it. It is written here once and emitted twice: as the SourcePawn that
// asks a predicate, and as the SourcePawn that hands out a behaviour. Nothing
// downstream keeps a second copy, which is the whole reason this repository
// exists.

// Predicate is one of the facts the decision reads, and the SourcePawn that
// answers it. Both come from actionsel.Predicate, which is the one place a
// question is named and paired with the call that answers it. Order is the
// order the ids run in.
type Predicate struct {
	Field string
	Call  string
}

// ActionSelPredicates is every question the decision can ask, in id order.
var ActionSelPredicates = actionSelPredicates()

func actionSelPredicates() []Predicate {
	all := actionsel.Predicates()
	out := make([]Predicate, 0, len(all))
	for _, p := range all {
		out = append(out, Predicate{Field: p.String(), Call: p.Call()})
	}
	return out
}

func predicateFields() []string {
	out := make([]string, 0, len(ActionSelPredicates))
	for _, p := range ActionSelPredicates {
		out = append(out, p.Field)
	}
	return out
}

// Outcome is one call site of GetDesiredBotAction: the behaviour it suspends
// for, the reason string it logs, and the statement it runs first. A reason is
// part of the behaviour, because it reaches the debug output and the
// test-bed's telemetry, so two call sites with the same behaviour and
// different reasons are two outcomes.
type Outcome struct {
	// Const is the Go constant in internal/actionsel, without its Action
	// prefix stripped: it must match the enum name for name to be one fact.
	Const string
	// Behaviour is the constructor, empty when the call site is a
	// Plugin_Continue.
	Behaviour string
	// Reason is the SuspendFor string, byte for byte.
	Reason string
	// Effect is the statement the shipped call site runs before suspending.
	Effect string
	// Note explains a Plugin_Continue.
	Note string
}

// ActionSelOutcomes is every outcome of the choice, in the order of the Go
// enum, so the emitted switch and the enum cannot drift apart.
var ActionSelOutcomes = []Outcome{
	{Const: "ActionNone", Note: "never returned; reaching it is a bug in the table"},
	{Const: "ActionCollectMoneyIsPossible", Behaviour: "CTFBotCollectMoney", Reason: "Is possible"},
	{Const: "ActionGotoUpgradeBetweenRounds", Behaviour: "CTFBotGotoUpgrade", Reason: "!IsInUpgradeZone && RoundState_BetweenRounds"},
	{Const: "ActionMoveToFrontSkipUpgrading", Behaviour: "CTFBotMoveToFront", Reason: "Skip upgrading", Effect: "SetPlayerReady(client, true);"},
	{Const: "ActionMoveToFrontShoppingDone", Behaviour: "CTFBotMoveToFront", Reason: "Shopping is done, so take up a position"},
	{Const: "ActionGotoUpgradeBuyNow", Behaviour: "CTFBotGotoUpgrade", Reason: "Buy upgrades now", Effect: "g_iBuyUpgradesNumber[client] = 0;"},
	{Const: "ActionCollectMoneyCollecting", Behaviour: "CTFBotCollectMoney", Reason: "Collecting money"},
	{Const: "ActionMarkGiant", Behaviour: "CTFBotMarkGiant", Reason: "Marking giant"},
	{Const: "ActionAttackTankScout", Behaviour: "CTFBotAttackTank", Reason: "Scout: Attacking tank"},
	{Const: "ActionDefenderAttackScout", Behaviour: "CTFBotDefenderAttack", Reason: "Scout: Attacking robots"},
	{Const: "ActionDefenderAttackSniper", Behaviour: "CTFBotDefenderAttack", Reason: "Sniper Attacking robots"},
	{Const: "ActionEngineerIdle", Behaviour: "CTFBotMvMEngineerIdle", Reason: "Engineer Start building"},
	{Const: "ActionSpyLurk", Behaviour: "CTFBotSpyLurkMvM", Reason: "Spy do be lurking"},
	{Const: "ActionDefenderAttackIsPossible", Behaviour: "CTFBotDefenderAttack", Reason: "CTFBotAttack_IsPossible"},
	{Const: "ActionAttackTank", Behaviour: "CTFBotAttackTank", Reason: "Attacking tank"},
	{Const: "ActionCollectNearMoney", Behaviour: "CTFBotCollectNearMoney", Reason: "Nearby money"},
	{Const: "ActionStickyTrap", Behaviour: "CTFBotStickyTrap", Reason: "Nothing to fight, so lay a trap"},
	{Const: "ActionGuardPoint", Behaviour: "CTFBotGuardPoint", Reason: "Nothing to do, so hold the hatch"},
	{Const: "ActionKeepWalkingToFront", Note: "DefenderMoveToFront is on the stack already"},
	{Const: "ActionKeepOwnBreakBehaviour", Note: "the engineer's nest, the spy's lurk, the rifle sniper's perch"},
	{Const: "ActionKeepSnipingPosition", Note: "Timer_PlayerSpawn gave him the sniping behaviour"},
	{Const: "ActionKeepHealing", Note: "the medic starts healing on the game's own behaviour"},
	{Const: "ActionKeepWaitingForClass", Note: "no class yet"},
	{Const: "ActionWaitOutsideRound", Note: "the nine round states the choice never reads"},
	{Const: "ActionStrandedAsShipped", Note: "mvm-vnn: the shipped chain leaves this bot standing, and the port does too"},
}
