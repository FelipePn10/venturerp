//go:build integration

package routing_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
	routingrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// cenário: um item com um centro de trabalho e uma operação de biblioteca.
type cenario struct {
	ctx      context.Context
	pool     *pgxpool.Pool
	repo     domainrepo.RoutingRepository
	sqlc     *routingrepo.RoutingRepositorySQLC
	ator     uuid.UUID
	empresa  int64
	item     int64
	centro   int64
	operacao int64
}

func novoCenario(t *testing.T) *cenario {
	t.Helper()
	q, pool := testutil.Queries(t)
	ctx := testutil.TenantContext(t, pool)
	c := &cenario{ctx: ctx, pool: pool, ator: testutil.Actor(t, pool), empresa: testutil.EnterpriseID(t, ctx)}
	c.repo = routingrepo.New(q)
	c.sqlc = c.repo.(*routingrepo.RoutingRepositorySQLC)
	c.item = testutil.UniqueCode()
	testutil.SeedItem(t, pool, ctx, c.item, c.ator)
	if err := pool.QueryRow(ctx, `INSERT INTO machine_types (code, name, type, enterprise_id, created_by)
		VALUES ($1, 'CT teste roteiro', 'CUT', $2, $3) RETURNING id`, testutil.UniqueCode(), c.empresa, c.ator).Scan(&c.centro); err != nil {
		t.Fatalf("centro: %v", err)
	}
	op, err := entity.NewOperation(testutil.UniqueCode(), "Op teste roteiro", nil, entity.OriginInternal, &c.centro, 0.5, 0.25, c.ator)
	if err != nil {
		t.Fatal(err)
	}
	criada, err := c.repo.CreateOperation(ctx, op)
	if err != nil {
		t.Fatalf("operação: %v", err)
	}
	c.operacao = criada.ID
	t.Cleanup(func() {
		bg := context.Background()
		pool.Exec(bg, `DELETE FROM inspection_plan_characteristics WHERE plan_id IN (SELECT id FROM inspection_plans WHERE item_code = $1)`, c.item)
		pool.Exec(bg, `DELETE FROM inspection_plans WHERE item_code = $1`, c.item)
		pool.Exec(bg, `DELETE FROM operation_documents WHERE route_operation_id IN (SELECT ro.id FROM route_operations ro JOIN manufacturing_routes mr ON mr.id = ro.route_id WHERE mr.item_code = $1 OR (mr.is_template AND mr.created_by = $2))`, c.item, c.ator)
		pool.Exec(bg, `DELETE FROM route_operation_network WHERE predecessor_id IN (SELECT ro.id FROM route_operations ro JOIN manufacturing_routes mr ON mr.id = ro.route_id WHERE mr.item_code = $1 OR mr.description LIKE 'Modelo teste %')`, c.item)
		pool.Exec(bg, `DELETE FROM route_operation_resources WHERE route_operation_id IN (SELECT ro.id FROM route_operations ro JOIN manufacturing_routes mr ON mr.id = ro.route_id WHERE mr.item_code = $1 OR mr.description LIKE 'Modelo teste %')`, c.item)
		pool.Exec(bg, `DELETE FROM manufacturing_routes WHERE item_code = $1 OR description LIKE 'Modelo teste %'`, c.item)
		pool.Exec(bg, `DELETE FROM operations WHERE id = $1`, c.operacao)
		pool.Exec(bg, `DELETE FROM machine_types WHERE id = $1`, c.centro)
		pool.Exec(bg, `DELETE FROM items WHERE code = $1`, c.item)
	})
	return c
}

func (c *cenario) roteiro(t *testing.T, mascara *string, alt int16, padrao bool, lotMin, lotMax *decimal.Decimal) *entity.ManufacturingRoute {
	t.Helper()
	desc := "Roteiro teste"
	rt, err := entity.NewManufacturingRoute(testutil.UniqueCode(), c.item, mascara, alt, &desc, padrao, nil, nil, c.ator)
	if err != nil {
		t.Fatal(err)
	}
	rt.LotMin, rt.LotMax = lotMin, lotMax
	criado, err := c.repo.CreateRoute(c.ctx, rt)
	if err != nil {
		t.Fatalf("roteiro alt %d: %v", alt, err)
	}
	return criado
}

func (c *cenario) etapa(t *testing.T, rota int64, seq int16) *entity.RouteOperation {
	t.Helper()
	ro, err := entity.NewRouteOperation(rota, seq, c.operacao, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	criada, err := c.repo.AddRouteOperation(c.ctx, ro)
	if err != nil {
		t.Fatalf("etapa %d: %v", seq, err)
	}
	return criada
}

func dec(v int64) *decimal.Decimal { d := decimal.NewFromInt(v); return &d }
func s(v string) *string           { return &v }

// OP da máscara "AZUL" de um item que só tem roteiro genérico: antes não
// achava roteiro e a OP era recusada.
func TestRoteiroGenericoServeParaQualquerMascara(t *testing.T) {
	c := novoCenario(t)
	generico := c.roteiro(t, nil, 1, true, nil, nil)
	got, err := c.repo.GetRouteForItem(c.ctx, c.item, "AZUL")
	if err != nil || got.ID != generico.ID {
		t.Fatalf("máscara sem roteiro próprio usa o genérico: veio %+v, erro %v", got, err)
	}
	// Com roteiro da máscara, ele vence o genérico; roteiro de OUTRA máscara nunca serve.
	azul := c.roteiro(t, s("AZUL"), 2, true, nil, nil)
	c.roteiro(t, s("VERDE"), 3, true, nil, nil)
	if got, _ := c.repo.GetRouteForItem(c.ctx, c.item, "AZUL"); got.ID != azul.ID {
		t.Fatalf("roteiro da máscara vence o genérico; veio alt %d", got.Alternative)
	}
	if got, _ := c.repo.GetRouteForItem(c.ctx, c.item, "PRETO"); got.ID != generico.ID {
		t.Fatalf("máscara sem roteiro cai no genérico, não no de outra cor; veio alt %d", got.Alternative)
	}
}

// Laser para lote grande, guilhotina para lote pequeno.
func TestFaixaDeLoteEscolheOAlternativo(t *testing.T) {
	c := novoCenario(t)
	guilhotina := c.roteiro(t, nil, 1, true, nil, nil)
	laser := c.roteiro(t, nil, 2, false, dec(50), nil)
	if got, _ := c.sqlc.GetRouteForItemQty(c.ctx, c.item, "", dec(500)); got.ID != laser.ID {
		t.Fatalf("500 peças cabem na faixa do laser, que vence o padrão sem faixa; veio alt %d", got.Alternative)
	}
	if got, _ := c.sqlc.GetRouteForItemQty(c.ctx, c.item, "", dec(10)); got.ID != guilhotina.ID {
		t.Fatalf("10 peças ficam fora da faixa do laser; veio alt %d", got.Alternative)
	}
	if got, _ := c.repo.GetRouteForItem(c.ctx, c.item, ""); got.ID != guilhotina.ID {
		t.Fatalf("sem quantidade, vale o padrão; veio alt %d", got.Alternative)
	}
}

// O modelo nunca é escolhido para planejar, e listar modelos não traz roteiro de item.
func TestModeloNaoEntraNoPlanejamento(t *testing.T) {
	c := novoCenario(t)
	modelo, err := entity.NewRouteTemplate(testutil.UniqueCode(), "Modelo teste chapa dobrada", c.ator)
	if err != nil {
		t.Fatal(err)
	}
	criado, err := c.repo.CreateRoute(c.ctx, modelo)
	if err != nil {
		t.Fatalf("modelo: %v", err)
	}
	if criado.ItemCode != 0 || !criado.IsTemplate {
		t.Fatalf("modelo sem item: %+v", criado)
	}
	modelos, err := c.repo.ListRouteTemplates(c.ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	achou := false
	for _, m := range modelos {
		achou = achou || m.ID == criado.ID
		if !m.IsTemplate {
			t.Fatalf("lista de modelos trouxe roteiro de item: %+v", m)
		}
	}
	if !achou {
		t.Fatal("o modelo criado não aparece na lista")
	}
}

// Remover a etapa do meio liga as vizinhas; a sequência removida pode ser reusada.
func TestRemoverEtapaNaoPartaARedeNemTravaASequencia(t *testing.T) {
	c := novoCenario(t)
	rt := c.roteiro(t, nil, 1, true, nil, nil)
	a, b, d := c.etapa(t, rt.ID, 10), c.etapa(t, rt.ID, 20), c.etapa(t, rt.ID, 30)
	for _, e := range []*entity.NetworkEdge{{PredecessorID: a.ID, SuccessorID: b.ID}, {PredecessorID: b.ID, SuccessorID: d.ID}} {
		if _, err := c.repo.SetNetworkEdge(c.ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.repo.RemoveRouteOperation(c.ctx, b.ID); err != nil {
		t.Fatal(err)
	}
	edges, err := c.repo.GetNetworkEdges(c.ctx, rt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 1 || edges[0].PredecessorID != a.ID || edges[0].SuccessorID != d.ID {
		t.Fatalf("A→B→C sem B vira A→C; veio %+v", edges)
	}
	ops, _ := c.repo.GetRouteOperations(c.ctx, rt.ID)
	if entity.CriticalPath(ops, edges, 1).HasCycle() {
		t.Fatal("remover etapa não pode fazer o roteiro parecer cíclico")
	}
	c.etapa(t, rt.ID, 20) // antes: duplicate key route_operations_route_id_sequence_key
}

// Dois padrões com vigência cruzada não convivem; em janelas diferentes, sim.
func TestUmPadraoPorPeriodo(t *testing.T) {
	c := novoCenario(t)
	primeiro := c.roteiro(t, nil, 1, true, nil, nil)
	c.roteiro(t, nil, 2, true, nil, nil)
	got, err := c.repo.GetRouteByID(c.ctx, primeiro.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.IsStandard {
		t.Fatal("o novo padrão sem vigência cruza com o antigo: o antigo deixa de ser padrão")
	}
}

// Mudar o centro da etapa leva junto o recurso principal.
func TestCentroDaEtapaEORecursoPrincipalAndamJuntos(t *testing.T) {
	c := novoCenario(t)
	rt := c.roteiro(t, nil, 1, true, nil, nil)
	etapa := c.etapa(t, rt.ID, 10)
	var outro int64
	if err := c.pool.QueryRow(c.ctx, `INSERT INTO machine_types (code, name, type, enterprise_id, created_by)
		VALUES ($1, 'CT alternativo', 'CUT', $2, $3) RETURNING id`, testutil.UniqueCode(), c.empresa, c.ator).Scan(&outro); err != nil {
		t.Fatal(err)
	}
	defer c.pool.Exec(context.Background(), `DELETE FROM machine_types WHERE id = $1`, outro)
	principal, err := c.repo.AddRouteOpResource(c.ctx, &entity.RouteOpResource{RouteOperationID: etapa.ID, WorkCenterID: c.centro, Priority: 1, TimeFactor: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.repo.SetRouteOpResourcePrimary(c.ctx, principal.ID, etapa.ID, c.centro); err != nil {
		t.Fatal(err)
	}
	if _, err := c.repo.AddRouteOpResource(c.ctx, &entity.RouteOpResource{RouteOperationID: etapa.ID, WorkCenterID: outro, Priority: 2, TimeFactor: 1.2}); err != nil {
		t.Fatal(err)
	}
	etapa.WorkCenterID = &outro
	etapa.Situation = entity.RouteOpApproved
	if _, err := c.repo.UpdateRouteOperation(c.ctx, etapa); err != nil {
		t.Fatal(err)
	}
	recursos, err := c.repo.ListResourcesByRouteOp(c.ctx, etapa.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range recursos {
		if r.IsPrimary != (r.WorkCenterID == outro) {
			t.Fatalf("o principal tem de ser o centro da etapa (%d): %+v", outro, recursos)
		}
	}
}

// A cópia leva etapas, rede, recursos, documentos e inspeções numa transação.
func TestCopiaDeRoteiroLevaTudo(t *testing.T) {
	c := novoCenario(t)
	origem := c.roteiro(t, nil, 1, true, nil, nil)
	a, b := c.etapa(t, origem.ID, 10), c.etapa(t, origem.ID, 20)
	if _, err := c.repo.SetNetworkEdge(c.ctx, &entity.NetworkEdge{PredecessorID: a.ID, SuccessorID: b.ID, OverlapPct: 25}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.repo.AddRouteOpResource(c.ctx, &entity.RouteOpResource{RouteOperationID: a.ID, WorkCenterID: c.centro, Priority: 1, TimeFactor: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.repo.CreateOperationDocument(c.ctx, &entity.OperationDocument{RouteOperationID: &b.ID, Kind: "DESENHO", Title: "Desenho da peça", CreatedBy: c.ator}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.repo.CreateRouteInspection(c.ctx, &entity.RouteInspection{RouteOperationID: b.ID, ItemCode: c.item, PointType: "PROCESSO", Description: "Dureza", SampleSize: decimal.NewFromInt(2), CreatedBy: c.ator}); err != nil {
		t.Fatal(err)
	}

	// Para um modelo: etapas, rede, recursos e documentos; inspeção não (é do item).
	modelo, _ := entity.NewRouteTemplate(testutil.UniqueCode(), "Modelo teste cópia", c.ator)
	_, levou, err := c.repo.CopyRoute(c.ctx, origem.ID, modelo)
	if err != nil {
		t.Fatalf("cópia para modelo: %v", err)
	}
	if levou.Steps != 2 || levou.Dependencies != 1 || levou.Resources != 1 || levou.Documents != 1 || levou.Inspections != 0 {
		t.Fatalf("cópia para modelo levou %+v", levou)
	}

	// De volta para o item, outra alternativa: leva também a inspeção.
	desc := "Cópia"
	destino, _ := entity.NewManufacturingRoute(testutil.UniqueCode(), c.item, nil, 2, &desc, false, nil, nil, c.ator)
	criado, levou, err := c.repo.CopyRoute(c.ctx, origem.ID, destino)
	if err != nil {
		t.Fatalf("cópia para item: %v", err)
	}
	if levou.Inspections != 1 {
		t.Fatalf("cópia para item leva a inspeção: %+v", levou)
	}
	edges, _ := c.repo.GetNetworkEdges(c.ctx, criado.ID)
	if len(edges) != 1 || edges[0].OverlapPct != 25 {
		t.Fatalf("a rede vai com a sobreposição: %+v", edges)
	}

	// Alternativa repetida: recusa clara e nada fica pela metade.
	repetido, _ := entity.NewManufacturingRoute(testutil.UniqueCode(), c.item, nil, 2, &desc, false, nil, nil, c.ator)
	if _, _, err := c.repo.CopyRoute(c.ctx, origem.ID, repetido); err == nil {
		t.Fatal("alternativa já usada tem de ser recusada")
	}
}

// Tempo por perímetro lido das dimensões do item.
func TestBaseDeTempoLeAsDimensoesDoItem(t *testing.T) {
	c := novoCenario(t)
	testutil.Exec(t, c.pool, `UPDATE items SET engineering_dimensions = '{"Length": 600, "Width": 400, "Height": 18}' WHERE code = $1`, c.item)
	rt := c.roteiro(t, nil, 1, true, nil, nil)
	etapa := c.etapa(t, rt.ID, 10)
	etapa.TimeBasis = s(entity.BasePerimetro)
	etapa.Situation = entity.RouteOpApproved
	if _, err := c.repo.UpdateRouteOperation(c.ctx, etapa); err != nil {
		t.Fatal(err)
	}
	ops, err := c.repo.GetRouteOperations(c.ctx, rt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ops[0].EffectiveTimeBasis != entity.BasePerimetro || ops[0].EffTime.UnitsPerPiece != 2 || ops[0].EffTime.SemMedida {
		t.Fatalf("porta 600×400 tem 2 m de perímetro: %+v / %+v", ops[0].EffectiveTimeBasis, ops[0].EffTime)
	}
	// Por atributo que o item não tem: marcada sem medida (não chuta 1).
	etapa.TimeBasis, etapa.TimeBasisAttribute = s(entity.BaseAtributo), s("FUROS")
	if _, err := c.repo.UpdateRouteOperation(c.ctx, etapa); err != nil {
		t.Fatal(err)
	}
	ops, _ = c.repo.GetRouteOperations(c.ctx, rt.ID)
	if !ops[0].EffTime.SemMedida {
		t.Fatalf("item sem o atributo FUROS: a etapa fica sem medida; veio %+v", ops[0].EffTime)
	}
}

// Componente por etapa: grava, lista e recusa etapa inexistente no use case.
func TestComponenteAlocadoNaEtapa(t *testing.T) {
	c := novoCenario(t)
	filho := testutil.UniqueCode()
	testutil.SeedItem(t, c.pool, c.ctx, filho, c.ator)
	defer testutil.Exec(t, c.pool, `DELETE FROM items WHERE code = $1`, filho)
	var linha int64
	if err := c.pool.QueryRow(c.ctx, `INSERT INTO item_structures (parent_code, child_code, quantity, unit_of_measurement, created_by)
		VALUES ($1, $2, 4, 'UN', $3) RETURNING id`, c.item, filho, c.ator).Scan(&linha); err != nil {
		t.Fatalf("estrutura: %v", err)
	}
	defer testutil.Exec(t, c.pool, `DELETE FROM item_structures WHERE id = $1`, linha)
	rt := c.roteiro(t, nil, 1, true, nil, nil)
	c.etapa(t, rt.ID, 10)
	c.etapa(t, rt.ID, 30)
	etapa := int16(30)
	if err := c.sqlc.SetMaterialStep(c.ctx, rt.ID, linha, &etapa); err != nil {
		t.Fatal(err)
	}
	mats, err := c.sqlc.ListRouteMaterials(c.ctx, rt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(mats) != 1 || mats[0].StepSequence == nil || *mats[0].StepSequence != 30 {
		t.Fatalf("componente na etapa 30: %+v", mats)
	}
}

// "Tornar principal" não pode fazer o centro antigo sumir da etapa: ele vira
// alternativo, com a prioridade que o promovido tinha.
func TestTrocarPrincipalGuardaOCentroAntigo(t *testing.T) {
	c := novoCenario(t)
	rt := c.roteiro(t, nil, 1, true, nil, nil)
	etapa := c.etapa(t, rt.ID, 10)
	etapa.WorkCenterID = &c.centro
	etapa.Situation = entity.RouteOpApproved
	if _, err := c.repo.UpdateRouteOperation(c.ctx, etapa); err != nil {
		t.Fatal(err)
	}
	var outro int64
	if err := c.pool.QueryRow(c.ctx, `INSERT INTO machine_types (code, name, type, enterprise_id, created_by)
		VALUES ($1, 'CT alternativo', 'CUT', $2, $3) RETURNING id`, testutil.UniqueCode(), c.empresa, c.ator).Scan(&outro); err != nil {
		t.Fatal(err)
	}
	defer c.pool.Exec(context.Background(), `DELETE FROM machine_types WHERE id = $1`, outro)
	alt, err := c.repo.AddRouteOpResource(c.ctx, &entity.RouteOpResource{RouteOperationID: etapa.ID, WorkCenterID: outro, Priority: 2, TimeFactor: 1.3})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.repo.SetRouteOpResourcePrimary(c.ctx, alt.ID, etapa.ID, outro); err != nil {
		t.Fatal(err)
	}
	recursos, _ := c.repo.ListResourcesByRouteOp(c.ctx, etapa.ID)
	achouAntigo := false
	for _, r := range recursos {
		if r.WorkCenterID == c.centro {
			achouAntigo = !r.IsPrimary && r.Priority == 2
		}
	}
	if !achouAntigo {
		t.Fatalf("o centro antigo tem de ficar como alternativo (prioridade 2): %+v", recursos)
	}
	ops, _ := c.repo.GetRouteOperations(c.ctx, rt.ID)
	if ops[0].EffectiveWorkCenterID == nil || *ops[0].EffectiveWorkCenterID != outro {
		t.Fatalf("a etapa passa a rodar no centro promovido")
	}
}

// A ordem de serviço nasce do roteiro que a OP daquela quantidade vai usar.
func TestOperacoesExternasSeguemAFaixaDeLote(t *testing.T) {
	c := novoCenario(t)
	externa, err := entity.NewOperation(testutil.UniqueCode(), "Zincagem teste", nil, entity.OriginThirdPart, nil, 0, 0, c.ator)
	if err != nil {
		t.Fatal(err)
	}
	prazo := int32(4)
	externa.LeadTimeDays = &prazo
	ext, err := c.repo.CreateOperation(c.ctx, externa)
	if err != nil {
		t.Fatal(err)
	}
	defer c.pool.Exec(context.Background(), `DELETE FROM operations WHERE id = $1`, ext.ID)
	padrao := c.roteiro(t, nil, 1, true, nil, nil)
	c.etapa(t, padrao.ID, 10)
	grande := c.roteiro(t, nil, 2, false, dec(100), nil)
	c.etapa(t, grande.ID, 10)
	ro, _ := entity.NewRouteOperation(grande.ID, 20, ext.ID, nil, nil, nil, nil)
	if _, err := c.repo.AddRouteOperation(c.ctx, ro); err != nil {
		t.Fatal(err)
	}
	pequeno, err := c.sqlc.GetExternalOpsForOrder(c.ctx, c.item, "", decimal.NewFromInt(10))
	if err != nil {
		t.Fatal(err)
	}
	lote, err := c.sqlc.GetExternalOpsForOrder(c.ctx, c.item, "", decimal.NewFromInt(500))
	if err != nil {
		t.Fatal(err)
	}
	if len(pequeno) != 0 || len(lote) != 1 || lote[0].LeadTimeDays != 4 {
		t.Fatalf("10 peças usam o padrão (sem terceiro); 500 usam o alternativo com zincagem: %d e %+v", len(pequeno), lote)
	}
}
