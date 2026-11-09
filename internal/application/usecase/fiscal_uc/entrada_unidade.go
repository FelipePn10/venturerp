package fiscal_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entrada"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	convservice "github.com/FelipePn10/panossoerp/internal/domain/item_conversion/service"
	uomrepo "github.com/FelipePn10/panossoerp/internal/domain/unit_of_measure/repository"
)

// Origens do fator gravado na linha da nota (fiscal_entry_items.origem_conversao).
const (
	OrigemMesmaUnidade      = "MESMA_UNIDADE"
	OrigemSinonimo          = "SINONIMO"
	OrigemVinculoFornecedor = "VINCULO_FORNECEDOR"
	OrigemPedidoCompra      = "PEDIDO_COMPRA"
	OrigemCadastroItem      = "CADASTRO_ITEM"
	OrigemLote              = "LOTE"
	OrigemPadrao            = "PADRAO"
	OrigemXMLTributavel     = "XML_TRIBUTAVEL"
	OrigemManual            = "MANUAL"
)

// Desvio de custo a partir do qual a conferência desconfia do fator: o custo
// por unidade de estoque ficou 5× maior ou menor que o da última entrada.
var (
	desvioCustoAlto  = decimal.NewFromInt(5)
	desvioCustoBaixo = decimal.RequireFromString("0.2")
)

// fatorFixo: o fator da linha foi decidido por alguém (pessoa, vínculo com o
// fornecedor, pedido) e não é recalculado. Linha antiga, sem origem, com
// fator diferente de 1 também é respeitada — não se sabe de onde veio.
func fatorFixo(it *entity.FiscalEntryItem) bool {
	if it.OrigemConversao == nil {
		return !semFator(it)
	}
	switch *it.OrigemConversao {
	case OrigemManual, OrigemVinculoFornecedor, OrigemPedidoCompra:
		return true
	}
	return false
}

// definirFator grava fator, quantidade em estoque, origem e explicação.
func definirFator(it *entity.FiscalEntryItem, f decimal.Decimal, q decimal.Decimal, origem string, caminho string) {
	it.FatorConversao, it.QuantidadeEstoque = &f, &q
	it.OrigemConversao = strPtr(origem)
	if caminho == "" {
		it.CaminhoConversao = nil
	} else {
		it.CaminhoConversao = strPtr(caminho)
	}
}

// limparFator volta a linha para "sem conversão" (fator 1, sem origem).
func limparFator(it *entity.FiscalEntryItem) {
	q := it.Quantity.Round(6)
	f := umDec
	it.FatorConversao, it.QuantidadeEstoque = &f, &q
	it.OrigemConversao, it.CaminhoConversao = nil, nil
}

func textoUnidade(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

// normalizacoes traduz para o catálogo as unidades comerciais e tributáveis
// da nota (UND, PÇ, "Kg."), considerando os sinônimos do fornecedor.
func (s *EntradaServico) normalizacoes(ctx context.Context, e *entity.FiscalEntry) (map[string]uomrepo.Resolucao, error) {
	vistos := map[string]bool{}
	textos := []string{}
	for _, it := range e.Itens {
		for _, t := range []string{textoUnidade(it.UOM), textoUnidade(it.UnidadeTributavel)} {
			if t != "" && !vistos[t] {
				vistos[t] = true
				textos = append(textos, t)
			}
		}
	}
	return s.Docs.NormalizarUnidades(ctx, e.SupplierCode, textos)
}

// canonica: a unidade do catálogo para o texto; sem tradução, o próprio texto
// em maiúsculas (a comparação continua valendo para "kg" × "KG").
func canonica(norm map[string]uomrepo.Resolucao, texto string) (string, string) {
	texto = strings.TrimSpace(texto)
	if r, ok := norm[texto]; ok && r.Unidade != "" {
		return r.Unidade, r.Via
	}
	return strings.ToUpper(texto), ""
}

func origemDoCaminho(c *convservice.Caminho) string {
	switch c.OrigemPrincipal() {
	case convservice.OrigemLote:
		return OrigemLote
	case convservice.OrigemPadrao:
		return OrigemPadrao
	}
	return OrigemCadastroItem
}

// resolverUnidades leva para a unidade de estoque as linhas cujo fator não
// foi decidido por alguém. Precedência (o que vem antes já foi aplicado em
// Preparar e é fixo): vínculo com o fornecedor → pedido de compra → mesma
// unidade/sinônimo → conversão do item (lote, máscara, item, padrão; com
// encadeamento) → unidade tributável do XML (qTrib/qCom).
//
// "Automático com evidência forte": só aplica fator que vem de cadastro ou do
// próprio XML. Sem evidência a linha fica sem fator e a conferência impede a
// aprovação, com sugestões (SugestoesDeUnidade).
func (s *EntradaServico) resolverUnidades(ctx context.Context, e *entity.FiscalEntry, cadastro map[int64]repository.ItemCandidate) (bool, error) {
	if s.Docs == nil {
		return false, nil
	}
	norm, err := s.normalizacoes(ctx, e)
	if err != nil {
		return false, err
	}
	mudou := false
	for _, it := range e.Itens {
		if it.ItemCode == nil {
			continue
		}
		antes := assinaturaFator(it)
		canon, via := canonica(norm, textoUnidade(it.UOM))
		if canon != "" {
			it.UnidadeNormalizada = strPtr(canon)
		}
		if fatorFixo(it) {
			if assinaturaFator(it) != antes {
				mudou = true
			}
			continue
		}
		c, ok := cadastro[*it.ItemCode]
		if !ok || strings.TrimSpace(c.UOM) == "" || canon == "" {
			continue
		}
		estoque := strings.ToUpper(strings.TrimSpace(c.UOM))
		if err := s.resolverLinha(ctx, e, it, norm, canon, via, estoque); err != nil {
			return false, err
		}
		if assinaturaFator(it) != antes {
			mudou = true
		}
	}
	return mudou, nil
}

func (s *EntradaServico) resolverLinha(ctx context.Context, e *entity.FiscalEntry, it *entity.FiscalEntryItem, norm map[string]uomrepo.Resolucao, canon, via, estoque string) error {
	// 1. Mesma unidade (direto ou por sinônimo).
	if canon == estoque {
		origem, caminho := OrigemMesmaUnidade, ""
		if via != "" && via != "CODIGO" {
			origem = OrigemSinonimo
			caminho = fmt.Sprintf("%q da nota é %s no catálogo", textoUnidade(it.UOM), canon)
		}
		definirFator(it, umDec, it.Quantity.Round(6), origem, caminho)
		return nil
	}
	// 2. Conversão cadastrada (com encadeamento e padrão da empresa).
	cam, err := s.Docs.ConversaoDeUnidade(ctx, *it.ItemCode, it.Mask, e.DataEntrada, canon, estoque)
	if err != nil {
		return err
	}
	if cam != nil {
		definirFator(it, cam.Fator(), cam.Converter(it.Quantity).Round(6), origemDoCaminho(cam), cam.Explicacao())
		return nil
	}
	// 3. Unidade tributável do XML: o emitente declarou a mesma mercadoria em
	// outra unidade (qTrib uTrib para qCom uCom). Só vale se a tributável é
	// outra unidade e chega à de estoque (direto ou por conversão cadastrada).
	if f, caminho, ok, err := s.fatorTributavel(ctx, e, it, norm, canon, estoque); err != nil {
		return err
	} else if ok {
		definirFator(it, f, it.Quantity.Mul(f).Round(6), OrigemXMLTributavel, caminho)
		return nil
	}
	limparFator(it)
	return nil
}

func (s *EntradaServico) fatorTributavel(ctx context.Context, e *entity.FiscalEntry, it *entity.FiscalEntryItem, norm map[string]uomrepo.Resolucao, canon, estoque string) (decimal.Decimal, string, bool, error) {
	if it.QuantidadeTributavel == nil || !it.QuantidadeTributavel.IsPositive() || !it.Quantity.IsPositive() {
		return decimal.Zero, "", false, nil
	}
	trib, _ := canonica(norm, textoUnidade(it.UnidadeTributavel))
	if trib == "" || trib == canon {
		return decimal.Zero, "", false, nil
	}
	porCom := it.QuantidadeTributavel.DivRound(it.Quantity, 10) // 1 uCom = porCom uTrib
	base := fmt.Sprintf("XML: %s %s = %s %s (unidade tributável)",
		it.Quantity.String(), canon, it.QuantidadeTributavel.String(), trib)
	if trib == estoque {
		return porCom, base, true, nil
	}
	cam, err := s.Docs.ConversaoDeUnidade(ctx, *it.ItemCode, it.Mask, e.DataEntrada, trib, estoque)
	if err != nil || cam == nil {
		return decimal.Zero, "", false, err
	}
	return porCom.Mul(cam.Fator()).Round(10), base + " → " + cam.Explicacao(), true, nil
}

func assinaturaFator(it *entity.FiscalEntryItem) string {
	f, q := "", ""
	if it.FatorConversao != nil {
		f = it.FatorConversao.String()
	}
	if it.QuantidadeEstoque != nil {
		q = it.QuantidadeEstoque.String()
	}
	return strings.Join([]string{f, q, textoUnidade(it.OrigemConversao), textoUnidade(it.CaminhoConversao), textoUnidade(it.UnidadeNormalizada)}, "|")
}

// AplicarConversoes reaplica a resolução de unidades numa nota já gravada
// (nota importada antes da correção, conversão cadastrada depois da
// importação, sinônimo novo). Devolve se alguma linha mudou.
func (s *EntradaServico) AplicarConversoes(ctx context.Context, e *entity.FiscalEntry) (bool, error) {
	cadastro, err := s.cadastroDosItens(ctx, e)
	if err != nil {
		return false, err
	}
	return s.resolverUnidades(ctx, e, cadastro)
}

func (s *EntradaServico) cadastroDosItens(ctx context.Context, e *entity.FiscalEntry) (map[int64]repository.ItemCandidate, error) {
	codigos := []int64{}
	for _, it := range e.Itens {
		if it.ItemCode != nil {
			codigos = append(codigos, *it.ItemCode)
		}
	}
	return s.Docs.ItemsByCode(ctx, codigos)
}

// semFator: a linha não tem conversão própria (nula ou 1, que é o padrão
// gravado quando nada converteu).
func semFator(it *entity.FiscalEntryItem) bool {
	return it.FatorConversao == nil || it.FatorConversao.Equal(umDec)
}

// pendenciasDeUnidade:
//   - IMPEDE: linha que entra no estoque numa unidade diferente da de estoque
//     sem conversão nenhuma — entraria com o número errado;
//   - ATENÇÃO: linha convertida cujo custo por unidade de estoque ficou 5×
//     maior ou menor que o da última entrada do item — fator provavelmente
//     invertido ou na unidade errada.
func (s *EntradaServico) pendenciasDeUnidade(ctx context.Context, e *entity.FiscalEntry) ([]entrada.Pendencia, error) {
	if s.Docs == nil {
		return nil, nil
	}
	cadastro, err := s.cadastroDosItens(ctx, e)
	if err != nil {
		return nil, err
	}
	norm, err := s.normalizacoes(ctx, e)
	if err != nil {
		return nil, err
	}
	var out []entrada.Pendencia
	for _, it := range e.Itens {
		if it.ItemCode == nil || !it.MovimentaEstoque {
			continue
		}
		c, ok := cadastro[*it.ItemCode]
		if !ok || strings.TrimSpace(c.UOM) == "" || textoUnidade(it.UOM) == "" {
			continue
		}
		estoque := strings.ToUpper(strings.TrimSpace(c.UOM))
		canon, _ := canonica(norm, textoUnidade(it.UOM))
		codigo := c.BusinessCode
		if codigo == "" {
			codigo = fmt.Sprint(c.Code)
		}
		if canon == estoque {
			continue
		}
		resolvida := it.OrigemConversao != nil || !semFator(it)
		if !resolvida {
			// Conversão com fator 1 (PC → UN) cadastrada depois da última
			// preparação também resolve.
			cam, err := s.Docs.ConversaoDeUnidade(ctx, *it.ItemCode, it.Mask, e.DataEntrada, canon, estoque)
			if err != nil {
				return nil, err
			}
			resolvida = cam != nil
		}
		if !resolvida {
			daNota := strings.ToUpper(textoUnidade(it.UOM))
			unidade := daNota
			if canon != daNota {
				unidade = fmt.Sprintf("%s (%s no catálogo)", daNota, canon)
			}
			out = append(out, entrada.Pendencia{
				Nivel: entrada.NivelImpede,
				Campo: fmt.Sprintf("itens[%d].fator_conversao", it.Sequence),
				Mensagem: fmt.Sprintf("item %d (%s): a nota vem em %s e o item é estocado em %s, sem conversão — cadastre a conversão %s→%s do item (VSUP0110), escolha uma das sugestões ou informe o fator na conciliação",
					it.Sequence, codigo, unidade, estoque, canon, estoque),
			})
			continue
		}
		if p, err := s.pendenciaDeCusto(ctx, e, it, codigo, estoque); err != nil {
			return nil, err
		} else if p != nil {
			out = append(out, *p)
		}
	}
	return out, nil
}

// pendenciaDeCusto compara o custo por unidade de estoque desta linha com o
// da última entrada aprovada do item.
func (s *EntradaServico) pendenciaDeCusto(ctx context.Context, e *entity.FiscalEntry, it *entity.FiscalEntryItem, codigo, estoque string) (*entrada.Pendencia, error) {
	if it.QuantidadeEstoque == nil || !it.QuantidadeEstoque.IsPositive() || !it.TotalPrice.IsPositive() {
		return nil, nil
	}
	ref, err := s.Docs.CustoDeReferencia(ctx, *it.ItemCode, e.ID)
	if err != nil || ref == nil || !ref.IsPositive() {
		return nil, err
	}
	custo := it.TotalPrice.DivRound(*it.QuantidadeEstoque, 6)
	razao := custo.DivRound(*ref, 6)
	if razao.LessThan(desvioCustoAlto) && razao.GreaterThan(desvioCustoBaixo) {
		return nil, nil
	}
	fator := "1"
	if it.FatorConversao != nil {
		fator = it.FatorConversao.String()
	}
	return &entrada.Pendencia{
		Nivel: entrada.NivelAtencao,
		Campo: fmt.Sprintf("itens[%d].fator_conversao", it.Sequence),
		Mensagem: fmt.Sprintf("item %d (%s): com o fator %s o custo fica %s por %s, e a última entrada foi %s por %s (%s×) — confira se o fator não está invertido ou na unidade errada",
			it.Sequence, codigo, fator, entrada.Moeda(custo), estoque, entrada.Moeda(*ref), estoque, razao.StringFixed(2)),
	}, nil
}
