package delivery_reschedule_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/repository"
)

type allowDeliveryRescheduleAuth struct{ ports.AuthService }

func (allowDeliveryRescheduleAuth) CanCreateDeliveryReschedule(context.Context) bool { return true }

type captureDeliveryRescheduleRepository struct {
	repository.DeliveryRescheduleRepository
	created   *entity.DeliveryReschedule
	inputCode int64
}

func (r *captureDeliveryRescheduleRepository) Create(_ context.Context, value *entity.DeliveryReschedule) (*entity.DeliveryReschedule, error) {
	r.created = value
	r.inputCode = value.Code
	value.Code = 42
	return value, nil
}

func TestCreateDeliveryRescheduleLeavesCodeGenerationToRepository(t *testing.T) {
	repo := &captureDeliveryRescheduleRepository{}
	uc := &CreateDeliveryRescheduleUseCase{Repo: repo, Auth: allowDeliveryRescheduleAuth{}}

	result, err := uc.Execute(context.Background(), request.CreateDeliveryRescheduleDTO{SalesOrderCode: 10, ItemCode: 4853})
	if err != nil {
		t.Fatalf("Execute() retornou erro: %v", err)
	}
	if repo.created == nil {
		t.Fatal("repositório não recebeu a reprogramação")
	}
	if repo.inputCode != 0 {
		t.Fatalf("use case enviou código %d; a geração deve ficar no repositório", repo.inputCode)
	}
	if result.Code != 42 {
		t.Fatalf("código retornado = %d, esperado o código gerado pelo repositório", result.Code)
	}
}
