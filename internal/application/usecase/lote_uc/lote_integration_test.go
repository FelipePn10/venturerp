//go:build integration

package lote_uc_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/lote_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/procurement_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/quality/amostragem"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/lote"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	amostragempg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/amostragem"
	lotepg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/lote"
	mrppg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation"
	procurementpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/procurement"
	stockpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func dd(s string) decimal.Decimal { return decimal.RequireFromString(s) }
func ip(v int64) *int64           { return &v }
func sp(v string) *string         { return &v }
func bp(v bool) *bool             { return &v }

// O lote de inspeção de ponta a ponta numa empresa com código ≠ id.
func TestLoteDeInspecaoDoRecebimentoADecisao(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	item, fornecedor := 96_000_000+u%1_000_000, int64(7000+u%1000)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []testutil.Empresa{a, b} {
			for _, q := range []string{
				`DELETE FROM non_conformances WHERE lot_id IN (SELECT id FROM quality_inspection_lots WHERE enterprise_id=$1)`,
				`ALTER TABLE quality_lot_decisions DISABLE TRIGGER trg_quality_lot_decisions_imutaveis`,
				`DELETE FROM quality_inspection_lots WHERE enterprise_id=$1`,
				`ALTER TABLE quality_lot_decisions ENABLE TRIGGER trg_quality_lot_decisions_imutaveis`,
				`DELETE FROM receiving_inspection_routes WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1)`,
				`DELETE FROM quality_sampling_procedures WHERE enterprise_id=$1`,
				`DELETE FROM stock_lot_balances WHERE enterprise_id=$1`,
				`DELETE FROM stock_movements WHERE enterprise_id=$1`, `DELETE FROM stock_balances WHERE enterprise_id=$1`,
				`DELETE FROM items WHERE enterprise_id=$1`, `DELETE FROM warehouse WHERE enterprise_id=$1`,
			} {
				if strings.HasPrefix(q, "ALTER") {
					_, _ = pool.Exec(bg, q)
					continue
				}
				if _, err := pool.Exec(bg, q, e.ID); err != nil {
					t.Logf("limpeza: %s: %v", q, err)
				}
			}
		}
	})
	almox := func(code, local string, disp bool) int64 {
		var id int64
		if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
			VALUES($1,$1,$2,$3,'NORMAL',$4,$4,$5) RETURNING id`, fmt.Sprintf("%s-%d", code, u), actor, local, disp, a.ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	insp, alm, rej := almox("INSP", "INSPECAO", false), almox("ALM", "INTERNO", true), almox("REJ", "REJEICAO", false)
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name) VALUES($1,($1::bigint)::text,$2,$3,$4,'PARAFUSO')`, item, alm, actor, a.ID)

	ctx := a.Ctx
	am := amostragempg.New(pool)
	proc := &amostragem.Procedimento{Codigo: "REC", Descricao: "NQA 1,0", Tipo: amostragem.TipoNorma, Nivel: amostragem.NivelII, NQA: "1.0", Ativo: true, UpdatedBy: actor}
	if err := am.SalvarProcedimento(ctx, proc); err != nil {
		t.Fatal(err)
	}
	var rota int64
	if err := pool.QueryRow(bg, `INSERT INTO receiving_inspection_routes(enterprise_code,basis,item_code,mask,inspection_warehouse_id,valid_from,is_active,created_by,sampling_procedure_id)
		VALUES($1,'ITEM',$2,'',$3,$4::date,TRUE,$5,$6) RETURNING id`, a.Code, item, insp, time.Now().AddDate(0, 0, -10), actor, proc.ID).Scan(&rota); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO receiving_inspection_route_steps(route_id,sequence,inspection_name,kind,appointment_mode,is_required,min_value,max_value)
		VALUES($1,1,'Diâmetro','VALUE','ALL_MEASUREMENTS',TRUE,7.9,8.1),($1,2,'Rosca','ATTRIBUTE','STATUS_ONLY',TRUE,NULL,NULL)`, rota)

	stock := stockpg.NewStockRepositorySQLC(pool)
	entrar := func(qtd string) {
		ref := stockentity.ReferenceTypePurchaseOrder
		if _, err := stock.CreateMovement(ctx, &stockentity.StockMovement{ItemCode: item, WarehouseID: insp, MovementType: stockentity.MovementTypeIn,
			Quantity: dd(qtd), UnitPrice: dd("2"), ReferenceType: &ref, CreatedBy: actor}); err != nil {
			t.Fatal(err)
		}
	}
	saldo := func(wh int64) decimal.Decimal {
		var q decimal.Decimal
		if err := pool.QueryRow(bg, `SELECT COALESCE(SUM(quantity),0) FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3`, a.ID, item, wh).Scan(&q); err != nil {
			t.Fatal(err)
		}
		return q
	}

	uc := &lote_uc.UseCase{Repo: lotepg.New(pool), Fontes: lotepg.NewFontes(pool), Amostragem: am, Auth: &auth.AuthService{}}

	// 1. Recebimento: o roteiro da empresa A dá o almoxarifado, as características e a amostragem.
	if wh, ok := uc.ResolveInspectionRoute(ctx, item, ""); !ok || wh != insp {
		t.Fatalf("roteiro da empresa: %d %v", wh, ok)
	}
	if _, ok := uc.ResolveInspectionRoute(b.Ctx, item, ""); ok {
		t.Fatal("B achou o roteiro de A")
	}
	entrar("1000")
	id, numero, err := uc.AbrirDoRecebimento(ctx, item, "", dd("1000"), insp, &fornecedor, nil, nil, sp("CORRIDA-77"), nil)
	if err != nil {
		t.Fatal(err)
	}
	l, err := uc.Obter(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if numero != 1 || len(l.Caracteristicas) != 2 || l.Plano == nil || l.Plano.N != 80 || l.Plano.Ac != 2 || *l.ChaveAmostragem != amostragem.ChaveFornecedor(fornecedor) || *l.Lote.Lote != "CORRIDA-77" {
		t.Fatalf("lote aberto: numero %d caracs %d plano %+v chave %v", numero, len(l.Caracteristicas), l.Plano, l.ChaveAmostragem)
	}
	if _, err := uc.Obter(b.Ctx, id); err == nil {
		t.Fatal("B leu o lote de A")
	}

	// 2. Bloqueio: o material em inspeção não sai à mão.
	if _, err := stock.TransferirEntreAlmoxarifados(ctx, stockentity.TransferenciaEstoque{ItemCode: item, DeAlmox: insp, ParaAlmox: alm, Quantidade: dd("10"), Ator: actor}); err == nil ||
		!strings.Contains(err.Error(), "decisão de uso") {
		t.Fatalf("transferência manual da inspeção deveria ser barrada: %v", err)
	}
	manual := stockentity.ReferenceTypeRequisition
	if _, err := stock.CreateMovement(ctx, &stockentity.StockMovement{ItemCode: item, WarehouseID: insp, MovementType: stockentity.MovementTypeOut,
		Quantity: dd("1"), ReferenceType: &manual, CreatedBy: actor}); err == nil {
		t.Fatal("requisição da inspeção deveria ser barrada")
	}

	// 3. Aceitar sem inspecionar é recusado.
	if _, err := uc.Decidir(ctx, id, lote_uc.DecisaoDTO{Aceita: dd("1000"), DestinoAceito: &alm}); err == nil || !strings.Contains(err.Error(), "não está completa") {
		t.Fatalf("aceitar sem resultados: %v", err)
	}

	// 4. Resultados: 80 unidades × 2 características; 3 diâmetros fora. A
	// tela dizer "conforme" não muda o que o limite diz.
	diam, rosca := l.Caracteristicas[0].ID, l.Caracteristicas[1].ID
	var rs []lote_uc.ResultadoDTO
	for i := 1; i <= 80; i++ {
		v := dd("8.0")
		if i <= 3 {
			v = dd("8.25")
		}
		rs = append(rs, lote_uc.ResultadoDTO{CaracteristicaID: diam, AmostraIndice: i, Valor: &v, Conforme: bp(true)},
			lote_uc.ResultadoDTO{CaracteristicaID: rosca, AmostraIndice: i, Conforme: bp(true)})
	}
	if _, err := uc.GravarResultados(ctx, id, []lote_uc.ResultadoDTO{{CaracteristicaID: diam, AmostraIndice: 81, Valor: &[]decimal.Decimal{dd("8")}[0]}}); err == nil {
		t.Fatal("unidade 81 numa amostra de 80 deveria ser recusada")
	}
	l, err = uc.GravarResultados(ctx, id, rs)
	if err != nil {
		t.Fatal(err)
	}
	if l.Avaliacao.NaoConformes != 3 || l.Avaliacao.AceitoPelaAmostragem || !l.Avaliacao.Completa {
		t.Fatalf("avaliação: %+v", l.Avaliacao)
	}

	// 5. Recusado pela amostragem: aceitar tudo exige concessão; separar 50.
	if _, err := uc.Decidir(ctx, id, lote_uc.DecisaoDTO{Aceita: dd("1000"), DestinoAceito: &alm}); err == nil || !strings.Contains(err.Error(), "CONCESSAO") {
		t.Fatalf("aceite sem concessão: %v", err)
	}
	l, err = uc.Decidir(ctx, id, lote_uc.DecisaoDTO{Aceita: dd("950"), Rejeitada: dd("50"), DestinoAceito: &alm, DestinoRejeitado: &rej,
		Tratamento: sp("selecionar"), Motivo: sp("triagem 100% do diâmetro")})
	if err != nil {
		t.Fatal(err)
	}
	if l.Status != dom.Decidido || l.Decisao.Tipo != dom.DecisaoParcial || !saldo(insp).IsZero() || !saldo(alm).Equal(dd("950")) || !saldo(rej).Equal(dd("50")) {
		t.Fatalf("decisão parcial: %+v | insp %s alm %s rej %s", l.Decisao, saldo(insp), saldo(alm), saldo(rej))
	}
	var ncs int
	if err := pool.QueryRow(bg, `SELECT COUNT(*) FROM non_conformances WHERE lot_id=$1 AND nonconform_qty=50`, id).Scan(&ncs); err != nil || ncs != 1 {
		t.Fatalf("NC do lote: %d %v", ncs, err)
	}
	est, err := am.Estado(ctx, proc.ID, item, amostragem.ChaveFornecedor(fornecedor))
	if err != nil || len(est.Ultimos) != 1 || est.Ultimos[0] {
		t.Fatalf("a severidade registra o lote recusado: %+v %v", est, err)
	}
	if _, err := pool.Exec(bg, `UPDATE quality_lot_decisions SET motivo='x' WHERE lot_id=$1`, id); err == nil {
		t.Fatal("a decisão de uso deveria ser imutável")
	}
	if _, err := uc.Decidir(ctx, id, lote_uc.DecisaoDTO{Rejeitada: dd("1000"), DestinoRejeitado: &rej, Motivo: sp("de novo")}); err == nil {
		t.Fatal("lote decidido não recebe outra decisão")
	}

	// 6. Rota antiga mostra o lote como ordem de inspeção.
	compat := &procurement_uc.UseCase{Repo: procurementpg.New(pool), Auth: &auth.AuthService{}, Lotes: uc}
	ord, err := compat.GetReceivingInspectionOrder(ctx, id)
	if err != nil || ord.Status != "PARTIAL" || ord.ApprovedQty != 950 || ord.RejectedQty != 50 || ord.WarehouseID != insp {
		t.Fatalf("ordem antiga: %+v %v", ord, err)
	}

	// 7. Duas decisões ao mesmo tempo: uma passa, o estoque move uma vez.
	entrar("20")
	id2, _, err := uc.AbrirDoRecebimento(ctx, item, "", dd("20"), insp, &fornecedor, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	erros := make(chan error, 2)
	for k := 0; k < 2; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := uc.Decidir(ctx, id2, lote_uc.DecisaoDTO{Rejeitada: dd("20"), DestinoRejeitado: &rej, Tratamento: sp("DEVOLVER"), Motivo: sp("avaria")})
			erros <- err
		}()
	}
	wg.Wait()
	close(erros)
	ok := 0
	for err := range erros {
		if err == nil {
			ok++
		}
	}
	if ok != 1 || !saldo(rej).Equal(dd("70")) || !saldo(insp).IsZero() {
		t.Fatalf("decisão concorrente: %d sucesso(s), rejeição %s, inspeção %s", ok, saldo(rej), saldo(insp))
	}

	// 8. Cancelar com o material na inspeção é recusado; depois do estorno da entrada, não.
	entrar("5")
	id3, _, err := uc.AbrirDoRecebimento(ctx, item, "", dd("5"), insp, &fornecedor, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Cancelar(ctx, id3, "nota cancelada"); err == nil {
		t.Fatal("cancelar com o material na inspeção deveria ser recusado")
	}
	estorno := stockentity.ReferenceTypeNFEntry
	if _, err := stock.CreateMovement(ctx, &stockentity.StockMovement{ItemCode: item, WarehouseID: insp, MovementType: stockentity.MovementTypeOut,
		Quantity: dd("5"), ReferenceType: &estorno, CreatedBy: actor}); err != nil {
		t.Fatalf("o estorno da entrada tira da inspeção: %v", err)
	}
	if c, err := uc.Cancelar(ctx, id3, "nota cancelada"); err != nil || c.Status != dom.Cancelado {
		t.Fatalf("cancelamento depois do estorno: %+v %v", c, err)
	}

	// 9. O MRP não conta a rejeição (70) nem o almoxarifado sem disposição.
	snap, err := mrppg.NewMRPCalculationRepositorySQLC(sqlc.New(pool), pool).GetStockSnapshot(ctx, item)
	if err != nil || snap == nil || !snap.Quantity.Equal(decimal.NewFromFloat(950)) {
		t.Fatalf("MRP: %+v %v (esperado 950: só o almoxarifado disponível)", snap, err)
	}
	if lista, err := uc.Listar(b.Ctx, dom.Filtro{}); err != nil || len(lista) != 0 {
		t.Fatalf("B listou lotes de A: %d %v", len(lista), err)
	}
	_ = ip
}
