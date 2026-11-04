package formula

import (
	"fmt"
	"unicode"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/shopspring/decimal"
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
			return nil, domainerr.NewValidation(fmt.Sprintf("caractere inesperado %q na posição %d da fórmula", ch, i))
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
	vars   map[string]decimal.Decimal
}

func (p *parser) peek() token { return p.tokens[p.pos] }

func (p *parser) consume() token {
	t := p.tokens[p.pos]
	p.pos++
	return t
}

func (p *parser) parseExpr() (decimal.Decimal, error) {
	left, err := p.parseTerm()
	if err != nil {
		return decimal.Zero, err
	}
	for p.peek().kind == tokPlus || p.peek().kind == tokMinus {
		op := p.consume()
		right, err := p.parseTerm()
		if err != nil {
			return decimal.Zero, err
		}
		if op.kind == tokPlus {
			left = left.Add(right)
		} else {
			left = left.Sub(right)
		}
	}
	return left, nil
}

func (p *parser) parseTerm() (decimal.Decimal, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return decimal.Zero, err
	}
	for p.peek().kind == tokMul || p.peek().kind == tokDiv {
		op := p.consume()
		right, err := p.parsePrimary()
		if err != nil {
			return decimal.Zero, err
		}
		if op.kind == tokMul {
			left = left.Mul(right)
		} else {
			if right.IsZero() {
				return decimal.Zero, domainerr.NewValidation("divisão por zero na fórmula")
			}
			left = left.Div(right)
		}
	}
	return left, nil
}

func (p *parser) parsePrimary() (decimal.Decimal, error) {
	tok := p.peek()
	switch tok.kind {
	case tokNum:
		p.consume()
		v, err := decimal.NewFromString(tok.value)
		if err != nil {
			return decimal.Zero, domainerr.NewValidation(fmt.Sprintf("número inválido na fórmula: %q", tok.value))
		}
		return v, nil
	case tokIdent:
		p.consume()
		v, ok := p.vars[tok.value]
		if !ok {
			return decimal.Zero, domainerr.NewValidation(fmt.Sprintf("variável %q não definida na fórmula", tok.value))
		}
		return v, nil
	case tokMinus:
		p.consume()
		v, err := p.parsePrimary()
		if err != nil {
			return decimal.Zero, err
		}
		return v.Neg(), nil
	case tokLParen:
		p.consume()
		v, err := p.parseExpr()
		if err != nil {
			return decimal.Zero, err
		}
		if p.peek().kind != tokRParen {
			return decimal.Zero, domainerr.NewValidation(fmt.Sprintf("falta fechar um parêntese perto da posição %d", p.pos))
		}
		p.consume()
		return v, nil
	default:
		return decimal.Zero, domainerr.NewValidation(fmt.Sprintf("símbolo inesperado %q na posição %d da fórmula", tok.value, p.pos))
	}
}

// Evaluate evaluates a mathematical formula string, substituting UPPERCASE
// identifiers from vars (question_name → numeric value). Returns an error when
// a variable is undefined or the expression is malformed. A conta é decimal:
// "0.1 * 3" dá 0,3, não 0,30000000000000004.
func Evaluate(expr string, vars map[string]decimal.Decimal) (decimal.Decimal, error) {
	tokens, err := tokenize(expr)
	if err != nil {
		return decimal.Zero, err
	}
	p := &parser{tokens: tokens, vars: vars}
	result, err := p.parseExpr()
	if err != nil {
		return decimal.Zero, err
	}
	if p.peek().kind != tokEOF {
		return decimal.Zero, domainerr.NewValidation(fmt.Sprintf("conteúdo sobrando na fórmula a partir da posição %d", p.pos))
	}
	return result, nil
}

// EvaluateSafe evaluates the formula, returning (0, false) on any error.
func EvaluateSafe(expr string, vars map[string]decimal.Decimal) (decimal.Decimal, bool) {
	v, err := Evaluate(expr, vars)
	if err != nil {
		return decimal.Zero, false
	}
	return v, true
}

// ParseOptionValue extracts the leading numeric value from a string like "1.94M".
func ParseOptionValue(s string) (decimal.Decimal, bool) {
	v, err := decimal.NewFromString(s)
	if err == nil {
		return v, true
	}
	i := 0
	for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == '.' || (i == 0 && s[i] == '-')) {
		i++
	}
	if i > 0 {
		v, err := decimal.NewFromString(s[:i])
		if err == nil {
			return v, true
		}
	}
	return decimal.Zero, false
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
	vars := make(map[string]decimal.Decimal, len(tokens))
	for _, tok := range tokens {
		if tok.kind == tokIdent {
			// Valor arbitrário não-nulo: só queremos exercitar a gramática sem
			// esbarrar em divisão por zero.
			vars[tok.value] = decimal.NewFromInt(1)
		}
	}
	_, err = Evaluate(expr, vars)
	return err
}
