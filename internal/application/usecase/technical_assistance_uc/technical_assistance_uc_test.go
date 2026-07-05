package technical_assistance_uc

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity"
	tarepo "github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/repository"
	"github.com/google/uuid"
)

type taAllowAuth struct{ ports.AuthService }

func (taAllowAuth) CanManageTechnicalAssistance(context.Context) bool { return true }

type fakeTARepo struct {
	reasons map[int64]*entity.DefectReason
	call    *entity.Call
	items   []*entity.CallItem
	notes   []*entity.ReturnNote
}

func (f *fakeTARepo) NextCallNumber(context.Context, int64) (int64, error) { return 1, nil }
func (f *fakeTARepo) CreateDefectGroup(context.Context, *entity.DefectGroup) (*entity.DefectGroup, error) {
	return nil, nil
}
func (f *fakeTARepo) ListDefectGroups(context.Context, bool) ([]*entity.DefectGroup, error) {
	return nil, nil
}
func (f *fakeTARepo) CreateDefectReason(_ context.Context, reason *entity.DefectReason) (*entity.DefectReason, error) {
	return reason, nil
}
func (f *fakeTARepo) ListDefectReasons(context.Context, *int64, bool) ([]*entity.DefectReason, error) {
	return nil, nil
}
func (f *fakeTARepo) GetDefectReason(_ context.Context, code int64) (*entity.DefectReason, error) {
	return f.reasons[code], nil
}
func (f *fakeTARepo) CreateWarrantyResponsible(context.Context, *entity.WarrantyResponsible) (*entity.WarrantyResponsible, error) {
	return nil, nil
}
func (f *fakeTARepo) ListWarrantyResponsibles(context.Context, bool) ([]*entity.WarrantyResponsible, error) {
	return nil, nil
}
func (f *fakeTARepo) CreateCall(_ context.Context, call *entity.Call) (*entity.Call, error) {
	call.Code = 1
	f.call = call
	return call, nil
}
func (f *fakeTARepo) UpdateCall(_ context.Context, call *entity.Call) (*entity.Call, error) {
	f.call = call
	return call, nil
}
func (f *fakeTARepo) GetCall(context.Context, int64) (*entity.Call, error) {
	f.call.Items = f.items
	f.call.ReturnNotes = f.notes
	return f.call, nil
}
func (f *fakeTARepo) ListCalls(context.Context, tarepo.CallFilter) ([]*entity.Call, error) {
	return nil, nil
}
func (f *fakeTARepo) AddCallItem(_ context.Context, item *entity.CallItem) (*entity.CallItem, error) {
	item.Code = int64(len(f.items) + 1)
	f.items = append(f.items, item)
	return item, nil
}
func (f *fakeTARepo) ListCallItems(context.Context, int64) ([]*entity.CallItem, error) {
	return f.items, nil
}
func (f *fakeTARepo) AddReturnNote(_ context.Context, note *entity.ReturnNote) (*entity.ReturnNote, error) {
	f.notes = append(f.notes, note)
	return note, nil
}
func (f *fakeTARepo) ListReturnNotes(context.Context, int64) ([]*entity.ReturnNote, error) {
	return f.notes, nil
}
func (f *fakeTARepo) AddOrderLink(context.Context, *entity.OrderLink) (*entity.OrderLink, error) {
	return nil, nil
}
func (f *fakeTARepo) ListOrderLinks(context.Context, int64) ([]*entity.OrderLink, error) {
	return nil, nil
}
func (f *fakeTARepo) Report(context.Context, tarepo.ReportFilter) (*tarepo.Report, error) {
	return nil, nil
}

func TestAddCallItemRequiresComplementWhenReasonAllowsIt(t *testing.T) {
	reasonCode := int64(10)
	uc := &UseCase{
		Repo: &fakeTARepo{reasons: map[int64]*entity.DefectReason{
			reasonCode: {Code: reasonCode, AllowsComplement: true},
		}},
		Auth: taAllowAuth{},
	}

	_, err := uc.AddCallItem(context.Background(), request.CreateTechnicalAssistanceCallItemDTO{
		CallCode: 1, ItemCode: 100, DefectReasonCode: &reasonCode,
	})
	if err == nil {
		t.Fatalf("expected complement validation error")
	}
}

func TestAddCallItemCalculatesWarrantyAndRevenueFromReason(t *testing.T) {
	reasonCode := int64(11)
	repo := &fakeTARepo{reasons: map[int64]*entity.DefectReason{
		reasonCode: {Code: reasonCode, GeneratesRevenue: true},
	}}
	uc := &UseCase{Repo: repo, Auth: taAllowAuth{}}

	item, err := uc.AddCallItem(context.Background(), request.CreateTechnicalAssistanceCallItemDTO{
		CallCode: 1, ItemCode: 100, DefectReasonCode: &reasonCode,
		PurchaseInvoiceDate: time.Now().AddDate(0, 0, -10).Format(time.DateOnly),
		WarrantyDays:        30,
	})
	if err != nil {
		t.Fatalf("AddCallItem() error = %v", err)
	}
	if !item.InWarranty || !item.GeneratesRevenue {
		t.Fatalf("item warranty/revenue = %+v, want both true", item)
	}
}

func TestUpdateStatusRequiresReturnNoteFromReason(t *testing.T) {
	reasonCode := int64(12)
	repo := &fakeTARepo{
		reasons: map[int64]*entity.DefectReason{reasonCode: {Code: reasonCode, RequiresReturnNote: true}},
		call: &entity.Call{
			Code: 1, CallNumber: 1, EnterpriseCode: 1, CustomerCode: 1,
			Status: entity.CallStatusPending, Priority: "NORMAL", OpenedAt: time.Now(), Subject: "Falha",
			CreatedBy: uuid.Nil,
		},
		items: []*entity.CallItem{{Code: 1, CallCode: 1, ItemCode: 100, Quantity: 1, DefectReasonCode: &reasonCode}},
	}
	uc := &UseCase{Repo: repo, Auth: taAllowAuth{}}

	_, err := uc.UpdateStatus(context.Background(), request.UpdateTechnicalAssistanceCallStatusDTO{
		Code: 1, Status: string(entity.CallStatusAttended),
	})
	if err == nil {
		t.Fatalf("expected return note validation error")
	}
}
