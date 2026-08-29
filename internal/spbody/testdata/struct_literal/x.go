package x

type Pair struct {
	A int32
	B int32
}

// Make wants a struct literal, which SourcePawn has no form for.
func Make(a int32) Pair { return Pair{A: a, B: 0} }
