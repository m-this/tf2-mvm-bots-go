package engine

import (
	"fmt"
	"reflect"
)

/*
An answer that was not installed

Nothing in this package means anything in a Go process. A body compiled against
it runs under the differential test, which installs a set of answers and
compares the call trace against the SourcePawn's; a call the test did not expect
the body to make has no answer behind it, and reaching one is a failed
expectation rather than a zero value quietly standing in for one.

Saying so used to be the call site's job. Every one of 1108 wrappers opened with
the same three lines:

	if ammo.IsAmmoFull == nil {
		missing("IsAmmoFull")
	}

which is four fifths of internal/engine and none of its meaning. Worse, it is
the signature written a third time -- once in the struct field, once in the
wrapper's parameters, once in the forwarded call -- and nothing holds the three
against each other, so a wrapper that forwards the wrong argument to the right
type still builds.

Fill does it once instead, at install time. Every field the caller left nil gets
a function of the right type that panics with its name, so a call with no answer
behind it fails exactly as loudly as before and the wrapper is one line.
*/

// Fill replaces every nil function field of the answers with one that panics
// naming itself. The argument is a pointer to the answers struct, and the
// struct is the caller's own copy, so this changes nothing anybody else holds.
func Fill(answers any) {
	v := reflect.ValueOf(answers)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("engine: Fill wants a pointer to an answers struct and was given %T", answers))
	}
	v = v.Elem()
	name := v.Type().Name()
	for i := range v.NumField() {
		f := v.Field(i)
		if f.Kind() != reflect.Func || !f.IsNil() || !f.CanSet() {
			continue
		}
		called := name + "." + v.Type().Field(i).Name
		f.Set(reflect.MakeFunc(f.Type(), func([]reflect.Value) []reflect.Value {
			missing(called)
			return nil // missing panics; this is what the compiler needs to hear.
		}))
	}
}

func missing(name string) {
	panic(fmt.Sprintf("engine: %s was called and no answer is installed; this call has meaning on a game server and none here", name))
}
