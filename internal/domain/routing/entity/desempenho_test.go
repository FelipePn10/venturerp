package entity

import "testing"

// Cinco OPs, 100 peças no total. Padrão atual: preparação 0,4 h por ordem e
// 0,1 h por peça → 2 + 10 = 12 h. Apontado: 15 h. A produção real foi
// 15 − 2 = 13 h para 100 peças: 0,13 h por peça.
func TestSugereTempoQuandoORealSeAfasta(t *testing.T) {
	padrao := OperationTime{Setup: 0.4, Run: 0.1, RunBaseQty: 1}
	c := CompararTempo(StepActuals{Orders: 5, ActualHours: 15, Quantity: 100}, padrao)
	if c.SuggestedRunHours == nil || *c.SuggestedRunHours != 0.13 {
		t.Fatalf("quer 0,13 h por ciclo; veio %v (%s)", c.SuggestedRunHours, c.SuggestionReason)
	}
	if c.PlannedHours != 12 || c.DeviationPct != 25 {
		t.Fatalf("padrão 12 h, desvio 25%%; veio %v h e %v%%", c.PlannedHours, c.DeviationPct)
	}
}

// Depois de aplicar a sugestão, as MESMAS ordens não podem sugerir de novo:
// antes a comparação era com o planejado gravado nelas, e cada "Aplicar"
// inflava o tempo outra vez.
func TestSugestaoAplicadaNaoSeRepete(t *testing.T) {
	ordens := StepActuals{Orders: 5, ActualHours: 15, Quantity: 100}
	c := CompararTempo(ordens, OperationTime{Setup: 0.4, Run: 0.1, RunBaseQty: 1})
	aplicado := OperationTime{Setup: 0.4, Run: *c.SuggestedRunHours, RunBaseQty: 1}
	if de := CompararTempo(ordens, aplicado); de.SuggestedRunHours != nil {
		t.Fatalf("com o tempo já ajustado não há o que sugerir; veio %v (%s)", *de.SuggestedRunHours, de.SuggestionReason)
	}
}

func TestNaoSugereComPoucaAmostraOuDesvioPequeno(t *testing.T) {
	padrao := OperationTime{Run: 0.1, RunBaseQty: 1}
	if c := CompararTempo(StepActuals{Orders: 2, ActualHours: 20, Quantity: 100}, padrao); c.SuggestedRunHours != nil {
		t.Fatal("duas OPs não bastam para mexer no padrão")
	}
	if c := CompararTempo(StepActuals{Orders: 8, ActualHours: 10.5, Quantity: 100}, padrao); c.SuggestedRunHours != nil {
		t.Fatal("5% de diferença é ruído, não sugestão")
	}
}
