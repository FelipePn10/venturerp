package handler

import (
	"github.com/FelipePn10/panossoerp/internal/application/usecase"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/allocation_base_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_reschedule_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/employee"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/independent_demand_uc"
	industrial_calendar_uc "github.com/FelipePn10/panossoerp/internal/application/usecase/industrial_calendar"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_calendar_promise_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_calculation_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/order_priority_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/overhead_allocation_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planned_order_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

type ProductHandler struct {
	*security.BaseHandler
	createProductUC *usecase.CreateProductUseCase
	deleteProductUC *usecase.DeleteProductUseCase
}

type ItemHandler struct {
	*security.BaseHandler
	createItemUC     *usecase.CreateItemUseCase
	findItemByCodeUC *usecase.FindItemByCode
}

type UserHandler struct {
	*security.BaseHandler
	registerUC *usecase.RegisterUserUseCase
	loginUC    *usecase.LoginUserUseCase
	jwtSecret  string
}

type QuestionHandler struct {
	*security.BaseHandler
	createQuestionUC     *usecase.CreateQuestion
	deleteQuestionUC     *usecase.DeleteQuestionUseCase
	findQuestionByNameUC *usecase.FindQuestionByName
}

type QuestionOptionHandler struct {
	*security.BaseHandler
	createQuestionOptionUC *usecase.CreateQuestionOptionUseCase
	deleteQuestionOptionUC *usecase.DeleteQuestionOptionUseCase
}

type AssociateByQuestionItemHandler struct {
	*security.BaseHandler
	associateByQuestionProductUC *usecase.AssociateByQuestionItemUseCase
}

type GenerateMaskHandler struct {
	*security.BaseHandler
	generateMask *usecase.GenerateMaskForItemUseCase
}

type BomHandler struct {
	*security.BaseHandler
	createBomUC *usecase.CreateBomUseCase
}

type BomItemHandler struct {
	*security.BaseHandler
	createBomItemUC *usecase.CreateBomItemUseCase
}

type WarehouseHandler struct {
	*security.BaseHandler
	createWarehouseUC *usecase.CreateWarehouseUseCase
}

type GroupHandler struct {
	*security.BaseHandler
	createGroupUC *usecase.CreateGroupUseCase
}

type EnterpriseHandler struct {
	*security.BaseHandler
	createEnterpriseUC *usecase.CreateEnterpriseUseCase
}

type ModifierHandler struct {
	*security.BaseHandler
	createModifierUC *usecase.CreateModifierUseCase
}

type EmployeeHandler struct {
	*security.BaseHandler
	createEmployeeUC *employee.CreateEmployeeUseCase
}

type ItemStructureHandler struct {
	*security.BaseHandler
	createUC        *usecase.CreateStructureComponentUseCase
	updateUC        *usecase.UpdateStructureComponentUseCase
	getAllStructure *usecase.GetAllDirectChildrenUseCase
	treeUC          *usecase.GetStructureTreeUseCase
	resolveUC       *usecase.ResolveStructureQueryUseCase
	//deleteUC  *usecase.DeleteStructureComponentUseCase
}

type ItemQueryStructureHandler struct {
	*security.BaseHandler
	resolveUC *usecase.ResolveStructureQueryUseCase
}

type AllocationBaseHandler struct {
	*security.BaseHandler
	createUC *allocation_base_uc.CreateAllocationBaseUseCase
	listUC   *allocation_base_uc.ListAllocationBasesUseCase
}

type DeliveryRescheduleHandler struct {
	*security.BaseHandler
	createUC *delivery_reschedule_uc.CreateDeliveryRescheduleUseCase
	listUC   *delivery_reschedule_uc.ListDeliveryReschedulesUseCase
}

type IndependentDemandHandler struct {
	*security.BaseHandler
	createUC       *independent_demand_uc.CreateIndependentDemandUseCase
	updateUC       *independent_demand_uc.UpdateIndependentDemandUseCase
	deleteUC       *independent_demand_uc.DeleteIndependentDemandUseCase
	listFromDateUC *independent_demand_uc.ListIndependentDemandFromDateUseCase
	listByItemUC   *independent_demand_uc.ListIndependentDemandByItemUseCase
	listUC         *independent_demand_uc.ListIndependentDemandsUseCase
	getByCodeUC    *independent_demand_uc.GetIndependentDemandByCodeUseCase
}

type IndustrialCalendarHandler struct {
	*security.BaseHandler
	uc *industrial_calendar_uc.ManageCalendarUseCase
}

type ItemCalendarPromiseHandler struct {
	*security.BaseHandler
	uc *item_calendar_promise_uc.ManageItemCalendarPromiseUseCase
}

type MRPCalculationHandler struct {
	*security.BaseHandler
	runUC             *mrp_calculation_uc.RunMRPCalculationUseCase
	getProfileUC      *mrp_calculation_uc.GetItemProfileUseCase
	configuredRulesUC *mrp_calculation_uc.ManageConfiguredItemRulesUseCase
}

type OrderPriorityHandler struct {
	*security.BaseHandler
	createUC *order_priority_uc.CreateOrderPriorityUseCase
	listUC   *order_priority_uc.ListOrderPrioritiesUseCase
	findUC   *order_priority_uc.FindPriorityByValueUseCase
}

type OverheadAllocationHandler struct {
	*security.BaseHandler
	createUC *overhead_allocation_uc.CreateOverheadAllocationUseCase
	listUC   *overhead_allocation_uc.ListOverheadAllocationsUseCase
}

type PlannedOrderHandler struct {
	*security.BaseHandler
	createUC *planned_order_uc.CreatePlannedOrderUseCase
	listUC   *planned_order_uc.ListPlannedOrdersUseCase
	firmUC   *planned_order_uc.FirmPlannedOrderUseCase
}
