package customer_material_uc

// Pedido de beneficiamento: o pedido de venda do serviço, com a natureza de
// beneficiamento (VVEN). Antes a remessa guardava só um NÚMERO digitado — sem
// conferência nenhuma, qualquer número tirava o material do bloqueio, inclusive
// o de pedido inexistente, de outro cliente ou de outra empresa. E o faturamento
// não tocava o pedido: o serviço era redigitado e o pedido nunca era atendido.

import (
	"context"
	"fmt"
	"strings"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
	"github.com/shopspring/decimal"
)

// PedidosDeVenda lê o pedido de venda pelo CÓDIGO (sales_orders.code) na
// empresa da sessão. O pedido de outra empresa é "não encontrado".
type PedidosDeVenda interface {
	PedidoDeBeneficiamento(ctx context.Context, code int64, remessaID int64) (*entity.PedidoDeBeneficiamento, error)
	// VincularPedidoDaOF grava o pedido da OF de beneficiamento; nulo limpa.
	VincularPedidoDaOF(ctx context.Context, ofID int64, salesOrderCode *int64) error
}

// VincularPedidoDaOF dá cliente e pedido à OF que não veio do MRP: a aberta à mão e a
// de subconjunto, que o pegging não liga a uma linha de pedido.
func (uc *UseCase) VincularPedidoDaOF(ctx context.Context, ofID int64, salesOrderCode *int64) error {
	if uc.Pedidos == nil {
		return errorsuc.NewValidationError("a conferência de pedidos não está disponível nesta instalação: fale com o suporte")
	}
	if salesOrderCode != nil {
		pedido, err := uc.Pedidos.PedidoDeBeneficiamento(ctx, *salesOrderCode, 0)
		if err != nil {
			return err
		}
		// Pedido faturado vale: a hora da OF pode ser apontada depois da nota.
		if !pedido.Ativo || pedido.Status == entity.PedidoCancelado {
			return errorsuc.NewValidationError(fmt.Sprintf("o pedido %d está cancelado", pedido.Numero))
		}
	}
	return uc.Pedidos.VincularPedidoDaOF(ctx, ofID, salesOrderCode)
}

// MotivoSemPedido é o bloqueio automático do recebimento sem pedido. Vincular o
// pedido depois libera o material só quando o bloqueio é ESTE — bloqueio por
// divergência ou feito à mão continua de pé.
const MotivoSemPedido = "material recebido sem pedido de beneficiamento cadastrado"

// conferirPedido garante que o pedido existe nesta empresa, é do mesmo cliente
// dono do material e ainda pode ser faturado.
func (uc *UseCase) conferirPedido(ctx context.Context, code, clienteDaRemessa, remessaID int64) (*entity.PedidoDeBeneficiamento, error) {
	if code <= 0 {
		return nil, errorsuc.NewValidationError("informe o pedido de beneficiamento")
	}
	if uc.Pedidos == nil {
		return nil, nil
	}
	pedido, err := uc.Pedidos.PedidoDeBeneficiamento(ctx, code, remessaID)
	if err != nil {
		if errorsuc.EhNaoEncontrado(err) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("pedido de venda %d não encontrado nesta empresa", code))
		}
		return nil, err
	}
	if !pedido.Ativo || pedido.Status == entity.PedidoCancelado {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o pedido %d está cancelado", pedido.Numero))
	}
	if pedido.Status == entity.PedidoFaturado {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o pedido %d já foi todo faturado", pedido.Numero))
	}
	if pedido.CustomerCode != clienteDaRemessa {
		return nil, errorsuc.NewValidationError(fmt.Sprintf(
			"o pedido %d é de outro cliente: o material só pode ser vinculado a pedido do cliente que o enviou", pedido.Numero))
	}
	return pedido, nil
}

// VincularPedido liga a remessa ao pedido depois do recebimento — o caso do
// material que chega antes de o pedido ser implantado. Se a remessa estava
// bloqueada SÓ por falta de pedido, o material é liberado na mesma operação.
func (uc *UseCase) VincularPedido(ctx context.Context, id, salesOrderCode int64, usuario string) (*entity.Remessa, error) {
	if strings.TrimSpace(usuario) == "" {
		return nil, errorsuc.NewValidationError("sua sessão não identificou o usuário: entre no sistema novamente")
	}
	remessa, err := uc.Repo.BuscarPorID(ctx, id)
	if err != nil {
		return nil, err
	}
	if remessa.Status == entity.StatusEncerrada || remessa.Status == entity.StatusCancelada {
		return nil, errorsuc.NewValidationError("remessa encerrada não muda de pedido")
	}
	// A nota já aponta para a linha do pedido atual: autorizada depois da troca,
	// atenderia um pedido que a remessa não tem mais.
	mudaDePedido := remessa.SalesOrderCode == nil || *remessa.SalesOrderCode != salesOrderCode
	if remessa.NotaPendente != nil && mudaDePedido {
		return nil, errorsuc.NewConflictError(fmt.Sprintf(
			"a nota %d/%s desta remessa ainda não foi autorizada: descarte-a antes de trocar o pedido",
			remessa.NotaPendente.Numero, remessa.NotaPendente.Serie))
	}
	if _, err := uc.conferirPedido(ctx, salesOrderCode, remessa.CustomerCode, remessa.ID); err != nil {
		return nil, err
	}
	desbloquear := remessa.Blocked && remessa.BlockReason != nil && *remessa.BlockReason == MotivoSemPedido
	if err := uc.Repo.VincularPedido(ctx, id, salesOrderCode, desbloquear, usuario); err != nil {
		return nil, err
	}
	return uc.Repo.BuscarPorID(ctx, id)
}

// servicoDoPedido completa o serviço da nota com a linha do pedido.
//
// Remessa sem pedido (legado, ou ambiente sem leitor de pedidos) fatura como
// antes, com o serviço informado. Com pedido, a linha é a escolhida ou — havendo
// uma só com saldo — a única; o que a tela não mandou vem do pedido, e a
// quantidade não pode passar do saldo da linha.
func (uc *UseCase) servicoDoPedido(ctx context.Context, remessa *entity.Remessa, servico ServicoFaturado) (ServicoFaturado, error) {
	if remessa.SalesOrderCode == nil || uc.Pedidos == nil {
		if servico.SalesOrderItemCode != nil {
			return servico, errorsuc.NewValidationError("a remessa não tem pedido de beneficiamento vinculado: vincule o pedido antes de escolher a linha")
		}
		return servico, nil
	}
	pedido, err := uc.conferirPedido(ctx, *remessa.SalesOrderCode, remessa.CustomerCode, remessa.ID)
	if err != nil {
		return servico, err
	}
	linha, err := escolherLinha(pedido, servico.SalesOrderItemCode)
	if err != nil {
		return servico, err
	}

	if strings.TrimSpace(servico.CodigoItem) == "" {
		servico.CodigoItem = linha.ItemBusinessCode
	}
	if strings.TrimSpace(servico.Descricao) == "" {
		servico.Descricao = linha.Descricao
	}
	if strings.TrimSpace(servico.Unidade) == "" {
		servico.Unidade = linha.Unidade
	}
	if strings.TrimSpace(servico.Quantidade) == "" {
		servico.Quantidade = linha.Saldo().String()
	}
	if strings.TrimSpace(servico.ValorUnit) == "" {
		servico.ValorUnit = linha.PrecoUnitario.String()
	}
	qtd, err := decimal.NewFromString(strings.TrimSpace(servico.Quantidade))
	if err == nil && qtd.GreaterThan(linha.Saldo()) {
		return servico, errorsuc.NewValidationError(fmt.Sprintf(
			"a quantidade do serviço (%s) passa do saldo da linha %d do pedido %d (%s)",
			qtd.String(), linha.Sequencia, pedido.Numero, linha.Saldo().String()))
	}
	code := linha.Code
	servico.SalesOrderItemCode = &code
	return servico, nil
}

func escolherLinha(pedido *entity.PedidoDeBeneficiamento, escolhida *int64) (entity.LinhaDoPedido, error) {
	if escolhida != nil {
		linha, ok := pedido.Linha(*escolhida)
		if !ok {
			return linha, errorsuc.NewValidationError(fmt.Sprintf("a linha escolhida não é do pedido %d", pedido.Numero))
		}
		if !linha.Saldo().IsPositive() {
			return linha, errorsuc.NewValidationError(fmt.Sprintf("a linha %d do pedido %d não tem saldo a faturar", linha.Sequencia, pedido.Numero))
		}
		return linha, nil
	}
	abertas := pedido.LinhasComSaldo()
	switch len(abertas) {
	case 0:
		return entity.LinhaDoPedido{}, errorsuc.NewValidationError(fmt.Sprintf("o pedido %d não tem saldo a faturar", pedido.Numero))
	case 1:
		return abertas[0], nil
	default:
		return entity.LinhaDoPedido{}, errorsuc.NewValidationError(fmt.Sprintf(
			"o pedido %d tem %d linhas com saldo: escolha qual serviço esta nota fatura", pedido.Numero, len(abertas)))
	}
}

// PropostaDeFaturamento é o que a tela mostra ao abrir o faturamento: o pedido
// com as linhas e saldos, o serviço sugerido e a devolução do saldo inteiro de
// cada linha da remessa — o caso comum no fechamento do pedido.
type PropostaDeFaturamento struct {
	Remessa    *entity.Remessa
	Pedido     *entity.PedidoDeBeneficiamento
	Servico    ServicoFaturado
	Devolucoes []DevolucaoDeMaterial
	// Aviso explica por que o serviço não pôde ser sugerido (pedido com várias
	// linhas, sem saldo, remessa sem pedido).
	Aviso string
}

func (uc *UseCase) PropostaDeFaturamento(ctx context.Context, id int64) (*PropostaDeFaturamento, error) {
	remessa, err := uc.Repo.BuscarPorID(ctx, id)
	if err != nil {
		return nil, err
	}
	proposta := &PropostaDeFaturamento{Remessa: remessa}
	for _, item := range remessa.Itens {
		if item.BalanceQty.IsPositive() {
			proposta.Devolucoes = append(proposta.Devolucoes, DevolucaoDeMaterial{
				RemittanceItemID: item.ID, Quantidade: item.BalanceQty.String(), Tipo: entity.MovimentoRetorno,
			})
		}
	}
	if remessa.SalesOrderCode == nil || uc.Pedidos == nil {
		proposta.Aviso = "a remessa não tem pedido de beneficiamento: informe o serviço ou vincule o pedido"
		return proposta, nil
	}
	pedido, err := uc.Pedidos.PedidoDeBeneficiamento(ctx, *remessa.SalesOrderCode, remessa.ID)
	if err != nil && !errorsuc.EhNaoEncontrado(err) {
		return nil, err
	}
	proposta.Pedido = pedido
	servico, errServico := uc.servicoDoPedido(ctx, remessa, ServicoFaturado{})
	if errServico != nil {
		if _, ok := errorsuc.AsValidation(errServico); !ok {
			return nil, errServico
		}
		proposta.Aviso = errServico.Error()
		return proposta, nil
	}
	proposta.Servico = servico
	return proposta, nil
}
