package plugin

import (
	"os"
	"strings"
)

// VersionsPath is the test-bed's pins. tf2-archipelago's bots build reads the
// mod's compiler out of this same file, which is why the names in it are a
// contract and not a local detail.
func VersionsPath() (string, error) { return Path("testbed", "versions.env") }

/*
Versions is the pins in testbed/versions.env, by key.

Read the way the shell reads it: one KEY=value per line, comments and blanks
skipped, nothing continued and nothing expanded. build.sh sources this file and
docker compose takes it with --env-file, so anything fancier would be a third
dialect for one file.
*/
func Versions() (map[string]string, error) {
	path, err := VersionsPath()
	if err != nil {
		return nil, err
	}
	body, err := os.ReadFile(path) //nolint:gosec // the pins of the tree Dir resolved
	if err != nil {
		return nil, err
	}

	pins := map[string]string{}
	for line := range strings.SplitSeq(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if key, value, found := strings.Cut(line, "="); found {
			pins[key] = value
		}
	}
	return pins, nil
}
