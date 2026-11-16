package validation

import "strings"

// IEValida confere o formato e o dígito da inscrição estadual pela regra da UF
// (roteiros do SINTEGRA). "ISENTO" e vazio passam; produtor rural de SP ("P…")
// passa sem conferência. ok=false também quando a UF não é conhecida.
//
// Serve para AVISAR antes da transmissão: a SEFAZ é a palavra final e ainda
// recusa IE válida no dígito mas não cadastrada (rejeição 233).
func IEValida(ie, uf string) bool {
	ie = strings.ToUpper(strings.TrimSpace(ie))
	if ie == "" || ie == "ISENTO" {
		return true
	}
	uf = strings.ToUpper(strings.TrimSpace(uf))
	if uf == "SP" && strings.HasPrefix(ie, "P") {
		return true
	}
	d := stripNonDigits(ie)
	if d == "" {
		return false
	}
	if f, ok := regrasIE[uf]; ok {
		return f(d)
	}
	return false
}

func dig(s string, i int) int { return int(s[i] - '0') }

// soma pondera os dígitos de s pelos pesos, na ordem.
func soma(s string, pesos ...int) int {
	t := 0
	for i, p := range pesos {
		t += dig(s, i) * p
	}
	return t
}

// decrescente: pesos n, n-1, …, 2.
func decrescente(n int) []int {
	p := make([]int, 0, n-1)
	for i := n; i >= 2; i-- {
		p = append(p, i)
	}
	return p
}

// mod11 "resto < 2 → 0": a regra mais comum.
func mod11(t int) int {
	if r := t % 11; r >= 2 {
		return 11 - r
	}
	return 0
}

// mod11Dez: 11 − resto, e 10 ou 11 viram 0.
func mod11Dez(t int) int {
	if d := 11 - t%11; d < 10 {
		return d
	}
	return 0
}

// simples9: 9 dígitos, pesos 9..2 sobre os 8 primeiros.
func simples9(regra func(int) int) func(string) bool {
	return func(s string) bool {
		return len(s) == 9 && regra(soma(s, decrescente(9)...)) == dig(s, 8)
	}
}

var regrasIE = map[string]func(string) bool{
	"AC": func(s string) bool {
		return len(s) == 13 && strings.HasPrefix(s, "01") &&
			mod11Dez(soma(s, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2)) == dig(s, 11) &&
			mod11Dez(soma(s, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2)) == dig(s, 12)
	},
	"AL": func(s string) bool {
		return len(s) == 9 && strings.HasPrefix(s, "24") && (soma(s, decrescente(9)...)*10)%11%10 == dig(s, 8)
	},
	"AP": func(s string) bool {
		if len(s) != 9 || !strings.HasPrefix(s, "03") {
			return false
		}
		n, p, d := atoi(s[:8]), 0, 0
		switch {
		case n >= 3000001 && n <= 3017000:
			p, d = 5, 0
		case n >= 3017001 && n <= 3019022:
			p, d = 9, 1
		}
		x := 11 - (p+soma(s, decrescente(9)...))%11
		switch x {
		case 10:
			x = 0
		case 11:
			x = d
		}
		return x == dig(s, 8)
	},
	"AM": func(s string) bool {
		if len(s) != 9 {
			return false
		}
		t := soma(s, decrescente(9)...)
		x := 0
		if t < 11 {
			x = 11 - t
		} else {
			x = mod11(t)
		}
		return x == dig(s, 8)
	},
	"BA": func(s string) bool {
		if len(s) != 8 && len(s) != 9 {
			return false
		}
		base := len(s) - 2
		chave := dig(s, 0)
		if len(s) == 9 {
			chave = dig(s, 1)
		}
		regra := func(t int) int { // módulo 11
			return mod11(t)
		}
		if chave <= 5 || chave == 8 { // módulo 10
			regra = func(t int) int { return (10 - t%10) % 10 }
		}
		// O segundo dígito sai primeiro, só da base; o primeiro leva o segundo junto.
		d2 := regra(soma(s, decrescente(base+1)...))
		comD2 := s[:base] + string(rune('0'+d2))
		d1 := regra(soma(comD2, decrescente(base+2)...))
		return d1 == dig(s, base) && d2 == dig(s, base+1)
	},
	"CE": simples9(mod11Dez),
	"DF": func(s string) bool {
		return len(s) == 13 && strings.HasPrefix(s, "07") &&
			mod11Dez(soma(s, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2)) == dig(s, 11) &&
			mod11Dez(soma(s, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2)) == dig(s, 12)
	},
	"ES": simples9(mod11),
	"GO": func(s string) bool {
		if len(s) != 9 {
			return false
		}
		r := soma(s, decrescente(9)...) % 11
		x := 0
		switch {
		case r == 0:
		case r == 1:
			if n := atoi(s[:8]); n >= 10103105 && n <= 10119997 {
				x = 1
			}
		default:
			x = 11 - r
		}
		return x == dig(s, 8)
	},
	"MA": simples9(mod11),
	"MT": func(s string) bool {
		if len(s) > 11 {
			return false
		}
		s = strings.Repeat("0", 11-len(s)) + s
		return mod11(soma(s, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2)) == dig(s, 10)
	},
	"MS": simples9(mod11Dez),
	"MG": func(s string) bool {
		if len(s) != 13 {
			return false
		}
		// 1º dígito: "0" depois do município, pesos 1 e 2 alternados, soma dos algarismos.
		x := s[:3] + "0" + s[3:11]
		t := 0
		for i := 0; i < 12; i++ {
			p := dig(x, i) * (1 + i%2)
			t += p/10 + p%10
		}
		d1 := (10 - t%10) % 10
		return d1 == dig(s, 11) && mod11(soma(s, 3, 2, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2)) == dig(s, 12)
	},
	"PA": simples9(mod11),
	"PB": simples9(mod11Dez),
	"PR": func(s string) bool {
		return len(s) == 10 &&
			mod11(soma(s, 3, 2, 7, 6, 5, 4, 3, 2)) == dig(s, 8) &&
			mod11(soma(s, 4, 3, 2, 7, 6, 5, 4, 3, 2)) == dig(s, 9)
	},
	"PE": func(s string) bool {
		switch len(s) {
		case 9:
			return mod11(soma(s, decrescente(8)...)) == dig(s, 7) && mod11(soma(s, decrescente(9)...)) == dig(s, 8)
		case 14:
			x := 11 - soma(s, 5, 4, 3, 2, 1, 9, 8, 7, 6, 5, 4, 3, 2)%11
			if x > 9 {
				x -= 10
			}
			return x == dig(s, 13)
		}
		return false
	},
	"PI": simples9(mod11Dez),
	"RJ": func(s string) bool {
		return len(s) == 8 && mod11(soma(s, 2, 7, 6, 5, 4, 3, 2)) == dig(s, 7)
	},
	"RN": func(s string) bool {
		if (len(s) != 9 && len(s) != 10) || !strings.HasPrefix(s, "20") {
			return false
		}
		n := len(s)
		return (soma(s, decrescente(n)...)*10)%11%10 == dig(s, n-1)
	},
	"RS": func(s string) bool {
		return len(s) == 10 && mod11Dez(soma(s, 2, 9, 8, 7, 6, 5, 4, 3, 2)) == dig(s, 9)
	},
	"RO": func(s string) bool {
		var x int
		switch len(s) {
		case 14:
			x = 11 - soma(s, 6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2)%11
		case 9: // anterior a 2000: município (3) + empresa (5) + dígito
			x = 11 - soma(s[3:], 6, 5, 4, 3, 2)%11
		default:
			return false
		}
		if x >= 10 {
			x -= 10
		}
		return x == dig(s, len(s)-1)
	},
	"RR": func(s string) bool {
		return len(s) == 9 && strings.HasPrefix(s, "24") && soma(s, 1, 2, 3, 4, 5, 6, 7, 8)%9 == dig(s, 8)
	},
	"SC": simples9(mod11),
	"SP": func(s string) bool {
		return len(s) == 12 &&
			soma(s, 1, 3, 4, 5, 6, 7, 8, 10)%11%10 == dig(s, 8) &&
			soma(s, 3, 2, 10, 9, 8, 7, 6, 5, 4, 3, 2)%11%10 == dig(s, 11)
	},
	"SE": simples9(mod11Dez),
	"TO": func(s string) bool {
		if len(s) == 11 { // antigo: o 3º e 4º dígitos (tipo) ficam fora da conta
			if t := s[2:4]; t != "01" && t != "02" && t != "03" && t != "99" {
				return false
			}
			s = s[:2] + s[4:]
		}
		return simples9(mod11)(s)
	},
}

func atoi(s string) int {
	n := 0
	for i := range s {
		n = n*10 + dig(s, i)
	}
	return n
}
