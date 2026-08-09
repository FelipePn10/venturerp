//go:build integration

package item_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	itemrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler"
	"github.com/go-chi/chi/v5"
)

type itemUpdateAuth struct{ ports.AuthService }

func (itemUpdateAuth) CanCreateItem(context.Context) bool { return true }

func TestItemCommercialAccountingRepositoryAndHTTPPartialUpdate(t *testing.T) {
	pool := testutil.Pool(t)
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	code := testutil.UniqueCode()
	_, err = tx.Exec(context.Background(), `INSERT INTO items (warehouse_code, code, name, nature, engineering_weight, created_by) VALUES (1,$1,'Integration item',2,'{"gross":1,"net":1,"unit":"KG"}'::jsonb,'00000000-0000-0000-0000-000000000001')`, code)
	if err != nil {
		t.Fatalf("create item fixture: %v", err)
	}
	repo := itemrepo.NewRepositoryItemSQLC(sqlc.New(tx))
	original, err := repo.FindItemByCode(context.Background(), valueobject.ItemCode(code))
	if err != nil {
		t.Fatal(err)
	}
	description := "PERSISTED COMMERCIAL DESCRIPTION"
	original.Commercial.Description = &description
	original.Commercial.WarrantyDays = 365
	if _, err = repo.UpdateCommercialAccounting(context.Background(), original); err != nil {
		t.Fatal(err)
	}

	uc := item_uc.NewUpdateItemUseCase(repo, itemUpdateAuth{})
	h := handler.NewCreateItemHandler(nil, uc, nil, nil, nil)
	router := chi.NewRouter()
	router.Put("/api/items/{code}", h.UpdateItem)
	req := httptest.NewRequest(http.MethodPut, "/api/items/"+strconv.FormatInt(int64(original.Code), 10), strings.NewReader(`{"commercial":{"warranty_days":730},"accounting":{"origin":0,"calculate_pis_cofins":false}}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status=%d body=%s", rec.Code, rec.Body.String())
	}
	got, err := repo.FindItemByCode(context.Background(), valueobject.ItemCode(code))
	if err != nil {
		t.Fatal(err)
	}
	if got.Commercial.WarrantyDays != 730 || got.Commercial.Description == nil || *got.Commercial.Description != description {
		t.Fatalf("partial update lost fields: %+v", got.Commercial)
	}
	if got.Accounting.Origin == nil || *got.Accounting.Origin != 0 || got.Accounting.CalculatePISCOFINS {
		t.Fatalf("accounting zero/false not persisted: %+v", got.Accounting)
	}

	if _, err = tx.Exec(context.Background(), "SAVEPOINT invalid_reference"); err != nil {
		t.Fatal(err)
	}
	badReq := httptest.NewRequest(http.MethodPut, "/api/items/"+strconv.FormatInt(int64(original.Code), 10), strings.NewReader(`{"commercial":{"transfer_warehouse_code":999999999}}`))
	badRec := httptest.NewRecorder()
	router.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid reference status=%d body=%s", badRec.Code, badRec.Body.String())
	}
	if _, err = tx.Exec(context.Background(), "ROLLBACK TO SAVEPOINT invalid_reference"); err != nil {
		t.Fatal(err)
	}
	afterRejected, err := repo.FindItemByCode(context.Background(), valueobject.ItemCode(code))
	if err != nil {
		t.Fatal(err)
	}
	if afterRejected.Commercial.TransferWarehouseCode != nil || afterRejected.Commercial.WarrantyDays != 730 {
		t.Fatalf("rejected update changed item: %+v", afterRejected.Commercial)
	}
}
