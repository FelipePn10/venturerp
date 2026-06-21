package entity

import "time"

type PlanStatus string

const (
	PlanStatusIncluido PlanStatus = "I"
	PlanStatusAtivo    PlanStatus = "A"
	PlanStatusInativo  PlanStatus = "X"
)

type AccountingPlan struct {
	ID          int64      `json:"id"`
	PlanNumber  int        `json:"plan_number"`
	Description string     `json:"description"`
	ValidFrom   time.Time  `json:"valid_from"`
	ValidTo     *time.Time `json:"valid_to,omitempty"`
	Status      PlanStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
}

type AccountingAccount struct {
	ID                 int64      `json:"id"`
	PlanID             int64      `json:"plan_id"`
	ParentID           *int64     `json:"parent_id,omitempty"`
	AccountNumber      string     `json:"account_number"`
	Description        string     `json:"description"`
	NatureCode         string     `json:"nature_code"`
	ReducedCode        *string    `json:"reduced_code,omitempty"`
	RequiresCostCenter bool       `json:"requires_cost_center"`
	ValidFrom          time.Time  `json:"valid_from"`
	ValidTo            *time.Time `json:"valid_to,omitempty"`
	IsAnalytic         bool       `json:"is_analytic"`
	CreatedAt          time.Time  `json:"created_at"`
}

type AccountingReferenceAccount struct {
	ID              int64     `json:"id"`
	InstitutionCode int       `json:"institution_code"`
	ParentRefID     *int64    `json:"parent_ref_id,omitempty"`
	AccountNumber   string    `json:"account_number"`
	Description     string    `json:"description"`
	AccountType     string    `json:"account_type"`
	CreatedAt       time.Time `json:"created_at"`
}

type AccountingAccountRef struct {
	ID           int64     `json:"id"`
	AccountID    int64     `json:"account_id"`
	RefAccountID int64     `json:"ref_account_id"`
	EmpresaID    int       `json:"empresa_id"`
	CostCenterID *int64    `json:"cost_center_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type AccountingJournalEntry struct {
	ID              int64     `json:"id"`
	PlanID          int64     `json:"plan_id"`
	EmpresaID       int       `json:"empresa_id"`
	EntryDate       time.Time `json:"entry_date"`
	EntryNumber     string    `json:"entry_number"`
	BatchNumber     string    `json:"batch_number"`
	DebitAccountID  int64     `json:"debit_account_id"`
	CreditAccountID int64     `json:"credit_account_id"`
	DebitCCID       *int64    `json:"debit_cc_id,omitempty"`
	CreditCCID      *int64    `json:"credit_cc_id,omitempty"`
	Value           float64   `json:"value"`
	HistoryCode     string    `json:"history_code"`
	Description     string    `json:"description"`
	EntryType       string    `json:"entry_type"`
	CreatedAt       time.Time `json:"created_at"`
}

type AccountingDemonstrative struct {
	ID          int64                        `json:"id"`
	Code        string                       `json:"code"`
	Description string                       `json:"description"`
	TermText    string                       `json:"term_text"`
	Items       []AccountingDemonstrativeItem `json:"items"`
	CreatedAt   time.Time                    `json:"created_at"`
}

type AccountingDemonstrativeItem struct {
	ID              int64     `json:"id"`
	DemonstrativeID int64     `json:"demonstrative_id"`
	ItemCode        int       `json:"item_code"`
	Description     string    `json:"description"`
	Formula         string    `json:"formula"`
	IndicatorGroup  string    `json:"indicator_group"`
	ShowInReport    bool      `json:"show_in_report"`
	ShowBold        bool      `json:"show_bold"`
	IsResult        bool      `json:"is_result"`
	Is100Pct        bool      `json:"is_100_pct"`
	SpedEcfDigit    string    `json:"sped_ecf_digit"`
	SpedEcfType     string    `json:"sped_ecf_type"`
	CreatedAt       time.Time `json:"created_at"`
}
