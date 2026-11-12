//go:build integration

package impressao_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/impressao"
	stockrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/shopspring/decimal"
)

// Item cadastrado pelo PDM só tem a descrição técnica: o documento impresso
// saía com a descrição em branco. E toda consulta de impressão tem de rodar —
// a pasta não tinha teste e um erro de SQL só aparecia na tela do cliente.
func TestIntegration_DocumentosImpressosRodamEMostramONomeDoPDM(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	c := impressao.New(pool)
	ator := testutil.Actor(t, pool)
	var tn uc.Tenant
	if err := pool.QueryRow(bg, `SELECT id, code FROM enterprise ORDER BY id LIMIT 1`).Scan(&tn.ID, &tn.Code); err != nil {
		t.Fatal(err)
	}

	u := testutil.UniqueCode()
	item := u
	testutil.NovosItens(t, pool, tn.ID, ator, item)
	testutil.Exec(t, pool, `UPDATE items SET name='', commercial_description=NULL, pdm_description_technique='Chapa PDM 3mm' WHERE code=$1`, item)
	const nome = "Chapa PDM 3mm"

	// Nota de entrada sem descrição própria na linha.
	var nota int64
	if err := pool.QueryRow(bg, `INSERT INTO fiscal_entries(numero_nf,serie,modelo,data_emissao,data_entrada,cnpj_emitente,razao_social_emitente,valor_produtos,valor_total,tipo_documento,status,created_by,enterprise_id)
		VALUES($1,'1','55',$2::date,$2::date,'11111111000191','FORNECEDOR TESTE',100,100,'NFE','APPROVED',$3,$4) RETURNING id`,
		u%1_000_000_000, time.Now(), ator, tn.ID).Scan(&nota); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(bg, `DELETE FROM fiscal_entries WHERE id=$1`, nota) })
	testutil.Exec(t, pool, `INSERT INTO fiscal_entry_items(fiscal_entry_id,sequence,item_code,cfop,quantity,unit_price,total_price)
		VALUES($1,1,$2,'1101',1,100,100)`, nota, item)

	// Cotação de compra.
	testutil.Exec(t, pool, `INSERT INTO purchase_quotations(code,enterprise_code,created_by) VALUES($1,$2,$3)`, u, tn.Code, ator)
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM purchase_quotation_items WHERE quotation_code=$1`, u)
		_, _ = pool.Exec(bg, `DELETE FROM purchase_quotations WHERE code=$1`, u)
	})
	testutil.Exec(t, pool, `INSERT INTO purchase_quotation_items(quotation_code,item_code,quantity) VALUES($1,$2,5)`, u, item)

	// Proposta (orçamento de venda).
	var proposta int64
	if err := pool.QueryRow(bg, `INSERT INTO sales_quotations(quotation_number,enterprise_code,status,quotation_type,emission_date,digit_date,currency_code,probability_pct,commission_pct,created_by)
		VALUES($1,$2,'OV','VENDA',$3::date,$3::date,'BRL',50,0,$4) RETURNING code`, u%1_000_000_000, tn.Code, time.Now(), ator).Scan(&proposta); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM sales_quotation_items WHERE sales_quotation_code=$1`, proposta)
		_, _ = pool.Exec(bg, `DELETE FROM sales_quotations WHERE code=$1`, proposta)
	})
	testutil.Exec(t, pool, `INSERT INTO sales_quotation_items(sales_quotation_code,sequence,item_code,requested_qty,unit_price) VALUES($1,1,$2,2,10)`, proposta, item)

	ne, err := c.NotaEntrada(bg, tn, nota)
	if err != nil || ne == nil || len(ne.Itens) != 1 || ne.Itens[0].Descricao != nome {
		t.Fatalf("nota de entrada: err=%v doc=%+v", err, ne)
	}
	ct, err := c.Cotacao(bg, tn, u)
	if err != nil || ct == nil || len(ct.Itens) != 1 || ct.Itens[0].Descricao != nome {
		t.Fatalf("cotação: err=%v doc=%+v", err, ct)
	}
	pr, err := c.Proposta(bg, tn, proposta)
	if err != nil || pr == nil || len(pr.Itens) != 1 || pr.Itens[0].Descricao != nome {
		t.Fatalf("proposta: err=%v doc=%+v", err, pr)
	}

	// Outra empresa não enxerga os documentos desta.
	outra := uc.Tenant{ID: -1, Code: -1}
	if d, err := c.NotaEntrada(bg, outra, nota); err != nil || d != nil {
		t.Fatalf("nota vista por outra empresa: %v %+v", err, d)
	}
	if d, err := c.Cotacao(bg, outra, u); err != nil || d != nil {
		t.Fatalf("cotação vista por outra empresa: %v %+v", err, d)
	}
	if d, err := c.Proposta(bg, outra, proposta); err != nil || d != nil {
		t.Fatalf("proposta vista por outra empresa: %v %+v", err, d)
	}

	// As demais consultas rodam (SQL válido) mesmo sem documento.
	hoje := time.Now()
	de := hoje.AddDate(0, -1, 0)
	kardexFechaComOSaldo(t, c, tn, item, de, hoje)
	if _, err := c.Emitente(bg, tn); err != nil {
		t.Fatalf("emitente: %v", err)
	}
	for nomeDoc, f := range map[string]func() error{
		"pedido":     func() error { _, err := c.PedidoVenda(bg, tn, -1); return err },
		"romaneio":   func() error { _, err := c.Romaneio(bg, tn, -1); return err },
		"plano":      func() error { _, err := c.PlanoMateriais(bg, tn, -1); return err },
		"capacidade": func() error { _, err := c.Capacidade(bg, tn, -1); return err },
		"cliente":    func() error { _, err := c.FichaCliente(bg, tn, -1); return err },
		"receber":    func() error { _, err := c.Titulos(bg, tn, true, uc.FiltroTitulos{De: &de, Ate: &hoje}); return err },
		"pagar":      func() error { _, err := c.Titulos(bg, tn, false, uc.FiltroTitulos{SomenteAbertos: true}); return err },
		"vencidos": func() error {
			_, err := c.Titulos(bg, tn, true, uc.FiltroTitulos{De: &de, Ate: &hoje, PorEmissao: true, Situacao: "pendente", SomenteVencidos: true})
			return err
		},
		"contas":   func() error { _, err := c.ContasBancarias(bg, tn); return err },
		"programa": func() error { _, err := c.Programacao(bg, tn, de, hoje, nil); return err },
		"kardex":   func() error { _, err := c.Kardex(bg, tn, uc.FiltroKardex{De: de, Ate: hoje}); return err },
		"custo":    func() error { _, err := c.CustoProducao(bg, tn, nil, &de, &hoje); return err },
	} {
		if err := f(); err != nil {
			t.Errorf("%s: %v", nomeDoc, err)
		}
	}
}

// O kardex impresso tem de fechar com o kardex da tela e com o saldo oficial
// (SignedQuantity): mesmo sinal por tipo, data do documento (o lançamento
// retroativo entra no saldo anterior) e o ajuste de custo no valor.
func kardexFechaComOSaldo(t *testing.T, c *impressao.Consultas, tn uc.Tenant, item int64, de, hoje time.Time) {
	t.Helper()
	pool := testutil.Pool(t)
	ator := testutil.Actor(t, pool)
	dep := testutil.NovoAlmoxarifado(t, pool, tn.ID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM stock_movements WHERE item_code=$1`, item) })
	inserir := func(tipo, qtd, total string, documento time.Time) {
		t.Helper()
		if _, err := pool.Exec(context.Background(), `INSERT INTO stock_movements(item_code,warehouse_id,movement_type,quantity,unit_price,total_price,created_by,enterprise_id,movement_date)
			VALUES($1,$2,$3,$4,10,$5,$6,$7,$8::date)`, item, dep, tipo, qtd, total, ator, tn.ID, documento.Format("2006-01-02")); err != nil {
			t.Fatalf("movimento %s: %v", tipo, err)
		}
	}
	movs := []struct {
		tipo string
		qtd  string
	}{
		{stockentity.MovementTypeIn, "100"}, {stockentity.MovementTypeOut, "10"},
		{stockentity.MovementTypeProductionEntry, "5"}, {stockentity.MovementTypePlannedProductionEntry, "7"},
		{stockentity.MovementTypeProductionExcess, "2"}, {stockentity.MovementTypePlannedRequisition, "3"},
		{stockentity.MovementTypeTransferIn, "4"}, {stockentity.MovementTypeTransferOut, "1"},
		{stockentity.MovementTypeAdjustment, "-6"}, {stockentity.MovementTypeAdjustment, "8"},
		{stockentity.MovementTypeAddressTransfer, "50"},
	}
	esperado := decimal.Zero
	dentro := de.AddDate(0, 0, 2)
	for _, m := range movs {
		q := decimal.RequireFromString(m.qtd)
		esperado = esperado.Add(stockentity.SignedQuantity(m.tipo, q))
		total := "0"
		if m.tipo != stockentity.MovementTypeAddressTransfer {
			total = q.Abs().Mul(decimal.NewFromInt(10)).String()
		}
		inserir(m.tipo, m.qtd, total, dentro)
	}
	// Gravado hoje com data de documento anterior ao período: é saldo anterior.
	antes := de.AddDate(0, 0, -5)
	inserir(stockentity.MovementTypeIn, "20", "200", antes)
	esperado = esperado.Add(decimal.NewFromInt(20))
	// Frete lançado depois da nota: só valor.
	inserir(stockentity.MovementTypeCostAdjustment, "0", "37.50", dentro)

	k, err := c.Kardex(context.Background(), tn, uc.FiltroKardex{Item: fmt.Sprint(item), Deposito: &dep, De: de, Ate: hoje})
	if err != nil || len(k.Itens) != 1 {
		t.Fatalf("kardex: err=%v itens=%d", err, len(k.Itens))
	}
	it := k.Itens[0]
	if !it.SaldoFinal.Equal(esperado) || !it.SaldoInicial.Equal(decimal.NewFromInt(20)) {
		t.Fatalf("kardex impresso: anterior %s final %s; esperado anterior 20 final %s", it.SaldoInicial, it.SaldoFinal, esperado)
	}
	// Os 11 do período menos a troca de endereço, mais o ajuste de custo.
	if len(it.Movimentos) != len(movs) {
		t.Fatalf("movimentos no período = %d, esperado %d", len(it.Movimentos), len(movs))
	}

	ctx := context.WithValue(context.Background(), contextkey.UserKey, &security.AuthUser{ID: ator.String(), Role: "ADMIN", EnterpriseID: tn.ID, EnterpriseCode: tn.Code})
	tela, err := stockrepo.NewStockRepositorySQLC(pool).Kardex(ctx, item, &dep, de, hoje)
	if err != nil {
		t.Fatal(err)
	}
	valor := it.ValorFinal
	if !tela.SaldoFinal.Equal(it.SaldoFinal) || !tela.SaldoAnterior.Equal(it.SaldoInicial) ||
		!tela.ValorFinal.Equal(valor) || !tela.ValorAnterior.Equal(it.CustoInicial) {
		t.Fatalf("tela: anterior %s/%s final %s/%s; impresso: anterior %s/%s final %s/%s",
			tela.SaldoAnterior, tela.ValorAnterior, tela.SaldoFinal, tela.ValorFinal, it.SaldoInicial, it.CustoInicial, it.SaldoFinal, valor)
	}
}

// O PDF da carteira traz o que a lista da tela mostra: documento por trecho,
// plano de contas e centro de custo pelo título ou pelo rateio, faixa de valor,
// tipo, aprovação e, em receber, o pedido de venda.
func TestIntegration_TitulosImpressosSeguemOsFiltrosDaTela(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	c := impressao.New(pool)
	ator := testutil.Actor(t, pool)
	var tn uc.Tenant
	if err := pool.QueryRow(bg, `SELECT id, code FROM enterprise ORDER BY id LIMIT 1`).Scan(&tn.ID, &tn.Code); err != nil {
		t.Fatal(err)
	}
	u := testutil.UniqueCode()
	pref := fmt.Sprintf("IMPF%d", u)
	hoje := time.Now()

	var plano, centro int64
	if err := pool.QueryRow(bg, `INSERT INTO plano_contas(codigo,descricao,tipo,natureza,enterprise_id) VALUES($1,'Matéria-prima teste','DESPESA','DEVEDORA',$2) RETURNING id`,
		pref, tn.ID).Scan(&plano); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO cost_centers(code,description,type,start_date,created_by,enterprise_id) VALUES($1,'Usinagem teste','PRODUCTIVE',$2::date,$3,$4) RETURNING id`,
		u%1_000_000_000, hoje, ator, tn.ID).Scan(&centro); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM contas_pagar WHERE numero_documento LIKE $1`, pref+"%")
		_, _ = pool.Exec(bg, `DELETE FROM contas_receber WHERE numero_documento LIKE $1`, pref+"%")
		_, _ = pool.Exec(bg, `DELETE FROM cost_centers WHERE id=$1`, centro)
		_, _ = pool.Exec(bg, `DELETE FROM plano_contas WHERE id=$1`, plano)
	})

	pagar := func(sufixo, tipo, aprovacao string, valor int, plano *int64) int64 {
		var id int64
		if err := pool.QueryRow(bg, `INSERT INTO contas_pagar(numero_documento,tipo_documento,data_emissao,data_vencimento,valor_bruto,status_aprovacao,plano_contas_id,criado_por,enterprise_id)
			VALUES($1,$2,$3::date,$3::date,$4,$5,$6,$7,$8) RETURNING id`, pref+sufixo, tipo, hoje, valor, aprovacao, plano, ator, tn.ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	pagar("-A", "NF", "APROVADO", 100, &plano)
	b := pagar("-B", "BOLETO", "PENDENTE", 500, nil)
	pagar("-C", "NF", "PENDENTE", 50, nil)
	testutil.Exec(t, pool, `INSERT INTO contas_pagar_rateios(enterprise_id,conta_pagar_id,plano_contas_id,centro_custo_id,valor) VALUES($1,$2,$3,$4,500)`, tn.ID, b, plano, centro)

	docs := func(receber bool, f uc.FiltroTitulos) []string {
		t.Helper()
		f.Documento = pref + f.Documento
		l, err := c.Titulos(bg, tn, receber, f)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, x := range l {
			out = append(out, x.Documento[len(pref):])
		}
		return out
	}
	um := decimal.NewFromInt(100)
	for nome, caso := range map[string]struct {
		f    uc.FiltroTitulos
		quer string
	}{
		"documento":       {uc.FiltroTitulos{Documento: "-B"}, "[-B]"},
		"plano (rateio)":  {uc.FiltroTitulos{PlanoContasID: &plano}, "[-A -B]"},
		"centro (rateio)": {uc.FiltroTitulos{CentroCustoID: &centro}, "[-B]"},
		"valor mínimo":    {uc.FiltroTitulos{ValorMinimo: &um}, "[-A -B]"},
		"valor máximo":    {uc.FiltroTitulos{ValorMaximo: &um}, "[-A -C]"},
		"tipo":            {uc.FiltroTitulos{TipoDocumento: "NF"}, "[-A -C]"},
		"aprovação":       {uc.FiltroTitulos{Aprovacao: "aprovado"}, "[-A]"},
	} {
		if got := fmt.Sprint(docs(false, caso.f)); got != caso.quer {
			t.Errorf("pagar %s: %s, esperado %s", nome, got, caso.quer)
		}
	}

	// Receber: pedido de venda e plano de contas do título (sem rateio).
	receber := func(sufixo string, pedido *int64, plano *int64) {
		testutil.Exec(t, pool, `INSERT INTO contas_receber(numero_documento,data_emissao,data_vencimento,valor_bruto,sales_order_id,plano_contas_id,criado_por,enterprise_id)
			VALUES($1,$2::date,$2::date,10,$3,$4,$5,$6)`, pref+sufixo, hoje, pedido, plano, ator, tn.ID)
	}
	pedido := u
	receber("-R1", &pedido, nil)
	receber("-R2", nil, &plano)
	if got := fmt.Sprint(docs(true, uc.FiltroTitulos{PedidoID: &pedido})); got != "[-R1]" {
		t.Errorf("receber por pedido: %s", got)
	}
	if got := fmt.Sprint(docs(true, uc.FiltroTitulos{PlanoContasID: &plano, TipoDocumento: "NF", Aprovacao: "APROVADO"})); got != "[-R2]" {
		t.Errorf("receber por plano (tipo e aprovação não existem em receber): %s", got)
	}

	p, cc, err := c.RotulosContabeis(bg, tn, &plano, &centro)
	if err != nil || p != pref+" Matéria-prima teste" || cc != fmt.Sprint(u%1_000_000_000)+" Usinagem teste" {
		t.Fatalf("rótulos: %q %q %v", p, cc, err)
	}
	if p, cc, err := c.RotulosContabeis(bg, uc.Tenant{ID: -1, Code: -1}, &plano, &centro); err != nil || p != "" || cc != "" {
		t.Fatalf("rótulos de outra empresa: %q %q %v", p, cc, err)
	}
}
