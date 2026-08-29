package threat

import "github.com/m-this/tf2-mvm-bots-go/internal/tf"

/*
	Ranges is every distance that can change an answer

Six of them, not a sweep of every float32. The decision reads the range through
exactly two comparisons, so the only values that can tell one answer from another
are the two boundaries and a point either side of each. A million distances in
the middle of a band would cost time and prove what these six prove.

The boundaries are included on purpose, because that is where a port goes wrong:
the plugin's test is strictly less than for urgent and strictly greater than for
out of range, so a range exactly on a boundary is in the band above it and a
port that wrote <= would answer differently only there.
*/
func Ranges() []float32 {
	urgent := UrgentRange * UrgentRange
	far := PriorityRange * PriorityRange
	return []float32{
		0,
		urgent - 1,
		urgent,
		(urgent + far) / 2,
		far,
		far + 1,
	}
}

// Threats yields every combination the decision can be asked about: every
// range above, both answers to each of the four questions, and every class.
func Threats(yield func(Threat) bool) {
	for _, rangeSq := range Ranges() {
		for _, isPlayer := range []bool{false, true} {
			for _, inGame := range []bool{false, true} {
				for _, class := range tf.Classes() {
					for _, giant := range []bool{false, true} {
						for _, carrier := range []bool{false, true} {
							t := Threat{
								RangeSquared: rangeSq, IsPlayer: isPlayer, InGame: inGame,
								Class: class, Giant: giant, Carrier: carrier,
							}
							if !yield(t) {
								return
							}
						}
					}
				}
			}
		}
	}
}

// DomainSize is how many combinations Threats yields, written down so a test
// can say the sweep ran rather than trusting that it did.
var DomainSize = len(Ranges()) * 2 * 2 * int(tf.NumClasses) * 2 * 2
