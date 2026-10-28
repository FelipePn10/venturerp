package reforma

import "time"

// PrimeiroAnoDeRecolhimento é quando IBS e CBS destacados viram dívida da
// empresa. Em 2026, o ano de teste da LC 214/2025, o destaque na NF-e é
// obrigatório mas quem cumpre as obrigações acessórias fica dispensado do
// recolhimento (e o valor seria compensável com PIS/COFINS): lançar a pagar
// ou apurar saldo devedor em 2026 criaria um imposto que não existe.
const PrimeiroAnoDeRecolhimento = 2027

// Recolhivel diz se IBS e CBS de um documento com essa data entram na apuração
// e na contabilização como tributo a recolher (ou crédito a recuperar).
func Recolhivel(data time.Time) bool {
	return data.Year() >= PrimeiroAnoDeRecolhimento
}

// RecolhivelNaCompetencia é Recolhivel para a competência "MM/AAAA" da
// apuração. Competência mal formada não é recolhível.
func RecolhivelNaCompetencia(competencia string) bool {
	t, err := time.Parse("01/2006", competencia)
	return err == nil && Recolhivel(t)
}
