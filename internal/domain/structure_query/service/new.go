package service

import "github.com/FelipePn10/panossoerp/internal/domain/structure_query/repository"

type Resolver struct {
	repo repository.StructureQueryRepository
}

func NewResolver(repo repository.StructureQueryRepository) *Resolver {
	return &Resolver{repo: repo}
}
