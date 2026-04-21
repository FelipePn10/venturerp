package ports

import (
	"context"

	"github.com/google/uuid"
)

type AuthService interface {
	CanCreateComponent(ctx context.Context) bool
	CanCreateProduct(ctx context.Context) bool
	CanCreateBom(ctx context.Context) bool
	CanCreateBomItems(ctx context.Context) bool
	CanAssociateByQuestionProduct(ctx context.Context) bool
	CanCreateQuestion(ctx context.Context) bool
	CanCreateQuestionOption(ctx context.Context) bool
	CanDeleteProduct(ctx context.Context) bool
	CanCreateItem(ctx context.Context) bool
	CanCreateWarehouse(ctx context.Context) bool
	CanCreateGroup(ctx context.Context) bool
	CanCreateEnterprise(ctx context.Context) bool
	CanCreateModifier(ctx context.Context) bool
	CanCreateEmployee(ctx context.Context) bool
	CanGenerateMaskForItem(ctx context.Context) bool
	CanCreateStructure(ctx context.Context) bool
	UpdateStructure(ctx context.Context) bool
	GetStructureTree(ctx context.Context) bool
	GetAllStructure(ctx context.Context) bool
	ResolveStructureForMask(ctx context.Context) bool
	FindItemByCode(ctx context.Context) bool
	CanResolveStructure(ctx context.Context) bool
	UserID(ctx context.Context) (uuid.UUID, error)
}
