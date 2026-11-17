// Package creditorevisao revê sozinho o limite de crédito dos clientes no
// intervalo da política (padrão: a cada 3 meses), pelo comportamento de
// pagamento e de compra do período.
package creditorevisao

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

type fila interface {
	ListCustomersForAutoCreditReview(ctx context.Context, max int) ([]entity.RegistrationCheckTarget, error)
}

type revisor interface {
	RevisarCreditoAutomatico(ctx context.Context, code int64) (bool, error)
}

type registrador interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// Scheduler roda de hora em hora e revê um lote por ciclo. Não consulta a
// Receita (não gasta a cota): usa a situação gravada pelo agendador dela.
type Scheduler struct {
	fila     fila
	uc       revisor
	logger   registrador
	interval time.Duration
	porCiclo int
}

func NewScheduler(f fila, uc revisor, logger registrador) *Scheduler {
	return &Scheduler{fila: f, uc: uc, logger: logger, interval: time.Hour, porCiclo: 200}
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

// Ciclo devolve quantos clientes foram revistos e em quantos o limite mudou.
func (s *Scheduler) Ciclo(ctx context.Context) (revistos, mudaram int) {
	alvos, err := s.fila.ListCustomersForAutoCreditReview(ctx, s.porCiclo)
	if err != nil {
		s.logger.Error("falha ao listar clientes para a revisão de crédito", "error", err)
		return 0, 0
	}
	for _, a := range alvos {
		if ctx.Err() != nil {
			return
		}
		// Sessão da empresa do cliente: mantém o isolamento por empresa.
		ectx := context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{
			ID: a.Actor.String(), Role: "ADMIN", EnterpriseID: a.EnterpriseID, EnterpriseCode: a.EnterpriseCode,
		})
		mudou, err := s.uc.RevisarCreditoAutomatico(ectx, a.Code)
		if err != nil {
			s.logger.Error("revisão automática de crédito falhou", "empresa", a.EnterpriseID, "cliente", a.Code, "error", err)
			continue
		}
		revistos++
		if mudou {
			mudaram++
		}
	}
	if revistos > 0 {
		s.logger.Info("revisão automática de crédito", "revistos", revistos, "limites_alterados", mudaram)
	}
	return
}
