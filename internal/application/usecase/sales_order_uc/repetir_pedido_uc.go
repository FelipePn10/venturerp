package sales_order_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
)

// RepetirPedidoUseCase copia um pedido como rascunho novo (melhoria 13, base do
// B2B "repetir pedido"): mesmo cliente, endereços, condição e linhas, com o
// preço de HOJE da tabela — a criação da linha resolve o preço, nada é copiado.
type RepetirPedidoUseCase struct {
	Pedido *CreateSalesOrderUseCase
	Linha  *CreateSalesOrderItemUseCase
}

// RepetirResultado: o pedido novo e as linhas que não entraram (item inativo,
// sem preço na tabela…).
type RepetirResultado struct {
	Pedido *response.SalesOrderResponse `json:"sales_order"`
	Linhas int                          `json:"lines"`
	Falhas []string                     `json:"skipped"`
}

func (uc *RepetirPedidoUseCase) Repetir(ctx context.Context, code int64) (*RepetirResultado, error) {
	return uc.RepetirComOrigem(ctx, code, "COPY", "")
}

// RepetirComOrigem: a origem marca quem pediu (COPY = vendedor, PORTAL = o
// próprio cliente) e a observação vai junto nas notas do rascunho.
func (uc *RepetirPedidoUseCase) RepetirComOrigem(ctx context.Context, code int64, origem, observacao string) (*RepetirResultado, error) {
	orig, err := uc.Pedido.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	linhas, err := uc.Pedido.Repo.ListItems(ctx, code)
	if err != nil {
		return nil, err
	}
	hoje := time.Now().Format("2006-01-02")
	// Pedido do portal: a lista do que o cliente pediu vai inteira nas notas.
	// Se algum item não entrar (sem tabela, inativo), o vendedor ainda vê tudo.
	nota := notaDaRepeticao(orig.OrderNumber, origem, observacao)
	if origem == "PORTAL" {
		var itens []string
		for _, l := range linhas {
			if !l.IsActive || l.Status == entity.SalesOrderItemStatusCancelled || !l.RequestedQty.IsPositive() {
				continue
			}
			cod := fmt.Sprint(l.ItemCode)
			if it, err := itemresolution.PelaChave(ctx, uc.Linha.Items, l.ItemCode); err == nil && string(it.BusinessCode) != "" {
				cod = string(it.BusinessCode)
			}
			desc := l.RequestedQty.String()
			if l.SalesUOM != nil && *l.SalesUOM != "" {
				desc += " " + *l.SalesUOM
			}
			desc += " × " + cod
			if l.Mask != "" {
				desc += " (" + l.Mask + ")"
			}
			itens = append(itens, desc)
		}
		nota += ". Itens pedidos: " + strings.Join(itens, "; ")
	}
	novo, err := uc.Pedido.Execute(ctx, request.CreateSalesOrderDTO{
		Status: string(entity.SalesOrderStatusDraft), Origin: origem, EmissionDate: hoje,
		CustomerCode: orig.CustomerCode, BillingAddressCode: orig.BillingAddressCode, ShippingAddressCode: orig.ShippingAddressCode,
		RepresentativeCode: orig.RepresentativeCode, SalesDivisionCode: orig.SalesDivisionCode, TaxTypeCode: orig.TaxTypeCode,
		FinalidadeCompra: orig.FinalidadeCompra, PriceTableCode: orig.PriceTableCode, CurrencyCode: orig.CurrencyCode,
		PaymentTermCode: orig.PaymentTermCode, CarrierCode: orig.CarrierCode, FreightType: orig.FreightType,
		Notes: strPtr(nota),
	})
	if err != nil {
		return nil, err
	}
	out := &RepetirResultado{Pedido: novo, Falhas: []string{}}
	seq := 0
	for _, l := range linhas {
		if !l.IsActive || l.Status == entity.SalesOrderItemStatusCancelled || !l.RequestedQty.IsPositive() {
			continue
		}
		item, err := itemresolution.PelaChave(ctx, uc.Linha.Items, l.ItemCode)
		if err != nil {
			out.Falhas = append(out.Falhas, fmt.Sprintf("item %d: %v", l.ItemCode, err))
			continue
		}
		seq++
		if _, err := uc.Linha.Execute(ctx, request.CreateSalesOrderItemDTO{SalesOrderCode: novo.Code, Sequence: seq,
			ItemCode: request.TextCode(item.BusinessCode), Mask: l.Mask, DigitDate: hoje, NFType: l.NFType, SalesUOM: l.SalesUOM,
			WarehouseCode: l.WarehouseCode, RequestedQty: l.RequestedQty, DiscountPct: l.DiscountPct,
			// A tabela da própria linha: a capa pode não ter (pedido antigo) e a
			// linha foi vendida por ela. O preço é o de hoje nessa tabela.
			PriceTableCode: l.PriceTableCode}); err != nil {
			out.Falhas = append(out.Falhas, fmt.Sprintf("%s: %v", item.BusinessCode, err))
			seq--
			continue
		}
		out.Linhas++
	}
	if out.Linhas == 0 && origem == "PORTAL" {
		// O pedido do cliente nunca se perde: o rascunho fica, com a lista do
		// que ele pediu nas notas, e o comercial inclui os itens.
		return out, nil
	}
	if out.Linhas == 0 {
		// Sem linha nenhuma o rascunho não serve: cancela para não ficar um
		// pedido vazio na carteira.
		if err := uc.Pedido.Repo.Cancel(ctx, novo.Code, "repetição sem nenhuma linha válida", nil); err != nil {
			return nil, fmt.Errorf("cancelando o rascunho vazio da repetição: %w", err)
		}
		motivo := "o pedido não tem linhas ativas"
		if len(out.Falhas) > 0 {
			motivo = out.Falhas[0]
		}
		return nil, errorsuc.NewValidationError("nenhuma linha pôde ser repetida: " + motivo)
	}
	return out, nil
}

func strPtr(s string) *string { return &s }

func notaDaRepeticao(numero int64, origem, observacao string) string {
	n := fmt.Sprintf("Repetição do pedido %d", numero)
	if origem == "PORTAL" {
		n += " pedida pelo cliente no portal"
	}
	if observacao != "" {
		n += ". Observação do cliente: " + strings.TrimRight(observacao, ". ")
	}
	return n
}
