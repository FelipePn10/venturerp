package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

type AuthService struct{}

func (a *AuthService) hasWriteRole(ctx context.Context) bool {
	user, ok := ctx.Value(contextkey.UserKey).(*security.AuthUser)
	if !ok {
		return false
	}

	role := strings.ToUpper(strings.TrimSpace(user.Role))
	return role == "ADMIN" || role == "USER"
}

func (a *AuthService) CanCreateComponent(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateItem(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateProduct(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateBom(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateBomItems(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanAssociateByQuestionProduct(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateQuestion(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateQuestionOption(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanDeleteProduct(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateWarehouse(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateGroup(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateEnterprise(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateModifier(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateEmployee(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGenerateMaskForItem(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateStructure(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) UpdateStructure(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) GetStructureTree(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) GetAllStructure(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) ResolveStructureForMask(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanResolveStructure(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) FindItemByCode(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) UserID(ctx context.Context) (uuid.UUID, error) {
	user, ok := ctx.Value(contextkey.UserKey).(*security.AuthUser)
	if !ok {
		return uuid.Nil, errors.New("unauthenticated request")
	}
	id, err := uuid.Parse(user.ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user id in context: %w", err)
	}
	return id, nil
}

func (a *AuthService) CreateAllocation(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) ListAllocation(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateCostCenter(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListCostCenter(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanGetCostCenter(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateDeliveryReschedule(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListDeliveryReschedule(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanCreateIndependentDemand(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanListIndependentDemand(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanViewIndependentDemand(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanUpdateIndependentDemand(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanDeleteIndependentDemand(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}

func (a *AuthService) CanManageIndustrialCalendar(ctx context.Context) bool {
	return a.hasWriteRole(ctx)
}
