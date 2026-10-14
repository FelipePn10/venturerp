package planning_uc

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_calculation_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_report_uc"
	mrpentity "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
)

type scenarioMRP interface {
	Calculate(ctx context.Context, planCode, initialOrderNumber int64, generateLLC bool) (*mrpentity.MRPCalculationLog, error)
}

type scenarioCRP interface {
	CalculateCRP(ctx context.Context, dto request.CalculateCRPDTO) (*response.CRPSummaryResponse, error)
	ListByPlan(ctx context.Context, planCode int64) ([]*response.CRPEntryResponse, error)
}

type scenarioCoverage interface {
	Coverage(ctx context.Context, filter mrp_report_uc.Filter, today time.Time) ([]mrp_report_uc.CoverageRow, error)
}

type scenarioResults interface {
	ListSuggestionsByPlan(ctx context.Context, planCode int64) ([]*mrpentity.PlannedOrderSuggestion, error)
	ListExceptionsByPlan(ctx context.Context, planCode int64) ([]*mrpentity.MRPExceptionMessage, error)
}

type scenarioPegging interface {
	Roots(ctx context.Context, planCode int64) ([]mrp_calculation_uc.PeggingRoot, error)
}

type scenarioLock interface {
	TryLock(ctx context.Context, enterpriseID int64) (bool, error)
	Unlock(ctx context.Context, enterpriseID int64) error
}

type planCalculatedAt interface {
	LastCalculatedAt(ctx context.Context, planCode int64) (*time.Time, error)
}

// ScenarioUseCase é o what-if do planejamento: cria cenários sobre um plano,
// lança ajustes de demanda, estoque, prazo e capacidade, calcula MRP + CRP num
// plano oculto e compara com o plano real — o long-term planning do SAP
// (MS31/MS02) e os planos de simulação do Oracle. Só a demanda é promovida ao
// plano real.
//
// O APS não roda no cenário: ele sequencia as OFs reais da empresa inteira e
// sobrescreveria a programação da fábrica.
type ScenarioUseCase struct {
	Repo     ScenarioRepository
	MRP      scenarioMRP
	CRP      scenarioCRP
	Coverage scenarioCoverage
	Results  scenarioResults
	Pegging  scenarioPegging
	Lock     scenarioLock
	Plans    planCalculatedAt
	Auth     ports.AuthService
	Now      func() time.Time
}

func (uc *ScenarioUseCase) now() time.Time {
	if uc.Now != nil {
		return uc.Now()
	}
	return time.Now()
}

func (uc *ScenarioUseCase) canRead(ctx context.Context) error {
	if uc.Auth == nil || !uc.Auth.CanRunMRPCalculation(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return nil
}

// ScenarioDetail é o cenário com os ajustes.
type ScenarioDetail struct {
	Scenario
	Changes []ScenarioChange `json:"changes"`
}

func (uc *ScenarioUseCase) List(ctx context.Context) ([]Scenario, error) {
	if err := uc.canRead(ctx); err != nil {
		return nil, err
	}
	return uc.Repo.List(ctx)
}

func (uc *ScenarioUseCase) Get(ctx context.Context, id int64) (*ScenarioDetail, error) {
	if err := uc.canRead(ctx); err != nil {
		return nil, err
	}
	s, err := uc.Repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	changes, err := uc.Repo.Changes(ctx, id)
	if err != nil {
		return nil, err
	}
	return &ScenarioDetail{Scenario: *s, Changes: changes}, nil
}

func validName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errorsuc.NewValidationError("informe o nome do cenário")
	}
	if len([]rune(name)) > 120 {
		return "", errorsuc.NewValidationError("o nome do cenário pode ter até 120 caracteres")
	}
	return name, nil
}

func (uc *ScenarioUseCase) Create(ctx context.Context, dto request.CreateScenarioDTO) (*ScenarioDetail, error) {
	if err := uc.canRead(ctx); err != nil {
		return nil, err
	}
	name, err := validName(dto.Name)
	if err != nil {
		return nil, err
	}
	if dto.BasePlanCode <= 0 {
		return nil, errorsuc.NewValidationError("informe o plano base do cenário")
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	s, err := uc.Repo.Create(ctx, name, strings.TrimSpace(dto.Description), dto.BasePlanCode, ator)
	if err != nil {
		return nil, err
	}
	return &ScenarioDetail{Scenario: *s, Changes: []ScenarioChange{}}, nil
}

func (uc *ScenarioUseCase) Update(ctx context.Context, id int64, dto request.UpdateScenarioDTO) (*ScenarioDetail, error) {
	if err := uc.canRead(ctx); err != nil {
		return nil, err
	}
	name, err := validName(dto.Name)
	if err != nil {
		return nil, err
	}
	if err := uc.Repo.UpdateMeta(ctx, id, name, strings.TrimSpace(dto.Description)); err != nil {
		return nil, err
	}
	return uc.Get(ctx, id)
}

// Delete descarta o cenário e tudo o que o cálculo dele gravou.
func (uc *ScenarioUseCase) Delete(ctx context.Context, id int64) error {
	if err := uc.canRead(ctx); err != nil {
		return err
	}
	return uc.Repo.Delete(ctx, id)
}

func parseDay(raw *string, campo string) (*time.Time, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	d, err := time.Parse("2006-01-02", strings.TrimSpace(*raw))
	if err != nil {
		return nil, errorsuc.NewValidationError(campo + " inválida: use o formato ano-mês-dia")
	}
	return &d, nil
}

// AddChange valida e grava um ajuste. Cada tipo exige o que faz sentido para
// ele; referências (item, máquina, demanda) têm de ser da empresa.
func (uc *ScenarioUseCase) AddChange(ctx context.Context, id int64, dto request.ScenarioChangeDTO) (*ScenarioChange, error) {
	if err := uc.canRead(ctx); err != nil {
		return nil, err
	}
	s, err := uc.Repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.PromotedAt != nil {
		return nil, errorsuc.NewConflictError("o cenário já foi aplicado ao plano real; crie um cenário novo para outra simulação")
	}
	c := ScenarioChange{Kind: strings.ToUpper(strings.TrimSpace(dto.Kind)), Mask: strings.TrimSpace(dto.Mask), Notes: strings.TrimSpace(dto.Notes)}
	need, err := parseDay(dto.NeedDate, "data da necessidade")
	if err != nil {
		return nil, err
	}
	stopFrom, err := parseDay(dto.StopFrom, "data de início da parada")
	if err != nil {
		return nil, err
	}
	stopTo, err := parseDay(dto.StopTo, "data de fim da parada")
	if err != nil {
		return nil, err
	}
	exigeItem := func() error {
		if dto.ItemCode == nil || *dto.ItemCode <= 0 {
			return errorsuc.NewValidationError("informe o item")
		}
		ok, err := uc.Repo.ItemExists(ctx, *dto.ItemCode)
		if err != nil {
			return err
		}
		if !ok {
			return errorsuc.NewNotFoundError("item não encontrado nesta empresa")
		}
		c.ItemCode = dto.ItemCode
		return nil
	}

	switch c.Kind {
	case ChangeDemand:
		c.DemandRefKind = strings.ToUpper(strings.TrimSpace(dto.DemandRefKind))
		switch c.DemandRefKind {
		case DemandRefNew:
			if err := exigeItem(); err != nil {
				return nil, err
			}
			if dto.Quantity == nil || !dto.Quantity.IsPositive() {
				return nil, errorsuc.NewValidationError("informe a quantidade da demanda nova")
			}
			if need == nil {
				return nil, errorsuc.NewValidationError("informe a data da demanda nova")
			}
		case DemandRefIndependent, DemandRefSalesOrder:
			if dto.DemandRefCode == nil || *dto.DemandRefCode <= 0 {
				return nil, errorsuc.NewValidationError("informe a demanda a alterar")
			}
			ok, err := uc.Repo.DemandRefExists(ctx, c.DemandRefKind, *dto.DemandRefCode)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, errorsuc.NewNotFoundError("demanda não encontrada nesta empresa")
			}
			if dto.Quantity == nil && need == nil {
				return nil, errorsuc.NewValidationError("informe a nova quantidade ou a nova data (quantidade 0 retira a demanda)")
			}
			if dto.Quantity != nil && dto.Quantity.IsNegative() {
				return nil, errorsuc.NewValidationError("a quantidade não pode ser negativa")
			}
			c.DemandRefCode = dto.DemandRefCode
		default:
			return nil, errorsuc.NewValidationError("ajuste de demanda: use NOVA, DEMANDA_INDEPENDENTE ou PEDIDO_VENDA")
		}
		c.Quantity, c.NeedDate = dto.Quantity, need
	case ChangeStock:
		if err := exigeItem(); err != nil {
			return nil, err
		}
		if dto.Quantity == nil || dto.Quantity.IsZero() {
			return nil, errorsuc.NewValidationError("informe a entrada (positiva) ou a baixa (negativa) de estoque")
		}
		c.Quantity = dto.Quantity
	case ChangeLeadTime:
		if err := exigeItem(); err != nil {
			return nil, err
		}
		if dto.LeadTimeDays == nil || *dto.LeadTimeDays < 0 || *dto.LeadTimeDays > 3650 {
			return nil, errorsuc.NewValidationError("informe o prazo em dias (de 0 a 3650)")
		}
		c.LeadTimeDays = dto.LeadTimeDays
	case ChangeCapacity:
		if dto.MachineID == nil || *dto.MachineID <= 0 {
			return nil, errorsuc.NewValidationError("informe a máquina")
		}
		ok, err := uc.Repo.MachineExists(ctx, *dto.MachineID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errorsuc.NewNotFoundError("máquina não encontrada nesta empresa")
		}
		if dto.HoursPerDay == nil && stopFrom == nil {
			return nil, errorsuc.NewValidationError("informe as horas por dia ou o período de parada")
		}
		if dto.HoursPerDay != nil && (dto.HoursPerDay.IsNegative() || dto.HoursPerDay.GreaterThan(decimal.NewFromInt(24))) {
			return nil, errorsuc.NewValidationError("as horas por dia devem estar entre 0 e 24")
		}
		if stopTo != nil && stopFrom == nil {
			return nil, errorsuc.NewValidationError("informe o início da parada")
		}
		if stopFrom != nil && stopTo != nil && stopTo.Before(*stopFrom) {
			return nil, errorsuc.NewValidationError("o fim da parada não pode ser anterior ao início")
		}
		c.MachineID, c.HoursPerDay, c.StopFrom, c.StopTo = dto.MachineID, dto.HoursPerDay, stopFrom, stopTo
	default:
		return nil, errorsuc.NewValidationError("tipo de ajuste inválido: use DEMANDA, ESTOQUE, LEAD_TIME ou CAPACIDADE")
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.AddChange(ctx, id, c, ator)
}

func (uc *ScenarioUseCase) DeleteChange(ctx context.Context, id, changeID int64) error {
	if err := uc.canRead(ctx); err != nil {
		return err
	}
	s, err := uc.Repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if s.PromotedAt != nil {
		return errorsuc.NewConflictError("o cenário já foi aplicado ao plano real")
	}
	return uc.Repo.DeleteChange(ctx, id, changeID)
}

// BaseDemands lista as demandas do plano base que podem ser ajustadas.
func (uc *ScenarioUseCase) BaseDemands(ctx context.Context, id int64) ([]BaseDemand, error) {
	if err := uc.canRead(ctx); err != nil {
		return nil, err
	}
	s, err := uc.Repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return uc.Repo.BaseDemands(ctx, s.BasePlanCode)
}

// ScenarioRunResult resume o cálculo do cenário.
type ScenarioRunResult struct {
	MRPStatus   string   `json:"mrp_status"`
	MRPItems    int      `json:"mrp_items"`
	MRPOrders   int      `json:"mrp_orders"`
	CRPOverload int      `json:"crp_overload"`
	Notes       []string `json:"notes"`
}

// Calculate roda MRP e CRP no plano oculto, com a configuração atual do plano
// base e os ajustes do cenário. Usa o mesmo cadeado da execução automática:
// cenário e plano real não calculam ao mesmo tempo na empresa.
func (uc *ScenarioUseCase) Calculate(ctx context.Context, id int64) (*ScenarioRunResult, error) {
	if err := uc.canRead(ctx); err != nil {
		return nil, err
	}
	s, err := uc.Repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	empresa, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	if uc.Lock != nil {
		ok, err := uc.Lock.TryLock(ctx, empresa)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errorsuc.NewConflictError("já existe um cálculo de planejamento em andamento nesta empresa; aguarde o término")
		}
		defer func() { _ = uc.Lock.Unlock(context.WithoutCancel(ctx), empresa) }()
	}
	if err := uc.Repo.SyncPlan(ctx, id); err != nil {
		return nil, err
	}
	// O número inicial não é usado: sugestão de cenário não recebe número.
	log, err := uc.MRP.Calculate(ctx, s.ScenarioPlanCode, 1, false)
	if err != nil {
		return nil, fmt.Errorf("cálculo do MRP do cenário: %w", err)
	}
	out := &ScenarioRunResult{MRPStatus: log.Status, MRPItems: log.TotalItems, MRPOrders: log.TotalOrders, Notes: []string{}}
	if log.Status == "ERROR" {
		return nil, errorsuc.NewValidationError("o MRP do cenário não pôde ser calculado: " + resumoErros(log.Errors))
	}
	if log.Status != "COMPLETED" {
		out.Notes = append(out.Notes, "MRP concluído com avisos: "+resumoErros(log.Errors))
	}
	crp, err := uc.CRP.CalculateCRP(ctx, request.CalculateCRPDTO{PlanCode: s.ScenarioPlanCode})
	if err != nil {
		return nil, fmt.Errorf("cálculo do CRP do cenário: %w", err)
	}
	out.CRPOverload = crp.OverloadCount
	if crp.OverloadCount > 0 {
		out.Notes = append(out.Notes, fmt.Sprintf("%d centro(s)/dia em sobrecarga de capacidade", crp.OverloadCount))
	}
	if err := uc.Repo.MarkCalculated(ctx, id, strings.Join(out.Notes, " · ")); err != nil {
		return nil, err
	}
	return out, nil
}

func resumoErros(errs map[string]interface{}) string {
	if len(errs) == 0 {
		return "sem detalhe"
	}
	chaves := make([]string, 0, len(errs))
	for k := range errs {
		chaves = append(chaves, k)
	}
	sort.Strings(chaves)
	partes := []string{}
	for i, k := range chaves {
		if i == 3 {
			partes = append(partes, fmt.Sprintf("e mais %d", len(chaves)-3))
			break
		}
		partes = append(partes, fmt.Sprintf("%s: %v", k, errs[k]))
	}
	return strings.Join(partes, "; ")
}

// Promote aplica ao plano real os ajustes de demanda do cenário.
func (uc *ScenarioUseCase) Promote(ctx context.Context, id int64) (*PromoteResult, error) {
	if err := uc.canRead(ctx); err != nil {
		return nil, err
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.Promote(ctx, id, ator)
}
