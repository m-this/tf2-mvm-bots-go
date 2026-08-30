/*
Package actionstack is the part of source/redbots3/util.sp that reads what a bot
is doing, as the chain of behaviours the engine has on it.

The chain is what every measurement of a stalled bot is written against: a stack
with no leaf under the monitor is a bot that was handed nothing to do.
*/
package actionstack

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// The stack being built, which the visitor appends to because SourceMod's
// iterator takes a function and no state.
//
//sp:name m_sActionStack
var actionStack engine.Text

// CollectActionName appends one behaviour's name.
//
//sp:name CollectActionName
//nolint:ineffassign,wastedassign,staticcheck // the writes are the point: StrCat appends into the buffer named on the left
func CollectActionName(action engine.Behaviour) {
	name := action.ActionName()

	if actionStack[0] != 0 {
		actionStack = engine.AppendText(" < ")
	}

	actionStack = engine.AppendTextFrom(name)
}

// ActionStackOf is the chain, newest first, as one line.
//
//sp:name ActionStackOf
//sp:length buffer maxlength
//nolint:revive,ineffassign,staticcheck,wastedassign // the write is the point: SourcePawn passes the buffer by reference and //sp:length carries its size
func ActionStackOf(client int32, buffer engine.Text, maxlength int32) {
	actionStack[0] = 0

	engine.IterateActions(client, CollectActionName)

	buffer = engine.CopyTextInto(actionStack)
}
