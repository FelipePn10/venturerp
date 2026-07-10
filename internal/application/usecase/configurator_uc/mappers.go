package configurator_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

// textOrNull maps an empty string to a NULL pgtype.Text (so optional text
// columns stay NULL instead of storing empty strings).
func textOrNull(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

// numPtr converts a nullable pg numeric to *float64 (nil when NULL).
func numPtr(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	v := pgutil.FromPgNumericToFloat64(n)
	return &v
}

func setToResponse(s sqlc.DBCfgSet) *response.CfgSetResponse {
	return &response.CfgSetResponse{
		ID:          s.ID,
		Description: s.Description,
		IsActive:    s.IsActive,
		VariableQty: s.VariableQty,
		CreatedAt:   pgutil.FromPgTimestamptz(s.CreatedAt),
	}
}

func variableToResponse(v sqlc.DBCfgVariable, langs []sqlc.DBCfgVariableLanguage) *response.CfgVariableResponse {
	r := &response.CfgVariableResponse{
		ID:                 v.ID,
		SetID:              v.SetID,
		Code:               v.Code,
		Description:        v.Description,
		MaskComposition:    v.MaskComposition,
		IsActive:           v.IsActive,
		IsSpecial:          v.IsSpecial,
		IncludeDescription: v.IncludeDescription,
		SpecialData:        pgutil.FromPgText(v.SpecialData),
		Marketing:          v.Marketing,
	}
	for _, l := range langs {
		r.Languages = append(r.Languages, response.CfgVariableLanguageResponse{
			ID: l.ID, VariableID: l.VariableID, Language: l.Language,
			Country: pgutil.FromPgText(l.Country), Translation: l.Translation,
		})
	}
	return r
}

func characteristicToResponse(c sqlc.DBCfgCharacteristic, langs []sqlc.DBCfgCharacteristicLanguage) *response.CfgCharacteristicResponse {
	r := &response.CfgCharacteristicResponse{
		ID:                  c.ID,
		Code:                c.Code,
		Description:         c.Description,
		Type:                c.CharType,
		IsActive:            c.IsActive,
		SetID:               pgutil.FromPgInt8Ptr(c.SetID),
		SetDescription:      pgutil.FromPgText(c.SetDescription),
		DefaultVariableID:   pgutil.FromPgInt8Ptr(c.DefaultVariableID),
		DefaultVariableCode: pgutil.FromPgText(c.DefaultVariableStr),
		Mask:                pgutil.FromPgText(c.Mask),
		IsSpecial:           c.IsSpecial,
		AffectsPrice:        c.AffectsPrice,
		ControlsGoals:       c.ControlsGoals,
		ReceivingType:       c.ReceivingType,
		FieldSource:         pgutil.FromPgText(c.FieldSource),
		Formula:             pgutil.FromPgText(c.Formula),
		IsRequired:          c.IsRequired,
		NumMin:              numPtr(c.NumMin),
		NumMax:              numPtr(c.NumMax),
		NumMultiple:         numPtr(c.NumMultiple),
		OptionTrue:          pgutil.FromPgText(c.OptionTrue),
		OptionFalse:         pgutil.FromPgText(c.OptionFalse),
		CreatedAt:           pgutil.FromPgTimestamptz(c.CreatedAt),
	}
	for _, l := range langs {
		r.Languages = append(r.Languages, response.CfgCharacteristicLanguageResponse{
			ID: l.ID, CharacteristicID: l.CharacteristicID, Language: l.Language,
			Description: l.Description, Mask: pgutil.FromPgText(l.Mask),
		})
	}
	return r
}

func itemCharToResponse(ic sqlc.DBCfgItemCharacteristic, defaults []int64) *response.CfgItemCharacteristicResponse {
	return &response.CfgItemCharacteristicResponse{
		ID:                 ic.ID,
		ItemCode:           ic.ItemCode,
		CharacteristicID:   ic.CharacteristicID,
		CharacteristicCode: ic.CharCode,
		CharacteristicName: ic.CharName,
		CharacteristicType: ic.CharType,
		CharacteristicMask: pgutil.FromPgText(ic.CharMask),
		Sequence:           int(ic.Sequence),
		DefaultVariableID:  pgutil.FromPgInt8Ptr(ic.DefaultVariableID),
		ParentID:           pgutil.FromPgInt8Ptr(ic.ParentID),
		IsSpecial:          ic.IsSpecial,
		IsDrawing:          ic.IsDrawing,
		IsLoad:             ic.IsLoad,
		Formula:            pgutil.FromPgText(ic.Formula),
		DefaultAnswers:     defaults,
	}
}
