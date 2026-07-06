package entity

import (
	"time"

	"github.com/google/uuid"
)

type FiscalExitStatus string

const (
	ExitStatusDraft      FiscalExitStatus = "DRAFT"
	ExitStatusAuthorized FiscalExitStatus = "AUTHORIZED"
	ExitStatusCancelled  FiscalExitStatus = "CANCELLED"
	ExitStatusRejected   FiscalExitStatus = "REJECTED"
)

type FiscalExit struct {
	ID                      int64
	ChaveAcesso             *string
	NumeroNF                int64
	Serie                   string
	DataEmissao             time.Time
	DataSaida               *time.Time
	CnpjDestinatario        *string
	RazaoSocialDestinatario *string
	IEDestinatario          *string
	UFDestinatario          *string
	Cfop                    string
	NaturezaOperacao        string
	ValorProdutos           float64
	ValorFrete              float64
	ValorSeguro             float64
	ValorDesconto           float64
	ValorIPI                float64
	ValorICMS               float64
	ValorPIS                float64
	ValorCOFINS             float64
	BaseICMSST              float64
	ValorICMSST             float64
	ValorTotal              float64
	SalesOrderCode          *int64
	SourceType              *string
	ShipmentLoadCode        *int64
	ShipmentCode            *int64
	FiscalCouponNumber      *string
	FiscalCouponDate        *time.Time
	FiscalCouponECFSerial   *string
	Status                  FiscalExitStatus
	Protocolo               *string
	XmlPath                 *string
	DanfePath               *string
	FocusRef                *string
	IsActive                bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
	CreatedBy               uuid.UUID
	Itens                   []*FiscalExitItem
}

type FiscalExitItem struct {
	ID                int64
	FiscalExitID      int64
	Sequence          int
	ItemCode          *int64
	Ncm               *string
	Cfop              string
	Quantity          float64
	UnitPrice         float64
	TotalPrice        float64
	BaseICMS          float64
	AliqICMS          float64
	ValorICMS         float64
	ValorICMSDiferido float64
	BaseIPI           float64
	AliqIPI           float64
	ValorIPI          float64
	AliqPIS           float64
	ValorPIS          float64
	AliqCOFINS        float64
	ValorCOFINS       float64
	BaseICMSST        float64
	AliqICMSST        float64
	ValorICMSST       float64
	MVA               float64
	CstICMS           *string
	CstIPI            *string
	CstPIS            *string
	CstCOFINS         *string
	OrigemMercadoria  string
	Description       *string
	CreatedAt         time.Time
}
