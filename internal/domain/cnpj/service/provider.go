// Package service defines the port through which the application looks a CNPJ up
// against an external registry. Concrete adapters live in
// internal/infrastructure/cnpj.
package service

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/domain/cnpj/entity"
)

// ErrNotFound is returned when the registry has no record for the CNPJ.
var ErrNotFound = errors.New("cnpj não encontrado na base da Receita")

// ErrUnavailable is returned when the external provider is unreachable, timed
// out, or rate-limited. Callers should treat lookups as a non-critical
// convenience and degrade gracefully.
var ErrUnavailable = errors.New("serviço de consulta de CNPJ indisponível")

// Provider resolves company data for a (digits-only) CNPJ.
type Provider interface {
	Lookup(ctx context.Context, cnpj string) (*entity.Company, error)
}
