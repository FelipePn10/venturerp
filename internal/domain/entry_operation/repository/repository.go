package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/entry_operation/entity"
)

type EntryOperationRepository interface {
	// State groups
	CreateStateGroup(ctx context.Context, g *entity.StateGroup) (*entity.StateGroup, error)
	GetStateGroupByCode(ctx context.Context, code int64) (*entity.StateGroup, error)
	ListStateGroups(ctx context.Context) ([]*entity.StateGroup, error)
	NextStateGroupCode(ctx context.Context) (int64, error)
	AddStateGroupUF(ctx context.Context, stateGroupCode int64, uf string) error
	ListStateGroupUFs(ctx context.Context, stateGroupCode int64) ([]string, error)
	UFInGroup(ctx context.Context, stateGroupCode int64, uf string) (bool, error)

	// Entry operation types
	CreateEntryOperation(ctx context.Context, o *entity.EntryOperationType) (*entity.EntryOperationType, error)
	UpdateEntryOperation(ctx context.Context, o *entity.EntryOperationType) (*entity.EntryOperationType, error)
	GetEntryOperationByCode(ctx context.Context, code int64) (*entity.EntryOperationType, error)
	ListEntryOperations(ctx context.Context, onlyActive bool) ([]*entity.EntryOperationType, error)
	NextEntryOperationCode(ctx context.Context) (int64, error)
}
