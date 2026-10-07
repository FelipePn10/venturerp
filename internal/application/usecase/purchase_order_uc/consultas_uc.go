package purchase_order_uc

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

// ─── Acompanhamento de entregas ─────────────────────────────────────────────

// Situações do acompanhamento, do jeito que o comprador filtra.
const (
	AcompanhamentoAtrasadas   = "ATRASADAS"
	AcompanhamentoProximos7   = "PROXIMOS_7"
	AcompanhamentoSemPromessa = "SEM_PROMESSA"
	AcompanhamentoTodas       = "TODAS"
)

type AcompanhamentoUseCase struct {
	Repo      repository.PurchaseOrderRepository
	Consultas ConsultasCompras
	Auth      ports.AuthService
}

// Linhas devolve o que os fornecedores ainda devem entregar, com os dias de
// atraso contados pela data prometida (ou, sem promessa, pela pedida).
func (uc *AcompanhamentoUseCase) Linhas(ctx context.Context, situacao string, fornecedor *int64, hoje time.Time) ([]LinhaEmAberto, error) {
	if !uc.Auth.CanListPurchaseOrders(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	situacao = strings.ToUpper(strings.TrimSpace(situacao))
	if situacao == "" {
		situacao = AcompanhamentoTodas
	}
	switch situacao {
	case AcompanhamentoAtrasadas, AcompanhamentoProximos7, AcompanhamentoSemPromessa, AcompanhamentoTodas:
	default:
		return nil, errorsuc.NewValidationError("situação do acompanhamento inválida: use ATRASADAS, PROXIMOS_7, SEM_PROMESSA ou TODAS")
	}
	linhas, err := uc.Consultas.LinhasEmAberto(ctx, FiltroAcompanhamento{SupplierCode: fornecedor})
	if err != nil {
		return nil, err
	}
	limite7 := hoje.AddDate(0, 0, 7)
	out := make([]LinhaEmAberto, 0, len(linhas))
	for _, l := range linhas {
		var prevista *time.Time
		switch {
		case l.PromisedDate != nil:
			prevista = l.PromisedDate
		case l.DeliveryDate != nil:
			prevista = l.DeliveryDate
		}
		l.DataPrevista = prevista
		if prevista != nil {
			l.DiasAtraso = entity.DiasDeAtraso(*prevista, hoje)
		}
		manter := true
		switch situacao {
		case AcompanhamentoAtrasadas:
			manter = l.DiasAtraso > 0
		case AcompanhamentoProximos7:
			manter = prevista != nil && l.DiasAtraso == 0 && !prevista.After(limite7)
		case AcompanhamentoSemPromessa:
			manter = l.PromisedDate == nil
		}
		if manter {
			out = append(out, l)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].DiasAtraso != out[j].DiasAtraso {
			return out[i].DiasAtraso > out[j].DiasAtraso
		}
		a, b := out[i].DataPrevista, out[j].DataPrevista
		switch {
		case a == nil:
			return false
		case b == nil:
			return true
		}
		return a.Before(*b)
	})
	return out, nil
}

// FollowupDTO é o que o comprador anota depois de falar com o fornecedor.
type FollowupDTO struct {
	DataPrometida string `json:"data_prometida"`
	Contato       string `json:"contato"`
	Observacao    string `json:"observacao"`
}

// RegistrarFollowup grava o contato e, havendo data, a nova promessa na linha.
// Vale para pedido aprovado: a data prometida não muda o que foi comprado.
func (uc *AcompanhamentoUseCase) RegistrarFollowup(ctx context.Context, pedido, linha int64, dto FollowupDTO) (*Followup, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	po, _, it, err := linhaDoPedido(ctx, uc.Repo, pedido, linha)
	if err != nil {
		return nil, err
	}
	switch po.Status {
	case entity.PurchaseOrderStatusAPPROVED, entity.PurchaseOrderStatusPARTIAL:
	default:
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o pedido %d está %s; o acompanhamento de entrega vale para pedido aprovado", po.Code, entity.RotuloSituacao(po.Status)))
	}
	if it.Status == entity.PurchaseOrderItemStatusCANCELLED || it.Saldo() <= 0.0001 {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a linha %d não tem saldo a receber", it.Sequence))
	}
	data, err := dataOpcional(&dto.DataPrometida, "prometida")
	if err != nil {
		return nil, err
	}
	contato, obs := strings.TrimSpace(dto.Contato), strings.TrimSpace(dto.Observacao)
	if data == nil && contato == "" && obs == "" {
		return nil, errorsuc.NewValidationError("informe a data prometida, o contato ou uma observação")
	}
	if len(contato) > 150 {
		return nil, errorsuc.NewValidationError("o contato tem no máximo 150 caracteres")
	}
	por, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	return uc.Consultas.RegistrarFollowup(ctx, Followup{PurchaseOrderCode: po.Code, LineCode: it.Code, DataPrometida: data, Contato: contato, Observacao: obs}, por)
}

// Followups lista o histórico de contatos da linha, do mais recente.
func (uc *AcompanhamentoUseCase) Followups(ctx context.Context, pedido, linha int64) ([]Followup, error) {
	if !uc.Auth.CanGetPurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if _, _, _, err := linhaDoPedidoInclusiveCancelado(ctx, uc.Repo, pedido, linha); err != nil {
		return nil, err
	}
	return uc.Consultas.Followups(ctx, linha)
}

// linhaDoPedidoInclusiveCancelado confere o vínculo linha × pedido sem exigir
// pedido ativo (consulta de histórico).
func linhaDoPedidoInclusiveCancelado(ctx context.Context, repo repository.PurchaseOrderRepository, pedido, linha int64) (*entity.PurchaseOrder, []*entity.PurchaseOrderItem, *entity.PurchaseOrderItem, error) {
	po, err := repo.GetByCode(ctx, pedido)
	if err != nil {
		return nil, nil, nil, err
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

// ─── Histórico de preço do item ─────────────────────────────────────────────

// HistoricoPreco é o que o comprador consulta antes de fechar o preço.
type HistoricoPreco struct {
	ItemCode      int64               `json:"item_code"`
	Ultima        *CompraDoItem       `json:"ultima,omitempty"`
	Compras       []CompraDoItem      `json:"compras"`
	QtdCompras12m int                 `json:"qtd_compras_12m"`
	Media12m      *float64            `json:"custo_medio_12m,omitempty"` // ponderado pela quantidade, por unidade de estoque
	Minimo12m     *float64            `json:"custo_minimo_12m,omitempty"`
	Maximo12m     *float64            `json:"custo_maximo_12m,omitempty"`
	UltimoPedido  *UltimoPedidoDoItem `json:"ultimo_pedido,omitempty"`
}

type HistoricoPrecoUseCase struct {
	Consultas ConsultasCompras
	Auth      ports.AuthService
}

// Execute junta as compras dos últimos 12 meses (notas aprovadas) e o último
// pedido do item. O custo médio é por unidade de ESTOQUE — é o que permite
// comparar fornecedores que vendem em caixa com os que vendem em unidade.
func (uc *HistoricoPrecoUseCase) Execute(ctx context.Context, itemCode int64, hoje time.Time) (*HistoricoPreco, error) {
	if !uc.Auth.CanListPurchaseOrders(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if itemCode <= 0 {
		return nil, errorsuc.NewValidationError("informe o item")
	}
	desde := hoje.AddDate(-1, 0, 0)
	compras, err := uc.Consultas.ComprasDoItem(ctx, itemCode, desde, 200)
	if err != nil {
		return nil, err
	}
	out := &HistoricoPreco{ItemCode: itemCode, Compras: []CompraDoItem{}}
	custo, qtd := decimal.Zero, decimal.Zero
	for i, c := range compras {
		if i == 0 {
			ult := c
			out.Ultima = &ult
		}
		if i < 10 {
			out.Compras = append(out.Compras, c)
		}
		out.QtdCompras12m++
		if c.QtdEstoque > 0 {
			custo = custo.Add(decimal.NewFromFloat(c.CustoAquisicao))
			qtd = qtd.Add(decimal.NewFromFloat(c.QtdEstoque))
			u := c.CustoEstoque
			if out.Minimo12m == nil || u < *out.Minimo12m {
				v := u
				out.Minimo12m = &v
			}
			if out.Maximo12m == nil || u > *out.Maximo12m {
				v := u
				out.Maximo12m = &v
			}
		}
	}
	if qtd.IsPositive() {
		m, _ := custo.Div(qtd).Round(6).Float64()
		out.Media12m = &m
	}
	if out.UltimoPedido, err = uc.Consultas.UltimoPedidoDoItem(ctx, itemCode); err != nil {
		return nil, err
	}
	return out, nil
}

// ─── Notas que atenderam o pedido ───────────────────────────────────────────

type NotasDoPedidoUseCase struct {
	Repo      repository.PurchaseOrderRepository
	Consultas ConsultasCompras
	Auth      ports.AuthService
}

func (uc *NotasDoPedidoUseCase) Execute(ctx context.Context, code int64) ([]NotaDaLinha, error) {
	if !uc.Auth.CanGetPurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if _, err := uc.Repo.GetByCode(ctx, code); err != nil {
		return nil, err
	}
	return uc.Consultas.NotasDoPedido(ctx, code)
}

// ─── Previsão de pagamentos ─────────────────────────────────────────────────

// ParcelaPrevista é um pagamento que o pedido ainda vai gerar.
type ParcelaPrevista struct {
	PurchaseOrderCode int64     `json:"purchase_order_code"`
	OrderNumber       int64     `json:"order_number"`
	SupplierCode      *int64    `json:"supplier_code,omitempty"`
	Numero            int       `json:"numero"`
	Vencimento        time.Time `json:"vencimento"`
	Valor             float64   `json:"valor"`
	Estimada          bool      `json:"estimada"`
	Condicao          string    `json:"condicao"`
	Descricao         string    `json:"descricao,omitempty"`
	EntregaPrevista   time.Time `json:"entrega_prevista"`
}

// PrevisaoPedido é a previsão de um pedido com os avisos de como foi montada.
type PrevisaoPedido struct {
	Parcelas []ParcelaPrevista `json:"parcelas"`
	Total    float64           `json:"total"`
	Avisos   []string          `json:"avisos,omitempty"`
}

type PrevisaoPagamentosUseCase struct {
	Repo      repository.PurchaseOrderRepository
	Condicoes CondicoesDePagamento
	Auth      ports.AuthService
}

// DoPedido calcula o que falta pagar de um pedido, pela condição de pagamento
// dele, a partir da data em que cada parte deve chegar.
func (uc *PrevisaoPagamentosUseCase) DoPedido(ctx context.Context, code int64) (*PrevisaoPedido, error) {
	if !uc.Auth.CanGetPurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	po, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	itens, err := uc.Repo.ListItems(ctx, code)
	if err != nil {
		return nil, err
	}
	return uc.prever(ctx, po, itens, map[int64]*customerentity.PaymentCondition{})
}

// Geral soma a previsão dos pedidos aprovados (os rascunhos ainda não são
// compromisso) com vencimento no período.
func (uc *PrevisaoPagamentosUseCase) Geral(ctx context.Context, de, ate time.Time) (*PrevisaoPedido, error) {
	if !uc.Auth.CanListPurchaseOrders(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if ate.Before(de) {
		return nil, errorsuc.NewValidationError("a data final é anterior à inicial")
	}
	out := &PrevisaoPedido{Parcelas: []ParcelaPrevista{}}
	cache := map[int64]*customerentity.PaymentCondition{}
	total := decimal.Zero
	for _, s := range []entity.PurchaseOrderStatus{entity.PurchaseOrderStatusAPPROVED, entity.PurchaseOrderStatusPARTIAL} {
		pedidos, err := uc.Repo.ListByStatus(ctx, s)
		if err != nil {
			return nil, err
		}
		for _, po := range pedidos {
			itens, err := uc.Repo.ListItems(ctx, po.Code)
			if err != nil {
				return nil, err
			}
			p, err := uc.prever(ctx, po, itens, cache)
			if err != nil {
				return nil, err
			}
			out.Avisos = append(out.Avisos, p.Avisos...)
			for _, parc := range p.Parcelas {
				d := dataSemHora(parc.Vencimento)
				if d.Before(dataSemHora(de)) || d.After(dataSemHora(ate)) {
					continue
				}
				out.Parcelas = append(out.Parcelas, parc)
				total = total.Add(decimal.NewFromFloat(parc.Valor))
			}
		}
	}
	sort.SliceStable(out.Parcelas, func(i, j int) bool { return out.Parcelas[i].Vencimento.Before(out.Parcelas[j].Vencimento) })
	out.Total, _ = total.Round(2).Float64()
	return out, nil
}

func dataSemHora(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func (uc *PrevisaoPagamentosUseCase) prever(ctx context.Context, po *entity.PurchaseOrder, itens []*entity.PurchaseOrderItem, cache map[int64]*customerentity.PaymentCondition) (*PrevisaoPedido, error) {
	out := &PrevisaoPedido{Parcelas: []ParcelaPrevista{}}
	if po.Status == entity.PurchaseOrderStatusCANCELLED || !po.IsActive {
		return out, nil
	}
	grupos := entity.ValoresAFaturar(po, itens)
	if len(grupos) == 0 {
		return out, nil
	}
	var cond *customerentity.PaymentCondition
	nomeCond := "sem condição de pagamento: à vista na entrega"
	if po.PaymentTermCode != nil && *po.PaymentTermCode > 0 && uc.Condicoes != nil {
		c, ok := cache[*po.PaymentTermCode]
		if !ok {
			var err error
			c, err = uc.Condicoes.GetPaymentConditionByCode(ctx, *po.PaymentTermCode)
			if err != nil {
				c = nil
			}
			cache[*po.PaymentTermCode] = c
		}
		if c == nil {
			out.Avisos = append(out.Avisos, fmt.Sprintf("pedido %d: condição de pagamento %d não encontrada; previsto à vista na entrega", po.Code, *po.PaymentTermCode))
		} else {
			cond = c
			nomeCond = c.Description
		}
	} else {
		out.Avisos = append(out.Avisos, fmt.Sprintf("pedido %d sem condição de pagamento; previsto à vista na entrega", po.Code))
	}
	total := decimal.Zero
	numero := 0
	for _, g := range grupos {
		entrega := g.Entrega
		var parcelas []customerentity.ParcelaCalculada
		if cond != nil {
			var err error
			parcelas, err = customerentity.CalcularPlano(cond, g.Valor, customerentity.DatasBase{Emissao: po.EmissionDate, Entrega: &entrega, Faturamento: &entrega})
			if err != nil {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("pedido %d: %v", po.Code, err))
			}
		} else {
			parcelas = []customerentity.ParcelaCalculada{{Numero: 1, Valor: g.Valor, Vencimento: entrega, Descricao: "na entrega"}}
		}
		for _, p := range parcelas {
			numero++
			v, _ := p.Valor.Float64()
			out.Parcelas = append(out.Parcelas, ParcelaPrevista{
				PurchaseOrderCode: po.Code, OrderNumber: po.OrderNumber, SupplierCode: po.SupplierCode,
				Numero: numero, Vencimento: p.Vencimento, Valor: v, Estimada: g.Estimada || p.Estimado,
				Condicao: nomeCond, Descricao: p.Descricao, EntregaPrevista: entrega,
			})
			total = total.Add(p.Valor)
		}
	}
	out.Total, _ = total.Round(2).Float64()
	return out, nil
}
