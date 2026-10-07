package purchase_quotation_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	poentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	porepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
	qentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/entity"
	qrepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/repository"
	reqrepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/repository"
)

// GenerateOrdersFromQuotationUseCase turns the selected quotation prices into
// purchase orders (one per supplier), registering attendance back on the source
// requisition items and closing the quotation.
type GenerateOrdersFromQuotationUseCase struct {
	Quotations       qrepo.PurchaseQuotationRepository
	Reqs             reqrepo.PurchaseRequisitionRepository
	POs              porepo.PurchaseOrderRepository
	Auth             ports.AuthService
	SupplierDefaults ports.SupplierPurchasingDefaultsProvider
	// Geracao completa o almoxarifado e passa o pedido pela alçada.
	Geracao *purchase_order_uc.Geracao
}

type GenerateOrdersFromQuotationResult struct {
	Orders  []*poentity.PurchaseOrder `json:"orders"`
	Skipped []string                  `json:"skipped,omitempty"`
}

func (uc *GenerateOrdersFromQuotationUseCase) Execute(ctx context.Context, dto request.GenerateOrdersFromQuotationDTO) (*GenerateOrdersFromQuotationResult, error) {
	if !uc.Auth.CanCreatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	// O autor vem da sessão (o handler não o preenchia: o pedido nascia sem
	// responsável).
	createdBy, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	dto.CreatedBy = createdBy

	q, err := uc.Quotations.GetByCode(ctx, dto.QuotationCode)
	if err != nil {
		return nil, err
	}

	items, err := uc.Quotations.ListItems(ctx, dto.QuotationCode)
	if err != nil {
		return nil, err
	}
	itemByID := make(map[int64]*qentity.PurchaseQuotationItem, len(items))
	for _, it := range items {
		itemByID[it.ID] = it
	}

	selected, err := uc.Quotations.ListSelectedPrices(ctx, dto.QuotationCode)
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("nenhum preço selecionado na cotação %d", dto.QuotationCode)
	}

	result := &GenerateOrdersFromQuotationResult{}

	// Group selected prices by supplier (preserving order).
	grouped := map[int64][]*qentity.PurchaseQuotationPrice{}
	var supplierOrder []int64
	for _, p := range selected {
		if _, ok := grouped[p.SupplierCode]; !ok {
			supplierOrder = append(supplierOrder, p.SupplierCode)
		}
		grouped[p.SupplierCode] = append(grouped[p.SupplierCode], p)
	}

	for _, supplierCode := range supplierOrder {
		prices := grouped[supplierCode]
		sc := supplierCode

		var priceTable, paymentTerm, invoiceType *int64
		var financialAccount *string
		freightType := "SEM_FRETE"
		if uc.SupplierDefaults != nil {
			// Também é aqui que se confere que o fornecedor é desta empresa e
			// está ativo. Um fornecedor imprestável não derruba a geração
			// inteira: só os itens dele ficam de fora, com o motivo.
			def, derr := uc.SupplierDefaults.GetPurchasingDefaults(ctx, supplierCode, q.EnterpriseCode)
			if derr != nil || def == nil {
				result.Skipped = append(result.Skipped, fmt.Sprintf("fornecedor %d: não encontrado no cadastro desta empresa", supplierCode))
				continue
			}
			if !def.IsActive {
				result.Skipped = append(result.Skipped, fmt.Sprintf("fornecedor %d (%s): cadastro inativo", supplierCode, def.SupplierName))
				continue
			}
			priceTable = def.PurchasePriceTableID
			paymentTerm = def.PaymentConditionID
			invoiceType = def.DefaultInvoiceTypeID
			financialAccount = def.FinancialAccount
			if def.FreightType != "" {
				freightType = def.FreightType
			}
		}

		// A condição que o fornecedor cotou vale mais que a do cadastro dele.
		for _, p := range prices {
			if p.PaymentTermCode != nil && *p.PaymentTermCode > 0 {
				paymentTerm = p.PaymentTermCode
				break
			}
		}

		orderNum, oerr := uc.POs.NextOrderNumber(ctx, q.EnterpriseCode)
		if oerr != nil {
			return nil, oerr
		}

		po := &poentity.PurchaseOrder{
			OrderNumber:      orderNum,
			EnterpriseCode:   q.EnterpriseCode,
			Origin:           poentity.PurchaseOrderOriginNORMAL,
			EmissionDate:     time.Now(),
			SupplierCode:     &sc,
			PaymentTermCode:  paymentTerm,
			PriceTableCode:   priceTable,
			InvoiceTypeCode:  invoiceType,
			FinancialAccount: financialAccount,
			FreightType:      freightType,
			CurrencyCode:     "BRL",
			IsFirm:           true,
			CreatedBy:        dto.CreatedBy,
		}
		uc.Geracao.NovaCapa(po)

		poItems := make([]*poentity.PurchaseOrderItem, 0, len(prices))
		seq := 0
		for _, p := range prices {
			qi := itemByID[p.QuotationItemID]
			if qi == nil {
				continue
			}
			seq++
			cotacao := dto.QuotationCode
			entrega := qi.DeliveryDate
			if entrega == nil && p.LeadTimeDays > 0 {
				d := time.Now().AddDate(0, 0, int(p.LeadTimeDays))
				entrega = &d
			}
			linha := &poentity.PurchaseOrderItem{
				QuotationCode: &cotacao,
				Sequence:      seq,
				ItemCode:      qi.ItemCode,
				RequestedQty:  qi.Quantity,
				UnitPrice:     p.UnitPrice,
				Status:        poentity.PurchaseOrderItemStatusOPEN,
				PurchaseUOM:   qi.UOM,
				DeliveryDate:  entrega,
				IsActive:      true,
			}
			if qi.SourceType == qentity.SourceRequisition && qi.SourceItemID != nil {
				reqItem := *qi.SourceItemID
				linha.PurchaseRequisitionItemID = &reqItem
			}
			poItems = append(poItems, linha)
		}

		avisos, aerr := uc.Geracao.CompletarLinhas(ctx, poItems)
		if aerr != nil {
			return nil, aerr
		}
		result.Skipped = append(result.Skipped, avisos...)
		created, cerr := uc.POs.CreateWithItems(ctx, po, poItems)
		if cerr != nil {
			return nil, fmt.Errorf("creating purchase order for supplier %d: %w", supplierCode, cerr)
		}
		created, aviso, cerr := uc.Geracao.Aprovar(ctx, created)
		if cerr != nil {
			return nil, cerr
		}
		if aviso != "" {
			result.Skipped = append(result.Skipped, aviso)
		}
		result.Orders = append(result.Orders, created)

		// Register attendance for requisition-sourced items.
		for _, p := range prices {
			qi := itemByID[p.QuotationItemID]
			if qi != nil && qi.SourceType == qentity.SourceRequisition && qi.SourceItemID != nil {
				if _, rerr := uc.Reqs.RegisterAttendance(ctx, *qi.SourceItemID, qi.Quantity); rerr != nil {
					result.Skipped = append(result.Skipped, fmt.Sprintf("pedido %d gerado, mas o atendimento do item %d da requisição não foi registrado: %v", created.Code, *qi.SourceItemID, rerr))
				}
			}
		}
	}

	if err := uc.Quotations.UpdateStatus(ctx, dto.QuotationCode, string(qentity.QuotationClosed)); err != nil {
		result.Skipped = append(result.Skipped, fmt.Sprintf("pedidos gerados, mas a cotação %d não foi encerrada: %v", dto.QuotationCode, err))
	}
	return result, nil
}
