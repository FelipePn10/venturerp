//go:build integration

package fiscal_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/reforma"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A parametrização da reforma é da empresa: alíquotas, CFOP e cClassTrib das
// linhas de uma não aparecem nem são alteradas pela outra. A precedência na
// nota é item › CFOP › classificação fiscal (vigente na emissão).
func TestReformaTributariaPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM fiscal_exit_items WHERE fiscal_exit_id IN (SELECT id FROM fiscal_exits WHERE enterprise_id = ANY($1))`, ids)
		pool.Exec(base, `DELETE FROM fiscal_exits WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM fiscal_classifications WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM ibscbs_cfop_classificacoes WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM ibscbs_aliquotas WHERE enterprise_id = ANY($1)`, ids)
	})
	repo := fiscalrepo.NewReformaPG(pool)
	d := decimal.RequireFromString

	// Tabela oficial carregada pela migração.
	tab, err := repo.BuscarClassificacoes(base, []string{"000001", "410001", "200030", "999999"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tab) != 3 || tab["410001"].ExigeTributacao || !tab["200030"].PercReducaoCBS.Equal(d("60")) || tab["000001"].CST != "000" {
		t.Fatalf("tabela oficial: %+v", tab)
	}

	// Alíquotas: A cadastra 2027, B não enxerga.
	if err := repo.SalvarAliquotas(a.Ctx, 2027, reforma.Aliquotas{IBSUF: d("0.05"), IBSMun: d("0.05"), CBS: d("8.7")}, "teste", actor); err != nil {
		t.Fatal(err)
	}
	if got, err := repo.AliquotasDoAno(a.Ctx, 2027); err != nil || got == nil || !got.CBS.Equal(d("8.7")) {
		t.Fatalf("A deveria ler a própria CBS: %+v %v", got, err)
	}
	if got, err := repo.AliquotasDoAno(b.Ctx, 2027); err != nil || got != nil {
		t.Fatalf("B leu a alíquota da A: %+v %v", got, err)
	}
	if err := repo.ExcluirAliquotas(b.Ctx, 2027); err == nil {
		t.Fatal("B excluiu a alíquota da A")
	}

	// CFOP 5910 (bonificação) → 410001 só na A.
	if err := repo.SalvarCFOP(a.Ctx, "5910", "410001", nil, actor); err != nil {
		t.Fatal(err)
	}
	if lista, _ := repo.ListarCFOP(b.Ctx); len(lista) != 0 {
		t.Fatalf("B viu o CFOP da A: %+v", lista)
	}

	// Item da A com classificação fiscal (cClassTrib 200030) e nota com três
	// linhas: venda (classificação), bonificação (CFOP) e uma com cClassTrib próprio.
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, actor)
	classif := testutil.UniqueCode()
	if _, err := pool.Exec(base, `UPDATE items SET accounting_sale_fiscal_classification_code = $1 WHERE code = $2`, fmt.Sprint(classif), item); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(base, `INSERT INTO fiscal_classifications(code, description, ncm, cod_clas_trib, created_by, enterprise_id, valid_from)
		VALUES ($1, 'Classificação reduzida', '73269000', '200030', $2, $3, DATE '2025-01-01')`, classif, actor, a.ID); err != nil {
		t.Fatal(err)
	}
	var notaID int64
	if err := pool.QueryRow(base, `INSERT INTO fiscal_exits(numero_nf, data_emissao, cfop, natureza_operacao, valor_total, status, created_by, enterprise_id)
		VALUES (1, DATE '2026-11-10', '5101', 'Venda', 300, 'DRAFT', $1, $2) RETURNING id`, actor, a.ID).Scan(&notaID); err != nil {
		t.Fatal(err)
	}
	linha := func(seq int, cfop string) int64 {
		var id int64
		if err := pool.QueryRow(base, `INSERT INTO fiscal_exit_items(fiscal_exit_id, sequence, item_code, ncm, cfop, quantity, unit_price, total_price)
			VALUES ($1, $2, $3, '73269000', $4, 1, 100, 100) RETURNING id`, notaID, seq, item, cfop).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	venda, bonif, propria := linha(1, "5101"), linha(2, "5910"), linha(3, "5101")
	codigo := "000001"
	if err := repo.DefinirClassTribDoItem(b.Ctx, notaID, propria, &codigo); err == nil {
		t.Fatal("B alterou a linha da nota da A")
	}
	if err := repo.DefinirClassTribDoItem(a.Ctx, notaID, propria, &codigo); err != nil {
		t.Fatal(err)
	}
	cand, err := repo.CandidatosDaNota(a.Ctx, notaID)
	if err != nil {
		t.Fatal(err)
	}
	escolha := func(id int64) string {
		c, _ := reforma.EscolherClassTrib(cand[id].DoItem, cand[id].DoCFOP, cand[id].DaClassificacao)
		return c
	}
	if escolha(venda) != "200030" || escolha(bonif) != "410001" || escolha(propria) != "000001" {
		t.Fatalf("precedência: %+v", cand)
	}
	if outra, _ := repo.CandidatosDaNota(b.Ctx, notaID); len(outra) != 0 {
		t.Fatalf("B leu as linhas da nota da A: %+v", outra)
	}

	// Retrato gravado na linha e no total; nota transmitida não muda mais.
	g := reforma.Calcular(reforma.Aliquotas{IBSUF: d("0.1"), CBS: d("0.9")}, tab["200030"].Classificacao, d("100"), d("0"), d("0"), d("0"))
	if err := repo.GravarRetrato(a.Ctx, notaID, []reforma.RetratoItem{{ItemID: venda, Origem: reforma.OrigemClassificacao, Grupo: g}}, g.IBS, g.CBS.Valor); err != nil {
		t.Fatal(err)
	}
	var cst, origem string
	var cbs decimal.Decimal
	if err := pool.QueryRow(base, `SELECT ibscbs_cst_enviado, ibscbs_origem, cbs_valor FROM fiscal_exit_items WHERE id = $1`, venda).Scan(&cst, &origem, &cbs); err != nil {
		t.Fatal(err)
	}
	if cst != "200" || origem != reforma.OrigemClassificacao || !cbs.Equal(g.CBS.Valor) {
		t.Fatalf("retrato: %s %s %s", cst, origem, cbs)
	}
	if err := repo.GravarRetrato(b.Ctx, notaID, nil, d("0"), d("0")); err == nil {
		t.Fatal("B gravou o retrato da nota da A")
	}
	if _, err := pool.Exec(base, `UPDATE fiscal_exits SET status = 'AUTHORIZED' WHERE id = $1`, notaID); err != nil {
		t.Fatal(err)
	}
	if err := repo.DefinirClassTribDoItem(a.Ctx, notaID, propria, nil); err == nil {
		t.Fatal("linha de nota autorizada não pode mudar de classificação")
	}
}
