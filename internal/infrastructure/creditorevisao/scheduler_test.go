package creditorevisao

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

type filaFake []entity.RegistrationCheckTarget

func (f filaFake) ListCustomersForAutoCreditReview(context.Context, int) ([]entity.RegistrationCheckTarget, error) {
	return f, nil
}

type revisorFake struct{ empresas map[int64]int64 }

func (r *revisorFake) RevisarCreditoAutomatico(ctx context.Context, code int64) (bool, error) {
	u, _ := ctx.Value(contextkey.UserKey).(*security.AuthUser)
	r.empresas[code] = u.EnterpriseID
	if code == 2 {
		return false, errors.New("falhou")
	}
	return code == 3, nil
}

type logMudo struct{}

func (logMudo) Info(string, ...any)  {}
func (logMudo) Error(string, ...any) {}

// Cada cliente é revisto na sessão da SUA empresa, e a falha de um não para o lote.
func TestCicloSessaoDaEmpresaEFalhaNaoPara(t *testing.T) {
	rev := &revisorFake{empresas: map[int64]int64{}}
	s := NewScheduler(filaFake{{EnterpriseID: 10, Code: 1}, {EnterpriseID: 20, Code: 2}, {EnterpriseID: 30, Code: 3}}, rev, logMudo{})
	revistos, mudaram := s.Ciclo(context.Background())
	if revistos != 2 || mudaram != 1 {
		t.Fatalf("revistos %d mudaram %d, quer 2 e 1", revistos, mudaram)
	}
	if rev.empresas[1] != 10 || rev.empresas[2] != 20 || rev.empresas[3] != 30 {
		t.Fatalf("sessão com empresa errada: %v", rev.empresas)
	}
}
