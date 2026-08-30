package engine

/*
The engineer's nest, and the building machinery around it.

m_aNestArea lives in engineeridle.sp, which is not ported, so it is a slot read
and written. Everything else here is the plugin's own and goes when the file it
lives in does.
*/

// NestCalls are the answers.
type NestCalls struct {
	NestArea              func(actor int32) Area
	SetNestArea           func(actor int32, area Area)
	NestBuildPosition     func(area Area) [3]float32
	BuildReachTime        func(from [3]float32, to [3]float32) float32
	LogBuildFailure       func(actor int32, what string, why string)
	PickBuildArea         func(actor int32) Area
	ShouldAdvanceNestSpot func(actor int32) bool
	SetAbsOrigin          func(e Entity, origin [3]float32)
}

var nests NestCalls

// InstallNests puts a set of answers behind them.
func InstallNests(c NestCalls) func() {
	previous := nests
	nests = c
	return func() { nests = previous }
}

// StepHeight is TFBOT_STEP_HEIGHT, which a teleport is lifted by so the bot
// does not arrive inside the floor.
//
//sp:global TFBOT_STEP_HEIGHT
func StepHeight() float32 { return 18.0 }

// InDuck is IN_DUCK, which an engineer holds so the building goes down lower.
//
//sp:global IN_DUCK
func InDuck() int32 { return 4 }

// NestAreaOf is the ground this engineer's nest is on.
//
//sp:slot m_aNestArea
func NestAreaOf(actor int32) Area {
	if nests.NestArea == nil {
		missing("m_aNestArea")
	}
	return nests.NestArea(actor)
}

// SetNestArea moves him to a different one.
//
//sp:slotset m_aNestArea
func SetNestArea(actor int32, area Area) {
	if nests.SetNestArea == nil {
		missing("m_aNestArea")
	}
	nests.SetNestArea(actor, area)
}

// NestBuildPosition is where in that area the building goes.
//
//sp:body NestBuildPosition
func NestBuildPosition(area Area) (position [3]float32) {
	if nests.NestBuildPosition == nil {
		missing("NestBuildPosition")
	}
	return nests.NestBuildPosition(area)
}

// BuildReachTime prices the walk by its length, because the walk stopped being
// inside the nest the moment he started every one of them at the station.
//
//sp:plugin BuildReachTime
func BuildReachTime(from [3]float32, to [3]float32) float32 {
	if nests.BuildReachTime == nil {
		missing("BuildReachTime")
	}
	return nests.BuildReachTime(from, to)
}

// LogBuildFailure says out loud why a build ended without a building, which was
// invisible until it did.
//
//sp:body LogBuildFailure
func LogBuildFailure(actor int32, what string, why string) {
	if nests.LogBuildFailure == nil {
		missing("LogBuildFailure")
	}
	nests.LogBuildFailure(actor, what, why)
}

// PickBuildArea scores the nav mesh and picks somewhere to build, which is the
// expensive answer and is given only when the spot itself is the suspect.
//
//sp:body PickBuildArea
func PickBuildArea(actor int32) Area {
	if nests.PickBuildArea == nil {
		missing("PickBuildArea")
	}
	return nests.PickBuildArea(actor)
}

// ShouldAdvanceNestSpot says the idle action wants him somewhere else.
//
//sp:body CTFBotMvMEngineerIdle_ShouldAdvanceNestSpot
func ShouldAdvanceNestSpot(actor int32) bool {
	if nests.ShouldAdvanceNestSpot == nil {
		missing("CTFBotMvMEngineerIdle_ShouldAdvanceNestSpot")
	}
	return nests.ShouldAdvanceNestSpot(actor)
}

// SetAbsOrigin puts the entity somewhere, which between rounds is how the
// engineer is teleported onto his nest for a faster setup.
//
//sp:method SetAbsOrigin
func (e Entity) SetAbsOrigin(origin [3]float32) {
	if nests.SetAbsOrigin == nil {
		missing("CBaseEntity.SetAbsOrigin")
	}
	nests.SetAbsOrigin(e, origin)
}
