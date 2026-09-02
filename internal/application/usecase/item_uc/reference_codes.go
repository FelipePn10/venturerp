package item_uc

import (
	"context"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

// resolveReferenceCodes converte os códigos de negócio que a tela de cadastro de
// item envia (item-base e item de embalagem) nas chaves legadas guardadas no
// banco. Códigos em branco significam "não informado".
func resolveReferenceCodes(ctx context.Context, items any, item *entity.Item) error {
	if code := item.Engineering.ItemBaseBusinessCode; code != "" {
		base, err := itemresolution.Resolve(ctx, items, request.TextCode(code))
		if err != nil {
			return errorsuc.NewValidationError("item-base " + code + " não encontrado nesta empresa")
		}
		if item.Code.IsValid() && base.Code == item.Code {
			return errorsuc.NewValidationError("o item não pode ser o próprio item-base")
		}
		legacy := int(base.Code)
		item.Engineering.ItemBaseCod = &legacy
	}
	if code := item.Commercial.PackagingItemBusinessCode; code != "" {
		packaging, err := itemresolution.Resolve(ctx, items, request.TextCode(code))
		if err != nil {
			return errorsuc.NewValidationError("item de embalagem " + code + " não encontrado nesta empresa")
		}
		if item.Code.IsValid() && packaging.Code == item.Code {
			return errorsuc.NewValidationError("o item não pode ser a própria embalagem")
		}
		legacy := int64(packaging.Code)
		item.Commercial.PackagingItemCode = &legacy
	}
	return nil
}

// legacyItemFinder é a fatia do repositório usada para traduzir a chave legada
// de volta ao código de negócio na leitura do item.
type legacyItemFinder interface {
	FindItemByCode(context.Context, valueobject.ItemCode) (*entity.Item, error)
}

// fillReferenceBusinessCodes traduz item-base e item de embalagem da chave
// legada para o código de negócio, para que a tela de cadastro reabra o item
// com os mesmos códigos que enviou. Referência ausente não impede a leitura.
func fillReferenceBusinessCodes(ctx context.Context, items any, item *entity.Item) {
	finder, ok := items.(legacyItemFinder)
	if !ok || item == nil {
		return
	}
	if item.Engineering.ItemBaseCod != nil && item.Engineering.ItemBaseBusinessCode == "" {
		if base, err := finder.FindItemByCode(ctx, valueobject.ItemCode(*item.Engineering.ItemBaseCod)); err == nil {
			item.Engineering.ItemBaseBusinessCode = string(base.BusinessCode)
		}
	}
	if item.Commercial.PackagingItemCode != nil && item.Commercial.PackagingItemBusinessCode == "" {
		if packaging, err := finder.FindItemByCode(ctx, valueobject.ItemCode(*item.Commercial.PackagingItemCode)); err == nil {
			item.Commercial.PackagingItemBusinessCode = string(packaging.BusinessCode)
		}
	}
}

// fiscalClassificationCatalog é a fatia do catálogo canônico de classificações
// fiscais (/api/fiscal-classifications) usada para validar o cadastro do item.
type fiscalClassificationCatalog interface {
	GetByCode(ctx context.Context, enterpriseID, code int64) (*fiscalentity.FiscalClassification, error)
}

// validateFiscalClassifications recusa classificações fiscais de venda/compra
// que não existam (ou estejam inativas) no catálogo canônico da empresa. Sem
// catálogo configurado, os códigos passam como estão.
func validateFiscalClassifications(ctx context.Context, catalog fiscalClassificationCatalog, enterpriseID int64, item *entity.Item) error {
	if catalog == nil || item == nil {
		return nil
	}
	checks := []struct {
		label string
		code  *string
	}{
		{"venda", item.Accounting.SaleFiscalClassificationCode},
		{"compra", item.Accounting.PurchaseFiscalClassificationCode},
	}
	for _, check := range checks {
		if check.code == nil || strings.TrimSpace(*check.code) == "" {
			continue
		}
		raw := strings.TrimSpace(*check.code)
		code, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return errorsuc.NewValidationError(
				"classificação fiscal de " + check.label + " inválida: informe um código do cadastro de classificações fiscais")
		}
		found, err := catalog.GetByCode(ctx, enterpriseID, code)
		if err != nil || found == nil {
			return errorsuc.NewValidationError(
				"classificação fiscal de " + check.label + " " + raw + " não encontrada no cadastro de classificações fiscais")
		}
		if !found.IsActive {
			return errorsuc.NewValidationError(
				"a classificação fiscal de " + check.label + " " + raw + " está inativa")
		}
	}
	return nil
}
