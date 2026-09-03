package body_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
	"github.com/m-this/tf2-mvm-bots-go/internal/spbody"
)

/*
TestABodyCallsAnotherBodyByItsSourcePawnName is the shape this rests on.

internal/action/attack asks whether camping the bomb is possible. It used to ask
through an extern in internal/engine that restated campbomb.IsPossible's
signature by hand; it imports internal/action/campbomb and calls the function
now. The emitted SourcePawn has to be the same either way, because the two
languages are the same call: what changed is who typed the name.
*/
func TestABodyCallsAnotherBodyByItsSourcePawnName(t *testing.T) {
	out, err := body.Generate("../..")
	if err != nil {
		t.Fatalf("generating the bodies: %v", err)
	}
	attack, emitted := string(out["sourcepawn/attack.sp"]), "CTFBotCampBomb_IsPossible("
	if !strings.Contains(attack, emitted) {
		t.Errorf("attack.sp does not call %s; a cross-package call came out as something else", emitted)
	}
	if strings.Contains(attack, "campbomb.IsPossible") || strings.Contains(attack, "Go_IsPossible") {
		t.Error("attack.sp calls the Go name; the emitted name was not resolved")
	}
}

// TestNoExternNamesAFunctionAnImportReaches is the other half: the extern the
// import replaced has to be gone, or both would exist and nothing would say
// which the caller meant.
func TestNoExternNamesAFunctionAnImportReaches(t *testing.T) {
	declared, err := spbody.ExternsFromDir("../engine")
	if err != nil {
		t.Fatalf("reading the extern declarations: %v", err)
	}
	if _, still := declared.Funcs["engine.CampBombIsPossible"]; still {
		t.Error("engine.CampBombIsPossible is still declared beside the import that replaced it")
	}
}

/*
TestEveryGeneratedPackageIsImportable proves the table is the registry.

A package a body may import is one this repository emits a file for. Anything
else would be a call to SourcePawn that is never written, which compiles here
and fails at spcomp, a long way from the import that caused it.
*/
func TestEveryGeneratedPackageIsImportable(t *testing.T) {
	cfg, err := body.SubsetConfig("../..")
	if err != nil {
		t.Fatalf("reading the subset config: %v", err)
	}
	const module = "github.com/m-this/tf2-mvm-bots-go/"
	for _, b := range slices.Concat(body.All, body.Actions) {
		if _, importable := cfg.Packages[module+b.Dir]; !importable {
			t.Errorf("%s is generated and cannot be imported", b.Dir)
		}
	}
	for p := range cfg.Packages {
		if p == "math" || p == module+body.ExternDir {
			continue
		}
		dir := strings.TrimPrefix(p, module)
		if !slices.ContainsFunc(slices.Concat(body.All, body.Actions), func(b body.Body) bool { return b.Dir == dir }) {
			t.Errorf("%s may be imported and is not in the registry", p)
		}
	}
}

// TestOnlyExportedNamesCrossAPackage keeps a helper a helper. SourcePawn has no
// visibility and Go does, so the Go side is where the line gets drawn: a
// lower-case function is this package's business.
func TestOnlyExportedNamesCrossAPackage(t *testing.T) {
	exports, err := spbody.Exports("../action/campbomb", "Go_")
	if err != nil {
		t.Fatalf("reading the exports: %v", err)
	}
	if len(exports) == 0 {
		t.Fatal("campbomb offers nothing")
	}
	for _, e := range exports {
		if e.Go[0] < 'A' || e.Go[0] > 'Z' {
			t.Errorf("%s is offered and is not exported", e.Go)
		}
	}
	if !slices.ContainsFunc(exports, func(e spbody.Export) bool {
		return e.Go == "IsPossible" && e.SP == "CTFBotCampBomb_IsPossible"
	}) {
		t.Errorf("IsPossible is not offered as CTFBotCampBomb_IsPossible; got %v", exports)
	}
}

/*
TestOneDeclarationPerSharedConstant is what the slots package is for.

SourcePawn writes a constant as a #define, in one flat namespace, so a constant
declared in 41 packages is 41 defines of the same name. spcomp warned about the
redefinition on 39 of them, on every build, for as long as the port existed.

A constant folds at the call site now, so the only file that may write the name
is the one that declared it, and no generated file writes this one at all.
*/
func TestOneDeclarationPerSharedConstant(t *testing.T) {
	out, err := body.Generate("../..")
	if err != nil {
		t.Fatalf("generating the bodies: %v", err)
	}
	for name, source := range out {
		if strings.Contains(string(source), "#define Go_Count") {
			t.Errorf("%s defines the shared client array size; it should have folded", name)
		}
	}
	attack := string(out["sourcepawn/attack.sp"])
	if !strings.Contains(attack, "int m_iAttackTarget[65];") {
		t.Error("attack.sp does not size its client array 65; the constant did not fold")
	}
}
