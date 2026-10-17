package fiscal_uc

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type itensComUnidade map[int64]string

func (i itensComUnidade) FindItemByCode(_ context.Context, code valueobject.ItemCode) (*itementity.Item, error) {
	u, ok := i[int64(code)]
	if !ok {
		return nil, errorsuc.NewNotFoundError("item não encontrado")
	}
	return &itementity.Item{Warehouse: itementity.Warehouse{UnitOfMeasurement: types.TypeUnitOfMeasurementItem(u)}}, nil
}

type conversorSaida struct {
	ports.UOMConverter
	fatores map[string]float64
}

func (c conversorSaida) ConvertQuantity(_ context.Context, item int64, q float64, de, para string) (float64, bool, error) {
	f, ok := c.fatores[fmt.Sprintf("%d:%s>%s", item, de, para)]
	return q * f, ok, nil
}

func linhaSaida(id, item int64, uom string, qtd float64) *entity.FiscalExitItem {
	u := uom
	return &entity.FiscalExitItem{ID: id, ItemCode: &item, UnidadeComercial: &u, Quantity: qtd}
}

func TestBaixaDaNotaNaUnidadeDeEstoque(t *testing.T) {
	ctx := context.Background()
	e := &EfeitosNFe{
		Itens:     itensComUnidade{10: "UN", 20: "M", 30: "UN"},
		Conversor: conversorSaida{fatores: map[string]float64{"10:CX>UN": 12}},
	}
	q, err := e.quantidadesEmEstoque(ctx, []*entity.FiscalExitItem{
		linhaSaida(1, 10, "cx", 2), // 2 caixas de 12
		linhaSaida(2, 30, "UN", 5), // mesma unidade: não converte
	})
	if err != nil {
		t.Fatal(err)
	}
	if q[1] != 24 {
		t.Fatalf("2 CX de 12 UN devem baixar 24 UN, veio %v", q[1])
	}
	if _, ok := q[2]; ok {
		t.Fatalf("mesma unidade não entra no mapa: %v", q)
	}
	_, err = e.quantidadesEmEstoque(ctx, []*entity.FiscalExitItem{linhaSaida(3, 20, "KG", 10)})
	if _, ok := errorsuc.AsValidation(err); !ok || !strings.Contains(err.Error(), "cadastre a conversão KG→M") {
		t.Fatalf("KG sem conversão para M deveria ser recusado: %v", err)
	}
}
