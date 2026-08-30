package engine

// MissionCalls are the answers about the mission and who is in it.
type MissionCalls struct {
	ClientModel   func(client int32) Text
	WaveCount     func(resource int32) int32
	MaxWaveCount  func(resource int32) int32
	IsTankWave    func() bool
	WaveClassName func(resource int32, index int32) Text
	StrContainsIn func(haystack Text, needle Text, caseSensitive bool) int32
}

var missions MissionCalls

// InstallMissions puts a set of answers behind them.
func InstallMissions(c MissionCalls) func() {
	previous := missions
	missions = c
	return func() { missions = previous }
}

// MissionDestroySentries is CTFBot_MISSION_DESTROY_SENTRIES, which is what the
// game calls a sentry buster.
//
//sp:global CTFBot_MISSION_DESTROY_SENTRIES
func MissionDestroySentries() int32 { return 1 }

// ClientModel is the model a player is wearing, which is how a buster is told
// from a demoman when the mission field says nothing.
//
//sp:native GetClientModel sized
func ClientModel(client int32) (model Text) {
	if missions.ClientModel == nil {
		missing("GetClientModel")
	}
	return missions.ClientModel(client)
}

// WaveCount is which wave the mission is on.
//
//sp:plugin TF2_GetMannVsMachineWaveCount
func WaveCount(resource int32) int32 {
	if missions.WaveCount == nil {
		missing("TF2_GetMannVsMachineWaveCount")
	}
	return missions.WaveCount(resource)
}

// MaxWaveCount is how many the mission has.
//
//sp:plugin TF2_GetMannVsMachineMaxWaveCount
func MaxWaveCount(resource int32) int32 {
	if missions.MaxWaveCount == nil {
		missing("TF2_GetMannVsMachineMaxWaveCount")
	}
	return missions.MaxWaveCount(resource)
}

// WaveClassName is one of the icons on the wave bar, which is how the mod knows
// what is coming before it arrives.
//
//sp:plugin TF2_GetMannVsMachineWaveClassName sized
func WaveClassName(resource int32, index int32) (icon Text) {
	if missions.WaveClassName == nil {
		missing("TF2_GetMannVsMachineWaveClassName")
	}
	return missions.WaveClassName(resource, index)
}

// IsTankWave says a tank is coming, which decides which of the map's two nest
// lists applies.
//
//sp:body IsTankWave
func IsTankWave() bool {
	if missions.IsTankWave == nil {
		missing("IsTankWave")
	}
	return missions.IsTankWave()
}
