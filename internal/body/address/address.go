/*
Package address is where a pointer crosses between a cell and an Address.

SourceMod 1.13 declares typedef Address = int64. The pointers the mod keeps in
a methodmap (the nextbot, a nav area, a row of the upgrade table, a countdown
timer) are still one 32-bit cell, because a methodmap cannot derive from an
int64. So a pointer is a cell inside the mod and an Address wherever a native
takes one, and every crossing is one of the two functions below. internal/spbody
refuses the others: a cell of any tag but int converted to an Address, and an
Address converted to anything, fail generation and name this package.

Widening zero-extends. A cell holding 0x80001000 is a pointer above two
gigabytes, not a negative number, and sign extension would make it one.

Narrowing is checked. A 32-bit pointer reaches the plugin in two forms:
zero-extended, from GameConfGetAddress and the natives that write a uintptr_t,
and sign-extended, from a DHooks virtual hook, which pushes
reinterpret_cast<int64_t> of the pointer. Both narrow to the same cell. Anything
else does not fit one and throws, rather than going on with the low half.

An int offset added to an Address is not a crossing: it is a signed number and
widens as one, which is what view_as<Address> does and what Go does too.
*/
package address

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// cellMask is the low 32 bits of an Address. SourcePawn reads the literal as
// an int64: 0xFFFFFFFF would be the int -1, which masks nothing.
//
//sp:name ADDRESS_CELL_MASK
const cellMask = 1<<32 - 1

// Widen is the Address of a pointer held in a cell, zero-extended.
//
//sp:name AddressOfCell
func Widen(cell int32) engine.Address {
	return engine.Address(cell) & cellMask
}

// Narrow is the cell a pointer fits in. It throws when the high half is
// anything but the extension of the low one: zero, or all ones above a low
// half whose top bit is set.
//
//sp:name CellOfAddress
//sp:narrows
func Narrow(address engine.Address) int32 {
	high := address >> 32

	low := int32(address) //nolint:gosec // G115: the low half is the cell; the check below is what makes keeping only it safe

	if high != 0 && (high != -1 || low >= 0) {
		engine.ThrowError("Address %08x%08x does not fit a cell", int32(high), low) //nolint:gosec // G115: high is printed, and it is the reason for the throw
	}

	return low
}
