package entity

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/shopspring/decimal"
)

func TestMarcadoresQueSeContradizem(t *testing.T) {
	casos := map[string]func(i *Item){
		"sucata e item base":      func(i *Item) { i.IsScrap, i.IsBase = true, true },
		"sucata e ferramenta":     func(i *Item) { i.IsScrap, i.IsTool = true, true },
		"item de processo e base": func(i *Item) { i.IsProcessItem, i.IsBase = true, true },
		"fantasma comprado":       func(i *Item) { i.Planning.Ghost, i.Engineering.Type = true, types.COMPRADO },
		"fantasma crítico":        func(i *Item) { i.Planning.Ghost, i.Planning.Critical = true, true },
		"duas embalagens":         func(i *Item) { i.Commercial.ExportPackaging, i.Commercial.RequiresSpecialPackaging = true, true },
		"serviço perigoso":        func(i *Item) { i.Engineering.Type, i.Warehouse.Hazardous = types.SERVICO, true },
		"serviço com contagem cíclica": func(i *Item) {
			i.Engineering.Type, i.Warehouse.CyclicalCountConfig = types.SERVICO, &valueobject.CyclicalCountConfig{DaysInterval: 30}
		},
		"revenda fabricada":         func(i *Item) { r := "REVENDA"; i.Commercial.SaleType = &r },
		"embalagem fora do tipo 02": func(i *Item) { t := "04"; i.Commercial.IsPackaging, i.Accounting.EFDItemType = true, &t },
		"mínimo fora do múltiplo": func(i *Item) {
			m, mult := decimal.NewFromInt(5), decimal.NewFromInt(2)
			i.Commercial.MinimumSaleQuantity, i.Commercial.SaleMultiple = &m, &mult
		},
	}
	for nome, marca := range casos {
		i := &Item{}
		marca(i)
		if i.conferirMarcadores() == nil {
			t.Errorf("%s devia ser recusado", nome)
		}
	}
	ok := &Item{IsBase: true, IsConfigured: true, IsTool: true}
	ok.Planning.Ghost = true
	if err := ok.conferirMarcadores(); err != nil {
		t.Fatalf("base, configurado, ferramenta e fantasma fabricado combinam: %v", err)
	}
}
