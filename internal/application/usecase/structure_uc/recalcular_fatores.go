package structure_uc

import (
	"context"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
)

// RecalcularFatores mantém a quantidade da linha de estrutura na unidade de
// estoque ATUAL do componente.
//
// A linha grava a quantidade convertida e o fator (migração 354). Gravados
// uma vez só, ficavam velhos quando a unidade de estoque ou a conversão do
// componente mudavam: o tubo da Tecnofer passou a ser estocado em metro e a
// linha "104 MM, fator 1" seguiu dizendo 104 — o MRP comprou 4.160 m em vez
// de 4,16 m; noutros dois tubos o fator de barra (1/6000) ficou, e o MRP
// comprou 6 vezes menos.
//
// A decisão antiga era congelar o fator para não mudar o consumo de ordens já
// abertas — mas a OF copia o material na criação (production_order_materials),
// então recalcular a estrutura não toca ordem nenhuma. Roda quando a unidade
// de estoque do item ou uma conversão muda, e antes de cada cálculo do MRP.
type RecalcularFatores struct {
	Repo repository.FatoresDaEstruturaRepository
	Conv ports.UOMConverter
}

// LinhaSemConversao: a unidade da linha não converte para a de estoque atual
// — a quantidade gravada não pode ser confiada e a linha precisa de revisão.
type LinhaSemConversao struct {
	ParentCode  int64
	ParentLabel string
	ChildCode   int64
	ChildLabel  string
	Unidade     string
	Estoque     string
}

type ResultadoDoRecalculo struct {
	Atualizadas  int
	SemConversao []LinhaSemConversao
}

// Mensagens dá, por linha sem conversão, o texto para o aviso.
func (r ResultadoDoRecalculo) Mensagens() []string {
	out := make([]string, 0, len(r.SemConversao))
	for _, s := range r.SemConversao {
		pai := s.ParentLabel
		if pai == "" {
			pai = fmt.Sprintf("%d", s.ParentCode)
		}
		out = append(out, fmt.Sprintf("estrutura do item %s: o componente %s é estocado em %s e a linha está em %s, sem conversão entre as duas — cadastre a conversão ou corrija a linha",
			pai, s.ChildLabel, s.Estoque, s.Unidade))
	}
	return out
}

// Executar recalcula as linhas do filtro; com gravar=false só confere (a
// liberação do item usa assim).
func (uc *RecalcularFatores) Executar(ctx context.Context, filtro entity.FiltroDeFatores, gravar bool, motivo string) (ResultadoDoRecalculo, []entity.LinhaDeFator, error) {
	var res ResultadoDoRecalculo
	linhas, err := uc.Repo.LinhasParaConferir(ctx, filtro)
	if err != nil {
		return res, nil, err
	}
	var defasadas []entity.LinhaDeFator
	conv := novoConversorDaRodada(uc.Conv)
	for _, l := range linhas {
		qtd, fator, err := resolverUnidadeDeEstoque(ctx, conv, l.ChildCode, l.ChildLabel, l.Quantidade, l.Unidade, l.UnidadeEstoque)
		if err != nil {
			var v *errorsuc.ValidationError
			if errors.As(err, &v) {
				res.SemConversao = append(res.SemConversao, LinhaSemConversao{ParentCode: l.ParentCode, ParentLabel: l.ParentLabel, ChildCode: l.ChildCode,
					ChildLabel: l.ChildLabel, Unidade: l.Unidade, Estoque: l.UnidadeEstoque})
				continue
			}
			return res, nil, err
		}
		if !l.Defasada(qtd, fator) {
			continue
		}
		defasadas = append(defasadas, l)
		if gravar {
			gravou, err := uc.Repo.GravarFator(ctx, l, qtd, fator, motivo)
			if err != nil {
				return res, defasadas, err
			}
			if gravou {
				res.Atualizadas++
			}
		}
	}
	return res, defasadas, nil
}

// DoComponente: depois de mudar a unidade de estoque ou a conversão do item.
func (uc *RecalcularFatores) DoComponente(ctx context.Context, itemCode int64, motivo string) (ResultadoDoRecalculo, error) {
	res, _, err := uc.Executar(ctx, entity.FiltroDeFatores{Filhos: []int64{itemCode}}, true, motivo)
	return res, err
}

// DaEmpresa: todas as linhas ativas (conversão padrão do catálogo, MRP).
func (uc *RecalcularFatores) DaEmpresa(ctx context.Context, motivo string) (ResultadoDoRecalculo, error) {
	res, _, err := uc.Executar(ctx, entity.FiltroDeFatores{}, true, motivo)
	return res, err
}

// RecalcularDoPai regrava as linhas da estrutura de UM item: a OF chama antes
// de copiar a estrutura, para a OF manual criada antes do primeiro MRP (ou
// depois de mudar a conversão por um caminho que não avisa) não copiar a
// quantidade do fator velho.
func (uc *RecalcularFatores) RecalcularDoPai(ctx context.Context, pai int64, motivo string) (int, []string, error) {
	res, _, err := uc.Executar(ctx, entity.FiltroDeFatores{Pai: &pai}, true, motivo)
	return res.Atualizadas, res.Mensagens(), err
}

// RecalcularDoComponente implementa ports.RecalculoDeEstrutura.
func (uc *RecalcularFatores) RecalcularDoComponente(ctx context.Context, itemCode int64, motivo string) (int, []string, error) {
	res, err := uc.DoComponente(ctx, itemCode, motivo)
	return res.Atualizadas, res.Mensagens(), err
}

// RecalcularDaEmpresa implementa ports.RecalculoDeEstrutura.
func (uc *RecalcularFatores) RecalcularDaEmpresa(ctx context.Context, motivo string) (int, []string, error) {
	res, err := uc.DaEmpresa(ctx, motivo)
	return res.Atualizadas, res.Mensagens(), err
}

var _ ports.RecalculoDeEstrutura = (*RecalcularFatores)(nil)

// ConferirDoPai só confere (não grava) as linhas da estrutura do pai: a
// liberação do item mostra o que está na unidade velha e o que não converte.
func (uc *RecalcularFatores) ConferirDoPai(ctx context.Context, pai int64) ([]string, []string, error) {
	res, defasadas, err := uc.Executar(ctx, entity.FiltroDeFatores{Pai: &pai}, false, "conferência da liberação")
	if err != nil {
		return nil, nil, err
	}
	out := make([]string, 0, len(defasadas))
	for _, l := range defasadas {
		out = append(out, fmt.Sprintf("%s %s %s gravado como %s %s (fator %s)",
			l.ChildLabel, l.Quantidade.String(), l.Unidade, l.QtdGravada.String(), l.UnidadeEstoque, l.FatorGravado.String()))
	}
	return out, res.Mensagens(), nil
}
