package purchase_order_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

func dataOpcional(v *string, nome string) (*time.Time, error) {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(*v))
	if err != nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("data %s inválida; use AAAA-MM-DD", nome))
	}
	return &t, nil
}

// linhaDoPedido carrega o pedido e a linha, conferindo que ela é dele.
func linhaDoPedido(ctx context.Context, repo repository.PurchaseOrderRepository, pedido, linha int64) (*entity.PurchaseOrder, []*entity.PurchaseOrderItem, *entity.PurchaseOrderItem, error) {
	po, err := repo.GetByCode(ctx, pedido)
	if err != nil {
		return nil, nil, nil, err
	}
	if !po.IsActive {
		return nil, nil, nil, errorsuc.NewValidationError(fmt.Sprintf("o pedido %d está cancelado", po.Code))
	}
	itens, err := repo.ListItems(ctx, pedido)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, it := range itens {
		if it.Code == linha {
			return po, itens, it, nil
		}
	}
	return nil, nil, nil, errorsuc.NewNotFoundError(fmt.Sprintf("a linha %d não pertence ao pedido %d", linha, pedido))
}

// UpdatePurchaseOrderItemUseCase altera quantidade, preço, impostos, entrega e
// destino de uma linha enquanto o pedido não foi aprovado.
type UpdatePurchaseOrderItemUseCase struct {
	Repo repository.PurchaseOrderRepository
	Auth ports.AuthService
}

func (uc *UpdatePurchaseOrderItemUseCase) Execute(ctx context.Context, dto request.UpdatePurchaseOrderItemDTO) (*response.PurchaseOrderResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	po, _, it, err := linhaDoPedido(ctx, uc.Repo, dto.PurchaseOrderCode, dto.ItemLineCode)
	if err != nil {
		return nil, err
	}
	if err := po.Editavel(); err != nil {
		return nil, errorsuc.NewValidationError(err.Error())
	}
	if it.ReceivedQty > 0 || it.Status == entity.PurchaseOrderItemStatusCANCELLED {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a linha %d já tem recebimento ou está cancelada", it.Sequence))
	}
	set := func(dst *float64, v *float64, nome string, max float64) error {
		if v == nil {
			return nil
		}
		if *v < 0 || (max > 0 && *v > max) {
			return errorsuc.NewValidationError(nome + " inválido")
		}
		*dst = *v
		return nil
	}
	if dto.RequestedQty != nil && *dto.RequestedQty <= 0 {
		return nil, errorsuc.NewValidationError("a quantidade deve ser maior que zero")
	}
	qtdAntes := it.RequestedQty
	for _, e := range []error{
		set(&it.RequestedQty, dto.RequestedQty, "quantidade", 0),
		set(&it.UnitPrice, dto.UnitPrice, "preço", 0),
		set(&it.DiscountPct, dto.DiscountPct, "desconto", 100),
		set(&it.IPIPct, dto.IPIPct, "IPI", 100),
		set(&it.ICMSPct, dto.ICMSPct, "ICMS", 100),
		set(&it.TolerancePct, dto.TolerancePct, "tolerância", 100),
	} {
		if e != nil {
			return nil, e
		}
	}
	// A quantidade em UM de estoque acompanha a de compra na mesma proporção.
	if qtdAntes > 0 && it.InternalQty > 0 {
		it.InternalQty = it.InternalQty * it.RequestedQty / qtdAntes
	} else {
		it.InternalQty = it.RequestedQty
	}
	if dto.UnitPrice != nil && (it.PurchaseUOM == nil || it.InternalUOM == nil || *it.PurchaseUOM == *it.InternalUOM) {
		it.InternalPrice = it.UnitPrice
	}
	if dto.WarehouseID != nil {
		if *dto.WarehouseID <= 0 {
			return nil, errorsuc.NewValidationError("almoxarifado inválido")
		}
		it.WarehouseID = dto.WarehouseID
	}
	if dto.CostCenterCode != nil {
		it.CostCenterCode = positivoOuNil(dto.CostCenterCode)
	}
	if dto.FiscalClassificationCode != nil {
		it.FiscalClassificationCode = positivoOuNil(dto.FiscalClassificationCode)
	}
	if dto.InvoiceTypeCode != nil {
		it.InvoiceTypeCode = positivoOuNil(dto.InvoiceTypeCode)
	}
	if dto.UtilizationType != nil {
		it.UtilizationType = vazioOuNil(dto.UtilizationType)
	}
	if dto.Notes != nil {
		it.Notes = vazioOuNil(dto.Notes)
	}
	if dto.DeliveryDate != nil {
		if it.DeliveryDate, err = dataOpcional(dto.DeliveryDate, "de entrega"); err != nil {
			return nil, err
		}
	}
	if dto.PromisedDate != nil {
		if it.PromisedDate, err = dataOpcional(dto.PromisedDate, "prometida"); err != nil {
			return nil, err
		}
	}
	it.TotalPrice = entity.TotalDaLinha(it.RequestedQty, it.UnitPrice, it.DiscountPct)
	if _, err := uc.Repo.UpdateItem(ctx, it); err != nil {
		return nil, err
	}
	return regravarComTotais(ctx, uc.Repo, po)
}

func positivoOuNil(v *int64) *int64 {
	if v == nil || *v <= 0 {
		return nil
	}
	return v
}

func vazioOuNil(v *string) *string {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	s := strings.TrimSpace(*v)
	return &s
}

// CancelPurchaseOrderItemUseCase tira uma linha do pedido.
//
// Pedido ainda não aprovado: a linha é removida. Pedido aprovado: o que já
// chegou fica; o que falta é eliminado (resíduo) e deixa de ser esperado —
// a linha fecha, e a capa fecha quando não sobra saldo em nenhuma.
type CancelPurchaseOrderItemUseCase struct {
	Repo repository.PurchaseOrderRepository
	Auth ports.AuthService
}

func (uc *CancelPurchaseOrderItemUseCase) Execute(ctx context.Context, dto request.CancelPurchaseOrderItemDTO) (*response.PurchaseOrderResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	po, itens, it, err := linhaDoPedido(ctx, uc.Repo, dto.PurchaseOrderCode, dto.ItemLineCode)
	if err != nil {
		return nil, err
	}
	if it.Status == entity.PurchaseOrderItemStatusCANCELLED {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a linha %d já está cancelada", it.Sequence))
	}
	motivo := strings.TrimSpace(dto.Motivo)

	if po.Editavel() == nil && it.ReceivedQty == 0 {
		it.IsActive = false
		it.Status = entity.PurchaseOrderItemStatusCANCELLED
		if _, err := uc.Repo.UpdateItem(ctx, it); err != nil {
			return nil, err
		}
		return regravarComTotais(ctx, uc.Repo, po)
	}

	switch po.Status {
	case entity.PurchaseOrderStatusAPPROVED, entity.PurchaseOrderStatusPARTIAL:
	default:
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o pedido %d está %s; não há saldo a eliminar", po.Code, entity.RotuloSituacao(po.Status)))
	}
	saldo := it.Saldo()
	if saldo <= 0.0001 {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a linha %d não tem saldo a receber", it.Sequence))
	}
	if motivo == "" {
		return nil, errorsuc.NewValidationError("informe o motivo da eliminação do saldo (fica no histórico da linha)")
	}
	it.CancelledQty = it.RequestedQty - it.ReceivedQty
	if it.ReceivedQty > 0 {
		it.Status = entity.PurchaseOrderItemStatusRECEIVED
	} else {
		it.Status = entity.PurchaseOrderItemStatusCANCELLED
	}
	nota := fmt.Sprintf("saldo de %s eliminado em %s: %s", formatarQtd(saldo), time.Now().Format("02/01/2006"), motivo)
	if it.Notes != nil && strings.TrimSpace(*it.Notes) != "" {
		nota = strings.TrimSpace(*it.Notes) + " | " + nota
	}
	it.Notes = &nota
	if _, err := uc.Repo.UpdateItem(ctx, it); err != nil {
		return nil, err
	}
	po.Status = entity.SituacaoPelasLinhas(po.Status, itens)
	// O valor do compromisso caiu; a aprovação já dada continua valendo (um
	// pedido menor não precisa de nova alçada).
	items, err := uc.Repo.ListItems(ctx, po.Code)
	if err != nil {
		return nil, err
	}
	po.AplicarTotais(entity.CalcularTotais(po, items))
	updated, err := uc.Repo.Update(ctx, po)
	if err != nil {
		return nil, err
	}
	updated.Items = items
	return toPurchaseOrderResponse(updated), nil
}

func formatarQtd(v float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", v), "0"), ".")
}
