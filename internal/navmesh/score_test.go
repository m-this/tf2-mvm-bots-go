package navmesh

import (
	"math"
	"testing"
)

// mannworksBomb is where the bomb starts on Mannworks, which is what
// ScoreNestArea scores against for that map. It is the hatch, taken as the
// centre of the lowest area at the RED end; the exact point matters less than
// that every case here uses the same one.
var mannworksBomb = Vec3{-96, 1600, 256}

// TestScoreNestAreaTerms is mvm-dop's precondition: the score has to be
// separable before a seventh term can be argued about.
//
// Each case moves one input and asserts the term it belongs to moves and the
// others do not.
func TestScoreNestAreaTerms(t *testing.T) {
	m := loadMap(t, "mvm_mannworks")

	area := m.AreaAt(Vec3{-178, 1131, 249}, HalfHumanHeight)
	if area == nil {
		t.Fatal("the first mannworks nest spot is not on the mesh")
	}

	base := NestInputs{Target: mannworksBomb, SentryRange: DefaultSentryRange}
	got := m.ScoreNestArea(area.ID, base)

	t.Run("room is the shorter side", func(t *testing.T) {
		want := minf(area.SizeX(), area.SizeY()) * NestRoomWeight
		if got.Room != want {
			t.Errorf("room %g, want %g", got.Room, want)
		}
	})

	t.Run("no approach means no sight term", func(t *testing.T) {
		if got.Sight != 0 || got.SeenFraction != 0 {
			t.Errorf("sight %g over %g seen, want nothing without an approach", got.Sight, got.SeenFraction)
		}
	})

	t.Run("crowding is passed straight through", func(t *testing.T) {
		in := base
		in.Crowding = -150
		with := m.ScoreNestArea(area.ID, in)

		if with.Crowding != -150 {
			t.Errorf("crowding %g, want -150", with.Crowding)
		}
		if with.Total() != got.Total()-150 {
			t.Errorf("total %g, want %g", with.Total(), got.Total()-150)
		}
		if with.Range != got.Range || with.Height != got.Height || with.Room != got.Room {
			t.Error("crowding moved a term that is not crowding")
		}
	})

	t.Run("a gunslinger scores range the other way and pays no height", func(t *testing.T) {
		in := base
		in.Disposable = true
		mini := m.ScoreNestArea(area.ID, in)

		if mini.Height != 0 {
			t.Errorf("height %g, want nothing for a mini sentry", mini.Height)
		}
		if mini.RangeToTarget != got.RangeToTarget {
			t.Error("the loadout moved the range to the target")
		}
		if mini.Range == got.Range {
			t.Error("the loadout did not move the range term")
		}
	})

	t.Run("height is capped", func(t *testing.T) {
		in := base
		in.Target = Vec3{mannworksBomb.X, mannworksBomb.Y, mannworksBomb.Z - 100000}
		high := m.ScoreNestArea(area.ID, in)

		if want := NestHeightCap * NestHeightWeight; high.Height != want {
			t.Errorf("height %g, want the cap %g", high.Height, want)
		}
	})

	t.Run("below the target pays no height", func(t *testing.T) {
		in := base
		in.Target = Vec3{mannworksBomb.X, mannworksBomb.Y, mannworksBomb.Z + 100000}
		low := m.ScoreNestArea(area.ID, in)

		if low.Height != 0 {
			t.Errorf("height %g, want nothing for ground below the target", low.Height)
		}
	})
}

// TestScoreRangeIdeal pins the shape of the range term: it peaks where the score
// wants the target and falls away either side, symmetrically.
func TestScoreRangeIdeal(t *testing.T) {
	m := loadMap(t, "mvm_mannworks")

	area := m.Areas[0]
	center := area.Center()
	center.Z += NestEyeHeight

	cases := []struct {
		name       string
		disposable bool
		ideal      float32
	}{
		{"held sentry", false, DefaultSentryRange * NestIdealRangeHeld},
		{"gunslinger", true, DefaultSentryRange * NestIdealRangeDisposable},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			at := func(offset float32) float32 {
				in := NestInputs{
					Target:      Vec3{center.X + c.ideal + offset, center.Y, center.Z},
					SentryRange: DefaultSentryRange,
					Disposable:  c.disposable,
				}
				return m.ScoreNestArea(area.ID, in).Range
			}

			peak := at(0)
			if math.Abs(float64(peak-100)) > 0.01 {
				t.Errorf("the range term peaks at %g, want 100", peak)
			}
			if near, far := at(-200), at(200); math.Abs(float64(near-far)) > 0.1 {
				t.Errorf("the range term is %g short and %g long of the ideal, want them equal", near, far)
			}
			if at(400) >= peak {
				t.Error("the range term does not fall away from the ideal")
			}
		})
	}
}

// TestScoreNestAreaOnRealAreas is mvm-dop and mvm-nza's point: the scorer runs
// over real geometry with real visibility, and the ranking it produces can be
// looked at without a wave.
func TestScoreNestAreaOnRealAreas(t *testing.T) {
	m := loadMap(t, "mvm_mannworks")

	approach := m.ApproachAreas(mannworksBomb, DefaultSentryRange)
	if len(approach) == 0 {
		t.Fatal("no approach areas round the bomb")
	}

	in := NestInputs{
		Target:      mannworksBomb,
		SentryRange: DefaultSentryRange,
		Approach:    approach,
	}

	candidates := make([]AreaID, 0, len(m.Areas))
	for _, a := range m.Areas {
		candidates = append(candidates, a.ID)
	}

	best, all := m.BestNestArea(candidates, in)
	if len(all) != len(candidates) {
		t.Fatalf("scored %d of %d candidates", len(all), len(candidates))
	}

	for _, s := range all {
		if s.Total() > best.Total() {
			t.Fatalf("area %d scores %g, above the best of %g", s.Area, s.Total(), best.Total())
		}
		if s.SeenFraction < 0 || s.SeenFraction > 1 {
			t.Fatalf("area %d sees %g of the approach", s.Area, s.SeenFraction)
		}
	}

	// The sight term is the one that needs the mesh's visibility data, and a
	// mesh read without it would score every area zero there. That is the
	// failure mvm-dop's fix would be measured against, so it has to be real.
	sighted := 0
	for _, s := range all {
		if s.Sight > 0 {
			sighted++
		}
	}
	if sighted == 0 {
		t.Fatal("no area sees any of the approach, so the visibility set was not read")
	}
}

// TestVisibilityIsSymmetricEnough checks the visibility data itself.
//
// Complete visibility is not a symmetric relation and is not expected to be:
// it means every corner of the other area is in sight, so a small area can be
// completely visible from a large one that is not completely visible back. What
// would be wrong is no agreement at all, which is what reading the visible set
// at the wrong offset looks like. Three quarters agreeing is the shape of real
// data; a tenth would not be.
func TestVisibilityIsSymmetricEnough(t *testing.T) {
	m := loadMap(t, "mvm_mannhattan")

	pairs, agree := 0, 0
	for _, a := range m.Areas {
		for _, e := range a.Visible {
			if !e.Visibility.Has(CompletelyVisible) {
				continue
			}
			pairs++
			if m.IsCompletelyVisible(e.ID, a.ID) {
				agree++
			}
		}
	}

	if pairs == 0 {
		t.Fatal("no visible pairs at all")
	}
	if share := float64(agree) / float64(pairs); share < 0.7 {
		t.Fatalf("only %.0f%% of visible pairs are visible both ways", share*100)
	}
}

// TestScoreNestAreaUnknownArea covers the one refusal the scorer has.
func TestScoreNestAreaUnknownArea(t *testing.T) {
	m := loadMap(t, "mvm_mannworks")

	if got := m.ScoreNestArea(0, NestInputs{SentryRange: DefaultSentryRange}); got != (NestScore{}) {
		t.Fatalf("scoring an area that is not in the mesh gave %v", got)
	}
}
