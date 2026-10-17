package request

import (
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/google/uuid"
)

type CreateWarehouseRequestDTO struct {
	Code        TextCode `json:"code"`
	Description string   `json:"description"`

	Location types.TypeLocation  `json:"location"`
	Type     types.TypeWarehouse `json:"type"`

	Disposition         bool `json:"disposition"`
	ReservationsAllowed bool `json:"reservations_allowed"`
	// AllowNegativeStock: nulo = padrão do tipo (linha de produção permite,
	// por causa da baixa automática; os demais não).
	AllowNegativeStock *bool `json:"allow_negative_stock,omitempty"`

	CreatedBy uuid.UUID `json:"-"`
}

// UpdateWarehouseRequestDTO altera o almoxarifado; o código vem da rota e não muda.
type UpdateWarehouseRequestDTO struct {
	Description         string              `json:"description"`
	Location            types.TypeLocation  `json:"location"`
	Type                types.TypeWarehouse `json:"type"`
	Disposition         bool                `json:"disposition"`
	ReservationsAllowed bool                `json:"reservations_allowed"`
	AllowNegativeStock  bool                `json:"allow_negative_stock"`
}
