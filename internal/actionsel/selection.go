package actionsel

// Select is GetDesiredBotAction as it stands in nextbot_behavior.sp, branch
// for branch and predicate for predicate, with each Plugin_Continue replaced
// by the name of the reason it is a Plugin_Continue. Nothing else changed:
// every combination reaches the same call site as the shipped chain, in the
// same order, and the one combination the shipped chain strands is stranded
// here too.
func Select(state RoundState, class Class, f Facts) Action {
	switch state {
	case RoundBetweenRounds:
		return selectBetweenRounds(class, f)
	case RoundRunning:
		return selectRoundRunning(class, f)
	}
	return ActionWaitOutsideRound
}

func selectBetweenRounds(class Class, f Facts) Action {
	if f.Ask(MoneyToCollect) {
		return ActionCollectMoneyIsPossible
	}
	if !f.Ask(InUpgradeZone) && !f.Ask(ShoppedThisBreak) && !f.Ask(MovingToFront) {
		if f.Ask(UpgradesEnabled) {
			return ActionGotoUpgradeBetweenRounds
		}
		return ActionMoveToFrontSkipUpgrading
	}
	if (shouldTakeUpPosition(class, f) || f.Ask(SniperStalled)) && !f.Ask(MovingToFront) {
		return ActionMoveToFrontShoppingDone
	}
	if f.Ask(MovingToFront) {
		return ActionKeepWalkingToFront
	}
	return ActionKeepOwnBreakBehaviour
}

func selectRoundRunning(class Class, f Facts) Action {
	if f.Ask(UpgradesEnabled) && (!f.Ask(HasUpgraded) || f.Ask(UpgradeMidRound)) && !f.Ask(InUpgradeZone) {
		return ActionGotoUpgradeBuyNow
	}

	switch class {
	case ClassMedic:
		return ActionKeepHealing
	case ClassScout:
		if f.Ask(MoneyToCollect) {
			return ActionCollectMoneyCollecting
		}
		if f.Ask(GiantToMark) {
			return ActionMarkGiant
		}
		if f.Ask(TankTargetFound) {
			return ActionAttackTankScout
		}
		if f.Ask(AttackTargetFound) {
			return ActionDefenderAttackScout
		}
	case ClassSniper:
		if f.Ask(HasSniperRifle) && !f.Ask(SniperStalled) {
			return ActionKeepSnipingPosition
		}
		return ActionDefenderAttackSniper
	case ClassEngineer:
		return ActionEngineerIdle
	case ClassSpy:
		return ActionSpyLurk
	case ClassHeavy:
		if f.Ask(AttackTargetFound) {
			return ActionDefenderAttackIsPossible
		}
		if f.Ask(TankTargetFound) {
			return ActionAttackTank
		}
		if f.Ask(NearbyMoney) {
			return ActionCollectNearMoney
		}
	case ClassDemoMan:
		if f.Ask(TankTargetFound) {
			return ActionAttackTank
		}
		if f.Ask(AttackTargetFound) {
			return ActionDefenderAttackIsPossible
		}
		if f.Ask(StickyTrapPossible) {
			return ActionStickyTrap
		}
		if f.Ask(NearbyMoney) {
			return ActionCollectNearMoney
		}
	case ClassSoldier, ClassPyro:
		if f.Ask(TankTargetFound) {
			return ActionAttackTank
		}
		if f.Ask(AttackTargetFound) {
			return ActionDefenderAttackIsPossible
		}
		if f.Ask(NearbyMoney) {
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
func shouldTakeUpPosition(class Class, f Facts) bool {
	switch class {
	case ClassEngineer, ClassSpy:
		return false
	case ClassSniper:
		return !f.Ask(HasSniperRifle)
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
