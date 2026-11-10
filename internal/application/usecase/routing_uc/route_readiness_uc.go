package routing_uc

import (
	"context"
	"fmt"
	"sort"
	"strings"

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
	if route.IsTemplate {
		// Modelo não é planejado: só faz sentido conferir a estrutura do
		// processo; máquina apta e tarifa dependem do item que receber a cópia.
		report.Warnings = append(report.Warnings,
			"roteiro-modelo: não entra no planejamento; copie-o para um item para planejar")
	}
	if route.Description != nil {
		report.Description = *route.Description
	}

	ativas := make([]*routingentity.RouteOperation, 0, len(ops))
	for _, op := range ops {
		// Fantasma e inativa não são executadas: não entram na conferência.
		if !op.ConsomeTempo() {
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
			// Prazo e fornecedor valem da etapa OU da operação de biblioteca.
			// Antes só a etapa contava, e a pendência aparecia com o prazo
			// preenchido na biblioteca.
			if op.EffectiveSubcontractDays <= 0 {
				report.Issues = append(report.Issues, fmt.Sprintf(
					"etapa %d (%s) sai da fábrica e está sem prazo em dias: o planejamento não sabe quando a peça volta",
					op.Sequence, op.OperationName))
			}
			if op.EffectiveSupplierID == nil {
				report.Warnings = append(report.Warnings, fmt.Sprintf(
					"etapa %d (%s) não diz qual fornecedor executa", op.Sequence, op.OperationName))
			}
			continue
		}
		// A tela antiga gravava o "tempo de máquina" da etapa no campo legado,
		// que o modelo de tempo não lê: o número ficava no cadastro e nunca
		// entrava em OP, prazo nem custo.
		if op.StandardTime != nil && op.RunTime == nil && *op.StandardTime > 0 {
			report.Warnings = append(report.Warnings, fmt.Sprintf(
				"etapa %d (%s) tem tempo de máquina %.4g gravado no campo antigo, que o planejamento ignora: abra a etapa e confirme o tempo",
				op.Sequence, op.OperationName, *op.StandardTime))
		}
		if op.EffTime.SemMedida {
			report.Issues = append(report.Issues, fmt.Sprintf(
				"etapa %d (%s) conta o tempo %s e o item não informa essa medida: complete as dimensões ou o atributo no cadastro do item",
				op.Sequence, op.OperationName, routingentity.RotuloDaBase(op.EffectiveTimeBasis, op.EffectiveTimeBasisAttribute)))
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
	if route.IsTemplate {
		report.Ready = len(report.Issues) == 0
		return report, nil
	}
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
			if errorsuc.EhNaoEncontrado(err) {
				continue // máquina inativa ou sem tipo resolvido não conta como apta
			}
			if err != nil {
				return nil, fmt.Errorf("lendo o centro da máquina %d: %w", tempo.MachineCode, err)
			}
			comMaquina[centro] = true
		}
	}

	for _, check := range centros {
		check.HasMachine = comMaquina[check.WorkCenterID]
		semTarifaDeMaquina := false
		if detalhadas, ok := uc.Rates.(tarifasDoCentro); ok {
			// A mesma regra do custo padrão: tarifa de máquina (ou a hora
			// geral, quando a de máquina não foi informada) mais a de mão de
			// obra. Antes valia só a hora geral: centro com tarifa só de mão de
			// obra aparecia "valendo zero" (mentira) e linha cadastrada com
			// zero passava como tarifada (outra mentira).
			maquina, maoDeObra, err := detalhadas.Tarifas(ctx, check.WorkCenterID)
			if err != nil {
				return nil, fmt.Errorf("lendo a taxa horária do centro %d: %w", check.WorkCenterID, err)
			}
			check.HasRate, check.HourlyRate = maquina > 0 || maoDeObra > 0, maquina+maoDeObra
			semTarifaDeMaquina = maquina <= 0 && maoDeObra > 0
		} else if uc.Rates != nil {
			rate, found, err := uc.Rates.HourlyRate(ctx, check.WorkCenterID)
			if err != nil {
				return nil, fmt.Errorf("lendo a taxa horária do centro %d: %w", check.WorkCenterID, err)
			}
			check.HasRate, check.HourlyRate = found && rate > 0, rate
		}
		nome := check.WorkCenterName
		if nome == "" {
			nome = fmt.Sprintf("centro %d", check.WorkCenterID)
		}
		if !check.HasMachine {
			report.Issues = append(report.Issues, fmt.Sprintf(
				"nenhuma máquina de %s tem a produtividade deste item cadastrada (%s): sem isso o planejamento não tem onde rodar a ordem",
				nome, etapasPorExtenso(check.Steps)))
		}
		switch {
		case uc.Rates == nil:
		case !check.HasRate:
			// Não trava o plano — trava o custo, e em silêncio: a operação entra
			// no cálculo valendo zero e a margem sai maior do que é.
			report.Warnings = append(report.Warnings, fmt.Sprintf(
				"%s está sem tarifa por hora (VCUS0100): %s no custo valendo zero", nome, etapasEntram(check.Steps)))
		case semTarifaDeMaquina:
			report.Warnings = append(report.Warnings, fmt.Sprintf(
				"%s tem só a tarifa de mão de obra (VCUS0100): a hora-máquina de %s entra no custo valendo zero", nome, etapasPorExtenso(check.Steps)))
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
				"%s %s ponto de inspeção sem plano ativo: a ordem será criada, mas a conferência não abre até o plano existir",
				etapasPorExtenso(semPlano), map[bool]string{true: "é", false: "são"}[len(semPlano) == 1]))
		}
	}

	report.Ready = len(report.Issues) == 0
	return report, nil
}

// tarifasDoCentro: leitura das duas tarifas do centro, como o custo as usa.
type tarifasDoCentro interface {
	Tarifas(ctx context.Context, workCenterID int64) (maquina, maoDeObra float64, err error)
}

// etapasPorExtenso: [20 30] → "etapas 20 e 30"; [10] → "etapa 10".
func etapasPorExtenso(seqs []int16) string {
	partes := make([]string, len(seqs))
	for i, s := range seqs {
		partes[i] = fmt.Sprint(s)
	}
	switch len(partes) {
	case 0:
		return "nenhuma etapa"
	case 1:
		return "etapa " + partes[0]
	}
	return "etapas " + strings.Join(partes[:len(partes)-1], ", ") + " e " + partes[len(partes)-1]
}

// etapasEntram: "a etapa 10 entra" / "as etapas 20 e 30 entram".
func etapasEntram(seqs []int16) string {
	if len(seqs) == 1 {
		return "a " + etapasPorExtenso(seqs) + " entra"
	}
	return "as " + etapasPorExtenso(seqs) + " entram"
}
