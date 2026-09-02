package formula

import (
	"fmt"
	"strconv"
	"unicode"
)

type tokenKind int

const (
	tokNum tokenKind = iota
	tokIdent
	tokPlus
	tokMinus
	tokMul
	tokDiv
	tokLParen
	tokRParen
	tokEOF
)

type token struct {
	kind  tokenKind
	value string
}

func tokenize(expr string) ([]token, error) {
	var raw []token
	i := 0
	for i < len(expr) {
		ch := rune(expr[i])
		switch {
		case ch == ' ' || ch == '\t':
			i++
		case ch == '+':
			raw = append(raw, token{tokPlus, "+"})
			i++
		case ch == '-':
			raw = append(raw, token{tokMinus, "-"})
			i++
		case ch == '*':
			raw = append(raw, token{tokMul, "*"})
			i++
		case ch == '/':
			raw = append(raw, token{tokDiv, "/"})
			i++
		case ch == '(':
			raw = append(raw, token{tokLParen, "("})
			i++
		case ch == ')':
			raw = append(raw, token{tokRParen, ")"})
			i++
		case (ch >= '0' && ch <= '9') || ch == '.':
			j := i
			for j < len(expr) && ((rune(expr[j]) >= '0' && rune(expr[j]) <= '9') || rune(expr[j]) == '.') {
				j++
			}
			raw = append(raw, token{tokNum, expr[i:j]})
			i = j
		case unicode.IsUpper(ch) || ch == '_':
			j := i
			for j < len(expr) && (unicode.IsUpper(rune(expr[j])) || unicode.IsDigit(rune(expr[j])) || rune(expr[j]) == '_') {
				j++
			}
			raw = append(raw, token{tokIdent, expr[i:j]})
			i = j
		default:
			return nil, fmt.Errorf("unexpected character %q at position %d", ch, i)
		}
	}
	raw = append(raw, token{tokEOF, ""})

	// Insert implicit '*' for juxtaposition: 5(...) or IDENT(...) or )(
	out := make([]token, 0, len(raw))
	for idx, tok := range raw {
		out = append(out, tok)
		if idx+1 < len(raw) {
			next := raw[idx+1]
			cur := tok.kind
			if (cur == tokNum || cur == tokIdent || cur == tokRParen) && next.kind == tokLParen {
				out = append(out, token{tokMul, "*"})
			}
		}
	}
	return out, nil
}

type parser struct {
	tokens []token
	pos    int
	vars   map[string]float64
}

func (p *parser) peek() token { return p.tokens[p.pos] }

func (p *parser) consume() token {
	t := p.tokens[p.pos]
	p.pos++
	return t
}

func (p *parser) parseExpr() (float64, error) {
	left, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for p.peek().kind == tokPlus || p.peek().kind == tokMinus {
		op := p.consume()
		right, err := p.parseTerm()
		if err != nil {
			return 0, err
		}
		if op.kind == tokPlus {
			left += right
		} else {
			left -= right
		}
	}
	return left, nil
}

func (p *parser) parseTerm() (float64, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return 0, err
	}
	for p.peek().kind == tokMul || p.peek().kind == tokDiv {
		op := p.consume()
		right, err := p.parsePrimary()
		if err != nil {
			return 0, err
		}
		if op.kind == tokMul {
			left *= right
		} else {
			if right == 0 {
				return 0, fmt.Errorf("division by zero in formula")
			}
			left /= right
		}
	}
	return left, nil
}

func (p *parser) parsePrimary() (float64, error) {
	tok := p.peek()
	switch tok.kind {
	case tokNum:
		p.consume()
		v, err := strconv.ParseFloat(tok.value, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number %q: %w", tok.value, err)
		}
		return v, nil
	case tokIdent:
		p.consume()
		v, ok := p.vars[tok.value]
		if !ok {
			return 0, fmt.Errorf("undefined variable %q in formula", tok.value)
		}
		return v, nil
	case tokMinus:
		p.consume()
		v, err := p.parsePrimary()
		if err != nil {
			return 0, err
		}
		return -v, nil
	case tokLParen:
		p.consume()
		v, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		if p.peek().kind != tokRParen {
			return 0, fmt.Errorf("expected ')' near position %d", p.pos)
		}
		p.consume()
		return v, nil
	default:
		return 0, fmt.Errorf("unexpected token %q at position %d", tok.value, p.pos)
	}
}

// Evaluate evaluates a mathematical formula string, substituting UPPERCASE
// identifiers from vars (question_name → numeric value). Returns an error when
// a variable is undefined or the expression is malformed.
func Evaluate(expr string, vars map[string]float64) (float64, error) {
	tokens, err := tokenize(expr)
	if err != nil {
		return 0, err
	}
	p := &parser{tokens: tokens, vars: vars}
	result, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	if p.peek().kind != tokEOF {
		return 0, fmt.Errorf("trailing content in formula at position %d", p.pos)
	}
	return result, nil
}

// EvaluateSafe evaluates the formula, returning (0, false) on any error.
func EvaluateSafe(expr string, vars map[string]float64) (float64, bool) {
	v, err := Evaluate(expr, vars)
	if err != nil {
		return 0, false
	}
	return v, true
}

// ParseOptionValue extracts the leading numeric value from a string like "1.94M".
func ParseOptionValue(s string) (float64, bool) {
	v, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return v, true
	}
	i := 0
	for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == '.' || (i == 0 && s[i] == '-')) {
		i++
	}
	if i > 0 {
		v, err := strconv.ParseFloat(s[:i], 64)
		if err == nil {
			return v, true
		}
	}
	return 0, false
}

// Variables lista, em ordem de aparição e sem repetição, os identificadores
// usados pela expressão. Uma expressão malformada devolve nil.
func Variables(expr string) []string {
	tokens, err := tokenize(expr)
	if err != nil {
		return nil
	}
	seen := make(map[string]bool, len(tokens))
	out := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		if tok.kind == tokIdent && !seen[tok.value] {
			seen[tok.value] = true
			out = append(out, tok.value)
		}
	}
	return out
}

// Validate reports whether the expression parses. Variables are not required to
// have a value yet — this is the check made when the formula is registered.
func Validate(expr string) error {
	tokens, err := tokenize(expr)
	if err != nil {
		return err
	}
	vars := make(map[string]float64, len(tokens))
	for _, tok := range tokens {
		if tok.kind == tokIdent {
			// Valor arbitrário não-nulo: só queremos exercitar a gramática sem
			// esbarrar em divisão por zero.
			vars[tok.value] = 1
		}
	}
	_, err = Evaluate(expr, vars)
	return err
}
