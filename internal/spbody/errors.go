package spbody

import "errors"

var (
	errUnnamedParam  = errors.New("a parameter with no name; SourcePawn declares every parameter with one")
	errUnnamedResult = errors.New("a result with no name that becomes a parameter; name it, because the name is what the emitted parameter is called")
)
