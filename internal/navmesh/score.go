package navmesh

import "fmt"

// The numbers ScoreNestArea is written against, from source/redbots3/util.sp.
const (
	// NestSightScore is the whole of the sight term, awarded in proportion to
	// how much of the approach the area can see, NEST_SIGHT_SCORE.
	NestSightScore float32 = 80

	// NestSpacing is how close another engineer's nest or sentry has to be
	// before it costs this area, NEST_SPACING.
	NestSpacing float32 = 500

	// NestEyeHeight is how far above the area's surface the score reasons from,
	// which is the 50 added to the centre before anything is measured.
	NestEyeHeight float32 = 50

	// NestHeightCap is the most height the score will pay for, and
	// NestHeightWeight what each unit of it is worth.
	NestHeightCap    float32 = 300
	NestHeightWeight float32 = 0.1

	// NestRoomWeight is what each unit of the area's shorter side is worth.
	NestRoomWeight float32 = 0.05

	// NestIdealRangeHeld and NestIdealRangeDisposable are the fractions of a
	// sentry's range the score wants the target at, for a held sentry and for a
	// gunslinger's.
	NestIdealRangeHeld       float32 = 0.75
	NestIdealRangeDisposable float32 = 0.35

	// DefaultSentryRange is the range PickBuildArea passes when nobody says
	// otherwise.
	DefaultSentryRange float32 = 1300
)

// NestInputs is everything ScoreNestArea reads that is not the area itself.
//
// The plugin reads three of these off the engine at the moment of the call: the
// gunslinger from the client's loadout, the approach from a radius query around
// the bomb, and the crowding from a loop over every other player. They are
// parameters here, which is the split mvm-dop asks for and the only way the
// scorer is testable at all.
type NestInputs struct {
	// Target is what the sentry is being placed against, which today is the
	// bomb.
	Target Vec3

	// SentryRange is the range the score is scaled by.
	SentryRange float32

	// Disposable is true for a gunslinger engineer, who scores the opposite way
	// on range and height.
	Disposable bool

	// Approach is the ground the robots cross to reach the target, as area ids.
	//
	// The plugin builds this from areas within SentryRange of the target that
	// carry the BOMB_DROP attribute. That attribute is computed when the mission
	// loads and is not written to the nav file, so an offline caller supplies
	// the set it wants to score against. An empty set scores the sight term at
	// zero, exactly as the plugin does when the mesh has no visibility data.
	Approach []AreaID

	// Crowding is the penalty another engineer's nest or sentry imposes. It is
	// a loop over live players in the plugin and a number here.
	Crowding float32
}

// NestScore is one area's score with the terms kept apart.
//
// The total is what BestNestArea compares. The terms are what a change to the
// scoring has to be argued in, and mvm-dop is a request for a seventh one, so a
// scorer that returned only the sum would make its own bead untestable.
type NestScore struct {
	Area     AreaID
	Range    float32
	Height   float32
	Room     float32
	Sight    float32
	Crowding float32

	// RangeToTarget and SeenFraction are the readings behind the terms, kept
	// because a score that moved is worth less than the reason it moved.
	RangeToTarget float32
	SeenFraction  float32
}

// Total is the number BestNestArea ranks on.
func (s NestScore) Total() float32 {
	return s.Range + s.Height + s.Room + s.Sight + s.Crowding
}

// String is the score as one report line.
func (s NestScore) String() string {
	return fmt.Sprintf("area %d: %.1f (range %.1f at %.0f, height %.1f, room %.1f, sight %.1f of %.0f%% seen, crowding %.1f)",
		s.Area, s.Total(), s.Range, s.RangeToTarget, s.Height, s.Room, s.Sight, s.SeenFraction*100, s.Crowding)
}

// ScoreNestArea is the port of the same function in source/redbots3/util.sp,
// term by term and in the same order.
//
// It returns zero for an area that is not in the mesh, which a caller iterating
// the mesh cannot produce.
func (m *Mesh) ScoreNestArea(id AreaID, in NestInputs) NestScore {
	a := m.byID[id]
	if a == nil {
		return NestScore{}
	}

	center := a.Center()
	center.Z += NestEyeHeight

	rangeToTarget := center.Distance(in.Target)

	idealFraction := NestIdealRangeHeld
	if in.Disposable {
		idealFraction = NestIdealRangeDisposable
	}
	ideal := in.SentryRange * idealFraction

	s := NestScore{
		Area:          id,
		Range:         100 - (absf(rangeToTarget-ideal)/in.SentryRange)*100,
		RangeToTarget: rangeToTarget,
		Crowding:      in.Crowding,
	}

	if !in.Disposable {
		if height := center.Z - in.Target.Z; height > 0 {
			s.Height = minf(height, NestHeightCap) * NestHeightWeight
		}
	}

	s.Room = minf(a.SizeX(), a.SizeY()) * NestRoomWeight

	if len(in.Approach) > 0 {
		seen := 0
		for _, other := range in.Approach {
			if m.IsCompletelyVisible(id, other) {
				seen++
			}
		}
		s.SeenFraction = float32(seen) / float32(len(in.Approach))
		s.Sight = s.SeenFraction * NestSightScore
	}

	return s
}

// BestNestArea is the highest scoring area of a list, which is what
// BestNestArea in the plugin does once the candidates are collected. It returns
// the winning score, whose Area is zero for an empty list.
func (m *Mesh) BestNestArea(candidates []AreaID, in NestInputs) (NestScore, []NestScore) {
	scores := make([]NestScore, 0, len(candidates))
	best := NestScore{}
	haveBest := false

	for _, id := range candidates {
		s := m.ScoreNestArea(id, in)
		scores = append(scores, s)
		if !haveBest || s.Total() > best.Total() {
			best, haveBest = s, true
		}
	}

	return best, scores
}

// ApproachAreas is the offline stand-in for CollectBombApproachAreas: every area
// within radius of the target, which is what the plugin collects before it
// filters on the BOMB_DROP attribute.
//
// The filter cannot be reproduced from a file, so this returns the unfiltered
// set and the caller narrows it. Used whole it scores the sight term against all
// the ground near the bomb rather than the bomb's path, which is a different
// question and a wider one.
func (m *Mesh) ApproachAreas(target Vec3, radius float32) []AreaID {
	areas := m.AreasWithin(target, radius)
	out := make([]AreaID, 0, len(areas))
	for _, a := range areas {
		out = append(out, a.ID)
	}
	return out
}
