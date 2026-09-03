package engine

/*
The queries the engine asks a behaviour.

Different in shape from the five it enters: the engine wants an answer, so the
answer comes back through a by-reference parameter and the return says only
whether the behaviour had an opinion. In Go that is a second result, which the
body generator already turns into a by-reference parameter.
*/

// Answer is SourceMod's QueryResultType, what a behaviour says when asked.
//
//sp:tag QueryResultType
type Answer int32

// AnswerNo is ANSWER_NO: do not.
//
//sp:global ANSWER_NO
func AnswerNo() Answer { return 0 }

// AnswerYes is ANSWER_YES.
//
//sp:global ANSWER_YES
func AnswerYes() Answer { return 1 }

// AnswerUndefined is ANSWER_UNDEFINED: no opinion, ask somebody else.
//
//sp:global ANSWER_UNDEFINED
func AnswerUndefined() Answer { return 2 }

// Changed is Plugin_Changed: the behaviour answered.
//
//sp:global Plugin_Changed
func Changed() Outcome { return 1 }

// Actor is the client the action is running for. A query is not given one, so
// it reads it off the action the engine handed in.
//
//sp:global action.Actor
func Actor() int32 { return actions.Actor() }
