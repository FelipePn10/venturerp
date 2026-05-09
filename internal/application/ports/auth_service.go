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
}
