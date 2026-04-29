package handler

import (
	"github.com/FelipePn10/panossoerp/internal/application/usecase"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/allocation_base_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_reschedule_uc"
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
	createEmployeeUC *usecase.CreateEmployeeUseCase
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
	createUC *delivery_reschedule_uc.CreateDeliveryRescheduleUseCase
	listUC   *delivery_reschedule_uc.ListDeliveryReschedulesUseCase
}
