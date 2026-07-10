package configurator_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/configurator/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (uc *ConfiguratorUseCase) AddItemCharacteristic(ctx context.Context, dto request.AddCfgItemCharacteristicDTO) (*response.CfgItemCharacteristicResponse, error) {
	ic, err := entity.NewItemCharacteristic(dto.ItemCode, dto.CharacteristicID, dto.Sequence)
	if err != nil {
		return nil, err
	}
	char, err := uc.Q.GetCfgCharacteristic(ctx, dto.CharacteristicID)
	if err != nil {
		return nil, fmt.Errorf("característica %d não encontrada", dto.CharacteristicID)
	}
	// A fórmula é obrigatória quando a característica é do tipo FORMULA (o cálculo
	// depende de informações do item, logo é definido aqui).
	if char.CharType == entity.TypeFormula && dto.Formula == "" {
		return nil, fmt.Errorf("característica do tipo fórmula exige o preenchimento da fórmula")
	}
	if err := uc.validateParent(ctx, dto.ParentID, dto.ItemCode); err != nil {
		return nil, err
	}
	row, err := uc.Q.AddCfgItemCharacteristic(ctx, sqlc.CfgItemCharacteristicParams{
		ItemCode:          ic.ItemCode,
		CharacteristicID:  ic.CharacteristicID,
		Sequence:          int32(ic.Sequence),
		DefaultVariableID: pgutil.ToPgInt8Ptr(dto.DefaultVariableID),
		ParentID:          pgutil.ToPgInt8Ptr(dto.ParentID),
		IsSpecial:         dto.IsSpecial,
		IsDrawing:         dto.IsDrawing,
		IsLoad:            dto.IsLoad,
		Formula:           textOrNull(dto.Formula),
	})
	if err != nil {
		return nil, fmt.Errorf("vinculando característica ao item: %w", err)
	}
	if err := uc.Q.ReplaceCfgItemCharDefaultAnswers(ctx, row.ID, dto.DefaultAnswers); err != nil {
		return nil, fmt.Errorf("gravando respostas default: %w", err)
	}
	return itemCharToResponse(row, dto.DefaultAnswers), nil
}

func (uc *ConfiguratorUseCase) UpdateItemCharacteristic(ctx context.Context, dto request.UpdateCfgItemCharacteristicDTO) (*response.CfgItemCharacteristicResponse, error) {
	existing, err := uc.Q.GetCfgItemCharacteristic(ctx, dto.ID)
	if err != nil {
		return nil, fmt.Errorf("característica do item não encontrada: %w", err)
	}
	if dto.Sequence <= 0 {
		return nil, fmt.Errorf("sequência deve ser positiva")
	}
	// Guard: a sequência só pode mudar enquanto o item não tiver máscara gerada
	// nem fórmula no cadastro de estruturas.
	if int32(dto.Sequence) != existing.Sequence {
		if locked, reason := uc.itemLocked(ctx, existing.ItemCode); locked {
			return nil, fmt.Errorf("não é permitido alterar a sequência: %s", reason)
		}
	}
	if err := uc.validateParent(ctx, dto.ParentID, existing.ItemCode); err != nil {
		return nil, err
	}
	row, err := uc.Q.UpdateCfgItemCharacteristic(ctx, sqlc.CfgItemCharacteristicParams{
		ID:                dto.ID,
		Sequence:          int32(dto.Sequence),
		DefaultVariableID: pgutil.ToPgInt8Ptr(dto.DefaultVariableID),
		ParentID:          pgutil.ToPgInt8Ptr(dto.ParentID),
		IsSpecial:         dto.IsSpecial,
		IsDrawing:         dto.IsDrawing,
		IsLoad:            dto.IsLoad,
		Formula:           textOrNull(dto.Formula),
	})
	if err != nil {
		return nil, fmt.Errorf("atualizando característica do item: %w", err)
	}
	if err := uc.Q.ReplaceCfgItemCharDefaultAnswers(ctx, dto.ID, dto.DefaultAnswers); err != nil {
		return nil, fmt.Errorf("gravando respostas default: %w", err)
	}
	return itemCharToResponse(row, dto.DefaultAnswers), nil
}

func (uc *ConfiguratorUseCase) ListItemCharacteristics(ctx context.Context, itemCode int64) ([]*response.CfgItemCharacteristicResponse, error) {
	rows, err := uc.Q.ListCfgItemCharacteristics(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CfgItemCharacteristicResponse, 0, len(rows))
	for _, r := range rows {
		defaults, _ := uc.Q.ListCfgItemCharDefaultAnswers(ctx, r.ID)
		out = append(out, itemCharToResponse(r, defaults))
	}
	return out, nil
}

func (uc *ConfiguratorUseCase) RemoveItemCharacteristic(ctx context.Context, id int64) error {
	existing, err := uc.Q.GetCfgItemCharacteristic(ctx, id)
	if err != nil {
		return fmt.Errorf("característica do item não encontrada: %w", err)
	}
	if locked, reason := uc.itemLocked(ctx, existing.ItemCode); locked {
		return fmt.Errorf("não é permitido excluir: %s", reason)
	}
	return uc.Q.RemoveCfgItemCharacteristic(ctx, id)
}

// itemLocked reports whether the item's characteristics/sequences are frozen: it
// already has a generated mask or a structure loss formula (spec guard).
func (uc *ConfiguratorUseCase) itemLocked(ctx context.Context, itemCode int64) (bool, string) {
	if hasMask, _ := uc.Q.ItemHasGeneratedMask(ctx, itemCode); hasMask {
		return true, "o item já possui máscara gerada"
	}
	if hasFormula, _ := uc.Q.ItemHasStructureFormula(ctx, itemCode); hasFormula {
		return true, "o item possui fórmula no cadastro de estruturas"
	}
	return false, ""
}

// ListItemsByCharacteristic returns the item codes that use a characteristic
// (Botão Itens Vinculados).
func (uc *ConfiguratorUseCase) ListItemsByCharacteristic(ctx context.Context, characteristicID int64) ([]int64, error) {
	return uc.Q.ListItemsByCharacteristic(ctx, characteristicID)
}

// validateParent ensures a parent item-characteristic belongs to the same item.
func (uc *ConfiguratorUseCase) validateParent(ctx context.Context, parentID *int64, itemCode int64) error {
	if parentID == nil {
		return nil
	}
	parent, err := uc.Q.GetCfgItemCharacteristic(ctx, *parentID)
	if err != nil {
		return fmt.Errorf("característica pai %d não encontrada", *parentID)
	}
	if parent.ItemCode != itemCode {
		return fmt.Errorf("a característica pai pertence a outro item")
	}
	return nil
}
