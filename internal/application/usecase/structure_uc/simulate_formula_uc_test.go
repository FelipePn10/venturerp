package structure_uc

import (
	"context"
	"testing"
)

// A simulação existe para o usuário conferir o número antes de gravar: uma
// fórmula errada só apareceria depois, como necessidade errada no MRP.
func TestSimulacaoCalculaComArredondamentoEPerda(t *testing.T) {
	uc := &SimulateQuantityFormulaUseCase{}
	out, err := uc.Execute(context.Background(), SimulateFormulaInput{
		Formula:        "2*(COMPRIMENTO/1000)+2*(PROFUNDIDADE/1000)",
		Rounding:       "UP",
		Scale:          2,
		LossPercentage: 10,
		SetupLoss:      0.5,
		Variables:      map[string]float64{"COMPRIMENTO": 1500, "PROFUNDIDADE": 600},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Valid {
		t.Fatalf("simulação inválida: %s", out.Error)
	}
	// 2*1,5 + 2*0,6 = 4,2
	if out.RawResult < 4.19 || out.RawResult > 4.21 {
		t.Fatalf("resultado bruto = %v, esperado 4,2", out.RawResult)
	}
	if out.QuantityWithLoss <= out.RoundedResult {
		t.Fatalf("a perda não foi aplicada: %v <= %v", out.QuantityWithLoss, out.RoundedResult)
	}
	if out.QuantityPerOrder <= out.QuantityWithLoss {
		t.Fatalf("a perda de preparação não entrou: %v", out.QuantityPerOrder)
	}
}

// Faltando resposta, a simulação diz o que falta em vez de calcular errado.
func TestSimulacaoApontaVariavelSemResposta(t *testing.T) {
	uc := &SimulateQuantityFormulaUseCase{}
	out, err := uc.Execute(context.Background(), SimulateFormulaInput{
		Formula:   "LARGURA*ALTURA",
		Variables: map[string]float64{"LARGURA": 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Valid {
		t.Fatal("calculou sem ter todas as respostas")
	}
	if len(out.MissingVariables) != 1 || out.MissingVariables[0] != "ALTURA" {
		t.Fatalf("não apontou a variável faltante: %+v", out.MissingVariables)
	}
}

// Fórmula malformada devolve o motivo, não um número silenciosamente errado.
func TestSimulacaoRecusaFormulaMalformada(t *testing.T) {
	uc := &SimulateQuantityFormulaUseCase{}
	out, err := uc.Execute(context.Background(), SimulateFormulaInput{Formula: "2*(LARGURA"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Valid || out.Error == "" {
		t.Fatalf("aceitou fórmula malformada: %+v", out)
	}
}

// Sem fórmula não há o que simular.
func TestSimulacaoExigeFormula(t *testing.T) {
	uc := &SimulateQuantityFormulaUseCase{}
	if _, err := uc.Execute(context.Background(), SimulateFormulaInput{}); err == nil {
		t.Fatal("aceitou simulação sem fórmula")
	}
}
