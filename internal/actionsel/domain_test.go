package actionsel

import (
	"fmt"
	"strings"
)

// flagCount is the number of booleans in Flags. The enumeration is one bit per
// flag, so a field added without a line in flagBit fails TestFlagBitsAreWhole.
const flagCount = 14

var roundStates = []RoundState{
	RoundInit, RoundPregame, RoundStartGame, RoundPreround, RoundRunning,
	RoundTeamWin, RoundRestart, RoundStalemate, RoundGameOver, RoundBonus,
	RoundBetweenRounds,
}

var roundStateNames = map[RoundState]string{
	RoundInit: "Init", RoundPregame: "Pregame", RoundStartGame: "StartGame",
	RoundPreround: "Preround", RoundRunning: "RoundRunning", RoundTeamWin: "TeamWin",
	RoundRestart: "Restart", RoundStalemate: "Stalemate", RoundGameOver: "GameOver",
	RoundBonus: "Bonus", RoundBetweenRounds: "BetweenRounds",
}

var classes = []Class{
	ClassUnknown, ClassScout, ClassSniper, ClassSoldier, ClassDemoMan,
	ClassMedic, ClassHeavy, ClassPyro, ClassSpy, ClassEngineer,
}

var classNames = map[Class]string{
	ClassUnknown: "Unknown", ClassScout: "Scout", ClassSniper: "Sniper",
	ClassSoldier: "Soldier", ClassDemoMan: "DemoMan", ClassMedic: "Medic",
	ClassHeavy: "Heavy", ClassPyro: "Pyro", ClassSpy: "Spy",
	ClassEngineer: "Engineer",
}

var actionNames = map[Action]string{
	ActionNone:                     "None",
	ActionCollectMoneyIsPossible:   "CollectMoneyIsPossible",
	ActionGotoUpgradeBetweenRounds: "GotoUpgradeBetweenRounds",
	ActionMoveToFrontSkipUpgrading: "MoveToFrontSkipUpgrading",
	ActionMoveToFrontShoppingDone:  "MoveToFrontShoppingDone",
	ActionGotoUpgradeBuyNow:        "GotoUpgradeBuyNow",
	ActionCollectMoneyCollecting:   "CollectMoneyCollecting",
	ActionMarkGiant:                "MarkGiant",
	ActionAttackTankScout:          "AttackTankScout",
	ActionDefenderAttackScout:      "DefenderAttackScout",
	ActionDefenderAttackSniper:     "DefenderAttackSniper",
	ActionEngineerIdle:             "EngineerIdle",
	ActionSpyLurk:                  "SpyLurk",
	ActionDefenderAttackIsPossible: "DefenderAttackIsPossible",
	ActionAttackTank:               "AttackTank",
	ActionCollectNearMoney:         "CollectNearMoney",
	ActionStickyTrap:               "StickyTrap",
	ActionGuardPoint:               "GuardPoint",
	ActionKeepWalkingToFront:       "KeepWalkingToFront",
	ActionKeepOwnBreakBehaviour:    "KeepOwnBreakBehaviour",
	ActionKeepSnipingPosition:      "KeepSnipingPosition",
	ActionKeepHealing:              "KeepHealing",
	ActionKeepWaitingForClass:      "KeepWaitingForClass",
	ActionWaitOutsideRound:         "WaitOutsideRound",
	ActionStrandedAsShipped:        "StrandedAsShipped",
}

// deliberateNothing is the set of outcomes where the shipped chain says
// nothing because something else already owns the bot. ActionStrandedAsShipped
// is not one of them: it is the same Plugin_Continue with nobody behind it.
var deliberateNothing = map[Action]bool{
	ActionKeepWalkingToFront:    true,
	ActionKeepOwnBreakBehaviour: true,
	ActionKeepSnipingPosition:   true,
	ActionKeepHealing:           true,
	ActionKeepWaitingForClass:   true,
	ActionWaitOutsideRound:      true,
}

// flagBit maps one bit of the enumeration to one field of Flags.
func flagBit(bits uint32) Flags {
	return Flags{
		MoneyToCollect:     bits&1 != 0,
		InUpgradeZone:      bits&(1<<1) != 0,
		ShoppedThisBreak:   bits&(1<<2) != 0,
		MovingToFront:      bits&(1<<3) != 0,
		UpgradesEnabled:    bits&(1<<4) != 0,
		HasUpgraded:        bits&(1<<5) != 0,
		UpgradeMidRound:    bits&(1<<6) != 0,
		HasSniperRifle:     bits&(1<<7) != 0,
		SniperStalled:      bits&(1<<8) != 0,
		AttackTargetFound:  bits&(1<<9) != 0,
		TankTargetFound:    bits&(1<<10) != 0,
		GiantToMark:        bits&(1<<11) != 0,
		NearbyMoney:        bits&(1<<12) != 0,
		StickyTrapPossible: bits&(1<<13) != 0,
	}
}

var flagNames = []string{
	"MoneyToCollect", "InUpgradeZone", "ShoppedThisBreak", "MovingToFront",
	"UpgradesEnabled", "HasUpgraded", "UpgradeMidRound", "HasSniperRifle",
	"SniperStalled", "AttackTargetFound", "TankTargetFound", "GiantToMark",
	"NearbyMoney", "StickyTrapPossible",
}

// combination names one point of the domain, so a failure reads as the input
// somebody can go and reproduce rather than as a count.
type combination struct {
	state RoundState
	class Class
	bits  uint32
}

func (c combination) String() string {
	set := make([]string, 0, flagCount)
	for i := range flagCount {
		if c.bits&(1<<uint32(i)) != 0 {
			set = append(set, flagNames[i])
		}
	}
	on := "no flags set"
	if len(set) > 0 {
		on = strings.Join(set, "+")
	}
	return fmt.Sprintf("%s / %s / %s", roundStateNames[c.state], classNames[c.class], on)
}

func (c combination) flags() Flags { return flagBit(c.bits) }

// reachable walks every combination the engine can produce.
func reachable(yield func(combination) bool) {
	for _, state := range roundStates {
		for _, class := range classes {
			for bits := range uint32(1) << flagCount {
				c := combination{state: state, class: class, bits: bits}
				if !Reachable(class, c.flags()) {
					continue
				}
				if !yield(c) {
					return
				}
			}
		}
	}
}

func name(a Action) string {
	if n, ok := actionNames[a]; ok {
		return n
	}
	return fmt.Sprintf("an action with no name, %d", int32(a))
}
