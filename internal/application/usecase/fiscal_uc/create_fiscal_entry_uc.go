package fiscal_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	porepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

type CreateFiscalEntryUseCase struct {
	Repo           repository.FiscalRepository
	Auth           ports.AuthService
	PurchaseOrders porepo.PurchaseOrderRepository
	Tolerances     ports.PurchaseToleranceEvaluator
}

func (uc *CreateFiscalEntryUseCase) Execute(ctx context.Context, dto request.CreateFiscalEntryDTO) (*response.FiscalEntryResponse, error) {
	if !uc.Auth.CanCreateFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	enterpriseID, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}

	dataEmissao, _ := time.Parse("2006-01-02", dto.DataEmissao)
	dataEntrada, _ := time.Parse("2006-01-02", dto.DataEntrada)

	entry := &entity.FiscalEntry{
		EnterpriseID:        enterpriseID,
		ChaveAcesso:         dto.ChaveAcesso,
		NumeroNF:            dto.NumeroNF,
		Serie:               dto.Serie,
		Modelo:              dto.Modelo,
		DataEmissao:         dataEmissao,
		DataEntrada:         dataEntrada,
		CnpjEmitente:        dto.CnpjEmitente,
		RazaoSocialEmitente: dto.RazaoSocialEmitente,
		IEEmitente:          dto.IEEmitente,
		UFEmitente:          dto.UFEmitente,
		ValorProdutos:       dto.ValorProdutos,
		ValorFrete:          dto.ValorFrete,
		ValorSeguro:         dto.ValorSeguro,
		ValorDesconto:       dto.ValorDesconto,
		ValorIPI:            dto.ValorIPI,
		ValorICMS:           dto.ValorICMS,
		ValorPIS:            dto.ValorPIS,
		ValorCOFINS:         dto.ValorCOFINS,
		ValorTotal:          dto.ValorTotal,
		TipoDocumento:       dto.TipoDocumento,
		PurchaseOrderCode:   dto.PurchaseOrderCode,
		CteCode:             dto.CteCode,
		Status:              entity.EntryStatusPending,
		Notes:               dto.Notes,
		CreatedBy:           userID,
	}
	pendingItems := make([]*entity.FiscalEntryItem, 0, len(dto.Itens))
	for _, itemDTO := range dto.Itens {
		pendingItems = append(pendingItems, entryItemFromDTO(itemDTO))
	}
	entry.SupplierCode, entry.Warnings, err = validatePurchaseEntryTolerances(ctx, uc.PurchaseOrders, uc.Tolerances, dto.PurchaseOrderCode, pendingItems, dto.ValorProdutos)
	if err != nil {
		return nil, err
	}

	created, err := uc.Repo.CreateEntry(ctx, entry)
	if err != nil {
		return nil, err
	}

	for _, item := range pendingItems {
		item.FiscalEntryID = created.ID
		if _, err := uc.Repo.CreateEntryItem(ctx, item); err != nil {
			return nil, err
		}
	}

	items, _ := uc.Repo.GetEntryItems(ctx, created.ID)
	created.Itens = items

	return toFiscalEntryResponse(created), nil
}

func entryItemFromDTO(itemDTO request.CreateFiscalEntryItemDTO) *entity.FiscalEntryItem {
	return &entity.FiscalEntryItem{
		Sequence:          itemDTO.Sequence,
		ItemCode:          itemDTO.ItemCode,
		UOM:               itemDTO.UOM,
		Ncm:               itemDTO.Ncm,
		Cfop:              itemDTO.Cfop,
		Quantity:          itemDTO.Quantity,
		UnitPrice:         itemDTO.UnitPrice,
		TotalPrice:        itemDTO.TotalPrice,
		BaseICMS:          itemDTO.BaseICMS,
		AliqICMS:          itemDTO.AliqICMS,
		ValorICMS:         itemDTO.ValorICMS,
		BaseIPI:           itemDTO.BaseIPI,
		AliqIPI:           itemDTO.AliqIPI,
		ValorIPI:          itemDTO.ValorIPI,
		ValorPIS:          itemDTO.ValorPIS,
		ValorCOFINS:       itemDTO.ValorCOFINS,
		CstICMS:           itemDTO.CstICMS,
		CstIPI:            itemDTO.CstIPI,
		CstPIS:            itemDTO.CstPIS,
		CstCOFINS:         itemDTO.CstCOFINS,
		GeraCreditoICMS:   itemDTO.GeraCreditoICMS,
		GeraCreditoIPI:    itemDTO.GeraCreditoIPI,
		GeraCreditoPIS:    itemDTO.GeraCreditoPIS,
		GeraCreditoCOFINS: itemDTO.GeraCreditoCOFINS,
		Description:       itemDTO.Description,
		Notes:             itemDTO.Notes,
	}
}
