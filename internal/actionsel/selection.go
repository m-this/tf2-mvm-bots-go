// Package actionsel is the choice of which behaviour a defender bot is handed,
// written as a total function over the inputs the choice actually reads.
//
// It models GetDesiredBotAction and ShouldTakeUpPosition in
// source/redbots3/nextbot_behavior.sp. Select is what the choice should be:
// every reachable combination of round state, class and flags returns a named
// action, including the ones where the right answer is "leave him alone",
// which are actions with names rather than a fallthrough. Shipped is what the
// plugin does today, transcribed branch for branch, and it returns ActionNone
// wherever the plugin returns Plugin_Continue with nothing on the stack.
// The difference between the two is the list of holes still shipping.
//
// The whole file is inside the internal/gosubset subset, checked by the test,
// because this is meant to be generated into the SourcePawn it replaces.
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

// Select is the choice with no holes in it.
func Select(state RoundState, class Class, f Flags) Action {
	switch state {
	case RoundBetweenRounds:
		return selectBetweenRounds(class, f)
	case RoundRunning:
		return selectRoundRunning(class, f)
	}
	return ActionWaitOutsideRound
}

func selectBetweenRounds(class Class, f Flags) Action {
	if f.MoneyToCollect {
		return ActionCollectMoney
	}
	if !f.InUpgradeZone && !f.ShoppedThisBreak && !f.MovingToFront {
		if f.UpgradesEnabled {
			return ActionGotoUpgrade
		}
		return ActionMoveToFrontSkippingUpgrades
	}
	if f.MovingToFront {
		return ActionKeepWalkingToFront
	}
	if shouldTakeUpPosition(class, f) || f.SniperStalled {
		return ActionMoveToFront
	}
	return ActionKeepOwnBreakBehaviour
}

func selectRoundRunning(class Class, f Flags) Action {
	if f.UpgradesEnabled && (!f.HasUpgraded || f.UpgradeMidRound) && !f.InUpgradeZone {
		return ActionGotoUpgrade
	}

	switch class {
	case ClassUnknown:
		return ActionKeepWaitingForClass
	case ClassMedic:
		return ActionKeepHealing
	case ClassEngineer:
		return ActionEngineerIdle
	case ClassSpy:
		return ActionSpyLurk
	case ClassSniper:
		if f.HasSniperRifle && !f.SniperStalled {
			return ActionKeepSnipingPosition
		}
		return ActionDefenderAttack
	case ClassScout:
		if f.MoneyToCollect {
			return ActionCollectMoney
		}
		if f.GiantToMark {
			return ActionMarkGiant
		}
		if f.TankTargetFound {
			return ActionAttackTank
		}
		if f.AttackTargetFound {
			return ActionDefenderAttack
		}
	case ClassHeavy:
		if f.AttackTargetFound {
			return ActionDefenderAttack
		}
		if f.TankTargetFound {
			return ActionAttackTank
		}
		if f.NearbyMoney {
			return ActionCollectNearMoney
		}
	case ClassDemoMan:
		if f.TankTargetFound {
			return ActionAttackTank
		}
		if f.AttackTargetFound {
			return ActionDefenderAttack
		}
		if f.StickyTrapPossible {
			return ActionStickyTrap
		}
		if f.NearbyMoney {
			return ActionCollectNearMoney
		}
	case ClassSoldier, ClassPyro:
		if f.TankTargetFound {
			return ActionAttackTank
		}
		if f.AttackTargetFound {
			return ActionDefenderAttack
		}
		if f.NearbyMoney {
			return ActionCollectNearMoney
		}
	}

	return ActionGuardPoint
}

// shouldTakeUpPosition is ShouldTakeUpPosition: whether the break ends with
// this one walking to where the robots come out.
func shouldTakeUpPosition(class Class, f Flags) bool {
	switch class {
	case ClassEngineer, ClassSpy:
		return false
	case ClassSniper:
		return !f.HasSniperRifle
	}
	return true
}

// canGuardTheHatch is CanGuardTheHatch, used only by Shipped: the four classes
// the plugin lets fall back to holding the hatch.
func canGuardTheHatch(class Class) bool {
	switch class {
	case ClassSoldier, ClassPyro, ClassDemoMan, ClassHeavy:
		return true
	}
	return false
}

// Shipped is GetDesiredBotAction as it stands in nextbot_behavior.sp, branch
// for branch. ActionNone is Plugin_Continue reached with no behaviour handed
// out: the bot goes back to the game, which has no answer for a defender.
func Shipped(state RoundState, class Class, f Flags) Action {
	if state == RoundBetweenRounds {
		return shippedBetweenRounds(class, f)
	}
	if state == RoundRunning {
		return shippedRoundRunning(class, f)
	}
	return ActionNone
}

func shippedBetweenRounds(class Class, f Flags) Action {
	if f.MoneyToCollect {
		return ActionCollectMoney
	}
	if !f.InUpgradeZone && !f.ShoppedThisBreak && !f.MovingToFront {
		if f.UpgradesEnabled {
			return ActionGotoUpgrade
		}
		return ActionMoveToFrontSkippingUpgrades
	}
	if (shouldTakeUpPosition(class, f) || f.SniperStalled) && !f.MovingToFront {
		return ActionMoveToFront
	}
	return ActionNone
}

func shippedRoundRunning(class Class, f Flags) Action {
	if f.UpgradesEnabled && (!f.HasUpgraded || f.UpgradeMidRound) && !f.InUpgradeZone {
		return ActionGotoUpgrade
	}

	switch class {
	case ClassMedic:
		return ActionNone
	case ClassEngineer:
		return ActionEngineerIdle
	case ClassSpy:
		return ActionSpyLurk
	case ClassSniper:
		if f.HasSniperRifle && !f.SniperStalled {
			return ActionNone
		}
		return ActionDefenderAttack
	case ClassScout:
		if f.MoneyToCollect {
			return ActionCollectMoney
		}
		if f.GiantToMark {
			return ActionMarkGiant
		}
		if f.TankTargetFound {
			return ActionAttackTank
		}
		if f.AttackTargetFound {
			return ActionDefenderAttack
		}
	case ClassHeavy:
		if f.AttackTargetFound {
			return ActionDefenderAttack
		}
		if f.TankTargetFound {
			return ActionAttackTank
		}
		if f.NearbyMoney {
			return ActionCollectNearMoney
		}
	case ClassDemoMan:
		if f.TankTargetFound {
			return ActionAttackTank
		}
		if f.AttackTargetFound {
			return ActionDefenderAttack
		}
		if f.StickyTrapPossible {
			return ActionStickyTrap
		}
		if f.NearbyMoney {
			return ActionCollectNearMoney
		}
	case ClassSoldier, ClassPyro:
		if f.TankTargetFound {
			return ActionAttackTank
		}
		if f.AttackTargetFound {
			return ActionDefenderAttack
		}
		if f.NearbyMoney {
			return ActionCollectNearMoney
		}
	}

	if canGuardTheHatch(class) {
		return ActionGuardPoint
	}
	return ActionNone
}
