// Package xmlnotas completa, em segundo plano, a cópia do XML das notas
// emitidas que ainda não a têm no banco: as autorizadas antes de o ERP guardar
// o XML e as que a Focus não entregou na hora da autorização ou do cancelamento.
package xmlnotas

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/nfse_uc"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

type fila interface {
	NotasSemXML(ctx context.Context, max int) ([]fiscalrepo.NotaSemXML, error)
}

type xmlDaNFe interface {
	Execute(ctx context.Context, id int64) (*fiscal_uc.ExitXMLResult, error)
}

type xmlDaNFSe interface {
	Execute(ctx context.Context, id int64) (*nfse_uc.NFSeXMLResult, error)
}

type registrador interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// Scheduler baixa poucas notas por ciclo, com folga entre elas, para não
// disputar a cota da Focus com quem está emitindo nota na tela.
type Scheduler struct {
	fila     fila
	nfe      xmlDaNFe
	nfse     xmlDaNFSe
	logger   registrador
	interval time.Duration
	porCiclo int
	espaco   time.Duration
}

func NewScheduler(f fila, nfe xmlDaNFe, nfse xmlDaNFSe, logger registrador) *Scheduler {
	return &Scheduler{fila: f, nfe: nfe, nfse: nfse, logger: logger, interval: 10 * time.Minute, porCiclo: 30, espaco: time.Second}
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

// Ciclo devolve quantas notas ficaram com o XML guardado.
func (s *Scheduler) Ciclo(ctx context.Context) int {
	alvos, err := s.fila.NotasSemXML(ctx, s.porCiclo)
	if err != nil {
		s.logger.Error("falha ao listar notas sem XML", "error", err)
		return 0
	}
	ok := 0
	for i, a := range alvos {
		if i > 0 && s.espaco > 0 {
			select {
			case <-ctx.Done():
				return ok
			case <-time.After(s.espaco):
			}
		}
		// Sessão da empresa da nota: mantém o isolamento por empresa.
		ectx := context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{
			ID: a.Actor.String(), Role: "ADMIN", EnterpriseID: a.EnterpriseID, EnterpriseCode: a.EnterpriseCode,
		})
		if a.Tipo == "NFSE" {
			_, err = s.nfse.Execute(ectx, a.ID)
		} else {
			_, err = s.nfe.Execute(ectx, a.ID)
		}
		if err != nil {
			s.logger.Error("XML da nota emitida não foi guardado; nova tentativa amanhã", "tipo", a.Tipo, "empresa", a.EnterpriseID, "nota", a.ID, "error", err)
			continue
		}
		ok++
	}
	if ok > 0 {
		s.logger.Info("XML de notas emitidas guardado", "notas", ok)
	}
	return ok
}
