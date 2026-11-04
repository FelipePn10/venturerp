package entity

import (
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/shopspring/decimal"

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
	ID           int64
	EnterpriseID int64
	Code         int64
	Description  string
	ItemCode     *int64
	NCM          *string
	CEST         *string
	// IPI
	IPIRate       decimal.Decimal
	IPIIndicator  RateIndicator
	Apuracao      *string
	CSTIPIEntrada *string
	CSTIPISaida   *string
	// PIS
	PISRate       decimal.Decimal
	PISIndicator  RateIndicator
	CSTPISEntrada *string
	CSTPISSaida   *string
	// COFINS
	COFINSRate        decimal.Decimal
	COFINSIndicator   RateIndicator
	CSTCOFINSEntrada  *string
	CSTCOFINSSaida    *string
	COFINSMajoradoPct decimal.Decimal
	// Substituição tributária
	PISSTPct    decimal.Decimal
	COFINSSTPct decimal.Decimal
	// Consumo
	PISConsumoPct           decimal.Decimal
	CSTPISConsumoEntrada    *string
	CSTPISConsumoSaida      *string
	COFINSConsumoPct        decimal.Decimal
	CSTCOFINSConsumoEntrada *string
	CSTCOFINSConsumoSaida   *string
	// Retenção
	PISRetencaoPct    decimal.Decimal
	CSTPISRetencao    *string
	COFINSRetencaoPct decimal.Decimal
	CSTCOFINSRetencao *string
	// Redução
	PISReducaoPct    decimal.Decimal
	CSTPISReducao    *string
	COFINSReducaoPct decimal.Decimal
	CSTCOFINSReducao *string
	// Zona Franca
	DescPISZFPct    decimal.Decimal
	DescCOFINSZFPct decimal.Decimal
	// Outros
	ExTarifario               *string
	UNIPI                     *string
	UNTributacao              *string
	ModBCICMS                 *string
	ModBCICMSST               *string
	CodClasTrib               *string
	CodClasTribTribReg        *string
	ObsFiscal                 *string
	ValidFrom                 *time.Time
	ValidUntil                *time.Time
	DefaultOrigin             *string
	DefaultICMSRate           decimal.Decimal
	DefaultCalculatePISCOFINS bool
	IsActive                  bool
	CreatedAt                 time.Time
	CreatedBy                 uuid.UUID
	UpdatedAt                 time.Time

	Languages        []*FiscalClassificationLanguage
	ExportAttributes []*FiscalClassificationExportAttribute
}

func NewFiscalClassification(code int64, description string, createdBy uuid.UUID) (*FiscalClassification, error) {
	if code == 0 {
		return nil, domainerr.NewValidation("informe o código")
	}
	if description == "" {
		return nil, domainerr.NewValidation("informe a descrição")
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
