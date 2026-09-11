package entity

import "time"

// VigenteEm diz se o componente vale na data informada. É o critério único de
// efetividade da estrutura: MRP, ordem de produção e plano de corte usam este
// método, em vez de cada um comparar datas do seu jeito.
//
// A comparação é por dia (a vigência é cadastrada como data, sem hora). Início
// e fim nulos significam "sem restrição" naquela ponta.
//
// Antes deste método nenhum dos três consumidores olhava a vigência: um
// componente vencido continuava gerando necessidade no MRP e entrando na lista
// de materiais da ordem, e um componente que ainda não entrou em vigor era
// consumido antes da hora.
func (s *ItemStructure) VigenteEm(data time.Time) bool {
	dia := func(t time.Time) time.Time {
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}
	hoje := dia(data)
	if s.StartDate != nil && hoje.Before(dia(*s.StartDate)) {
		return false
	}
	if s.EndDate != nil && hoje.After(dia(*s.EndDate)) {
		return false
	}
	return true
}
