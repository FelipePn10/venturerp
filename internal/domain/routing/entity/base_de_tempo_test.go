package entity

import (
	"math"
	"testing"
)

func quaseIgual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// Porta de 600 × 400 mm: 0,24 m², 0,6 m de comprimento, 2 m de perímetro.
func TestFatorDaBaseLeAsMedidasDoItem(t *testing.T) {
	porta := MedidasDaPeca{ComprimentoMM: 600, LarguraMM: 400, Atributos: map[string]string{"FUROS": "8", "METROS_FITA": "1,5"}}
	furos, fita := "furos", "METROS_FITA"
	casos := []struct {
		base     string
		atributo *string
		quer     float64
	}{
		{BasePeca, nil, 1},
		{BaseM2, nil, 0.24},
		{BaseMetroLinear, nil, 0.6},
		{BasePerimetro, nil, 2},
		{BaseAtributo, &furos, 8},
		{BaseAtributo, &fita, 1.5},
	}
	for _, c := range casos {
		got, ok := FatorDaBase(c.base, c.atributo, porta)
		if !ok || !quaseIgual(got, c.quer) {
			t.Fatalf("%s: quer %v, veio %v (ok=%v)", c.base, c.quer, got, ok)
		}
	}
}

// Item sem a medida não é medido como "1": a etapa fica marcada sem medida.
func TestFatorDaBaseSemMedidaNaoChuta(t *testing.T) {
	semMedida := MedidasDaPeca{}
	for _, base := range []string{BaseM2, BaseMetroLinear, BasePerimetro} {
		if _, ok := FatorDaBase(base, nil, semMedida); ok {
			t.Fatalf("%s sem dimensões não pode ser medido", base)
		}
	}
	nome := "FUROS"
	if _, ok := FatorDaBase(BaseAtributo, &nome, semMedida); ok {
		t.Fatal("atributo ausente não pode ser medido")
	}
}

// Fita de borda: 0,5 h por 10 m de perímetro. Porta com 2 m de perímetro, 100
// portas = 200 m = 20 ciclos = 10 h. Por peça seriam 100/10 = 10 ciclos = 5 h.
func TestTempoPorPerimetroEscalaComAMedida(t *testing.T) {
	tempo := OperationTime{Run: 0.5, RunBaseQty: 10, UnitsPerPiece: 2}
	if got := tempo.MachineHours(100); !quaseIgual(got, 10) {
		t.Fatalf("100 portas de 2 m de perímetro: quer 10 h, veio %v", got)
	}
	sem := OperationTime{Run: 0.5, RunBaseQty: 10, SemMedida: true}
	if got := sem.MachineHours(100); got != 0 {
		t.Fatalf("sem medida, nenhum ciclo é inventado: veio %v h", got)
	}
}

// 0,36 m² × 100 = 36,000000000000004 em float: não pode virar 37 ciclos.
func TestCiclosNaoSobemPorRuidoDeFloat(t *testing.T) {
	tempo := OperationTime{Run: 1, RunBaseQty: 1, UnitsPerPiece: 0.6 * 0.6}
	if got := tempo.Batches(100); got != 36 {
		t.Fatalf("quer 36 ciclos, veio %v", got)
	}
}

func TestNormalizaBaseDeTempo(t *testing.T) {
	if b, _, err := NormalizaBaseDeTempo("", nil); err != nil || b != BasePeca {
		t.Fatalf("vazio = por peça; veio %q %v", b, err)
	}
	if _, _, err := NormalizaBaseDeTempo("ATRIBUTO", nil); err == nil {
		t.Fatal("atributo sem nome tem de ser recusado")
	}
	nome := " furos "
	if b, a, err := NormalizaBaseDeTempo("atributo", &nome); err != nil || b != BaseAtributo || *a != "FUROS" {
		t.Fatalf("veio %q %v %v", b, a, err)
	}
	if _, _, err := NormalizaBaseDeTempo("KG", nil); err == nil {
		t.Fatal("base desconhecida tem de ser recusada")
	}
}

// Operação antiga, só com o tempo padrão legado: ele vale como tempo de máquina.
func TestTempoLegadoDaOperacaoValeQuandoNaoHaTempoDeMaquina(t *testing.T) {
	if got := TempoDeMaquinaDaOperacao(0, 0.083); got != 0.083 {
		t.Fatalf("legado vale quando o tempo de máquina é zero; veio %v", got)
	}
	if got := TempoDeMaquinaDaOperacao(0.05, 0.083); got != 0.05 {
		t.Fatalf("com tempo de máquina, ele manda; veio %v", got)
	}
}
