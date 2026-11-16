package entity

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// CustomerUsage é um lugar do sistema onde o cliente aparece. Open conta só o
// que ainda está em andamento (pedido aberto, OP em fabricação, título a
// receber...): isso impede inativar. Total conta qualquer registro: isso
// impede excluir, porque o histórico ficaria apontando para o nada.
type CustomerUsage struct {
	Code  string `json:"code"`
	Label string `json:"label"`
	Total int64  `json:"total"`
	Open  int64  `json:"open"`
}

// CheckCanInactivate recusa inativar quem ainda tem algo em andamento.
func CheckCanInactivate(usages []CustomerUsage) error {
	var pend []string
	for _, u := range usages {
		if u.Open > 0 {
			pend = append(pend, fmt.Sprintf("%d %s", u.Open, u.Label))
		}
	}
	if len(pend) > 0 {
		return domainerr.NewValidation("o cliente ainda tem movimento em andamento: " + strings.Join(pend, "; ") +
			". Conclua ou cancele antes de inativar")
	}
	return nil
}

// CheckCanDelete só deixa excluir quem nunca foi usado; quem já teve movimento
// pode ser inativado.
func CheckCanDelete(usages []CustomerUsage) error {
	var usos []string
	for _, u := range usages {
		if u.Total > 0 {
			usos = append(usos, fmt.Sprintf("%d %s", u.Total, u.Label))
		}
	}
	if len(usos) > 0 {
		return domainerr.NewConflict("o cliente já tem histórico e não pode ser excluído (" + strings.Join(usos, "; ") +
			"). Use Inativar: ele some dos pedidos novos e o histórico continua consultável")
	}
	return nil
}

// RegistrationSnapshot é o que a consulta do CNPJ trouxe.
type RegistrationSnapshot struct {
	Status       string
	OpeningDate  *time.Time
	ShareCapital *decimal.Decimal
	CompanySize  string
	LegalNature  string
	MainCNAE     string
}

// RegistrationCheckTarget é um cliente a reconsultar na Receita.
type RegistrationCheckTarget struct {
	EnterpriseID   int64
	EnterpriseCode int64
	Actor          uuid.UUID
	Code           int64
	DocumentNumber string
	Blocked        bool
}

// CreditReview é uma linha da trilha de crédito.
type CreditReview struct {
	ID             int64
	CustomerCode   int64
	Origin         string // CADASTRO, REVISAO, LEGADO
	SuggestedLimit decimal.Decimal
	ApprovedLimit  decimal.Decimal
	PreviousLimit  *decimal.Decimal
	Score          int
	RiskClass      RiskClass
	ExpiresAt      time.Time
	Deviates       bool
	Reason         *string
	Factors        []CreditFactor
	CreatedBy      uuid.UUID
	CreatedByName  string // só na leitura da trilha
	CreatedAt      time.Time
}

// CustomerOverview é o painel 360 do cliente.
type CustomerOverview struct {
	OpenOrders         int64           `json:"open_orders"`
	OpenOrdersValue    decimal.Decimal `json:"open_orders_value"`
	ProductionOrders   int64           `json:"production_orders_in_progress"`
	OpenQuotations     int64           `json:"open_quotations"`
	OpenReceivables    decimal.Decimal `json:"open_receivables"`
	OverdueReceivables decimal.Decimal `json:"overdue_receivables"`
	MaxOpenOverdueDays int             `json:"max_open_overdue_days"`
	Billed12m          decimal.Decimal `json:"billed_12m"`
	LastOrderDate      *time.Time      `json:"last_order_date"`
	LastInvoiceDate    *time.Time      `json:"last_invoice_date"`
	AvgDelayDays       decimal.Decimal `json:"avg_delay_days"`
	GroupMembers       int             `json:"group_members"`
	GroupExposure      decimal.Decimal `json:"group_exposure"`
	ConcentrationPct   decimal.Decimal `json:"concentration_pct"` // fatia do contas a receber em aberto da empresa
	CreditUsed         decimal.Decimal `json:"credit_used"`
	CreditAvailable    decimal.Decimal `json:"credit_available"`
	BillingByMonth     []MonthValue    `json:"billing_by_month"`
	OpenServiceOrders  int64           `json:"open_service_orders"`
}

type MonthValue struct {
	Month string          `json:"month"` // AAAA-MM
	Value decimal.Decimal `json:"value"`
}
