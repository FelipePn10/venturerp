package planning_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

// historicoPadrao é quantas execuções a tela mostra quando não pede outro
// tamanho: um mês de janelas noturnas mais as rodadas manuais.
const historicoPadrao = 50

func (uc *AutoRunPlanningUseCase) enterprise(ctx context.Context) (int64, error) {
	if uc.Auth == nil {
		return 0, errorsuc.ErrUnauthorized
	}
	return uc.Auth.EnterpriseID(ctx)
}

// GetSettings devolve a parametrização da empresa da sessão. Empresa que nunca
// configurou recebe o padrão (desligada, 02:00).
func (uc *AutoRunPlanningUseCase) GetSettings(ctx context.Context) (*response.AutoRunSettingsResponse, error) {
	id, err := uc.enterprise(ctx)
	if err != nil {
		return nil, err
	}
	s, err := uc.Repo.Settings(ctx, id)
	if err != nil {
		return nil, err
	}
	return toAutoRunSettingsResponse(s), nil
}

// UpdateSettings grava a janela noturna. Quem salva passa a ser o responsável:
// a execução automática roda em nome dele.
func (uc *AutoRunPlanningUseCase) UpdateSettings(ctx context.Context, dto request.UpdateAutoRunSettingsDTO) (*response.AutoRunSettingsResponse, error) {
	if uc.Auth == nil || !uc.Auth.CanRunMRPCalculation(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	id, err := uc.enterprise(ctx)
	if err != nil {
		return nil, err
	}
	if dto.RunHour < 0 || dto.RunHour > 23 {
		return nil, errorsuc.NewValidationError("a hora da execução deve estar entre 0 e 23")
	}
	if dto.RunMinute < 0 || dto.RunMinute > 59 {
		return nil, errorsuc.NewValidationError("o minuto da execução deve estar entre 0 e 59")
	}
	if dto.InitialOrderNumber <= 0 {
		dto.InitialOrderNumber = 1
	}
	if dto.PlanCode < 0 {
		return nil, errorsuc.NewValidationError("plano de planejamento inválido")
	}
	if dto.IsEnabled && dto.PlanCode == 0 {
		return nil, errorsuc.NewValidationError("informe o plano de planejamento para ligar a execução automática")
	}
	if dto.PlanCode > 0 {
		ok, err := uc.Repo.PlanExists(ctx, id, dto.PlanCode)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errorsuc.NewNotFoundError("plano de planejamento não encontrado nesta empresa")
		}
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	s := AutoRunSettings{
		EnterpriseID:       id,
		IsEnabled:          dto.IsEnabled,
		RunHour:            dto.RunHour,
		RunMinute:          dto.RunMinute,
		PlanCode:           dto.PlanCode,
		InitialOrderNumber: dto.InitialOrderNumber,
		GenerateLLC:        dto.GenerateLLC,
		UpdatedBy:          &ator,
	}
	if err := uc.Repo.SaveSettings(ctx, s); err != nil {
		return nil, err
	}
	return uc.GetSettings(ctx)
}

// ListRuns devolve o histórico de execuções da empresa da sessão.
func (uc *AutoRunPlanningUseCase) ListRuns(ctx context.Context, limit int) ([]response.PlanningRunResponse, error) {
	id, err := uc.enterprise(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = historicoPadrao
	}
	runs, err := uc.Repo.ListRuns(ctx, id, limit)
	if err != nil {
		return nil, err
	}
	out := make([]response.PlanningRunResponse, 0, len(runs))
	for _, r := range runs {
		out = append(out, toPlanningRunResponse(r))
	}
	return out, nil
}

// RunNow executa agora, com a parametrização salva, em nome de quem clicou. A
// execução entra no mesmo histórico e respeita o mesmo cadeado da noturna.
func (uc *AutoRunPlanningUseCase) RunNow(ctx context.Context) (*response.PlanningRunResponse, error) {
	if uc.Auth == nil || !uc.Auth.CanRunMRPCalculation(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	id, err := uc.enterprise(ctx)
	if err != nil {
		return nil, err
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	s, err := uc.Repo.Settings(ctx, id)
	if err != nil {
		return nil, err
	}
	rec, err := uc.Run(ctx, *s, TriggerManual, &ator)
	if err != nil {
		return nil, err
	}
	out := toPlanningRunResponse(*rec)
	return &out, nil
}

func toAutoRunSettingsResponse(s *AutoRunSettings) *response.AutoRunSettingsResponse {
	out := &response.AutoRunSettingsResponse{
		IsEnabled:          s.IsEnabled,
		RunHour:            s.RunHour,
		RunMinute:          s.RunMinute,
		InitialOrderNumber: s.InitialOrderNumber,
		GenerateLLC:        s.GenerateLLC,
		LastSnapshotAt:     s.LastSnapshotAt,
		LastRunAt:          s.LastRunAt,
		LastRunStatus:      s.LastRunStatus,
	}
	if s.PlanCode > 0 {
		plano := s.PlanCode
		out.PlanCode = &plano
	}
	return out
}

func toPlanningRunResponse(r RunRecord) response.PlanningRunResponse {
	out := response.PlanningRunResponse{
		ID:              r.ID,
		PlanCode:        r.PlanCode,
		Trigger:         string(r.Trigger),
		SnapshotAt:      r.SnapshotAt,
		StartedAt:       r.StartedAt,
		FinishedAt:      r.FinishedAt,
		Status:          r.Status,
		ChangedItems:    r.ChangedItems,
		MRPOrders:       r.MRPOrders,
		CRPOverload:     r.CRPOverload,
		Viable:          r.Viable,
		Notes:           r.Notes,
		TriggeredByName: r.TriggeredByName,
	}
	if r.TriggeredBy != nil {
		quem := r.TriggeredBy.String()
		out.TriggeredBy = &quem
	}
	return out
}
