package engine

/*
Tracing where somebody is looking, so a spot can be written down.

The map config wants coordinates and the person authoring it is standing in the
map. Either their feet or their crosshair is the answer.
*/

// DumpSpotCalls are the answers.
type DumpSpotCalls struct {
	TraceRayFilterEx   func(from [3]float32, angles [3]float32, mask int32, rayType int32, data int32) TraceHandle
	DidHitTrace        func(trace TraceHandle) bool
	TraceEndPositionOf func(trace TraceHandle) [3]float32
	CloseTrace         func(trace TraceHandle)
	StrEqualCased      func(a Text, b string, caseSensitive bool) bool
}

var dumpSpots DumpSpotCalls

// InstallDumpSpots puts a set of answers behind them.
func InstallDumpSpots(c DumpSpotCalls) func() {
	previous := dumpSpots
	dumpSpots = c
	return func() { dumpSpots = previous }
}

// TraceHandle is the ray SourceMod hands back, which the caller owns.
//
//sp:tag Handle
type TraceHandle int32

// MaskSolid is MASK_SOLID, everything a ray stops at.
//
//sp:global MASK_SOLID
func MaskSolid() int32 { return 0 }

// RayTypeInfinite is RayType_Infinite, a ray with a direction and no end.
//
//sp:global RayType_Infinite
func RayTypeInfinite() int32 { return 0 }

// TraceRayFilterEx fires a ray and keeps the result, rather than leaving it in
// the one global slot the plain form writes.
//
//sp:native TR_TraceRayFilterEx
//nolint:revive // unused-parameter: the filter is a name the emitter writes, not something the Go calls
func TraceRayFilterEx(from [3]float32, angles [3]float32, mask int32, rayType int32, filter func(entity int32, contentsMask int32, data Cell) bool, data int32) TraceHandle {
	if dumpSpots.TraceRayFilterEx == nil {
		missing("TR_TraceRayFilterEx")
	}
	return dumpSpots.TraceRayFilterEx(from, angles, mask, rayType, data)
}

// DidHitTrace says that ray hit something.
//
//sp:native TR_DidHit
func DidHitTrace(trace TraceHandle) bool {
	if dumpSpots.DidHitTrace == nil {
		missing("TR_DidHit")
	}
	return dumpSpots.DidHitTrace(trace)
}

// TraceEndPositionOf is where that ray stopped.
//
//sp:native TR_GetEndPosition into
func TraceEndPositionOf(trace TraceHandle) (position [3]float32) {
	if dumpSpots.TraceEndPositionOf == nil {
		missing("TR_GetEndPosition")
	}
	return dumpSpots.TraceEndPositionOf(trace)
}

// Close releases the ray.
//
//sp:delete Close
func (t TraceHandle) Close() {
	if dumpSpots.CloseTrace == nil {
		missing("delete Handle")
	}
	dumpSpots.CloseTrace(t)
}

// LiteralText is a string literal used where a buffer is wanted, which is how
// a local is declared with a value already in it.
//
//sp:same LiteralText
func LiteralText(s string) Text {
	var out Text
	copy(out[:], s)
	return out
}

// StrEqualCased is StrEqual with the case flag written out.
//
//sp:native StrEqual
func StrEqualCased(a Text, b string, caseSensitive bool) bool {
	if dumpSpots.StrEqualCased == nil {
		missing("StrEqual")
	}
	return dumpSpots.StrEqualCased(a, b, caseSensitive)
}
