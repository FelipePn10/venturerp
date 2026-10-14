package repository

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/domain/commercial_commission/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

var ErrInvalidState = domainerr.NewValidation("estado da comissão não permite a operação")
var ErrIdempotencyConflict = domainerr.NewConflict("chave de idempotência reutilizada em outra operação")

type Repository interface {
	UpsertSettings(context.Context, int64, *entity.Settings) (*entity.Settings, error)
	GetSettings(context.Context, int64) (*entity.Settings, error)
	List(context.Context, int64, entity.Filter) ([]*entity.LedgerEntry, error)
	Transition(context.Context, int64, entity.TransitionCommand) (*entity.LedgerEntry, error)
}
