// Package itemrevisoes leva ao cadastro a revisão de item que começou a valer.
package itemrevisoes

import (
	"context"
	"time"
)

type promotor interface {
	PromoverRevisoes(ctx context.Context) (int64, error)
}

type registrador interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// Scheduler confere ao subir e depois de hora em hora. A OF não depende dele: o
// gatilho já carimba a revisão vigente na data.
type Scheduler struct {
	repo     promotor
	logger   registrador
	interval time.Duration
}

func NewScheduler(repo promotor, logger registrador) *Scheduler {
	return &Scheduler{repo: repo, logger: logger, interval: time.Hour}
}

func (s *Scheduler) Run(ctx context.Context) {
	s.tick(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	n, err := s.repo.PromoverRevisoes(ctx)
	if err != nil {
		s.logger.Error("falha ao promover as revisões de item", "error", err)
		return
	}
	if n > 0 {
		s.logger.Info("revisões de item promovidas", "itens", n)
	}
}
