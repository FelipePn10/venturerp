package shipment_uc

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
)

type romaneiosDoPedido struct {
	*fakeShipmentRepo
	lista []*entity.Shipment
}

func (r *romaneiosDoPedido) ListBySalesOrder(context.Context, int64) ([]*entity.Shipment, error) {
	return r.lista, nil
}
func (r *romaneiosDoPedido) GetByCode(_ context.Context, code int64) (*entity.Shipment, error) {
	for _, s := range r.lista {
		if s.Code == code {
			return s, nil
		}
	}
	return nil, nil
}

type notaFixa string

func (n notaFixa) SituacaoDaNota(context.Context, int64) (string, error) { return string(n), nil }

// Melhoria 6: o fluxo diz a próxima ação de cada romaneio e do pedido.
func TestFluxoDaExpedicaoProximaAcao(t *testing.T) {
	d := decimal.NewFromInt
	nota := int64(9)
	casos := []struct {
		nome  string
		lista []*entity.Shipment
		nfe   string
		acao  string
	}{
		{"sem romaneio", nil, "", "MONTAR_ROMANEIO"},
		{"aberto", []*entity.Shipment{{Code: 1, Status: entity.ShipmentStatusOpen, Items: []*entity.ShipmentItem{{Quantity: d(5)}}}}, "", "SEPARAR"},
		{"conferido sem nota", []*entity.Shipment{{Code: 1, Status: entity.ShipmentStatusConferred}}, "", "EMITIR_NFE"},
		{"nota em rascunho", []*entity.Shipment{{Code: 1, Status: entity.ShipmentStatusConferred, FiscalExitID: &nota}}, "DRAFT", "AUTORIZAR_NFE"},
		{"nota autorizada", []*entity.Shipment{{Code: 1, Status: entity.ShipmentStatusConferred, FiscalExitID: &nota}}, "AUTHORIZED", "DESPACHAR"},
	}
	for _, c := range casos {
		uc := &FluxoDaExpedicaoUseCase{Repo: &romaneiosDoPedido{fakeShipmentRepo: &fakeShipmentRepo{}, lista: c.lista}, Notas: notaFixa(c.nfe),
			Pedidos: &fakeSalesReader{resp: &SalesOrderHeader{Code: 7, Items: []SalesOrderItemHeader{{PendingQty: d(5)}}}}}
		f, err := uc.Fluxo(context.Background(), 7)
		if err != nil {
			t.Fatal(err)
		}
		if f.ProximaAcao == nil || f.ProximaAcao.Codigo != c.acao {
			t.Errorf("%s: próxima ação %+v, quer %s", c.nome, f.ProximaAcao, c.acao)
		}
	}
}
