package entity

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// Revisão automática do limite (a cada AutoReviewMonths da política).
//
// O sistema refaz a análise com o comportamento do período — pontualidade,
// atraso, renegociação, devoluções, vencidos e o quanto o cliente comprou — e:
//   - reduz na hora quando a análise pede menos (ou zera num bloqueio) — mas
//     só com histórico de pagamento: sem ele a análise mede a falta de dado,
//     não o comportamento, e o limite fica para uma pessoa revisar (Skip);
//   - mantém a exceção aprovada por um administrador (limite acima do sugerido)
//     enquanto o cliente não piora: classe pior ou piora recente reduzem;
//   - aumenta só para quem tem histórico de pagamento, no máximo
//     AutoIncreaseMaxPct do limite atual por revisão e dentro do teto da
//     empresa. Limite zerado não sobe sozinho: alguém precisa revisar.
// Toda revisão renova a validade, então o pedido não trava por limite vencido.

// AutoReviewDecision é o que a revisão automática grava.
type AutoReviewDecision struct {
	NewLimit decimal.Decimal
	Reason   string
	Changed  bool
	// Skip: nada é gravado — o limite segue com a validade que tem e vence para
	// uma pessoa revisar.
	Skip bool
}

// LastCreditReview é o que a revisão precisa saber da análise anterior.
type LastCreditReview struct {
	RiskClass RiskClass
	// Exception: o limite aprovado ficou acima do sugerido (exceção aprovada).
	Exception bool
}

var ordemClasse = map[RiskClass]int{RiskA: 1, RiskB: 2, RiskC: 3, RiskD: 4, RiskE: 5}

// DecideAutoReview decide o novo limite a partir da análise feita agora.
func DecideAutoReview(s CreditSuggestion, current decimal.Decimal, last *LastCreditReview, pol CreditPolicy) AutoReviewDecision {
	sugerido := s.SuggestedLimit
	piorou := last != nil && last.RiskClass != "" && ordemClasse[s.RiskClass] > ordemClasse[last.RiskClass]
	quem := func() string {
		if last != nil && last.RiskClass != "" && last.RiskClass != s.RiskClass {
			return fmt.Sprintf("classe %s → %s, score %d", last.RiskClass, s.RiskClass, s.Score)
		}
		return fmt.Sprintf("classe %s, score %d", s.RiskClass, s.Score)
	}()
	negativos := fatoresNegativos(s)

	switch {
	case s.HardStop:
		return AutoReviewDecision{NewLimit: decimal.Zero, Changed: current.IsPositive(),
			Reason: fmt.Sprintf("Revisão automática: bloqueio de crédito (%s). Limite zerado.", bloqueios(s))}

	case sugerido.LessThan(current):
		if last != nil && last.Exception && !piorou && !s.Deteriorating {
			return AutoReviewDecision{NewLimit: current,
				Reason: fmt.Sprintf("Revisão automática: mantido o limite de %s aprovado por exceção (sugerido %s); o cliente não piorou (%s).",
					Reais(current), Reais(sugerido), quem)}
		}
		if !s.HasHistory {
			return AutoReviewDecision{NewLimit: current, Skip: true,
				Reason: fmt.Sprintf("Revisão automática: sem histórico de pagamento para reavaliar %s (sugerido %s); fica para revisão manual.",
					Reais(current), Reais(sugerido))}
		}
		motivo := "a análise do período pede menos"
		if negativos != "" {
			motivo = negativos
		}
		return AutoReviewDecision{NewLimit: sugerido, Changed: true,
			Reason: fmt.Sprintf("Revisão automática: reduzido de %s para %s — %s (%s).", Reais(current), Reais(sugerido), motivo, quem)}

	case sugerido.GreaterThan(current):
		switch {
		case !current.IsPositive():
			return AutoReviewDecision{NewLimit: current,
				Reason: fmt.Sprintf("Revisão automática: sugerido %s, mas limite zerado não sobe sozinho — revise na VCLI0500 (%s).", Reais(sugerido), quem)}
		case !s.HasHistory:
			return AutoReviewDecision{NewLimit: current,
				Reason: fmt.Sprintf("Revisão automática: mantido %s; o aumento automático exige histórico de pagamento (sugerido %s, %s).", Reais(current), Reais(sugerido), quem)}
		case !pol.AutoIncreaseMaxPct.IsPositive():
			return AutoReviewDecision{NewLimit: current,
				Reason: fmt.Sprintf("Revisão automática: mantido %s; a política não permite aumento automático (sugerido %s, %s).", Reais(current), Reais(sugerido), quem)}
		}
		teto := current.Mul(decimal.NewFromInt(100).Add(pol.AutoIncreaseMaxPct)).Div(decimal.NewFromInt(100))
		teto = teto.Div(decimal.NewFromInt(100)).Floor().Mul(decimal.NewFromInt(100))
		novo := decimal.Min(sugerido, teto)
		if !novo.GreaterThan(current) {
			return AutoReviewDecision{NewLimit: current,
				Reason: fmt.Sprintf("Revisão automática: mantido %s (sugerido %s, %s).", Reais(current), Reais(sugerido), quem)}
		}
		limitado := ""
		if novo.LessThan(sugerido) {
			limitado = fmt.Sprintf("; limitado a +%s%% por revisão (sugerido %s)", Numero(pol.AutoIncreaseMaxPct, 0), Reais(sugerido))
		}
		return AutoReviewDecision{NewLimit: novo, Changed: true,
			Reason: fmt.Sprintf("Revisão automática: aumentado de %s para %s pelo histórico de pagamento%s (%s).", Reais(current), Reais(novo), limitado, quem)}
	}
	return AutoReviewDecision{NewLimit: current,
		Reason: fmt.Sprintf("Revisão automática: mantido %s, igual ao sugerido (%s).", Reais(current), quem)}
}

// fatoresNegativos resume o que puxou a análise para baixo: bloqueios, piora
// recente, vencidos e fatores com menos da metade dos pontos.
func fatoresNegativos(s CreditSuggestion) string {
	var out []string
	for _, f := range s.Factors {
		switch {
		case f.Blocking, f.Points < 0:
			out = append(out, strings.ToLower(f.Label)+": "+f.Detail)
		case f.MaxPoints > 0 && f.Points*2 < f.MaxPoints && f.Code != "SEM_HISTORICO":
			out = append(out, strings.ToLower(f.Label)+" "+f.Detail)
		}
	}
	return strings.Join(out, "; ")
}

// bloqueios: só o que zerou o limite — os fatores fracos esconderiam a causa.
func bloqueios(s CreditSuggestion) string {
	var out []string
	for _, f := range s.Factors {
		if f.Blocking {
			out = append(out, strings.ToLower(f.Label)+": "+f.Detail)
		}
	}
	return strings.Join(out, "; ")
}
