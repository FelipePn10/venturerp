package structure_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
)

type GetAllDirectChildrenUseCase struct {
	Repo  repository.ItemStructureRepository
	Auth  ports.AuthService
	Items any
}

func NewGetAllDirectChildrenUseCase(
	repo repository.ItemStructureRepository,
	auth ports.AuthService,
	items ...any,
) *GetAllDirectChildrenUseCase {
	uc := &GetAllDirectChildrenUseCase{
		Repo: repo,
		Auth: auth,
	}
	if len(items) > 0 {
		uc.Items = items[0]
	}
	return uc
}

func (uc *GetAllDirectChildrenUseCase) Execute(
	ctx context.Context,
	dto request.GetAllDirectChildrenDTO,
) ([]*response.ItemStructureResponse, error) {

	if !uc.Auth.GetAllStructure(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	parentCode, err := resolveItemCode(ctx, uc.Items, dto.ParentItemCode)
	if err != nil {
		return nil, err
	}
	items, err := uc.Repo.GetAllDirectChildren(ctx, parentCode)
	if err != nil {
		return nil, err
	}
	return toItemStructureResponses(items), nil
}

func toItemStructureResponse(s *entity.ItemStructure) *response.ItemStructureResponse {
	if s == nil {
		return nil
	}
	return &response.ItemStructureResponse{
		ID:                 s.ID,
		ParentCode:         s.ParentCode,
		ChildCode:          s.ChildCode,
		ChildDescription:   s.ChildDescription,
		Inherit:            s.Inherit,
		ParentMask:         s.ParentMask,
		Quantity:           s.Quantity,
		LossPercentage:     s.LossPercentage,
		LossFormula:        s.LossFormula,
		QuantityFormula:    s.QuantityFormula,
		QuantityRounding:   s.QuantityRounding,
		QuantityScale:      s.QuantityScale,
		FormulaVariables:   s.FormulaVariables(),
		UnitOfMeasurement:  string(s.UnitOfMeasurement),
		Sequence:           s.Sequence,
		Notes:              s.Notes,
		StartDate:          s.StartDate,
		EndDate:            s.EndDate,
		IsCoproduct:        s.IsCoproduct,
		IsFixedQty:         s.IsFixedQty,
		SubstituteGroup:    s.SubstituteGroup,
		SubstitutePriority: s.SubstitutePriority,
		IsActive:           s.IsActive,
		CreatedBy:          s.CreatedBy,
		CreatedAt:          s.CreatedAt,
		UpdatedAt:          s.UpdatedAt,
	}
}

func toItemStructureResponses(items []*entity.ItemStructure) []*response.ItemStructureResponse {
	out := make([]*response.ItemStructureResponse, 0, len(items))
	for _, s := range items {
		out = append(out, toItemStructureResponse(s))
	}
	return out
}
