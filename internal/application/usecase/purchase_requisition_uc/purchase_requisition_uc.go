package purchase_requisition_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/repository"
)

type PurchaseRequisitionUseCase struct {
	repo repository.PurchaseRequisitionRepository
}

func NewPurchaseRequisitionUseCase(repo repository.PurchaseRequisitionRepository) *PurchaseRequisitionUseCase {
	return &PurchaseRequisitionUseCase{repo: repo}
}

func parseDate(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil
	}
	return &t
}

func (uc *PurchaseRequisitionUseCase) Create(ctx context.Context, dto request.CreatePurchaseRequisitionDTO) (*entity.PurchaseRequisition, error) {
	code, err := uc.repo.NextCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating code: %w", err)
	}
	req, err := entity.NewPurchaseRequisition(code, dto.EnterpriseCode, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	req.RequestTypeCode = dto.RequestTypeCode
	req.RequesterEmployeeCode = dto.RequesterEmployeeCode
	req.Notes = dto.Notes

	created, err := uc.repo.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	for i, it := range dto.Items {
		item := buildItem(created.Code, int32(i+1), it)
		createdItem, ierr := uc.repo.AddItem(ctx, item)
		if ierr != nil {
			return nil, ierr
		}
		created.Items = append(created.Items, createdItem)
	}
	return created, nil
}

func buildItem(reqCode int64, seq int32, in request.RequisitionItemInput) *entity.PurchaseRequisitionItem {
	return &entity.PurchaseRequisitionItem{
		RequisitionCode:   reqCode,
		Sequence:          seq,
		ItemCode:          in.ItemCode,
		Quantity:          in.Quantity,
		UOM:               in.UOM,
		CostCenterCode:    in.CostCenterCode,
		AccountingAccount: in.AccountingAccount,
		SuggestedPrice:    in.SuggestedPrice,
		DeliveryDate:      parseDate(in.DeliveryDate),
		Application:       in.Application,
		UtilizationType:   in.UtilizationType,
		Status:            entity.ReqStatusOpen,
	}
}

func (uc *PurchaseRequisitionUseCase) Get(ctx context.Context, code int64) (*entity.PurchaseRequisition, error) {
	req, err := uc.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if req.Items, err = uc.repo.ListItems(ctx, code); err != nil {
		return nil, err
	}
	return req, nil
}

func (uc *PurchaseRequisitionUseCase) List(ctx context.Context, onlyOpen bool) ([]*entity.PurchaseRequisition, error) {
	return uc.repo.List(ctx, onlyOpen)
}

func (uc *PurchaseRequisitionUseCase) AddItem(ctx context.Context, dto request.AddRequisitionItemDTO) (*entity.PurchaseRequisitionItem, error) {
	existing, err := uc.repo.ListItems(ctx, dto.RequisitionCode)
	if err != nil {
		return nil, err
	}
	item := buildItem(dto.RequisitionCode, int32(len(existing)+1), dto.RequisitionItemInput)
	return uc.repo.AddItem(ctx, item)
}

func (uc *PurchaseRequisitionUseCase) ListItems(ctx context.Context, code int64) ([]*entity.PurchaseRequisitionItem, error) {
	return uc.repo.ListItems(ctx, code)
}
