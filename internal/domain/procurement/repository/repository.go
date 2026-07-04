package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/procurement/entity"
	"github.com/google/uuid"
)

type Repository interface {
	CreateRecord(ctx context.Context, rec *entity.Record) (*entity.Record, error)
	GetRecord(ctx context.Context, id int64) (*entity.Record, error)
	ListRecords(ctx context.Context, recordType, status string) ([]*entity.Record, error)
	UpdateRecordStatus(ctx context.Context, id int64, status entity.RecordStatus) (*entity.Record, error)
	CreateInspectionDisposition(ctx context.Context, d *entity.InspectionDisposition) (*entity.InspectionDisposition, error)
	CreateSupplierScorecard(ctx context.Context, s *entity.SupplierScorecard) (*entity.SupplierScorecard, error)
	ListSupplierScorecards(ctx context.Context, supplierCode int64) ([]*entity.SupplierScorecard, error)
	CreateReceivingInspectionRoute(ctx context.Context, route *entity.ReceivingInspectionRoute) (*entity.ReceivingInspectionRoute, error)
	GetReceivingInspectionRoute(ctx context.Context, id int64) (*entity.ReceivingInspectionRoute, error)
	FindReceivingInspectionRoute(ctx context.Context, enterpriseCode int64, itemCode int64, mask string, classificationCode *string) (*entity.ReceivingInspectionRoute, error)
	CreateReceivingInspectionOrder(ctx context.Context, order *entity.ReceivingInspectionOrder) (*entity.ReceivingInspectionOrder, error)
	GetReceivingInspectionOrder(ctx context.Context, id int64) (*entity.ReceivingInspectionOrder, error)
	ListReceivingInspectionOrders(ctx context.Context, status string) ([]*entity.ReceivingInspectionOrder, error)
	CreateReceivingInspectionResult(ctx context.Context, result *entity.ReceivingInspectionResult) (*entity.ReceivingInspectionResult, error)
	CreateReceivingInspectionAnalysis(ctx context.Context, analysis *entity.ReceivingInspectionAnalysis) (*entity.ReceivingInspectionAnalysis, error)

	// Approval limits (alçada de valores).
	CreateApprovalLimit(ctx context.Context, limit *entity.ApprovalLimit) (*entity.ApprovalLimit, error)
	ListApprovalLimits(ctx context.Context, enterpriseCode int64) ([]*entity.ApprovalLimit, error)
	// FindApprovalLimit resolves the most specific active rule for the given scope
	// keys (supplier / cost center / category), falling back to GLOBAL.
	FindApprovalLimit(ctx context.Context, enterpriseCode int64, supplierRef, costCenterRef, categoryRef *string) (*entity.ApprovalLimit, error)

	// Supplier contracts.
	CreateSupplierContract(ctx context.Context, c *entity.SupplierContract) (*entity.SupplierContract, error)
	GetSupplierContract(ctx context.Context, id int64) (*entity.SupplierContract, error)
	ListSupplierContracts(ctx context.Context, supplierCode int64, status string) ([]*entity.SupplierContract, error)
	UpdateSupplierContractStatus(ctx context.Context, id int64, status string) (*entity.SupplierContract, error)
	FindContractItem(ctx context.Context, contractID, itemCode int64, mask string) (*entity.SupplierContractItem, error)
	ConsumeContractItem(ctx context.Context, contractItemID int64, qty float64) (*entity.SupplierContractItem, error)

	// Supplier performance for IQF auto-computation.
	AggregateSupplierPerformance(ctx context.Context, supplierCode int64, from, to time.Time) (*entity.SupplierPerformanceAggregate, error)

	// Consolidated purchase movement history (buyer/supplier performance consult).
	ListPurchaseMovementHistory(ctx context.Context, supplierCode *int64, itemCode *int64, limit int) ([]*entity.PurchaseMovementHistoryRow, error)

	// Receiving notice + divergences (FAVR).
	CreateReceivingNotice(ctx context.Context, n *entity.ReceivingNotice) (*entity.ReceivingNotice, error)
	GetReceivingNotice(ctx context.Context, id int64) (*entity.ReceivingNotice, error)
	ListReceivingNotices(ctx context.Context, status string) ([]*entity.ReceivingNotice, error)
	UpdateReceivingNoticeStatus(ctx context.Context, id int64, status string, blocked bool) (*entity.ReceivingNotice, error)
	CreateReceivingDivergence(ctx context.Context, d *entity.ReceivingDivergence) (*entity.ReceivingDivergence, error)
	ListReceivingDivergences(ctx context.Context, supplierCode *int64, resolution string) ([]*entity.ReceivingDivergence, error)
	ResolveReceivingDivergence(ctx context.Context, id int64, resolution string) (*entity.ReceivingDivergence, error)

	// Supplier EDI (FEDS).
	CreateEDIMessage(ctx context.Context, m *entity.SupplierEDIMessage) (*entity.SupplierEDIMessage, error)
	GetEDIMessage(ctx context.Context, id int64) (*entity.SupplierEDIMessage, error)
	ListEDIMessages(ctx context.Context, supplierCode *int64) ([]*entity.SupplierEDIMessage, error)

	// Import processes (FREC0203 / FIMP).
	CreateImportProcess(ctx context.Context, p *entity.ImportProcess) (*entity.ImportProcess, error)
	GetImportProcess(ctx context.Context, id int64) (*entity.ImportProcess, error)
	ListImportProcesses(ctx context.Context, status string) ([]*entity.ImportProcess, error)
	UpdateImportItemCosts(ctx context.Context, items []*entity.ImportProcessItem) error
	UpdateImportProcessStatus(ctx context.Context, id int64, status string) (*entity.ImportProcess, error)

	// Procurement parameters (FUTL0125).
	UpsertParameter(ctx context.Context, p *entity.ProcurementParameter) (*entity.ProcurementParameter, error)
	ListParameters(ctx context.Context, enterpriseCode int64, domain string) ([]*entity.ProcurementParameter, error)

	// Supplier homologation (FAVF0203).
	CreateHomologation(ctx context.Context, h *entity.SupplierHomologation) (*entity.SupplierHomologation, error)
	ListHomologations(ctx context.Context, supplierCode int64) ([]*entity.SupplierHomologation, error)

	// Item-supplier generation from purchase history (FFOR0204).
	GenerateItemSuppliersFromHistory(ctx context.Context, supplierCode int64, actor uuid.UUID) (int, error)
}
