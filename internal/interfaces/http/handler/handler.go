package handler

import (
	"github.com/FelipePn10/panossoerp/internal/application/usecase/allocation_base_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/aps_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/cost_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/crp_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/cutting_plan_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_reschedule_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/employee"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/enterprise_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/generate_mask_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/group_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/independent_demand_uc"
	industrial_calendar_uc "github.com/FelipePn10/panossoerp/internal/application/usecase/industrial_calendar"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_calendar_promise_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/modifier_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_calculation_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/order_priority_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/overhead_allocation_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planned_order_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planning_params_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/product_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/production_plan_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/quality_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/question_option_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/question_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/restriction_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/routing_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/structure_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/user_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/warehouse_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

type ProductHandler struct {
	*security.BaseHandler
	createProductUC *product_uc.CreateProductUseCase
	deleteProductUC *product_uc.DeleteProductUseCase
}

type ItemHandler struct {
	*security.BaseHandler
	createItemUC         *item_uc.CreateItemUseCase
	findItemByCodeUC     *item_uc.FindItemByCode
	listItemsUC          *item_uc.ListItemsUseCase
	listItemsWithMasksUC *item_uc.ListItemsWithMasksUseCase
}

type UserHandler struct {
	*security.BaseHandler
	registerUC *user_uc.RegisterUserUseCase
	loginUC    *user_uc.LoginUserUseCase
	jwtSecret  string
}

type QuestionHandler struct {
	*security.BaseHandler
	createQuestionUC     *question_uc.CreateQuestion
	deleteQuestionUC     *question_uc.DeleteQuestionUseCase
	findQuestionByNameUC *question_uc.FindQuestionByName
}

type QuestionOptionHandler struct {
	*security.BaseHandler
	createQuestionOptionUC  *question_option_uc.CreateQuestionOptionUseCase
	deleteQuestionOptionUC  *question_option_uc.DeleteQuestionOptionUseCase
	listOptionsByQuestionUC *question_option_uc.ListOptionsByQuestionUseCase
}

type AssociateByQuestionItemHandler struct {
	*security.BaseHandler
	associateByQuestionProductUC *question_uc.AssociateByQuestionItemUseCase
	getQuestionsByItemUC         *question_uc.GetQuestionsByItemUseCase
	listAllItemQuestionsUC       *question_uc.ListAllItemQuestionsUseCase
}

type GenerateMaskHandler struct {
	*security.BaseHandler
	generateMask *generate_mask_uc.GenerateMaskForItemUseCase
}

type WarehouseHandler struct {
	*security.BaseHandler
	createWarehouseUC *warehouse_uc.CreateWarehouseUseCase
	listWarehousesUC  *warehouse_uc.ListWarehousesUseCase
	getWarehouseUC    *warehouse_uc.GetWarehouseUseCase
}

type GroupHandler struct {
	*security.BaseHandler
	createGroupUC *group_uc.CreateGroupUseCase
	getGroupUC    *group_uc.GetGroupUseCase
	listGroupsUC  *group_uc.ListGroupsUseCase
	updateGroupUC *group_uc.UpdateGroupUseCase
}

type EnterpriseHandler struct {
	*security.BaseHandler
	createEnterpriseUC *enterprise_uc.CreateEnterpriseUseCase
	getEnterpriseUC    *enterprise_uc.GetEnterpriseUseCase
	listEnterprisesUC  *enterprise_uc.ListEnterprisesUseCase
}

type ModifierHandler struct {
	*security.BaseHandler
	createModifierUC *modifier_uc.CreateModifierUseCase
	getModifierUC    *modifier_uc.GetModifierUseCase
	listModifiersUC  *modifier_uc.ListModifiersUseCase
	updateModifierUC *modifier_uc.UpdateModifierUseCase
}

type EmployeeHandler struct {
	*security.BaseHandler
	createUC     *employee.CreateEmployeeUseCase
	listUC       *employee.ListEmployeesUseCase
	getUC        *employee.GetEmployeeUseCase
	updateUC     *employee.UpdateEmployeeUseCase
	deactivateUC *employee.DeactivateEmployeeUseCase
}

type ItemStructureHandler struct {
	*security.BaseHandler
	createUC        *structure_uc.CreateStructureComponentUseCase
	updateUC        *structure_uc.UpdateStructureComponentUseCase
	getAllStructure *structure_uc.GetAllDirectChildrenUseCase
	treeUC          *structure_uc.GetStructureTreeUseCase
	resolveUC       *structure_uc.ResolveStructureQueryUseCase
	//deleteUC  *structure_uc.DeleteStructureComponentUseCase
}

type ItemQueryStructureHandler struct {
	*security.BaseHandler
	resolveUC   *structure_uc.ResolveStructureQueryUseCase
	consultUC   *structure_uc.ConsultStructureUseCase
	whereUsedUC *structure_uc.WhereUsedUseCase
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
	listExceptionsUC  *mrp_calculation_uc.ListMRPExceptionsUseCase
	firmarSugestaoUC  *mrp_uc.FirmarSugestaoMRPUseCase
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

type PlanningParamsHandler struct {
	*security.BaseHandler
	getUC    *planning_params_uc.GetPlanningParamUseCase
	listUC   *planning_params_uc.ListPlanningParamsUseCase
	updateUC *planning_params_uc.UpdatePlanningParamUseCase
}

type ProductionPlanHandler struct {
	*security.BaseHandler
	createUC *production_plan_uc.CreateProductionPlanUseCase
	getUC    *production_plan_uc.GetProductionPlanUseCase
	listUC   *production_plan_uc.ListProductionPlansUseCase
	updateUC *production_plan_uc.UpdateProductionPlanUseCase
	deleteUC *production_plan_uc.DeleteProductionPlanUseCase
}

type RestrictionHandler struct {
	*security.BaseHandler
	createUC        *restriction_uc.CreateRestrictionUseCase
	getUC           *restriction_uc.GetRestrictionUseCase
	listUC          *restriction_uc.ListRestrictionsUseCase
	getByItemUC     *restriction_uc.GetRestrictionsByItemUseCase
	getByCustomerUC *restriction_uc.GetRestrictionsByCustomerUseCase
	updateUC        *restriction_uc.UpdateRestrictionUseCase
	deactivateUC    *restriction_uc.DeactivateRestrictionUseCase
	evaluateUC      *restriction_uc.EvaluateRestrictionsUseCase
}

type RestrictionReasonHandler struct {
	*security.BaseHandler
	createUC *restriction_uc.CreateRestrictionReasonUseCase
	getUC    *restriction_uc.GetRestrictionReasonUseCase
	listUC   *restriction_uc.ListRestrictionReasonsUseCase
	updateUC *restriction_uc.UpdateRestrictionReasonUseCase
	deleteUC *restriction_uc.DeleteRestrictionReasonUseCase
}

type RoutingHandler struct {
	*security.BaseHandler
	operationUC *routing_uc.OperationUseCase
	routeUC     *routing_uc.RouteUseCase
	leadTimeUC  *routing_uc.LeadTimeUseCase
}

type CuttingPlanHandler struct {
	*security.BaseHandler
	uc     *cutting_plan_uc.CuttingPlanUseCase
	demand *cutting_plan_uc.DemandUseCase
}

type QualityHandler struct {
	*security.BaseHandler
	uc *quality_uc.QualityUseCase
}

type StandardCostHandler struct {
	*security.BaseHandler
	uc *cost_uc.StandardCostUseCase
}

type CRPHandler struct {
	*security.BaseHandler
	uc *crp_uc.CRPUseCase
}

type APSHandler struct {
	*security.BaseHandler
	uc     *aps_uc.APSUseCase
	fiscal fiscalConfigReader // optional letterhead for the Gantt export; may be nil
}
