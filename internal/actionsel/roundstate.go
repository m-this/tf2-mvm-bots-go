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

// numRoundStates is one past the last state, so RoundStates covers the enum.
const numRoundStates = RoundBetweenRounds + 1

// RoundStates is every round state, in declared order.
func RoundStates() []RoundState {
	all := make([]RoundState, 0, numRoundStates)
	for s := RoundInit; s < numRoundStates; s++ {
		all = append(all, s)
	}
	return all
}
