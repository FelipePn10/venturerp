//go:build integration

package devolucao_venda_uc_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/devolucao_venda_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/devolucaovenda"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	devpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/devolucaovenda"
	financialRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	fiscalRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// A devolução nasce da NF de venda autorizada e só devolve o que foi vendido.
// Pela NF do cliente: efetiva na hora — estoque volta pelo custo da saída, o
// título em aberto é abatido, o que sobra vira crédito do cliente, a comissão
// é estornada na proporção e a série volta ao estoque. Pela emissão própria:
// gera a NF-e de ENTRADA (tipo 0, finalidade 4, NF de venda referenciada, CFOP
// de devolução) e efetiva quando ela é autorizada; depois disso a NF-e não se
// cancela. Outra empresa não vê nada.
func TestDevolucaoDeVenda(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	peca, equip := 88_000_000+u%1_000_000, 88_500_000+u%1_000_000
	cliente := 875_000 + u%90_000
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []testutil.Empresa{a, b} {
			for _, q := range []string{
				`DELETE FROM commercial_commission_ledger WHERE enterprise_id=$1`,
				`DELETE FROM fiscal_return_settlements WHERE enterprise_id=$1`, `DELETE FROM adiantamentos WHERE enterprise_id=$1`,
				`DELETE FROM devolucoes_venda WHERE enterprise_id=$1`,
				`DELETE FROM contas_receber WHERE enterprise_id=$1`, `DELETE FROM stock_movements WHERE enterprise_id=$1`, `DELETE FROM stock_balances WHERE enterprise_id=$1`,
				`DELETE FROM fiscal_exit_item_series WHERE enterprise_id=$1`, `DELETE FROM numeros_serie WHERE enterprise_id=$1`, `DELETE FROM item_pos_venda WHERE enterprise_id=$1`,
				`DELETE FROM fiscal_exit_items WHERE fiscal_exit_id IN (SELECT id FROM fiscal_exits WHERE enterprise_id=$1)`, `DELETE FROM fiscal_exits WHERE enterprise_id=$1`,
				`DELETE FROM customers WHERE enterprise_id=$1`, `DELETE FROM items WHERE enterprise_id=$1`, `DELETE FROM warehouse WHERE enterprise_id=$1`,
			} {
				if _, err := pool.Exec(bg, q, e.ID); err != nil {
					t.Logf("limpeza %s: %v", q, err)
				}
			}
		}
	})
	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		VALUES($1,'Acabado',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`, fmt.Sprintf("DV-%d", u), actor, a.ID).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	for _, it := range []int64{peca, equip} {
		testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name) VALUES($1,'DV'||($1::bigint)::text,$2,$3,$4,'ITEM DEV')`, it, almox, actor, a.ID)
	}
	testutil.Exec(t, pool, `INSERT INTO item_pos_venda(enterprise_id,item_code,controla_serie) VALUES($1,$2,TRUE)`, a.ID, equip)
	testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES($1,'CLIENTE DEV','11444777000161',$2,$3)`, cliente, actor, a.ID)

	// NF de venda autorizada: 10 peças a 100 (ICMS 18%, IPI 5%) e 1 balança a 1000.
	chave := fmt.Sprintf("41%042d", u)
	var nota, lPeca, lEquip int64
	if err := pool.QueryRow(bg, `INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,created_by,enterprise_id,customer_code,status,chave_acesso,valor_produtos,valor_total,cnpj_destinatario,razao_social_destinatario,uf_destinatario)
		VALUES($1,$2::date,'5101','VENDA',$3,$4,$5,'AUTHORIZED',$6,2000,2050,'11444777000161','CLIENTE DEV','PR') RETURNING id`, 5000+u%1000, time.Now(), actor, a.ID, cliente, chave).Scan(&nota); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO fiscal_exit_items(fiscal_exit_id,sequence,item_code,cfop,quantity,unit_price,total_price,base_icms,aliq_icms,valor_icms,base_ipi,aliq_ipi,valor_ipi,valor_pis,valor_cofins,description,cst_icms,origem_mercadoria,ncm)
		VALUES($1,1,$2,'5101',10,100,1000,1000,18,180,1000,5,50,16.5,76,'PECA','00','0','73269090') RETURNING id`, nota, peca).Scan(&lPeca); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO fiscal_exit_items(fiscal_exit_id,sequence,item_code,cfop,quantity,unit_price,total_price,description,cst_icms,origem_mercadoria,ncm)
		VALUES($1,2,$2,'5101',1,1000,1000,'BALANCA','00','0','84238200') RETURNING id`, nota, equip).Scan(&lEquip); err != nil {
		t.Fatal(err)
	}
	// O que a venda baixou do estoque (custo 40/peça, 600 a balança).
	testutil.Exec(t, pool, `INSERT INTO stock_movements(item_code,warehouse_id,movement_type,quantity,unit_price,total_price,reference_type,reference_code,created_by,enterprise_id)
		VALUES($1,$2,'OUT',10,40,400,'NF_SAIDA',$3,$4,$5),($6,$2,'OUT',1,600,600,'NF_SAIDA',$3,$4,$5)`, peca, almox, nota, actor, a.ID, equip)
	var serie int64
	if err := pool.QueryRow(bg, `INSERT INTO numeros_serie(enterprise_id,item_code,numero_serie,situacao,origem,customer_code,fiscal_exit_id,data_venda,created_by)
		VALUES($1,$2,'DV-1','NO_CLIENTE','CADASTRO',$3,$4,CURRENT_DATE,$5) RETURNING id`, a.ID, equip, cliente, nota, actor).Scan(&serie); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO fiscal_exit_item_series(enterprise_id,fiscal_exit_id,fiscal_exit_item_id,serie_id,ativo,created_by) VALUES($1,$2,$3,$4,TRUE,$5)`, a.ID, nota, lEquip, serie, actor)
	// Títulos: 1ª parcela já recebida, 2ª em aberto (1025).
	testutil.Exec(t, pool, `INSERT INTO contas_receber(numero_documento,cliente_id,fiscal_exit_id,data_emissao,data_vencimento,valor_bruto,valor_recebido,status,is_active,criado_por,enterprise_id)
		VALUES('NF-1',$1,$2,CURRENT_DATE,CURRENT_DATE+30,1025,1025,'RECEBIDO',TRUE,$3,$4),('NF-2',$1,$2,CURRENT_DATE,CURRENT_DATE+60,1025,0,'PENDENTE',TRUE,$3,$4)`, cliente, nota, actor, a.ID)
	// Comissão de 5% sobre 2050.
	testutil.Exec(t, pool, `INSERT INTO commercial_commission_ledger(enterprise_id,representative_code,sales_order_code,fiscal_exit_id,event_type,competence_date,base_amount,commission_pct,amount,idempotency_key,actor_id)
		VALUES($1,7,1,$2,'COMPETENCIA_FATURAMENTO',CURRENT_DATE,2050,5,102.5,$3,$4)`, a.ID, nota, fmt.Sprintf("fiscal:%d:authorized", nota), actor)

	fiscal := fiscalRepo.NewFiscalRepositoryPG(pool)
	uc := &devolucao_venda_uc.UseCase{Repo: devpg.New(pool), Fiscal: fiscal, FinancialRepo: financialRepo.NewFinancialRepositoryPG(pool), Auth: &auth.AuthService{}}

	p, err := uc.Previa(a.Ctx, nota)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Linhas) != 2 || !p.Linhas[0].Disponivel.Equal(d("10")) || !p.Linhas[1].ControlaSerie || len(p.Linhas[1].Series) != 1 {
		t.Fatalf("prévia: %+v", p.Linhas)
	}
	if _, err := uc.Previa(b.Ctx, nota); err == nil {
		t.Fatal("a outra empresa não vê a NF")
	}

	// 1. Pela NF do cliente: 4 peças + a balança com a série.
	chaveCliente := fmt.Sprintf("41%042d", u+7)
	dv, err := uc.Criar(a.Ctx, devolucao_venda_uc.CriarDTO{FiscalExitID: nota, Modo: "NF_CLIENTE", Motivo: "DEFEITO", WarehouseID: almox, NFClienteChave: &chaveCliente,
		Itens: []dom.Pedido{{FiscalExitItemID: lPeca, Quantidade: d("4")}, {FiscalExitItemID: lEquip, Quantidade: d("1"), Series: []int64{serie}}}})
	if err != nil {
		t.Fatal(err)
	}
	// peças: 400 + IPI 20 = 420; balança 1000 → 1420.
	if !dv.ValorTotal.Equal(d("1420")) || dv.Itens[0].CFOP != "1201" {
		t.Fatalf("devolução: total %s, itens %+v", dv.ValorTotal, dv.Itens)
	}
	if _, err := uc.Criar(a.Ctx, devolucao_venda_uc.CriarDTO{FiscalExitID: nota, Modo: "NF_CLIENTE", Motivo: "DEFEITO", WarehouseID: almox, NFClienteChave: &chaveCliente,
		Itens: []dom.Pedido{{FiscalExitItemID: lPeca, Quantidade: d("7")}}}); err == nil {
		t.Fatal("4 já em devolução + 7 passa das 10 vendidas")
	}
	efetivada, avisos, err := uc.Efetivar(a.Ctx, dv.ID)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(avisos)
	if efetivada.Situacao != dom.Efetivada || !efetivada.Abatido.Equal(d("1025")) || !efetivada.CreditoCliente.Equal(d("395")) || efetivada.AdiantamentoID == nil {
		t.Fatalf("efetivação: %+v", efetivada)
	}
	var custoPeca, saldo decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT unit_price, quantity FROM stock_movements WHERE enterprise_id=$1 AND reference_type='DEVOLUCAO_VENDA' AND reference_code=$2 AND item_code=$3`,
		a.ID, dv.ID, peca).Scan(&custoPeca, &saldo); err != nil {
		t.Fatal(err)
	}
	if !custoPeca.Equal(d("40")) || !saldo.Equal(d("4")) {
		t.Fatalf("estoque volta pelo custo da saída: %s × %s", saldo, custoPeca)
	}
	var tituloStatus string
	_ = pool.QueryRow(bg, `SELECT status FROM contas_receber WHERE enterprise_id=$1 AND numero_documento='NF-2'`, a.ID).Scan(&tituloStatus)
	if tituloStatus != "RECEBIDO" {
		t.Fatalf("título em aberto abatido por inteiro: %s", tituloStatus)
	}
	var credito decimal.Decimal
	var tipo string
	_ = pool.QueryRow(bg, `SELECT tipo, valor_original FROM adiantamentos WHERE id=$1 AND enterprise_id=$2`, *efetivada.AdiantamentoID, a.ID).Scan(&tipo, &credito)
	if tipo != "RECEBER" || !credito.Equal(d("395")) {
		t.Fatalf("crédito do cliente: %s %s", tipo, credito)
	}
	var estorno decimal.Decimal
	_ = pool.QueryRow(bg, `SELECT amount FROM commercial_commission_ledger WHERE enterprise_id=$1 AND fiscal_exit_id=$2 AND event_type='ESTORNO_FATURAMENTO'`, a.ID, nota).Scan(&estorno)
	// 102,5 × 1420/2050 = 71
	if !estorno.Equal(d("-71")) {
		t.Fatalf("estorno da comissão: %s", estorno)
	}
	var situacaoSerie string
	var almoxSerie *int64
	_ = pool.QueryRow(bg, `SELECT situacao, warehouse_id FROM numeros_serie WHERE id=$1`, serie).Scan(&situacaoSerie, &almoxSerie)
	if situacaoSerie != "EM_ESTOQUE" || almoxSerie == nil || *almoxSerie != almox {
		t.Fatalf("série volta ao estoque: %s %v", situacaoSerie, almoxSerie)
	}
	if again, _, err := uc.Efetivar(a.Ctx, dv.ID); err != nil || !again.Abatido.Equal(d("1025")) {
		t.Fatalf("efetivar de novo não duplica: %v", err)
	}
	var movs int
	_ = pool.QueryRow(bg, `SELECT COUNT(*) FROM stock_movements WHERE enterprise_id=$1 AND reference_type='DEVOLUCAO_VENDA'`, a.ID).Scan(&movs)
	if movs != 2 {
		t.Fatalf("movimentos da devolução: %d", movs)
	}

	// 2. Emissão própria: as 6 peças restantes.
	dp, err := uc.Criar(a.Ctx, devolucao_venda_uc.CriarDTO{FiscalExitID: nota, Modo: "EMISSAO_PROPRIA", Motivo: "DESISTENCIA", WarehouseID: almox,
		Itens: []dom.Pedido{{FiscalExitItemID: lPeca, Quantidade: d("6")}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := uc.Efetivar(a.Ctx, dp.ID); err == nil || !strings.Contains(err.Error(), "autorizada") {
		t.Fatalf("emissão própria só efetiva com a NF-e autorizada: %v", err)
	}
	dp, err = uc.GerarNF(a.Ctx, dp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dp.NFPropriaID == nil {
		t.Fatal("NF-e de entrada gerada")
	}
	nf, err := fiscal.GetExitByID(a.Ctx, *dp.NFPropriaID)
	if err != nil {
		t.Fatal(err)
	}
	if nf.TipoDocumento == nil || *nf.TipoDocumento != 0 || nf.Finalidade != 4 || nf.NFeReferenciada == nil || *nf.NFeReferenciada != chave || nf.Cfop != "1201" || nf.DevolucaoVendaID == nil {
		t.Fatalf("NF-e de entrada: tipo %v finalidade %d ref %v cfop %s", nf.TipoDocumento, nf.Finalidade, nf.NFeReferenciada, nf.Cfop)
	}
	itensNF, _ := fiscal.GetExitItems(a.Ctx, nf.ID)
	if len(itensNF) != 1 || itensNF[0].Quantity != 6 || itensNF[0].ValorICMS != 108 || itensNF[0].BaseICMS != 600 {
		t.Fatalf("item da NF-e: %+v", itensNF)
	}
	// Autorizada na SEFAZ → efetiva.
	testutil.Exec(t, pool, `UPDATE fiscal_exits SET status='AUTHORIZED' WHERE id=$1`, nf.ID)
	nf.Status = "AUTHORIZED"
	for _, av := range uc.AoAutorizar(a.Ctx, nf, actor) {
		t.Log(av)
	}
	dp, _ = uc.Obter(a.Ctx, dp.ID)
	if dp.Situacao != dom.Efetivada || !dp.CreditoCliente.Equal(d("630")) {
		t.Fatalf("emissão própria efetivada (tudo vira crédito: 600 + IPI 30): %+v", dp)
	}
	if err := uc.PodeCancelarNF(a.Ctx, nf); err == nil {
		t.Fatal("a NF-e da devolução efetivada não se cancela")
	}
	if p, _ := uc.Previa(a.Ctx, nota); !p.Linhas[0].Disponivel.IsZero() {
		t.Fatalf("tudo devolvido: %s", p.Linhas[0].Disponivel)
	}

	// Isolamento.
	if _, err := uc.Obter(b.Ctx, dv.ID); err == nil {
		t.Fatal("a outra empresa não lê a devolução")
	}
	if l, _ := uc.Listar(b.Ctx, nil, ""); len(l) != 0 {
		t.Fatal("a outra empresa não lista as devoluções")
	}
}
