package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	waveline "github.com/m-this/tf2-mvm-bots-go/gen/go/wave"
)

/*
Asking one narrow question about a results file, without writing a program.

Every question narrower than the prose report was answered with a Python
one-liner over the .jsonl: ninety-six scripts reading the statistics file and
fifty-five reading a results file, counting the samples where a bot's action
stack is empty, taking a median distance, pulling one field over time. Each was
written once, read once and thrown away, and each was a fresh chance to parse
the file wrongly.

The names are not guessed at here. The wave record is generated from
internal/tables, the same table the plugin's FormatEx is generated from, so a
field this reads is a field the plugin writes by construction. A name that is
not in one of the three records is a refusal that says what the near ones are,
which is the failure mode the hand-written 38-of-112 subset had. See mvm-1ro.
*/

// The three shapes of line in a results file, by the event that names them.
// Every field belongs to exactly one, so naming a field picks the lines.
var families = []struct {
	event  string
	record any
}{
	{waveline.Event, waveline.Record{}},
	{botEvent, botSample{}},
	{buildingEvent, buildingSample{}},
}

// fieldFamily is the event whose lines carry the named field, and whether any
// does. The lookup is over the json tags, which is what the file actually has.
func fieldFamily(name string) (string, bool) {
	for _, f := range families {
		if hasField(f.record, name) {
			return f.event, true
		}
	}
	return "", false
}

func hasField(record any, name string) bool {
	t := reflect.TypeOf(record)
	for i := range t.NumField() {
		if tagName(t.Field(i).Tag.Get("json")) == name {
			return true
		}
	}
	return false
}

func tagName(tag string) string {
	name, _, _ := strings.Cut(tag, ",")
	return name
}

// knownFields is every name a selector may use, for the refusal to list.
func knownFields() []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range families {
		t := reflect.TypeOf(f.record)
		for i := range t.NumField() {
			name := tagName(t.Field(i).Tag.Get("json"))
			if name == "" || name == "-" || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// Selection is what a field selector found: the series, its shape if it is
// numeric, and the counts by value if it is not.
type Selection struct {
	File   string             `json:"file"`
	Field  string             `json:"field"`
	Event  string             `json:"event"`
	Where  []string           `json:"where,omitempty"`
	Count  int                `json:"count"`
	Series []float64          `json:"series,omitempty"`
	Min    float64            `json:"min,omitempty"`
	Q1     float64            `json:"q1,omitempty"`
	Median float64            `json:"median,omitempty"`
	Q3     float64            `json:"q3,omitempty"`
	Max    float64            `json:"max,omitempty"`
	Mean   float64            `json:"mean,omitempty"`
	Values map[string]int     `json:"values,omitempty"`
	Sums   map[string]float64 `json:"-"`
}

/*
selectField reads one field out of one file, over the lines a filter leaves.

Every line is read as a map rather than into the record, because a filter may
name a field the series does not and both have to be readable from the same
pass. The file is a few megabytes, so one pass and no index is the whole
algorithm.
*/
func selectField(path, field string, where []string) (Selection, error) {
	event, known := fieldFamily(field)
	if !known {
		return Selection{}, fmt.Errorf("no field %q in any line the plugin writes; the names are: %s",
			field, strings.Join(knownFields(), ", "))
	}
	clauses, err := parseWhere(where)
	if err != nil {
		return Selection{}, err
	}

	file, err := os.Open(path) //nolint:gosec // a results file the caller named
	if err != nil {
		return Selection{}, err
	}
	defer func() { _ = file.Close() }()

	got := Selection{File: path, Field: field, Event: event, Where: where, Values: map[string]int{}}

	scan := bufio.NewScanner(file)
	scan.Buffer(make([]byte, 0, 1<<20), 1<<22)
	for scan.Scan() {
		var line map[string]json.RawMessage
		if err := json.Unmarshal(scan.Bytes(), &line); err != nil {
			continue
		}
		if text(line["event"]) != event || !matches(line, clauses) {
			continue
		}
		raw, present := line[field]
		if !present {
			continue
		}
		got.Count++
		if n, ok := number(raw); ok {
			got.Series = append(got.Series, n)
			continue
		}
		got.Values[text(raw)]++
	}
	if err := scan.Err(); err != nil {
		return Selection{}, err
	}
	got.shape()
	return got, nil
}

// shape fills the quartiles. A series of one has them all equal, which is true
// and is better than leaving them at zero.
func (s *Selection) shape() {
	if len(s.Series) == 0 {
		return
	}
	sorted := append([]float64(nil), s.Series...)
	sort.Float64s(sorted)

	total := 0.0
	for _, n := range sorted {
		total += n
	}
	s.Min, s.Max = sorted[0], sorted[len(sorted)-1]
	s.Q1, s.Median, s.Q3 = at(sorted, 0.25), at(sorted, 0.5), at(sorted, 0.75)
	s.Mean = total / float64(len(sorted))
}

// at is the value at a quantile of a sorted series, by nearest rank. Nearest
// rank and not an interpolation: the series is often integers and a median of
// 3.5 healing samples is a number nobody wrote.
func at(sorted []float64, q float64) float64 {
	i := int(math.Ceil(q*float64(len(sorted)))) - 1
	return sorted[max(0, min(i, len(sorted)-1))]
}

type clause struct{ field, value string }

func parseWhere(where []string) ([]clause, error) {
	out := make([]clause, 0, len(where))
	for _, one := range where {
		field, value, found := strings.Cut(one, "=")
		if !found || field == "" {
			return nil, fmt.Errorf("-where is field=value, not %q", one)
		}
		if _, known := fieldFamily(field); !known {
			return nil, fmt.Errorf("no field %q to filter on; the names are: %s", field, strings.Join(knownFields(), ", "))
		}
		out = append(out, clause{field: field, value: value})
	}
	return out, nil
}

func matches(line map[string]json.RawMessage, clauses []clause) bool {
	for _, c := range clauses {
		if text(line[c.field]) != c.value {
			return false
		}
	}
	return true
}

// text is a JSON value as the caller would have typed it: a string without its
// quotes, a number as written, anything else as its own JSON.
func text(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

func number(raw json.RawMessage) (float64, bool) {
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, true
	}
	return 0, false
}

func printSelection(s Selection, asJSON bool) error {
	if asJSON {
		return writeJSON(s)
	}
	fmt.Printf("%s: %s over %s lines", s.File, s.Field, s.Event)
	if len(s.Where) > 0 {
		fmt.Printf(", where %s", strings.Join(s.Where, " and "))
	}
	fmt.Printf("\n  %d values\n", s.Count)
	if s.Count == 0 {
		return nil
	}
	if len(s.Series) > 0 {
		fmt.Printf("  min %s  q1 %s  median %s  q3 %s  max %s  mean %s\n",
			num(s.Min), num(s.Q1), num(s.Median), num(s.Q3), num(s.Max), num(s.Mean))
		return nil
	}
	for _, value := range rankedStrings(s.Values) {
		fmt.Printf("  %-24s %d\n", value, s.Values[value])
	}
	return nil
}

func num(f float64) string { return strconv.FormatFloat(f, 'g', 6, 64) }

func rankedStrings(of map[string]int) []string {
	out := make([]string, 0, len(of))
	for value := range of {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		if of[out[i]] != of[out[j]] {
			return of[out[i]] > of[out[j]]
		}
		return out[i] < out[j]
	})
	return out
}

func writeJSON(v any) error {
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(body))
	return err
}
