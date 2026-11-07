package service

import (
	"context"
	"testing"

	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type itensFixos struct {
	itemrepo.ItemRepository
	itens map[int64]*itementity.Item
}

func (f itensFixos) FindItemByCode(_ context.Context, c valueobject.ItemCode) (*itementity.Item, error) {
	return f.itens[int64(c)], nil
}

func TestCadastroDecideQuemEntraNoMRPEOPrazo(t *testing.T) {
	sete, dez := 7, 10
	ligado := &itementity.Item{Planning: itementity.Planning{Active: true, PurchaseLeadTimeDays: &sete, ManufacturingLeadTimeDays: &dez}}
	ligado.Engineering.Type = 1
	s := &MRPServiceImpl{ItemRepo: itensFixos{itens: map[int64]*itementity.Item{
		1: ligado,
		2: {Planning: itementity.Planning{Active: false}},
		3: {Planning: itementity.Planning{Active: true}, IsPrototype: true},
		4: {Planning: itementity.Planning{Active: true}, IsScrap: true},
	}}}
	cache := map[int64]*cachedItemMRP{}
	for item, fora := range map[int64]bool{1: false, 2: true, 3: true, 4: true} {
		c, err := s.ensureItemCache(context.Background(), cache, item)
		if err != nil {
			t.Fatal(err)
		}
		if c.foraDoMRP != fora {
			t.Fatalf("item %d: fora do MRP = %v", item, c.foraDoMRP)
		}
	}
	// Sem fornecedor com prazo, vale o prazo de compra do item.
	if dias, err := s.prazoDoCadastro(context.Background(), cache[1], 1); err != nil || dias != 7 {
		t.Fatalf("prazo de compra = %d, %v", dias, err)
	}
	cache[1].engineeringType = 0
	if dias, err := s.prazoDoCadastro(context.Background(), cache[1], 1); err != nil || dias != 10 {
		t.Fatalf("prazo de fabricação = %d, %v", dias, err)
	}
}
