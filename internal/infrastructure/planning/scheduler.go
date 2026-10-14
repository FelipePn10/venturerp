// Package planning contém o agendador da janela noturna do planejamento.
package planning

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planning_uc"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// registrador é o mínimo que o agendador precisa de um log; aceitar a interface
// em vez do tipo concreto mantém o pacote testável e desacoplado.
type registrador interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// Scheduler dispara o planejamento na janela configurada por empresa.
//
// Ele acorda de minuto em minuto e pergunta ao caso de uso quais empresas
// venceram — em vez de dormir até o próximo horário. Assim uma mudança de
// parametrização vale já no próximo minuto, e um processo que ficou parado
// (deploy, suspensão da máquina) não perde a janela: ao voltar, a execução do
// dia ainda está pendente e roda.
type Scheduler struct {
	uc       *planning_uc.AutoRunPlanningUseCase
	logger   registrador
	interval time.Duration
}

func NewScheduler(uc *planning_uc.AutoRunPlanningUseCase, logger registrador) *Scheduler {
	return &Scheduler{uc: uc, logger: logger, interval: time.Minute}
}

// Run bloqueia até o contexto ser cancelado.
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	s.logger.Info("agendador do planejamento iniciado", "intervalo", s.interval.String())
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("agendador do planejamento encerrado")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	// Um ciclo do planejamento pode ser longo; o limite evita que uma execução
	// travada segure o agendador para sempre.
	ctx, cancel := context.WithTimeout(ctx, 2*time.Hour)
	defer cancel()

	executadas, err := s.uc.RunDue(ctx)
	if err != nil {
		s.logger.Error("falha ao verificar a janela do planejamento", "error", err)
		return
	}
	if executadas > 0 {
		s.logger.Info("planejamento executado na janela noturna", "empresas", executadas)
	}
}

// SessaoDaEmpresa monta a sessão em que a execução automática roda: a empresa
// da parametrização e o usuário que a salvou, como se ele tivesse clicado em
// "Calcular". É o mesmo desenho da sincronização das NF-e recebidas e o que
// mantém o isolamento — o MRP lê a empresa da sessão, nunca de um parâmetro.
func SessaoDaEmpresa(ctx context.Context, s planning_uc.AutoRunSettings) context.Context {
	ator := ""
	if s.UpdatedBy != nil {
		ator = s.UpdatedBy.String()
	}
	return context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{
		ID: ator, Role: "ADMIN", EnterpriseID: s.EnterpriseID, EnterpriseCode: s.EnterpriseCode,
	})
}
