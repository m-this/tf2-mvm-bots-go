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
