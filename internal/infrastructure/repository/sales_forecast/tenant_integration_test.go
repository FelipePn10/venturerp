//go:build integration

package sales_forecast_test

import (
	"context"
	"strconv"
	"testing"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	repository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_forecast"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

func forecastTenantContext(id, code int64, userID uuid.UUID) context.Context {
	user := &appsecurity.AuthUser{ID: userID.String(), EnterpriseID: id, EnterpriseCode: code}
	return context.WithValue(context.Background(), contextkey.UserKey, user)
}

func TestSalesForecastIsIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	userID := uuid.New()
	if _, err := pool.Exec(base, `INSERT INTO users(id,name,email,password) VALUES($1,'Forecast test',$2,'x')`, userID, userID.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	codeA := int64(1_800_000_000 + testutil.UniqueCode()%100_000_000)
	codeB := codeA + 1
	var enterpriseA, enterpriseB int64
	if err := pool.QueryRow(base, `INSERT INTO enterprise(code,name) VALUES($1,'Forecast A') RETURNING id`, codeA).Scan(&enterpriseA); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(base, `INSERT INTO enterprise(code,name) VALUES($1,'Forecast B') RETURNING id`, codeB).Scan(&enterpriseB); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(base, `DELETE FROM sales_forecasts WHERE created_by=$1`, userID)
		_, _ = pool.Exec(base, `DELETE FROM enterprise WHERE id=ANY($1)`, []int64{enterpriseA, enterpriseB})
		_, _ = pool.Exec(base, `DELETE FROM users WHERE id=$1`, userID)
	})

	repo := repository.NewSalesForecastRepositorySQLC(sqlc.New(pool), pool)
	forecastA, _ := entity.NewSalesForecast(991001, nil, 10, 2026, 12, userID)
	createdA, err := repo.CreateForecast(forecastTenantContext(enterpriseA, codeA, userID), forecastA)
	if err != nil {
		t.Fatal(err)
	}
	forecastB, _ := entity.NewSalesForecast(991001, nil, 10, 2026, 34, userID)
	if _, err = repo.CreateForecast(forecastTenantContext(enterpriseB, codeB, userID), forecastB); err != nil {
		t.Fatal(err)
	}
	listA, err := repo.ListForecasts(forecastTenantContext(enterpriseA, codeA, userID), 2026)
	if err != nil || len(listA) != 1 || listA[0].Quantity != 12 {
		t.Fatalf("previsão do tenant A incorreta: forecasts=%+v err=%v", listA, err)
	}
	if err = repo.DeleteForecast(forecastTenantContext(enterpriseB, codeB, userID), createdA.ID); err != nil {
		t.Fatal(err)
	}
	listA, err = repo.ListForecasts(forecastTenantContext(enterpriseA, codeA, userID), 2026)
	if err != nil || len(listA) != 1 {
		t.Fatalf("tenant B excluiu previsão do A: forecasts=%+v err=%v", listA, err)
	}
	var auditedTenant int64
	if err = pool.QueryRow(base, `SELECT enterprise_id FROM operational_mutation_audit WHERE table_name='sales_forecasts' AND row_key=$1 ORDER BY id DESC LIMIT 1`, strconv.FormatInt(createdA.ID, 10)).Scan(&auditedTenant); err != nil {
		t.Fatal(err)
	}
	if auditedTenant != enterpriseA {
		t.Fatalf("tenant da auditoria=%d, esperado=%d", auditedTenant, enterpriseA)
	}
}
