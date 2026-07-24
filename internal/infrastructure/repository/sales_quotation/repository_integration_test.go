//go:build integration

package sales_quotation_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	orderrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	quoteentity "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	quoterepo "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository"
	quotepg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_quotation"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func tenantContext(enterpriseID int64) context.Context {
	return context.WithValue(context.Background(), contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
}

func newQuotation(number, enterpriseID int64) *quoteentity.SalesQuotation {
	now := time.Now()
	return &quoteentity.SalesQuotation{
		QuotationNumber: number,
		EnterpriseCode:  enterpriseID,
		Status:          quoteentity.SalesQuotationStatusVentureBudget,
		QuotationType:   quoteentity.SalesQuotationTypeSale,
		EmissionDate:    now,
		DigitDate:       now,
		CurrencyCode:    "BRL",
		ReleaseStatus:   quoteentity.SalesQuotationReleaseOK,
		IsActive:        true,
		CreatedBy:       uuid.New(),
	}
}

func cleanupQuotationTest(t *testing.T, pool *pgxpool.Pool, enterpriseIDs ...int64) {
	t.Helper()
	for _, enterpriseID := range enterpriseIDs {
		testutil.Exec(t, pool, `DELETE FROM sales_order_items WHERE sales_order_code IN (SELECT code FROM sales_orders WHERE enterprise_code=$1)`, enterpriseID)
		testutil.Exec(t, pool, `DELETE FROM sales_orders WHERE enterprise_code=$1`, enterpriseID)
		testutil.Exec(t, pool, `DELETE FROM sales_quotations WHERE enterprise_code=$1`, enterpriseID)
		testutil.Exec(t, pool, `DELETE FROM sales_order_sequences WHERE enterprise_code=$1`, enterpriseID)
		testutil.Exec(t, pool, `DELETE FROM sales_quotation_sequences WHERE enterprise_code=$1`, enterpriseID)
	}
}

func TestRepositoryIsolatesQuotationsBetweenTenants(t *testing.T) {
	pool := testutil.Pool(t)
	repo := quotepg.New(pool)
	tenantA := testutil.UniqueCode()
	tenantB := testutil.UniqueCode()
	t.Cleanup(func() { cleanupQuotationTest(t, pool, tenantA, tenantB) })

	qa, err := repo.Create(tenantContext(tenantA), newQuotation(101, tenantA))
	if err != nil {
		t.Fatal(err)
	}
	qb, err := repo.Create(tenantContext(tenantB), newQuotation(202, tenantB))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := repo.GetByCode(tenantContext(tenantA), qb.Code); err == nil {
		t.Fatal("tenant A read tenant B quotation")
	}
	if _, err := repo.GetByCode(tenantContext(tenantB), qa.Code); err == nil {
		t.Fatal("tenant B read tenant A quotation")
	}
	listA, err := repo.List(tenantContext(tenantA), quoterepo.SalesQuotationFilter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, quotation := range listA {
		if quotation.EnterpriseCode != tenantA {
			t.Fatalf("tenant A list leaked enterprise %d", quotation.EnterpriseCode)
		}
	}
}

func createOrderCallback(ctx context.Context, enterpriseID int64, createdBy uuid.UUID) func(orderrepo.SalesOrderRepository) (*orderentity.SalesOrder, error) {
	return func(orders orderrepo.SalesOrderRepository) (*orderentity.SalesOrder, error) {
		number, err := orders.NextOrderNumber(ctx, enterpriseID)
		if err != nil {
			return nil, err
		}
		now := time.Now()
		return orders.Create(ctx, &orderentity.SalesOrder{
			OrderNumber:    number,
			EnterpriseCode: enterpriseID,
			Status:         orderentity.SalesOrderStatusOrder,
			Origin:         orderentity.SalesOrderOriginNormal,
			EmissionDate:   now,
			DigitDate:      now,
			CurrencyCode:   "BRL",
			CreatedBy:      createdBy,
		})
	}
}

func TestConversionUnitOfWorkRollsBackAndPreventsConcurrentDuplicate(t *testing.T) {
	pool := testutil.Pool(t)
	repo := quotepg.New(pool)
	uow := quotepg.NewConversionUnitOfWork(pool)
	tenantID := testutil.UniqueCode()
	ctx := tenantContext(tenantID)
	t.Cleanup(func() { cleanupQuotationTest(t, pool, tenantID) })

	rollbackQuote, err := repo.Create(ctx, newQuotation(301, tenantID))
	if err != nil {
		t.Fatal(err)
	}
	rollbackMarker := errors.New("force rollback")
	_, err = uow.Execute(ctx, rollbackQuote.Code, func(orders orderrepo.SalesOrderRepository) (*orderentity.SalesOrder, error) {
		created, createErr := createOrderCallback(ctx, tenantID, rollbackQuote.CreatedBy)(orders)
		if createErr != nil {
			return nil, createErr
		}
		return created, rollbackMarker
	})
	if !errors.Is(err, rollbackMarker) {
		t.Fatalf("expected rollback marker, got %v", err)
	}
	assertConversionCounts(t, pool, tenantID, rollbackQuote.Code, 0, 0)

	concurrentQuote, err := repo.Create(ctx, newQuotation(302, tenantID))
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for range 2 {
		go func() {
			ready.Done()
			<-start
			_, executeErr := uow.Execute(ctx, concurrentQuote.Code, createOrderCallback(ctx, tenantID, concurrentQuote.CreatedBy))
			results <- executeErr
		}()
	}
	ready.Wait()
	close(start)
	var successes int
	for range 2 {
		if executeErr := <-results; executeErr == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent conversion successes=%d, want 1", successes)
	}
	assertConversionCounts(t, pool, tenantID, concurrentQuote.Code, 1, 1)
}

func assertConversionCounts(t *testing.T, pool *pgxpool.Pool, tenantID, quotationCode int64, wantOrders, wantEvents int) {
	t.Helper()
	var orders, events int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM sales_orders WHERE enterprise_code=$1`, tenantID).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM sales_quotation_events WHERE sales_quotation_code=$1 AND event_type='CONVERT'`, quotationCode).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if orders != wantOrders || events != wantEvents {
		t.Fatal(fmt.Sprintf("conversion partial state: orders=%d events=%d, want %d/%d", orders, events, wantOrders, wantEvents))
	}
}
