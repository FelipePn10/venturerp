package entity

import (
	"time"

	"github.com/google/uuid"
)

type CallPosition string

const (
	CallPositionPending   CallPosition = "PENDING"
	CallPositionScheduled CallPosition = "SCHEDULED"
	CallPositionResolved  CallPosition = "RESOLVED"
)

type CallSituation string

const (
	CallSituationOther          CallSituation = "OTHER"
	CallSituationOrder          CallSituation = "ORDER"
	CallSituationDiscontinued   CallSituation = "DISCONTINUED_ORDER"
	CallSituationTechnicalVisit CallSituation = "TECHNICAL_VISIT"
)

type CallDirection string

const (
	CallDirectionReceived CallDirection = "RECEIVED"
	CallDirectionMade     CallDirection = "MADE"
	CallDirectionWarranty CallDirection = "WARRANTY"
)

type CallType struct {
	Code        int64
	Description string
	IsComplaint bool
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   uuid.UUID
}

type KnowledgeSource struct {
	Code        int64
	Description string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   uuid.UUID
}

type Consumer struct {
	Code              int64
	Name              string
	IsActive          bool
	PersonType        string
	CPF               *string
	RG                *string
	CNPJ              *string
	StateRegistration *string
	ZipCode           *string
	City              *string
	State             *string
	Address           *string
	AddressNumber     *string
	Complement        *string
	District          *string
	MarketSegmentCode *int64
	KnowledgeCode     *int64
	Notes             *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CreatedBy         uuid.UUID
	Phones            []*ConsumerPhone
	Emails            []*ConsumerEmail
	Contacts          []*ConsumerContact
}

type ConsumerPhone struct {
	Code         int64
	ConsumerCode int64
	ContactCode  *int64
	PhoneType    string
	Number       string
	IsPrimary    bool
	CreatedAt    time.Time
}

type ConsumerEmail struct {
	Code         int64
	ConsumerCode int64
	ContactCode  *int64
	Email        string
	IsPrimary    bool
	CreatedAt    time.Time
}

type ConsumerContact struct {
	Code         int64
	ConsumerCode int64
	Name         string
	Role         *string
	ContactType  *string
	Notes        *string
	CreatedAt    time.Time
}

type CustomerContactHistory struct {
	Code         int64
	CustomerCode int64
	OpenedAt     time.Time
	ScheduledAt  time.Time
	UserCode     *int64
	ContactType  string
	Description  string
	CreatedAt    time.Time
	CreatedBy    uuid.UUID
}

type Call struct {
	Code                  int64
	CallNumber            int64
	EnterpriseCode        int64
	ConsumerCode          int64
	CustomerCode          *int64
	CallTypeCode          int64
	Direction             CallDirection
	InWarranty            bool
	DefectGroupCode       *int64
	DefectReasonCode      *int64
	ResponsibleUserCode   *int64
	Position              CallPosition
	Situation             CallSituation
	OpenedAt              time.Time
	ReturnDate            *time.Time
	VisitRequestedDate    *time.Time
	VisitReturnedDate     *time.Time
	SaleStoreCode         *int64
	EstablishmentCode     *int64
	TechnicianDescription *string
	Symptoms              *string
	ForwardedStoreCode    *int64
	Subject               string
	Description           *string
	Solution              *string
	ChecklistCode         *int64
	IsActive              bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
	CreatedBy             uuid.UUID
	Returns               []*CallReturn
	Attachments           []*CallAttachment
	ChecklistItems        []*CallChecklistItem
}

type CallReturn struct {
	Code         int64
	CallCode     int64
	ContactedAt  time.Time
	ContactType  string
	Description  string
	NextReturnAt *time.Time
	UserCode     *int64
	CreatedAt    time.Time
	CreatedBy    uuid.UUID
}

type CallAttachment struct {
	Code        int64
	CallCode    int64
	FileName    string
	FilePath    string
	ContentType *string
	Notes       *string
	CreatedAt   time.Time
	CreatedBy   uuid.UUID
}

type CallChecklistItem struct {
	Code        int64
	CallCode    int64
	Sequence    int
	Description string
	IsDone      bool
	DoneAt      *time.Time
	Notes       *string
	CreatedAt   time.Time
}
