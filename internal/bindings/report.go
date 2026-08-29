package bindings

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Coverage counts what a parse run understood and what it turned down.
type Coverage struct {
	Files       int
	Natives     int
	Stocks      int
	Methodmaps  int
	Methods     int
	Properties  int
	Enums       int
	EnumEntries int
	EnumStructs int
	Defines     int
	Typedefs    int
	Refusals    []Refusal
}

// Add folds one parsed file into the running totals.
func (c *Coverage) Add(f *File) {
	c.Files++
	c.Natives += len(f.Natives)
	c.Stocks += len(f.Stocks)
	c.Methodmaps += len(f.Methodmaps)
	for _, mm := range f.Methodmaps {
		c.Methods += len(mm.Methods)
		c.Properties += len(mm.Properties)
	}
	c.Enums += len(f.Enums)
	for _, e := range f.Enums {
		c.EnumEntries += len(e.Entries)
	}
	c.EnumStructs += len(f.EnumStructs)
	c.Defines += len(f.Defines)
	c.Typedefs += len(f.Typedefs)
	c.Refusals = append(c.Refusals, f.Refusals...)
}

// RefusalsByReason groups the refusals so a caller sees kinds, not a wall.
func (c *Coverage) RefusalsByReason() map[string]int {
	byReason := make(map[string]int, len(c.Refusals))
	for _, r := range c.Refusals {
		byReason[r.Kind+": "+r.Reason]++
	}
	return byReason
}

// WriteTo prints a human-readable coverage summary.
func (c *Coverage) WriteTo(w io.Writer) (int64, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "files        %d\n", c.Files)
	fmt.Fprintf(&b, "natives      %d\n", c.Natives)
	fmt.Fprintf(&b, "stocks       %d\n", c.Stocks)
	fmt.Fprintf(&b, "methodmaps   %d\n", c.Methodmaps)
	fmt.Fprintf(&b, "methods      %d\n", c.Methods)
	fmt.Fprintf(&b, "properties   %d\n", c.Properties)
	fmt.Fprintf(&b, "enums        %d (%d entries)\n", c.Enums, c.EnumEntries)
	fmt.Fprintf(&b, "enum structs %d\n", c.EnumStructs)
	fmt.Fprintf(&b, "typedefs     %d\n", c.Typedefs)
	fmt.Fprintf(&b, "defines      %d\n", c.Defines)
	fmt.Fprintf(&b, "refusals     %d\n", len(c.Refusals))
	byReason := c.RefusalsByReason()
	keys := make([]string, 0, len(byReason))
	for k := range byReason {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "  %5d  %s\n", byReason[k], k)
	}
	n, err := io.WriteString(w, b.String())
	return int64(n), err
}
