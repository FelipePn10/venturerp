package usecase

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/structure_query/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/structure_query/service"
	mapper "github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/structure_query"
)

type ResolveStructureQueryUseCase struct {
	repo     repository.StructureQueryRepository
	resolver *service.Resolver
	auth     ports.AuthService
}

func (uc *ResolveStructureQueryUseCase) Execute(
	ctx context.Context,
	dto request.ResolveStructureQueryDTO,
) (*response.StructureTreeResponse, error) {
	if !uc.auth.CanResolveStructure(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	// CreatedBy é necessário para auto-criar máscaras propagadas durante a consulta.
	createdBy, err := uc.auth.UserID(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolving authenticated user: %w", err)
	}

	if dto.ItemCode <= 0 {
		return nil, fmt.Errorf("invalid item code")
	}

	var nodes []*service.Node

	if dto.Mask == "" {
		nodes, err = uc.resolver.ResolveGeneric(ctx, dto.ItemCode, 1, make(map[int64]bool))
	} else {
		rootAnswers, err := uc.repo.GetMaskAnswersByItemAndValue(ctx, dto.ItemCode, dto.Mask)
		if err != nil {
			return nil, fmt.Errorf("fetching mask answers for item %d mask %q: %w", dto.ItemCode, dto.Mask, err)
		}
		if len(rootAnswers) == 0 {
			return nil, fmt.Errorf("mask %q not registered for item %d", dto.Mask, dto.ItemCode)
		}
		nodes, err = uc.resolver.Resolve(ctx, dto.ItemCode, dto.Mask, rootAnswers, 1, make(map[int64]bool), createdBy)
	}

	if err != nil {
		return nil, fmt.Errorf("resolving structure for item %d: %w", dto.ItemCode, err)
	}

	respNodes := mapper.MapNodes(nodes)
	return &response.StructureTreeResponse{
		RootItemCode: dto.ItemCode,
		RootMask:     nullableString(dto.Mask),
		Components:   respNodes,
		TotalNodes:   countNodes(respNodes),
		TotalLevels:  maxLevel(respNodes),
	}, nil
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func countNodes(nodes []*response.StructureTreeNodeResponse) int {
	total := 0
	for _, n := range nodes {
		total += 1 + countNodes(n.Children)
	}
	return total
}

func maxLevel(nodes []*response.StructureTreeNodeResponse) int {
	max := 0
	for _, n := range nodes {
		if n.Level > max {
			max = n.Level
		}
		if childMax := maxLevel(n.Children); childMax > max {
			max = childMax
		}
	}
	return max
}
