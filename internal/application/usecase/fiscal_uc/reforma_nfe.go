package fiscal_uc

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/reforma"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

// TributacaoReforma é o que a NF-e precisa para o grupo IBS/CBS: as alíquotas
// do ano e a classificação tributária de cada linha (pelo ID da linha).
type TributacaoReforma struct {
	// Aplica é falso no Simples Nacional, que não leva o grupo, e quando a
	// empresa escolheu emitir só com o modelo atual.
	Aplica bool
	// SomenteAtual: a empresa desligou o grupo IBS/CBS (configuração).
	SomenteAtual bool
	Aliquotas    reforma.Aliquotas
	PorItem      map[int64]ClassificacaoDaLinha
}

// ClassificacaoDaLinha é a classificação aplicada e de onde ela veio.
type ClassificacaoDaLinha struct {
	Classificacao reforma.Classificacao
	Origem        string
}

// resolverReformaDaNota decide alíquotas e classificação de cada linha antes
// da transmissão, e recusa o que a SEFAZ rejeitaria: ano sem alíquota,
// cClassTrib fora da tabela oficial, fora de vigência ou não admitido na NF-e.
//
// Sem repositório (dublês de teste), vale a lei e a tributação integral.
func resolverReformaDaNota(ctx context.Context, repo repository.ReformaTributariaRepository, exit *entity.FiscalExit,
	items []*entity.FiscalExitItem, cfg *entity.FiscalConfig) (TributacaoReforma, error) {
	if regimeSimples(cfg) {
		return TributacaoReforma{}, nil
	}
	if repo != nil {
		par, err := repo.Parametros(ctx)
		if err != nil {
			return TributacaoReforma{}, err
		}
		if par.Modo == reforma.ModoSomenteAtual {
			return TributacaoReforma{SomenteAtual: true}, nil
		}
	}
	ano := exit.DataEmissao.Year()
	var cadastro *reforma.Aliquotas
	if repo != nil {
		var err error
		if cadastro, err = repo.AliquotasDoAno(ctx, ano); err != nil {
			return TributacaoReforma{}, err
		}
	}
	aliq, err := reforma.ResolverAliquotas(ano, cadastro)
	if err != nil {
		return TributacaoReforma{}, err
	}
	rt := TributacaoReforma{Aplica: true, Aliquotas: aliq, PorItem: map[int64]ClassificacaoDaLinha{}}
	if repo == nil {
		for _, it := range items {
			rt.PorItem[it.ID] = ClassificacaoDaLinha{Classificacao: reforma.ClassificacaoPadrao(), Origem: reforma.OrigemPadrao}
		}
		return rt, nil
	}
	candidatos, err := repo.CandidatosDaNota(ctx, exit.ID)
	if err != nil {
		return TributacaoReforma{}, err
	}
	escolhas := make(map[int64][2]string, len(items))
	codigos := []string{}
	vistos := map[string]bool{}
	for _, it := range items {
		c := candidatos[it.ID]
		codigo, origem := reforma.EscolherClassTrib(c.DoItem, c.DoCFOP, c.DaClassificacao)
		escolhas[it.ID] = [2]string{codigo, origem}
		if !vistos[codigo] {
			vistos[codigo] = true
			codigos = append(codigos, codigo)
		}
	}
	tabela, err := repo.BuscarClassificacoes(ctx, codigos)
	if err != nil {
		return TributacaoReforma{}, err
	}
	for _, it := range items {
		codigo, origem := escolhas[it.ID][0], escolhas[it.ID][1]
		oficial, ok := tabela[codigo]
		if !ok {
			return TributacaoReforma{}, errorsuc.NewValidationError(fmt.Sprintf(
				"linha %d: o cClassTrib %q (%s) não existe na tabela oficial de classificação tributária do IBS/CBS",
				it.Sequence, codigo, descreverOrigem(origem)))
		}
		if err := oficial.ValidarParaNFe(exit.DataEmissao); err != nil {
			return TributacaoReforma{}, errorsuc.NewValidationError(fmt.Sprintf("linha %d (%s): %s", it.Sequence, descreverOrigem(origem), err.Error()))
		}
		rt.PorItem[it.ID] = ClassificacaoDaLinha{Classificacao: oficial.Classificacao, Origem: origem}
	}
	return rt, nil
}

func descreverOrigem(origem string) string {
	switch origem {
	case reforma.OrigemItem:
		return "informado na linha da nota"
	case reforma.OrigemCFOP:
		return "padrão do CFOP"
	case reforma.OrigemClassificacao:
		return "da classificação fiscal do item"
	}
	return "tributação integral padrão"
}

// aplicarReforma calcula o grupo IBS/CBS de cada linha do payload (mesma ordem
// dos itens da nota) e devolve o retrato do que vai ser transmitido.
func aplicarReforma(rt TributacaoReforma, items []*entity.FiscalExitItem, itens []focusnfe.NFEItem) []reforma.RetratoItem {
	if !rt.Aplica {
		return nil
	}
	retrato := make([]reforma.RetratoItem, 0, len(itens))
	for i := range itens {
		if i >= len(items) {
			break
		}
		it := &itens[i]
		linha, ok := rt.PorItem[items[i].ID]
		if !ok {
			linha = ClassificacaoDaLinha{Classificacao: reforma.ClassificacaoPadrao(), Origem: reforma.OrigemPadrao}
		}
		op := decimal.NewFromFloat(it.ValorBruto).Add(decimal.NewFromFloat(it.ValorFrete)).Add(decimal.NewFromFloat(it.ValorSeguro)).
			Sub(decimal.NewFromFloat(it.ValorDesconto))
		g := reforma.Calcular(rt.Aliquotas, linha.Classificacao, op,
			decimal.NewFromFloat(it.ValorICMS), decimal.NewFromFloat(it.ValorPIS), decimal.NewFromFloat(it.ValorCOFINS))
		it.IBSCBS = grupoFocus(g)
		retrato = append(retrato, reforma.RetratoItem{ItemID: items[i].ID, Origem: linha.Origem, Grupo: g})
	}
	return retrato
}

func grupoFocus(g reforma.Grupo) *focusnfe.NFEItemIBSCBS {
	out := &focusnfe.NFEItemIBSCBS{CST: g.CST, ClassTrib: g.ClassTrib, SemValores: !g.ComValores}
	if !g.ComValores {
		return out
	}
	out.Base = g.Base.InexactFloat64()
	out.AliqIBSUF, out.ValorIBSUF = g.IBSUF.Aliquota.InexactFloat64(), g.IBSUF.Valor.InexactFloat64()
	out.AliqIBSMun, out.ValorIBSMun = g.IBSMun.Aliquota.InexactFloat64(), g.IBSMun.Valor.InexactFloat64()
	out.AliqCBS, out.ValorCBS = g.CBS.Aliquota.InexactFloat64(), g.CBS.Valor.InexactFloat64()
	out.ValorIBS = g.IBS.InexactFloat64()
	if g.ComReducao {
		out.ComReducao = true
		out.RedIBSUF, out.EfetIBSUF = g.IBSUF.PercReducao.InexactFloat64(), g.IBSUF.AliquotaEfetiva.InexactFloat64()
		out.RedIBSMun, out.EfetIBSMun = g.IBSMun.PercReducao.InexactFloat64(), g.IBSMun.AliquotaEfetiva.InexactFloat64()
		out.RedCBS, out.EfetCBS = g.CBS.PercReducao.InexactFloat64(), g.CBS.AliquotaEfetiva.InexactFloat64()
	}
	return out
}

// totaisDoRetrato soma IBS e CBS transmitidos.
func totaisDoRetrato(retrato []reforma.RetratoItem) (ibs, cbs decimal.Decimal) {
	for _, r := range retrato {
		ibs = ibs.Add(r.Grupo.IBS)
		cbs = cbs.Add(r.Grupo.CBS.Valor)
	}
	return ibs, cbs
}

// avisoPISCOFINSExtintos: a LC 214/2025 revoga o PIS e a COFINS a partir de
// 1º/01/2027 (art. 542). Nota de 2027 em diante com valor desses tributos é
// sinal de classificação fiscal desatualizada — avisa, não impede, porque o
// tratamento desses grupos no leiaute de 2027 depende de nota técnica.
func avisoPISCOFINSExtintos(exit *entity.FiscalExit, items []*entity.FiscalExitItem) []response.PreviaPendenciaNFe {
	if exit.DataEmissao.Year() < 2027 {
		return nil
	}
	for _, it := range items {
		if it.ValorPIS > 0 || it.ValorCOFINS > 0 {
			return []response.PreviaPendenciaNFe{{
				Nivel: NivelAtencao, Campo: "itens.pis_cofins",
				Mensagem:     fmt.Sprintf("a linha %d tem PIS/COFINS, extintos a partir de 1º/01/2027 (LC 214/2025, art. 542)", it.Sequence),
				ComoResolver: "revise as alíquotas de PIS e COFINS da classificação fiscal do item (VFIS0350) conforme a nota técnica vigente",
			}}
		}
	}
	return nil
}
