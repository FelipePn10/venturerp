package fiscal_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

// LeitorEfeitosTipoNota lê do cadastro o que o tipo de nota faz na autorização.
type LeitorEfeitosTipoNota interface {
	EfeitosDoTipoNota(ctx context.Context, id int64) (entity.EfeitosDoTipoNota, error)
}

// efeitosDaNota: os efeitos do tipo escolhido na nota; sem tipo, venda comum.
func efeitosDaNota(ctx context.Context, leitor LeitorEfeitosTipoNota, exit *entity.FiscalExit) (entity.EfeitosDoTipoNota, error) {
	if leitor == nil || exit == nil || exit.InvoiceTypeID == nil {
		return entity.EfeitosPadraoDaNota(), nil
	}
	return leitor.EfeitosDoTipoNota(ctx, *exit.InvoiceTypeID)
}

// planoDoTipoNota: tipo de nota que não gera título (bonificação, remessa) não tem
// cobrança nem duplicata: a NF-e declara "sem pagamento". Sem isto o cliente recebia
// uma nota dizendo que deve e um título a receber que ninguém queria.
func planoDoTipoNota(plano PlanoDaNota, efeitos entity.EfeitosDoTipoNota) PlanoDaNota {
	if efeitos.GeraTitulo {
		return plano
	}
	plano.SemCobranca, plano.Origem, plano.Parcelas = true, "SEM_COBRANCA", nil
	return plano
}
