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
	if nextPatientNudge[actor] > engine.GameTime() {
		return
	}

	nextPatientNudge[actor] = engine.GameTime() + MedicPatientInterval

	have := action.HandleEntity(engine.ActionHealPatientOffset())
	want := engine.BiggestBodyFor(actor, have)

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
