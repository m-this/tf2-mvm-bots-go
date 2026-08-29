package navmesh

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the report golden from the current meshes and configs")

// TestReport writes what every query in this package says about the shipped
// maps, and compares it with the golden beside it.
//
// The golden is the finding, not the code. It is a fact about upstream's configs
// and Valve's meshes, so it must not fail the build when it changes: it changes
// when a spot is moved, and then the diff is the review. Regenerate with
//
//	go test ./internal/navmesh -update
func TestReport(t *testing.T) {
	cfgs := loadConfigs(t)

	var b strings.Builder
	reportHeader(&b, cfgs)
	reportSnap(&b, t, cfgs)
	reportDrops(&b, t, cfgs)
	reportFooting(&b, t, cfgs)
	reportFindings(&b, t, cfgs)
	reportWedges(&b, t)
	reportScores(&b, t)

	golden := filepath.Join("testdata", "report.txt")
	got := b.String()

	if *update {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", golden)
		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v; run go test ./internal/navmesh -update", err)
	}
	if got != string(want) {
		t.Errorf("the report has changed; run go test ./internal/navmesh -update and read the diff")
	}
}

func section(b *strings.Builder, title string) {
	fmt.Fprintf(b, "\n%s\n%s\n\n", title, strings.Repeat("-", len(title)))
}

func reportHeader(b *strings.Builder, cfgs []*MapConfig) {
	section(b, "What is covered")

	var with, without []string
	for _, c := range cfgs {
		if haveNav(c.Map) {
			with = append(with, c.Map)
		} else {
			without = append(without, c.Map)
		}
	}

	fmt.Fprintf(b, "%d map configs, of which %d have a nav mesh here and %d do not.\n",
		len(cfgs), len(with), len(without))
	fmt.Fprintf(b, "The meshes are nav version %d subversion %d, out of the game's own\n",
		SupportedVersion, SupportedSubVersion)
	fmt.Fprintf(b, "tf2_misc_dir.vpk. The %d without are community maps whose .nav files are\n", len(without))
	fmt.Fprint(b, "not in the dedicated server install, so nothing geometric below is said\n")
	fmt.Fprint(b, "about them. The two rules that count config entries rather than read the\n")
	fmt.Fprint(b, "mesh do cover them, and are the last section.\n\nNo mesh:\n")
	for _, name := range without {
		fmt.Fprintf(b, "  %s\n", name)
	}
}

func reportSnap(b *strings.Builder, t *testing.T, cfgs []*MapConfig) {
	section(b, "mvm-fgs: where a build spot snaps to")

	fmt.Fprint(b, "Every declared spot, with the gap to the nearest nav surface and what the\n")
	fmt.Fprint(b, "eight BuildStandPoint sides do with it. \"elsewhere\" counts the sides that\n")
	fmt.Fprint(b, "were accepted onto ground other than the spot's own area, and \"drop\" is how\n")
	fmt.Fprint(b, "far below the spot the shallowest and the deepest of those sides land.\n\n")
	fmt.Fprint(b, "\"over\" is the height of the spot above the ground round it, and it is the\n")
	fmt.Fprint(b, "column the verdict is drawn from rather than a number to read: a spot with\n")
	fmt.Fprint(b, "every side a storey down and nothing at its own level is a spot the building\n")
	fmt.Fprint(b, "silently moves off.\n\n")
	fmt.Fprintf(b, "%-16s %-16s %-22s %6s %6s %10s %13s %6s\n",
		"map", "spot", "origin", "offset", "sides", "elsewhere", "drop", "over")

	for _, c := range cfgs {
		if !haveNav(c.Map) {
			continue
		}
		m := loadMap(t, c.Map)

		for _, s := range c.Spots {
			v := m.CheckSnap(s, s.Origin, BuildTryPoints, BuildReach, HalfHumanHeight)
			p := m.CheckPoint(s.Origin)

			fmt.Fprintf(b, "%-16s %-16s %-22s %6.0f %6s %10d %13s %6.0f\n",
				c.Map,
				string(s.Kind)+" "+s.Index,
				fmt.Sprintf("%.0f %.0f %.0f", s.Origin.X, s.Origin.Y, s.Origin.Z),
				p.NearestDistance,
				fmt.Sprintf("%d/%d", v.Accepted, len(v.Sides)),
				v.Elsewhere,
				fmt.Sprintf("%.0f to %.0f", v.LeastDrop, v.WorstDrop),
				p.SurroundHeight)
		}
	}
}

func reportDrops(b *strings.Builder, t *testing.T, cfgs []*MapConfig) {
	section(b, "mvm-0am and mvm-778: spots beside a drop")

	fmt.Fprintf(b, "The deepest fall within %.0f units of each declared spot. Fall damage starts\n", ExitDropRadius)
	fmt.Fprintf(b, "at about %.0f units of drop and about %.0f kills a light class. A blank is no\n",
		FallDamageHeight, FallLethalHeight)
	fmt.Fprint(b, "fall over a step; the query under-reports, because a pit with no nav in the\n")
	fmt.Fprint(b, "bottom of it is a pit this cannot measure.\n\n")

	for _, c := range cfgs {
		if !haveNav(c.Map) {
			continue
		}
		m := loadMap(t, c.Map)

		for _, s := range c.Spots {
			v := m.CheckDrop(s, ExitDropRadius, HalfHumanHeight)
			if v.Worst.Descent <= StepHeight {
				continue
			}

			note := ""
			switch {
			case v.Kills():
				note = "  KILLS"
			case v.Hurts():
				note = "  HURTS"
			}
			fmt.Fprintf(b, "%-16s %-16s %s%s\n", c.Map, string(s.Kind)+" "+s.Index, v.Worst, note)
		}
	}

	fmt.Fprint(b, "\nAnd the whole mesh, for scale: every fall on each map, and how much of the\n")
	fmt.Fprint(b, "map has a teleporter exit ring point beside one. The exit ring is what the\n")
	fmt.Fprintf(b, "engineer falls back to when the named spot beats him, %.0f units out from his\n", ExitRingRadius)
	fmt.Fprint(b, "nest on eight sides, and nothing vets it.\n\n")
	fmt.Fprintf(b, "%-16s %7s %7s %7s %7s %14s\n", "map", "falls", "hurts", "kills", "areas", "risky ring")

	for _, name := range shippedMaps {
		m := loadMap(t, name)

		var falls, hurt, kill int
		for _, a := range m.Areas {
			for _, f := range m.Falls(a.ID) {
				falls++
				switch {
				case f.Descent >= FallLethalHeight:
					kill++
				case f.Descent >= FallDamageHeight:
					hurt++
				}
			}
		}

		risky := 0
		for _, a := range m.Areas {
			if _, ok := ringWorstFall(m, a.Center()); ok {
				risky++
			}
		}

		fmt.Fprintf(b, "%-16s %7d %7d %7d %7d %13.0f%%\n",
			name, falls, hurt, kill, len(m.Areas), 100*float64(risky)/float64(len(m.Areas)))
	}
}

func reportFooting(b *strings.Builder, t *testing.T, cfgs []*MapConfig) {
	section(b, "mvm-z83.32: declared spots not level with the ground round them")

	fmt.Fprint(b, "These used to be reported as one thing, \"in a hole in the mesh\", on the one\n")
	fmt.Fprint(b, "test of whether the mesh has a footprint under the coordinate. They are three\n")
	fmt.Fprint(b, "things, and the footprint is not what separates them.\n\n")
	fmt.Fprint(b, "A spot raised over the ground round it is on top of something: a rock, a\n")
	fmt.Fprint(b, "roof, a container. Deliberate authoring, and it costs the bot nothing to\n")
	fmt.Fprintf(b, "arrive, because every stand side snaps to the floor at its foot. Whether it\n")
	fmt.Fprint(b, "is honoured is the mvm-fgs question above, not this one.\n\n")
	fmt.Fprint(b, "A spot level with the ground round it is in a hole at ground level, with\n")
	fmt.Fprint(b, "nothing lower to snap down to. That is the one that strands a bot: a path to\n")
	fmt.Fprint(b, "it ends at the edge of the hole, the arrival test never comes true, and every\n")
	fmt.Fprint(b, "attempt sends him back to the same place.\n\n")
	fmt.Fprint(b, "A spot with no ground within a snap of it at all is off the mesh, and no\n")
	fmt.Fprint(b, "bot reaches it by any route.\n\n")
	fmt.Fprintf(b, "The line between the first two is %.0f units, a body's step. The rock tops\n", RaisedStep)
	fmt.Fprint(b, "read 60 to 79 and the holes read -4 to 4, so nothing sits near it.\n\n")

	for _, c := range cfgs {
		if !haveNav(c.Map) {
			continue
		}
		m := loadMap(t, c.Map)

		for _, s := range c.Spots {
			v := m.CheckPoint(s.Origin)
			if v.Footing == FootingGround {
				continue
			}
			if !s.Kind.IsGround() && v.Footing == FootingRaised {
				// A sniper spot is written at a player's eye, so it is
				// raised by construction and says nothing.
				continue
			}
			fmt.Fprintf(b, "%-16s %-16s %v\n", c.Map, string(s.Kind)+" "+s.Index, v)
		}
	}
}

func reportFindings(b *strings.Builder, t *testing.T, cfgs []*MapConfig) {
	section(b, "mvm-z83.25: the configs as a checked table")

	fmt.Fprint(b, "Every rule, and every config it is broken by. The rules that count entries\n")
	fmt.Fprint(b, "need no nav mesh, so they are the only thing said here about the twenty\n")
	fmt.Fprint(b, "community maps; the rest are read on the seven meshes.\n\n")

	for _, r := range Rules {
		scope := "all 27 configs"
		if r.NeedsMesh() {
			scope = "the 7 with a mesh"
		}
		fmt.Fprintf(b, "  %-22s %-9s over %s\n", r, r.Severity(), scope)
	}
	fmt.Fprint(b, "\n")

	found := 0
	for _, c := range cfgs {
		var m *Mesh
		if haveNav(c.Map) {
			m = loadMap(t, c.Map)
		}
		for _, f := range CheckConfig(m, c) {
			fmt.Fprintf(b, "%v\n", f)
			found++
		}
	}

	fmt.Fprintf(b, "\n%d findings.\n", found)
}

func reportWedges(b *strings.Builder, t *testing.T) {
	section(b, "mvm-wb0 and mvm-ipf: the coordinate")

	m := loadMap(t, "mvm_mannworks")
	v := m.CheckPoint(mannworksWedge)

	fmt.Fprintf(b, "%v\n\n", v)
	fmt.Fprint(b, "The ground round it, by distance:\n")
	for _, a := range m.AreasWithin(mannworksWedge, 120) {
		fmt.Fprintf(b, "  area %-6d %5.1f away, %3.0fx%-3.0f, surface at z %.0f\n",
			a.ID, a.Distance(mannworksWedge), a.SizeX(), a.SizeY(), a.Center().Z)
	}

	fmt.Fprint(b, "\nWhich nav areas cover a grid round it, on 25 unit steps:\n")
	for y := float32(810); y <= 960; y += 25 {
		fmt.Fprintf(b, "  y=%4.0f  ", y)
		for x := float32(939); x <= 1089; x += 25 {
			if m.AreaUnder(Vec3{x, y, mannworksWedge.Z}, BeneathLimit) != nil {
				fmt.Fprint(b, "#")
			} else {
				fmt.Fprint(b, ".")
			}
		}
		fmt.Fprint(b, "\n")
	}
	fmt.Fprintf(b, "          %s\n", strings.Repeat(" ", 3)+"^ x=1014")
}

func reportScores(b *strings.Builder, t *testing.T) {
	section(b, "mvm-dop and mvm-nza: ScoreNestArea on real areas")

	m := loadMap(t, "mvm_mannworks")
	approach := m.ApproachAreas(mannworksBomb, DefaultSentryRange)

	fmt.Fprintf(b, "Mannworks, target %.0f %.0f %.0f, sentry range %.0f, %d approach areas.\n",
		mannworksBomb.X, mannworksBomb.Y, mannworksBomb.Z, DefaultSentryRange, len(approach))
	fmt.Fprint(b, "The approach set here is every area within range of the target: the plugin\n")
	fmt.Fprint(b, "narrows that by the BOMB_DROP attribute, which is computed when the mission\n")
	fmt.Fprint(b, "loads and is not in the file, so it is a caller's input rather than a lookup.\n")

	candidates := make([]AreaID, 0, len(m.Areas))
	for _, a := range m.Areas {
		candidates = append(candidates, a.ID)
	}

	for _, disposable := range []bool{false, true} {
		who := "held sentry"
		if disposable {
			who = "gunslinger"
		}

		in := NestInputs{
			Target:      mannworksBomb,
			SentryRange: DefaultSentryRange,
			Approach:    approach,
			Disposable:  disposable,
		}
		_, all := m.BestNestArea(candidates, in)
		sortScores(all)

		fmt.Fprintf(b, "\nTop ten for a %s:\n", who)
		for _, s := range all[:10] {
			fmt.Fprintf(b, "  %s\n", s)
		}
	}
}
