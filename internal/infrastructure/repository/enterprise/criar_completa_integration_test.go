//go:build integration

package enterprise_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	mapper "github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/enterprise"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/enterprise"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O cadastro de empresa (VEMP0100) cria, numa transação, a empresa com o
// código informado, o vínculo do criador como ADMIN e a identidade fiscal
// normalizada. CNPJ inválido é recusado antes de gravar.
func TestCadastroDeEmpresaComIdentidadeFiscal(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	code := int(1_700_000_000 + testutil.UniqueCode()%100_000_000)
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM fiscal_configs WHERE enterprise_id IN (SELECT id FROM enterprise WHERE code=$1)`, code)
		_, _ = pool.Exec(bg, `DELETE FROM user_enterprises WHERE enterprise_id IN (SELECT id FROM enterprise WHERE code=$1)`, code)
		_, _ = pool.Exec(bg, `DELETE FROM enterprise WHERE code=$1`, code)
	})
	repo := enterprise.NewRepositoryEnterpriseSQLC(sqlc.New(pool)).ComPool(pool)

	if _, err := mapper.ToEnterpriseEntity(request.CreateEnterpriseDTO{CreatedBy: actor, CNPJ: "123", RazaoSocial: "X", RegimeTributario: "3", UF: "PR"}); err == nil {
		t.Fatal("CNPJ curto deveria ser recusado")
	}
	e, err := mapper.ToEnterpriseEntity(request.CreateEnterpriseDTO{Code: code, CreatedBy: actor, CNPJ: "11.222.333/0001-81",
		RazaoSocial: " EMPRESA TESTE LTDA ", RegimeTributario: "1", UF: "sp", CEP: "01310-100", CodigoMunicipio: "3550308"})
	if err != nil {
		t.Fatal(err)
	}
	if e.Name != "EMPRESA TESTE LTDA" {
		t.Fatalf("nome cai na razão social: %q", e.Name)
	}
	criada, err := repo.Create(bg, e)
	if err != nil {
		t.Fatal(err)
	}
	var cnpj, uf, regime, cep, papel string
	if err := pool.QueryRow(bg, `SELECT f.cnpj_empresa, f.uf_empresa, f.regime_tributario, f.cep, ue.role
		FROM fiscal_configs f JOIN user_enterprises ue ON ue.enterprise_id=f.enterprise_id AND ue.user_id=$2
		WHERE f.enterprise_id=$1`, criada.ID, actor).Scan(&cnpj, &uf, &regime, &cep, &papel); err != nil {
		t.Fatal(err)
	}
	if cnpj != "11222333000181" || uf != "SP" || regime != "1" || cep != "01310100" || papel != "ADMIN" {
		t.Fatalf("identidade/vínculo: %s %s %s %s %s", cnpj, uf, regime, cep, papel)
	}
	// Mesmo código de novo: conflito, e nada fica pela metade.
	if _, err := repo.Create(bg, e); err == nil {
		t.Fatal("código repetido deveria ser conflito")
	}
	var n int
	_ = pool.QueryRow(bg, `SELECT count(*) FROM fiscal_configs WHERE enterprise_id IN (SELECT id FROM enterprise WHERE code=$1)`, code).Scan(&n)
	if n != 1 {
		t.Fatalf("uma identidade fiscal só: %d", n)
	}
}
