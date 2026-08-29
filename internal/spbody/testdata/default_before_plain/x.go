package x

// Scan gives a default to a parameter that is followed by one without, which no
// SourcePawn caller could ever omit.
//
//sp:default giantsOnly false
func Scan(client int32, giantsOnly bool, team int32) int32 {
	if giantsOnly {
		return team
	}
	return client
}
