package engine

/*
ArrayList, the other handle.

The nav mesh collector is handed to a body; this one a body makes. Same rule
either way: it is a lifetime, so it is deferred, and internal/spbody puts the
delete at every way out and refuses one that nothing closes.
*/

// ListCalls are the answers.
type ListCalls struct {
	SortCustom func(l List, cmp Compare)
	PushFloat  func(l List, value float32) int32
	PushAt     func(l List, value int32) int32
	SetFloatAt func(l List, index int32, value float32, block int32)
	NewList    func() List
	NewBlocks  func(blockSize int32) List
	ListGetAt  func(l List, index int32, block int32) int32
	ListPush   func(l List, value int32)
	ListGet    func(l List, index int32) int32
	ListLength func(l List) int32
	ListClose  func(l List)
}

var lists ListCalls

// InstallLists puts a set of answers behind them.
func InstallLists(c ListCalls) func() {
	previous := lists
	lists = c
	return func() { lists = previous }
}

// List is SourceMod's ArrayList.
//
//sp:tag ArrayList
type List int32

// NewList makes one. The caller owns it.
//
//sp:new ArrayList
func NewList() List {
	if lists.NewList == nil {
		missing("new ArrayList")
	}
	return lists.NewList()
}

// Push adds one to the end.
//
//sp:method Push
func (l List) Push(value int32) {
	if lists.ListPush == nil {
		missing("ArrayList.Push")
	}
	lists.ListPush(l, value)
}

// Get is the one at that index.
//
//sp:method Get
func (l List) Get(index int32) int32 {
	if lists.ListGet == nil {
		missing("ArrayList.Get")
	}
	return lists.ListGet(l, index)
}

// Length is how many there are.
//
//sp:property Length
func (l List) Length() int32 {
	if lists.ListLength == nil {
		missing("ArrayList.Length")
	}
	return lists.ListLength(l)
}

// Close deletes it. Deferred, never called by hand.
//
//sp:delete Close
func (l List) Close() {
	if lists.ListClose == nil {
		missing("delete ArrayList")
	}
	lists.ListClose(l)
}

/*
Handle is SourcePawn's untyped handle, which is what a sort callback is handed.

An ArrayList is a methodmap over one, and the callback's own declaration takes
the raw handle rather than the methodmap, so the port takes it the same way.

//sp:tag Handle
*/
type Handle int32

// Compare is a sort callback: which of two entries comes first.
type Compare func(index1 int32, index2 int32, array Handle, hndl Handle) int32

// SortCustom sorts by a comparison this port declares. The callback is passed
// by name, which is the one place a function is a value in the subset.
//
//sp:method SortCustom
//nolint:revive // unused-parameter: the comparison is a name the emitter writes, not something the Go calls
func (l List) SortCustom(cmp Compare) {
	if lists.SortCustom == nil {
		missing("ArrayList.SortCustom")
	}
	lists.SortCustom(l, cmp)
}

// PushFloat adds a float and answers where it landed, which is what a two-cell
// entry needs before its second cell is written.
//
//sp:method Push
func (l List) PushFloat(value float32) int32 {
	if lists.PushFloat == nil {
		missing("ArrayList.Push")
	}
	return lists.PushFloat(l, value)
}

// PushAt is Push answering where the entry landed.
//
//sp:method Push
func (l List) PushAt(value int32) int32 {
	if lists.PushAt == nil {
		missing("ArrayList.Push")
	}
	return lists.PushAt(l, value)
}

// SetFloatAt writes a float into one cell of a wide entry.
//
//sp:method Set
func (l List) SetFloatAt(index int32, value float32, block int32) {
	if lists.SetFloatAt == nil {
		missing("ArrayList.Set")
	}
	lists.SetFloatAt(l, index, value, block)
}

// ListOf is the methodmap over a handle a callback was handed.
//
//sp:cast ArrayList
func ListOf(h Handle) List {
	return List(h)
}

// NoList is null, which is what a caller passes when it has no list to give.
//
//sp:global null
func NoList() List { return 0 }

// NewBlocks makes one whose entries are several cells wide, which is how the
// plugin keeps an entity and a distance side by side.
//
//sp:new ArrayList
func NewBlocks(blockSize int32) List {
	if lists.NewBlocks == nil {
		missing("new ArrayList")
	}
	return lists.NewBlocks(blockSize)
}

// GetAt is one cell of a wide entry.
//
//sp:method Get
func (l List) GetAt(index int32, block int32) int32 {
	if lists.ListGetAt == nil {
		missing("ArrayList.Get")
	}
	return lists.ListGetAt(l, index, block)
}
