package customer_material_uc

import (
	"context"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/customer_material/repository"
)

// RelatorioDeMovimentos é a movimentação de material de terceiro (entrada e
// saída) pedida no levantamento da Usimac para a rastreabilidade fiscal.
func (uc *UseCase) RelatorioDeMovimentos(ctx context.Context, f domrepo.FiltroMovimentos) ([]*entity.MovimentoDoRelatorio, error) {
	if f.De != nil && f.Ate != nil && f.Ate.Before(*f.De) {
		return nil, errorsuc.NewValidationError("a data final não pode ser anterior à inicial")
	}
	for _, t := range f.Tipos {
		switch entity.TipoMovimento(t) {
		case entity.MovimentoRecebimento, entity.MovimentoRetorno, entity.MovimentoSobra,
			entity.MovimentoSucata, entity.MovimentoAjuste:
		default:
			return nil, errorsuc.NewValidationError("tipo de movimento não reconhecido: escolha um da lista")
		}
	}
	return uc.Repo.RelatorioDeMovimentos(ctx, f)
}

// HorasDeBeneficiamento é o apontamento "apenas horas" das OFs de
// beneficiamento, por cliente e pedido.
func (uc *UseCase) HorasDeBeneficiamento(ctx context.Context, f domrepo.FiltroHoras) ([]*entity.HorasDeBeneficiamento, error) {
	if f.De != nil && f.Ate != nil && f.Ate.Before(*f.De) {
		return nil, errorsuc.NewValidationError("a data final não pode ser anterior à inicial")
	}
	return uc.Repo.HorasDeBeneficiamento(ctx, f)
}
