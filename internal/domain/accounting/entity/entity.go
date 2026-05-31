package entity

import "time"

type PlanStatus string

const (
	PlanStatusIncluido PlanStatus = "I"
	PlanStatusAtivo    PlanStatus = "A"
	PlanStatusInativo  PlanStatus = "X"
)

type AccountingPlan struct {
	ID          int64
	PlanNumber  int
	Description string
	ValidFrom   time.Time
	ValidTo     *time.Time
	Status      PlanStatus
	CreatedAt   time.Time
}

type AccountingAccount struct {
	ID                 int64
	PlanID             int64
	ParentID           *int64
	AccountNumber      string
	Description        string
	NatureCode         string
	ReducedCode        *string
	RequiresCostCenter bool
	ValidFrom          time.Time
	ValidTo            *time.Time
	IsAnalytic         bool
	CreatedAt          time.Time
}

type AccountingReferenceAccount struct {
	ID              int64
	InstitutionCode int
	ParentRefID     *int64
	AccountNumber   string
	Description     string
	AccountType     string
	CreatedAt       time.Time
}

type AccountingAccountRef struct {
	ID           int64
	AccountID    int64
	RefAccountID int64
	EmpresaID    int
	CostCenterID *int64
	CreatedAt    time.Time
}

type AccountingJournalEntry struct {
	ID              int64
	PlanID          int64
	EmpresaID       int
	EntryDate       time.Time
	EntryNumber     string
	BatchNumber     string
	DebitAccountID  int64
	CreditAccountID int64
	DebitCCID       *int64
	CreditCCID      *int64
	Value           float64
	HistoryCode     string
	Description     string
	EntryType       string
	CreatedAt       time.Time
}

type AccountingDemonstrative struct {
	ID          int64
	Code        string
	Description string
	TermText    string
	Items       []AccountingDemonstrativeItem
	CreatedAt   time.Time
}

type AccountingDemonstrativeItem struct {
	ID              int64
	DemonstrativeID int64
	ItemCode        int
	Description     string
	Formula         string
	IndicatorGroup  string
	ShowInReport    bool
	ShowBold        bool
	IsResult        bool
	Is100Pct        bool
	SpedEcfDigit    string
	SpedEcfType     string
	CreatedAt       time.Time
}
