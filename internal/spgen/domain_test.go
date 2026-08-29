package spgen_test

import (
	"fmt"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/actionsel"
	"github.com/m-this/tf2-mvm-bots-go/internal/spgen"
)

// The domain, enumerated here rather than borrowed from internal/actionsel's
// own test, so the sweep and the thing it checks do not share a helper.

const (
	stateCount = 11
	classCount = 10
	predCount  = 14
)

// flagsOf turns a bit pattern into the struct, in the order the edge lists the
// predicates, which is the order the table's ids run in.
func flagsOf(bits uint32) actionsel.Flags {
	on := func(i uint) bool { return bits&(1<<i) != 0 }
	return actionsel.Flags{
		MoneyToCollect:     on(0),
		InUpgradeZone:      on(1),
		ShoppedThisBreak:   on(2),
		MovingToFront:      on(3),
		UpgradesEnabled:    on(4),
		HasUpgraded:        on(5),
		UpgradeMidRound:    on(6),
		HasSniperRifle:     on(7),
		SniperStalled:      on(8),
		AttackTargetFound:  on(9),
		TankTargetFound:    on(10),
		GiantToMark:        on(11),
		NearbyMoney:        on(12),
		StickyTrapPossible: on(13),
	}
}

type point struct {
	state actionsel.RoundState
	class actionsel.Class
	bits  uint32
}

func (p point) String() string {
	var set []string
	for i, pr := range spgen.ActionSelPredicates {
		if p.bits&(1<<uint(i)) != 0 {
			set = append(set, pr.Field)
		}
	}
	on := "no flags set"
	if len(set) > 0 {
		on = strings.Join(set, "+")
	}
	return fmt.Sprintf("state %d / class %d / %s", int32(p.state), int32(p.class), on)
}

func (p point) flags() actionsel.Flags { return flagsOf(p.bits) }
func (p point) axes() []int64          { return []int64{int64(p.state), int64(p.class)} }

// sweep walks the domain in exactly the order testdata/sweep.sp does, so the
// two result streams line up index for index.
func sweep(yield func(point) bool) {
	for state := range int32(stateCount) {
		for class := range int32(classCount) {
			for bits := range uint32(1) << predCount {
				p := point{state: actionsel.RoundState(state), class: actionsel.Class(class), bits: bits}
				if !actionsel.Reachable(p.class, p.flags()) {
					continue
				}
				if !yield(p) {
					return
				}
			}
		}
	}
}
