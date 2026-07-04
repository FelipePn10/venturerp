package purchase_order_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	procuremententity "github.com/FelipePn10/panossoerp/internal/domain/procurement/entity"
	poentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	porepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

// Alçada status codes stored on the purchase order.
const (
	alcadaApproved = "A" // liberado
	alcadaBlocked  = "B" // aguardando autorização de alçada
	alcadaRejected = "R" // acima do teto absoluto; não pode ser autorizado
)

// ApprovalPolicy resolves whether a purchase amount can be auto-approved or needs
// hierarchical authorization. Implemented by the procurement use case; kept as a
// port so the purchase order package does not depend on the procurement use case.
type ApprovalPolicy interface {
	EvaluatePurchaseApproval(ctx context.Context, enterpriseCode int64, supplierCode *int64, amount float64) (*procuremententity.ApprovalDecision, error)
}

// ApprovePurchaseOrderUseCase evaluates the approval limit (alçada de valores) and
// either approves the order, blocks it pending authorization, or hard-rejects it.
type ApprovePurchaseOrderUseCase struct {
	Repo   porepo.PurchaseOrderRepository
	Auth   ports.AuthService
	Policy ApprovalPolicy // optional; nil disables alçada control (auto-approve)
}

func (uc *ApprovePurchaseOrderUseCase) Execute(ctx context.Context, code int64) (*response.ApprovePurchaseOrderResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	order, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	switch order.Status {
	case poentity.PurchaseOrderStatusDRAFT, poentity.PurchaseOrderStatusREQUESTED:
		// approvable
	default:
		return nil, fmt.Errorf("purchase order %d cannot be approved in status %s", code, order.Status)
	}

	amount := order.TotalNet
	if amount <= 0 {
		amount = order.TotalGross
	}

	out := &response.ApprovePurchaseOrderResponse{AppliedAmount: amount}
	if uc.Policy == nil {
		return uc.finishApproval(ctx, order, out)
	}
	decision, err := uc.Policy.EvaluatePurchaseApproval(ctx, order.EnterpriseCode, order.SupplierCode, amount)
	if err != nil {
		return nil, err
	}
	if decision.LimitID != nil {
		out.AppliedCeiling = decision.Ceiling
	}
	switch {
	case decision.Blocked:
		order.Status = poentity.PurchaseOrderStatusREQUESTED
		order.AlcadaStatus = alcadaRejected
		out.Blocked = true
		out.AlcadaStatus = alcadaRejected
		out.Message = fmt.Sprintf("valor %.2f acima do teto absoluto de alçada; pedido não pode ser aprovado", amount)
		return uc.persist(ctx, order, out)
	case !decision.AutoApprove:
		order.Status = poentity.PurchaseOrderStatusREQUESTED
		order.AlcadaStatus = alcadaBlocked
		out.RequiresAuthorization = true
		out.AlcadaStatus = alcadaBlocked
		out.Message = fmt.Sprintf("valor %.2f acima do limite de alçada (%.2f); requer autorização", amount, decision.Ceiling)
		return uc.persist(ctx, order, out)
	default:
		return uc.finishApproval(ctx, order, out)
	}
}

// Authorize releases a purchase order that is blocked pending alçada authorization.
// It is meant to be gated to a higher authority at the route level.
func (uc *ApprovePurchaseOrderUseCase) Authorize(ctx context.Context, code int64) (*response.ApprovePurchaseOrderResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	order, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if order.AlcadaStatus != alcadaBlocked {
		return nil, fmt.Errorf("purchase order %d is not pending alçada authorization (alcada_status=%s)", code, order.AlcadaStatus)
	}
	out := &response.ApprovePurchaseOrderResponse{Message: "pedido autorizado e aprovado"}
	return uc.finishApproval(ctx, order, out)
}

func (uc *ApprovePurchaseOrderUseCase) finishApproval(ctx context.Context, order *poentity.PurchaseOrder, out *response.ApprovePurchaseOrderResponse) (*response.ApprovePurchaseOrderResponse, error) {
	order.Status = poentity.PurchaseOrderStatusAPPROVED
	order.AlcadaStatus = alcadaApproved
	out.Approved = true
	out.AlcadaStatus = alcadaApproved
	if out.Message == "" {
		out.Message = "pedido aprovado"
	}
	return uc.persist(ctx, order, out)
}

func (uc *ApprovePurchaseOrderUseCase) persist(ctx context.Context, order *poentity.PurchaseOrder, out *response.ApprovePurchaseOrderResponse) (*response.ApprovePurchaseOrderResponse, error) {
	updated, err := uc.Repo.Update(ctx, order)
	if err != nil {
		return nil, err
	}
	updated.Items, err = uc.Repo.ListItems(ctx, updated.Code)
	if err != nil {
		return nil, err
	}
	out.PurchaseOrder = toPurchaseOrderResponse(updated)
	return out, nil
}
