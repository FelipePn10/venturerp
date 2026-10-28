package governanca

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

var semAcento = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a", "é", "e", "ê", "e", "è", "e", "ë", "e",
	"í", "i", "î", "i", "ì", "i", "ï", "i", "ó", "o", "ô", "o", "õ", "o", "ò", "o", "ö", "o",
	"ú", "u", "û", "u", "ù", "u", "ü", "u", "ç", "c", "ñ", "n",
)

// palavras que não distinguem um item de outro
var vazias = map[string]bool{"de": true, "da": true, "do": true, "das": true, "dos": true, "com": true, "para": true, "em": true, "e": true, "x": true}

// Normalizar: minúsculas, sem acento, só letras e números separados por espaço.
func Normalizar(s string) string {
	s = semAcento.Replace(strings.ToLower(s))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// Palavras significativas do texto (sem repetição, sem as vazias).
func Palavras(s string) []string {
	visto := map[string]bool{}
	out := []string{}
	for _, p := range strings.Fields(Normalizar(s)) {
		if vazias[p] || visto[p] || (len(p) < 2 && !unicode.IsDigit(rune(p[0]))) {
			continue
		}
		visto[p] = true
		out = append(out, p)
	}
	return out
}

func levenshtein(a, b []rune) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			c := 1
			if a[i-1] == b[j-1] {
				c = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+c)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

// Semelhanca de dois nomes entre 0 e 1: o maior entre a sobreposição das
// palavras (mesmas palavras em outra ordem) e a proximidade dos textos (erro de
// digitação, abreviação curta).
func Semelhanca(a, b string) float64 {
	na, nb := Normalizar(a), Normalizar(b)
	if na == "" || nb == "" {
		return 0
	}
	if na == nb {
		return 1
	}
	pa, pb := Palavras(a), Palavras(b)
	comum := 0
	conj := map[string]bool{}
	for _, p := range pb {
		conj[p] = true
	}
	for _, p := range pa {
		if conj[p] {
			comum++
		}
	}
	uniao := len(pa) + len(pb) - comum
	jac := 0.0
	if uniao > 0 {
		jac = float64(comum) / float64(uniao)
	}
	ra, rb := []rune(na), []rune(nb)
	maior := math.Max(float64(len(ra)), float64(len(rb)))
	lev := 1 - float64(levenshtein(ra, rb))/maior
	return math.Round(math.Max(jac, lev)*100) / 100
}

// LimiteDuplicidade: a partir desta semelhança o cadastro avisa.
const LimiteDuplicidade = 0.6

// Classificar dá a nota aos candidatos e devolve os parecidos, mais parecidos
// primeiro. Mesmo NCM com nome parecido reforça o aviso.
func Classificar(nome, ncm string, cands []Candidato) []Candidato {
	out := []Candidato{}
	ncm = strings.TrimSpace(ncm)
	for _, c := range cands {
		n := Semelhanca(nome, c.Nome)
		motivo := "nome parecido"
		if ncm != "" && c.NCM == ncm && n >= LimiteDuplicidade-0.15 {
			n = math.Min(1, n+0.15)
			motivo = "nome parecido e mesmo NCM"
		}
		if n >= 0.999 {
			motivo = "mesmo nome"
		}
		if n < LimiteDuplicidade {
			continue
		}
		c.Nota, c.Motivo = math.Round(n*100)/100, motivo
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Nota > out[j].Nota })
	if len(out) > 10 {
		out = out[:10]
	}
	return out
}
