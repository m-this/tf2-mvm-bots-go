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
	Fill(&c)
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
func ClientModel(client int32) (model Text) { return missions.ClientModel(client) }

// WaveCount is which wave the mission is on.
//
//sp:library TF2_GetMannVsMachineWaveCount
func WaveCount(resource int32) int32 { return missions.WaveCount(resource) }

// MaxWaveCount is how many the mission has.
//
//sp:library TF2_GetMannVsMachineMaxWaveCount
func MaxWaveCount(resource int32) int32 { return missions.MaxWaveCount(resource) }

// WaveClassName is one of the icons on the wave bar, which is how the mod knows
// what is coming before it arrives.
//
//sp:library TF2_GetMannVsMachineWaveClassName sized
func WaveClassName(resource int32, index int32) (icon Text) {
	return missions.WaveClassName(resource, index)
}

// IsTankWave says a tank is coming, which decides which of the map's two nest
// lists applies.
//
//sp:body IsTankWave
func IsTankWave() bool { return missions.IsTankWave() }
