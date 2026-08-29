package x

// Fill writes through an array parameter. Go copies it in and SourcePawn passes
// it by reference, so the two disagree about what the caller sees.
func Fill(slots [4]int32) int32 {
	slots[0] = 1
	return slots[0]
}
