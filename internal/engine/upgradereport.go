package engine

/*
Printing what a bot is actually carrying.

The answer goes back wherever the command came from, which on a test server is
usually rcon and not a chat window.
*/

// UpgradeReportCalls are the answers.
type UpgradeReportCalls struct {
	FormatEx      func(format string, args []any) Text
	AttributeName func(index int32) (bool, Text)
	StrcopyText   func(out Text, maxlen int32, from string)
}

var upgradeReports UpgradeReportCalls

// InstallUpgradeReports puts a set of answers behind them.
func InstallUpgradeReports(c UpgradeReportCalls) func() {
	previous := upgradeReports
	Fill(&c)
	upgradeReports = c
	return func() { upgradeReports = previous }
}

// FormatEx writes a formatted line into a fresh buffer.
//
//sp:native FormatEx fills
func FormatEx(format string, args ...any) (out Text) { return upgradeReports.FormatEx(format, args) }

// AttributeName is what the schema calls that attribute, and whether it knows
// it at all.
//
//sp:native TF2Econ_GetAttributeName sized
func AttributeName(index int32) (ok bool, name Text) { return upgradeReports.AttributeName(index) }

// StrcopyText overwrites a buffer with a literal.
//
//sp:native strcopy
func StrcopyText(out Text, maxlen int32, from string) { upgradeReports.StrcopyText(out, maxlen, from) }

// AttributeNameSize is the 128 the printer declares for a name.
//
//sp:global 128
func AttributeNameSize() int32 { return 128 }
