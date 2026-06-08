package item_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	conversionrepo "github.com/FelipePn10/panossoerp/internal/domain/item_conversion/repository"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	routingrepo "github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
	structurerepo "github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
)

// ItemActivationReport is the readiness verdict for an item to take part in the
// MRP/production/purchasing flow, with the specific gaps found.
type ItemActivationReport struct {
	ItemCode int64    `json:"item_code"`
	ItemType string   `json:"item_type"`
	Ready    bool     `json:"ready"`
	Issues   []string `json:"issues,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ValidateItemActivationUseCase runs the cross-validation item × structure ×
// routing × supplier × UOM-conversion that should hold before an item is put to
// work: manufactured items need a BOM and a routing; purchased items need a
// preferred supplier (and a UOM conversion when bought in a different unit).
type ValidateItemActivationUseCase struct {
	ItemRepo      itemrepo.ItemRepository
	StructureRepo structurerepo.ItemStructureRepository
	RoutingRepo   routingrepo.RoutingRepository
	Suppliers     ports.PreferredSupplierProvider
	Conversions   conversionrepo.ItemConversionRepository
	Auth          ports.AuthService
}

func (uc *ValidateItemActivationUseCase) Execute(ctx context.Context, code int64) (*ItemActivationReport, error) {
	if !uc.Auth.FindItemByCode(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	ic, err := valueobject.NewItemCode(code)
	if err != nil {
		return nil, err
	}
	item, err := uc.ItemRepo.FindItemByCode(ctx, ic)
	if err != nil {
		return nil, err
	}

	report := &ItemActivationReport{
		ItemCode: code,
		ItemType: item.Engineering.Type.String(),
	}

	switch item.Engineering.Type {
	case types.FABRICADO:
		children, _ := uc.StructureRepo.GetAllDirectChildren(ctx, code)
		if len(children) == 0 {
			report.Issues = append(report.Issues, "item fabricado sem estrutura (BOM): cadastre os componentes")
		}
		if uc.RoutingRepo != nil {
			routes, _ := uc.RoutingRepo.ListRoutesByItem(ctx, code)
			if len(routes) == 0 {
				report.Issues = append(report.Issues, "item fabricado sem roteiro de fabricação: cadastre o roteiro")
			}
		}

	case types.COMPRADO:
		if uc.Suppliers != nil {
			if _, found, _ := uc.Suppliers.GetPreferredSupplier(ctx, code); !found {
				report.Issues = append(report.Issues, "item comprado sem fornecedor preferencial")
			}
		}
		if uc.Conversions != nil {
			conversions, _ := uc.Conversions.ListByItem(ctx, code)
			if len(conversions) == 0 {
				report.Warnings = append(report.Warnings,
					"sem conversão de UM cadastrada — obrigatória se a UM de compra difere da de estoque")
			}
		}
	}

	report.Ready = len(report.Issues) == 0
	return report, nil
}
