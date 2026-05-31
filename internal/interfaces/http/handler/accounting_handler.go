package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	accounting_uc "github.com/FelipePn10/panossoerp/internal/application/usecase/accounting_uc"
	accountingEntity "github.com/FelipePn10/panossoerp/internal/domain/accounting/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/accounting/ecd"
)

type AccountingHandler struct {
	planUC  *accounting_uc.AccountingPlanUseCase
	acctUC  *accounting_uc.AccountingAccountUseCase
	entryUC *accounting_uc.JournalEntryUseCase
	demUC   *accounting_uc.DemonstrativeUseCase
	ecdUC   *accounting_uc.ECDUseCase
}

func NewAccountingHandler(
	planUC *accounting_uc.AccountingPlanUseCase,
	acctUC *accounting_uc.AccountingAccountUseCase,
	entryUC *accounting_uc.JournalEntryUseCase,
	demUC *accounting_uc.DemonstrativeUseCase,
	ecdUC *accounting_uc.ECDUseCase,
) *AccountingHandler {
	return &AccountingHandler{
		planUC:  planUC,
		acctUC:  acctUC,
		entryUC: entryUC,
		demUC:   demUC,
		ecdUC:   ecdUC,
	}
}

// ─── Plans ────────────────────────────────────────────────────────────────────

type createPlanRequest struct {
	PlanNumber  int    `json:"plan_number"`
	Description string `json:"description"`
	ValidFrom   string `json:"valid_from"`
	ValidTo     string `json:"valid_to"`
	Status      string `json:"status"`
}

func (h *AccountingHandler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	var req createPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	validFrom, err := time.Parse("2006-01-02", req.ValidFrom)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid valid_from date")
		return
	}
	p := &accountingEntity.AccountingPlan{
		PlanNumber:  req.PlanNumber,
		Description: req.Description,
		ValidFrom:   validFrom,
		Status:      accountingEntity.PlanStatus(req.Status),
	}
	if req.ValidTo != "" {
		t, err := time.Parse("2006-01-02", req.ValidTo)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid valid_to date")
			return
		}
		p.ValidTo = &t
	}
	created, err := h.planUC.Create(r.Context(), p)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, created)
}

func (h *AccountingHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.planUC.List(r.Context())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, plans)
}

// ─── Accounts ─────────────────────────────────────────────────────────────────

type createAccountRequest struct {
	PlanID             int64   `json:"plan_id"`
	ParentID           *int64  `json:"parent_id"`
	AccountNumber      string  `json:"account_number"`
	Description        string  `json:"description"`
	NatureCode         string  `json:"nature_code"`
	ReducedCode        *string `json:"reduced_code"`
	RequiresCostCenter bool    `json:"requires_cost_center"`
	ValidFrom          string  `json:"valid_from"`
	ValidTo            string  `json:"valid_to"`
	IsAnalytic         bool    `json:"is_analytic"`
}

func (h *AccountingHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req createAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	validFrom, err := time.Parse("2006-01-02", req.ValidFrom)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid valid_from date")
		return
	}
	a := &accountingEntity.AccountingAccount{
		PlanID:             req.PlanID,
		ParentID:           req.ParentID,
		AccountNumber:      req.AccountNumber,
		Description:        req.Description,
		NatureCode:         req.NatureCode,
		ReducedCode:        req.ReducedCode,
		RequiresCostCenter: req.RequiresCostCenter,
		ValidFrom:          validFrom,
		IsAnalytic:         req.IsAnalytic,
	}
	if req.ValidTo != "" {
		t, err := time.Parse("2006-01-02", req.ValidTo)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid valid_to date")
			return
		}
		a.ValidTo = &t
	}
	created, err := h.acctUC.Create(r.Context(), a)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, created)
}

func (h *AccountingHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	planIDStr := r.URL.Query().Get("plan_id")
	planID, err := strconv.ParseInt(planIDStr, 10, 64)
	if err != nil || planID == 0 {
		jsonError(w, http.StatusBadRequest, "plan_id required")
		return
	}
	accounts, err := h.acctUC.ListByPlan(r.Context(), planID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, accounts)
}

// ─── Journal Entries ──────────────────────────────────────────────────────────

type createJournalEntryRequest struct {
	PlanID          int64   `json:"plan_id"`
	EmpresaID       int     `json:"empresa_id"`
	EntryDate       string  `json:"entry_date"`
	EntryNumber     string  `json:"entry_number"`
	BatchNumber     string  `json:"batch_number"`
	DebitAccountID  int64   `json:"debit_account_id"`
	CreditAccountID int64   `json:"credit_account_id"`
	DebitCCID       *int64  `json:"debit_cc_id"`
	CreditCCID      *int64  `json:"credit_cc_id"`
	Value           float64 `json:"value"`
	HistoryCode     string  `json:"history_code"`
	Description     string  `json:"description"`
	EntryType       string  `json:"entry_type"`
}

func (h *AccountingHandler) CreateJournalEntry(w http.ResponseWriter, r *http.Request) {
	var req createJournalEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	entryDate, err := time.Parse("2006-01-02", req.EntryDate)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid entry_date")
		return
	}
	e := &accountingEntity.AccountingJournalEntry{
		PlanID:          req.PlanID,
		EmpresaID:       req.EmpresaID,
		EntryDate:       entryDate,
		EntryNumber:     req.EntryNumber,
		BatchNumber:     req.BatchNumber,
		DebitAccountID:  req.DebitAccountID,
		CreditAccountID: req.CreditAccountID,
		DebitCCID:       req.DebitCCID,
		CreditCCID:      req.CreditCCID,
		Value:           req.Value,
		HistoryCode:     req.HistoryCode,
		Description:     req.Description,
		EntryType:       req.EntryType,
	}
	created, err := h.entryUC.Create(r.Context(), e)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, created)
}

func (h *AccountingHandler) ListJournalEntries(w http.ResponseWriter, r *http.Request) {
	empresaIDStr := r.URL.Query().Get("empresa_id")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	planIDStr := r.URL.Query().Get("plan_id")

	planID, _ := strconv.ParseInt(planIDStr, 10, 64)
	empresaID, _ := strconv.Atoi(empresaIDStr)
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid from date")
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid to date")
		return
	}

	entries, err := h.entryUC.ListByPeriod(r.Context(), planID, empresaID, from, to)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, entries)
}

// ─── Demonstratives ───────────────────────────────────────────────────────────

type createDemonstrativeRequest struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	TermText    string `json:"term_text"`
}

func (h *AccountingHandler) CreateDemonstrative(w http.ResponseWriter, r *http.Request) {
	var req createDemonstrativeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	d := &accountingEntity.AccountingDemonstrative{
		Code:        req.Code,
		Description: req.Description,
		TermText:    req.TermText,
	}
	created, err := h.demUC.Create(r.Context(), d)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, created)
}

// ─── ECD ──────────────────────────────────────────────────────────────────────

type generateECDRequest struct {
	PlanID    int64          `json:"plan_id"`
	EmpresaID int            `json:"empresa_id"`
	From      string         `json:"from"`
	To        string         `json:"to"`
	Empresa   ecd.ECDEmpresa `json:"empresa"`
	Livros    []ecd.ECDLivro `json:"livros"`
}

func (h *AccountingHandler) GenerateECD(w http.ResponseWriter, r *http.Request) {
	var req generateECDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	from, err := time.Parse("2006-01-02", req.From)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid from date")
		return
	}
	to, err := time.Parse("2006-01-02", req.To)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid to date")
		return
	}

	content, err := h.ecdUC.GenerateECD(r.Context(), accounting_uc.ECDRequest{
		PlanID:    req.PlanID,
		EmpresaID: req.EmpresaID,
		From:      from,
		To:        to,
		Empresa:   req.Empresa,
		Livros:    req.Livros,
	})
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="SPED_ECD.txt"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}
