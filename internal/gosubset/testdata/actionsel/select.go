package actionsel

// select.go is the choice with no holes in it, over the domain in domain.go.

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
