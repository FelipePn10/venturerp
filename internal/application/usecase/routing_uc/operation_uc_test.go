package routing_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
)

type operationRepoStub struct {
	repository.RoutingRepository
	operation *entity.Operation
	used      bool
}

func (s *operationRepoStub) GetOperationByID(context.Context, int64) (*entity.Operation, error) {
	copy := *s.operation
	return &copy, nil
}
func (s *operationRepoStub) OperationUsedInRoutes(context.Context, int64) (bool, error) {
	return s.used, nil
}
func (s *operationRepoStub) UpdateOperation(_ context.Context, operation *entity.Operation) (*entity.Operation, error) {
	s.operation = operation
	return operation, nil
}

func TestUpdateRejectsChangingUsedExternalOperationToInternal(t *testing.T) {
	repo := &operationRepoStub{operation: &entity.Operation{ID: 1, Name: "Zincar", Origin: entity.OriginThirdPart}, used: true}
	_, err := NewOperationUseCase(repo).Update(context.Background(), request.UpdateOperationDTO{ID: 1, Name: "Zincar", Origin: string(entity.OriginInternal), Situation: string(entity.SituationApproved), TimeUnit: entity.TimeUnitHour})
	if err == nil {
		t.Fatal("used external operation must preserve its origin")
	}
	if repo.operation.Origin != entity.OriginThirdPart {
		t.Fatalf("operation was mutated after validation failure: %+v", repo.operation)
	}
}
