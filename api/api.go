package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/allocation_base_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/bom_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/cost_center_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_promise_params_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_reschedule_uc"
	employeeUC "github.com/FelipePn10/panossoerp/internal/application/usecase/employee"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/enterprise_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc"
	fiscalUC "github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/generate_mask_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/group_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/independent_demand_uc"
	industrial_calendar_uc "github.com/FelipePn10/panossoerp/internal/application/usecase/industrial_calendar"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_calendar_promise_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/machine_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/modifier_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_calculation_uc"
	mrpservice "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/service"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/order_priority_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/overhead_allocation_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planned_order_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planning_params_uc"
	productionOrderUc "github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/production_plan_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/question_option_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/question_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/restriction_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_division_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_forecast_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/stock_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/structure_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/user_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/warehouse_uc"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	financialRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/config"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database"
	applogger "github.com/FelipePn10/panossoerp/internal/infrastructure/logger"
	fiscalRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	allocation "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/allocation_base"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/bom"
	bomitem "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/bom_item"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/cost_center"
	deliveryPromiseParams "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_promise_params"
	deliveryReschedule "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_reschedule"
	employee "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/employee"
	planningParams "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planning_params"
	productionPlan "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_plan"
	restrictionRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/restriction"
	salesDivisionRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_division"
	salesForecastRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_forecast"
	salesOrderRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	stockRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	enterprise "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/enterprise"
	generatemask "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/generate_mask"
	group "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/group"
	independentDemand "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/independent_demand"
	industrialCalendar "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/industrial_calendar"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	itemCalendarPromise "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_calendar_promise"
	itemquestion "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_question"
	machine "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/machine"
	modifier "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/modifier"
	mrpCalculation "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation"
	op "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/order_priority"
	over 	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/overhead_allocation"
	planned "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planned_order"
	productionOrderRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	purchaseOrderRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/questions"
	questionsoptions "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/questions_options"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure_query"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/user"
	warehouse "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/warehouse"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler"
	httpmw "github.com/FelipePn10/panossoerp/internal/interfaces/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type application struct {
	config *config.Config
	logger *applogger.Logger
	db     *database.DB
}

func (app *application) mount() chi.Router {
	r := chi.NewRouter()

	r.Use(httpmw.CorrelationMiddleware)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.StripSlashes)
	r.Use(httpmw.RequestLoggerMiddleware(app.logger))

	queries := app.db.Queries()
	authService := &infraauth.AuthService{}

	userRepo := user.NewRepositoryUserSQLC(queries)

	registerUserUC := user_uc.NewRegisterUserUseCase(userRepo)
	loginUserUC := user_uc.NewLoginUserUseCase(userRepo)

	userHandler := handler.NewUserHandler(
		registerUserUC,
		loginUserUC,
		app.config.JWTSecret,
	)

	r.Route("/users", func(r chi.Router) {
		r.Post("/register", userHandler.RegisterUserHandler)
		r.Post("/login", userHandler.LoginHandler)
	})

	// question
	questionRepo := questions.NewRepositoryQuestionSQLC(queries)
	createQuestionUC := question_uc.NewCreateQuestion(questionRepo, authService)
	findQuestionByNameUC := question_uc.NewFindQuestionByName(questionRepo)

	questionCreateHandler := handler.NewQuestionHandler(createQuestionUC)
	findQuestionByNameHandler := handler.NewFindQuestionByName(findQuestionByNameUC)

	// question option
	questionOptionRepo := questionsoptions.NewRepositoryQuestionOptionSQLC(queries)

	createQuestionOptionUC := question_option_uc.NewCreateQuestionOptionUseCase(questionOptionRepo, authService)
	questionOptionCreateHandler := handler.NewCreateQuestionOptionHandler(createQuestionOptionUC)

	// associate question in item
	itemByQuestionItemRepo := itemquestion.NewAssociateQuestionItemRepositorySQLC(queries)
	associateByQuestionItemUC := question_uc.NewAssociateByQuestionItemUseCase(itemByQuestionItemRepo, authService)
	associateByQuestionItemHandler := handler.NewAssociateByQuestionItemHandler(associateByQuestionItemUC)

	// generate mask item
	generateMaskItem := generatemask.NewRepositoryGenerateMaskSQLC(queries)
	generateMaskItemUC := generate_mask_uc.NewGenerateMaskItemUseCase(generateMaskItem, authService)
	generateMaskItemHandler := handler.NewGeneratMaskItemHandler(generateMaskItemUC)

	// Item
	itemRepo := item.NewRepositoryItemSQLC(queries)
	createItemUc := item_uc.NewCreateItemUseCase(itemRepo, authService)
	findItemByCodeUc := item_uc.NewFindItemByCode(itemRepo, authService)
	itemHandler := handler.NewCreateItemHandler(createItemUc, findItemByCodeUc)

	// Item Structure
	itemRepoStructure := structure.NewItemStructureRepository(queries)
	createStructureUc := structure_uc.NewCreateStructureComponentUseCase(itemRepoStructure, authService)
	updateStructureUc := structure_uc.NewUpdateStructureComponentUseCase(itemRepoStructure, authService)
	getAllStructureUc := structure_uc.NewGetAllDirectChildrenUseCase(itemRepoStructure, authService)
	treeStructureUc := structure_uc.NewGetStructureTreeUseCase(itemRepoStructure, authService)
	structureHandler := handler.NewItemStructureHandler(createStructureUc, updateStructureUc, getAllStructureUc, treeStructureUc)

	// Item Structure Query
	itemRepoStructureQuery := structure_query.NewStructureQueryRepository(queries)
	queryStructureUc := structure_uc.NewResolveStructureQueryUseCase(itemRepoStructureQuery, authService)
	queryStructureHandler := handler.NewQueryStructureHandler(queryStructureUc)
	// bom
	bomRepo := bom.NewRepostioryBomSQLC(queries)

	createBomUc := bom_uc.NewCreateBomUseCase(bomRepo, authService)
	bomHandler := handler.NewCreateBomHandler(createBomUc)

	// bom item
	bomItemRepo := bomitem.NewRepositoryBomItemSQLC(queries)

	createBomItemUc := &bom_uc.CreateBomItemUseCase{Repo: bomItemRepo, Auth: authService}
	bomItemHandler := handler.NewCreateBomItemHandler(createBomItemUc)

	// warehouse
	warehouseRepo := warehouse.NewRepositoryQuestionSQLC(queries)
	createWarehouseUc := warehouse_uc.NewCreateWarehouseUseCase(warehouseRepo, authService)
	warehouseHandler := handler.NewCreateWarehouseHandler(createWarehouseUc)

	// group
	groupRepo := group.NewRepositoryGroupSQLC(queries)
	createGroupUc := group_uc.NewCreateGroupUseCase(groupRepo, authService)
	groupHandler := handler.NewCreateGroupHandler(createGroupUc)

	// enterprise
	enterpriseRepo := enterprise.NewRepositoryEnterpriseSQLC(queries)
	createEnterpriseUc := enterprise_uc.NewCreateEnterpriseUseCase(enterpriseRepo, authService)
	enterpriseHandler := handler.NewCreateEnterpriseHandler(createEnterpriseUc)

	// modifier
	modifierRepo := modifier.NewRepositoryModifierSQLC(queries)
	createModifierUc := modifier_uc.NewCreateModifierUseCase(modifierRepo, authService)
	modifierHandler := handler.NewCreateModifierHandler(createModifierUc)

	// employee
	employeeRepo := employee.NewRepositoryEmployeeSQLC(queries)
	createEmployeeUc := &employeeUC.CreateEmployeeUseCase{Repo: employeeRepo, Auth: authService}
	listEmployeesUC := &employeeUC.ListEmployeesUseCase{Repo: employeeRepo, Auth: authService}
	getEmployeeUC := &employeeUC.GetEmployeeUseCase{Repo: employeeRepo, Auth: authService}
	updateEmployeeUC := &employeeUC.UpdateEmployeeUseCase{Repo: employeeRepo, Auth: authService}
	deactivateEmployeeUC := &employeeUC.DeactivateEmployeeUseCase{Repo: employeeRepo, Auth: authService}
	employeeHandler := handler.NewEmployeeHandler(createEmployeeUc, listEmployeesUC, getEmployeeUC, updateEmployeeUC, deactivateEmployeeUC)

	// planning params
	planningParamsRepo := planningParams.NewPlanningParamRepositorySQLC(queries)
	getPlanningParamUC := &planning_params_uc.GetPlanningParamUseCase{Repo: planningParamsRepo, Auth: authService}
	listPlanningParamsUC := &planning_params_uc.ListPlanningParamsUseCase{Repo: planningParamsRepo, Auth: authService}
	updatePlanningParamUC := &planning_params_uc.UpdatePlanningParamUseCase{Repo: planningParamsRepo, Auth: authService}
	planningParamsHandler := handler.NewPlanningParamsHandler(getPlanningParamUC, listPlanningParamsUC, updatePlanningParamUC)

	// production plan
	productionPlanRepo := productionPlan.NewProductionPlanRepositorySQLC(queries)
	createProductionPlanUC := &production_plan_uc.CreateProductionPlanUseCase{Repo: productionPlanRepo, Auth: authService}
	getProductionPlanUC := &production_plan_uc.GetProductionPlanUseCase{Repo: productionPlanRepo, Auth: authService}
	listProductionPlansUC := &production_plan_uc.ListProductionPlansUseCase{Repo: productionPlanRepo, Auth: authService}
	updateProductionPlanUC := &production_plan_uc.UpdateProductionPlanUseCase{Repo: productionPlanRepo, Auth: authService}
	deleteProductionPlanUC := &production_plan_uc.DeleteProductionPlanUseCase{Repo: productionPlanRepo, Auth: authService}
	productionPlanHandler := handler.NewProductionPlanHandler(createProductionPlanUC, getProductionPlanUC, listProductionPlansUC, updateProductionPlanUC, deleteProductionPlanUC)

	// restriction
	restrictionR := restrictionRepo.NewRestrictionRepositorySQLC(queries)
	createRestrictionUC := &restriction_uc.CreateRestrictionUseCase{Repo: restrictionR, Auth: authService}
	getRestrictionUC := &restriction_uc.GetRestrictionUseCase{Repo: restrictionR, Auth: authService}
	listRestrictionsUC := &restriction_uc.ListRestrictionsUseCase{Repo: restrictionR, Auth: authService}
	getRestrictionsByItemUC := &restriction_uc.GetRestrictionsByItemUseCase{Repo: restrictionR, Auth: authService}
	updateRestrictionUC := &restriction_uc.UpdateRestrictionUseCase{Repo: restrictionR, Auth: authService}
	deactivateRestrictionUC := &restriction_uc.DeactivateRestrictionUseCase{Repo: restrictionR, Auth: authService}
	restrictionHandler := handler.NewRestrictionHandler(createRestrictionUC, getRestrictionUC, listRestrictionsUC, getRestrictionsByItemUC, updateRestrictionUC, deactivateRestrictionUC)

	// sales division
	sdRepo := salesDivisionRepo.NewSalesDivisionRepositorySQLC(queries)
	salesDivisionHandler := handler.NewSalesDivisionHandler(
		&sales_division_uc.CreateSalesDivisionUseCase{Repo: sdRepo, Auth: authService},
		&sales_division_uc.ListSalesDivisionsUseCase{Repo: sdRepo, Auth: authService},
		&sales_division_uc.GetSalesDivisionUseCase{Repo: sdRepo, Auth: authService},
		&sales_division_uc.UpdateSalesDivisionUseCase{Repo: sdRepo, Auth: authService},
		&sales_division_uc.DeleteSalesDivisionUseCase{Repo: sdRepo, Auth: authService},
	)

	// sales forecast
	sfRepo := salesForecastRepo.NewSalesForecastRepositorySQLC(queries)
	salesForecastHandler := handler.NewSalesForecastHandler(
		&sales_forecast_uc.CreateSalesForecastUseCase{Repo: sfRepo, Auth: authService},
		&sales_forecast_uc.ListSalesForecastsUseCase{Repo: sfRepo, Auth: authService},
		&sales_forecast_uc.GetForecastByItemUseCase{Repo: sfRepo, Auth: authService},
		&sales_forecast_uc.CreateForecastBlockUseCase{Repo: sfRepo, Auth: authService},
		&sales_forecast_uc.ListForecastBlocksUseCase{Repo: sfRepo, Auth: authService},
		&sales_forecast_uc.CreateAppropriationTableUseCase{Repo: sfRepo, Auth: authService},
		&sales_forecast_uc.ListAppropriationTablesUseCase{Repo: sfRepo, Auth: authService},
		&sales_forecast_uc.SetDefaultAppropriationUseCase{Repo: sfRepo, Auth: authService},
	)

	// allocation base
	allocationBaseRepo := allocation.NewAllocationBaseRepositorySQLC(queries)
	createAllocationBaseUC := &allocation_base_uc.CreateAllocationBaseUseCase{Repo: allocationBaseRepo, Auth: authService}
	listAllocationBaseUC := &allocation_base_uc.ListAllocationBasesUseCase{Repo: allocationBaseRepo, Auth: authService}
	allocationBaseHandler := handler.NewAllocationBaseHandler(createAllocationBaseUC, listAllocationBaseUC)

	// cost center
	costCenterRepo := cost_center.NewCostCenterRepositorySQLC(queries)
	createCostCenterUC := &cost_center_uc.CreateCostCenterUseCase{Repo: costCenterRepo, Auth: authService}
	listCostCenterUC := &cost_center_uc.ListCostCentersUseCase{Repo: costCenterRepo, Auth: authService}
	getCostCenterUC := &cost_center_uc.GetCostCenterUseCase{Repo: costCenterRepo, Auth: authService}
	costCenterHandler := handler.NewCostCenterHandler(createCostCenterUC, listCostCenterUC, getCostCenterUC)

	// delivery promise params
	deliveryPromiseParamsRepo := deliveryPromiseParams.NewDeliveryPromiseParamsRepositorySQLC(queries)
	manageDeliveryPromiseParamsUC := &delivery_promise_params_uc.ManageDeliveryPromiseParamsUseCase{Repo: deliveryPromiseParamsRepo, Auth: authService}
	deliveryPromiseParamsHandler := handler.NewDeliveryPromiseParamsHandler(manageDeliveryPromiseParamsUC)

	// delivery reschedule
	deliveryRescheduleRepo := deliveryReschedule.NewDeliveryRescheduleRepositorySQLC(queries)
	createDeliveryRescheduleUC := &delivery_reschedule_uc.CreateDeliveryRescheduleUseCase{Repo: deliveryRescheduleRepo, Auth: authService}
	listDeliveryRescheduleUC := &delivery_reschedule_uc.ListDeliveryReschedulesUseCase{Repo: deliveryRescheduleRepo, Auth: authService}
	deliveryRescheduleHandler := handler.NewDeliveryRescheduleHandler(createDeliveryRescheduleUC, listDeliveryRescheduleUC)

	// independent demand
	independentDemandRepo := independentDemand.NewIndependentDemandRepositorySQLC(queries)
	createIndependentDemandUC := &independent_demand_uc.CreateIndependentDemandUseCase{Repo: independentDemandRepo, Auth: authService}
	updateIndependentDemandUC := &independent_demand_uc.UpdateIndependentDemandUseCase{Repo: independentDemandRepo, Auth: authService}
	deleteIndependentDemandUC := &independent_demand_uc.DeleteIndependentDemandUseCase{Repo: independentDemandRepo, Auth: authService}
	listFromDateIndependentDemandUC := &independent_demand_uc.ListIndependentDemandFromDateUseCase{Repo: independentDemandRepo, Auth: authService}
	listByItemIndependentDemandUC := &independent_demand_uc.ListIndependentDemandByItemUseCase{Repo: independentDemandRepo, Auth: authService}
	listIndependentDemandUC := &independent_demand_uc.ListIndependentDemandsUseCase{Repo: independentDemandRepo, Auth: authService}
	getByCodeDemandUC := &independent_demand_uc.GetIndependentDemandByCodeUseCase{Repo: independentDemandRepo, Auth: authService}
	independentDemandHandler := handler.NewIndependentDemandHandler(createIndependentDemandUC, updateIndependentDemandUC, deleteIndependentDemandUC, listFromDateIndependentDemandUC, listByItemIndependentDemandUC, listIndependentDemandUC, getByCodeDemandUC)

	// industrial calendar
	industrialCalendarRepo := industrialCalendar.NewIndustrialCalendarRepositorySQLC(queries)
	manageIndustrialCalendarRepoUC := &industrial_calendar_uc.ManageCalendarUseCase{Repo: industrialCalendarRepo, Auth: authService}
	industrialCalendarHandler := handler.NewIndustrialCalendarHandler(manageIndustrialCalendarRepoUC)

	// item calendar promise
	itemCalendarPromise := itemCalendarPromise.NewItemCalendarPromiseRepositorySQLC(queries)
	itemCalendarPromiseUC := &item_calendar_promise_uc.ManageItemCalendarPromiseUseCase{Repo: itemCalendarPromise, Auth: authService}
	itemCalendarPromiseHandler := handler.NewItemCalendarPromiseHandler(itemCalendarPromiseUC)

	// machine
	machineRepo := machine.NewMachineRepositorySQLC(queries)
	machineUC := &machine_uc.CreateMachineUseCase{Repo: machineRepo, Auth: authService}
	machineListUC := &machine_uc.ListMachinesUseCase{Repo: machineRepo, Auth: authService}
	machineGetByCodeUC := &machine_uc.GetMachineUseCase{Repo: machineRepo, Auth: authService}
	//type
	machineTypeCreateUC := &machine_uc.CreateMachineTypeUseCase{Repo: machineRepo, Auth: authService}
	machineListTypesUC := &machine_uc.ListMachineTypesUseCase{Repo: machineRepo, Auth: authService}
	machineTypeGetByCodeUC := &machine_uc.GetMachineTypeUseCase{Repo: machineRepo, Auth: authService}
	//item times
	machineItemTimeUC := &machine_uc.CreateItemMachineTimeUseCase{Repo: machineRepo, ItemRepo: itemRepo, Auth: authService}
	machineListItemTimeUC := &machine_uc.ListItemMachineTimesUseCase{Repo: machineRepo, Auth: authService}
	//machineGetItemTimeUC := &machine_uc.GetItemMachineTimeUseCase{Repo: machineRepo, Auth: authService}
	machineCalcProductionUC := &machine_uc.CalculateProductionTimeUseCase{Repo: machineRepo, ItemRepo: itemRepo, Auth: authService}
	// schedule
	scheduleUC := &machine_uc.ScheduleMachineUseCase{Repo: machineRepo, Auth: authService}

	machineHandler := handler.NewMachineHandler(
		machineUC,
		machineListUC,
		machineGetByCodeUC,
		machineTypeCreateUC,
		machineListTypesUC,
		machineTypeGetByCodeUC,
		machineItemTimeUC,
		machineListItemTimeUC,
		//machineGetItemTimeUC,
		machineCalcProductionUC,
		scheduleUC,
	)

	// mrp_calculation
	mrpRepo := mrpCalculation.NewMRPCalculationRepositorySQLC(queries, app.db.Pool)
	supplyPort := planned.NewPlannedOrderSupplyAdapter(queries)
	mrpService := mrpservice.NewMRPService(mrpRepo, itemRepoStructure, independentDemandRepo, industrialCalendarRepo, itemRepo, supplyPort, productionPlanRepo, sfRepo, restrictionR)
	mrpRunUC := &mrp_calculation_uc.RunMRPCalculationUseCase{Service: mrpService, Auth: authService}
	mrpGetProfileUC := &mrp_calculation_uc.GetItemProfileUseCase{Repo: mrpRepo, Auth: authService}
	mrpCreateConfiguredRule := &mrp_calculation_uc.ManageConfiguredItemRulesUseCase{Repo: mrpRepo, Auth: authService}
	mrpListExceptionsUC := &mrp_calculation_uc.ListMRPExceptionsUseCase{Repo: mrpRepo, Auth: authService}
	mrpHandler := handler.NewMRPCalculationHandler(mrpRunUC, mrpGetProfileUC, mrpCreateConfiguredRule, mrpListExceptionsUC)

	//order priority
	opRepo := op.NewOrderPriorityRepositorySQLC(queries)
	opCreateUC := &order_priority_uc.CreateOrderPriorityUseCase{Repo: opRepo, Auth: authService}
	opListUC := &order_priority_uc.ListOrderPrioritiesUseCase{Repo: opRepo, Auth: authService}
	opFindUC := &order_priority_uc.FindPriorityByValueUseCase{Repo: opRepo, Auth: authService}
	opHandler := handler.NewOrderPriorityHandler(opCreateUC, opListUC, opFindUC)

	// overhead allocation
	overRepo := over.NewOverheadAllocationRepositorySQLC(queries)
	overCreateUC := &overhead_allocation_uc.CreateOverheadAllocationUseCase{Repo: overRepo, Auth: authService}
	overListUC := &overhead_allocation_uc.ListOverheadAllocationsUseCase{Repo: overRepo, Auth: authService}
	overHandler := handler.NewOverheadAllocationHandler(overCreateUC, overListUC)

	// planned order
	plannedRepo := planned.NewPlannedOrderRepositorySQLC(queries)
	plannedCreateUC := &planned_order_uc.CreatePlannedOrderUseCase{Repo: plannedRepo, Auth: authService}
	plannedListUC := &planned_order_uc.ListPlannedOrdersUseCase{Repo: plannedRepo, Auth: authService}
	plannedFirmUC := &planned_order_uc.FirmPlannedOrderUseCase{Repo: plannedRepo, Auth: authService}
	plannedHandler := handler.NewPlannedOrderHandler(plannedCreateUC, plannedListUC, plannedFirmUC)

	// production order
	prodOrderRepo := productionOrderRepo.NewProductionOrderRepositoryPGX(app.db.Pool)
	prodOrderCreateUC := &productionOrderUc.CreateProductionOrderUseCase{Repo: prodOrderRepo, Auth: authService}
	prodOrderGetByCodeUC := &productionOrderUc.GetProductionOrderUseCase{Repo: prodOrderRepo, Auth: authService}
	prodOrderListUC := &productionOrderUc.ListProductionOrdersUseCase{Repo: prodOrderRepo, Auth: authService}
	prodOrderStartUC := &productionOrderUc.StartProductionOrderUseCase{Repo: prodOrderRepo, Auth: authService}
	prodOrderAddAppointmentUC := &productionOrderUc.AddAppointmentUseCase{Repo: prodOrderRepo, Auth: authService}
	prodOrderAddConsumptionUC := &productionOrderUc.AddConsumptionUseCase{Repo: prodOrderRepo, Auth: authService}
	prodOrderCompleteUC := &productionOrderUc.CompleteProductionOrderUseCase{Repo: prodOrderRepo, Auth: authService}
	prodOrderCloseUC := &productionOrderUc.CloseProductionOrderUseCase{Repo: prodOrderRepo, Auth: authService}
	prodOrderCancelUC := &productionOrderUc.CancelProductionOrderUseCase{Repo: prodOrderRepo, Auth: authService}
	prodOrderGetAppointmentsUC := &productionOrderUc.GetAppointmentsUseCase{Repo: prodOrderRepo, Auth: authService}
	prodOrderGetConsumptionsUC := &productionOrderUc.GetConsumptionsUseCase{Repo: prodOrderRepo, Auth: authService}
	prodOrderHandler := handler.NewProductionOrderHandler(
		prodOrderCreateUC, prodOrderGetByCodeUC, prodOrderListUC,
		prodOrderStartUC, prodOrderAddAppointmentUC, prodOrderAddConsumptionUC,
		prodOrderCompleteUC, prodOrderCloseUC, prodOrderCancelUC,
		prodOrderGetAppointmentsUC, prodOrderGetConsumptionsUC,
	)

	// purchase order
	poRepo := purchaseOrderRepo.NewPurchaseOrderRepositorySQLC(app.db.Pool)
	purchaseOrderHandler := handler.NewPurchaseOrderHandler(
		&purchase_order_uc.CreatePurchaseOrderUseCase{Repo: poRepo, Auth: authService},
		&purchase_order_uc.UpdatePurchaseOrderUseCase{Repo: poRepo, Auth: authService},
		&purchase_order_uc.GetPurchaseOrderUseCase{Repo: poRepo, Auth: authService},
		&purchase_order_uc.ListPurchaseOrdersUseCase{Repo: poRepo, Auth: authService},
		&purchase_order_uc.ListPurchaseOrdersBySupplierUseCase{Repo: poRepo, Auth: authService},
		&purchase_order_uc.ListPurchaseOrdersByStatusUseCase{Repo: poRepo, Auth: authService},
		&purchase_order_uc.CancelPurchaseOrderUseCase{Repo: poRepo, Auth: authService},
	)

	// sales order
	soRepo := salesOrderRepo.NewSalesOrderRepositorySQLC(queries)
	salesOrderHandler := handler.NewSalesOrderHandler(
		&sales_order_uc.CreateSalesOrderUseCase{Repo: soRepo, Auth: authService},
		&sales_order_uc.UpdateSalesOrderUseCase{Repo: soRepo, Auth: authService},
		&sales_order_uc.GetSalesOrderUseCase{Repo: soRepo, Auth: authService},
		&sales_order_uc.ListSalesOrdersUseCase{Repo: soRepo, Auth: authService},
		&sales_order_uc.ListSalesOrdersByCustomerUseCase{Repo: soRepo, Auth: authService},
		&sales_order_uc.ListSalesOrdersByStatusUseCase{Repo: soRepo, Auth: authService},
		&sales_order_uc.CancelSalesOrderUseCase{Repo: soRepo, Auth: authService},
		&sales_order_uc.BlockSalesOrderUseCase{Repo: soRepo, Auth: authService},
		&sales_order_uc.UnblockSalesOrderUseCase{Repo: soRepo, Auth: authService},
		&sales_order_uc.ChangeStatusSalesOrderUseCase{Repo: soRepo, Auth: authService},
		&sales_order_uc.CreateSalesOrderItemUseCase{Repo: soRepo, Auth: authService},
		&sales_order_uc.UpdateSalesOrderItemUseCase{Repo: soRepo, Auth: authService},
		&sales_order_uc.ListSalesOrderItemsUseCase{Repo: soRepo, Auth: authService},
		&sales_order_uc.CancelSalesOrderItemUseCase{Repo: soRepo, Auth: authService},
	)

	// stock management
	stockRepository := stockRepo.NewStockRepositorySQLC(app.db.Pool)
	createStockMovementUC := &stock_uc.CreateStockMovementUseCase{Repo: stockRepository, Auth: authService}
	listStockMovementsUC := &stock_uc.ListStockMovementsUseCase{Repo: stockRepository, Auth: authService}
	getStockBalanceUC := &stock_uc.GetStockBalanceUseCase{Repo: stockRepository, Auth: authService}
	reserveStockUC := &stock_uc.ReserveStockUseCase{Repo: stockRepository, Auth: authService}
	releaseReserveUC := &stock_uc.ReleaseReservationUseCase{Repo: stockRepository, Auth: authService}
	consumeReserveUC := &stock_uc.ConsumeReservationUseCase{Repo: stockRepository, Auth: authService}
	createInventoryUC := &stock_uc.CreateInventoryUseCase{Repo: stockRepository, Auth: authService}
	countInventoryUC := &stock_uc.CountInventoryItemUseCase{Repo: stockRepository, Auth: authService}
	adjustInventoryUC := &stock_uc.AdjustInventoryUseCase{Repo: stockRepository, Auth: authService}
	closeInventoryUC := &stock_uc.CloseInventoryUseCase{Repo: stockRepository, Auth: authService}
	getInventoryUC := &stock_uc.GetInventoryUseCase{Repo: stockRepository, Auth: authService}
	listInventoriesUC := &stock_uc.ListInventoriesUseCase{Repo: stockRepository, Auth: authService}
	stockHandler := handler.NewStockHandler(
		createStockMovementUC,
		listStockMovementsUC,
		getStockBalanceUC,
		reserveStockUC,
		releaseReserveUC,
		consumeReserveUC,
		createInventoryUC,
		countInventoryUC,
		adjustInventoryUC,
		closeInventoryUC,
		getInventoryUC,
		listInventoriesUC,
	)

	// financial
	fRepo := financialRepo.NewFinancialRepositoryPG(app.db.Pool)
	fiscalRepository := fiscalRepo.NewFiscalRepositoryPG(app.db.Pool)
	financialHandler := handler.NewFinancialHandler(
		&financial_uc.CreateContaBancariaUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.ListContasBancariasUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.CreateCondicaoPagamentoUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.ListCondicoesPagamentoUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.CreatePlanoContasUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.ListPlanoContasUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.CreateCentroCustoUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.ListCentrosCustoUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.CreateContaPagarUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.ListContasPagarUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetContaPagarUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.ApproveContaPagarUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.BaixarContaPagarUseCase{Repo: fRepo, Auth: authService, FiscalRepo: fiscalRepository},
		&financial_uc.CancelContaPagarUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetAgingPagarUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.CreateContaReceberUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.ListContasReceberUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetContaReceberUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.BaixarContaReceberUseCase{Repo: fRepo, Auth: authService, FiscalRepo: fiscalRepository},
		&financial_uc.CancelContaReceberUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetAgingReceberUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetFluxoCaixaUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetFluxoProjetadoUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetSaldoContasUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.ApurarImpostosUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetTaxAssessmentUseCase{Repo: fRepo, Auth: authService},
		// Reports
		&financial_uc.GetLivroEntradasUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetLivroSaidasUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetImpostosSaidasUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetImpostosEntradasUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetDREUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetAgingReceberDetalhadoUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetAgingPagarDetalhadoUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetExtratoPorFornecedorUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetExtratoPorClienteUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetProdutosVendidosUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetProdutosProduzidosUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetHistoricoCustosUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetFichaTecnicaCustoUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetCurvaABCClientesUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetCurvaABCProdutosUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetComprasPeriodoUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.ImportarOFXUseCase{Repo: fRepo, Auth: authService},
	)

	// fiscal module
	fiscalHandler := handler.NewFiscalHandler(
		&fiscalUC.CreateFiscalEntryUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.UploadNFEEntryUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.ApproveFiscalEntryUseCase{FiscalRepo: fiscalRepository, FinancialRepo: fRepo, Auth: authService},
		&fiscalUC.ListFiscalEntriesUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.GetFiscalEntryUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.CreateFiscalExitUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.AuthorizeFiscalExitUseCase{Repo: fiscalRepository, FinancialRepo: fRepo, Auth: authService},
		&fiscalUC.CancelFiscalExitUseCase{Repo: fiscalRepository, FinancialRepo: fRepo, Auth: authService},
		&fiscalUC.ListFiscalExitsUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.GetFiscalExitUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.GetFiscalConfigUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.UpdateFiscalConfigUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.EmitirCCeUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.CreateCTeUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.ListCTeUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.GetCTeUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.UpsertNcmTaxUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.ListNcmTaxesUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.DeleteNcmTaxUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.UpsertICMSInterstateUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.ListICMSInterstateUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.UpsertICMSInternalUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.ListICMSInternalUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.ConsultarNFeUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.ListCartasCorrecaoUseCase{Repo: fiscalRepository, Auth: authService},
	)

	// routes
	r.Group(func(r chi.Router) {
		r.Use(httpmw.JWT(app.config.JWTSecret, app.logger))
		r.Route("/api/items", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", itemHandler.CreateItem)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/search/{code}", itemHandler.FindItemByCodeHandler)

			r.Route("/mask", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/generate", generateMaskItemHandler.GenerateMask)
			})
			r.Route("/structure", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", structureHandler.Create)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/update", structureHandler.Update)
				//r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{parentItemCode}/children", structureHandler.GetAllDirectChildren)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/resolve/{itemCode}", queryStructureHandler.ResolveStructure)
			})
		})
		r.Route("/api/allocations", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", allocationBaseHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", allocationBaseHandler.List)
		})
		r.Route("/api/cost-center", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", costCenterHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", costCenterHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{costCenterCode}", costCenterHandler.Get)
		})
		r.Route("/api/delivery-promise-params", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", deliveryPromiseParamsHandler.Get)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/update", deliveryPromiseParamsHandler.Update)
		})
		r.Route("/api/delivery-reschedule", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", deliveryRescheduleHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list/{sales_order_code}", deliveryRescheduleHandler.ListByOrder)
		})
		r.Route("/api/independent-demand", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", independentDemandHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/update/{code}", independentDemandHandler.Update)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/delete/{code}", independentDemandHandler.Delete)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list-from-date/{date}", independentDemandHandler.ListFromDate)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list-by-item/{itemCode}", independentDemandHandler.ListByItem)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", independentDemandHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/get-by-code/{code}", independentDemandHandler.GetByCode)
		})
		r.Route("/api/industrial-calendar", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", industrialCalendarHandler.CreateDay)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/month/{year}/{month}", industrialCalendarHandler.GetMonth)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/workdays/{year}/{month}", industrialCalendarHandler.GetWorkdays)
		})
		r.Route("/api/machine", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", machineHandler.CreateMachine)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", machineHandler.ListMachines)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", machineHandler.GetMachineByCode)
			r.Route("/types", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", machineHandler.CreateType)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", machineHandler.ListTypes)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", machineHandler.GetTypeByCode)
			})
			r.Route("/time", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", machineHandler.CreateItemTime)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", machineHandler.ListItemTimes)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{code}", machineHandler.GetItemTime)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/production/calculate", machineHandler.CalculateProductionTime)
			})
			r.Route("/schedule", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", machineHandler.CreateSchedule)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", machineHandler.ListSchedules)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{code}", machineHandler.GetSchedule)
			})
		})
		r.Route("/api/mrp-calculation", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/run", mrpHandler.Run)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/profile/{item_code}/{plan_code}", mrpHandler.GetProfile)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/configured-rules", mrpHandler.CreateConfiguredRule)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/configured-rules/{item_code}", mrpHandler.ListConfiguredRules)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/exceptions/{plan_code}", mrpHandler.ListExceptions)
		})
		r.Route("/api/item-calendar-promise", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", itemCalendarPromiseHandler.UpsertDay)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{item_code}/{mask}/{year}/{month}", itemCalendarPromiseHandler.ListMonth)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{item_code}/{mask}/{year}/{month}/workdays", itemCalendarPromiseHandler.GetWorkdays)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{item_code}/{mask}/{year}/{month}/{day}", itemCalendarPromiseHandler.GetDay)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/{item_code}/{mask}/{year}/{month}/{day}", itemCalendarPromiseHandler.DeleteDay)
		})
		r.Route("/api/order-priority", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", opHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", opHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/find/{value}", opHandler.FindByValue)
		})
		r.Route("/api/overhead-allocation", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", overHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", overHandler.List)
		})
		r.Route("/api/planned-order", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", plannedHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", plannedHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}/firm", plannedHandler.Firm)
		})
		r.Route("/api/production-order", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", prodOrderHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", prodOrderHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}", prodOrderHandler.GetByCode)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{id}/start", prodOrderHandler.Start)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/appointment", prodOrderHandler.AddAppointment)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/consumption", prodOrderHandler.AddConsumption)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{id}/complete", prodOrderHandler.Complete)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{id}/close", prodOrderHandler.Close)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{id}/cancel", prodOrderHandler.Cancel)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}/appointments", prodOrderHandler.GetAppointments)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}/consumptions", prodOrderHandler.GetConsumptions)
		})
		r.Route("/api/questions", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/questions/create", questionCreateHandler.CreateQuestion)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", findQuestionByNameHandler.FindQuestionByName)
			r.Route("/options", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create-option", questionOptionCreateHandler.CreateQuestionOptionHandler)
			})
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/associate", associateByQuestionItemHandler.AssociateQuestions)
		})
		r.Route("/api/bom", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", bomHandler.Create)
			r.Route("/bom-items", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", bomItemHandler.Create)
			})
		})
		r.Route("/api/warehouse", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", warehouseHandler.CreateWarehouse)
		})
		r.Route("/api/pdm", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create-group", groupHandler.CreateGroup)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create-modifier", modifierHandler.CreateModifier)
		})
		r.Route("/api/enterprise", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", enterpriseHandler.CreateEnterprise)
		})
		r.Route("/api/employee", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", employeeHandler.CreateEmployee)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", employeeHandler.ListEmployees)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", employeeHandler.GetEmployee)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/update", employeeHandler.UpdateEmployee)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/{code}/deactivate", employeeHandler.DeactivateEmployee)
		})
		r.Route("/api/planning-params", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", planningParamsHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{number}", planningParamsHandler.GetByNumber)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/update", planningParamsHandler.Update)
		})
		r.Route("/api/production-plan", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", productionPlanHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", productionPlanHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", productionPlanHandler.GetByCode)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/update", productionPlanHandler.Update)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/{code}", productionPlanHandler.Delete)
		})
		r.Route("/api/restriction", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", restrictionHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", restrictionHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", restrictionHandler.GetByCode)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/item/{itemCode}", restrictionHandler.GetByItem)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/{code}", restrictionHandler.Update)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Patch("/{code}/deactivate", restrictionHandler.Deactivate)
		})
		r.Route("/api/sales-division", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", salesDivisionHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", salesDivisionHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", salesDivisionHandler.GetByCode)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/{code}", salesDivisionHandler.Update)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/{code}", salesDivisionHandler.Delete)
		})
		r.Route("/api/sales-order", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", salesOrderHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", salesOrderHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", salesOrderHandler.GetByCode)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/{code}", salesOrderHandler.Update)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/{code}/cancel", salesOrderHandler.Cancel)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Patch("/{code}/block", salesOrderHandler.Block)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Patch("/{code}/unblock", salesOrderHandler.Unblock)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Patch("/{code}/status", salesOrderHandler.ChangeStatus)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/customer/{customerCode}", salesOrderHandler.ListByCustomer)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/status/{status}", salesOrderHandler.ListByStatus)
			r.Route("/items", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", salesOrderHandler.CreateItem)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", salesOrderHandler.ListItems)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/{itemCode}", salesOrderHandler.UpdateItem)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/{itemCode}/cancel", salesOrderHandler.CancelItem)
			})
		})
		r.Route("/api/purchase-order", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", purchaseOrderHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", purchaseOrderHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", purchaseOrderHandler.GetByCode)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/{code}", purchaseOrderHandler.Update)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/{code}/cancel", purchaseOrderHandler.Cancel)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/supplier/{supplierCode}", purchaseOrderHandler.ListBySupplier)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/status/{status}", purchaseOrderHandler.ListByStatus)
		})
		r.Route("/api/sales-forecast", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", salesForecastHandler.CreateForecast)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list/{year}", salesForecastHandler.ListForecasts)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/item/{itemCode}", salesForecastHandler.GetForecastByItem)
			r.Route("/blocks", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", salesForecastHandler.CreateBlock)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", salesForecastHandler.ListBlocks)
			})
			r.Route("/appropriation", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", salesForecastHandler.CreateAppropriation)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", salesForecastHandler.ListAppropriations)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/set-default", salesForecastHandler.SetDefaultAppropriation)
			})
		})
		r.Route("/api/stock", func(r chi.Router) {
			r.Route("/movements", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", stockHandler.CreateMovement)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", stockHandler.ListMovements)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/item/{itemCode}", stockHandler.ListMovementsByItem)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/warehouse/{warehouseId}", stockHandler.ListMovementsByWarehouse)
			})
			r.Route("/balances", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/get", stockHandler.GetBalance)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", stockHandler.ListBalances)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/warehouse/{warehouseId}", stockHandler.ListBalancesByWarehouse)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/item/{itemCode}", stockHandler.ListBalancesByItem)
			})
			r.Route("/reservations", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", stockHandler.ReserveStock)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Patch("/{id}/release", stockHandler.ReleaseReservation)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Patch("/{id}/consume", stockHandler.ConsumeReservation)
			})
			r.Route("/inventories", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", stockHandler.CreateInventory)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", stockHandler.ListInventories)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}", stockHandler.GetInventory)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{id}/close", stockHandler.CloseInventory)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/count", stockHandler.CountInventoryItem)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/adjust", stockHandler.AdjustInventoryItem)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}/items", stockHandler.ListInventoryItems)
			})
		})
		r.Route("/api/fiscal", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/entries/create", fiscalHandler.CreateEntry)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/entries/upload-nfe", fiscalHandler.UploadNFE)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/entries/{code}/approve", fiscalHandler.ApproveEntry)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/entries/list", fiscalHandler.ListEntries)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/entries/{code}", fiscalHandler.GetEntry)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/exits/create", fiscalHandler.CreateExit)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/exits/{code}/authorize", fiscalHandler.AuthorizeExit)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/exits/{code}/cancel", fiscalHandler.CancelExit)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/exits/{code}/carta-correcao", fiscalHandler.EmitirCCe)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/exits/list", fiscalHandler.ListExits)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/exits/{code}", fiscalHandler.GetExit)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/config", fiscalHandler.GetConfig)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/config", fiscalHandler.UpdateConfig)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/cte/create", fiscalHandler.CreateCTe)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/cte/list", fiscalHandler.ListCTe)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/cte/{code}", fiscalHandler.GetCTe)
			// NF-e status consultation & CC-e list
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/exits/{id}/status", fiscalHandler.ConsultarNFe)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/exits/{id}/cartas-correcao", fiscalHandler.ListCartasCorrecao)
			// NCM tax table management
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/tabelas/ncm", fiscalHandler.UpsertNcmTax)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/tabelas/ncm", fiscalHandler.ListNcmTaxes)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/tabelas/ncm/{ncm}", fiscalHandler.DeleteNcmTax)
			// ICMS table management
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/tabelas/icms-interestadual", fiscalHandler.UpsertICMSInterstate)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/tabelas/icms-interestadual", fiscalHandler.ListICMSInterstate)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/tabelas/icms-interno", fiscalHandler.UpsertICMSInternal)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/tabelas/icms-interno", fiscalHandler.ListICMSInternal)
		})
		r.Route("/api/financial", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/contas-bancarias/create", financialHandler.CreateContaBancaria)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/contas-bancarias/list", financialHandler.ListContasBancarias)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/condicoes-pagamento/create", financialHandler.CreateCondicaoPagamento)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/condicoes-pagamento/list", financialHandler.ListCondicoesPagamento)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/plano-contas/create", financialHandler.CreatePlanoContas)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/plano-contas/list", financialHandler.ListPlanoContas)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/centros-custo/create", financialHandler.CreateCentroCusto)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/centros-custo/list", financialHandler.ListCentrosCusto)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/contas-pagar/create", financialHandler.CreateContaPagar)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/contas-pagar/list", financialHandler.ListContasPagar)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/contas-pagar/{id}", financialHandler.GetContaPagar)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/contas-pagar/{id}/approve", financialHandler.ApproveContaPagar)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/contas-pagar/{id}/baixar", financialHandler.BaixarContaPagar)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/contas-pagar/{id}/cancel", financialHandler.CancelContaPagar)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/contas-pagar/aging", financialHandler.GetAgingPagar)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/contas-receber/create", financialHandler.CreateContaReceber)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/contas-receber/list", financialHandler.ListContasReceber)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/contas-receber/{id}", financialHandler.GetContaReceber)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/contas-receber/{id}/baixar", financialHandler.BaixarContaReceber)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/contas-receber/{id}/cancel", financialHandler.CancelContaReceber)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/contas-receber/aging", financialHandler.GetAgingReceber)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/fluxo-caixa", financialHandler.GetFluxoCaixa)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/fluxo-projetado", financialHandler.GetFluxoProjetado)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/saldo-contas", financialHandler.GetSaldoContas)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/apuracao-impostos", financialHandler.ApurarImpostos)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/apuracao-impostos/{competencia}", financialHandler.GetTaxAssessment)
			// Reports
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/relatorios/livro-entradas", financialHandler.GetLivroEntradas)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/relatorios/livro-saidas", financialHandler.GetLivroSaidas)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/relatorios/impostos-saidas", financialHandler.GetImpostosSaidas)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/relatorios/impostos-entradas", financialHandler.GetImpostosEntradas)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/relatorios/dre", financialHandler.GetDRE)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/relatorios/aging-receber", financialHandler.GetAgingReceberDetalhado)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/relatorios/aging-pagar", financialHandler.GetAgingPagarDetalhado)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/relatorios/extrato-fornecedor/{id}", financialHandler.GetExtratoPorFornecedor)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/relatorios/extrato-cliente/{id}", financialHandler.GetExtratoPorCliente)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/relatorios/produtos-vendidos", financialHandler.GetProdutosVendidos)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/relatorios/produtos-produzidos", financialHandler.GetProdutosProduzidos)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/relatorios/historico-custos", financialHandler.GetHistoricoCustos)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/relatorios/ficha-tecnica/{item_code}", financialHandler.GetFichaTecnicaCusto)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/relatorios/curva-abc-clientes", financialHandler.GetCurvaABCClientes)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/relatorios/curva-abc-produtos", financialHandler.GetCurvaABCProdutos)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/relatorios/compras-periodo", financialHandler.GetComprasPeriodo)
			// Bank statement reconciliation
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/conciliacao/{conta_id}/importar-ofx", financialHandler.ImportarOFX)
		})
	})
	// Health check
	r.Get("/health", app.healthHandler)

	return r
}

func (app *application) healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"mask":      "core-api",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		return
	}
}

func (app *application) run(r chi.Router) error {
	addr := app.config.ServerPort
	if addr == "" {
		addr = "5070"
	}
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
		IdleTimeout:  time.Minute,
	}

	app.logger.Info("server listening", "addr", addr)
	return srv.ListenAndServe()
}
