package customer_material_uc

import (
	"strings"
	"testing"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
	"github.com/shopspring/decimal"
)

// dec encurta a montagem dos casos.
func dec(v string) decimal.Decimal { return decimal.RequireFromString(v) }

// remessaDaNF5911 reproduz a remessa que originou a NF-e 5.911 de 02/09/2026 — a
// única nota de beneficiamento emitida de verdade que temos como referência.
func remessaDaNF5911() *entity.Remessa {
	return &entity.Remessa{
		ID: 1, NFeNumber: 16906, CustomerCode: 100, Status: entity.StatusAberta,
		Itens: []*entity.ItemRemessa{
			{
				ID: 10, LineNumber: 1, CustomerItemCode: "11000155",
				Description: "ALUMINIO TUBO AA1050H14 50,8 X 44,45", NCM: "76081000",
				UOM: "KG", QtyReceived: dec("10"), BalanceQty: dec("10"),
				UnitValue: dec("36.93"),
			},
			{
				ID: 11, LineNumber: 2, CustomerItemCode: "10018009",
				Description: "ALUMINIO BARRA CHATA 9,5X38,1 X 6000 MM", NCM: "76041029",
				UOM: "KG", QtyReceived: dec("5"), BalanceQty: dec("5"),
				UnitValue: dec("31.05"),
			},
		},
	}
}

func servicoDaNF5911() ServicoFaturado {
	return ServicoFaturado{
		CodigoItem: "25027532", Descricao: "SERVICO INDUSTRIALIZACAO",
		Unidade: "PC", Quantidade: "1", ValorUnit: "230.00",
	}
}

// TestNotaReproduzANF5911 é o teste que ancora tudo: as mesmas entradas da nota
// real têm de produzir as mesmas linhas, CFOPs, CSTs e valores.
func TestNotaReproduzANF5911(t *testing.T) {
	nota, err := MontarNotaDeRetorno(remessaDaNF5911(), "SP", servicoDaNF5911(),
		[]DevolucaoDeMaterial{
			{RemittanceItemID: 10, Quantidade: "3.848", Tipo: entity.MovimentoRetorno},
			{RemittanceItemID: 11, Quantidade: "1.304", Tipo: entity.MovimentoRetorno},
		})
	if err != nil {
		t.Fatalf("MontarNotaDeRetorno() error = %v", err)
	}
	if len(nota.Linhas) != 3 {
		t.Fatalf("%d linha(s), esperado 3 (serviço + dois materiais)", len(nota.Linhas))
	}

	// Linha 1 — serviço.
	servico := nota.Linhas[0]
	if servico.CFOP != "5124" {
		t.Errorf("CFOP do serviço = %s, esperado 5124", servico.CFOP)
	}
	if servico.CSTICMS != "051" {
		t.Errorf("CST ICMS do serviço = %s, esperado 051 (diferimento)", servico.CSTICMS)
	}
	if servico.NCM != "00000000" {
		t.Errorf("NCM do serviço = %s, esperado 00000000", servico.NCM)
	}
	if got := servico.ValorTotal.String(); got != "230" {
		t.Errorf("valor do serviço = %s, esperado 230", got)
	}

	// Linhas 2 e 3 — material, com os valores exatos da nota real.
	esperado := map[string]struct{ ncm, total string }{
		"11000155": {"76081000", "142.11"},
		"10018009": {"76041029", "40.49"},
	}
	for _, linha := range nota.Linhas[1:] {
		if linha.CFOP != "5902" {
			t.Errorf("CFOP do material = %s, esperado 5902", linha.CFOP)
		}
		if linha.CSTICMS != "050" {
			t.Errorf("CST ICMS do material = %s, esperado 050 (suspensão)", linha.CSTICMS)
		}
		if linha.ItemDaRemessa == nil {
			t.Fatal("a linha de material precisa apontar para o item da remessa")
		}
		ref, ok := esperado[linha.ItemDaRemessa.CustomerItemCode]
		if !ok {
			t.Fatalf("linha inesperada: %s", linha.ItemDaRemessa.CustomerItemCode)
		}
		if linha.NCM != ref.ncm {
			t.Errorf("NCM de %s = %s, esperado %s (o mesmo da nota de entrada)",
				linha.ItemDaRemessa.CustomerItemCode, linha.NCM, ref.ncm)
		}
		if got := linha.ValorTotal.String(); got != ref.total {
			t.Errorf("valor de %s = %s, esperado %s",
				linha.ItemDaRemessa.CustomerItemCode, got, ref.total)
		}
	}

	// Total da nota real: 230,00 de serviço + 182,60 de material = 412,60.
	if got := nota.ValorTotal.String(); got != "412.6" {
		t.Errorf("total da nota = %s, esperado 412.6 (como na NF-e 5.911)", got)
	}
	if got := nota.ValorMaterial.String(); got != "182.6" {
		t.Errorf("total de material = %s, esperado 182.6", got)
	}
}

// TestPISeCOFINSDoServico confere a conta que a contadora mandou por escrito:
// sobre R$ 230,00, PIS R$ 1,50 e COFINS R$ 6,90.
func TestPISeCOFINSDoServico(t *testing.T) {
	nota, err := MontarNotaDeRetorno(remessaDaNF5911(), "SP", servicoDaNF5911(),
		[]DevolucaoDeMaterial{{RemittanceItemID: 10, Quantidade: "1", Tipo: entity.MovimentoRetorno}})
	if err != nil {
		t.Fatalf("MontarNotaDeRetorno() error = %v", err)
	}
	servico := nota.Linhas[0]
	if got := servico.ValorPIS.String(); got != "1.5" {
		t.Errorf("PIS = %s, esperado 1.5 (0,65%% de 230)", got)
	}
	if got := servico.ValorCOFINS.String(); got != "6.9" {
		t.Errorf("COFINS = %s, esperado 6.9 (3%% de 230)", got)
	}
	if servico.CSTPIS != "01" || servico.CSTCOFINS != "01" {
		t.Errorf("CST PIS/COFINS = %s/%s, esperado 01/01", servico.CSTPIS, servico.CSTCOFINS)
	}
	if got := servico.ValorPIS.Add(servico.ValorCOFINS).String(); got != "8.4" {
		t.Errorf("PIS+COFINS = %s, esperado 8.4", got)
	}
	// O material não é tributado: a suspensão cobre o que volta.
	for _, linha := range nota.Linhas[1:] {
		if !linha.ValorPIS.IsZero() || !linha.ValorCOFINS.IsZero() {
			t.Errorf("a linha de material saiu com PIS/COFINS: %s/%s",
				linha.ValorPIS, linha.ValorCOFINS)
		}
		if linha.CSTPIS != "" {
			t.Errorf("a linha de material recebeu CST PIS %q", linha.CSTPIS)
		}
	}
}

// TestSobraESucataSaemPor5903: a contadora confirmou 5903 com CST 050 para as duas.
func TestSobraESucataSaemPor5903(t *testing.T) {
	for _, tipo := range []entity.TipoMovimento{entity.MovimentoSobra, entity.MovimentoSucata} {
		nota, err := MontarNotaDeRetorno(remessaDaNF5911(), "SP", servicoDaNF5911(),
			[]DevolucaoDeMaterial{{RemittanceItemID: 10, Quantidade: "1", Tipo: tipo}})
		if err != nil {
			t.Fatalf("%s: error = %v", tipo, err)
		}
		linha := nota.Linhas[1]
		if linha.CFOP != "5903" {
			t.Errorf("%s: CFOP = %s, esperado 5903", tipo, linha.CFOP)
		}
		if linha.CSTICMS != "050" {
			t.Errorf("%s: CST = %s, esperado 050", tipo, linha.CSTICMS)
		}
		if linha.Movimento != tipo {
			t.Errorf("%s: movimento = %s; a nota precisa dizer qual baixa ela representa", tipo, linha.Movimento)
		}
	}
}

// TestAjusteNaoVaiNaNota: ajuste é correção interna, não documento fiscal.
func TestAjusteNaoVaiNaNota(t *testing.T) {
	_, err := MontarNotaDeRetorno(remessaDaNF5911(), "SP", servicoDaNF5911(),
		[]DevolucaoDeMaterial{{RemittanceItemID: 10, Quantidade: "1", Tipo: entity.MovimentoAjuste}})
	if err == nil {
		t.Fatal("ajuste foi aceito na nota fiscal")
	}
	if _, ok := errorsuc.AsValidation(err); !ok {
		t.Fatalf("erro deveria ser de validação: %v", err)
	}
}

// TestInterestadualEhRecusado: 6124/6902/6903 não foram validados pelo responsável
// fiscal. Emitir com CFOP interno para outra UF é nota errada, não nota a menos.
func TestInterestadualEhRecusado(t *testing.T) {
	_, err := MontarNotaDeRetorno(remessaDaNF5911(), "MG", servicoDaNF5911(),
		[]DevolucaoDeMaterial{{RemittanceItemID: 10, Quantidade: "1", Tipo: entity.MovimentoRetorno}})
	if err == nil {
		t.Fatal("destinatário em outra UF foi aceito com CFOP interno")
	}
	if !strings.Contains(err.Error(), "6124") {
		t.Fatalf("a mensagem não diz qual CFOP seria: %v", err)
	}
	// Mesma UF passa.
	if _, err := MontarNotaDeRetorno(remessaDaNF5911(), "sp", servicoDaNF5911(),
		[]DevolucaoDeMaterial{{RemittanceItemID: 10, Quantidade: "1", Tipo: entity.MovimentoRetorno}}); err != nil {
		t.Fatalf("destinatário em SP foi recusado: %v", err)
	}
}

// TestNaoDevolveMaisQueOSaldo: a nota é o documento que baixa o saldo, então o
// limite tem de ser conferido aqui também — não só no movimento.
func TestNaoDevolveMaisQueOSaldo(t *testing.T) {
	_, err := MontarNotaDeRetorno(remessaDaNF5911(), "SP", servicoDaNF5911(),
		[]DevolucaoDeMaterial{{RemittanceItemID: 10, Quantidade: "11", Tipo: entity.MovimentoRetorno}})
	if err == nil {
		t.Fatal("devolver 11 de um saldo de 10 foi aceito")
	}
	if !strings.Contains(err.Error(), "saldo") {
		t.Fatalf("a mensagem não menciona o saldo: %v", err)
	}
}

// TestDuasLinhasDoMesmoItemSomamContraOSaldo: duas devoluções da mesma linha na
// mesma nota não podem passar do saldo somadas — conferir uma por uma deixaria
// passar 6+6 sobre um saldo de 10.
func TestDuasLinhasDoMesmoItemSomamContraOSaldo(t *testing.T) {
	_, err := MontarNotaDeRetorno(remessaDaNF5911(), "SP", servicoDaNF5911(),
		[]DevolucaoDeMaterial{
			{RemittanceItemID: 10, Quantidade: "6", Tipo: entity.MovimentoRetorno},
			{RemittanceItemID: 10, Quantidade: "6", Tipo: entity.MovimentoSobra},
		})
	if err == nil {
		t.Fatal("6+6 sobre um saldo de 10 foi aceito")
	}
	// E 6+4 fecha exatamente.
	if _, err := MontarNotaDeRetorno(remessaDaNF5911(), "SP", servicoDaNF5911(),
		[]DevolucaoDeMaterial{
			{RemittanceItemID: 10, Quantidade: "6", Tipo: entity.MovimentoRetorno},
			{RemittanceItemID: 10, Quantidade: "4", Tipo: entity.MovimentoSobra},
		}); err != nil {
		t.Fatalf("6+4 sobre um saldo de 10 foi recusado: %v", err)
	}
}

// TestRemessaBloqueadaNaoFatura: material retido não pode virar nota.
func TestRemessaBloqueadaNaoFatura(t *testing.T) {
	remessa := remessaDaNF5911()
	remessa.Blocked = true
	motivo := "material sem pedido"
	remessa.BlockReason = &motivo

	_, err := MontarNotaDeRetorno(remessa, "SP", servicoDaNF5911(),
		[]DevolucaoDeMaterial{{RemittanceItemID: 10, Quantidade: "1", Tipo: entity.MovimentoRetorno}})
	if err == nil {
		t.Fatal("remessa bloqueada foi faturada")
	}
	if _, ok := errorsuc.AsConflict(err); !ok {
		t.Fatalf("erro deveria ser de conflito: %v", err)
	}
}

// TestNotaSemMaterialEhRecusada: a nota de beneficiamento devolve o material JUNTO
// com o serviço. Só o serviço deixaria o material do cliente aqui sem documento.
func TestNotaSemMaterialEhRecusada(t *testing.T) {
	_, err := MontarNotaDeRetorno(remessaDaNF5911(), "SP", servicoDaNF5911(), nil)
	if err == nil {
		t.Fatal("nota sem material foi aceita")
	}
	if !strings.Contains(err.Error(), "material") {
		t.Fatalf("a mensagem não explica: %v", err)
	}
}

// TestItemDeOutraRemessaEhRecusado evita faturar contra o saldo errado.
func TestItemDeOutraRemessaEhRecusado(t *testing.T) {
	_, err := MontarNotaDeRetorno(remessaDaNF5911(), "SP", servicoDaNF5911(),
		[]DevolucaoDeMaterial{{RemittanceItemID: 999, Quantidade: "1", Tipo: entity.MovimentoRetorno}})
	if err == nil {
		t.Fatal("item de outra remessa foi aceito")
	}
}

// TestObservacaoTrazODiferimento: o ICMS zero na linha do serviço se sustenta no
// diferimento, e isso precisa estar escrito na nota.
func TestObservacaoTrazODiferimento(t *testing.T) {
	nota, err := MontarNotaDeRetorno(remessaDaNF5911(), "SP", servicoDaNF5911(),
		[]DevolucaoDeMaterial{{RemittanceItemID: 10, Quantidade: "1", Tipo: entity.MovimentoRetorno}})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(nota.Observacao, "CAT 22/2007") {
		t.Errorf("a observação não cita a Portaria CAT 22/2007: %q", nota.Observacao)
	}
	if !strings.Contains(strings.ToUpper(nota.Observacao), "DIFERIMENTO") {
		t.Errorf("a observação não cita o diferimento: %q", nota.Observacao)
	}
}

// TestValorDoMaterialUsaOValorDaEntrada: devolver com outro valor mudaria o valor
// do material do cliente no documento fiscal.
func TestValorDoMaterialUsaOValorDaEntrada(t *testing.T) {
	nota, err := MontarNotaDeRetorno(remessaDaNF5911(), "SP", servicoDaNF5911(),
		[]DevolucaoDeMaterial{{RemittanceItemID: 11, Quantidade: "2", Tipo: entity.MovimentoRetorno}})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	linha := nota.Linhas[1]
	if got := linha.ValorUnit.String(); got != "31.05" {
		t.Errorf("valor unitário = %s, esperado 31.05 (o da nota de entrada)", got)
	}
	if got := linha.ValorTotal.String(); got != "62.1" {
		t.Errorf("valor total = %s, esperado 62.1", got)
	}
}
