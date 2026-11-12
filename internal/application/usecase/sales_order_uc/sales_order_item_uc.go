package sales_order_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/salespricing"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/unit_of_measure_uc"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type CreateSalesOrderItemUseCase struct {
	Repo   repository.SalesOrderRepository
	Auth   ports.AuthService
	Prices customerrepo.CustomerRepository
	Items  any
	// Tributos sugere os impostos da linha pelo cálculo da nota. Opcional.
	Tributos ports.SugestorTributosVenda
	// Politica aplica as políticas comerciais e a alçada de desconto. Opcional.
	Politica *PoliticaDoPedido
	// Unidades confere a unidade de venda no catálogo da empresa. Opcional.
	// A nota de saída usa essa unidade e converte para a de estoque: "KGS"
	// gravado aqui só falhava no faturamento.
	Unidades unit_of_measure_uc.Catalogo
}

func (uc *CreateSalesOrderItemUseCase) Execute(
	ctx context.Context,
	dto request.CreateSalesOrderItemDTO,
) (*response.SalesOrderItemResponse, error) {
	if msg := datetime.PrimeiraInvalida(datetime.Campo("data de entrega", dto.DeliveryDate)); msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	}
	if !uc.Auth.CanCreateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	if dto.SalesOrderCode == 0 {
		return nil, errorsuc.NewValidationError("o código do pedido é obrigatório")
	}
	dto.SalesUOM = unit_of_measure_uc.NormalizarUnidade(dto.SalesUOM)
	if err := unit_of_measure_uc.ConferirNoCatalogo(ctx, uc.Unidades, "unidade de venda", dto.SalesUOM); err != nil {
		return nil, err
	}
	resolvedItem, err := itemresolution.ResolveActive(ctx, uc.Items, dto.ItemCode)
	if err != nil {
		return nil, err
	}
	itemCode := int64(resolvedItem.Code)
	if err = itemresolution.ValidateMask(ctx, uc.Items, itemCode, dto.Mask); err != nil {
		return nil, err
	}
	order, err := uc.Repo.GetByCode(ctx, dto.SalesOrderCode)
	if err != nil {
		return nil, err
	}
	if err = resolvedItem.ConferirVenda(dto.RequestedQty, order.CustomerCode); err != nil {
		return nil, err
	}
	tableCode := order.PriceTableCode
	if dto.PriceTableCode != nil {
		tableCode = dto.PriceTableCode
	}
	if tableCode == nil {
		return nil, errorsuc.NewValidationError("o pedido não possui tabela de preço")
	}
	pricing, err := salespricing.Resolve(ctx, uc.Prices, *tableCode, itemCode, dto.RequestedQty)
	if err != nil {
		return nil, err
	}
	dto.UnitPrice, dto.PriceTableCode = pricing.AppliedPrice, tableCode
	if err := uc.Politica.ConferirDesconto(ctx, dto.DiscountPct); err != nil {
		return nil, err
	}

	if err := preencherTributos(ctx, uc.Tributos, dto.TributosManuais, dto.SalesOrderCode, itemCode, dto.RequestedQty, dto.UnitPrice, dto.DiscountPct,
		percentuaisDaLinha{IPI: &dto.IPIPct, ICMS: &dto.ICMSPct, PIS: &dto.PISPct, COFINS: &dto.COFINSPct, ST: &dto.STPct}); err != nil {
		return nil, err
	}

	digitDate := datetime.ParseDateOrDefault(dto.DigitDate, time.Now())

	item := &entity.SalesOrderItem{
		SalesOrderCode:   dto.SalesOrderCode,
		Sequence:         dto.Sequence,
		ItemCode:         itemCode,
		Mask:             dto.Mask,
		DigitDate:        digitDate,
		NFType:           dto.NFType,
		SalesUOM:         dto.SalesUOM,
		WarehouseCode:    dto.WarehouseCode,
		PriceTableCode:   dto.PriceTableCode,
		RequestedQty:     dto.RequestedQty,
		UnitPrice:        dto.UnitPrice,
		DeliveryDateFirm: dto.DeliveryDateFirm,
		CustomerDelivery: dto.CustomerDelivery,
		Lot:              dto.Lot,
		CouponDelivery:   dto.CouponDelivery,
		PaidAtCashier:    dto.PaidAtCashier,
		IPIPct:           dto.IPIPct,
		ICMSPct:          dto.ICMSPct,
		PISPct:           dto.PISPct,
		COFINSPct:        dto.COFINSPct,
		STPct:            dto.STPct,
		DiscountPct:      dto.DiscountPct,
		UnitWeightNet:    dto.UnitWeightNet,
		UnitWeightGross:  dto.UnitWeightGross,
		Notes:            dto.Notes,
		Status:           entity.SalesOrderItemStatusOpen,
	}

	item.DeliveryDate = datetime.ParseDatePtr(dto.DeliveryDate)
	if item.DeliveryDate == nil && order.DeliveryDate == nil {
		// Sem data na linha nem no pedido, vale o prazo estimado do cadastro. A
		// data do pedido vence: o MRP lê a da linha antes da do cabeçalho.
		item.DeliveryDate = resolvedItem.EntregaPadrao(time.Now())
	}

	item.RecalcularTotais()

	item.UnitWeightNet = dto.UnitWeightNet
	item.UnitWeightGross = dto.UnitWeightGross

	created, err := uc.Repo.CreateItem(ctx, item)
	if err != nil {
		return nil, err
	}
	// A capa (crédito, comissão, rateio do frete no faturamento) lê os totais
	// do pedido: sem refazer, eles ficavam zerados depois de incluir a linha.
	if err := uc.Repo.RecalculateTotals(ctx, dto.SalesOrderCode); err != nil {
		return nil, err
	}
	if err := uc.Politica.Aplicar(ctx, dto.SalesOrderCode); err != nil {
		return nil, err
	}
	return toSalesOrderItemResponse(created), nil
}

type UpdateSalesOrderItemUseCase struct {
	Repo   repository.SalesOrderRepository
	Auth   ports.AuthService
	Prices customerrepo.CustomerRepository
	Items  any
	// Tributos sugere os impostos da linha pelo cálculo da nota. Opcional.
	Tributos ports.SugestorTributosVenda
	// Politica aplica as políticas comerciais e a alçada de desconto. Opcional.
	Politica *PoliticaDoPedido
	// Contratos mantém o preço da linha de chamada de contrato. Opcional.
	Contratos PrecosContratados
	// Transacao: a linha e o ajuste da reserva dela juntos. Opcional.
	Transacao EmTransacao
}

// quantidadesDaAlteracao: o atendido é o que as notas (ou o atendimento
// manual) registraram — o corpo da alteração não o muda; antes qualquer
// usuário marcava a linha como entregue sem nota. O cancelado não volta atrás
// e solicitado não fica abaixo do atendido + cancelado.
func quantidadesDaAlteracao(current *entity.SalesOrderItem, dto *request.UpdateSalesOrderItemDTO) error {
	dto.AttendedQty = current.AttendedQty
	if dto.CancelledQty.LessThan(current.CancelledQty) {
		dto.CancelledQty = current.CancelledQty
	}
	if !dto.RequestedQty.IsPositive() {
		return errorsuc.NewValidationError("a quantidade da linha deve ser maior que zero")
	}
	if dto.RequestedQty.LessThan(dto.AttendedQty.Add(dto.CancelledQty)) {
		return errorsuc.NewValidationError(fmt.Sprintf(
			"a quantidade %s fica abaixo do já atendido (%s) mais o cancelado (%s)",
			dto.RequestedQty.String(), dto.AttendedQty.String(), dto.CancelledQty.String()))
	}
	return nil
}

func (uc *UpdateSalesOrderItemUseCase) Execute(
	ctx context.Context,
	dto request.UpdateSalesOrderItemDTO,
) (*response.SalesOrderItemResponse, error) {
	if msg := datetime.PrimeiraInvalida(datetime.Campo("data de entrega", dto.DeliveryDate)); msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	}
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	current, err := uc.Repo.GetItem(ctx, dto.Code)
	if err != nil {
		return nil, err
	}
	if dto.ItemCode != nil {
		resolvedItem, resolveErr := itemresolution.ResolveActive(ctx, uc.Items, *dto.ItemCode)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if int64(resolvedItem.Code) != current.ItemCode {
			return nil, errorsuc.NewValidationError("o item da linha não pode ser alterado; cancele a linha e inclua outro item")
		}
	}
	order, err := uc.Repo.GetByCode(ctx, current.SalesOrderCode)
	if err != nil {
		return nil, err
	}
	switch {
	case order.Status == entity.SalesOrderStatusCancelled:
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o pedido %d está cancelado", order.Code))
	case order.Status == entity.SalesOrderStatusInvoiced:
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o pedido %d está faturado: cancele a NF-e antes de mexer nas linhas", order.Code))
	case current.Status == entity.SalesOrderItemStatusCancelled:
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a linha %d está cancelada", current.Code))
	}
	if err := quantidadesDaAlteracao(current, &dto); err != nil {
		return nil, err
	}
	if uc.Items != nil {
		cadastro, errItem := itemresolution.PelaChave(ctx, uc.Items, current.ItemCode)
		if errItem != nil {
			return nil, errItem
		}
		if err = cadastro.ConferirVenda(dto.RequestedQty, order.CustomerCode); err != nil {
			return nil, err
		}
	}
	tableCode := current.PriceTableCode
	if tableCode == nil {
		tableCode = order.PriceTableCode
	}
	var contratado *decimal.Decimal
	if uc.Contratos != nil {
		if contratado, err = uc.Contratos.PrecoContratado(ctx, current.Code); err != nil {
			return nil, err
		}
	}
	if contratado != nil {
		// Linha de chamada de contrato: o preço é o do contrato, não o da tabela.
		dto.UnitPrice = *contratado
	} else {
		if tableCode == nil {
			return nil, errorsuc.NewValidationError("o pedido não possui tabela de preço")
		}
		pricing, err := salespricing.Resolve(ctx, uc.Prices, *tableCode, current.ItemCode, dto.RequestedQty)
		if err != nil {
			return nil, err
		}
		dto.UnitPrice = pricing.AppliedPrice
	}
	// Só confere quando o desconto sobe: alterar a quantidade de uma linha já
	// aceita não deve ser recusado por causa do desconto que ela já tinha.
	if dto.DiscountPct.GreaterThan(current.DiscountPct) {
		if err := uc.Politica.ConferirDesconto(ctx, dto.DiscountPct); err != nil {
			return nil, err
		}
	}
	if err := preencherTributos(ctx, uc.Tributos, dto.TributosManuais, current.SalesOrderCode, current.ItemCode, dto.RequestedQty, dto.UnitPrice, dto.DiscountPct,
		percentuaisDaLinha{IPI: &dto.IPIPct, ICMS: &dto.ICMSPct, PIS: &dto.PISPct, COFINS: &dto.COFINSPct, ST: &dto.STPct}); err != nil {
		return nil, err
	}

	item := &entity.SalesOrderItem{
		Code:             dto.Code,
		RequestedQty:     dto.RequestedQty,
		UnitPrice:        dto.UnitPrice,
		AttendedQty:      dto.AttendedQty,
		CancelledQty:     dto.CancelledQty,
		DeliveryDateFirm: dto.DeliveryDateFirm,
		CustomerDelivery: dto.CustomerDelivery,
		Lot:              dto.Lot,
		CouponDelivery:   dto.CouponDelivery,
		PaidAtCashier:    dto.PaidAtCashier,
		IPIPct:           dto.IPIPct,
		ICMSPct:          dto.ICMSPct,
		PISPct:           dto.PISPct,
		COFINSPct:        dto.COFINSPct,
		STPct:            dto.STPct,
		DiscountPct:      dto.DiscountPct,
		UnitWeightNet:    dto.UnitWeightNet,
		UnitWeightGross:  dto.UnitWeightGross,
		Notes:            dto.Notes,
	}

	// Campo opcional que não veio no corpo mantém o valor da linha: o PUT
	// apagava data de entrega, lote, entrega ao cliente, cupom e observação de
	// quem mandava só a quantidade. Para limpar, envie o campo vazio.
	item.DeliveryDate = current.DeliveryDate
	if dto.DeliveryDate != nil {
		item.DeliveryDate = datetime.ParseDatePtr(dto.DeliveryDate)
	}
	item.CustomerDelivery = manterSeAusente(dto.CustomerDelivery, current.CustomerDelivery)
	item.Lot = manterSeAusente(dto.Lot, current.Lot)
	item.CouponDelivery = manterSeAusente(dto.CouponDelivery, current.CouponDelivery)
	item.Notes = manterSeAusente(dto.Notes, current.Notes)

	// Determine status from quantities
	balance := dto.RequestedQty.Sub(dto.AttendedQty).Sub(dto.CancelledQty)
	switch {
	case dto.CancelledQty.GreaterThanOrEqual(dto.RequestedQty):
		item.Status = entity.SalesOrderItemStatusCancelled
	case dto.AttendedQty.GreaterThanOrEqual(dto.RequestedQty):
		item.Status = entity.SalesOrderItemStatusDelivered
	case dto.AttendedQty.IsPositive() || balance.LessThan(dto.RequestedQty):
		item.Status = entity.SalesOrderItemStatusPartial
	default:
		item.Status = entity.SalesOrderItemStatusOpen
	}

	item.RecalcularTotais()

	var updated *entity.SalesOrderItem
	alterar := func(r ConfirmacaoRepos) error {
		var errUpd error
		if updated, errUpd = r.Pedidos.UpdateItem(ctx, item); errUpd != nil {
			return errUpd
		}
		if r.Estoque == nil {
			return nil
		}
		aberto := dto.RequestedQty.Sub(dto.AttendedQty).Sub(dto.CancelledQty)
		if err := aparaReservaDaLinha(ctx, r, current, aberto); err != nil {
			return err
		}
		// A linha que aumentou reserva o acréscimo (dentro do disponível), como
		// a confirmação faria; antes a reserva ficava na quantidade antiga.
		return reporReservas(ctx, r, current.SalesOrderCode, current.Code)
	}
	if uc.Transacao != nil {
		err = uc.Transacao(ctx, alterar)
	} else {
		err = alterar(ConfirmacaoRepos{Pedidos: uc.Repo})
	}
	if err != nil {
		return nil, err
	}
	if err := uc.Repo.RecalculateTotals(ctx, current.SalesOrderCode); err != nil {
		return nil, err
	}
	if err := uc.Politica.Aplicar(ctx, current.SalesOrderCode); err != nil {
		return nil, err
	}
	return toSalesOrderItemResponse(updated), nil
}

type ListSalesOrderItemsUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *ListSalesOrderItemsUseCase) Execute(ctx context.Context, salesOrderCode int64) ([]*response.SalesOrderItemResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	items, err := uc.Repo.ListItems(ctx, salesOrderCode)
	if err != nil {
		return nil, err
	}
	return toSalesOrderItemResponsePtrs(items), nil
}

type CancelSalesOrderItemUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
	// Transacao: linha, reserva da linha e totais juntos. Opcional (sem ela,
	// só a linha e os totais, sem soltar reserva).
	Transacao EmTransacao
}

func (uc *CancelSalesOrderItemUseCase) Execute(ctx context.Context, itemCode int64) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	item, err := uc.Repo.GetItem(ctx, itemCode)
	if err != nil {
		return err
	}
	switch item.Status {
	case entity.SalesOrderItemStatusCancelled:
		return errorsuc.NewValidationError(fmt.Sprintf("a linha %d já está cancelada", itemCode))
	case entity.SalesOrderItemStatusDelivered:
		return errorsuc.NewValidationError(fmt.Sprintf("a linha %d já foi atendida por inteiro: não há saldo a cancelar", itemCode))
	}
	pedido, err := uc.Repo.GetByCode(ctx, item.SalesOrderCode)
	if err != nil {
		return err
	}
	switch pedido.Status {
	case entity.SalesOrderStatusCancelled:
		return errorsuc.NewValidationError(fmt.Sprintf("o pedido %d está cancelado", item.SalesOrderCode))
	case entity.SalesOrderStatusInvoiced:
		return errorsuc.NewValidationError(fmt.Sprintf("o pedido %d está faturado: cancele a NF-e antes de mexer nas linhas", item.SalesOrderCode))
	}
	cancelar := func(r ConfirmacaoRepos) error {
		if err := r.Pedidos.CancelItem(ctx, itemCode); err != nil {
			return err
		}
		// A reserva que a confirmação fez para a linha deixava o estoque preso.
		if r.Estoque != nil {
			linhas, err := r.Pedidos.ListItems(ctx, item.SalesOrderCode)
			if err != nil {
				return err
			}
			alvo := linhaDaReserva{Code: itemCode, ItemCode: item.ItemCode, Unica: true}
			for _, outra := range linhas {
				if outra.Code != itemCode && outra.ItemCode == item.ItemCode && outra.IsActive {
					alvo.Unica = false
				}
			}
			if err := soltarReservas(ctx, r.Estoque, item.SalesOrderCode, &alvo); err != nil {
				return err
			}
		}
		return r.Pedidos.RecalculateTotals(ctx, item.SalesOrderCode)
	}
	if uc.Transacao != nil {
		return uc.Transacao(ctx, cancelar)
	}
	return cancelar(ConfirmacaoRepos{Pedidos: uc.Repo})
}

// aparaReservaDaLinha: a linha que diminuiu não pode continuar segurando o
// estoque da quantidade antiga. Com reserva acima do saldo em aberto, as
// reservas da linha são soltas e uma nova fica só com o saldo.
func aparaReservaDaLinha(ctx context.Context, r ConfirmacaoRepos, linha *entity.SalesOrderItem, aberto decimal.Decimal) error {
	linhas, err := r.Pedidos.ListItems(ctx, linha.SalesOrderCode)
	if err != nil {
		return err
	}
	alvo := linhaDaReserva{Code: linha.Code, ItemCode: linha.ItemCode, Unica: true}
	for _, outra := range linhas {
		if outra.Code != linha.Code && outra.ItemCode == linha.ItemCode && outra.IsActive {
			alvo.Unica = false
		}
	}
	ativas, err := r.Estoque.ListActiveReservations(ctx)
	if err != nil {
		return fmt.Errorf("lendo as reservas do pedido %d: %w", linha.SalesOrderCode, err)
	}
	var daLinha []*stockentity.StockReservation
	reservado := decimal.Zero
	for _, res := range ativas {
		if res.ReferenceType == stockentity.ReferenceTypeSalesOrder && res.ReferenceCode == linha.SalesOrderCode && alvo.eDaLinha(res) {
			daLinha = append(daLinha, res)
			reservado = reservado.Add(res.Quantity)
		}
	}
	if !reservado.GreaterThan(aberto) {
		return nil
	}
	for _, res := range daLinha {
		if err := r.Estoque.CancelReservation(ctx, res.ID); err != nil {
			return fmt.Errorf("soltando a reserva %d da linha %d: %w", res.ID, linha.Code, err)
		}
	}
	if !aberto.IsPositive() {
		return nil
	}
	modelo := daLinha[0]
	code := linha.Code
	if _, err := r.Estoque.CreateReservation(ctx, &stockentity.StockReservation{
		ItemCode: modelo.ItemCode, Mask: modelo.Mask, WarehouseID: modelo.WarehouseID, Quantity: aberto,
		ReferenceType: stockentity.ReferenceTypeSalesOrder, ReferenceCode: linha.SalesOrderCode, ReferenceItemCode: &code,
		ReservationDate: modelo.ReservationDate, ExpirationDate: modelo.ExpirationDate, Status: "ACTIVE",
		Notes: modelo.Notes, CreatedBy: modelo.CreatedBy,
	}); err != nil {
		return fmt.Errorf("refazendo a reserva da linha %d com %s: %w", linha.Code, aberto.String(), err)
	}
	return nil
}

// manterSeAusente: nil é "não informado" (mantém); "" limpa.
func manterSeAusente(novo, atual *string) *string {
	if novo == nil {
		return atual
	}
	if *novo == "" {
		return nil
	}
	return novo
}
