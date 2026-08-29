package actionsel

// shipped.go is what the plugin does today. It calls shouldTakeUpPosition
// from select.go, which is the cross-file reference the checker used to refuse.

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
