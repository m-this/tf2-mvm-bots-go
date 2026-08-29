package navmesh

import (
	"cmp"
	"slices"
)

func sortAreasByDistance(areas []*Area, pos Vec3) {
	slices.SortFunc(areas, func(a, b *Area) int {
		if c := cmp.Compare(a.Distance(pos), b.Distance(pos)); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
}

func sortFalls(falls []Fall) {
	slices.SortFunc(falls, func(a, b Fall) int {
		if c := cmp.Compare(b.Descent, a.Descent); c != 0 {
			return c
		}
		if c := cmp.Compare(a.From, b.From); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Direction, b.Direction); c != 0 {
			return c
		}
		return cmp.Compare(a.To, b.To)
	})
}

func sortSpans(spans [][2]float32) {
	slices.SortFunc(spans, func(a, b [2]float32) int {
		if c := cmp.Compare(a[0], b[0]); c != 0 {
			return c
		}
		return cmp.Compare(a[1], b[1])
	})
}

func sortAreasByHeightAt(areas []*Area, pos Vec3) {
	slices.SortFunc(areas, func(a, b *Area) int {
		if c := cmp.Compare(a.ZAt(pos.X, pos.Y), b.ZAt(pos.X, pos.Y)); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
}
