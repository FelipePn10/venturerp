package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/config"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database"
	allocation "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/allocation_base"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/bom"
	bomitem "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/bom_item"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/cost_center"
	deliveryPromiseParams "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_promise_params"
	deliveryReschedule "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_reschedule"
	employee "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/employee"
	enterprise "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/enterprise"
	generatemask "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/generate_mask"
	group "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/group"
	independentDemand "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/independent_demand"
	industrialCalendar "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/industrial_calendar"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	itemquestion "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_question"
	modifier "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/modifier"
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
	logger *slog.Logger
	db     *database.DB
}

func (app *application) traceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		app.logger.Info("request completed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			slog.String("client_ip", r.RemoteAddr),
			slog.Int("status", ww.Status()),
		)
	})
}

func (app *application) mount() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.StripSlashes)
	r.Use(app.traceMiddleware)

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
	employeeHandler := handler.NewCreateEmployeeHandler(createEmployeeUc)

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

	log.Printf("Starting server on %s", addr)
	return srv.ListenAndServe()
}
