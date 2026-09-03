package engine

/*
Hearing a spy, and remembering him for a moment before acting on it.

A disguised spy still laughs in a robot's voice. Any defender bot near enough
with a clear line to him hears that and calls him out, after a delay that stands
in for the second it takes a person to place the sound.
*/

// SoundHookCalls are the answers.
type SoundHookCalls struct {
	CreateDataTimerWith func(interval float32, pack DataPack, flags int32) Timer
	WriteCell           func(pack DataPack, value int32)
	ReadCell            func(pack DataPack) int32
	ResetPack           func(pack DataPack)
}

var soundHooks SoundHookCalls

// InstallSoundHooks puts a set of answers behind them.
func InstallSoundHooks(c SoundHookCalls) func() {
	previous := soundHooks
	Fill(&c)
	soundHooks = c
	return func() { soundHooks = previous }
}

// DataPack is SourceMod's cell buffer, which a timer carries to its callback.
// The timer owns it, so nothing here closes it.
//
//sp:tag DataPack
type DataPack int32

/*
CreateDataTimerWith is the timer that carries a pack.

The pack is written through the third argument rather than returned: SourceMod
hands the caller a fresh one and keeps the timer handle for itself. Passing a
DataPack local by value emits its name, which is what SourcePawn wants there.

//sp:native CreateDataTimer
*/
//
//nolint:revive // unused-parameter: the callback is a name the emitter writes, not something the Go calls
func CreateDataTimerWith(interval float32, callback func(timer Timer, pack DataPack), pack DataPack, flags int32) Timer {
	return soundHooks.CreateDataTimerWith(interval, pack, flags)
}

// WriteCell puts one cell on the end of the pack.
//
//sp:method WriteCell
func (p DataPack) WriteCell(value int32) { soundHooks.WriteCell(p, value) }

// ReadCell takes the next cell off it.
//
//sp:method ReadCell
func (p DataPack) ReadCell() int32 { return soundHooks.ReadCell(p) }

// Reset puts the read position back to the start, which is what makes the
// cells the writer just wrote readable by the callback.
//
//sp:method Reset
func (p DataPack) Reset() { soundHooks.ResetPack(p) }

// SoundChannelVoice is SNDCHAN_VOICE, the channel a player's own lines play on.
//
//sp:global SNDCHAN_VOICE
func SoundChannelVoice() int32 { return 0 }

// HearSpyRange is redbots_manager_bot_hear_spy_range, how far a laugh carries.
//
//sp:global redbots_manager_bot_hear_spy_range
func HearSpyRange() ConVar { return 0 }

// NoticeSpyTime is redbots_manager_bot_notice_spy_time, the reaction delay.
//
//sp:global redbots_manager_bot_notice_spy_time
func NoticeSpyTime() ConVar { return 0 }
