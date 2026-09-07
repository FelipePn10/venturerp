package configurator_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

// itemsFake devolve o item 900700 (código de negócio) apontando para a chave
// interna 3 — exatamente o cenário que deixava as características órfãs.
type itemsFake struct{}

func (itemsFake) FindItemByBusinessCode(_ context.Context, code valueobject.BusinessCode) (*itementity.Item, error) {
	if string(code) == "900700" {
		return &itementity.Item{Code: 3, BusinessCode: "900700"}, nil
	}
	return nil, itemrepo.ErrNotFound
}

func (itemsFake) FindItemByCode(_ context.Context, code valueobject.ItemCode) (*itementity.Item, error) {
	if int64(code) == 3 {
		return &itementity.Item{Code: 3, BusinessCode: "900700"}, nil
	}
	return nil, itemrepo.ErrNotFound
}

func TestResolveItemCodeTraduzCodigoDeNegocioParaChaveInterna(t *testing.T) {
	uc := New(nil).WithItems(itemsFake{})

	got, err := uc.ResolveItemCode(context.Background(), request.TextCode("900700"))
	if err != nil {
		t.Fatalf("resolver 900700: %v", err)
	}
	if got != 3 {
		t.Fatalf("código de negócio 900700 deveria virar a chave interna 3, veio %d", got)
	}
}

func TestResolveItemCodeAceitaAChaveInternaDireta(t *testing.T) {
	uc := New(nil).WithItems(itemsFake{})

	got, err := uc.ResolveNumericItemCode(context.Background(), 3)
	if err != nil {
		t.Fatalf("resolver chave interna: %v", err)
	}
	if got != 3 {
		t.Fatalf("a chave interna 3 deveria continuar valendo 3, veio %d", got)
	}
}

func TestResolveItemCodeRecusaItemInexistente(t *testing.T) {
	uc := New(nil).WithItems(itemsFake{})

	if _, err := uc.ResolveItemCode(context.Background(), request.TextCode("404404")); err == nil {
		t.Fatal("item inexistente deveria ser recusado em vez de gravar característica órfã")
	}
}

func TestResolveItemCodeRecusaCodigoVazio(t *testing.T) {
	uc := New(nil).WithItems(itemsFake{})

	if _, err := uc.ResolveNumericItemCode(context.Background(), 0); err == nil {
		t.Fatal("código zero deveria ser recusado")
	}
}
