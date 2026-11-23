package retorno

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

// linha monta um registro de 240 posições com os campos nas posições 1-based.
func linha(campos map[int]string) string {
	b := []byte(strings.Repeat(" ", 240))
	for pos, v := range campos {
		copy(b[pos-1:], v)
	}
	return string(b)
}

func TestLerRetornoCNAB240(t *testing.T) {
	arq := strings.Join([]string{
		linha(map[int]string{1: "341", 8: "0"}), // header
		linha(map[int]string{8: "3", 14: "T", 16: "06", 38: "00000000000000012345", 82: "000000000036050"}),
		linha(map[int]string{8: "3", 14: "U", 16: "06", 18: "000000000000150", 33: "000000000000000", 78: "000000000036200", 138: "20112026", 146: "21112026"}),
		linha(map[int]string{8: "3", 14: "T", 16: "02", 38: "00000000000000099999"}), // entrada confirmada
		linha(map[int]string{8: "3", 14: "U", 16: "02"}),
	}, "\r\n")
	liq, outras, err := Ler(arq)
	if err != nil {
		t.Fatal(err)
	}
	if len(liq) != 1 || len(outras) != 1 {
		t.Fatalf("liquidações %+v, outras %+v", liq, outras)
	}
	l := liq[0]
	if l.NossoNumero != "12345" || !l.ValorTitulo.Equal(decimal.RequireFromString("360.50")) || !l.ValorPago.Equal(decimal.RequireFromString("362")) ||
		!l.Juros.Equal(decimal.RequireFromString("1.5")) || l.Credito.Format("2006-01-02") != "2026-11-21" {
		t.Fatalf("liquidação: %+v", l)
	}
	if _, _, err := Ler("curta demais"); err == nil {
		t.Fatal("linha com menos de 240 posições deveria ser recusada")
	}
}
