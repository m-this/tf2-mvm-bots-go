package spbody

import "errors"

var (
	errReturnsArray  = errors.New("a function returning an array; SourcePawn returns a cell, so take the array as a parameter and fill it")
	errUnnamedParam  = errors.New("a parameter with no name; SourcePawn declares every parameter with one")
	errUnnamedResult = errors.New("a result after the first with no name; it becomes a by-reference parameter and needs one")
)
