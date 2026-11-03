package planning_params_uc

import (
	"context"
	mrpentity "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/planning_params/repository"
)

type UpdatePlanningParamUseCase struct {
	Repo repository.PlanningParamRepository
	Auth ports.AuthService
}

func (uc *UpdatePlanningParamUseCase) Execute(
	ctx context.Context,
	dto request.UpdatePlanningParamDTO,
) (*response.PlanningParamResponse, error) {
	if !uc.Auth.CanManagePlanningParams(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	autor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	dto.UpdatedBy = autor
	// 46/47 (janela de consumo pela venda), 48 (horizonte firme) e 49 (tempo
	// de segurança): número de dias.
	if nome, ok := parametrosEmDias[dto.ParamNumber]; ok {
		dias, errDias := strconv.Atoi(strings.TrimSpace(dto.Value))
		if errDias != nil || dias < 0 || dias > 365 {
			return nil, errorsuc.NewValidationError(nome + " é um número de dias entre 0 e 365")
		}
		dto.Value = strconv.Itoa(dias)
	}
	// 50: período de agrupamento das necessidades (lote por período).
	if dto.ParamNumber == 50 {
		v := strings.ToUpper(strings.TrimSpace(dto.Value))
		if !mrpentity.LotePeriodoValido(v) {
			return nil, errorsuc.NewValidationError("o período de lote é NENHUM, DIA, SEMANA ou MES")
		}
		dto.Value = v
	}
	p, err := uc.Repo.Update(ctx, dto.ParamNumber, dto.Value, dto.UpdatedBy)
	if err != nil {
		return nil, err
	}
	return toPlanningParamResponse(p), nil
}

var parametrosEmDias = map[int]string{
	46: "a janela de consumo",
	47: "a janela de consumo",
	48: "o horizonte firme",
	49: "o tempo de segurança",
}
