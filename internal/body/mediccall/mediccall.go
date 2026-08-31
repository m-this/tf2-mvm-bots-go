/*
Package mediccall is the medic's ranking out of
source/redbots3/nextbot_behavior.sp: who called, and which teammate the medigun
is worth the most on.
*/
package mediccall

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

/*
MedicPatientMargin is how much more maximum health a rival body needs before the
beam moves for arithmetic alone.

Mid wave several bodies sit within a few points of each other, so an unmargined
winner flips between them every time it is asked.
*/
//
//sp:name MEDIC_PATIENT_MARGIN
const MedicPatientMargin = 25

// MedicCallAnswerTime is how long a call keeps its weight.
//
//sp:name MEDIC_CALL_ANSWER_TIME
const MedicCallAnswerTime = 10.0

//sp:name m_ctMedicCalled
var medicCalled [Slots]float32

// NoteMedicCall starts the clock a call runs down.
//
//sp:name NoteMedicCall
func NoteMedicCall(client int32) {
	medicCalled[client] = engine.GameTime() + MedicCallAnswerTime
}

// ForgetMedicCall ends it, which a death does.
//
//sp:name ForgetMedicCall
func ForgetMedicCall(client int32) {
	medicCalled[client] = 0.0
}

// IsCallingForMedic says the call is still running.
//
//sp:name IsCallingForMedic
func IsCallingForMedic(client int32) bool {
	return medicCalled[client] > engine.GameTime()
}

/*
BiggestBody is which teammate a medigun is worth the most on.

A medigun is worth what the body in front of it is worth, so it belongs on the
biggest one: the Heavy, and failing that whoever has the most health to work
with. Maximum health rather than a class table, because that follows the health
upgrades the team buys without anybody keeping a list up to date.

A player outranks every body, and a player who called outranks a player who did
not; ranked rather than special-cased, so the tie-break at the bottom still
applies and the beam does not flicker between two players who both called. See
mvm-w9b.

Where anybody is standing is deliberately not in this: the last ranking had a
"nearby wins outright" bucket and that bucket was a fixed point. The walking is
the game's job again. This only has to answer who.

A patient he already has keeps the beam unless somebody is plainly worth more:
a switch costs the walk to the new one and the healing that is not happening
during it, so a tie keeps the man he has, and neither half of the ask can churn.
Whether somebody is a player does not flip, and a call runs down a clock and
does not come back on its own.
*/
//
//sp:name BiggestBody
//sp:default current -1
func BiggestBody(medic int32, current int32) int32 {
	best := int32(-1)
	bestHealth := int32(0)
	bestIsHeavy := false
	bestIsPlayer := false
	bestIsCalling := false
	currentHealth := int32(0)
	currentIsHeavy := false
	currentIsPlayer := false
	currentIsCalling := false
	currentStands := false

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if i == medic || !engine.IsClientInGame(i) || !engine.IsPlayerAlive(i) {
			continue
		}

		if engine.GetClientTeam(i) != engine.GetClientTeam(medic) {
			continue
		}

		// A medic healing a medic is two classes doing nothing.
		if engine.PlayerClass(i) == engine.ClassMedic() {
			continue
		}

		isHeavy := engine.PlayerClass(i) == engine.ClassHeavyweapons()
		health := engine.EntityMaxHealth(i)

		isPlayer := engine.Feature(engine.FeatureMedicAnswersCall()) && !engine.IsTFBotPlayer(i)
		isCalling := isPlayer && IsCallingForMedic(i)

		if i == current {
			currentStands = true
			currentHealth = health
			currentIsHeavy = isHeavy
			currentIsPlayer = isPlayer
			currentIsCalling = isCalling
		}

		better := best <= 0 ||
			(isCalling && !bestIsCalling) ||
			(isCalling == bestIsCalling && isPlayer && !bestIsPlayer) ||
			(isCalling == bestIsCalling && isPlayer == bestIsPlayer && isHeavy && !bestIsHeavy) ||
			(isCalling == bestIsCalling && isPlayer == bestIsPlayer && isHeavy == bestIsHeavy && health > bestHealth)

		if better {
			best = i
			bestHealth = health
			bestIsHeavy = isHeavy
			bestIsPlayer = isPlayer
			bestIsCalling = isCalling
		}
	}

	if currentStands && best > 0 && best != current {
		answersCall := bestIsCalling && !currentIsCalling
		betterSeat := bestIsCalling == currentIsCalling && bestIsPlayer && !currentIsPlayer
		betterClass := bestIsCalling == currentIsCalling && bestIsPlayer == currentIsPlayer &&
			bestIsHeavy && !currentIsHeavy
		betterBody := bestIsCalling == currentIsCalling && bestIsPlayer == currentIsPlayer &&
			bestIsHeavy == currentIsHeavy && bestHealth > currentHealth+MedicPatientMargin

		if !answersCall && !betterSeat && !betterClass && !betterBody {
			return current
		}
	}

	return best
}
