package service

import (
	"testing"
	"time"

	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
)

// TestExplosaoRespeitaVigencia trava o critério de efetividade do MRP: a
// explosão usa a data em que o componente será consumido. Antes, nenhuma data
// era olhada — componente vencido seguia gerando necessidade.
func TestExplosaoRespeitaVigencia(t *testing.T) {
	d := func(s string) *time.Time { v, _ := time.Parse("2006-01-02", s); return &v }
	necessidade, _ := time.Parse("2006-01-02", "2026-09-15")

	bom := map[int64][]*structentity.ItemStructure{1: {
		{ParentCode: 1, ChildCode: 10, Quantity: 1},                             // sem datas: entra
		{ParentCode: 1, ChildCode: 20, Quantity: 1, EndDate: d("2026-08-12")},   // vencido: sai
		{ParentCode: 1, ChildCode: 30, Quantity: 1, StartDate: d("2026-10-01")}, // futuro: sai
		// Alternativos: o principal (prioridade 1) venceu, então a necessidade
		// tem de ir para o de prioridade 2 — e não desaparecer.
		{ParentCode: 1, ChildCode: 40, Quantity: 1, SubstituteGroup: 1, SubstitutePriority: 1, EndDate: d("2026-09-01")},
		{ParentCode: 1, ChildCode: 41, Quantity: 1, SubstituteGroup: 1, SubstitutePriority: 2},
	}}

	got := map[int64]bool{}
	for _, in := range explodeFromBOMWithVars(bom, 1, "", 1, 1, 1, nil, necessidade) {
		got[in.ItemCode] = true
	}
	for codigo, esperado := range map[int64]bool{10: true, 20: false, 30: false, 40: false, 41: true} {
		if got[codigo] != esperado {
			t.Errorf("componente %d: esperado entrar=%v, obtido %v", codigo, esperado, got[codigo])
		}
	}

	// Sem data de necessidade, nada é filtrado (compatibilidade das chamadas
	// que não conhecem a data).
	if n := len(explodeFromBOMWithVars(bom, 1, "", 1, 1, 1, nil, time.Time{})); n != 4 {
		t.Errorf("sem data: esperado 4 componentes (3 simples + 1 alternativo), obtido %d", n)
	}
}
