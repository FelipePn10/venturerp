package request

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// FiscalClassificationFields holds the editable fields shared by create/update.
type FiscalClassificationFields struct {
	ItemCode                *int64          `json:"item_code,omitempty"`
	Description             string          `json:"description"`
	NCM                     *string         `json:"ncm,omitempty"`
	CEST                    *string         `json:"cest,omitempty"`
	IPIRate                 decimal.Decimal `json:"ipi_rate"`
	IPIIndicator            string          `json:"ipi_indicator,omitempty"`
	Apuracao                *string         `json:"apuracao,omitempty"`
	CSTIPIEntrada           *string         `json:"cst_ipi_entrada,omitempty"`
	CSTIPISaida             *string         `json:"cst_ipi_saida,omitempty"`
	PISRate                 decimal.Decimal `json:"pis_rate"`
	PISIndicator            string          `json:"pis_indicator,omitempty"`
	CSTPISEntrada           *string         `json:"cst_pis_entrada,omitempty"`
	CSTPISSaida             *string         `json:"cst_pis_saida,omitempty"`
	COFINSRate              decimal.Decimal `json:"cofins_rate"`
	COFINSIndicator         string          `json:"cofins_indicator,omitempty"`
	CSTCOFINSEntrada        *string         `json:"cst_cofins_entrada,omitempty"`
	CSTCOFINSSaida          *string         `json:"cst_cofins_saida,omitempty"`
	COFINSMajoradoPct       decimal.Decimal `json:"cofins_majorado_pct"`
	PISSTPct                decimal.Decimal `json:"pis_st_pct"`
	COFINSSTPct             decimal.Decimal `json:"cofins_st_pct"`
	PISConsumoPct           decimal.Decimal `json:"pis_consumo_pct"`
	CSTPISConsumoEntrada    *string         `json:"cst_pis_consumo_entrada,omitempty"`
	CSTPISConsumoSaida      *string         `json:"cst_pis_consumo_saida,omitempty"`
	COFINSConsumoPct        decimal.Decimal `json:"cofins_consumo_pct"`
	CSTCOFINSConsumoEntrada *string         `json:"cst_cofins_consumo_entrada,omitempty"`
	CSTCOFINSConsumoSaida   *string         `json:"cst_cofins_consumo_saida,omitempty"`
	PISRetencaoPct          decimal.Decimal `json:"pis_retencao_pct"`
	CSTPISRetencao          *string         `json:"cst_pis_retencao,omitempty"`
	COFINSRetencaoPct       decimal.Decimal `json:"cofins_retencao_pct"`
	CSTCOFINSRetencao       *string         `json:"cst_cofins_retencao,omitempty"`
	PISReducaoPct           decimal.Decimal `json:"pis_reducao_pct"`
	CSTPISReducao           *string         `json:"cst_pis_reducao,omitempty"`
	COFINSReducaoPct        decimal.Decimal `json:"cofins_reducao_pct"`
	CSTCOFINSReducao        *string         `json:"cst_cofins_reducao,omitempty"`
	DescPISZFPct            decimal.Decimal `json:"desc_pis_zf_pct"`
	DescCOFINSZFPct         decimal.Decimal `json:"desc_cofins_zf_pct"`
	ExTarifario             *string         `json:"ex_tarifario,omitempty"`
	UNIPI                   *string         `json:"un_ipi,omitempty"`
	UNTributacao            *string         `json:"un_tributacao,omitempty"`
	ModBCICMS               *string         `json:"mod_bc_icms,omitempty"`
	ModBCICMSST             *string         `json:"mod_bc_icms_st,omitempty"`
	CodClasTrib             *string         `json:"cod_clas_trib,omitempty"`
	CodClasTribTribReg      *string         `json:"cod_clas_trib_trib_reg,omitempty"`
	ObsFiscal               *string         `json:"obs_fiscal,omitempty"`
	// Aceitam "2026-09-20" (o que o <input type="date"> envia) e RFC3339. Com
	// *time.Time puro o usuário recebia o erro cru do parser do Go.
	ValidFrom                 *DataFlexivel   `json:"valid_from,omitempty"`
	ValidUntil                *DataFlexivel   `json:"valid_until,omitempty"`
	DefaultOrigin             *string         `json:"default_origin,omitempty"`
	DefaultICMSRate           decimal.Decimal `json:"default_icms_rate"`
	DefaultCalculatePISCOFINS bool            `json:"default_calculate_pis_cofins"`
}

type CreateFiscalClassificationDTO struct {
	FiscalClassificationFields
	CreatedBy uuid.UUID `json:"created_by"`
}

type UpdateFiscalClassificationDTO struct {
	Code int64 `json:"code"`
	FiscalClassificationFields
	IsActive *bool `json:"is_active,omitempty"`
}

type AddFiscalClassificationLanguageDTO struct {
	ClassificationCode int64  `json:"classification_code"`
	Language           string `json:"language"`
	Description        string `json:"description"`
}

type AddFiscalClassificationExportAttributeDTO struct {
	ClassificationCode int64   `json:"classification_code"`
	Code               string  `json:"code"`
	Description        *string `json:"description,omitempty"`
	Domain             *string `json:"domain,omitempty"`
	StartDate          *string `json:"start_date,omitempty"`
	EndDate            *string `json:"end_date,omitempty"`
}
