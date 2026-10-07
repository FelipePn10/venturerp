package dfe

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

type logNulo struct{ erros int }

func (l *logNulo) Info(string, ...any)  {}
func (l *logNulo) Error(string, ...any) { l.erros++ }

type reservaFixa struct {
	empresas  []repository.EmpresaDFe
	intervalo time.Duration
}

func (r *reservaFixa) ReservarSincronizacaoDFe(_ context.Context, i time.Duration) ([]repository.EmpresaDFe, error) {
	r.intervalo = i
	return r.empresas, nil
}

type sincronizadorFake struct {
	vistas []security.AuthUser
	falha  int64
}

func (s *sincronizadorFake) Sincronizar(ctx context.Context) (*fiscal_uc.SincronizacaoDFe, error) {
	u := ctx.Value(contextkey.UserKey).(*security.AuthUser)
	s.vistas = append(s.vistas, *u)
	if u.EnterpriseID == s.falha {
		return nil, errors.New("CNPJ do emitente não autorizado")
	}
	return &fiscal_uc.SincronizacaoDFe{Novas: 2}, nil
}

// Cada empresa é sincronizada na sessão dela (isolamento), e a falha de uma não
// impede as outras.
func TestCicloSincronizaCadaEmpresaNaSuaSessao(t *testing.T) {
	ator := uuid.New()
	res := &reservaFixa{empresas: []repository.EmpresaDFe{
		{EnterpriseID: 1, EnterpriseCode: 1, Ator: ator},
		{EnterpriseID: 7, EnterpriseCode: 7702},
		{EnterpriseID: 9, EnterpriseCode: 90},
	}}
	uc := &sincronizadorFake{falha: 7}
	log := &logNulo{}
	ok := NewScheduler(uc, res, log).Ciclo(context.Background())
	if ok != 2 || log.erros != 1 {
		t.Fatalf("sincronizadas %d (esperado 2), erros registrados %d", ok, log.erros)
	}
	if res.intervalo != time.Hour {
		t.Fatalf("intervalo da reserva %v, esperado 1h", res.intervalo)
	}
	if len(uc.vistas) != 3 || uc.vistas[1].EnterpriseID != 7 || uc.vistas[1].EnterpriseCode != 7702 || uc.vistas[0].ID != ator.String() || uc.vistas[2].Role != "ADMIN" {
		t.Fatalf("sessões montadas: %+v", uc.vistas)
	}
}
