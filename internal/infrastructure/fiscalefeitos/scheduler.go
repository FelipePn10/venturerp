// Package fiscalefeitos reprocessa, em segundo plano, os efeitos de NF-e que
// ficaram pendentes (estoque, títulos, pedido) depois da autorização ou do
// cancelamento na SEFAZ.
package fiscalefeitos

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

type registrador interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// Scheduler acorda de tempos em tempos e reprocessa as pendências vencidas
// (a espera entre tentativas cresce a cada falha, até 60 minutos).
type Scheduler struct {
	efeitos  *fiscal_uc.EfeitosNFe
	logger   registrador
	interval time.Duration
}

func NewScheduler(efeitos *fiscal_uc.EfeitosNFe, logger registrador) *Scheduler {
	return &Scheduler{efeitos: efeitos, logger: logger, interval: time.Minute}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	s.logger.Info("reprocesso das pendências de NF-e iniciado", "intervalo", s.interval.String())
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("reprocesso das pendências de NF-e encerrado")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	resolvidas, falhas, err := s.efeitos.ProcessarVencidas(ctx, 100, Sessao)
	if err != nil {
		s.logger.Error("falha ao ler a fila de pendências de NF-e", "error", err)
		return
	}
	if resolvidas > 0 || falhas > 0 {
		s.logger.Info("pendências de NF-e reprocessadas", "resolvidas", resolvidas, "continuam_pendentes", falhas)
	}
	if falhas > 0 {
		s.logger.Error("pendências de NF-e seguem sem aplicar — confira a nota", "quantidade", falhas)
	}
}

// Sessao monta a sessão em que a pendência roda: a empresa dona da nota e o
// usuário que a autorizou/cancelou — o mesmo desenho do planejamento
// automático, que mantém o isolamento (o ERP lê a empresa da sessão).
func Sessao(ctx context.Context, p entity.PendenciaVencida) context.Context {
	ator := ""
	if p.CriadoPor != nil {
		ator = p.CriadoPor.String()
	}
	return context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{
		ID: ator, Role: "ADMIN", EnterpriseID: p.EnterpriseID, EnterpriseCode: p.EnterpriseCode,
	})
}
