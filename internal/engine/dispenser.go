package engine

// DispenserCalls are the answers.
type DispenserCalls struct {
	NestZoneOf            func(area Area) Text
	NearestConfiguredSpot func(spots List, from [3]float32) (bool, [3]float32)
	IsSentrySafe          func(sentry int32) bool
	StrEqualText          func(a Text, b Text) bool
	ListGetArray          func(l List, index int32) [3]float32
	ListPushArray         func(l List, value [3]float32)
	ListGetString         func(l List, index int32) Text
}

var dispensers DispenserCalls

// InstallDispensers puts a set of answers behind them.
func InstallDispensers(c DispenserCalls) func() {
	previous := dispensers
	dispensers = c
	return func() { dispensers = previous }
}

// DispenserSpots are the dispenser coordinates the map configuration names.
//
//sp:global g_arrMapConfig.adtDispenserLocation
func DispenserSpots() List { return 0 }

// DispenserZones are the zones those spots belong to, one per spot.
//
//sp:global g_arrMapConfig.adtDispenserZone
func DispenserZones() List { return 0 }

// NoZone is the empty zone name, which is what a spot the map reserved for
// nobody carries.
//
//sp:global ""
func NoZone() Text { return Text{} }

// NestZoneOf is the zone a nest area belongs to, empty when the map names none.
//
//sp:plugin NestZoneOf sized
func NestZoneOf(area Area) (zone Text) {
	if dispensers.NestZoneOf == nil {
		missing("NestZoneOf")
	}
	return dispensers.NestZoneOf(area)
}

// NearestConfiguredSpot is the one of them closest to a position.
//
//sp:body NearestConfiguredSpot
func NearestConfiguredSpot(spots List, from [3]float32) (ok bool, spot [3]float32) {
	if dispensers.NearestConfiguredSpot == nil {
		missing("NearestConfiguredSpot")
	}
	return dispensers.NearestConfiguredSpot(spots, from)
}

// IsSentrySafe says the sentry is not currently being taken apart.
//
//sp:plugin IsSentrySafe
func IsSentrySafe(sentry int32) bool {
	if dispensers.IsSentrySafe == nil {
		missing("IsSentrySafe")
	}
	return dispensers.IsSentrySafe(sentry)
}

// StrEqualText compares two buffers, which is what a zone name is on both
// sides.
//
//sp:native StrEqual
func StrEqualText(a Text, b Text) bool {
	if dispensers.StrEqualText == nil {
		missing("StrEqual")
	}
	return dispensers.StrEqualText(a, b)
}

// GetArray is one entry of a list of vectors.
//
//sp:method GetArray
func (l List) GetArray(index int32) (out [3]float32) {
	if dispensers.ListGetArray == nil {
		missing("ArrayList.GetArray")
	}
	return dispensers.ListGetArray(l, index)
}

// PushArray adds one.
//
//sp:method PushArray
func (l List) PushArray(value [3]float32) {
	if dispensers.ListPushArray == nil {
		missing("ArrayList.PushArray")
	}
	dispensers.ListPushArray(l, value)
}

// GetString is one entry of a list of names.
//
//sp:method GetString sized
func (l List) GetString(index int32) (out Text) {
	if dispensers.ListGetString == nil {
		missing("ArrayList.GetString")
	}
	return dispensers.ListGetString(l, index)
}
