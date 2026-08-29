package wave

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

/*
Defenders that were present and did nothing, which no A/B would ever report.

The stock sniper bug lived in the result files for a whole day. Every run wrote a
bot sample every half second carrying the action stack, the position and whether
the bot was firing, and nothing ever read them: the runner compared waves cleared
and robots killed, and six defenders doing the work of five still clears waves.

The fault has a shape, and it is the same shape for every class. A bot whose
action stack has no leaf under ScenarioMonitor has been handed nothing to do. It
is not stuck, it is not pathing, and no watchdog arms for it, so the only trace
it leaves is the sample that says what it was doing: nothing.

Checked on every run rather than asked for, because the whole point is finding
the cell of the matrix nobody thought to test.
*/

// Sample is one bot at one moment, as the statistics plugin wrote it.
type Sample struct {
	Event  string    `json:"event"`
	Wave   int       `json:"wave"`
	Time   float64   `json:"t"`
	Who    string    `json:"who"`
	Class  string    `json:"class"`
	At     []float64 `json:"at"`
	Firing int       `json:"firing"`
	Action string    `json:"action"`
}

// Idle is one defender that spent part of a run with nothing to do.
type Idle struct {
	Who       string
	Class     string
	Samples   int
	NoAction  int // samples whose stack ends at the monitor that hands out actions
	Still     int // samples at the position the bot held the sample before
	Fired     int
	Positions int
}

// Leafless is the share of the run this bot had no action of its own.
func (i Idle) Leafless() float64 {
	if i.Samples == 0 {
		return 0
	}
	return float64(i.NoAction) / float64(i.Samples)
}

/*
IdleShare is how much of a run each defender spent with no action of its own.

A stack holds nothing to do when every entry in it is a monitor. Monitors choose
an action and do none themselves, so a bot carrying only those has been given no
work at all:

	MainAction < TacticalMonitor < ScenarioMonitor                         nothing
	MainAction < TacticalMonitor < ScenarioMonitor < SniperLurk            working
	MainAction < TacticalMonitor < DefenderEngineerIdle < ScenarioMonitor  working

The first is Peppy's stalled sniper. The third is why the leaf cannot be the
test: a working engineer carries ScenarioMonitor last, and reading the leaf alone
called every engineer of every run broken.

Named rather than matched on the word "Monitor", so a real behaviour ending in it
is not mistaken for one of these.
*/
var monitors = []string{"ScenarioMonitor", "TacticalMonitor", "MainAction", "Taunt"}

// IdleShare is the fraction of the samples in which a bot had no behaviour at
// all, which is the shape mvm-ipf leaves in a file.
func IdleShare(path string) ([]Idle, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	byBot := map[string]*Idle{}
	last := map[string][]float64{}

	scan := bufio.NewScanner(file)
	scan.Buffer(make([]byte, 0, 1<<20), 1<<22)

	for scan.Scan() {
		line := scan.Bytes()
		if !strings.Contains(string(line), `"event":"bot"`) {
			continue
		}

		var s Sample
		if err := json.Unmarshal(line, &s); err != nil {
			continue
		}

		bot := byBot[s.Who]
		if bot == nil {
			bot = &Idle{Who: s.Who, Class: s.Class}
			byBot[s.Who] = bot
		}

		bot.Samples++
		bot.Fired += s.Firing
		if leafless(s.Action) {
			bot.NoAction++
		}
		if samePlace(last[s.Who], s.At) {
			bot.Still++
		} else {
			bot.Positions++
		}
		last[s.Who] = s.At
	}
	if err := scan.Err(); err != nil {
		return nil, err
	}

	out := make([]Idle, 0, len(byBot))
	for _, bot := range byBot {
		out = append(out, *bot)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Leafless() > out[j].Leafless() })

	return out, nil
}

func leafless(action string) bool {
	if action == "" {
		return true
	}

	for _, part := range strings.Split(action, " < ") {
		if !isMonitor(strings.TrimSpace(part)) {
			return false
		}
	}
	return true
}

func isMonitor(name string) bool {
	for _, monitor := range monitors {
		if name == monitor {
			return true
		}
	}
	return false
}

// samePlace is exact, because the plugin writes rounded integers and a bot that
// has not moved writes the same three of them.
func samePlace(before, now []float64) bool {
	if len(before) != len(now) {
		return false
	}
	for i := range now {
		if before[i] != now[i] {
			return false
		}
	}
	return true
}

/*
IdleShareWorthReporting is the share of a run a bot has to spend with no
behaviour before it is worth naming.

A share rather than a count of seconds: a bot idle for a moment between actions
is normal, and one idle for a third of a wave has been handed nothing. Peppy's
stalled sniper was at 1.00.
*/
const IdleShareWorthReporting = 0.30

// IdleReport names the defenders worth looking at, and says nothing when there
// are none.
func IdleReport(path string) string {
	bots, err := IdleShare(path)
	if err != nil {
		return ""
	}

	var lines []string
	for _, bot := range bots {
		if bot.Leafless() < IdleShareWorthReporting {
			continue
		}
		lines = append(lines, fmt.Sprintf("  %-20s %-8s no action for %3.0f%% of %d samples, %d positions, fired %d",
			bot.Who, bot.Class, bot.Leafless()*100, bot.Samples, bot.Positions, bot.Fired))
	}
	if len(lines) == 0 {
		return ""
	}

	return "defenders with nothing to do:\n" + strings.Join(lines, "\n")
}
