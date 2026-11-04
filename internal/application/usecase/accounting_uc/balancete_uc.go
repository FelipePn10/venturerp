package accounting_uc

import (
	"context"
	"sort"
	"time"

	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/accounting/repository"
	"github.com/shopspring/decimal"
)

// BalanceteLine is one account row of the trial balance (Balancete).
type BalanceteLine struct {
	AccountID     int64   `json:"account_id"`
	AccountNumber string  `json:"account_number"`
	Description   string  `json:"description"`
	Debit         float64 `json:"debit"`
	Credit        float64 `json:"credit"`
	Balance       float64 `json:"balance"` // debit - credit
}

// BalanceteResult is the consolidated trial balance for a period.
type BalanceteResult struct {
	PlanID      int64           `json:"plan_id"`
	PeriodStart string          `json:"period_start"`
	PeriodEnd   string          `json:"period_end"`
	Lines       []BalanceteLine `json:"lines"`
	TotalDebit  float64         `json:"total_debit"`
	TotalCredit float64         `json:"total_credit"`
	Balanced    bool            `json:"balanced"`
}

// BalanceteUseCase builds a trial balance by aggregating the journal entries of
// the period per account (a posting debits one account and credits another).
type BalanceteUseCase struct {
	Repo domainrepo.AccountingRepository
}

func (uc *BalanceteUseCase) Execute(ctx context.Context, planID int64, empresaID int, from, to time.Time) (*BalanceteResult, error) {
	entries, err := uc.Repo.ListJournalEntries(ctx, planID, empresaID, from, to)
	if err != nil {
		return nil, err
	}
	accounts, err := uc.Repo.ListAccountingAccountsByPlan(ctx, planID)
	if err != nil {
		return nil, err
	}

	type accInfo struct {
		number string
		desc   string
	}
	info := make(map[int64]accInfo, len(accounts))
	for _, a := range accounts {
		info[a.ID] = accInfo{number: a.AccountNumber, desc: a.Description}
	}

	debits := map[int64]decimal.Decimal{}
	credits := map[int64]decimal.Decimal{}
	for _, e := range entries {
		debits[e.DebitAccountID] = debits[e.DebitAccountID].Add(e.Value)
		credits[e.CreditAccountID] = credits[e.CreditAccountID].Add(e.Value)
	}

	touched := map[int64]struct{}{}
	for id := range debits {
		touched[id] = struct{}{}
	}
	for id := range credits {
		touched[id] = struct{}{}
	}

	res := &BalanceteResult{
		PlanID:      planID,
		PeriodStart: from.Format("2006-01-02"),
		PeriodEnd:   to.Format("2006-01-02"),
	}
	totalDebit, totalCredit := decimal.Zero, decimal.Zero
	for id := range touched {
		d := debits[id]
		c := credits[id]
		line := BalanceteLine{
			AccountID:     id,
			AccountNumber: info[id].number,
			Description:   info[id].desc,
			Debit:         d.InexactFloat64(),
			Credit:        c.InexactFloat64(),
			Balance:       d.Sub(c).InexactFloat64(),
		}
		res.Lines = append(res.Lines, line)
		totalDebit = totalDebit.Add(d)
		totalCredit = totalCredit.Add(c)
	}
	sort.Slice(res.Lines, func(i, j int) bool {
		return res.Lines[i].AccountNumber < res.Lines[j].AccountNumber
	})
	// Double-entry bookkeeping: total debits must equal total credits.
	res.TotalDebit, res.TotalCredit = totalDebit.InexactFloat64(), totalCredit.InexactFloat64()
	res.Balanced = totalDebit.Equal(totalCredit)
	return res, nil
}
