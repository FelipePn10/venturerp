package fiscal_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	financialEntity "github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	financialRepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	salesentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

// EfeitosRepos são os repositórios ligados à transação dos efeitos da NF-e.
type EfeitosRepos struct {
	Financeiro  financialRepo.FinancialRepository
	Estoque     stockrepo.StockRepository
	Pedidos     salesrepo.SalesOrderRepository
	Faturamento repository.SalesOrderInvoicingRepository
	Pendencias  repository.PendenciasNFeRepository
	// Series: a série que sai na nota vai para o cliente (e volta no cancelamento).
	Series SeriesNaNota
	// Consignacao: remessa, venda consignada e transferência entre CNPJs.
	Consignacao stockrepo.ConsignacaoEstoque
}

// SeriesNaNota: o efeito da nota sobre o número de série (pós-venda 3.1).
type SeriesNaNota interface {
	SaidaPelaNota(ctx context.Context, fiscalExitID int64, cliente *int64, emissao time.Time, por *uuid.UUID) error
	EstornoDaNota(ctx context.Context, fiscalExitID int64, por *uuid.UUID) error
}

// EmTransacaoEfeitos roda fn numa transação; erro devolvido desfaz tudo.
type EmTransacaoEfeitos func(ctx context.Context, fn func(EfeitosRepos) error) error

// itemDeposito lê o cadastro do item para o depósito padrão da baixa.
type itemDeposito interface {
	FindItemByCode(ctx context.Context, code valueobject.ItemCode) (*itementity.Item, error)
}

// EfeitosNFe aplica no ERP o que uma NF-e autorizada (ou cancelada) na SEFAZ
// causa: baixa de estoque, consumo das reservas, títulos a receber e situação
// do pedido. A nota não pode ser desfeita, então os efeitos são uma PENDÊNCIA
// registrada antes de rodar: rodam numa transação só, que também a marca como
// resolvida — ou entram todos, ou nenhum, e a pendência fica visível com o
// motivo até o reprocesso (automático ou pelo botão) dar certo.
type EfeitosNFe struct {
	Fiscal     repository.FiscalRepository
	Customers  customerrepo.CustomerRepository
	Orders     salesrepo.SalesOrderRepository
	Itens      itemDeposito
	Pendencias repository.PendenciasNFeRepository
	Transacao  EmTransacaoEfeitos
	// GeraTitulos: com o financeiro configurado, a autorização gera os títulos.
	GeraTitulos bool
	// TiposNota diz o que o tipo de nota escolhido faz: título, estoque e pedido.
	TiposNota LeitorEfeitosTipoNota
	// Conversor leva a quantidade da nota (unidade comercial) para a unidade de
	// estoque do item antes da baixa.
	Conversor ports.UOMConverter
	// Consignacao confere, antes de transmitir, o saldo do cliente na venda consignada.
	Consignacao stockrepo.ConsignacaoEstoque
}

// Registrar abre a pendência da etapa (antes de tentar aplicar os efeitos).
func (e *EfeitosNFe) Registrar(ctx context.Context, exitID int64, etapa string, por uuid.UUID) error {
	return e.Pendencias.Registrar(ctx, exitID, etapa, por)
}

// Executar aplica os efeitos da etapa. Falhando, guarda o motivo na pendência
// (que continua aberta para o reprocesso) e devolve o erro.
func (e *EfeitosNFe) Executar(ctx context.Context, exitID int64, etapa string, userID uuid.UUID) error {
	var err error
	switch etapa {
	case entity.PendenciaAutorizacao:
		err = e.autorizacao(ctx, exitID, userID)
	case entity.PendenciaCancelamento:
		err = e.cancelamento(ctx, exitID)
	default:
		return fmt.Errorf("etapa de pendência desconhecida: %s", etapa)
	}
	if err != nil {
		if errFalha := e.Pendencias.RegistrarFalha(ctx, exitID, etapa, err.Error()); errFalha != nil {
			return fmt.Errorf("%w; e o motivo não pôde ser guardado na pendência: %v", err, errFalha)
		}
		return err
	}
	return nil
}

// Reprocessar tenta de novo as pendências abertas da nota (botão da tela).
func (e *EfeitosNFe) Reprocessar(ctx context.Context, exitID int64, userID uuid.UUID) ([]entity.PendenciaNFe, error) {
	pend, err := e.Pendencias.DaNota(ctx, exitID)
	if err != nil {
		return nil, err
	}
	var primeiro error
	for _, p := range pend {
		if p.Situacao != "PENDENTE" {
			continue
		}
		if err := e.Executar(ctx, exitID, p.Etapa, userID); err != nil && primeiro == nil {
			primeiro = err
		}
	}
	if primeiro != nil {
		return nil, errorsuc.NewValidationError("os efeitos da nota continuam pendentes: " + primeiro.Error())
	}
	return e.Pendencias.DaNota(ctx, exitID)
}

// DaNota lista as pendências (abertas e resolvidas) da nota.
func (e *EfeitosNFe) DaNota(ctx context.Context, exitID int64) ([]entity.PendenciaNFe, error) {
	return e.Pendencias.DaNota(ctx, exitID)
}

// ProcessarVencidas é o reprocesso automático: cada pendência roda na sessão
// da empresa dona (montada por quem chama — o agendador), em nome de quem a
// originou.
func (e *EfeitosNFe) ProcessarVencidas(ctx context.Context, limite int, sessao func(context.Context, entity.PendenciaVencida) context.Context) (resolvidas, falhas int, err error) {
	fila, err := e.Pendencias.Vencidas(ctx, limite)
	if err != nil {
		return 0, 0, err
	}
	for _, p := range fila {
		ator := uuid.Nil
		if p.CriadoPor != nil {
			ator = *p.CriadoPor
		}
		if errExec := e.Executar(sessao(ctx, p), p.FiscalExitID, p.Etapa, ator); errExec != nil {
			falhas++
			continue
		}
		resolvidas++
	}
	return resolvidas, falhas, nil
}

// PreConferirDepositos confere, ANTES de transmitir, que cada item que movimenta
// estoque tem depósito de saída: o da linha do pedido ou, sem ele, o depósito
// padrão do item. Sem nenhum dos dois a autorização é recusada — depois de
// autorizada a nota não volta, e a baixa ficaria sem depósito.
func (e *EfeitosNFe) PreConferirDepositos(ctx context.Context, exit *entity.FiscalExit, items []*entity.FiscalExitItem) error {
	efeitos, err := efeitosDaNota(ctx, e.TiposNota, exit)
	if err != nil {
		return err
	}
	if !efeitos.MovimentaEstoque {
		return nil
	}
	if efeitos.Tipo == entity.TipoNotaVendaConsignacao || efeitos.Tipo == entity.TipoNotaRemessaConsignacao {
		if err := e.conferirConsignacao(ctx, exit, items, efeitos); err != nil {
			return err
		}
	}
	if efeitos.Tipo == entity.TipoNotaVendaConsignacao {
		return nil // sai do almoxarifado de consignação, não do depósito do item
	}
	if efeitos.Tipo == entity.TipoNotaTransferencia && e.Consignacao != nil && exit.CnpjDestinatario != nil {
		irma, err := e.Consignacao.EmpresaIrma(ctx, *exit.CnpjDestinatario)
		if err != nil {
			return err
		}
		if irma > 0 {
			if err := e.Consignacao.ConferirTransitoEntreCNPJs(ctx); err != nil {
				return err
			}
		}
	}
	_, err = e.depositos(ctx, exit, items)
	return err
}

// conferirConsignacao barra antes da SEFAZ o que a baixa recusaria depois: nota
// autorizada não volta.
func (e *EfeitosNFe) conferirConsignacao(ctx context.Context, exit *entity.FiscalExit, items []*entity.FiscalExitItem, efeitos entity.EfeitosDoTipoNota) error {
	if e.Consignacao == nil {
		return errorsuc.NewValidationError("consignação indisponível nesta instalação")
	}
	if exit.CustomerCode == nil {
		return errorsuc.NewValidationError("a nota de consignação precisa de um cliente cadastrado")
	}
	if _, err := e.Consignacao.AlmoxDeConsignacao(ctx); err != nil {
		return err
	}
	if efeitos.Tipo != entity.TipoNotaVendaConsignacao {
		return nil
	}
	quantidades, err := e.quantidadesEmEstoque(ctx, items)
	if err != nil {
		return err
	}
	mascaras, err := e.mascarasDoPedido(ctx, exit)
	if err != nil {
		return err
	}
	pedido := map[[3]string]decimal.Decimal{}
	for _, it := range items {
		if it.ItemCode == nil {
			continue
		}
		q := it.Quantity
		if v, ok := quantidades[it.ID]; ok {
			q = v
		}
		mask := mascaraDaLinha(it, mascaras)
		k := [3]string{fmt.Sprint(*it.ItemCode), mask, ""}
		if it.Lot != nil {
			k[2] = *it.Lot
		}
		pedido[k] = pedido[k].Add(q)
		saldo, err := e.Consignacao.SaldoConsignado(ctx, *exit.CustomerCode, *it.ItemCode, mask, it.Lot)
		if err != nil {
			return err
		}
		if pedido[k].GreaterThan(saldo) {
			return errorsuc.NewValidationError(fmt.Sprintf("o item %s tem só %s em consignação com este cliente", descricaoItem(it), stockentity.QuantidadeBR(saldo)))
		}
	}
	return nil
}

// aplicarEstoque escolhe o caminho do estoque pelo tipo de nota. A remessa e a
// transferência a outro estabelecimento não saem do estoque: o material continua nosso.
func aplicarEstoque(ctx context.Context, r EfeitosRepos, exit *entity.FiscalExit, items []*entity.FiscalExitItem, efeitos entity.EfeitosDoTipoNota, depositos map[int64]int64, quantidades map[int64]decimal.Decimal, mascaras map[int64]string, userID uuid.UUID) error {
	qtd := func(it *entity.FiscalExitItem) decimal.Decimal {
		if q, ok := quantidades[it.ID]; ok {
			return q
		}
		return it.Quantity
	}
	documento := fmt.Sprintf("%d/%s", exit.NumeroNF, exit.Serie)
	switch efeitos.Tipo {
	case entity.TipoNotaRemessaConsignacao:
		if r.Consignacao == nil || exit.CustomerCode == nil {
			return errorsuc.NewValidationError("a remessa em consignação precisa de um cliente cadastrado")
		}
		for _, it := range items {
			if it.ItemCode == nil {
				continue
			}
			valor := decimal.Zero
			if q := qtd(it); q.IsPositive() {
				valor = it.TotalPrice.DivRound(q, stockentity.CasasCusto)
			}
			// Pedido já separado: o material sai de onde a onda pôs, e a onda fecha.
			saidas, err := saidasDaSeparacao(ctx, r.Estoque, exit, it, stockentity.StockMovement{ItemCode: *it.ItemCode, Mask: mascaraDaLinha(it, mascaras),
				WarehouseID: depositos[*it.ItemCode], Quantity: qtd(it), Lot: it.Lot, Address: it.Address})
			if err != nil {
				return err
			}
			for _, s := range saidas {
				if err := r.Consignacao.RegistrarRemessa(ctx, stockentity.RemessaConsignada{NotaSaida: exit.ID, Cliente: *exit.CustomerCode, ItemCode: *it.ItemCode,
					Mask: s.Mask, Lote: s.Lot, Endereco: s.Address, LinhaDaOnda: s.LinhaDaOnda, AlmoxOrigem: s.WarehouseID, Quantidade: s.Quantity,
					ValorUnitario: valor, Documento: documento, Emissao: exit.DataEmissao, Ator: userID}); err != nil {
					return fmt.Errorf("enviando o item %s à consignação: %w", descricaoItem(it), err)
				}
			}
		}
		return consumirReservasDoPedido(ctx, r.Estoque, exit, faturadoDaNota(items, qtd))
	case entity.TipoNotaVendaConsignacao:
		if r.Consignacao == nil || exit.CustomerCode == nil {
			return errorsuc.NewValidationError("a venda em consignação precisa de um cliente cadastrado")
		}
		almox, err := r.Consignacao.AlmoxDeConsignacao(ctx)
		if err != nil {
			return err
		}
		daConsignacao := map[int64]int64{}
		for _, it := range items {
			if it.ItemCode != nil {
				daConsignacao[*it.ItemCode] = almox
			}
		}
		// Linha a linha: a baixa de uma muda o saldo que a seguinte reparte pelos lotes.
		for _, it := range items {
			if it.ItemCode == nil {
				continue
			}
			partes, qtds, err := partesDaVendaConsignada(ctx, r.Consignacao, *exit.CustomerCode, it, mascaraDaLinha(it, mascaras), qtd(it))
			if err != nil {
				return err
			}
			if err := baixarEstoque(ctx, r.Estoque, exit, partes, daConsignacao, qtds, mascaras, userID); err != nil {
				return err
			}
			for _, p := range partes {
				if err := r.Consignacao.RegistrarVendaConsignada(ctx, stockentity.VendaConsignada{NotaSaida: exit.ID, Cliente: *exit.CustomerCode, ItemCode: *it.ItemCode,
					Mask: mascaraDaLinha(it, mascaras), Lote: p.Lot, Quantidade: qtds[p.ID], Documento: documento, Emissao: exit.DataEmissao, Ator: userID}); err != nil {
					return err
				}
			}
		}
		return nil
	case entity.TipoNotaTransferencia:
		if r.Consignacao != nil && exit.CnpjDestinatario != nil && exit.ChaveAcesso != nil {
			irma, err := r.Consignacao.EmpresaIrma(ctx, *exit.CnpjDestinatario)
			if err != nil {
				return err
			}
			if irma > 0 {
				envio := stockentity.EnvioEntreCNPJs{NotaSaida: exit.ID, Destino: irma, Chave: *exit.ChaveAcesso, Ator: userID}
				// O envio grava tudo no fim: duas linhas do item não podem levar a mesma separação.
				usado := map[int64]decimal.Decimal{}
				for _, it := range items {
					if it.ItemCode == nil {
						continue
					}
					saidas, err := saidasDaSeparacao(ctx, r.Estoque, exit, it, stockentity.StockMovement{ItemCode: *it.ItemCode, Mask: mascaraDaLinha(it, mascaras),
						WarehouseID: depositos[*it.ItemCode], Quantity: qtd(it), Lot: it.Lot, Address: it.Address}, usado)
					if err != nil {
						return err
					}
					for _, s := range saidas {
						envio.Linhas = append(envio.Linhas, stockentity.LinhaEntreCNPJs{ItemCode: *it.ItemCode, Mask: s.Mask,
							Lote: s.Lot, Endereco: s.Address, LinhaDaOnda: s.LinhaDaOnda, AlmoxDe: s.WarehouseID, Quantidade: s.Quantity})
					}
				}
				if err := r.Consignacao.EnviarEntreCNPJs(ctx, envio); err != nil {
					return err
				}
				return consumirReservasDoPedido(ctx, r.Estoque, exit, faturadoDaNota(items, qtd))
			}
		}
	}
	return baixarEstoque(ctx, r.Estoque, exit, items, depositos, quantidades, mascaras, userID)
}

func (e *EfeitosNFe) depositos(ctx context.Context, exit *entity.FiscalExit, items []*entity.FiscalExitItem) (map[int64]int64, error) {
	porItem := map[int64]int64{}
	if e.Orders != nil && exit.SalesOrderCode != nil {
		linhas, err := e.Orders.ListItems(ctx, *exit.SalesOrderCode)
		if err != nil {
			return nil, fmt.Errorf("lendo os itens do pedido da nota: %w", err)
		}
		for _, l := range linhas {
			if l.WarehouseCode != nil {
				porItem[l.ItemCode] = *l.WarehouseCode
			}
		}
	}
	for _, it := range items {
		if it.ItemCode == nil {
			continue
		}
		if _, ok := porItem[*it.ItemCode]; ok {
			continue
		}
		padrao := int64(0)
		if e.Itens != nil {
			code, err := valueobject.NewItemCode(*it.ItemCode)
			if err != nil {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("item %d da nota inválido: %v", *it.ItemCode, err))
			}
			item, err := e.Itens.FindItemByCode(ctx, code)
			if err != nil && !errorsuc.EhNaoEncontrado(err) {
				return nil, fmt.Errorf("lendo o depósito padrão do item %d: %w", *it.ItemCode, err)
			}
			if item != nil && item.Warehouse.WarehouseCode > 0 {
				padrao = int64(item.Warehouse.WarehouseCode)
			}
		}
		if padrao <= 0 {
			return nil, errorsuc.NewValidationError(fmt.Sprintf(
				"o item %s da nota não tem depósito de saída: informe o depósito na linha do pedido ou o depósito padrão no cadastro do item",
				descricaoItem(it)))
		}
		porItem[*it.ItemCode] = padrao
	}
	return porItem, nil
}

// mascaraDaLinha: a da própria linha; nota antiga vai pela do pedido.
func mascaraDaLinha(it *entity.FiscalExitItem, mascaras map[int64]string) string {
	if it.Mask != "" {
		return it.Mask
	}
	return mascaras[*it.ItemCode]
}

// mascarasDoPedido: a máscara de cada item, que só o pedido guarda — a linha da
// nota não tem. Sem ela o item configurado baixava de um saldo sem máscara que
// não existe, e a nota parava em "saldo insuficiente" com o produto na prateleira.
func (e *EfeitosNFe) mascarasDoPedido(ctx context.Context, exit *entity.FiscalExit) (map[int64]string, error) {
	out := map[int64]string{}
	if e.Orders == nil || exit.SalesOrderCode == nil {
		return out, nil
	}
	linhas, err := e.Orders.ListItems(ctx, *exit.SalesOrderCode)
	if err != nil {
		return nil, fmt.Errorf("lendo os itens do pedido da nota: %w", err)
	}
	for _, l := range linhas {
		if l.Mask != "" {
			out[l.ItemCode] = l.Mask
		}
	}
	return out, nil
}

func descricaoItem(it *entity.FiscalExitItem) string {
	if it.Description != nil && *it.Description != "" {
		return *it.Description
	}
	return fmt.Sprintf("%d", safeInt64(it.ItemCode))
}

func (e *EfeitosNFe) autorizacao(ctx context.Context, exitID int64, userID uuid.UUID) error {
	exit, err := e.Fiscal.GetExitByID(ctx, exitID)
	if err != nil {
		return fmt.Errorf("lendo a nota: %w", err)
	}
	items, err := e.Fiscal.GetExitItems(ctx, exitID)
	if err != nil {
		return fmt.Errorf("lendo os itens da nota: %w", err)
	}
	plano, err := resolverPlanoDaNota(ctx, exit, items, e.Customers, e.Orders)
	if err != nil {
		return err
	}
	efeitos, err := efeitosDaNota(ctx, e.TiposNota, exit)
	if err != nil {
		return err
	}
	plano = planoDoTipoNota(plano, efeitos)
	// Tipo de nota que não mexe no estoque não pede depósito nem conversão de unidade.
	var depositos map[int64]int64
	var quantidades map[int64]decimal.Decimal
	var mascaras map[int64]string
	if efeitos.MovimentaEstoque {
		if efeitos.Tipo != entity.TipoNotaVendaConsignacao {
			if depositos, err = e.depositos(ctx, exit, items); err != nil {
				return err
			}
		}
		if quantidades, err = e.quantidadesEmEstoque(ctx, items); err != nil {
			return err
		}
		if mascaras, err = e.mascarasDoPedido(ctx, exit); err != nil {
			return err
		}
	}
	var clienteID *int64
	if e.Customers != nil && exit.CustomerCode != nil {
		// O título guarda o CÓDIGO do cliente: é por ele que Contas a Receber
		// filtra e que a análise de crédito soma o que está em aberto.
		cliente, err := e.Customers.GetCustomerByCode(ctx, *exit.CustomerCode)
		if err != nil && !errorsuc.EhNaoEncontrado(err) {
			return fmt.Errorf("lendo o cliente da nota: %w", err)
		}
		if err == nil && cliente != nil {
			code := cliente.Code
			clienteID = &code
		}
	}
	return e.Transacao(ctx, func(r EfeitosRepos) error {
		if aberta, err := r.Pendencias.Reservar(ctx, exitID, entity.PendenciaAutorizacao); err != nil || !aberta {
			return err
		}
		if e.GeraTitulos {
			if err := gerarTitulos(ctx, r.Financeiro, exit, plano, clienteID, userID); err != nil {
				return err
			}
		}
		if efeitos.MovimentaEstoque {
			if err := aplicarEstoque(ctx, r, exit, items, efeitos, depositos, quantidades, mascaras, userID); err != nil {
				return err
			}
		}
		if r.Series != nil {
			if err := r.Series.SaidaPelaNota(ctx, exitID, exit.CustomerCode, exit.DataEmissao, atorOuNil(userID)); err != nil {
				return fmt.Errorf("levando as séries da nota ao cliente: %w", err)
			}
		}
		if efeitos.BaixaPedido {
			if err := faturarPedido(ctx, r, exit); err != nil {
				return err
			}
		}
		return r.Pendencias.Resolver(ctx, exitID, entity.PendenciaAutorizacao)
	})
}

func (e *EfeitosNFe) cancelamento(ctx context.Context, exitID int64) error {
	exit, err := e.Fiscal.GetExitByID(ctx, exitID)
	if err != nil {
		return fmt.Errorf("lendo a nota: %w", err)
	}
	return e.Transacao(ctx, func(r EfeitosRepos) error {
		if aberta, err := r.Pendencias.Reservar(ctx, exitID, entity.PendenciaCancelamento); err != nil || !aberta {
			return err
		}
		if e.GeraTitulos {
			if err := r.Financeiro.CancelContasReceberByFiscalExit(ctx, exitID); err != nil {
				return fmt.Errorf("cancelando os títulos da nota: %w", err)
			}
		}
		if r.Series != nil {
			if err := r.Series.EstornoDaNota(ctx, exitID, nil); err != nil {
				return fmt.Errorf("devolvendo ao estoque as séries da nota: %w", err)
			}
		}
		// O que a nota atendeu do pedido volta a ficar pendente; pedido que
		// estava "faturado" por esta nota volta a "pedido" para refaturar.
		if r.Faturamento != nil && exit.SalesOrderCode != nil && r.Pedidos != nil {
			ligada, _, err := r.Faturamento.RegistrarFaturamento(ctx, exitID, true)
			if err != nil {
				return fmt.Errorf("devolvendo ao pedido %d o que a nota atendeu: %w", *exit.SalesOrderCode, err)
			}
			if ligada {
				pedido, err := r.Pedidos.GetByCode(ctx, *exit.SalesOrderCode)
				if err != nil {
					return fmt.Errorf("lendo o pedido %d: %w", *exit.SalesOrderCode, err)
				}
				if pedido != nil && pedido.Status == salesentity.SalesOrderStatusInvoiced {
					if err := r.Pedidos.ChangeStatus(ctx, *exit.SalesOrderCode, salesentity.SalesOrderStatusOrder); err != nil {
						return fmt.Errorf("voltando o pedido %d para pedido: %w", *exit.SalesOrderCode, err)
					}
				}
			}
		}
		return r.Pendencias.Resolver(ctx, exitID, entity.PendenciaCancelamento)
	})
}

// gerarTitulos grava um título do contas a receber por parcela da condição de
// pagamento da nota. O valor parcelado é o TOTAL da nota (produtos + IPI +
// ICMS-ST + frete + seguro − desconto): é o que o cliente paga.
func gerarTitulos(ctx context.Context, fin financialRepo.FinancialRepository, exit *entity.FiscalExit, plano PlanoDaNota, clienteID *int64, userID uuid.UUID) error {
	// Nota que só devolve material do cliente não gera conta a receber.
	if plano.SemCobranca {
		return nil
	}
	parcelas := plano.Parcelas
	if len(parcelas) == 0 {
		parcelas = []customerentity.ParcelaCalculada{{
			Numero: 1, Valor: exit.ValorTotal,
			Vencimento: exit.DataEmissao.AddDate(0, 0, 30), Descricao: "30 dias",
		}}
	}
	total := int32(len(parcelas))
	exitID := exit.ID
	for _, p := range parcelas {
		numDoc := fmt.Sprintf("NF-%d/%d", exit.NumeroNF, p.Numero)
		if total == 1 {
			numDoc = fmt.Sprintf("NF-%d", exit.NumeroNF)
		}
		forma := formaPagamentoNFe(p)
		cr := &financialEntity.ContaReceber{
			NumeroDocumento: &numDoc,
			ClienteID:       clienteID,
			FiscalExitID:    &exitID,
			SalesOrderID:    exit.SalesOrderCode,
			DataLancamento:  time.Now(),
			DataEmissao:     exit.DataEmissao,
			DataVencimento:  p.Vencimento,
			ValorBruto:      p.Valor,
			Desconto:        decimal.Zero,
			Juros:           decimal.Zero,
			Multa:           decimal.Zero,
			ValorRecebido:   decimal.Zero,
			ParcelaNumero:   int32(p.Numero),
			ParcelaTotal:    total,
			FormaPagamento:  &forma,
			Status:          financialEntity.ContaReceberStatusPendente,
			IsActive:        true,
			CriadoPor:       userID,
		}
		if _, err := fin.CreateContaReceber(ctx, cr); err != nil {
			return fmt.Errorf("gerando o título da parcela %d: %w", p.Numero, err)
		}
	}
	return nil
}

// quantidadesEmEstoque converte a quantidade de cada linha da unidade da nota
// (uCom — a de venda) para a unidade de estoque do item. A nota de 2 CX de um
// item estocado em UN baixava 2 UN; agora baixa 2 × o fator da caixa. Unidade
// diferente sem conversão é recusada: a baixa sairia com o número errado.
func (e *EfeitosNFe) quantidadesEmEstoque(ctx context.Context, items []*entity.FiscalExitItem) (map[int64]decimal.Decimal, error) {
	out := map[int64]decimal.Decimal{}
	for _, it := range items {
		if it.ItemCode == nil || it.UnidadeComercial == nil || strings.TrimSpace(*it.UnidadeComercial) == "" || e.Itens == nil {
			continue
		}
		code, err := valueobject.NewItemCode(*it.ItemCode)
		if err != nil {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("item %d da nota inválido: %v", *it.ItemCode, err))
		}
		item, err := e.Itens.FindItemByCode(ctx, code)
		if err != nil {
			if errorsuc.EhNaoEncontrado(err) {
				continue
			}
			return nil, fmt.Errorf("lendo a unidade de estoque do item %d: %w", *it.ItemCode, err)
		}
		daNota := strings.ToUpper(strings.TrimSpace(*it.UnidadeComercial))
		doEstoque := strings.ToUpper(strings.TrimSpace(string(item.Warehouse.UnitOfMeasurement)))
		if doEstoque == "" || daNota == doEstoque {
			continue
		}
		semConversao := errorsuc.NewValidationError(fmt.Sprintf(
			"item %s: a nota sai em %s e o item é estocado em %s, sem conversão — cadastre a conversão %s→%s do item (VSUP0110)",
			descricaoItem(it), daNota, doEstoque, daNota, doEstoque))
		if e.Conversor == nil {
			return nil, semConversao
		}
		qf, achou, err := e.Conversor.ConvertQuantity(ctx, *it.ItemCode, it.Quantity, daNota, doEstoque)
		if err != nil {
			return nil, err
		}
		if !achou {
			return nil, semConversao
		}
		out[it.ID] = qf
	}
	return out, nil
}

// saidasDaSeparacao reparte a saída pelo que a onda do pedido separou: a nota leva
// o lote e o endereço pegos, não os que a baixa automática escolheria. Linha com lote
// ou endereço digitado na nota fica como está.
func saidasDaSeparacao(ctx context.Context, estoque stockrepo.StockRepository, exit *entity.FiscalExit, it *entity.FiscalExitItem, base stockentity.StockMovement, usado ...map[int64]decimal.Decimal) ([]stockentity.StockMovement, error) {
	leitor, ok := estoque.(interface {
		SeparadoParaFaturar(ctx context.Context, pedido, item int64, mask string, almox int64) ([]stockentity.SeparadoDaOnda, error)
	})
	if !ok || exit.SalesOrderCode == nil || it.Lot != nil || it.Address != nil {
		return []stockentity.StockMovement{base}, nil
	}
	separado, err := leitor.SeparadoParaFaturar(ctx, *exit.SalesOrderCode, base.ItemCode, base.Mask, base.WarehouseID)
	if err != nil {
		return nil, err
	}
	saidas := []stockentity.StockMovement{}
	resta := base.Quantity
	for _, s := range separado {
		if !resta.IsPositive() {
			break
		}
		livre := s.Quantidade
		if len(usado) > 0 {
			livre = livre.Sub(usado[0][s.Linha])
		}
		if !livre.IsPositive() {
			continue
		}
		q := decimal.Min(resta, livre)
		if len(usado) > 0 {
			usado[0][s.Linha] = usado[0][s.Linha].Add(q)
		}
		m := base
		m.Quantity = q
		linha := s.Linha
		m.LinhaDaOnda = &linha
		if s.Lote != "" {
			lote := s.Lote
			m.Lot = &lote
		}
		if s.Endereco != "" {
			endereco := s.Endereco
			m.Address = &endereco
		}
		saidas = append(saidas, m)
		resta = resta.Sub(q)
	}
	if resta.IsPositive() {
		m := base
		m.Quantity = resta
		saidas = append(saidas, m)
	}
	return saidas, nil
}

// baixarEstoque lança a saída de cada item no depósito resolvido e consome as
// reservas do pedido.
func baixarEstoque(ctx context.Context, estoque stockrepo.StockRepository, exit *entity.FiscalExit, items []*entity.FiscalExitItem, depositos map[int64]int64, quantidades map[int64]decimal.Decimal, mascaras map[int64]string, userID uuid.UUID) error {
	if estoque == nil {
		return nil
	}
	faturado := map[int64]decimal.Decimal{}
	for _, it := range items {
		if it.ItemCode == nil {
			continue
		}
		refType := stockentity.ReferenceTypeNFExit
		refCode := exit.ID
		qtd := it.Quantity
		if q, ok := quantidades[it.ID]; ok {
			qtd = q
		}
		faturado[*it.ItemCode] = faturado[*it.ItemCode].Add(qtd)
		base := stockentity.StockMovement{
			ItemCode:     *it.ItemCode,
			Mask:         mascaraDaLinha(it, mascaras),
			WarehouseID:  depositos[*it.ItemCode],
			MovementType: stockentity.MovementTypeOut,
			Quantity:     qtd,
			Lot:          it.Lot,
			Address:      it.Address,
			// Sem preço: a saída é valorizada pelo custo médio (é o custo que sai
			// do estoque e o CMV da venda).
			ReferenceType: &refType,
			ReferenceCode: &refCode,
			CreatedBy:     userID,
		}
		saidas, err := saidasDaSeparacao(ctx, estoque, exit, it, base)
		if err != nil {
			return err
		}
		for i := range saidas {
			if _, err := estoque.CreateMovement(ctx, &saidas[i]); err != nil {
				return fmt.Errorf("baixando o estoque do item %s: %w", descricaoItem(it), err)
			}
		}
	}
	return consumirReservasDoPedido(ctx, estoque, exit, faturado)
}

// consumirReservasDoPedido solta da reserva do pedido o que a nota levou. A remessa e
// a transferência também tiram a mercadoria da origem: a reserva vai junto.
func consumirReservasDoPedido(ctx context.Context, estoque stockrepo.StockRepository, exit *entity.FiscalExit, faturado map[int64]decimal.Decimal) error {
	if estoque == nil || exit.SalesOrderCode == nil {
		return nil
	}
	reservas, err := estoque.ListActiveReservations(ctx)
	if err != nil {
		return fmt.Errorf("lendo as reservas do pedido: %w", err)
	}
	for _, r := range reservas {
		if r.ReferenceType != stockentity.ReferenceTypeSalesOrder || r.ReferenceCode != *exit.SalesOrderCode {
			continue
		}
		// Consome só o que esta nota faturou: a nota parcial soltava a reserva
		// inteira, e o saldo do pedido ficava livre para outro cliente.
		consumido, err := estoque.ConsumirReservaParcial(ctx, r.ID, faturado[r.ItemCode])
		if err != nil {
			return fmt.Errorf("consumindo a reserva %d do pedido: %w", r.ID, err)
		}
		faturado[r.ItemCode] = faturado[r.ItemCode].Sub(consumido)
	}
	return nil
}

func faturadoDaNota(items []*entity.FiscalExitItem, qtd func(*entity.FiscalExitItem) decimal.Decimal) map[int64]decimal.Decimal {
	out := map[int64]decimal.Decimal{}
	for _, it := range items {
		if it.ItemCode != nil {
			out[*it.ItemCode] = out[*it.ItemCode].Add(qtd(it))
		}
	}
	return out
}

// partesDaVendaConsignada divide a linha pelos lotes que o cliente tem. Linha com lote
// segue inteira; o que passar do saldo fica sem lote, e a baixa recusa com a falta.
func partesDaVendaConsignada(ctx context.Context, c stockrepo.ConsignacaoEstoque, cliente int64, it *entity.FiscalExitItem, mask string, q decimal.Decimal) ([]*entity.FiscalExitItem, map[int64]decimal.Decimal, error) {
	qtds := map[int64]decimal.Decimal{}
	var partes []*entity.FiscalExitItem
	parte := func(lote *string, q decimal.Decimal) {
		p := *it
		p.ID = int64(-1 - len(partes))
		p.Lot = lote
		qtds[p.ID] = q
		partes = append(partes, &p)
	}
	if it.Lot != nil {
		parte(it.Lot, q)
		return partes, qtds, nil
	}
	lotes, err := c.LotesConsignados(ctx, cliente, *it.ItemCode, mask, q)
	if err != nil {
		return nil, nil, err
	}
	resta := q
	for _, l := range lotes {
		parte(l.Lote, l.Quantidade)
		resta = resta.Sub(l.Quantidade)
	}
	if resta.IsPositive() {
		parte(nil, resta)
	}
	return partes, qtds, nil
}

// faturarPedido soma ao atendido do pedido o que a nota faturou e marca o
// pedido como faturado quando tudo foi atendido (ou quando a nota não é ligada
// às linhas do pedido).
func faturarPedido(ctx context.Context, r EfeitosRepos, exit *entity.FiscalExit) error {
	if r.Pedidos == nil || exit.SalesOrderCode == nil {
		return nil
	}
	ligada, atendido := false, false
	if r.Faturamento != nil {
		var err error
		if ligada, atendido, err = r.Faturamento.RegistrarFaturamento(ctx, exit.ID, false); err != nil {
			return fmt.Errorf("registrando no pedido o que a nota faturou: %w", err)
		}
	}
	if !ligada || atendido {
		if err := r.Pedidos.ChangeStatus(ctx, *exit.SalesOrderCode, salesentity.SalesOrderStatusInvoiced); err != nil {
			return fmt.Errorf("marcando o pedido %d como faturado: %w", *exit.SalesOrderCode, err)
		}
	}
	return nil
}

func atorOuNil(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}
