package routing_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	machineentity "github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	routingrepo "github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
)

type rotasFalsas struct {
	rota    *routingentity.ManufacturingRoute
	etapas  []*routingentity.RouteOperation
	arestas []*routingentity.NetworkEdge
	routingrepo.RoutingRepository
}

func (r rotasFalsas) GetRouteByID(context.Context, int64) (*routingentity.ManufacturingRoute, error) {
	return r.rota, nil
}
func (r rotasFalsas) GetRouteOperations(context.Context, int64) ([]*routingentity.RouteOperation, error) {
	return r.etapas, nil
}
func (r rotasFalsas) GetNetworkEdges(context.Context, int64) ([]*routingentity.NetworkEdge, error) {
	return r.arestas, nil
}

type maquinasFalsas struct {
	tempos []*machineentity.ItemMachineTime
}

func (m maquinasFalsas) ListItemMachineTimes(context.Context, int64) ([]*machineentity.ItemMachineTime, error) {
	return m.tempos, nil
}

type centrosFalsos struct{ de map[int64]int64 }

func (c centrosFalsos) WorkCenterOfMachine(_ context.Context, maquina int64) (int64, error) {
	return c.de[maquina], nil
}

type tarifasFalsas struct{ com map[int64]float64 }

func (t tarifasFalsas) HourlyRate(_ context.Context, centro int64) (float64, bool, error) {
	valor, ok := t.com[centro]
	return valor, ok, nil
}

type authSempre struct{ ports.AuthService }

func (authSempre) CanResolveStructure(context.Context) bool { return true }

func etapa(seq int16, nome string, centro *int64, origem routingentity.OperationOrigin) *routingentity.RouteOperation {
	return &routingentity.RouteOperation{
		ID: int64(seq), Sequence: seq, OperationName: nome,
		EffectiveWorkCenterID: centro, OperationOrigin: origem,
		EffTime: routingentity.OperationTime{Run: 0.5, RunBaseQty: 1},
	}
}

func i64(v int64) *int64 { return &v }

// O checklist existe para a recusa chegar ANTES do MRP e dizer o que falta,
// não "cadastre a produtividade do item 900500 no centro 8" no meio do cálculo.
func TestProntidaoDoRoteiroApontaOQueFalta(t *testing.T) {
	desc := "Roteiro da chapa"
	centroLaser, centroSolda := i64(8), i64(9)
	uc := &RouteReadinessUseCase{
		Routes: rotasFalsas{
			rota: &routingentity.ManufacturingRoute{ID: 42, ItemCode: 900500, Description: &desc},
			etapas: []*routingentity.RouteOperation{
				etapa(10, "Cortar no laser", centroLaser, routingentity.OriginInternal),
				etapa(20, "Soldar", centroSolda, routingentity.OriginInternal),
			},
		},
		Machines: maquinasFalsas{tempos: []*machineentity.ItemMachineTime{
			{MachineCode: 30, IsActive: true}, // pertence ao laser
		}},
		Centers: centrosFalsos{de: map[int64]int64{30: 8}},
		Rates:   tarifasFalsas{com: map[int64]float64{8: 37.09}},
		Auth:    authSempre{},
	}

	rel, err := uc.Execute(context.Background(), 42)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if rel.Ready {
		t.Fatal("roteiro sem máquina apta na solda não pode ser dado como pronto")
	}
	if rel.Steps != 2 {
		t.Fatalf("etapas contadas: %d", rel.Steps)
	}
	juntas := strings.Join(rel.Issues, " | ")
	if !strings.Contains(juntas, "nenhuma máquina") {
		t.Fatalf("faltou apontar o centro sem máquina apta: %s", juntas)
	}
	avisos := strings.Join(rel.Warnings, " | ")
	if !strings.Contains(avisos, "tarifa por hora") {
		t.Fatalf("faltou avisar da tarifa ausente no centro 9: %s", avisos)
	}
	// O centro que está completo não pode virar pendência.
	for _, c := range rel.WorkCenters {
		if c.WorkCenterID == 8 && (!c.HasMachine || !c.HasRate) {
			t.Fatalf("centro completo foi reportado como incompleto: %+v", c)
		}
	}
}

// Etapa que sai da fábrica não precisa de máquina — precisa de prazo.
func TestProntidaoExigePrazoDaEtapaExterna(t *testing.T) {
	externa := etapa(30, "Cementar em terceiro", nil, routingentity.OriginThirdPart)
	uc := &RouteReadinessUseCase{
		Routes: rotasFalsas{
			rota:   &routingentity.ManufacturingRoute{ID: 7, ItemCode: 1},
			etapas: []*routingentity.RouteOperation{externa},
		},
		Auth: authSempre{},
	}
	rel, err := uc.Execute(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if rel.Ready {
		t.Fatal("etapa externa sem prazo deixa o planejamento sem saber quando a peça volta")
	}
	if !strings.Contains(strings.Join(rel.Issues, " | "), "sem prazo em dias") {
		t.Fatalf("mensagem não explica o que falta: %v", rel.Issues)
	}

	prazo := int32(5)
	externa.LeadTimeDays = &prazo
	fornecedor := int64(10)
	externa.SupplierID = &fornecedor
	rel, _ = uc.Execute(context.Background(), 7)
	if !rel.Ready {
		t.Fatalf("com prazo e fornecedor o roteiro está pronto: %v", rel.Issues)
	}
}

// Roteiro vazio é o erro mais comum de quem criou a capa e parou ali.
func TestProntidaoRecusaRoteiroSemEtapa(t *testing.T) {
	uc := &RouteReadinessUseCase{
		Routes: rotasFalsas{rota: &routingentity.ManufacturingRoute{ID: 1, ItemCode: 1}},
		Auth:   authSempre{},
	}
	rel, err := uc.Execute(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if rel.Ready || !strings.Contains(strings.Join(rel.Issues, " "), "sem etapas ativas") {
		t.Fatalf("roteiro vazio deveria ser recusado: %+v", rel)
	}
}

type inspecaoFalsa struct{ sem []int16 }

func (i inspecaoFalsa) InspectionStepsWithoutPlan(context.Context, int64) ([]int16, error) {
	return i.sem, nil
}

// Etapa de inspeção sem plano NÃO impede o roteiro de ser planejado — a ordem
// nasce e a etapa volta marcada. Mas precisa aparecer aqui, enquanto o roteiro
// está sendo montado e ainda é barato resolver.
func TestProntidaoAvisaInspecaoSemPlanoSemBloquear(t *testing.T) {
	centro := i64(8)
	uc := &RouteReadinessUseCase{
		Routes: rotasFalsas{
			rota:   &routingentity.ManufacturingRoute{ID: 5, ItemCode: 900500},
			etapas: []*routingentity.RouteOperation{etapa(10, "Cortar", centro, routingentity.OriginInternal)},
		},
		Machines:   maquinasFalsas{tempos: []*machineentity.ItemMachineTime{{MachineCode: 30, IsActive: true}}},
		Centers:    centrosFalsos{de: map[int64]int64{30: 8}},
		Rates:      tarifasFalsas{com: map[int64]float64{8: 37.09}},
		Inspection: inspecaoFalsa{sem: []int16{10}},
		Auth:       authSempre{},
	}
	rel, err := uc.Execute(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if !rel.Ready {
		t.Fatalf("plano de inspeção ausente não pode impedir o planejamento: %v", rel.Issues)
	}
	if !strings.Contains(strings.Join(rel.Warnings, " "), "plano ativo") {
		t.Fatalf("o aviso precisa aparecer: %v", rel.Warnings)
	}
}
