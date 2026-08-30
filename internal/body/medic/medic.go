/*
Package medic is what is left of source/redbots3/behavior/medicheal.sp.

It used to hold CTFBotDefenderMedicHeal: an action that took the walking, the
aim and the button away from the game's own medic behaviour so the mod could
choose the patient itself. Choosing the patient was the point and the walking was
the price, and the price turned out to be the whole medic.

The mod walked by setting a goal on esPluginBot and trusting
PluginBot_SimulateFrame. A refused path computation leaves the path object
holding the last one that worked, so the failure reads as a healthy path from
every angle, and the bot falls through to NudgeTowardsGoal and its 120 unit
steps. Measured on Decoy: the medic reported a path 10400 units long, constant to
within fifty units over eighty seconds, while his nearest teammate stood four
hundred units away. He was not walking. He was being nudged, and he never
arrived.

Against the game's own behaviour, on Coaltown:

	beam connected            5-17%   ->  61% of samples
	movement between samples  0-70    -> 337 units
	path computations failed  most    ->   0, it does not use ours

So the action is gone and the game does the healing again. The mod keeps the
parts that were actually improvements and do not touch locomotion: the uber and
resistance handling, the revive, and holding the hatch instead of fetching the
bomb when there is nobody to heal. All of those live in
CTFBotMedicHeal_UpdatePost.

What went with it is the patient ranking, which is a real loss: the game picks
whoever it likes and the mod picked the biggest body. Getting that back means
changing the game's mind about its patient rather than replacing it, and the last
attempt at that wrote into the action's own field and segfaulted the server. It
is worth another try; it is not worth this.

This package is a body rather than an action, because what is left is a console
command and not a behaviour.
*/
package medic

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
DumpMedic says where each medic is, who he is beaming and how far that is from
the bomb.

The bomb is the fight: a medic a little behind his patient is a medic doing his
job, and one much further from the bomb than the man he is healing has stopped
following anybody.

Who he is beaming is read off the medigun rather than asked of the mod, because
the mod no longer has an opinion and the medigun is the only thing that knows.
*/
//
//sp:public
//sp:name Command_DumpMedic
func DumpMedic(client int32, args int32) engine.Outcome { //nolint:revive // unused-parameter: the signature is SourceMod's
	haveBomb, bomb := engine.GetBombInfo()

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if !engine.IsClientInGame(i) || !engine.IsPlayerAlive(i) || engine.PlayerClass(i) != engine.ClassMedic() {
			continue
		}

		mine := engine.AbsOriginOf(i)

		fromBomb := float32(-1.0)

		if haveBomb {
			fromBomb = engine.VectorDistance(mine, bomb.Position)
		}

		stack := engine.ActionStackOf(i)

		medigun := engine.PlayerWeaponSlot(i, engine.WeaponSlotSecondary())
		patient := int32(-1)

		if medigun != -1 && engine.HasEntProp(medigun, engine.PropSend(), "m_hHealingTarget") {
			patient = engine.EntPropEnt(medigun, engine.PropSend(), "m_hHealingTarget")
		}

		if patient <= 0 || patient > engine.MaxClients() || !engine.IsClientInGame(patient) {
			engine.ReplyToCommand(client, "%N: beam on nobody, %.0f from the bomb, %s", i, fromBomb, stack)

			continue
		}

		theirs := engine.AbsOriginOf(patient)

		fromBombTheirs := float32(-1.0)

		if haveBomb {
			fromBombTheirs = engine.VectorDistance(theirs, bomb.Position)
		}

		engine.ReplyToCommand(client, "%N: healing %N, %.0f behind him, %.0f from the bomb, he is %.0f from it, %s",
			i, patient, engine.VectorDistance(mine, theirs), fromBomb,
			fromBombTheirs, stack)
	}

	return engine.PluginHandled()
}
