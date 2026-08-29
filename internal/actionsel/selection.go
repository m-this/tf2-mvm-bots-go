package actionsel

// Select is GetDesiredBotAction as it stands in nextbot_behavior.sp, branch
// for branch and predicate for predicate, with each Plugin_Continue replaced
// by the name of the reason it is a Plugin_Continue. Nothing else changed:
// every combination reaches the same call site as the shipped chain, in the
// same order, and the one combination the shipped chain strands is stranded
// here too.
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
		return ActionCollectMoneyIsPossible
	}
	if !f.InUpgradeZone && !f.ShoppedThisBreak && !f.MovingToFront {
		if f.UpgradesEnabled {
			return ActionGotoUpgradeBetweenRounds
		}
		return ActionMoveToFrontSkipUpgrading
	}
	if (shouldTakeUpPosition(class, f) || f.SniperStalled) && !f.MovingToFront {
		return ActionMoveToFrontShoppingDone
	}
	if f.MovingToFront {
		return ActionKeepWalkingToFront
	}
	return ActionKeepOwnBreakBehaviour
}

func selectRoundRunning(class Class, f Flags) Action {
	if f.UpgradesEnabled && (!f.HasUpgraded || f.UpgradeMidRound) && !f.InUpgradeZone {
		return ActionGotoUpgradeBuyNow
	}

	switch class {
	case ClassMedic:
		return ActionKeepHealing
	case ClassScout:
		if f.MoneyToCollect {
			return ActionCollectMoneyCollecting
		}
		if f.GiantToMark {
			return ActionMarkGiant
		}
		if f.TankTargetFound {
			return ActionAttackTankScout
		}
		if f.AttackTargetFound {
			return ActionDefenderAttackScout
		}
	case ClassSniper:
		if f.HasSniperRifle && !f.SniperStalled {
			return ActionKeepSnipingPosition
		}
		return ActionDefenderAttackSniper
	case ClassEngineer:
		return ActionEngineerIdle
	case ClassSpy:
		return ActionSpyLurk
	case ClassHeavy:
		if f.AttackTargetFound {
			return ActionDefenderAttackIsPossible
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
			return ActionDefenderAttackIsPossible
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
			return ActionDefenderAttackIsPossible
		}
		if f.NearbyMoney {
			return ActionCollectNearMoney
		}
	}

	if canGuardTheHatch(class) {
		return ActionGuardPoint
	}
	if class == ClassUnknown {
		return ActionKeepWaitingForClass
	}
	return ActionStrandedAsShipped
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

// canGuardTheHatch is CanGuardTheHatch: the four classes the shipped chain
// lets fall back to holding the hatch.
func canGuardTheHatch(class Class) bool {
	switch class {
	case ClassSoldier, ClassPyro, ClassDemoMan, ClassHeavy:
		return true
	}
	return false
}
