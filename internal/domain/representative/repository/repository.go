package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/representative/entity"
)

type RepresentativeFilter struct {
	Codes        []int64
	Description  *string
	TypeCode     *int64
	State        *string
	RegionCode   *int64
	ActiveStatus string
	SortBy       string
	WithAccounts bool
}

type FollowUpFilter struct {
	RepresentativeCodes []int64
	CustomerCodes       []int64
	From                *time.Time
	To                  *time.Time
}

type RepresentativeReportRow struct {
	Code                 int64
	Name                 string
	TradeName            *string
	TypeCode             *int64
	TypeDescription      *string
	State                *string
	City                 *string
	MainPhone            *string
	MainEmail            *string
	RegionCodes          []int64
	IsActive             bool
	CommissionPct        float64
	DebitAccountCode     *int64
	CreditAccountCode    *int64
	GeneratedHistoryCode *int64
}

type RepresentativeFollowUp struct {
	RepresentativeCode int64
	RepresentativeName string
	CustomerCount      int64
	QuotationCount     int64
	OrderCount         int64
	TotalQuoted        float64
	TotalOrdered       float64
	AverageTicket      float64
	CommissionBase     float64
	CommissionValue    float64
	LastQuotationDate  *time.Time
	LastOrderDate      *time.Time
	Customers          []RepresentativeCustomerFollowUp
}

type RepresentativeCustomerFollowUp struct {
	CustomerCode      int64
	QuotationCount    int64
	OrderCount        int64
	TotalQuoted       float64
	TotalOrdered      float64
	LastQuotationDate *time.Time
	LastOrderDate     *time.Time
}

type RepresentativeRepository interface {
	CreateType(ctx context.Context, t *entity.RepresentativeType) (*entity.RepresentativeType, error)
	UpdateType(ctx context.Context, t *entity.RepresentativeType) (*entity.RepresentativeType, error)
	GetType(ctx context.Context, code int64) (*entity.RepresentativeType, error)
	ListTypes(ctx context.Context, onlyActive bool) ([]*entity.RepresentativeType, error)

	Create(ctx context.Context, rep *entity.Representative) (*entity.Representative, error)
	Update(ctx context.Context, rep *entity.Representative) (*entity.Representative, error)
	Get(ctx context.Context, code int64) (*entity.Representative, error)
	List(ctx context.Context, filter RepresentativeFilter) ([]*entity.Representative, error)
	Block(ctx context.Context, code int64, reason string) error
	Unblock(ctx context.Context, code int64) error

	AddEnterprise(ctx context.Context, row *entity.RepresentativeEnterprise) (*entity.RepresentativeEnterprise, error)
	AddAccounting(ctx context.Context, row *entity.RepresentativeAccounting) (*entity.RepresentativeAccounting, error)
	AddRegion(ctx context.Context, row *entity.RepresentativeRegion) (*entity.RepresentativeRegion, error)
	AddSegment(ctx context.Context, row *entity.RepresentativeSegment) (*entity.RepresentativeSegment, error)
	AddSalesPlan(ctx context.Context, row *entity.RepresentativeSalesPlan) (*entity.RepresentativeSalesPlan, error)
	AddInterest(ctx context.Context, row *entity.RepresentativeInterest) (*entity.RepresentativeInterest, error)
	AddPhone(ctx context.Context, row *entity.RepresentativePhone) (*entity.RepresentativePhone, error)
	AddEmail(ctx context.Context, row *entity.RepresentativeEmail) (*entity.RepresentativeEmail, error)
	AddCorrespondenceAddress(ctx context.Context, row *entity.RepresentativeCorrespondenceAddress) (*entity.RepresentativeCorrespondenceAddress, error)
	AddContact(ctx context.Context, row *entity.RepresentativeContact) (*entity.RepresentativeContact, error)

	Report(ctx context.Context, filter RepresentativeFilter) ([]RepresentativeReportRow, error)
	FollowUp(ctx context.Context, filter FollowUpFilter) ([]RepresentativeFollowUp, error)
}
