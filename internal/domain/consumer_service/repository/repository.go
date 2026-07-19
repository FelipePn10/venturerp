package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/consumer_service/entity"
)

type ConsumerFilter struct {
	Search     *string
	State      *string
	City       *string
	OnlyActive bool
}

type CustomerContactFilter struct {
	CustomerCode *int64
	From         *time.Time
	To           *time.Time
	ContactType  *string
}

type CallFilter struct {
	CallNumber          *int64
	CallTypeCode        *int64
	ConsumerCode        *int64
	ResponsibleUserCode *int64
	DefectGroupCode     *int64
	DefectReasonCode    *int64
	Position            *entity.CallPosition
	Situation           *entity.CallSituation
	From                *time.Time
	To                  *time.Time
	ReturnFrom          *time.Time
	ReturnTo            *time.Time
	VisitState          *string
	OnlyActive          bool
}

type CallReport struct {
	TotalCalls             int64   `json:"total_calls"`
	PendingCalls           int64   `json:"pending_calls"`
	ScheduledCalls         int64   `json:"scheduled_calls"`
	ResolvedCalls          int64   `json:"resolved_calls"`
	TechnicalVisitCalls    int64   `json:"technical_visit_calls"`
	PendingVisitCalls      int64   `json:"pending_visit_calls"`
	ReturnedVisitCalls     int64   `json:"returned_visit_calls"`
	AverageResolutionHours float64 `json:"average_resolution_hours"`
}

type Repository interface {
	NextConsumerCode(ctx context.Context) (int64, error)
	NextCallNumber(ctx context.Context, enterpriseID int64) (int64, error)
	CreateCallType(ctx context.Context, v *entity.CallType) (*entity.CallType, error)
	ListCallTypes(ctx context.Context, onlyActive bool) ([]*entity.CallType, error)
	GetCallType(ctx context.Context, code int64) (*entity.CallType, error)
	CreateKnowledgeSource(ctx context.Context, v *entity.KnowledgeSource) (*entity.KnowledgeSource, error)
	ListKnowledgeSources(ctx context.Context, onlyActive bool) ([]*entity.KnowledgeSource, error)
	CreateConsumer(ctx context.Context, enterpriseID int64, v *entity.Consumer) (*entity.Consumer, error)
	UpdateConsumer(ctx context.Context, enterpriseID int64, v *entity.Consumer) (*entity.Consumer, error)
	GetConsumer(ctx context.Context, enterpriseID, code int64) (*entity.Consumer, error)
	ListConsumers(ctx context.Context, enterpriseID int64, filter ConsumerFilter) ([]*entity.Consumer, error)
	AddConsumerPhone(ctx context.Context, enterpriseID int64, v *entity.ConsumerPhone) (*entity.ConsumerPhone, error)
	AddConsumerEmail(ctx context.Context, enterpriseID int64, v *entity.ConsumerEmail) (*entity.ConsumerEmail, error)
	AddConsumerContact(ctx context.Context, enterpriseID int64, v *entity.ConsumerContact) (*entity.ConsumerContact, error)
	CreateCustomerContact(ctx context.Context, enterpriseID int64, v *entity.CustomerContactHistory) (*entity.CustomerContactHistory, error)
	ListCustomerContacts(ctx context.Context, enterpriseID int64, filter CustomerContactFilter) ([]*entity.CustomerContactHistory, error)
	CreateCall(ctx context.Context, enterpriseID int64, v *entity.Call) (*entity.Call, error)
	UpdateCall(ctx context.Context, enterpriseID int64, v *entity.Call) (*entity.Call, error)
	GetCall(ctx context.Context, enterpriseID, code int64) (*entity.Call, error)
	ListCalls(ctx context.Context, enterpriseID int64, filter CallFilter) ([]*entity.Call, error)
	AddCallReturn(ctx context.Context, enterpriseID int64, v *entity.CallReturn) (*entity.CallReturn, error)
	AddCallAttachment(ctx context.Context, enterpriseID int64, v *entity.CallAttachment) (*entity.CallAttachment, error)
	AddChecklistItem(ctx context.Context, enterpriseID int64, v *entity.CallChecklistItem) (*entity.CallChecklistItem, error)
	SetChecklistItemDone(ctx context.Context, enterpriseID, code int64, done bool, notes *string) (*entity.CallChecklistItem, error)
	ReportCalls(ctx context.Context, enterpriseID int64, filter CallFilter) (*CallReport, error)
}
