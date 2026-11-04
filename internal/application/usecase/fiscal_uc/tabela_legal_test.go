package fiscal_uc

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

type authTabela struct {
	ports.AuthService
	escreve, admin bool
}

func (a authTabela) CanManageFiscalConfig(context.Context) bool { return a.escreve }
func (a authTabela) IsAdmin(context.Context) bool               { return a.admin }

// As tabelas legais da VFIS0110 valem para todas as empresas: só o
// administrador as altera; o usuário fiscal comum apenas consulta.
func TestTabelaLegalSoAdministradorEdita(t *testing.T) {
	ctx := context.Background()
	if podeEditarTabelaLegal(ctx, authTabela{escreve: true}) {
		t.Fatal("usuário comum não pode alterar a tabela legal de todas as empresas")
	}
	if !podeEditarTabelaLegal(ctx, authTabela{escreve: true, admin: true}) {
		t.Fatal("administrador deveria editar")
	}
	if err := (&UpsertICMSInternalUseCase{Auth: authTabela{escreve: true}}).Execute(ctx, request.UpsertICMSInternalDTO{UF: "SP", AliqICMS: decimal.NewFromFloat(18)}); err == nil {
		t.Fatal("o use case deveria recusar o usuário comum")
	}
}
