//go:build integration

package customer_uc_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/customer_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	customerpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Revisão automática de ponta a ponta: o cliente que atrasou além da
// tolerância tem o limite zerado; o bom pagador sobe com o teto de +50%; a fila
// vê os dois e não vê quem foi revisto agora; a empresa B não revê nem lê
// nada da A — inclusive o faturamento declarado.
func TestRevisaoAutomaticaDeCredito(t *testing.T) {
	pool := testutil.Pool(t)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		testutil.Exec(t, pool, `DELETE FROM contas_receber WHERE enterprise_id = ANY($1)`, ids)
		testutil.Exec(t, pool, `DELETE FROM customer_credit_reviews WHERE enterprise_id = ANY($1)`, ids)
		testutil.Exec(t, pool, `DELETE FROM customer_financial_statements WHERE enterprise_id = ANY($1)`, ids)
		testutil.Exec(t, pool, `DELETE FROM sales_quotation_parameters WHERE enterprise_code = ANY($1)`, []int64{a.Code, b.Code})
		testutil.Exec(t, pool, `DELETE FROM customers WHERE enterprise_id = ANY($1)`, ids)
	})
	actor := testutil.Actor(t, pool)
	repo := customerpg.New(sqlc.New(pool), pool)
	life := customerpg.NewLifecycle(repo)
	uc := customer_uc.NewCustomerUseCase(repo).WithLifecycle(life, nil, nil)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	novo := func(nome, cpf string, limite int64) int64 {
		code, err := repo.NextCustomerCode(a.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		c, err := repo.CreateCustomer(a.Ctx, &entity.Customer{Code: code, Name: nome, DocumentType: entity.DocumentCPF,
			DocumentNumber: cpf, IsActive: true, AllowsPartialBilling: true, CreatedBy: actor})
		if err != nil {
			t.Fatal(err)
		}
		// Limite aprovado há 4 meses (classe B), como uma análise manual.
		if err := life.ApplyCreditReview(a.Ctx, &entity.CreditReview{CustomerCode: c.Code, Origin: "REVISAO",
			SuggestedLimit: decimal.NewFromInt(limite), ApprovedLimit: decimal.NewFromInt(limite), Score: 700,
			RiskClass: entity.RiskB, ExpiresAt: today.AddDate(0, 2, 0), CreatedBy: actor}); err != nil {
			t.Fatal(err)
		}
		testutil.Exec(t, pool, `UPDATE customers SET credit_reviewed_at = NOW() - interval '4 months' WHERE enterprise_id = $1 AND code = $2`, a.ID, c.Code)
		return c.Code
	}
	titulo := func(cliente int64, emissao, venc time.Time, receb *time.Time, valor string) {
		status := "PENDENTE"
		if receb != nil {
			status = "RECEBIDO"
		}
		testutil.Exec(t, pool, `INSERT INTO contas_receber (cliente_id, data_emissao, data_vencimento, data_recebimento,
		    valor_bruto, valor_recebido, status, criado_por, enterprise_id)
		  VALUES ($1, $2::date, $3::date, $4::date, $5::numeric, CASE WHEN $4::date IS NULL THEN NULL ELSE $5::numeric END, $6, $7, $8)`,
			cliente, emissao, venc, receb, valor, status, actor, a.ID)
	}

	atrasado := novo("ATRASOU", "52998224725", 50000)
	for i := 1; i <= 3; i++ {
		e := today.AddDate(0, -i-1, 0)
		v := e.AddDate(0, 0, 30)
		titulo(atrasado, e, v, &v, "10000")
	}
	titulo(atrasado, today.AddDate(0, 0, -80), today.AddDate(0, 0, -45), nil, "10000") // vencido há 45 dias

	pontual := novo("PONTUAL", "11144477735", 10000)
	for i := 1; i <= 12; i++ {
		e := today.AddDate(0, -i, 0)
		v := e.AddDate(0, 0, 30)
		titulo(pontual, e, v, &v, "20000")
	}

	semHistorico := novo("SEM HISTORICO", "39053344705", 80000)
	bloqueado := novo("BLOQUEADO", "86288366757", 30000)
	testutil.Exec(t, pool, `UPDATE customers SET blocked = true WHERE enterprise_id = $1 AND code = $2`, a.ID, bloqueado)

	// A fila vê os dois (todas as empresas).
	fila, err := life.ListCustomersForAutoCreditReview(a.Ctx, 5000)
	if err != nil {
		t.Fatal(err)
	}
	naFila := map[int64]bool{}
	for _, f := range fila {
		if f.EnterpriseID == a.ID {
			naFila[f.Code] = true
		}
	}
	if !naFila[atrasado] || !naFila[pontual] || naFila[bloqueado] {
		t.Fatalf("fila da revisão: quer atrasado e pontual, sem o bloqueado: %v", naFila)
	}

	// B não revê o cliente da A.
	if _, err := uc.RevisarCreditoAutomatico(b.Ctx, atrasado); err == nil {
		t.Fatal("empresa B reviu o crédito do cliente da A")
	}

	mudou, err := uc.RevisarCreditoAutomatico(a.Ctx, atrasado)
	if err != nil || !mudou {
		t.Fatalf("revisão do atrasado: mudou=%v %v", mudou, err)
	}
	c, _ := repo.GetCustomerByCode(a.Ctx, atrasado)
	if !c.CreditLimit.IsZero() || c.CreditRiskClass == nil || *c.CreditRiskClass != "E" {
		t.Fatalf("atrasado além da tolerância: limite %s classe %v, quer 0/E", c.CreditLimit, c.CreditRiskClass)
	}

	mudou, err = uc.RevisarCreditoAutomatico(a.Ctx, pontual)
	if err != nil || !mudou {
		t.Fatalf("revisão do pontual: mudou=%v %v", mudou, err)
	}
	c, _ = repo.GetCustomerByCode(a.Ctx, pontual)
	if !c.CreditLimit.Equal(decimal.NewFromInt(15000)) {
		t.Fatalf("pontual: limite %s, quer 15.000 (+50%% de 10.000)", c.CreditLimit)
	}
	if c.CreditLimitExpiresAt == nil || !c.CreditLimitExpiresAt.After(today) {
		t.Fatalf("a revisão não renovou a validade: %v", c.CreditLimitExpiresAt)
	}
	revs, err := uc.ListCreditReviews(a.Ctx, pontual)
	if err != nil || len(revs) == 0 || revs[0].Origin != "AUTOMATICA" {
		t.Fatalf("trilha da revisão automática: %+v %v", revs, err)
	}

	// Sem histórico a análise só mede falta de dado: não reduz nem renova.
	mudou, err = uc.RevisarCreditoAutomatico(a.Ctx, semHistorico)
	if err != nil || mudou {
		t.Fatalf("revisão sem histórico: mudou=%v %v", mudou, err)
	}
	c, _ = repo.GetCustomerByCode(a.Ctx, semHistorico)
	if !c.CreditLimit.Equal(decimal.NewFromInt(80000)) {
		t.Fatalf("sem histórico: limite %s, quer 80.000 intacto", c.CreditLimit)
	}
	if revs, _ := uc.ListCreditReviews(a.Ctx, semHistorico); len(revs) != 1 {
		t.Fatalf("sem histórico gravou revisão automática: %d análises", len(revs))
	}

	// Revisto agora: sai da fila.
	fila, _ = life.ListCustomersForAutoCreditReview(a.Ctx, 5000)
	for _, f := range fila {
		if f.EnterpriseID == a.ID && (f.Code == pontual || f.Code == atrasado) {
			t.Fatalf("cliente %d revisto agora continua na fila", f.Code)
		}
	}

	// Limite zerado vencendo não volta para a fila (só no intervalo da política).
	testutil.Exec(t, pool, `UPDATE customers SET credit_limit_expires_at = $3::date, credit_review_attempt_at = NULL
	  WHERE enterprise_id = $1 AND code = $2`, a.ID, atrasado, today.AddDate(0, 0, 3))
	fila, _ = life.ListCustomersForAutoCreditReview(a.Ctx, 5000)
	for _, f := range fila {
		if f.EnterpriseID == a.ID && f.Code == atrasado {
			t.Fatal("limite zerado perto de vencer voltou para a fila")
		}
	}

	// Faturamento declarado: a empresa B não grava, não lê e não apaga o da A.
	decl, err := uc.AddFinancialStatement(a.Ctx, pontual, request.FinancialStatementDTO{
		AnnualRevenue: decimal.NewFromInt(2_400_000), ReferenceDate: today.AddDate(0, -3, 0).Format("2006-01-02"), Source: "dre"})
	if err != nil || decl.Source != "DRE" || !decl.Valid {
		t.Fatalf("declaração: %+v %v", decl, err)
	}
	for nome, dto := range map[string]request.FinancialStatementDTO{
		"valor acima da coluna": {AnnualRevenue: decimal.New(1, 13), ReferenceDate: today.Format("2006-01-02"), Source: "DRE"},
		"três casas":            {AnnualRevenue: decimal.RequireFromString("10.123"), ReferenceDate: today.Format("2006-01-02"), Source: "DRE"},
		"data futura":           {AnnualRevenue: decimal.NewFromInt(10), ReferenceDate: today.AddDate(0, 0, 3).Format("2006-01-02"), Source: "DRE"},
		"fonte inválida":        {AnnualRevenue: decimal.NewFromInt(10), ReferenceDate: today.Format("2006-01-02"), Source: "XPTO"},
	} {
		if _, err := uc.AddFinancialStatement(a.Ctx, pontual, dto); err == nil {
			t.Errorf("declaração com %s foi aceita", nome)
		}
	}
	if _, err := uc.AddFinancialStatement(b.Ctx, pontual, request.FinancialStatementDTO{
		AnnualRevenue: decimal.NewFromInt(1), ReferenceDate: today.Format("2006-01-02"), Source: "DRE"}); err == nil {
		t.Fatal("empresa B gravou declaração no cliente da A")
	}
	if l, _ := uc.ListFinancialStatements(b.Ctx, pontual); len(l) != 0 {
		t.Fatalf("empresa B leu %d declaração(ões) da A", len(l))
	}
	if err := uc.DeleteFinancialStatement(b.Ctx, pontual, decl.ID); err == nil {
		t.Fatal("empresa B apagou a declaração da A")
	}
	if l, _ := uc.ListFinancialStatements(a.Ctx, pontual); len(l) != 1 || !l[0].InUse {
		t.Fatalf("a declaração da A sumiu ou não está em uso: %+v", l)
	}

	// Saneamento: os clientes da A (sem endereço) aparecem só no relatório da A.
	acha := func(r *response.SaneamentoResponse, code int64) bool {
		for _, cat := range r.Categorias {
			for _, c := range cat.Clientes {
				if c.CustomerCode == code {
					return true
				}
			}
		}
		return false
	}
	sa, err := uc.Saneamento(a.Ctx)
	if err != nil || !acha(sa, pontual) || sa.ClientesAtivos < 2 {
		t.Fatalf("saneamento da A: %+v %v", sa, err)
	}
	sb, err := uc.Saneamento(b.Ctx)
	if err != nil || acha(sb, pontual) || acha(sb, atrasado) {
		t.Fatalf("saneamento da B mostrou cliente da A (%v)", err)
	}
}
