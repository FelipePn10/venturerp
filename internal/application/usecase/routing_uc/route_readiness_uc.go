package routing_uc

import (
	"context"
	"fmt"
	"sort"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	machineentity "github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	routingrepo "github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
)

// RouteReadinessReport é o veredito de "este roteiro consegue ser planejado?",
// com a lista do que falta.
//
// Existe porque a recusa chegava tarde e fora de contexto: o usuário montava o
// roteiro, rodava o MRP e recebia "cadastre a produtividade do item 900500 no
// centro 8" no meio do cálculo — sem saber qual etapa, qual centro pelo nome,
// nem que faltava também a tarifa/hora. É o mesmo gesto do
// `activation-readiness` do item: conferir antes, no lugar onde se conserta.
type RouteReadinessReport struct {
	RouteID     int64             `json:"route_id"`
	ItemCode    int64             `json:"item_code"`
	Description string            `json:"description"`
	Ready       bool              `json:"ready"`
	Steps       int               `json:"steps"`
	Issues      []string          `json:"issues,omitempty"`
	Warnings    []string          `json:"warnings,omitempty"`
	WorkCenters []WorkCenterCheck `json:"work_centers,omitempty"`
}

// WorkCenterCheck é o que cada centro usado pelo roteiro tem e não tem.
type WorkCenterCheck struct {
	WorkCenterID   int64   `json:"work_center_id"`
	WorkCenterName string  `json:"work_center_name"`
	Steps          []int16 `json:"steps"`
	HasMachine     bool    `json:"has_machine"`
	HasRate        bool    `json:"has_rate"`
	HourlyRate     float64 `json:"hourly_rate"`
}

type machineTimeReader interface {
	ListItemMachineTimes(ctx context.Context, itemCode int64) ([]*machineentity.ItemMachineTime, error)
}

type workCenterRateReader interface {
	HourlyRate(ctx context.Context, workCenterID int64) (float64, bool, error)
}

// machineWorkCenterReader traduz máquina → centro de trabalho. A produtividade
// é cadastrada por MÁQUINA; o roteiro aponta para o CENTRO. Sem essa tradução
// não dá para dizer se o centro da etapa tem alguma máquina apta.
type machineWorkCenterReader interface {
	WorkCenterOfMachine(ctx context.Context, machineCode int64) (int64, error)
}

// inspectionPlanReader lista as etapas marcadas para inspeção que ainda não têm
// plano ativo. É aviso, não impedimento: a ordem nasce assim mesmo e a etapa
// volta marcada — mas quem monta o roteiro fica sabendo antes.
type inspectionPlanReader interface {
	InspectionStepsWithoutPlan(ctx context.Context, routeID int64) ([]int16, error)
}

// RouteReadinessUseCase confere um roteiro contra o que o planejamento vai
// exigir dele.
type RouteReadinessUseCase struct {
	Routes     routingrepo.RoutingRepository
	Machines   machineTimeReader
	Rates      workCenterRateReader
	Centers    machineWorkCenterReader
	Inspection inspectionPlanReader
	Auth       ports.AuthService
}

func (uc *RouteReadinessUseCase) Execute(ctx context.Context, routeID int64) (*RouteReadinessReport, error) {
	if uc.Auth == nil || !uc.Auth.CanResolveStructure(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	route, err := uc.Routes.GetRouteByID(ctx, routeID)
	if err != nil {
		return nil, errorsuc.NewValidationError("roteiro não encontrado na empresa autenticada")
	}
	ops, err := uc.Routes.GetRouteOperations(ctx, routeID)
	if err != nil {
		return nil, err
	}
	edges, err := uc.Routes.GetNetworkEdges(ctx, routeID)
	if err != nil {
		return nil, err
	}

	report := &RouteReadinessReport{RouteID: routeID, ItemCode: route.ItemCode}
	if route.Description != nil {
		report.Description = *route.Description
	}

	ativas := make([]*routingentity.RouteOperation, 0, len(ops))
	for _, op := range ops {
		if op.Situation == routingentity.RouteOpInactive {
			continue
		}
		ativas = append(ativas, op)
	}
	report.Steps = len(ativas)
	if len(ativas) == 0 {
		report.Issues = append(report.Issues, "roteiro sem etapas ativas: inclua ao menos uma etapa")
		return report, nil
	}
	if routingentity.CriticalPath(ativas, edges, 1).HasCycle() {
		report.Issues = append(report.Issues,
			"a rede de dependências tem um ciclo: alguma etapa depende, direta ou indiretamente, de si mesma")
	}

	// Centros usados pelas etapas internas, com as etapas que caem em cada um.
	centros := map[int64]*WorkCenterCheck{}
	for _, op := range ativas {
		switch op.OperationOrigin {
		case routingentity.OriginExternal, routingentity.OriginThirdPart:
			if op.LeadTimeDays == nil || *op.LeadTimeDays <= 0 {
				report.Issues = append(report.Issues, fmt.Sprintf(
					"etapa %d (%s) sai da fábrica e está sem prazo em dias: o planejamento não sabe quando a peça volta",
					op.Sequence, op.OperationName))
			}
			if op.SupplierID == nil {
				report.Warnings = append(report.Warnings, fmt.Sprintf(
					"etapa %d (%s) não diz qual fornecedor executa", op.Sequence, op.OperationName))
			}
			continue
		}
		if op.EffTime.Run <= 0 && op.EffectiveStdTime <= 0 {
			report.Warnings = append(report.Warnings, fmt.Sprintf(
				"etapa %d (%s) está sem tempo de máquina: o tempo virá da produtividade por máquina",
				op.Sequence, op.OperationName))
		}
		if op.EffectiveWorkCenterID == nil {
			report.Issues = append(report.Issues, fmt.Sprintf(
				"etapa %d (%s) sem centro de trabalho: escolha o centro na etapa ou defina o centro padrão da operação",
				op.Sequence, op.OperationName))
			continue
		}
		id := *op.EffectiveWorkCenterID
		if centros[id] == nil {
			centros[id] = &WorkCenterCheck{WorkCenterID: id, WorkCenterName: op.WorkCenterName}
		}
		if centros[id].WorkCenterName == "" {
			centros[id].WorkCenterName = op.WorkCenterName
		}
		centros[id].Steps = append(centros[id].Steps, op.Sequence)
	}

	// Máquinas do item, agrupadas pelo centro a que pertencem.
	comMaquina := map[int64]bool{}
	if uc.Machines != nil && uc.Centers != nil {
		tempos, err := uc.Machines.ListItemMachineTimes(ctx, route.ItemCode)
		if err != nil {
			return nil, err
		}
		for _, tempo := range tempos {
			if !tempo.IsActive {
				continue
			}
			centro, err := uc.Centers.WorkCenterOfMachine(ctx, tempo.MachineCode)
			if err != nil {
				continue // máquina sem tipo resolvido não conta como apta
			}
			comMaquina[centro] = true
		}
	}

	for _, check := range centros {
		check.HasMachine = comMaquina[check.WorkCenterID]
		if uc.Rates != nil {
			if rate, found, err := uc.Rates.HourlyRate(ctx, check.WorkCenterID); err == nil {
				check.HasRate, check.HourlyRate = found, rate
			}
		}
		nome := check.WorkCenterName
		if nome == "" {
			nome = fmt.Sprintf("centro %d", check.WorkCenterID)
		}
		if !check.HasMachine {
			report.Issues = append(report.Issues, fmt.Sprintf(
				"nenhuma máquina de %s tem a produtividade deste item cadastrada (etapas %v): sem isso o planejamento não tem onde rodar a ordem",
				nome, check.Steps))
		}
		if !check.HasRate {
			// Não trava o plano — trava o custo, e em silêncio: a operação entra
			// no cálculo valendo zero e a margem sai maior do que é.
			report.Warnings = append(report.Warnings, fmt.Sprintf(
				"%s está sem tarifa por hora (VCUS0100): as etapas %v entram no custo valendo zero", nome, check.Steps))
		}
		report.WorkCenters = append(report.WorkCenters, *check)
	}
	sort.Slice(report.WorkCenters, func(i, j int) bool {
		return report.WorkCenters[i].WorkCenterID < report.WorkCenters[j].WorkCenterID
	})

	if uc.Inspection != nil {
		semPlano, err := uc.Inspection.InspectionStepsWithoutPlan(ctx, routeID)
		if err != nil {
			return nil, err
		}
		if len(semPlano) > 0 {
			report.Warnings = append(report.Warnings, fmt.Sprintf(
				"as etapas %v são ponto de inspeção e não têm plano ativo: a ordem será criada, mas a conferência não abre até o plano existir",
				semPlano))
		}
	}

	report.Ready = len(report.Issues) == 0
	return report, nil
}
