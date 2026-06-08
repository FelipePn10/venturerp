package cnab

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateRemessa240(t *testing.T) {
	cfg := RemessaConfig{
		BankCode:    "341",
		BankName:    "Banco Itau",
		CompanyName: "Panosso ERP LTDA",
		CompanyCNPJ: "12345678000199",
		Agencia:     "1234",
		AgenciaDV:   "5",
		Conta:       "67890",
		ContaDV:     "1",
		Convenio:    "123456",
		SequenceNSA: 1,
	}
	titulos := []Titulo{
		{
			NossoNumero: "123", NumeroDoc: "NF-1", Vencimento: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			Valor: 1500.50, Emissao: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			SacadoNome: "Cliente Teste", SacadoTipo: 2, SacadoDoc: "99888777000166",
			SacadoCidade: "Curitiba", SacadoUF: "PR", SacadoCEP: "80000000",
		},
		{
			NossoNumero: "124", NumeroDoc: "NF-2", Vencimento: time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
			Valor: 250.00, SacadoNome: "Outro Cliente", SacadoTipo: 1, SacadoDoc: "11122233344",
			SacadoUF: "SP",
		},
	}

	out, err := GenerateRemessa240(cfg, titulos)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimRight(out, "\r\n"), "\r\n")
	// header arq + header lote + 2 segments × 2 títulos + trailer lote + trailer arq = 8
	if len(lines) != 8 {
		t.Fatalf("expected 8 records, got %d", len(lines))
	}
	for i, l := range lines {
		if len(l) != 240 {
			t.Errorf("line %d has width %d, want 240", i, len(l))
		}
	}
	if lines[0][7] != '0' {
		t.Errorf("header arquivo record type = %q, want 0", lines[0][7])
	}
	if lines[len(lines)-1][7] != '9' {
		t.Errorf("trailer arquivo record type = %q, want 9", lines[len(lines)-1][7])
	}
}

func TestBankProfile_PerBankFields(t *testing.T) {
	titulos := []Titulo{{
		NossoNumero: "123", NumeroDoc: "NF-1", Vencimento: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		Valor: 100.00, SacadoNome: "Cliente", SacadoTipo: 2, SacadoDoc: "99888777000166", SacadoUF: "PR",
	}}

	// Bradesco (237): carteira "9", layout arquivo "084".
	cfg := RemessaConfig{BankCode: "237", BankName: "Bradesco", CompanyCNPJ: "12345678000199", Agencia: "1", Conta: "2", Convenio: "3", SequenceNSA: 1}
	out, err := GenerateRemessa240(cfg, titulos)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(out, "\r\n"), "\r\n")

	// Header arquivo: layout do arquivo nas posições 164-166 (índices 163-165).
	if got := lines[0][163:166]; got != "084" {
		t.Errorf("Bradesco layout arquivo = %q, want 084", got)
	}
	// Segmento P é a 3ª linha (índice 2): carteira na posição 058 (índice 57).
	if got := lines[2][57]; got != '9' {
		t.Errorf("Bradesco carteira = %q, want 9", got)
	}
	// Trailers devem conter o código do banco (não mais 000).
	trailerArq := lines[len(lines)-1]
	if got := trailerArq[0:3]; got != "237" {
		t.Errorf("trailer arquivo banco = %q, want 237", got)
	}

	// Explicit override wins over the registry default.
	cfgOver := cfg
	cfgOver.Carteira = "7"
	out2, _ := GenerateRemessa240(cfgOver, titulos)
	lines2 := strings.Split(strings.TrimRight(out2, "\r\n"), "\r\n")
	if got := lines2[2][57]; got != '7' {
		t.Errorf("override carteira = %q, want 7", got)
	}
}

func TestValorStr(t *testing.T) {
	if got := valorStr(1500.50); got != "150050" {
		t.Errorf("valorStr(1500.50) = %q, want 150050", got)
	}
	if got := padNum(valorStr(250), 15); got != "000000000025000" {
		t.Errorf("padNum value = %q", got)
	}
}
