package bindings

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	gotoken "go/token"
	"regexp"
	"strconv"
	"strings"
)

// viewAs matches the cast SourcePawn writes as view_as<Tag>(expr).
var viewAs = regexp.MustCompile(`view_as\s*<[^>]*>`)

// evalConst folds a SourcePawn constant expression to an integer. The subset
// it accepts is the one the includes actually use for constants: integer
// literals, references to constants already resolved, parentheses, and the
// arithmetic and bitwise operators. Everything else is refused, because a
// constant guessed wrong is a silently misaimed call.
//
// Go's expression parser is reused rather than written again: this subset of
// SourcePawn expression syntax is also valid Go syntax.
func evalConst(expr string, lookup map[string]int64) (int64, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return 0, fmt.Errorf("empty expression")
	}
	// SourcePawn spells bitwise NOT '~'; Go spells it '^'. A view_as<T> cast
	// does not change an integer's value, so it drops out.
	expr = viewAs.ReplaceAllString(expr, "")
	node, err := goparser.ParseExpr(strings.ReplaceAll(expr, "~", "^"))
	if err != nil {
		return 0, fmt.Errorf("not a constant expression: %q", expr)
	}
	return foldExpr(node, lookup)
}

// maxConstDepth bounds the recursion; the deepest expression in the tree is
// four levels.
const maxConstDepth = 32

func foldExpr(node ast.Expr, lookup map[string]int64) (int64, error) {
	return foldAt(node, lookup, 0)
}

func foldAt(node ast.Expr, lookup map[string]int64, depth int) (int64, error) {
	if depth > maxConstDepth {
		return 0, fmt.Errorf("expression nested too deeply")
	}
	switch n := node.(type) {
	case *ast.ParenExpr:
		return foldAt(n.X, lookup, depth+1)
	case *ast.BasicLit:
		if n.Kind != gotoken.INT {
			return 0, fmt.Errorf("non-integer literal %s", n.Value)
		}
		return strconv.ParseInt(strings.ReplaceAll(n.Value, "_", ""), 0, 64)
	case *ast.Ident:
		v, ok := lookup[n.Name]
		if !ok {
			return 0, fmt.Errorf("unknown constant %s", n.Name)
		}
		return v, nil
	case *ast.UnaryExpr:
		return foldUnary(n, lookup, depth)
	case *ast.BinaryExpr:
		return foldBinary(n, lookup, depth)
	default:
		return 0, fmt.Errorf("unsupported expression")
	}
}

func foldUnary(n *ast.UnaryExpr, lookup map[string]int64, depth int) (int64, error) {
	v, err := foldAt(n.X, lookup, depth+1)
	if err != nil {
		return 0, err
	}
	switch n.Op {
	case gotoken.SUB:
		return -v, nil
	case gotoken.ADD:
		return v, nil
	case gotoken.XOR:
		return ^v, nil
	default:
		return 0, fmt.Errorf("unsupported unary operator %s", n.Op)
	}
}

func foldBinary(n *ast.BinaryExpr, lookup map[string]int64, depth int) (int64, error) {
	l, err := foldAt(n.X, lookup, depth+1)
	if err != nil {
		return 0, err
	}
	r, err := foldAt(n.Y, lookup, depth+1)
	if err != nil {
		return 0, err
	}
	switch n.Op {
	case gotoken.ADD:
		return l + r, nil
	case gotoken.SUB:
		return l - r, nil
	case gotoken.MUL:
		return l * r, nil
	case gotoken.QUO:
		if r == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return l / r, nil
	case gotoken.REM:
		if r == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return l % r, nil
	case gotoken.SHL:
		if r < 0 || r > 62 {
			return 0, fmt.Errorf("shift count %d out of range", r)
		}
		return l << uint(r), nil
	case gotoken.SHR:
		if r < 0 || r > 62 {
			return 0, fmt.Errorf("shift count %d out of range", r)
		}
		return l >> uint(r), nil
	case gotoken.OR:
		return l | r, nil
	case gotoken.AND:
		return l & r, nil
	case gotoken.XOR:
		return l ^ r, nil
	default:
		return 0, fmt.Errorf("unsupported operator %s", n.Op)
	}
}
