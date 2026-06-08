package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/nfse/entity"
)

type NFSeRepository interface {
	Create(ctx context.Context, n *entity.NFSe) (*entity.NFSe, error)
	GetByID(ctx context.Context, id int64) (*entity.NFSe, error)
	List(ctx context.Context) ([]*entity.NFSe, error)
	UpdateStatus(ctx context.Context, id int64, status entity.NFSeStatus) (*entity.NFSe, error)
	UpdateAuthorization(ctx context.Context, id int64, numeroNFSe, codigoVerificacao, url, focusRef string) (*entity.NFSe, error)
}
