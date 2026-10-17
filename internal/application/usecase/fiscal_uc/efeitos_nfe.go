package fiscal_uc

import (
	"context"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"strings"
	"time"

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
	// Conversor leva a quantidade da nota (unidade comercial) para a unidade de
	// estoque do item antes da baixa.
	Conversor ports.UOMConverter
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
	_, err := e.depositos(ctx, exit, items)
	return err
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
	plano, err := resolverPlanoDaNota(ctx, exit, e.Customers, e.Orders)
	if err != nil {
		return err
	}
	depositos, err := e.depositos(ctx, exit, items)
	if err != nil {
		return err
	}
	quantidades, err := e.quantidadesEmEstoque(ctx, items)
	if err != nil {
		return err
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
		if e.GeraTitulos {
			if err := gerarTitulos(ctx, r.Financeiro, exit, plano, clienteID, userID); err != nil {
				return err
			}
		}
		if err := baixarEstoque(ctx, r.Estoque, exit, items, depositos, quantidades, userID); err != nil {
			return err
		}
		if err := faturarPedido(ctx, r, exit); err != nil {
			return err
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
		if e.GeraTitulos {
			if err := r.Financeiro.CancelContasReceberByFiscalExit(ctx, exitID); err != nil {
				return fmt.Errorf("cancelando os títulos da nota: %w", err)
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
	parcelas := plano.Parcelas
	if len(parcelas) == 0 {
		parcelas = []customerentity.ParcelaCalculada{{
			Numero: 1, Valor: decimal.NewFromFloat(exit.ValorTotal),
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
func (e *EfeitosNFe) quantidadesEmEstoque(ctx context.Context, items []*entity.FiscalExitItem) (map[int64]float64, error) {
	out := map[int64]float64{}
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
		q, achou, err := e.Conversor.ConvertQuantity(ctx, *it.ItemCode, it.Quantity, daNota, doEstoque)
		if err != nil {
			return nil, err
		}
		if !achou {
			return nil, semConversao
		}
		out[it.ID] = q
	}
	return out, nil
}

// baixarEstoque lança a saída de cada item no depósito resolvido e consome as
// reservas do pedido.
func baixarEstoque(ctx context.Context, estoque stockrepo.StockRepository, exit *entity.FiscalExit, items []*entity.FiscalExitItem, depositos map[int64]int64, quantidades map[int64]float64, userID uuid.UUID) error {
	if estoque == nil {
		return nil
	}
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
		mov := &stockentity.StockMovement{
			ItemCode:     *it.ItemCode,
			WarehouseID:  depositos[*it.ItemCode],
			MovementType: stockentity.MovementTypeOut,
			Quantity:     qtd,
			// Sem preço: a saída é valorizada pelo custo médio (é o custo que sai
			// do estoque e o CMV da venda).
			UnitPrice:     0,
			TotalPrice:    0,
			ReferenceType: &refType,
			ReferenceCode: &refCode,
			CreatedBy:     userID,
		}
		if _, err := estoque.CreateMovement(ctx, mov); err != nil {
			return fmt.Errorf("baixando o estoque do item %s: %w", descricaoItem(it), err)
		}
	}
	if exit.SalesOrderCode != nil {
		reservas, err := estoque.ListActiveReservations(ctx)
		if err != nil {
			return fmt.Errorf("lendo as reservas do pedido: %w", err)
		}
		for _, r := range reservas {
			if r.ReferenceType == stockentity.ReferenceTypeSalesOrder && r.ReferenceCode == *exit.SalesOrderCode {
				if err := estoque.ConsumeReservation(ctx, r.ID); err != nil {
					return fmt.Errorf("consumindo a reserva %d do pedido: %w", r.ID, err)
				}
			}
		}
	}
	return nil
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
