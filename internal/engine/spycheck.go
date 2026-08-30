package engine

// SpyCheckCalls are the answers.
type SpyCheckCalls struct {
	AngleForward         func(angles [3]float32) [3]float32
	ClientEyeAngles      func(client int32) [3]float32
	IsAbleToSeeTarget    func(v Vision, target int32, useFOV bool) bool
	IsFakeClient         func(client int32) bool
	HasTheFlag           func(client int32) bool
	TimeSinceWeaponFired func(client int32) float32
	MinFloat             func(a float32, b float32) float32
}

var spyChecks SpyCheckCalls

// InstallSpyChecks puts a set of answers behind them.
func InstallSpyChecks(c SpyCheckCalls) func() {
	previous := spyChecks
	spyChecks = c
	return func() { spyChecks = previous }
}

// UseFOV is USE_FOV, which asks whether the bot could see it rather than
// whether the map lets a line through.
//
//sp:global USE_FOV
func UseFOV() bool { return true }

// AimImportant is IMPORTANT, the aim priority a glance uses: worth doing, and
// dropped for anything real.
//
//sp:global IMPORTANT
func AimImportant() int32 { return 1 }

// ConceptCloakedSpy is MP_CONCEPT_PLAYER_CLOAKEDSPY, the line a bot says while
// it frisks its own team.
//
//sp:global MP_CONCEPT_PLAYER_CLOAKEDSPY
func ConceptCloakedSpy() int32 { return 6 }

// FeatureSpyGlance is the switch on a bot turning to look behind itself.
//
//sp:global FEATURE_SPY_GLANCE
func FeatureSpyGlance() int32 { return 2 }

/*
AngleForward is the way an angle faces.

SourcePawn's GetAngleVectors fills three of them and there is no way to ask for
one: the right and the up go to NULL_VECTOR, written after the buffer because
that is the order the native declares.

//sp:native GetAngleVectors after NULL_VECTOR NULL_VECTOR
*/
func AngleForward(angles [3]float32) (forward [3]float32) {
	if spyChecks.AngleForward == nil {
		missing("GetAngleVectors")
	}
	return spyChecks.AngleForward(angles)
}

// ClientEyeAngles is where the client is looking.
//
//sp:native GetClientEyeAngles
func ClientEyeAngles(client int32) (angles [3]float32) {
	if spyChecks.ClientEyeAngles == nil {
		missing("GetClientEyeAngles")
	}
	return spyChecks.ClientEyeAngles(client)
}

// IsAbleToSeeTarget is the bot's own eyes, field of view included.
//
//sp:method IsAbleToSeeTarget
func (v Vision) IsAbleToSeeTarget(target int32, useFOV bool) bool {
	if spyChecks.IsAbleToSeeTarget == nil {
		missing("IVision.IsAbleToSeeTarget")
	}
	return spyChecks.IsAbleToSeeTarget(v, target, useFOV)
}

// IsFakeClient says the client is a bot, which is how a human teammate is kept
// out of the frisking.
//
//sp:native IsFakeClient
func IsFakeClient(client int32) bool {
	if spyChecks.IsFakeClient == nil {
		missing("IsFakeClient")
	}
	return spyChecks.IsFakeClient(client)
}

// HasTheFlag says the client is carrying the bomb.
//
//sp:plugin TF2_HasTheFlag
func HasTheFlag(client int32) bool {
	if spyChecks.HasTheFlag == nil {
		missing("TF2_HasTheFlag")
	}
	return spyChecks.HasTheFlag(client)
}

// TimeSinceWeaponFired is the alibi a disguised spy cannot produce.
//
//sp:plugin GetTimeSinceWeaponFired
func TimeSinceWeaponFired(client int32) float32 {
	if spyChecks.TimeSinceWeaponFired == nil {
		missing("GetTimeSinceWeaponFired")
	}
	return spyChecks.TimeSinceWeaponFired(client)
}

// MinFloat is the smaller of two. The plugin has its own, and the generator's
// own min would emit a second one beside it.
//
//sp:plugin MinFloat
func MinFloat(a float32, b float32) float32 {
	if spyChecks.MinFloat == nil {
		missing("MinFloat")
	}
	return spyChecks.MinFloat(a, b)
}
