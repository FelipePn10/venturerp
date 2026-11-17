package entity

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// Limite de crédito sugerido pelo sistema (VCLI0500).
//
// O score vai de 0 a 1000 e soma fatores explicados um a um, para quem aprova
// ver por que o número saiu assim. Quem já comprou a prazo é avaliado pelo que
// pagou (até 850 pontos) e pela idade da empresa (150); cliente novo parte de
// 350 e só soma idade e capital social — nunca passa da classe C sem histórico.
// Fatos graves (CNPJ irregular na Receita, título vencido além da tolerância,
// protesto, título baixado como perda) zeram o limite: classe E.

type RiskClass string

const (
	RiskA RiskClass = "A"
	RiskB RiskClass = "B"
	RiskC RiskClass = "C"
	RiskD RiskClass = "D"
	RiskE RiskClass = "E"
)

// CreditPolicy são os parâmetros de crédito da empresa (VCLI0500 › apoio).
type CreditPolicy struct {
	MaxLimit              decimal.Decimal
	NewCustomerCapitalPct decimal.Decimal
	// NewCustomerCap é o piso do apetite: o mínimo que a empresa aceita expor
	// num cliente novo mesmo com pouco faturamento no sistema.
	NewCustomerCap decimal.Decimal
	// NewCustomerRevenuePct: o apetite por cliente novo, em % do faturamento
	// médio mensal da empresa (regra de concentração).
	NewCustomerRevenuePct decimal.Decimal
	OverrideTolerancePct  decimal.Decimal
	OverdueBlockDays      int
	HistoryMonths         int
	ValidityMonths        map[RiskClass]int
	RegistrationCheckDays int
}

func DefaultCreditPolicy() CreditPolicy {
	return CreditPolicy{
		NewCustomerCapitalPct: decimal.NewFromInt(10),
		NewCustomerCap:        decimal.NewFromInt(30000),
		NewCustomerRevenuePct: decimal.NewFromInt(20),
		OverrideTolerancePct:  decimal.NewFromInt(10),
		OverdueBlockDays:      30,
		HistoryMonths:         24,
		ValidityMonths:        map[RiskClass]int{RiskA: 12, RiskB: 6, RiskC: 3, RiskD: 1, RiskE: 1},
		RegistrationCheckDays: 30,
	}
}

func (p CreditPolicy) Validate() error {
	switch {
	case p.MaxLimit.IsNegative(), p.NewCustomerCap.IsNegative():
		return domainerr.NewValidation("os valores da política de crédito não podem ser negativos")
	case p.NewCustomerCapitalPct.IsNegative() || p.NewCustomerCapitalPct.GreaterThan(decimal.NewFromInt(100)),
		p.NewCustomerRevenuePct.IsNegative() || p.NewCustomerRevenuePct.GreaterThan(decimal.NewFromInt(100)),
		p.OverrideTolerancePct.IsNegative() || p.OverrideTolerancePct.GreaterThan(decimal.NewFromInt(100)):
		return domainerr.NewValidation("os percentuais da política de crédito vão de 0 a 100")
	case p.OverdueBlockDays <= 0, p.RegistrationCheckDays <= 0:
		return domainerr.NewValidation("informe os dias de tolerância de atraso e de reconsulta da Receita")
	case p.HistoryMonths < 6 || p.HistoryMonths > 60:
		return domainerr.NewValidation("o histórico analisado vai de 6 a 60 meses")
	}
	for _, c := range []RiskClass{RiskA, RiskB, RiskC, RiskD, RiskE} {
		if p.ValidityMonths[c] <= 0 {
			return domainerr.NewValidation(fmt.Sprintf("informe a validade do limite da classe %s", c))
		}
	}
	return nil
}

// CreditHistory é o comportamento de pagamento do cliente (ou do grupo
// econômico) nos títulos a receber desta empresa, dentro da janela da política.
type CreditHistory struct {
	PaidTitles              int
	PaidValue               decimal.Decimal
	PaidOnTimeValue         decimal.Decimal
	WeightedDelayDays       decimal.Decimal // atraso médio ponderado pelo valor
	RecentPaidTitles        int             // pagos nos últimos 90 dias
	RecentWeightedDelayDays decimal.Decimal
	MaxDelayDays            int
	AvgTermDays             decimal.Decimal // prazo concedido médio (vencimento − emissão)
	Billed12m               decimal.Decimal
	RelationshipMonths      int // desde o primeiro título
	Renegotiated            int
	LostTitles              int
	OpenBalance             decimal.Decimal
	OverdueBalance          decimal.Decimal
	MaxOpenOverdueDays      int
	OpenProtest             bool
	Returns12m              decimal.Decimal
}

// CompanyProfile é o que a Receita diz da empresa (consulta do CNPJ).
type CompanyProfile struct {
	Known              bool
	RegistrationStatus string
	OpeningDate        *time.Time
	ShareCapital       *decimal.Decimal
	MEI                bool
}

type CreditFactor struct {
	Code      string `json:"code"`
	Label     string `json:"label"`
	Points    int    `json:"points"`
	MaxPoints int    `json:"max_points"`
	Detail    string `json:"detail"`
	Blocking  bool   `json:"blocking,omitempty"`
}

type CreditSuggestion struct {
	Score          int
	RiskClass      RiskClass
	SuggestedLimit decimal.Decimal
	ExpiresAt      time.Time
	HasHistory     bool
	HardStop       bool
	Deteriorating  bool
	Factors        []CreditFactor
}

const minPaidTitlesForHistory = 3

// SuggestCreditLimit calcula score, classe, limite e validade.
// SellerVolume é o faturamento da própria empresa (notas de venda autorizadas
// nos últimos 12 meses) e em quantos meses ela faturou: mede quanto ela pode
// arriscar num cliente novo.
type SellerVolume struct {
	Billed12m decimal.Decimal
	Months    int
}

// MonthlyAverage: a média dos meses em que a empresa faturou (no máximo 12).
func (v SellerVolume) MonthlyAverage() decimal.Decimal {
	if v.Months <= 0 || !v.Billed12m.IsPositive() {
		return decimal.Zero
	}
	return v.Billed12m.Div(decimal.NewFromInt(int64(min(v.Months, 12))))
}

func SuggestCreditLimit(h CreditHistory, c CompanyProfile, pol CreditPolicy, seller SellerVolume, today time.Time) CreditSuggestion {
	s := CreditSuggestion{HasHistory: h.PaidTitles >= minPaidTitlesForHistory}
	add := func(f CreditFactor) { s.Factors = append(s.Factors, f); s.Score += f.Points }

	if s.HasHistory {
		onTime := decimal.Zero
		if h.PaidValue.IsPositive() {
			onTime = h.PaidOnTimeValue.Div(h.PaidValue)
		}
		add(CreditFactor{Code: "PONTUALIDADE", Label: "Valor pago em dia", MaxPoints: 300,
			Points: int(onTime.Mul(decimal.NewFromInt(300)).IntPart()),
			Detail: fmt.Sprintf("%s%% do valor de %d títulos pagos até o vencimento", Numero(onTime.Mul(decimal.NewFromInt(100)), 1), h.PaidTitles)})

		delay := h.WeightedDelayDays
		pts := decimal.NewFromInt(1).Sub(delay.Div(decimal.NewFromInt(30)))
		if pts.IsNegative() {
			pts = decimal.Zero
		}
		add(CreditFactor{Code: "ATRASO_MEDIO", Label: "Atraso médio ponderado", MaxPoints: 200,
			Points: int(pts.Mul(decimal.NewFromInt(200)).IntPart()),
			Detail: fmt.Sprintf("%s dias em média, ponderado pelo valor", Numero(delay, 1))})

		maior := 0
		switch {
		case h.MaxDelayDays <= 5:
			maior = 100
		case h.MaxDelayDays <= 15:
			maior = 60
		case h.MaxDelayDays <= 30:
			maior = 30
		}
		add(CreditFactor{Code: "MAIOR_ATRASO", Label: "Maior atraso", MaxPoints: 100, Points: maior,
			Detail: fmt.Sprintf("%d dias", h.MaxDelayDays)})

		meses := min(h.RelationshipMonths, 24)
		add(CreditFactor{Code: "RELACIONAMENTO", Label: "Tempo de relacionamento", MaxPoints: 100, Points: meses * 100 / 24,
			Detail: fmt.Sprintf("%d meses desde o primeiro título", h.RelationshipMonths)})

		reneg := 0
		switch h.Renegotiated {
		case 0:
			reneg = 100
		case 1:
			reneg = 50
		}
		add(CreditFactor{Code: "RENEGOCIACOES", Label: "Renegociações", MaxPoints: 100, Points: reneg,
			Detail: fmt.Sprintf("%d título(s) renegociado(s)", h.Renegotiated)})

		devol, devolPct := 50, decimal.Zero
		if h.Billed12m.IsPositive() {
			devolPct = h.Returns12m.Div(h.Billed12m).Mul(decimal.NewFromInt(100))
			switch {
			case devolPct.GreaterThan(decimal.NewFromInt(5)):
				devol = 0
			case devolPct.GreaterThan(decimal.NewFromInt(2)):
				devol = 25
			}
		}
		add(CreditFactor{Code: "DEVOLUCOES", Label: "Devoluções (12 meses)", MaxPoints: 50, Points: devol,
			Detail: fmt.Sprintf("%s%% do faturado", Numero(devolPct, 1))})

		if h.RecentPaidTitles > 0 && h.RecentWeightedDelayDays.Sub(h.WeightedDelayDays).GreaterThan(decimal.NewFromInt(10)) {
			s.Deteriorating = true
			add(CreditFactor{Code: "DETERIORACAO", Label: "Piora recente nos pagamentos", Points: -100,
				Detail: fmt.Sprintf("atraso médio dos últimos 90 dias (%s) passou o histórico (%s) em mais de 10 dias",
					Numero(h.RecentWeightedDelayDays, 1), Numero(h.WeightedDelayDays, 1))})
		}
		if h.OverdueBalance.IsPositive() && h.MaxOpenOverdueDays <= pol.OverdueBlockDays {
			add(CreditFactor{Code: "VENCIDO_EM_ABERTO", Label: "Vencido em aberto (dentro da tolerância)", Points: -50,
				Detail: fmt.Sprintf("%s vencidos há até %d dias", Reais(h.OverdueBalance), h.MaxOpenOverdueDays)})
		}
	} else {
		add(CreditFactor{Code: "SEM_HISTORICO", Label: "Cliente sem histórico de pagamento", Points: 350, MaxPoints: 350,
			Detail: fmt.Sprintf("menos de %d títulos pagos nesta empresa; vale a avaliação cadastral", minPaidTitlesForHistory)})
		capital := 0
		detail := "não informado pela Receita"
		if c.ShareCapital != nil {
			detail = Reais(*c.ShareCapital)
			switch {
			case c.ShareCapital.GreaterThanOrEqual(decimal.NewFromInt(1_000_000)):
				capital = 150
			case c.ShareCapital.GreaterThanOrEqual(decimal.NewFromInt(100_000)):
				capital = 100
			case c.ShareCapital.GreaterThanOrEqual(decimal.NewFromInt(20_000)):
				capital = 50
			}
		}
		add(CreditFactor{Code: "CAPITAL_SOCIAL", Label: "Capital social", MaxPoints: 150, Points: capital, Detail: detail})
	}

	idade, idadeDetalhe := 0, "data de abertura desconhecida"
	if c.OpeningDate != nil {
		anos := yearsBetween(*c.OpeningDate, today)
		idadeDetalhe = fmt.Sprintf("%d ano(s) de CNPJ", anos)
		switch {
		case anos >= 5:
			idade = 150
		case anos >= 3:
			idade = 100
		case anos >= 1:
			idade = 50
		}
	}
	add(CreditFactor{Code: "IDADE_EMPRESA", Label: "Idade da empresa", MaxPoints: 150, Points: idade, Detail: idadeDetalhe})

	// Fatos que zeram o crédito.
	if c.Known && c.RegistrationStatus != "" && !strings.EqualFold(strings.TrimSpace(c.RegistrationStatus), "ATIVA") {
		s.block(CreditFactor{Code: "RECEITA_IRREGULAR", Label: "Situação cadastral na Receita", Detail: c.RegistrationStatus})
	}
	if h.MaxOpenOverdueDays > pol.OverdueBlockDays {
		s.block(CreditFactor{Code: "VENCIDO_ALEM_TOLERANCIA", Label: "Título vencido além da tolerância",
			Detail: fmt.Sprintf("vencido há %d dias (tolerância %d); %s vencidos", h.MaxOpenOverdueDays, pol.OverdueBlockDays, Reais(h.OverdueBalance))})
	}
	if h.OpenProtest {
		s.block(CreditFactor{Code: "PROTESTO", Label: "Título em protesto", Detail: "há título em aberto enviado a protesto"})
	}
	if h.LostTitles > 0 {
		s.block(CreditFactor{Code: "PERDA", Label: "Título baixado como perda",
			Detail: fmt.Sprintf("%d título(s) baixado(s) como perda na janela analisada", h.LostTitles)})
	}

	s.Score = max(0, min(s.Score, 1000))
	s.RiskClass = classFor(s.Score)
	if !s.HasHistory && (s.RiskClass == RiskA || s.RiskClass == RiskB) {
		s.RiskClass = RiskC
	}
	if c.MEI && (s.RiskClass == RiskA || s.RiskClass == RiskB) {
		s.RiskClass = RiskC
	}
	if s.HardStop {
		s.RiskClass = RiskE
		s.Score = min(s.Score, 199)
	}

	s.SuggestedLimit = s.limit(h, c, pol, seller)
	s.ExpiresAt = today.AddDate(0, pol.ValidityMonths[s.RiskClass], 0)
	return s
}

func (s *CreditSuggestion) block(f CreditFactor) {
	f.Blocking = true
	s.HardStop = true
	s.Factors = append(s.Factors, f)
}

func classFor(score int) RiskClass {
	switch {
	case score >= 800:
		return RiskA
	case score >= 650:
		return RiskB
	case score >= 500:
		return RiskC
	case score >= 350:
		return RiskD
	default:
		return RiskE
	}
}

var classFactor = map[RiskClass]decimal.Decimal{
	RiskA: decimal.NewFromInt(1), RiskB: decimal.RequireFromString("0.8"), RiskC: decimal.RequireFromString("0.6"),
	RiskD: decimal.RequireFromString("0.3"), RiskE: decimal.Zero,
}

// limit: com histórico, a média mensal faturada a prazo vezes os meses de
// prazo que ele usa mais o mês corrente (o que fica "na rua" num ciclo); sem
// histórico, um percentual do capital social com teto. Depois o fator da
// classe, o teto da política e o arredondamento para baixo em R$ 100.
func (s *CreditSuggestion) limit(h CreditHistory, c CompanyProfile, pol CreditPolicy, seller SellerVolume) decimal.Decimal {
	var base decimal.Decimal
	detail := ""
	if s.HasHistory && h.Billed12m.IsPositive() {
		meses := max(1, min(h.RelationshipMonths, 12))
		mensal := h.Billed12m.Div(decimal.NewFromInt(int64(meses)))
		ciclos := h.AvgTermDays.Div(decimal.NewFromInt(30)).Add(decimal.NewFromInt(1))
		base = mensal.Mul(ciclos)
		detail = fmt.Sprintf("média mensal %s × (%s dias de prazo ÷ 30 + 1)", Reais(mensal), Numero(h.AvgTermDays, 0))
	} else {
		// Sem histórico: o menor entre o que o cliente aguenta (capital social)
		// e o que a empresa aceita arriscar num cliente ainda não testado.
		apetite, comoApetite := apetiteDaEmpresa(pol, seller)
		if c.ShareCapital != nil && c.ShareCapital.IsPositive() {
			capacidade := c.ShareCapital.Mul(pol.NewCustomerCapitalPct).Div(decimal.NewFromInt(100))
			base = decimal.Min(capacidade, apetite)
			detail = fmt.Sprintf("menor entre a capacidade do cliente (%s%% do capital social = %s) e o apetite da empresa (%s)",
				Numero(pol.NewCustomerCapitalPct, 1), Reais(capacidade), comoApetite)
		} else {
			base = pol.NewCustomerCap.Mul(decimal.RequireFromString("0.25"))
			detail = fmt.Sprintf("capital social desconhecido: 25%% do piso de cliente novo (%s)", Reais(pol.NewCustomerCap))
		}
	}
	if c.MEI {
		base = decimal.Min(base, pol.NewCustomerCap.Mul(decimal.RequireFromString("0.25")))
		detail += "; MEI limitado a 25% do piso de cliente novo"
	}
	limit := base.Mul(classFactor[s.RiskClass])
	detail += fmt.Sprintf("; fator da classe %s = %s", s.RiskClass, Numero(classFactor[s.RiskClass], 1))
	if pol.MaxLimit.IsPositive() && limit.GreaterThan(pol.MaxLimit) {
		limit = pol.MaxLimit
		detail += "; teto da empresa " + Reais(pol.MaxLimit)
	}
	arredondado := limit.Div(decimal.NewFromInt(100)).Floor().Mul(decimal.NewFromInt(100))
	if !arredondado.Equal(limit) {
		detail += fmt.Sprintf("; %s arredondado para baixo em múltiplos de R$ 100", Reais(limit))
	}
	limit = arredondado
	s.Factors = append(s.Factors, CreditFactor{Code: "BASE_DO_LIMITE", Label: "Como o valor foi calculado", Detail: detail})
	return limit
}

// apetiteDaEmpresa: quanto a empresa aceita expor num cliente novo — o % da
// política sobre o faturamento médio mensal dela, nunca abaixo do piso. Assim o
// limite acompanha o porte de quem vende, e não uma faixa fixa.
func apetiteDaEmpresa(pol CreditPolicy, seller SellerVolume) (decimal.Decimal, string) {
	mensal := seller.MonthlyAverage()
	apetite := mensal.Mul(pol.NewCustomerRevenuePct).Div(decimal.NewFromInt(100))
	if mensal.IsPositive() && apetite.GreaterThanOrEqual(pol.NewCustomerCap) {
		return apetite, fmt.Sprintf("%s%% do faturamento médio mensal de %s em %d mês(es) = %s",
			Numero(pol.NewCustomerRevenuePct, 1), Reais(mensal), min(seller.Months, 12), Reais(apetite))
	}
	if mensal.IsPositive() {
		return pol.NewCustomerCap, fmt.Sprintf("piso de cliente novo %s; %s%% do faturamento médio mensal de %s daria %s",
			Reais(pol.NewCustomerCap), Numero(pol.NewCustomerRevenuePct, 1), Reais(mensal), Reais(apetite))
	}
	return pol.NewCustomerCap, fmt.Sprintf("piso de cliente novo %s: a empresa ainda não tem faturamento no sistema", Reais(pol.NewCustomerCap))
}

func yearsBetween(from, to time.Time) int {
	y := to.Year() - from.Year()
	if to.YearDay() < from.YearDay() {
		y--
	}
	return max(y, 0)
}

// CheckCreditApproval confere o limite que o usuário quer gravar contra a
// sugestão: qualquer diferença exige motivo; aumentar além da tolerância da
// política, ou dar crédito a quem caiu num bloqueio, só o ADMIN.
func CheckCreditApproval(s CreditSuggestion, approved decimal.Decimal, reason string, isAdmin bool, pol CreditPolicy) (deviates bool, err error) {
	if approved.IsNegative() {
		return false, domainerr.NewValidation("o limite de crédito não pode ser negativo")
	}
	if approved.Exponent() < -2 {
		return false, domainerr.NewValidation("o limite de crédito tem no máximo 2 casas decimais")
	}
	deviates = !approved.Equal(s.SuggestedLimit)
	if !deviates {
		return false, nil
	}
	if len([]rune(strings.TrimSpace(reason))) < 15 {
		return true, domainerr.NewValidation(fmt.Sprintf(
			"o limite informado (%s) é diferente do sugerido (%s): explique o motivo em pelo menos 15 caracteres",
			Reais(approved), Reais(s.SuggestedLimit)))
	}
	if isAdmin {
		return true, nil
	}
	if s.HardStop && approved.IsPositive() {
		return true, domainerr.NewValidation("o cliente caiu num bloqueio de crédito; só um administrador pode liberar limite")
	}
	teto := toleranciaDoSugerido(s, pol)
	if pol.MaxLimit.IsPositive() && approved.GreaterThan(pol.MaxLimit) {
		return true, domainerr.NewValidation(fmt.Sprintf(
			"acima do teto de crédito da empresa (%s) só um administrador pode aprovar", Reais(pol.MaxLimit)))
	}
	if approved.GreaterThan(teto) {
		return true, domainerr.NewValidation(fmt.Sprintf(
			"acima de %s (sugerido + %s%% de tolerância) só um administrador pode aprovar",
			Reais(teto), Numero(pol.OverrideTolerancePct, 1)))
	}
	return true, nil
}

func toleranciaDoSugerido(s CreditSuggestion, pol CreditPolicy) decimal.Decimal {
	return s.SuggestedLimit.Mul(decimal.NewFromInt(100).Add(pol.OverrideTolerancePct)).Div(decimal.NewFromInt(100))
}

// MaxWithoutAdmin é o maior limite que quem não é administrador aprova:
// sugerido + tolerância, nunca acima do teto da empresa; zero num bloqueio.
func MaxWithoutAdmin(s CreditSuggestion, pol CreditPolicy) decimal.Decimal {
	if s.HardStop {
		return decimal.Zero
	}
	teto := toleranciaDoSugerido(s, pol)
	if pol.MaxLimit.IsPositive() && teto.GreaterThan(pol.MaxLimit) {
		teto = pol.MaxLimit
	}
	return teto
}

// CreditGate é o que a análise de crédito do cadastro decide antes de olhar o
// limite: limite vencido, classe D/E, limite zerado numa análise e título
// vencido além da tolerância no grupo econômico mandam o pedido para análise.
// Cliente que nunca passou pela análise segue só a regra do limite.
func CreditGate(c *Customer, group *CreditHistory, pol CreditPolicy, today time.Time) (bool, string) {
	if c.CreditLimitExpiresAt != nil && c.CreditLimitExpiresAt.Before(today) {
		return false, fmt.Sprintf("limite de crédito vencido em %s: revise o limite na VCLI0500", c.CreditLimitExpiresAt.Format("02/01/2006"))
	}
	if c.CreditRiskClass != nil && (*c.CreditRiskClass == string(RiskD) || *c.CreditRiskClass == string(RiskE)) {
		return false, fmt.Sprintf("cliente na classe de risco %s: todo pedido passa pela análise de crédito", *c.CreditRiskClass)
	}
	if c.CreditReviewedAt != nil && !c.CreditLimit.IsPositive() {
		return false, "cliente sem limite de crédito aprovado na última análise"
	}
	if group != nil && group.MaxOpenOverdueDays > pol.OverdueBlockDays {
		return false, fmt.Sprintf("título vencido há %d dias no cliente ou no grupo econômico (tolerância %d dias)", group.MaxOpenOverdueDays, pol.OverdueBlockDays)
	}
	return true, ""
}

// CondicaoAVista: sem parcela a prazo — nenhuma parcela, ou todas no dia,
// contadas da emissão ou da entrada. Pagamento "na entrega" não é à vista:
// a mercadoria sai antes do dinheiro.
func CondicaoAVista(parcelas []*PaymentInstallment) bool {
	for _, p := range parcelas {
		if p == nil || !p.IsActive {
			continue
		}
		if p.DueDays != 0 || (p.BaseEvent != "" && p.BaseEvent != "EMISSAO" && p.BaseEvent != "ENTRADA") {
			return false
		}
	}
	return true
}

// DecisaoDaCondicao: o que o tipo de análise da condição de pagamento do
// pedido manda fazer com o crédito. BLOQUEIA_SEMPRE manda todo pedido para
// liberação. LIBERA_SEM_ANALISE só dispensa a análise quando a condição é de
// fato à vista — LIBERA_SEM_ANALISE é o padrão da coluna e está em condições a
// prazo que ninguém escolheu liberar; o cliente bloqueado continua barrado.
// O resto segue a análise normal.
func DecisaoDaCondicao(a PaymentAnalysis, aVista, clienteBloqueado bool) (analisar bool, bloquear bool, motivo string) {
	switch {
	case a == AnalysisBloqueiaSempre, a == "BLOCK_ALWAYS": // o valor em inglês só existe em base antiga
		return false, true, "a condição de pagamento do pedido exige liberação de crédito"
	case a == AnalysisLiberaSemAnalise && aVista:
		if clienteBloqueado {
			return false, true, "cliente bloqueado para crédito"
		}
		return false, false, ""
	default:
		return true, false, ""
	}
}
