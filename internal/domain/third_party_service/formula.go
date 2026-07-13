package third_party_service

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/shopspring/decimal"
)

// EvaluateFormula evaluates the intentionally small, deterministic price formula
// language: decimal literals, uppercase variables, + - * / and parentheses.
func EvaluateFormula(expr string, vars map[string]decimal.Decimal) (decimal.Decimal, error) {
	tokens := []string{}
	for i := 0; i < len(expr); {
		r := rune(expr[i])
		if unicode.IsSpace(r) {
			i++
			continue
		}
		if strings.ContainsRune("+-*/()", r) {
			tokens = append(tokens, string(r))
			i++
			continue
		}
		j := i
		if unicode.IsDigit(r) || r == '.' {
			for j < len(expr) && (unicode.IsDigit(rune(expr[j])) || expr[j] == '.') {
				j++
			}
		} else if unicode.IsUpper(r) || r == '_' {
			for j < len(expr) && (unicode.IsUpper(rune(expr[j])) || unicode.IsDigit(rune(expr[j])) || expr[j] == '_') {
				j++
			}
		} else {
			return decimal.Zero, fmt.Errorf("invalid formula character")
		}
		tokens = append(tokens, expr[i:j])
		i = j
	}
	prec := func(v string) int {
		if v == "*" || v == "/" {
			return 2
		}
		if v == "+" || v == "-" {
			return 1
		}
		return 0
	}
	out, ops := []string{}, []string{}
	for _, t := range tokens {
		switch t {
		case "+", "-", "*", "/":
			for len(ops) > 0 && prec(ops[len(ops)-1]) >= prec(t) {
				out = append(out, ops[len(ops)-1])
				ops = ops[:len(ops)-1]
			}
			ops = append(ops, t)
		case "(":
			ops = append(ops, t)
		case ")":
			for len(ops) > 0 && ops[len(ops)-1] != "(" {
				out = append(out, ops[len(ops)-1])
				ops = ops[:len(ops)-1]
			}
			if len(ops) == 0 {
				return decimal.Zero, fmt.Errorf("unbalanced formula")
			}
			ops = ops[:len(ops)-1]
		default:
			out = append(out, t)
		}
	}
	for len(ops) > 0 {
		if ops[len(ops)-1] == "(" {
			return decimal.Zero, fmt.Errorf("unbalanced formula")
		}
		out = append(out, ops[len(ops)-1])
		ops = ops[:len(ops)-1]
	}
	stack := []decimal.Decimal{}
	for _, t := range out {
		if !strings.Contains("+-*/", t) || len(t) != 1 {
			v, e := decimal.NewFromString(t)
			if e != nil {
				var ok bool
				v, ok = vars[t]
				if !ok {
					return decimal.Zero, fmt.Errorf("formula variable %s is undefined", t)
				}
			}
			stack = append(stack, v)
			continue
		}
		if len(stack) < 2 {
			return decimal.Zero, fmt.Errorf("invalid formula")
		}
		b, a := stack[len(stack)-1], stack[len(stack)-2]
		stack = stack[:len(stack)-2]
		var v decimal.Decimal
		switch t {
		case "+":
			v = a.Add(b)
		case "-":
			v = a.Sub(b)
		case "*":
			v = a.Mul(b)
		case "/":
			if b.IsZero() {
				return decimal.Zero, fmt.Errorf("formula division by zero")
			}
			v = a.Div(b)
		}
		stack = append(stack, v)
	}
	if len(stack) != 1 {
		return decimal.Zero, fmt.Errorf("invalid formula")
	}
	return stack[0], nil
}
