package actionmeta

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// Condition is a read-only boolean expression over explicit model state paths.
// The zero value is unconditional.
type Condition struct{ expr ast.Expr }

type StateProvider interface{ QueryState(string) (any, bool) }

// ParseCondition accepts state paths, true/false, !, &&, || and parentheses.
func ParseCondition(source string) (Condition, error) {
	if strings.TrimSpace(source) == "" {
		return Condition{}, nil
	}
	expr, err := parser.ParseExpr(source)
	if err != nil {
		return Condition{}, fmt.Errorf("invalid when condition: %w", err)
	}
	if !validCondition(expr) {
		return Condition{}, fmt.Errorf("invalid when condition %q: use boolean state paths, true/false, !, &&, || and parentheses", source)
	}
	return Condition{expr: expr}, nil
}

func statePath(expr ast.Expr) (string, bool) {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name, true
	case *ast.SelectorExpr:
		base, ok := statePath(e.X)
		return base + "." + e.Sel.Name, ok
	}
	return "", false
}

func validCondition(expr ast.Expr) bool {
	if _, ok := statePath(expr); ok {
		return true
	}
	switch e := expr.(type) {
	case *ast.ParenExpr:
		return validCondition(e.X)
	case *ast.UnaryExpr:
		return e.Op == token.NOT && validCondition(e.X)
	case *ast.BinaryExpr:
		return (e.Op == token.LAND || e.Op == token.LOR) && validCondition(e.X) && validCondition(e.Y)
	}
	return false
}

// Matches fails closed if any referenced property is unknown, unavailable or
// non-boolean, including inside a negation or an otherwise true OR expression.
func (c Condition) Matches(state StateProvider) bool {
	if c.expr == nil {
		return true
	}
	value, valid := evaluateCondition(c.expr, state)
	return valid && value
}

func evaluateCondition(expr ast.Expr, state StateProvider) (bool, bool) {
	if path, ok := statePath(expr); ok {
		if path == "true" {
			return true, true
		}
		if path == "false" {
			return false, true
		}
		if state == nil {
			return false, false
		}
		value, found := state.QueryState(path)
		b, boolean := value.(bool)
		return b, found && boolean
	}
	switch e := expr.(type) {
	case *ast.ParenExpr:
		return evaluateCondition(e.X, state)
	case *ast.UnaryExpr:
		value, valid := evaluateCondition(e.X, state)
		return !value, valid
	case *ast.BinaryExpr:
		left, leftValid := evaluateCondition(e.X, state)
		right, rightValid := evaluateCondition(e.Y, state)
		if e.Op == token.LAND {
			return left && right, leftValid && rightValid
		}
		return left || right, leftValid && rightValid
	}
	return false, false
}
