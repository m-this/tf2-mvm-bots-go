/*
Package dispatch is GetDesiredBotAction out of
source/redbots3/nextbot_behavior.sp: the busiest hand-off in the mod, where a
bot with nothing on its stack is given the behaviour its class and the round
state call for.
*/
package dispatch

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// ShouldTakeUpPosition says this bot's class walks to the front before the wave
// rather than waiting where it shopped.
//
//sp:name ShouldTakeUpPosition
func ShouldTakeUpPosition(client int32) bool {
	switch engine.PlayerClass(client) {
	case engine.ClassEngineer(), engine.ClassSpy():
		return false

	case engine.ClassSniper():
		return !engine.HasSniperRifle(client)
	}

	return true
}

// CanGuardTheHatch says the class fights well enough standing still for the
// hatch to be worth holding.
//
//sp:name CanGuardTheHatch
func CanGuardTheHatch(client int32) bool {
	switch engine.PlayerClass(client) {
	case engine.ClassSoldier(), engine.ClassPyro(), engine.ClassDemoMan(), engine.ClassHeavyweapons():
		return true
	}

	return false
}

/*
GetDesiredBotAction is the whole hand-off.

Between rounds: leftover money first, then the shopping trip, then the walk to
the front, because a behaviour that says nothing hands the bot back to the game,
whose answer for a defender with no mission is to roam.

Mid-round, per class, and when every branch needs something that is not
happening: the hatch. CTFBotGuardPoint was in this repository the whole time and
nothing ever called it; it walks to the ground around the hatch and hands the
bot back the moment there is something to fight.

A rifle sniper is given nothing here because Timer_PlayerSpawn is meant to have
given him his sniping behaviour already, and a sniper caught stalling fights
like one who never had a rifle. See mvm-bj8.
*/
//
//sp:name GetDesiredBotAction
//
//nolint:revive,gocritic // unused-parameter, singleCaseSwitch: the action is what SuspendFor writes through, and the switch is the shipped shape
func GetDesiredBotAction(client int32, action engine.Behaviour) engine.Outcome {
	state := engine.RoundState()

	if state == engine.RoundStateRunning() && engine.HasUpgraded(client) {
		engine.RecoverDefenderFromDisconnectedSpawn(client)
	}

	if state == engine.RoundStateBetweenRounds() {
		if engine.CollectMoneyIsPossible(client) {
			// Collect any leftover money that my team didn't collect.
			return engine.SuspendFor(engine.CollectMoney(), "Is possible")
		} else if !engine.IsInUpgradeZone(client) && !engine.ShoppedThisBreak(client) && engine.LookupEntityActionByName(client, "DefenderMoveToFront") == engine.InvalidAction() {
			if engine.UseUpgrades().Bool() {
				return engine.SuspendFor(engine.GotoUpgrade(), "!IsInUpgradeZone && RoundState_BetweenRounds")
			}

			engine.SetPlayerReady(client, true)
			return engine.SuspendFor(engine.MoveToFront(), "Skip upgrading")
		} else if (ShouldTakeUpPosition(client) || engine.IsSniperStalled(client)) && engine.LookupEntityActionByName(client, "DefenderMoveToFront") == engine.InvalidAction() {
			/* Shopping is finished, so go and stand where the robots come out

			Without this the break has nothing left to say to a bot that has
			bought its upgrades. Reported as the Heavy, the Medic and the Pyro
			wandering off before the wave, and found inside the middle house on
			Coaltown. */
			return engine.SuspendFor(engine.MoveToFront(), "Shopping is done, so take up a position")
		}
	} else if state == engine.RoundStateRunning() {
		if engine.UseUpgrades().Bool() && (!engine.HasUpgraded(client) || engine.ShouldUpgradeMidRoundNow(client)) && !engine.IsInUpgradeZone(client) {
			// We probably just joined in the middle of an active game, or we
			// want to buy upgrades again right now.
			engine.SetBuyUpgradesNumber(client, 0)

			return engine.SuspendFor(engine.GotoUpgrade(), "Buy upgrades now")
		}

		// Health and ammo live in CTFBotTacticalMonitor_Update, which takes
		// precedence over ScenarioMonitor.

		switch engine.PlayerClass(client) {
		case engine.ClassMedic():
			// Medics automatically start healing.
			return engine.PluginContinue()
		case engine.ClassScout():
			if engine.CollectMoneyIsPossible(client) {
				return engine.SuspendFor(engine.CollectMoney(), "Collecting money")
			} else if engine.MarkGiantIsPossible(client) {
				return engine.SuspendFor(engine.MarkGiant(), "Marking giant")
			} else if engine.AttackTankSelectTarget(client) {
				return engine.SuspendFor(engine.AttackTank(), "Scout: Attacking tank")
			} else if engine.DefenderAttackSelectTarget(client) {
				return engine.SuspendFor(engine.DefenderAttack(), "Scout: Attacking robots")
			}
		case engine.ClassSniper():
			if engine.HasSniperRifle(client) && !engine.IsSniperStalled(client) {
				// The sniping behaviour is set manually in Timer_PlayerSpawn.
				return engine.PluginContinue()
			}

			return engine.SuspendFor(engine.DefenderAttack(), "Sniper Attacking robots")
		case engine.ClassEngineer():
			return engine.SuspendFor(engine.EngineerIdle(), "Engineer Start building")
		case engine.ClassSpy():
			return engine.SuspendFor(engine.SpyLurk(), "Spy do be lurking")
		case engine.ClassHeavyweapons():
			if engine.DefenderAttackSelectTarget(client) {
				return engine.SuspendFor(engine.DefenderAttack(), "CTFBotAttack_IsPossible")
			} else if engine.AttackTankSelectTarget(client) {
				return engine.SuspendFor(engine.AttackTank(), "Attacking tank")
			} else if engine.CollectNearMoneySelectTarget(client) {
				return engine.SuspendFor(engine.CollectNearMoney(), "Nearby money")
			}
		case engine.ClassDemoMan():
			if engine.AttackTankSelectTarget(client) {
				return engine.SuspendFor(engine.AttackTank(), "Attacking tank")
			} else if engine.DefenderAttackSelectTarget(client) {
				return engine.SuspendFor(engine.DefenderAttack(), "CTFBotAttack_IsPossible")
			} else if engine.StickyTrapIsPossible(client) {
				return engine.SuspendFor(engine.StickyTrap(), "Nothing to fight, so lay a trap")
			} else if engine.CollectNearMoneySelectTarget(client) {
				return engine.SuspendFor(engine.CollectNearMoney(), "Nearby money")
			}
		case engine.ClassSoldier(), engine.ClassPyro():
			if engine.AttackTankSelectTarget(client) {
				return engine.SuspendFor(engine.AttackTank(), "Attacking tank")
			} else if engine.DefenderAttackSelectTarget(client) {
				return engine.SuspendFor(engine.DefenderAttack(), "CTFBotAttack_IsPossible")
			} else if engine.CollectNearMoneySelectTarget(client) {
				return engine.SuspendFor(engine.CollectNearMoney(), "Nearby money")
			}
		}

		if CanGuardTheHatch(client) {
			return engine.SuspendFor(engine.GuardPoint(), "Nothing to do, so hold the hatch")
		}
	}

	return engine.PluginContinue()
}

/*
GetUpgradePostAction is what a bot does the moment it walks out of the upgrade
zone: the engineer builds, the medic heals, the spy lurks, the sniper takes his
mission, and everybody else walks to the front and presses F4.
*/
//
//sp:name GetUpgradePostAction
//
//nolint:revive,gocritic // unused-parameter, ifElseChain: the action is what ChangeTo writes through, and the chain is the shipped shape
func GetUpgradePostAction(client int32, action engine.Behaviour) engine.Outcome {
	if engine.RoundState() == engine.RoundStateBetweenRounds() {
		if engine.PlayerClass(client) == engine.ClassEngineer() {
			return engine.ChangeTo(engine.EngineerIdle(), "Start building")
		} else if engine.PlayerClass(client) == engine.ClassMedic() {
			return engine.Done("Start heal mission")
		} else if engine.PlayerClass(client) == engine.ClassSpy() {
			return engine.ChangeTo(engine.SpyLurk(), "Start spy lurking")
		} else if engine.HasSniperRifle(client) {
			return engine.Done("Start lurking")
		}

		return engine.ChangeTo(engine.MoveToFront(), "Finished upgrading; Move to front and press F4")
	}

	/* The round's probably already running.
	CTFBotScenarioMonitor_Update will assign the appropriate task. */
	return engine.Done("I finished upgrading")
}
