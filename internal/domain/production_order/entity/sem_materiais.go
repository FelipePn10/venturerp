package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OFSemMateriais é a OF aberta ou em produção que não tem lista de materiais
// (criada antes de a lista existir ou por carga direta no banco). A rotina que
// completa a lista a trata na sessão da empresa dela.
type OFSemMateriais struct {
	ID             int64
	EnterpriseID   int64
	EnterpriseCode int64
	ItemCode       int64
	Mask           string
	PlannedQty     decimal.Decimal
	StartDate      *time.Time
	CreatedBy      uuid.UUID
	// JaProduziu: a OF já tem apontamento, entrega ou material que saiu para
	// ela. A lista completada agora não pode baixar de uma vez, na data de
	// hoje, o consumo do que foi produzido antes.
	JaProduziu bool
}
