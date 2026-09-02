package sales_goal_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	representativeentity "github.com/FelipePn10/panossoerp/internal/domain/representative/entity"
	goalrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_goal/repository"
)

type goalRepresentativeAuth struct{ ports.AuthService }

func (goalRepresentativeAuth) CanCreateSalesOrder(context.Context) bool { return true }
func (goalRepresentativeAuth) CanUpdateSalesOrder(context.Context) bool { return true }

type untouchedGoalRepo struct{ goalrepo.Repository }

type blockedGoalRepresentativeRepo struct{}

func (blockedGoalRepresentativeRepo) Get(_ context.Context, code int64) (*representativeentity.Representative, error) {
	return &representativeentity.Representative{Code: code, IsActive: true, Blocked: true}, nil
}

func TestGoalCreateAndUpdateRejectBlockedRepresentative(t *testing.T) {
	uc := &UseCase{Repo: untouchedGoalRepo{}, Auth: goalRepresentativeAuth{}, Representatives: blockedGoalRepresentativeRepo{}}
	if _, err := uc.CreateGoal(context.Background(), request.CreateSalesGoalDTO{RepresentativeCode: 5}); err == nil || !strings.Contains(err.Error(), "bloqueado") {
		t.Fatalf("criação de meta deveria rejeitar representante bloqueado: %v", err)
	}
	if _, err := uc.UpdateGoal(context.Background(), request.UpdateSalesGoalDTO{RepresentativeCode: 5}); err == nil || !strings.Contains(err.Error(), "bloqueado") {
		t.Fatalf("alteração de meta deveria rejeitar representante bloqueado: %v", err)
	}
}
