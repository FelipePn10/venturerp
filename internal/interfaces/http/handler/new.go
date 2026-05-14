package handler

import (
	"github.com/FelipePn10/panossoerp/internal/application/usecase"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/allocation_base_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/cost_center_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_promise_params_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_reschedule_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/employee"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/independent_demand_uc"
	industrial_calendar_uc "github.com/FelipePn10/panossoerp/internal/application/usecase/industrial_calendar"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_calendar_promise_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/machine_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_calculation_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/order_priority_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/overhead_allocation_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planned_order_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planning_params_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/production_plan_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/restriction_uc"
)

func NewCreateProductHandler(
	createProductUC *usecase.CreateProductUseCase,
) *ProductHandler {
	return &ProductHandler{
		createProductUC: createProductUC,
	}
}

func NewDeleteProductHandler(
	deleteProductUC *usecase.DeleteProductUseCase,
) *ProductHandler {
	return &ProductHandler{
		deleteProductUC: deleteProductUC,
	}
}

func NewFindItemCodeHandler(
	findItemByCodeUC *usecase.FindItemByCode,
) *ItemHandler {
	return &ItemHandler{
		findItemByCodeUC: findItemByCodeUC,
	}
}

func NewFindQuestionByName(
	findQuestionByNameUC *usecase.FindQuestionByName,
) *QuestionHandler {
	return &QuestionHandler{
		findQuestionByNameUC: findQuestionByNameUC,
	}
}

func NewUserHandler(
	registerUC *usecase.RegisterUserUseCase,
	loginUC *usecase.LoginUserUseCase,
	jwtSecret string,
) *UserHandler {
	return &UserHandler{
		registerUC: registerUC,
		loginUC:    loginUC,
		jwtSecret:  jwtSecret,
	}
}

func NewQuestionHandler(
	createQuestionUC *usecase.CreateQuestion,
) *QuestionHandler {
	return &QuestionHandler{
		createQuestionUC: createQuestionUC,
	}
}

func NewDeleteQuestionHandler(
	deleteQuestionUC *usecase.DeleteQuestionUseCase,
) *QuestionHandler {
	return &QuestionHandler{
		deleteQuestionUC: deleteQuestionUC,
	}
}

func NewCreateQuestionOptionHandler(
	createQuestionOptionUC *usecase.CreateQuestionOptionUseCase,
) *QuestionOptionHandler {
	return &QuestionOptionHandler{
		createQuestionOptionUC: createQuestionOptionUC,
	}
}

func NewDeleteQuestionOptionHandler(
	deleteQuestionOptionUC *usecase.DeleteQuestionOptionUseCase,
) *QuestionOptionHandler {
	return &QuestionOptionHandler{
		deleteQuestionOptionUC: deleteQuestionOptionUC,
	}
}

func NewAssociateByQuestionItemHandler(
	associateByQuestionProductUC *usecase.AssociateByQuestionItemUseCase,
) *AssociateByQuestionItemHandler {
	return &AssociateByQuestionItemHandler{
		associateByQuestionProductUC: associateByQuestionProductUC,
	}
}

func NewGeneratMaskItemHandler(
	generateMaskProductUC *usecase.GenerateMaskForItemUseCase,
) *GenerateMaskHandler {
	return &GenerateMaskHandler{
		generateMask: generateMaskProductUC,
	}
}

func NewCreateBomHandler(
	createBomUC *usecase.CreateBomUseCase,
) *BomHandler {
	return &BomHandler{
		createBomUC: createBomUC,
	}
}

func NewCreateBomItemHandler(
	createBomItemUC *usecase.CreateBomItemUseCase,
) *BomItemHandler {
	return &BomItemHandler{
		createBomItemUC: createBomItemUC,
	}
}

func NewCreateItemHandler(
	createItemUc *usecase.CreateItemUseCase,
	findItemByCodeUc *usecase.FindItemByCode,
) *ItemHandler {
	return &ItemHandler{
		createItemUC:     createItemUc,
		findItemByCodeUC: findItemByCodeUc,
	}
}

func NewCreateWarehouseHandler(
	createWarehouse *usecase.CreateWarehouseUseCase,
) *WarehouseHandler {
	return &WarehouseHandler{
		createWarehouseUC: createWarehouse,
	}
}

func NewCreateGroupHandler(
	createGroupUc *usecase.CreateGroupUseCase,
) *GroupHandler {
	return &GroupHandler{
		createGroupUC: createGroupUc,
	}
}

func NewCreateEnterpriseHandler(
	createEnterprisepUc *usecase.CreateEnterpriseUseCase,
) *EnterpriseHandler {
	return &EnterpriseHandler{
		createEnterpriseUC: createEnterprisepUc,
	}
}

func NewCreateModifierHandler(
	createModifierUc *usecase.CreateModifierUseCase,
) *ModifierHandler {
	return &ModifierHandler{
		createModifierUC: createModifierUc,
	}
}

func NewEmployeeHandler(
	createUC *employee.CreateEmployeeUseCase,
	listUC *employee.ListEmployeesUseCase,
	getUC *employee.GetEmployeeUseCase,
	updateUC *employee.UpdateEmployeeUseCase,
	deactivateUC *employee.DeactivateEmployeeUseCase,
) *EmployeeHandler {
	return &EmployeeHandler{
		createUC:     createUC,
		listUC:       listUC,
		getUC:        getUC,
		updateUC:     updateUC,
		deactivateUC: deactivateUC,
	}
}

func NewItemStructureHandler(
	createUC *usecase.CreateStructureComponentUseCase,
	updateUC *usecase.UpdateStructureComponentUseCase,
	getAllStructureUC *usecase.GetAllDirectChildrenUseCase,
	treeUC *usecase.GetStructureTreeUseCase,
	// deleteUC *usecase.DeleteStructureComponentUseCase,
) *ItemStructureHandler {
	return &ItemStructureHandler{
		createUC:        createUC,
		updateUC:        updateUC,
		getAllStructure: getAllStructureUC,
		treeUC:          treeUC,
		//deleteUC:  deleteUC,
	}
}

func NewQueryStructureHandler(
	resolveUc *usecase.ResolveStructureQueryUseCase,
) *ItemQueryStructureHandler {
	return &ItemQueryStructureHandler{
		resolveUC: resolveUc,
	}
}

func NewAllocationBaseHandler(
	createUC *allocation_base_uc.CreateAllocationBaseUseCase,
	listUC *allocation_base_uc.ListAllocationBasesUseCase,
) *AllocationBaseHandler {
	return &AllocationBaseHandler{
		createUC: createUC,
		listUC:   listUC,
	}
}

func NewCostCenterHandler(
	createUC *cost_center_uc.CreateCostCenterUseCase,
	listUC *cost_center_uc.ListCostCentersUseCase,
	getUC *cost_center_uc.GetCostCenterUseCase,
) *CostCenterHandler {
	return &CostCenterHandler{
		createUC: createUC,
		listUC:   listUC,
		getUC:    getUC,
	}
}

func NewDeliveryPromiseParamsHandler(
	uc *delivery_promise_params_uc.ManageDeliveryPromiseParamsUseCase,
) *DeliveryPromiseParamsHandler {
	return &DeliveryPromiseParamsHandler{
		uc: uc,
	}
}

func NewDeliveryRescheduleHandler(
	createUC *delivery_reschedule_uc.CreateDeliveryRescheduleUseCase,
	listUC *delivery_reschedule_uc.ListDeliveryReschedulesUseCase,
) *DeliveryRescheduleHandler {
	return &DeliveryRescheduleHandler{
		createUC: createUC,
		listUC:   listUC,
	}
}

func NewIndependentDemandHandler(
	createUC *independent_demand_uc.CreateIndependentDemandUseCase,
	updateUC *independent_demand_uc.UpdateIndependentDemandUseCase,
	deleteUC *independent_demand_uc.DeleteIndependentDemandUseCase,
	listFromDateUC *independent_demand_uc.ListIndependentDemandFromDateUseCase,
	listByItemCodeUC *independent_demand_uc.ListIndependentDemandByItemUseCase,
	listUC *independent_demand_uc.ListIndependentDemandsUseCase,
	getByCodeUC *independent_demand_uc.GetIndependentDemandByCodeUseCase,
) *IndependentDemandHandler {
	return &IndependentDemandHandler{
		createUC:       createUC,
		updateUC:       updateUC,
		deleteUC:       deleteUC,
		listFromDateUC: listFromDateUC,
		listByItemUC:   listByItemCodeUC,
		listUC:         listUC,
		getByCodeUC:    getByCodeUC,
	}
}

func NewIndustrialCalendarHandler(
	uc *industrial_calendar_uc.ManageCalendarUseCase,
) *IndustrialCalendarHandler {
	return &IndustrialCalendarHandler{
		uc: uc,
	}
}

func NewItemCalendarPromiseHandler(
	uc *item_calendar_promise_uc.ManageItemCalendarPromiseUseCase,
) *ItemCalendarPromiseHandler {
	return &ItemCalendarPromiseHandler{
		uc: uc,
	}
}

func NewMachineHandler(
	createMachineUC *machine_uc.CreateMachineUseCase,
	listMachinesUC *machine_uc.ListMachinesUseCase,
	getMachineUC *machine_uc.GetMachineUseCase,

	createTypeUC *machine_uc.CreateMachineTypeUseCase,
	listTypesUC *machine_uc.ListMachineTypesUseCase,
	getMachineTypeUC *machine_uc.GetMachineTypeUseCase,

	createItemTimeUC *machine_uc.CreateItemMachineTimeUseCase,
	listItemTimesUC *machine_uc.ListItemMachineTimesUseCase,
	//getItemTimeUC *machine_uc.GetItemMachineTimeUseCase,
	calculateProductionTimeUC *machine_uc.CalculateProductionTimeUseCase,

	scheduleUC *machine_uc.ScheduleMachineUseCase,
) *MachineHandler {
	return &MachineHandler{
		createMachineUC: createMachineUC,
		listMachinesUC:  listMachinesUC,
		getMachineUC:    getMachineUC,

		createTypeUC:     createTypeUC,
		listTypesUC:      listTypesUC,
		getMachineTypeUC: getMachineTypeUC,

		createItemTimeUC:          createItemTimeUC,
		listItemTimesUC:           listItemTimesUC,
		calculateProductionTimeUC: calculateProductionTimeUC,
		//getItemTimeUC:    getItemTimeUC,

		scheduleUC: scheduleUC,
	}
}

func NewMRPCalculationHandler(
	runUC *mrp_calculation_uc.RunMRPCalculationUseCase,
	getProfileUC *mrp_calculation_uc.GetItemProfileUseCase,
	configuredRulesUC *mrp_calculation_uc.ManageConfiguredItemRulesUseCase,
	listExceptionsUC *mrp_calculation_uc.ListMRPExceptionsUseCase,
) *MRPCalculationHandler {
	return &MRPCalculationHandler{
		runUC:             runUC,
		getProfileUC:      getProfileUC,
		configuredRulesUC: configuredRulesUC,
		listExceptionsUC:  listExceptionsUC,
	}
}

func NewOrderPriorityHandler(
	createUC *order_priority_uc.CreateOrderPriorityUseCase,
	listUC *order_priority_uc.ListOrderPrioritiesUseCase,
	findUC *order_priority_uc.FindPriorityByValueUseCase,
) *OrderPriorityHandler {
	return &OrderPriorityHandler{
		createUC: createUC,
		listUC:   listUC,
		findUC:   findUC,
	}
}

func NewOverheadAllocationHandler(
	createUC *overhead_allocation_uc.CreateOverheadAllocationUseCase,
	listUC *overhead_allocation_uc.ListOverheadAllocationsUseCase,
) *OverheadAllocationHandler {
	return &OverheadAllocationHandler{
		createUC: createUC,
		listUC:   listUC,
	}
}

func NewPlannedOrderHandler(
	createUC *planned_order_uc.CreatePlannedOrderUseCase,
	listUC *planned_order_uc.ListPlannedOrdersUseCase,
	firmUC *planned_order_uc.FirmPlannedOrderUseCase,
) *PlannedOrderHandler {
	return &PlannedOrderHandler{
		createUC: createUC,
		listUC:   listUC,
		firmUC:   firmUC,
	}
}

func NewPlanningParamsHandler(
	getUC *planning_params_uc.GetPlanningParamUseCase,
	listUC *planning_params_uc.ListPlanningParamsUseCase,
	updateUC *planning_params_uc.UpdatePlanningParamUseCase,
) *PlanningParamsHandler {
	return &PlanningParamsHandler{
		getUC:    getUC,
		listUC:   listUC,
		updateUC: updateUC,
	}
}

func NewProductionPlanHandler(
	createUC *production_plan_uc.CreateProductionPlanUseCase,
	getUC *production_plan_uc.GetProductionPlanUseCase,
	listUC *production_plan_uc.ListProductionPlansUseCase,
	updateUC *production_plan_uc.UpdateProductionPlanUseCase,
	deleteUC *production_plan_uc.DeleteProductionPlanUseCase,
) *ProductionPlanHandler {
	return &ProductionPlanHandler{
		createUC: createUC,
		getUC:    getUC,
		listUC:   listUC,
		updateUC: updateUC,
		deleteUC: deleteUC,
	}
}

func NewRestrictionHandler(
	createUC *restriction_uc.CreateRestrictionUseCase,
	getUC *restriction_uc.GetRestrictionUseCase,
	listUC *restriction_uc.ListRestrictionsUseCase,
	getByItemUC *restriction_uc.GetRestrictionsByItemUseCase,
	updateUC *restriction_uc.UpdateRestrictionUseCase,
	deactivateUC *restriction_uc.DeactivateRestrictionUseCase,
) *RestrictionHandler {
	return &RestrictionHandler{
		createUC:     createUC,
		getUC:        getUC,
		listUC:       listUC,
		getByItemUC:  getByItemUC,
		updateUC:     updateUC,
		deactivateUC: deactivateUC,
	}
}
