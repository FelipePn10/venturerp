package item_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/governanca"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type aplicadorDePDM interface {
	AplicarNoItem(ctx context.Context, item *entity.Item, alteracao bool) error
}

type leitorDeUso interface {
	UsoDoItem(ctx context.Context, itemCode int64) (*governanca.Uso, error)
}

func retratoDoItem(i *entity.Item) governanca.Retrato {
	texto := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	r := governanca.Retrato{
		Unidade: i.Warehouse.UnitOfMeasurement.String(), Tipo: i.Engineering.Type.String(), Saude: i.Health.String(),
		GTIN: texto(i.Accounting.GTIN), TipoEFD: texto(i.Accounting.EFDItemType), UnidadeTributavel: texto(i.Accounting.TaxUnit),
		Roteiro: i.Supplies.ReceiptRouting,
	}
	if i.Supplies.PurchaseUOM != nil {
		r.UnidadeCompra = i.Supplies.PurchaseUOM.String()
	}
	if i.Accounting.TaxUnitFactor != nil {
		r.Fator = i.Accounting.TaxUnitFactor.String()
	}
	if i.Warehouse.ShelfLifeDays != nil {
		r.Validade = fmt.Sprint(*i.Warehouse.ShelfLifeDays)
	}
	return r
}

func (uc *UpdateItemUseCase) impactos(ctx context.Context, antes governanca.Retrato, depois *entity.Item) ([]governanca.Impacto, error) {
	uso, err := uc.Uso.UsoDoItem(ctx, int64(depois.Code))
	if err != nil {
		return nil, err
	}
	return governanca.AvaliarImpacto(antes, retratoDoItem(depois), *uso), nil
}

// Impacto simula a alteração sem gravar: a tela mostra o que muda fora do cadastro
// antes de a pessoa confirmar. Erro de validação da simulação sobe como no salvar.
func (uc *UpdateItemUseCase) Impacto(ctx context.Context, rawCode string, dto request.UpdateItemDTO) ([]governanca.Impacto, error) {
	if !uc.Auth.CanCreateItem(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if uc.Uso == nil {
		return []governanca.Impacto{}, nil
	}
	code, err := valueobject.NewBusinessCode(rawCode)
	if err != nil {
		return nil, err
	}
	repo, ok := uc.Repo.(updateItemBusinessRepository)
	if !ok {
		return nil, fmt.Errorf("leitura do item indisponível")
	}
	item, err := repo.FindItemByBusinessCode(ctx, code)
	if err != nil {
		return nil, err
	}
	antes := retratoDoItem(item)
	if err := uc.aplicar(ctx, item, dto); err != nil {
		return nil, err
	}
	return uc.impactos(ctx, antes, item)
}
