package entity

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func d(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func q(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestConsumirPrevisaoPelaVenda(t *testing.T) {
	// Previsão de 100 em 10/11 e de 100 em 17/11; vendas de 60 em 12/11 e de
	// 70 em 18/11. Para trás, 30 dias: a de 12/11 consome a de 10/11; a de
	// 18/11 consome 70 da de 17/11.
	cons := []Consumivel{{Chave: "A", Data: d("2026-11-10"), Quantidade: q("100")}, {Chave: "A", Data: d("2026-11-17"), Quantidade: q("100")}}
	vendas := []Venda{{Chave: "A", Data: d("2026-11-18"), Quantidade: q("70")}, {Chave: "A", Data: d("2026-11-12"), Quantidade: q("60")}}
	got := Consumir(cons, vendas, 30, 0)
	if !got[0].Equal(q("60")) || !got[1].Equal(q("70")) {
		t.Fatalf("consumo: %v", got)
	}
}

func TestConsumirTransbordaParaOAnteriorENaoPassaDaJanela(t *testing.T) {
	// A venda de 150 em 18/11 consome toda a de 17/11 (100) e 50 da de 10/11;
	// a de 01/09 está fora da janela de 30 dias e fica intacta.
	cons := []Consumivel{
		{Chave: "A", Data: d("2026-09-01"), Quantidade: q("100")},
		{Chave: "A", Data: d("2026-11-10"), Quantidade: q("100")},
		{Chave: "A", Data: d("2026-11-17"), Quantidade: q("100")},
	}
	got := Consumir(cons, []Venda{{Chave: "A", Data: d("2026-11-18"), Quantidade: q("150")}}, 30, 0)
	if !got[0].IsZero() || !got[1].Equal(q("50")) || !got[2].Equal(q("100")) {
		t.Fatalf("consumo: %v", got)
	}
}

func TestConsumirParaAFrenteSoComJanela(t *testing.T) {
	cons := []Consumivel{{Chave: "A", Data: d("2026-11-20"), Quantidade: q("10")}}
	vendas := []Venda{{Chave: "A", Data: d("2026-11-18"), Quantidade: q("4")}}
	if got := Consumir(cons, vendas, 30, 0); !got[0].IsZero() {
		t.Fatalf("sem janela para a frente a venda não consome a demanda posterior: %v", got)
	}
	if got := Consumir(cons, vendas, 30, 5); !got[0].Equal(q("4")) {
		t.Fatalf("com 5 dias para a frente consome: %v", got)
	}
}

func TestConsumirSoOMesmoItemEMascara(t *testing.T) {
	cons := []Consumivel{{Chave: "A|", Data: d("2026-11-10"), Quantidade: q("10")}}
	vendas := []Venda{{Chave: "A|AZUL", Data: d("2026-11-10"), Quantidade: q("10")}, {Chave: "B|", Data: d("2026-11-10"), Quantidade: q("10")}}
	if got := Consumir(cons, vendas, 30, 0); !got[0].IsZero() {
		t.Fatalf("venda de outro item ou máscara não consome: %v", got)
	}
}

func TestVencida(t *testing.T) {
	if Vencida(d("2026-10-01"), d("2026-10-31"), 30) {
		t.Fatal("01/10 + 30 dias = 31/10: ainda pode ser consumida hoje")
	}
	if !Vencida(d("2026-10-01"), d("2026-11-01"), 30) {
		t.Fatal("passou da janela: vencida")
	}
}

func TestProgramar(t *testing.T) {
	e, msg := Programar(q("100"), d("2026-11-02"), 3, 7)
	if msg != "" || len(e) != 3 {
		t.Fatalf("programação: %v %s", e, msg)
	}
	soma := e[0].Quantidade.Add(e[1].Quantidade).Add(e[2].Quantidade)
	if !soma.Equal(q("100")) || !e[0].Quantidade.Equal(q("33.3333")) || !e[2].Quantidade.Equal(q("33.3334")) {
		t.Fatalf("a soma bate com o total, a sobra vai na última: %v", e)
	}
	if !e[2].Data.Equal(d("2026-11-16")) {
		t.Fatalf("datas a cada 7 dias: %v", e[2].Data)
	}
	if _, msg := Programar(q("10"), d("2026-11-02"), 2, 0); msg == "" {
		t.Fatal("mais de uma entrega exige intervalo")
	}
}

func TestValidar(t *testing.T) {
	base := IndependentDemand{ItemCode: 1, Quantity: q("1"), DemandDate: d("2026-11-02"), Motivo: MotivoEstoque, Prioridade: 3}
	if msg := base.Validar(); msg != "" {
		t.Fatal(msg)
	}
	for nome, mut := range map[string]func(*IndependentDemand){
		"quantidade zero": func(x *IndependentDemand) { x.Quantity = decimal.Zero },
		"motivo":          func(x *IndependentDemand) { x.Motivo = "X" },
		"prioridade":      func(x *IndependentDemand) { x.Prioridade = 9 },
		"projeto":         func(x *IndependentDemand) { x.Motivo = MotivoProjeto },
		"sem data":        func(x *IndependentDemand) { x.DemandDate = time.Time{} },
	} {
		x := base
		mut(&x)
		if x.Validar() == "" {
			t.Errorf("%s: deveria recusar", nome)
		}
	}
}

func TestFamiliaDoMotivo(t *testing.T) {
	for _, m := range []string{MotivoEstoque, MotivoReposicao, MotivoAntecipacaoVenda} {
		if !ConsumidaPelaVenda(m) {
			t.Errorf("%s é expectativa de venda", m)
		}
	}
	for _, m := range []string{MotivoAmostra, MotivoFeira, MotivoConsumoInterno, MotivoAssistencia, MotivoProjeto} {
		if ConsumidaPelaVenda(m) {
			t.Errorf("%s tem finalidade própria", m)
		}
	}
}

func TestPosicionar(t *testing.T) {
	hoje := d("2026-11-15")
	mask := ""
	ds := []*IndependentDemand{
		// Estoque de 10 em 10/11, venda de 4 em 12/11: planeja 6 da demanda.
		{CodeDemand: 1, ItemCode: 7, Mask: &mask, Quantity: q("10"), DemandDate: d("2026-11-10"), Motivo: MotivoEstoque, Situacao: SituacaoAberta},
		// Estoque antigo (01/09) sem venda: vencido, não planeja.
		{CodeDemand: 2, ItemCode: 8, Quantity: q("5"), DemandDate: d("2026-09-01"), Motivo: MotivoEstoque, Situacao: SituacaoAberta},
		// Amostra de 3 com 1 atendida: planeja 2 e a venda não a consome.
		{CodeDemand: 3, ItemCode: 7, Quantity: q("3"), DemandDate: d("2026-11-10"), Motivo: MotivoAmostra, Situacao: SituacaoAberta, Atendida: q("1")},
		// Encerrada: não planeja.
		{CodeDemand: 4, ItemCode: 7, Quantity: q("9"), DemandDate: d("2026-11-10"), Motivo: MotivoEstoque, Situacao: SituacaoEncerrada},
	}
	// Previsão de 20 em 14/11 do item 7: a venda de 4 em 12/11 consome a
	// demanda de 10/11 (para trás); a previsão (posterior) fica inteira.
	prev := []Consumivel{{Chave: ChaveItem(7, ""), Data: d("2026-11-14"), Quantidade: q("20")}}
	vendas := []Venda{{Chave: ChaveItem(7, ""), Data: d("2026-11-12"), Quantidade: q("4")}}
	pos, resto := Posicionar(ds, prev, vendas, 30, 0, hoje)
	if !pos[0].APlanejar.Equal(q("6")) || pos[0].Situacao != PosicaoParcial || !pos[0].Consumida.Equal(q("4")) {
		t.Fatalf("estoque consumido em parte: %+v", pos[0])
	}
	if !pos[1].Vencida || pos[1].Situacao != PosicaoVencida || !pos[1].APlanejar.IsZero() {
		t.Fatalf("estoque vencido: %+v", pos[1])
	}
	if !pos[2].APlanejar.Equal(q("2")) || pos[2].Situacao != PosicaoParcial || !pos[2].Consumida.IsZero() {
		t.Fatalf("amostra: %+v", pos[2])
	}
	if !pos[3].APlanejar.IsZero() || pos[3].Situacao != SituacaoEncerrada {
		t.Fatalf("encerrada: %+v", pos[3])
	}
	if !resto[0].Equal(q("20")) {
		t.Fatalf("previsão posterior à venda fica inteira: %v", resto)
	}
	// Com a venda de 15: consome os 10 da demanda e 5... não da previsão
	// posterior (sem janela para a frente). A demanda fica CONSUMIDA.
	pos, _ = Posicionar(ds, prev, []Venda{{Chave: ChaveItem(7, ""), Data: d("2026-11-12"), Quantidade: q("15")}}, 30, 0, hoje)
	if pos[0].Situacao != PosicaoConsumida || !pos[0].APlanejar.IsZero() {
		t.Fatalf("demanda toda consumida: %+v", pos[0])
	}
}

func TestCobertaESemOrdem(t *testing.T) {
	ordens := []OrdemDaDemanda{
		{Status: "OPEN", Quantidade: decimal.NewFromInt(40)},
		{Status: "CANCELLED", Quantidade: decimal.NewFromInt(100)},
		{Status: "RELEASED", Quantidade: decimal.NewFromInt(20)},
	}
	coberta := Coberta(ordens)
	if !coberta.Equal(decimal.NewFromInt(60)) {
		t.Fatalf("a cancelada não cobre: %s", coberta)
	}
	if got := SemOrdem(decimal.NewFromInt(100), coberta); !got.Equal(decimal.NewFromInt(40)) {
		t.Fatalf("sem ordem = a planejar − coberta: %s", got)
	}
	if got := SemOrdem(decimal.NewFromInt(50), coberta); !got.IsZero() {
		t.Fatalf("coberta além do a planejar não fica negativa: %s", got)
	}
}

func TestRastreioDoConsumo(t *testing.T) {
	hoje := d("2026-11-15")
	ds := []*IndependentDemand{
		{CodeDemand: 1, ItemCode: 7, Quantity: q("10"), DemandDate: d("2026-11-10"), Motivo: MotivoEstoque, Situacao: SituacaoAberta},
	}
	prev := []Consumivel{{Chave: ChaveItem(7, ""), Item: 7, Data: d("2026-11-09"), Quantidade: q("20")}}
	l1, l2 := int64(11), int64(12)
	vendas := []Venda{
		{Chave: ChaveItem(7, ""), Data: d("2026-11-12"), Quantidade: q("4"), Linha: &l1, Pedido: 501, Sequencia: 1, Cliente: "ACME"},
		{Chave: ChaveItem(7, ""), Data: d("2026-11-13"), Quantidade: q("30"), Linha: &l2, Pedido: 502, Sequencia: 2},
	}
	pos, pp, sobra := PosicionarComPolitica(ds, prev, vendas, JanelaFixa(30, 0), hoje)
	// 501: 4 da demanda. 502: 6 da demanda, 20 da previsão, sobram 4.
	if len(pos[0].Consumo) != 2 || pos[0].Consumo[0].Venda.Pedido != 501 || !pos[0].Consumo[0].Quantidade.Equal(q("4")) ||
		pos[0].Consumo[1].Venda.Pedido != 502 || !pos[0].Consumo[1].Quantidade.Equal(q("6")) {
		t.Fatalf("rastreio da demanda: %+v", pos[0].Consumo)
	}
	if !pp[0].Consumida.Equal(q("20")) || !pp[0].Resto.IsZero() || len(pp[0].Consumo) != 1 || pp[0].Consumo[0].Venda.Pedido != 502 {
		t.Fatalf("rastreio da previsão: %+v", pp[0])
	}
	if !sobra[0].IsZero() || !sobra[1].Equal(q("4")) {
		t.Fatalf("venda acima da expectativa: %v", sobra)
	}
}

func TestSobEncomenda(t *testing.T) {
	hoje := d("2026-11-15")
	ds := []*IndependentDemand{
		{CodeDemand: 1, ItemCode: 7, Quantity: q("10"), DemandDate: d("2026-11-20"), Motivo: MotivoEstoque, Situacao: SituacaoAberta},
		{CodeDemand: 2, ItemCode: 7, Quantity: q("3"), DemandDate: d("2026-11-20"), Motivo: MotivoAmostra, Situacao: SituacaoAberta},
	}
	prev := []Consumivel{{Chave: ChaveItem(7, ""), Item: 7, Data: d("2026-11-16"), Quantidade: q("20")}}
	vendas := []Venda{{Chave: ChaveItem(7, ""), Data: d("2026-11-21"), Quantidade: q("5")}}
	mto := func(int64) Consumo { return Consumo{Antes: 30, SobEncomenda: true} }
	pos, pp, sobra := PosicionarComPolitica(ds, prev, vendas, mto, hoje)
	if pos[0].Situacao != PosicaoSobEncomenda || !pos[0].APlanejar.IsZero() || !pos[0].Consumida.IsZero() {
		t.Fatalf("expectativa de venda de item sob encomenda: %+v", pos[0])
	}
	if pos[1].Situacao != PosicaoAberta || !pos[1].APlanejar.Equal(q("3")) {
		t.Fatalf("finalidade própria continua valendo: %+v", pos[1])
	}
	if !pp[0].SobEncomenda || !pp[0].Resto.IsZero() || !sobra[0].Equal(q("5")) {
		t.Fatalf("previsão e venda de item sob encomenda: %+v %v", pp[0], sobra)
	}
}
