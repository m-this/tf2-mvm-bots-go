package body

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/spbody"
)

/*
What a seat has to forget between bots

The shipped ResetNextBot was one flat list of every field the mod keeps per
client. Nobody could add a behaviour without finding that list and remembering
to add to it, and forgetting was silent: the next bot in the seat inherited the
last one's target. The port split it up, so each package clears its own and
ResetNextBot calls them, which is better and still not checked. A package can
declare a per-client array and clear none of it, and the failure is the same one
as before -- a bot that starts a wave believing something about the last bot in
its seat -- arriving through a different hole.

So this walks it. From ResetNextBot, through every call it makes, into the
packages those calls land in, collecting the per-client arrays that actually get
written. Anything declared and not in that set has to say so, with a reason, and
the reasons are worth reading: several of these are deliberate.

The reset itself stays hand-written. internal/spbody/state.go says why and the
reason holds: a generated reset is a function nobody proved, where a written one
is walked by the differential test like any other body. What was missing was not
the writing but the accounting.
*/

// keepDirective says a per-client array is deliberately not cleared between
// bots, and why. The reason is the rest of the line.
const keepDirective = "//sp:keep"

// slotVar is one per-client array a generated package declares.
type slotVar struct {
	Dir  string
	Name string
	SP   string // what //sp:name called it, which is what a slotset extern names
	Keep string // the reason it is not cleared, empty when it should be
	Pos  token.Position
}

/*
resetEntries are the places a seat is put back.

There is more than one, which is itself worth knowing. ResetNextBot is the one
the shipped plugin named and the one everybody thinks of, and it is not where
the shared globals are cleared: OnClientDisconnect is, because a seat that
empties has to forget its player as well as its bot. OnClientPutInServer is the
other side of the same door.

Nothing said this. The three were found by asking which function clears
g_bIsDefenderBot and following the answer.
*/
var resetEntries = []string{
	"internal/body/botreset@ResetNextBot",
	"internal/body/lifecycle@OnClientDisconnect",
	"internal/body/lifecycle@OnClientPutInServer",
}

/*
entries are the other places a per-client array is put back.

A behaviour's OnStart runs before its Update ever reads anything, and a new bot
in a seat gets a new behaviour, so a value written there cannot be the last
bot's. That is a real answer to the question and not an excuse: what it is not
an answer to is a value read by something outside the behaviour, which is why
this stops at OnStart and does not walk Update.
*/
func entries(pkgs map[string]*pkg) []string {
	out := slices.Clone(resetEntries)
	for _, a := range Actions {
		if p, generated := pkgs[a.Dir]; generated {
			if _, starts := p.Funcs["OnStart"]; starts {
				out = append(out, a.Dir+"@OnStart")
			}
		}
	}
	return out
}

/*
CheckResets reports every per-client array that ResetNextBot does not reach and
does not explain.

owned is what Generate already built, so the walk can follow a call into
internal/engine back to the package that generates it: that is how ResetNextBot
reaches ResetAttack, and the only way, because an extern is what a cross-package
call was before mvm-z83.84 and still is for these.
*/
func CheckResets(root string, declared spbody.Declared, owned map[string]owner) error {
	pkgs, err := readPackages(root)
	if err != nil {
		return err
	}
	cleared := clearedFrom(pkgs, declared, owned, entries(pkgs)...)

	var missing []slotVar
	for _, dir := range slices.Sorted(maps.Keys(pkgs)) {
		for _, v := range pkgs[dir].Slots {
			if cleared[dir+"."+v.Name] || v.Keep != "" || slices.Contains(unreviewed, dir+"."+v.Name) {
				continue
			}
			missing = append(missing, v)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	lines := make([]string, 0, len(missing))
	for _, v := range missing {
		lines = append(lines, fmt.Sprintf("\t%s: %s is per client and nothing ResetNextBot calls clears it", v.Pos, v.Name))
	}
	return fmt.Errorf("a seat would carry these from one bot to the next:\n%s\n"+
		"clear each in the package's own reset, or say why not with %s <reason>", strings.Join(lines, "\n"), keepDirective)
}

/*
unreviewed are the per-client arrays nothing clears, as they stood when this
check was written.

45 of them, and each is a question about the game rather than about the code: a
stuck watchdog holding the last bot's position, an attack strafe still mid-flip,
a medic's revive answer cached for somebody who left. Some are certainly
deliberate. Deciding which is not something the check can do, and guessing at it
would be a fix riding along with a port, which mvm-z83.41 is the standing
version of.

So they are listed rather than annotated, the way generatedElsewhere is listed:
one place, shrinking. Adding a per-client array to this list is not how a new
one gets past the check -- a new one gets cleared, or says //sp:keep with a
reason. This list only holds what was already here.

mvm-z83.91 is working through it.
*/
var unreviewed = []string{
	"internal/action/attack.strafeFlip",
	"internal/action/attack.strafeRight",
	"internal/action/engineerbuilddisposable.gaveUp",
	"internal/action/engineerbuildteleporter.mode",
	"internal/action/engineerbuildteleporter.spawnOf",
	"internal/action/engineerbuildteleporter.nestOf",
	"internal/action/engineerbuildteleporter.namedSpot",
	"internal/action/engineeridle.advanceAgain",
	"internal/action/engineeridle.carryDeadline",
	"internal/action/engineeridle.sentryUnderFire",
	"internal/action/engineeridle.stallReportAt",
	"internal/action/engineeridle.rangeRepairStalls",
	"internal/action/getammo.ammoAsk",
	"internal/action/getammo.ammoPossible",
	"internal/action/gethealth.healthAsk",
	"internal/action/gethealth.healthPossible",
	"internal/action/medicrevive.reviveAsk",
	"internal/action/medicrevive.revivePossible",
	"internal/action/spycheck.nextGlance",
	"internal/body/botnames.findNameTries",
	"internal/body/cosmetics.wardrobe",
	"internal/body/declarations.isBeingRevived",
	"internal/body/declarations.extraButtons",
	"internal/body/declarations.nextSnipeFireTime",
	"internal/body/loadouts.attribPrimary",
	"internal/body/loadouts.attribSecondary",
	"internal/body/loadouts.attribMelee",
	"internal/body/loadouts.attrValPrimary",
	"internal/body/loadouts.attrValSecondary",
	"internal/body/loadouts.attrValMelee",
	"internal/body/medicnudge.nextPatientNudge",
	"internal/body/pathing.pathFailed",
	"internal/body/pathing.pathFailures",
	"internal/body/pluginbot.pluginBot",
	"internal/body/readiness.readyDeadline",
	"internal/body/roster.defenderBot",
	"internal/body/scoutjump.scoutDoubleJumpTime",
	"internal/body/scoutjump.scoutDoubleJumpSide",
	"internal/body/stuckwatch.stuckOrigin",
	"internal/body/stuckwatch.stuckDeadline",
	"internal/body/stuckwatch.stuckCount",
	"internal/body/stuckwatch.stuckWedge",
	"internal/body/stuckwatch.stuckWedgeCount",
	"internal/body/stuckwatch.sniperStallDeadline",
	"internal/body/stuckwatch.sniperStalled",
}

// pkg is one generated package, read once: the per-client arrays it declares,
// and what each of its functions writes and calls.
type pkg struct {
	Dir   string
	Name  string
	Slots []slotVar
	Funcs map[string]*fn
}

// fn is one function: the per-client arrays it writes by index, and the
// functions it calls, qualified the way a call site writes them.
type fn struct {
	Writes []string
	Calls  []string
}

// readPackages parses every generated package for the two things the walk
// needs. Parsing and not type checking: a name is enough, because the emitted
// SourcePawn is one flat namespace and so is this.
func readPackages(root string) (map[string]*pkg, error) {
	out := make(map[string]*pkg, len(All)+len(Actions))
	for _, b := range slices.Concat(All, Actions) {
		p, err := readPackage(filepath.Join(root, b.Dir), b.Dir)
		if err != nil {
			return nil, err
		}
		out[b.Dir] = p
	}
	return out, nil
}

func readPackage(dir, rel string) (*pkg, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("body: reading %s: %w", dir, err)
	}
	fset := token.NewFileSet()
	p := &pkg{Dir: rel, Funcs: map[string]*fn{}}
	slots := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("body: parsing %s: %w", name, err)
		}
		p.Name = f.Name.Name
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				if d.Tok == token.VAR {
					readSlots(p, slots, d, fset, rel)
				}
			case *ast.FuncDecl:
				if d.Recv == nil && d.Body != nil {
					p.Funcs[d.Name.Name] = readFunc(d)
				}
			}
		}
	}
	return p, nil
}

// readSlots collects the package-level arrays sized by the shared client count,
// which is what "per client" means here.
func readSlots(p *pkg, seen map[string]bool, d *ast.GenDecl, fset *token.FileSet, rel string) {
	for _, spec := range d.Specs {
		vs, isValue := spec.(*ast.ValueSpec)
		if !isValue {
			continue
		}
		at, isArray := vs.Type.(*ast.ArrayType)
		if !isArray {
			continue
		}
		sel, qualified := at.Len.(*ast.SelectorExpr)
		if !qualified || sel.Sel.Name != "Count" {
			continue
		}
		keep := keepReason(vs.Doc, d.Doc)
		sp := spNameOf(vs.Doc, d.Doc)
		for _, id := range vs.Names {
			if id.Name == "_" || seen[id.Name] {
				continue
			}
			seen[id.Name] = true
			p.Slots = append(p.Slots, slotVar{Dir: rel, Name: id.Name, SP: sp, Keep: keep, Pos: fset.Position(id.Pos())})
		}
	}
}

// keepReason reads //sp:keep off a var declaration, from the spec's own doc or
// the group's, the way every other directive on a var is read.
func keepReason(docs ...*ast.CommentGroup) string {
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		for _, c := range doc.List {
			for line := range strings.Lines(c.Text) {
				fields := strings.Fields(line)
				if len(fields) > 1 && fields[0] == keepDirective {
					return strings.Join(fields[1:], " ")
				}
			}
		}
	}
	return ""
}

// spNameOf reads //sp:name off a var declaration. A per-client array carries one
// because the unported plugin still reads it under that name, and that is the
// name another package's slotset extern writes it by.
func spNameOf(docs ...*ast.CommentGroup) string {
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		for _, c := range doc.List {
			for line := range strings.Lines(c.Text) {
				fields := strings.Fields(line)
				if len(fields) == 2 && fields[0] == "//sp:name" {
					return fields[1]
				}
			}
		}
	}
	return ""
}

// readFunc records what one function writes by index and what it calls, both
// as written: resolving them is the walk's job, which has the tables.
func readFunc(d *ast.FuncDecl) *fn {
	out := &fn{}
	ast.Inspect(d.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			for _, lhs := range x.Lhs {
				if name, indexed := indexedName(lhs); indexed {
					out.Writes = append(out.Writes, name)
				}
			}
		case *ast.IncDecStmt:
			if name, indexed := indexedName(x.X); indexed {
				out.Writes = append(out.Writes, name)
			}
		case *ast.CallExpr:
			switch f := x.Fun.(type) {
			case *ast.Ident:
				out.Calls = append(out.Calls, f.Name)
			case *ast.SelectorExpr:
				if id, plain := f.X.(*ast.Ident); plain {
					out.Calls = append(out.Calls, id.Name+"."+f.Sel.Name)
				}
			}
		}
		return true
	})
	return out
}

// indexedName is the array a subscripted assignment writes, at whatever depth:
// m_iSpentOnUpgrade[client][index] is still a write of m_iSpentOnUpgrade.
func indexedName(e ast.Expr) (string, bool) {
	for {
		ix, indexed := e.(*ast.IndexExpr)
		if !indexed {
			return "", false
		}
		if id, plain := ix.X.(*ast.Ident); plain {
			return id.Name, true
		}
		e = ix.X
	}
}

/*
clearedFrom walks out from one function and collects every per-client array
anything it reaches writes.

Three kinds of call have to be followed and they resolve differently. A bare
name is this package's own. A qualified name into another generated package is
an ordinary import. A qualified name into internal/engine is the older form of
the same thing: the extern names SourcePawn this port generates, and owned says
which package generates it, so the call lands in a package either way.

Anything else is the engine, and the engine does not hold this mod's per-client
state.
*/
func clearedFrom(pkgs map[string]*pkg, declared spbody.Declared, owned map[string]owner, from ...string) map[string]bool {
	byName := make(map[string]*pkg, len(pkgs))
	// Every per-client array by the SourcePawn name it is written under, so
	// a slotset extern in one package can be matched to the array another
	// package declares. That is how the shared globals are cleared:
	// declarations owns g_bIsDefenderBot and lifecycle is what puts it back.
	bySP := map[string]string{}
	for _, p := range pkgs {
		byName[p.Name] = p
		for _, v := range p.Slots {
			if v.SP != "" {
				bySP[v.SP] = v.Dir + "." + v.Name
			}
		}
	}

	cleared := map[string]bool{}
	seen := map[string]bool{}
	queue := slices.Clone(from)
	for len(queue) > 0 {
		at := queue[0]
		queue = queue[1:]
		if seen[at] {
			continue
		}
		seen[at] = true

		dir, name, _ := strings.Cut(at, "@")
		p, generated := pkgs[dir]
		if !generated {
			continue
		}
		f, declaredHere := p.Funcs[name]
		if !declaredHere {
			continue
		}
		for _, w := range f.Writes {
			cleared[dir+"."+w] = true
		}
		for _, call := range f.Calls {
			// A slotset extern is a write and not a call: the
			// array it names belongs to whichever package declared
			// it, and reaching this call is clearing it.
			if x, isExtern := declared.Funcs[call]; isExtern && x.Slot && x.Set {
				if qualified, declaredHere := bySP[x.Func]; declaredHere {
					cleared[qualified] = true
				}
				continue
			}
			queue = append(queue, resolve(call, p, byName, declared, owned)...)
		}
	}
	return cleared
}

// resolve turns one written call site into the generated functions it may land
// in, keyed the way the walk holds them.
func resolve(call string, from *pkg, byName map[string]*pkg, declared spbody.Declared, owned map[string]owner) []string {
	qualifier, name, isQualified := strings.Cut(call, ".")
	if !isQualified {
		if _, here := from.Funcs[call]; here {
			return []string{from.Dir + "@" + call}
		}
		return nil
	}
	if other, generated := byName[qualifier]; generated {
		if _, there := other.Funcs[name]; there {
			return []string{other.Dir + "@" + name}
		}
	}
	// An extern: engine.ResetAttack names SourcePawn a package generates.
	x, isExtern := declared.Funcs[call]
	if !isExtern || !x.Body {
		return nil
	}
	o, ported := owned[x.Func]
	if !ported || o.Decl.Go == "" {
		return nil
	}
	return []string{o.Dir + "@" + o.Decl.Go}
}
