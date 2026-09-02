package formula

import "testing"

func TestValidate_AcceptsConfiguratorExpressions(t *testing.T) {
	valid := []string{
		"2*(COMPRIMENTO/1000)+2*(PROFUNDIDADE/1000)",
		"ALTURA*LARGURA",
		"-1+ALTURA",
		"3",
		"2(ALTURA)", // justaposição vira multiplicação implícita
	}
	for _, expr := range valid {
		if err := Validate(expr); err != nil {
			t.Errorf("expressão válida rejeitada %q: %v", expr, err)
		}
	}
}

func TestValidate_RejectsMalformedExpressions(t *testing.T) {
	invalid := []string{
		"2*(COMPRIMENTO/1000",
		"2**3",
		"comprimento*2", // variáveis são maiúsculas
		"2 +",
		"@ALTURA",
	}
	for _, expr := range invalid {
		if err := Validate(expr); err == nil {
			t.Errorf("expressão malformada aceita: %q", expr)
		}
	}
}

func TestVariables_DeduplicatesAndPreservesOrder(t *testing.T) {
	got := Variables("LARGURA+ALTURA*LARGURA")
	want := []string{"LARGURA", "ALTURA"}
	if len(got) != len(want) {
		t.Fatalf("variáveis = %v, quer %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("variáveis = %v, quer %v", got, want)
		}
	}
}

func TestVariables_MalformedReturnsNil(t *testing.T) {
	if got := Variables("2 * @"); got != nil {
		t.Fatalf("variáveis de expressão inválida = %v, quer nil", got)
	}
}

func TestEvaluate_DivisionByZeroIsAnError(t *testing.T) {
	if _, err := Evaluate("ALTURA/0", map[string]float64{"ALTURA": 2}); err == nil {
		t.Fatal("divisão por zero aceita")
	}
}
