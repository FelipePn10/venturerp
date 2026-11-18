package entity

import (
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ItemPreferredSupplier struct {
	ID                    int64
	EnterpriseID          int64
	ItemCode              int64
	SupplierCode          int64
	Mask                  string
	Ranking               int32
	SupplierItemCode      *string
	SupplierDescription   *string
	UOM                   *string
	XMLUOM                *string
	ConversionFactor      *decimal.Decimal
	PackageQuantity       decimal.Decimal
	IsPreferred           bool
	SupplierUF            *string
	ClassificationID      *int64
	ClassificationDate    *time.Time
	ClassificationGrade   *decimal.Decimal
	DirectBilling         bool
	ThirdPartyOrder       bool
	IgnoreAvgCostAddition bool
	Ecommerce             bool
	Barcode               *string
	Notes                 *string
	ValidFrom             *time.Time
	ValidUntil            *time.Time
	LeadTimeDays          int32
	// Política de compra DESTE fornecedor (unidade de estoque). Zero = sem.
	MinOrderQty   decimal.Decimal
	OrderMultiple decimal.Decimal
	// QuotaPercent: cota da compra do item para este fornecedor. nil = fora
	// da divisão por cota.
	QuotaPercent *decimal.Decimal
	// Bloqueio do vínculo: bloqueado não é fonte (nem para o MRP, nem para o
	// documento de compra), mas o histórico fica.
	Blocked       bool
	BlockedReason *string
	BlockedAt     *time.Time
	BlockedBy     *uuid.UUID
	IsActive      bool
	CreatedAt     time.Time
	CreatedBy     uuid.UUID
	UpdatedAt     time.Time
}

func NewItemPreferredSupplier(enterpriseID, itemCode, supplierCode int64, mask string, ranking int32, createdBy uuid.UUID) (*ItemPreferredSupplier, error) {
	if enterpriseID <= 0 || itemCode <= 0 || supplierCode <= 0 {
		return nil, domainerr.NewValidation("informe a empresa, o item e o fornecedor")
	}
	if ranking <= 0 {
		ranking = 1
	}
	now := time.Now()
	return &ItemPreferredSupplier{EnterpriseID: enterpriseID, ItemCode: itemCode, SupplierCode: supplierCode, Mask: strings.TrimSpace(mask), Ranking: ranking, PackageQuantity: decimal.Zero, MinOrderQty: decimal.Zero, OrderMultiple: decimal.Zero, IsActive: true, CreatedAt: now, CreatedBy: createdBy, UpdatedAt: now}, nil
}
func (s *ItemPreferredSupplier) Validate() error {
	if s.PackageQuantity.IsNegative() {
		return domainerr.NewValidation("a quantidade por embalagem não pode ser negativa")
	}
	if s.LeadTimeDays < 0 {
		return domainerr.NewValidation("o prazo de entrega do fornecedor não pode ser negativo")
	}
	if s.MinOrderQty.IsNegative() {
		return domainerr.NewValidation("a quantidade mínima de compra não pode ser negativa")
	}
	if s.OrderMultiple.IsNegative() {
		return domainerr.NewValidation("o múltiplo de compra não pode ser negativo")
	}
	if s.QuotaPercent != nil && (!s.QuotaPercent.IsPositive() || s.QuotaPercent.GreaterThan(decimal.NewFromInt(100))) {
		return domainerr.NewValidation("a cota do fornecedor deve ficar entre 0 e 100%")
	}
	if s.ValidFrom != nil && s.ValidUntil != nil && s.ValidFrom.After(*s.ValidUntil) {
		return domainerr.NewValidation("o início da vigência não pode ser depois do fim")
	}
	if s.IsPreferred {
		s.Ranking = 1
	}
	if s.DirectBilling && !s.IsPreferred {
		return domainerr.NewValidation("faturamento direto exige fornecedor preferencial")
	}
	if s.ThirdPartyOrder && !s.DirectBilling {
		return domainerr.NewValidation("ordem de terceiro exige faturamento direto")
	}
	if s.XMLUOM != nil && s.UOM == nil {
		return domainerr.NewValidation("a unidade do XML exige a unidade do fornecedor")
	}
	return nil
}

// Situação do vínculo numa data: o que a tela mostra e o que decide se ele é
// fonte de suprimento.
const (
	SituacaoVigente   = "VIGENTE"
	SituacaoBloqueado = "BLOQUEADO"
	SituacaoVencido   = "VENCIDO"
	SituacaoFuturo    = "FUTURO"
)

func (s *ItemPreferredSupplier) Situacao(on time.Time) string {
	dia := time.Date(on.Year(), on.Month(), on.Day(), 0, 0, 0, 0, time.UTC)
	data := func(t *time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }
	switch {
	case s.Blocked:
		return SituacaoBloqueado
	case s.ValidUntil != nil && data(s.ValidUntil).Before(dia):
		return SituacaoVencido
	case s.ValidFrom != nil && data(s.ValidFrom).After(dia):
		return SituacaoFuturo
	}
	return SituacaoVigente
}

// Bloqueio/desbloqueio do vínculo: o motivo é obrigatório nos dois sentidos.
const (
	AcaoBloqueio    = "BLOQUEIO"
	AcaoDesbloqueio = "DESBLOQUEIO"
)

type BlockEvent struct {
	ID             int64
	EnterpriseID   int64
	ItemSupplierID int64
	Action         string
	Reason         string
	ActorID        uuid.UUID
	CreatedAt      time.Time
}

func NewBlockEvent(enterpriseID, link int64, action, reason string, actor uuid.UUID) (*BlockEvent, error) {
	reason = strings.TrimSpace(reason)
	if enterpriseID <= 0 || link <= 0 {
		return nil, domainerr.NewValidation("informe o vínculo do fornecedor")
	}
	if action != AcaoBloqueio && action != AcaoDesbloqueio {
		return nil, domainerr.NewValidation("ação de bloqueio inválida")
	}
	if reason == "" {
		return nil, domainerr.NewValidation("informe o motivo")
	}
	return &BlockEvent{EnterpriseID: enterpriseID, ItemSupplierID: link, Action: action, Reason: reason, ActorID: actor}, nil
}

// CotaMaxima: a soma das cotas dos fornecedores ativos de um item.
var CotaMaxima = decimal.NewFromInt(100)

type QualityReport struct {
	ID             int64
	EnterpriseID   int64
	ItemSupplierID int64
	RegisteredOn   time.Time
	Status         string
	FileName       *string
	ContentType    *string
	Content        []byte
	Notes          *string
	CreatedAt      time.Time
	CreatedBy      uuid.UUID
}

func NewQualityReport(e, link int64, on time.Time, status string, by uuid.UUID) (*QualityReport, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	if e <= 0 || link <= 0 || on.IsZero() || (status != "PENDENTE" && status != "APROVADO" && status != "REJEITADO" && status != "EXPIRADO") {
		return nil, domainerr.NewValidation("laudo de qualidade inválido")
	}
	return &QualityReport{EnterpriseID: e, ItemSupplierID: link, RegisteredOn: on, Status: status, CreatedBy: by}, nil
}
