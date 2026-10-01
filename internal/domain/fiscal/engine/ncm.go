package engine

import (
	"sort"
	"strings"
)

// NormalizarNCM devolve o NCM apenas com dígitos, descartando ponto, espaço e
// qualquer outra máscara.
//
// Por que isto existe: a tabela tributária e a classificação fiscal do item são
// dois cadastros independentes, e nada obrigava os dois a usarem a mesma máscara.
// O motor casa o NCM do item com a tabela por string exata, e quando não acha
// NÃO reclama: cai na alíquota padrão (IPI 0 / CST 50 e PIS/COFINS do cenário).
// Ou seja, "8466.20.90" no item e "84662090" na tabela produziam uma nota com
// imposto zerado, sem erro em lugar nenhum — o pior tipo de defeito fiscal.
//
// Normalizar para dígitos também é o que a SEFAZ exige: o campo <NCM> do XML da
// NF-e tem 8 dígitos e é recusado com máscara.
func NormalizarNCM(ncm string) string {
	var b strings.Builder
	b.Grow(len(ncm))
	for _, r := range ncm {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// NormalizarTabelaNCM reindexa a tabela tributária por NCM normalizado, para que
// o motor ache a linha independentemente da máscara com que ela foi cadastrada.
//
// Quando duas linhas colapsam na mesma chave (a mesma classificação cadastrada
// com e sem máscara), a primeira em ordem de chave original vence — determinístico,
// e a migração 000375 elimina a duplicidade na origem.
func NormalizarTabelaNCM(tabela map[string]*NcmTaxConfig) map[string]*NcmTaxConfig {
	if tabela == nil {
		return nil
	}
	fora := make(map[string]*NcmTaxConfig, len(tabela))
	chaves := make([]string, 0, len(tabela))
	for k := range tabela {
		chaves = append(chaves, k)
	}
	sort.Strings(chaves)
	for _, k := range chaves {
		n := NormalizarNCM(k)
		if n == "" {
			continue
		}
		if _, repetido := fora[n]; repetido {
			continue
		}
		fora[n] = tabela[k]
	}
	return fora
}
