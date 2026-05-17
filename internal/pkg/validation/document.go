package validation

import (
	"regexp"
	"strconv"
	"strings"
)

var nonDigit = regexp.MustCompile(`\D`)

func stripNonDigits(s string) string {
	return nonDigit.ReplaceAllString(s, "")
}

// ValidateCNPJ validates a Brazilian CNPJ number.
func ValidateCNPJ(cnpj string) bool {
	s := stripNonDigits(cnpj)
	if len(s) != 14 {
		return false
	}
	// Reject all-same-digit CNPJs (e.g., 00000000000000)
	if strings.Count(s, string(s[0])) == 14 {
		return false
	}

	weights1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	weights2 := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}

	digit := func(digits string, weights []int) int {
		sum := 0
		for i, w := range weights {
			d, _ := strconv.Atoi(string(digits[i]))
			sum += d * w
		}
		r := sum % 11
		if r < 2 {
			return 0
		}
		return 11 - r
	}

	d1 := digit(s[:12], weights1)
	d2 := digit(s[:13], weights2)

	return strconv.Itoa(d1) == string(s[12]) && strconv.Itoa(d2) == string(s[13])
}

// ValidateCPF validates a Brazilian CPF number.
func ValidateCPF(cpf string) bool {
	s := stripNonDigits(cpf)
	if len(s) != 11 {
		return false
	}
	if strings.Count(s, string(s[0])) == 11 {
		return false
	}

	digit := func(digits string, n int) int {
		sum := 0
		for i := 0; i < n; i++ {
			d, _ := strconv.Atoi(string(digits[i]))
			sum += d * (n + 1 - i)
		}
		r := (sum * 10) % 11
		if r == 10 || r == 11 {
			return 0
		}
		return r
	}

	d1 := digit(s, 9)
	d2 := digit(s, 10)
	return strconv.Itoa(d1) == string(s[9]) && strconv.Itoa(d2) == string(s[10])
}

// ValidateCNPJOrCPF validates either CNPJ or CPF.
func ValidateCNPJOrCPF(doc string) bool {
	s := stripNonDigits(doc)
	switch len(s) {
	case 11:
		return ValidateCPF(s)
	case 14:
		return ValidateCNPJ(s)
	default:
		return false
	}
}
