package mrp_report_uc

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func dia(d int) time.Time       { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }
func q(n int64) decimal.Decimal { return decimal.NewFromInt(n) }

// O farol separa o que já está perdido (falta vencida ou ordem planejada que
// devia ter começado), o que depende de firmar ordem planejada e o que está
// coberto — e a lista vem do mais urgente para o mais tranquilo.
func TestCoberturaFarolEDiasDeCobertura(t *testing.T) {
	inicio := dia(10)
	reader := &reportReader{coverage: []CoverageInput{
		// Coberto: 100 em estoque para 60 de necessidade.
		{ItemCode: 1, Stock: q(100), Buckets: []CoverageBucket{{Date: dia(20), Demand: q(60)}}},
		// Estoque acaba dia 17; a ordem firme cobre até dia 24; dali em diante
		// só a planejada (que começa no futuro) cobre.
		{ItemCode: 2, Stock: q(10), FirstPlannedStart: ptr(dia(20)), PlannedOrders: 1, Buckets: []CoverageBucket{
			{Date: dia(17), Demand: q(15), Firm: q(10)},
			{Date: dia(24), Demand: q(20), Planned: q(25)},
		}},
		// Falta firme hoje: crítico mesmo com planejada.
		{ItemCode: 3, Stock: q(0), Buckets: []CoverageBucket{{Date: dia(12), Demand: q(5), Planned: q(5)}}},
		// Planejada devia ter começado dia 10 para cobrir a falta do dia 20.
		{ItemCode: 4, Stock: q(0), FirstPlannedStart: &inicio, Buckets: []CoverageBucket{{Date: dia(20), Demand: q(5), Planned: q(5)}}},
		// Sem falta, mas o projetado fura o estoque de segurança.
		{ItemCode: 5, Stock: q(30), SafetyStock: q(20), Buckets: []CoverageBucket{{Date: dia(25), Demand: q(15)}}},
	}}
	uc := UseCase{Reader: reader, Auth: reportAuth{allowed: true}}
	plano := int64(1)
	rows, err := uc.Coverage(context.Background(), Filter{PlanCode: &plano}, time.Date(2026, 10, 12, 15, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatal(err)
	}
	ordem := []int64{}
	por := map[int64]CoverageRow{}
	for _, r := range rows {
		ordem = append(ordem, r.ItemCode)
		por[r.ItemCode] = r
	}
	if got := []string{por[1].Status, por[2].Status, por[3].Status, por[4].Status, por[5].Status}; got[0] != CoverageOK || got[1] != CoverageWarning || got[2] != CoverageCritical || got[3] != CoverageCritical || got[4] != CoverageWarning {
		t.Fatalf("farol = %v", got)
	}
	if ordem[0] != 3 || ordem[1] != 4 || ordem[len(ordem)-1] != 1 {
		t.Fatalf("ordem = %v, esperado críticos primeiro (3, 4) e o coberto por último", ordem)
	}
	i2 := por[2]
	if i2.DaysOfSupply == nil || *i2.DaysOfSupply != 5 || i2.StockShortageDate == nil || !i2.StockShortageDate.Equal(dia(17)) {
		t.Fatalf("dias de cobertura do estoque: %+v", i2)
	}
	if i2.ReceiptDaysOfSupply == nil || *i2.ReceiptDaysOfSupply != 12 || !i2.FirmShortageQty.Equal(q(15)) {
		t.Fatalf("cobertura com firmes: %+v", i2)
	}
	if !i2.MinProjected.Equal(q(5)) || !i2.Demand.Equal(q(35)) {
		t.Fatalf("totais: %+v", i2)
	}
	if por[1].DaysOfSupply != nil || por[1].FirmShortageDate != nil {
		t.Fatalf("item coberto com data de falta: %+v", por[1])
	}

	so, err := uc.Coverage(context.Background(), Filter{PlanCode: &plano, OnlyWithMessage: true}, dia(12))
	if err != nil || len(so) != 4 {
		t.Fatalf("filtro só com problema: %d linhas (%v)", len(so), err)
	}
	if _, err := uc.Coverage(context.Background(), Filter{}, dia(12)); err == nil {
		t.Fatal("aceitou cobertura sem plano")
	}
}

func ptr(t time.Time) *time.Time { return &t }
