//go:build integration

package planning_run_test

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/planning_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planning_run"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A janela noturna vence pela hora da EMPRESA (Brasília), não pela do processo
// nem pela da sessão do banco; a parametrização e o histórico são da empresa.
func TestJanelaNoturnaUsaOFusoDaEmpresaEIsolaHistorico(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM planning_run_history WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM planning_auto_run_settings WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM production_plans WHERE enterprise_id = ANY($1)`, ids)
	})
	plano := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO production_plans(code,name,created_by,enterprise_id) VALUES($1,'Noturno',$2,$3)`, plano, actor, a.ID)
	repo := planning_run.New(pool)
	ator := actor

	if ok, err := repo.PlanExists(base, a.ID, plano); err != nil || !ok {
		t.Fatalf("plano da empresa A não encontrado: %v %v", ok, err)
	}
	if ok, err := repo.PlanExists(base, b.ID, plano); err != nil || ok {
		t.Fatalf("plano da empresa A visível para a B: %v %v", ok, err)
	}

	if err := repo.SaveSettings(base, planning_uc.AutoRunSettings{
		EnterpriseID: a.ID, IsEnabled: true, RunHour: 2, RunMinute: 0, PlanCode: plano,
		InitialOrderNumber: 1, GenerateLLC: true, UpdatedBy: &ator,
	}); err != nil {
		t.Fatal(err)
	}
	s, err := repo.Settings(base, a.ID)
	if err != nil || !s.IsEnabled || s.PlanCode != plano || s.EnterpriseCode != a.Code || s.UpdatedBy == nil || *s.UpdatedBy != ator {
		t.Fatalf("parametrização lida errada: %+v (%v)", s, err)
	}
	if outra, err := repo.Settings(base, b.ID); err != nil || outra.IsEnabled || outra.PlanCode != 0 {
		t.Fatalf("empresa B herdou a parametrização da A: %+v (%v)", outra, err)
	}

	vence := func(now time.Time) bool {
		t.Helper()
		lista, err := repo.DueSettings(base, now)
		if err != nil {
			t.Fatal(err)
		}
		for _, x := range lista {
			if x.EnterpriseID == b.ID {
				t.Fatal("empresa B sem parametrização entrou na janela")
			}
			if x.EnterpriseID == a.ID {
				return true
			}
		}
		return false
	}
	// 04:30 UTC = 01:30 em Brasília: antes da janela das 02:00 — em UTC já
	// teria passado.
	if vence(time.Date(2026, 10, 12, 4, 30, 0, 0, time.UTC)) {
		t.Fatal("venceu antes das 02:00 de Brasília")
	}
	agora := time.Date(2026, 10, 12, 6, 0, 0, 0, time.UTC) // 03:00 em Brasília
	if !vence(agora) {
		t.Fatal("não venceu às 03:00 de Brasília")
	}
	// Última execução às 23:00 de Brasília do dia anterior (02:00 UTC do dia
	// 12): em UTC seria "hoje" e a janela seria pulada.
	testutil.Exec(t, pool, `UPDATE planning_auto_run_settings SET last_run_at = $2 WHERE enterprise_id = $1`,
		a.ID, time.Date(2026, 10, 12, 2, 0, 0, 0, time.UTC))
	if !vence(agora) {
		t.Fatal("execução da noite anterior impediu a janela de hoje")
	}
	if err := repo.SaveSnapshot(base, a.ID, agora, "SUCCESS"); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `UPDATE planning_auto_run_settings SET last_run_at = $2 WHERE enterprise_id = $1`, a.ID, agora)
	if vence(agora.Add(time.Hour)) {
		t.Fatal("rodou duas vezes no mesmo dia")
	}

	// Salvar de novo não pode apagar o corte nem a última execução.
	if err := repo.SaveSettings(base, planning_uc.AutoRunSettings{
		EnterpriseID: a.ID, IsEnabled: false, RunHour: 3, PlanCode: plano, InitialOrderNumber: 5, UpdatedBy: &ator,
	}); err != nil {
		t.Fatal(err)
	}
	if s, _ = repo.Settings(base, a.ID); s.LastSnapshotAt == nil || s.LastRunAt == nil || s.LastRunStatus != "SUCCESS" || s.RunHour != 3 || s.IsEnabled {
		t.Fatalf("salvar a parametrização apagou a situação da última execução: %+v", s)
	}

	// Da segunda rodada em diante existe corte e o que mudou é contado: a
	// consulta lia uma coluna que o cadastro de item não tem.
	if _, err := repo.ChangedItemsSince(base, a.ID, agora.Add(-24*time.Hour)); err != nil {
		t.Fatalf("contagem do que mudou desde o corte: %v", err)
	}

	id, err := repo.StartRun(base, planning_uc.RunRecord{EnterpriseID: a.ID, PlanCode: plano, Trigger: planning_uc.TriggerManual, TriggeredBy: &ator, SnapshotAt: agora})
	if err != nil {
		t.Fatal(err)
	}
	viavel := true
	if err := repo.FinishRun(base, id, planning_uc.RunRecord{Status: "SUCCESS", MRPOrders: 4, Viable: &viavel}); err != nil {
		t.Fatal(err)
	}
	runs, err := repo.ListRuns(base, a.ID, 10)
	if err != nil || len(runs) != 1 || runs[0].MRPOrders != 4 || runs[0].FinishedAt == nil || runs[0].TriggeredByName == "" || runs[0].Trigger != planning_uc.TriggerManual {
		t.Fatalf("histórico da empresa A: %+v (%v)", runs, err)
	}
	if runs, err := repo.ListRuns(base, b.ID, 10); err != nil || len(runs) != 0 {
		t.Fatalf("empresa B vê o histórico da A: %+v (%v)", runs, err)
	}
}
