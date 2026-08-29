// Package actionsel is a fixture: the shipped internal/actionsel code, split
// across the files it should have been, so the checker is proved against a
// package rather than a single file. Every file here leans on names declared
// in another one.
//
// domain.go holds the types and the reachability rule.
package actionsel

// RoundState is SourceMod's RoundState, in its declared order, from
// sdktools_gamerules.inc. The choice reads only two of the eleven.
type RoundState int32

// The eleven round states, in declared order.
const (
	RoundInit RoundState = iota
	RoundPregame
	RoundStartGame
	RoundPreround
	RoundRunning
	RoundTeamWin
	RoundRestart
	RoundStalemate
	RoundGameOver
	RoundBonus
	RoundBetweenRounds
)

// Class is TFClassType from tf2.inc, in its declared order.
type Class int32

// The ten classes, in declared order.
const (
	ClassUnknown Class = iota
	ClassScout
	ClassSniper
	ClassSoldier
	ClassDemoMan
	ClassMedic
	ClassHeavy
	ClassPyro
	ClassSpy
	ClassEngineer
)

// Flags is every per-bot fact GetDesiredBotAction branches on, in the order it
// reads them. Each one is a call the hand-written SourcePawn makes before it
// asks for an action.
type Flags struct {
	// MoneyToCollect is CTFBotCollectMoney_IsPossible.
	MoneyToCollect bool
	// InUpgradeZone is TF2_IsInUpgradeZone.
	InUpgradeZone bool
	// ShoppedThisBreak is g_bShoppedThisBreak.
	ShoppedThisBreak bool
	// MovingToFront is a DefenderMoveToFront already on the action stack.
	MovingToFront bool
	// UpgradesEnabled is redbots_manager_bot_use_upgrades.
	UpgradesEnabled bool
	// HasUpgraded is g_bHasUpgraded.
	HasUpgraded bool
	// UpgradeMidRound is ShouldUpgradeMidRound.
	UpgradeMidRound bool
	// HasSniperRifle is HasSniperRifle.
	HasSniperRifle bool
	// SniperStalled is IsSniperStalled.
	SniperStalled bool
	// AttackTargetFound is CTFBotDefenderAttack_SelectTarget.
	AttackTargetFound bool
	// TankTargetFound is CTFBotAttackTank_SelectTarget.
	TankTargetFound bool
	// GiantToMark is CTFBotMarkGiant_IsPossible.
	GiantToMark bool
	// NearbyMoney is CTFBotCollectNearMoney_SelectTarget.
	NearbyMoney bool
	// StickyTrapPossible is CTFBotStickyTrap_IsPossible.
	StickyTrapPossible bool
}

// Action is what the bot is handed. The five Keep actions are the cases where
// doing nothing here is correct because something else already owns the bot;
// they exist so that "nothing" is a decision with a name and ActionNone is
// only ever a hole.
type Action int32

// Every action the choice can hand out.
const (
	// ActionNone is the hole. Select never returns it.
	ActionNone Action = iota
	ActionCollectMoney
	ActionGotoUpgrade
	ActionMoveToFront
	// ActionMoveToFrontSkippingUpgrades also readies the bot, which is what
	// the plugin does on this branch and nowhere else.
	ActionMoveToFrontSkippingUpgrades
	ActionMarkGiant
	ActionAttackTank
	ActionDefenderAttack
	ActionStickyTrap
	ActionCollectNearMoney
	ActionEngineerIdle
	ActionSpyLurk
	ActionGuardPoint
	// ActionKeepWalkingToFront means DefenderMoveToFront is on the stack already.
	ActionKeepWalkingToFront
	// ActionKeepOwnBreakBehaviour is the engineer's nest, the spy's lurk
	// and the rifle sniper's perch, given by GetUpgradePostAction.
	ActionKeepOwnBreakBehaviour
	// ActionKeepSnipingPosition is given by Timer_PlayerSpawn.
	ActionKeepSnipingPosition
	// ActionKeepHealing is the game's own medic behaviour.
	ActionKeepHealing
	// ActionKeepWaitingForClass is a bot with no class yet.
	ActionKeepWaitingForClass
	// ActionWaitOutsideRound is the nine round states the choice never reads.
	ActionWaitOutsideRound
)

// Reachable refuses the combinations the engine cannot produce, so that
// exhaustiveness is asserted over the real domain and not over the product.
// Both rifle and stall are sniper state: m_bSniperStalled is only ever set by
// the sniper stall rescue, and HasSniperRifle reads the primary slot of a
// class that has one.
func Reachable(class Class, f Flags) bool {
	if class == ClassSniper {
		return true
	}
	return !f.HasSniperRifle && !f.SniperStalled
}
