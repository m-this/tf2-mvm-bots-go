package engine

/*
The nav mesh, and the one handle in it.

TheNavMesh.CollectAreasInRadius hands back an AreasCollector, and it has to be
deleted. That is a lifetime, so the Go says it the Go way:

	areas := engine.CollectAreasInRadius(origin, 300.0)
	defer areas.Close()

internal/spbody puts the delete at every way out of the function rather than at
the one the author remembered, and refuses a handle nothing closes at all. The
plugin deletes it once, after the loop, which is right until somebody adds a
return inside the loop.
*/

// NavCalls are the answers.
type NavCalls struct {
	CollectAreasInRadius func(origin [3]float32, radius float32) Areas
	AreasCount           func(a Areas) int32
	AreasGet             func(a Areas, index int32) NavArea
	AreasClose           func(a Areas)
	AreaCenter           func(area NavArea) [3]float32
	HasAttributeTF       func(area NavArea, attribute int32) bool
	IsZeroVector         func(v [3]float32) bool
}

var nav NavCalls

// InstallNav puts a set of answers behind them.
func InstallNav(c NavCalls) func() {
	previous := nav
	nav = c
	return func() { nav = previous }
}

// Areas is CBaseNPC's AreasCollector, a list the caller owns and must delete.
//
//sp:tag AreasCollector
type Areas int32

// NavArea is a CTFNavArea, one piece of walkable ground.
//
//sp:tag CTFNavArea
type NavArea int32

// RedSpawnRoom is RED_SPAWN_ROOM, an area nobody should be sent into.
//
//sp:global RED_SPAWN_ROOM
func RedSpawnRoom() int32 { return 1 }

// BlueSpawnRoom is BLUE_SPAWN_ROOM.
//
//sp:global BLUE_SPAWN_ROOM
func BlueSpawnRoom() int32 { return 2 }

// CollectAreasInRadius is every piece of ground near a point. The caller owns
// what comes back.
//
//sp:native TheNavMesh.CollectAreasInRadius
func CollectAreasInRadius(origin [3]float32, radius float32) Areas {
	if nav.CollectAreasInRadius == nil {
		missing("TheNavMesh.CollectAreasInRadius")
	}
	return nav.CollectAreasInRadius(origin, radius)
}

// Count is how many there are.
//
//sp:method Count
func (a Areas) Count() int32 {
	if nav.AreasCount == nil {
		missing("AreasCollector.Count")
	}
	return nav.AreasCount(a)
}

// Get is one of them.
//
//sp:method Get
func (a Areas) Get(index int32) NavArea {
	if nav.AreasGet == nil {
		missing("AreasCollector.Get")
	}
	return nav.AreasGet(a, index)
}

// Close deletes the collector. Deferred, never called by hand: the generator
// writes it at every way out.
//
//sp:delete Close
func (a Areas) Close() {
	if nav.AreasClose == nil {
		missing("delete AreasCollector")
	}
	nav.AreasClose(a)
}

// Center is the middle of the area, filled into the array it is given.
//
//sp:method GetCenter
func (n NavArea) Center() (centre [3]float32) {
	if nav.AreaCenter == nil {
		missing("CTFNavArea.GetCenter")
	}
	return nav.AreaCenter(n)
}

// HasAttributeTF says whether the area is marked with it.
//
//sp:method HasAttributeTF
func (n NavArea) HasAttributeTF(attribute int32) bool {
	if nav.HasAttributeTF == nil {
		missing("CTFNavArea.HasAttributeTF")
	}
	return nav.HasAttributeTF(n, attribute)
}

// IsZeroVector says whether the position was never filled in.
//
//sp:plugin IsZeroVector
func IsZeroVector(v [3]float32) bool {
	if nav.IsZeroVector == nil {
		missing("IsZeroVector")
	}
	return nav.IsZeroVector(v)
}
