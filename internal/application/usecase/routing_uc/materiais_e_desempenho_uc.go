package routing_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
)

// MateriaisEDesempenhoRepo é o que este caso de uso precisa do repositório.
type MateriaisEDesempenhoRepo interface {
	GetRouteByID(ctx context.Context, id int64) (*entity.ManufacturingRoute, error)
	GetRouteOperations(ctx context.Context, routeID int64) ([]*entity.RouteOperation, error)
	ListRouteMaterials(ctx context.Context, routeID int64) ([]entity.RouteMaterial, error)
	SetMaterialStep(ctx context.Context, routeID, structureID int64, step *int16) error
	RouteStepActuals(ctx context.Context, routeID int64, dias int) ([]entity.StepActuals, error)
}

// MateriaisEDesempenhoUseCase cobre duas perguntas do roteiro que olham para
// fora dele: em que etapa cada componente entra, e quanto cada etapa realmente
// leva nas ordens concluídas.
type MateriaisEDesempenhoUseCase struct {
	Repo MateriaisEDesempenhoRepo
}

// RouteMaterialResponse é um componente da estrutura e a etapa em que entra.
type RouteMaterialResponse struct {
	StructureID  int64   `json:"structure_id"`
	ChildCode    string  `json:"child_code"`
	ChildName    string  `json:"child_name"`
	Quantity     float64 `json:"quantity"`
	Unit         string  `json:"unit"`
	StepSequence *int16  `json:"step_sequence,omitempty"`
	StepName     string  `json:"step_name,omitempty"`
	ParentMask   *string `json:"parent_mask,omitempty"`
	IsCoproduct  bool    `json:"is_coproduct"`
	// StepMissing: o componente aponta para uma etapa que não existe neste
	// roteiro (removida, ou só existe em outra alternativa). O MRP o trata
	// como "no início da ordem"; a tela avisa.
	StepMissing bool `json:"step_missing"`
}

func (uc *MateriaisEDesempenhoUseCase) ListMaterials(ctx context.Context, routeID int64) ([]RouteMaterialResponse, error) {
	rt, err := uc.Repo.GetRouteByID(ctx, routeID)
	if err != nil {
		return nil, errorsuc.NewNotFoundError("roteiro não encontrado nesta empresa")
	}
	if rt.IsTemplate {
		return []RouteMaterialResponse{}, nil // modelo não tem item, logo não tem estrutura
	}
	materiais, err := uc.Repo.ListRouteMaterials(ctx, routeID)
	if err != nil {
		return nil, err
	}
	nomes, err := uc.etapasPorSequencia(ctx, routeID)
	if err != nil {
		return nil, err
	}
	out := make([]RouteMaterialResponse, 0, len(materiais))
	for _, m := range materiais {
		codigo := m.ChildBusiness
		if codigo == "" {
			codigo = fmt.Sprint(m.ChildCode)
		}
		r := RouteMaterialResponse{
			StructureID: m.StructureID, ChildCode: codigo, ChildName: m.ChildName,
			Quantity: m.Quantity.InexactFloat64(), Unit: m.Unit, StepSequence: m.StepSequence,
			ParentMask: m.ParentMask, IsCoproduct: m.IsCoproduct,
		}
		if m.StepSequence != nil {
			nome, existe := nomes[*m.StepSequence]
			r.StepName, r.StepMissing = nome, !existe
		}
		out = append(out, r)
	}
	return out, nil
}

// SetMaterialStep grava a etapa do componente. A etapa precisa existir neste
// roteiro: gravar uma sequência inexistente faria o MRP datar o componente
// pelo início da ordem sem ninguém perceber.
func (uc *MateriaisEDesempenhoUseCase) SetMaterialStep(ctx context.Context, routeID int64, dto request.SetMaterialStepDTO) error {
	if dto.StructureID <= 0 {
		return errorsuc.NewValidationError("informe o componente da estrutura")
	}
	if dto.Sequence != nil {
		nomes, err := uc.etapasPorSequencia(ctx, routeID)
		if err != nil {
			return err
		}
		if _, ok := nomes[*dto.Sequence]; !ok {
			return errorsuc.NewValidationError(fmt.Sprintf("o roteiro não tem etapa na sequência %d", *dto.Sequence))
		}
	}
	return uc.Repo.SetMaterialStep(ctx, routeID, dto.StructureID, dto.Sequence)
}

func (uc *MateriaisEDesempenhoUseCase) etapasPorSequencia(ctx context.Context, routeID int64) (map[int16]string, error) {
	ops, err := uc.Repo.GetRouteOperations(ctx, routeID)
	if err != nil {
		return nil, err
	}
	out := make(map[int16]string, len(ops))
	for _, op := range ops {
		out[op.Sequence] = op.OperationName
	}
	return out, nil
}

// StepPerformanceResponse: padrão × real de uma etapa.
type StepPerformanceResponse struct {
	RouteOperationID  int64      `json:"route_operation_id"`
	Sequence          int16      `json:"sequence"`
	OperationName     string     `json:"operation_name"`
	Orders            int        `json:"orders"`
	PlannedHours      float64    `json:"planned_hours"`
	ActualHours       float64    `json:"actual_hours"`
	DeviationPct      float64    `json:"deviation_pct"`
	PlannedPerPiece   float64    `json:"planned_hours_per_piece"`
	ActualPerPiece    float64    `json:"actual_hours_per_piece"`
	CurrentRunHours   float64    `json:"current_run_hours"`
	SuggestedRunHours *float64   `json:"suggested_run_hours,omitempty"`
	Reason            string     `json:"reason"`
	LastCompleted     *time.Time `json:"last_completed,omitempty"`
}

// Performance compara, etapa a etapa, o tempo padrão com o apontado nas OPs
// concluídas nos últimos `dias` (padrão 180). Nenhum ERP grande traz isso na
// tela do roteiro: lá é relatório separado, e o padrão envelhece sem ninguém
// ver.
func (uc *MateriaisEDesempenhoUseCase) Performance(ctx context.Context, routeID int64, dias int) ([]StepPerformanceResponse, error) {
	if dias <= 0 || dias > 3650 {
		dias = 180
	}
	if _, err := uc.Repo.GetRouteByID(ctx, routeID); err != nil {
		return nil, errorsuc.NewNotFoundError("roteiro não encontrado nesta empresa")
	}
	ops, err := uc.Repo.GetRouteOperations(ctx, routeID)
	if err != nil {
		return nil, err
	}
	reais, err := uc.Repo.RouteStepActuals(ctx, routeID, dias)
	if err != nil {
		return nil, err
	}
	porEtapa := make(map[int64]entity.StepActuals, len(reais))
	for _, a := range reais {
		porEtapa[a.RouteOperationID] = a
	}
	out := make([]StepPerformanceResponse, 0, len(ops))
	for _, op := range ops {
		a, ok := porEtapa[op.ID]
		r := StepPerformanceResponse{
			RouteOperationID: op.ID, Sequence: op.Sequence, OperationName: op.OperationName,
			CurrentRunHours: op.EffTime.Run,
		}
		if !op.ConsomeTempo() {
			r.Reason = "etapa fantasma ou inativa: não é executada, não há o que comparar"
			out = append(out, r)
			continue
		}
		if op.OperationOrigin == entity.OriginExternal || op.OperationOrigin == entity.OriginThirdPart {
			r.Reason = "etapa de terceiro: o prazo é do fornecedor, não há hora apontada"
			out = append(out, r)
			continue
		}
		if !ok {
			r.Reason = "nenhuma ordem concluída apontou esta etapa no período"
			out = append(out, r)
			continue
		}
		c := entity.CompararTempo(a, op.EffTime)
		r.Orders, r.PlannedHours, r.ActualHours, r.DeviationPct = c.Orders, c.PlannedHours, c.ActualHours, c.DeviationPct
		r.PlannedPerPiece, r.ActualPerPiece = c.PlannedPerPiece, c.ActualPerPiece
		r.SuggestedRunHours, r.Reason, r.LastCompleted = c.SuggestedRunHours, c.SuggestionReason, c.LastCompleted
		out = append(out, r)
	}
	return out, nil
}
