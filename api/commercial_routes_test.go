package main

import (
	"os"
	"strings"
	"testing"
)

func TestCommercialCanonicalRoutesArePublished(t *testing.T) {
	source, err := os.ReadFile("api.go")
	if err != nil {
		t.Fatalf("não foi possível ler api.go: %v", err)
	}
	apiSource := string(source)

	required := []string{
		`r.Route("/api/sales-order"`,
		`Get("/list", salesOrderHandler.List)`,
		`Get("/status/{status}", salesOrderHandler.ListByStatus)`,
		`Get("/{code}", salesOrderHandler.ListItems)`,
		`Post("/{code}/analyze", salesOrderHandler.Analyze)`,
		`Post("/{code}/attend", salesOrderHandler.Attend)`,
		`Post("/{code}/conference", salesOrderHandler.Confer)`,
		`Post("/{code}/delay-reason", salesOrderHandler.SaveDelayReason)`,
		`r.Route("/api/delivery-reschedule"`,
		`Post("/create", deliveryRescheduleHandler.Create)`,
		`Get("/list/{sales_order_code}", deliveryRescheduleHandler.ListByOrder)`,
		`Get("/preview/{sales_order_code}", deliveryRescheduleHandler.Preview)`,
		`Post("/batch", deliveryRescheduleHandler.CreateBatch)`,
		`Delete("/parameters", salesQuotationHandler.ResetParameters)`,
		`Patch("/commission-patterns/{code}/status", salesQuotationHandler.SetCommissionPatternStatus)`,
		`Patch("/cancellation-reasons/{code}/status", salesQuotationHandler.SetCancellationReasonStatus)`,
		`r.Route("/api/sales-quotation"`,
		`r.Route("/api/representatives"`,
		`Get("/sales-plans", representativeHandler.ListSalesPlans)`,
		`r.Route("/api/sales-goals"`,
		`r.Route("/api/consumer-service"`,
		`r.Route("/api/delivery-promise"`,
		`Get("/occupation", deliveryPromiseHandler.Occupation)`,
		`Post("/reschedule", deliveryPromiseHandler.Reschedule)`,
		`r.Route("/api/recurring-sales"`,
		`Post("/{code}/recalculate-adjustment", recurringSalesHandler.RecalculateAdjustment)`,
		`r.Route("/sales-tables"`,
		`Get("/resolve-by-item", customerHandler.ResolveSalesTablesForItem)`,
		`r.Route("/sales-price-policies"`,
		`r.Route("/commercial-policies"`,
		`Patch("/{code}/status", salesDivisionHandler.SetStatus)`,
	}
	for _, route := range required {
		if !strings.Contains(apiSource, route) {
			t.Errorf("rota comercial canônica ausente: %s", route)
		}
	}

	for _, obsolete := range []string{
		`Get("/invoiced", salesOrderHandler.`,
		`Get("/{code}/items", salesOrderHandler.`,
		`r.Route("/api/delivery-reschedules"`,
	} {
		if strings.Contains(apiSource, obsolete) {
			t.Errorf("alias obsoleto publicado: %s", obsolete)
		}
	}
}

func TestExpandedOperationalRoutesArePublished(t *testing.T) {
	source, err := os.ReadFile("api.go")
	if err != nil {
		t.Fatalf("não foi possível ler api.go: %v", err)
	}
	apiSource := string(source)

	required := []string{
		`r.Route("/api/notifications"`,
		`Get("/events", notificationHandler.ListEvents)`,
		`Get("/settings", notificationHandler.GetSettings)`,
		`Get("/subscriptions", notificationHandler.ListSubscriptions)`,
		`Post("/subscriptions", notificationHandler.CreateSubscription)`,
		`Get("/recipients/users", notificationHandler.ListEligibleUsers)`,
		`Get("/recipients/departments", notificationHandler.ListEligibleDepartments)`,
		`Post("/test-email", notificationHandler.TestEmail)`,
		`Get("/alerts", notificationHandler.ListAlerts)`,
		`Post("/deliveries/{id}/retry", notificationHandler.RetryDelivery)`,
		`r.Route("/api/stock/cycle-counts"`,
		`Get("/", notificationHandler.ListCycleCounts)`,
		`Post("/", notificationHandler.CreateCycleCount)`,
		`Post("/{id}/transition", notificationHandler.TransitionCycleCount)`,
		`r.Route("/api/production-order"`,
		`Post("/appointment", prodOrderHandler.AddAppointment)`,
		`Post("/consumption", prodOrderHandler.AddConsumption)`,
		`Post("/scanner/tokens", prodOrderHandler.CreateScanToken)`,
		`Post("/scanner/scan", prodOrderHandler.Scan)`,
		`r.Route("/api/item-suppliers"`,
		`Get("/search", itemSupplierHandler.SearchExternal)`,
		`Post("/{id}/quality-reports", itemSupplierHandler.CreateQualityReport)`,
		`Get("/{id}/quality-reports", itemSupplierHandler.ListQualityReports)`,
		`Get("/quality-reports/{reportID}/download", itemSupplierHandler.DownloadQualityReport)`,
		`r.Route("/api/procurement"`,
		`Post("/receiving-inspection-orders", procurementHandler.GenerateReceivingInspectionOrder)`,
		`Get("/receiving-inspection-orders", procurementHandler.ListReceivingInspectionOrders)`,
		`Post("/receiving-inspection-orders/{id}/results", procurementHandler.RecordReceivingInspectionResult)`,
		`Post("/receiving-inspection-orders/{id}/analysis", procurementHandler.AnalyzeReceivingInspectionOrder)`,
		`r.Route("/api/industrial-calendar"`,
		`r.Mount("/", industrialCalendarHandler.Routes())`,
	}
	for _, route := range required {
		if !strings.Contains(apiSource, route) {
			t.Errorf("rota operacional canônica ausente: %s", route)
		}
	}
}
