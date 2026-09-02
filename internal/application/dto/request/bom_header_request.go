package request

import (
	"time"

	"github.com/google/uuid"
)

// CreateBomHeaderDTO: o item chega pelo código de negócio (texto) e o autor vem
// do JWT — a tela não envia, e não seria confiável se enviasse. Versão, situação
// e tipo têm padrão no servidor (1, DRAFT e MBOM).
type CreateBomHeaderDTO struct {
	ItemCode TextCode `json:"item_code"`
	Mask     *string  `json:"mask,omitempty"`
	// BomType é opcional: EBOM (engenharia) ou MBOM (fabricação, padrão).
	BomType   string     `json:"bom_type,omitempty"`
	ValidFrom *time.Time `json:"valid_from,omitempty"`
	CreatedBy uuid.UUID  `json:"-"`
}

type UpdateBomHeaderStatusDTO struct {
	ID     int64  `json:"id"`
	Status string `json:"status"` // DRAFT | APPROVED | OBSOLETE
}
