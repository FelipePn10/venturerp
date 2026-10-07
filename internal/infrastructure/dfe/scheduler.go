// Package dfe contém o agendador da sincronização das NF-e recebidas
// (distribuição DF-e da SEFAZ, via Focus NF-e).
package dfe

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

type registrador interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// Sincronizador é o caso de uso que consulta a SEFAZ para a empresa da sessão.
type Sincronizador interface {
	Sincronizar(ctx context.Context) (*fiscal_uc.SincronizacaoDFe, error)
}

// Reservador escolhe as empresas da vez (atomicamente, entre instâncias).
type Reservador interface {
	ReservarSincronizacaoDFe(ctx context.Context, intervalo time.Duration) ([]repository.EmpresaDFe, error)
}

// Scheduler acorda a cada `tick` e sincroniza as empresas cuja última
// tentativa passou do intervalo. A vez é reservada no banco — com várias
// instâncias da API, cada empresa é consultada por uma só — e uma empresa com
// erro (token inválido, SEFAZ fora) não impede as outras: o erro fica
// registrado na configuração fiscal e aparece na tela.
type Scheduler struct {
	uc        Sincronizador
	repo      Reservador
	logger    registrador
	tick      time.Duration
	intervalo time.Duration
}

func NewScheduler(uc Sincronizador, repo Reservador, logger registrador) *Scheduler {
	return &Scheduler{uc: uc, repo: repo, logger: logger, tick: 5 * time.Minute, intervalo: fiscal_uc.IntervaloSincronizacaoDFe}
}

// Run bloqueia até o contexto ser cancelado.
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.tick)
	defer ticker.Stop()
	s.logger.Info("agendador das NF-e recebidas iniciado", "intervalo", s.intervalo.String())
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("agendador das NF-e recebidas encerrado")
			return
		case <-ticker.C:
			s.Ciclo(ctx)
		}
	}
}

// Ciclo faz uma rodada: reserva as empresas da vez e sincroniza cada uma.
// Devolve quantas foram sincronizadas sem erro.
func (s *Scheduler) Ciclo(ctx context.Context) int {
	empresas, err := s.repo.ReservarSincronizacaoDFe(ctx, s.intervalo)
	if err != nil {
		s.logger.Error("falha ao escolher as empresas para sincronizar as NF-e recebidas", "error", err)
		return 0
	}
	ok := 0
	for _, e := range empresas {
		// Sessão da própria empresa, como se um administrador dela clicasse em
		// "Buscar na SEFAZ": é o que mantém o isolamento por empresa.
		ectx := context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{
			ID: e.Ator.String(), Role: "ADMIN", EnterpriseID: e.EnterpriseID, EnterpriseCode: e.EnterpriseCode,
		})
		ectx, cancel := context.WithTimeout(ectx, 5*time.Minute)
		r, err := s.uc.Sincronizar(ectx)
		cancel()
		if err != nil {
			s.logger.Error("sincronização das NF-e recebidas falhou", "empresa", e.EnterpriseID, "error", err)
			continue
		}
		ok++
		if r.Novas > 0 {
			s.logger.Info("NF-e recebidas sincronizadas", "empresa", e.EnterpriseID, "novas", r.Novas)
		}
	}
	return ok
}
