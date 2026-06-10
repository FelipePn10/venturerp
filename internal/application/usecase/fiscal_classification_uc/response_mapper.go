package fiscal_classification_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/entity"
)

func toFiscalClassificationResponse(c *entity.FiscalClassification) *response.FiscalClassificationResponse {
	if c == nil {
		return nil
	}
	return &response.FiscalClassificationResponse{
		ID:                      c.ID,
		Code:                    c.Code,
		Description:             c.Description,
		NCM:                     c.NCM,
		CEST:                    c.CEST,
		IPIRate:                 c.IPIRate,
		IPIIndicator:            string(c.IPIIndicator),
		Apuracao:                c.Apuracao,
		CSTIPIEntrada:           c.CSTIPIEntrada,
		CSTIPISaida:             c.CSTIPISaida,
		PISRate:                 c.PISRate,
		PISIndicator:            string(c.PISIndicator),
		CSTPISEntrada:           c.CSTPISEntrada,
		CSTPISSaida:             c.CSTPISSaida,
		COFINSRate:              c.COFINSRate,
		COFINSIndicator:         string(c.COFINSIndicator),
		CSTCOFINSEntrada:        c.CSTCOFINSEntrada,
		CSTCOFINSSaida:          c.CSTCOFINSSaida,
		COFINSMajoradoPct:       c.COFINSMajoradoPct,
		PISSTPct:                c.PISSTPct,
		COFINSSTPct:             c.COFINSSTPct,
		PISConsumoPct:           c.PISConsumoPct,
		CSTPISConsumoEntrada:    c.CSTPISConsumoEntrada,
		CSTPISConsumoSaida:      c.CSTPISConsumoSaida,
		COFINSConsumoPct:        c.COFINSConsumoPct,
		CSTCOFINSConsumoEntrada: c.CSTCOFINSConsumoEntrada,
		CSTCOFINSConsumoSaida:   c.CSTCOFINSConsumoSaida,
		PISRetencaoPct:          c.PISRetencaoPct,
		CSTPISRetencao:          c.CSTPISRetencao,
		COFINSRetencaoPct:       c.COFINSRetencaoPct,
		CSTCOFINSRetencao:       c.CSTCOFINSRetencao,
		PISReducaoPct:           c.PISReducaoPct,
		CSTPISReducao:           c.CSTPISReducao,
		COFINSReducaoPct:        c.COFINSReducaoPct,
		CSTCOFINSReducao:        c.CSTCOFINSReducao,
		DescPISZFPct:            c.DescPISZFPct,
		DescCOFINSZFPct:         c.DescCOFINSZFPct,
		ExTarifario:             c.ExTarifario,
		UNIPI:                   c.UNIPI,
		UNTributacao:            c.UNTributacao,
		ModBCICMS:               c.ModBCICMS,
		ModBCICMSST:             c.ModBCICMSST,
		CodClasTrib:             c.CodClasTrib,
		CodClasTribTribReg:      c.CodClasTribTribReg,
		ObsFiscal:               c.ObsFiscal,
		IsActive:                c.IsActive,
		CreatedAt:               c.CreatedAt,
		CreatedBy:               c.CreatedBy,
		UpdatedAt:               c.UpdatedAt,
		Languages:               toFiscalClassificationLanguageValues(c.Languages),
		ExportAttributes:        toFiscalClassificationExportAttributeValues(c.ExportAttributes),
	}
}

func toFiscalClassificationResponses(list []*entity.FiscalClassification) []*response.FiscalClassificationResponse {
	out := make([]*response.FiscalClassificationResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toFiscalClassificationResponse(c))
	}
	return out
}

func toFiscalClassificationLanguageResponse(l *entity.FiscalClassificationLanguage) *response.FiscalClassificationLanguageResponse {
	if l == nil {
		return nil
	}
	return &response.FiscalClassificationLanguageResponse{
		ID:               l.ID,
		ClassificationID: l.ClassificationID,
		Language:         l.Language,
		Description:      l.Description,
	}
}

func toFiscalClassificationLanguageValues(list []*entity.FiscalClassificationLanguage) []response.FiscalClassificationLanguageResponse {
	if len(list) == 0 {
		return nil
	}
	out := make([]response.FiscalClassificationLanguageResponse, 0, len(list))
	for _, l := range list {
		out = append(out, *toFiscalClassificationLanguageResponse(l))
	}
	return out
}

func toFiscalClassificationExportAttributeResponse(a *entity.FiscalClassificationExportAttribute) *response.FiscalClassificationExportAttributeResponse {
	if a == nil {
		return nil
	}
	return &response.FiscalClassificationExportAttributeResponse{
		ID:               a.ID,
		ClassificationID: a.ClassificationID,
		Code:             a.Code,
		Description:      a.Description,
		Domain:           a.Domain,
		StartDate:        a.StartDate,
		EndDate:          a.EndDate,
	}
}

func toFiscalClassificationExportAttributeValues(list []*entity.FiscalClassificationExportAttribute) []response.FiscalClassificationExportAttributeResponse {
	if len(list) == 0 {
		return nil
	}
	out := make([]response.FiscalClassificationExportAttributeResponse, 0, len(list))
	for _, a := range list {
		out = append(out, *toFiscalClassificationExportAttributeResponse(a))
	}
	return out
}
