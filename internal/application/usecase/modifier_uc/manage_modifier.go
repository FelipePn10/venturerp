package modifier_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/modifier/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/modifier/repository"
)

// GetModifierUseCase fetches a single PDM modifier by its id.
type GetModifierUseCase struct {
	Repo repository.ModifierRepository
	Auth ports.AuthService
}

func NewGetModifierUseCase(repo repository.ModifierRepository, auth ports.AuthService) *GetModifierUseCase {
	return &GetModifierUseCase{Repo: repo, Auth: auth}
}

func (uc *GetModifierUseCase) Execute(ctx context.Context, id int) (*response.ModifierResponse, error) {
	m, err := uc.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toModifierResponse(m), nil
}

// ListModifiersUseCase returns all PDM modifiers.
type ListModifiersUseCase struct {
	Repo repository.ModifierRepository
	Auth ports.AuthService
}

func NewListModifiersUseCase(repo repository.ModifierRepository, auth ports.AuthService) *ListModifiersUseCase {
	return &ListModifiersUseCase{Repo: repo, Auth: auth}
}

func (uc *ListModifiersUseCase) Execute(ctx context.Context) ([]*response.ModifierResponse, error) {
	items, err := uc.Repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ModifierResponse, 0, len(items))
	for _, m := range items {
		out = append(out, toModifierResponse(m))
	}
	return out, nil
}

// UpdateModifierUseCase changes a modifier's description.
type UpdateModifierUseCase struct {
	Repo repository.ModifierRepository
	Auth ports.AuthService
}

func NewUpdateModifierUseCase(repo repository.ModifierRepository, auth ports.AuthService) *UpdateModifierUseCase {
	return &UpdateModifierUseCase{Repo: repo, Auth: auth}
}

func (uc *UpdateModifierUseCase) Execute(ctx context.Context, id int, description string, ativo *bool) (*response.ModifierResponse, error) {
	if !uc.Auth.CanCreateModifier(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	description = strings.TrimSpace(description)
	if description == "" {
		return nil, errorsuc.NewValidationError("informe a descrição")
	}
	atual, err := uc.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Só confere quando o nome muda: o duplicado antigo precisa poder ser inativado.
	if !strings.EqualFold(strings.TrimSpace(atual.Description), description) {
		if err = descricaoLivre(ctx, uc.Repo, description, id); err != nil {
			return nil, err
		}
	}
	situacao := atual.IsActive
	if ativo != nil {
		situacao = *ativo
	}
	updated, err := uc.Repo.Update(ctx, &entity.Modifier{ID: id, Description: description, IsActive: situacao})
	if err != nil {
		return nil, err
	}
	return toModifierResponse(updated), nil
}

// descricaoLivre recusa dois modificadores com o mesmo nome na empresa.
func descricaoLivre(ctx context.Context, repo repository.ModifierRepository, descricao string, idAtual int) error {
	mods, err := repo.List(ctx)
	if err != nil {
		return err
	}
	alvo := strings.ToUpper(strings.TrimSpace(descricao))
	for _, m := range mods {
		if m.ID != idAtual && strings.ToUpper(strings.TrimSpace(m.Description)) == alvo {
			return errorsuc.NewValidationError(fmt.Sprintf("já existe o modificador %d com esta descrição", m.ID))
		}
	}
	return nil
}
