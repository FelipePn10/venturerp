// Package materiaisof completa, uma vez ao subir a API, a lista de materiais
// das OFs abertas ou em produção que nasceram sem ela (antes de a lista existir
// ou por carga direta no banco). Depois de completadas elas saem da fila, então
// as próximas subidas não fazem nada.
package materiaisof

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

type fila interface {
	OFsSemMateriais(ctx context.Context, max int) ([]entity.OFSemMateriais, error)
}

type completador interface {
	Completar(ctx context.Context, of entity.OFSemMateriais) (int, error)
}

type registrador interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

type Rotina struct {
	fila   fila
	uc     completador
	logger registrador
	lote   int
}

func NewRotina(f fila, uc completador, logger registrador) *Rotina {
	return &Rotina{fila: f, uc: uc, logger: logger, lote: 200}
}

// Run percorre a fila em lotes até esvaziar. OF que falhar (estrutura com
// erro, por exemplo) fica registrada no log e não trava as demais; para não
// repetir a mesma OF para sempre, o lote seguinte começa depois dela.
func (r *Rotina) Run(ctx context.Context) (completadas, materiais int) {
	falhas := map[int64]bool{}
	for ctx.Err() == nil {
		fila, err := r.fila.OFsSemMateriais(ctx, r.lote+len(falhas))
		if err != nil {
			r.logger.Error("falha ao listar as OFs sem lista de materiais", "error", err)
			return
		}
		novas := 0
		for _, of := range fila {
			if ctx.Err() != nil || falhas[of.ID] {
				continue
			}
			novas++
			// Sessão da empresa da OF: mantém o isolamento por empresa.
			ectx := context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{
				ID: of.CreatedBy.String(), Role: "ADMIN", EnterpriseID: of.EnterpriseID, EnterpriseCode: of.EnterpriseCode,
			})
			n, err := r.uc.Completar(ectx, of)
			if err != nil {
				falhas[of.ID] = true
				r.logger.Error("não foi possível completar a lista de materiais da OF", "empresa", of.EnterpriseID, "of", of.ID, "error", err)
				continue
			}
			if n == 0 {
				falhas[of.ID] = true // sem componente vigente: não volta para a fila nesta subida
				continue
			}
			completadas++
			materiais += n
		}
		if novas == 0 {
			break
		}
	}
	if completadas > 0 {
		r.logger.Info("listas de materiais completadas pela estrutura", "ofs", completadas, "materiais", materiais)
	}
	return
}
