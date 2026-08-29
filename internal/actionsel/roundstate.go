package actionsel

// RoundState is SourceMod's RoundState, in its declared order, from
// sdktools_gamerules.inc. The choice reads only two of the eleven.
type RoundState int32

// The eleven round states, in declared order.
const (
	RoundInit RoundState = iota
	RoundPregame
	RoundStartGame
	RoundPreround
	RoundRunning
	RoundTeamWin
	RoundRestart
	RoundStalemate
	RoundGameOver
	RoundBonus
	RoundBetweenRounds
)
