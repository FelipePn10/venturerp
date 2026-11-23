package service

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestAssistenteDeConversao(t *testing.T) {
	d := decimal.RequireFromString
	casos := []struct {
		nome string
		m    MedidasDaPeca
		peso string
		para string
		fat  string
	}{
		// 3000 × 1200 × 3 mm × 7,85 = 84,78 kg; 3,6 m².
		{"chapa de aço", MedidasDaPeca{Formato: "CHAPA", Material: "ACO", ComprimentoMM: d("3000"), LarguraMM: d("1200"), EspessuraMM: d("3")}, "84.78", "M2", "3.6"},
		// π/4 × 25² × 6000 × 7,85 = 23,120158 kg.
		{"barra redonda 1\" × 6 m", MedidasDaPeca{Formato: "BARRA_REDONDA", Material: "aco", ComprimentoMM: d("6000"), DiametroMM: d("25")}, "23.120158", "MM", "6000"},
		// π × (50,8 − 3) × 3 × 6000 × 7,85 = 21,218757 kg.
		{"tubo redondo", MedidasDaPeca{Formato: "TUBO_REDONDO", Material: "ACO", ComprimentoMM: d("6000"), DiametroMM: d("50.8"), EspessuraMM: d("3")}, "21.218757", "MM", "6000"},
		{"perfil por kg/m", MedidasDaPeca{Formato: "KG_POR_METRO", ComprimentoMM: d("6000"), KgPorMetro: d("4.6")}, "27.6", "MM", "6000"},
	}
	for _, c := range casos {
		r, err := CalcularPeca(c.m)
		if err != nil {
			t.Fatalf("%s: %v", c.nome, err)
		}
		if !r.PesoKg.Round(5).Equal(d(c.peso).Round(5)) {
			t.Errorf("%s: peso %s, quer %s", c.nome, r.PesoKg, c.peso)
		}
		if len(r.Sugestoes) != 2 || r.Sugestoes[0].Para != "KG" || r.Sugestoes[1].Para != c.para || !r.Sugestoes[1].Fator.Equal(d(c.fat)) {
			t.Errorf("%s: sugestões %+v", c.nome, r.Sugestoes)
		}
		if !r.LoteMultiplo.Equal(decimal.NewFromInt(1)) {
			t.Errorf("%s: lote múltiplo %s", c.nome, r.LoteMultiplo)
		}
	}
	if _, err := CalcularPeca(MedidasDaPeca{Formato: "CHAPA", Material: "MADEIRA", ComprimentoMM: d("1")}); err == nil {
		t.Error("material desconhecido sem densidade deveria ser recusado")
	}
	if _, err := CalcularPeca(MedidasDaPeca{Formato: "TUBO_REDONDO", Material: "ACO", ComprimentoMM: d("10"), DiametroMM: d("10"), EspessuraMM: d("5")}); err == nil {
		t.Error("parede maior que o raio deveria ser recusada")
	}
}
