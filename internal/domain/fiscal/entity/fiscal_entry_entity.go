package entity

import (
	"time"

	"github.com/google/uuid"
)

type FiscalEntryStatus string

const (
	EntryStatusPending    FiscalEntryStatus = "PENDING"
	EntryStatusConferred  FiscalEntryStatus = "CONFERRED"
	EntryStatusApproved   FiscalEntryStatus = "APPROVED"
	EntryStatusWrittenOff FiscalEntryStatus = "WRITTEN_OFF"
	EntryStatusCancelled  FiscalEntryStatus = "CANCELLED"
)

type FiscalEntry struct {
	ID                   int64
	ChaveAcesso          *string
	NumeroNF             int64
	Serie                string
	Modelo               string
	DataEmissao          time.Time
	DataEntrada          time.Time
	CnpjEmitente         string
	RazaoSocialEmitente  string
	IEEmitente           *string
	UFEmitente           *string
	ValorProdutos        float64
	ValorFrete           float64
	ValorSeguro          float64
	ValorDesconto        float64
	ValorIPI             float64
	ValorICMS            float64
	ValorPIS             float64
	ValorCOFINS          float64
	ValorTotal           float64
	TipoDocumento        string
	PurchaseOrderCode    *int64
	CteCode              *int64
	Status               FiscalEntryStatus
	XmlPath              *string
	Notes                *string
	IsActive             bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
	CreatedBy            uuid.UUID
	Itens                []*FiscalEntryItem
}

type FiscalEntryItem struct {
	ID                int64
	FiscalEntryID     int64
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
	BaseIPI           float64
	AliqIPI           float64
	ValorIPI          float64
	ValorPIS          float64
	ValorCOFINS       float64
	CstICMS           *string
	CstIPI            *string
	CstPIS            *string
	CstCOFINS         *string
	GeraCreditoICMS   bool
	GeraCreditoIPI    bool
	GeraCreditoPIS    bool
	GeraCreditoCOFINS bool
	Description       *string
	Notes             *string
	CreatedAt         time.Time
}
