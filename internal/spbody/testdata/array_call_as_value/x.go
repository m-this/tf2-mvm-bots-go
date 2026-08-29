package x

// Origin returns an array, which SourcePawn fills through a parameter.
func Origin(entity int32) (origin [3]float32) {
	origin[0] = float32(entity)
	return origin
}

// First uses the call as a value, and there is no value: the SourcePawn form
// has no return at all.
func First(entity int32) float32 {
	return Origin(entity)[0]
}
