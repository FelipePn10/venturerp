//go:build integration

package sla_uc_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sla_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/technical_assistance_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	notificationrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/notification"
	slapg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/slaassistencia"
	tapg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/technical_assistance"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func h(v int64) decimal.Decimal { return decimal.NewFromInt(v) }

// O prazo nasce na abertura pela regra da prioridade (a do cliente vence a
// geral), a primeira resposta é gravada uma vez, o painel classifica e a
// Central de Alertas recebe o risco e o vencimento — e resolve quando o
// chamado anda. Outra empresa não vê regras nem chamados.
func TestSLADoChamadoPrazosRespostaPainelEAlertas(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	acordo, comum := 820_000+u%90_000, 830_000+u%90_000
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []testutil.Empresa{a, b} {
			for _, q := range []string{
				`DELETE FROM notification_alerts WHERE enterprise_id=$1`, `DELETE FROM notification_outbox WHERE enterprise_id=$1`,
				`DELETE FROM enterprise_notification_subscriptions WHERE enterprise_id=$1`,
				`DELETE FROM technical_assistance_calls WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1)`,
				`DELETE FROM sla_assistencia WHERE enterprise_id=$1`, `DELETE FROM customers WHERE enterprise_id=$1`,
			} {
				if _, err := pool.Exec(bg, q, e.ID); err != nil {
					t.Logf("limpeza %s: %v", q, err)
				}
			}
		}
	})
	for _, c := range []int64{acordo, comum} {
		testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES($1,'CLIENTE SLA','11444777000161',$2,$3)`, c, actor, a.ID)
	}

	uc := &sla_uc.UseCase{Repo: slapg.New(pool), Auth: &auth.AuthService{}}
	if _, err := uc.SalvarRegra(a.Ctx, 0, sla_uc.RegraDTO{Prioridade: "high", HorasResposta: h(2), HorasSolucao: h(8)}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.SalvarRegra(a.Ctx, 0, sla_uc.RegraDTO{Prioridade: "HIGH", HorasResposta: h(3), HorasSolucao: h(9)}); err == nil {
		t.Fatal("segunda regra geral da mesma prioridade")
	}
	especial, err := uc.SalvarRegra(a.Ctx, 0, sla_uc.RegraDTO{Prioridade: "HIGH", CustomerCode: &acordo, HorasResposta: h(1), HorasSolucao: h(4), AlertaPct: 75})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.SalvarRegra(b.Ctx, 0, sla_uc.RegraDTO{Prioridade: "HIGH", CustomerCode: &acordo, HorasResposta: h(1), HorasSolucao: h(4)}); err == nil {
		t.Fatal("a outra empresa não usa o cliente desta")
	}
	if rs, _ := uc.Regras(b.Ctx); len(rs) != 0 {
		t.Fatalf("a outra empresa não vê as regras: %d", len(rs))
	}

	ta := &technical_assistance_uc.UseCase{Repo: tapg.New(pool), Auth: &auth.AuthService{}, SLA: uc}
	abrir := func(cliente int64, prioridade string) int64 {
		c, err := ta.CreateCall(a.Ctx, request.CreateTechnicalAssistanceCallDTO{EnterpriseCode: a.Code, CustomerCode: cliente, Subject: "Chamado SLA", Priority: prioridade})
		if err != nil {
			t.Fatal(err)
		}
		return c.Code
	}
	prazo := func(call int64) (regra *int64, resp, sol *time.Time, aberto time.Time) {
		if err := pool.QueryRow(bg, `SELECT sla_id, sla_resposta_ate, sla_solucao_ate, opened_at FROM technical_assistance_calls WHERE code=$1`, call).Scan(&regra, &resp, &sol, &aberto); err != nil {
			t.Fatal(err)
		}
		return
	}
	cAcordo, cComum, cBaixa := abrir(acordo, "HIGH"), abrir(comum, "HIGH"), abrir(comum, "LOW")
	if r, resp, sol, ab := prazo(cAcordo); r == nil || *r != especial.ID || !resp.Equal(ab.Add(time.Hour)) || !sol.Equal(ab.Add(4*time.Hour)) {
		t.Fatalf("cliente com acordo: regra %v resposta %v solução %v (aberto %v)", r, resp, sol, ab)
	}
	if r, _, sol, ab := prazo(cComum); r == nil || *r == especial.ID || !sol.Equal(ab.Add(8*time.Hour)) {
		t.Fatalf("outro cliente usa a regra geral: %v %v", r, sol)
	}
	if r, resp, _, _ := prazo(cBaixa); r != nil || resp != nil {
		t.Fatal("prioridade sem regra fica sem SLA")
	}

	// Primeira resposta: sair de pendente; a segunda mudança não regrava.
	if _, err := ta.UpdateStatus(a.Ctx, request.UpdateTechnicalAssistanceCallStatusDTO{Code: cComum, Status: "IN_ANALYSIS"}); err != nil {
		t.Fatal(err)
	}
	var resp1, resp2 *time.Time
	_ = pool.QueryRow(bg, `SELECT respondido_em FROM technical_assistance_calls WHERE code=$1`, cComum).Scan(&resp1)
	if _, err := ta.UpdateStatus(a.Ctx, request.UpdateTechnicalAssistanceCallStatusDTO{Code: cComum, Status: "WAITING_RETURN"}); err != nil {
		t.Fatal(err)
	}
	_ = pool.QueryRow(bg, `SELECT respondido_em FROM technical_assistance_calls WHERE code=$1`, cComum).Scan(&resp2)
	if resp1 == nil || resp2 == nil || !resp1.Equal(*resp2) {
		t.Fatalf("primeira resposta gravada uma vez: %v %v", resp1, resp2)
	}

	// O chamado do acordo "envelhece": resposta vencida, solução em risco (3,5h de 4h).
	testutil.Exec(t, pool, `UPDATE technical_assistance_calls SET opened_at=NOW()-INTERVAL '210 minutes', sla_resposta_ate=NOW()-INTERVAL '150 minutes',
		sla_solucao_ate=NOW()+INTERVAL '30 minutes', sla_resposta_risco_em=NOW()-INTERVAL '165 minutes', sla_solucao_risco_em=NOW()-INTERVAL '30 minutes' WHERE code=$1`, cAcordo)
	p, err := uc.Painel(a.Ctx, "", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if p.Resumo.Abertos != 2 || p.Resumo.Vencidos != 1 {
		t.Fatalf("painel: %+v", p.Resumo)
	}
	if v, _ := uc.Painel(a.Ctx, "VENCIDO", nil, 0); len(v.Chamados) != 1 || v.Chamados[0].CallCode != cAcordo || v.Chamados[0].SituacaoSoluc != "EM_RISCO" {
		t.Fatalf("filtro de vencidos: %+v", v.Chamados)
	}
	if pb, _ := uc.Painel(b.Ctx, "TODOS", nil, 0); len(pb.Chamados) != 0 {
		t.Fatal("a outra empresa não vê os chamados")
	}

	// Central de Alertas: só com assinatura; risco e vencimento por prazo.
	notif := notificationrepo.New(pool)
	if err := notif.ScheduleOperationalAlerts(bg); err != nil {
		t.Fatal(err)
	}
	contar := func() int {
		var n int
		_ = pool.QueryRow(bg, `SELECT COUNT(*) FROM notification_outbox WHERE enterprise_id=$1 AND event_key LIKE 'ASSISTENCIA_SLA_%'`, a.ID).Scan(&n)
		return n
	}
	if n := contar(); n != 0 {
		t.Fatalf("sem assinatura não gera alerta: %d", n)
	}
	for _, ev := range []string{"ASSISTENCIA_SLA_EM_RISCO", "ASSISTENCIA_SLA_VENCIDO"} {
		testutil.Exec(t, pool, `INSERT INTO enterprise_notification_subscriptions(enterprise_id,event_key,cadence,created_by) VALUES($1,$2,'RESUMO_DIARIO',$3)`, a.ID, ev, actor)
	}
	if err := notif.ScheduleOperationalAlerts(bg); err != nil {
		t.Fatal(err)
	}
	rows, err := pool.Query(bg, `SELECT deduplication_key FROM notification_outbox WHERE enterprise_id=$1 AND event_key LIKE 'ASSISTENCIA_SLA_%' ORDER BY 1`, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	var chaves []string
	for rows.Next() {
		var k string
		_ = rows.Scan(&k)
		chaves = append(chaves, k)
	}
	rows.Close()
	quer := []string{"chamado:" + itoa(cAcordo) + ":resposta:vencido:", "chamado:" + itoa(cAcordo) + ":solucao:risco:"}
	if len(chaves) != 2 || !strings.HasPrefix(chaves[0], quer[0]) || !strings.HasPrefix(chaves[1], quer[1]) {
		t.Fatalf("alertas: %v, queria %v…", chaves, quer)
	}
	quer[0] = chaves[0]
	if err := notif.ScheduleOperationalAlerts(bg); err != nil || contar() != 2 {
		t.Fatalf("rodar de novo não duplica: %d %v", contar(), err)
	}

	// O alerta aberto se resolve quando o chamado responde.
	testutil.Exec(t, pool, `INSERT INTO notification_alerts(enterprise_id,event_key,aggregate_type,aggregate_internal_id,deduplication_key,severity,summary)
		VALUES($1,'ASSISTENCIA_SLA_VENCIDO','CHAMADO_ASSISTENCIA',$2,$3,'CRITICO','vencido')`, a.ID, itoa(cAcordo), quer[0])
	if err := uc.Responder(a.Ctx, cAcordo); err != nil {
		t.Fatal(err)
	}
	if err := notif.ScheduleOperationalAlerts(bg); err != nil {
		t.Fatal(err)
	}
	var estado, motivo string
	if err := pool.QueryRow(bg, `SELECT state, resolution_reason FROM notification_alerts WHERE enterprise_id=$1 AND deduplication_key=$2`, a.ID, quer[0]).Scan(&estado, &motivo); err != nil {
		t.Fatal(err)
	}
	if estado != "RESOLVIDO" || motivo != "Chamado respondido" {
		t.Fatalf("alerta de resposta: %s %s", estado, motivo)
	}
}

func itoa(v int64) string { return decimal.NewFromInt(v).String() }

// Horas úteis e relógio parado: a regra em horas úteis conta só o expediente
// (com o feriado do calendário industrial); aguardando o cliente o relógio
// para — o painel mostra PAUSADO e a Central não alerta — e na retomada a
// solução anda o tempo parado, em horas úteis. A pausa fica registrada.
func TestSLAHorasUteisEPausa(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	cliente := 840_000 + u%90_000
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []testutil.Empresa{a, b} {
			for _, q := range []string{
				`DELETE FROM notification_alerts WHERE enterprise_id=$1`, `DELETE FROM notification_outbox WHERE enterprise_id=$1`,
				`DELETE FROM enterprise_notification_subscriptions WHERE enterprise_id=$1`,
				`DELETE FROM sla_pausas WHERE enterprise_id=$1`,
				`DELETE FROM technical_assistance_calls WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1)`,
				`DELETE FROM sla_assistencia WHERE enterprise_id=$1`, `DELETE FROM sla_expediente WHERE enterprise_id=$1`,
				`DELETE FROM industrial_calendar WHERE enterprise_id=$1`, `DELETE FROM customers WHERE enterprise_id=$1`,
			} {
				if _, err := pool.Exec(bg, q, e.ID); err != nil {
					t.Logf("limpeza %s: %v", q, err)
				}
			}
		}
	})
	testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES($1,'CLIENTE SLA ÚTIL','11444777000161',$2,$3)`, cliente, actor, a.ID)
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Skip("sem tzdata")
	}
	em := func(dia, hora int) time.Time { return time.Date(2026, 10, dia, hora, 0, 0, 0, loc) }
	agora := em(26, 16) // segunda 16h
	uc := &sla_uc.UseCase{Repo: slapg.New(pool), Auth: &auth.AuthService{}, Agora: func() time.Time { return agora }}

	// Expediente: o padrão até cadastrar; horário inválido é recusado.
	if e, err := uc.Expediente(a.Ctx); err != nil || !e.Padrao || len(e.Intervalos) != 10 {
		t.Fatalf("expediente padrão: %+v %v", e, err)
	}
	if _, err := uc.SalvarExpediente(a.Ctx, []sla_uc.HorarioDTO{{DiaSemana: 1, Inicio: "12:00", Fim: "08:00"}}); err == nil {
		t.Fatal("intervalo invertido")
	}
	var semana []sla_uc.HorarioDTO
	for d := 1; d <= 5; d++ {
		semana = append(semana, sla_uc.HorarioDTO{DiaSemana: d, Inicio: "08:00", Fim: "12:00"}, sla_uc.HorarioDTO{DiaSemana: d, Inicio: "13:00", Fim: "18:00"})
	}
	if e, err := uc.SalvarExpediente(a.Ctx, semana); err != nil || e.Padrao || len(e.Intervalos) != 10 {
		t.Fatalf("salvar expediente: %+v %v", e, err)
	}
	if e, _ := uc.Expediente(b.Ctx); !e.Padrao {
		t.Fatal("a outra empresa segue no padrão")
	}
	// Quarta 28/10 é feriado na empresa A.
	testutil.Exec(t, pool, `INSERT INTO industrial_calendar(year,month,day,is_workday,description,enterprise_id) VALUES(2026,10,28,false,'Feriado',$1)`, a.ID)

	if _, err := uc.SalvarRegra(a.Ctx, 0, sla_uc.RegraDTO{Prioridade: "HIGH", HorasResposta: h(2), HorasSolucao: h(18), AlertaPct: 50, HorasUteis: true}); err != nil {
		t.Fatal(err)
	}
	ta := &technical_assistance_uc.UseCase{Repo: tapg.New(pool), Auth: &auth.AuthService{}, SLA: uc}
	c, err := ta.CreateCall(a.Ctx, request.CreateTechnicalAssistanceCallDTO{EnterpriseCode: a.Code, CustomerCode: cliente, Subject: "SLA útil", Priority: "HIGH"})
	if err != nil {
		t.Fatal(err)
	}
	// A abertura é a de segunda 16h (fixada para o teste não depender do relógio).
	testutil.Exec(t, pool, `UPDATE technical_assistance_calls SET opened_at=$2 WHERE code=$1`, c.Code, agora)
	if err := uc.AplicarAoChamado(a.Ctx, c.Code); err != nil {
		t.Fatal(err)
	}
	ler := func() (resp, sol, solRisco, pausado *time.Time, uteis bool, horas decimal.Decimal) {
		t.Helper()
		if err := pool.QueryRow(bg, `SELECT sla_resposta_ate, sla_solucao_ate, sla_solucao_risco_em, sla_pausado_desde, sla_horas_uteis, sla_horas_pausadas
			FROM technical_assistance_calls WHERE code=$1`, c.Code).Scan(&resp, &sol, &solRisco, &pausado, &uteis, &horas); err != nil {
			t.Fatal(err)
		}
		return
	}
	// Resposta 2 h: segunda 18h. Solução 18 h: seg 2 + ter 9 + (qua feriado) + qui 7 → quinta 16h.
	// Risco a 50% (9 h): seg 2 + ter 7 → terça 16h.
	resp, sol, risco, _, uteis, _ := ler()
	if !uteis || !resp.Equal(em(26, 18)) || !sol.Equal(em(29, 16)) || !risco.Equal(em(27, 16)) {
		t.Fatalf("prazos úteis: resposta %v solução %v risco %v", resp.In(loc), sol.In(loc), risco.In(loc))
	}

	// Terça 10h passa a aguardar o cliente: relógio parado.
	agora = em(27, 10)
	if _, err := ta.UpdateStatus(a.Ctx, request.UpdateTechnicalAssistanceCallStatusDTO{Code: c.Code, Status: "WAITING_RETURN"}); err != nil {
		t.Fatal(err)
	}
	if _, _, _, p, _, _ := ler(); p == nil || !p.Equal(em(27, 10)) {
		t.Fatalf("pausado desde terça 10h: %v", p)
	}
	// Quinta 17h, ainda parado: no painel, PAUSADO (e não vencido); sem alerta.
	agora = em(29, 17)
	pn, err := uc.Painel(a.Ctx, "PAUSADO", nil, 0)
	if err != nil || len(pn.Chamados) != 1 || pn.Chamados[0].SituacaoSoluc != "PAUSADO" || pn.Resumo.Pausados != 1 || pn.Resumo.Vencidos != 0 {
		t.Fatalf("painel pausado: %+v %v", pn, err)
	}
	for _, ev := range []string{"ASSISTENCIA_SLA_EM_RISCO", "ASSISTENCIA_SLA_VENCIDO"} {
		testutil.Exec(t, pool, `INSERT INTO enterprise_notification_subscriptions(enterprise_id,event_key,cadence,created_by) VALUES($1,$2,'RESUMO_DIARIO',$3)`, a.ID, ev, actor)
	}
	// Deixa o prazo gravado no passado para provar que a pausa é que segura o alerta.
	testutil.Exec(t, pool, `UPDATE technical_assistance_calls SET sla_solucao_ate=NOW()-INTERVAL '1 hour', sla_solucao_risco_em=NOW()-INTERVAL '2 hours',
		sla_resposta_ate=NOW()-INTERVAL '1 hour', sla_resposta_risco_em=NOW()-INTERVAL '2 hours' WHERE code=$1`, c.Code)
	notif := notificationrepo.New(pool)
	if err := notif.ScheduleOperationalAlerts(bg); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = pool.QueryRow(bg, `SELECT COUNT(*) FROM notification_outbox WHERE enterprise_id=$1 AND event_key LIKE 'ASSISTENCIA_SLA_%'`, a.ID).Scan(&n)
	if n != 0 {
		t.Fatalf("parado não alerta: %d", n)
	}
	testutil.Exec(t, pool, `UPDATE technical_assistance_calls SET sla_solucao_ate=$2, sla_solucao_risco_em=$3, sla_resposta_ate=$4, sla_resposta_risco_em=NULL WHERE code=$1`,
		c.Code, sol, risco, resp)

	// Quinta 10h volta à análise: parado ter 10–18 (7 h) + qua feriado + qui 8–10 (2 h) = 9 h úteis.
	// A solução de quinta 16h anda 9 h úteis: qui 2 + sex 7 → sexta 16h. Risco terça 16h + 9 h → quinta 10h... + o que faltava.
	agora = em(29, 10)
	if _, err := ta.UpdateStatus(a.Ctx, request.UpdateTechnicalAssistanceCallStatusDTO{Code: c.Code, Status: "IN_ANALYSIS"}); err != nil {
		t.Fatal(err)
	}
	_, sol2, risco2, pausado, _, horas := ler()
	if pausado != nil || !sol2.Equal(em(30, 16)) || !horas.Equal(decimal.NewFromInt(9)) {
		t.Fatalf("retomada: solução %v pausado %v horas %s", sol2.In(loc), pausado, horas)
	}
	// Risco: terça 16h + 9 h úteis = ter 2 h + qui 7 h → quinta 16h.
	if !risco2.Equal(em(29, 16)) {
		t.Fatalf("risco estendido: %v", risco2.In(loc))
	}
	var motivo string
	var hp decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT motivo, horas FROM sla_pausas WHERE enterprise_id=$1 AND call_code=$2 AND fim IS NOT NULL`, a.ID, c.Code).Scan(&motivo, &hp); err != nil || motivo != "WAITING_RETURN" || !hp.Equal(decimal.NewFromInt(9)) {
		t.Fatalf("pausa registrada: %s %s %v", motivo, hp, err)
	}
	// Ir de uma espera a outra não reinicia; atender saindo da espera retoma.
	agora = em(30, 9)
	_, _ = ta.UpdateStatus(a.Ctx, request.UpdateTechnicalAssistanceCallStatusDTO{Code: c.Code, Status: "WAITING_ORDER"})
	agora = em(30, 11)
	_, _ = ta.UpdateStatus(a.Ctx, request.UpdateTechnicalAssistanceCallStatusDTO{Code: c.Code, Status: "WAITING_RETURN"})
	var abertas int
	_ = pool.QueryRow(bg, `SELECT COUNT(*) FROM sla_pausas WHERE call_code=$1 AND fim IS NULL`, c.Code).Scan(&abertas)
	if _, _, _, p, _, _ := ler(); abertas != 1 || p == nil || !p.Equal(em(30, 9)) {
		t.Fatalf("espera contínua: abertas %d desde %v", abertas, p)
	}
	if pb, _ := uc.Painel(b.Ctx, "TODOS", nil, 0); len(pb.Chamados) != 0 {
		t.Fatal("a outra empresa não vê")
	}
}
