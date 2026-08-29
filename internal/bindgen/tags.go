package bindgen

import (
	"maps"
	"slices"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/bindings"
)

// cellTypes are the SourcePawn primitives. Everything else a declaration
// names is a tag some include is expected to declare.
var cellTypes = map[string]bool{
	"int": true, "float": true, "bool": true, "char": true,
	"any": true, "void": true,
}

// undeclaredTags is every tag the tree's declarations name and none of them
// declares. They are the SourcePawn compiler's own tags plus, when the list
// grows, the sign that a declaration was refused somewhere upstream.
func undeclaredTags(parsed map[string]*bindings.File, order []string) []string {
	declared := map[string]bool{}
	for _, rel := range order {
		for name := range declaredTypes(parsed[rel]) {
			declared[name] = true
		}
	}
	missing := map[string]bool{}
	for _, rel := range order {
		for name := range referencedTags(parsed[rel]) {
			if !declared[name] && !cellTypes[name] {
				missing[goIdent(name)] = true
			}
		}
	}
	return slices.Sorted(maps.Keys(missing))
}

// declaredTypes lists the tag names one include declares.
func declaredTypes(f *bindings.File) map[string]bool {
	out := map[string]bool{}
	for _, mm := range f.Methodmaps {
		out[mm.Name] = true
	}
	for _, en := range f.Enums {
		if en.Name != "" {
			out[en.Name] = true
		}
	}
	for _, es := range f.EnumStructs {
		out[es.Name] = true
	}
	for _, td := range f.Typedefs {
		out[td.Name] = true
	}
	for _, ts := range f.Typesets {
		out[ts.Name] = true
	}
	return out
}

// referencedTags lists the tag names one include's declarations mention, in
// any position the emitter renders as a Go type.
func referencedTags(f *bindings.File) map[string]bool {
	out := map[string]bool{}
	add := func(t bindings.Type) {
		if t.Name != "" {
			out[t.Name] = true
		}
	}
	addSig := func(params []bindings.Param, ret bindings.Type) {
		add(ret)
		for _, pm := range params {
			add(pm.Type)
		}
	}
	for _, n := range slices.Concat(f.Natives, f.Stocks) {
		addSig(n.Params, n.Return)
	}
	for _, mm := range f.Methodmaps {
		if mm.Parent != "" {
			out[mm.Parent] = true
		}
		for _, m := range mm.Methods {
			addSig(m.Params, m.Return)
		}
		for _, pr := range mm.Properties {
			add(pr.Type)
		}
	}
	for _, es := range f.EnumStructs {
		for _, fl := range es.Fields {
			add(fl.Type)
		}
		for _, m := range es.Methods {
			addSig(m.Params, m.Return)
		}
	}
	for _, td := range f.Typedefs {
		addSig(td.Params, td.Return)
	}
	for _, ts := range f.Typesets {
		for _, v := range ts.Variants {
			addSig(v.Params, v.Return)
		}
	}
	return out
}

// goIdent mirrors the renaming the emitter applies, so a tag declared here
// carries the name the emitted references use.
func goIdent(name string) string { return strings.ReplaceAll(name, "@", "_") }
