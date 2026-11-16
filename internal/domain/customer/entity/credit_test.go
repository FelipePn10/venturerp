package entity

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

var hoje = time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)

func dc(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func anosAtras(n int) *time.Time { t := hoje.AddDate(-n, 0, 0); return &t }

func bomPagador() CreditHistory {
	return CreditHistory{
		PaidTitles: 40, PaidValue: dc("240000"), PaidOnTimeValue: dc("240000"),
		MaxDelayDays: 2, AvgTermDays: dc("30"), Billed12m: dc("120000"), RelationshipMonths: 30,
	}
}

func TestBomPagadorClasseAComLimiteDoCiclo(t *testing.T) {
	s := SuggestCreditLimit(bomPagador(), CompanyProfile{Known: true, RegistrationStatus: "ATIVA", OpeningDate: anosAtras(10)}, DefaultCreditPolicy(), hoje)
	if s.Score != 1000 || s.RiskClass != RiskA || s.HardStop {
		t.Fatalf("score %d classe %s hardstop %v", s.Score, s.RiskClass, s.HardStop)
	}
	// 120.000/12 = 10.000 por mês × (30/30 + 1) × 1,0
	if !s.SuggestedLimit.Equal(dc("20000")) {
		t.Fatalf("limite = %s, quer 20000", s.SuggestedLimit)
	}
	if !s.ExpiresAt.Equal(hoje.AddDate(1, 0, 0)) {
		t.Fatalf("validade = %s", s.ExpiresAt)
	}
}

func TestClienteNovoNuncaPassaDaClasseC(t *testing.T) {
	capital := dc("5000000")
	s := SuggestCreditLimit(CreditHistory{}, CompanyProfile{Known: true, RegistrationStatus: "ATIVA", OpeningDate: anosAtras(20), ShareCapital: &capital}, DefaultCreditPolicy(), hoje)
	if s.HasHistory || s.RiskClass != RiskC || s.Score != 650 {
		t.Fatalf("score %d classe %s", s.Score, s.RiskClass)
	}
	// min(10% de 5 mi, 30.000) × 0,6
	if !s.SuggestedLimit.Equal(dc("18000")) {
		t.Fatalf("limite = %s, quer 18000", s.SuggestedLimit)
	}
}

func TestClienteNovoSemDadosDaReceitaFicaNaClasseD(t *testing.T) {
	s := SuggestCreditLimit(CreditHistory{}, CompanyProfile{}, DefaultCreditPolicy(), hoje)
	if s.RiskClass != RiskD {
		t.Fatalf("classe %s", s.RiskClass)
	}
	// 25% de 30.000 × 0,3 = 2.250 → arredonda para baixo em 100
	if !s.SuggestedLimit.Equal(dc("2200")) {
		t.Fatalf("limite = %s", s.SuggestedLimit)
	}
	conta := s.Factors[len(s.Factors)-1]
	if conta.Code != "BASE_DO_LIMITE" || !strings.Contains(conta.Detail, "R$ 2.250,00 arredondado para baixo") {
		t.Fatalf("a conta do valor não explica o arredondamento: %+v", conta)
	}
}

func TestBloqueiosZeramOLimite(t *testing.T) {
	casos := map[string]func(*CreditHistory, *CompanyProfile){
		"receita baixada": func(_ *CreditHistory, c *CompanyProfile) { c.RegistrationStatus = "BAIXADA" },
		"vencido além da tolerância": func(h *CreditHistory, _ *CompanyProfile) {
			h.MaxOpenOverdueDays, h.OverdueBalance = 31, dc("500")
		},
		"protesto": func(h *CreditHistory, _ *CompanyProfile) { h.OpenProtest = true },
		"perda":    func(h *CreditHistory, _ *CompanyProfile) { h.LostTitles = 1 },
	}
	for nome, muda := range casos {
		h, c := bomPagador(), CompanyProfile{Known: true, RegistrationStatus: "ATIVA", OpeningDate: anosAtras(10)}
		muda(&h, &c)
		s := SuggestCreditLimit(h, c, DefaultCreditPolicy(), hoje)
		if !s.HardStop || s.RiskClass != RiskE || !s.SuggestedLimit.IsZero() || s.Score > 199 {
			t.Errorf("%s: hardstop %v classe %s limite %s score %d", nome, s.HardStop, s.RiskClass, s.SuggestedLimit, s.Score)
		}
	}
}

func TestPioraRecenteTiraCemPontos(t *testing.T) {
	h := bomPagador()
	h.WeightedDelayDays, h.RecentPaidTitles, h.RecentWeightedDelayDays = dc("1"), 4, dc("15")
	s := SuggestCreditLimit(h, CompanyProfile{OpeningDate: anosAtras(10)}, DefaultCreditPolicy(), hoje)
	// 300 + 193 (1 dia de atraso médio) + 100 + 100 + 100 + 50 + 150 − 100
	if !s.Deteriorating || s.Score != 893 {
		t.Fatalf("deterioração %v score %d", s.Deteriorating, s.Score)
	}
}

func TestTetoDaEmpresaEMEI(t *testing.T) {
	pol := DefaultCreditPolicy()
	pol.MaxLimit = dc("5000")
	s := SuggestCreditLimit(bomPagador(), CompanyProfile{OpeningDate: anosAtras(10)}, pol, hoje)
	if !s.SuggestedLimit.Equal(dc("5000")) {
		t.Fatalf("teto: %s", s.SuggestedLimit)
	}
	s = SuggestCreditLimit(bomPagador(), CompanyProfile{OpeningDate: anosAtras(10), MEI: true}, DefaultCreditPolicy(), hoje)
	if s.RiskClass != RiskC || !s.SuggestedLimit.Equal(dc("4500")) { // 7.500 × 0,6
		t.Fatalf("MEI: classe %s limite %s", s.RiskClass, s.SuggestedLimit)
	}
}

func TestAprovacaoDoLimite(t *testing.T) {
	pol := DefaultCreditPolicy()
	s := CreditSuggestion{SuggestedLimit: dc("10000")}
	if dev, err := CheckCreditApproval(s, dc("10000"), "", false, pol); err != nil || dev {
		t.Fatalf("igual à sugestão: dev %v err %v", dev, err)
	}
	if _, err := CheckCreditApproval(s, dc("9000"), "curto", false, pol); err == nil {
		t.Fatal("desvio sem motivo suficiente devia falhar")
	}
	if dev, err := CheckCreditApproval(s, dc("11000"), "cliente apresentou balanço 2025", false, pol); err != nil || !dev {
		t.Fatalf("dentro da tolerância: dev %v err %v", dev, err)
	}
	if _, err := CheckCreditApproval(s, dc("11000.01"), "cliente apresentou balanço 2025", false, pol); err == nil {
		t.Fatal("acima da tolerância sem ADMIN devia falhar")
	}
	if _, err := CheckCreditApproval(s, dc("50000"), "cliente apresentou balanço 2025", true, pol); err != nil {
		t.Fatalf("ADMIN: %v", err)
	}
	bloqueado := CreditSuggestion{HardStop: true}
	if _, err := CheckCreditApproval(bloqueado, dc("100"), "acordo de pagamento assinado", false, pol); err == nil {
		t.Fatal("bloqueio só o ADMIN libera")
	}
	if _, err := CheckCreditApproval(s, dc("-1"), "x", true, pol); err == nil {
		t.Fatal("negativo")
	}
	// A tolerância não passa do teto da empresa (só o ADMIN passa).
	comTeto := pol
	comTeto.MaxLimit = dc("10000")
	if _, err := CheckCreditApproval(s, dc("10500"), "cliente apresentou balanço 2025", false, comTeto); err == nil {
		t.Fatal("USER aprovou acima do teto da empresa")
	}
	if _, err := CheckCreditApproval(s, dc("10500"), "cliente apresentou balanço 2025", true, comTeto); err != nil {
		t.Fatalf("ADMIN acima do teto: %v", err)
	}
	if _, err := CheckCreditApproval(s, dc("9500"), "cliente apresentou balanço 2025", false, comTeto); err != nil {
		t.Fatalf("abaixo do teto: %v", err)
	}
	// O que a tela mostra como "sem administrador: até" é o que a aprovação aceita.
	if m := MaxWithoutAdmin(s, comTeto); !m.Equal(dc("10000")) {
		t.Fatalf("máximo sem ADMIN com teto: %s", m)
	}
	if m := MaxWithoutAdmin(s, pol); !m.Equal(dc("11000")) {
		t.Fatalf("máximo sem ADMIN sem teto: %s", m)
	}
	for _, v := range []string{"10000", "10000.01"} {
		_, err := CheckCreditApproval(s, dc(v), "cliente apresentou balanço 2025", false, comTeto)
		if (err == nil) != !dc(v).GreaterThan(MaxWithoutAdmin(s, comTeto)) {
			t.Fatalf("aprovação de %s diverge do máximo mostrado: %v", v, err)
		}
	}
	if m := MaxWithoutAdmin(CreditSuggestion{HardStop: true, SuggestedLimit: dc("500")}, pol); !m.IsZero() {
		t.Fatalf("bloqueio: %s", m)
	}
}

func TestCreditGate(t *testing.T) {
	pol := DefaultCreditPolicy()
	ontem, amanha := hoje.AddDate(0, 0, -1), hoje.AddDate(0, 0, 1)
	d, c := "D", "C"
	reviewed := hoje
	casos := []struct {
		nome  string
		cli   Customer
		grupo *CreditHistory
		ok    bool
	}{
		{"nunca analisado", Customer{}, nil, true},
		{"limite vencido", Customer{CreditLimit: dc("1000"), CreditLimitExpiresAt: &ontem}, nil, false},
		{"limite vigente", Customer{CreditLimit: dc("1000"), CreditLimitExpiresAt: &amanha, CreditRiskClass: &c, CreditReviewedAt: &reviewed}, nil, true},
		{"classe D", Customer{CreditLimit: dc("1000"), CreditRiskClass: &d}, nil, false},
		{"analisado com limite zero", Customer{CreditReviewedAt: &reviewed}, nil, false},
		{"grupo com vencido", Customer{CreditLimit: dc("1000")}, &CreditHistory{MaxOpenOverdueDays: 31}, false},
		{"grupo dentro da tolerância", Customer{CreditLimit: dc("1000")}, &CreditHistory{MaxOpenOverdueDays: 30}, true},
	}
	for _, tc := range casos {
		if ok, motivo := CreditGate(&tc.cli, tc.grupo, pol, hoje); ok != tc.ok {
			t.Errorf("%s: ok = %v (%s)", tc.nome, ok, motivo)
		}
	}
}

func TestDecisaoDaCondicao(t *testing.T) {
	if analisar, bloquear, _ := DecisaoDaCondicao(AnalysisLiberaSemAnalise, true, false); analisar || bloquear {
		t.Fatal("à vista com LIBERA dispensa a análise")
	}
	if analisar, _, _ := DecisaoDaCondicao(AnalysisLiberaSemAnalise, false, false); !analisar {
		t.Fatal("LIBERA numa condição a prazo (padrão da coluna) não pode pular o crédito")
	}
	if _, bloquear, _ := DecisaoDaCondicao(AnalysisLiberaSemAnalise, true, true); !bloquear {
		t.Fatal("cliente bloqueado continua barrado mesmo à vista")
	}
	if analisar, bloquear, motivo := DecisaoDaCondicao(AnalysisBloqueiaSempre, true, false); analisar || !bloquear || motivo == "" {
		t.Fatal("BLOQUEIA_SEMPRE manda para liberação")
	}
	if analisar, bloquear, _ := DecisaoDaCondicao(AnalysisSempreAnalisa, true, false); !analisar || bloquear {
		t.Fatal("SEMPRE_ANALISA segue a análise")
	}
}

func TestCondicaoAVista(t *testing.T) {
	p := func(dias int16, evento string) *PaymentInstallment {
		return &PaymentInstallment{DueDays: dias, BaseEvent: evento, IsActive: true}
	}
	casos := []struct {
		nome     string
		parcelas []*PaymentInstallment
		aVista   bool
	}{
		{"sem parcelas", nil, true},
		{"no dia da emissão", []*PaymentInstallment{p(0, "EMISSAO")}, true},
		{"entrada", []*PaymentInstallment{p(0, "ENTRADA")}, true},
		{"28 dias", []*PaymentInstallment{p(28, "EMISSAO")}, false},
		{"entrada + 28", []*PaymentInstallment{p(0, "ENTRADA"), p(28, "EMISSAO")}, false},
		{"na entrega", []*PaymentInstallment{p(0, "ENTREGA")}, false},
		{"parcela inativa a prazo", []*PaymentInstallment{p(0, "EMISSAO"), {DueDays: 30, IsActive: false}}, true},
	}
	for _, c := range casos {
		if got := CondicaoAVista(c.parcelas); got != c.aVista {
			t.Errorf("%s: à vista = %v", c.nome, got)
		}
	}
}

func TestTipoDeAnaliseDaCondicao(t *testing.T) {
	if _, err := NewPaymentCondition(1, "28 dias", "BLOCK_ALWAYS"); err == nil {
		t.Fatal("aceitou tipo de análise da divisão de vendas")
	}
	if pc, err := NewPaymentCondition(1, "28 dias", ""); err != nil || pc.AnalysisType != AnalysisLiberaSemAnalise {
		t.Fatalf("padrão: %v %v", pc, err)
	}
	if _, bloquear, _ := DecisaoDaCondicao("BLOCK_ALWAYS", false, false); !bloquear {
		t.Fatal("BLOCK_ALWAYS gravado em base antiga deixou de exigir liberação")
	}
}
