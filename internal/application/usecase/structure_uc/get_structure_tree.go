package structure_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/valueobject"
	mapper "github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/structure"
)

const maxBOMDepth = 30

// GetStructureTreeUseCase retorna a árvore BOM GENÉRICA (sem máscara)
type GetStructureTreeUseCase struct {
	Repo repository.ItemStructureRepository
	Auth ports.AuthService
}

func NewGetStructureTreeUseCase(
	repo repository.ItemStructureRepository,
	auth ports.AuthService,
) *GetStructureTreeUseCase {
	return &GetStructureTreeUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func (uc *GetStructureTreeUseCase) Execute(
	ctx context.Context,
	dto request.GetStructureTreeDTO,
) (*response.StructureTreeResponse, error) {

	if !uc.Auth.GetStructureTree(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	exists, err := uc.Repo.ItemExists(ctx, dto.RootItemCode)
	if err != nil {
		return nil, fmt.Errorf("checking root item: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("item %d not found", dto.RootItemCode)
	}

	visited := make(map[int64]bool)

	nodes, err := uc.buildTree(
		ctx,
		dto.RootItemCode,
		1,
		visited,
	)
	if err != nil {
		return nil, err
	}

	respNodes := mapper.MapNodes(nodes)

	return &response.StructureTreeResponse{
		RootItemCode: dto.RootItemCode,
		RootMask:     nil, // árvore genérica
		Components:   respNodes,
		TotalLevels:  mapper.MaxLevel(respNodes) + 1,
		TotalNodes:   mapper.CountNodes(respNodes),
	}, nil
}

func (uc *GetStructureTreeUseCase) buildTree(
	ctx context.Context,
	parentCode int64,
	level int,
	visited map[int64]bool,
) ([]*valueobject.StructureNode, error) {

	if level > maxBOMDepth {
		return nil, fmt.Errorf("max BOM depth reached (%d)", maxBOMDepth)
	}

	if visited[parentCode] {
		return nil, nil
	}

	visited[parentCode] = true
	defer delete(visited, parentCode)

	children, err := uc.Repo.GetAllDirectChildren(ctx, parentCode)
	if err != nil {
		return nil, fmt.Errorf("fetching children of %d: %w", parentCode, err)
	}

	nodes := make([]*valueobject.StructureNode, 0, len(children))

	for _, comp := range children {

		code, desc, err := uc.Repo.GetItemCodeAndDesc(ctx, comp.ChildCode)
		if err != nil {
			return nil, fmt.Errorf("fetching item %d: %w", comp.ChildCode, err)
		}

		node := valueobject.NewStructureNode(
			comp,
			code,
			desc,
			level,
			nil, // árvore genérica NÃO TEM máscara
		)

		sub, err := uc.buildTree(
			ctx,
			comp.ChildCode,
			level+1,
			visited,
		)
		if err != nil {
			return nil, err
		}

		for _, s := range sub {
			node.AddChild(s)
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}
