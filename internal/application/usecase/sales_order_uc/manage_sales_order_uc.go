package sales_order_uc

import (
	"context"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/domain/aprovacao"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type CancelSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
	// Titulos: os títulos a receber ligados ao pedido (adiantamento, sinal,
	// título lançado à mão) caem junto com o pedido.
	Titulos RecebiveisDoPedido
	// Transacao cancela o pedido, o saldo das linhas e as reservas de estoque
	// juntos. Sem ela (testes), roda direto em Repo.
	Transacao EmTransacao
}

// linhasDoCancelamento encerra o saldo não atendido das linhas do pedido.
type linhasDoCancelamento interface {
	CancelarLinhasEmAberto(ctx context.Context, code int64) error
}

// reservasDoPedido é o que o cancelamento usa do estoque para soltar a reserva
// feita na confirmação (ATP).
type reservasDoPedido interface {
	ListActiveReservations(ctx context.Context) ([]*stockentity.StockReservation, error)
	CancelReservation(ctx context.Context, id int64) error
}

// RecebiveisDoPedido confere (cancelar=false) ou cancela os títulos a receber
// do pedido; recusa se algum já teve recebimento.
type RecebiveisDoPedido interface {
	RecebiveisDoPedidoDeVenda(ctx context.Context, pedido int64, cancelar bool) (int, error)
}

func (uc *CancelSalesOrderUseCase) Execute(ctx context.Context, dto request.CancelSalesOrderDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if strings.TrimSpace(dto.Reason) == "" {
		return errorsuc.NewValidationError("o motivo do cancelamento é obrigatório")
	}
	// Confere antes: título recebido impede o cancelamento, e o pedido não
	// pode ficar cancelado com o título dele em aberto.
	if uc.Titulos != nil {
		if _, err := uc.Titulos.RecebiveisDoPedidoDeVenda(ctx, dto.Code, false); err != nil {
			return err
		}
	}
	pedido, err := uc.Repo.GetByCode(ctx, dto.Code)
	if err != nil {
		return err
	}
	switch pedido.Status {
	case entity.SalesOrderStatusCancelled:
		return errorsuc.NewValidationError(fmt.Sprintf("o pedido %d já está cancelado", dto.Code))
	case entity.SalesOrderStatusInvoiced:
		return errorsuc.NewValidationError(fmt.Sprintf("o pedido %d está faturado: cancele a NF-e antes — ela devolve o pedido à carteira", dto.Code))
	}
	cancelar := func(r ConfirmacaoRepos) error {
		if err := r.Pedidos.Cancel(ctx, dto.Code, dto.Reason, dto.Complement); err != nil {
			return err
		}
		if l, ok := r.Pedidos.(linhasDoCancelamento); ok {
			if err := l.CancelarLinhasEmAberto(ctx, dto.Code); err != nil {
				return err
			}
		}
		if r.Estoque != nil {
			return soltarReservasDoPedido(ctx, r.Estoque, dto.Code)
		}
		return nil
	}
	if uc.Transacao != nil {
		err = uc.Transacao(ctx, cancelar)
	} else {
		err = cancelar(ConfirmacaoRepos{Pedidos: uc.Repo})
	}
	if err != nil {
		return err
	}
	if uc.Titulos != nil {
		if _, err := uc.Titulos.RecebiveisDoPedidoDeVenda(ctx, dto.Code, true); err != nil {
			return fmt.Errorf("o pedido %d foi cancelado, mas os títulos a receber dele não: %w", dto.Code, err)
		}
	}
	return nil
}

type BlockSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *BlockSalesOrderUseCase) Execute(ctx context.Context, dto request.BlockSalesOrderDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Block(ctx, dto.Code, dto.Reason)
}

type UnblockSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
	// Aprovacoes: com aprovação de crédito ou de política comercial pendente,
	// o desbloqueio manual é recusado — a liberação passa pela alçada.
	Aprovacoes Aprovacoes
	// Conclusao termina a confirmação interrompida pelo bloqueio (demanda do
	// MRP e reserva). Sem ela, só limpa o bloqueio, como antes.
	Conclusao *ConclusaoDoDesbloqueio
}

func (uc *UnblockSalesOrderUseCase) Execute(ctx context.Context, code int64) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if uc.Aprovacoes != nil {
		// Crédito ou política comercial/alçada de desconto pendente: a liberação
		// é da alçada — o desbloqueio manual contornaria a aprovação.
		pendente, err := outraPendente(ctx, uc.Aprovacoes, code, "")
		if err != nil {
			return err
		}
		if pendente != nil {
			oQue := "de crédito"
			if pendente.Tipo == aprovacao.PedidoPoliticaComercial {
				oQue = "da política comercial/desconto"
			}
			return errorsuc.NewValidationError(fmt.Sprintf(
				"o pedido %d aguarda a aprovação %s nº %d (nível %d de %d): a liberação é feita pelos aprovadores em Aprovações",
				code, oQue, pendente.ID, pendente.NivelAtual, pendente.NiveisExigidos))
		}
	}
	if uc.Conclusao != nil {
		return uc.Conclusao.Concluir(ctx, code)
	}
	return uc.Repo.Unblock(ctx, code)
}

type ChangeStatusSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
	// CreditChecker is optional. When set, confirming an order runs an automatic
	// credit-limit check; an order that exceeds the customer's limit is blocked
	// (and does not feed the MRP) instead of flowing through unchecked.
	CreditChecker *CreditChecker
	// Transacao roda a confirmação inteira (status, crédito e reservas) numa
	// transação só: ou tudo grava, ou nada — o pedido nunca fica confirmado sem
	// a reserva. O MRP lê o pedido confirmado direto (saldo aberto da linha);
	// a confirmação não cria demanda independente, que contava o pedido duas
	// vezes.
	Transacao EmTransacao
	// Reserver is optional. When set, confirming an order reserves available
	// stock per line (ATP) so the promise is backed by real availability.
	Reserver *OrderStockReserver
	// Aprovacoes é opcional: com política de LIBERACAO_CREDITO ativa, o pedido
	// bloqueado pelo crédito abre a solicitação de aprovação.
	Aprovacoes Aprovacoes
}

func (uc *ChangeStatusSalesOrderUseCase) Execute(ctx context.Context, dto request.ChangeStatusDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	newStatus := entity.SalesOrderStatus(dto.Status)
	if err := conferirTrocaDeStatus(ctx, uc.Repo, dto.Code, newStatus); err != nil {
		return err
	}
	// Pedido (P) e pedido em análise (A) são venda firme: o MRP planeja por
	// eles e o crédito os conta como exposição. Sem linha com quantidade não há
	// o que vender — antes o pedido vazio era confirmado e ficava na carteira.
	if newStatus == entity.SalesOrderStatusOrder || newStatus == entity.SalesOrderStatusAnalysis {
		if err := conferirLinhasParaConfirmar(ctx, uc.Repo, dto.Code); err != nil {
			return err
		}
	}
	if newStatus != entity.SalesOrderStatusOrder {
		return uc.Repo.ChangeStatus(ctx, dto.Code, newStatus)
	}
	emTransacao := uc.Transacao
	if emTransacao == nil {
		// Sem transação configurada (testes), roda direto nos repositórios.
		emTransacao = func(ctx context.Context, fn func(ConfirmacaoRepos) error) error {
			r := ConfirmacaoRepos{Pedidos: uc.Repo}
			if uc.Reserver != nil {
				r.Estoque = uc.Reserver.StockRepo
			}
			return fn(r)
		}
	}
	bloqueadoPorCredito := false
	err := emTransacao(ctx, func(r ConfirmacaoRepos) error {
		if err := r.Pedidos.ChangeStatus(ctx, dto.Code, newStatus); err != nil {
			return err
		}
		// Crédito primeiro: pedido bloqueado não alimenta o MRP nem reserva estoque.
		if uc.CreditChecker != nil {
			aprovado, err := uc.CreditChecker.Check(ctx, dto.Code, r.Pedidos)
			if err != nil {
				return errorsuc.NewValidationError("não foi possível conferir o crédito do cliente agora — o pedido não foi confirmado; tente de novo (" + err.Error() + ")")
			}
			if !aprovado {
				bloqueadoPorCredito = true
				return nil
			}
		}
		if uc.Reserver != nil {
			if err := uc.Reserver.Reserve(ctx, dto.Code, r.Pedidos, r.Estoque); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil || !bloqueadoPorCredito || uc.Aprovacoes == nil {
		return err
	}
	// Depois do commit: a solicitação aponta para um pedido que já está gravado
	// como bloqueado.
	return solicitarLiberacao(ctx, uc.Aprovacoes, uc.Repo, dto.Code)
}

// conferirLinhasParaConfirmar exige ao menos uma linha ativa com quantidade a
// vender (pedida menos cancelada).
func conferirLinhasParaConfirmar(ctx context.Context, repo repository.SalesOrderRepository, code int64) error {
	linhas, err := repo.ListItems(ctx, code)
	if err != nil {
		return fmt.Errorf("lendo os itens do pedido %d: %w", code, err)
	}
	for _, l := range linhas {
		if l != nil && l.IsActive && l.RequestedQty.Sub(l.CancelledQty).IsPositive() {
			return nil
		}
	}
	return errorsuc.NewValidationError("inclua ao menos um item com quantidade antes de confirmar o pedido")
}

// ConfirmacaoRepos são os repositórios ligados à transação da confirmação.
type ConfirmacaoRepos struct {
	Pedidos repository.SalesOrderRepository
	Estoque stockrepo.StockRepository
}

// EmTransacao roda fn numa transação com os repositórios ligados a ela; erro
// devolvido por fn desfaz tudo.
type EmTransacao func(ctx context.Context, fn func(ConfirmacaoRepos) error) error

type AnalyzeSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *AnalyzeSalesOrderUseCase) Execute(ctx context.Context, dto request.AnalyzeSalesOrderDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	area := strings.ToUpper(strings.TrimSpace(dto.Area))
	if area != "COMMERCIAL" && area != "FINANCIAL" {
		return errorsuc.NewValidationError("a área deve ser COMMERCIAL ou FINANCIAL")
	}
	status := entity.SalesOrderAnalysisStatus(dto.Status)
	if status != entity.SalesOrderAnalysisApproved && status != entity.SalesOrderAnalysisRejected && status != entity.SalesOrderAnalysisNotAnalyzed {
		return errorsuc.NewValidationError("a situação da análise deve ser APPROVED, REJECTED ou NOT_ANALYZED")
	}
	if strings.TrimSpace(dto.Reason) == "" {
		return errorsuc.NewValidationError("o motivo da análise é obrigatório")
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Analyze(ctx, dto.Code, area, status, strings.TrimSpace(dto.Reason), actor)
}

type ReleaseSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *ReleaseSalesOrderUseCase) Execute(ctx context.Context, dto request.ReleaseSalesOrderDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	status := entity.SalesOrderReleaseStatus(dto.ReleaseStatus)
	if status != entity.SalesOrderReleaseBlocked && status != entity.SalesOrderReleaseManual && status != entity.SalesOrderReleaseOK {
		return errorsuc.NewValidationError("a situação de liberação deve ser BLOCKED, MANUAL ou OK")
	}
	if strings.TrimSpace(dto.Reason) == "" {
		return errorsuc.NewValidationError("o motivo da liberação é obrigatório")
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Release(ctx, dto.Code, status, strings.TrimSpace(dto.Reason), strings.TrimSpace(dto.Area), actor)
}

type AttendSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *AttendSalesOrderUseCase) Execute(ctx context.Context, dto request.AttendSalesOrderDTO) error {
	if msg := datetime.PrimeiraInvalida(datetime.Campo("data do atendimento", &dto.EventDate)); msg != "" {
		return errorsuc.NewValidationError(msg)
	}
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if strings.TrimSpace(dto.Reason) == "" {
		return errorsuc.NewValidationError("o motivo do atendimento é obrigatório")
	}
	eventDate := datetime.ParseDatePtr(&dto.EventDate)
	if strings.TrimSpace(dto.EventDate) != "" && eventDate == nil {
		return errorsuc.NewValidationError("a data do atendimento deve estar no formato ISO AAAA-MM-DD")
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Attend(ctx, dto.Code, strings.TrimSpace(dto.Reason), eventDate, actor)
}

type ConferSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *ConferSalesOrderUseCase) Execute(ctx context.Context, dto request.ConferSalesOrderDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	status := entity.SalesOrderConferenceStatus(dto.Status)
	if status != entity.SalesOrderConferencePending && status != entity.SalesOrderConferenceConferred && status != entity.SalesOrderConferenceDivergent {
		return errorsuc.NewValidationError("a situação da conferência deve ser PENDING, CONFERRED ou DIVERGENT")
	}
	if strings.TrimSpace(dto.Status) == "" {
		return errorsuc.NewValidationError("a situação da conferência é obrigatória")
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Confer(ctx, dto.Code, status, strings.TrimSpace(dto.Reason), actor)
}

type SaveSalesOrderDelayReasonUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *SaveSalesOrderDelayReasonUseCase) Execute(ctx context.Context, dto request.SaveSalesOrderDelayReasonDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if strings.TrimSpace(dto.Reason) == "" {
		return errorsuc.NewValidationError("o motivo do atraso é obrigatório")
	}
	if strings.TrimSpace(dto.Action) == "" {
		return errorsuc.NewValidationError("a ação para o atraso é obrigatória")
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.SaveDelayReason(ctx, dto.Code, strings.TrimSpace(dto.Reason), strings.TrimSpace(dto.Action), actor)
}

// soltarReservasDoPedido cancela as reservas ativas que a confirmação fez para
// o pedido. Antes o pedido cancelado deixava o estoque reservado, e o
// disponível para promessa ficava menor para sempre.
func soltarReservasDoPedido(ctx context.Context, estoque reservasDoPedido, code int64) error {
	return soltarReservas(ctx, estoque, code, nil)
}

// linhaDaReserva identifica as reservas de uma linha do pedido. A reserva
// antiga (e a que a onda devolve) guarda o código do produto em vez da linha:
// ela só é da linha quando o produto aparece numa linha só do pedido (Unica).
type linhaDaReserva struct {
	Code, ItemCode int64
	Unica          bool
}

func (l linhaDaReserva) eDaLinha(r *stockentity.StockReservation) bool {
	if r.ItemCode != l.ItemCode {
		return false
	}
	if r.ReferenceItemCode != nil && *r.ReferenceItemCode == l.Code {
		return true
	}
	return l.Unica && (r.ReferenceItemCode == nil || *r.ReferenceItemCode == l.ItemCode)
}

// soltarReservas cancela as reservas ativas do pedido; com linha, só as dela.
func soltarReservas(ctx context.Context, estoque reservasDoPedido, code int64, linha *linhaDaReserva) error {
	ativas, err := estoque.ListActiveReservations(ctx)
	if err != nil {
		return fmt.Errorf("lendo as reservas do pedido %d: %w", code, err)
	}
	for _, r := range ativas {
		if r.ReferenceType != stockentity.ReferenceTypeSalesOrder || r.ReferenceCode != code {
			continue
		}
		if linha != nil && !linha.eDaLinha(r) {
			continue
		}
		if err := estoque.CancelReservation(ctx, r.ID); err != nil {
			return fmt.Errorf("liberando a reserva %d do pedido %d: %w", r.ID, code, err)
		}
	}
	return nil
}

// conferirTrocaDeStatus: a troca de status anda só entre rascunho, orçamento,
// análise e pedido. Cancelar tem rotina própria (motivo, títulos, reservas e
// linhas) e faturado é a NF-e autorizada que põe — antes o PATCH de status
// aceitava CANCELLED, F e até reabrir pedido cancelado ou faturado.
func conferirTrocaDeStatus(ctx context.Context, repo repository.SalesOrderRepository, code int64, novo entity.SalesOrderStatus) error {
	switch novo {
	case entity.SalesOrderStatusDraft, entity.SalesOrderStatusOrder, entity.SalesOrderStatusAnalysis,
		entity.SalesOrderStatusBudget, entity.SalesOrderStatusBudgetAnalysis:
	case entity.SalesOrderStatusCancelled:
		return errorsuc.NewValidationError("para cancelar o pedido use o cancelamento, que pede o motivo e solta as reservas e os títulos")
	case entity.SalesOrderStatusInvoiced:
		return errorsuc.NewValidationError("o pedido fica faturado pela NF-e autorizada, não pela troca de status")
	default:
		return errorsuc.NewValidationError(fmt.Sprintf("status %q não existe para o pedido de venda", string(novo)))
	}
	pedido, err := repo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	switch pedido.Status {
	case entity.SalesOrderStatusCancelled:
		return errorsuc.NewValidationError(fmt.Sprintf("o pedido %d está cancelado e não volta à carteira", code))
	case entity.SalesOrderStatusInvoiced:
		return errorsuc.NewValidationError(fmt.Sprintf("o pedido %d está faturado: para mexer nele, cancele a NF-e", code))
	}
	return nil
}
