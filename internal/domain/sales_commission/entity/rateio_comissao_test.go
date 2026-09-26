package entity

import (
	"testing"

	"github.com/shopspring/decimal"
)

func rateio(rep int64, papel Papel, pct string, base Base) *Rateio {
	return &Rateio{RepresentativeCode: rep, Role: papel, CommissionPct: decimal.RequireFromString(pct), CommissionBase: base}
}

func TestRateioComDoisRepresentantesEhValido(t *testing.T) {
	linhas := []*Rateio{
		rateio(10, PapelPrincipal, "3", BaseTotalProdutos),
		rateio(20, PapelParceiro, "1.5", BaseTotalProdutos),
	}
	if err := ValidarRateio(linhas); err != nil {
		t.Fatalf("rateio válido recusado: %v", err)
	}
	if p := Principal(linhas); p == nil || p.RepresentativeCode != 10 {
		t.Fatalf("principal errado: %+v", p)
	}
}

func TestRateioExigeUmPrincipal(t *testing.T) {
	sem := []*Rateio{rateio(10, PapelParceiro, "3", BaseTotalProdutos)}
	if err := ValidarRateio(sem); err == nil {
		t.Fatal("rateio sem principal deveria ser recusado")
	}
	dois := []*Rateio{
		rateio(10, PapelPrincipal, "3", BaseTotalProdutos),
		rateio(20, PapelPrincipal, "3", BaseTotalProdutos),
	}
	if err := ValidarRateio(dois); err == nil {
		t.Fatal("dois principais deveriam ser recusados")
	}
}

func TestRateioRecusaRepetidoESomaAcimaDeCem(t *testing.T) {
	repetido := []*Rateio{
		rateio(10, PapelPrincipal, "3", BaseTotalProdutos),
		rateio(10, PapelParceiro, "2", BaseTotalProdutos),
	}
	if err := ValidarRateio(repetido); err == nil {
		t.Fatal("representante repetido deveria ser recusado")
	}
	acima := []*Rateio{
		rateio(10, PapelPrincipal, "60", BaseTotalProdutos),
		rateio(20, PapelParceiro, "50", BaseTotalProdutos),
	}
	if err := ValidarRateio(acima); err == nil {
		t.Fatal("soma acima de 100% deveria ser recusada")
	}
}

func TestRateioRespeitaLimiteDeRepresentantes(t *testing.T) {
	linhas := []*Rateio{rateio(1, PapelPrincipal, "1", BaseTotalProdutos)}
	for i := int64(2); i <= MaxRepresentantesPorDocumento+1; i++ {
		linhas = append(linhas, rateio(i, PapelParceiro, "1", BaseTotalProdutos))
	}
	if err := ValidarRateio(linhas); err == nil {
		t.Fatalf("acima de %d representantes deveria ser recusado", MaxRepresentantesPorDocumento)
	}
}

// A base é o ponto que o usuário erra: comissão sobre o líquido do documento
// (que carrega frete e desconto de capa) dá um valor diferente da comissão
// sobre os produtos, e o documento tem de dizer qual foi usada.
func TestValorRespeitaBaseGravadaNaLinha(t *testing.T) {
	produtos := decimal.RequireFromString("10000")
	liquido := decimal.RequireFromString("10500")

	sobreProdutos := rateio(10, PapelPrincipal, "3", BaseTotalProdutos)
	if got := sobreProdutos.Valor(produtos, liquido); !got.Equal(decimal.RequireFromString("300")) {
		t.Fatalf("comissão sobre produtos = %s, esperado 300", got)
	}
	sobreLiquido := rateio(20, PapelParceiro, "3", BaseTotalLiquido)
	if got := sobreLiquido.Valor(produtos, liquido); !got.Equal(decimal.RequireFromString("315")) {
		t.Fatalf("comissão sobre líquido = %s, esperado 315", got)
	}
}

func TestNormalizarPreencheOsPadroes(t *testing.T) {
	obs := "   "
	r := &Rateio{RepresentativeCode: 10, Notes: &obs}
	r.Normalizar()
	if r.Role != PapelPrincipal || r.CommissionBase != BaseTotalProdutos {
		t.Fatalf("padrões não aplicados: %+v", r)
	}
	if r.Notes != nil {
		t.Fatal("observação em branco deveria virar nula")
	}
	if err := r.Validar(); err != nil {
		t.Fatalf("linha normalizada recusada: %v", err)
	}
}
