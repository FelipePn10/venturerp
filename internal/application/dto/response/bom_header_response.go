package response

import "time"

type BomHeaderResponse struct {
	ID int64 `json:"id"`
	// ItemCode é o código de negócio do item (texto), como a tela o conhece.
	ItemCode string `json:"item_code"`
	ItemName string `json:"item_name,omitempty"`
	// LegacyCode é a chave numérica interna, mantida para integrações antigas.
	LegacyCode int64   `json:"legacy_item_code"`
	Mask       *string `json:"mask,omitempty"`
	// BomType: MBOM (fabricação) ou EBOM (engenharia).
	BomType      string `json:"bom_type"`
	BomTypeLabel string `json:"bom_type_label"`
	// Version é atribuída pelo servidor; a tela não a envia.
	Version int32 `json:"version"`
	// Status: DRAFT | APPROVED | OBSOLETE, com rótulo pronto para exibição.
	Status      string     `json:"status"`
	StatusLabel string     `json:"status_label"`
	ValidFrom   *time.Time `json:"valid_from,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
}
