package delivery_reschedule_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	reschedulerepo "github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type capturePlanningRepo struct {
	command *reschedulerepo.BatchCommand
	result  *reschedulerepo.BatchResult
	err     error
}

func (r *capturePlanningRepo) Preview(context.Context, int64) ([]reschedulerepo.PlanningItem, error) {
	return []reschedulerepo.PlanningItem{{ItemCode: 1, CanReschedule: true}}, r.err
}
func (r *capturePlanningRepo) CreateBatch(_ context.Context, c reschedulerepo.BatchCommand) (*reschedulerepo.BatchResult, error) {
	r.command = &c
	if r.result == nil {
		r.result = &reschedulerepo.BatchResult{Codes: []int64{1, 2}}
	}
	return r.result, r.err
}

func TestPlanningBatchValidatesAndForwardsAtomicCommand(t *testing.T) {
	repo := &capturePlanningRepo{}
	uc := PlanningUseCase{Repo: repo, Auth: allowDeliveryRescheduleAuth{}}
	old := time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC)
	newDate := old.AddDate(0, 0, 2)
	result, err := uc.CreateBatch(context.Background(), request.IntegratedDeliveryRescheduleBatchDTO{SalesOrderCode: 10, IdempotencyKey: "batch-1", Items: []request.DeliveryRescheduleBatchItemDTO{{SalesOrderItemCode: 101, ItemCode: valueobject.ItemCode(11), OldDate: old, NewDate: newDate}, {SalesOrderItemCode: 102, ItemCode: valueobject.ItemCode(12), OldDate: old, NewDate: newDate}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Codes) != 2 || repo.command == nil || len(repo.command.Lines) != 2 {
		t.Fatalf("lote não foi encaminhado integralmente: %+v", result)
	}
	if repo.command.CreatedBy.String() != "00000000-0000-0000-0000-000000000042" {
		t.Fatalf("ator não veio do JWT: %s", repo.command.CreatedBy)
	}
}

func TestPlanningBatchRejectsDuplicateAndStaleIdempotency(t *testing.T) {
	old := time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC)
	item := request.DeliveryRescheduleBatchItemDTO{SalesOrderItemCode: 101, ItemCode: 1, OldDate: old, NewDate: old.AddDate(0, 0, 1)}
	uc := PlanningUseCase{Repo: &capturePlanningRepo{}, Auth: allowDeliveryRescheduleAuth{}}
	_, err := uc.CreateBatch(context.Background(), request.IntegratedDeliveryRescheduleBatchDTO{SalesOrderCode: 1, IdempotencyKey: "x", Items: []request.DeliveryRescheduleBatchItemDTO{item, item}})
	if err == nil || !strings.Contains(err.Error(), "mais de uma vez") {
		t.Fatalf("duplicidade não rejeitada: %v", err)
	}
	uc.Repo = &capturePlanningRepo{err: reschedulerepo.ErrBatchConflict}
	_, err = uc.CreateBatch(context.Background(), request.IntegratedDeliveryRescheduleBatchDTO{SalesOrderCode: 1, IdempotencyKey: "x", Items: []request.DeliveryRescheduleBatchItemDTO{item}})
	if err == nil || !strings.Contains(err.Error(), "outro conteúdo") {
		t.Fatalf("conflito idempotente não traduzido: %v", err)
	}
}
