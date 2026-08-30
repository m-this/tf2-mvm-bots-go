package engine

// TeleporterCalls are the answers.
type TeleporterCalls struct {
	SpawnRoutePoints   func(actor int32, spawn [3]float32, first float32, step float32, reach float32) (int32, [8][3]float32, [8][3]float32)
	NearestSpawnPoint  func(actor int32) (bool, [3]float32)
	HasObjectOfType    func(client int32, objectType Object, mode ObjectMode) int32
	ObjectOfTypeMode   func(client int32, objectType Object, mode ObjectMode) int32
	IsBuilderSetToMode func(client int32, objectType Object, mode ObjectMode) bool
	LastResult         func(actor int32, buffer Text, maxlength int32)
	HasGivenUp         func(actor int32) bool
	Mode               func(actor int32) ObjectMode
	Spot               func(actor int32) [3]float32
}

var teleporters TeleporterCalls

// InstallTeleporters puts a set of answers behind them.
func InstallTeleporters(c TeleporterCalls) func() {
	previous := teleporters
	teleporters = c
	return func() { teleporters = previous }
}

// ObjectMode is TFObjectMode, which is the half of a teleporter.
//
//sp:tag TFObjectMode
type ObjectMode int32

// ModeNone is TFObjectMode_None, which is what a building with no halves takes.
//
//sp:global TFObjectMode_None
func ModeNone() ObjectMode { return 0 }

// ModeEntrance is TFObjectMode_Entrance.
//
//sp:global TFObjectMode_Entrance
func ModeEntrance() ObjectMode { return 0 }

// ModeExit is TFObjectMode_Exit.
//
//sp:global TFObjectMode_Exit
func ModeExit() ObjectMode { return 1 }

// ObjectTeleporter is TFObject_Teleporter.
//
//sp:global TFObject_Teleporter
func ObjectTeleporter() Object { return 1 }

// FeatureEngineerClimbs is the switch on an engineer crouch jumping onto the
// ground a spot sits on.
//
//sp:global FEATURE_ENGINEER_CLIMBS
func FeatureEngineerClimbs() int32 { return 16 }

// InForward is IN_FORWARD.
//
//sp:global IN_FORWARD
func InForward() int32 { return 8 }

// InJump is IN_JUMP.
//
//sp:global IN_JUMP
func InJump() int32 { return 2 }

// TeleporterEntranceSpots are the entrance coordinates the map names.
//
//sp:global g_arrMapConfig.adtTeleporterEntranceLocation
func TeleporterEntranceSpots() List { return 0 }

// TeleporterExitSpots are the exit coordinates the map names.
//
//sp:global g_arrMapConfig.adtTeleporterExitLocation
func TeleporterExitSpots() List { return 0 }

/*
SpawnRoutePoints samples the way out of spawn: where a teleporter goes, and
where the man stands to put it there, one pair per step along the route.

The count of points comes last in the plugin's own declaration, after the two
arrays it fills, so it is written as a trailing argument rather than a Go one.

//sp:body SpawnRoutePoints after TELEPORTER_TRY_POINTS
*/
func SpawnRoutePoints(actor int32, spawn [3]float32, first float32, step float32, reach float32) (found int32, spots [8][3]float32, stands [8][3]float32) {
	if teleporters.SpawnRoutePoints == nil {
		missing("SpawnRoutePoints")
	}
	return teleporters.SpawnRoutePoints(actor, spawn, first, step, reach)
}

// NearestSpawnPoint is where this bot's team respawns, and false when there is
// no such thing on this map.
//
//sp:body NearestSpawnPoint
func NearestSpawnPoint(actor int32) (ok bool, spawn [3]float32) {
	if teleporters.NearestSpawnPoint == nil {
		missing("NearestSpawnPoint")
	}
	return teleporters.NearestSpawnPoint(actor)
}

// HasObjectOfType counts what is in the engineer's hands as well as what is
// standing.
//
//sp:body HasObjectOfType
func HasObjectOfType(client int32, objectType Object, mode ObjectMode) int32 {
	if teleporters.HasObjectOfType == nil {
		missing("HasObjectOfType")
	}
	return teleporters.HasObjectOfType(client, objectType, mode)
}

// ObjectOfTypeMode is GetObjectOfType for a building that has halves.
//
//sp:body GetObjectOfType
func ObjectOfTypeMode(client int32, objectType Object, mode ObjectMode) int32 {
	if teleporters.ObjectOfTypeMode == nil {
		missing("GetObjectOfType")
	}
	return teleporters.ObjectOfTypeMode(client, objectType, mode)
}

// IsBuilderSetToMode is IsBuilderSetTo for one half of a teleporter.
//
//sp:body IsBuilderSetTo
func IsBuilderSetToMode(client int32, objectType Object, mode ObjectMode) bool {
	if teleporters.IsBuilderSetToMode == nil {
		missing("IsBuilderSetTo")
	}
	return teleporters.IsBuilderSetToMode(client, objectType, mode)
}

// TeleporterLastResult is why the last teleporter attempt ended.
//
//sp:body EngineerTeleporter_LastResult
func TeleporterLastResult(actor int32, buffer Text, maxlength int32) {
	if teleporters.LastResult == nil {
		missing("EngineerTeleporter_LastResult")
	}
	teleporters.LastResult(actor, buffer, maxlength)
}

// TeleporterHasGivenUp says the engineer stopped asking for this round.
//
//sp:body EngineerTeleporter_HasGivenUp
func TeleporterHasGivenUp(actor int32) bool {
	if teleporters.HasGivenUp == nil {
		missing("EngineerTeleporter_HasGivenUp")
	}
	return teleporters.HasGivenUp(actor)
}

// TeleporterMode is which half he is building.
//
//sp:body EngineerTeleporter_Mode
func TeleporterMode(actor int32) ObjectMode {
	if teleporters.Mode == nil {
		missing("EngineerTeleporter_Mode")
	}
	return teleporters.Mode(actor)
}

// TeleporterSpot is where this attempt puts it.
//
//sp:body EngineerTeleporter_Spot
func TeleporterSpot(actor int32) (spot [3]float32) {
	if teleporters.Spot == nil {
		missing("EngineerTeleporter_Spot")
	}
	return teleporters.Spot(actor)
}
