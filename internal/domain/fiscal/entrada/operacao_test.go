package entrada

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

func sp(s string) *string { return &s }

func TestCFOPEntrada(t *testing.T) {
	casos := []struct {
		forn, natureza, ufForn, ufEmp, want string
	}{
		{"5101", "", "PR", "PR", "1101"},
		{"6102", "", "SP", "PR", "2102"},
		{"6405", "", "SP", "PR", "2403"}, // ST
		{"5405", "", "PR", "PR", "1403"},
		{"7101", "", "EX", "PR", "3101"},
		{"6101", "1556", "SP", "PR", "2556"}, // natureza do tipo de operação, ajustada à UF
		{"5102", "2.551", "PR", "PR", "1551"},
		{"6101", "1556", "EX", "PR", "3556"},
		{"6101", "", "PR", "PR", "1101"}, // 6xxx entre empresas do mesmo estado: vale a UF
		{"6101", "", "", "PR", "2101"},   // sem a UF do fornecedor, vale o CFOP dele
		{"1202", "", "PR", "PR", "1202"}, // já é de entrada
		{"abc", "", "PR", "PR", ""},
	}
	for _, c := range casos {
		if got := CFOPEntrada(c.forn, c.natureza, c.ufForn, c.ufEmp); got != c.want {
			t.Errorf("CFOPEntrada(%q,%q,%q,%q) = %q, esperado %q", c.forn, c.natureza, c.ufForn, c.ufEmp, got, c.want)
		}
	}
}

func TestCustoAquisicao_SubtraiSoOsCreditos(t *testing.T) {
	it := &entity.FiscalEntryItem{
		ValorContabil: d("50000"), ValorICMS: 5820, ValorIPI: 1500, ValorPIS: 792, ValorCOFINS: 3648,
		GeraCreditoICMS: true, GeraCreditoIPI: true, GeraCreditoPIS: true, GeraCreditoCOFINS: true,
	}
	if got := CustoAquisicao(it); !got.Equal(d("38240")) {
		t.Fatalf("custo com todos os créditos = %s, esperado 38240", got)
	}
	// Uso e consumo (EPI): não credita nada, o custo é o valor contábil.
	it.GeraCreditoICMS, it.GeraCreditoIPI, it.GeraCreditoPIS, it.GeraCreditoCOFINS = false, false, false, false
	if got := CustoAquisicao(it); !got.Equal(d("50000")) {
		t.Fatalf("custo sem crédito = %s", got)
	}
}

func TestPlanoFinanceiro_RetencoesEItensSemFinanceiro(t *testing.T) {
	code := int64(1)
	e := &entity.FiscalEntry{
		ValorTotal: 10000,
		ValorIRRF:  d("150"), ValorRetPIS: d("65"), ValorRetCOFINS: d("300"), ValorRetCSLL: d("100"),
		Itens: []*entity.FiscalEntryItem{
			{Sequence: 1, ItemCode: &code, PlanoContasID: p64(planoMP), ValorContabil: d("6000"), GeraFinanceiro: true},
			{Sequence: 2, ItemCode: &code, PlanoContasID: p64(planoEPI), ValorContabil: d("4000"), GeraFinanceiro: true},
		},
	}
	base, aPagar, alvos, residuo := PlanoFinanceiro(e)
	if !base.Equal(d("10000")) || !aPagar.Equal(d("9385")) {
		t.Fatalf("base %s a pagar %s", base, aPagar)
	}
	soma := decimal.Zero
	for _, v := range alvos {
		soma = soma.Add(v)
	}
	if !soma.Equal(aPagar) {
		t.Fatalf("alvos somam %s, esperado %s", soma, aPagar)
	}
	somaRes := decimal.Zero
	for _, v := range residuo {
		somaRes = somaRes.Add(v)
	}
	if !somaRes.Equal(e.TotalRetencoes()) {
		t.Fatalf("resíduo para os títulos de retenção soma %s, esperado %s", somaRes, e.TotalRetencoes())
	}

	// Item de bonificação (não gera financeiro) fica fora da base.
	e.Itens[1].GeraFinanceiro = false
	base, _, alvos, _ = PlanoFinanceiro(e)
	if !base.Equal(d("6000")) || len(alvos) != 1 {
		t.Fatalf("base sem a bonificação = %s, alvos %v", base, alvos)
	}
}

func TestConferirEntrada_ParcelasPeloLiquidoEAlmoxarifado(t *testing.T) {
	code := int64(1)
	wh := int64(3)
	forn := int64(77)
	e := &entity.FiscalEntry{
		SupplierCode: &forn, ValorTotal: 10000, ValorIRRF: d("150"),
		Itens: []*entity.FiscalEntryItem{
			{Sequence: 1, ItemCode: &code, PlanoContasID: p64(planoMP), ValorContabil: d("10000"), GeraFinanceiro: true, MovimentaEstoque: true, CfopEntrada: sp("1101")},
		},
		Parcelas: []*entity.FiscalEntryInstallment{{Numero: 1, Valor: d("9850")}},
	}
	_, _, alvos, _ := PlanoFinanceiro(e)
	_ = DistribuirProporcional(e.Parcelas, alvos)
	pend := ConferirEntrada(e)
	if !TemImpedimento(pend) {
		t.Fatal("item que movimenta estoque sem almoxarifado deveria impedir")
	}
	e.Itens[0].WarehouseID = &wh
	if pend := ConferirEntrada(e); TemImpedimento(pend) {
		t.Fatalf("parcela pelo líquido (10000 − 150) deveria fechar: %+v", pend)
	}
	e.Parcelas[0].Valor = d("10000")
	_ = DistribuirProporcional(e.Parcelas, alvos)
	if !TemImpedimento(ConferirEntrada(e)) {
		t.Fatal("parcela pelo bruto, com retenção, deveria impedir")
	}
}

func TestRetencoes_Vencimentos(t *testing.T) {
	e := &entity.FiscalEntry{
		DataEmissao: time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
		ValorIRRF:   d("15"), ValorRetPIS: d("6.5"), ValorISSRet: d("50"),
	}
	rs := Retencoes(e)
	if len(rs) != 3 {
		t.Fatalf("retenções = %+v", rs)
	}
	for _, r := range rs {
		switch r.Tipo {
		case "IRRF", "PIS":
			// 20/10/2026 é terça-feira.
			if r.Vencimento.Format("2006-01-02") != "2026-10-20" {
				t.Errorf("%s vence %s", r.Tipo, r.Vencimento.Format("2006-01-02"))
			}
		case "ISS":
			// 10/10/2026 é sábado: antecipa para sexta, 09/10.
			if r.Vencimento.Format("2006-01-02") != "2026-10-09" {
				t.Errorf("ISS vence %s", r.Vencimento.Format("2006-01-02"))
			}
		}
	}
}

func TestEscalar_FechaExato(t *testing.T) {
	totais := map[entity.ChaveConta]decimal.Decimal{
		{PlanoContasID: 1}: d("33.33"), {PlanoContasID: 2}: d("33.33"), {PlanoContasID: 3}: d("33.34"),
	}
	out := Escalar(totais, d("90.01"))
	soma := decimal.Zero
	for _, v := range out {
		soma = soma.Add(v)
	}
	if !soma.Equal(d("90.01")) {
		t.Fatalf("soma %s", soma)
	}
}

func TestDivergencias(t *testing.T) {
	code := int64(77)
	linha := int64(500)
	uf := "SP"
	e := &entity.FiscalEntry{
		UFEmitente: &uf, ValorProdutos: 1000, ValorICMS: 120, ValorIPI: 50,
		Itens: []*entity.FiscalEntryItem{{
			Sequence: 1, ItemCode: &code, Ncm: sp("72085200"), Quantity: 10, UnitPrice: 100, TotalPrice: 1000,
			BaseICMS: 1000, AliqICMS: 0.07, ValorICMS: 120, CstICMS: sp("00"), Origem: sp("0"),
			BaseIPI: 1000, AliqIPI: 0.05, ValorIPI: 50, CstIPI: sp("50"),
			PurchaseOrderItemCode: &linha,
		}},
	}
	p := ParametrosConferencia{
		UFEmpresa:         "PR",
		AliqIPIPorNCM:     map[string]decimal.Decimal{"72085200": d("0.0325")},
		ICMSInterestadual: map[string]decimal.Decimal{"SPPR": d("0.12")},
		NCMDoItem:         map[int64]string{77: "7208.51.00"},
		LinhasPedido: map[int64]LinhaPedido{500: {
			Codigo: 500, PedidoCodigo: 9, ItemCode: 77, PrecoUnitario: d("90"), SaldoAFaturar: d("8"),
		}},
	}
	tipos := map[string]bool{}
	for _, dv := range Divergencias(e, p) {
		tipos[dv.Tipo] = true
	}
	for _, esperado := range []string{"CALCULO_ICMS", "ALIQUOTA_IPI", "ALIQUOTA_ICMS", "NCM", "PEDIDO_PRECO", "PEDIDO_QUANTIDADE"} {
		if !tipos[esperado] {
			t.Errorf("divergência %s não apontada; achou %v", esperado, tipos)
		}
	}

	// Mercadoria importada (origem 1): a alíquota esperada é 4%.
	e.Itens[0].Origem = sp("1")
	e.Itens[0].AliqICMS, e.Itens[0].ValorICMS = 0.04, 40
	e.ValorICMS = 40
	for _, dv := range Divergencias(e, p) {
		if dv.Tipo == "ALIQUOTA_ICMS" || dv.Tipo == "CALCULO_ICMS" {
			t.Errorf("importado a 4%% não deveria divergir: %+v", dv)
		}
	}

	// Linha de pedido de outro item impede.
	p.LinhasPedido[500] = LinhaPedido{Codigo: 500, PedidoCodigo: 9, ItemCode: 99, PrecoUnitario: d("100"), SaldoAFaturar: d("100")}
	impede := false
	for _, dv := range Divergencias(e, p) {
		if dv.Tipo == "PEDIDO_ITEM" && dv.Nivel == NivelImpede {
			impede = true
		}
	}
	if !impede {
		t.Error("linha de pedido de outro item deveria impedir")
	}

	// Pedido ainda em rascunho: a nota não pode dar entrada contra ele.
	p.LinhasPedido[500] = LinhaPedido{Codigo: 500, PedidoCodigo: 9, ItemCode: 77, PrecoUnitario: d("100"), SaldoAFaturar: d("100"),
		PedidoNaoAprovado: true, SituacaoPedido: "em rascunho"}
	naoAprovado := false
	for _, dv := range Divergencias(e, p) {
		if dv.Tipo == "PEDIDO" && dv.Nivel == NivelImpede && strings.Contains(dv.Mensagem, "não foi aprovado") {
			naoAprovado = true
		}
	}
	if !naoAprovado {
		t.Error("linha de pedido não aprovado deveria impedir a aprovação da nota")
	}
}

func TestConferirEntrada_SemFornecedorImpede(t *testing.T) {
	code, wh, forn := int64(1), int64(3), int64(77)
	e := &entity.FiscalEntry{
		CnpjEmitente: "12345678000190", RazaoSocialEmitente: "ACO LTDA", ValorTotal: 100,
		Itens: []*entity.FiscalEntryItem{
			{Sequence: 1, ItemCode: &code, PlanoContasID: p64(planoMP), ValorContabil: d("100"), GeraFinanceiro: true, MovimentaEstoque: true, WarehouseID: &wh, CfopEntrada: sp("1101")},
		},
		Parcelas: []*entity.FiscalEntryInstallment{{Numero: 1, Valor: d("100")}},
	}
	_, _, alvos, _ := PlanoFinanceiro(e)
	_ = DistribuirProporcional(e.Parcelas, alvos)
	pend := ConferirEntrada(e)
	if !TemImpedimento(pend) || pend[0].Campo != "supplier_code" || !strings.Contains(pend[0].Mensagem, "12345678000190") {
		t.Fatalf("emitente sem cadastro deveria impedir: %+v", pend)
	}
	e.SupplierCode = &forn
	if pend := ConferirEntrada(e); TemImpedimento(pend) {
		t.Fatalf("com fornecedor a nota fecha: %+v", pend)
	}
}
