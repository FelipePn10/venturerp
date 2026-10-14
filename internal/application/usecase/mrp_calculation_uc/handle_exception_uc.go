package mrp_calculation_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	plannedentity "github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
)

// exceptionStore é a parte do repositório do MRP que o tratamento usa.
type exceptionStore interface {
	GetException(ctx context.Context, code int64) (*entity.MRPExceptionMessage, error)
	MarkExceptionHandled(ctx context.Context, code int64, action string, note *string, by uuid.UUID) (*entity.MRPExceptionMessage, error)
}

// plannedOrderActions é o que a exceção pode fazer com a ordem planejada de
// origem.
type plannedOrderActions interface {
	Reschedule(ctx context.Context, code int64, needDate time.Time) (*plannedentity.PlannedOrder, error)
	Cancel(ctx context.Context, code int64) (*plannedentity.PlannedOrder, error)
	ActiveProductionOrderNumber(ctx context.Context, code int64) (*int64, error)
}

// HandleMRPExceptionUseCase executa a ação sugerida por uma exceção do MRP
// direto da mensagem — o que o SAP faz no MD06/MD07 e o Oracle no workbench de
// exceções: reprogramar a ordem para a data em que o cálculo precisa dela,
// cancelar a ordem sem demanda ou registrar que a mensagem foi analisada.
//
// A ordem é alterada antes de marcar a exceção: se a marcação falhar, a ordem
// já está certa e a mensagem só continua pendente até o próximo cálculo (que
// não a gera de novo). O contrário — marcar e não alterar — esconderia uma
// falta real.
type HandleMRPExceptionUseCase struct {
	Exceptions exceptionStore
	Orders     plannedOrderActions
	Auth       ports.AuthService
	// Now permite fixar o relógio nos testes.
	Now func() time.Time
}

const (
	acaoReprogramar = "REPROGRAMAR"
	acaoCancelar    = "CANCELAR"
	acaoTratar      = "TRATAR"
)

func (uc *HandleMRPExceptionUseCase) hoje() time.Time {
	t := time.Now()
	if uc.Now != nil {
		t = uc.Now()
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func (uc *HandleMRPExceptionUseCase) Execute(ctx context.Context, code int64, dto request.HandleMRPExceptionDTO) (*response.MRPExceptionMessageResponse, error) {
	if !uc.Auth.CanRunMRPCalculation(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	acao := strings.ToUpper(strings.TrimSpace(dto.Action))
	var nota *string
	if dto.Note != nil {
		if n := strings.TrimSpace(*dto.Note); n != "" {
			nota = &n
		}
	}

	exc, err := uc.Exceptions.GetException(ctx, code)
	if err != nil {
		return nil, err
	}
	if exc.HandledAt != nil {
		return nil, errorsuc.NewConflictError("esta exceção já foi tratada")
	}
	if checker, ok := uc.Exceptions.(interface {
		IsScenarioPlan(ctx context.Context, planCode int64) (bool, error)
	}); ok {
		cenario, err := checker.IsScenarioPlan(ctx, exc.PlanCode)
		if err != nil {
			return nil, err
		}
		if cenario {
			return nil, errorsuc.NewValidationError("exceção de cenário de simulação não altera ordens reais")
		}
	}

	var registro string
	switch acao {
	case acaoReprogramar:
		if exc.MessageType != entity.ExceptionExpedite && exc.MessageType != entity.ExceptionRescheduleIn && exc.MessageType != entity.ExceptionRescheduleOut {
			return nil, errorsuc.NewValidationError("só exceções de antecipar, acelerar ou postergar reprogramam a ordem")
		}
		ordem, err := uc.ordemDeOrigem(ctx, exc)
		if err != nil {
			return nil, err
		}
		data, err := uc.novaData(exc, dto.NewDate)
		if err != nil {
			return nil, err
		}
		if _, err := uc.Orders.Reschedule(ctx, ordem, data); err != nil {
			return nil, err
		}
		registro = entity.ExceptionHandledRescheduled
		if nota == nil {
			n := fmt.Sprintf("necessidade reprogramada para %s", data.Format("02/01/2006"))
			nota = &n
		}
	case acaoCancelar:
		if exc.MessageType != entity.ExceptionCancel {
			return nil, errorsuc.NewValidationError("só a exceção de cancelar ordem cancela a ordem planejada")
		}
		ordem, err := uc.ordemDeOrigem(ctx, exc)
		if err != nil {
			return nil, err
		}
		if _, err := uc.Orders.Cancel(ctx, ordem); err != nil {
			return nil, err
		}
		registro = entity.ExceptionHandledCancelled
	case acaoTratar:
		if nota == nil {
			return nil, errorsuc.NewValidationError("informe o que foi feito ou por que a mensagem pode ser ignorada")
		}
		registro = entity.ExceptionHandledAcknowledged
	default:
		return nil, errorsuc.NewValidationError("ação inválida: use REPROGRAMAR, CANCELAR ou TRATAR")
	}

	marcada, err := uc.Exceptions.MarkExceptionHandled(ctx, code, registro, nota, ator)
	if err != nil {
		return nil, err
	}
	return toMRPExceptionResponse(marcada), nil
}

// ordemDeOrigem devolve a ordem planejada que gerou a exceção, recusando a que
// já virou OF: a partir daí quem manda é a OF, e mexer só na planejada
// deixaria as duas com datas diferentes.
func (uc *HandleMRPExceptionUseCase) ordemDeOrigem(ctx context.Context, exc *entity.MRPExceptionMessage) (int64, error) {
	if exc.SourceCode == nil || exc.SourceType == nil || *exc.SourceType != "PLANNED_ORDER" {
		return 0, errorsuc.NewValidationError("a exceção não aponta uma ordem planejada para alterar")
	}
	of, err := uc.Orders.ActiveProductionOrderNumber(ctx, *exc.SourceCode)
	if err != nil {
		return 0, err
	}
	if of != nil {
		return 0, errorsuc.NewValidationError(fmt.Sprintf("a ordem planejada já gerou a ordem de fabricação %d; ajuste a ordem de fabricação", *of))
	}
	return *exc.SourceCode, nil
}

// novaData: a informada pelo planejador ou a sugerida pelo cálculo. Uma data
// sugerida no passado (acelerar o que já devia ter chegado) vira hoje — não se
// planeja para ontem.
func (uc *HandleMRPExceptionUseCase) novaData(exc *entity.MRPExceptionMessage, informada *string) (time.Time, error) {
	hoje := uc.hoje()
	if informada != nil && strings.TrimSpace(*informada) != "" {
		d, err := time.Parse("2006-01-02", strings.TrimSpace(*informada))
		if err != nil {
			return time.Time{}, errorsuc.NewValidationError("data inválida: use o formato ano-mês-dia")
		}
		if d.Before(hoje) {
			return time.Time{}, errorsuc.NewValidationError("a nova data não pode ser anterior a hoje")
		}
		return d, nil
	}
	if exc.SuggestedDate == nil {
		return time.Time{}, errorsuc.NewValidationError("a exceção não tem data sugerida; rode o MRP de novo ou informe a nova data")
	}
	d := time.Date(exc.SuggestedDate.Year(), exc.SuggestedDate.Month(), exc.SuggestedDate.Day(), 0, 0, 0, 0, time.UTC)
	if d.Before(hoje) {
		d = hoje
	}
	return d, nil
}
