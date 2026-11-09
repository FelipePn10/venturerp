package entity

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestNormalizarTexto(t *testing.T) {
	casos := map[string]string{"Pç.": "PC", "m²": "M2", " Kg ": "KG", "UNID.": "UNID", "Mt³": "MT3", "caixa": "CAIXA", "": ""}
	for in, want := range casos {
		if got := NormalizarTexto(in); got != want {
			t.Errorf("NormalizarTexto(%q) = %q, queria %q", in, got, want)
		}
	}
}

func TestNewUnit(t *testing.T) {
	u, err := NewUnit(" barra ", "Barra de 6 m", DimContagem, "", 0)
	if err != nil || u.Code != "BARRA" || u.FiscalCode != "BARRA" {
		t.Fatalf("unidade válida recusada: %v %+v", err, u)
	}
	if _, err := NewUnit("TONELADAS", "Toneladas", DimMassa, "", 4); err == nil {
		t.Fatal("código com mais de 6 caracteres exige código fiscal")
	}
	if u, err := NewUnit("TONELADAS", "Toneladas", DimMassa, "t", 4); err != nil || u.FiscalCode != "T" {
		t.Fatalf("com código fiscal deveria aceitar: %v", err)
	}
	if _, err := NewUnit("M 2", "x", DimArea, "", 4); err == nil {
		t.Fatal("código com espaço é inválido")
	}
	if _, err := NewUnit("X", "x", "PESO", "", 4); err == nil {
		t.Fatal("dimensão desconhecida")
	}
}

func TestNewAlias(t *testing.T) {
	a, err := NewAlias("Pç.", "pc", nil)
	if err == nil {
		t.Fatalf("PÇ normaliza para PC, que já é o código: não precisa de sinônimo (%+v)", a)
	}
	a, err = NewAlias("Und", "UN", nil)
	if err != nil || a.Alias != "UND" {
		t.Fatalf("sinônimo válido recusado: %v", err)
	}
}

func TestStandardConversionMesmaDimensao(t *testing.T) {
	um := decimal.NewFromInt(1)
	if _, err := NewStandardConversion("KG", "UN", um, decimal.NewFromInt(2), DimMassa, DimContagem); err == nil {
		t.Fatal("padrão entre massa e contagem depende do item: deve recusar")
	}
	c, err := NewStandardConversion("m", "mm", um, decimal.NewFromInt(1000), DimComprimento, DimComprimento)
	if err != nil || !c.Factor.Equal(decimal.NewFromInt(1000)) {
		t.Fatalf("M→MM é padrão válida: %v", err)
	}
}
