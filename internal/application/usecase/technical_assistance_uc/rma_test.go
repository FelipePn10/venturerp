package technical_assistance_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity"
	"github.com/google/uuid"
)

type fakeRMARepository struct{ value *entity.RMA }

func (f *fakeRMARepository) CreateRMA(_ context.Context, tenant int64, value *entity.RMA) (*entity.RMA, error) {
	value.Code = 10
	value.EnterpriseID = tenant
	value.CreatedAt = time.Now()
	value.UpdatedAt = value.CreatedAt
	f.value = value
	return value, nil
}
func (f *fakeRMARepository) GetRMA(_ context.Context, tenant, code int64) (*entity.RMA, error) {
	return f.value, nil
}
func (f *fakeRMARepository) ListRMAsByCall(context.Context, int64, int64) ([]*entity.RMA, error) {
	return []*entity.RMA{f.value}, nil
}
func (f *fakeRMARepository) TransitionRMA(_ context.Context, _ int64, _ int64, status string, _ *string, _ *string, _ uuid.UUID, changes *entity.RMA) (*entity.RMA, error) {
	changes.Status = status
	f.value = changes
	return changes, nil
}

func TestRMARequiresIdempotencyAndUsesAuthenticatedActor(t *testing.T) {
	repo := &fakeRMARepository{}
	uc := &UseCase{Repo: &fakeTARepo{}, RMAs: repo, Auth: taAllowAuth{}}
	dto := request.CreateTechnicalAssistanceRMADTO{CallCode: 1, ReasonCode: "DEFEITO", EligibilityStatus: "ELEGIVEL", EligibilityReason: "dentro da garantia", SLADueAt: "2026-09-10T12:00:00Z", Items: []request.TechnicalAssistanceRMAItemDTO{{CallItemCode: 2, ItemCode: 3, Quantity: 1}}}
	if _, err := uc.CreateRMA(context.Background(), dto); err == nil || !strings.Contains(err.Error(), "idempotência") {
		t.Fatalf("esperava chave obrigatória, recebeu %v", err)
	}
	dto.IdempotencyKey = "rma-1"
	got, err := uc.CreateRMA(context.Background(), dto)
	if err != nil {
		t.Fatal(err)
	}
	if got.Code != 10 || repo.value.CreatedBy != uuid.MustParse("00000000-0000-0000-0000-000000000042") {
		t.Fatalf("RMA/ator inesperado: %+v", repo.value)
	}
}

func TestRMAStateMachineAndRequiredEvidence(t *testing.T) {
	repo := &fakeRMARepository{value: &entity.RMA{Code: 10, EnterpriseID: 1, CallCode: 1, Status: "SOLICITADO", EligibilityStatus: "ELEGIVEL", SLADueAt: time.Now()}}
	uc := &UseCase{Repo: &fakeTARepo{}, RMAs: repo, Auth: taAllowAuth{}}
	if _, err := uc.TransitionRMA(context.Background(), 10, request.TransitionTechnicalAssistanceRMADTO{Status: "RECEBIDO"}); err == nil {
		t.Fatal("transição fora de ordem deveria falhar")
	}
	if _, err := uc.TransitionRMA(context.Background(), 10, request.TransitionTechnicalAssistanceRMADTO{Status: "AUTORIZADO"}); err == nil || !strings.Contains(err.Error(), "autorização") {
		t.Fatalf("esperava autorização obrigatória: %v", err)
	}
	auth := "AUT-1"
	got, err := uc.TransitionRMA(context.Background(), 10, request.TransitionTechnicalAssistanceRMADTO{Status: "AUTORIZADO", AuthorizationNumber: &auth})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "AUTORIZADO" {
		t.Fatalf("status=%s", got.Status)
	}
}
