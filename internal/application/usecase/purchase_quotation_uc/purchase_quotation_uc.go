package purchase_quotation_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	plannedrepo "github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/entity"
	qrepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/repository"
	reqrepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/repository"
)

type PurchaseQuotationUseCase struct {
	repo    qrepo.PurchaseQuotationRepository
	reqs    reqrepo.PurchaseRequisitionRepository
	planned plannedrepo.PlannedOrderRepository
}

func NewPurchaseQuotationUseCase(
	repo qrepo.PurchaseQuotationRepository,
	reqs reqrepo.PurchaseRequisitionRepository,
	planned plannedrepo.PlannedOrderRepository,
) *PurchaseQuotationUseCase {
	return &PurchaseQuotationUseCase{repo: repo, reqs: reqs, planned: planned}
}

// Create releases requisition items / planned orders into a new quotation.
func (uc *PurchaseQuotationUseCase) Create(ctx context.Context, dto request.CreatePurchaseQuotationDTO) (*entity.PurchaseQuotation, error) {
	code, err := uc.repo.NextCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating code: %w", err)
	}
	q, err := entity.NewPurchaseQuotation(code, dto.EnterpriseCode, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	q.Notes = dto.Notes
	created, err := uc.repo.Create(ctx, q)
	if err != nil {
		return nil, err
	}

	seq := int32(0)
	for _, reqItemID := range dto.RequisitionItemIDs {
		ri, gerr := uc.reqs.GetItem(ctx, reqItemID)
		if gerr != nil {
			return nil, fmt.Errorf("requisition item %d: %w", reqItemID, gerr)
		}
		seq++
		srcCode := ri.RequisitionCode
		srcItem := ri.ID
		item := &entity.PurchaseQuotationItem{
			QuotationCode: created.Code,
			Sequence:      seq,
			ItemCode:      ri.ItemCode,
			Quantity:      ri.Balance(),
			UOM:           ri.UOM,
			DeliveryDate:  ri.DeliveryDate,
			SourceType:    entity.SourceRequisition,
			SourceCode:    &srcCode,
			SourceItemID:  &srcItem,
		}
		ci, aerr := uc.repo.AddItem(ctx, item)
		if aerr != nil {
			return nil, aerr
		}
		created.Items = append(created.Items, ci)
	}

	for _, poCode := range dto.PlannedOrderCodes {
		po, gerr := uc.planned.GetByCode(ctx, poCode)
		if gerr != nil {
			return nil, fmt.Errorf("planned order %d: %w", poCode, gerr)
		}
		qty := po.QuantityCorrected
		if qty <= 0 {
			qty = po.Quantity
		}
		seq++
		src := po.Code
		need := po.NeedDate
		item := &entity.PurchaseQuotationItem{
			QuotationCode: created.Code,
			Sequence:      seq,
			ItemCode:      po.ItemCode,
			Quantity:      qty,
			DeliveryDate:  &need,
			SourceType:    entity.SourcePlannedOrder,
			SourceCode:    &src,
		}
		ci, aerr := uc.repo.AddItem(ctx, item)
		if aerr != nil {
			return nil, aerr
		}
		created.Items = append(created.Items, ci)
	}

	for _, sc := range dto.SupplierCodes {
		s, serr := uc.repo.AddSupplier(ctx, &entity.PurchaseQuotationSupplier{QuotationCode: created.Code, SupplierCode: sc})
		if serr != nil {
			return nil, serr
		}
		created.Suppliers = append(created.Suppliers, s)
	}

	return created, nil
}

func (uc *PurchaseQuotationUseCase) Get(ctx context.Context, code int64) (*entity.PurchaseQuotation, error) {
	q, err := uc.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	items, err := uc.repo.ListItems(ctx, code)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		if it.Prices, err = uc.repo.ListPricesByItem(ctx, it.ID); err != nil {
			return nil, err
		}
	}
	q.Items = items
	if q.Suppliers, err = uc.repo.ListSuppliers(ctx, code); err != nil {
		return nil, err
	}
	return q, nil
}

func (uc *PurchaseQuotationUseCase) List(ctx context.Context, onlyOpen bool) ([]*entity.PurchaseQuotation, error) {
	return uc.repo.List(ctx, onlyOpen)
}

func (uc *PurchaseQuotationUseCase) AddSupplier(ctx context.Context, dto request.AddQuotationSupplierDTO) (*entity.PurchaseQuotationSupplier, error) {
	return uc.repo.AddSupplier(ctx, &entity.PurchaseQuotationSupplier{QuotationCode: dto.QuotationCode, SupplierCode: dto.SupplierCode})
}

func (uc *PurchaseQuotationUseCase) RecordPrice(ctx context.Context, dto request.RecordQuotationPriceDTO) (*entity.PurchaseQuotationPrice, error) {
	item, err := uc.repo.GetItem(ctx, dto.QuotationItemID)
	if err != nil {
		return nil, err
	}
	price, err := uc.repo.UpsertPrice(ctx, &entity.PurchaseQuotationPrice{
		QuotationItemID: dto.QuotationItemID,
		SupplierCode:    dto.SupplierCode,
		UnitPrice:       dto.UnitPrice,
		LeadTimeDays:    dto.LeadTimeDays,
		PaymentTermCode: dto.PaymentTermCode,
		Notes:           dto.Notes,
	})
	if err != nil {
		return nil, err
	}
	_ = uc.repo.UpdateStatus(ctx, item.QuotationCode, string(entity.QuotationQuoted))
	return price, nil
}

func (uc *PurchaseQuotationUseCase) SelectPrice(ctx context.Context, priceID int64) (*entity.PurchaseQuotationPrice, error) {
	return uc.repo.SelectPrice(ctx, priceID)
}
