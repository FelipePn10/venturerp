package entity

import "testing"

func i64(v int64) *int64   { return &v }
func str(v string) *string { return &v }

// TestSetupDaTransicaoPreferRegraMaisEspecifica: a regra de item vence a de
// família, que vence a genérica — como qualquer tabela de exceções.
func TestSetupDaTransicaoPreferRegraMaisEspecifica(t *testing.T) {
	regras := []SetupTransicao{
		{WorkCenterID: 1, SetupMinutes: 60, IsActive: true, ToFamily: str("PINTURA")},
		{WorkCenterID: 1, SetupMinutes: 15, IsActive: true, FromItemCode: i64(10), ToItemCode: i64(20)},
		{WorkCenterID: 1, SetupMinutes: 40, IsActive: true, FromFamily: str("PINTURA"), ToFamily: str("PINTURA")},
	}
	got, achou := SetupDaTransicao(regras, ContextoDeSetup{
		DeItem: i64(10), ParaItem: 20, DeFamilia: "PINTURA", ParaFam: "PINTURA",
	})
	if !achou {
		t.Fatal("nenhuma regra casou, esperado a de item")
	}
	if got != 15 {
		t.Fatalf("setup = %.0f min, esperado 15 (regra de item é a mais específica)", got)
	}
}

// TestSetupDaTransicaoUsaFamiliaQuandoNaoHaItem: o caso comum — trocar de cor
// dentro da mesma família custa menos que entrar nela vindo de fora.
func TestSetupDaTransicaoUsaFamiliaQuandoNaoHaItem(t *testing.T) {
	regras := []SetupTransicao{
		{WorkCenterID: 1, SetupMinutes: 10, IsActive: true, FromFamily: str("BRANCO"), ToFamily: str("BRANCO")},
		{WorkCenterID: 1, SetupMinutes: 45, IsActive: true, FromFamily: str("PRETO"), ToFamily: str("BRANCO")},
	}
	mesmo, _ := SetupDaTransicao(regras, ContextoDeSetup{ParaItem: 7, DeFamilia: "BRANCO", ParaFam: "BRANCO"})
	if mesmo != 10 {
		t.Fatalf("mesma cor = %.0f min, esperado 10", mesmo)
	}
	troca, _ := SetupDaTransicao(regras, ContextoDeSetup{ParaItem: 7, DeFamilia: "PRETO", ParaFam: "BRANCO"})
	if troca != 45 {
		t.Fatalf("troca de cor = %.0f min, esperado 45", troca)
	}
}

// TestSetupDaTransicaoSemRegraNaoInventa: sem regra o chamador mantém o setup
// fixo da operação. A matriz refina, não substitui.
func TestSetupDaTransicaoSemRegraNaoInventa(t *testing.T) {
	if _, achou := SetupDaTransicao(nil, ContextoDeSetup{ParaItem: 1}); achou {
		t.Fatal("sem regras não deve achar transição")
	}
	regras := []SetupTransicao{{WorkCenterID: 1, SetupMinutes: 99, IsActive: false, ToItemCode: i64(1)}}
	if _, achou := SetupDaTransicao(regras, ContextoDeSetup{ParaItem: 1}); achou {
		t.Fatal("regra inativa não deve valer")
	}
}

// TestSetupDaTransicaoPrimeiraOrdemDoDia: sem item anterior, a regra que só
// exige o destino ainda vale (é o setup de entrada na máquina).
func TestSetupDaTransicaoPrimeiraOrdemDoDia(t *testing.T) {
	regras := []SetupTransicao{{WorkCenterID: 1, SetupMinutes: 30, IsActive: true, ToItemCode: i64(5)}}
	got, achou := SetupDaTransicao(regras, ContextoDeSetup{DeItem: nil, ParaItem: 5})
	if !achou || got != 30 {
		t.Fatalf("setup de entrada = %.0f (achou=%v), esperado 30", got, achou)
	}
}
