package production_order_uc

import (
	"context"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
)

// Situação de disponibilidade de material — por linha e por OF.
const (
	AvailabilityComplete = "COMPLETA"  // tudo o que falta baixar está livre no depósito
	AvailabilityPartial  = "PARCIAL"   // há saldo, mas não para tudo
	AvailabilityMissing  = "SEM_SALDO" // nada livre para esta OF
	AvailabilityNone     = "SEM_LISTA" // a OF não tem lista de materiais a baixar
)

type materialAvailabilityRepository interface {
	MaterialAvailability(ctx context.Context, orderIDs []int64) ([]entity.MaterialAvailabilityLine, error)
}

// MaterialAvailabilityLine é a linha de faltas da OF.
type MaterialAvailabilityLine struct {
	MaterialID   int64           `json:"material_id"`
	ItemCode     int64           `json:"item_code"`
	BusinessCode string          `json:"business_code,omitempty"`
	Description  string          `json:"description"`
	Mask         string          `json:"mask,omitempty"`
	UOM          string          `json:"uom,omitempty"`
	WarehouseID  *int64          `json:"warehouse_id,omitempty"`
	Required     decimal.Decimal `json:"required"`
	Issued       decimal.Decimal `json:"issued"`
	Remaining    decimal.Decimal `json:"remaining"`
	FreeStock    decimal.Decimal `json:"free_stock"`
	// CommittedEarlier: o que OFs anteriores vão consumir do mesmo saldo.
	CommittedEarlier decimal.Decimal `json:"committed_earlier"`
	// Available é o que sobra do saldo livre para esta OF.
	Available decimal.Decimal `json:"available"`
	Shortage  decimal.Decimal `json:"shortage"`
	Status    string          `json:"status"`
}

// MaterialAvailabilitySummary é o resumo de faltas de uma OF (selo da lista).
type MaterialAvailabilitySummary struct {
	ProductionOrderID int64  `json:"production_order_id"`
	Status            string `json:"status"`
	Items             int    `json:"items"`
	MissingItems      int    `json:"missing_items"`
	// Lines só vem na consulta de uma OF.
	Lines []MaterialAvailabilityLine `json:"lines,omitempty"`
}

func (uc *ProductionMaterialControlUseCase) availabilityRepo() (materialAvailabilityRepository, error) {
	repo, ok := uc.Repo.(materialAvailabilityRepository)
	if !ok {
		return nil, errorsuc.NewValidationError("a checagem de disponibilidade de material não está disponível")
	}
	return repo, nil
}

// Availability é a checagem de faltas de uma OF antes de liberar — o "missing
// parts" do SAP (CO24) e a checagem de disponibilidade na liberação: para cada
// material, quanto falta baixar, quanto está livre no depósito depois das OFs
// que estão na frente, e quanto falta.
func (uc *ProductionMaterialControlUseCase) Availability(ctx context.Context, orderID int64) (*MaterialAvailabilitySummary, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	// OF inexistente ou de outra empresa é "não encontrada", não "sem lista".
	if _, err := uc.Repo.GetByCode(ctx, orderID); err != nil {
		return nil, err
	}
	out, err := uc.AvailabilitySummary(ctx, []int64{orderID}, true)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return &MaterialAvailabilitySummary{ProductionOrderID: orderID, Status: AvailabilityNone, Lines: []MaterialAvailabilityLine{}}, nil
	}
	return &out[0], nil
}

// AvailabilitySummary resume as faltas de várias OFs (o selo da lista). Com
// detalhe, traz as linhas.
func (uc *ProductionMaterialControlUseCase) AvailabilitySummary(ctx context.Context, orderIDs []int64, detalhe bool) ([]MaterialAvailabilitySummary, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if len(orderIDs) == 0 {
		return []MaterialAvailabilitySummary{}, nil
	}
	if len(orderIDs) > 500 {
		return nil, errorsuc.NewValidationError("consulte no máximo 500 ordens por vez")
	}
	repo, err := uc.availabilityRepo()
	if err != nil {
		return nil, err
	}
	linhas, err := repo.MaterialAvailability(ctx, orderIDs)
	if err != nil {
		return nil, err
	}
	porOF := map[int64]*MaterialAvailabilitySummary{}
	comSaldo := map[int64]bool{}
	for _, l := range linhas {
		s := porOF[l.ProductionOrderID]
		if s == nil {
			s = &MaterialAvailabilitySummary{ProductionOrderID: l.ProductionOrderID}
			porOF[l.ProductionOrderID] = s
		}
		linha := availabilityLine(l)
		if linha.Remaining.IsZero() {
			continue // já baixado por completo: não conta na checagem
		}
		s.Items++
		if linha.Available.IsPositive() {
			comSaldo[l.ProductionOrderID] = true
		}
		if linha.Status != AvailabilityComplete {
			s.MissingItems++
		}
		if detalhe {
			s.Lines = append(s.Lines, linha)
		}
	}
	out := make([]MaterialAvailabilitySummary, 0, len(orderIDs))
	for _, id := range orderIDs {
		s := porOF[id]
		if s == nil {
			out = append(out, MaterialAvailabilitySummary{ProductionOrderID: id, Status: AvailabilityNone})
			continue
		}
		switch {
		case s.MissingItems == 0:
			s.Status = AvailabilityComplete
		case s.MissingItems == s.Items && !comSaldo[id]:
			s.Status = AvailabilityMissing
		default:
			s.Status = AvailabilityPartial
		}
		if detalhe && s.Lines == nil {
			s.Lines = []MaterialAvailabilityLine{}
		}
		out = append(out, *s)
	}
	return out, nil
}

func availabilityLine(l entity.MaterialAvailabilityLine) MaterialAvailabilityLine {
	restante := decimal.Max(l.Required.Sub(l.Issued), decimal.Zero)
	livre := decimal.Max(l.FreeStock.Sub(l.CommittedEarlier), decimal.Zero)
	falta := decimal.Max(restante.Sub(livre), decimal.Zero)
	status := AvailabilityComplete
	switch {
	case falta.IsPositive() && livre.IsPositive():
		status = AvailabilityPartial
	case falta.IsPositive():
		status = AvailabilityMissing
	}
	return MaterialAvailabilityLine{
		MaterialID: l.MaterialID, ItemCode: l.ItemCode, BusinessCode: l.BusinessCode, Description: l.Description,
		Mask: l.Mask, UOM: l.UOM, WarehouseID: l.WarehouseID,
		Required: l.Required, Issued: l.Issued, Remaining: restante,
		FreeStock: l.FreeStock, CommittedEarlier: l.CommittedEarlier,
		Available: livre, Shortage: falta, Status: status,
	}
}
