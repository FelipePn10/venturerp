package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/bom_header/entity"
)

type BomHeaderRepository interface {
	Create(ctx context.Context, h *entity.BomHeader) (*entity.BomHeader, error)
	GetByID(ctx context.Context, id int64) (*entity.BomHeader, error)
	ListByItem(ctx context.Context, itemCode int64) ([]*entity.BomHeader, error)
	NextVersion(ctx context.Context, itemCode int64, mask string) (int32, error)
	// Approve aprova a versão numa transação: torna obsoleta a aprovada anterior
	// do mesmo item/máscara/tipo e grava a foto das linhas atuais.
	Approve(ctx context.Context, id int64, actor uuid.UUID, validFrom time.Time) (*entity.BomHeader, error)
	Obsolete(ctx context.Context, id int64) (*entity.BomHeader, error)
	// ApprovedLines é a foto gravada na aprovação; CurrentLines, a estrutura de agora.
	ApprovedLines(ctx context.Context, id int64) ([]entity.Line, error)
	CurrentLines(ctx context.Context, itemCode int64, mask string) ([]entity.Line, error)
}
