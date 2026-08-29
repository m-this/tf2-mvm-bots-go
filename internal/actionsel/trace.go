package actionsel

/* Turning the decision into a table by running it

The plugin cannot call Select. It has to ask its predicates one at a time, in
the order the shipped chain asks them, because three of the answers cost
something. So what the plugin gets is a table: at each node, one question, and
where to go for each answer.

The table is extracted by running the real Select against a Facts that refuses
to answer a question it has not been told about. The refusal names the question,
the explorer answers it both ways, and recurs. The order in the table is
therefore the order the Go actually evaluates in, short circuits included,
rather than an order inferred from the answers.

This is why the decision is written against an interface. An earlier version
translated the Go syntax tree symbolically, which worked and was a second
implementation of the same decision that could drift from it. Running the real
thing cannot drift from itself. */

// Node is one question and where each answer leads. A leaf has Leaf set and
// asks nothing.
type Node struct {
	Ask         Predicate
	True, False *Node

	Leaf   bool
	Action Action
}

// unanswered is how partial refuses a question it has no answer for. It never
// escapes Explore.
type unanswered struct{ p Predicate }

// partial answers only what it has been told, and records the order asked.
type partial struct {
	known map[Predicate]bool
	asked []Predicate
}

func (s *partial) Ask(p Predicate) bool {
	v, ok := s.known[p]
	if !ok {
		panic(unanswered{p})
	}
	for _, seen := range s.asked {
		if seen == p {
			return v
		}
	}
	s.asked = append(s.asked, p)
	return v
}

// Explore builds the decision table for one round state and class by running
// Select and answering each question it asks both ways.
func Explore(state RoundState, class Class) *Node {
	return explore(state, class, map[Predicate]bool{})
}

func explore(state RoundState, class Class, known map[Predicate]bool) (n *Node) {
	s := &partial{known: known}

	action, asked := run(state, class, s)
	if asked == nil {
		return &Node{Leaf: true, Action: action}
	}

	both := func(v bool) *Node {
		next := make(map[Predicate]bool, len(known)+1)
		for p, b := range known {
			next[p] = b
		}
		next[*asked] = v
		return explore(state, class, next)
	}
	return &Node{Ask: *asked, True: both(true), False: both(false)}
}

// run calls Select and reports either its answer or the first question it
// could not have answered.
func run(state RoundState, class Class, s *partial) (action Action, asked *Predicate) {
	defer func() {
		if r := recover(); r != nil {
			u, ok := r.(unanswered)
			if !ok {
				panic(r)
			}
			asked = &u.p
		}
	}()
	return Select(state, class, s), nil
}

/*
	AskOrder is the questions Select asks, in order, for one full answer set.

It is what proves the table asks nothing the shipped chain would not have asked:
the walk's questions must be a subsequence of these, and these are produced by
running the decision rather than by reading it.
*/
func AskOrder(state RoundState, class Class, f Flags) []Predicate {
	s := &recorder{answers: f}
	Select(state, class, s)
	return s.asked
}

type recorder struct {
	answers Flags
	asked   []Predicate
}

func (r *recorder) Ask(p Predicate) bool {
	for _, seen := range r.asked {
		if seen == p {
			return r.answers.Ask(p)
		}
	}
	r.asked = append(r.asked, p)
	return r.answers.Ask(p)
}
