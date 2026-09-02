//go:build integration

package fiscal_test

import (
	"context"
	"testing"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	repository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

func fiscalTenantContext(id int64) context.Context {
	return context.WithValue(context.Background(), contextkey.UserKey, &appsecurity.AuthUser{EnterpriseID: id})
}

func TestFiscalConfigAndBrandingAreIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	codeA := int64(1_600_000_000 + testutil.UniqueCode()%100_000_000)
	codeB := codeA + 1
	var a, b int64
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name) VALUES($1,'Fiscal A') RETURNING id`, codeA).Scan(&a); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name) VALUES($1,'Fiscal B') RETURNING id`, codeB).Scan(&b); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM fiscal_configs WHERE enterprise_id=ANY($1)`, []int64{a, b})
		_, _ = pool.Exec(ctx, `DELETE FROM enterprise WHERE id=ANY($1)`, []int64{a, b})
	})
	repo := repository.NewFiscalRepositoryPG(pool)
	actor := uuid.New()
	base := func(name, email string) *entity.FiscalConfig {
		return &entity.FiscalConfig{CnpjEmpresa: "12345678000199", RazaoSocial: name, TradeName: "Fantasia " + name, Email: email, RegimeTributario: "lucro_real", UFEmpresa: "PR", FocusNfeAmbiente: "homologacao", UpdatedBy: actor}
	}
	if _, err := repo.UpdateFiscalConfig(fiscalTenantContext(a), base("Empresa A", "a@example.test")); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateFiscalConfig(fiscalTenantContext(b), base("Empresa B", "b@example.test")); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetBranding(fiscalTenantContext(a), nil, "", "#112233", actor); err != nil {
		t.Fatal(err)
	}
	gotA, err := repo.GetFiscalConfig(fiscalTenantContext(a))
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := repo.GetFiscalConfig(fiscalTenantContext(b))
	if err != nil {
		t.Fatal(err)
	}
	if gotA.RazaoSocial != "Empresa A" || gotA.Email != "a@example.test" || gotA.BrandColor == nil || *gotA.BrandColor != "#112233" {
		t.Fatalf("tenant A incorreto: %+v", gotA)
	}
	if gotB.RazaoSocial != "Empresa B" || gotB.Email != "b@example.test" || gotB.BrandColor != nil {
		t.Fatalf("tenant B recebeu dados do A: %+v", gotB)
	}
}
