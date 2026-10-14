package repository

import (
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

var ErrNotFound = domainerr.NewNotFound("registro não encontrado")
