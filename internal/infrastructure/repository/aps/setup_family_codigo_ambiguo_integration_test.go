//go:build integration

package aps_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	apsrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/aps"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

// Agrupar o item fabricado (chave K) numa família de preparação não pode gravar
// a família também na chapa cujo código comercial é o texto "K". O UPDATE
// comparava a chave com business_code e atingia os dois.
func TestFamiliaDeSetupGravaSoNaChaveInterna(t *testing.T) {
	_, pool := testutil.Queries(t)
	base := context.Background()
	usuario := uuid.New()
	if _, err := pool.Exec(base, `INSERT INTO users(id,name,email,password) VALUES($1,'Família ambígua',$2,'x')`, usuario, usuario.String()+"@test.local"); err != nil {
		t.Fatal(err)
	}
	codigoEmpresa := int64(1_600_000_000) + testutil.UniqueCode()%10_000_000
	var empresa int64
	if err := pool.QueryRow(base, `INSERT INTO enterprise(code,name,created_by) VALUES($1,'Família ambígua',$2) RETURNING id`, codigoEmpresa, usuario).Scan(&empresa); err != nil {
		t.Fatal(err)
	}
	var almox int64
	if err := pool.QueryRow(base, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		VALUES($1,'Família ambígua',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`, fmt.Sprintf("W-%d", testutil.UniqueCode()), usuario, empresa).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(base, `DELETE FROM items WHERE enterprise_id=$1`, empresa)
		testutil.ApagarEmpresas(t, pool, empresa)
		_, _ = pool.Exec(base, `DELETE FROM warehouse WHERE id=$1`, almox)
		_, _ = pool.Exec(base, `DELETE FROM users WHERE id=$1`, usuario)
	})
	fabricado, chapa := testutil.UniqueCode(), testutil.UniqueCode()
	if _, err := pool.Exec(base, `INSERT INTO items(warehouse_code,code,name,created_by,enterprise_id,business_code,engineering_weight)
		VALUES ($1,$2,'BALANÇA',$4,$5,$6,'{"gross":1,"net":1,"unit":"KG"}'),
		       ($1,$3,'CHAPA 3MM',$4,$5,$7,'{"gross":1,"net":1,"unit":"KG"}')`,
		almox, fabricado, chapa, usuario, empresa, fmt.Sprintf("RN-%d", fabricado), strconv.FormatInt(fabricado, 10)); err != nil {
		t.Fatal(err)
	}

	ctx := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: empresa})
	repo, ok := apsrepo.New(sqlc.New(pool), pool).(interface {
		DefinirFamiliaDeSetup(context.Context, string, []int64) (int64, error)
	})
	if !ok {
		t.Fatal("repositório do APS sem DefinirFamiliaDeSetup")
	}
	n, err := repo.DefinirFamiliaDeSetup(ctx, "SOLDA MIG", []int64{fabricado})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("esperava 1 item agrupado, foram %d", n)
	}
	var familiaChapa *string
	if err := pool.QueryRow(base, `SELECT setup_family FROM items WHERE enterprise_id=$1 AND code=$2`, empresa, chapa).Scan(&familiaChapa); err != nil {
		t.Fatal(err)
	}
	if familiaChapa != nil && *familiaChapa != "" {
		t.Fatalf("a chapa não foi agrupada e recebeu a família %q", *familiaChapa)
	}
}
