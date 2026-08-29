package navmesh

import (
	"fmt"
	"slices"
	"strings"
)

// Severity is how badly a finding hurts, in the maintainer's own three steps.
//
// The distinction is the point of the table. A config that quietly does nothing
// reads exactly like one that works, and until the reading says which of these
// it is, everything gets the same attention and so nothing does.
type Severity uint8

// The three severities, least first.
const (
	// SeverityNote is a fact worth reading that may well be deliberate.
	SeverityNote Severity = iota

	// SeverityDegraded is a spot that is not honoured: the bot arrives
	// somewhere, so nothing stalls and nothing reports, but it is not the
	// place the config named.
	SeverityDegraded

	// SeverityBroken is a spot the bot cannot arrive at, or a behaviour the map
	// declares nothing for. It shipped doing nothing.
	SeverityBroken
)

// String names the severity for a report line.
func (s Severity) String() string {
	switch s {
	case SeverityNote:
		return "note"
	case SeverityDegraded:
		return "degraded"
	case SeverityBroken:
		return "broken"
	default:
		return "?"
	}
}

// Rule is one invariant a map config is held to.
type Rule uint8

// Every rule the table checks. Each names the bead it came from, because each
// was a bug that shipped.
const (
	// RuleNoSpots is a map that declares no spot at all of a kind the mod needs
	// (mvm-tz9: Decoy shipped with zero sniper spots for releases).
	RuleNoSpots Rule = iota

	// RuleTooFewExits is a map that declares teleporter exits but fewer of them
	// than its composition has engineers, so the rest are ring-placed with no
	// drop check (mvm-1pq).
	RuleTooFewExits

	// RuleSpotOffMesh is a spot with no nav surface within a snap of it
	// (mvm-0dn: Rottenburg SniperSpot 4, 213 units from anything).
	RuleSpotOffMesh

	// RuleSpotInHole is a spot in a ground-level hole in the mesh, with nothing
	// lower to snap down to (mvm-dx2).
	RuleSpotInHole

	// RuleSpotRelocated is a spot whose every accepted stand side is a storey
	// below it, so the building goes down elsewhere and nothing says so
	// (mvm-wxp, mvm-fgs).
	RuleSpotRelocated

	// RuleSpotBesideFall is a declared spot with a hurting fall beside it
	// (mvm-1yo, mvm-0am).
	RuleSpotBesideFall

	// RuleExitRingBesideFall is a nest whose fallback exit ring puts a player
	// beside a hurting fall (mvm-1yo).
	RuleExitRingBesideFall
)

// Rules is every rule, in declaration order, so a report and a test can walk the
// set rather than a list somebody remembered to update.
var Rules = []Rule{
	RuleNoSpots,
	RuleTooFewExits,
	RuleSpotOffMesh,
	RuleSpotInHole,
	RuleSpotRelocated,
	RuleSpotBesideFall,
	RuleExitRingBesideFall,
}

// String is the rule's stable short name, which is what a report groups by and
// a test writes down.
func (r Rule) String() string {
	switch r {
	case RuleNoSpots:
		return "no-spots"
	case RuleTooFewExits:
		return "too-few-exits"
	case RuleSpotOffMesh:
		return "spot-off-mesh"
	case RuleSpotInHole:
		return "spot-in-hole"
	case RuleSpotRelocated:
		return "spot-relocated"
	case RuleSpotBesideFall:
		return "spot-beside-fall"
	case RuleExitRingBesideFall:
		return "exit-ring-beside-fall"
	default:
		return "?"
	}
}

// Severity is how much this rule's findings matter.
func (r Rule) Severity() Severity {
	switch r {
	case RuleNoSpots, RuleSpotOffMesh, RuleSpotInHole:
		return SeverityBroken
	case RuleTooFewExits, RuleSpotRelocated:
		return SeverityDegraded
	case RuleSpotBesideFall, RuleExitRingBesideFall:
		return SeverityNote
	default:
		return SeverityNote
	}
}

// NeedsMesh reports whether the rule can only be read with a nav mesh in hand.
// The counting rules cannot, which is why they are the only thing this package
// can say about the twenty community maps whose .nav files are not shipped.
func (r Rule) NeedsMesh() bool {
	return r != RuleNoSpots && r != RuleTooFewExits
}

// Finding is one rule broken by one config, naming the entry a person would
// edit and the measurement that decided it.
type Finding struct {
	Map  string
	Rule Rule

	// Spot is the declared entry at fault, zero for a rule about the file as a
	// whole.
	Spot Spot

	// Detail is the numbers behind the verdict, in the rule's own terms.
	Detail string
}

// String is the finding as one report line.
func (f Finding) String() string {
	where := "the config"
	if f.Spot.Kind != "" {
		where = string(f.Spot.Kind) + " " + f.Spot.Index
	}
	return fmt.Sprintf("%-16s %-8s %-22s %-16s %s",
		f.Map, f.Rule.Severity(), f.Rule, where, f.Detail)
}

// RequiredKinds is the spot lists a map has to fill itself, because nothing at
// runtime fills them for it.
//
// Only the sniper. SetupSniperSpotHints in tf2_defenderbots.sp switches the
// map's own func_tfbot_hint entities off when the config names none, so a map
// with no sniper spot list has no sniper spots at all, which is mvm-tz9. The
// nest and exit blocks are overrides on top of a runtime search: a map that
// declares none gets the nest scorer and the bot_hint_engineer_nest entities,
// so their absence is a choice rather than a fault, and twenty-one of the
// twenty-seven shipped configs make it.
var RequiredKinds = []SpotKind{SniperSpot}

// CheckConfig reads one map config against the mesh and returns everything it
// finds, in rule order.
//
// The mesh may be nil, and then only the rules that need no mesh are read. That
// is not a degraded mode to apologise for: two of the bugs this table exists for
// are counting bugs, and counting works on a config for a map we do not have.
func CheckConfig(m *Mesh, c *MapConfig) []Finding {
	var out []Finding

	for _, kind := range RequiredKinds {
		if len(c.SpotsOf(kind)) == 0 {
			out = append(out, Finding{
				Map:    c.Map,
				Rule:   RuleNoSpots,
				Detail: fmt.Sprintf("no %s is declared", kind),
			})
		}
	}

	// Only for a map that declares exits at all. A map that declares none has
	// asked for the ring fallback everywhere; a map that declares one and runs
	// two engineers has silently given the second one the fallback, which is
	// mvm-1pq.
	if exits := len(c.SpotsOf(TeleporterExit)); exits > 0 && c.Engineers() > exits {
		out = append(out, Finding{
			Map:  c.Map,
			Rule: RuleTooFewExits,
			Detail: fmt.Sprintf("%d engineers in the composition, %d exits declared",
				c.Engineers(), exits),
		})
	}

	if m == nil {
		return out
	}

	for _, s := range c.Spots {
		out = append(out, checkSpot(m, c.Map, s)...)
	}

	slices.SortStableFunc(out, func(a, b Finding) int { return int(a.Rule) - int(b.Rule) })

	return out
}

func checkSpot(m *Mesh, mapName string, s Spot) []Finding {
	var out []Finding

	find := func(r Rule, detail string) {
		out = append(out, Finding{Map: mapName, Rule: r, Spot: s, Detail: detail})
	}

	p := m.CheckPoint(s.Origin)
	snap := m.CheckSnap(s, s.Origin, BuildTryPoints, BuildReach, HalfHumanHeight)

	switch p.Footing {
	case FootingOffMesh:
		find(RuleSpotOffMesh, fmt.Sprintf("nearest surface %.0f away, %d of %d stand sides accepted",
			p.NearestDistance, snap.Accepted, len(snap.Sides)))
	case FootingPocket:
		if s.Kind.IsGround() {
			find(RuleSpotInHole, fmt.Sprintf("no area under it, %.0f over the ground round it, nearest surface %.0f away",
				height(p.SurroundHeight), p.NearestDistance))
		}
	case FootingGround, FootingRaised:
	}

	if s.Kind.IsGround() && snap.Relocated() {
		find(RuleSpotRelocated, fmt.Sprintf("%.0f over the ground round it, all %d accepted stand sides %.0f to %.0f below it",
			height(p.SurroundHeight), snap.Accepted, snap.LeastDrop, snap.WorstDrop))
	}

	if s.Kind == EngineerNest || s.Kind == TeleporterExit {
		if d := m.CheckDrop(s, ExitDropRadius, HalfHumanHeight); d.Hurts() {
			find(RuleSpotBesideFall, fmt.Sprintf("%.0f down, %s", d.Worst.Descent, killOrHurt(d)))
		}
	}

	if s.Kind == EngineerNest {
		if worst, ok := ringWorstFall(m, s.Origin); ok {
			find(RuleExitRingBesideFall, fmt.Sprintf("a fallback exit side has %.0f down beside it", worst))
		}
	}

	return out
}

func killOrHurt(d DropVerdict) string {
	if d.Kills() {
		return "which kills a light class"
	}
	return "which hurts"
}

// ExitRingRadius is TELEPORTER_EXIT_RADIUS_SAFE, BUSTER_BLAST_RANGE plus a
// hundred, from engineerbuildteleporter.sp. It is how far out from his nest the
// engineer ring-places the exit when the named spot beats him.
const ExitRingRadius float32 = 500

// ringWorstFall is the deepest hurting fall beside any side of the fallback exit
// ring round a nest, and whether there is one at all. Nothing in the plugin vets
// these sides, which is mvm-1yo.
func ringWorstFall(m *Mesh, nest Vec3) (float32, bool) {
	worst := float32(0)
	found := false

	for side := range BuildTryPoints {
		sp := m.BuildStandPoint(nest, nest, side, BuildTryPoints, ExitRingRadius)
		if !sp.OK() {
			continue
		}
		d := m.CheckDrop(Spot{Kind: "ExitRing", Origin: sp.Stand}, ExitDropRadius, HalfHumanHeight)
		if d.Hurts() && (!found || d.Worst.Descent > worst) {
			worst, found = d.Worst.Descent, true
		}
	}

	return worst, found
}

// Engineers is how many engineers the composition asks for, which is how many
// nests and exits the map has to feed. It is the fallback lineup rather than a
// guarantee, because players choosing their own classes override it.
func (c *MapConfig) Engineers() int {
	n := 0
	for _, class := range c.Composition {
		if strings.EqualFold(class, "engineer") {
			n++
		}
	}
	return n
}
