// Package custoof apura, uma vez ao subir a API, o custo real das OFs que já
// se moveram e nunca tiveram custo (movidas antes de o custo ser automático,
// ou por carga direta no banco). Sem isso a aba Custo dizia "esta ordem não
// teve movimento" para OF concluída e entregue. Apurada, a OF sai da fila; as
// próximas subidas não fazem nada.
package custoof

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

type fila interface {
	OFsSemCusto(ctx context.Context, max int) ([]entity.OFSemCusto, error)
}

type apurador interface {
	Atualizar(ctx context.Context, productionOrderID int64) error
}

type registrador interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

type Rotina struct {
	fila   fila
	uc     apurador
	logger registrador
	lote   int
}

func NewRotina(f fila, uc apurador, logger registrador) *Rotina {
	return &Rotina{fila: f, uc: uc, logger: logger, lote: 200}
}

// Run percorre a fila em lotes até esvaziar. OF que falhar fica marcada com
// custo pendente (a tela mostra o motivo), vai para o log e não trava as
// demais nem volta nesta subida.
func (r *Rotina) Run(ctx context.Context) (apuradas int) {
	falhas := map[int64]bool{}
	for ctx.Err() == nil {
		fila, err := r.fila.OFsSemCusto(ctx, r.lote+len(falhas))
		if err != nil {
			r.logger.Error("falha ao listar as OFs sem custo apurado", "error", err)
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
			if err := r.uc.Atualizar(ectx, of.ID); err != nil {
				falhas[of.ID] = true
				r.logger.Error("não foi possível apurar o custo da OF", "empresa", of.EnterpriseID, "of", of.ID, "error", err)
				continue
			}
			apuradas++
		}
		if novas == 0 {
			break
		}
	}
	if apuradas > 0 {
		r.logger.Info("custo real apurado das OFs antigas", "ofs", apuradas)
	}
	return
}
