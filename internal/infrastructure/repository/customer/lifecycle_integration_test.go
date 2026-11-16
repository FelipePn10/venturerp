//go:build integration

package customer_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	customerpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestCustomerCreditLifecycle executa de ponta a ponta o SQL do crédito, do
// uso do cliente, do painel e das exclusões, numa empresa descartável — e
// confere que a outra empresa não enxerga nada.
func TestCustomerCreditLifecycle(t *testing.T) {
	pool := testutil.Pool(t)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		testutil.Exec(t, pool, `DELETE FROM contas_receber WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
		testutil.Exec(t, pool, `DELETE FROM sales_quotation_parameters WHERE enterprise_code = ANY($1)`, []int64{a.Code, b.Code})
		testutil.Exec(t, pool, `DELETE FROM customers WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
	})
	actor := testutil.Actor(t, pool)
	repo := customerpg.New(sqlc.New(pool), pool)
	life := customerpg.NewLifecycle(repo)
	// Data vinda do processo (regra dos dois relógios do AGENTS.md).
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	code, err := repo.NextCustomerCode(a.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	matriz, err := repo.CreateCustomer(a.Ctx, &entity.Customer{Code: code, Name: "MATRIZ", DocumentType: entity.DocumentCNPJ,
		DocumentNumber: "11.222.333/0001-81", IsActive: true, AllowsPartialBilling: true, CreatedBy: actor})
	if err != nil {
		t.Fatal(err)
	}
	filial, err := repo.CreateCustomer(a.Ctx, &entity.Customer{Code: code + 1, Name: "FILIAL", DocumentType: entity.DocumentCNPJ,
		DocumentNumber: "11222333000262", IsActive: true, AllowsPartialBilling: true, CreatedBy: actor})
	if err != nil {
		t.Fatal(err)
	}

	group, err := life.GroupCustomerCodes(a.Ctx, entity.DocumentCNPJ, "11222333000199")
	if err != nil || len(group) != 2 {
		t.Fatalf("grupo econômico pela raiz: %v %v", group, err)
	}
	if g, _ := life.GroupCustomerCodes(b.Ctx, entity.DocumentCNPJ, "11222333000199"); len(g) != 0 {
		t.Fatalf("empresa B viu o grupo da A: %v", g)
	}

	// Títulos: 4 pagos (um com 10 dias de atraso), 1 vencido há 40 dias na filial.
	titulo := func(cliente int64, emissao, venc time.Time, receb *time.Time, valor, status string) {
		testutil.Exec(t, pool, `INSERT INTO contas_receber (cliente_id, data_emissao, data_vencimento, data_recebimento,
		    valor_bruto, valor_recebido, status, criado_por, enterprise_id)
		  VALUES ($1, $2::date, $3::date, $4::date, $5::numeric, CASE WHEN $4::date IS NULL THEN NULL ELSE $5::numeric END, $6, $7, $8)`,
			cliente, emissao, venc, receb, valor, status, actor, a.ID)
	}
	for i := 1; i <= 3; i++ {
		e := today.AddDate(0, -i, 0)
		v := e.AddDate(0, 0, 30)
		titulo(matriz.Code, e, v, &v, "1000", "RECEBIDO")
	}
	e := today.AddDate(0, -5, 0)
	v := e.AddDate(0, 0, 30)
	r := v.AddDate(0, 0, 10)
	titulo(matriz.Code, e, v, &r, "1000", "RECEBIDO")
	titulo(filial.Code, today.AddDate(0, 0, -70), today.AddDate(0, 0, -40), nil, "500", "PENDENTE")

	h, err := life.GetCreditHistory(a.Ctx, []int64{matriz.Code}, 24, today)
	if err != nil {
		t.Fatal(err)
	}
	if h.PaidTitles != 4 || !h.PaidOnTimeValue.Equal(decimal.NewFromInt(3000)) || h.MaxDelayDays != 10 ||
		!h.Billed12m.Equal(decimal.NewFromInt(4000)) || !h.AvgTermDays.Equal(decimal.NewFromInt(30)) || h.MaxOpenOverdueDays != 0 {
		t.Fatalf("histórico da matriz: %+v", h)
	}
	hg, err := life.GetCreditHistory(a.Ctx, group, 24, today)
	if err != nil {
		t.Fatal(err)
	}
	if hg.MaxOpenOverdueDays != 40 || !hg.OverdueBalance.Equal(decimal.NewFromInt(500)) {
		t.Fatalf("o vencido da filial não pesou no grupo: %+v", hg)
	}
	if hb, _ := life.GetCreditHistory(b.Ctx, group, 24, today); hb.PaidTitles != 0 {
		t.Fatal("empresa B leu o histórico da A")
	}

	// Trilha + limite na mesma transação.
	reason := "cliente apresentou garantias"
	if err := life.ApplyCreditReview(a.Ctx, &entity.CreditReview{CustomerCode: matriz.Code, Origin: "REVISAO",
		SuggestedLimit: decimal.NewFromInt(2000), ApprovedLimit: decimal.NewFromInt(2100), Score: 700, RiskClass: entity.RiskB,
		ExpiresAt: today.AddDate(0, 6, 0), Deviates: true, Reason: &reason,
		Factors: []entity.CreditFactor{{Code: "PONTUALIDADE", Points: 225}}, CreatedBy: actor}); err != nil {
		t.Fatal(err)
	}
	if err := life.ApplyCreditReview(b.Ctx, &entity.CreditReview{CustomerCode: matriz.Code, Origin: "REVISAO", RiskClass: entity.RiskA,
		ExpiresAt: today, CreatedBy: actor}); err == nil {
		t.Fatal("empresa B gravou limite no cliente da A")
	}
	revs, err := life.ListCreditReviews(a.Ctx, matriz.Code)
	if err != nil || len(revs) != 1 || revs[0].Factors[0].Points != 225 || !revs[0].ApprovedLimit.Equal(decimal.NewFromInt(2100)) {
		t.Fatalf("trilha: %+v %v", revs, err)
	}
	if revs[0].CreatedByName == "" {
		t.Fatal("a trilha não traz o nome de quem aprovou")
	}
	got, _ := repo.GetCustomerByCode(a.Ctx, matriz.Code)
	if !got.CreditLimit.Equal(decimal.NewFromInt(2100)) || got.CreditRiskClass == nil || *got.CreditRiskClass != "B" || got.CreditReviewedAt == nil {
		t.Fatalf("limite não gravado: %+v", got)
	}

	// Política: sem linha vale o padrão; gravada, volta igual.
	pol, err := life.GetCreditPolicy(a.Ctx)
	if err != nil || pol.OverdueBlockDays != 30 {
		t.Fatalf("política padrão: %+v %v", pol, err)
	}
	pol.MaxLimit = decimal.NewFromInt(50000)
	if err := life.SaveCreditPolicy(a.Ctx, pol, actor); err != nil {
		t.Fatal(err)
	}
	if p2, _ := life.GetCreditPolicy(a.Ctx); !p2.MaxLimit.Equal(decimal.NewFromInt(50000)) || p2.ValidityMonths[entity.RiskA] != 12 {
		t.Fatalf("política gravada: %+v", p2)
	}
	if pb, _ := life.GetCreditPolicy(b.Ctx); pb.MaxLimit.IsPositive() {
		t.Fatal("empresa B herdou a política da A")
	}

	// Situação na Receita.
	capital := decimal.NewFromInt(150000)
	if err := life.SetCustomerRegistration(a.Ctx, matriz.Code, entity.RegistrationSnapshot{Status: "ATIVA", ShareCapital: &capital, MainCNAE: "2599399"}); err != nil {
		t.Fatal(err)
	}
	got, _ = repo.GetCustomerByCode(a.Ctx, matriz.Code)
	if got.RegistrationStatus == nil || *got.RegistrationStatus != "ATIVA" || got.ShareCapital == nil || !got.ShareCapital.Equal(capital) {
		t.Fatalf("situação cadastral: %+v", got)
	}
	if _, err := life.ListCustomersForRegistrationCheck(a.Ctx, 5); err != nil {
		t.Fatal(err)
	}

	// Uso, painel e exclusão.
	usages, err := life.CustomerUsage(a.Ctx, matriz.Code)
	if err != nil {
		t.Fatal(err)
	}
	if err := entity.CheckCanDelete(usages); err == nil {
		t.Fatal("cliente com títulos não pode ser excluído")
	}
	if err := entity.CheckCanInactivate(usages); err != nil {
		t.Fatalf("títulos pagos não impedem inativar: %v", err)
	}
	fu, _ := life.CustomerUsage(a.Ctx, filial.Code)
	if err := entity.CheckCanInactivate(fu); err == nil {
		t.Fatal("título em aberto devia impedir inativar a filial")
	}
	o, err := life.CustomerOverview(a.Ctx, filial.Code, group, today)
	if err != nil {
		t.Fatal(err)
	}
	if !o.OverdueReceivables.Equal(decimal.NewFromInt(500)) || o.MaxOpenOverdueDays != 40 || len(o.BillingByMonth) != 12 || o.GroupMembers != 2 {
		t.Fatalf("painel: %+v", o)
	}

	nunca, err := repo.CreateCustomer(a.Ctx, &entity.Customer{Code: code + 2, Name: "SEM USO", DocumentType: entity.DocumentCPF,
		DocumentNumber: "52998224725", IsActive: true, CreatedBy: actor})
	if err != nil {
		t.Fatal(err)
	}
	if err := life.DeleteCustomer(b.Ctx, nunca.Code, entity.CheckCanDelete); err == nil {
		t.Fatal("empresa B excluiu cliente da A")
	}
	// Consumidor final padrão do orçamento é vínculo sem FK: sem a conferência,
	// a exclusão deixava o parâmetro apontando para um código que outro cliente
	// herdaria.
	testutil.Exec(t, pool, `INSERT INTO sales_quotation_parameters (enterprise_code, final_consumer_customer_code) VALUES ($1, $2)`, a.Code, nunca.Code)
	if err := life.DeleteCustomer(a.Ctx, nunca.Code, entity.CheckCanDelete); err == nil {
		t.Fatal("excluiu o cliente que é o consumidor final padrão do orçamento")
	}
	testutil.Exec(t, pool, `DELETE FROM sales_quotation_parameters WHERE enterprise_code = $1`, a.Code)
	nu, _ := life.CustomerUsage(a.Ctx, nunca.Code)
	if err := entity.CheckCanDelete(nu); err != nil {
		t.Fatalf("cliente sem uso devia poder ser excluído: %v", err)
	}
	if err := life.DeleteCustomer(a.Ctx, matriz.Code, entity.CheckCanDelete); err == nil {
		t.Fatal("a exclusão transacional deixou passar cliente com títulos")
	}
	if err := life.DeleteCustomer(a.Ctx, nunca.Code, entity.CheckCanDelete); err != nil {
		t.Fatal(err)
	}
	// Código reaproveitado não herda a trilha do cliente excluído.
	if err := life.ApplyCreditReview(a.Ctx, &entity.CreditReview{CustomerCode: nunca.Code, Origin: "CADASTRO", RiskClass: entity.RiskD,
		ExpiresAt: today, CreatedBy: actor}); err == nil {
		t.Fatal("gravou análise para cliente excluído")
	}
	novo, err := repo.CreateCustomer(a.Ctx, &entity.Customer{Code: nunca.Code, Name: "MESMO CÓDIGO", DocumentType: entity.DocumentCPF,
		DocumentNumber: "11144477735", IsActive: true, CreatedBy: actor})
	if err != nil {
		t.Fatal(err)
	}
	if revs, err := life.ListCreditReviews(a.Ctx, novo.Code); err != nil || len(revs) != 0 {
		t.Fatalf("o cliente novo herdou %d análise(s): %v", len(revs), err)
	}
	if err := life.SetCustomerActive(a.Ctx, matriz.Code, false); err != nil {
		t.Fatal(err)
	}

	// Cadastro de apoio: exclui o livre, recusa o usado.
	reg, err := repo.CreateRegion(a.Ctx, &entity.Region{Code: 1, Description: "SUL", UF: "PR", City: "Curitiba", IsActive: true, CreatedBy: actor})
	if err != nil {
		t.Fatal(err)
	}
	matriz.RegionID = &reg.ID
	if _, err := repo.UpdateCustomer(a.Ctx, matriz); err != nil {
		t.Fatal(err)
	}
	if err := life.DeleteSupport(a.Ctx, "regions", reg.Code); err == nil {
		t.Fatal("região em uso foi excluída")
	}
	// Em uso não sai, mas inativa: some da lista de escolha e volta ao reativar.
	if err := life.SetSupportActive(a.Ctx, "regions", reg.Code, false); err != nil {
		t.Fatalf("inativando a região: %v", err)
	}
	if ativas, err := repo.ListRegions(a.Ctx, true); err != nil || len(ativas) != 0 {
		t.Fatalf("região inativa ainda na lista de escolha: %d %v", len(ativas), err)
	}
	if err := life.SetSupportActive(a.Ctx, "regions", reg.Code, true); err != nil {
		t.Fatal(err)
	}
	if ativas, _ := repo.ListRegions(a.Ctx, true); len(ativas) != 1 {
		t.Fatal("região reativada não voltou")
	}
	if err := life.SetSupportActive(a.Ctx, "contact-types", 1, false); err == nil {
		t.Fatal("tipo de contato não tem situação e aceitou inativar")
	}
	if err := life.SetSupportActive(b.Ctx, "regions", reg.Code, false); err == nil {
		t.Fatal("empresa B inativou a região da A")
	}
	livre, err := repo.CreateRegion(a.Ctx, &entity.Region{Code: 2, Description: "NORTE", UF: "PA", City: "Belém", IsActive: true, CreatedBy: actor})
	if err != nil {
		t.Fatal(err)
	}
	if err := life.DeleteSupport(a.Ctx, "regions", livre.Code); err != nil {
		t.Fatalf("região livre devia ser excluída: %v", err)
	}
	for _, res := range []string{"regions", "market-segments", "contact-types", "customer-types", "carriers", "carrier-groups", "invoice-types", "tax-types"} {
		if err := life.DeleteSupport(a.Ctx, res, 999999); err == nil {
			t.Fatalf("%s inexistente devia dar não encontrado", res)
		}
	}
}
