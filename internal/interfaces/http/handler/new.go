package handler

import (
	"github.com/FelipePn10/panossoerp/internal/application/usecase"
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

func NewCreateEmployeeHandler(
	createEmployeeUc *usecase.CreateEmployeeUseCase,
) *EmployeeHandler {
	return &EmployeeHandler{
		createEmployeeUC: createEmployeeUc,
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
