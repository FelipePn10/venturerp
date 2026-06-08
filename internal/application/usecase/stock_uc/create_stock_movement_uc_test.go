package stock_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/google/uuid"
)

// fakeAuth embeds the interface so only the methods under test need overriding.
type fakeAuth struct {
	ports.AuthService
	can bool
	uid uuid.UUID
	err error
}

func (f fakeAuth) CanCreateStockMovement(context.Context) bool { return f.can }
func (f fakeAuth) UserID(context.Context) (uuid.UUID, error)   { return f.uid, f.err }

type fakeStockRepo struct {
	repository.StockRepository
	got    *entity.StockMovement
	called bool
	err    error
}

func (f *fakeStockRepo) CreateMovement(_ context.Context, m *entity.StockMovement) (*entity.StockMovement, error) {
	f.called = true
	f.got = m
	return m, f.err
}

func sampleDTO() request.CreateStockMovementDTO {
	return request.CreateStockMovementDTO{
		ItemCode:     1001,
		Mask:         "0001",
		WarehouseID:  7,
		MovementType: entity.MovementTypeIn,
		Quantity:     10,
		UnitPrice:    5.5,
		TotalPrice:   55,
	}
}

func TestCreateStockMovement_DeniedWhenUnauthorized(t *testing.T) {
	repo := &fakeStockRepo{}
	uc := CreateStockMovementUseCase{Repo: repo, Auth: fakeAuth{can: false}}

	_, err := uc.Execute(context.Background(), sampleDTO())
	if !errors.Is(err, errorsuc.ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	if repo.called {
		t.Fatal("repository must not be called when unauthorized")
	}
}

func TestCreateStockMovement_PropagatesUserIDError(t *testing.T) {
	repo := &fakeStockRepo{}
	wantErr := errors.New("no user in context")
	uc := CreateStockMovementUseCase{Repo: repo, Auth: fakeAuth{can: true, err: wantErr}}

	_, err := uc.Execute(context.Background(), sampleDTO())
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if repo.called {
		t.Fatal("repository must not be called when the actor is unknown")
	}
}

func TestCreateStockMovement_MapsDTOAndStampsActor(t *testing.T) {
	repo := &fakeStockRepo{}
	uid := uuid.New()
	uc := CreateStockMovementUseCase{Repo: repo, Auth: fakeAuth{can: true, uid: uid}}

	out, err := uc.Execute(context.Background(), sampleDTO())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.called || out == nil {
		t.Fatal("repository should have persisted the movement")
	}
	m := repo.got
	if m.ItemCode != 1001 || m.Mask != "0001" || m.WarehouseID != 7 {
		t.Fatalf("identity mapped wrong: %+v", m)
	}
	if m.MovementType != entity.MovementTypeIn || m.Quantity != 10 || m.UnitPrice != 5.5 || m.TotalPrice != 55 {
		t.Fatalf("movement values mapped wrong: %+v", m)
	}
	if m.CreatedBy != uid {
		t.Fatalf("CreatedBy = %v, want actor %v", m.CreatedBy, uid)
	}
	if m.ExpirationDate != nil {
		t.Fatalf("ExpirationDate should be nil when not provided, got %v", *m.ExpirationDate)
	}
}

func TestCreateStockMovement_ParsesExpirationDate(t *testing.T) {
	repo := &fakeStockRepo{}
	uc := CreateStockMovementUseCase{Repo: repo, Auth: fakeAuth{can: true, uid: uuid.New()}}

	dto := sampleDTO()
	exp := "2026-12-31"
	dto.ExpirationDate = &exp

	if _, err := uc.Execute(context.Background(), dto); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := repo.got.ExpirationDate
	if got == nil {
		t.Fatal("ExpirationDate should be set")
	}
	if got.Year() != 2026 || got.Month() != 12 || got.Day() != 31 {
		t.Fatalf("ExpirationDate parsed wrong: %v", got)
	}
}
