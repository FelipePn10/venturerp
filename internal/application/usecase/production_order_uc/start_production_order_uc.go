package production_order_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type StartProductionOrderUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
	// Etapas gera as etapas da OF pelo roteiro vigente quando ela ainda não
	// tem (OF antiga, criada antes do roteiro). Opcional.
	Etapas geradorDeEtapas
}

type geradorDeEtapas interface {
	ExplodeRoute(ctx context.Context, orderID, routeID int64) ([]*response.ProductionOrderOperationResponse, error)
}
type productionOperationsChecker interface {
	HasProductionOperations(context.Context, int64) (bool, error)
}

// conferenciaDoInicio lê o parâmetro 52 e as pendências da OF (componente
// sem saldo, OF filha aberta).
type conferenciaDoInicio interface {
	ModoConferenciaInicio(context.Context) (string, error)
	PendenciasDoInicio(context.Context, int64) (entity.PendenciasDoInicio, error)
}

func (uc *StartProductionOrderUseCase) Execute(
	ctx context.Context,
	dto request.StartProductionOrderDTO,
) (*entity.ProductionOrder, error) {
	if !uc.Auth.CanReleaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.ID == 0 {
		return nil, errorsuc.NewValidationError("informe o identificador")
	}
	// O UPDATE do início só pega OF aberta: a que está em outra situação
	// ouviria "registro não encontrado" (ou a falta de etapas) — dizer qual.
	atual, err := uc.Repo.GetByCode(ctx, dto.ID)
	if err != nil {
		return nil, err
	}
	if atual.Status != entity.StatusOpen {
		return nil, errorsuc.NewConflictError(fmt.Sprintf("a OF %d já está %s; só OF aberta pode ser iniciada", atual.OrderNumber, situacaoDaOF(atual.Status)))
	}
	if checker, ok := uc.Repo.(productionOperationsChecker); ok {
		hasOperations, err := checker.HasProductionOperations(ctx, dto.ID)
		if err != nil {
			return nil, err
		}
		// A OF nasce com as etapas do roteiro; a que não tem (criada antes do
		// roteiro existir) ganha agora, sem o usuário precisar explodir à mão.
		if !hasOperations && uc.Etapas != nil {
			if _, err := uc.Etapas.ExplodeRoute(ctx, dto.ID, 0); err != nil {
				return nil, errorsuc.NewValidationError("a OF não tem etapas e o roteiro não pôde ser aplicado: " + err.Error() + " — cadastre e aprove o roteiro do item")
			}
			if hasOperations, err = checker.HasProductionOperations(ctx, dto.ID); err != nil {
				return nil, err
			}
		}
		if !hasOperations {
			return nil, errorsuc.NewValidationError("a ordem de produção não possui etapas: o item não tem roteiro aprovado; cadastre e aprove o roteiro antes de iniciar")
		}
	}

	avisos, err := conferirInicio(ctx, uc.Repo, dto.ID)
	if err != nil {
		return nil, err
	}

	// Default to the real start moment (today) when no valid date is sent,
	// instead of persisting the zero time (0001-01-01).
	startDate := datetime.ParseDateOrDefault(dto.StartDate, time.Now())

	order, err := uc.Repo.Start(ctx, dto.ID, startDate)
	if err != nil {
		return nil, err
	}
	order.AvisosDoInicio = avisos
	return order, nil
}

func situacaoDaOF(s entity.ProductionOrderStatus) string {
	switch s {
	case entity.StatusInProgress:
		return "em andamento (já foi iniciada)"
	case entity.StatusCompleted:
		return "concluída"
	case entity.StatusClosed:
		return "encerrada"
	case entity.StatusCancelled:
		return "cancelada"
	}
	return "na situação " + string(s)
}

// conferirInicio aplica o parâmetro 52: BLOQUEAR com pendência recusa o
// início; AVISAR devolve os avisos; NAO não confere.
func conferirInicio(ctx context.Context, repo any, orderID int64) ([]string, error) {
	c, ok := repo.(conferenciaDoInicio)
	if !ok {
		return nil, nil
	}
	modo, err := c.ModoConferenciaInicio(ctx)
	if err != nil {
		return nil, err
	}
	if modo == entity.ConferenciaInicioNao {
		return nil, nil
	}
	pendencias, err := c.PendenciasDoInicio(ctx, orderID)
	if err != nil {
		return nil, err
	}
	recusa, avisos := entity.AvaliarInicio(modo, pendencias)
	if recusa != "" {
		return nil, errorsuc.NewValidationError(recusa)
	}
	return avisos, nil
}
