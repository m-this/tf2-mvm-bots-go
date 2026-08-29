package x

// Band is a named integer with nothing to enumerate, so there is no SourcePawn
// enum to declare and the name would be lost in the output.
type Band int32

func Widen(b Band) Band { return b + 1 }
