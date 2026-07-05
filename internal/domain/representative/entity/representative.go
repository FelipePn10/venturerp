package entity

import "time"

type RepresentativeType struct {
	Code                 int64
	Description          string
	IsFree               bool
	IgnoresDirectBilling bool
	IsActive             bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type Representative struct {
	Code           int64
	IsCustomer     bool
	CustomerCode   *int64
	IsSupplier     bool
	SupplierCode   *int64
	Name           string
	TradeName      *string
	TypeCode       *int64
	CategoryCode   *int64
	RegisterDate   time.Time
	CoreNumber     *string
	DocumentNumber string
	PostalCode     *string
	City           *string
	State          *string
	FullAddress    *string
	Street         *string
	StreetNumber   *string
	Complement     *string
	District       *string
	MainPhone      *string
	MainEmail      *string
	DeviceQuantity int
	IsActive       bool
	Blocked        bool
	BlockReason    *string
	CreatedAt      time.Time
	UpdatedAt      time.Time

	Enterprises             []*RepresentativeEnterprise
	Accounting              []*RepresentativeAccounting
	Regions                 []*RepresentativeRegion
	Segments                []*RepresentativeSegment
	SalesPlans              []*RepresentativeSalesPlan
	Interests               []*RepresentativeInterest
	Phones                  []*RepresentativePhone
	Emails                  []*RepresentativeEmail
	CorrespondenceAddresses []*RepresentativeCorrespondenceAddress
	Contacts                []*RepresentativeContact
}

type RepresentativeEnterprise struct {
	ID                    int64
	RepresentativeCode    int64
	EnterpriseCode        int64
	EnterpriseName        *string
	CommissionPatternCode *int64
	CommissionPct         float64
	IsDefault             bool
	IsActive              bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type RepresentativeAccounting struct {
	ID                   int64
	RepresentativeCode   int64
	EnterpriseCode       *int64
	EventType            string
	DebitAccountCode     *int64
	DebitCostCenterCode  *int64
	CreditAccountCode    *int64
	CreditCostCenterCode *int64
	HistoryCode          *int64
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type RepresentativeRegion struct {
	ID                 int64
	RepresentativeCode int64
	EnterpriseCode     *int64
	RegionCode         int64
	MicroregionCode    *int64
	IsActive           bool
	CreatedAt          time.Time
}

type RepresentativeSegment struct {
	ID                 int64
	RepresentativeCode int64
	EnterpriseCode     *int64
	MicroregionCode    *int64
	MarketSegmentCode  int64
	IsActive           bool
	CreatedAt          time.Time
}

type RepresentativeSalesPlan struct {
	ID                 int64
	RepresentativeCode int64
	EnterpriseCode     *int64
	MicroregionCode    *int64
	SalesPlanCode      int64
	IsActive           bool
	CreatedAt          time.Time
}

type RepresentativeInterest struct {
	ID                     int64
	RepresentativeCode     int64
	ItemClassificationCode int64
	IsActive               bool
	CreatedAt              time.Time
}

type RepresentativePhone struct {
	ID                 int64
	RepresentativeCode int64
	DDI                *string
	DDD                *string
	Phone              string
	PhoneType          string
	Ranking            int
	CreatedAt          time.Time
}

type RepresentativeEmail struct {
	ID                 int64
	RepresentativeCode int64
	Email              string
	Ranking            int
	CreatedAt          time.Time
}

type RepresentativeCorrespondenceAddress struct {
	ID                 int64
	RepresentativeCode int64
	PostalCode         *string
	City               *string
	State              *string
	FullAddress        *string
	Street             *string
	StreetNumber       *string
	Complement         *string
	District           *string
	IsDefault          bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type RepresentativeContact struct {
	ID                 int64
	RepresentativeCode int64
	ContactTypeCode    *int64
	Name               string
	Role               *string
	Phone              *string
	Email              *string
	Notes              *string
	IsActive           bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
