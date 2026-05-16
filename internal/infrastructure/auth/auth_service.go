package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

type AuthService struct{}

func (a *AuthService) hasWriteRole(ctx context.Context) bool {
	user, ok := ctx.Value(contextkey.UserKey).(*security.AuthUser)
	if !ok {
		return false
	}

	role := strings.ToUpper(strings.TrimSpace(user.Role))
	return role == "ADMIN" || role == "USER"
}

func (a *AuthService) CanCreateComponent(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateItem(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateProduct(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateBom(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateBomItems(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanAssociateByQuestionProduct(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateQuestion(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateQuestionOption(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanDeleteProduct(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateWarehouse(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateGroup(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateEnterprise(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateModifier(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateEmployee(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGenerateMaskForItem(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateStructure(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) UpdateStructure(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) GetStructureTree(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) GetAllStructure(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) ResolveStructureForMask(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanResolveStructure(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) FindItemByCode(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) UserID(ctx context.Context) (uuid.UUID, error) {
	user, ok := ctx.Value(contextkey.UserKey).(*security.AuthUser)
	if !ok {
		return uuid.Nil, errors.New("unauthenticated request")
	}
	id, err := uuid.Parse(user.ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user id in context: %w", err)
	}
	return id, nil
}

func (a *AuthService) CreateAllocation(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) ListAllocation(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateCostCenter(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListCostCenter(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetCostCenter(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateDeliveryReschedule(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListDeliveryReschedule(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateIndependentDemand(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListIndependentDemand(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanViewIndependentDemand(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanUpdateIndependentDemand(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanDeleteIndependentDemand(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanManageIndustrialCalendar(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanManageItemCalendarPromise(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateItemTimeMachine(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateMachine(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateType(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListItemMachineTimes(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListMachines(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListTypes(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanSchedule(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanUpdateMachineType(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetMachineType(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanDeleteMachineType(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanUpdateMachine(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetMachine(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListByType(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanDeleteMachine(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetItemMachineTime(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) ListByMachine(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanRunMRPCalculation(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanConfiguredRulesMRP(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateOrderPriority(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanOrderPriority(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateOverheadAllocation(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListOverheadAllocation(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreatePlannedOrder(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanReleaseOrder(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListOrder(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateSalesDivision(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListSalesDivisions(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetSalesDivision(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanUpdateSalesDivision(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanDeleteSalesDivision(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateSalesForecast(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListSalesForecasts(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateForecastBlock(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListForecastBlocks(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateAppropriationTable(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListAppropriationTables(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanManagePlanningParams(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateProductionPlan(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListProductionPlans(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanUpdateProductionPlan(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanDeleteProductionPlan(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateRestriction(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListRestrictions(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetRestriction(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanUpdateRestriction(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanDeactivateRestriction(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListMRPExceptions(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanDeleteSalesDivisionRecord(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateSalesOrder(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanUpdateSalesOrder(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetSalesOrder(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListSalesOrders(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateStockMovement(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListStockMovements(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetStockBalance(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanReserveStock(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanReleaseReservation(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanConsumeReservation(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateInventory(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetInventory(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListInventories(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCountInventoryItem(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanAdjustInventory(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCloseInventory(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreatePurchaseOrder(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanUpdatePurchaseOrder(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetPurchaseOrder(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListPurchaseOrders(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

// Financial - Contas Bancarias
func (a *AuthService) CanCreateContaBancaria(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListContasBancarias(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

// Financial - Condicoes Pagamento
func (a *AuthService) CanCreateCondicaoPagamento(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListCondicoesPagamento(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

// Financial - Plano de Contas
func (a *AuthService) CanCreatePlanoContas(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListPlanoContas(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

// Financial - Centros de Custo
func (a *AuthService) CanCreateCentroCustoFinancial(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListCentrosCustoFinancial(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

// Financial - Contas a Pagar
func (a *AuthService) CanCreateContaPagar(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetContaPagar(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListContasPagar(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanApproveContaPagar(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanBaixarContaPagar(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCancelContaPagar(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetAgingPagar(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

// Financial - Contas a Receber
func (a *AuthService) CanCreateContaReceber(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetContaReceber(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListContasReceber(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanBaixarContaReceber(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCancelContaReceber(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetAgingReceber(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

// Financial - Fluxo Caixa
func (a *AuthService) CanGetFluxoCaixa(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetFluxoProjetado(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetSaldoContas(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

// Financial - Tax Assessment
func (a *AuthService) CanApurarImpostos(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetTaxAssessment(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListTaxAssessments(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateFiscalEntry(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanApproveFiscalEntry(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateFiscalExit(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanAuthorizeFiscalExit(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanManageFiscalConfig(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListFiscalEntries(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetFiscalEntry(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCancelFiscalExit(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListFiscalExits(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetFiscalExit(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}
