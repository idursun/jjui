package bindings

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Condition is a parsed boolean expression that can be evaluated against state.
type Condition struct {
	root node
	raw  string
}

func (c Condition) Eval(state map[string]any) bool {
	if c.root == nil {
		return true
	}
	return c.root.eval(state)
}

// ParseCondition parses a VS Code–style when expression.
func ParseCondition(expr string) (Condition, error) {
	tokens, err := lex(expr)
	if err != nil {
		return Condition{}, err
	}
	p := parser{tokens: tokens}
	root, err := p.parse()
	if err != nil {
		return Condition{}, err
	}
	return Condition{root: root, raw: expr}, nil
}

type tokenType int

const (
	tEOF tokenType = iota
	tIdent
	tString
	tNumber
	tBool
	tAnd
	tOr
	tNot
	tEq
	tNeq
	tLParen
	tRParen
)

type token struct {
	typ tokenType
	val string
	pos int
}

func lex(input string) ([]token, error) {
	var tokens []token
	for i := 0; i < len(input); {
		ch := input[i]
		if unicode.IsSpace(rune(ch)) {
			i++
			continue
		}
		switch ch {
		case '&':
			if i+1 < len(input) && input[i+1] == '&' {
				tokens = append(tokens, token{typ: tAnd, pos: i})
				i += 2
				continue
			}
		case '|':
			if i+1 < len(input) && input[i+1] == '|' {
				tokens = append(tokens, token{typ: tOr, pos: i})
				i += 2
				continue
			}
		case '!':
			if i+1 < len(input) && input[i+1] == '=' {
				tokens = append(tokens, token{typ: tNeq, pos: i})
				i += 2
				continue
			}
			tokens = append(tokens, token{typ: tNot, pos: i})
			i++
			continue
		case '=':
			if i+1 < len(input) && input[i+1] == '=' {
				tokens = append(tokens, token{typ: tEq, pos: i})
				i += 2
				continue
			}
		case '(':
			tokens = append(tokens, token{typ: tLParen, pos: i})
			i++
			continue
		case ')':
			tokens = append(tokens, token{typ: tRParen, pos: i})
			i++
			continue
		case '\'', '"':
			quote := ch
			start := i
			i++
			var sb strings.Builder
			for i < len(input) && input[i] != quote {
				sb.WriteByte(input[i])
				i++
			}
			if i >= len(input) {
				return nil, fmt.Errorf("unterminated string starting at %d", start)
			}
			i++ // closing quote
			tokens = append(tokens, token{typ: tString, val: sb.String(), pos: start})
			continue
		}

		if isIdentStart(ch) {
			start := i
			i++
			for i < len(input) && isIdentPart(input[i]) {
				i++
			}
			val := input[start:i]
			switch val {
			case "true", "false":
				tokens = append(tokens, token{typ: tBool, val: val, pos: start})
			default:
				tokens = append(tokens, token{typ: tIdent, val: val, pos: start})
			}
			continue
		}

		if isDigit(ch) {
			start := i
			i++
			for i < len(input) && isDigit(input[i]) {
				i++
			}
			tokens = append(tokens, token{typ: tNumber, val: input[start:i], pos: start})
			continue
		}

		return nil, fmt.Errorf("unexpected character %q at %d", ch, i)
	}
	tokens = append(tokens, token{typ: tEOF, pos: len(input)})
	return tokens, nil
}

func isIdentStart(b byte) bool {
	return unicode.IsLetter(rune(b)) || b == '_' || b == '.'
}

func isIdentPart(b byte) bool {
	return isIdentStart(b) || isDigit(b)
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

type parser struct {
	tokens []token
	pos    int
}

func (p *parser) current() token {
	return p.tokens[p.pos]
}

func (p *parser) consume() token {
	t := p.current()
	if t.typ != tEOF {
		p.pos++
	}
	return t
}

func (p *parser) parse() (node, error) {
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.current().typ != tEOF {
		return nil, fmt.Errorf("unexpected token at %d", p.current().pos)
	}
	return expr, nil
}

func (p *parser) parseOr() (node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.current().typ == tOr {
		p.consume()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = binaryNode{op: tOr, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseAnd() (node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.current().typ == tAnd {
		p.consume()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = binaryNode{op: tAnd, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseUnary() (node, error) {
	if p.current().typ == tNot {
		p.consume()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return unaryNode{op: tNot, value: operand}, nil
	}
	return p.parseComparison()
}

func (p *parser) parseComparison() (node, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	switch p.current().typ {
	case tEq, tNeq:
		op := p.consume().typ
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return compareNode{op: op, left: left, right: right}, nil
	}
	return left, nil
}

func (p *parser) parsePrimary() (node, error) {
	tok := p.current()
	switch tok.typ {
	case tLParen:
		p.consume()
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.current().typ != tRParen {
			return nil, fmt.Errorf("expected ')' at %d", p.current().pos)
		}
		p.consume()
		return expr, nil
	case tIdent, tString, tNumber, tBool:
		p.consume()
		return literalNode{token: tok}, nil
	default:
		return nil, fmt.Errorf("unexpected token at %d", tok.pos)
	}
}

type node interface {
	eval(state map[string]any) bool
}

type literalNode struct {
	token token
}

func (n literalNode) eval(state map[string]any) bool {
	val := n.value(state)
	switch v := val.(type) {
	case bool:
		return v
	case string:
		return v != ""
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	default:
		return false
	}
}

func (n literalNode) value(state map[string]any) any {
	switch n.token.typ {
	case tIdent:
		return state[n.token.val]
	case tString:
		return n.token.val
	case tBool:
		return n.token.val == "true"
	case tNumber:
		if strings.Contains(n.token.val, ".") {
			if f, err := strconv.ParseFloat(n.token.val, 64); err == nil {
				return f
			}
			return nil
		}
		if i, err := strconv.Atoi(n.token.val); err == nil {
			return i
		}
		return nil
	default:
		return nil
	}
}

type unaryNode struct {
	op    tokenType
	value node
}

func (n unaryNode) eval(state map[string]any) bool {
	switch n.op {
	case tNot:
		return !n.value.eval(state)
	default:
		return false
	}
}

type binaryNode struct {
	op    tokenType
	left  node
	right node
}

func (n binaryNode) eval(state map[string]any) bool {
	switch n.op {
	case tAnd:
		return n.left.eval(state) && n.right.eval(state)
	case tOr:
		return n.left.eval(state) || n.right.eval(state)
	default:
		return false
	}
}

type compareNode struct {
	op    tokenType
	left  node
	right node
}

func (n compareNode) eval(state map[string]any) bool {
	lv := valueOf(n.left, state)
	rv := valueOf(n.right, state)

	switch l := lv.(type) {
	case string:
		r, ok := rv.(string)
		if !ok {
			return false
		}
		if n.op == tEq {
			return l == r
		}
		return l != r
	case bool:
		r, ok := rv.(bool)
		if !ok {
			return false
		}
		if n.op == tEq {
			return l == r
		}
		return l != r
	case int:
		r, ok := rv.(int)
		if !ok {
			return false
		}
		if n.op == tEq {
			return l == r
		}
		return l != r
	case int64:
		switch r := rv.(type) {
		case int64:
			if n.op == tEq {
				return l == r
			}
			return l != r
		case int:
			if n.op == tEq {
				return l == int64(r)
			}
			return l != int64(r)
		default:
			return false
		}
	case float64:
		r, ok := toFloat(rv)
		if !ok {
			return false
		}
		if n.op == tEq {
			return l == r
		}
		return l != r
	default:
		return false
	}
}

func valueOf(n node, state map[string]any) any {
	switch v := n.(type) {
	case literalNode:
		return v.value(state)
	default:
		if v.eval(state) {
			return true
		}
		return false
	}
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
