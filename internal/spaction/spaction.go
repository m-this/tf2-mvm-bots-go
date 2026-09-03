/*
Package spaction turns a Go package into a BehaviorAction subclass.

This is the shape, not the translation. Most of the plugin's line count is
action plumbing: a constructor that asks ActionsManager for an action and wires
five callbacks onto it, and then five functions with signatures the engine
decides. Thirty files under source/redbots3/behavior are that plumbing with a
different body in the middle.

The bodies come from internal/spbody, which is why this package is small. What
it owns is the constructor and the five signatures, because those are the engine
speaking and not the plugin: getting one parameter wrong is a callback the
engine enters with the arguments in the wrong registers.

A behaviour is a package carrying

	//sp:action DefenderCollectNearMoney CTFBotCollectNearMoney

on its package clause: the name ActionsManager registers it under, then the
prefix every emitted function carries, and optionally the word static, which
some of the shipped files declare their callbacks as. It declares whichever of OnStart, Update,
OnEnd, OnSuspend and OnResume it needs, and each of those may take any of the
parameters the engine passes that callback, by the engine's own name for it.
*/
package spaction

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/spbody"
)

// directive is the package comment that says this is a behaviour.
const directive = "//sp:action"

/*
	callback is one of the five the engine enters

The signature is written out rather than built, because it is the engine's and
not ours: actions_processors.inc declares OnStart, OnSuspend and OnResume with a
prior action, Update with an interval, and OnEnd returning nothing. A generated
declaration that disagreed would compile and then be entered with the arguments
in the wrong places.
*/
type callback struct {
	// Wire is the property the constructor assigns, which is the Go name
	// for all of them except Update.
	Wire string
	// Returns is the SourcePawn return type.
	Returns string
	// Params is the declaration, after the action itself.
	Params string
	// Names are the parameters a body may take, in the engine's names.
	Names []string
}

var callbacks = map[string]callback{
	// The two queries. The engine asks rather than tells, so the answer
	// comes back through a by-reference parameter and the return says only
	// whether the answer was changed at all.
	"ShouldAttack": {
		Wire: "ShouldAttack", Returns: "Action",
		Params: "INextBot nextbot, CKnownEntity knownEntity, QueryResultType& result",
		Names:  []string{"nextbot", "knownEntity", "result"},
	},
	// The engine telling a behaviour about ground it was holding.
	"OnTerritoryContested": {
		Wire: "OnTerritoryContested", Returns: "Action",
		Params: "int actor, int territory",
		Names:  []string{"actor", "territory"},
	},
	"OnTerritoryLost": {
		Wire: "OnTerritoryLost", Returns: "Action",
		Params: "int actor, int territory",
		Names:  []string{"actor", "territory"},
	},
	// The engine telling the behaviour something happened. The result is a
	// desired one rather than an outcome, and the behaviour usually just
	// carries on.
	"OnInjured": {
		Wire: "OnInjured", Returns: "Action",
		Params: "int actor, Address takedamageinfo, ActionDesiredResult result",
		Names:  []string{"actor", "takedamageinfo", "result"},
	},
	// The engine telling a behaviour the bot walked onto different ground.
	"OnNavAreaChanged": {
		Wire: "OnNavAreaChanged", Returns: "Action",
		Params: "int actor, CTFNavArea newArea, CTFNavArea oldArea, ActionDesiredResult result",
		Names:  []string{"actor", "newArea", "oldArea", "result"},
	},
	// The engine asking which of two threats matters more. The answer is the
	// by-reference one; the return only says the behaviour had an opinion.
	"SelectMoreDangerousThreat": {
		Wire: "SelectMoreDangerousThreat", Returns: "Action",
		Params: "INextBot nextbot, int entity, CKnownEntity threat1, CKnownEntity threat2, CKnownEntity& knownEntity",
		Names:  []string{"nextbot", "entity", "threat1", "threat2", "knownEntity"},
	},
	"ShouldHurry": {
		Wire: "ShouldHurry", Returns: "Action",
		Params: "INextBot nextbot, QueryResultType& result",
		Names:  []string{"nextbot", "result"},
	},
	"IsHindrance": {
		Wire: "IsHindrance", Returns: "Action",
		Params: "INextBot nextbot, int entity, QueryResultType& result",
		Names:  []string{"nextbot", "entity", "result"},
	},
	// The engine telling a behaviour the bot arrived where it was walking.
	// The path is any because the include's typeset says so: a PathFollower
	// and a Path are both passed to it.
	"OnMoveToSuccess": {
		Wire: "OnMoveToSuccess", Returns: "Action",
		Params: "int actor, any path, ActionDesiredResult result",
		Names:  []string{"actor", "path", "result"},
	},
	"OnStart": {
		Wire: "OnStart", Returns: "Action",
		Params: "int actor, BehaviorAction priorAction, ActionResult result",
		Names:  []string{"actor", "priorAction", "result"},
	},
	"Update": {
		Wire: "Update", Returns: "Action",
		Params: "int actor, float interval, ActionResult result",
		Names:  []string{"actor", "interval", "result"},
	},
	// The include's typeset calls OnEnd's third parameter nextAction and the
	// plugin calls it priorAction. It is positional and the name is a label,
	// so this takes the plugin's: the port is compared against what ships.
	"OnEnd": {
		Wire: "OnEnd", Returns: "void",
		Params: "int actor, BehaviorAction priorAction, ActionResult result",
		Names:  []string{"actor", "priorAction", "result"},
	},
	"OnSuspend": {
		Wire: "OnSuspend", Returns: "Action",
		Params: "int actor, BehaviorAction priorAction, ActionResult result",
		Names:  []string{"actor", "priorAction", "result"},
	},
	"OnResume": {
		Wire: "OnResume", Returns: "Action",
		Params: "int actor, BehaviorAction priorAction, ActionResult result",
		Names:  []string{"actor", "priorAction", "result"},
	},
}

// order is the order the constructor wires them in, which is the order the
// plugin writes and so the order a reviewer expects.
var order = []string{
	"OnStart", "Update", "OnEnd", "OnSuspend", "OnResume", "OnMoveToSuccess",
	"OnInjured", "OnNavAreaChanged", "OnTerritoryContested", "OnTerritoryLost",
	"SelectMoreDangerousThreat", "ShouldHurry", "ShouldAttack", "IsHindrance",
}

// Action is one behaviour: what it is registered as, what its functions are
// called, and which callbacks it has.
type Action struct {
	// Registered is the name ActionsManager.Create is given, which is what
	// the action reports itself as in the game's own debug output.
	Registered string
	// Prefix is what every emitted function is called, before the
	// underscore: CTFBotCollectNearMoney.
	Prefix string
	// Has are the callbacks the package declares, in wiring order.
	Has []string
	// Static says the callbacks are declared static rather than public,
	// which some of the shipped files do. Both work: the constructor that
	// takes their address is in the same file. It is carried so the port
	// reads as what it replaces.
	Static bool
}

// Generate emits the constructor and the callbacks of the action package in dir,
// and what it declares, so a caller can hold that against what is still an
// extern.
func Generate(dir string, cfg spbody.Config) (source string, declares []spbody.Declaration, err error) {
	action, err := Read(dir)
	if err != nil {
		return "", nil, err
	}

	cfg.Declare = make(map[string]string, len(action.Has))
	for _, name := range action.Has {
		c := callbacks[name]
		visibility := "public"
		if action.Static {
			visibility = "static"
		}
		cfg.Declare[name] = fmt.Sprintf("%s %s %s_%s(BehaviorAction action, %s)",
			visibility, c.Returns, action.Prefix, name, c.Params)
	}

	generated, err := spbody.GenerateDir(dir, cfg)
	if err != nil {
		return "", nil, err
	}

	var b strings.Builder
	b.WriteString(constructor(action))
	b.WriteString(afterBanner(generated.Source))

	/*
		A callback is emitted under the name the constructor wires, not
		under the one the body generator would have given it: cfg.Declare
		replaced the whole declaration line, prefix included. Its
		SourcePawn signature was replaced with it, so there is no Go
		signature left to hold an extern against, and it carries none.

		Nothing outside calls one anyway. A behaviour that wants this one
		hands the engine its constructor.
	*/
	callback := make(map[string]bool, len(action.Has))
	for _, name := range action.Has {
		callback[name] = true
	}
	for _, d := range generated.Declares {
		if callback[d.Go] {
			d.SP, d.Sig = action.Prefix+"_"+d.Go, nil
		}
		declares = append(declares, d)
	}
	// The constructor is a name this file owns too, and the one another
	// behaviour reaches for when it hands the engine this one. It is written
	// here rather than translated, so it has no Go signature either.
	declares = append(declares, spbody.Declaration{SP: action.Prefix})
	return b.String(), declares, nil
}

// constructor is the function the plugin calls to get one of these.
func constructor(a Action) string {
	var b strings.Builder
	fmt.Fprintf(&b, "BehaviorAction %s()\n{\n", a.Prefix)
	fmt.Fprintf(&b, "\tBehaviorAction action = ActionsManager.Create(%q);\n\n", a.Registered)
	for _, name := range a.Has {
		fmt.Fprintf(&b, "\taction.%s = %s_%s;\n", callbacks[name].Wire, a.Prefix, name)
	}
	b.WriteString("\n\treturn action;\n}\n\n")
	return b.String()
}

// afterBanner drops the body generator's banner, because the file already has
// one from this package.
func afterBanner(source string) string {
	if _, rest, found := strings.Cut(source, "*/\n\n"); found {
		return rest
	}
	return source
}

// Read parses the action package's directive and finds its callbacks.
func Read(dir string) (Action, error) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Action{}, fmt.Errorf("spaction: reading %s: %w", dir, err)
	}

	var a Action
	found := false
	var has []string
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if err != nil {
			return Action{}, fmt.Errorf("spaction: parsing %s: %w", name, err)
		}
		if declared, ok := parseDirective(file.Doc); ok {
			if found {
				return Action{}, fmt.Errorf("spaction: %s declares a second //sp:action", name)
			}
			a, found = declared, true
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			if _, isCallback := callbacks[fn.Name.Name]; isCallback {
				has = append(has, fn.Name.Name)
			}
		}
	}

	if !found {
		return Action{}, fmt.Errorf("spaction: %s has no //sp:action on its package clause", dir)
	}
	if len(has) == 0 {
		return Action{}, fmt.Errorf("spaction: %s declares no callback; an action that does nothing is not one", dir)
	}
	a.Has = slices.DeleteFunc(slices.Clone(order), func(name string) bool {
		return !slices.Contains(has, name)
	})
	return a, nil
}

func parseDirective(doc *ast.CommentGroup) (Action, bool) {
	if doc == nil {
		return Action{}, false
	}
	// Line by line inside each comment, because a package comment is one
	// /* */ block and the directive is a line in the middle of it.
	for _, c := range doc.List {
		for line := range strings.Lines(c.Text) {
			fields := strings.Fields(line)
			if len(fields) == 3 && fields[0] == directive {
				return Action{Registered: fields[1], Prefix: fields[2]}, true
			}
			if len(fields) == 4 && fields[0] == directive && fields[3] == "static" {
				return Action{Registered: fields[1], Prefix: fields[2], Static: true}, true
			}
		}
	}
	return Action{}, false
}
