package usecase

import (
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/allocation_base_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/cost_center_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_promise_params_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_reschedule_uc"
	employee2 "github.com/FelipePn10/panossoerp/internal/application/usecase/employee"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/independent_demand_uc"
	industrial_calendar_uc "github.com/FelipePn10/panossoerp/internal/application/usecase/industrial_calendar"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_calendar_promise_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/machine_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_calculation_uc"
	mrpports "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/ports"
	mrpservice "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/service"
	allocation "github.com/FelipePn10/panossoerp/internal/domain/allocation_base/repository"
	ast "github.com/FelipePn10/panossoerp/internal/domain/associate_questions/repository"
	bom "github.com/FelipePn10/panossoerp/internal/domain/bom/repository"
	bomitem "github.com/FelipePn10/panossoerp/internal/domain/bom_items/repository"
	component "github.com/FelipePn10/panossoerp/internal/domain/component/repository"
	cost_center "github.com/FelipePn10/panossoerp/internal/domain/cost_center/repository"
	delivery_promise_params "github.com/FelipePn10/panossoerp/internal/domain/delivery_promise_params/repository"
	delivery_reschedule "github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/repository"
	employee "github.com/FelipePn10/panossoerp/internal/domain/employee/repository"
	enterprise "github.com/FelipePn10/panossoerp/internal/domain/enterprise/repository"
	mask "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/repository"
	group "github.com/FelipePn10/panossoerp/internal/domain/group/repository"
	independent_demand "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository"
	industrial_calendar "github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/repository"
	item_calendar_promise "github.com/FelipePn10/panossoerp/internal/domain/item_calendar_promise/repository"
	item "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	machine "github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
	modifier "github.com/FelipePn10/panossoerp/internal/domain/modifier/repository"
	mrp_calculation "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
	qst "github.com/FelipePn10/panossoerp/internal/domain/questions/repository"
	qstops "github.com/FelipePn10/panossoerp/internal/domain/questions_options/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
	qr "github.com/FelipePn10/panossoerp/internal/domain/structure_query/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/structure_query/service"
	user "github.com/FelipePn10/panossoerp/internal/domain/user/repository"
	warehouse "github.com/FelipePn10/panossoerp/internal/domain/warehouse/repository"
)

func NewFindItemByCode(
	repo item.ItemRepository,
	auth ports.AuthService,
) *FindItemByCode {
	return &FindItemByCode{
		repo: repo,
		auth: auth,
	}
}

func NewFindQuestionByName(
	repo qst.QuestionsRepository,
) *FindQuestionByName {
	return &FindQuestionByName{
		repo: repo,
	}
}

func NewDeleteQuestionUseCase(
	repo qst.QuestionsRepository,
) *DeleteQuestionUseCase {
	return &DeleteQuestionUseCase{
		repo: repo,
	}
}

// func NewSearchByIDProductUseCase(
// 	repo repository.ProductRepository,
// ) *SearchByIDProductUseCase {
// 	return &SearchByIDProductUseCase{
// 		repo: repo,
// 	}
// }

func NewLoginUserUseCase(
	repo user.UserRepository,
) *LoginUserUseCase {
	return &LoginUserUseCase{repo: repo}
}

func NewRegisterUserUseCase(
	repo user.UserRepository,
) *RegisterUserUseCase {
	return &RegisterUserUseCase{repo: repo}
}

func NewQuestionUserUseCase(
	repo qst.QuestionsRepository,
	auth ports.AuthService,
) *CreateQuestion {
	return &CreateQuestion{
		repo: repo,
		auth: auth,
	}
}

func NewCreateQuestionOptionUseCase(
	repo qstops.QuestionsOptionsRepository,
	auth ports.AuthService,
) *CreateQuestionOptionUseCase {
	return &CreateQuestionOptionUseCase{
		repo: repo,
		auth: auth,
	}
}
func NewDeleteQuestionOptionUseCase(
	repo qstops.QuestionsOptionsRepository,
) *DeleteQuestionOptionUseCase {
	return &DeleteQuestionOptionUseCase{
		repo: repo,
	}
}

func NewAssociateByQuestionItemUseCase(
	repo ast.AssociateQuestionsRepository,
	auth ports.AuthService,
) *AssociateByQuestionItemUseCase {
	return &AssociateByQuestionItemUseCase{
		repo: repo,
		auth: auth,
	}
}

func NewGenerateMaskItemUseCase(
	repo mask.GenerateMaskForItemRepository,
	auth ports.AuthService,
) *GenerateMaskForItemUseCase {
	return &GenerateMaskForItemUseCase{
		repo: repo,
		auth: auth,
	}
}
func NewCreateBomUseCase(
	repo bom.BomRepository,
	auth ports.AuthService,
) *CreateBomUseCase {
	return &CreateBomUseCase{
		repo: repo,
		auth: auth,
	}
}

func NewCreatBomItemUseCase(
	repo bomitem.BomItemsRepository,
	auth ports.AuthService,
) *CreateBomItemUseCase {
	return &CreateBomItemUseCase{
		repo: repo,
		auth: auth,
	}
}

func NewCreateItem(
	repo item.ItemRepository,
	auth ports.AuthService,
) *CreateItemUseCase {
	return &CreateItemUseCase{
		repo: repo,
		auth: auth,
	}
}

func NewCreateComponentUseCase(
	repo component.ComponentRepository,
	auth ports.AuthService,
) *CreateComponentUseCase {
	return &CreateComponentUseCase{
		repo: repo,
		auth: auth,
	}
}

func NewCreateWarehouseUseCase(
	repo warehouse.WarehouseRepository,
	auth ports.AuthService,
) *CreateWarehouseUseCase {
	return &CreateWarehouseUseCase{
		repo: repo,
		auth: auth,
	}
}

func NewCreateGroupUseCase(
	repo group.GroupRepository,
	auth ports.AuthService,
) *CreateGroupUseCase {
	return &CreateGroupUseCase{
		repo: repo,
		auth: auth,
	}
}

func NewCreateEnterpriseUseCase(
	repo enterprise.EnterpriseRepository,
	auth ports.AuthService,
) *CreateEnterpriseUseCase {
	return &CreateEnterpriseUseCase{
		repo: repo,
		auth: auth,
	}
}

func NewCreateModifierUseCase(
	repo modifier.ModifierRepository,
	auth ports.AuthService,
) *CreateModifierUseCase {
	return &CreateModifierUseCase{
		repo: repo,
		auth: auth,
	}
}

func NewCreateEmployeeUseCase(
	repo employee.EmployeeRepository,
	auth ports.AuthService,
) *employee2.CreateEmployeeUseCase {
	return &employee2.CreateEmployeeUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewCreateStructureComponentUseCase(
	repo repository.ItemStructureRepository,
	auth ports.AuthService,
) *CreateStructureComponentUseCase {
	return &CreateStructureComponentUseCase{
		repo: repo,
		auth: auth,
	}
}

func NewUpdateStructureComponentUseCase(
	repo repository.ItemStructureRepository,
	auth ports.AuthService,
) *UpdateStructureComponentUseCase {
	return &UpdateStructureComponentUseCase{
		repo: repo,
		auth: auth,
	}
}

func NewGetStructureTreeUseCase(
	repo repository.ItemStructureRepository,
	auth ports.AuthService,
) *GetStructureTreeUseCase {
	return &GetStructureTreeUseCase{
		repo: repo,
		auth: auth,
	}
}

//func NewResolveStructureForMaskUseCase(
//	repo repository.ItemStructureRepository,
//	auth ports.AuthService,
//) *ResolveStructureForMaskUseCase {
//	return &ResolveStructureForMaskUseCase{
//		repo: repo,
//		auth: auth,
//	}
//}

func NewResolveStructureQueryUseCase(
	repo qr.StructureQueryRepository,
	auth ports.AuthService,
) *ResolveStructureQueryUseCase {
	return &ResolveStructureQueryUseCase{
		repo:     repo,
		resolver: service.NewResolver(repo),
		auth:     auth,
	}
}

func NewGetAllDirectChildrenUseCase(
	repo repository.ItemStructureRepository,
	auth ports.AuthService,
) *GetAllDirectChildrenUseCase {
	return &GetAllDirectChildrenUseCase{
		repo: repo,
		auth: auth,
	}
}

func NewCreateAllocationBaseUseCase(
	repo allocation.AllocationBaseRepository,
	auth ports.AuthService,
) *allocation_base_uc.CreateAllocationBaseUseCase {
	return &allocation_base_uc.CreateAllocationBaseUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewListAllocationBasesUseCase(
	repo allocation.AllocationBaseRepository,
	auth ports.AuthService,
) *allocation_base_uc.ListAllocationBasesUseCase {
	return &allocation_base_uc.ListAllocationBasesUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewCreateCostCenterUseCase(
	repo cost_center.CostCenterRepository,
	auth ports.AuthService,
) *cost_center_uc.CreateCostCenterUseCase {
	return &cost_center_uc.CreateCostCenterUseCase{
		Repo: repo,
		Auth: auth}
}

func NewGetCostCenterUseCase(
	repo cost_center.CostCenterRepository,
	auth ports.AuthService,
) *cost_center_uc.GetCostCenterUseCase {
	return &cost_center_uc.GetCostCenterUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewListCostCentersUseCase(
	repo cost_center.CostCenterRepository,
	auth ports.AuthService,
) *cost_center_uc.ListCostCentersUseCase {
	return &cost_center_uc.ListCostCentersUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewManageDeliveryPromiseParamsUseCase(
	repo delivery_promise_params.DeliveryPromiseParamsRepository,
	auth ports.AuthService,
) *delivery_promise_params_uc.ManageDeliveryPromiseParamsUseCase {
	return &delivery_promise_params_uc.ManageDeliveryPromiseParamsUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewCreateDeliveryRescheduleUseCase(
	repo delivery_reschedule.DeliveryRescheduleRepository,
	auth ports.AuthService,
) *delivery_reschedule_uc.CreateDeliveryRescheduleUseCase {
	return &delivery_reschedule_uc.CreateDeliveryRescheduleUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewListDeliveryReschedulesUseCase(
	repo delivery_reschedule.DeliveryRescheduleRepository,
	auth ports.AuthService,
) *delivery_reschedule_uc.ListDeliveryReschedulesUseCase {
	return &delivery_reschedule_uc.ListDeliveryReschedulesUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewCreateIndependentDemandUseCase(
	repo independent_demand.IndependentDemandRepository,
	auth ports.AuthService,
) *independent_demand_uc.CreateIndependentDemandUseCase {
	return &independent_demand_uc.CreateIndependentDemandUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewListIndependentDemandsUseCase(
	repo independent_demand.IndependentDemandRepository,
	auth ports.AuthService,
) *independent_demand_uc.ListIndependentDemandsUseCase {
	return &independent_demand_uc.ListIndependentDemandsUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewGetIndependentDemandByCodeUseCase(
	repo independent_demand.IndependentDemandRepository,
	auth ports.AuthService,
) *independent_demand_uc.GetIndependentDemandByCodeUseCase {
	return &independent_demand_uc.GetIndependentDemandByCodeUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewListIndependentDemandByItemUseCase(
	repo independent_demand.IndependentDemandRepository,
	auth ports.AuthService,
) *independent_demand_uc.ListIndependentDemandByItemUseCase {
	return &independent_demand_uc.ListIndependentDemandByItemUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewUpdateIndependentDemandUseCase(
	repo independent_demand.IndependentDemandRepository,
	auth ports.AuthService,
) *independent_demand_uc.UpdateIndependentDemandUseCase {
	return &independent_demand_uc.UpdateIndependentDemandUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewListIndependentDemandFromDateUseCase(
	repo independent_demand.IndependentDemandRepository,
	auth ports.AuthService,
) *independent_demand_uc.ListIndependentDemandFromDateUseCase {
	return &independent_demand_uc.ListIndependentDemandFromDateUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewDeleteIndependentDemandUseCase(
	repo independent_demand.IndependentDemandRepository,
	auth ports.AuthService,
) *independent_demand_uc.DeleteIndependentDemandUseCase {
	return &independent_demand_uc.DeleteIndependentDemandUseCase{
		Repo: repo,
		Auth: auth,
	}
}
func NewManageCalendarUseCase(
	repo industrial_calendar.IndustrialCalendarRepository,
	auth ports.AuthService,
) *industrial_calendar_uc.ManageCalendarUseCase {
	return &industrial_calendar_uc.ManageCalendarUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewManageItemCalendarPromiseUseCase(
	repo item_calendar_promise.ItemCalendarPromiseRepository,
	auth ports.AuthService,
) *item_calendar_promise_uc.ManageItemCalendarPromiseUseCase {
	return &item_calendar_promise_uc.ManageItemCalendarPromiseUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewCreateItemMachineTimeUseCase(
	repo machine.MachineRepository,
	itemRepo item.ItemRepository,
	auth ports.AuthService,
) *machine_uc.CreateItemMachineTimeUseCase {
	return &machine_uc.CreateItemMachineTimeUseCase{
		Repo:     repo,
		ItemRepo: itemRepo,
		Auth:     auth,
	}
}

func NewCalculateProductionTimeUseCase(
	repo machine.MachineRepository,
	itemRepo item.ItemRepository,
	auth ports.AuthService,
) *machine_uc.CalculateProductionTimeUseCase {
	return &machine_uc.CalculateProductionTimeUseCase{
		Repo:     repo,
		ItemRepo: itemRepo,
		Auth:     auth,
	}
}

func NewCreateMachineUseCase(
	repo machine.MachineRepository,
	auth ports.AuthService,
) *machine_uc.CreateMachineUseCase {
	return &machine_uc.CreateMachineUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewCreateMachineTypeUseCase(
	repo machine.MachineRepository,
	auth ports.AuthService,
) *machine_uc.CreateMachineTypeUseCase {
	return &machine_uc.CreateMachineTypeUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewListItemMachineTimesUseCase(
	repo machine.MachineRepository,
	auth ports.AuthService,
) *machine_uc.ListItemMachineTimesUseCase {
	return &machine_uc.ListItemMachineTimesUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewListMachinesUseCase(
	repo machine.MachineRepository,
	auth ports.AuthService,
) *machine_uc.ListMachinesUseCase {
	return &machine_uc.ListMachinesUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewListMachineTypesUseCase(
	repo machine.MachineRepository,
	auth ports.AuthService,
) *machine_uc.ListMachineTypesUseCase {
	return &machine_uc.ListMachineTypesUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewScheduleMachineUseCase(
	repo machine.MachineRepository,
	auth ports.AuthService,
) *machine_uc.ScheduleMachineUseCase {
	return &machine_uc.ScheduleMachineUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewUpdateMachineTypeUseCase(
	repo machine.MachineRepository,
	auth ports.AuthService,
) *machine_uc.UpdateMachineTypeUseCase {
	return &machine_uc.UpdateMachineTypeUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewGetMachineTypeUseCase(
	repo machine.MachineRepository,
	auth ports.AuthService,
) *machine_uc.GetMachineTypeUseCase {
	return &machine_uc.GetMachineTypeUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewDeleteMachineTypeUseCase(
	repo machine.MachineRepository,
	auth ports.AuthService,
) *machine_uc.DeleteMachineTypeUseCase {
	return &machine_uc.DeleteMachineTypeUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewUpdateMachineUseCase(
	repo machine.MachineRepository,
	auth ports.AuthService,
) *machine_uc.UpdateMachineUseCase {
	return &machine_uc.UpdateMachineUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewGetMachineUseCase(
	repo machine.MachineRepository,
	auth ports.AuthService,
) *machine_uc.GetMachineUseCase {
	return &machine_uc.GetMachineUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewListMachinesByTypeUseCase(
	repo machine.MachineRepository,
	auth ports.AuthService,
) *machine_uc.ListMachinesByTypeUseCase {
	return &machine_uc.ListMachinesByTypeUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewDeleteMachineUseCase(
	repo machine.MachineRepository,
	auth ports.AuthService,
) *machine_uc.DeleteMachineUseCase {
	return &machine_uc.DeleteMachineUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewGetItemMachineTimeUseCase(
	repo machine.MachineRepository,
	auth ports.AuthService,
) *machine_uc.GetItemMachineTimeUseCase {
	return &machine_uc.GetItemMachineTimeUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewListItemsByMachineUseCase(
	repo machine.MachineRepository,
	auth ports.AuthService,
) *machine_uc.ListItemsByMachineUseCase {
	return &machine_uc.ListItemsByMachineUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewMRPService(
	mrpRepo mrp_calculation.MRPCalculationRepository,
	structRepo repository.ItemStructureRepository,
	demandRepo independent_demand.IndependentDemandRepository,
	calRepo industrial_calendar.IndustrialCalendarRepository,
	itemRepo item.ItemRepository,
	supplyPort mrpports.PlannedOrderSupplyPort, // pass nil until planned_order is created
) mrpservice.MRPService {
	return mrpservice.NewMRPService(mrpRepo, structRepo, demandRepo, calRepo, itemRepo, supplyPort)
}

func NewRunMRPCalculationUseCase(
	svc mrpservice.MRPService,
	auth ports.AuthService,
) *mrp_calculation_uc.RunMRPCalculationUseCase {
	return &mrp_calculation_uc.RunMRPCalculationUseCase{
		Service: svc,
		Auth:    auth,
	}
}

func NewManageConfiguredItemRulesUseCase(
	repo mrp_calculation.MRPCalculationRepository,
	auth ports.AuthService,
) *mrp_calculation_uc.ManageConfiguredItemRulesUseCase {
	return &mrp_calculation_uc.ManageConfiguredItemRulesUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func NewGetItemProfileUseCase(
	repo mrp_calculation.MRPCalculationRepository,
	auth ports.AuthService,
) *mrp_calculation_uc.GetItemProfileUseCase {
	return &mrp_calculation_uc.GetItemProfileUseCase{
		Repo: repo,
		Auth: auth,
	}
}
