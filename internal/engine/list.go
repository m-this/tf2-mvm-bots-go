package engine

/*
ArrayList, the other handle.

The nav mesh collector is handed to a body; this one a body makes. Same rule
either way: it is a lifetime, so it is deferred, and internal/spbody puts the
delete at every way out and refuses one that nothing closes.
*/

// ListCalls are the answers.
type ListCalls struct {
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
