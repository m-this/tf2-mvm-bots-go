/*
Command deadsweep refuses a function nothing reaches.

	go run ./cmd/deadsweep

Nothing looked for unreachable code here, and the obvious tool does not work out
of the box: deadcode over this repository reports about 2400 functions, and
almost all of them are correct by design. internal/engine, internal/action and
internal/body are Go that becomes SourcePawn, and none of it is called from a Go
main; a tool that cries two thousand times is a tool nobody runs.

So this subtracts those and refuses whatever is left, rather than printing a
list for somebody to skim. There is no allow list on purpose: an entry is
decided, not silenced. See mvm-h5f.
*/
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
)

/*
deadcodePin is the version of the tool this gate runs.

Pinned rather than @latest so the gate does not change under a green build, and
run through `go run` rather than added to go.mod so a check tool is not a
dependency of the mod tf2-archipelago pulls in. v0.37.0 and earlier panic on the
generics in this tree, which is why the pin is not older.
*/
const deadcodePin = "golang.org/x/tools/cmd/deadcode@v0.49.0"

/*
generatedPrefixes are the packages that become SourcePawn.

A prefix and not a list of package names, because that is what makes a new
package under any of them covered without anybody remembering to add it. The
reason is the same for all three: the functions are reached from generated
SourcePawn, and the Go program that would call them does not exist.
*/
var generatedPrefixes = []string{
	"internal/engine/",
	"internal/action/",
	"internal/body/",
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "deadsweep: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if err := everyBodyIsCovered(); err != nil {
		return err
	}

	//nolint:gosec // the arguments are constants in this file
	out, err := exec.Command("go", "run", deadcodePin, "-test", "./...").Output()
	if err != nil {
		return fmt.Errorf("running %s: %w", deadcodePin, err)
	}

	var found []string
	lines := bufio.NewScanner(strings.NewReader(string(out)))
	for lines.Scan() {
		if line := lines.Text(); line != "" && !generated(line) {
			found = append(found, line)
		}
	}
	if err := lines.Err(); err != nil {
		return err
	}
	if len(found) == 0 {
		return nil
	}

	sort.Strings(found)
	return fmt.Errorf("%d function(s) nothing reaches. Delete each, or give it a caller, or say in a comment why it is kept:\n  %s",
		len(found), strings.Join(found, "\n  "))
}

func generated(line string) bool {
	for _, prefix := range generatedPrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

/*
everyBodyIsCovered refuses a body package that the prefixes above do not cover.

Without this the gate fails open: a body package put somewhere else would have
every one of its functions reported, and the person who added it would reach for
the allow list this deliberately does not have.
*/
func everyBodyIsCovered() error {
	for _, b := range body.All {
		if !generated(b.Dir + "/") {
			return fmt.Errorf("%s becomes SourcePawn and no prefix in generatedPrefixes covers it", b.Dir)
		}
	}
	return nil
}
