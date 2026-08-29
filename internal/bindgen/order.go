package bindgen

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// includeDirective matches `#include <x>`, `#include "x"` and the tryinclude
// forms. The path inside is what one include says it needs.
var includeDirective = regexp.MustCompile(`(?m)^\s*#\s*try?include\s*[<"]([^>"]+)[>"]`)

// includeFiles lists every .inc under root, tree-relative and sorted.
//
// prebuilt/ is skipped: it is a copy of src/, declaration for declaration,
// and emitting both would refuse the second copy of every name in it.
func includeFiles(root string) ([]string, error) {
	if root == "" {
		return nil, fmt.Errorf("bindgen: no include root")
	}
	var rels []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "prebuilt" {
			return fs.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(p, ".inc") {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rels = append(rels, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("bindgen: walking %s: %w", root, err)
	}
	slices.Sort(rels)
	return rels, nil
}

// includeOrder sorts the tree so a file is emitted after the files it
// includes. That is what lets an enum initialiser and a #define body fold
// against constants an earlier file resolved.
//
// The graph is read from the #include lines rather than guessed from paths. A
// file whose includes cannot all be resolved still gets emitted; it just
// carries no incoming edge for the unresolved one. SourcePawn include graphs
// contain cycles, so the cyclic remainder is appended in path order, which is
// what the whole tree used to be.
func includeOrder(root string, rels []string) ([]string, error) {
	index := make(map[string][]string, len(rels)*2)
	for _, rel := range rels {
		for suffix := rel; suffix != ""; suffix = trimFirstSegment(suffix) {
			index[suffix] = append(index[suffix], rel)
		}
	}
	deps := make(map[string][]string, len(rels))
	for _, rel := range rels {
		src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, fmt.Errorf("bindgen: reading %s: %w", rel, err)
		}
		for _, m := range includeDirective.FindAllSubmatch(src, -1) {
			want := path.Clean(strings.TrimSpace(string(m[1])))
			if !strings.HasSuffix(want, ".inc") {
				want += ".inc"
			}
			if dep, ok := resolve(index, rel, want); ok && dep != rel {
				deps[rel] = append(deps[rel], dep)
			}
		}
	}
	return topoSort(rels, deps), nil
}

// trimFirstSegment drops the leading path element, so an index keyed on every
// suffix of a path answers an include written relative to any include root.
func trimFirstSegment(p string) string {
	i := strings.IndexByte(p, '/')
	if i < 0 {
		return ""
	}
	return p[i+1:]
}

// resolve picks the file an include line means. When several files in the
// tree end with the same suffix, the one sharing the longest directory prefix
// with the includer wins: that is the copy the compiler's include path would
// have found first.
func resolve(index map[string][]string, from, want string) (string, bool) {
	candidates := index[want]
	if len(candidates) == 0 {
		return "", false
	}
	best, bestScore := "", -1
	for _, c := range candidates {
		if score := commonPrefixSegments(from, c); score > bestScore {
			best, bestScore = c, score
		}
	}
	return best, true
}

func commonPrefixSegments(a, b string) int {
	as, bs := strings.Split(a, "/"), strings.Split(b, "/")
	n := 0
	for n < len(as) && n < len(bs) && as[n] == bs[n] {
		n++
	}
	return n
}

// topoSort returns the nodes in dependency order, deterministic for a given
// input: ready nodes are taken in path order, and whatever a cycle leaves
// behind is appended in path order too.
func topoSort(nodes []string, deps map[string][]string) []string {
	remaining := make(map[string]map[string]bool, len(nodes))
	for _, n := range nodes {
		remaining[n] = map[string]bool{}
		for _, d := range deps[n] {
			remaining[n][d] = true
		}
	}
	out := make([]string, 0, len(nodes))
	done := make(map[string]bool, len(nodes))
	// Each pass emits at least one node or the rest is a cycle, so the loop
	// runs at most len(nodes) times.
	for range len(nodes) {
		progress := false
		for _, n := range nodes {
			if done[n] {
				continue
			}
			ready := true
			for d := range remaining[n] {
				if !done[d] {
					ready = false
					break
				}
			}
			if ready {
				out = append(out, n)
				done[n] = true
				progress = true
			}
		}
		if !progress {
			break
		}
	}
	for _, n := range nodes {
		if !done[n] {
			out = append(out, n)
		}
	}
	return out
}
