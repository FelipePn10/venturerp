package bom_header_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/bom_header/entity"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
)

// statusLabels traduz a situação do cabeçalho para a tela; o código estável
// continua sendo o valor em inglês gravado no banco.
var statusLabels = map[string]string{
	entity.StatusDraft:    "Rascunho",
	entity.StatusApproved: "Aprovado",
	entity.StatusObsolete: "Obsoleto",
}

// bomTypeLabels traduz o tipo de estrutura para a tela.
var bomTypeLabels = map[string]string{
	"MBOM": "Estrutura de fabricação",
	"EBOM": "Estrutura de engenharia",
}

func toResponse(h *entity.BomHeader, item *itementity.Item) *response.BomHeaderResponse {
	if h == nil {
		return nil
	}
	out := &response.BomHeaderResponse{
		ID:           h.ID,
		LegacyCode:   h.ItemCode,
		Mask:         h.Mask,
		BomType:      h.BomType,
		BomTypeLabel: bomTypeLabels[h.BomType],
		Version:      h.Version,
		Status:       h.Status,
		StatusLabel:  statusLabels[h.Status],
		ValidFrom:    h.ValidFrom,
		IsActive:     h.IsActive,
		CreatedAt:    h.CreatedAt,
	}
	if item != nil {
		out.ItemCode = string(item.BusinessCode)
		out.ItemName = item.Name
	}
	return out
}
