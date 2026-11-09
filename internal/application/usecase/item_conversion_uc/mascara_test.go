package item_conversion_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/repository"
	"strings"
)

type repoComMascaras struct {
	repository.ManagedItemConversionRepository
	mascaras map[string]bool
}

func (r repoComMascaras) MaskExists(_ context.Context, _ int64, mask string) (bool, error) {
	return r.mascaras[mask], nil
}

func TestChecarMascara(t *testing.T) {
	uc := &ItemConversionUseCase{repo: repoComMascaras{mascaras: map[string]bool{"AZ-01": true}}}
	ctx := context.Background()
	if err := uc.checarMascara(ctx, 1, ""); err != nil {
		t.Fatalf("sem máscara deveria passar: %v", err)
	}
	if err := uc.checarMascara(ctx, 1, "AZ-01"); err != nil {
		t.Fatalf("máscara existente recusada: %v", err)
	}
	err := uc.checarMascara(ctx, 1, "AZ-99")
	if err == nil || !strings.Contains(err.Error(), "AZ-99") {
		t.Fatalf("esperava erro de validação para máscara inexistente, veio %v", err)
	}
}
