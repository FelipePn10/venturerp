//go:build integration

// Código de benefício fiscal (cBenef) e rejeição da NF-e (migração 000404).
//
//	TEST_DATABASE_URL=... go test -tags=integration -run "BeneficioFiscal|RejeicaoDaNFe" ./internal/infrastructure/repository/fiscal/
package fiscal_test

import (
	"context"
	"testing"
	"time"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

func empresaFiscal(t *testing.T) (context.Context, int64) {
	t.Helper()
	pool := testutil.Pool(t)
	ator := testutil.Actor(t, pool)
	code := 1_985_000_000 + testutil.UniqueCode()%10_000_000
	var id int64
	if err := pool.QueryRow(context.Background(), `INSERT INTO enterprise(code,name) VALUES($1,'cBenef') RETURNING id`, code).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM icms_ipi_tax_params WHERE enterprise_id=$1`, id)
		testutil.Exec(t, pool, `DELETE FROM fiscal_exits WHERE enterprise_id=$1`, id)
		testutil.Exec(t, pool, `DELETE FROM items WHERE enterprise_id=$1`, id)
		testutil.Exec(t, pool, `DELETE FROM notification_outbox WHERE enterprise_id=$1`, id)
		testutil.Exec(t, pool, `DELETE FROM enterprise WHERE id=$1`, id)
	})
	return context.WithValue(context.Background(), contextkey.UserKey,
		&appsecurity.AuthUser{ID: ator.String(), Role: "ADMIN", EnterpriseID: id, EnterpriseCode: code}), id
}

// Parâmetro do item vence o do NCM; o do cliente vence o geral; outra UF e
// parâmetro de entrada não valem.
func TestBeneficioFiscalPrioridade(t *testing.T) {
	pool := testutil.Pool(t)
	ctx, emp := empresaFiscal(t)
	ator := testutil.Actor(t, pool)
	item := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,1,$2,$3)`, item, ator, emp)
	ins := `INSERT INTO icms_ipi_tax_params(enterprise_id,uf,operation_type,ncm_code,item_code,customer_code,cod_beneficio_fiscal,is_active) VALUES($1,$2,$3,$4,$5,$6,$7,true)`
	testutil.Exec(t, pool, ins, emp, "PR", "SAIDA", "7326.90.00", nil, nil, "PR800001")
	testutil.Exec(t, pool, ins, emp, "SP", "SAIDA", nil, item, nil, "SP000001")
	testutil.Exec(t, pool, ins, emp, "PR", "ENTRADA", nil, item, nil, "PR000099")
	r := fiscalpg.NewBeneficioFiscalPG(pool)
	ncm := "73269000"

	got, err := r.BeneficioFiscal(ctx, &item, &ncm, "pr", nil)
	if err != nil || got == nil || *got != "PR800001" {
		t.Fatalf("pelo NCM: %v %v, quer PR800001", got, err)
	}
	testutil.Exec(t, pool, ins, emp, "PR", "AMBAS", nil, item, nil, "PR830001")
	if got, _ := r.BeneficioFiscal(ctx, &item, &ncm, "PR", nil); got == nil || *got != "PR830001" {
		t.Fatalf("parâmetro do item tem de vencer o do NCM: %v", got)
	}
	// Código único: o 77 fixo colidia com cliente real de outra empresa num
	// banco de teste com dados, e a conferência de empresa recusava.
	cliente := testutil.UniqueCode()
	testutil.Exec(t, pool, ins, emp, "PR", "SAIDA", nil, item, cliente, "PR830077")
	if got, _ := r.BeneficioFiscal(ctx, &item, &ncm, "PR", &cliente); got == nil || *got != "PR830077" {
		t.Fatalf("parâmetro do cliente tem de vencer o geral: %v", got)
	}
	outro := "84714900"
	if got, err := r.BeneficioFiscal(ctx, nil, &outro, "PR", nil); err != nil || got != nil {
		t.Fatalf("sem parâmetro: %v %v, quer nil", got, err)
	}
}

// Rejeição guarda o motivo; o novo envio limpa.
func TestRejeicaoDaNFeGuardaMotivo(t *testing.T) {
	pool := testutil.Pool(t)
	ctx, emp := empresaFiscal(t)
	var nota int64
	if err := pool.QueryRow(context.Background(), `INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,valor_total,status,created_by,enterprise_id)
		VALUES(1,$3::date,'5101','Venda',10,'AGUARDANDO_AUTORIZACAO',$1,$2) RETURNING id`, testutil.Actor(t, pool), emp, time.Now()).Scan(&nota); err != nil {
		t.Fatal(err)
	}
	r := fiscalpg.NewFiscalRepositoryPG(pool).(*fiscalpg.FiscalRepositoryPG)
	if err := r.RegistrarEnvioFocus(ctx, nota, "ref-1"); err != nil {
		t.Fatal(err)
	}
	if err := r.RegistrarRejeicao(ctx, nota, "930 — CST com benefício sem código"); err != nil {
		t.Fatal(err)
	}
	m, err := r.MotivoRejeicao(ctx, nota)
	if err != nil || m == nil || *m != "930 — CST com benefício sem código" {
		t.Fatalf("motivo = %v %v", m, err)
	}
	x, err := r.GetExitByID(ctx, nota)
	if err != nil || string(x.Status) != "REJECTED" || x.FocusRef == nil || *x.FocusRef != "ref-1" {
		t.Fatalf("nota = %+v %v, quer REJECTED com a referência", x, err)
	}
	if err := r.RegistrarEnvioFocus(ctx, nota, "ref-2"); err != nil {
		t.Fatal(err)
	}
	if m, _ := r.MotivoRejeicao(ctx, nota); m != nil {
		t.Fatalf("novo envio tem de limpar o motivo: %q", *m)
	}
}
