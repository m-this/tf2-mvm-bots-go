package actionsel

// Action is one outcome of the choice: one call site of GetDesiredBotAction,
// not one behaviour. Two call sites that hand out the same behaviour with
// different SuspendFor reasons are two outcomes, because the reason reaches
// the debug output and the test-bed's telemetry, and renaming one is a change
// to the reporting even when the bot does the same thing.
//
// internal/spgen turns this enum into the SourcePawn enum and the dispatch
// function that maps an id back onto the SuspendFor call it came from. The
// pairing of an id with its action and its reason lives once, in
// internal/spgen's edge table, and a test checks it against the plugin source.
type Action int32

// Every outcome the choice can hand out. The first seventeen suspend the bot
// for a behaviour; the rest are the plugin returning Plugin_Continue.
const (
	// ActionNone is the hole. Select never returns it, and the dispatch
	// treats it as a bug rather than as a silence.
	ActionNone Action = iota

	// ActionCollectMoneyIsPossible is CTFBotCollectMoney, "Is possible".
	ActionCollectMoneyIsPossible
	// ActionGotoUpgradeBetweenRounds is CTFBotGotoUpgrade,
	// "!IsInUpgradeZone && RoundState_BetweenRounds".
	ActionGotoUpgradeBetweenRounds
	// ActionMoveToFrontSkipUpgrading is CTFBotMoveToFront, "Skip upgrading".
	// The shipped call site readies the player first, which is the one
	// side effect the dispatch carries.
	ActionMoveToFrontSkipUpgrading
	// ActionMoveToFrontShoppingDone is CTFBotMoveToFront,
	// "Shopping is done, so take up a position".
	ActionMoveToFrontShoppingDone
	// ActionGotoUpgradeBuyNow is CTFBotGotoUpgrade, "Buy upgrades now". The
	// shipped call site zeroes g_iBuyUpgradesNumber first.
	ActionGotoUpgradeBuyNow
	// ActionCollectMoneyCollecting is CTFBotCollectMoney, "Collecting money".
	ActionCollectMoneyCollecting
	// ActionMarkGiant is CTFBotMarkGiant, "Marking giant".
	ActionMarkGiant
	// ActionAttackTankScout is CTFBotAttackTank, "Scout: Attacking tank".
	ActionAttackTankScout
	// ActionDefenderAttackScout is CTFBotDefenderAttack,
	// "Scout: Attacking robots".
	ActionDefenderAttackScout
	// ActionDefenderAttackSniper is CTFBotDefenderAttack,
	// "Sniper Attacking robots".
	ActionDefenderAttackSniper
	// ActionEngineerIdle is CTFBotMvMEngineerIdle, "Engineer Start building".
	ActionEngineerIdle
	// ActionSpyLurk is CTFBotSpyLurkMvM, "Spy do be lurking".
	ActionSpyLurk
	// ActionDefenderAttackIsPossible is CTFBotDefenderAttack,
	// "CTFBotAttack_IsPossible", the Heavy, DemoMan, Soldier and Pyro one.
	ActionDefenderAttackIsPossible
	// ActionAttackTank is CTFBotAttackTank, "Attacking tank".
	ActionAttackTank
	// ActionCollectNearMoney is CTFBotCollectNearMoney, "Nearby money".
	ActionCollectNearMoney
	// ActionStickyTrap is CTFBotStickyTrap,
	// "Nothing to fight, so lay a trap".
	ActionStickyTrap
	// ActionGuardPoint is CTFBotGuardPoint,
	// "Nothing to do, so hold the hatch".
	ActionGuardPoint

	// ActionKeepWalkingToFront is Plugin_Continue with DefenderMoveToFront
	// already on the stack.
	ActionKeepWalkingToFront
	// ActionKeepOwnBreakBehaviour is Plugin_Continue during the break: the
	// engineer's nest, the spy's lurk and the rifle sniper's perch, given by
	// GetUpgradePostAction.
	ActionKeepOwnBreakBehaviour
	// ActionKeepSnipingPosition is Plugin_Continue for a rifle sniper who is
	// not stalled, given his behaviour by Timer_PlayerSpawn.
	ActionKeepSnipingPosition
	// ActionKeepHealing is Plugin_Continue for a medic, who starts healing on
	// the game's own behaviour.
	ActionKeepHealing
	// ActionKeepWaitingForClass is Plugin_Continue for a bot with no class.
	ActionKeepWaitingForClass
	// ActionWaitOutsideRound is Plugin_Continue in the nine round states the
	// choice never reads.
	ActionWaitOutsideRound
	// ActionStrandedAsShipped is Plugin_Continue where the shipped chain has
	// nothing to say and the bot is left standing. That is mvm-vnn, it is a
	// defect, and it stays here because this is a port. SelectFilled is the
	// candidate fix, measured on its own.
	ActionStrandedAsShipped
)

// Suspends says whether the plugin puts a behaviour on the stack for this
// outcome. ActionKeepWalkingToFront is the first Plugin_Continue, and every
// outcome declared after it is one too, which is why the enum is ordered that
// way and why this is a range check rather than a list to keep in step.
func Suspends(a Action) bool {
	return a != ActionNone && a < ActionKeepWalkingToFront
}
