package sales_order_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	demandentity "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
	demandrepo "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type CancelSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *CancelSalesOrderUseCase) Execute(ctx context.Context, dto request.CancelSalesOrderDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if strings.TrimSpace(dto.Reason) == "" {
		return errorsuc.NewValidationError("o motivo do cancelamento é obrigatório")
	}
	return uc.Repo.Cancel(ctx, dto.Code, dto.Reason, dto.Complement)
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
}

func (uc *UnblockSalesOrderUseCase) Execute(ctx context.Context, code int64) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Unblock(ctx, code)
}

type ChangeStatusSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
	// DemandRepo is optional. When set, moving the order to "Pedido" (confirmed)
	// automatically feeds the MRP by creating an independent demand per order
	// line, so the planner no longer has to register demand by hand.
	DemandRepo demandrepo.IndependentDemandRepository
	// CreditChecker is optional. When set, confirming an order runs an automatic
	// credit-limit check; an order that exceeds the customer's limit is blocked
	// (and does not feed the MRP) instead of flowing through unchecked.
	CreditChecker *CreditChecker
	// Transacao roda a confirmação inteira (status, crédito, demandas e
	// reservas) numa transação só: ou tudo grava, ou nada — o pedido nunca fica
	// confirmado sem a demanda do MRP ou sem a reserva.
	Transacao EmTransacao
	// Reserver is optional. When set, confirming an order reserves available
	// stock per line (ATP) so the promise is backed by real availability.
	Reserver *OrderStockReserver
}

func (uc *ChangeStatusSalesOrderUseCase) Execute(ctx context.Context, dto request.ChangeStatusDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	newStatus := entity.SalesOrderStatus(dto.Status)
	if newStatus != entity.SalesOrderStatusOrder {
		return uc.Repo.ChangeStatus(ctx, dto.Code, newStatus)
	}
	emTransacao := uc.Transacao
	if emTransacao == nil {
		// Sem transação configurada (testes), roda direto nos repositórios.
		emTransacao = func(ctx context.Context, fn func(ConfirmacaoRepos) error) error {
			r := ConfirmacaoRepos{Pedidos: uc.Repo, Demandas: uc.DemandRepo}
			if uc.Reserver != nil {
				r.Estoque = uc.Reserver.StockRepo
			}
			return fn(r)
		}
	}
	return emTransacao(ctx, func(r ConfirmacaoRepos) error {
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
				return nil
			}
		}
		if uc.DemandRepo != nil {
			if err := generateDemands(ctx, dto.Code, r.Pedidos, r.Demandas); err != nil {
				return err
			}
		}
		if uc.Reserver != nil {
			if err := uc.Reserver.Reserve(ctx, dto.Code, r.Pedidos, r.Estoque); err != nil {
				return err
			}
		}
		return nil
	})
}

// ConfirmacaoRepos são os repositórios ligados à transação da confirmação.
type ConfirmacaoRepos struct {
	Pedidos  repository.SalesOrderRepository
	Demandas demandrepo.IndependentDemandRepository
	Estoque  stockrepo.StockRepository
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

// generateDemands cria uma demanda independente por linha aberta do pedido.
// O código da demanda deriva da linha, então reconfirmar mantém a que já
// existe; qualquer outra falha desfaz a confirmação (antes era ignorada e o
// pedido ficava confirmado sem alimentar o MRP).
func generateDemands(ctx context.Context, code int64, pedidos repository.SalesOrderRepository, demandas demandrepo.IndependentDemandRepository) error {
	order, err := pedidos.GetByCode(ctx, code)
	if err != nil {
		return fmt.Errorf("lendo o pedido: %w", err)
	}
	items, err := pedidos.ListItems(ctx, code)
	if err != nil {
		return fmt.Errorf("lendo os itens do pedido: %w", err)
	}
	for _, it := range items {
		if !it.IsActive || it.Status == entity.SalesOrderItemStatusCancelled {
			continue
		}
		qty := it.RequestedQty
		if qty <= 0 {
			continue
		}

		demandDate := time.Now()
		switch {
		case it.DeliveryDate != nil:
			demandDate = *it.DeliveryDate
		case order.DeliveryDate != nil:
			demandDate = *order.DeliveryDate
		}

		var mask *string
		if it.Mask != "" {
			m := it.Mask
			mask = &m
		}

		// Deterministic, order-line-scoped code keeps re-confirmation idempotent.
		demandCode := code*100000 + int64(it.Sequence)
		demand := &demandentity.IndependentDemand{
			CodeDemand: demandCode,
			ItemCode:   it.ItemCode,
			Mask:       mask,
			Quantity:   qty,
			DemandDate: demandDate,
			CreatedBy:  order.CreatedBy,
		}
		if _, err := demandas.GetByCode(ctx, demandCode); err == nil {
			continue // reconfirmação: a demanda desta linha já existe
		} else if !errorsuc.EhNaoEncontrado(err) {
			return fmt.Errorf("conferindo a demanda da linha %d: %w", it.Sequence, err)
		}
		if _, err := demandas.Create(ctx, demand); err != nil {
			return fmt.Errorf("gerando a demanda do MRP da linha %d: %w", it.Sequence, err)
		}
	}
	return nil
}
