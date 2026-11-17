package approvals

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type celTokKind int

const (
	tokEOF celTokKind = iota
	tokIdent
	tokInt
	tokString
	tokBool
	tokNull
	tokLParen
	tokRParen
	tokLBracket
	tokRBracket
	tokComma
	tokDot
	tokBang
	tokAnd
	tokOr
	tokEq
	tokNeq
	tokLt
	tokLte
	tokGt
	tokGte
	tokIn
	tokNotIn
)

type celToken struct {
	kind celTokKind
	val  string
	pos  int
}

func tokenizeCEL(src string) ([]celToken, error) {
	tokens := make([]celToken, 0, 16)
	i := 0
	n := len(src)

	for i < n {
		ch := src[i]
		if unicode.IsSpace(rune(ch)) {
			i++
			continue
		}
		start := i
		switch ch {
		case '(':
			tokens = append(tokens, celToken{kind: tokLParen, val: "(", pos: start})
			i++
		case ')':
			tokens = append(tokens, celToken{kind: tokRParen, val: ")", pos: start})
			i++
		case '[':
			tokens = append(tokens, celToken{kind: tokLBracket, val: "[", pos: start})
			i++
		case ']':
			tokens = append(tokens, celToken{kind: tokRBracket, val: "]", pos: start})
			i++
		case ',':
			tokens = append(tokens, celToken{kind: tokComma, val: ",", pos: start})
			i++
		case '.':
			// Reject leading-dot float literal like ".5"
			if i+1 < n && src[i+1] >= '0' && src[i+1] <= '9' {
				return nil, fmt.Errorf("%w: floating-point literal at pos %d", ErrInvalidCEL, start)
			}
			tokens = append(tokens, celToken{kind: tokDot, val: ".", pos: start})
			i++
		case '!':
			if i+1 < n && src[i+1] == '=' {
				tokens = append(tokens, celToken{kind: tokNeq, val: "!=", pos: start})
				i += 2
			} else {
				tokens = append(tokens, celToken{kind: tokBang, val: "!", pos: start})
				i++
			}
		case '=':
			if i+1 < n && src[i+1] == '=' {
				tokens = append(tokens, celToken{kind: tokEq, val: "==", pos: start})
				i += 2
			} else {
				return nil, fmt.Errorf("%w: unexpected '=' at pos %d (use '==')", ErrInvalidCEL, start)
			}
		case '&':
			if i+1 < n && src[i+1] == '&' {
				tokens = append(tokens, celToken{kind: tokAnd, val: "&&", pos: start})
				i += 2
			} else {
				return nil, fmt.Errorf("%w: unexpected '&' at pos %d (use '&&')", ErrInvalidCEL, start)
			}
		case '|':
			if i+1 < n && src[i+1] == '|' {
				tokens = append(tokens, celToken{kind: tokOr, val: "||", pos: start})
				i += 2
			} else {
				return nil, fmt.Errorf("%w: unexpected '|' at pos %d (use '||')", ErrInvalidCEL, start)
			}
		case '<':
			if i+1 < n && src[i+1] == '=' {
				tokens = append(tokens, celToken{kind: tokLte, val: "<=", pos: start})
				i += 2
			} else {
				tokens = append(tokens, celToken{kind: tokLt, val: "<", pos: start})
				i++
			}
		case '>':
			if i+1 < n && src[i+1] == '=' {
				tokens = append(tokens, celToken{kind: tokGte, val: ">=", pos: start})
				i += 2
			} else {
				tokens = append(tokens, celToken{kind: tokGt, val: ">", pos: start})
				i++
			}
		case '"', '\'':
			quote := ch
			i++
			var sb strings.Builder
			closed := false
			for i < n {
				c := src[i]
				if c == '\\' {
					i++
					if i >= n {
						return nil, fmt.Errorf("%w: unterminated string escape at pos %d", ErrInvalidCEL, start)
					}
					switch src[i] {
					case 'n':
						sb.WriteByte('\n')
					case 't':
						sb.WriteByte('\t')
					case 'r':
						sb.WriteByte('\r')
					case '\\', '"', '\'':
						sb.WriteByte(src[i])
					default:
						return nil, fmt.Errorf("%w: invalid escape '\\%c' at pos %d", ErrInvalidCEL, src[i], i)
					}
					i++
					continue
				}
				if c == quote {
					closed = true
					i++
					break
				}
				sb.WriteByte(c)
				i++
			}
			if !closed {
				return nil, fmt.Errorf("%w: unterminated string literal at pos %d", ErrInvalidCEL, start)
			}
			tokens = append(tokens, celToken{kind: tokString, val: sb.String(), pos: start})
		default:
			if ch == '-' || (ch >= '0' && ch <= '9') {
				if ch == '-' {
					if i+1 >= n || src[i+1] < '0' || src[i+1] > '9' {
						return nil, fmt.Errorf("%w: unexpected '-' at pos %d", ErrInvalidCEL, start)
					}
					i++
				}
				for i < n && src[i] >= '0' && src[i] <= '9' {
					i++
				}
				if i < n && (src[i] == '.' || src[i] == 'e' || src[i] == 'E') {
					return nil, fmt.Errorf("%w: floating-point literal near pos %d (%w)", ErrInvalidCEL, start, ErrFloatForbidden)
				}
				if i < n && isIdentPart(src[i]) {
					return nil, fmt.Errorf("%w: invalid integer literal at pos %d", ErrInvalidCEL, start)
				}
				numStr := src[start:i]
				if _, err := strconv.ParseInt(numStr, 10, 64); err != nil {
					return nil, fmt.Errorf("%w: int64 overflow %q at pos %d", ErrInvalidCEL, numStr, start)
				}
				tokens = append(tokens, celToken{kind: tokInt, val: numStr, pos: start})
				continue
			}
			if isIdentStart(ch) {
				for i < n && isIdentPart(src[i]) {
					i++
				}
				word := src[start:i]
				switch word {
				case "true", "false":
					tokens = append(tokens, celToken{kind: tokBool, val: word, pos: start})
				case "null":
					tokens = append(tokens, celToken{kind: tokNull, val: word, pos: start})
				case "in":
					tokens = append(tokens, celToken{kind: tokIn, val: word, pos: start})
				case "not_in":
					tokens = append(tokens, celToken{kind: tokNotIn, val: word, pos: start})
				default:
					tokens = append(tokens, celToken{kind: tokIdent, val: word, pos: start})
				}
				continue
			}
			return nil, fmt.Errorf("%w: unexpected character %q at pos %d", ErrInvalidCEL, ch, start)
		}
	}
	tokens = append(tokens, celToken{kind: tokEOF, pos: n})
	return tokens, nil
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

type celNode interface {
	eval(ctx map[string]any) (any, error)
}

type celOrNode struct {
	left, right celNode
}

type celAndNode struct {
	left, right celNode
}

type celNotNode struct {
	inner celNode
}

type celCmpNode struct {
	op          celTokKind
	left, right celNode
}

type celInNode struct {
	negate    bool
	left      celNode
	items     []celNode
	listField celNode
}

type celLiteralNode struct {
	val any
}

type celFieldNode struct {
	path []string
	full string
}

type celMethodNode struct {
	target []string
	method string
	arg    celNode
}

type celSizeNode struct {
	target celNode
}

// CompiledCEL is an immutable, deterministic, zero-dependency compiled CEL AST.
type CompiledCEL struct {
	Source string
	root   celNode
}

// ValidateCEL parses and validates a CEL condition expression, returning ErrInvalidCEL on syntax errors.
func ValidateCEL(expr string) error {
	_, err := CompileCEL(expr)
	return err
}

const (
	maxCELSourceBytes = 8192
	maxCELDepth       = 64
)

// CompileCEL parses expr into a deterministic AST and rejects malformed expressions or float literals.
func CompileCEL(expr string) (*CompiledCEL, error) {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: empty expression", ErrInvalidCEL)
	}
	if len(trimmed) > maxCELSourceBytes {
		return nil, fmt.Errorf("%w: expression exceeds %d bytes", ErrInvalidCEL, maxCELSourceBytes)
	}
	toks, err := tokenizeCEL(trimmed)
	if err != nil {
		return nil, err
	}
	p := &celParser{tokens: toks}
	root, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tokEOF {
		return nil, fmt.Errorf("%w: unexpected token %q at pos %d", ErrInvalidCEL, p.peek().val, p.peek().pos)
	}
	if !isBooleanCapableNode(root) {
		return nil, fmt.Errorf("%w: root expression must evaluate to a boolean condition", ErrInvalidCEL)
	}
	return &CompiledCEL{Source: trimmed, root: root}, nil
}

func isBooleanCapableNode(n celNode) bool {
	switch t := n.(type) {
	case *celOrNode, *celAndNode, *celNotNode, *celCmpNode, *celInNode, *celMethodNode, *celFieldNode:
		return true
	case *celLiteralNode:
		_, ok := t.val.(bool)
		return ok
	default:
		return false
	}
}

// EvaluateCEL compiles and evaluates expr against ctx in a single deterministic pass.
func EvaluateCEL(expr string, ctx map[string]any) (bool, error) {
	compiled, err := CompileCEL(expr)
	if err != nil {
		return false, err
	}
	return compiled.Eval(ctx)
}

// Eval evaluates the compiled CEL AST against ctx (rejecting any float values in ctx).
func (c *CompiledCEL) Eval(ctx map[string]any) (bool, error) {
	if c == nil || c.root == nil {
		return false, fmt.Errorf("%w: uncompiled expression", ErrInvalidCEL)
	}
	if ctx == nil {
		ctx = map[string]any{}
	}
	if err := validateNoFloats(ctx); err != nil {
		return false, err
	}
	v, err := c.root.eval(ctx)
	if err != nil {
		return false, err
	}
	if v == nil {
		return false, nil
	}
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("%w: condition evaluated to %T instead of bool", ErrInvalidCEL, v)
	}
	return b, nil
}

type celParser struct {
	tokens []celToken
	idx    int
	depth  int
}

func (p *celParser) enter() error {
	p.depth++
	if p.depth > maxCELDepth {
		return fmt.Errorf("%w: nesting depth exceeds %d", ErrInvalidCEL, maxCELDepth)
	}
	return nil
}

func (p *celParser) leave() {
	if p.depth > 0 {
		p.depth--
	}
}

func (p *celParser) peek() celToken {
	if p.idx >= len(p.tokens) {
		return celToken{kind: tokEOF}
	}
	return p.tokens[p.idx]
}

func (p *celParser) next() celToken {
	tok := p.peek()
	if p.idx < len(p.tokens) {
		p.idx++
	}
	return tok
}

func (p *celParser) parseOr() (celNode, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokOr {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &celOrNode{left: left, right: right}
	}
	return left, nil
}

func (p *celParser) parseAnd() (celNode, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokAnd {
		p.next()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &celAndNode{left: left, right: right}
	}
	return left, nil
}

func (p *celParser) parseNot() (celNode, error) {
	if p.peek().kind == tokBang {
		if err := p.enter(); err != nil {
			return nil, err
		}
		defer p.leave()
		p.next()
		inner, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &celNotNode{inner: inner}, nil
	}
	return p.parseCmp()
}

func (p *celParser) parseCmp() (celNode, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	op := p.peek()
	switch op.kind {
	case tokEq, tokNeq, tokLt, tokLte, tokGt, tokGte:
		p.next()
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return &celCmpNode{op: op.kind, left: left, right: right}, nil
	case tokIn, tokNotIn:
		p.next()
		if p.peek().kind == tokIdent {
			rhs, err := p.parsePrimary()
			if err != nil {
				return nil, err
			}
			return &celInNode{negate: op.kind == tokNotIn, left: left, listField: rhs}, nil
		}
		items, err := p.parseInList()
		if err != nil {
			return nil, err
		}
		return &celInNode{negate: op.kind == tokNotIn, left: left, items: items}, nil
	default:
		return left, nil
	}
}

func (p *celParser) parseInList() ([]celNode, error) {
	open := p.peek()
	var closeKind celTokKind
	switch open.kind {
	case tokLBracket:
		closeKind = tokRBracket
	case tokLParen:
		closeKind = tokRParen
	default:
		return nil, fmt.Errorf("%w: expected '[' or '(' after 'in' at pos %d", ErrInvalidCEL, open.pos)
	}
	p.next()
	items := make([]celNode, 0, 4)
	if p.peek().kind == closeKind {
		p.next()
		return items, nil
	}
	for {
		elem, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		items = append(items, elem)
		if p.peek().kind == tokComma {
			p.next()
			continue
		}
		if p.peek().kind == closeKind {
			p.next()
			return items, nil
		}
		return nil, fmt.Errorf("%w: expected ',' or closing bracket in list at pos %d", ErrInvalidCEL, p.peek().pos)
	}
}

func (p *celParser) parsePrimary() (celNode, error) {
	if err := p.enter(); err != nil {
		return nil, err
	}
	defer p.leave()
	tok := p.peek()
	switch tok.kind {
	case tokLParen:
		p.next()
		inner, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tokRParen {
			return nil, fmt.Errorf("%w: missing closing ')' at pos %d", ErrInvalidCEL, p.peek().pos)
		}
		p.next()
		return inner, nil
	case tokInt:
		p.next()
		v, err := strconv.ParseInt(tok.val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid int64 %q", ErrInvalidCEL, tok.val)
		}
		return &celLiteralNode{val: v}, nil
	case tokString:
		p.next()
		return &celLiteralNode{val: tok.val}, nil
	case tokBool:
		p.next()
		return &celLiteralNode{val: tok.val == "true"}, nil
	case tokNull:
		p.next()
		return &celLiteralNode{val: nil}, nil
	case tokIdent:
		p.next()
		if tok.val == "size" && p.peek().kind == tokLParen {
			p.next()
			arg, err := p.parsePrimary()
			if err != nil {
				return nil, err
			}
			if p.peek().kind != tokRParen {
				return nil, fmt.Errorf("%w: missing ')' after size(...) at pos %d", ErrInvalidCEL, p.peek().pos)
			}
			p.next()
			return &celSizeNode{target: arg}, nil
		}
		parts := []string{tok.val}
		for p.peek().kind == tokDot {
			p.next()
			seg := p.peek()
			if seg.kind != tokIdent {
				return nil, fmt.Errorf("%w: expected field name after '.' at pos %d", ErrInvalidCEL, seg.pos)
			}
			p.next()
			parts = append(parts, seg.val)
		}
		if p.peek().kind == tokLParen {
			if len(parts) < 2 {
				return nil, fmt.Errorf("%w: unsupported function call %q at pos %d", ErrInvalidCEL, parts[0], tok.pos)
			}
			method := parts[len(parts)-1]
			target := parts[:len(parts)-1]
			switch method {
			case "contains", "startsWith", "endsWith":
			default:
				return nil, fmt.Errorf("%w: unsupported method %q at pos %d", ErrInvalidCEL, method, tok.pos)
			}
			p.next()
			arg, err := p.parsePrimary()
			if err != nil {
				return nil, err
			}
			if p.peek().kind != tokRParen {
				return nil, fmt.Errorf("%w: missing ')' after %s(...) at pos %d", ErrInvalidCEL, method, p.peek().pos)
			}
			p.next()
			return &celMethodNode{target: target, method: method, arg: arg}, nil
		}
		return &celFieldNode{path: parts, full: strings.Join(parts, ".")}, nil
	default:
		return nil, fmt.Errorf("%w: unexpected token %q at pos %d", ErrInvalidCEL, tok.val, tok.pos)
	}
}

func (n *celOrNode) eval(ctx map[string]any) (any, error) {
	lv, err := n.left.eval(ctx)
	if err != nil {
		return nil, err
	}
	lb, ok := toBool(lv)
	if !ok {
		return nil, fmt.Errorf("%w: left side of '||' is %T, want bool", ErrInvalidCEL, lv)
	}
	if lb {
		return true, nil
	}
	rv, err := n.right.eval(ctx)
	if err != nil {
		return nil, err
	}
	rb, ok := toBool(rv)
	if !ok {
		return nil, fmt.Errorf("%w: right side of '||' is %T, want bool", ErrInvalidCEL, rv)
	}
	return rb, nil
}

func (n *celAndNode) eval(ctx map[string]any) (any, error) {
	lv, err := n.left.eval(ctx)
	if err != nil {
		return nil, err
	}
	lb, ok := toBool(lv)
	if !ok {
		return nil, fmt.Errorf("%w: left side of '&&' is %T, want bool", ErrInvalidCEL, lv)
	}
	if !lb {
		return false, nil
	}
	rv, err := n.right.eval(ctx)
	if err != nil {
		return nil, err
	}
	rb, ok := toBool(rv)
	if !ok {
		return nil, fmt.Errorf("%w: right side of '&&' is %T, want bool", ErrInvalidCEL, rv)
	}
	return rb, nil
}

func (n *celNotNode) eval(ctx map[string]any) (any, error) {
	v, err := n.inner.eval(ctx)
	if err != nil {
		return nil, err
	}
	b, ok := toBool(v)
	if !ok {
		return nil, fmt.Errorf("%w: operand of '!' is %T, want bool", ErrInvalidCEL, v)
	}
	return !b, nil
}

func (n *celCmpNode) eval(ctx map[string]any) (any, error) {
	lv, err := n.left.eval(ctx)
	if err != nil {
		return nil, err
	}
	rv, err := n.right.eval(ctx)
	if err != nil {
		return nil, err
	}

	switch n.op {
	case tokEq:
		return celEqual(lv, rv), nil
	case tokNeq:
		return !celEqual(lv, rv), nil
	case tokLt, tokLte, tokGt, tokGte:
		if lv == nil || rv == nil {
			return false, nil
		}
		li, lok := toInt64(lv)
		ri, rok := toInt64(rv)
		if lok && rok {
			switch n.op {
			case tokLt:
				return li < ri, nil
			case tokLte:
				return li <= ri, nil
			case tokGt:
				return li > ri, nil
			case tokGte:
				return li >= ri, nil
			}
		}
		ls, lsok := lv.(string)
		rs, rsok := rv.(string)
		if lsok && rsok {
			switch n.op {
			case tokLt:
				return ls < rs, nil
			case tokLte:
				return ls <= rs, nil
			case tokGt:
				return ls > rs, nil
			case tokGte:
				return ls >= rs, nil
			}
		}
		return false, fmt.Errorf("%w: incompatible comparison types %T and %T", ErrInvalidCEL, lv, rv)
	default:
		return false, fmt.Errorf("%w: unknown comparison operator", ErrInvalidCEL)
	}
}

func (n *celInNode) eval(ctx map[string]any) (any, error) {
	lv, err := n.left.eval(ctx)
	if err != nil {
		return nil, err
	}
	matched := false
	if n.listField != nil {
		rv, err := n.listField.eval(ctx)
		if err != nil {
			return nil, err
		}
		switch s := rv.(type) {
		case []string:
			for _, elem := range s {
				if celEqual(lv, elem) {
					matched = true
					break
				}
			}
		case []int64:
			for _, elem := range s {
				if celEqual(lv, elem) {
					matched = true
					break
				}
			}
		case []int:
			for _, elem := range s {
				if celEqual(lv, elem) {
					matched = true
					break
				}
			}
		case []any:
			for _, elem := range s {
				norm, err := normalizeCELValue(elem)
				if err != nil {
					return nil, err
				}
				if celEqual(lv, norm) {
					matched = true
					break
				}
			}
		case nil:
			matched = false
		default:
			return false, fmt.Errorf("%w: right side of 'in' must be a list, got %T", ErrInvalidCEL, rv)
		}
	} else {
		for _, item := range n.items {
			iv, err := item.eval(ctx)
			if err != nil {
				return nil, err
			}
			if celEqual(lv, iv) {
				matched = true
				break
			}
		}
	}
	if n.negate {
		return !matched, nil
	}
	return matched, nil
}

func (n *celLiteralNode) eval(_ map[string]any) (any, error) {
	return n.val, nil
}

func (n *celFieldNode) eval(ctx map[string]any) (any, error) {
	if len(n.path) == 0 {
		return nil, nil
	}
	if v, ok := ctx[n.full]; ok {
		return normalizeCELValue(v)
	}
	var cur any = ctx
	for _, seg := range n.path {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, nil
		}
		next, exists := m[seg]
		if !exists {
			return nil, nil
		}
		cur = next
	}
	return normalizeCELValue(cur)
}

func (n *celMethodNode) eval(ctx map[string]any) (any, error) {
	field := &celFieldNode{path: n.target, full: strings.Join(n.target, ".")}
	tv, err := field.eval(ctx)
	if err != nil {
		return nil, err
	}
	if tv == nil {
		return false, nil
	}
	ts, ok := tv.(string)
	if !ok {
		return false, nil
	}
	av, err := n.arg.eval(ctx)
	if err != nil {
		return nil, err
	}
	as, ok := av.(string)
	if !ok {
		return false, fmt.Errorf("%w: %s() expects string argument, got %T", ErrInvalidCEL, n.method, av)
	}
	switch n.method {
	case "contains":
		return strings.Contains(ts, as), nil
	case "startsWith":
		return strings.HasPrefix(ts, as), nil
	case "endsWith":
		return strings.HasSuffix(ts, as), nil
	default:
		return false, fmt.Errorf("%w: unsupported method %s", ErrInvalidCEL, n.method)
	}
}

func (n *celSizeNode) eval(ctx map[string]any) (any, error) {
	v, err := n.target.eval(ctx)
	if err != nil {
		return nil, err
	}
	switch tv := v.(type) {
	case nil:
		return int64(0), nil
	case string:
		return int64(len(tv)), nil
	case []string:
		return int64(len(tv)), nil
	case []int64:
		return int64(len(tv)), nil
	case []any:
		return int64(len(tv)), nil
	case map[string]any:
		return int64(len(tv)), nil
	default:
		return int64(0), nil
	}
}

func normalizeCELValue(v any) (any, error) {
	switch tv := v.(type) {
	case float32, float64:
		return nil, ErrFloatForbidden
	case json.Number:
		s := tv.String()
		if strings.ContainsAny(s, ".eE") {
			return nil, ErrFloatForbidden
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid integer %q", ErrInvalidCEL, s)
		}
		return n, nil
	case int:
		return int64(tv), nil
	case int32:
		return int64(tv), nil
	case int64:
		return tv, nil
	case uint:
		return int64(tv), nil
	case uint32:
		return int64(tv), nil
	case uint64:
		return int64(tv), nil
	default:
		return v, nil
	}
}

func toBool(v any) (bool, bool) {
	if v == nil {
		return false, true
	}
	b, ok := v.(bool)
	return b, ok
}

func toInt64(v any) (int64, bool) {
	switch tv := v.(type) {
	case int:
		return int64(tv), true
	case int32:
		return int64(tv), true
	case int64:
		return tv, true
	case uint:
		return int64(tv), true
	case uint32:
		return int64(tv), true
	case uint64:
		return int64(tv), true
	case json.Number:
		if strings.ContainsAny(tv.String(), ".eE") {
			return 0, false
		}
		n, err := strconv.ParseInt(tv.String(), 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func celEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if ai, aok := toInt64(a); aok {
		if bi, bok := toInt64(b); bok {
			return ai == bi
		}
		return false
	}
	if ab, aok := a.(bool); aok {
		bb, bok := b.(bool)
		return bok && ab == bb
	}
	if as, aok := a.(string); aok {
		bs, bok := b.(string)
		return bok && as == bs
	}
	return false
}
