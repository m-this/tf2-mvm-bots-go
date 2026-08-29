// Package spgen turns the decision in internal/actionsel into the SourcePawn
// the plugin includes: the decision as a table, and the edge that walks it.
//
// Nothing here reads the Go source. The table is extracted by running the
// decision, in actionsel.Explore, against a Facts that refuses a question it
// has not been told about; the refusal names the question, the explorer
// answers it both ways and recurs. A table produced by running the decision
// cannot disagree with the decision, which is what an earlier symbolic
// translation of the Go syntax tree could.
//
// The decision therefore has no syntax to translate and is written in ordinary
// Go. What has to become SourcePawn arithmetic rather than a table, nest
// scoring and threat priority, will need a body generator, and it will be
// written when it is needed.
//
// Names. Every emitted identifier carries Config.Prefix, because Action,
// RoundState and Address are SourceMod's names already.
package spgen
