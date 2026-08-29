/*
Package runmap draws a test-bed run as a picture of the map it was played on.

The test-bed records where every bot stood twice a second and the only way to
read that has been as numbers. A fault that is about ground - a bot wedged in a
corner, a nest built in the bot lane, a team that never leaves spawn - is a
shape, and a column of coordinates is the worst way to look at a shape.

What it draws is the nav mesh as a floor plan with the run laid over it. The
mesh is the right background because it is what the bots actually navigate: a
bot standing somewhere with no nav area under it is a different bug from a bot
standing somewhere inconvenient, and only the mesh tells them apart.

It says where things were. It does not say whether that is unusual, which is
what the trace assertions are for. See mvm-17o.
*/
package runmap

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// Sample is one bot at one instant, as the statistics plugin writes it. Only
// the fields this package draws with are listed; the rest of the line is
// ignored, so an older run missing a field still reads.
type Sample struct {
	Wave    int       `json:"wave"`
	Clock   float64   `json:"clock"`
	Who     string    `json:"who"`
	Class   string    `json:"class"`
	At      []float64 `json:"at"`
	Action  string    `json:"action"`
	Pathing int       `json:"pathing"`
}

// Building is one building at one instant. Sampled rather than placed: the
// position is where it stood when it was read, which is what a picture wants.
type Building struct {
	Map   string    `json:"map"`
	Wave  int       `json:"wave"`
	Clock float64   `json:"clock"`
	Owner string    `json:"owner"`
	Type  string    `json:"type"`
	Mode  int       `json:"mode"`
	Level int       `json:"level"`
	At    []float64 `json:"at"`
}

// Track is one bot's samples through one wave, in the order they were written.
type Track struct {
	Who     string
	Class   string
	Samples []Sample
}

// Wave is everything drawn on one picture.
type Wave struct {
	Map       string
	Number    int
	Tracks    []Track
	Buildings []Building
}

// Run is a results file, split by wave.
type Run struct {
	Map   string
	Waves []Wave
}

/*
Read parses a results file into waves.

Line oriented and forgiving on purpose. A run that crashed halfway leaves a
truncated last line, and refusing the whole file for it would throw away the
part that was written, which is usually the part worth looking at.
*/
func Read(r io.Reader) (Run, error) {
	var (
		run     Run
		samples = map[int][]Sample{}
		builds  = map[int][]Building{}
		waves   []int
	)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "{") {
			continue
		}

		switch {
		case strings.Contains(line, `"event":"bot"`):
			var s Sample
			if json.Unmarshal([]byte(line), &s) != nil || s.Wave <= 0 || len(s.At) < 2 {
				continue
			}
			if _, seen := samples[s.Wave]; !seen {
				waves = append(waves, s.Wave)
			}
			samples[s.Wave] = append(samples[s.Wave], s)

		case strings.Contains(line, `"event":"building"`):
			var b Building
			if json.Unmarshal([]byte(line), &b) != nil || b.Wave <= 0 || len(b.At) < 2 {
				continue
			}
			if run.Map == "" {
				run.Map = b.Map
			}
			builds[b.Wave] = append(builds[b.Wave], b)

		case strings.Contains(line, `"event":"wave_begin"`):
			var head struct {
				Map string `json:"map"`
			}
			if json.Unmarshal([]byte(line), &head) == nil && head.Map != "" {
				run.Map = head.Map
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return run, fmt.Errorf("runmap: reading samples: %w", err)
	}

	sort.Ints(waves)

	for _, number := range waves {
		run.Waves = append(run.Waves, Wave{
			Map:       run.Map,
			Number:    number,
			Tracks:    tracksOf(samples[number]),
			Buildings: builds[number],
		})
	}

	return run, nil
}

// ReadFile is Read over a path.
func ReadFile(path string) (Run, error) {
	file, err := os.Open(path)
	if err != nil {
		return Run{}, fmt.Errorf("runmap: %w", err)
	}
	// Read only, so a failed close says nothing a caller could act on.
	defer func() { _ = file.Close() }()

	return Read(file)
}

/*
tracksOf groups a wave's samples by bot.

Sorted by name rather than left in file order, so two runs of the same mission
draw their bots in the same order and the pictures can be put side by side.
*/
func tracksOf(samples []Sample) []Track {
	byWho := map[string]*Track{}
	var names []string

	for _, s := range samples {
		track := byWho[s.Who]
		if track == nil {
			track = &Track{Who: s.Who, Class: s.Class}
			byWho[s.Who] = track
			names = append(names, s.Who)
		}
		// The class is read from the last sample that named one: a bot that
		// changed class mid-wave is drawn as what it ended up being, and a
		// sample with no class does not blank a track that had one.
		if s.Class != "" {
			track.Class = s.Class
		}
		track.Samples = append(track.Samples, s)
	}

	sort.Strings(names)

	tracks := make([]Track, 0, len(names))
	for _, name := range names {
		tracks = append(tracks, *byWho[name])
	}

	return tracks
}
