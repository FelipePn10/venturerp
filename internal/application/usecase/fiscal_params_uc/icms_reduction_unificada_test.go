package fiscal_params_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

// A VFIS0330 foi unificada na VFIS0320: gravar regra no cadastro antigo criaria
// uma regra que nenhuma nota usa, então a gravação é recusada apontando a tela
// nova — sem tocar no repositório.
func TestReducaoICMSRecusaGravacaoNoCadastroAntigo(t *testing.T) {
	uc := &ICMSReductionSubstitutionUseCase{} // Repo nil: nada pode chegar a ele
	regra := &entity.ICMSReductionSubstitution{ID: 1, UF: "PR"}
	for nome, gravar := range map[string]func() error{
		"incluir": func() error { _, err := uc.Create(context.Background(), regra); return err },
		"alterar": func() error { _, err := uc.Update(context.Background(), regra); return err },
	} {
		err := gravar()
		if err == nil || !strings.Contains(err.Error(), "VFIS0320") {
			t.Fatalf("%s: erro %v, quer recusa apontando a VFIS0320", nome, err)
		}
	}
}
