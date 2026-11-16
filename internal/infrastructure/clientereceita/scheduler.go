// Package clientereceita reconsulta na Receita a situação cadastral dos
// clientes CNPJ e bloqueia quem deixou de estar ATIVA.
package clientereceita

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

type fila interface {
	ListCustomersForRegistrationCheck(ctx context.Context, max int) ([]entity.RegistrationCheckTarget, error)
}

type reconsulta interface {
	RecheckRegistration(ctx context.Context, code int64) error
}

type registrador interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// Scheduler roda de hora em hora e confere poucos clientes por vez, com
// intervalo entre as consultas: as APIs públicas de CNPJ aceitam só algumas
// consultas por minuto, e a tela do usuário não pode ficar sem cota.
type Scheduler struct {
	fila     fila
	uc       reconsulta
	logger   registrador
	interval time.Duration
	porCiclo int
	espaco   time.Duration
}

func NewScheduler(f fila, uc reconsulta, logger registrador) *Scheduler {
	return &Scheduler{fila: f, uc: uc, logger: logger, interval: time.Hour, porCiclo: 10, espaco: 30 * time.Second}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Ciclo(ctx)
		}
	}
}

// Ciclo devolve quantos clientes foram conferidos.
func (s *Scheduler) Ciclo(ctx context.Context) int {
	alvos, err := s.fila.ListCustomersForRegistrationCheck(ctx, s.porCiclo)
	if err != nil {
		s.logger.Error("falha ao listar clientes para reconsulta na Receita", "error", err)
		return 0
	}
	ok := 0
	for i, a := range alvos {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ok
			case <-time.After(s.espaco):
			}
		}
		// Sessão da empresa do cliente: mantém o isolamento por empresa.
		ectx := context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{
			ID: a.Actor.String(), Role: "ADMIN", EnterpriseID: a.EnterpriseID, EnterpriseCode: a.EnterpriseCode,
		})
		if err := s.uc.RecheckRegistration(ectx, a.Code); err != nil {
			s.logger.Error("reconsulta do cliente na Receita falhou", "empresa", a.EnterpriseID, "cliente", a.Code, "error", err)
			continue
		}
		ok++
	}
	if ok > 0 {
		s.logger.Info("situação cadastral de clientes conferida na Receita", "clientes", ok)
	}
	return ok
}
