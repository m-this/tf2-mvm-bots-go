/*
Package soundhook is the sound hook out of source/tf2_defenderbots.sp.

Robots have robotic voices even when disguised, so a spy who laughs gives
himself away to any defender bot near enough with a clear line to him.
*/
package soundhook

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Players is MAXPLAYERS, the width of the recipient list the engine hands the
// hook.
const Players = 101

/*
	TimerRealizeSpy is the delay between hearing the laugh and acting on it

Both sides are userids rather than client indices, because either can have left
in the meantime and a userid that no longer maps to anybody answers zero.
*/
//
//sp:name Timer_RealizeSpy
//sp:public
//nolint:revive // unused-parameter: the handle is the timer's own, and the pack is what carries the two players
func TimerRealizeSpy(timer engine.Timer, pack engine.DataPack) {
	client := engine.ClientOfUserID(pack.ReadCell())

	if client == 0 {
		return
	}

	threat := engine.ClientOfUserID(pack.ReadCell())

	if threat == 0 {
		return
	}

	engine.NoticeThreat(client, threat)
}

// SoundHookGeneral watches every sound the server plays for a spy's laugh.
//
//sp:name SoundHook_General
//sp:public
//sp:dim clients MAXPLAYERS
//sp:dim sample PLATFORM_MAX_PATH
//sp:dim soundEntry PLATFORM_MAX_PATH
//sp:byref numClients
//sp:byref entity
//sp:byref channel
//sp:byref volume
//sp:byref level
//sp:byref pitch
//sp:byref flags
//sp:byref seed
//sp:writable sample
//sp:writable soundEntry
//nolint:revive // unused-parameter: a sound hook is handed the whole call and reads four of it
func SoundHookGeneral(clients [Players]int32, numClients int32, sample engine.Text, entity int32, channel int32, volume float32, level int32, pitch int32, flags int32, soundEntry engine.Text, seed int32) engine.Outcome {
	if channel == engine.SoundChannelVoice() && volume > 0.0 && engine.IsPlayer(entity) {
		if engine.StrContains(sample, "spy_mvm_LaughShort", false) != -1 {
			if engine.IsPlayerInCondition(entity, engine.ConditionDisguised()) && !engine.IsStealthed(entity) {
				/* Robots have robotic voices even when disguised so any
				defender bot that can see him right now will call him out */
				for i := int32(1); i <= engine.MaxClients(); i++ {
					if i == entity {
						continue
					}

					if !engine.IsClientInGame(i) {
						continue
					}

					if !engine.DefenderBotFlag(i) {
						continue
					}

					if engine.GetClientTeam(entity) == engine.GetClientTeam(i) {
						continue
					}

					if engine.VectorDistance(engine.AbsOriginOf(i), engine.WorldSpaceCenter(entity)) > engine.HearSpyRange().Float() {
						continue
					}

					if engine.IsLineOfFireClearEntity(i, engine.EyePosition(i), entity) {
						var pack engine.DataPack
						engine.CreateDataTimerWith(engine.NoticeSpyTime().Float(), TimerRealizeSpy, pack, engine.TimerNoMapChange())
						pack.WriteCell(engine.ClientUserID(i))
						pack.WriteCell(engine.ClientUserID(entity))
						pack.Reset()
					}
				}
			}
		}
	}

	return engine.PluginContinue()
}
