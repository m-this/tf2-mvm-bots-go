package body_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/upstream"
)

/*
	TestNoShippedFunctionSizesItsOwnParameter keeps the shape comparison honest

shape drops what sits between the brackets of a parameter, because SourcePawn
passes an array parameter by reference and tells the callee nothing about its
length: char[] and char[PLATFORM_MAX_PATH] are the same contract to everybody
except sizeof. That is only true while no shipped function takes sizeof of its
own parameter, so this says so out loud rather than leaving it as a remark in a
comment.
*/
func TestNoShippedFunctionSizesItsOwnParameter(t *testing.T) {
	t.Parallel()

	header := regexp.MustCompile(`^(?:stock |static |public )?[A-Za-z_]\w*(?:\[\])? ([A-Za-z_]\w*)\(([^)]*)\)\s*$`)

	files, err := upstream.SnapshotFiles()
	if err != nil {
		t.Fatalf("listing the snapshot: %v", err)
	}

	checked := 0

	for _, file := range files {
		if !strings.HasSuffix(file.Path, ".sp") {
			continue
		}

		lines := strings.Split(file.Body, "\n")

		for i, line := range lines {
			m := header.FindStringSubmatch(line)
			if m == nil || i+1 >= len(lines) || !strings.HasPrefix(lines[i+1], "{") {
				continue
			}

			body, ok := bodyAfter(lines, i+1)
			if !ok {
				continue
			}
			checked++

			for _, p := range strings.Split(m[2], ",") {
				name := parameterName(p)
				if name == "" {
					continue
				}
				at := regexp.MustCompile(`sizeof\(\s*` + regexp.QuoteMeta(name) + `\s*[)\[]`)
				if at.MatchString(body) {
					t.Errorf("%s in %s takes sizeof of its own parameter %s, so shape cannot drop the dimension", m[1], file.Path, name)
				}
			}
		}
	}

	if checked == 0 {
		t.Fatal("no shipped function was read, so this proves nothing")
	}
}

// bodyAfter is the braced block starting at the line given.
func bodyAfter(lines []string, open int) (string, bool) {
	depth := 0
	for j := open; j < len(lines); j++ {
		if lines[j] == "{" {
			depth++
		}
		if lines[j] == "}" {
			depth--
			if depth == 0 {
				return strings.Join(lines[open+1:j], "\n"), true
			}
		}
	}
	return "", false
}

// parameterName is the name a declared parameter binds, without its type or
// its dimensions.
func parameterName(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if before, _, has := strings.Cut(p, "="); has {
		p = strings.TrimSpace(before)
	}
	if at := strings.Index(p, "["); at >= 0 {
		p = strings.TrimSpace(p[:at])
	}
	fields := strings.Fields(p)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimPrefix(fields[len(fields)-1], "&")
}
