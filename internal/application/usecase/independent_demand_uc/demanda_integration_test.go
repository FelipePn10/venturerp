//go:build integration

package independent_demand_uc_test

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/independent_demand_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/stock_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
	mrpentity "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	prodentity "github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	demandpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/independent_demand"
	itemrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	mrprepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation"
	politicapg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planning_policy"
	prodpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	stockpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// ofDireta abre a OF pelo repositório real, sem roteiro nem materiais: o
// teste é do vínculo (gravado na transação da OF), não da explosão da OF.
type ofDireta struct {
	pool  *pgxpool.Pool
	actor uuid.UUID
}

func (o ofDireta) Execute(ctx context.Context, dto request.CreateProductionOrderDTO) (*prodentity.ProductionOrder, error) {
	item, err := strconv.ParseInt(string(dto.ItemCode), 10, 64)
	if err != nil {
		return nil, err
	}
	var numero int64
	if err := o.pool.QueryRow(ctx, `SELECT COALESCE(MAX(order_number),0)+1 FROM production_orders`).Scan(&numero); err != nil {
		return nil, err
	}
	return prodpg.NewProductionOrderRepositoryPGX(o.pool).CreateWithMaterials(ctx, &prodentity.ProductionOrder{
		OrderNumber: numero, ItemCode: item, PlannedQty: dto.PlannedQty, Status: prodentity.StatusOpen, IsActive: true,
		CreatedBy: o.actor, Notes: dto.Notes, VinculoDemanda: dto.VinculoDemanda,
	}, nil)
}

func TestDemandaIndependenteDoCadastroAoAtendimento(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	code := testutil.UniqueCode() % 1_000_000_000
	var item, almox int64
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, q := range []string{
			`DELETE FROM politicas_planejamento WHERE enterprise_id = ANY($1)`,
			`DELETE FROM independent_demand_orders WHERE enterprise_id = ANY($1)`,
			`DELETE FROM production_orders WHERE enterprise_id = ANY($1)`,
			`DELETE FROM independent_demands WHERE enterprise_id = ANY($1)`,
			`DELETE FROM sales_order_items WHERE sales_order_code IN (SELECT code FROM sales_orders WHERE enterprise_code IN (SELECT code FROM enterprise WHERE id = ANY($1)))`,
			`DELETE FROM sales_orders WHERE enterprise_code IN (SELECT code FROM enterprise WHERE id = ANY($1))`,
			`DELETE FROM stock_movements WHERE enterprise_id = ANY($1)`,
			`DELETE FROM stock_balances WHERE enterprise_id = ANY($1)`,
			`DELETE FROM stock_movement_types WHERE enterprise_id = ANY($1)`,
			`DELETE FROM items WHERE enterprise_id = ANY($1)`,
			`DELETE FROM warehouse WHERE enterprise_id = ANY($1)`,
		} {
			_, _ = pool.Exec(bg, q, []int64{a.ID, b.ID})
		}
	})
	scalar := func(q string, args ...any) int64 {
		t.Helper()
		var v int64
		if err := pool.QueryRow(bg, q, args...).Scan(&v); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		return v
	}
	item = code
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,health,created_by,enterprise_id) VALUES($1::bigint,'DI-'||$1::bigint::text,0,'ATIVO',$2,$3)`, item, actor, a.ID)
	almox = scalar(`INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		VALUES($1,'Expedição',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`, strconv.FormatInt(code%100000, 10), actor, a.ID)
	testutil.Exec(t, pool, `INSERT INTO stock_movement_types(sigla,description,entry_exit,allows_manual,is_active,enterprise_id) VALUES('SDI','Saída para amostra','SAIDA',TRUE,TRUE,$1)`, a.ID)

	q := sqlc.New(pool)
	repo := demandpg.NewIndependentDemandRepositorySQLC(q).ComDB(pool)
	estoque := stockpg.NewStockRepositorySQLC(pool)
	saida := &stock_uc.CreateStockMovementUseCase{Repo: estoque, Auth: &auth.AuthService{}, Tipos: estoque}
	uc := &independent_demand_uc.DemandaUseCase{Repo: repo, Acomp: repo, Auth: &auth.AuthService{}, Items: itemrepo.NewRepositoryItemSQLC(q),
		Params: mrprepo.NewMRPCalculationRepositorySQLC(q, pool), Saida: saida,
		OF: ofDireta{pool: pool, actor: actor}, Politicas: politicapg.New(pool)}
	// A mesma transação do api.go: a saída de estoque do "Atender" grava
	// junto da trava da demanda.
	uc.Transacao = func(ctx context.Context, fn func(independent_demand_uc.Repos) error) error {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		r := demandpg.NewIndependentDemandRepositorySQLC(q.WithTx(tx)).ComDB(tx)
		estoqueTx := stockpg.NewStockRepositoryTx(tx)
		if err := fn(independent_demand_uc.Repos{Demandas: r, Acomp: r,
			Saida: &stock_uc.CreateStockMovementUseCase{Repo: estoqueTx, Auth: &auth.AuthService{}, Tipos: estoqueTx}}); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}

	hoje := time.Now()
	dia := func(n int) string { return hoje.AddDate(0, 0, n).Format("2006-01-02") }
	itemTxt := request.TextCode("DI-" + strconv.FormatInt(item, 10))

	// Validação: quantidade, projeto e cliente de outra empresa.
	if _, err := uc.Criar(a.Ctx, request.CreateIndependentDemandDTO{ItemCode: itemTxt, Quantity: decimal.Zero, DemandDate: dia(5)}); err == nil {
		t.Fatal("quantidade zero")
	}
	if _, err := uc.Criar(a.Ctx, request.CreateIndependentDemandDTO{ItemCode: itemTxt, Quantity: decimal.NewFromInt(1), DemandDate: dia(5), Motivo: entity.MotivoProjeto}); err == nil {
		t.Fatal("projeto sem a obra")
	}
	centro := int64(987654321)
	if _, err := uc.Criar(a.Ctx, request.CreateIndependentDemandDTO{ItemCode: itemTxt, Quantity: decimal.NewFromInt(1), DemandDate: dia(5), CostCenterCode: &centro}); err == nil || !strings.Contains(err.Error(), "centro de custo") {
		t.Fatalf("centro de custo inexistente: %v", err)
	}
	cliente := int64(987654321)
	if _, err := uc.Criar(a.Ctx, request.CreateIndependentDemandDTO{ItemCode: itemTxt, Quantity: decimal.NewFromInt(1), DemandDate: dia(5), CustomerCode: &cliente}); err == nil {
		t.Fatal("cliente inexistente")
	}

	// Almoxarifado de outra empresa não serve de destino.
	almoxB := scalar(`INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		VALUES($1,'Outra empresa',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`, strconv.FormatInt(code%100000+1, 10), actor, b.ID)
	if _, err := uc.Criar(a.Ctx, request.CreateIndependentDemandDTO{ItemCode: itemTxt, Quantity: decimal.NewFromInt(1), DemandDate: dia(5), WarehouseID: &almoxB}); err == nil || !strings.Contains(err.Error(), "almoxarifado") {
		t.Fatalf("almoxarifado de outra empresa: %v", err)
	}

	// Estoque de 10 (numeração automática → 1) e amostra de 9 em 3 entregas.
	est, err := uc.Criar(a.Ctx, request.CreateIndependentDemandDTO{ItemCode: itemTxt, Quantity: decimal.NewFromInt(10), DemandDate: dia(5), WarehouseID: &almox})
	if err != nil || len(est) != 1 || est[0].CodeDemand != 1 || est[0].Familia != "VENDA" {
		t.Fatalf("demanda de estoque: %+v %v", est, err)
	}
	amo, err := uc.Criar(a.Ctx, request.CreateIndependentDemandDTO{ItemCode: itemTxt, Quantity: decimal.NewFromInt(9), DemandDate: dia(5),
		Motivo: entity.MotivoAmostra, Entregas: 3, IntervaloDias: 7})
	if err != nil || len(amo) != 3 || amo[0].CodeDemand != 2 || amo[2].CodeDemand != 4 || amo[0].Programacao == nil ||
		!amo[1].Quantity.Equal(decimal.NewFromInt(3)) || amo[2].DemandDate.Format("2006-01-02") != dia(19) {
		t.Fatalf("programação: %+v %v", amo, err)
	}
	// A outra empresa numera do 1 também, e não vê as demandas da primeira.
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,health,created_by,enterprise_id) VALUES($1::bigint + 1,'DI-'||$1::bigint::text,0,'ATIVO',$2,$3)`, item, actor, b.ID)
	if outra, err := uc.Criar(b.Ctx, request.CreateIndependentDemandDTO{ItemCode: itemTxt, Quantity: decimal.NewFromInt(1), DemandDate: dia(5)}); err != nil || outra[0].CodeDemand != 1 {
		t.Fatalf("numeração por empresa: %+v %v", outra, err)
	}
	if lista, err := uc.Buscar(b.Ctx, entity.Filtro{}, ""); err != nil || len(lista) != 1 {
		t.Fatalf("isolamento: %d %v", len(lista), err)
	}

	if d1, err := uc.Detalhe(a.Ctx, 1); err != nil || d1.WarehouseNome == "" || !strings.Contains(d1.WarehouseNome, "Expedição") {
		t.Fatalf("o destino sai com o nome do almoxarifado: %+v %v", d1, err)
	}

	// Rascunho e orçamento não são venda: não consomem a demanda.
	rasc := scalar(`INSERT INTO sales_orders(order_number,enterprise_code,status,created_by,delivery_date) VALUES($1,$2,'R',$3,$4::date) RETURNING code`, code%100000+1, a.Code, actor, dia(6))
	testutil.Exec(t, pool, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,delivery_date) VALUES($1,1,$2,7,$3::date)`, rasc, item, dia(6))
	orc := scalar(`INSERT INTO sales_orders(order_number,enterprise_code,status,created_by,delivery_date) VALUES($1,$2,'OF',$3,$4::date) RETURNING code`, code%100000+2, a.Code, actor, dia(6))
	testutil.Exec(t, pool, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,delivery_date) VALUES($1,1,$2,7,$3::date)`, orc, item, dia(6))
	if d1, err := uc.Detalhe(a.Ctx, 1); err != nil || !d1.Consumida.IsZero() {
		t.Fatalf("rascunho e orçamento não consomem: %+v %v", d1, err)
	}

	// Venda de 4 do item em +6 dias consome a demanda de estoque (+5).
	ped := scalar(`INSERT INTO sales_orders(order_number,enterprise_code,status,created_by,delivery_date) VALUES($1,$2,'P',$3,$4::date) RETURNING code`, code%100000, a.Code, actor, dia(6))
	testutil.Exec(t, pool, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,delivery_date) VALUES($1,1,$2,4,$3::date)`, ped, item, dia(6))
	d1, err := uc.Detalhe(a.Ctx, 1)
	if err != nil || !d1.Consumida.Equal(decimal.NewFromInt(4)) || !d1.APlanejar.Equal(decimal.NewFromInt(6)) || d1.Posicao != entity.PosicaoParcial {
		t.Fatalf("estoque consumido pela venda: %+v %v", d1, err)
	}
	// Rastreio: a demanda diz que pedido a consumiu.
	if len(d1.Consumo) != 1 || d1.Consumo[0].Pedido != code%100000 || d1.Consumo[0].Sequencia != 1 || !d1.Consumo[0].Quantidade.Equal(decimal.NewFromInt(4)) {
		t.Fatalf("rastreio do consumo: %+v", d1.Consumo)
	}
	vis, err := uc.ConsumoDoItem(a.Ctx, itemTxt)
	if err != nil || len(vis.Demandas) != 1 || len(vis.Vendas) != 1 || !vis.TotalExpectativa.Equal(decimal.NewFromInt(10)) ||
		!vis.TotalConsumido.Equal(decimal.NewFromInt(4)) || !vis.TotalAPlanejar.Equal(decimal.NewFromInt(6)) ||
		!vis.Vendas[0].Consumiu.Equal(decimal.NewFromInt(4)) || !vis.Vendas[0].Acima.IsZero() || vis.SobEncomenda {
		t.Fatalf("visão do consumo do item: %+v %v", vis, err)
	}

	// Janela do item só para a frente (0 antes): a venda de +6 já não consome
	// a demanda de +5.
	pol := politicapg.New(pool)
	zeroDias, quinze := 0, 15
	if _, err := pol.Criar(a.Ctx, &mrpentity.PoliticaPlanejamento{ItemCode: &item, ConsumoDiasAntes: &zeroDias, ConsumoDiasDepois: &quinze, CreatedBy: actor}); err != nil {
		t.Fatal(err)
	}
	if d1, err = uc.Detalhe(a.Ctx, 1); err != nil || !d1.Consumida.IsZero() || !d1.APlanejar.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("janela do item: %+v %v", d1, err)
	}
	// Sob encomenda: a demanda de estoque sai do cálculo e não se cadastra
	// outra; a de finalidade própria continua.
	testutil.Exec(t, pool, `UPDATE politicas_planejamento SET estrategia='SOB_ENCOMENDA' WHERE enterprise_id=$1 AND item_code=$2`, a.ID, item)
	if d1, err = uc.Detalhe(a.Ctx, 1); err != nil || d1.Posicao != entity.PosicaoSobEncomenda || !d1.APlanejar.IsZero() {
		t.Fatalf("sob encomenda: %+v %v", d1, err)
	}
	if _, err := uc.Criar(a.Ctx, request.CreateIndependentDemandDTO{ItemCode: itemTxt, Quantity: decimal.NewFromInt(1), DemandDate: dia(5), Motivo: entity.MotivoReposicao}); err == nil || !strings.Contains(err.Error(), "sob encomenda") {
		t.Fatalf("reposição de item sob encomenda: %v", err)
	}
	if d2, err := uc.Detalhe(a.Ctx, 2); err != nil || !d2.APlanejar.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("amostra de item sob encomenda continua: %+v %v", d2, err)
	}
	if vis, err = uc.ConsumoDoItem(a.Ctx, itemTxt); err != nil || !vis.SobEncomenda || !vis.Vendas[0].Acima.Equal(decimal.NewFromInt(4)) {
		t.Fatalf("visão sob encomenda: %+v %v", vis, err)
	}
	testutil.Exec(t, pool, `DELETE FROM politicas_planejamento WHERE enterprise_id=$1`, a.ID)
	if d2, _ := uc.Detalhe(a.Ctx, 2); !d2.Consumida.IsZero() {
		t.Fatal("amostra não é consumida pela venda")
	}

	// Atender: só a de finalidade própria, até o saldo, com saída de estoque.
	if _, err := saida.Execute(a.Ctx, request.CreateStockMovementDTO{ItemCode: item, WarehouseID: almox, MovementType: "IN", Quantity: decimal.NewFromInt(10)}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Atender(a.Ctx, 1, request.AtenderDemandaDTO{Quantidade: decimal.NewFromInt(1), WarehouseID: almox, MovementTypeCode: "SDI"}); err == nil {
		t.Fatal("demanda de estoque não se atende por saída")
	}
	if _, err := uc.Atender(a.Ctx, 2, request.AtenderDemandaDTO{Quantidade: decimal.NewFromInt(4), WarehouseID: almox, MovementTypeCode: "SDI"}); err == nil {
		t.Fatal("acima do saldo")
	}
	d2, err := uc.Atender(a.Ctx, 2, request.AtenderDemandaDTO{Quantidade: decimal.NewFromInt(2), WarehouseID: almox, MovementTypeCode: "SDI"})
	if err != nil || !d2.Atendida.Equal(decimal.NewFromInt(2)) || !d2.APlanejar.Equal(decimal.NewFromInt(1)) || d2.Posicao != entity.PosicaoParcial {
		t.Fatalf("amostra atendida em parte: %+v %v", d2, err)
	}
	if saldo := scalar(`SELECT quantity::bigint FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2`, a.ID, item); saldo != 8 {
		t.Fatalf("a saída baixou o estoque: %d", saldo)
	}
	// "Atender" com a demanda travada por outra transação espera; quando a
	// outra grava a saída do saldo restante (1) e confirma, ele relê o saldo
	// e recusa, em vez de passar do saldo.
	trava, err := pool.Begin(bg)
	if err != nil {
		t.Fatal(err)
	}
	defer trava.Rollback(bg)
	if _, err := trava.Exec(bg, `SELECT 1 FROM independent_demands WHERE enterprise_id=$1 AND code=2 FOR UPDATE`, a.ID); err != nil {
		t.Fatal(err)
	}
	resultado := make(chan error, 1)
	go func() {
		_, err := uc.Atender(a.Ctx, 2, request.AtenderDemandaDTO{Quantidade: decimal.NewFromInt(1), WarehouseID: almox, MovementTypeCode: "SDI"})
		resultado <- err
	}()
	select {
	case err := <-resultado:
		t.Fatalf("o Atender não esperou a trava da demanda: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
	if err := stockpg.CreateMovementTx(a.Ctx, trava, a.ID, &stockentity.StockMovement{ItemCode: item, WarehouseID: almox, MovementType: stockentity.MovementTypeOut,
		Quantity: decimal.NewFromInt(1), ReferenceType: ptr(entity.ReferenciaAtendimento), ReferenceCode: ptr(int64(2)), CreatedBy: actor}); err != nil {
		t.Fatal(err)
	}
	if err := trava.Commit(bg); err != nil {
		t.Fatal(err)
	}
	if err := <-resultado; err == nil || !strings.Contains(err.Error(), "saldo") {
		t.Fatalf("depois da trava, o saldo relido já é zero: %v", err)
	}

	// Estornada a saída na Central de Estoque, a demanda volta a ter o saldo.
	d3, err := uc.Atender(a.Ctx, 3, request.AtenderDemandaDTO{Quantidade: decimal.NewFromInt(1), WarehouseID: almox, MovementTypeCode: "SDI"})
	if err != nil || !d3.Atendida.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("atender a 3: %+v %v", d3, err)
	}
	mov := scalar(`SELECT id FROM stock_movements WHERE enterprise_id=$1 AND reference_type='DEMANDA_INDEPENDENTE' AND reference_code=3`, a.ID)
	if _, err := estoque.EstornarMovimento(a.Ctx, mov, actor, "lançada por engano"); err != nil {
		t.Fatal(err)
	}
	if d3, _ = uc.Detalhe(a.Ctx, 3); !d3.Atendida.IsZero() || !d3.APlanejar.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("depois do estorno: %+v", d3)
	}

	// Cancelar exige que nada tenha saído; com saída, encerra o saldo.
	if err := uc.Cancelar(a.Ctx, 2, "não vai mais"); err == nil {
		t.Fatal("cancelar com saída")
	}
	if err := uc.Encerrar(a.Ctx, 2, ""); err == nil {
		t.Fatal("encerrar sem motivo")
	}
	if err := uc.Encerrar(a.Ctx, 2, "cliente desistiu da terceira peça"); err != nil {
		t.Fatal(err)
	}
	if d2, _ = uc.Detalhe(a.Ctx, 2); d2.Situacao != entity.SituacaoEncerrada || !d2.APlanejar.IsZero() || d2.IsActive {
		t.Fatalf("encerrada (e fora de is_active): %+v", d2)
	}
	// Sem situação, a lista traz só as abertas (o desktop até a 1.8.0 não
	// filtra); TODAS é pedido pela tela nova no handler.
	if lista, err := uc.Buscar(a.Ctx, entity.Filtro{Situacao: entity.SituacaoAberta}, ""); err != nil || len(lista) != 3 {
		t.Fatalf("abertas: %d %v", len(lista), err)
	}
	if _, err := uc.Alterar(a.Ctx, request.UpdateIndependentDemandDTO{CodeDemand: 2, ItemCode: itemTxt, Quantity: decimal.NewFromInt(5), DemandDate: dia(5), Motivo: entity.MotivoAmostra}); err == nil {
		t.Fatal("encerrada não se altera")
	}
	if d3, err = uc.Alterar(a.Ctx, request.UpdateIndependentDemandDTO{CodeDemand: 3, ItemCode: itemTxt, Quantity: decimal.NewFromInt(5), DemandDate: dia(12), Motivo: entity.MotivoAmostra, Prioridade: 1, WarehouseID: &almox}); err != nil ||
		!d3.Quantity.Equal(decimal.NewFromInt(5)) || d3.Prioridade != 1 {
		t.Fatalf("alterar: %+v %v", d3, err)
	}
	// A alteração do desktop antigo (sem motivo, destino, prioridade) mantém
	// o que já estava gravado.
	if d3, err = uc.Alterar(a.Ctx, request.UpdateIndependentDemandDTO{CodeDemand: 3, ItemCode: itemTxt, Quantity: decimal.NewFromInt(6), DemandDate: dia(12)}); err != nil ||
		d3.Motivo != entity.MotivoAmostra || d3.Prioridade != 1 || d3.WarehouseID == nil || *d3.WarehouseID != almox || !d3.Quantity.Equal(decimal.NewFromInt(6)) {
		t.Fatalf("alteração parcial: %+v %v", d3, err)
	}
	// Zero limpa o destino.
	zero := int64(0)
	if d3, err = uc.Alterar(a.Ctx, request.UpdateIndependentDemandDTO{CodeDemand: 3, ItemCode: itemTxt, Quantity: decimal.NewFromInt(5), DemandDate: dia(12), WarehouseID: &zero}); err != nil || d3.WarehouseID != nil {
		t.Fatalf("limpar o destino: %+v %v", d3, err)
	}

	// O MRP lê só as abertas, com o atendido.
	abertas, err := repo.List(a.Ctx)
	if err != nil || len(abertas) != 3 {
		t.Fatalf("abertas para o MRP: %d %v", len(abertas), err)
	}

	// Gerar OF agora: vincula a OF à demanda, na transação da OF.
	d4, err := uc.GerarOF(a.Ctx, 4, request.GerarOFDaDemandaDTO{})
	if err != nil || len(d4.Ordens) != 1 || d4.Ordens[0].ProductionOrderID == nil || d4.Ordens[0].OrderNumber == nil ||
		!d4.Ordens[0].Quantidade.Equal(decimal.NewFromInt(3)) || !d4.Coberta.Equal(decimal.NewFromInt(3)) || !d4.SemOrdem.IsZero() {
		t.Fatalf("OF da demanda: %+v %v", d4, err)
	}
	// OF cancelada não cobre: a demanda volta a oferecer "Gerar OF agora", e
	// a nova OF passa pela contagem com o vínculo cancelado no meio.
	if _, err := prodpg.NewProductionOrderRepositoryPGX(pool).Cancel(a.Ctx, *d4.Ordens[0].ProductionOrderID); err != nil {
		t.Fatal(err)
	}
	if d4, err = uc.Detalhe(a.Ctx, 4); err != nil || !d4.Coberta.IsZero() || !d4.SemOrdem.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("OF cancelada ainda cobre: %+v %v", d4, err)
	}
	if d4, err = uc.GerarOF(a.Ctx, 4, request.GerarOFDaDemandaDTO{}); err != nil || len(d4.Ordens) != 2 || !d4.Coberta.Equal(decimal.NewFromInt(3)) || !d4.SemOrdem.IsZero() {
		t.Fatalf("OF depois da cancelada: %+v %v", d4, err)
	}
	// Ordem planejada ligada cobre; excluída (sem OF), deixa de cobrir.
	po := scalar(`INSERT INTO planned_orders(order_number,item_code,quantity,order_type,demand_type,need_date,created_by,demand_code,enterprise_id)
		VALUES($1,$2,2,'PRODUCTION','INDEPENDENT',$3::date,$4,3,$5) RETURNING id`, code%100000+9, item, dia(12), actor, a.ID)
	testutil.Exec(t, pool, `INSERT INTO independent_demand_orders(enterprise_id,demand_id,planned_order_id,quantity,created_by)
		SELECT $1, id, $2, 2, $3 FROM independent_demands WHERE enterprise_id=$1 AND code=3`, a.ID, po, actor)
	if d3, err = uc.Detalhe(a.Ctx, 3); err != nil || !d3.Coberta.Equal(decimal.NewFromInt(2)) {
		t.Fatalf("ordem planejada ligada: %+v %v", d3, err)
	}
	testutil.Exec(t, pool, `UPDATE planned_orders SET is_active=FALSE WHERE id=$1`, po)
	if d3, err = uc.Detalhe(a.Ctx, 3); err != nil || !d3.Coberta.IsZero() || !d3.SemOrdem.Equal(d3.APlanejar) {
		t.Fatalf("ordem planejada excluída ainda cobre: %+v %v", d3, err)
	}
	// A sugestão firmada liga a ordem planejada às demandas pelo pegging; a
	// nova tentativa (firmar de novo) não duplica o vínculo.
	sug := code%100000 + 77
	po2 := scalar(`INSERT INTO planned_orders(order_number,item_code,quantity,order_type,demand_type,need_date,created_by,demand_code,enterprise_id)
		VALUES($1,$2,2,'PRODUCTION','INDEPENDENT',$3::date,$4,3,$5) RETURNING code`, code%100000+10, item, dia(12), actor, a.ID)
	testutil.Exec(t, pool, `INSERT INTO mrp_pegging(enterprise_id,plan_code,item_code,demand_kind,demand_code,need_date,supply_kind,supply_code,quantity)
		VALUES($1,0,$2,'DEMANDA_INDEPENDENTE',3,$3::date,'SUGESTAO',$4,2)`, a.ID, item, dia(12), sug)
	for range 2 {
		if err := repo.VincularSugestao(a.Ctx, sug, po2, actor); err != nil {
			t.Fatal(err)
		}
	}
	if n := scalar(`SELECT COUNT(*) FROM independent_demand_orders v JOIN planned_orders p ON p.id=v.planned_order_id WHERE p.code=$1 AND v.enterprise_id=$2`, po2, a.ID); n != 1 {
		t.Fatalf("vínculos da sugestão firmada duas vezes: %d", n)
	}
	if d3, err = uc.Detalhe(a.Ctx, 3); err != nil || !d3.Coberta.Equal(decimal.NewFromInt(2)) {
		t.Fatalf("cobertura pela sugestão firmada: %+v %v", d3, err)
	}
	testutil.Exec(t, pool, `UPDATE planned_orders SET is_active=FALSE WHERE code=$1 AND enterprise_id=$2`, po2, a.ID)
	// Item comprado não tem OF: a recusa diz o motivo e nenhuma OF nasce.
	comprado := item + 7
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,engineering_type,health,created_by,enterprise_id) VALUES($1::bigint,'DIC-'||$1::bigint::text,0,1,'ATIVO',$2,$3)`, comprado, actor, a.ID)
	dc, err := uc.Criar(a.Ctx, request.CreateIndependentDemandDTO{ItemCode: request.TextCode("DIC-" + strconv.FormatInt(comprado, 10)), Quantity: decimal.NewFromInt(3), DemandDate: dia(5), Motivo: entity.MotivoReposicao})
	if err != nil {
		t.Fatal(err)
	}
	antesComprado := scalar(`SELECT COUNT(*) FROM production_orders WHERE enterprise_id=$1`, a.ID)
	if _, err := uc.GerarOF(a.Ctx, dc[0].CodeDemand, request.GerarOFDaDemandaDTO{}); err == nil || !strings.Contains(err.Error(), "comprado") {
		t.Fatalf("OF de item comprado: %v", err)
	}
	if depois := scalar(`SELECT COUNT(*) FROM production_orders WHERE enterprise_id=$1`, a.ID); depois != antesComprado {
		t.Fatal("a OF recusada do item comprado não fica")
	}
	if err := uc.Cancelar(a.Ctx, dc[0].CodeDemand, "teste"); err != nil {
		t.Fatal(err)
	}

	// Coberta por inteiro: o segundo clique não abre outra OF.
	ofsAntes := scalar(`SELECT COUNT(*) FROM production_orders WHERE enterprise_id=$1`, a.ID)
	if _, err := uc.GerarOF(a.Ctx, 4, request.GerarOFDaDemandaDTO{}); err == nil {
		t.Fatal("segunda OF da demanda já coberta")
	}
	// Lote acima do saldo sem ordem vale, mas o vínculo cobre só o sem ordem.
	d1, err = uc.GerarOF(a.Ctx, 1, request.GerarOFDaDemandaDTO{Quantidade: decimal.NewFromInt(50)})
	if err != nil || len(d1.Ordens) != 1 || !d1.Ordens[0].Quantidade.Equal(decimal.NewFromInt(6)) || !d1.Ordens[0].Planejada.Equal(decimal.NewFromInt(50)) {
		t.Fatalf("OF com lote: %+v %v", d1, err)
	}
	// Vínculo com contagem velha (outra ordem ligada no meio) é recusado e
	// a OF não nasce.
	if _, err := (ofDireta{pool: pool, actor: actor}).Execute(a.Ctx, request.CreateProductionOrderDTO{ItemCode: request.TextCode(strconv.FormatInt(item, 10)),
		PlannedQty: decimal.NewFromInt(1), VinculoDemanda: &prodentity.VinculoDemanda{DemandID: scalar(`SELECT id FROM independent_demands WHERE enterprise_id=$1 AND code=4`, a.ID), Quantidade: decimal.NewFromInt(1), OrdensVistas: 0}}); err == nil {
		t.Fatal("vínculo com contagem velha")
	}
	if depois := scalar(`SELECT COUNT(*) FROM production_orders WHERE enterprise_id=$1`, a.ID); depois != ofsAntes+1 {
		t.Fatalf("a OF recusada não fica: antes %d (+1 do lote), depois %d", ofsAntes, depois)
	}

	// Lista filtrada por motivo e situação.
	if lista, err := uc.Buscar(a.Ctx, entity.Filtro{Motivo: entity.MotivoAmostra, Situacao: entity.SituacaoAberta}, string(itemTxt)); err != nil || len(lista) != 2 {
		t.Fatalf("filtro: %d %v", len(lista), err)
	}
}

func ptr[T any](v T) *T { return &v }
