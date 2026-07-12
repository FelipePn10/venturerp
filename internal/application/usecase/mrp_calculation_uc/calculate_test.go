package mrp_calculation_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
)

type calculationAuth struct{ ports.AuthService }

func (calculationAuth) CanRunMRPCalculation(context.Context) bool { return true }

type calculationServiceStub struct{ called bool }

func (s *calculationServiceStub) Calculate(context.Context, int64, int64, bool) (*entity.MRPCalculationLog, error) {
	s.called = true
	return &entity.MRPCalculationLog{}, nil
}
func (*calculationServiceStub) GenerateLLC(context.Context) error                    { return nil }
func (*calculationServiceStub) CalculateItemLLC(context.Context, int64) (int, error) { return 0, nil }
func (*calculationServiceStub) CalculateNetRequirements(context.Context, *entity.MRPInput) (*entity.MRPOutput, error) {
	return nil, nil
}
func (*calculationServiceStub) ExplodeStructure(context.Context, int64, string, float64, int) ([]*entity.MRPInput, error) {
	return nil, nil
}

func TestRunMRPCalculationRejectsInvalidPlanCodeBeforeCallingService(t *testing.T) {
	service := &calculationServiceStub{}
	uc := &RunMRPCalculationUseCase{Service: service, Auth: calculationAuth{}}

	_, err := uc.Execute(context.Background(), request.RunMRPCalculationDTO{PlanCode: 0})
	if err != ErrInvalidPlanCode {
		t.Fatalf("expected ErrInvalidPlanCode, got %v", err)
	}
	if service.called {
		t.Fatal("service must not be called for an invalid plan code")
	}
}

func TestRunMRPCalculationRejectsInvalidInitialOrderNumber(t *testing.T) {
	service := &calculationServiceStub{}
	uc := &RunMRPCalculationUseCase{Service: service, Auth: calculationAuth{}}
	_, err := uc.Execute(context.Background(), request.RunMRPCalculationDTO{PlanCode: 1})
	if err != ErrInvalidInitialOrderNumber {
		t.Fatalf("expected ErrInvalidInitialOrderNumber, got %v", err)
	}
	if service.called {
		t.Fatal("service must not be called for an invalid initial order number")
	}
}
