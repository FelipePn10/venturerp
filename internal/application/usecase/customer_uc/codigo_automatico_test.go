package customer_uc

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
)

// codigoRepo: o próximo código cresce a cada consulta e o primeiro já foi
// pego por um cadastro simultâneo.
type codigoRepo struct {
	customerrepo.CustomerRepository
	proximo int64
	usados  map[int64]bool
}

func (r *codigoRepo) NextCustomerCode(context.Context) (int64, error) {
	r.proximo++
	return r.proximo, nil
}
func (r *codigoRepo) GetCustomerByDocument(context.Context, string) (*entity.Customer, error) {
	return nil, errorsuc.NewNotFoundError("não há")
}
func (r *codigoRepo) CreateCustomer(_ context.Context, c *entity.Customer) (*entity.Customer, error) {
	if r.usados[c.Code] {
		return nil, errorsuc.NewConflictError(fmt.Sprintf("já existe cliente com o código %d nesta empresa", c.Code))
	}
	r.usados[c.Code] = true
	c.ID = c.Code
	return c, nil
}
func (r *codigoRepo) GetCustomerByCode(_ context.Context, code int64) (*entity.Customer, error) {
	return &entity.Customer{ID: code, Code: code, Name: "CLIENTE", DocumentType: entity.DocumentType("CNPJ"), DocumentNumber: "11444777000161"}, nil
}

func TestCodigoDoClienteEhAutomatico(t *testing.T) {
	repo := &codigoRepo{proximo: 40, usados: map[int64]bool{41: true}}
	uc := NewCustomerUseCase(repo)
	out, err := uc.CreateCustomer(context.Background(), request.CreateCustomerDTO{Name: "CLIENTE", DocumentType: "CNPJ", DocumentNumber: "11444777000161", CreatedBy: uuid.New()})
	if err != nil {
		t.Fatal(err)
	}
	if out.Code != 42 {
		t.Fatalf("código %d: o 41 já estava em uso, o próximo livre é 42", out.Code)
	}
}
