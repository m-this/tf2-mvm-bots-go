// ScoreNestArea and BestNestArea of source/redbots3/util.sp, written in the
// subset. The nav mesh walk stays in SourcePawn; what arrives here is the
// candidate list it produced.
package decisions

const (
	maxNestCandidates = 256
	maxApproachSample = 64
	maxEngineers      = 33
)

const (
	nestSightScore  float32 = 60.0
	nestSpacing     float32 = 400.0
	heightCap       float32 = 300.0
	heightWeight    float32 = 0.1
	roomWeight      float32 = 0.05
	sameAreaPenalty float32 = -100.0
	nearAreaPenalty float32 = -50.0
)

type NestCandidate struct {
	Center      [3]float32
	SizeX       float32
	SizeY       float32
	SeesApproch [maxApproachSample]bool
	AreaID      int32
}

type NestSurroundings struct {
	ApproachCount int32
	HeldAreaID    [maxEngineers]int32
	HeldCenter    [maxEngineers][3]float32
	HeldValid     [maxEngineers]bool
	SentryOrigin  [maxEngineers][3]float32
	SentryValid   [maxEngineers]bool
	EngineerCount int32
	Me            int32
}

func absFloat(x float32) float32 {
	if x < 0.0 {
		return -x
	}
	return x
}

func distance(a [3]float32, b [3]float32) float32 {
	dx := a[0] - b[0]
	dy := a[1] - b[1]
	dz := a[2] - b[2]
	return sqrtFloat(dx*dx + dy*dy + dz*dz)
}

func sightScore(c NestCandidate, approachCount int32) float32 {
	if approachCount <= 0 {
		return 0.0
	}
	seen := int32(0)
	for i := int32(0); i < approachCount; i++ {
		if c.SeesApproch[i] {
			seen++
		}
	}
	return (float32(seen) / float32(approachCount)) * nestSightScore
}

func crowdingPenalty(c NestCandidate, s NestSurroundings) float32 {
	penalty := float32(0.0)
	for i := int32(0); i < s.EngineerCount; i++ {
		if i == s.Me {
			continue
		}
		if s.HeldValid[i] && s.HeldAreaID[i] == c.AreaID {
			penalty += sameAreaPenalty
		} else if s.HeldValid[i] && distance(c.Center, s.HeldCenter[i]) < nestSpacing {
			penalty += nearAreaPenalty
		}
		if s.SentryValid[i] && distance(c.Center, s.SentryOrigin[i]) < nestSpacing {
			penalty += sameAreaPenalty
		}
	}
	return penalty
}

func ScoreNestArea(c NestCandidate, target [3]float32, sentryRange float32, disposable bool, s NestSurroundings) float32 {
	idealFraction := float32(0.75)
	if disposable {
		idealFraction = 0.35
	}

	center := c.Center
	center[2] += 50.0

	rangeToTarget := distance(center, target)
	ideal := sentryRange * idealFraction
	score := 100.0 - (absFloat(rangeToTarget-ideal)/sentryRange)*100.0

	if !disposable {
		height := center[2] - target[2]
		if height > 0.0 {
			score += min(height, heightCap) * heightWeight
		}
	}

	score += min(c.SizeX, c.SizeY) * roomWeight
	score += sightScore(c, s.ApproachCount)
	score += crowdingPenalty(c, s)
	return score
}

// BestNestArea answers the index of the best candidate, or -1 for none.
func BestNestArea(candidates [maxNestCandidates]NestCandidate, count int32, target [3]float32, sentryRange float32, disposable bool, s NestSurroundings) int32 {
	best := int32(-1)
	bestScore := float32(0.0)

	for i := int32(0); i < count; i++ {
		score := ScoreNestArea(candidates[i], target, sentryRange, disposable, s)
		if best < 0 || score > bestScore {
			best = i
			bestScore = score
		}
	}
	return best
}
