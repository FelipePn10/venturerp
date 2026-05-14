package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/restriction_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_division_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_forecast_uc"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/config"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database"
	applogger "github.com/FelipePn10/panossoerp/internal/infrastructure/logger"
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
	over "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/overhead_allocation"
	planned "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planned_order"
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

	registerUserUC := usecase.NewRegisterUserUseCase(userRepo)
	loginUserUC := usecase.NewLoginUserUseCase(userRepo)

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
	createQuestionUC := usecase.NewQuestionUserUseCase(questionRepo, authService)
	findQuestionByNameUC := usecase.NewFindQuestionByName(questionRepo)

	questionCreateHandler := handler.NewQuestionHandler(createQuestionUC)
	findQuestionByNameHandler := handler.NewFindQuestionByName(findQuestionByNameUC)

	// question option
	questionOptionRepo := questionsoptions.NewRepositoryQuestionOptionSQLC(queries)

	createQuestionOptionUC := usecase.NewCreateQuestionOptionUseCase(questionOptionRepo, authService)
	questionOptionCreateHandler := handler.NewCreateQuestionOptionHandler(createQuestionOptionUC)

	// associate question in item
	itemByQuestionItemRepo := itemquestion.NewAssociateQuestionItemRepositorySQLC(queries)
	associateByQuestionItemUC := usecase.NewAssociateByQuestionItemUseCase(itemByQuestionItemRepo, authService)
	associateByQuestionItemHandler := handler.NewAssociateByQuestionItemHandler(associateByQuestionItemUC)

	// generate mask item
	generateMaskItem := generatemask.NewRepositoryGenerateMaskSQLC(queries)
	generateMaskItemUC := usecase.NewGenerateMaskItemUseCase(generateMaskItem, authService)
	generateMaskItemHandler := handler.NewGeneratMaskItemHandler(generateMaskItemUC)

	// Item
	itemRepo := item.NewRepositoryItemSQLC(queries)
	createItemUc := usecase.NewCreateItem(itemRepo, authService)
	findItemByCodeUc := usecase.NewFindItemByCode(itemRepo, authService)
	itemHandler := handler.NewCreateItemHandler(createItemUc, findItemByCodeUc)

	// Item Structure
	itemRepoStructure := structure.NewItemStructureRepository(queries)
	createStructureUc := usecase.NewCreateStructureComponentUseCase(itemRepoStructure, authService)
	updateStructureUc := usecase.NewUpdateStructureComponentUseCase(itemRepoStructure, authService)
	getAllStructureUc := usecase.NewGetAllDirectChildrenUseCase(itemRepoStructure, authService)
	treeStructureUc := usecase.NewGetStructureTreeUseCase(itemRepoStructure, authService)
	structureHandler := handler.NewItemStructureHandler(createStructureUc, updateStructureUc, getAllStructureUc, treeStructureUc)

	// Item Structure Query
	itemRepoStructureQuery := structure_query.NewStructureQueryRepository(queries)
	queryStructureUc := usecase.NewResolveStructureQueryUseCase(itemRepoStructureQuery, authService)
	queryStructureHandler := handler.NewQueryStructureHandler(queryStructureUc)
	// bom
	bomRepo := bom.NewRepostioryBomSQLC(queries)

	createBomUc := usecase.NewCreateBomUseCase(bomRepo, authService)
	bomHandler := handler.NewCreateBomHandler(createBomUc)

	// bom item
	bomItemRepo := bomitem.NewRepositoryBomItemSQLC(queries)

	createBomItemUc := usecase.NewCreatBomItemUseCase(bomItemRepo, authService)
	bomItemHandler := handler.NewCreateBomItemHandler(createBomItemUc)

	// warehouse
	warehouseRepo := warehouse.NewRepositoryQuestionSQLC(queries)
	createWarehouseUc := usecase.NewCreateWarehouseUseCase(warehouseRepo, authService)
	warehouseHandler := handler.NewCreateWarehouseHandler(createWarehouseUc)

	// group
	groupRepo := group.NewRepositoryGroupSQLC(queries)
	createGroupUc := usecase.NewCreateGroupUseCase(groupRepo, authService)
	groupHandler := handler.NewCreateGroupHandler(createGroupUc)

	// enterprise
	enterpriseRepo := enterprise.NewRepositoryEnterpriseSQLC(queries)
	createEnterpriseUc := usecase.NewCreateEnterpriseUseCase(enterpriseRepo, authService)
	enterpriseHandler := handler.NewCreateEnterpriseHandler(createEnterpriseUc)

	// modifier
	modifierRepo := modifier.NewRepositoryModifierSQLC(queries)
	createModifierUc := usecase.NewCreateModifierUseCase(modifierRepo, authService)
	modifierHandler := handler.NewCreateModifierHandler(createModifierUc)

	// employee
	employeeRepo := employee.NewRepositoryEmployeeSQLC(queries)
	createEmployeeUc := usecase.NewCreateEmployeeUseCase(employeeRepo, authService)
	listEmployeesUC := usecase.NewListEmployeesUseCase(employeeRepo, authService)
	getEmployeeUC := usecase.NewGetEmployeeUseCase(employeeRepo, authService)
	updateEmployeeUC := usecase.NewUpdateEmployeeUseCase(employeeRepo, authService)
	deactivateEmployeeUC := usecase.NewDeactivateEmployeeUseCase(employeeRepo, authService)
	employeeHandler := handler.NewEmployeeHandler(createEmployeeUc, listEmployeesUC, getEmployeeUC, updateEmployeeUC, deactivateEmployeeUC)

	// planning params
	planningParamsRepo := planningParams.NewPlanningParamRepositorySQLC(queries)
	getPlanningParamUC := usecase.NewGetPlanningParamUseCase(planningParamsRepo, authService)
	listPlanningParamsUC := usecase.NewListPlanningParamsUseCase(planningParamsRepo, authService)
	updatePlanningParamUC := usecase.NewUpdatePlanningParamUseCase(planningParamsRepo, authService)
	planningParamsHandler := handler.NewPlanningParamsHandler(getPlanningParamUC, listPlanningParamsUC, updatePlanningParamUC)

	// production plan
	productionPlanRepo := productionPlan.NewProductionPlanRepositorySQLC(queries)
	createProductionPlanUC := usecase.NewCreateProductionPlanUseCase(productionPlanRepo, authService)
	getProductionPlanUC := usecase.NewGetProductionPlanUseCase(productionPlanRepo, authService)
	listProductionPlansUC := usecase.NewListProductionPlansUseCase(productionPlanRepo, authService)
	updateProductionPlanUC := usecase.NewUpdateProductionPlanUseCase(productionPlanRepo, authService)
	deleteProductionPlanUC := usecase.NewDeleteProductionPlanUseCase(productionPlanRepo, authService)
	productionPlanHandler := handler.NewProductionPlanHandler(createProductionPlanUC, getProductionPlanUC, listProductionPlansUC, updateProductionPlanUC, deleteProductionPlanUC)

	// restriction
	restrictionR := restrictionRepo.NewRestrictionRepositorySQLC(queries)
	createRestrictionUC := usecase.NewCreateRestrictionUseCase(restrictionR, authService)
	getRestrictionUC := usecase.NewGetRestrictionUseCase(restrictionR, authService)
	listRestrictionsUC := usecase.NewListRestrictionsUseCase(restrictionR, authService)
	getRestrictionsByItemUC := usecase.NewGetRestrictionsByItemUseCase(restrictionR, authService)
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
	createAllocationBaseUC := usecase.NewCreateAllocationBaseUseCase(allocationBaseRepo, authService)
	listAllocationBaseUC := usecase.NewListAllocationBasesUseCase(allocationBaseRepo, authService)
	allocationBaseHandler := handler.NewAllocationBaseHandler(createAllocationBaseUC, listAllocationBaseUC)

	// cost center
	costCenterRepo := cost_center.NewCostCenterRepositorySQLC(queries)
	createCostCenterUC := usecase.NewCreateCostCenterUseCase(costCenterRepo, authService)
	listCostCenterUC := usecase.NewListCostCentersUseCase(costCenterRepo, authService)
	getCostCenterUC := usecase.NewGetCostCenterUseCase(costCenterRepo, authService)
	costCenterHandler := handler.NewCostCenterHandler(createCostCenterUC, listCostCenterUC, getCostCenterUC)

	// delivery promise params
	deliveryPromiseParamsRepo := deliveryPromiseParams.NewDeliveryPromiseParamsRepositorySQLC(queries)
	manageDeliveryPromiseParamsUC := usecase.NewManageDeliveryPromiseParamsUseCase(deliveryPromiseParamsRepo, authService)
	deliveryPromiseParamsHandler := handler.NewDeliveryPromiseParamsHandler(manageDeliveryPromiseParamsUC)

	// delivery reschedule
	deliveryRescheduleRepo := deliveryReschedule.NewDeliveryRescheduleRepositorySQLC(queries)
	createDeliveryRescheduleUC := usecase.NewCreateDeliveryRescheduleUseCase(deliveryRescheduleRepo, authService)
	listDeliveryRescheduleUC := usecase.NewListDeliveryReschedulesUseCase(deliveryRescheduleRepo, authService)
	deliveryRescheduleHandler := handler.NewDeliveryRescheduleHandler(createDeliveryRescheduleUC, listDeliveryRescheduleUC)

	// independent demand
	independentDemandRepo := independentDemand.NewIndependentDemandRepositorySQLC(queries)
	createIndependentDemandUC := usecase.NewCreateIndependentDemandUseCase(independentDemandRepo, authService)
	updateIndependentDemandUC := usecase.NewUpdateIndependentDemandUseCase(independentDemandRepo, authService)
	deleteIndependentDemandUC := usecase.NewDeleteIndependentDemandUseCase(independentDemandRepo, authService)
	listFromDateIndependentDemandUC := usecase.NewListIndependentDemandFromDateUseCase(independentDemandRepo, authService)
	listByItemIndependentDemandUC := usecase.NewListIndependentDemandByItemUseCase(independentDemandRepo, authService)
	listIndependentDemandUC := usecase.NewListIndependentDemandsUseCase(independentDemandRepo, authService)
	getByCodeDemandUC := usecase.NewGetIndependentDemandByCodeUseCase(independentDemandRepo, authService)
	independentDemandHandler := handler.NewIndependentDemandHandler(createIndependentDemandUC, updateIndependentDemandUC, deleteIndependentDemandUC, listFromDateIndependentDemandUC, listByItemIndependentDemandUC, listIndependentDemandUC, getByCodeDemandUC)

	// industrial calendar
	industrialCalendarRepo := industrialCalendar.NewIndustrialCalendarRepositorySQLC(queries)
	manageIndustrialCalendarRepoUC := usecase.NewManageCalendarUseCase(industrialCalendarRepo, authService)
	industrialCalendarHandler := handler.NewIndustrialCalendarHandler(manageIndustrialCalendarRepoUC)

	// item calendar promise
	itemCalendarPromise := itemCalendarPromise.NewItemCalendarPromiseRepositorySQLC(queries)
	itemCalendarPromiseUC := usecase.NewManageItemCalendarPromiseUseCase(itemCalendarPromise, authService)
	itemCalendarPromiseHandler := handler.NewItemCalendarPromiseHandler(itemCalendarPromiseUC)

	// machine
	machineRepo := machine.NewMachineRepositorySQLC(queries)
	machineUC := usecase.NewCreateMachineUseCase(machineRepo, authService)
	machineListUC := usecase.NewListMachinesUseCase(machineRepo, authService)
	machineGetByCodeUC := usecase.NewGetMachineUseCase(machineRepo, authService)
	//type
	machineTypeCreateUC := usecase.NewCreateMachineTypeUseCase(machineRepo, authService)
	machineListTypesUC := usecase.NewListMachineTypesUseCase(machineRepo, authService)
	machineTypeGetByCodeUC := usecase.NewGetMachineTypeUseCase(machineRepo, authService)
	//item times
	machineItemTimeUC := usecase.NewCreateItemMachineTimeUseCase(machineRepo, itemRepo, authService)
	machineListItemTimeUC := usecase.NewListItemMachineTimesUseCase(machineRepo, authService)
	//machineGetItemTimeUC := usecase.NewGetItemMachineTimeUseCase(machineRepo, authService)
	machineCalcProductionUC := usecase.NewCalculateProductionTimeUseCase(machineRepo, itemRepo, authService)
	// schedule
	scheduleUC := usecase.NewScheduleMachineUseCase(machineRepo, authService)

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
	mrpRepo := mrpCalculation.NewMRPCalculationRepositorySQLC(queries)
	supplyPort := planned.NewPlannedOrderSupplyAdapter(queries)
	mrpService := usecase.NewMRPService(mrpRepo, itemRepoStructure, independentDemandRepo, industrialCalendarRepo, itemRepo, supplyPort, productionPlanRepo, sfRepo, restrictionR)
	mrpRunUC := usecase.NewRunMRPCalculationUseCase(mrpService, authService)
	mrpGetProfileUC := usecase.NewGetItemProfileUseCase(mrpRepo, authService)
	mrpCreateConfiguredRule := usecase.NewManageConfiguredItemRulesUseCase(mrpRepo, authService)
	mrpListExceptionsUC := usecase.NewListMRPExceptionsUseCase(mrpRepo, authService)
	mrpHandler := handler.NewMRPCalculationHandler(mrpRunUC, mrpGetProfileUC, mrpCreateConfiguredRule, mrpListExceptionsUC)

	//order priority
	opRepo := op.NewOrderPriorityRepositorySQLC(queries)
	opCreateUC := usecase.NewCreateOrderPriorityUseCase(opRepo, authService)
	opListUC := usecase.NewListOrderPrioritiesUseCase(opRepo, authService)
	opFindUC := usecase.NewFindPriorityByValueUseCase(opRepo, authService)
	opHandler := handler.NewOrderPriorityHandler(opCreateUC, opListUC, opFindUC)

	// overhead allocation
	overRepo := over.NewOverheadAllocationRepositorySQLC(queries)
	overCreateUC := usecase.NewCreateOverheadAllocationUseCase(overRepo, authService)
	overListUC := usecase.NewListOverheadAllocationsUseCase(overRepo, authService)
	overHandler := handler.NewOverheadAllocationHandler(overCreateUC, overListUC)

	// planned order
	plannedRepo := planned.NewPlannedOrderRepositorySQLC(queries)
	plannedCreateUC := usecase.NewCreatePlannedOrderUseCase(plannedRepo, authService)
	plannedListUC := usecase.NewListPlannedOrdersUseCase(plannedRepo, authService)
	plannedFirmUC := usecase.NewFirmPlannedOrderUseCase(plannedRepo, authService)
	plannedHandler := handler.NewPlannedOrderHandler(plannedCreateUC, plannedListUC, plannedFirmUC)

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
