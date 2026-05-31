package sqlc

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ─── Accounting Plans ─────────────────────────────────────────────────────────

const createAccountingPlan = `
INSERT INTO accounting_plans (plan_number, description, valid_from, valid_to, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, plan_number, description, valid_from, valid_to, status, created_at
`

type CreateAccountingPlanParams struct {
	PlanNumber  int32
	Description string
	ValidFrom   pgtype.Date
	ValidTo     pgtype.Date
	Status      string
}

type AccountingPlanRow struct {
	ID          int64
	PlanNumber  int32
	Description string
	ValidFrom   pgtype.Date
	ValidTo     pgtype.Date
	Status      string
	CreatedAt   pgtype.Timestamptz
}

func (q *Queries) CreateAccountingPlan(ctx context.Context, arg CreateAccountingPlanParams) (AccountingPlanRow, error) {
	row := q.db.QueryRow(ctx, createAccountingPlan,
		arg.PlanNumber, arg.Description, arg.ValidFrom, arg.ValidTo, arg.Status,
	)
	var i AccountingPlanRow
	err := row.Scan(&i.ID, &i.PlanNumber, &i.Description, &i.ValidFrom, &i.ValidTo, &i.Status, &i.CreatedAt)
	return i, err
}

const getAccountingPlan = `
SELECT id, plan_number, description, valid_from, valid_to, status, created_at
FROM accounting_plans WHERE id = $1
`

func (q *Queries) GetAccountingPlan(ctx context.Context, id int64) (AccountingPlanRow, error) {
	row := q.db.QueryRow(ctx, getAccountingPlan, id)
	var i AccountingPlanRow
	err := row.Scan(&i.ID, &i.PlanNumber, &i.Description, &i.ValidFrom, &i.ValidTo, &i.Status, &i.CreatedAt)
	return i, err
}

const listAccountingPlans = `
SELECT id, plan_number, description, valid_from, valid_to, status, created_at
FROM accounting_plans ORDER BY plan_number
`

func (q *Queries) ListAccountingPlans(ctx context.Context) ([]AccountingPlanRow, error) {
	rows, err := q.db.Query(ctx, listAccountingPlans)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []AccountingPlanRow
	for rows.Next() {
		var i AccountingPlanRow
		if err := rows.Scan(&i.ID, &i.PlanNumber, &i.Description, &i.ValidFrom, &i.ValidTo, &i.Status, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const updateAccountingPlan = `
UPDATE accounting_plans SET plan_number=$1, description=$2, valid_from=$3, valid_to=$4, status=$5
WHERE id=$6
RETURNING id, plan_number, description, valid_from, valid_to, status, created_at
`

type UpdateAccountingPlanParams struct {
	PlanNumber  int32
	Description string
	ValidFrom   pgtype.Date
	ValidTo     pgtype.Date
	Status      string
	ID          int64
}

func (q *Queries) UpdateAccountingPlan(ctx context.Context, arg UpdateAccountingPlanParams) (AccountingPlanRow, error) {
	row := q.db.QueryRow(ctx, updateAccountingPlan,
		arg.PlanNumber, arg.Description, arg.ValidFrom, arg.ValidTo, arg.Status, arg.ID,
	)
	var i AccountingPlanRow
	err := row.Scan(&i.ID, &i.PlanNumber, &i.Description, &i.ValidFrom, &i.ValidTo, &i.Status, &i.CreatedAt)
	return i, err
}

// ─── Accounting Accounts ──────────────────────────────────────────────────────

const createAccountingAccount = `
INSERT INTO accounting_accounts (plan_id, parent_id, account_number, description, nature_code, reduced_code, requires_cost_center, valid_from, valid_to, is_analytic)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, plan_id, parent_id, account_number, description, nature_code, reduced_code, requires_cost_center, valid_from, valid_to, is_analytic, created_at
`

type CreateAccountingAccountParams struct {
	PlanID             int64
	ParentID           *int64
	AccountNumber      string
	Description        string
	NatureCode         string
	ReducedCode        pgtype.Text
	RequiresCostCenter bool
	ValidFrom          pgtype.Date
	ValidTo            pgtype.Date
	IsAnalytic         bool
}

type AccountingAccountRow struct {
	ID                 int64
	PlanID             int64
	ParentID           *int64
	AccountNumber      string
	Description        string
	NatureCode         string
	ReducedCode        pgtype.Text
	RequiresCostCenter bool
	ValidFrom          pgtype.Date
	ValidTo            pgtype.Date
	IsAnalytic         bool
	CreatedAt          pgtype.Timestamptz
}

func (q *Queries) CreateAccountingAccount(ctx context.Context, arg CreateAccountingAccountParams) (AccountingAccountRow, error) {
	row := q.db.QueryRow(ctx, createAccountingAccount,
		arg.PlanID, arg.ParentID, arg.AccountNumber, arg.Description, arg.NatureCode,
		arg.ReducedCode, arg.RequiresCostCenter, arg.ValidFrom, arg.ValidTo, arg.IsAnalytic,
	)
	var i AccountingAccountRow
	err := row.Scan(
		&i.ID, &i.PlanID, &i.ParentID, &i.AccountNumber, &i.Description,
		&i.NatureCode, &i.ReducedCode, &i.RequiresCostCenter, &i.ValidFrom, &i.ValidTo,
		&i.IsAnalytic, &i.CreatedAt,
	)
	return i, err
}

const getAccountingAccount = `
SELECT id, plan_id, parent_id, account_number, description, nature_code, reduced_code, requires_cost_center, valid_from, valid_to, is_analytic, created_at
FROM accounting_accounts WHERE id = $1
`

func (q *Queries) GetAccountingAccount(ctx context.Context, id int64) (AccountingAccountRow, error) {
	row := q.db.QueryRow(ctx, getAccountingAccount, id)
	var i AccountingAccountRow
	err := row.Scan(
		&i.ID, &i.PlanID, &i.ParentID, &i.AccountNumber, &i.Description,
		&i.NatureCode, &i.ReducedCode, &i.RequiresCostCenter, &i.ValidFrom, &i.ValidTo,
		&i.IsAnalytic, &i.CreatedAt,
	)
	return i, err
}

const listAccountingAccountsByPlan = `
SELECT id, plan_id, parent_id, account_number, description, nature_code, reduced_code, requires_cost_center, valid_from, valid_to, is_analytic, created_at
FROM accounting_accounts WHERE plan_id = $1 ORDER BY account_number
`

func (q *Queries) ListAccountingAccountsByPlan(ctx context.Context, planID int64) ([]AccountingAccountRow, error) {
	rows, err := q.db.Query(ctx, listAccountingAccountsByPlan, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []AccountingAccountRow
	for rows.Next() {
		var i AccountingAccountRow
		if err := rows.Scan(
			&i.ID, &i.PlanID, &i.ParentID, &i.AccountNumber, &i.Description,
			&i.NatureCode, &i.ReducedCode, &i.RequiresCostCenter, &i.ValidFrom, &i.ValidTo,
			&i.IsAnalytic, &i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const updateAccountingAccount = `
UPDATE accounting_accounts SET plan_id=$1, parent_id=$2, account_number=$3, description=$4, nature_code=$5,
    reduced_code=$6, requires_cost_center=$7, valid_from=$8, valid_to=$9, is_analytic=$10
WHERE id=$11
RETURNING id, plan_id, parent_id, account_number, description, nature_code, reduced_code, requires_cost_center, valid_from, valid_to, is_analytic, created_at
`

type UpdateAccountingAccountParams struct {
	PlanID             int64
	ParentID           *int64
	AccountNumber      string
	Description        string
	NatureCode         string
	ReducedCode        pgtype.Text
	RequiresCostCenter bool
	ValidFrom          pgtype.Date
	ValidTo            pgtype.Date
	IsAnalytic         bool
	ID                 int64
}

func (q *Queries) UpdateAccountingAccount(ctx context.Context, arg UpdateAccountingAccountParams) (AccountingAccountRow, error) {
	row := q.db.QueryRow(ctx, updateAccountingAccount,
		arg.PlanID, arg.ParentID, arg.AccountNumber, arg.Description, arg.NatureCode,
		arg.ReducedCode, arg.RequiresCostCenter, arg.ValidFrom, arg.ValidTo, arg.IsAnalytic, arg.ID,
	)
	var i AccountingAccountRow
	err := row.Scan(
		&i.ID, &i.PlanID, &i.ParentID, &i.AccountNumber, &i.Description,
		&i.NatureCode, &i.ReducedCode, &i.RequiresCostCenter, &i.ValidFrom, &i.ValidTo,
		&i.IsAnalytic, &i.CreatedAt,
	)
	return i, err
}

// ─── Reference Accounts ───────────────────────────────────────────────────────

const createReferenceAccount = `
INSERT INTO accounting_reference_accounts (institution_code, parent_ref_id, account_number, description, account_type)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, institution_code, parent_ref_id, account_number, description, account_type, created_at
`

type CreateReferenceAccountParams struct {
	InstitutionCode int32
	ParentRefID     *int64
	AccountNumber   string
	Description     string
	AccountType     string
}

type ReferenceAccountRow struct {
	ID              int64
	InstitutionCode int32
	ParentRefID     *int64
	AccountNumber   string
	Description     string
	AccountType     string
	CreatedAt       pgtype.Timestamptz
}

func (q *Queries) CreateReferenceAccount(ctx context.Context, arg CreateReferenceAccountParams) (ReferenceAccountRow, error) {
	row := q.db.QueryRow(ctx, createReferenceAccount,
		arg.InstitutionCode, arg.ParentRefID, arg.AccountNumber, arg.Description, arg.AccountType,
	)
	var i ReferenceAccountRow
	err := row.Scan(&i.ID, &i.InstitutionCode, &i.ParentRefID, &i.AccountNumber, &i.Description, &i.AccountType, &i.CreatedAt)
	return i, err
}

const getReferenceAccount = `
SELECT id, institution_code, parent_ref_id, account_number, description, account_type, created_at
FROM accounting_reference_accounts WHERE id = $1
`

func (q *Queries) GetReferenceAccount(ctx context.Context, id int64) (ReferenceAccountRow, error) {
	row := q.db.QueryRow(ctx, getReferenceAccount, id)
	var i ReferenceAccountRow
	err := row.Scan(&i.ID, &i.InstitutionCode, &i.ParentRefID, &i.AccountNumber, &i.Description, &i.AccountType, &i.CreatedAt)
	return i, err
}

const listReferenceAccounts = `
SELECT id, institution_code, parent_ref_id, account_number, description, account_type, created_at
FROM accounting_reference_accounts WHERE institution_code = $1 ORDER BY account_number
`

func (q *Queries) ListReferenceAccounts(ctx context.Context, institutionCode int32) ([]ReferenceAccountRow, error) {
	rows, err := q.db.Query(ctx, listReferenceAccounts, institutionCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ReferenceAccountRow
	for rows.Next() {
		var i ReferenceAccountRow
		if err := rows.Scan(&i.ID, &i.InstitutionCode, &i.ParentRefID, &i.AccountNumber, &i.Description, &i.AccountType, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// ─── Account Refs ─────────────────────────────────────────────────────────────

const createAccountRef = `
INSERT INTO accounting_account_refs (account_id, ref_account_id, empresa_id, cost_center_id)
VALUES ($1, $2, $3, $4)
RETURNING id, account_id, ref_account_id, empresa_id, cost_center_id, created_at
`

type CreateAccountRefParams struct {
	AccountID    int64
	RefAccountID int64
	EmpresaID    int32
	CostCenterID *int64
}

type AccountRefRow struct {
	ID           int64
	AccountID    int64
	RefAccountID int64
	EmpresaID    int32
	CostCenterID *int64
	CreatedAt    pgtype.Timestamptz
}

func (q *Queries) CreateAccountRef(ctx context.Context, arg CreateAccountRefParams) (AccountRefRow, error) {
	row := q.db.QueryRow(ctx, createAccountRef,
		arg.AccountID, arg.RefAccountID, arg.EmpresaID, arg.CostCenterID,
	)
	var i AccountRefRow
	err := row.Scan(&i.ID, &i.AccountID, &i.RefAccountID, &i.EmpresaID, &i.CostCenterID, &i.CreatedAt)
	return i, err
}

const listAccountRefs = `
SELECT id, account_id, ref_account_id, empresa_id, cost_center_id, created_at
FROM accounting_account_refs WHERE empresa_id = $1 ORDER BY id
`

func (q *Queries) ListAccountRefs(ctx context.Context, empresaID int32) ([]AccountRefRow, error) {
	rows, err := q.db.Query(ctx, listAccountRefs, empresaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []AccountRefRow
	for rows.Next() {
		var i AccountRefRow
		if err := rows.Scan(&i.ID, &i.AccountID, &i.RefAccountID, &i.EmpresaID, &i.CostCenterID, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// ─── Journal Entries ──────────────────────────────────────────────────────────

const createJournalEntry = `
INSERT INTO accounting_journal_entries (plan_id, empresa_id, entry_date, entry_number, batch_number, debit_account_id, credit_account_id, debit_cc_id, credit_cc_id, value, history_code, description, entry_type)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING id, plan_id, empresa_id, entry_date, entry_number, batch_number, debit_account_id, credit_account_id, debit_cc_id, credit_cc_id, value, history_code, description, entry_type, created_at
`

type CreateJournalEntryParams struct {
	PlanID          int64
	EmpresaID       int32
	EntryDate       pgtype.Date
	EntryNumber     string
	BatchNumber     string
	DebitAccountID  int64
	CreditAccountID int64
	DebitCCID       *int64
	CreditCCID      *int64
	Value           pgtype.Numeric
	HistoryCode     string
	Description     string
	EntryType       string
}

type JournalEntryRow struct {
	ID              int64
	PlanID          int64
	EmpresaID       int32
	EntryDate       pgtype.Date
	EntryNumber     string
	BatchNumber     string
	DebitAccountID  int64
	CreditAccountID int64
	DebitCCID       *int64
	CreditCCID      *int64
	Value           pgtype.Numeric
	HistoryCode     string
	Description     string
	EntryType       string
	CreatedAt       pgtype.Timestamptz
}

func (q *Queries) CreateJournalEntry(ctx context.Context, arg CreateJournalEntryParams) (JournalEntryRow, error) {
	row := q.db.QueryRow(ctx, createJournalEntry,
		arg.PlanID, arg.EmpresaID, arg.EntryDate, arg.EntryNumber, arg.BatchNumber,
		arg.DebitAccountID, arg.CreditAccountID, arg.DebitCCID, arg.CreditCCID,
		arg.Value, arg.HistoryCode, arg.Description, arg.EntryType,
	)
	var i JournalEntryRow
	err := row.Scan(
		&i.ID, &i.PlanID, &i.EmpresaID, &i.EntryDate, &i.EntryNumber, &i.BatchNumber,
		&i.DebitAccountID, &i.CreditAccountID, &i.DebitCCID, &i.CreditCCID,
		&i.Value, &i.HistoryCode, &i.Description, &i.EntryType, &i.CreatedAt,
	)
	return i, err
}

const listJournalEntries = `
SELECT id, plan_id, empresa_id, entry_date, entry_number, batch_number, debit_account_id, credit_account_id, debit_cc_id, credit_cc_id, value, history_code, description, entry_type, created_at
FROM accounting_journal_entries
WHERE plan_id = $1 AND empresa_id = $2 AND entry_date >= $3 AND entry_date <= $4
ORDER BY entry_date, entry_number
`

type ListJournalEntriesParams struct {
	PlanID    int64
	EmpresaID int32
	From      time.Time
	To        time.Time
}

func (q *Queries) ListJournalEntries(ctx context.Context, arg ListJournalEntriesParams) ([]JournalEntryRow, error) {
	rows, err := q.db.Query(ctx, listJournalEntries,
		arg.PlanID, arg.EmpresaID, arg.From, arg.To,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []JournalEntryRow
	for rows.Next() {
		var i JournalEntryRow
		if err := rows.Scan(
			&i.ID, &i.PlanID, &i.EmpresaID, &i.EntryDate, &i.EntryNumber, &i.BatchNumber,
			&i.DebitAccountID, &i.CreditAccountID, &i.DebitCCID, &i.CreditCCID,
			&i.Value, &i.HistoryCode, &i.Description, &i.EntryType, &i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// ─── Demonstratives ───────────────────────────────────────────────────────────

const createDemonstrative = `
INSERT INTO accounting_demonstratives (code, description, term_text)
VALUES ($1, $2, $3)
RETURNING id, code, description, term_text, created_at
`

type CreateDemonstrativeParams struct {
	Code        string
	Description string
	TermText    string
}

type DemonstrativeRow struct {
	ID          int64
	Code        string
	Description string
	TermText    string
	CreatedAt   pgtype.Timestamptz
}

func (q *Queries) CreateDemonstrative(ctx context.Context, arg CreateDemonstrativeParams) (DemonstrativeRow, error) {
	row := q.db.QueryRow(ctx, createDemonstrative, arg.Code, arg.Description, arg.TermText)
	var i DemonstrativeRow
	err := row.Scan(&i.ID, &i.Code, &i.Description, &i.TermText, &i.CreatedAt)
	return i, err
}

const getDemonstrative = `
SELECT id, code, description, term_text, created_at FROM accounting_demonstratives WHERE id = $1
`

func (q *Queries) GetDemonstrative(ctx context.Context, id int64) (DemonstrativeRow, error) {
	row := q.db.QueryRow(ctx, getDemonstrative, id)
	var i DemonstrativeRow
	err := row.Scan(&i.ID, &i.Code, &i.Description, &i.TermText, &i.CreatedAt)
	return i, err
}

const listDemonstratives = `
SELECT id, code, description, term_text, created_at FROM accounting_demonstratives ORDER BY code
`

func (q *Queries) ListDemonstratives(ctx context.Context) ([]DemonstrativeRow, error) {
	rows, err := q.db.Query(ctx, listDemonstratives)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DemonstrativeRow
	for rows.Next() {
		var i DemonstrativeRow
		if err := rows.Scan(&i.ID, &i.Code, &i.Description, &i.TermText, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// ─── Demonstrative Items ──────────────────────────────────────────────────────

const createDemonstrativeItem = `
INSERT INTO accounting_demonstrative_items (demonstrative_id, item_code, description, formula, indicator_group, show_in_report, show_bold, is_result, is_100pct, sped_ecf_digit, sped_ecf_type)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, demonstrative_id, item_code, description, formula, indicator_group, show_in_report, show_bold, is_result, is_100pct, sped_ecf_digit, sped_ecf_type, created_at
`

type CreateDemonstrativeItemParams struct {
	DemonstrativeID int64
	ItemCode        int32
	Description     string
	Formula         string
	IndicatorGroup  string
	ShowInReport    bool
	ShowBold        bool
	IsResult        bool
	Is100Pct        bool
	SpedEcfDigit    string
	SpedEcfType     string
}

type DemonstrativeItemRow struct {
	ID              int64
	DemonstrativeID int64
	ItemCode        int32
	Description     string
	Formula         string
	IndicatorGroup  string
	ShowInReport    bool
	ShowBold        bool
	IsResult        bool
	Is100Pct        bool
	SpedEcfDigit    string
	SpedEcfType     string
	CreatedAt       pgtype.Timestamptz
}

func (q *Queries) CreateDemonstrativeItem(ctx context.Context, arg CreateDemonstrativeItemParams) (DemonstrativeItemRow, error) {
	row := q.db.QueryRow(ctx, createDemonstrativeItem,
		arg.DemonstrativeID, arg.ItemCode, arg.Description, arg.Formula, arg.IndicatorGroup,
		arg.ShowInReport, arg.ShowBold, arg.IsResult, arg.Is100Pct, arg.SpedEcfDigit, arg.SpedEcfType,
	)
	var i DemonstrativeItemRow
	err := row.Scan(
		&i.ID, &i.DemonstrativeID, &i.ItemCode, &i.Description, &i.Formula,
		&i.IndicatorGroup, &i.ShowInReport, &i.ShowBold, &i.IsResult, &i.Is100Pct,
		&i.SpedEcfDigit, &i.SpedEcfType, &i.CreatedAt,
	)
	return i, err
}

const listDemonstrativeItems = `
SELECT id, demonstrative_id, item_code, description, formula, indicator_group, show_in_report, show_bold, is_result, is_100pct, sped_ecf_digit, sped_ecf_type, created_at
FROM accounting_demonstrative_items WHERE demonstrative_id = $1 ORDER BY item_code
`

func (q *Queries) ListDemonstrativeItems(ctx context.Context, demonstrativeID int64) ([]DemonstrativeItemRow, error) {
	rows, err := q.db.Query(ctx, listDemonstrativeItems, demonstrativeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DemonstrativeItemRow
	for rows.Next() {
		var i DemonstrativeItemRow
		if err := rows.Scan(
			&i.ID, &i.DemonstrativeID, &i.ItemCode, &i.Description, &i.Formula,
			&i.IndicatorGroup, &i.ShowInReport, &i.ShowBold, &i.IsResult, &i.Is100Pct,
			&i.SpedEcfDigit, &i.SpedEcfType, &i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
