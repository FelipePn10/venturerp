package entity

import (
	"math"
	"testing"
)

func etapa(id int64, seq int16, refugo, run float64) *RouteOperation {
	return &RouteOperation{
		ID: id, Sequence: seq, EffectiveScrap: refugo,
		EffTime: OperationTime{Run: run, RunBaseQty: 1, CrewSize: 1},
	}
}

func perto(t *testing.T, nome string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("%s: %.6f, esperado %.6f", nome, got, want)
	}
}

// O caso que o planejador vive: três etapas com refugo, 100 peças boas no fim.
// Solta-se mais do que se entrega, e cada etapa processa uma quantidade
// diferente.
func TestCascataDeRefugoEmRoteiroLinear(t *testing.T) {
	ops := []*RouteOperation{
		etapa(1, 10, 3, 1), // corte refuga 3%
		etapa(2, 20, 1, 1), // solda refuga 1%
		etapa(3, 30, 2, 1), // pintura refuga 2%
	}
	entra := QuantidadePorOperacao(ops, nil, 100)

	// De trás para frente: 100 / 0,98 → / 0,99 → / 0,97
	perto(t, "entra na pintura", entra[3], 100/0.98)
	perto(t, "entra na solda", entra[2], 100/0.98/0.99)
	perto(t, "entra no corte", entra[1], 100/0.98/0.99/0.97)

	soltar := QuantidadeASoltar(ops, nil, 100)
	perto(t, "quantidade a soltar", soltar, 100/0.98/0.99/0.97)
	if soltar <= 100 {
		t.Fatalf("com refugo é preciso soltar MAIS que a quantidade boa; soltou %.4f", soltar)
	}
}

// Sem refugo nada muda: toda etapa processa a quantidade do pedido. É o que
// garante que roteiros antigos continuam se comportando igual.
func TestSemRefugoTodasAsEtapasProcessamAMesmaQuantidade(t *testing.T) {
	ops := []*RouteOperation{etapa(1, 10, 0, 1), etapa(2, 20, 0, 1)}
	entra := QuantidadePorOperacao(ops, nil, 250)
	perto(t, "etapa 10", entra[1], 250)
	perto(t, "etapa 20", entra[2], 250)
	perto(t, "a soltar", QuantidadeASoltar(ops, nil, 250), 250)
}

// Numa rede que converge, o predecessor precisa alimentar o sucessor mais
// exigente — não a soma, porque é a mesma peça seguindo caminhos alternativos
// de controle, e não metade para cada lado.
func TestRedeQueConvergeUsaOSucessorMaisExigente(t *testing.T) {
	ops := []*RouteOperation{
		etapa(1, 10, 0, 1),
		etapa(2, 20, 10, 1), // este sucessor refuga muito
		etapa(3, 30, 1, 1),
		etapa(4, 40, 0, 1),
	}
	edges := []*NetworkEdge{
		{PredecessorID: 1, SuccessorID: 2},
		{PredecessorID: 1, SuccessorID: 3},
		{PredecessorID: 2, SuccessorID: 4},
		{PredecessorID: 3, SuccessorID: 4},
	}
	entra := QuantidadePorOperacao(ops, edges, 100)
	perto(t, "entra na etapa 20", entra[2], 100/0.90)
	perto(t, "entra na etapa 30", entra[3], 100/0.99)
	// A etapa 10 tem de alimentar o ramo que mais pede.
	perto(t, "entra na etapa 10", entra[1], 100/0.90)
}

// O lead time cresce com o refugo: as primeiras operações processam mais peças,
// então demoram mais. Antes o CPM usava a quantidade final em todas as etapas e
// subestimava exatamente o começo do roteiro.
func TestLeadTimeCresceComORefugo(t *testing.T) {
	semRefugo := []*RouteOperation{etapa(1, 10, 0, 0.1), etapa(2, 20, 0, 0.1)}
	comRefugo := []*RouteOperation{etapa(1, 10, 20, 0.1), etapa(2, 20, 20, 0.1)}

	a := CriticalPath(semRefugo, nil, 100).TotalHours
	b := CriticalPath(comRefugo, nil, 100).TotalHours
	if b <= a {
		t.Fatalf("o refugo tem de aumentar o lead time: sem=%.4f h, com=%.4f h", a, b)
	}
	// Etapa 20 processa 125 (=100/0,8) e a 10 processa 156,25 (=125/0,8).
	// Um ciclo parcial ocupa um ciclo inteiro — não se roda um quarto de peça —
	// então 156,25 viram 157 ciclos, a mesma regra do tempo de máquina.
	perto(t, "lead time com refugo", b, 0.1*157+0.1*125)
}

// Refugo herdado: a etapa sem valor próprio usa o da operação de biblioteca.
func TestRefugoDaEtapaSobrepoeODaBiblioteca(t *testing.T) {
	daEtapa := 7.5
	comOverride := &RouteOperation{ScrapPct: &daEtapa}
	semOverride := &RouteOperation{}

	perto(t, "com override", comOverride.EffectiveScrapPct(2), 7.5)
	perto(t, "sem override", semOverride.EffectiveScrapPct(2), 2)
}

// Roteiro vazio e quantidade inválida não podem explodir nem devolver zero:
// zero faria a ordem nascer sem nada a produzir.
func TestBordasNaoDevolvemZero(t *testing.T) {
	if got := QuantidadeASoltar(nil, nil, 100); got != 100 {
		t.Errorf("roteiro sem etapas deve devolver a própria quantidade; devolveu %.2f", got)
	}
	ops := []*RouteOperation{etapa(1, 10, 50, 1)}
	if got := QuantidadeASoltar(ops, nil, 0); got <= 0 {
		t.Errorf("quantidade não positiva deve cair no mínimo de 1 peça; devolveu %.2f", got)
	}
}

// O prazo do terceiro entra no resultado como DIAS CORRIDOS, separado das horas
// úteis. O caso da bucha: torneia aqui, cementa fora, retifica aqui.
func TestPrazoDeTerceiroVemSeparadoDasHoras(t *testing.T) {
	interna1 := etapa(1, 10, 0, 1)
	terceiro := etapa(2, 20, 0, 0) // no terceiro nenhum recurso nosso trabalha
	terceiro.EffectiveSubcontractDays = 9
	interna2 := etapa(3, 30, 0, 1)

	r := CriticalPath([]*RouteOperation{interna1, terceiro, interna2}, nil, 10)

	perto(t, "horas internas", r.TotalHours, 20) // 10 peças × 1 h em duas etapas
	if r.SubcontractDays != 9 {
		t.Fatalf("prazo do terceiro: %d dias, esperado 9", r.SubcontractDays)
	}
}

// Terceiro num ramo paralelo mais curto não atrasa a entrega, então não entra
// na conta — só o que está no caminho crítico conta.
func TestTerceiroForaDoCaminhoCriticoNaoContaPrazo(t *testing.T) {
	inicio := etapa(1, 10, 0, 1)
	ramoLongo := etapa(2, 20, 0, 50) // domina o caminho crítico
	ramoCurto := etapa(3, 30, 0, 0)
	ramoCurto.EffectiveSubcontractDays = 5
	fim := etapa(4, 40, 0, 1)

	edges := []*NetworkEdge{
		{PredecessorID: 1, SuccessorID: 2}, {PredecessorID: 1, SuccessorID: 3},
		{PredecessorID: 2, SuccessorID: 4}, {PredecessorID: 3, SuccessorID: 4},
	}
	r := CriticalPath([]*RouteOperation{inicio, ramoLongo, ramoCurto, fim}, edges, 1)
	if r.SubcontractDays != 0 {
		t.Fatalf("o ramo curto não é o gargalo; prazo contado: %d", r.SubcontractDays)
	}
}
