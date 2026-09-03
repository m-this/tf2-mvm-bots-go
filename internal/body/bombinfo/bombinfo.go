/*
Package bombinfo is the part of source/redbots3/util.sp that answers where the
bomb is and how far along its route the fighting has reached.

Every nest decision starts here: the bomb in play that is furthest along is the one
worth covering, and the two battle fronts are the window a bot is expected to fight
inside.
*/
package bombinfo

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// GetBombInfo fills the record, and says whether there was a bomb to fill it
// from.
//
//sp:name GetBombInfo
func GetBombInfo() (found bool, info engine.BombInfo) {
	areaCount := engine.NavAreaCount()

	if areaCount <= 0 {
		return false, info
	}

	hatchDist := float32(0.0)

	for i := int32(0); i < (areaCount - 1); i++ {
		area := engine.AllNavAreas().NavAreaAt(i)

		// Skip spawn areas
		if area.HasAttributeTF(engine.BlueSpawnRoom()) || area.HasAttributeTF(engine.RedSpawnRoom()) {
			continue
		}

		bombTargetDistance := engine.TravelDistanceToBombTarget(area)

		hatchDist = engine.MaxFloat(engine.MaxFloat(bombTargetDistance, hatchDist), 0.0)
	}

	closestFlag := engine.InvalidEntReference()

	var closestFlagPos [3]float32

	flag := int32(-1)

	for {
		flag = engine.FindEntityByClassname(flag, "item_teamflag")

		if flag == -1 {
			break
		}

		// Ignore bombs not in play
		if engine.EntProp(flag, engine.PropSend(), "m_nFlagStatus") == engine.FlagInfoHome() {
			continue
		}

		var flagPos [3]float32

		owner := engine.OwnerEntity(flag)

		if engine.IsValidClientIndex(owner) {
			flagPos = engine.AbsOriginOf(owner)
		} else {
			flagPos = engine.WorldSpaceCenter(flag)
		}

		area := engine.NavArea(engine.NearestNavAreaAt(flagPos))

		if area == engine.NavArea(engine.NullArea()) {
			continue
		}

		if area.HasAttributeTF(engine.BlueSpawnRoom()) || area.HasAttributeTF(engine.RedSpawnRoom()) {
			continue
		}

		bombTargetDistance := engine.TravelDistanceToBombTarget(area)

		if bombTargetDistance < hatchDist {
			closestFlag = flag
			hatchDist = bombTargetDistance
			closestFlagPos = flagPos
		}
	}

	rangeFwd := float32(2300.0)
	rangeBack := float32(1000.0)

	info.Position = closestFlagPos
	info.MaxBattleFront = hatchDist + rangeBack
	info.MinBattleFront = hatchDist - rangeFwd

	return closestFlag != engine.InvalidEntReference(), info
}
