//go:build integration

package sales_commission_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_commission/entity"
	commissionpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_commission"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/shopspring/decimal"
)

// O rateio é filho do orçamento/pedido, então usa `enterprise_code` — a mesma
// convenção do pai. Este teste prova a substituição atômica, o espelho do
// principal na capa e o filtro de empresa.
func TestRateioSubstituiEspelhaEIsolaEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := testutil.TenantContext(t, pool)
	actor := testutil.Actor(t, pool)
	repo := commissionpg.New(pool)

	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		t.Fatal(err)
	}

	repA, repB := testutil.UniqueCode(), testutil.UniqueCode()
	for _, code := range []int64{repA, repB} {
		if _, err := pool.Exec(ctx, `INSERT INTO representatives(code,name,document_number,is_active,blocked)
			VALUES($1,'Representante '||$2,$2,TRUE,FALSE)`, code, strconv.FormatInt(code, 10)); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO representative_enterprises(representative_code,enterprise_code,is_active)
			VALUES($1,$2,TRUE)`, code, enterpriseCode); err != nil {
			t.Fatal(err)
		}
	}

	numero := testutil.UniqueCode()
	var orcamento int64
	if err := pool.QueryRow(ctx, `INSERT INTO sales_quotations
		(quotation_number,enterprise_code,status,quotation_type,emission_date,digit_date,currency_code,
		 probability_pct,commission_pct,representative_code,created_by)
		VALUES($1,$2,'OV','VENDA',CURRENT_DATE,CURRENT_DATE,'BRL',50,0,$3,$4) RETURNING code`,
		numero, enterpriseCode, repA, actor).Scan(&orcamento); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM sales_quotation_representatives WHERE sales_quotation_code=$1`, orcamento)
		_, _ = pool.Exec(bg, `DELETE FROM sales_quotations WHERE code=$1`, orcamento)
		_, _ = pool.Exec(bg, `DELETE FROM representative_enterprises WHERE representative_code=ANY($1)`, []int64{repA, repB})
		_, _ = pool.Exec(bg, `DELETE FROM representatives WHERE code=ANY($1)`, []int64{repA, repB})
	})

	linhas := []*entity.Rateio{
		{RepresentativeCode: repA, Role: entity.PapelPrincipal, CommissionPct: decimal.NewFromInt(3), CommissionBase: entity.BaseTotalProdutos},
		{RepresentativeCode: repB, Role: entity.PapelParceiro, CommissionPct: decimal.RequireFromString("1.5"), CommissionBase: entity.BaseTotalLiquido},
	}
	gravadas, err := repo.Substituir(ctx, entity.DocumentoOrcamento, orcamento, linhas)
	if err != nil {
		t.Fatal(err)
	}
	if len(gravadas) != 2 || gravadas[0].Role != entity.PapelPrincipal {
		t.Fatalf("rateio gravado errado: %+v", gravadas)
	}
	if gravadas[0].RepresentativeName == "" {
		t.Fatal("o nome do representante não veio na leitura")
	}

	// O principal fica espelhado na capa: relatórios antigos leem de lá.
	var capaRep int64
	var capaPct decimal.Decimal
	if err := pool.QueryRow(ctx, `SELECT representative_code, commission_pct FROM sales_quotations WHERE code=$1`, orcamento).Scan(&capaRep, &capaPct); err != nil {
		t.Fatal(err)
	}
	if capaRep != repA || !capaPct.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("capa não espelhou o principal: rep=%d pct=%s", capaRep, capaPct)
	}

	// Gravar de novo com uma linha só substitui o rateio inteiro.
	segunda, err := repo.Substituir(ctx, entity.DocumentoOrcamento, orcamento, []*entity.Rateio{
		{RepresentativeCode: repB, Role: entity.PapelPrincipal, CommissionPct: decimal.NewFromInt(5), CommissionBase: entity.BaseTotalProdutos},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(segunda) != 1 || segunda[0].RepresentativeCode != repB {
		t.Fatalf("rateio não foi substituído: %+v", segunda)
	}

	// Totais vêm da capa e dos itens, e trazem o representante da capa para a
	// tela abrir preenchida.
	totais, err := repo.Totais(ctx, entity.DocumentoOrcamento, orcamento)
	if err != nil {
		t.Fatal(err)
	}
	if totais.RepresentativeCode == nil || *totais.RepresentativeCode != repB {
		t.Fatalf("a capa não devolveu o representante espelhado: %+v", totais.RepresentativeCode)
	}

	// Outra empresa não lê nem grava.
	outra := context.WithValue(context.Background(), contextkey.UserKey, &security.AuthUser{
		Role: "ADMIN", EnterpriseID: enterpriseCode + 90_000, EnterpriseCode: enterpriseCode + 90_000,
	})
	if lidas, err := repo.Listar(outra, entity.DocumentoOrcamento, orcamento); err != nil || len(lidas) != 0 {
		t.Fatalf("outra empresa leu %d linha(s) do rateio (err=%v)", len(lidas), err)
	}
	if _, err := repo.Substituir(outra, entity.DocumentoOrcamento, orcamento, linhas); err == nil {
		t.Fatal("outra empresa gravou o rateio de um orçamento que não é dela")
	}
	if _, err := repo.Totais(outra, entity.DocumentoOrcamento, orcamento); err == nil {
		t.Fatal("outra empresa leu os totais de um orçamento que não é dela")
	}
	// Representante de outra empresa não entra no rateio.
	if ativos, err := repo.RepresentantesAtivos(outra, []int64{repA, repB}); err != nil || len(ativos) != 0 {
		t.Fatalf("representantes de outra empresa foram aceitos: %v (err=%v)", ativos, err)
	}
}
