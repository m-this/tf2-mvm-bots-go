/*
Package mediccall is the medic's ranking out of
source/redbots3/nextbot_behavior.sp: who called, and which teammate the medigun
is worth the most on.
*/
package mediccall

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

/*
MedicPatientMargin is how much more a rival body has to be worth before the beam
moves for arithmetic alone.

Mid wave several bodies sit within a few points of each other, so an unmargined
winner flips between them every time it is asked.
*/
//
//sp:name MEDIC_PATIENT_MARGIN
const MedicPatientMargin = 25

/*
The three seats a teammate can hold in the queue for the beam.

Numbered rather than named booleans because the ranking compares them, and
because a seat and a worth are two different questions: the seat says who has
first claim on a medigun and the worth says what one is worth on him.
*/
const (
	//sp:name MEDIC_SEAT_BOT
	SeatBot = 0
	//sp:name MEDIC_SEAT_PLAYER
	SeatPlayer = 1
	//sp:name MEDIC_SEAT_CALLING
	SeatCalling = 2
)

/*
MedicPatientHeavyWorth is what a Heavy is worth beyond his own body, in health
points.

Outright rather than weighted: more than any maximum health the upgrade station
sells, so no amount of missing health and no distance moves the beam off him. He
is the body the medigun is carried for.

Being outright has a cost, and it is why medic_ranks_hurt and medic_ranks_range
have nothing to decide while a Heavy is alive: the seat is compared before the
worth, and he then wins every worth comparison he is in. Lowering this to about
a class gap is what would give those two a domain, and it belongs in the same
change that turns them on. On its own it is a live change to the shipped ranking
with no measurement behind it, which is what this paragraph exists to stop
somebody doing twice.
*/
//
//sp:name MEDIC_PATIENT_HEAVY_WORTH
const MedicPatientHeavyWorth = 1000

// MedicPatientRangeUnits is how far the medic walks for one health point of
// worth, and MedicPatientRangeWorst the most a walk can cost him.
const (
	//sp:name MEDIC_PATIENT_RANGE_UNITS
	MedicPatientRangeUnits = 10.0
	//sp:name MEDIC_PATIENT_RANGE_WORST
	MedicPatientRangeWorst = 200
)

// MedicCallAnswerTime is how long a call keeps its weight.
//
//sp:name MEDIC_CALL_ANSWER_TIME
const MedicCallAnswerTime = 10.0

//sp:name m_ctMedicCalled
var medicCalled [slots.Count]float32

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
MedicPatientSeat is where a teammate stands in the queue for the beam.

A seat rather than a special case, so the worth below still decides inside one
and the beam does not flicker between two players who both called. See mvm-w9b.

Asking is what lifts a person out of the queue. mvm-w9b shipped a second lift,
the player who has not asked standing above every bot, and Mathis's call is that
it is the call and not the body behind it that should move a medic: a person
happy where he is has no more claim on the beam than the Heavy walking into the
wave. Behind medic_ranks_callers_only, because it is the half of a measured
decision that is being taken back.
*/
//
//sp:name MedicPatientSeat
func MedicPatientSeat(patient int32) int32 {
	if !engine.Feature(engine.FeatureMedicAnswersCall()) || engine.IsTFBotPlayer(patient) {
		return SeatBot
	}

	if IsCallingForMedic(patient) {
		return SeatCalling
	}

	if engine.Feature(engine.FeatureMedicRanksCallersOnly()) {
		return SeatBot
	}

	return SeatPlayer
}

/*
MedicPatientWorth is what the medigun is worth on this body, in health points.

A medigun is worth what the body in front of it is worth, so the Heavy is worth
more than the arithmetic says and the rest are worth their maximum health.
Maximum health rather than a class table, because that follows the health
upgrades the team buys without anybody keeping a list up to date.

The other two terms are what the shipped ranking had no way to say, and they
carry a switch each rather than one between them: measured together they cost
robots killed and the run could not say which of them did it.

Missing health is what a medigun undoes, so a man who is down two hundred is
worth two hundred more than the same man whole, and a man already overhealed is
worth less than one who is not.

The walk is charged against the worth rather than allowed to win a bucket of its
own. The last ranking had a "nearby wins outright" bucket and that bucket was a
fixed point, so the medic never left whoever he happened to be stood beside. The
cap is what keeps it a term rather than a bucket, so distance decides between
comparable bodies and never between a Scout at the medic's feet and a Heavy
across the map.
*/
//
//sp:name MedicPatientWorth
func MedicPatientWorth(medic int32, patient int32) int32 {
	maxHealth := engine.EntityMaxHealth(patient)
	worth := maxHealth

	if engine.PlayerClass(patient) == engine.ClassHeavyweapons() {
		worth += MedicPatientHeavyWorth
	}

	if engine.Feature(engine.FeatureMedicRanksHurt()) {
		worth += maxHealth - engine.ClientHealth(patient)
	}

	if !engine.Feature(engine.FeatureMedicRanksRange()) {
		return worth
	}

	walk := engine.RoundToFloor(
		engine.VectorDistance(engine.AbsOriginOf(medic), engine.AbsOriginOf(patient)) / MedicPatientRangeUnits)

	if walk > MedicPatientRangeWorst {
		walk = MedicPatientRangeWorst
	}

	return worth - walk
}

/*
BiggestBody is which teammate a medigun is worth the most on: the best seat,
and inside the seat the best worth.

A patient he already has keeps the beam unless somebody is plainly worth more:
a switch costs the walk to the new one and the healing that is not happening
during it, so a tie keeps the man he has. A seat does not churn either, because
whether somebody is a player does not flip and a call runs down a clock and does
not come back on its own.
*/
//
//sp:name BiggestBody
//sp:default current -1
func BiggestBody(medic int32, current int32) int32 {
	best := int32(-1)
	bestSeat := int32(0)
	bestWorth := int32(0)
	currentSeat := int32(0)
	currentWorth := int32(0)
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

		seat := MedicPatientSeat(i)
		worth := MedicPatientWorth(medic, i)

		if i == current {
			currentStands = true
			currentSeat = seat
			currentWorth = worth
		}

		if best <= 0 || seat > bestSeat || (seat == bestSeat && worth > bestWorth) {
			best = i
			bestSeat = seat
			bestWorth = worth
		}
	}

	if currentStands && best > 0 && best != current &&
		bestSeat == currentSeat && bestWorth <= currentWorth+MedicPatientMargin {
		return current
	}

	return best
}
