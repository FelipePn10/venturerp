package response

import (
	"time"

	"github.com/google/uuid"
)

// FiscalClassificationResponse is the API representation of a fiscal classification.
type FiscalClassificationResponse struct {
	ID          int64   `json:"id"`
	Code        int64   `json:"code"`
	Description string  `json:"description"`
	NCM         *string `json:"ncm,omitempty"`
	CEST        *string `json:"cest,omitempty"`

	IPIRate       float64 `json:"ipi_rate"`
	IPIIndicator  string  `json:"ipi_indicator"`
	Apuracao      *string `json:"apuracao,omitempty"`
	CSTIPIEntrada *string `json:"cst_ipi_entrada,omitempty"`
	CSTIPISaida   *string `json:"cst_ipi_saida,omitempty"`

	PISRate       float64 `json:"pis_rate"`
	PISIndicator  string  `json:"pis_indicator"`
	CSTPISEntrada *string `json:"cst_pis_entrada,omitempty"`
	CSTPISSaida   *string `json:"cst_pis_saida,omitempty"`

	COFINSRate        float64 `json:"cofins_rate"`
	COFINSIndicator   string  `json:"cofins_indicator"`
	CSTCOFINSEntrada  *string `json:"cst_cofins_entrada,omitempty"`
	CSTCOFINSSaida    *string `json:"cst_cofins_saida,omitempty"`
	COFINSMajoradoPct float64 `json:"cofins_majorado_pct"`

	PISSTPct    float64 `json:"pis_st_pct"`
	COFINSSTPct float64 `json:"cofins_st_pct"`

	PISConsumoPct           float64 `json:"pis_consumo_pct"`
	CSTPISConsumoEntrada    *string `json:"cst_pis_consumo_entrada,omitempty"`
	CSTPISConsumoSaida      *string `json:"cst_pis_consumo_saida,omitempty"`
	COFINSConsumoPct        float64 `json:"cofins_consumo_pct"`
	CSTCOFINSConsumoEntrada *string `json:"cst_cofins_consumo_entrada,omitempty"`
	CSTCOFINSConsumoSaida   *string `json:"cst_cofins_consumo_saida,omitempty"`

	PISRetencaoPct    float64 `json:"pis_retencao_pct"`
	CSTPISRetencao    *string `json:"cst_pis_retencao,omitempty"`
	COFINSRetencaoPct float64 `json:"cofins_retencao_pct"`
	CSTCOFINSRetencao *string `json:"cst_cofins_retencao,omitempty"`

	PISReducaoPct    float64 `json:"pis_reducao_pct"`
	CSTPISReducao    *string `json:"cst_pis_reducao,omitempty"`
	COFINSReducaoPct float64 `json:"cofins_reducao_pct"`
	CSTCOFINSReducao *string `json:"cst_cofins_reducao,omitempty"`

	DescPISZFPct    float64 `json:"desc_pis_zf_pct"`
	DescCOFINSZFPct float64 `json:"desc_cofins_zf_pct"`

	ExTarifario        *string `json:"ex_tarifario,omitempty"`
	UNIPI              *string `json:"un_ipi,omitempty"`
	UNTributacao       *string `json:"un_tributacao,omitempty"`
	ModBCICMS          *string `json:"mod_bc_icms,omitempty"`
	ModBCICMSST        *string `json:"mod_bc_icms_st,omitempty"`
	CodClasTrib        *string `json:"cod_clas_trib,omitempty"`
	CodClasTribTribReg *string `json:"cod_clas_trib_trib_reg,omitempty"`
	ObsFiscal          *string `json:"obs_fiscal,omitempty"`

	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy uuid.UUID `json:"created_by"`
	UpdatedAt time.Time `json:"updated_at"`

	Languages        []FiscalClassificationLanguageResponse        `json:"languages,omitempty"`
	ExportAttributes []FiscalClassificationExportAttributeResponse `json:"export_attributes,omitempty"`
}

// FiscalClassificationLanguageResponse is a translated description of a classification.
type FiscalClassificationLanguageResponse struct {
	ID               int64  `json:"id"`
	ClassificationID int64  `json:"classification_id"`
	Language         string `json:"language"`
	Description      string `json:"description"`
}

// FiscalClassificationExportAttributeResponse is an export attribute of a classification.
type FiscalClassificationExportAttributeResponse struct {
	ID               int64      `json:"id"`
	ClassificationID int64      `json:"classification_id"`
	Code             string     `json:"code"`
	Description      *string    `json:"description,omitempty"`
	Domain           *string    `json:"domain,omitempty"`
	StartDate        *time.Time `json:"start_date,omitempty"`
	EndDate          *time.Time `json:"end_date,omitempty"`
}
