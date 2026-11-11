package entity

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BOM statuses (engineering approval lifecycle).
const (
	StatusDraft    = "DRAFT"
	StatusApproved = "APPROVED"
	StatusObsolete = "OBSOLETE"
)

// BomHeader is the versioning/approval header of a product's structure. The lines
// live in item_structures; this carries version, status and BOM type per item+mask.
type BomHeader struct {
	ID        int64
	ItemCode  int64
	Mask      *string
	BomType   string // EBOM | MBOM (free VARCHAR)
	Version   int32
	Status    string // DRAFT | APPROVED | OBSOLETE
	ValidFrom *time.Time
	IsActive  bool
	CreatedBy uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	// Quem aprovou e quando; quando ficou obsoleta.
	ApprovedBy  *uuid.UUID
	ApprovedAt  *time.Time
	ObsoletedAt *time.Time
	Notes       *string
}

func ValidStatus(s string) bool {
	switch s {
	case StatusDraft, StatusApproved, StatusObsolete:
		return true
	default:
		return false
	}
}

func NewBomHeader(itemCode int64, mask *string, bomType string, version int32, validFrom *time.Time, createdBy uuid.UUID) (*BomHeader, error) {
	if itemCode <= 0 {
		return nil, domainerr.NewValidation("informe o item do cabeçalho de estrutura")
	}
	if bomType == "" {
		bomType = "MBOM"
	}
	if version <= 0 {
		version = 1
	}
	return &BomHeader{
		ItemCode:  itemCode,
		Mask:      mask,
		BomType:   bomType,
		Version:   version,
		Status:    StatusDraft,
		ValidFrom: validFrom,
		IsActive:  true,
		CreatedBy: createdBy,
	}, nil
}

// Tipos de estrutura: MBOM é a de fabricação (a que a produção usa); EBOM é a de
// engenharia (projeto), que não governa a produção.
const (
	TypeMBOM = "MBOM"
	TypeEBOM = "EBOM"
)

// PodeIrPara diz se a versão pode passar para a situação pedida. O ciclo é
// fechado como no SAP/Protheus: rascunho é aprovado ou descartado; aprovada só
// fica obsoleta (por outra aprovação ou à mão); obsoleta é definitiva. Aprovada
// não volta a rascunho — muda-se a estrutura abrindo uma nova versão.
func (h *BomHeader) PodeIrPara(status string) error {
	switch {
	case h.Status == status:
		return domainerr.NewValidation("a versão já está nesta situação")
	case h.Status == StatusObsolete:
		return domainerr.NewValidation("versão obsoleta não muda mais de situação: abra uma nova versão")
	case status == StatusDraft:
		return domainerr.NewValidation("versão aprovada não volta a rascunho: abra uma nova versão para alterar a estrutura")
	case h.Status == StatusDraft && (status == StatusApproved || status == StatusObsolete):
		return nil
	case h.Status == StatusApproved && status == StatusObsolete:
		return nil
	}
	return domainerr.NewValidation("mudança de situação inválida")
}

// Line é uma linha da estrutura: a atual (item_structures) ou a da foto tirada na
// aprovação (bom_header_lines).
type Line struct {
	ChildCode       int64
	ParentMask      string
	Sequence        int32
	Quantity        decimal.Decimal
	Unit            string
	LossPercentage  decimal.Decimal
	QuantityFormula string
	StartDate       *time.Time
	EndDate         *time.Time
	IsCoproduct     bool
}

// Change é uma diferença entre a estrutura aprovada e a atual.
type Change struct {
	Kind      string // INCLUIDO | REMOVIDO | ALTERADO
	ChildCode int64
	Sequence  int32
	Detail    string
}

func (l Line) chave() string { return fmt.Sprintf("%d|%d|%s", l.ChildCode, l.Sequence, l.ParentMask) }

func dataStr(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return t.Format("02/01/2006")
}

// CompararLinhas diz o que mudou da estrutura aprovada para a atual, componente a
// componente. A ordem segue a sequência da estrutura.
func CompararLinhas(aprovada, atual []Line) []Change {
	antes := map[string]Line{}
	for _, l := range aprovada {
		antes[l.chave()] = l
	}
	out := []Change{}
	vistos := map[string]bool{}
	for _, n := range atual {
		k := n.chave()
		vistos[k] = true
		a, ok := antes[k]
		if !ok {
			out = append(out, Change{Kind: "INCLUIDO", ChildCode: n.ChildCode, Sequence: n.Sequence, Detail: fmt.Sprintf("quantidade %s %s", n.Quantity.String(), n.Unit)})
			continue
		}
		var dif []string
		if !a.Quantity.Equal(n.Quantity) {
			dif = append(dif, fmt.Sprintf("quantidade %s → %s", a.Quantity.String(), n.Quantity.String()))
		}
		if a.Unit != n.Unit {
			dif = append(dif, fmt.Sprintf("unidade %s → %s", a.Unit, n.Unit))
		}
		if !a.LossPercentage.Equal(n.LossPercentage) {
			dif = append(dif, fmt.Sprintf("perda %s%% → %s%%", a.LossPercentage.String(), n.LossPercentage.String()))
		}
		if a.QuantityFormula != n.QuantityFormula {
			dif = append(dif, "fórmula de quantidade alterada")
		}
		if dataStr(a.StartDate) != dataStr(n.StartDate) || dataStr(a.EndDate) != dataStr(n.EndDate) {
			dif = append(dif, fmt.Sprintf("validade %s–%s → %s–%s", dataStr(a.StartDate), dataStr(a.EndDate), dataStr(n.StartDate), dataStr(n.EndDate)))
		}
		if a.IsCoproduct != n.IsCoproduct {
			dif = append(dif, "coproduto alterado")
		}
		if len(dif) > 0 {
			out = append(out, Change{Kind: "ALTERADO", ChildCode: n.ChildCode, Sequence: n.Sequence, Detail: strings.Join(dif, "; ")})
		}
	}
	for _, a := range aprovada {
		if !vistos[a.chave()] {
			out = append(out, Change{Kind: "REMOVIDO", ChildCode: a.ChildCode, Sequence: a.Sequence, Detail: fmt.Sprintf("quantidade %s %s", a.Quantity.String(), a.Unit)})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Sequence < out[j].Sequence })
	return out
}
