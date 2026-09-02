package queries_test

import (
	"os"
	"strings"
	"testing"
)

func TestSalesOrderQueriesDeclareTenantScope(t *testing.T) {
	raw, err := os.ReadFile("sales_order.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	for _, operation := range []string{
		"GetSalesOrderByCode", "ListSalesOrders", "ListSalesOrdersByCustomer",
		"ListSalesOrdersByStatus", "ListSalesOrdersByDateRange", "ListSalesOrdersAdvanced",
		"SalesOrderReport", "CancelSalesOrder", "BlockSalesOrder", "UnblockSalesOrder",
		"ChangeSalesOrderStatus", "AnalyzeSalesOrder", "ReleaseSalesOrder", "AttendSalesOrder",
		"ConferSalesOrder", "SaveSalesOrderDelayReason", "CreateSalesOrderItem",
		"UpdateSalesOrderItem", "GetSalesOrderItem", "ListSalesOrderItems", "CancelSalesOrderItem",
	} {
		start := strings.Index(sql, "-- name: "+operation+" ")
		if start < 0 {
			t.Fatalf("query %s não encontrada", operation)
		}
		section := sql[start:]
		if next := strings.Index(section[1:], "-- name: "); next >= 0 {
			section = section[:next+1]
		}
		if !strings.Contains(section, "enterprise_code") {
			t.Errorf("query %s não declara isolamento por empresa", operation)
		}
	}
}
