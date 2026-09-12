/*
Package medicnudge is the medic's two nudges out of
source/redbots3/nextbot_behavior.sp: which body the game's own heal action
points at, and when the charge and the resistance are worth changing.

The mod used to answer the first by replacing the whole action, which cost the
medic his walking and most of his output. The action is left alone now and only
its patient is written.
*/
package medicnudge

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

/*
MedicPatientInterval is how often the game is told who its patient should be.

Every frame would be arguing with the action rather than nudging it, and the
beam does not need an answer more often than the team changes shape.
*/
//
//sp:name MEDIC_PATIENT_INTERVAL
const MedicPatientInterval = 2.0

//sp:name m_ctNextPatientNudge
var nextPatientNudge [slots.Count]float32

// The man a releasing charge started on, so the rest of it goes into him.
//
//sp:name m_iUberPatient
var uberPatient [slots.Count]int32

/*
Who the ranking last chose, which is not who the beam is on.

The two come apart and the gap is the interesting part. BiggestBody answers who
the medigun is worth the most on; m_hHealingTarget says who it reached, and
reaching wants line of sight and 450 units. A run where the two disagree is a
medic pointed at the right man and stood in the wrong place, which is the half of
mvm-eil still open, and reading the beam alone cannot tell that from a ranking
that picked somebody else.
*/
//
//sp:name m_iWantedPatient
var wantedPatient [slots.Count]int32

// WantedPatient is the ranking's own answer, for the statistics plugin.
//
//sp:name WantedPatient
func WantedPatient(client int32) int32 {
	return wantedPatient[client]
}

/*
HoldTheUberPatient keeps the beam where the charge started, and says it did.

A charge is spent on one body. The nudge asks every couple of seconds who the
medigun is worth the most on, and a giant walking past is enough to move the beam
mid-uber: the rest of the charge goes into somebody who was not in the fight it
was spent on, and the man who was keeps the half he already had. So while the
charge is releasing the ranking is not asked at all, and the patient is written
back if anything else has moved it.

The man it started on is latched here rather than at the button, because Valve's
own panic rule deploys too and a charge this mod did not press is still a charge.
His dying ends the hold: a live charge with a dead patient is worth pointing at
somebody, and the ranking is the thing that knows who.
*/
//
//sp:name HoldTheUberPatient
func HoldTheUberPatient(action engine.Behaviour, actor int32) bool {
	medigun := engine.PlayerWeaponSlot(actor, engine.WeaponSlotSecondary())

	if !engine.IsChargeReleasing(medigun) {
		uberPatient[actor] = -1

		return false
	}

	if uberPatient[actor] <= 0 {
		uberPatient[actor] = engine.EntPropEnt(medigun, engine.PropSend(), "m_hHealingTarget")
	}

	want := uberPatient[actor]

	/* Everything SetHandleEntity was promised, asked again here

	The write into the action's own field is the one operation in this mod that has faulted a
	server, and what makes it safe is that the value is a checked, living, same team, non medic
	client. BiggestBody returns one by construction; m_hHealingTarget is the game's handle and is
	only a player by convention, so the same questions are asked of it rather than assumed. A man
	who fails them ends the hold, which puts the ranking back in charge of a live charge. */
	if !engine.IsValidClientIndex(want) || !engine.IsPlayerAlive(want) {
		return false
	}

	if engine.GetClientTeam(want) != engine.GetClientTeam(actor) || engine.PlayerClass(want) == engine.ClassMedic() {
		return false
	}

	if action.HandleEntity(engine.ActionHealPatientOffset()) != want {
		action.SetHandleEntity(engine.ActionHealPatientOffset(), want)
	}

	return true
}

// ResetMedicNudge forgets the clock and the charge's patient.
//
// A bot leaving takes its seat's state with it, and the next bot in that seat is
// a different bot.
//
//sp:name Go_ResetMedicNudge
func ResetMedicNudge(client int32) {
	nextPatientNudge[client] = 0.0
	uberPatient[client] = -1
	wantedPatient[client] = -1
}

/*
PointMedicAtBiggestBody writes the patient handle from inside the action's own
callback.

An earlier attempt at writing this field segfaulted the server, and this is
deliberately narrower: it runs only from CTFBotMedicHeal_UpdatePost, so the
action being written is the action the game is running; the same offset is read
in the same callback every frame and has never faulted; and it is only written
when it differs from what is already there.
*/
//
//sp:name PointMedicAtBiggestBody
func PointMedicAtBiggestBody(action engine.Behaviour, actor int32) {
	if HoldTheUberPatient(action, actor) {
		return
	}

	if nextPatientNudge[actor] > engine.GameTime() {
		return
	}

	nextPatientNudge[actor] = engine.GameTime() + MedicPatientInterval

	have := action.HandleEntity(engine.ActionHealPatientOffset())
	want := engine.BiggestBodyFor(actor, have)

	wantedPatient[actor] = want

	if want <= 0 {
		return
	}

	if have == want {
		return
	}

	action.SetHandleEntity(engine.ActionHealPatientOffset(), want)
}

/*
MedicUberAndResist deploys the charge and turns the vaccinator to whatever last
hurt the patient.

The resistance is reloaded rather than chosen: the vaccinator cycles, so asking
for the wrong one is a press that lands on the right one soon enough.
*/
//
//sp:name MedicUberAndResist
//
//nolint:gocritic // ifElseChain: three bit tests are not a switch, and the shipped file writes them out
func MedicUberAndResist(actor int32, medigun int32, patient int32) {
	engine.MedicProjectileShield(actor, patient)

	if engine.ShouldDeployUber(actor, medigun, patient) {
		engine.PressAltFireButton(actor)
	}

	if patient <= 0 || engine.MedigunType(medigun) != engine.MedigunResist() {
		return
	}

	iResistType := engine.ResistType(medigun)
	iLastDmgType := engine.LastDamageType(patient)

	if iLastDmgType&engine.DamageBullet() != 0 && iResistType != engine.MedigunBulletResist() {
		engine.ExtraButtonsOf(actor).PressButtonsNow(engine.InReload())
	} else if iLastDmgType&engine.DamageBlast() != 0 && iResistType != engine.MedigunBlastResist() {
		engine.ExtraButtonsOf(actor).PressButtonsNow(engine.InReload())
	} else if iLastDmgType&engine.DamageBurn() != 0 && iResistType != engine.MedigunFireResist() {
		engine.ExtraButtonsOf(actor).PressButtonsNow(engine.InReload())
	}
}
