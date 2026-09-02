package structure_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	sqrepo "github.com/FelipePn10/panossoerp/internal/domain/structure_query/repository"
)

type WhereUsedUseCase struct {
	repo  sqrepo.StructureQueryRepository
	items any
}

func NewWhereUsedUseCase(repo sqrepo.StructureQueryRepository, items ...any) *WhereUsedUseCase {
	uc := &WhereUsedUseCase{repo: repo}
	if len(items) > 0 {
		uc.items = items[0]
	}
	return uc
}

func (uc *WhereUsedUseCase) Execute(ctx context.Context, publicCode request.TextCode, levels int) (*response.WhereUsedResponse, error) {
	itemCode, err := resolveItemCode(ctx, uc.items, publicCode)
	if err != nil {
		return nil, err
	}
	rows, err := uc.repo.GetWhereUsed(ctx, itemCode, levels)
	if err != nil {
		return nil, err
	}

	out := make([]response.WhereUsedRowResponse, 0, len(rows))
	for _, r := range rows {
		row := response.WhereUsedRowResponse{
			Level:             r.Level,
			ParentCode:        r.ParentCode,
			ParentDescription: r.ParentDescription,
			ChildCode:         r.ChildCode,
			Quantity:          r.Quantity,
			LossPercentage:    r.LossPercentage,
			Sequence:          r.Sequence,
			ParentMask:        r.ParentMask,
		}
		out = append(out, row)
	}
	return &response.WhereUsedResponse{ItemCode: itemCode, Rows: out}, nil
}
