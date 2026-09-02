package sales_order_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/representative/entity"
	orderrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	"github.com/google/uuid"
)

type representativeValidationAuth struct{ ports.AuthService }

func (representativeValidationAuth) CanCreateSalesOrder(context.Context) bool      { return true }
func (representativeValidationAuth) CanUpdateSalesOrder(context.Context) bool      { return true }
func (representativeValidationAuth) EnterpriseCode(context.Context) (int64, error) { return 1, nil }
func (representativeValidationAuth) UserID(context.Context) (uuid.UUID, error) {
	return uuid.New(), nil
}

type untouchedSalesOrderRepo struct{ orderrepo.SalesOrderRepository }

type blockedRepresentativeRepository struct{}

func (blockedRepresentativeRepository) Get(_ context.Context, code int64) (*entity.Representative, error) {
	return &entity.Representative{Code: code, IsActive: true, Blocked: true}, nil
}

func TestCreateAndUpdateRejectBlockedRepresentativeBeforePersistence(t *testing.T) {
	code := int64(77)
	create := &CreateSalesOrderUseCase{Repo: untouchedSalesOrderRepo{}, Auth: representativeValidationAuth{}, Representatives: blockedRepresentativeRepository{}}
	if _, err := create.Execute(context.Background(), request.CreateSalesOrderDTO{EnterpriseCode: 1, RepresentativeCode: &code}); err == nil || !strings.Contains(err.Error(), "bloqueado") {
		t.Fatalf("criação deveria rejeitar representante bloqueado: %v", err)
	}
	update := &UpdateSalesOrderUseCase{Repo: untouchedSalesOrderRepo{}, Auth: representativeValidationAuth{}, Representatives: blockedRepresentativeRepository{}}
	if _, err := update.Execute(context.Background(), request.UpdateSalesOrderDTO{Code: 1, RepresentativeCode: &code}); err == nil || !strings.Contains(err.Error(), "bloqueado") {
		t.Fatalf("alteração deveria rejeitar representante bloqueado: %v", err)
	}
}
