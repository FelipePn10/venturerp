package fiscal_uc

import (
	"encoding/json"
	"testing"
)

// O pedido de compra do cliente vai no cabeçalho e em cada item da NF-e
// ("pedido_compra" da Focus = compra/xPed e det/prod/xPed).
func TestPedidoDeCompraDoClienteVaiParaANFe(t *testing.T) {
	p, _ := montarPayloadNFe(notaBase(), duasLinhas(), configBase(), PlanoDaNota{PedidoCompra: " OC-4512 "}, TributacaoReforma{})
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		PedidoCompra string `json:"pedido_compra"`
		Items        []struct {
			PedidoCompra string `json:"pedido_compra"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m.PedidoCompra != "OC-4512" || len(m.Items) != 2 || m.Items[0].PedidoCompra != "OC-4512" || m.Items[1].PedidoCompra != "OC-4512" {
		t.Fatalf("pedido de compra fora da nota: %s", raw)
	}

	sem, _ := montarPayloadNFe(notaBase(), duasLinhas(), configBase(), PlanoDaNota{}, TributacaoReforma{})
	raw, _ = json.Marshal(sem)
	var vazio map[string]any
	_ = json.Unmarshal(raw, &vazio)
	if _, ok := vazio["pedido_compra"]; ok {
		t.Fatal("sem pedido de compra o campo não deve ir")
	}
}
