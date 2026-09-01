package bindgen

import (
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/plugin"
)

// includeRoot is the include tree inside the plugin's test-bed build
// directory. The plugin repository is resolved by internal/upstream, which is
// where a relative MVMBOTS_UPSTREAM is read from the repository root rather
// than from this package: doing it here got it wrong, and every proof below
// skipped in silence.
func includeRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(pluginDir(t), "testbed", "build")
}

func pluginDir(t *testing.T) string {
	t.Helper()

	return plugin.SkipOrFail(t)
}

// generate runs the driver over the real tree, skipping when it is absent.
func generate(t *testing.T) *Result {
	t.Helper()
	root := includeRoot(t)
	if _, err := os.Stat(root); err != nil {
		t.Skipf("include tree not present: %v", err)
	}
	res, err := Generate(Options{Root: root, Package: "sp"})
	if err != nil {
		t.Fatalf("generating: %v", err)
	}
	return res
}

// TestGeneratedPackageBuilds is the test that matters. go/types over one
// emitted string proves the emitter writes Go; only the compiler over the
// whole directory proves the driver writes a package. Nothing here is edited
// by hand between generation and the build.
func TestGeneratedPackageBuilds(t *testing.T) {
	res := generate(t)
	dir := t.TempDir()
	pkg := filepath.Join(dir, "sp")
	if err := Write(pkg, res); err != nil {
		t.Fatalf("writing: %v", err)
	}
	gomod := "module example.test/sp\n\ngo 1.26\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build over the generated package failed: %v\n%s", err, out)
	}
	t.Logf("built %d files, %d refusals, builtin tags: %s",
		len(res.Files), len(res.Refusals), strings.Join(res.BuiltinTags, " "))
}

// TestGenerateIsReproducible pins the ordering: the driver threads state from
// one file into the next, so a run whose order moved would emit different
// constants for the same tree.
func TestGenerateIsReproducible(t *testing.T) {
	first, second := generate(t), generate(t)
	if !slices.Equal(first.Order, second.Order) {
		t.Fatal("emission order is not deterministic")
	}
	for name, body := range first.Files {
		other, ok := second.Files[name]
		if !ok {
			t.Fatalf("%s missing from the second run", name)
		}
		if string(body) != string(other) {
			t.Errorf("%s differs between two runs of the same tree", name)
		}
	}
}

// TestOrderingResolvesCrossFileConstants holds the driver to its reason for
// existing. Threading each file's constants into the next is what turns a
// #define whose body names another include's constant from a refusal into a
// value.
func TestOrderingResolvesCrossFileConstants(t *testing.T) {
	res := generate(t)
	defines := 0
	for _, r := range res.Refusals {
		if r.Kind == "define" && strings.Contains(r.Reason, "constant expression") {
			defines++
		}
	}
	t.Logf("defines refused as non-constant: %d", defines)
	if defines > 25 {
		t.Errorf("define refusals = %d, want the threading to keep it at or under 25", defines)
	}
}

// TestBuiltinTagsStaySmall guards the one place the driver declares something
// the includes do not. A jump here is a refused declaration upstream showing
// up as an invented type, which is exactly the guess this generator refuses
// to make silently.
func TestBuiltinTagsStaySmall(t *testing.T) {
	res := generate(t)
	t.Logf("builtin tags (%d): %s", len(res.BuiltinTags), strings.Join(res.BuiltinTags, " "))
	if len(res.BuiltinTags) > 12 {
		t.Errorf("builtin tags = %d, want at most 12: %v", len(res.BuiltinTags), res.BuiltinTags)
	}
}

// TestCrossFileTypesResolve names the edges mvm-z83.20 opened with. Each is
// declared by an include other than the one that needs it, and the package
// has to carry all of them.
func TestCrossFileTypesResolve(t *testing.T) {
	res := generate(t)
	for _, name := range []string{"TFCond", "TFClassType", "Address", "ActionHandler", "ActionResult"} {
		t.Run(name, func(t *testing.T) {
			if !res.Declared(name) {
				t.Errorf("%s is not declared by the generated package", name)
			}
		})
	}
}

// TestGoFileName covers the flattening that lets two includes with the same
// base name share one package directory.
func TestGoFileName(t *testing.T) {
	for _, tc := range []struct {
		rel  string
		want string
	}{
		{"spcomp/addons/sourcemod/scripting/include/tf2.inc", "spcomp_addons_sourcemod_scripting_include_tf2.go"},
		{"src/actions/sourcemod/include/actions.inc", "src_actions_sourcemod_include_actions.go"},
		{"ripext/addons/sourcemod/scripting/include/ripext/http.inc", "ripext_addons_sourcemod_scripting_include_ripext_http.go"},
		{"a-b/c.d.inc", "a_b_c_d.go"},
	} {
		t.Run(tc.rel, func(t *testing.T) {
			if got := goFileName(tc.rel); got != tc.want {
				t.Errorf("goFileName(%q) = %q, want %q", tc.rel, got, tc.want)
			}
		})
	}
}

// TestTopoSort covers the ordering in isolation, including the cycle that
// SourcePawn include graphs really contain.
func TestTopoSort(t *testing.T) {
	for _, tc := range []struct {
		name  string
		nodes []string
		deps  map[string][]string
		want  []string
	}{
		{
			name:  "chain",
			nodes: []string{"c", "b", "a"},
			deps:  map[string][]string{"c": {"b"}, "b": {"a"}},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "independent stays in path order",
			nodes: []string{"a", "b", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "cycle appended in path order",
			nodes: []string{"a", "b", "c"},
			deps:  map[string][]string{"b": {"c"}, "c": {"b"}, "a": nil},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "dependant after its dependency across a cycle",
			nodes: []string{"a", "b", "z"},
			deps:  map[string][]string{"a": {"b"}, "b": {"a"}, "z": {"a"}},
			want:  []string{"a", "b", "z"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := topoSort(tc.nodes, tc.deps)
			if !slices.Equal(got, tc.want) {
				t.Errorf("topoSort = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestResolvePrefersTheNearestCopy covers include resolution when the tree
// ships the same header under two roots.
func TestResolvePrefersTheNearestCopy(t *testing.T) {
	index := map[string][]string{}
	for _, rel := range []string{
		"spcomp/include/tf2.inc",
		"src/cbasenpc/include/tf2.inc",
	} {
		for suffix := rel; suffix != ""; suffix = trimFirstSegment(suffix) {
			index[suffix] = append(index[suffix], rel)
		}
	}
	for _, tc := range []struct{ from, want string }{
		{"src/cbasenpc/include/other.inc", "src/cbasenpc/include/tf2.inc"},
		{"spcomp/include/other.inc", "spcomp/include/tf2.inc"},
	} {
		t.Run(tc.from, func(t *testing.T) {
			got, ok := resolve(index, tc.from, "tf2.inc")
			if !ok || got != tc.want {
				t.Errorf("resolve(%q) = %q %v, want %q", tc.from, got, ok, tc.want)
			}
		})
	}
}

// TestUnbound reports the parity edge: what the plugin uses that the
// generated package does not declare. The bound is deliberately loose. The
// number is read, not asserted; the assertion is only that it has not moved
// far enough to mean the driver stopped emitting.
func TestUnbound(t *testing.T) {
	res := generate(t)
	root := filepath.Join(pluginDir(t), "source")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("plugin source not present: %v", err)
	}
	rep, err := Unbound(root, res)
	if err != nil {
		t.Fatalf("unbound: %v", err)
	}
	t.Logf("free call sites %d, unbound %d: %s",
		rep.Calls, len(rep.Natives), strings.Join(rep.Natives, " "))
	t.Logf("member uses %d, unbound %d: %s",
		rep.MemberUses, len(rep.Members), strings.Join(rep.Members, " "))
	if len(rep.Natives) > 5 || len(rep.Members) > 5 {
		t.Errorf("unbound natives = %d, members = %d; want both at or under 5",
			len(rep.Natives), len(rep.Members))
	}
}

// TestRefusalsAreAccountedFor keeps the refusal set readable and pins its
// size. A refusal is not noise to filter: it is a declaration somebody has to
// write by hand, so the total is the generator's honest edge.
func TestRefusalsAreAccountedFor(t *testing.T) {
	res := generate(t)
	byReason := res.RefusalsByReason()
	for _, reason := range slices.Sorted(maps.Keys(byReason)) {
		t.Logf("%4d  %s", byReason[reason], reason)
	}
	if len(res.Refusals) > 150 {
		t.Errorf("refusals = %d, want the known set to stay small", len(res.Refusals))
	}
}
