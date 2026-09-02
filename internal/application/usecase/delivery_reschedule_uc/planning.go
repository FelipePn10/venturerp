package delivery_reschedule_uc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	reschedulerepo "github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/repository"
	"github.com/google/uuid"
)

type PlanningUseCase struct {
	Repo reschedulerepo.PlanningRepository
	Auth ports.AuthService
}

func (uc *PlanningUseCase) Preview(ctx context.Context, orderCode int64) ([]reschedulerepo.PlanningItem, error) {
	if !uc.Auth.CanListDeliveryReschedule(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if orderCode <= 0 {
		return nil, errorsuc.NewValidationError("código do pedido inválido")
	}
	items, err := uc.Repo.Preview(ctx, orderCode)
	if errors.Is(err, reschedulerepo.ErrOrderNotFound) {
		return nil, errorsuc.NewValidationError("pedido não encontrado na empresa autenticada")
	}
	return items, err
}

func (uc *PlanningUseCase) CreateBatch(ctx context.Context, dto request.IntegratedDeliveryRescheduleBatchDTO) (*reschedulerepo.BatchResult, error) {
	if !uc.Auth.CanCreateDeliveryReschedule(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.SalesOrderCode <= 0 {
		return nil, errorsuc.NewValidationError("código do pedido inválido")
	}
	dto.IdempotencyKey = strings.TrimSpace(dto.IdempotencyKey)
	if dto.IdempotencyKey == "" || len(dto.IdempotencyKey) > 100 {
		return nil, errorsuc.NewValidationError("chave de idempotência deve ser informada e possuir até 100 caracteres")
	}
	if len(dto.Items) == 0 {
		return nil, errorsuc.NewValidationError("selecione ao menos um item para reprogramar")
	}
	seen := map[int64]bool{}
	lines := make([]reschedulerepo.BatchLine, 0, len(dto.Items))
	for _, item := range dto.Items {
		if item.SalesOrderItemCode <= 0 {
			return nil, errorsuc.NewValidationError("o código da linha do pedido é obrigatório")
		}
		code := int64(item.ItemCode)
		if code <= 0 {
			return nil, errorsuc.NewValidationError("código do item inválido")
		}
		if seen[item.SalesOrderItemCode] {
			return nil, errorsuc.NewValidationError("uma linha não pode aparecer mais de uma vez no lote")
		}
		seen[item.SalesOrderItemCode] = true
		if item.NewDate.IsZero() || item.OldDate.IsZero() {
			return nil, errorsuc.NewValidationError("datas atual e nova são obrigatórias")
		}
		if !item.NewDate.After(item.OldDate) {
			return nil, errorsuc.NewValidationError("a nova data deve ser posterior à data atual")
		}
		lines = append(lines, reschedulerepo.BatchLine{SalesOrderItemCode: item.SalesOrderItemCode, ItemCode: item.ItemCode, OldDate: item.OldDate, NewDate: item.NewDate, Reason: item.Reason})
	}
	payload, _ := json.Marshal(dto)
	sum := sha256.Sum256(payload)
	result, err := uc.Repo.CreateBatch(ctx, reschedulerepo.BatchCommand{ID: uuid.New(), IdempotencyKey: dto.IdempotencyKey, PayloadHash: hex.EncodeToString(sum[:]), SalesOrderCode: dto.SalesOrderCode, CreatedBy: actor, Lines: lines})
	if errors.Is(err, reschedulerepo.ErrBatchConflict) {
		return nil, errorsuc.NewValidationError("a chave de idempotência já foi usada com outro conteúdo")
	}
	if errors.Is(err, reschedulerepo.ErrOrderNotFound) {
		return nil, errorsuc.NewValidationError("pedido não encontrado na empresa autenticada")
	}
	if errors.Is(err, reschedulerepo.ErrLinePrecondition) {
		return nil, errorsuc.NewValidationError(strings.TrimPrefix(err.Error(), reschedulerepo.ErrLinePrecondition.Error()+": "))
	}
	return result, err
}
