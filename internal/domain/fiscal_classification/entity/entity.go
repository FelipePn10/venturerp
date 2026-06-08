package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type RateIndicator string

const (
	IndicatorPercentual RateIndicator = "PERCENTUAL"
	IndicatorValor      RateIndicator = "VALOR"
)

// FiscalClassification is the "Cadastro de Classificações Fiscais": NCM/CEST,
// CSTs and rates for IPI/PIS/COFINS (including consumo/retenção/redução/ZF),
// ICMS BC modalities and CBS/IBS classification codes.
type FiscalClassification struct {
	ID          int64
	Code        int64
	Description string
	NCM         *string
	CEST        *string
	// IPI
	IPIRate       float64
	IPIIndicator  RateIndicator
	Apuracao      *string
	CSTIPIEntrada *string
	CSTIPISaida   *string
	// PIS
	PISRate       float64
	PISIndicator  RateIndicator
	CSTPISEntrada *string
	CSTPISSaida   *string
	// COFINS
	COFINSRate        float64
	COFINSIndicator   RateIndicator
	CSTCOFINSEntrada  *string
	CSTCOFINSSaida    *string
	COFINSMajoradoPct float64
	// Substituição tributária
	PISSTPct    float64
	COFINSSTPct float64
	// Consumo
	PISConsumoPct           float64
	CSTPISConsumoEntrada    *string
	CSTPISConsumoSaida      *string
	COFINSConsumoPct        float64
	CSTCOFINSConsumoEntrada *string
	CSTCOFINSConsumoSaida   *string
	// Retenção
	PISRetencaoPct    float64
	CSTPISRetencao    *string
	COFINSRetencaoPct float64
	CSTCOFINSRetencao *string
	// Redução
	PISReducaoPct    float64
	CSTPISReducao    *string
	COFINSReducaoPct float64
	CSTCOFINSReducao *string
	// Zona Franca
	DescPISZFPct    float64
	DescCOFINSZFPct float64
	// Outros
	ExTarifario        *string
	UNIPI              *string
	UNTributacao       *string
	ModBCICMS          *string
	ModBCICMSST        *string
	CodClasTrib        *string
	CodClasTribTribReg *string
	ObsFiscal          *string
	IsActive           bool
	CreatedAt          time.Time
	CreatedBy          uuid.UUID
	UpdatedAt          time.Time

	Languages        []*FiscalClassificationLanguage
	ExportAttributes []*FiscalClassificationExportAttribute
}

func NewFiscalClassification(code int64, description string, createdBy uuid.UUID) (*FiscalClassification, error) {
	if code == 0 {
		return nil, fmt.Errorf("code is required")
	}
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	now := time.Now()
	return &FiscalClassification{
		Code:            code,
		Description:     description,
		IPIIndicator:    IndicatorPercentual,
		PISIndicator:    IndicatorPercentual,
		COFINSIndicator: IndicatorPercentual,
		IsActive:        true,
		CreatedAt:       now,
		CreatedBy:       createdBy,
		UpdatedAt:       now,
	}, nil
}

type FiscalClassificationLanguage struct {
	ID               int64
	ClassificationID int64
	Language         string
	Description      string
}

type FiscalClassificationExportAttribute struct {
	ID               int64
	ClassificationID int64
	Code             string
	Description      *string
	Domain           *string
	StartDate        *time.Time
	EndDate          *time.Time
}
