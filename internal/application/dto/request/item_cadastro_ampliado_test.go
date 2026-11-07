package request

import (
	"encoding/json"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
)

func TestOpcionalDistingueAusenteDeNull(t *testing.T) {
	validade, gtin := 30, "7891000100103"
	item := entity.Item{}
	item.Warehouse.ShelfLifeDays = &validade
	item.Accounting.GTIN = &gtin

	var ausente UpdateItemDTO
	if err := json.Unmarshal([]byte(`{"warehouse":{},"accounting":{}}`), &ausente); err != nil {
		t.Fatal(err)
	}
	ausente.Warehouse.EstoqueAmpliadoDTO.AplicarEm(&item.Warehouse)
	ausente.Accounting.FiscalAmpliadoDTO.AplicarEm(&item.Accounting)
	if item.Warehouse.ShelfLifeDays == nil || item.Accounting.GTIN == nil {
		t.Fatal("campo ausente deve preservar o gravado")
	}

	var limpa UpdateItemDTO
	if err := json.Unmarshal([]byte(`{"warehouse":{"shelf_life_days":null},"accounting":{"gtin":null}}`), &limpa); err != nil {
		t.Fatal(err)
	}
	limpa.Warehouse.EstoqueAmpliadoDTO.AplicarEm(&item.Warehouse)
	limpa.Accounting.FiscalAmpliadoDTO.AplicarEm(&item.Accounting)
	if item.Warehouse.ShelfLifeDays != nil || item.Accounting.GTIN != nil {
		t.Fatal("null deve limpar o campo")
	}
}

func TestCamposAmpliadosChegamPeloCreateEPeloUpdate(t *testing.T) {
	corpo := `{"warehouse":{"shelf_life_days":90,"hazardous":true},"planning":{"buyer_employee_code":7,"maximum_lot":"50"},
		"supplies":{"receipt_routing":"INSPECAO"},"accounting":{"gtin":" 7891000100103 ","tax_unit":"kg","tax_unit_factor":2}}`
	var c CreateItemDTO
	if err := json.Unmarshal([]byte(corpo), &c); err != nil {
		t.Fatal(err)
	}
	var u UpdateItemDTO
	if err := json.Unmarshal([]byte(corpo), &u); err != nil {
		t.Fatal(err)
	}
	var a, b entity.Accounting
	c.Accounting.FiscalAmpliadoDTO.AplicarEm(&a)
	u.Accounting.FiscalAmpliadoDTO.AplicarEm(&b)
	if *a.GTIN != "7891000100103" || *b.TaxUnit != "KG" || !a.TaxUnitFactor.Equal(*b.TaxUnitFactor) {
		t.Fatalf("fiscal: %+v / %+v", a, b)
	}
	var p entity.Planning
	u.Planning.PlanejamentoAmpliadoDTO.AplicarEm(&p)
	if *p.BuyerEmployeeCode != 7 || p.Extras.MaximumLot == nil {
		t.Fatalf("planejamento: %+v", p)
	}
}
