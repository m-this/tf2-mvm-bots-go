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
	IsCompletelyVisible        func(n NavArea, other Area) bool
	IsEntirelyVisible          func(n NavArea, position [3]float32) bool
	SizeX                      func(n NavArea) float32
	SizeY                      func(n NavArea) float32
	CollectAreasInRadius       func(origin [3]float32, radius float32) Areas
	AreasCount                 func(a Areas) int32
	AreasGet                   func(a Areas, index int32) NavArea
	AreasClose                 func(a Areas)
	AreaCenter                 func(area NavArea) [3]float32
	HasAttributeTF             func(area NavArea, attribute int32) bool
	IsZeroVector               func(v [3]float32) bool
	CapturableAreaTrigger      func(team Team) int32
	ControlPointByID           func(pointID int32) int32
	IsFailureImminent          func(client int32) bool
	DefenderAttackSelectTarget func(client int32) bool
}

var nav NavCalls

// InstallNav puts a set of answers behind them.
func InstallNav(c NavCalls) func() {
	previous := nav
	Fill(&c)
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
	return nav.CollectAreasInRadius(origin, radius)
}

// Count is how many there are.
//
//sp:method Count
func (a Areas) Count() int32 { return nav.AreasCount(a) }

// Get is one of them.
//
//sp:method Get
func (a Areas) Get(index int32) NavArea { return nav.AreasGet(a, index) }

// Close deletes the collector. Deferred, never called by hand: the generator
// writes it at every way out.
//
//sp:delete Close
func (a Areas) Close() { nav.AreasClose(a) }

// Center is the middle of the area, filled into the array it is given.
//
//sp:method GetCenter
func (n NavArea) Center() (centre [3]float32) { return nav.AreaCenter(n) }

// HasAttributeTF says whether the area is marked with it.
//
//sp:method HasAttributeTF
func (n NavArea) HasAttributeTF(attribute int32) bool { return nav.HasAttributeTF(n, attribute) }

// IsZeroVector says whether the position was never filled in.
//
//sp:body IsZeroVector
func IsZeroVector(v [3]float32) bool { return nav.IsZeroVector(v) }

// CapturableAreaTrigger is the point that team can take, and -1 for none.
//
//sp:body GetCapturableAreaTrigger
func CapturableAreaTrigger(team Team) int32 { return nav.CapturableAreaTrigger(team) }

// ControlPointByID is the entity for that point, which the debug output names.
//
//sp:body GetControlPointByID
func ControlPointByID(pointID int32) int32 { return nav.ControlPointByID(pointID) }

// IsFailureImminent says the wave is about to be lost, which outranks holding
// any point.
//
//sp:body IsFailureImminent
func IsFailureImminent(client int32) bool { return nav.IsFailureImminent(client) }

// ConceptHelp is MP_CONCEPT_PLAYER_HELP.
//
//sp:global MP_CONCEPT_PLAYER_HELP
func ConceptHelp() int32 { return 2 }

// DefenderAttackSelectTarget is the attack behaviour's own precondition.
//
//sp:body CTFBotDefenderAttack_SelectTarget
func DefenderAttackSelectTarget(client int32) bool { return nav.DefenderAttackSelectTarget(client) }

// Area is a CNavArea, the untagged form the collector hands back where the
// caller does not need the TF-specific attributes.
//
//sp:tag CNavArea
type Area int32

// Area is Get read back as the plain nav area rather than the TF one.
//
//sp:method Get
func (a Areas) Area(index int32) Area { return Area(nav.AreasGet(a, index)) }

// Center is the middle of the area.
//
//sp:method GetCenter
func (a Area) Center() (centre [3]float32) { return nav.AreaCenter(NavArea(a)) }

// AttributeBombDrop is BOMB_DROP, the ground the bomb is carried across.
//
//sp:global BOMB_DROP
func AttributeBombDrop() int32 { return 8 }

// IsCompletelyVisible says every corner of the other area can be seen from this
// one, which is what a sentry needs of the ground it covers.
//
//sp:method IsCompletelyVisible
func (n NavArea) IsCompletelyVisible(other Area) bool { return nav.IsCompletelyVisible(n, other) }

// SizeX is how wide the area is, which is how much room a building has.
//
//sp:method GetSizeX
func (n NavArea) SizeX() float32 { return nav.SizeX(n) }

// SizeY is the other side of it.
//
//sp:method GetSizeY
func (n NavArea) SizeY() float32 { return nav.SizeY(n) }

// AttributeBlocked is BLOCKED, the one nav attribute that changes during a
// mission: gates and func_nav_blocker set it.
//
//sp:global BLOCKED
func AttributeBlocked() int32 { return 1 }

// IsEntirelyVisible says every corner of this area can see the position.
//
//sp:method IsEntirelyVisible
func (n NavArea) IsEntirelyVisible(position [3]float32) bool {
	return nav.IsEntirelyVisible(n, position)
}
