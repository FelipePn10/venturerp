package fiscal_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	salesentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	shipentity "github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	shiprepo "github.com/FelipePn10/panossoerp/internal/domain/shipment/repository"
)

type CreateFiscalExitFromLoadUseCase struct {
	CreateUC       *CreateFiscalExitUseCase
	FiscalRepo     fiscalrepo.FiscalRepository
	ShipmentRepo   shiprepo.ShipmentRepository
	SalesOrderRepo salesrepo.SalesOrderRepository
}

func (uc *CreateFiscalExitFromLoadUseCase) Execute(ctx context.Context, dto request.CreateFiscalExitFromLoadDTO) (*response.FiscalExitResponse, error) {
	if uc.CreateUC == nil || uc.FiscalRepo == nil || uc.ShipmentRepo == nil {
		return nil, fmt.Errorf("dependências de faturamento por carga não configuradas")
	}
	if dto.LoadCode <= 0 {
		return nil, fmt.Errorf("load_code é obrigatório")
	}

	load, err := uc.ShipmentRepo.GetLoadByCode(ctx, dto.LoadCode)
	if err != nil {
		return nil, err
	}
	if load.Status == shipentity.LoadStatusCancelled || load.Status == shipentity.LoadStatusShipped {
		return nil, fmt.Errorf("carga %d não pode ser faturada no status %s", dto.LoadCode, load.Status)
	}
	if len(load.Shipments) == 0 {
		return nil, fmt.Errorf("carga %d não possui romaneios para faturar", dto.LoadCode)
	}
	if len(load.FiscalNotes) > 0 {
		return nil, fmt.Errorf("carga %d já possui nota fiscal vinculada", dto.LoadCode)
	}

	overrides := fiscalLoadOverrides(dto.ItemOverrides)
	items, salesOrderCode, err := uc.itemsFromLoad(ctx, load.Shipments, dto, overrides)
	if err != nil {
		return nil, err
	}

	numeroNF, err := uc.FiscalRepo.GetNextNFNumber(ctx)
	if err != nil {
		return nil, err
	}
	source := "LOAD"
	loadCode := dto.LoadCode
	createDTO := request.CreateFiscalExitDTO{
		NumeroNF:                numeroNF,
		Serie:                   dto.Serie,
		DataEmissao:             dto.DataEmissao,
		DataSaida:               dto.DataSaida,
		CnpjDestinatario:        dto.CnpjDestinatario,
		RazaoSocialDestinatario: dto.RazaoSocialDestinatario,
		IEDestinatario:          dto.IEDestinatario,
		UFDestinatario:          dto.UFDestinatario,
		TipoPessoa:              dto.TipoPessoa,
		Cfop:                    dto.Cfop,
		NaturezaOperacao:        dto.NaturezaOperacao,
		ValorProdutos:           sumFiscalExitItems(items),
		ValorFrete:              dto.ValorFrete,
		ValorSeguro:             dto.ValorSeguro,
		ValorDesconto:           dto.ValorDesconto,
		SalesOrderCode:          salesOrderCode,
		SourceType:              &source,
		ShipmentLoadCode:        &loadCode,
		Itens:                   items,
	}

	created, err := uc.CreateUC.Execute(ctx, createDTO)
	if err != nil {
		return nil, err
	}

	nfNumber := created.NumeroNF
	for idx, link := range load.Shipments {
		shipmentCode := link.ShipmentCode
		if err := uc.ShipmentRepo.SetFiscalExit(ctx, shipmentCode, &created.ID, &nfNumber, created.ChaveAcesso, nil); err != nil {
			return nil, fmt.Errorf("vinculando NF-e ao romaneio %d: %w", shipmentCode, err)
		}
		if _, err := uc.ShipmentRepo.AddFiscalNoteToLoad(ctx, shiprepo.AddFiscalNoteToLoadInput{
			LoadCode:     dto.LoadCode,
			ShipmentCode: &shipmentCode,
			FiscalExitID: created.ID,
			NFeNumber:    &nfNumber,
			NFeKey:       created.ChaveAcesso,
			Sequence:     idx + 1,
		}); err != nil {
			return nil, fmt.Errorf("vinculando NF-e à carga %d: %w", dto.LoadCode, err)
		}
	}
	if err := uc.ShipmentRepo.RecalcLoadTotals(ctx, dto.LoadCode); err != nil {
		return nil, fmt.Errorf("recalculando totais da carga %d: %w", dto.LoadCode, err)
	}

	return created, nil
}

func (uc *CreateFiscalExitFromLoadUseCase) itemsFromLoad(
	ctx context.Context,
	links []*shipentity.ShipmentLoadShipment,
	dto request.CreateFiscalExitFromLoadDTO,
	overrides map[fiscalLoadOverrideKey]request.FiscalExitLoadItemOverride,
) ([]request.CreateFiscalExitItemDTO, *int64, error) {
	var out []request.CreateFiscalExitItemDTO
	var commonSalesOrder *int64
	mixedSalesOrders := false
	seq := 1
	for _, link := range links {
		ship, err := uc.ShipmentRepo.GetByCode(ctx, link.ShipmentCode)
		if err != nil {
			return nil, nil, err
		}
		if ship.Status == shipentity.ShipmentStatusCancelled || ship.Status == shipentity.ShipmentStatusShipped {
			return nil, nil, fmt.Errorf("romaneio %d não pode ser faturado no status %s", ship.Code, ship.Status)
		}
		if len(ship.Items) == 0 {
			return nil, nil, fmt.Errorf("romaneio %d não possui itens para faturar", ship.Code)
		}

		orderItems := map[int64]*salesentity.SalesOrderItem{}
		orderItemsByCode := map[int64]*salesentity.SalesOrderItem{}
		if uc.SalesOrderRepo != nil && ship.SalesOrderCode != nil {
			items, _ := uc.SalesOrderRepo.ListItems(ctx, *ship.SalesOrderCode)
			for _, it := range items {
				orderItems[it.ItemCode] = it
				orderItemsByCode[it.Code] = it
			}
			if mixedSalesOrders {
				// Keep sales_order_code empty for multi-order loads.
			} else if commonSalesOrder == nil {
				v := *ship.SalesOrderCode
				commonSalesOrder = &v
			} else if *commonSalesOrder != *ship.SalesOrderCode {
				commonSalesOrder = nil
				mixedSalesOrders = true
			}
		}

		for _, si := range ship.Items {
			ov := findFiscalLoadOverride(overrides, ship.Code, si.ItemCode)
			line, err := fiscalItemFromShipmentItem(seq, ship.Code, si, orderItems, orderItemsByCode, dto, ov)
			if err != nil {
				return nil, nil, err
			}
			out = append(out, line)
			seq++
		}
	}
	return out, commonSalesOrder, nil
}

type fiscalLoadOverrideKey struct {
	shipmentCode int64
	itemCode     int64
}

func fiscalLoadOverrides(in []request.FiscalExitLoadItemOverride) map[fiscalLoadOverrideKey]request.FiscalExitLoadItemOverride {
	out := map[fiscalLoadOverrideKey]request.FiscalExitLoadItemOverride{}
	for _, ov := range in {
		shipmentCode := int64(0)
		if ov.ShipmentCode != nil {
			shipmentCode = *ov.ShipmentCode
		}
		out[fiscalLoadOverrideKey{shipmentCode: shipmentCode, itemCode: ov.ItemCode}] = ov
	}
	return out
}

func findFiscalLoadOverride(overrides map[fiscalLoadOverrideKey]request.FiscalExitLoadItemOverride, shipmentCode, itemCode int64) request.FiscalExitLoadItemOverride {
	if ov, ok := overrides[fiscalLoadOverrideKey{shipmentCode: shipmentCode, itemCode: itemCode}]; ok {
		return ov
	}
	return overrides[fiscalLoadOverrideKey{itemCode: itemCode}]
}

func fiscalItemFromShipmentItem(
	seq int,
	shipmentCode int64,
	si *shipentity.ShipmentItem,
	orderItems map[int64]*salesentity.SalesOrderItem,
	orderItemsByCode map[int64]*salesentity.SalesOrderItem,
	dto request.CreateFiscalExitFromLoadDTO,
	ov request.FiscalExitLoadItemOverride,
) (request.CreateFiscalExitItemDTO, error) {
	var orderItem *salesentity.SalesOrderItem
	if si.SalesOrderItemCode != nil {
		orderItem = orderItemsByCode[*si.SalesOrderItemCode]
	}
	if orderItem == nil {
		orderItem = orderItems[si.ItemCode]
	}

	unitPrice := 0.0
	if orderItem != nil {
		unitPrice = orderItem.UnitPrice
	}
	if ov.UnitPrice != nil {
		unitPrice = *ov.UnitPrice
	}
	if unitPrice <= 0 {
		return request.CreateFiscalExitItemDTO{}, fmt.Errorf("item %d do romaneio %d sem preço; informe item_overrides.unit_price ou vincule o pedido de venda", si.ItemCode, shipmentCode)
	}

	qty := si.Quantity
	if si.IsConferred {
		qty = si.ConferredQty
	}
	cfop := dto.Cfop
	if ov.Cfop != nil && *ov.Cfop != "" {
		cfop = *ov.Cfop
	}
	origem := dto.OrigemMercadoria
	if origem == "" {
		origem = "0"
	}
	if ov.OrigemMercadoria != nil && *ov.OrigemMercadoria != "" {
		origem = *ov.OrigemMercadoria
	}
	desc := fmt.Sprintf("Item %d - romaneio %d", si.ItemCode, shipmentCode)
	if ov.Description != nil && *ov.Description != "" {
		desc = *ov.Description
	}
	itemCode := si.ItemCode
	return request.CreateFiscalExitItemDTO{
		Sequence:             seq,
		ItemCode:             &itemCode,
		Ncm:                  ov.Ncm,
		Cfop:                 cfop,
		Quantity:             qty,
		UnitPrice:            unitPrice,
		TotalPrice:           qty * unitPrice,
		OrigemMercadoria:     origem,
		Description:          &desc,
		MvaPct:               ov.MvaPct,
		AliqInternaDestinoST: ov.AliqInternaDestinoST,
		RedBaseSTPct:         ov.RedBaseSTPct,
	}, nil
}

func sumFiscalExitItems(items []request.CreateFiscalExitItemDTO) float64 {
	total := 0.0
	for _, it := range items {
		total += it.TotalPrice
	}
	return total
}
