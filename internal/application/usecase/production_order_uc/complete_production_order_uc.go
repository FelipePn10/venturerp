package production_order_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

// coproductReader exposes the BOM's direct children so completion can receive
// co-products / returnable scrap into stock.
type coproductReader interface {
	GetAllDirectChildren(ctx context.Context, parentCode int64) ([]*structentity.ItemStructure, error)
}

// baixasDaOrdemRepository diz o que a ordem já baixou, para não baixar de novo.
type baixasDaOrdemRepository interface {
	BaixasDaOrdem(ctx context.Context, ordem int64) (map[int64]decimal.Decimal, decimal.Decimal, decimal.Decimal, error)
}

// quantidadeAindaABaixar é o que falta sair do componente: o que a produção
// acumulada exige menos o que já saiu para a ordem.
func quantidadeAindaABaixar(exigido, baixado decimal.Decimal) decimal.Decimal {
	return decimal.Max(exigido.Sub(baixado), decimal.Zero)
}

type atomicDeliveryRepository interface {
	RegisterDeliveryWithMovements(context.Context, *entity.ProductionDelivery, []*stockentity.StockMovement) (*entity.ProductionOrder, error)
}

type CompleteProductionOrderUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
	// StockRepo is optional. When set together with a target warehouse on the
	// DTO, completing the order posts an IN stock movement of the produced
	// quantity for the finished item.
	StockRepo stockrepo.StockRepository
	// Structure is optional. When set with StockRepo, completing the order also
	// receives the item's BOM co-products / returnable scrap into stock.
	Structure coproductReader
}

func (uc *CompleteProductionOrderUseCase) Execute(
	ctx context.Context,
	dto request.CompleteProductionOrderDTO,
) (*entity.ProductionOrder, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	// Quem responde pelo movimento é quem concluiu, não quem abriu a ordem.
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if dto.ID == 0 {
		return nil, errorsuc.NewValidationError("informe o identificador")
	}

	order, err := uc.Repo.GetByCode(ctx, dto.ID)
	if err != nil {
		return nil, err
	}
	warehouseID := dto.WarehouseID
	if warehouseID == nil {
		warehouseID = order.WarehouseID
	}
	if warehouseID == nil {
		return nil, errorsuc.NewValidationError("informe o almoxarifado, pois a ordem não tem almoxarifado de destino")
	}
	delivered, err := uc.Repo.GetDeliveredQuantity(ctx, dto.ID)
	if err != nil {
		return nil, err
	}
	plannedQty := order.PlannedQty
	producedQty := order.ProducedQty
	qty := plannedQty.Sub(delivered)
	if dto.Quantity == nil && order.ProducedQty.IsPositive() {
		qty = producedQty.Sub(delivered)
	}
	if dto.Quantity != nil {
		qty = *dto.Quantity
	}
	if qty.IsNegative() {
		return nil, errorsuc.NewValidationError("a quantidade deve ser maior ou igual a zero")
	}
	key := strings.TrimSpace(dto.IdempotencyKey)
	if key == "" {
		key = fmt.Sprintf("legacy:%d:%s:%s:%t", dto.ID, dto.EndDate, qty.String(), dto.Final)
	}
	if existing, existingErr := uc.Repo.GetDeliveryByIdempotencyKey(ctx, key); existingErr == nil {
		if existing.ProductionOrderID != dto.ID || existing.WarehouseID != *warehouseID || existing.IsFinal != dto.Final || (dto.Quantity != nil && !existing.Quantity.Equal(*dto.Quantity)) || !sameDeliveryLot(existing.Lot, dto.Lot) {
			return nil, errorsuc.NewValidationError("a chave desta entrega já foi usada com outros dados; consulte a entrega registrada antes de tentar novamente")
		}
		return uc.Repo.GetByCode(ctx, existing.ProductionOrderID)
	} else if !errorsuc.EhNaoEncontrado(existingErr) {
		return nil, fmt.Errorf("conferindo a chave da entrega: %w", existingErr)
	}
	if dto.Final {
		pending, pendingErr := uc.Repo.HasPendingServicePurchaseOrders(ctx, dto.ID)
		if pendingErr != nil {
			return nil, pendingErr
		}
		if pending {
			return nil, errorsuc.NewValidationError("a ordem tem pedidos de serviço de terceiros ainda pendentes; conclua ou cancele os pedidos antes de encerrar a ordem")
		}
	}
	treatExcess, err := uc.Repo.TreatProductionExcess(ctx)
	if err != nil {
		return nil, err
	}
	movementClass := "EP"
	lines := []entity.ProductionDeliveryLine{}
	if treatExcess {
		movementClass = "EPP"
	}
	plannedPart := qty
	excessPart := decimal.Zero
	if treatExcess && delivered.Add(qty).GreaterThan(plannedQty) {
		plannedPart = plannedQty.Sub(delivered)
		if plannedPart.IsNegative() {
			plannedPart = decimal.Zero
		}
		excessPart = qty.Sub(plannedPart)
	}
	if plannedPart.IsPositive() {
		lines = append(lines, entity.ProductionDeliveryLine{MovementClass: movementClass, Quantity: plannedPart})
	}
	if excessPart.IsPositive() {
		lines = append(lines, entity.ProductionDeliveryLine{MovementClass: "EPE", Quantity: excessPart})
		if plannedPart.IsZero() {
			movementClass = "EPE"
		}
	}
	endDate := datetime.ParseDateOrDefault(dto.EndDate, time.Now())
	delivery := &entity.ProductionDelivery{ProductionOrderID: dto.ID, Quantity: qty, IdempotencyKey: key,
		MovementClass: movementClass, WarehouseID: *warehouseID, Lot: dto.Lot, IsFinal: dto.Final,
		DeliveredAt: endDate, CreatedBy: ator, Lines: lines,
		ItemCode: order.ItemCode, Mask: mascaraDaOrdem(order.Mask), NumerosSerie: dto.NumerosSerie}
	var movements []*stockentity.StockMovement
	if uc.StockRepo != nil {
		if qty.IsPositive() {
			// A estrutura é lida antes de qualquer movimento: é ela que diz se a
			// OF é de beneficiamento. Falha de leitura é erro — seguir sem ela
			// daria entrada do produto do cliente no estoque próprio.
			var children []*structentity.ItemStructure
			if uc.Structure != nil {
				lidos, cerr := uc.Structure.GetAllDirectChildren(ctx, order.ItemCode)
				if cerr != nil {
					return nil, fmt.Errorf("lendo a estrutura do item para a entrega da OF: %w", cerr)
				}
				children = structentity.SelectPrimarySubstituteComponents(lidos)
			}
			// OF de beneficiamento (estrutura com material do cliente): o produto
			// processado é do CLIENTE e volta pela nota de retorno (CFOP 5902).
			// A entrega fica registrada, sem entrada no estoque próprio — nem do
			// produto nem da sucata, que também é do cliente (CFOP 5903).
			beneficiamento := structentity.TemMaterialDoCliente(children)
			refType := stockentity.ReferenceTypeProductionOrder
			refCode := order.ID
			baixado := map[int64]decimal.Decimal{}
			entregueAntes := decimal.Zero
			conheceBaixas := false
			if b, ok := uc.Repo.(baixasDaOrdemRepository); ok {
				if baixado, entregueAntes, _, err = b.BaixasDaOrdem(ctx, order.ID); err != nil {
					return nil, err
				}
				conheceBaixas = true
			}
			appendFinishedMovement := func(class string, movementQty decimal.Decimal) {
				mov := &stockentity.StockMovement{
					ItemCode:     order.ItemCode,
					Mask:         order.Mask,
					WarehouseID:  *warehouseID,
					MovementType: class,
					Quantity:     movementQty,
					// O repositório põe o custo da ordem na transação da entrega; sem custo,
					// entra pelo médio: a zero, diluía o saldo.
					ValorizarPelaMedia: true,
					ReferenceType:      &refType,
					ReferenceCode:      &refCode,
					Lot:                dto.Lot,
					Notes:              stringPtr(class),
					CreatedBy:          ator,
				}
				movements = append(movements, mov)
			}
			if !beneficiamento {
				if len(lines) == 0 {
					appendFinishedMovement(movementClass, qty)
				} else {
					for _, line := range lines {
						appendFinishedMovement(line.MovementClass, line.Quantity)
					}
				}
			}

			// Receive the BOM's co-products / returnable scrap into stock (an IN
			// movement of coproduct_qty × produced, into the same warehouse).
			for _, child := range children {
				if child.CustomerSupplied {
					continue // material do cliente não sai do estoque próprio
				}
				if child.IsCoproduct && beneficiamento {
					continue // sucata do material do cliente é do cliente
				}
				if !child.IsCoproduct || !child.Quantity.IsPositive() {
					automatic, componentWarehouse, infoErr := uc.Repo.GetItemAutomaticIssue(ctx, child.ChildCode)
					if infoErr != nil {
						return nil, fmt.Errorf("lendo a baixa automática do componente: %w", infoErr)
					}
					if !automatic || child.IsCoproduct {
						continue
					}
					componentQty := child.Quantity
					if !child.IsFixedQty {
						componentQty = componentQty.Mul(qty)
						if conheceBaixas {
							componentQty = child.Quantity.Mul(entregueAntes.Add(qty))
						}
					}
					componentQty = structentity.QuantidadeComPerdaDecimal(componentQty, child.LossPercentage, structentity.FormulaPerdaPadrao)
					if conheceBaixas {
						// Já saiu pelo apontamento ou por entrega anterior: não sai de novo.
						componentQty = quantidadeAindaABaixar(componentQty, baixado[child.ChildCode])
						if !componentQty.IsPositive() {
							continue
						}
					}
					rep := &stockentity.StockMovement{ItemCode: child.ChildCode, WarehouseID: componentWarehouse,
						MovementType: stockentity.MovementTypePlannedRequisition, Quantity: componentQty, ReferenceType: &refType,
						ReferenceCode: &refCode, Notes: stringPtr("REP"), CreatedBy: ator}
					movements = append(movements, rep)
					continue
				}
				coMov := &stockentity.StockMovement{
					ItemCode:      child.ChildCode,
					WarehouseID:   *warehouseID,
					MovementType:  stockentity.MovementTypeIn,
					Quantity:      child.Quantity.Mul(qty),
					ReferenceType: &refType,
					ReferenceCode: &refCode,
					CreatedBy:     ator,
					// Co-produto não tem preço: entra pelo médio do item. A custo
					// zero ele diluía o valor de todo o saldo.
					ValorizarPelaMedia: true,
				}
				movements = append(movements, coMov)
			}
		}
	}
	if len(movements) > 0 {
		atomicRepo, ok := uc.Repo.(atomicDeliveryRepository)
		if ok {
			return atomicRepo.RegisterDeliveryWithMovements(ctx, delivery, movements)
		}
		// Compatibility path for in-memory adapters. The PostgreSQL production
		// adapter always implements atomicDeliveryRepository.
		for _, movement := range movements {
			if _, moveErr := uc.StockRepo.CreateMovement(ctx, movement); moveErr != nil {
				return nil, moveErr
			}
		}
	}
	order, err = uc.Repo.RegisterDelivery(ctx, delivery)
	if err != nil {
		return nil, err
	}

	return order, nil
}

func stringPtr(value string) *string { return &value }

// mascaraDaOrdem: a ordem guarda "" quando o item não é configurado.
func mascaraDaOrdem(m string) *string {
	if strings.TrimSpace(m) == "" {
		return nil
	}
	return &m
}

func sameDeliveryLot(a, b *string) bool {
	if a == nil {
		return b == nil || *b == ""
	}
	if b == nil {
		return *a == ""
	}
	return *a == *b
}
