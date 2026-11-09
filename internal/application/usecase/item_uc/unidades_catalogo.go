package item_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
)

// catalogoDeUnidades confere unidades contra o catálogo da empresa (VSUP0110 ›
// Unidades). A unidade de estoque já tem FK no banco; as de compra, venda e a
// segunda unidade são texto e, sem esta conferência, aceitavam "CXA" ou "KGS"
// que nenhuma conversão alcança.
type catalogoDeUnidades interface {
	UnidadesForaDoCatalogo(ctx context.Context, codigos ...string) ([]string, error)
}

// unidadesDoItem: papel → unidade, só as preenchidas.
func unidadesDoItem(i *entity.Item) map[string]string {
	out := map[string]string{}
	put := func(papel, u string) {
		if u = strings.ToUpper(strings.TrimSpace(u)); u != "" {
			out[papel] = u
		}
	}
	put("unidade de estoque", i.Warehouse.UnitOfMeasurement.String())
	if i.Warehouse.SecondaryUOM != nil {
		put("segunda unidade", *i.Warehouse.SecondaryUOM)
	}
	if i.Supplies.PurchaseUOM != nil {
		put("unidade de compra", i.Supplies.PurchaseUOM.String())
	}
	if i.Accounting.SaleUnitOfMeasurement != nil {
		put("unidade fiscal de venda", i.Accounting.SaleUnitOfMeasurement.String())
	}
	if i.Accounting.PurchaseUnitOfMeasurement != nil {
		put("unidade fiscal de compra", i.Accounting.PurchaseUnitOfMeasurement.String())
	}
	return out
}

// validarUnidadesNoCatalogo recusa unidade fora do catálogo. Com `antes`, só
// confere as que mudaram: item antigo com unidade legada continua editável.
func validarUnidadesNoCatalogo(ctx context.Context, cat catalogoDeUnidades, item *entity.Item, antes map[string]string) error {
	if cat == nil {
		return nil
	}
	atuais := unidadesDoItem(item)
	papelDe := map[string]string{}
	var conferir []string
	for papel, u := range atuais {
		if antes != nil && antes[papel] == u {
			continue
		}
		if _, ok := papelDe[u]; !ok {
			conferir = append(conferir, u)
			papelDe[u] = papel
		}
	}
	if len(conferir) == 0 {
		return nil
	}
	fora, err := cat.UnidadesForaDoCatalogo(ctx, conferir...)
	if err != nil {
		return fmt.Errorf("conferindo as unidades no catálogo: %w", err)
	}
	if len(fora) == 0 {
		return nil
	}
	partes := make([]string, 0, len(fora))
	for _, u := range fora {
		partes = append(partes, fmt.Sprintf("%s (%s)", u, papelDe[u]))
	}
	return errorsuc.NewValidationError(fmt.Sprintf("unidade fora do catálogo da empresa ou inativa: %s — cadastre-a em Unidades e Conversões (VSUP0110 › Unidades) ou escolha outra",
		strings.Join(partes, ", ")))
}

// conversaoDaVSUP0110 é opcional no catálogo: o fator (de → para) da conversão
// do item cadastrada na VSUP0110 e vigente hoje, sem máscara.
type conversaoDaVSUP0110 interface {
	FatorDaConversaoDoItem(ctx context.Context, item int64, de, para string) (*decimal.Decimal, error)
}

// toleranciaSegundaUnidade: a mesma da consistência do grafo (0,01%).
var toleranciaSegundaUnidade = decimal.RequireFromString("0.0001")

// validarSegundaUnidade recusa o fator da segunda unidade que contradiz a
// conversão do mesmo par já cadastrada na VSUP0110. As duas valiam ao mesmo
// tempo: a tela do item mostrava 25 KG por chapa e a compra, a nota e o saldo
// usavam os 23 da VSUP0110. Só confere quando a unidade ou o fator mudou.
func validarSegundaUnidade(ctx context.Context, cat catalogoDeUnidades, item *entity.Item, uomAntes *string, fatorAntes *decimal.Decimal) error {
	conv, ok := cat.(conversaoDaVSUP0110)
	if !ok || item.Warehouse.SecondaryUOM == nil || item.Warehouse.SecondaryFactor == nil {
		return nil
	}
	segunda := strings.ToUpper(strings.TrimSpace(*item.Warehouse.SecondaryUOM))
	fator := *item.Warehouse.SecondaryFactor
	if uomAntes != nil && fatorAntes != nil && strings.EqualFold(strings.TrimSpace(*uomAntes), segunda) && fatorAntes.Equal(fator) {
		return nil
	}
	estoque := strings.ToUpper(strings.TrimSpace(item.Warehouse.UnitOfMeasurement.String()))
	if segunda == "" || estoque == "" || segunda == estoque || !fator.IsPositive() || item.Code == 0 {
		return nil
	}
	cadastrado, err := conv.FatorDaConversaoDoItem(ctx, int64(item.Code), estoque, segunda)
	if err != nil {
		return fmt.Errorf("conferindo a segunda unidade com as conversões do item: %w", err)
	}
	if cadastrado == nil || cadastrado.IsZero() {
		return nil
	}
	if fator.Sub(*cadastrado).Abs().DivRound(*cadastrado, 10).LessThanOrEqual(toleranciaSegundaUnidade) {
		return nil
	}
	return errorsuc.NewValidationError(fmt.Sprintf("a segunda unidade diz 1 %s = %s %s, mas a conversão do item (VSUP0110) diz 1 %s = %s %s — use o mesmo fator ou altere a conversão na VSUP0110",
		estoque, fator.String(), segunda, estoque, cadastrado.Round(6).String(), segunda))
}
