package threat

// Band is which side of the two range comparisons a target falls on. The
// decision reads the range only through those two tests, so a band is
// everything about a distance that can change an answer.
type Band int32

// The three bands, in increasing distance.
const (
	// BandUrgent is inside THREAT_URGENT_RANGE, where anything outranks the
	// list.
	BandUrgent Band = iota
	// BandPriority is between the two ranges, where the order applies.
	BandPriority
	// BandTooFar is past THREAT_PRIORITY_RANGE, where the nearest one wins
	// and this decision has no opinion.
	BandTooFar
	numBands
)

// BandOf is the two comparisons the shipped function makes, and nothing else.
// Both are strict, so a range exactly on a boundary is in the band above it.
func BandOf(rangeSq float32) Band {
	if rangeSq < UrgentRange*UrgentRange {
		return BandUrgent
	}
	if rangeSq > PriorityRange*PriorityRange {
		return BandTooFar
	}
	return BandPriority
}

// Bands is every band, in declared order.
func Bands() []Band {
	all := make([]Band, 0, numBands)
	for b := BandUrgent; b < numBands; b++ {
		all = append(all, b)
	}
	return all
}

func (b Band) String() string {
	switch b {
	case BandUrgent:
		return "Urgent"
	case BandPriority:
		return "Priority"
	case BandTooFar:
		return "TooFar"
	}
	return "Band(?)"
}

// NumBands is how many bands there are, for a table that indexes on one.
const NumBands = int(numBands)
