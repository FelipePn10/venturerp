package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity"
)

type CallFilter struct {
	Status       *entity.CallStatus
	CustomerCode *int64
	From         *time.Time
	To           *time.Time
	OnlyActive   bool
}

type ReportFilter struct {
	From         *time.Time
	To           *time.Time
	CustomerCode *int64
	Status       *entity.CallStatus
}

type Report struct {
	TotalCalls       int64
	PendingCalls     int64
	AttendedCalls    int64
	ClosedCalls      int64
	CancelledCalls   int64
	InWarrantyItems  int64
	RevenueItems     int64
	AverageLeadHours float64
}

type Repository interface {
	NextCallNumber(ctx context.Context, enterpriseID int64) (int64, error)
	CreateDefectGroup(ctx context.Context, group *entity.DefectGroup) (*entity.DefectGroup, error)
	ListDefectGroups(ctx context.Context, onlyActive bool) ([]*entity.DefectGroup, error)
	CreateDefectReason(ctx context.Context, reason *entity.DefectReason) (*entity.DefectReason, error)
	ListDefectReasons(ctx context.Context, groupCode *int64, onlyActive bool) ([]*entity.DefectReason, error)
	GetDefectReason(ctx context.Context, code int64) (*entity.DefectReason, error)
	CreateWarrantyResponsible(ctx context.Context, responsible *entity.WarrantyResponsible) (*entity.WarrantyResponsible, error)
	ListWarrantyResponsibles(ctx context.Context, onlyActive bool) ([]*entity.WarrantyResponsible, error)
	CreateCall(ctx context.Context, enterpriseID int64, call *entity.Call) (*entity.Call, error)
	UpdateCall(ctx context.Context, enterpriseID int64, call *entity.Call) (*entity.Call, error)
	GetCall(ctx context.Context, enterpriseID, code int64) (*entity.Call, error)
	ListCalls(ctx context.Context, enterpriseID int64, filter CallFilter) ([]*entity.Call, error)
	AddCallItem(ctx context.Context, enterpriseID int64, item *entity.CallItem) (*entity.CallItem, error)
	ListCallItems(ctx context.Context, enterpriseID, callCode int64) ([]*entity.CallItem, error)
	AddReturnNote(ctx context.Context, enterpriseID int64, note *entity.ReturnNote) (*entity.ReturnNote, error)
	ListReturnNotes(ctx context.Context, enterpriseID, callCode int64) ([]*entity.ReturnNote, error)
	AddOrderLink(ctx context.Context, enterpriseID int64, link *entity.OrderLink) (*entity.OrderLink, error)
	ListOrderLinks(ctx context.Context, enterpriseID, callCode int64) ([]*entity.OrderLink, error)
	Report(ctx context.Context, enterpriseID int64, filter ReportFilter) (*Report, error)
}
