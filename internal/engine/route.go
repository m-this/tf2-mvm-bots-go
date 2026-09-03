package engine

/*
The route out of spawn, which is a path object this port owns rather than one the
game keeps.

A PathFollower made here has to be destroyed here: it is not the bot's own path,
and leaking one leaks a nav search per engineer per attempt.
*/

// RouteCalls are the answers.
type RouteCalls struct {
	NewRoute      func(cost int32, filter int32) Route
	RouteCompute  func(r Route, bot Bot, goal [3]float32) bool
	RouteLength   func(r Route) float32
	RoutePosition func(r Route, distance float32) [3]float32
	RouteDestroy  func(r Route)
}

var routes RouteCalls

// InstallRoutes puts a set of answers behind them.
func InstallRoutes(c RouteCalls) func() {
	previous := routes
	Fill(&c)
	routes = c
	return func() { routes = previous }
}

// Route is a PathFollower this port made, as opposed to the one the bot carries.
//
//sp:tag PathFollower
type Route int32

// FilterIgnoreActors is Path_FilterIgnoreActors, which walks past the people in
// the way.
//
//sp:global Path_FilterIgnoreActors
func FilterIgnoreActors() int32 { return 0 }

// FilterOnlyActors is Path_FilterOnlyActors.
//
//sp:global Path_FilterOnlyActors
func FilterOnlyActors() int32 { return 0 }

/*
NewRoute makes one.

The plugin calls the constructor without new and hands it the default cost
function, which SourcePawn spells as a leading underscore: PathFollower is a
methodmap the game returns rather than an object this port allocates.

//sp:native PathFollower before _
*/
func NewRoute(ignoreActors int32, onlyActors int32) Route {
	return routes.NewRoute(ignoreActors, onlyActors)
}

// Compute builds the route to a position, and says whether there was one.
//
//sp:method ComputeToPos
func (r Route) Compute(bot Bot, goal [3]float32) bool { return routes.RouteCompute(r, bot, goal) }

// Length is how long the route is.
//
//sp:method GetLength
func (r Route) Length() float32 { return routes.RouteLength(r) }

// PositionAlong is the point that far along it.
//
//sp:method GetPosition
func (r Route) PositionAlong(distance float32) (position [3]float32) {
	return routes.RoutePosition(r, distance)
}

// Close releases it, which SourcePawn spells Destroy: a PathFollower is the
// game's object and it has its own teardown.
//
//sp:delete Destroy
func (r Route) Close() { routes.RouteDestroy(r) }
