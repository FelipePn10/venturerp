package delivery_promise_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	"github.com/FelipePn10/panossoerp/internal/domain/delivery_promise/service"
)

// LeitorDaPromessa lê a situação do item e a estrutura (repositório PG).
type LeitorDaPromessa interface {
	SituacaoParaPromessa(ctx context.Context, item int64) (service.SituacaoDoItem, error)
	ComponentesParaPromessa(ctx context.Context, item int64) ([]service.Componente, error)
}

// LinhaDoCenario é um item e a quantidade a prometer.
type LinhaDoCenario struct {
	ItemCode request.TextCode `json:"item_code"`
	Quantity decimal.Decimal  `json:"quantity"`
}

// CenarioDTO: as linhas simuladas ("e se entrar um pedido de 50?"). Nada é gravado.
type CenarioDTO struct {
	Lines []LinhaDoCenario `json:"lines"`
}

// PromessaUseCase calcula a data que dá para prometer.
type PromessaUseCase struct {
	Repo  LeitorDaPromessa
	Items any
	Agora func() time.Time
	// Prazos dá o prazo de fabricação do MRP (roteiro do lote com os dias no
	// terceiro, ou o do item). Sem ele vale o do cadastro + etapas de terceiro.
	Prazos interface {
		PrazoDeFabricacao(ctx context.Context, item int64, qtd decimal.Decimal) (int, error)
	}
}

const profundidadeMaxima = 6

// Simular promete cada linha na ordem; a linha seguinte enxerga o que a
// anterior já tomou do estoque e das entradas.
func (uc *PromessaUseCase) Simular(ctx context.Context, dto CenarioDTO) ([]service.Promessa, error) {
	if len(dto.Lines) == 0 {
		return nil, errorsuc.NewValidationError("informe ao menos um item e a quantidade")
	}
	agora := time.Now
	if uc.Agora != nil {
		agora = uc.Agora
	}
	n := agora()
	hoje := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, n.Location())
	consumido := map[int64]decimal.Decimal{}
	out := make([]service.Promessa, 0, len(dto.Lines))
	for _, l := range dto.Lines {
		if !l.Quantity.IsPositive() {
			return nil, errorsuc.NewValidationError("a quantidade precisa ser maior que zero")
		}
		item, err := itemresolution.Resolve(ctx, uc.Items, l.ItemCode)
		if err != nil {
			return nil, err
		}
		p, err := uc.prometer(ctx, hoje, int64(item.Code), l.Quantity, consumido, 0)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (uc *PromessaUseCase) prometer(ctx context.Context, hoje time.Time, item int64, qtd decimal.Decimal, consumido map[int64]decimal.Decimal, nivel int) (service.Promessa, error) {
	s, err := uc.Repo.SituacaoParaPromessa(ctx, item)
	if err != nil {
		return service.Promessa{}, err
	}
	p := service.Promessa{ItemCode: item, Quantidade: qtd}
	if data, origem, exp, ok := service.Atender(hoje, qtd, consumido[item], s); ok {
		consumido[item] = consumido[item].Add(qtd)
		p.Data, p.Origem, p.Explicacao = data, origem, exp
		return p, nil
	}
	livre := decimal.Max(s.Livre.Sub(consumido[item]), decimal.Zero)
	falta := qtd.Sub(livre)
	consumido[item] = consumido[item].Add(livre)
	componentes := time.Time{}
	if s.Fabricado && uc.Prazos != nil {
		dias, err := uc.Prazos.PrazoDeFabricacao(ctx, item, falta)
		if err != nil {
			return p, err
		}
		// O prazo do MRP já traz os dias no terceiro: não soma de novo.
		s.PrazoFabricacaoDias, s.DiasNoTerceiro = dias, 0
	}
	if s.Fabricado {
		if nivel >= profundidadeMaxima {
			return p, errorsuc.NewValidationError(fmt.Sprintf("a estrutura do item %d passa de %d níveis: confira se não é cíclica", item, profundidadeMaxima))
		}
		filhos, err := uc.Repo.ComponentesParaPromessa(ctx, item)
		if err != nil {
			return p, err
		}
		for _, f := range filhos {
			cp, err := uc.prometer(ctx, hoje, f.ItemCode, falta.Mul(f.QtdPor), consumido, nivel+1)
			if err != nil {
				return p, err
			}
			if cp.Origem != service.OrigemEstoque {
				p.Componentes = append(p.Componentes, cp)
			}
			if cp.Data.After(componentes) {
				componentes = cp.Data
			}
		}
	}
	p.Data, p.Origem, p.Explicacao = service.NovaReposicao(hoje, s, componentes)
	if livre.IsPositive() {
		p.Explicacao = append([]string{fmt.Sprintf("%s sai do estoque e %s precisa ser reposto", livre.String(), falta.String())}, p.Explicacao...)
	}
	return p, nil
}
