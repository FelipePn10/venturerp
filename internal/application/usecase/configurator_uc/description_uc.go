package configurator_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

const descTypeCompMascara = "COMP_MASCARA"

// ─── Tipos de Descrição ───────────────────────────────────────────────────────

func (uc *ConfiguratorUseCase) CreateDescriptionType(ctx context.Context, dto request.CfgDescriptionTypeDTO) (*response.CfgDescriptionTypeResponse, error) {
	if dto.Code == "" || dto.Description == "" {
		return nil, fmt.Errorf("código e descrição do tipo são obrigatórios")
	}
	kind := dto.Kind
	if kind == "" {
		kind = "GERAL"
	}
	row, err := uc.Q.CreateCfgDescriptionType(ctx, dto.Code, dto.Description, kind, pgutil.ToPgUUID(dto.CreatedBy))
	if err != nil {
		return nil, fmt.Errorf("criando tipo de descrição: %w", err)
	}
	return descTypeToResponse(row), nil
}

func (uc *ConfiguratorUseCase) UpdateDescriptionType(ctx context.Context, dto request.CfgDescriptionTypeDTO) (*response.CfgDescriptionTypeResponse, error) {
	if dto.Code == "" || dto.Description == "" {
		return nil, fmt.Errorf("código e descrição do tipo são obrigatórios")
	}
	kind := dto.Kind
	if kind == "" {
		kind = "GERAL"
	}
	row, err := uc.Q.UpdateCfgDescriptionType(ctx, dto.ID, dto.Code, dto.Description, kind, dto.IsActive)
	if err != nil {
		return nil, fmt.Errorf("atualizando tipo de descrição: %w", err)
	}
	return descTypeToResponse(row), nil
}

func (uc *ConfiguratorUseCase) GetDescriptionType(ctx context.Context, id int64) (*response.CfgDescriptionTypeResponse, error) {
	row, err := uc.Q.GetCfgDescriptionType(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("tipo de descrição não encontrado: %w", err)
	}
	return descTypeToResponse(row), nil
}

func (uc *ConfiguratorUseCase) ListDescriptionTypes(ctx context.Context, onlyActive bool) ([]*response.CfgDescriptionTypeResponse, error) {
	rows, err := uc.Q.ListCfgDescriptionTypes(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CfgDescriptionTypeResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, descTypeToResponse(r))
	}
	return out, nil
}

func (uc *ConfiguratorUseCase) DeactivateDescriptionType(ctx context.Context, id int64) error {
	return uc.Q.DeactivateCfgDescriptionType(ctx, id)
}

// ─── Descrição de Itens Configurados ──────────────────────────────────────────

// CreateItemDescription creates (or returns) the header for an (item, type) pair
// and loads one grid line per item characteristic.
func (uc *ConfiguratorUseCase) CreateItemDescription(ctx context.Context, dto request.CreateCfgItemDescriptionDTO) (*response.CfgItemDescriptionResponse, error) {
	if dto.ItemCode <= 0 || dto.DescriptionTypeID <= 0 {
		return nil, fmt.Errorf("item_code e description_type_id são obrigatórios")
	}
	if _, err := uc.Q.GetCfgDescriptionType(ctx, dto.DescriptionTypeID); err != nil {
		return nil, fmt.Errorf("tipo de descrição %d não encontrado", dto.DescriptionTypeID)
	}
	// idempotent: reuse an existing header
	if existing, err := uc.Q.GetCfgItemDescriptionByItemType(ctx, dto.ItemCode, dto.DescriptionTypeID); err == nil {
		return uc.itemDescriptionView(ctx, existing)
	}
	header, err := uc.Q.CreateCfgItemDescription(ctx, dto.ItemCode, dto.DescriptionTypeID, pgutil.ToPgUUID(dto.CreatedBy))
	if err != nil {
		return nil, fmt.Errorf("criando descrição do item: %w", err)
	}
	if err := uc.loadLines(ctx, header.ID, dto.ItemCode); err != nil {
		return nil, err
	}
	return uc.itemDescriptionView(ctx, header)
}

func (uc *ConfiguratorUseCase) GetItemDescription(ctx context.Context, id int64) (*response.CfgItemDescriptionResponse, error) {
	header, err := uc.Q.GetCfgItemDescription(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("descrição do item não encontrada: %w", err)
	}
	return uc.itemDescriptionView(ctx, header)
}

func (uc *ConfiguratorUseCase) ListItemDescriptionsByItem(ctx context.Context, itemCode int64) ([]*response.CfgItemDescriptionResponse, error) {
	headers, err := uc.Q.ListCfgItemDescriptionsByItem(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CfgItemDescriptionResponse, 0, len(headers))
	for _, h := range headers {
		v, err := uc.itemDescriptionView(ctx, h)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// ReloadLines rebuilds the grid from the item's current characteristics (use
// after the item characteristics change).
func (uc *ConfiguratorUseCase) ReloadLines(ctx context.Context, id int64) (*response.CfgItemDescriptionResponse, error) {
	header, err := uc.Q.GetCfgItemDescription(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("descrição do item não encontrada: %w", err)
	}
	if err := uc.Q.DeleteCfgItemDescriptionLinesByHeader(ctx, id); err != nil {
		return nil, fmt.Errorf("recarregando linhas: %w", err)
	}
	if err := uc.loadLines(ctx, id, header.ItemCode); err != nil {
		return nil, err
	}
	return uc.itemDescriptionView(ctx, header)
}

func (uc *ConfiguratorUseCase) UpdateItemDescriptionLines(ctx context.Context, id int64, dto request.UpdateCfgItemDescriptionLinesDTO) (*response.CfgItemDescriptionResponse, error) {
	header, err := uc.Q.GetCfgItemDescription(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("descrição do item não encontrada: %w", err)
	}
	for _, l := range dto.Lines {
		descType := l.DescType
		if descType != descTypeCompMascara {
			descType = "DESCRICAO"
		}
		if err := uc.Q.UpdateCfgItemDescriptionLine(ctx, l.ID, int32(l.OrderIndex),
			l.ShowCharacteristic, l.ShowMask, descType, l.Text, l.LineBreak); err != nil {
			return nil, fmt.Errorf("atualizando linha %d: %w", l.ID, err)
		}
	}
	return uc.itemDescriptionView(ctx, header)
}

func (uc *ConfiguratorUseCase) DeleteItemDescription(ctx context.Context, id int64) error {
	return uc.Q.DeleteCfgItemDescription(ctx, id)
}

// RenderItemDescription walks the grid (Botão V) and produces the formatted mask
// description for a set of answers.
func (uc *ConfiguratorUseCase) RenderItemDescription(ctx context.Context, id int64, dto request.CfgRenderDescriptionDTO) (*response.CfgRenderedDescriptionResponse, error) {
	if _, err := uc.Q.GetCfgItemDescription(ctx, id); err != nil {
		return nil, fmt.Errorf("descrição do item não encontrada: %w", err)
	}
	lines, err := uc.Q.ListCfgItemDescriptionLines(ctx, id)
	if err != nil {
		return nil, err
	}
	answers := uc.resolveAnswerValues(ctx, dto.Answers)

	var b strings.Builder
	var segments []string
	for _, l := range lines {
		var parts []string
		if l.ShowMask {
			label := l.CharDescription
			if l.DescType == descTypeCompMascara {
				label = pgutil.FromPgText(l.CharMask)
			}
			if label != "" {
				parts = append(parts, label)
			}
		}
		if l.Text != "" {
			parts = append(parts, l.Text)
		}
		if l.ShowCharacteristic {
			if v, ok := answers[l.CharacteristicID]; ok && v != "" {
				parts = append(parts, v)
			}
		}
		seg := strings.Join(parts, "")
		if seg == "" {
			continue
		}
		segments = append(segments, seg)
		b.WriteString(seg)
		if l.LineBreak {
			b.WriteString("\n")
		} else {
			b.WriteString(" ")
		}
	}
	return &response.CfgRenderedDescriptionResponse{
		ItemDescriptionID: id,
		Text:              strings.TrimRight(b.String(), " \n"),
		Segments:          segments,
	}, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func (uc *ConfiguratorUseCase) loadLines(ctx context.Context, headerID, itemCode int64) error {
	itemChars, err := uc.Q.ListCfgItemCharacteristics(ctx, itemCode)
	if err != nil {
		return fmt.Errorf("carregando características do item: %w", err)
	}
	for _, ic := range itemChars {
		if _, err := uc.Q.InsertCfgItemDescriptionLine(ctx, headerID, ic.ID, ic.Sequence,
			true, true, "DESCRICAO", "", false); err != nil {
			// ignore no-rows (ON CONFLICT DO NOTHING) — a duplicate line is fine
			continue
		}
	}
	return nil
}

func (uc *ConfiguratorUseCase) resolveAnswerValues(ctx context.Context, answers []request.CfgMaskAnswerInput) map[int64]string {
	out := map[int64]string{}
	for _, a := range answers {
		if a.VariableID != nil {
			if v, err := uc.Q.GetCfgVariable(ctx, *a.VariableID); err == nil {
				out[a.CharacteristicID] = v.MaskComposition
				continue
			}
		}
		out[a.CharacteristicID] = a.Value
	}
	return out
}

func (uc *ConfiguratorUseCase) itemDescriptionView(ctx context.Context, h sqlc.DBCfgItemDescription) (*response.CfgItemDescriptionResponse, error) {
	lines, err := uc.Q.ListCfgItemDescriptionLines(ctx, h.ID)
	if err != nil {
		return nil, err
	}
	out := &response.CfgItemDescriptionResponse{
		ID: h.ID, ItemCode: h.ItemCode, DescriptionTypeID: h.DescriptionTypeID,
	}
	for _, l := range lines {
		out.Lines = append(out.Lines, response.CfgItemDescriptionLineResponse{
			ID:                   l.ID,
			ItemCharacteristicID: l.ItemCharacteristicID,
			CharacteristicID:     l.CharacteristicID,
			CharacteristicCode:   l.CharCode,
			CharacteristicName:   l.CharDescription,
			CharacteristicMask:   pgutil.FromPgText(l.CharMask),
			Sequence:             int(l.Sequence),
			OrderIndex:           int(l.OrderIndex),
			ShowCharacteristic:   l.ShowCharacteristic,
			ShowMask:             l.ShowMask,
			DescType:             l.DescType,
			Text:                 l.Text,
			LineBreak:            l.LineBreak,
		})
	}
	return out, nil
}

func descTypeToResponse(t sqlc.DBCfgDescriptionType) *response.CfgDescriptionTypeResponse {
	return &response.CfgDescriptionTypeResponse{
		ID: t.ID, Code: t.Code, Description: t.Description, Kind: t.Kind, IsActive: t.IsActive,
	}
}
