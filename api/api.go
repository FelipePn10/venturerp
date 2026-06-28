package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	accounting_uc "github.com/FelipePn10/panossoerp/internal/application/usecase/accounting_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/allocation_base_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/aps_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/bom_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/cnpj_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/cost_center_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/cost_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/crp_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/customer_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/cutting_plan_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_promise_params_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_reschedule_uc"
	employeeUC "github.com/FelipePn10/panossoerp/internal/application/usecase/employee"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/enterprise_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/entry_operation_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_classification_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_params_uc"
	fiscalUC "github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/generate_mask_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/group_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/ibpt_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/independent_demand_uc"
	industrial_calendar_uc "github.com/FelipePn10/panossoerp/internal/application/usecase/industrial_calendar"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_calendar_promise_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_classification_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_conversion_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_supplier_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/location_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/machine_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/maintenance_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/modifier_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_calculation_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/nfse_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/order_priority_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/overhead_allocation_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planned_order_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planning_params_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planning_uc"
	productionOrderUc "github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/production_plan_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_price_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_quotation_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_requisition_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/quality_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/question_option_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/question_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/restriction_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/routing_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_division_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_forecast_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/shipment_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/stock_movement_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/stock_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/structure_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/supplier_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/user_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/warehouse_uc"
	mrpservice "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/service"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/audit"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	cnpjinfra "github.com/FelipePn10/panossoerp/internal/infrastructure/cnpj"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/config"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database"
	applogger "github.com/FelipePn10/panossoerp/internal/infrastructure/logger"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/nesting"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/notification"
	accountingRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/accounting"
	allocation "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/allocation_base"
	apsRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/aps"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/bom"
	bomitem "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/bom_item"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/cost_center"
	crpRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/crp"
	customerRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer"
	cuttingPlanRepository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/cutting_plan"
	deliveryPromiseParams "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_promise_params"
	deliveryReschedule "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_reschedule"
	employee "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/employee"
	enterprise "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/enterprise"
	entryOperationRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/entry_operation"
	financialRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	fiscalParamsRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	fiscalRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	fiscalClassRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal_classification"
	generatemask "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/generate_mask"
	group "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/group"
	ibptRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/ibpt"
	independentDemand "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/independent_demand"
	industrialCalendar "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/industrial_calendar"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	itemCalendarPromise "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_calendar_promise"
	itemClassificationRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_classification"
	itemConversionRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_conversion"
	itemquestion "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_question"
	itemSupplierRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_supplier"
	locationRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/location"
	machine "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/machine"
	maintenanceRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/maintenance"
	modifier "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/modifier"
	mrpCalculation "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation"
	nfseRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/nfse"
	op "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/order_priority"
	over "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/overhead_allocation"
	planned "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planned_order"
	planningParams "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planning_params"
	productionOrderRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	productionPlan "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_plan"
	purchaseOrderRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_order"
	purchasePriceRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_price"
	purchaseQuotationRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_quotation"
	purchaseReqRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_requisition"
	qualityRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/quality"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/questions"
	questionsoptions "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/questions_options"
	restrictionRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/restriction"
	routingRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing"
	salesDivisionRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_division"
	salesForecastRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_forecast"
	salesOrderRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	shipmentRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/shipment"
	standardCostRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/standard_cost"
	stockRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	stockMovementRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock_movement"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure_query"
	supplierRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/supplier"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/user"
	warehouse "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/warehouse"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler"
	httpmw "github.com/FelipePn10/panossoerp/internal/interfaces/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type application struct {
	config    *config.Config
	logger    *applogger.Logger
	db        *database.DB
	metrics   *httpmw.Metrics
	auditSink *audit.PgSink
}

func (app *application) mount() chi.Router {
	r := chi.NewRouter()

	r.Use(httpmw.CorrelationMiddleware)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(httpmw.SecurityHeaders)
	r.Use(httpmw.CORS(app.corsOrigins(), app.config.IsDevelopment() && app.config.CORSAllowedOrigins == ""))
	r.Use(httpmw.MaxBodyBytes(app.config.MaxBodyBytes))
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.StripSlashes)
	if app.metrics != nil {
		r.Use(app.metrics.Middleware)
	}
	r.Use(httpmw.RequestLoggerMiddleware(app.logger))
	r.Use(httpmw.NewRateLimiter(float64(app.config.RateLimitRPS), float64(app.config.RateLimitBurst)).Middleware)

	// Auth endpoints get a stricter, separate bucket to blunt credential
	// stuffing / brute force, independent of the global API budget.
	authLimiter := httpmw.NewRateLimiter(float64(app.config.AuthRateLimitRPM)/60.0, float64(app.config.AuthRateLimitBurst))

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
		r.Use(authLimiter.Middleware)
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
	listOptionsByQuestionUC := question_option_uc.NewListOptionsByQuestionUseCase(questionOptionRepo, authService)
	questionOptionCreateHandler := handler.NewCreateQuestionOptionHandler(createQuestionOptionUC, listOptionsByQuestionUC)

	// Item
	itemRepo := item.NewRepositoryItemSQLC(queries)

	// associate question in item
	itemByQuestionItemRepo := itemquestion.NewAssociateQuestionItemRepositorySQLC(queries)
	associateByQuestionItemUC := question_uc.NewAssociateByQuestionItemUseCase(itemByQuestionItemRepo, itemRepo, authService)
	getQuestionsByItemUC := question_uc.NewGetQuestionsByItemUseCase(itemByQuestionItemRepo, authService)
	listAllItemQuestionsUC := question_uc.NewListAllItemQuestionsUseCase(itemByQuestionItemRepo, authService)
	associateByQuestionItemHandler := handler.NewAssociateByQuestionItemHandler(associateByQuestionItemUC, getQuestionsByItemUC, listAllItemQuestionsUC)

	// generate mask item
	generateMaskItem := generatemask.NewRepositoryGenerateMaskSQLC(queries)
	generateMaskItemUC := generate_mask_uc.NewGenerateMaskItemUseCase(generateMaskItem, authService)
	// Evaluator is set after evaluateRestrictionsUC is declared (see restriction block below).
	generateMaskItemHandler := handler.NewGeneratMaskItemHandler(generateMaskItemUC)
	createItemUc := item_uc.NewCreateItemUseCase(itemRepo, authService)
	findItemByCodeUc := item_uc.NewFindItemByCode(itemRepo, authService)
	listItemsUC := item_uc.NewListItemsUseCase(itemRepo, authService)
	listItemsWithMasksUC := item_uc.NewListItemsWithMasksUseCase(itemRepo, authService)
	itemHandler := handler.NewCreateItemHandler(createItemUc, findItemByCodeUc, listItemsUC, listItemsWithMasksUC)

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
	consultStructureUc := structure_uc.NewConsultStructureUseCase(itemRepoStructureQuery)
	whereUsedUc := structure_uc.NewWhereUsedUseCase(itemRepoStructureQuery)
	queryStructureHandler := handler.NewQueryStructureHandler(queryStructureUc, consultStructureUc, whereUsedUc)
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
	listWarehousesUc := warehouse_uc.NewListWarehousesUseCase(warehouseRepo, authService)
	getWarehouseUc := warehouse_uc.NewGetWarehouseUseCase(warehouseRepo, authService)
	warehouseHandler := handler.NewCreateWarehouseHandler(createWarehouseUc, listWarehousesUc, getWarehouseUc)

	// group
	groupRepo := group.NewRepositoryGroupSQLC(queries)
	createGroupUc := group_uc.NewCreateGroupUseCase(groupRepo, authService)
	groupHandler := handler.NewCreateGroupHandler(createGroupUc)

	// enterprise
	enterpriseRepo := enterprise.NewRepositoryEnterpriseSQLC(queries)
	createEnterpriseUc := enterprise_uc.NewCreateEnterpriseUseCase(enterpriseRepo, authService)
	enterpriseHandler := handler.NewCreateEnterpriseHandler(createEnterpriseUc)

	// CNPJ auto-lookup (cadastro auto-fill) + generic report export
	cnpjProvider := cnpjinfra.New(cnpjinfra.Config{
		Provider:     app.config.CNPJProvider,
		BrasilAPIURL: app.config.CNPJBrasilAPIURL,
		CNPJaURL:     app.config.CNPJaURL,
		Timeout:      time.Duration(app.config.CNPJTimeoutSec) * time.Second,
	})
	cnpjHandler := handler.NewCNPJHandler(cnpj_uc.NewLookupCNPJUseCase(cnpjProvider))
	// The generic report export brands its output with the company's fiscal data.
	reportExportHandler := handler.NewReportExportHandler(fiscalRepo.NewFiscalRepositoryPG(app.db.Pool))

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
	restrictionReasonR := restrictionRepo.NewRestrictionReasonRepositorySQLC(queries)
	createRestrictionUC := &restriction_uc.CreateRestrictionUseCase{Repo: restrictionR, Auth: authService}
	getRestrictionUC := &restriction_uc.GetRestrictionUseCase{Repo: restrictionR, Auth: authService}
	listRestrictionsUC := &restriction_uc.ListRestrictionsUseCase{Repo: restrictionR, Auth: authService}
	getRestrictionsByItemUC := &restriction_uc.GetRestrictionsByItemUseCase{Repo: restrictionR, Auth: authService}
	getRestrictionsByCustomerUC := &restriction_uc.GetRestrictionsByCustomerUseCase{Repo: restrictionR, Auth: authService}
	updateRestrictionUC := &restriction_uc.UpdateRestrictionUseCase{Repo: restrictionR, Auth: authService}
	deactivateRestrictionUC := &restriction_uc.DeactivateRestrictionUseCase{Repo: restrictionR, Auth: authService}
	evaluateRestrictionsUC := &restriction_uc.EvaluateRestrictionsUseCase{Repo: restrictionR}
	// Wire restriction evaluator into generate mask so restrictions are enforced on mask creation.
	generateMaskItemUC.Evaluator = evaluateRestrictionsUC
	restrictionHandler := handler.NewRestrictionHandler(
		createRestrictionUC, getRestrictionUC, listRestrictionsUC,
		getRestrictionsByItemUC, getRestrictionsByCustomerUC,
		updateRestrictionUC, deactivateRestrictionUC, evaluateRestrictionsUC,
	)
	restrictionReasonHandler := handler.NewRestrictionReasonHandler(
		&restriction_uc.CreateRestrictionReasonUseCase{Repo: restrictionReasonR, Auth: authService},
		&restriction_uc.GetRestrictionReasonUseCase{Repo: restrictionReasonR, Auth: authService},
		&restriction_uc.ListRestrictionReasonsUseCase{Repo: restrictionReasonR, Auth: authService},
		&restriction_uc.UpdateRestrictionReasonUseCase{Repo: restrictionReasonR, Auth: authService},
		&restriction_uc.DeleteRestrictionReasonUseCase{Repo: restrictionReasonR, Auth: authService},
	)

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

	// routing (manufacturing routes)
	rRepo := routingRepo.New(queries)
	routingOperationUC := routing_uc.NewOperationUseCase(rRepo)
	routingRouteUC := routing_uc.NewRouteUseCase(rRepo)
	routingLeadTimeUC := routing_uc.NewLeadTimeUseCase(rRepo)
	routingHandler := handler.NewRoutingHandler(routingOperationUC, routingRouteUC, routingLeadTimeUC)

	// cutting plan repo (handler wired after the stock repository is built — fase 2)
	cuttingPlanRepo := cuttingPlanRepository.New(queries, app.db.Pool)

	// quality
	qRepo := qualityRepo.New(queries)
	qualityUC := quality_uc.New(qRepo)
	qualityHandler := handler.NewQualityHandler(qualityUC)

	// standard cost
	scRepo := standardCostRepo.New(queries)
	standardCostUC := cost_uc.New(scRepo)
	standardCostHandler := handler.NewStandardCostHandler(standardCostUC)

	// CRP
	crpRepository := crpRepo.New(queries)
	crpUC := crp_uc.New(crpRepository)
	crpHandler := handler.NewCRPHandler(crpUC)
	// maintenance repo wired after it is created (see below)

	// APS
	apsRepository := apsRepo.New(queries)
	apsUC := aps_uc.New(apsRepository).WithCalendar(industrialCalendarRepo)
	apsHandler := handler.NewAPSHandler(apsUC, fiscalRepo.NewFiscalRepositoryPG(app.db.Pool))

	// mrp_calculation
	mrpRepo := mrpCalculation.NewMRPCalculationRepositorySQLC(queries, app.db.Pool)
	supplyPort := planned.NewPlannedOrderSupplyAdapter(queries)
	mrpService := mrpservice.NewMRPService(mrpRepo, itemRepoStructure, independentDemandRepo, industrialCalendarRepo, itemRepo, supplyPort, productionPlanRepo, sfRepo, restrictionR, rRepo)
	mrpRunUC := &mrp_calculation_uc.RunMRPCalculationUseCase{Service: mrpService, Auth: authService}
	mrpGetProfileUC := &mrp_calculation_uc.GetItemProfileUseCase{Repo: mrpRepo, Auth: authService}
	mrpCreateConfiguredRule := &mrp_calculation_uc.ManageConfiguredItemRulesUseCase{Repo: mrpRepo, Auth: authService}
	mrpListExceptionsUC := &mrp_calculation_uc.ListMRPExceptionsUseCase{Repo: mrpRepo, Auth: authService}

	// planning pipeline (MRP → CRP → APS in one shot)
	planningPipelineUC := &planning_uc.RunPlanningPipelineUseCase{MRP: mrpRunUC, CRP: crpUC, APS: apsUC}
	planningHandler := handler.NewPlanningHandler(planningPipelineUC)

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

	mrpFirmarSugestaoUC := &mrp_uc.FirmarSugestaoMRPUseCase{MRPRepo: mrpRepo, PlannedRepo: plannedRepo, Auth: authService}
	mrpHandler := handler.NewMRPCalculationHandler(mrpRunUC, mrpGetProfileUC, mrpCreateConfiguredRule, mrpListExceptionsUC, mrpFirmarSugestaoUC)

	// production order
	prodOrderRepo := productionOrderRepo.NewProductionOrderRepositoryPGX(app.db.Pool)
	// Firming a PRODUCTION planned order auto-creates its OF.
	plannedFirmUC.ProdOrderRepo = prodOrderRepo
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
	// Custo real da OF: apuração (material a custo médio + conversão por horas
	// apontadas × custo/hora do CT) e consulta. Fechar a OF apura automaticamente.
	prodOrderSettleCostUC := &productionOrderUc.SettleProductionCostUseCase{Repo: prodOrderRepo, Auth: authService, StdCostRepo: scRepo}
	prodOrderGetCostUC := &productionOrderUc.GetProductionCostUseCase{Repo: prodOrderRepo, Auth: authService}
	prodOrderCloseUC.SettleUC = prodOrderSettleCostUC
	// Scrap return (sucata valorizada). StockRepo is wired below once available.
	prodOrderReturnScrapUC := &productionOrderUc.ReturnScrapUseCase{Repo: prodOrderRepo, Auth: authService}
	orderOpsUC := &productionOrderUc.OrderOperationsUseCase{Q: queries}
	prodOrderHandler := handler.NewProductionOrderHandler(
		prodOrderCreateUC, prodOrderGetByCodeUC, prodOrderListUC,
		prodOrderStartUC, prodOrderAddAppointmentUC, prodOrderAddConsumptionUC,
		prodOrderCompleteUC, prodOrderCloseUC, prodOrderCancelUC,
		prodOrderGetAppointmentsUC, prodOrderGetConsumptionsUC,
	).WithOrderOps(orderOpsUC).WithCost(prodOrderSettleCostUC, prodOrderGetCostUC).WithScrap(prodOrderReturnScrapUC)

	// supplier (created before purchase order so it can provide purchasing defaults)
	suppRepo := supplierRepo.New(queries, app.db.Pool)
	supplierUC := supplier_uc.NewSupplierUseCase(suppRepo)
	supplierHandler := handler.NewSupplierHandler(supplierUC)

	// fiscal classifications (Cadastro de Classificações Fiscais)
	fiscalClassUC := fiscal_classification_uc.NewFiscalClassificationUseCase(fiscalClassRepo.New(queries, app.db.Pool))
	fiscalClassHandler := handler.NewFiscalClassificationHandler(fiscalClassUC)

	// entry operation types + state groups (Cadastro de Tipos de Operação de Entrada)
	entryOperationUC := entry_operation_uc.NewEntryOperationUseCase(entryOperationRepo.New(queries, app.db.Pool))
	entryOperationHandler := handler.NewEntryOperationHandler(entryOperationUC)

	// item unit conversions (Cadastro de Conversões por Item)
	itemConversionUC := item_conversion_uc.NewItemConversionUseCase(itemConversionRepo.New(queries, app.db.Pool))
	itemConversionHandler := handler.NewItemConversionHandler(itemConversionUC)

	// purchase price tables (Tabela de Preço de Compra)
	purchasePriceUC := purchase_price_uc.NewPurchasePriceUseCase(purchasePriceRepo.New(queries, app.db.Pool))
	purchasePriceHandler := handler.NewPurchasePriceHandler(purchasePriceUC)

	// preferred supplier per item (Fornecedor preferencial / Descrição por fornecedor)
	itemSupplierUC := item_supplier_uc.NewItemSupplierUseCase(itemSupplierRepo.New(queries, app.db.Pool))
	itemSupplierHandler := handler.NewItemSupplierHandler(itemSupplierUC)

	// item activation readiness (cross-validation BOM/routing/supplier/UOM)
	itemActivationUC := &item_uc.ValidateItemActivationUseCase{
		ItemRepo:      itemRepo,
		StructureRepo: itemRepoStructure,
		RoutingRepo:   rRepo,
		Suppliers:     itemSupplierUC,
		Conversions:   itemConversionRepo.New(queries, app.db.Pool),
		Auth:          authService,
	}
	itemActivationHandler := handler.NewItemActivationHandler(itemActivationUC)

	// purchase order
	poRepo := purchaseOrderRepo.NewPurchaseOrderRepositorySQLC(app.db.Pool)
	purchaseOrderHandler := handler.NewPurchaseOrderHandler(
		&purchase_order_uc.CreatePurchaseOrderUseCase{Repo: poRepo, Auth: authService, SupplierDefaults: supplierUC},
		&purchase_order_uc.UpdatePurchaseOrderUseCase{Repo: poRepo, Auth: authService},
		&purchase_order_uc.GetPurchaseOrderUseCase{Repo: poRepo, Auth: authService},
		&purchase_order_uc.ListPurchaseOrdersUseCase{Repo: poRepo, Auth: authService},
		&purchase_order_uc.ListPurchaseOrdersBySupplierUseCase{Repo: poRepo, Auth: authService},
		&purchase_order_uc.ListPurchaseOrdersByStatusUseCase{Repo: poRepo, Auth: authService},
		&purchase_order_uc.CancelPurchaseOrderUseCase{Repo: poRepo, Auth: authService},
	)

	// purchase order item with cross-module resolution (price table / UOM / IPI).
	// Providers are wired after their use cases are created below.
	purchaseOrderItemHandler := handler.NewPurchaseOrderItemHandler(
		&purchase_order_uc.AddPurchaseOrderItemUseCase{
			Repo:          poRepo,
			Auth:          authService,
			PriceProvider: purchasePriceUC,
			UOMConverter:  itemConversionUC,
			FiscalClass:   fiscalClassUC,
		},
	)

	// MRP purchase suggestions (PURCHASE planned orders → purchase order)
	purchaseSuggestionHandler := handler.NewPurchaseSuggestionHandler(
		&purchase_order_uc.ListPurchaseSuggestionsUseCase{Planned: plannedRepo, Auth: authService},
		&purchase_order_uc.ApprovePurchaseSuggestionUseCase{Planned: plannedRepo, Repo: poRepo, Auth: authService, SupplierDefaults: supplierUC},
		&purchase_order_uc.RejectPurchaseSuggestionUseCase{Planned: plannedRepo, Auth: authService},
	)

	// purchase requisitions + generation of purchase orders from requisitions
	purchaseReqRepository := purchaseReqRepo.New(queries, app.db.Pool)
	purchaseRequisitionHandler := handler.NewPurchaseRequisitionHandler(
		purchase_requisition_uc.NewPurchaseRequisitionUseCase(purchaseReqRepository),
		&purchase_requisition_uc.GeneratePurchaseOrdersUseCase{
			Reqs:             purchaseReqRepository,
			POs:              poRepo,
			Auth:             authService,
			Preferred:        itemSupplierUC,
			SupplierDefaults: supplierUC,
			PriceProvider:    purchasePriceUC,
		},
	)

	// purchase quotations (liberação p/ cotação → preços → seleção → pedidos)
	purchaseQuotationRepository := purchaseQuotationRepo.New(queries, app.db.Pool)
	purchaseQuotationHandler := handler.NewPurchaseQuotationHandler(
		purchase_quotation_uc.NewPurchaseQuotationUseCase(purchaseQuotationRepository, purchaseReqRepository, plannedRepo),
		&purchase_quotation_uc.GenerateOrdersFromQuotationUseCase{
			Quotations:       purchaseQuotationRepository,
			Reqs:             purchaseReqRepository,
			POs:              poRepo,
			Auth:             authService,
			SupplierDefaults: supplierUC,
		},
	)

	// sales order
	soRepo := salesOrderRepo.NewSalesOrderRepositorySQLC(queries)
	// Captured so the automatic credit check and stock reservation (ATP) can be
	// attached once the customer/financial/stock repositories are available below.
	changeStatusSalesOrderUC := &sales_order_uc.ChangeStatusSalesOrderUseCase{Repo: soRepo, Auth: authService, DemandRepo: independentDemandRepo}
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
		changeStatusSalesOrderUC,
		&sales_order_uc.CreateSalesOrderItemUseCase{Repo: soRepo, Auth: authService},
		&sales_order_uc.UpdateSalesOrderItemUseCase{Repo: soRepo, Auth: authService},
		&sales_order_uc.ListSalesOrderItemsUseCase{Repo: soRepo, Auth: authService},
		&sales_order_uc.CancelSalesOrderItemUseCase{Repo: soRepo, Auth: authService},
	)

	// stock management
	stockRepository := stockRepo.NewStockRepositorySQLC(app.db.Pool)

	// cutting plan (plano de corte — fase 1: 1D; fase 2: firmar/baixa + retalhos)
	cuttingPlanUC := cutting_plan_uc.NewCuttingPlanUseCase(cuttingPlanRepo, stockRepository, itemRepo)
	// Optional external true-shape nesting engine (DeepNest/ProNest-style service).
	// When NESTING_SERVICE_URL is set it overrides the native bounding-box provider.
	if nestingURL := strings.TrimSpace(os.Getenv("NESTING_SERVICE_URL")); nestingURL != "" {
		cuttingPlanUC = cuttingPlanUC.WithTrueShapeProvider(nesting.NewHTTPProvider(nestingURL))
	}
	// Machine scheduling for cut plans (fase de complementos).
	cuttingPlanUC = cuttingPlanUC.WithMachineRepo(machineRepo)
	// Auto-generate cutting demand from production/planned orders (fase 5).
	cuttingDemandUC := cutting_plan_uc.NewDemandUseCase(cuttingPlanRepo, itemRepo, itemRepoStructureQuery, prodOrderRepo, plannedRepo)
	cuttingPlanHandler := handler.NewCuttingPlanHandler(cuttingPlanUC, cuttingDemandUC)

	// Production consumption/completion post stock movements automatically.
	prodOrderAddConsumptionUC.StockRepo = stockRepository
	prodOrderCompleteUC.StockRepo = stockRepository
	// Scrap return posts a valued IN movement of the scrap by-product.
	prodOrderReturnScrapUC.StockRepo = stockRepository
	// Appointment backflush: auto-consume BOM components from stock.
	prodOrderAddAppointmentUC.StructureRepo = itemRepoStructure
	prodOrderAddAppointmentUC.StockRepo = stockRepository
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
	// lot traceability (rastreabilidade de lote/corrida + genealogia)
	registerLotUC := &stock_uc.RegisterLotUseCase{Repo: stockRepository, Auth: authService}
	listLotBalancesUC := &stock_uc.ListLotBalancesUseCase{Repo: stockRepository, Auth: authService}
	getLotGenealogyUC := &stock_uc.GetLotGenealogyUseCase{Repo: stockRepository, Auth: authService}
	// consumption average (consumo médio mensal → ROP)
	recalcCMUC := &stock_uc.RecalcConsumptionAverageUseCase{Repo: stockRepository, Auth: authService}
	getCMUC := &stock_uc.GetConsumptionAverageUseCase{Repo: stockRepository, Auth: authService}
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
	).WithLot(registerLotUC, listLotBalancesUC, getLotGenealogyUC).
		WithConsumptionAverage(recalcCMUC, getCMUC)

	// financial
	fRepo := financialRepo.NewFinancialRepositoryPG(app.db.Pool)
	fiscalRepository := fiscalRepo.NewFiscalRepositoryPG(app.db.Pool)

	// Brand the PDF cutting map with the company letterhead from fiscal config.
	cuttingPlanUC.WithBranding(fiscalRepository)

	// supplier SEFAZ cadastral query (FocusNFE)
	supplierSefazHandler := handler.NewSupplierSefazHandler(&supplier_uc.ConsultSupplierSefazUseCase{
		Repo:       suppRepo,
		FiscalRepo: fiscalRepository,
		Auth:       authService,
	})

	cnabHandler := handler.NewCNABHandler()
	ibptHandler := handler.NewIBPTHandler(&ibpt_uc.IBPTUseCase{Repo: ibptRepo.NewIBPTRepositoryPG(app.db.Pool)})
	shipmentRepoPG := shipmentRepo.NewShipmentRepositoryPG(app.db.Pool)
	shipmentBaseUC := &shipment_uc.ShipmentUseCase{Repo: shipmentRepoPG, Stock: stockRepository}
	shipmentAutoFillUC := &shipment_uc.ShipmentAutoFillUseCase{
		ShipmentRepo:   shipmentRepoPG,
		SalesRepo:      &shipmentRepo.SalesOrderAdapter{Repo: soRepo},
		PurchaseRepo:   &shipmentRepo.PurchaseOrderAdapter{Repo: poRepo},
		ProductionRepo: &shipmentRepo.ProductionOrderAdapter{Repo: prodOrderRepo},
	}
	shipmentExportUC := &shipment_uc.ShipmentExportUseCase{
		ShipmentRepo: shipmentRepoPG,
		Enricher:     nil,
	}
	shipmentHandler := handler.NewShipmentHandler(shipmentBaseUC).
		WithAutoFill(shipmentAutoFillUC).
		WithExport(shipmentExportUC)
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

	// adiantamentos (advance payments)
	adiantamentoHandler := handler.NewAdiantamentoHandler(
		&financial_uc.CreateAdiantamentoUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.ListAdiantamentosUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.GetAdiantamentoUseCase{Repo: fRepo, Auth: authService},
		&financial_uc.AplicarAdiantamentoUseCase{Repo: fRepo, Auth: authService},
	)

	// fiscal module
	fiscalHandler := handler.NewFiscalHandler(
		&fiscalUC.CreateFiscalEntryUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.UploadNFEEntryUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.ApproveFiscalEntryUseCase{FiscalRepo: fiscalRepository, FinancialRepo: fRepo, Auth: authService},
		&fiscalUC.ListFiscalEntriesUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.GetFiscalEntryUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.CreateFiscalExitUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.AuthorizeFiscalExitUseCase{Repo: fiscalRepository, FinancialRepo: fRepo, Auth: authService, StockRepo: stockRepository, SalesOrderRepo: soRepo},
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
		&fiscalUC.GetDANFEUseCase{Repo: fiscalRepository, Auth: authService},
	)

	// company branding (report letterhead logo + brand colour)
	fiscalBrandingHandler := handler.NewFiscalBrandingHandler(
		&fiscalUC.UpdateBrandingUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.GetBrandingUseCase{Repo: fiscalRepository, Auth: authService},
	)

	// maintenance
	maintRepo := maintenanceRepo.New(queries)
	maintUC := maintenance_uc.New(maintRepo)
	maintHandler := handler.NewMaintenanceHandler(maintUC)
	crpUC.WithMaintenance(maintRepo)

	// mrp exception notifications
	emailSvc := notification.NewEmailService(notification.SMTPConfig{
		Host:     app.config.SMTPHost,
		Port:     app.config.SMTPPort,
		User:     app.config.SMTPUser,
		Password: app.config.SMTPPassword,
		From:     app.config.SMTPFrom,
	})
	notifyExcUC := mrp_uc.NewNotifyExceptionsUseCase(mrpRepo, emailSvc)
	mrpExcHandler := handler.NewMRPExceptionsHandler(notifyExcUC)

	// NF-e purchase import
	importNFeUC := &fiscalUC.ImportNFePurchaseUseCase{
		FiscalRepo:        fiscalRepository,
		StockRepo:         stockRepository,
		Auth:              authService,
		SupplierDefaults:  supplierUC,
		PurchaseOrderRepo: poRepo,
	}
	importNFeHandler := handler.NewImportNFePurchaseHandler(importNFeUC)

	// fiscal: CT-e SEFAZ authorization
	cteAuthorizeHandler := handler.NewCTeAuthorizeHandler(
		&fiscalUC.AuthorizeCTeUseCase{Repo: fiscalRepository, Auth: authService},
	)

	// fiscal: NFS-e (service invoices)
	nfseRepository := nfseRepo.NewNFSeRepositoryPG(app.db.Pool)
	nfseHandler := handler.NewNFSeHandler(
		&nfse_uc.CreateNFSeUseCase{Repo: nfseRepository, Auth: authService},
		&nfse_uc.AuthorizeNFSeUseCase{Repo: nfseRepository, Config: fiscalRepository, Auth: authService},
		&nfse_uc.CancelNFSeUseCase{Repo: nfseRepository, Config: fiscalRepository, Auth: authService},
		&nfse_uc.ListNFSeUseCase{Repo: nfseRepository, Auth: authService},
		&nfse_uc.GetNFSeUseCase{Repo: nfseRepository, Auth: authService},
	)

	// fiscal: manifestação do destinatário + inutilização de numeração
	fiscalManifestHandler := handler.NewFiscalManifestHandler(
		&fiscalUC.ManifestarDestinatarioUseCase{Repo: fiscalRepository, Auth: authService},
		&fiscalUC.InutilizarNumeracaoUseCase{Repo: fiscalRepository, Auth: authService},
	)

	// customer
	custRepo := customerRepo.New(queries, app.db.Pool)
	customerUC := customer_uc.NewCustomerUseCase(custRepo)
	customerHandler := handler.NewCustomerHandler(customerUC)

	// Romaneio export enrichment: real company (with branding), parties, carrier
	// and item data, so exported romaneios carry the professional letterhead.
	shipmentExportUC.Enricher = &shipmentRepo.RomaneioEnricherAdapter{
		Fiscal:     fiscalRepository,
		Customers:  custRepo,
		Suppliers:  suppRepo,
		Items:      itemRepo,
		Sales:      &shipmentRepo.SalesOrderAdapter{Repo: soRepo},
		Purchases:  &shipmentRepo.PurchaseOrderAdapter{Repo: poRepo},
		Production: &shipmentRepo.ProductionOrderAdapter{Repo: prodOrderRepo},
	}

	// Confirming a sales order (status "P") now runs an automatic credit-limit
	// check (exposure = open receivables + other open orders) and reserves
	// available stock per line (ATP). A credit-blocked order neither feeds the
	// MRP nor reserves stock.
	changeStatusSalesOrderUC.CreditChecker = &sales_order_uc.CreditChecker{
		SalesRepo:     soRepo,
		CustomerRepo:  custRepo,
		FinancialRepo: fRepo,
	}
	changeStatusSalesOrderUC.Reserver = &sales_order_uc.OrderStockReserver{
		SalesRepo: soRepo,
		StockRepo: stockRepository,
	}

	// fiscal params (legal devices, CFOPs, ICMS/IPI tax params + ICMS apuração + Simples Nacional)
	fpRepo := fiscalParamsRepo.NewFiscalParamsRepository(queries, app.db.Pool)
	legalDeviceUC := &fiscal_params_uc.LegalDeviceUseCase{Repo: fpRepo}
	cfopUC := &fiscal_params_uc.CFOPUseCase{Repo: fpRepo}
	taxParamUC := &fiscal_params_uc.TaxParamUseCase{Repo: fpRepo}
	fiscalParamsHandler := handler.NewFiscalParamsHandler(legalDeviceUC, cfopUC, taxParamUC)

	dapiUC := &fiscal_params_uc.DAPITransferReasonUseCase{Repo: fpRepo}
	apuracaoAdjUC := &fiscal_params_uc.ICMSApuracaoAdjCodeUseCase{Repo: fpRepo}
	adjCodeUC := &fiscal_params_uc.ICMSAdjustmentCodeUseCase{Repo: fpRepo}
	apuracaoLineUC := &fiscal_params_uc.ICMSApuracaoLineUseCase{Repo: fpRepo}
	summaryUC := &fiscal_params_uc.ICMSSummaryEntryUseCase{Repo: fpRepo}
	simplesUC := &fiscal_params_uc.SimplesNacionalUseCase{Repo: fpRepo}
	icmsApuracaoHandler := handler.NewICMSApuracaoHandler(dapiUC, apuracaoAdjUC, adjCodeUC, apuracaoLineUC, summaryUC, simplesUC)

	reductionUC := &fiscal_params_uc.ICMSReductionSubstitutionUseCase{Repo: fpRepo}
	summaryAdditionalUC := &fiscal_params_uc.ICMSSummaryAdditionalUseCase{Repo: fpRepo}
	stRestUC := &fiscal_params_uc.ICMSSTRestitutionUseCase{Repo: fpRepo}
	specialNoteUC := &fiscal_params_uc.SpecialAdjustmentNoteUseCase{Repo: fpRepo}
	icmsReductionHandler := handler.NewICMSReductionHandler(reductionUC, summaryAdditionalUC, stRestUC, specialNoteUC)

	spedUC := &fiscalUC.SPEDUseCase{FiscalParamsRepo: fpRepo}
	spedHandler := handler.NewSPEDHandler(spedUC)

	// accounting / SPED ECD
	acctRepo := accountingRepo.New(queries, app.db.Pool)
	acctPlanUC := &accounting_uc.AccountingPlanUseCase{Repo: acctRepo}
	acctAccountUC := &accounting_uc.AccountingAccountUseCase{Repo: acctRepo}
	acctEntryUC := &accounting_uc.JournalEntryUseCase{Repo: acctRepo}
	acctDemUC := &accounting_uc.DemonstrativeUseCase{Repo: acctRepo}
	acctECDUC := &accounting_uc.ECDUseCase{Repo: acctRepo}
	acctBalanceteUC := &accounting_uc.BalanceteUseCase{Repo: acctRepo}
	accountingHandler := handler.NewAccountingHandler(acctPlanUC, acctAccountUC, acctEntryUC, acctDemUC, acctECDUC, acctBalanceteUC)

	// stock movement types
	smtRepo := stockMovementRepo.New(app.db.Pool)
	smtUC := stock_movement_uc.New(smtRepo)
	smtHandler := handler.NewStockMovementTypeHandler(smtUC)

	// location (countries + UFs)
	locRepo := locationRepo.New(queries)
	locationUC := location_uc.New(locRepo)
	locationHandler := handler.NewLocationHandler(locationUC)

	// item classifications
	itemClassRepo := itemClassificationRepo.New(queries)
	itemClassUC := item_classification_uc.New(itemClassRepo)
	itemClassHandler := handler.NewItemClassificationHandler(itemClassUC)

	// Audit trail reader (ADMIN-only query side; writes happen in middleware).
	auditHandler := handler.NewAuditHandler(audit.NewReader(app.db.Pool))

	// routes
	idempotencyStore := httpmw.NewIdempotencyStore(24 * time.Hour)
	r.Group(func(r chi.Router) {
		r.Use(httpmw.JWT(app.config.JWTSecret, app.logger))
		// Audit trail for authenticated mutations (after JWT so the actor is known).
		r.Use(httpmw.Audit(app.auditSink))
		// Idempotency-Key support for mutating requests (safe retries).
		r.Use(httpmw.Idempotency(idempotencyStore))

		// Audit trail (read): who changed what, when. Restricted to ADMIN.
		r.With(httpmw.RequireRole("ADMIN")).Get("/api/audit-log", auditHandler.List)
		r.Route("/api/items", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", itemHandler.CreateItem)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", itemHandler.ListItems)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/with-masks", itemHandler.ListItemsWithMasks)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/search/{code}", itemHandler.FindItemByCodeHandler)
			r.With(httpmw.RequirePermission(httpmw.PermItemActivate)).Get("/{code}/activation-readiness", itemActivationHandler.ValidateActivation)

			r.Route("/mask", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/generate", generateMaskItemHandler.GenerateMask)
			})
			r.Route("/structure", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", structureHandler.Create)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/update", structureHandler.Update)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{parentItemCode}/children", structureHandler.GetAllDirectChildren)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/resolve/{itemCode}", queryStructureHandler.ResolveStructure)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/consult", queryStructureHandler.ConsultStructure)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/where-used/{itemCode}", queryStructureHandler.WhereUsed)
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
			// MRP suggestion bridge: list suggestions for a plan, firm one into planned_orders
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/suggestions/{plan_code}", mrpHandler.ListSuggestions)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/suggestions/{code}/firm", mrpHandler.FirmarSugestao)
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
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{id}/settle-cost", prodOrderHandler.SettleCost)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}/cost", prodOrderHandler.GetCost)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{id}/scrap-return", prodOrderHandler.ReturnScrap)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/operations/explode", prodOrderHandler.ExplodeRoute)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}/operations", prodOrderHandler.ListOrderOperations)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/operations/advance", prodOrderHandler.AdvanceOperation)
		})
		r.Route("/api/questions", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", questionCreateHandler.CreateQuestion)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", findQuestionByNameHandler.FindQuestionByName)
			r.Route("/options", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", questionOptionCreateHandler.CreateQuestionOptionHandler)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{questionID}", questionOptionCreateHandler.ListByQuestion)
			})
			r.Route("/associate", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", associateByQuestionItemHandler.AssociateQuestions)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", associateByQuestionItemHandler.ListAll)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/item/{itemCode}", associateByQuestionItemHandler.GetQuestionsByItem)
			})
		})
		r.Route("/api/bom", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", bomHandler.Create)
			r.Route("/bom-items", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", bomItemHandler.Create)
			})
		})
		r.Route("/api/warehouse", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", warehouseHandler.CreateWarehouse)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", warehouseHandler.ListWarehouses)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", warehouseHandler.GetWarehouse)
		})
		r.Route("/api/pdm", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create-group", groupHandler.CreateGroup)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create-modifier", modifierHandler.CreateModifier)
		})
		r.Route("/api/enterprise", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", enterpriseHandler.CreateEnterprise)
		})
		// CNPJ auto-fill: GET /api/cnpj/{cnpj} returns razão social, IE, endereço…
		r.Route("/api/cnpj", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{cnpj}", cnpjHandler.Lookup)
		})
		// Generic report export: POST /api/reports/export?format=xlsx|pdf|csv
		r.Route("/api/reports", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/export", reportExportHandler.Export)
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
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/customer/{customerCode}", restrictionHandler.GetByCustomer)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/evaluate", restrictionHandler.Evaluate)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/{code}", restrictionHandler.Update)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Patch("/{code}/deactivate", restrictionHandler.Deactivate)
		})
		r.Route("/api/restriction-reason", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/create", restrictionReasonHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/list", restrictionReasonHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", restrictionReasonHandler.GetByCode)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/{code}", restrictionReasonHandler.Update)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/{code}", restrictionReasonHandler.Delete)
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
			// MRP purchase suggestions
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/suggestions", purchaseSuggestionHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/suggestions/{code}/approve", purchaseSuggestionHandler.Approve)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/suggestions/{code}/reject", purchaseSuggestionHandler.Reject)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{code}/items", purchaseOrderItemHandler.AddItem)
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
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/atp/{itemCode}", stockHandler.GetATP)
			})
			r.Route("/lots", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/register", stockHandler.RegisterLot)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/item/{itemCode}", stockHandler.ListLotBalances)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/genealogy/{itemCode}/{lot}", stockHandler.GetLotGenealogy)
			})
			r.Route("/consumption-average", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/recalc", stockHandler.RecalcConsumptionAverage)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{itemCode}", stockHandler.GetConsumptionAverage)
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
			r.With(httpmw.RequirePermission(httpmw.PermFiscalAuthorize)).Post("/manifestacao", fiscalManifestHandler.Manifestar)
			r.With(httpmw.RequirePermission(httpmw.PermFiscalAuthorize)).Post("/inutilizacao", fiscalManifestHandler.Inutilizar)
			r.With(httpmw.RequirePermission(httpmw.PermAdmin)).Post("/ibpt/import", ibptHandler.Import)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/ibpt/lookup", ibptHandler.Lookup)
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
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/config/branding", fiscalBrandingHandler.Update)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/config/logo", fiscalBrandingHandler.Logo)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/cte/create", fiscalHandler.CreateCTe)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/cte/list", fiscalHandler.ListCTe)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/cte/{code}", fiscalHandler.GetCTe)
			r.With(httpmw.RequirePermission(httpmw.PermFiscalAuthorize)).Post("/cte/{code}/authorize", cteAuthorizeHandler.Authorize)
			// NFS-e (service invoices)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/nfse/create", nfseHandler.Create)
			r.With(httpmw.RequirePermission(httpmw.PermFiscalAuthorize)).Post("/nfse/{code}/authorize", nfseHandler.Authorize)
			r.With(httpmw.RequirePermission(httpmw.PermFiscalAuthorize)).Post("/nfse/{code}/cancel", nfseHandler.Cancel)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/nfse/list", nfseHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/nfse/{code}", nfseHandler.Get)
			// NF-e status consultation, CC-e list, DANFE PDF URL
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/exits/{id}/status", fiscalHandler.ConsultarNFe)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/exits/{id}/cartas-correcao", fiscalHandler.ListCartasCorrecao)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/exits/{id}/danfe", fiscalHandler.GetDANFE)
			// NCM tax table management
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/tabelas/ncm", fiscalHandler.UpsertNcmTax)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/tabelas/ncm", fiscalHandler.ListNcmTaxes)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/tabelas/ncm/{ncm}", fiscalHandler.DeleteNcmTax)
			// ICMS table management
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/tabelas/icms-interestadual", fiscalHandler.UpsertICMSInterstate)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/tabelas/icms-interestadual", fiscalHandler.ListICMSInterstate)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/tabelas/icms-interno", fiscalHandler.UpsertICMSInternal)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/tabelas/icms-interno", fiscalHandler.ListICMSInternal)
			// Fiscal params support tables
			r.Route("/support", func(r chi.Router) {
				r.Route("/dispositivos-legais", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", fiscalParamsHandler.CreateLegalDevice)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", fiscalParamsHandler.UpdateLegalDevice)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", fiscalParamsHandler.ListLegalDevices)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", fiscalParamsHandler.GetLegalDevice)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/tipo/{type}", fiscalParamsHandler.ListLegalDevicesByType)
				})
				r.Route("/cfops", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", fiscalParamsHandler.CreateCFOP)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", fiscalParamsHandler.UpdateCFOP)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", fiscalParamsHandler.ListCFOPs)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", fiscalParamsHandler.GetCFOP)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/direcao/{direction}", fiscalParamsHandler.ListCFOPsByDirection)
				})
				r.Route("/parametros-icms-ipi", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", fiscalParamsHandler.CreateTaxParam)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", fiscalParamsHandler.UpdateTaxParam)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", fiscalParamsHandler.ListTaxParams)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}", fiscalParamsHandler.GetTaxParam)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/uf/{uf}", fiscalParamsHandler.ListTaxParamsByUF)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/item/{itemCode}", fiscalParamsHandler.ListTaxParamsByItem)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/ncm/{ncmCode}", fiscalParamsHandler.ListTaxParamsByNCM)
				})
				// DAPI Transfer Reasons
				r.Route("/motivos-transferencia-dapi", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", icmsApuracaoHandler.CreateDAPIReason)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", icmsApuracaoHandler.UpdateDAPIReason)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", icmsApuracaoHandler.ListDAPIReasons)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", icmsApuracaoHandler.GetDAPIReason)
				})
				// ICMS Apuração Adjustment Codes (tabela 5.1.1)
				r.Route("/codigos-ajuste-apuracao-icms", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", icmsApuracaoHandler.CreateApuracaoAdjCode)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", icmsApuracaoHandler.UpdateApuracaoAdjCode)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", icmsApuracaoHandler.ListApuracaoAdjCodes)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}", icmsApuracaoHandler.GetApuracaoAdjCode)
				})
				// ICMS Adjustment Codes (tabelas 5.2/5.3/5.6/5.7)
				r.Route("/codigos-ajuste-icms", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", icmsApuracaoHandler.CreateAdjCode)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", icmsApuracaoHandler.UpdateAdjCode)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", icmsApuracaoHandler.ListAdjCodes)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}", icmsApuracaoHandler.GetAdjCode)
				})
				// ICMS Apuração Lines
				r.Route("/linhas-apuracao-icms", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", icmsApuracaoHandler.CreateApuracaoLine)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", icmsApuracaoHandler.UpdateApuracaoLine)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", icmsApuracaoHandler.ListApuracaoLines)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", icmsApuracaoHandler.GetApuracaoLine)
				})
				// ICMS Summary Entries
				r.Route("/lancamentos-resumo-icms", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", icmsApuracaoHandler.CreateSummaryEntry)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", icmsApuracaoHandler.UpdateSummaryEntry)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", icmsApuracaoHandler.ListSummaryEntries)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}", icmsApuracaoHandler.GetSummaryEntry)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{id}/notas", icmsApuracaoHandler.AddSummaryEntryNote)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}/notas", icmsApuracaoHandler.ListSummaryEntryNotes)
				})
				// Simples Nacional Apuração
				r.Route("/apuracao-simples-nacional", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", icmsApuracaoHandler.CreateSimplesApuracao)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", icmsApuracaoHandler.UpdateSimplesApuracao)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", icmsApuracaoHandler.ListSimplesApuracoes)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{period}/{annex}", icmsApuracaoHandler.GetSimplesApuracao)
				})
			})
		})
		// ICMS Reduction / Substitution / Deferral
		r.Route("/api/fiscal/icms-reducao", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", icmsReductionHandler.CreateReduction)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", icmsReductionHandler.UpdateReduction)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", icmsReductionHandler.ListReductions)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/find", icmsReductionHandler.FindReduction)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}", icmsReductionHandler.GetReduction)
		})
		// ICMS Summary Entry Additionals (aba Adicionais)
		r.Route("/api/fiscal/icms-resumo-adicionais", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", icmsReductionHandler.AddSummaryAdditional)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}", icmsReductionHandler.ListSummaryAdditionals)
		})
		// ICMS ST Restitution / Ressarcimento / Complementação
		r.Route("/api/fiscal/icms-st-restituicao", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", icmsReductionHandler.CreateSTRestitution)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", icmsReductionHandler.UpdateSTRestitution)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", icmsReductionHandler.ListSTRestitutions)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}", icmsReductionHandler.GetSTRestitution)
		})
		// Special Adjustment Notes
		r.Route("/api/fiscal/notas-especiais", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", icmsReductionHandler.CreateSpecialNote)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", icmsReductionHandler.UpdateSpecialNote)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", icmsReductionHandler.ListSpecialNotes)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}", icmsReductionHandler.GetSpecialNote)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{id}/itens", icmsReductionHandler.AddSpecialNoteItem)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}/itens", icmsReductionHandler.ListSpecialNoteItems)
		})
		// SPED EFD
		r.Route("/api/fiscal/sped", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/efd", spedHandler.GenerateEFD)
		})
		// Stock Movement Types
		r.Route("/api/estoque/tipos-movimento", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", smtHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", smtHandler.Update)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", smtHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}", smtHandler.GetByID)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/sigla/{sigla}", smtHandler.GetBySigla)
		})
		r.Route("/api/financial", func(r chi.Router) {
			r.With(httpmw.RequirePermission(httpmw.PermFinancialManage)).Post("/cnab/remessa-240", cnabHandler.GenerateRemessa240)
			// Adiantamentos (advance payments)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/adiantamentos/create", adiantamentoHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/adiantamentos/list", adiantamentoHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/adiantamentos/{id}", adiantamentoHandler.Get)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/adiantamentos/{id}/aplicar", adiantamentoHandler.Aplicar)
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
		r.Route("/api/routing", func(r chi.Router) {
			r.Route("/operations", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", routingHandler.CreateOperation)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", routingHandler.ListOperations)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}", routingHandler.GetOperation)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/{id}", routingHandler.UpdateOperation)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/{id}", routingHandler.DeactivateOperation)
			})
			r.Route("/routes", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", routingHandler.CreateRoute)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", routingHandler.ListRoutesByItem)
				r.Route("/{id}", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", routingHandler.GetRouteDetail)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", routingHandler.UpdateRoute)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/", routingHandler.DeactivateRoute)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/lead-time", routingHandler.GetLeadTime)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/edges", routingHandler.GetNetworkEdges)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/edges", routingHandler.SetNetworkEdge)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/edges", routingHandler.DeleteNetworkEdge)
				})
			})
			r.Route("/route-operations", func(r chi.Router) {
				r.Route("/{routeId}", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", routingHandler.AddRouteOperation)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/{opId}", routingHandler.UpdateRouteOperation)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/{opId}", routingHandler.RemoveRouteOperation)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/network/set", routingHandler.SetNetworkEdge)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/network", routingHandler.DeleteNetworkEdge)
				})
			})
		})
		r.Route("/api/cutting-plans", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", cuttingPlanHandler.CreatePlan)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/from-orders", cuttingPlanHandler.GenerateFromOrders)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", cuttingPlanHandler.ListPlans)
			r.Route("/{id}", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", cuttingPlanHandler.GetPlan)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/", cuttingPlanHandler.DeletePlan)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/optimize", cuttingPlanHandler.Optimize)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/release", cuttingPlanHandler.Release)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/export", cuttingPlanHandler.ExportMap)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/program", cuttingPlanHandler.GetProgram)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/schedule", cuttingPlanHandler.Schedule)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/order-costs", cuttingPlanHandler.ListOrderCosts)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/parts", cuttingPlanHandler.AddPart)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/parts/{partId}", cuttingPlanHandler.RemovePart)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/stock", cuttingPlanHandler.AddStock)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/stock/{stockId}", cuttingPlanHandler.RemoveStock)
			})
		})
		r.Route("/api/cutting-settings", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", cuttingPlanHandler.GetSettings)
			r.With(httpmw.RequireRole("ADMIN")).Put("/", cuttingPlanHandler.UpdateSettings)
		})
		r.Route("/api/stock-remnants", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", cuttingPlanHandler.ListRemnants)
		})
		r.Route("/api/aps", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/sequence", apsHandler.SequenceOrders)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/gantt/order/{orderID}", apsHandler.GetGanttByOrder)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/gantt/work-center", apsHandler.GetGanttByWorkCenter)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/gantt/month/{year}/{month}", apsHandler.GetMonthGantt)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/gantt/month/{year}/{month}/export", apsHandler.ExportMonthGantt)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/gantt/board", apsHandler.GetGanttBoard)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/gantt/board/export", apsHandler.ExportGanttBoard)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/gantt/reschedule", apsHandler.RescheduleSequence)
		})
		r.Route("/api/planning", func(r chi.Router) {
			r.With(httpmw.RequirePermission(httpmw.PermPlanningRun)).Post("/run-pipeline", planningHandler.RunPipeline)
		})
		r.Route("/api/shipments", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", shipmentHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", shipmentHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", shipmentHandler.Get)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{code}/items", shipmentHandler.AddItem)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{code}/items/confer", shipmentHandler.ConferItem)

			// Ciclo de vida (máquina de estado): separar → conferir → despachar.
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{code}/separate", shipmentHandler.Separate)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{code}/confer", shipmentHandler.Confer)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{code}/ship", shipmentHandler.Ship)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{code}/cancel", shipmentHandler.Cancel)

			// Transporte / viagem, volumes (packing), vínculo NF-e e auditoria.
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/{code}/transport", shipmentHandler.UpdateTransport)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{code}/volumes", shipmentHandler.AddVolume)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}/volumes", shipmentHandler.ListVolumes)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/{code}/volumes/{volumeID}", shipmentHandler.DeleteVolume)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{code}/nfe-link", shipmentHandler.LinkFiscalExit)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}/events", shipmentHandler.ListEvents)

			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/auto-fill/sales-order", shipmentHandler.AutoFillFromSalesOrder)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/auto-fill/purchase-order", shipmentHandler.AutoFillFromPurchaseOrder)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/auto-fill/production-order", shipmentHandler.AutoFillFromProductionOrder)

			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}/export/pdf", shipmentHandler.ExportPDF)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}/export/xlsx", shipmentHandler.ExportXLSX)
		})
		r.Route("/api/crp", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/calculate", crpHandler.CalculateCRP)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{planCode}", crpHandler.ListByPlan)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{planCode}/overloaded", crpHandler.ListOverloadedByPlan)
		})
		r.Route("/api/standard-cost", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/rollup", standardCostHandler.RollUp)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/items/{itemCode}", standardCostHandler.GetStandardCost)
			r.Route("/work-center-costs", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", standardCostHandler.UpsertWorkCenterCost)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", standardCostHandler.ListWorkCenterCosts)
			})
			r.Route("/purchase-costs", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", standardCostHandler.UpsertItemPurchaseCost)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{itemCode}", standardCostHandler.GetItemPurchaseCost)
			})
		})
		r.Route("/api/quality", func(r chi.Router) {
			r.Route("/plans", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", qualityHandler.CreatePlan)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}", qualityHandler.GetPlan)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/{id}", qualityHandler.DeactivatePlan)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/by-item/{itemCode}", qualityHandler.ListPlansByItem)
			})
			r.Route("/characteristics", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", qualityHandler.AddCharacteristic)
			})
			r.Route("/records", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", qualityHandler.CreateRecord)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}", qualityHandler.GetRecord)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/by-order/{orderID}", qualityHandler.ListRecordsByOrder)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/by-item/{itemCode}", qualityHandler.ListRecordsByItem)
			})
			r.Route("/non-conformances", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", qualityHandler.CreateNC)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/open", qualityHandler.ListOpenNCs)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}", qualityHandler.GetNC)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/by-item/{itemCode}", qualityHandler.ListNCsByItem)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{id}/disposition", qualityHandler.DispositionNC)
			})
		})
		r.Route("/api/maintenance", func(r chi.Router) {
			r.Route("/plans", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", maintHandler.CreatePlan)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", maintHandler.ListPlans)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{id}", maintHandler.GetPlan)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/{id}", maintHandler.DeactivatePlan)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/by-machine/{machineId}", maintHandler.ListPlansByMachine)
			})
			r.Route("/orders", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", maintHandler.CreateOrder)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/advance", maintHandler.AdvanceOrder)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/generate", maintHandler.GenerateOrders)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/by-plan/{planId}", maintHandler.ListOrdersByPlan)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/by-work-center/{wcId}", maintHandler.ListOrdersByWorkCenter)
			})
		})
		r.Route("/api/forecast", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/statistical", handler.StatisticalForecastHandler)
		})
		r.Route("/api/mrp-calculation/exceptions", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{plan_code}", mrpHandler.ListExceptions)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/notify", mrpExcHandler.Notify)
		})
		r.Route("/api/fiscal/entries/import-nfe", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", importNFeHandler.Import)
		})
		r.Route("/api/location", func(r chi.Router) {
			r.Route("/countries", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", locationHandler.CreateCountry)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", locationHandler.UpdateCountry)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", locationHandler.ListCountries)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{sigla}", locationHandler.GetCountry)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{sigla}/ufs", locationHandler.ListUFsByCountry)
			})
			r.Route("/ufs", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", locationHandler.CreateUF)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", locationHandler.UpdateUF)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", locationHandler.ListUFs)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{sigla}", locationHandler.GetUF)
			})
		})
		r.Route("/api/items/classifications", func(r chi.Router) {
			r.Route("/masks", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", itemClassHandler.CreateMask)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", itemClassHandler.UpdateMask)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", itemClassHandler.ListMasks)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", itemClassHandler.GetMask)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{maskID}/items", itemClassHandler.ListByMask)
			})
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", itemClassHandler.CreateClassification)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", itemClassHandler.UpdateClassification)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{maskCode}/{code}", itemClassHandler.GetClassification)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{parentID}/children", itemClassHandler.ListChildren)
		})
		r.Route("/api/accounting", func(r chi.Router) {
			r.Route("/plans", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", accountingHandler.CreatePlan)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", accountingHandler.ListPlans)
			})
			r.Route("/accounts", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", accountingHandler.CreateAccount)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", accountingHandler.ListAccounts)
			})
			r.Route("/journal-entries", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", accountingHandler.CreateJournalEntry)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", accountingHandler.ListJournalEntries)
			})
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/balancete", accountingHandler.Balancete)
			r.Route("/demonstratives", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", accountingHandler.CreateDemonstrative)
			})
			r.Route("/sped/ecd", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", accountingHandler.GenerateECD)
			})
		})
		r.Route("/api/customers", func(r chi.Router) {
			// ── Support tables (cadastros de apoio) ──────────────────────────
			r.Route("/support", func(r chi.Router) {
				r.Route("/regions", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", customerHandler.CreateRegion)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", customerHandler.UpdateRegion)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", customerHandler.ListRegions)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", customerHandler.GetRegion)
				})
				r.Route("/market-segments", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", customerHandler.CreateMarketSegment)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", customerHandler.ListMarketSegments)
				})
				r.Route("/contact-types", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", customerHandler.CreateContactType)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", customerHandler.ListContactTypes)
				})
				r.Route("/customer-types", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", customerHandler.CreateCustomerType)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", customerHandler.ListCustomerTypes)
				})
				r.Route("/carriers", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", customerHandler.CreateCarrier)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", customerHandler.ListCarriers)
				})
				r.Route("/carrier-groups", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", customerHandler.CreateCarrierGroup)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", customerHandler.ListCarrierGroups)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/members", customerHandler.AddCarrierToGroup)
				})
				r.Route("/payment-conditions", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", customerHandler.CreatePaymentCondition)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", customerHandler.ListPaymentConditions)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/installments", customerHandler.AddInstallment)
				})
				r.Route("/sales-tables", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", customerHandler.CreateSalesTable)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", customerHandler.ListSalesTables)
					r.Route("/{tableID}/prices", func(r chi.Router) {
						r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", customerHandler.CreateSalesTablePrice)
						r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", customerHandler.ListSalesTablePrices)
						r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{itemCode}", customerHandler.GetSalesTablePrice)
					})
					r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/prices/", customerHandler.UpdateSalesTablePrice)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/prices/{id}", customerHandler.DeleteSalesTablePrice)
				})
				r.Route("/invoice-types", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", customerHandler.CreateInvoiceType)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", customerHandler.UpdateInvoiceType)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", customerHandler.ListInvoiceTypes)
				})
				r.Route("/tax-types", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", customerHandler.CreateTaxType)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", customerHandler.ListTaxTypes)
				})
			})
			// ── Main customer CRUD ───────────────────────────────────────────
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", customerHandler.CreateCustomer)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", customerHandler.ListCustomers)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", customerHandler.GetCustomer)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Patch("/{code}/block", customerHandler.BlockCustomer)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Patch("/{code}/unblock", customerHandler.UnblockCustomer)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}/establishments", customerHandler.ListEstablishments)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{code}/addresses", customerHandler.AddAddress)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{code}/contacts", customerHandler.AddContact)
		})

		r.Route("/api/suppliers", func(r chi.Router) {
			// ── Support tables (cadastros de apoio) ──────────────────────────
			r.Route("/support", func(r chi.Router) {
				r.Route("/supplier-types", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", supplierHandler.CreateSupplierType)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", supplierHandler.UpdateSupplierType)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", supplierHandler.ListSupplierTypes)
				})
				r.Route("/contact-types", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", supplierHandler.CreateContactType)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", supplierHandler.ListContactTypes)
				})
				r.Route("/parameters", func(r chi.Router) {
					r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", supplierHandler.UpsertParameters)
					r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{enterpriseCode}", supplierHandler.GetParameters)
				})
			})
			// ── Main supplier CRUD ───────────────────────────────────────────
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", supplierHandler.CreateSupplier)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", supplierHandler.ListSuppliers)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", supplierHandler.GetSupplier)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", supplierHandler.UpdateSupplier)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Patch("/{code}/block", supplierHandler.BlockSupplier)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Patch("/{code}/unblock", supplierHandler.UnblockSupplier)
			r.With(httpmw.RequireRole("ADMIN")).Delete("/{code}", supplierHandler.DeleteSupplier)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}/establishments", supplierHandler.ListEstablishments)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}/purchasing-defaults", supplierHandler.GetPurchasingDefaults)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{code}/sefaz-query", supplierSefazHandler.Query)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/addresses", supplierHandler.AddAddress)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/phones", supplierHandler.AddPhone)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/emails", supplierHandler.AddEmail)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/due-dates", supplierHandler.AddDueDate)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/contacts", supplierHandler.AddContact)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/contacts/phones", supplierHandler.AddContactPhone)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/contacts/emails", supplierHandler.AddContactEmail)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}/enterprises", supplierHandler.ListEnterprises)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/enterprises", supplierHandler.AddEnterprise)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/enterprises", supplierHandler.UpdateEnterprise)
		})

		r.Route("/api/fiscal-classifications", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", fiscalClassHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", fiscalClassHandler.Update)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", fiscalClassHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", fiscalClassHandler.Get)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/languages", fiscalClassHandler.AddLanguage)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/export-attributes", fiscalClassHandler.AddExportAttribute)
		})

		r.Route("/api/item-conversions", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", itemConversionHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/convert", itemConversionHandler.Convert)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/item/{itemCode}", itemConversionHandler.ListByItem)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/{id}", itemConversionHandler.Delete)
		})

		r.Route("/api/purchase-requisitions", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", purchaseRequisitionHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", purchaseRequisitionHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", purchaseRequisitionHandler.Get)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{code}/items", purchaseRequisitionHandler.AddItem)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/generate-orders", purchaseRequisitionHandler.GeneratePurchaseOrders)
		})

		r.Route("/api/purchase-quotations", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", purchaseQuotationHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", purchaseQuotationHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", purchaseQuotationHandler.Get)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{code}/suppliers", purchaseQuotationHandler.AddSupplier)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/prices", purchaseQuotationHandler.RecordPrice)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Patch("/prices/{priceID}/select", purchaseQuotationHandler.SelectPrice)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{code}/generate-orders", purchaseQuotationHandler.GenerateOrders)
		})

		r.Route("/api/entry-operations", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", entryOperationHandler.Create)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", entryOperationHandler.Update)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", entryOperationHandler.List)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", entryOperationHandler.Get)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}/validate", entryOperationHandler.Validate)
			r.Route("/state-groups", func(r chi.Router) {
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", entryOperationHandler.CreateStateGroup)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", entryOperationHandler.ListStateGroups)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", entryOperationHandler.GetStateGroup)
				r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/{code}/ufs", entryOperationHandler.AddStateGroupUF)
			})
		})

		r.Route("/api/item-suppliers", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", itemSupplierHandler.Upsert)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/item/{itemCode}", itemSupplierHandler.ListByItem)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/{id}", itemSupplierHandler.Delete)
		})

		r.Route("/api/purchase-price-tables", func(r chi.Router) {
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/", purchasePriceHandler.CreateTable)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Put("/", purchasePriceHandler.UpdateTable)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/", purchasePriceHandler.ListTables)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}", purchasePriceHandler.GetTable)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Get("/{code}/items", purchasePriceHandler.ListItems)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Post("/items", purchasePriceHandler.AddItem)
			r.With(httpmw.RequireRole("ADMIN", "USER")).Delete("/items/{id}", purchasePriceHandler.DeleteItem)
		})
	})
	// Liveness: process is up (no dependency checks). Kept at /health for
	// backward compatibility, plus the conventional /health/live alias.
	r.Get("/health", app.livenessHandler)
	r.Get("/health/live", app.livenessHandler)

	// Readiness: the process can serve traffic — i.e. the database answers.
	// Load balancers / orchestrators should route only when this is 200.
	r.Get("/health/ready", app.readinessHandler)

	// Prometheus metrics, optionally guarded by a bearer token.
	if app.metrics != nil && app.config.MetricsEnabled {
		r.Get("/metrics", app.metricsHandler())
	}

	return r
}

// corsOrigins splits the comma-separated CORS_ALLOWED_ORIGINS config value.
func (app *application) corsOrigins() []string {
	raw := strings.TrimSpace(app.config.CORSAllowedOrigins)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (app *application) livenessHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"service":   "core-api",
	})
}

func (app *application) readinessHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := app.db.Pool.Ping(ctx); err != nil {
		app.logger.Error("readiness check failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status":   "unavailable",
			"database": "down",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ready",
		"database": "up",
	})
}

// metricsHandler exposes Prometheus metrics, optionally behind a static bearer
// token (METRICS_TOKEN) so the endpoint can be scraped over an untrusted network.
func (app *application) metricsHandler() http.HandlerFunc {
	h := app.metrics.Handler()
	token := app.config.MetricsToken
	return func(w http.ResponseWriter, r *http.Request) {
		if token != "" && r.Header.Get("Authorization") != "Bearer "+token {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		h(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
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
		WriteTimeout: 120 * time.Second,
		ReadTimeout:  30 * time.Second,
		IdleTimeout:  time.Minute,
	}

	// Listen for SIGINT/SIGTERM so the orchestrator (or Ctrl-C) can drain
	// in-flight requests instead of cutting connections mid-transaction.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		app.logger.Info("server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		timeout := time.Duration(app.config.ShutdownTimeoutSec) * time.Second
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		app.logger.Info("shutdown signal received, draining", "timeout", timeout.String())

		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			app.logger.Error("graceful shutdown failed, forcing close", "error", err)
			return srv.Close()
		}
		app.logger.Info("server stopped cleanly")
		return nil
	}
}
