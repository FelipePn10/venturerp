// Package sales_commission_uc é o rateio de comissão do pedido e do orçamento.
package sales_commission_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_commission/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_commission/repository"
	"github.com/shopspring/decimal"
)

type UseCase struct {
	Repo domrepo.Repository
}

func (uc *UseCase) Listar(ctx context.Context, doc entity.Documento, documentCode int64) (*response.RateioComissaoResponse, error) {
	if documentCode <= 0 {
		return nil, errorsuc.NewValidationError("informe o documento")
	}
	totais, err := uc.Repo.Totais(ctx, doc, documentCode)
	if err != nil {
		return nil, err
	}
	linhas, err := uc.Repo.Listar(ctx, doc, documentCode)
	if err != nil {
		return nil, err
	}
	// Documento sem rateio, mas com representante na capa: a tela abre com essa
	// linha já montada (id 0 = ainda não gravada). Sem isso o usuário teria de
	// redigitar o que o documento já sabe, e o rateio nasceria divergente da capa.
	if len(linhas) == 0 && totais.RepresentativeCode != nil {
		nomes, err := uc.Repo.RepresentantesAtivos(ctx, []int64{*totais.RepresentativeCode})
		if err != nil {
			return nil, err
		}
		linhas = []*entity.Rateio{{
			DocumentCode:       documentCode,
			RepresentativeCode: *totais.RepresentativeCode,
			RepresentativeName: nomes[*totais.RepresentativeCode],
			Role:               entity.PapelPrincipal,
			CommissionPct:      totais.CommissionPct,
			CommissionBase:     entity.BaseTotalProdutos,
		}}
	}
	return montar(documentCode, linhas, totais), nil
}

func (uc *UseCase) Substituir(ctx context.Context, doc entity.Documento, documentCode int64, dto *request.SalvarRateioComissaoDTO) (*response.RateioComissaoResponse, error) {
	if documentCode <= 0 {
		return nil, errorsuc.NewValidationError("informe o documento")
	}
	if dto == nil {
		return nil, errorsuc.NewValidationError("informe o rateio de comissão")
	}
	totais, err := uc.Repo.Totais(ctx, doc, documentCode)
	if err != nil {
		return nil, err
	}

	linhas := make([]*entity.Rateio, 0, len(dto.Representantes))
	codigos := make([]int64, 0, len(dto.Representantes))
	for _, in := range dto.Representantes {
		pct, err := percentual(in.CommissionPct)
		if err != nil {
			return nil, err
		}
		l := &entity.Rateio{
			DocumentCode:       documentCode,
			RepresentativeCode: in.RepresentativeCode,
			Role:               entity.Papel(in.Role),
			CommissionPct:      pct,
			CommissionBase:     entity.Base(in.CommissionBase),
			Notes:              in.Notes,
		}
		l.Normalizar()
		linhas = append(linhas, l)
		codigos = append(codigos, l.RepresentativeCode)
	}
	if err := entity.ValidarRateio(linhas); err != nil {
		return nil, errorsuc.NewValidationError(err.Error())
	}
	// Representante inativo ou bloqueado recebendo comissão é o erro que só
	// aparece no fechamento do mês — barra aqui.
	ativos, err := uc.Repo.RepresentantesAtivos(ctx, codigos)
	if err != nil {
		return nil, err
	}
	for _, l := range linhas {
		if _, ok := ativos[l.RepresentativeCode]; !ok {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("o representante %d não existe, está inativo ou está bloqueado", l.RepresentativeCode))
		}
	}

	gravadas, err := uc.Repo.Substituir(ctx, doc, documentCode, linhas)
	if err != nil {
		return nil, err
	}
	return montar(documentCode, gravadas, totais), nil
}

func percentual(v *float64) (decimal.Decimal, error) {
	if v == nil {
		return decimal.Zero, errorsuc.NewValidationError("informe o percentual de comissão de cada representante")
	}
	return decimal.NewFromFloat(*v), nil
}

func montar(documentCode int64, linhas []*entity.Rateio, totais domrepo.Totais) *response.RateioComissaoResponse {
	out := &response.RateioComissaoResponse{
		DocumentCode:   documentCode,
		TotalProdutos:  totais.TotalProdutos,
		TotalLiquido:   totais.TotalLiquido,
		Representantes: make([]response.RateioComissaoLinhaResponse, 0, len(linhas)),
	}
	somaPct := decimal.Zero
	somaValor := decimal.Zero
	for _, l := range linhas {
		valor := l.Valor(totais.TotalProdutos, totais.TotalLiquido)
		somaPct = somaPct.Add(l.CommissionPct)
		somaValor = somaValor.Add(valor)
		out.Representantes = append(out.Representantes, response.RateioComissaoLinhaResponse{
			ID:                  l.ID,
			RepresentativeCode:  l.RepresentativeCode,
			RepresentativeName:  l.RepresentativeName,
			Role:                string(l.Role),
			RoleLabel:           l.Role.Rotulo(),
			CommissionPct:       l.CommissionPct,
			CommissionBase:      string(l.CommissionBase),
			CommissionBaseLabel: l.CommissionBase.Rotulo(),
			CommissionValue:     valor,
			Notes:               l.Notes,
		})
	}
	out.TotalPct = somaPct
	out.TotalValor = somaValor
	return out
}
