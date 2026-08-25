package customer_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
)

type salesTablePriceContractRepository struct {
	repository.CustomerRepository
	requestedCode int64
	created       *entity.SalesTablePrice
}

func (r *salesTablePriceContractRepository) GetSalesTableByCode(_ context.Context, code int64) (*entity.SalesTable, error) {
	r.requestedCode = code
	return &entity.SalesTable{ID: 91, Code: code}, nil
}

func (r *salesTablePriceContractRepository) CreateSalesTablePrice(_ context.Context, price *entity.SalesTablePrice) (*entity.SalesTablePrice, error) {
	r.created = price
	return price, nil
}

func TestCreateSalesTablePriceResolvesTableIDFromSalesTableCode(t *testing.T) {
	repo := &salesTablePriceContractRepository{}
	uc := NewCustomerUseCase(repo)

	_, err := uc.CreateSalesTablePrice(context.Background(), request.CreateSalesTablePriceDTO{
		SalesTableCode: 7,
		ItemCode:       "4853",
		Price:          12.34,
	})
	if err != nil {
		t.Fatalf("CreateSalesTablePrice() retornou erro: %v", err)
	}
	if repo.requestedCode != 7 {
		t.Fatalf("código consultado = %d, esperado 7", repo.requestedCode)
	}
	if repo.created == nil || repo.created.SalesTableID != 91 {
		t.Fatalf("ID persistido = %v, esperado 91", repo.created)
	}
}
