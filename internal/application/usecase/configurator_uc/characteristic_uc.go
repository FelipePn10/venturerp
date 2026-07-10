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

func (uc *ConfiguratorUseCase) CreateCharacteristic(ctx context.Context, dto request.CreateCfgCharacteristicDTO) (*response.CfgCharacteristicResponse, error) {
	c, err := entity.NewCharacteristic(dto.Code, dto.Description, dto.Type, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	c.SetID = dto.SetID
	c.DefaultVariableID = dto.DefaultVariableID
	c.Mask = dto.Mask
	c.IsSpecial = dto.IsSpecial
	c.AffectsPrice = dto.AffectsPrice
	c.ControlsGoals = dto.ControlsGoals
	if dto.ReceivingType != "" {
		c.ReceivingType = dto.ReceivingType
	}
	c.FieldSource = dto.FieldSource
	c.Formula = dto.Formula
	c.IsRequired = dto.IsRequired
	c.NumMin, c.NumMax, c.NumMultiple = dto.NumMin, dto.NumMax, dto.NumMultiple
	c.OptionTrue, c.OptionFalse = dto.OptionTrue, dto.OptionFalse
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if err := uc.checkCharRefs(ctx, c); err != nil {
		return nil, err
	}
	row, err := uc.Q.CreateCfgCharacteristic(ctx, charParams(c, 0))
	if err != nil {
		return nil, fmt.Errorf("criando característica: %w", err)
	}
	return characteristicToResponse(row, nil), nil
}

func (uc *ConfiguratorUseCase) UpdateCharacteristic(ctx context.Context, dto request.UpdateCfgCharacteristicDTO) (*response.CfgCharacteristicResponse, error) {
	c := &entity.Characteristic{
		Code: dto.Code, Description: dto.Description, Type: dto.Type, IsActive: dto.IsActive,
		SetID: dto.SetID, DefaultVariableID: dto.DefaultVariableID, Mask: dto.Mask,
		IsSpecial: dto.IsSpecial, AffectsPrice: dto.AffectsPrice, ControlsGoals: dto.ControlsGoals,
		ReceivingType: dto.ReceivingType, FieldSource: dto.FieldSource, Formula: dto.Formula,
		IsRequired: dto.IsRequired, NumMin: dto.NumMin, NumMax: dto.NumMax, NumMultiple: dto.NumMultiple,
		OptionTrue: dto.OptionTrue, OptionFalse: dto.OptionFalse,
	}
	if c.Code == "" || c.Description == "" {
		return nil, fmt.Errorf("código e descrição da característica são obrigatórios")
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if err := uc.checkCharRefs(ctx, c); err != nil {
		return nil, err
	}
	row, err := uc.Q.UpdateCfgCharacteristic(ctx, charParams(c, dto.ID))
	if err != nil {
		return nil, fmt.Errorf("atualizando característica: %w", err)
	}
	langs, _ := uc.Q.ListCfgCharacteristicLanguages(ctx, dto.ID)
	return characteristicToResponse(row, langs), nil
}

func (uc *ConfiguratorUseCase) GetCharacteristic(ctx context.Context, id int64) (*response.CfgCharacteristicResponse, error) {
	row, err := uc.Q.GetCfgCharacteristic(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("característica não encontrada: %w", err)
	}
	langs, _ := uc.Q.ListCfgCharacteristicLanguages(ctx, id)
	return characteristicToResponse(row, langs), nil
}

func (uc *ConfiguratorUseCase) ListCharacteristics(ctx context.Context, onlyActive bool) ([]*response.CfgCharacteristicResponse, error) {
	rows, err := uc.Q.ListCfgCharacteristics(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CfgCharacteristicResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, characteristicToResponse(r, nil))
	}
	return out, nil
}

func (uc *ConfiguratorUseCase) DeactivateCharacteristic(ctx context.Context, id int64) error {
	return uc.Q.DeactivateCfgCharacteristic(ctx, id)
}

func (uc *ConfiguratorUseCase) SetCharacteristicLanguage(ctx context.Context, charID int64, dto request.CfgCharacteristicLanguageDTO) (*response.CfgCharacteristicLanguageResponse, error) {
	if dto.Language == "" || dto.Description == "" {
		return nil, fmt.Errorf("idioma e descrição são obrigatórios")
	}
	row, err := uc.Q.UpsertCfgCharacteristicLanguage(ctx, charID, dto.Language, dto.Description, textOrNull(dto.Mask))
	if err != nil {
		return nil, fmt.Errorf("gravando idioma da característica: %w", err)
	}
	return &response.CfgCharacteristicLanguageResponse{
		ID: row.ID, CharacteristicID: row.CharacteristicID, Language: row.Language,
		Description: row.Description, Mask: pgutil.FromPgText(row.Mask),
	}, nil
}

func (uc *ConfiguratorUseCase) DeleteCharacteristicLanguage(ctx context.Context, id int64) error {
	return uc.Q.DeleteCfgCharacteristicLanguage(ctx, id)
}

// checkCharRefs validates that referenced set/default-variable exist and are
// consistent (default variable must belong to the characteristic's set).
func (uc *ConfiguratorUseCase) checkCharRefs(ctx context.Context, c *entity.Characteristic) error {
	if c.SetID != nil {
		if _, err := uc.Q.GetCfgSet(ctx, *c.SetID); err != nil {
			return fmt.Errorf("conjunto %d não encontrado", *c.SetID)
		}
	}
	if c.DefaultVariableID != nil {
		v, err := uc.Q.GetCfgVariable(ctx, *c.DefaultVariableID)
		if err != nil {
			return fmt.Errorf("variável default %d não encontrada", *c.DefaultVariableID)
		}
		if c.SetID != nil && v.SetID != *c.SetID {
			return fmt.Errorf("a variável default não pertence ao conjunto da característica")
		}
	}
	return nil
}

func charParams(c *entity.Characteristic, id int64) sqlc.CfgCharacteristicParams {
	return sqlc.CfgCharacteristicParams{
		ID:                id,
		Code:              c.Code,
		Description:       c.Description,
		CharType:          c.Type,
		IsActive:          c.IsActive,
		SetID:             pgutil.ToPgInt8Ptr(c.SetID),
		DefaultVariableID: pgutil.ToPgInt8Ptr(c.DefaultVariableID),
		Mask:              textOrNull(c.Mask),
		IsSpecial:         c.IsSpecial,
		AffectsPrice:      c.AffectsPrice,
		ControlsGoals:     c.ControlsGoals,
		ReceivingType:     c.ReceivingType,
		FieldSource:       textOrNull(c.FieldSource),
		Formula:           textOrNull(c.Formula),
		IsRequired:        c.IsRequired,
		NumMin:            pgutil.ToPgNumericFromFloat64Ptr(c.NumMin),
		NumMax:            pgutil.ToPgNumericFromFloat64Ptr(c.NumMax),
		NumMultiple:       pgutil.ToPgNumericFromFloat64Ptr(c.NumMultiple),
		OptionTrue:        textOrNull(c.OptionTrue),
		OptionFalse:       textOrNull(c.OptionFalse),
		CreatedBy:         pgutil.ToPgUUID(c.CreatedBy),
	}
}
