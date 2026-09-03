/*
Package getammo is source/redbots3/behavior/getammo.sp.

Walking to ammo, and the twin of the health walk in every respect that matters.

ComputeHealthAndAmmoVectors and its sort stay hand-written next door: the search
reads a table of class names and hands the sort a function, and the generator has
neither yet. mvm-z83 carries that gap.

//sp:action DefenderGetAmmo CTFBotGetAmmo
*/
package getammo

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

//sp:name m_iAmmoPack
var ammoPack [slots.Count]int32

/*
The other packs he could have had, kept so a refused path is not the end of the walk

The choice was validated once, in OnStart, and then held for the life of the
action. Everything after that repathed to the same entity every second and threw
the answer away, so a bot whose route stopped existing walked along an empty path,
at 120 units a nudge, until the pack expired. That is the "does not know what a
wall is, then gives up" in the report: not a goal picked without a path, but a
goal that stopped having one and nothing watching.

So the ranked list survives OnStart. Three consecutive refusals is the point where
the route is not coming back, and the next candidate is tried. Running out ends the
action rather than leaving him walking at nothing, and holds the gate shut long
enough that the monitor does not send him straight back in.
*/
const (
	//sp:name AMMO_CANDIDATES_MAX
	candidatesMax = 4
	//sp:name AMMO_REPATH_FAILS_MAX
	repathFailsMax = 3
	//sp:name AMMO_GIVEUP_TIME
	giveUpTime = 3.0
)

var (
	//sp:name m_arrAmmoCandidates
	candidates [slots.Count][candidatesMax]int32
	//sp:name m_iAmmoCandidateCount
	candidateCount [slots.Count]int32
	//sp:name m_iAmmoCandidate
	candidate [slots.Count]int32
	//sp:name m_iAmmoRepathFails
	repathFails [slots.Count]int32
)

// OnStart ranks the packs in range and takes the nearest.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	// Nothing unless a debug convar is set, which is never on a real server
	engine.OnAmmoWalkStart(actor)

	ammo := engine.NewBlocks(2)
	defer ammo.Close()

	engine.ComputeHealthAndAmmoVectors(actor, ammo, engine.AmmoSearchRange().Float())

	ammoPack[actor] = -1
	candidateCount[actor] = 0
	candidate[actor] = 0
	repathFails[actor] = 0

	// Shortest travel first, so a failover walks outwards rather than anywhere
	for candidateCount[actor] < candidatesMax {
		best := int32(-1)
		flSmallestDistance := float32(0.0)

		for i := int32(0); i < ammo.Length(); i++ {
			entity := ammo.GetAt(i, 0)

			if entity == -1 || !IsValidAmmo(entity) {
				continue
			}

			flDistance := engine.AsFloat(ammo.GetAt(i, 1))

			if best == -1 || flDistance < flSmallestDistance {
				best = i
				flSmallestDistance = flDistance
			}
		}

		if best == -1 {
			break
		}

		candidates[actor][candidateCount[actor]] = ammo.GetAt(best, 0)
		candidateCount[actor]++
		ammo.SetAt(best, -1, 0)
	}

	if candidateCount[actor] > 0 {
		ammoPack[actor] = candidates[actor][0]

		if engine.PlayerClass(actor) == engine.ClassEngineer() {
			engine.UpdateLookAroundForEnemies(actor, true)
		}

		engine.SpeakConceptIfAllowed(actor, engine.ConceptDispenserHere())
		return engine.Continue()
	}

	return engine.Done("Could not find ammo")
}

// Update walks to the pack, and to the next one along when the route to this one
// stops existing.
func Update(actor int32) engine.Outcome {
	if !IsValidAmmo(ammoPack[actor]) {
		return engine.Done("ammo is not valid")
	}

	if engine.IsAmmoFull(actor) {
		return engine.Done("Ammo is full")
	}

	myBot := engine.NextBotOf(actor)

	if engine.RepathTime(actor) <= engine.GameTime() {
		engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(0.9, 1.0))
		engine.RepathToPos(actor, myBot, engine.WorldSpaceCenter(ammoPack[actor]))

		if engine.Feature(engine.FeatureAmmoFailover()) {
			// The return value is the only thing that says the route failed. The length lies.
			if !engine.RefuseAmmoPath(actor) && !engine.PathFailedFor(actor) {
				repathFails[actor] = 0
			} else {
				repathFails[actor]++

				if repathFails[actor] >= repathFailsMax {
					repathFails[actor] = 0

					if !NextCandidate(actor) {
						HoldOff(actor)
						return engine.Done("No reachable ammo")
					}

					engine.RepathToPos(actor, myBot, engine.WorldSpaceCenter(ammoPack[actor]))
				}
			}
		}
	}

	engine.PathOf(actor).Update(myBot)

	threat := myBot.Vision().PrimaryKnownThreat(false)

	if threat != 0 {
		engine.EquipBestWeaponForThreat(actor, threat)
	}

	return engine.Continue()
}

// OnEnd forgets the pack and the ranking behind it.
func OnEnd(actor int32) {
	ammoPack[actor] = -1
	candidateCount[actor] = 0
	candidate[actor] = 0
	repathFails[actor] = 0
}

// NextCandidate is the next pack he was ranked onto, skipping any taken while he
// walked. Bounded by the list.
//
//sp:name NextAmmoCandidate
func NextCandidate(actor int32) bool {
	for candidate[actor]++; candidate[actor] < candidateCount[actor]; candidate[actor]++ {
		pack := candidates[actor][candidate[actor]]

		if !IsValidAmmo(pack) {
			continue
		}

		ammoPack[actor] = pack
		return true
	}

	return false
}

// ShouldHurry disables dodging, and keeps the minigun unspun after recently
// seeing threats.
//
//nolint:revive // unused-parameter: the signature is the engine's, not ours
func ShouldHurry(nextbot engine.Bot) (changed engine.Outcome, result engine.Answer) {
	return engine.PluginHandled(), engine.AnswerYes()
}

// ShouldAttack keeps a spy walking for ammo out of a fight it cannot win.
//
//nolint:revive // unused-parameter: the signature is the engine's, not ours
func ShouldAttack(nextbot engine.Bot, knownEntity engine.Known) (changed engine.Outcome, result engine.Answer) {
	me := engine.Actor()

	if engine.PlayerClass(me) == engine.ClassSpy() {
		iThreat := knownEntity.Entity()

		if engine.IsPlayer(iThreat) && engine.ClientHealth(iThreat) > 360 && !engine.IsCritBoosted(me) {
			// Don't attack if we can't possibly kill them with our revolver (360 from 6 shots with max damage)
			return engine.Changed(), engine.AnswerNo()
		} else if engine.NearestEnemyCount(me, 1000.0, false) > 1 {
			// There's too many enemies nearby, it'd be better to redisguise so they'll forget about us
			return engine.Changed(), engine.AnswerNo()
		}
	}

	return engine.Changed(), engine.AnswerUndefined()
}

// IsValidAmmo says the entity is ammo the bot could still take.
//
//sp:name IsValidAmmo
func IsValidAmmo(pack int32) bool {
	if !engine.IsValidEntity(pack) {
		return false
	}

	if !engine.HasEntProp(pack, engine.PropSend(), "m_fEffects") {
		return false
	}

	// It has been taken.
	if engine.EntProp(pack, engine.PropSend(), "m_fEffects") != 0 {
		return false
	}

	class := engine.EntityClassname(pack)

	if engine.StrContains(class, "tf_ammo_pack", false) == -1 &&
		engine.StrContains(class, "item_ammo", false) == -1 &&
		engine.StrContains(class, "obj_dispenser", false) == -1 &&
		engine.StrContains(class, "func_regen", false) == -1 {
		return false
	}

	// Can't use a disabled dispenser
	if engine.StrContains(class, "obj_dispenser", false) != -1 && engine.HasSapper(pack) {
		return false
	}

	return true
}

/*
How long an answer about ammo is kept for.

The tactical monitor asks this every frame, for every bot, and a bot that is low
on ammo with nothing reachable takes the slow path on every one of those frames.
The slow path is a nav mesh search. Six bots doing that at sixty-six frames a
second, on a floor covered in what the dead robots dropped, is thousands of
searches a second for an answer that was no last frame and is no this frame.

Half a second, which is a bot walking about a hundred and fifty units. Nothing
that matters appears inside that.
*/
//
//sp:name AMMO_ASK_INTERVAL
const askInterval = 0.5

var (
	//sp:name m_ctAmmoAsk
	ammoAsk [slots.Count]float32
	//sp:name m_bAmmoPossible
	ammoPossible [slots.Count]bool
)

/*
HoldOff keeps the gate shut after a walk that ran out of reachable packs.

The cache answers from a nav search that said yes, and the walk that followed
said no. Without this the monitor re-enters the action on the next frame with the
same candidates and the bot spends the wave starting and abandoning it.
*/
//
//sp:name HoldOffAmmo
func HoldOff(actor int32) {
	ammoAsk[actor] = engine.GameTime() + giveUpTime
	ammoPossible[actor] = false
}

// IsPossible says whether there is ammo worth walking to.
//
//sp:name CTFBotGetAmmo_IsPossible
func IsPossible(actor int32) bool {
	// Skip lag.
	if ammoPack[actor] != -1 && IsValidAmmo(ammoPack[actor]) {
		return true
	}

	if ammoAsk[actor] > engine.GameTime() {
		return ammoPossible[actor]
	}

	ammoAsk[actor] = engine.GameTime() + askInterval

	ammo := engine.NewBlocks(2)
	defer ammo.Close()

	engine.ComputeHealthAndAmmoVectors(actor, ammo, engine.AmmoSearchRange().Float())

	bPossible := false

	for i := int32(0); i < ammo.Length(); i++ {
		if !IsValidAmmo(ammo.GetAt(i, 0)) {
			continue
		}

		bPossible = true
		break
	}

	ammoPossible[actor] = bPossible

	return bPossible
}

/*
The classnames a health or ammo search walks.

A table of names rather than ids: FindEntityByClassname wants a real string, and
two of these carry a wildcard the schema has no id for.
*/
//
//sp:name g_strHealthAndAmmoEntities
var healthAndAmmoEntities = [5]string{
	"func_regenerate",
	"item_ammopack*",
	"item_health*",
	"obj_dispenser",
	"tf_ammo_pack",
}

/*
The health and ammo this bot could actually walk to, and how far each one really is

Two costs hide in this and both were paid per candidate: a nav mesh search, and a
JSON object on the heap. MvM floors are covered in candidates, because tf_ammo_pack
is what a dead robot leaves behind and a wave leaves hundreds of them.

So the cheap question goes first. Straight-line distance costs a subtraction and
orders the candidates; the search is run only for the nearest few, because the
nearest few are where the answer is and the rest were never going to win. A pack
behind a wall now loses its place to the next one along instead of costing a search
of its own.

The list is entity index and travel distance, in pairs, and the caller takes the
shortest.
*/
const (
	//sp:name HEALTH_CANDIDATES_MAX
	candidatesSeen = 64
	//sp:name HEALTH_PATHS_MAX
	pathsMax = candidatesMax
)

// ComputeVectors fills the list with what is in range, nearest first.
//
//sp:name ComputeHealthAndAmmoVectors
func ComputeVectors(client int32, found engine.List, maxRange float32) {
	nearby := engine.NewBlocks(2)
	defer nearby.Close()

	myCentre := engine.WorldSpaceCenter(client)

	for i := int32(0); i < int32(len(healthAndAmmoEntities)); i++ {
		ammo := int32(-1)

		for {
			ammo = engine.FindEntityByClassname(ammo, healthAndAmmoEntities[i])

			if ammo == -1 {
				break
			}

			// A wave leaves more of these on the floor than anybody is going to walk to
			if nearby.Length() >= candidatesSeen {
				break
			}

			if engine.EntityTeamNumber(ammo) == int32(engine.PlayerEnemyTeam(client)) {
				continue
			}

			entityRange := engine.VectorDistance(myCentre, engine.WorldSpaceCenter(ammo))

			if entityRange > maxRange {
				continue
			}

			if engine.IsBaseObject(ammo) {
				// Can't get anything from still building buildings.
				if engine.IsBuildingUp(ammo) {
					continue
				}

				// Skip empty dispenser.
				if engine.ObjectType(ammo) == engine.ObjectDispenser() && engine.EntProp(ammo, engine.PropSend(), "m_iAmmoMetal") <= 0 {
					continue
				}
			}

			at := nearby.PushFloat(entityRange)
			nearby.SetAt(at, ammo, 1)
		}
	}

	nearby.SortCustom(SortByStraightLineRange)

	searches := int32(0)

	for i := int32(0); i < nearby.Length() && searches < pathsMax; i++ {
		ammo := nearby.GetAt(i, 1)

		searches++

		reachable, length := engine.IsPathToVectorPossibleLength(client, engine.WorldSpaceCenter(ammo))

		if !reachable {
			continue
		}

		if length > maxRange {
			continue
		}

		at := found.PushAt(ammo)
		found.SetFloatAt(at, length, 1)
	}
}

// SortByStraightLineRange is the cheap ordering the search runs before it spends
// a nav mesh query on anything.
//
//sp:name SortByStraightLineRange
//nolint:revive // unused-parameter: the signature is SourceMod's, not ours
func SortByStraightLineRange(index1 int32, index2 int32, array engine.Handle, hndl engine.Handle) int32 {
	list := engine.ListOf(array)

	first := engine.AsFloat(list.GetAt(index1, 0))
	second := engine.AsFloat(list.GetAt(index2, 0))

	if first < second {
		return -1
	}

	return engine.ChooseInt(first > second, 1, 0)
}

// ResetGetAmmo forgets the ammo pack this bot was walking to.
//
// A bot leaving takes its seat's state with it, and the next bot in that seat
// is a different bot.
func ResetGetAmmo(client int32) {
	ammoPack[client] = -1
}
