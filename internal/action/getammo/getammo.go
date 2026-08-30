/*
Package getammo is source/redbots3/behavior/getammo.sp.

Walking to ammo, and the twin of the health walk in every respect that matters.

ComputeHealthAndAmmoVectors and its sort stay hand-written next door: the search
reads a table of class names and hands the sort a function, and the generator has
neither yet. mvm-z83 carries that gap.

//sp:action DefenderGetAmmo CTFBotGetAmmo
*/
package getammo

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

//sp:name m_iAmmoPack
var ammoPack [Slots]int32

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
	candidates [Slots][candidatesMax]int32
	//sp:name m_iAmmoCandidateCount
	candidateCount [Slots]int32
	//sp:name m_iAmmoCandidate
	candidate [Slots]int32
	//sp:name m_iAmmoRepathFails
	repathFails [Slots]int32
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
	ammoAsk [Slots]float32
	//sp:name m_bAmmoPossible
	ammoPossible [Slots]bool
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
