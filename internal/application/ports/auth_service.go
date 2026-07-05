package ports

import (
	"context"

	"github.com/google/uuid"
)

type AuthService interface {
	CanCreateComponent(ctx context.Context) bool
	CanCreateProduct(ctx context.Context) bool
	CanCreateBom(ctx context.Context) bool
	CanCreateBomItems(ctx context.Context) bool
	CanAssociateByQuestionProduct(ctx context.Context) bool
	CanCreateQuestion(ctx context.Context) bool
	CanCreateQuestionOption(ctx context.Context) bool
	CanDeleteProduct(ctx context.Context) bool
	CanCreateItem(ctx context.Context) bool
	CanCreateWarehouse(ctx context.Context) bool
	CanCreateGroup(ctx context.Context) bool
	CanCreateEnterprise(ctx context.Context) bool
	CanCreateModifier(ctx context.Context) bool
	CanCreateEmployee(ctx context.Context) bool
	CanGenerateMaskForItem(ctx context.Context) bool
	CanCreateStructure(ctx context.Context) bool
	UpdateStructure(ctx context.Context) bool
	GetStructureTree(ctx context.Context) bool
	GetAllStructure(ctx context.Context) bool
	ResolveStructureForMask(ctx context.Context) bool
	FindItemByCode(ctx context.Context) bool
	CanResolveStructure(ctx context.Context) bool
	UserID(ctx context.Context) (uuid.UUID, error)
	CreateAllocation(ctx context.Context) bool
	ListAllocation(ctx context.Context) bool
	CanCreateCostCenter(ctx context.Context) bool
	CanListCostCenter(ctx context.Context) bool
	CanGetCostCenter(ctx context.Context) bool
	CanCreateDeliveryReschedule(ctx context.Context) bool
	CanListDeliveryReschedule(ctx context.Context) bool
	CanManageDeliveryPromise(ctx context.Context) bool
	CanCreateIndependentDemand(ctx context.Context) bool
	CanListIndependentDemand(ctx context.Context) bool
	CanViewIndependentDemand(ctx context.Context) bool
	CanUpdateIndependentDemand(ctx context.Context) bool
	CanDeleteIndependentDemand(ctx context.Context) bool
	CanManageIndustrialCalendar(ctx context.Context) bool
	CanManageItemCalendarPromise(ctx context.Context) bool
	CanCreateItemTimeMachine(ctx context.Context) bool
	CanCreateMachine(ctx context.Context) bool
	CanCreateType(ctx context.Context) bool
	CanListItemMachineTimes(ctx context.Context) bool
	CanListMachines(ctx context.Context) bool
	CanListTypes(ctx context.Context) bool
	CanSchedule(ctx context.Context) bool
	CanUpdateMachineType(ctx context.Context) bool
	CanGetMachineType(ctx context.Context) bool
	CanDeleteMachineType(ctx context.Context) bool
	CanUpdateMachine(ctx context.Context) bool
	CanGetMachine(ctx context.Context) bool
	CanListByType(ctx context.Context) bool
	CanDeleteMachine(ctx context.Context) bool
	CanGetItemMachineTime(ctx context.Context) bool
	ListByMachine(ctx context.Context) bool
	CanRunMRPCalculation(ctx context.Context) bool
	CanConfiguredRulesMRP(ctx context.Context) bool
	CanCreateOrderPriority(ctx context.Context) bool
	CanOrderPriority(ctx context.Context) bool
	CanCreateOverheadAllocation(ctx context.Context) bool
	CanListOverheadAllocation(ctx context.Context) bool
	CanCreatePlannedOrder(ctx context.Context) bool
	CanReleaseOrder(ctx context.Context) bool
	CanListOrder(ctx context.Context) bool
	CanCreateSalesDivision(ctx context.Context) bool
	CanListSalesDivisions(ctx context.Context) bool
	CanGetSalesDivision(ctx context.Context) bool
	CanUpdateSalesDivision(ctx context.Context) bool
	CanDeleteSalesDivision(ctx context.Context) bool
	CanCreateSalesForecast(ctx context.Context) bool
	CanListSalesForecasts(ctx context.Context) bool
	CanCreateForecastBlock(ctx context.Context) bool
	CanListForecastBlocks(ctx context.Context) bool
	CanCreateAppropriationTable(ctx context.Context) bool
	CanListAppropriationTables(ctx context.Context) bool
	CanManageTechnicalAssistance(ctx context.Context) bool
	CanManagePlanningParams(ctx context.Context) bool
	CanCreateProductionPlan(ctx context.Context) bool
	CanListProductionPlans(ctx context.Context) bool
	CanUpdateProductionPlan(ctx context.Context) bool
	CanDeleteProductionPlan(ctx context.Context) bool
	CanCreateRestriction(ctx context.Context) bool
	CanListRestrictions(ctx context.Context) bool
	CanGetRestriction(ctx context.Context) bool
	CanUpdateRestriction(ctx context.Context) bool
	CanDeactivateRestriction(ctx context.Context) bool
	CanListMRPExceptions(ctx context.Context) bool
	CanDeleteSalesDivisionRecord(ctx context.Context) bool
	CanCreateSalesOrder(ctx context.Context) bool
	CanUpdateSalesOrder(ctx context.Context) bool
	CanGetSalesOrder(ctx context.Context) bool
	CanListSalesOrders(ctx context.Context) bool
	CanCreateStockMovement(ctx context.Context) bool
	CanListStockMovements(ctx context.Context) bool
	CanGetStockBalance(ctx context.Context) bool
	CanReserveStock(ctx context.Context) bool
	CanReleaseReservation(ctx context.Context) bool
	CanConsumeReservation(ctx context.Context) bool
	CanCreateInventory(ctx context.Context) bool
	CanGetInventory(ctx context.Context) bool
	CanListInventories(ctx context.Context) bool
	CanCountInventoryItem(ctx context.Context) bool
	CanAdjustInventory(ctx context.Context) bool
	CanCloseInventory(ctx context.Context) bool
	CanCreatePurchaseOrder(ctx context.Context) bool
	CanUpdatePurchaseOrder(ctx context.Context) bool
	CanGetPurchaseOrder(ctx context.Context) bool
	CanListPurchaseOrders(ctx context.Context) bool

	// Financial - Contas Bancarias
	CanCreateContaBancaria(ctx context.Context) bool
	CanListContasBancarias(ctx context.Context) bool

	// Financial - Condicoes Pagamento
	CanCreateCondicaoPagamento(ctx context.Context) bool
	CanListCondicoesPagamento(ctx context.Context) bool

	// Financial - Plano de Contas
	CanCreatePlanoContas(ctx context.Context) bool
	CanListPlanoContas(ctx context.Context) bool

	// Financial - Centros de Custo
	CanCreateCentroCustoFinancial(ctx context.Context) bool
	CanListCentrosCustoFinancial(ctx context.Context) bool

	// Financial - Contas a Pagar
	CanCreateContaPagar(ctx context.Context) bool
	CanGetContaPagar(ctx context.Context) bool
	CanListContasPagar(ctx context.Context) bool
	CanApproveContaPagar(ctx context.Context) bool
	CanBaixarContaPagar(ctx context.Context) bool
	CanCancelContaPagar(ctx context.Context) bool
	CanGetAgingPagar(ctx context.Context) bool

	// Financial - Contas a Receber
	CanCreateContaReceber(ctx context.Context) bool
	CanGetContaReceber(ctx context.Context) bool
	CanListContasReceber(ctx context.Context) bool
	CanBaixarContaReceber(ctx context.Context) bool
	CanCancelContaReceber(ctx context.Context) bool
	CanGetAgingReceber(ctx context.Context) bool

	// Financial - Fluxo Caixa
	CanGetFluxoCaixa(ctx context.Context) bool
	CanGetFluxoProjetado(ctx context.Context) bool
	CanGetSaldoContas(ctx context.Context) bool

	// Financial - Tax Assessment
	CanApurarImpostos(ctx context.Context) bool
	CanGetTaxAssessment(ctx context.Context) bool
	CanListTaxAssessments(ctx context.Context) bool

	// Fiscal - Entries
	CanCreateFiscalEntry(ctx context.Context) bool
	CanApproveFiscalEntry(ctx context.Context) bool
	CanListFiscalEntries(ctx context.Context) bool
	CanGetFiscalEntry(ctx context.Context) bool

	// Fiscal - Exits
	CanCreateFiscalExit(ctx context.Context) bool
	CanAuthorizeFiscalExit(ctx context.Context) bool
	CanCancelFiscalExit(ctx context.Context) bool
	CanListFiscalExits(ctx context.Context) bool
	CanGetFiscalExit(ctx context.Context) bool

	// Fiscal - Config
	CanManageFiscalConfig(ctx context.Context) bool

	// Reports & Conciliation
	CanExportRelatorios(ctx context.Context) bool
	CanImportarOFX(ctx context.Context) bool
}
