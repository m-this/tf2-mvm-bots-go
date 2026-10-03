package plugin_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/plugin"
)

/*
testbed/versions.env keeps the keys its two readers need, in the form they read
them.

One of those readers is in another repository. tf2-archipelago's
deploy/bots/build.sh seds SOURCEMOD_BRANCH and SOURCEMOD_VERSION out of this
file to get the compiler it builds the mod with, because it used to keep its own
copy of them and the copy drifted ninety builds away from this one. A sed cannot
notice a renamed key, so this does: the gate here fails rather than an image
build there, six minutes in.

The other is cmd/testbed, which refuses a run on a server whose SourceMod is
older than SOURCEMOD_RUNTIME_MIN.

Matched with a regex rather than through plugin.Versions, so that the parser and
the file are not checked against each other: the contract is the literal line.
*/
func TestTheTestbedPinsKeepTheirContract(t *testing.T) {
	t.Parallel()

	plugin.SkipOrFail(t)
	path, err := plugin.VersionsPath()
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	drop := `[0-9]+\.[0-9]+\.[0-9]+-git[0-9]+`
	for key, want := range map[string]string{
		// Read by tf2-archipelago's deploy/bots/build.sh.
		"SOURCEMOD_BRANCH":  `[0-9]+\.[0-9]+`,
		"SOURCEMOD_VERSION": drop,
		// Read by cmd/testbed before it plays anything.
		"SOURCEMOD_RUNTIME_MIN":        drop,
		"SOURCEMOD_TF2_SERVER_VERSION": `[0-9]+`,
	} {
		line := regexp.MustCompile(`(?m)^` + key + `=(.*)$`)
		found := line.FindAllStringSubmatch(string(body), -1)
		if len(found) != 1 {
			t.Errorf("%s is named %d times in %s as a KEY=value line, want once", key, len(found), path)
			continue
		}
		if !regexp.MustCompile(`^` + want + `$`).MatchString(found[0][1]) {
			t.Errorf("%s = %q, want /%s/", key, found[0][1], want)
		}
	}
}
