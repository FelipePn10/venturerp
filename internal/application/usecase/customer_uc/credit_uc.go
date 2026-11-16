package customer_uc

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/security"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	cnpjentity "github.com/FelipePn10/panossoerp/internal/domain/cnpj/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// creditAnalysis é uma rodada da análise de crédito com o que a sustentou.
type creditAnalysis struct {
	suggestion entity.CreditSuggestion
	policy     entity.CreditPolicy
	group      []int64
	company    *cnpjentity.Company // nil quando não houve consulta nova à Receita
	fromCache  bool                // a situação veio do cadastro, não de consulta nova
}

func (uc *CustomerUseCase) requireLifecycle() error {
	if uc.life == nil {
		return fmt.Errorf("módulo de crédito do cliente não configurado")
	}
	return nil
}

func sessionUser(ctx context.Context) (*security.AuthUser, uuid.UUID) {
	u, ok := ctx.Value(contextkey.UserKey).(*security.AuthUser)
	if !ok || u == nil {
		return nil, uuid.Nil
	}
	id, _ := uuid.Parse(u.ID)
	return u, id
}

func isAdmin(ctx context.Context) bool {
	u, _ := sessionUser(ctx)
	return u != nil && u.Role == "ADMIN"
}

// ── Consulta à Receita com cache curto ──────────────────────────────────────
//
// A tela consulta o CNPJ, pede a sugestão e grava o cadastro em sequência; as
// APIs públicas aceitam poucas consultas por minuto. Dez minutos de cache
// evitam três consultas iguais seguidas.

type cachedCompany struct {
	company *cnpjentity.Company
	at      time.Time
}

var (
	companyCacheMu sync.Mutex
	companyCache   = map[string]cachedCompany{}
)

const companyCacheTTL = 10 * time.Minute

// lookupCompany consulta a Receita; fresh ignora o cache (o usuário pediu
// "Reconsultar Receita" e quer a situação de agora).
func (uc *CustomerUseCase) lookupCompany(ctx context.Context, cnpj string, fresh bool) (*cnpjentity.Company, error) {
	digits := onlyDigits(cnpj)
	companyCacheMu.Lock()
	if c, ok := companyCache[digits]; ok && !fresh && uc.now().Sub(c.at) < companyCacheTTL {
		companyCacheMu.Unlock()
		return c.company, nil
	}
	// O cache só serve à sequência consultar → sugerir → gravar: o que passou
	// do prazo sai, para o mapa não crescer com cada CNPJ consultado.
	for k, v := range companyCache {
		if uc.now().Sub(v.at) >= companyCacheTTL {
			delete(companyCache, k)
		}
	}
	companyCacheMu.Unlock()
	if uc.cnpj == nil {
		return nil, fmt.Errorf("consulta de CNPJ não configurada")
	}
	lctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	c, err := uc.cnpj.Lookup(lctx, digits)
	if err != nil {
		return nil, err
	}
	companyCacheMu.Lock()
	companyCache[digits] = cachedCompany{company: c, at: uc.now()}
	companyCacheMu.Unlock()
	return c, nil
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func profileOf(c *cnpjentity.Company) entity.CompanyProfile {
	p := entity.CompanyProfile{Known: true, RegistrationStatus: c.RegistrationStatus, MEI: c.MEI}
	if t, err := time.Parse("2006-01-02", strings.TrimSpace(c.OpeningDate)); err == nil {
		p.OpeningDate = &t
	}
	if d, err := decimal.NewFromString(c.ShareCapital); err == nil {
		p.ShareCapital = &d
	}
	return p
}

func snapshotOf(c *cnpjentity.Company) entity.RegistrationSnapshot {
	p := profileOf(c)
	return entity.RegistrationSnapshot{
		Status: strings.TrimSpace(c.RegistrationStatus), OpeningDate: p.OpeningDate, ShareCapital: p.ShareCapital,
		CompanySize: c.Size, LegalNature: c.LegalNature, MainCNAE: onlyDigits(c.MainActivity.Code),
	}
}

func storedProfile(c *entity.Customer) entity.CompanyProfile {
	p := entity.CompanyProfile{OpeningDate: c.OpeningDate, ShareCapital: c.ShareCapital}
	if c.RegistrationStatus != nil {
		p.Known, p.RegistrationStatus = true, *c.RegistrationStatus
	}
	p.MEI = c.TaxRegime != nil && *c.TaxRegime == "MEI"
	return p
}

// analyzeCredit roda a análise de crédito. Para cliente gravado, a situação na
// Receita só é consultada de novo se a última conferência passou do prazo da
// política; se a Receita não responder, vale o que está gravado e a análise
// diz isso.
func (uc *CustomerUseCase) analyzeCredit(ctx context.Context, existing *entity.Customer, docType entity.DocumentType, document string, _ *cnpjentity.Company) (*creditAnalysis, error) {
	if err := uc.requireLifecycle(); err != nil {
		return nil, err
	}
	pol, err := uc.life.GetCreditPolicy(ctx)
	if err != nil {
		return nil, err
	}
	group, err := uc.life.GroupCustomerCodes(ctx, docType, document)
	if err != nil {
		return nil, err
	}
	if existing != nil && !containsCode(group, existing.Code) {
		group = append(group, existing.Code)
	}
	today := dateOnly(uc.now())
	hist, err := uc.life.GetCreditHistory(ctx, group, pol.HistoryMonths, today)
	if err != nil {
		return nil, err
	}

	a := &creditAnalysis{policy: pol, group: group}
	var profile entity.CompanyProfile
	var aviso *entity.CreditFactor
	fresh := existing != nil && existing.RegistrationCheckedAt != nil &&
		uc.now().Sub(*existing.RegistrationCheckedAt) < time.Duration(pol.RegistrationCheckDays)*24*time.Hour
	switch {
	case docType != entity.DocumentCNPJ:
		if existing != nil {
			profile = storedProfile(existing)
		}
	case fresh:
		profile, a.fromCache = storedProfile(existing), true
	default:
		company, lerr := uc.lookupCompany(ctx, document, false)
		if lerr == nil {
			a.company, profile = company, profileOf(company)
		} else {
			if existing != nil {
				profile = storedProfile(existing)
			}
			aviso = &entity.CreditFactor{Code: "RECEITA_INDISPONIVEL", Label: "Consulta à Receita indisponível",
				Detail: "a análise usou os dados já gravados no cadastro; reconsulte antes de aprovar valores altos"}
		}
	}
	a.suggestion = entity.SuggestCreditLimit(hist, profile, pol, today)
	if aviso != nil {
		a.suggestion.Factors = append(a.suggestion.Factors, *aviso)
	}
	if len(group) > 1 {
		a.suggestion.Factors = append(a.suggestion.Factors, entity.CreditFactor{Code: "GRUPO_ECONOMICO", Label: "Grupo econômico",
			Detail: fmt.Sprintf("histórico somado de %d estabelecimentos com a mesma raiz de CNPJ", len(group))})
	}
	return a, nil
}

func containsCode(list []int64, code int64) bool {
	for _, c := range list {
		if c == code {
			return true
		}
	}
	return false
}

func (uc *CustomerUseCase) checkApproval(ctx context.Context, a *creditAnalysis, approved decimal.Decimal, reason string) (bool, error) {
	return entity.CheckCreditApproval(a.suggestion, approved, reason, isAdmin(ctx), a.policy)
}

func (uc *CustomerUseCase) recordCredit(ctx context.Context, c *entity.Customer, a *creditAnalysis, approved decimal.Decimal, reason, origin string) error {
	deviates := !approved.Equal(a.suggestion.SuggestedLimit)
	_, actor := sessionUser(ctx)
	if actor == uuid.Nil {
		actor = c.CreatedBy
	}
	rev := &entity.CreditReview{
		CustomerCode: c.Code, Origin: origin, SuggestedLimit: a.suggestion.SuggestedLimit, ApprovedLimit: approved,
		Score: a.suggestion.Score, RiskClass: a.suggestion.RiskClass, ExpiresAt: a.suggestion.ExpiresAt,
		Deviates: deviates, Factors: a.suggestion.Factors, CreatedBy: actor,
	}
	if origin != "CADASTRO" {
		prev := c.CreditLimit
		rev.PreviousLimit = &prev
	}
	if r := strings.TrimSpace(reason); r != "" {
		rev.Reason = &r
	}
	return uc.life.ApplyCreditReview(ctx, rev)
}

func toSuggestionResponse(a *creditAnalysis, current *decimal.Decimal, code *int64) *response.CreditSuggestionResponse {
	s := a.suggestion
	teto := entity.MaxWithoutAdmin(s, a.policy)
	r := &response.CreditSuggestionResponse{
		CustomerCode: code, Score: s.Score, RiskClass: string(s.RiskClass), SuggestedLimit: s.SuggestedLimit.InexactFloat64(),
		ExpiresAt: s.ExpiresAt, HasHistory: s.HasHistory, HardStop: s.HardStop, Deteriorating: s.Deteriorating,
		GroupMembers: a.group, RegistrationOK: a.company != nil || a.fromCache,
		MaxWithoutAdmin: teto.Round(2).InexactFloat64(), Factors: toFactorResponses(s.Factors),
	}
	if r.GroupMembers == nil {
		r.GroupMembers = []int64{}
	}
	if current != nil {
		f := current.InexactFloat64()
		r.CurrentLimit = &f
	}
	return r
}

func toFactorResponses(fs []entity.CreditFactor) []response.CreditFactorResponse {
	out := make([]response.CreditFactorResponse, 0, len(fs))
	for _, f := range fs {
		out = append(out, response.CreditFactorResponse(f))
	}
	return out
}

// SuggestCredit devolve a análise de um cliente gravado (code) ou de um CNPJ
// ainda não cadastrado (document), sem gravar nada.
func (uc *CustomerUseCase) SuggestCredit(ctx context.Context, code *int64, docType, document string) (*response.CreditSuggestionResponse, error) {
	if code != nil {
		c, err := uc.repo.GetCustomerByCode(ctx, *code)
		if err != nil {
			return nil, err
		}
		a, err := uc.analyzeCredit(ctx, c, c.DocumentType, c.DocumentNumber, nil)
		if err != nil {
			return nil, err
		}
		if a.company != nil {
			// Só guarda o que a Receita respondeu (poupa a cota da consulta). Não
			// bloqueia: isto é uma leitura; quem bloqueia é a gravação do limite,
			// a reconsulta e o agendador.
			if err := uc.life.SetCustomerRegistration(ctx, c.Code, snapshotOf(a.company)); err != nil {
				return nil, err
			}
		}
		return toSuggestionResponse(a, &c.CreditLimit, &c.Code), nil
	}
	dt := entity.DocumentType(strings.ToUpper(strings.TrimSpace(docType)))
	if dt == "" {
		dt = entity.DocumentCNPJ
	}
	digits := onlyDigits(document)
	if (dt == entity.DocumentCNPJ && len(digits) != 14) || (dt == entity.DocumentCPF && len(digits) != 11) {
		return nil, errorsuc.NewValidationError("informe o CNPJ (14 dígitos) ou o CPF (11 dígitos) para calcular o limite")
	}
	a, err := uc.analyzeCredit(ctx, nil, dt, digits, nil)
	if err != nil {
		return nil, err
	}
	return toSuggestionResponse(a, nil, nil), nil
}

// SetCreditLimit grava um limite revisado contra uma análise feita agora.
func (uc *CustomerUseCase) SetCreditLimit(ctx context.Context, code int64, dto request.SetCreditLimitDTO) (*response.CustomerResponse, error) {
	c, err := uc.repo.GetCustomerByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	a, err := uc.analyzeCredit(ctx, c, c.DocumentType, c.DocumentNumber, nil)
	if err != nil {
		return nil, err
	}
	if _, err := uc.checkApproval(ctx, a, dto.CreditLimit, dto.Reason); err != nil {
		return nil, err
	}
	if a.company != nil {
		if err := uc.applyRegistration(ctx, c, a.company); err != nil {
			return nil, err
		}
	}
	if err := uc.recordCredit(ctx, c, a, dto.CreditLimit, dto.Reason, "REVISAO"); err != nil {
		return nil, err
	}
	updated, err := uc.repo.GetCustomerByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return uc.customerResponse(ctx, updated)
}

func (uc *CustomerUseCase) ListCreditReviews(ctx context.Context, code int64) ([]response.CreditReviewResponse, error) {
	if err := uc.requireLifecycle(); err != nil {
		return nil, err
	}
	list, err := uc.life.ListCreditReviews(ctx, code)
	if err != nil {
		return nil, err
	}
	out := make([]response.CreditReviewResponse, 0, len(list))
	for _, r := range list {
		out = append(out, response.CreditReviewResponse{
			ID: r.ID, Origin: r.Origin, SuggestedLimit: r.SuggestedLimit.InexactFloat64(), ApprovedLimit: r.ApprovedLimit.InexactFloat64(),
			PreviousLimit: floatPtr(r.PreviousLimit), Score: r.Score, RiskClass: string(r.RiskClass), ExpiresAt: r.ExpiresAt,
			Deviates: r.Deviates, Reason: r.Reason, Factors: toFactorResponses(r.Factors), CreatedBy: r.CreatedBy.String(), CreatedByName: r.CreatedByName, CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

// ── Política de crédito ──────────────────────────────────────────────────────

func toPolicyResponse(p entity.CreditPolicy) *response.CreditPolicyResponse {
	return &response.CreditPolicyResponse{
		MaxLimit: p.MaxLimit.InexactFloat64(), NewCustomerCapitalPct: p.NewCustomerCapitalPct.InexactFloat64(),
		NewCustomerCap: p.NewCustomerCap.InexactFloat64(), OverrideTolerancePct: p.OverrideTolerancePct.InexactFloat64(),
		OverdueBlockDays: p.OverdueBlockDays, HistoryMonths: p.HistoryMonths,
		ValidityMonthsA: p.ValidityMonths[entity.RiskA], ValidityMonthsB: p.ValidityMonths[entity.RiskB],
		ValidityMonthsC: p.ValidityMonths[entity.RiskC], ValidityMonthsD: p.ValidityMonths[entity.RiskD],
		ValidityMonthsE: p.ValidityMonths[entity.RiskE], RegistrationCheckDays: p.RegistrationCheckDays,
	}
}

func (uc *CustomerUseCase) GetCreditPolicy(ctx context.Context) (*response.CreditPolicyResponse, error) {
	if err := uc.requireLifecycle(); err != nil {
		return nil, err
	}
	p, err := uc.life.GetCreditPolicy(ctx)
	if err != nil {
		return nil, err
	}
	return toPolicyResponse(p), nil
}

func (uc *CustomerUseCase) SaveCreditPolicy(ctx context.Context, dto request.CreditPolicyDTO) (*response.CreditPolicyResponse, error) {
	if err := uc.requireLifecycle(); err != nil {
		return nil, err
	}
	if !isAdmin(ctx) {
		return nil, errorsuc.NewValidationError("só um administrador altera a política de crédito")
	}
	p := entity.CreditPolicy{
		MaxLimit: dto.MaxLimit, NewCustomerCapitalPct: dto.NewCustomerCapitalPct, NewCustomerCap: dto.NewCustomerCap,
		OverrideTolerancePct: dto.OverrideTolerancePct, OverdueBlockDays: dto.OverdueBlockDays, HistoryMonths: dto.HistoryMonths,
		ValidityMonths: map[entity.RiskClass]int{
			entity.RiskA: dto.ValidityMonthsA, entity.RiskB: dto.ValidityMonthsB, entity.RiskC: dto.ValidityMonthsC,
			entity.RiskD: dto.ValidityMonthsD, entity.RiskE: dto.ValidityMonthsE,
		},
		RegistrationCheckDays: dto.RegistrationCheckDays,
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	_, actor := sessionUser(ctx)
	if err := uc.life.SaveCreditPolicy(ctx, p, actor); err != nil {
		return nil, err
	}
	return toPolicyResponse(p), nil
}

// ── Ciclo de vida do cadastro ────────────────────────────────────────────────

func (uc *CustomerUseCase) CustomerUsageReport(ctx context.Context, code int64) (*response.CustomerUsageReportResponse, error) {
	if err := uc.requireLifecycle(); err != nil {
		return nil, err
	}
	usages, err := uc.life.CustomerUsage(ctx, code)
	if err != nil {
		return nil, err
	}
	r := &response.CustomerUsageReportResponse{CanDelete: true, CanInactivate: true}
	for _, u := range usages {
		r.Usages = append(r.Usages, response.CustomerUsageResponse(u))
	}
	if err := entity.CheckCanDelete(usages); err != nil {
		r.CanDelete, r.DeleteReason = false, err.Error()
	}
	if err := entity.CheckCanInactivate(usages); err != nil {
		r.CanInactivate, r.InactivateReason = false, err.Error()
	}
	return r, nil
}

func (uc *CustomerUseCase) InactivateCustomer(ctx context.Context, code int64) error {
	if err := uc.requireLifecycle(); err != nil {
		return err
	}
	usages, err := uc.life.CustomerUsage(ctx, code)
	if err != nil {
		return err
	}
	if err := entity.CheckCanInactivate(usages); err != nil {
		return err
	}
	return uc.life.SetCustomerActive(ctx, code, false)
}

func (uc *CustomerUseCase) ReactivateCustomer(ctx context.Context, code int64) error {
	if err := uc.requireLifecycle(); err != nil {
		return err
	}
	return uc.life.SetCustomerActive(ctx, code, true)
}

// DeleteCustomer exclui só cliente que nunca foi usado. Endereços e contatos
// saem junto (cascata); a trilha de crédito fica.
func (uc *CustomerUseCase) DeleteCustomer(ctx context.Context, code int64) error {
	if err := uc.requireLifecycle(); err != nil {
		return err
	}
	return uc.life.DeleteCustomer(ctx, code, entity.CheckCanDelete)
}

// ── Painel 360 ───────────────────────────────────────────────────────────────

func (uc *CustomerUseCase) CustomerOverview(ctx context.Context, code int64) (*response.CustomerOverviewResponse, error) {
	if err := uc.requireLifecycle(); err != nil {
		return nil, err
	}
	c, err := uc.repo.GetCustomerByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	pol, err := uc.life.GetCreditPolicy(ctx)
	if err != nil {
		return nil, err
	}
	group, err := uc.life.GroupCustomerCodes(ctx, c.DocumentType, c.DocumentNumber)
	if err != nil {
		return nil, err
	}
	if !containsCode(group, c.Code) {
		group = append(group, c.Code)
	}
	today := dateOnly(uc.now())
	o, err := uc.life.CustomerOverview(ctx, code, group, today)
	if err != nil {
		return nil, err
	}
	hist, err := uc.life.GetCreditHistory(ctx, []int64{code}, pol.HistoryMonths, today)
	if err != nil {
		return nil, err
	}
	groupHist := hist
	if len(group) > 1 {
		if groupHist, err = uc.life.GetCreditHistory(ctx, group, pol.HistoryMonths, today); err != nil {
			return nil, err
		}
	}
	r := &response.CustomerOverviewResponse{
		OpenOrders: o.OpenOrders, OpenOrdersValue: o.OpenOrdersValue.InexactFloat64(), ProductionOrders: o.ProductionOrders,
		OpenQuotations: o.OpenQuotations, OpenServiceOrders: o.OpenServiceOrders,
		OpenReceivables: o.OpenReceivables.InexactFloat64(), OverdueReceivables: o.OverdueReceivables.InexactFloat64(),
		MaxOpenOverdueDays: o.MaxOpenOverdueDays, Billed12m: o.Billed12m.InexactFloat64(),
		LastOrderDate: o.LastOrderDate, LastInvoiceDate: o.LastInvoiceDate,
		AvgDelayDays: hist.WeightedDelayDays.Round(1).InexactFloat64(), MaxDelayDays: hist.MaxDelayDays,
		GroupMembers: o.GroupMembers, GroupExposure: o.GroupExposure.InexactFloat64(), ConcentrationPct: o.ConcentrationPct.InexactFloat64(),
		CreditLimit: c.CreditLimit.InexactFloat64(), CreditUsed: o.CreditUsed.InexactFloat64(),
		CreditAvailable: c.CreditLimit.Sub(o.CreditUsed).InexactFloat64(), Alerts: []string{},
	}
	if hist.PaidValue.IsPositive() {
		pct := hist.PaidOnTimeValue.Div(hist.PaidValue).Mul(decimal.NewFromInt(100)).Round(1).InexactFloat64()
		r.OnTimePct = &pct
	}
	for _, m := range o.BillingByMonth {
		r.BillingByMonth = append(r.BillingByMonth, response.MonthValueResponse{Month: m.Month, Value: m.Value.InexactFloat64()})
	}
	r.Alerts = overviewAlerts(c, o, hist, groupHist, pol, today, uc.now())
	return r, nil
}

func overviewAlerts(c *entity.Customer, o *entity.CustomerOverview, hist, groupHist entity.CreditHistory, pol entity.CreditPolicy, today, now time.Time) []string {
	var a []string
	if c.CreditReviewedAt == nil && c.CreditRiskClass == nil {
		a = append(a, "Cliente sem análise de crédito: calcule o limite na pasta Crédito.")
	}
	if c.CreditLimitExpiresAt != nil && c.CreditLimitExpiresAt.Before(today) {
		a = append(a, fmt.Sprintf("Limite de crédito vencido em %s: pedidos novos vão para análise até a revisão.", c.CreditLimitExpiresAt.Format("02/01/2006")))
	}
	if c.CreditRiskClass != nil && (*c.CreditRiskClass == "D" || *c.CreditRiskClass == "E") {
		a = append(a, fmt.Sprintf("Classe de risco %s: todo pedido passa pela análise de crédito.", *c.CreditRiskClass))
	}
	if c.RegistrationStatus != nil && !strings.EqualFold(*c.RegistrationStatus, "ATIVA") {
		a = append(a, "Situação na Receita: "+*c.RegistrationStatus+".")
	}
	if c.DocumentType == entity.DocumentCNPJ && (c.RegistrationCheckedAt == nil ||
		now.Sub(*c.RegistrationCheckedAt) > time.Duration(pol.RegistrationCheckDays)*24*time.Hour) {
		a = append(a, "Situação na Receita não conferida no prazo da política.")
	}
	if c.CreditLimit.IsPositive() && o.CreditUsed.GreaterThan(c.CreditLimit) {
		a = append(a, fmt.Sprintf("Exposição (%s) acima do limite (%s).", entity.Reais(o.CreditUsed), entity.Reais(c.CreditLimit)))
	}
	if hist.MaxOpenOverdueDays > 0 {
		a = append(a, fmt.Sprintf("%s vencidos, o mais antigo há %d dias.", entity.Reais(hist.OverdueBalance), hist.MaxOpenOverdueDays))
	}
	if groupHist.MaxOpenOverdueDays > pol.OverdueBlockDays && hist.MaxOpenOverdueDays <= pol.OverdueBlockDays {
		a = append(a, fmt.Sprintf("Outro estabelecimento do grupo econômico tem título vencido há %d dias.", groupHist.MaxOpenOverdueDays))
	}
	if hist.RecentPaidTitles > 0 && hist.RecentWeightedDelayDays.Sub(hist.WeightedDelayDays).GreaterThan(decimal.NewFromInt(10)) {
		a = append(a, "Os pagamentos dos últimos 90 dias atrasaram mais que o histórico do cliente.")
	}
	if o.ConcentrationPct.GreaterThanOrEqual(decimal.NewFromInt(20)) {
		a = append(a, fmt.Sprintf("Concentração: %s%% do contas a receber em aberto da empresa está neste cliente.", entity.Numero(o.ConcentrationPct, 1)))
	}
	if c.RequiresQualityCert {
		a = append(a, "Exige certificado de qualidade na entrega.")
	}
	if c.RequiresCustomerPO {
		a = append(a, "Exige o número do pedido de compra do cliente na nota.")
	}
	return a
}

// ── Situação na Receita ──────────────────────────────────────────────────────

// RefreshRegistration reconsulta o CNPJ na Receita, grava a situação e, se a
// empresa deixou de estar ATIVA, bloqueia o cliente.
func (uc *CustomerUseCase) RefreshRegistration(ctx context.Context, code int64) (*response.CustomerResponse, error) {
	if err := uc.requireLifecycle(); err != nil {
		return nil, err
	}
	c, err := uc.repo.GetCustomerByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if c.DocumentType != entity.DocumentCNPJ {
		return nil, errorsuc.NewValidationError("a reconsulta à Receita vale só para CNPJ")
	}
	if err := uc.life.MarkRegistrationAttempt(ctx, code); err != nil {
		return nil, err
	}
	company, err := uc.lookupCompany(ctx, c.DocumentNumber, true)
	if err != nil {
		return nil, errorsuc.NewValidationError("a Receita não respondeu agora; tente de novo em alguns minutos (" + err.Error() + ")")
	}
	if err := uc.applyRegistration(ctx, c, company); err != nil {
		return nil, err
	}
	updated, err := uc.repo.GetCustomerByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return uc.customerResponse(ctx, updated)
}

func (uc *CustomerUseCase) applyRegistration(ctx context.Context, c *entity.Customer, company *cnpjentity.Company) error {
	snap := snapshotOf(company)
	if err := uc.life.SetCustomerRegistration(ctx, c.Code, snap); err != nil {
		return err
	}
	if snap.Status != "" && !strings.EqualFold(snap.Status, "ATIVA") && !c.Blocked {
		reason := fmt.Sprintf("Situação na Receita: %s (conferido em %s). Bloqueado automaticamente.", snap.Status, uc.now().Format("02/01/2006"))
		if err := uc.repo.BlockCustomer(ctx, c.Code, reason); err != nil {
			return err
		}
	}
	return nil
}

// RecheckRegistration é o passo do agendador: reconsulta um cliente já com a
// sessão da empresa dele no contexto.
func (uc *CustomerUseCase) RecheckRegistration(ctx context.Context, code int64) error {
	if err := uc.requireLifecycle(); err != nil {
		return err
	}
	c, err := uc.repo.GetCustomerByCode(ctx, code)
	if err != nil {
		return err
	}
	if err := uc.life.MarkRegistrationAttempt(ctx, code); err != nil {
		return err
	}
	company, err := uc.lookupCompany(ctx, c.DocumentNumber, false)
	if err != nil {
		return err
	}
	return uc.applyRegistration(ctx, c, company)
}

// ── Endereços e contatos: alteração e exclusão ───────────────────────────────

func (uc *CustomerUseCase) customerAddresses(ctx context.Context, code int64) (*entity.Customer, []*entity.CustomerAddress, error) {
	c, err := uc.repo.GetCustomerByCode(ctx, code)
	if err != nil {
		return nil, nil, err
	}
	list, err := uc.repo.ListAddresses(ctx, c.ID)
	return c, list, err
}

func (uc *CustomerUseCase) UpdateAddress(ctx context.Context, code, id int64, dto request.AddAddressDTO) (*response.CustomerAddressResponse, error) {
	c, list, err := uc.customerAddresses(ctx, code)
	if err != nil {
		return nil, err
	}
	if !hasAddress(list, id) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("endereço %d não é do cliente %d", id, code))
	}
	country := dto.Country
	if country == "" {
		country = "Brasil"
	}
	updated, err := uc.repo.UpdateAddress(ctx, &entity.CustomerAddress{
		ID: id, CustomerID: c.ID, AddressType: entity.AddressType(dto.AddressType), ZipCode: dto.ZipCode, Street: dto.Street,
		Number: dto.Number, Complement: dto.Complement, Neighborhood: dto.Neighborhood, City: dto.City, UF: dto.UF,
		Country: country, IsDefault: dto.IsDefault,
	})
	if err != nil {
		return nil, err
	}
	r := toAddressResponse(updated)
	return &r, nil
}

// DeleteAddress não deixa cliente ativo sem endereço: a nota fiscal precisa
// de um.
func (uc *CustomerUseCase) DeleteAddress(ctx context.Context, code, id int64) error {
	c, list, err := uc.customerAddresses(ctx, code)
	if err != nil {
		return err
	}
	if !hasAddress(list, id) {
		return errorsuc.NewNotFoundError(fmt.Sprintf("endereço %d não é do cliente %d", id, code))
	}
	if len(list) == 1 && c.IsActive {
		return errorsuc.NewValidationError("o cliente precisa de ao menos um endereço para a nota fiscal; cadastre outro antes de excluir este")
	}
	return uc.repo.DeleteAddress(ctx, id)
}

func hasAddress(list []*entity.CustomerAddress, id int64) bool {
	for _, a := range list {
		if a.ID == id {
			return true
		}
	}
	return false
}

func (uc *CustomerUseCase) contactOf(ctx context.Context, code, id int64) (*entity.Customer, error) {
	c, err := uc.repo.GetCustomerByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	list, err := uc.repo.ListContacts(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	for _, ct := range list {
		if ct.ID == id {
			return c, nil
		}
	}
	return nil, errorsuc.NewNotFoundError(fmt.Sprintf("contato %d não é do cliente %d", id, code))
}

func (uc *CustomerUseCase) UpdateContact(ctx context.Context, code, id int64, dto request.AddContactDTO) (*response.CustomerContactResponse, error) {
	c, err := uc.contactOf(ctx, code, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(dto.Name) == "" {
		return nil, errorsuc.NewValidationError("informe o nome do contato")
	}
	var contactTypeID *int64
	if dto.ContactTypeCode != nil {
		ct, err := uc.repo.GetContactTypeByCode(ctx, *dto.ContactTypeCode)
		if err != nil {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("tipo de contato %d não encontrado", *dto.ContactTypeCode))
		}
		contactTypeID = &ct.ID
	}
	updated, err := uc.repo.UpdateContact(ctx, &entity.CustomerContact{
		ID: id, CustomerID: c.ID, ContactTypeID: contactTypeID, Name: dto.Name, Email: dto.Email, Phone: dto.Phone,
		Mobile: dto.Mobile, Position: dto.Position, IsPrimary: dto.IsPrimary, IsActive: true,
	})
	if err != nil {
		return nil, err
	}
	r := toContactResponse(updated)
	return &r, nil
}

func (uc *CustomerUseCase) DeleteContact(ctx context.Context, code, id int64) error {
	if _, err := uc.contactOf(ctx, code, id); err != nil {
		return err
	}
	return uc.repo.DeleteContact(ctx, id)
}

// DeleteSupport exclui um cadastro de apoio (região, segmento, tipos,
// portador, grupo, tipo de NF, tipo de imposto) que não esteja em uso.
func (uc *CustomerUseCase) DeleteSupport(ctx context.Context, resource string, code int64) error {
	if err := uc.requireLifecycle(); err != nil {
		return err
	}
	return uc.life.DeleteSupport(ctx, resource, code)
}

// SetSupportActive inativa (some das listas de escolha, o histórico fica) ou
// reativa um cadastro de apoio: é o caminho para o que já foi usado e não sai.
func (uc *CustomerUseCase) SetSupportActive(ctx context.Context, resource string, code int64, active bool) error {
	if err := uc.requireLifecycle(); err != nil {
		return err
	}
	return uc.life.SetSupportActive(ctx, resource, code, active)
}
