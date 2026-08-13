package production_order_uc

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/google/uuid"
)

type scannerAuth struct{ ports.AuthService }

func (scannerAuth) CanReleaseOrder(context.Context) bool        { return true }
func (scannerAuth) CanCreatePlannedOrder(context.Context) bool  { return true }
func (scannerAuth) EnterpriseID(context.Context) (int64, error) { return 42, nil }
func (scannerAuth) UserID(context.Context) (uuid.UUID, error) {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111"), nil
}

type scannerRepo struct {
	created *entity.ScanToken
	command *entity.ScanCommand
}

func (r *scannerRepo) CreateScanToken(_ context.Context, t *entity.ScanToken) error {
	r.created = t
	return nil
}
func (r *scannerRepo) ExecuteScan(_ context.Context, c entity.ScanCommand) (*entity.ScanResult, error) {
	r.command = &c
	return &entity.ScanResult{ProductionOrderID: 9}, nil
}
func (*scannerRepo) RevokeScanToken(context.Context, int64, []byte, time.Time) error { return nil }

func TestScannerCreatesOpaqueTokenWithoutEmbeddingOrder(t *testing.T) {
	repo := &scannerRepo{}
	uc := ProductionScannerUseCase{Repo: repo, Auth: scannerAuth{}}
	result, err := uc.CreateToken(context.Background(), request.CreateProductionScanTokenDTO{ProductionOrderID: 987654})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Token) < 40 || result.Token[:4] != "OF1_" {
		t.Fatalf("token opaco inesperado: %q", result.Token)
	}
	if repo.created == nil || repo.created.EnterpriseID != 42 || repo.created.ProductionOrderID != 987654 || len(repo.created.TokenHash) != 32 {
		t.Fatalf("registro incorreto: %#v", repo.created)
	}
}

func TestScannerValidatesOperatorScrapAndPortugueseAction(t *testing.T) {
	repo := &scannerRepo{}
	uc := ProductionScannerUseCase{Repo: repo, Auth: scannerAuth{}}
	base := request.ProductionScanDTO{Token: "OF1_token", Action: "APONTAR", IdempotencyKey: "k1", DeviceID: "coletor-1", GoodQuantity: "2.500", ScrapQuantity: "0.125"}
	if _, err := uc.Scan(context.Background(), base); err == nil {
		t.Fatal("esperava erro de operador")
	}
	employee := int64(7)
	base.EmployeeID = &employee
	if _, err := uc.Scan(context.Background(), base); err == nil {
		t.Fatal("esperava erro de motivo do refugo")
	}
	reason := "TRINCA"
	base.ScrapReason = &reason
	if _, err := uc.Scan(context.Background(), base); err != nil {
		t.Fatal(err)
	}
	if repo.command == nil || repo.command.Action != entity.ScanAppoint || repo.command.GoodQuantity.String() != "2.5" {
		t.Fatalf("comando incorreto: %#v", repo.command)
	}
}
