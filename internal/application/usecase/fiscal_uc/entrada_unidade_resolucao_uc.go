package fiscal_uc

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	uomrepo "github.com/FelipePn10/panossoerp/internal/domain/unit_of_measure/repository"
)

// Como lembrar a resolução de unidade de uma linha da nota.
const (
	MemorizarNenhum     = "NENHUM"
	MemorizarItem       = "ITEM"
	MemorizarFornecedor = "FORNECEDOR"
	MemorizarSinonimo   = "SINONIMO"
)

// CadastroDeConversao é o que a nota precisa da VSUP0110 para memorizar uma
// conversão do item — passa pelas mesmas validações da tela (catálogo,
// consistência com as demais conversões, vigência).
type CadastroDeConversao interface {
	Create(ctx context.Context, dto request.CreateItemConversionDTO) (*response.ItemUnitConversionResponse, error)
}

// CadastroDeSinonimo: o catálogo de unidades, para memorizar um sinônimo.
type CadastroDeSinonimo interface {
	CreateAlias(ctx context.Context, dto request.UnitAliasDTO) (*response.UnitAliasResponse, error)
}

// FiscalEntryUnitsUseCase: sugestões de fator para as linhas da nota que
// chegam em unidade diferente da de estoque e a aplicação da escolha, com a
// opção de memorizar (item, fornecedor ou sinônimo).
type FiscalEntryUnitsUseCase struct {
	Docs        repository.FiscalEntryDocumentRepository
	Fiscal      repository.FiscalRepository
	Tolerancias ports.PurchaseToleranceEvaluator
	Conversoes  CadastroDeConversao
	Sinonimos   CadastroDeSinonimo
	Auth        ports.AuthService
}

func (uc *FiscalEntryUnitsUseCase) servico() *EntradaServico {
	return &EntradaServico{Docs: uc.Docs, Fiscal: uc.Fiscal, Tolerancias: uc.Tolerancias}
}

// Suggestions lista as linhas com unidade diferente da de estoque e, para
// cada uma, os fatores possíveis com a fonte e a confiança.
func (uc *FiscalEntryUnitsUseCase) Suggestions(ctx context.Context, entryID int64) ([]response.UnitSuggestionLineResponse, error) {
	if !uc.Auth.CanCreateFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	doc, err := uc.Docs.GetEntryDocument(ctx, entryID)
	if err != nil {
		return nil, err
	}
	s := uc.servico()
	cadastro, err := s.cadastroDosItens(ctx, doc)
	if err != nil {
		return nil, err
	}
	norm, err := s.normalizacoes(ctx, doc)
	if err != nil {
		return nil, err
	}
	linhasPedido, err := uc.linhasDoPedido(ctx, doc)
	if err != nil {
		return nil, err
	}
	out := []response.UnitSuggestionLineResponse{}
	for _, it := range doc.Itens {
		if it.ItemCode == nil || textoUnidade(it.UOM) == "" {
			continue
		}
		c, ok := cadastro[*it.ItemCode]
		if !ok || strings.TrimSpace(c.UOM) == "" {
			continue
		}
		estoque := strings.ToUpper(strings.TrimSpace(c.UOM))
		canon, via := canonica(norm, textoUnidade(it.UOM))
		if canon == estoque {
			continue
		}
		linha := response.UnitSuggestionLineResponse{
			ItemID: it.ID, Sequence: it.Sequence, ItemCode: c.Code, ItemBusinessCode: c.BusinessCode, ItemName: c.Name,
			UnidadeNota: textoUnidade(it.UOM), UnidadeNormalizada: canon, UnidadeCatalogo: via != "", UnidadeEstoque: estoque,
			Quantidade: it.Quantity, ValorTotal: it.TotalPrice,
			FatorAtual: it.FatorConversao, OrigemAtual: it.OrigemConversao, CaminhoAtual: it.CaminhoConversao,
			Resolvida:  it.OrigemConversao != nil || !semFator(it),
			Candidatos: []response.UnitCandidateResponse{},
		}
		if linha.CustoReferencia, err = uc.Docs.CustoDeReferencia(ctx, c.Code, doc.ID); err != nil {
			return nil, err
		}
		if linha.Candidatos, err = uc.candidatos(ctx, doc, it, norm, canon, estoque, linha.CustoReferencia, linhasPedido); err != nil {
			return nil, err
		}
		if linha.Candidatos == nil {
			// Sem candidato a lista vai vazia, não null: o contrato é uma lista.
			linha.Candidatos = []response.UnitCandidateResponse{}
		}
		linha.Memorizacoes = []string{MemorizarNenhum}
		if via != "" {
			linha.Memorizacoes = append(linha.Memorizacoes, MemorizarItem)
		}
		if doc.SupplierCode != nil && it.SupplierItemCode != nil && strings.TrimSpace(*it.SupplierItemCode) != "" {
			linha.Memorizacoes = append(linha.Memorizacoes, MemorizarFornecedor)
		}
		if via == "" {
			linha.Memorizacoes = append(linha.Memorizacoes, MemorizarSinonimo)
		}
		out = append(out, linha)
	}
	return out, nil
}

func (uc *FiscalEntryUnitsUseCase) linhasDoPedido(ctx context.Context, doc *entity.FiscalEntry) (map[int64]repository.PurchaseOrderLine, error) {
	out := map[int64]repository.PurchaseOrderLine{}
	if doc.SupplierCode == nil {
		return out, nil
	}
	codigos := []int64{}
	for _, it := range doc.Itens {
		if it.PurchaseOrderItemCode != nil {
			codigos = append(codigos, *it.PurchaseOrderItemCode)
		}
	}
	if len(codigos) == 0 {
		return out, nil
	}
	linhas, err := uc.Docs.PurchaseOrderLines(ctx, *doc.SupplierCode, nil, codigos)
	if err != nil {
		return nil, err
	}
	for _, l := range linhas {
		out[l.Code] = l
	}
	return out, nil
}

var confiancaPeso = map[string]int{"ALTA": 0, "MEDIA": 1, "BAIXA": 2}

func (uc *FiscalEntryUnitsUseCase) candidatos(ctx context.Context, doc *entity.FiscalEntry, it *entity.FiscalEntryItem, norm map[string]uomrepo.Resolucao, canon, estoque string, ref *decimal.Decimal, pedido map[int64]repository.PurchaseOrderLine) ([]response.UnitCandidateResponse, error) {
	var out []response.UnitCandidateResponse
	vistos := map[string]bool{}
	add := func(fonte, confianca string, f decimal.Decimal, explicacao string) {
		if !f.IsPositive() {
			return
		}
		f = f.Round(10)
		chave := fonte + "|" + f.String()
		if vistos[chave] {
			return
		}
		vistos[chave] = true
		cand := response.UnitCandidateResponse{Fonte: fonte, Confianca: confianca, Fator: f, QuantidadeEstoque: it.Quantity.Mul(f).Round(6), Explicacao: explicacao}
		if cand.QuantidadeEstoque.IsPositive() && it.TotalPrice.IsPositive() {
			custo := it.TotalPrice.DivRound(cand.QuantidadeEstoque, 6)
			cand.CustoUnitario = &custo
			if ref != nil && ref.IsPositive() {
				d := custo.DivRound(*ref, 4)
				cand.DesvioCusto = &d
				// Custo fora do padrão rebaixa a confiança — é o sinal do
				// fator invertido.
				if d.GreaterThanOrEqual(desvioCustoAlto) || d.LessThanOrEqual(desvioCustoBaixo) {
					cand.Confianca = "BAIXA"
				}
			}
		}
		out = append(out, cand)
	}

	// Cadastro (item, lote, padrão; com encadeamento).
	if cam, err := uc.Docs.ConversaoDeUnidade(ctx, *it.ItemCode, it.Mask, doc.DataEntrada, canon, estoque); err != nil {
		return nil, err
	} else if cam != nil {
		add(origemDoCaminho(cam), "ALTA", cam.Fator(), cam.Explicacao())
	}
	// Unidade tributável do XML.
	s := uc.servico()
	if f, caminho, ok, err := s.fatorTributavel(ctx, doc, it, norm, canon, estoque); err != nil {
		return nil, err
	} else if ok {
		add(OrigemXMLTributavel, "ALTA", f, caminho)
	}
	// Pedido de compra vinculado.
	if it.PurchaseOrderItemCode != nil {
		if l, ok := pedido[*it.PurchaseOrderItemCode]; ok && l.InternalQty.IsPositive() {
			add(OrigemPedidoCompra, "ALTA", l.FatorEstoque(), fmt.Sprintf("pedido de compra %d", l.OrderNumber))
		}
	}
	// Última nota aprovada do mesmo fornecedor, mesmo item e unidade.
	if doc.SupplierCode != nil {
		if f, err := uc.Docs.FatorHistorico(ctx, *doc.SupplierCode, *it.ItemCode, canon); err != nil {
			return nil, err
		} else if f != nil && !f.Equal(umDec) {
			add("HISTORICO_FORNECEDOR", "MEDIA", *f, "fator usado na última nota aprovada deste fornecedor para o item")
		}
	}
	// Peso do cadastro de engenharia: nota em massa, peso por unidade de estoque.
	if peso, unidadePeso, err := uc.Docs.PesoDoItem(ctx, *it.ItemCode); err != nil {
		return nil, err
	} else if peso.IsPositive() {
		f, explic := decimal.Zero, ""
		if unidadePeso == canon {
			f = umDec.DivRound(peso, 10)
			explic = fmt.Sprintf("peso líquido do item: 1 %s = %s %s", estoque, peso.String(), unidadePeso)
		} else if cam, err := uc.Docs.ConversaoDeUnidade(ctx, *it.ItemCode, it.Mask, doc.DataEntrada, canon, unidadePeso); err != nil {
			return nil, err
		} else if cam != nil {
			f = cam.Fator().DivRound(peso, 10)
			explic = fmt.Sprintf("peso líquido do item: 1 %s = %s %s; %s", estoque, peso.String(), unidadePeso, cam.Explicacao())
		}
		add("PESO_DO_ITEM", "MEDIA", f, explic)
	}
	// Custo da última entrada: quantas unidades de estoque o preço unitário
	// da nota compra. É o palpite mais fraco — preço muda.
	if ref != nil && ref.IsPositive() && it.UnitPrice.IsPositive() {
		add("CUSTO_REFERENCIA", "BAIXA", it.UnitPrice.DivRound(*ref, 10),
			fmt.Sprintf("preço da nota (%s por %s) ÷ custo da última entrada (%s por %s)", it.UnitPrice.StringFixed(4), canon, ref.StringFixed(4), estoque))
	}
	sort.SliceStable(out, func(i, j int) bool { return confiancaPeso[out[i].Confianca] < confiancaPeso[out[j].Confianca] })
	return out, nil
}

// Apply aplica a resolução escolhida a uma linha e, se pedido, memoriza.
func (uc *FiscalEntryUnitsUseCase) Apply(ctx context.Context, entryID int64, dto request.ApplyUnitResolutionDTO) (*response.FiscalEntryResponse, error) {
	if !uc.Auth.CanCreateFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	doc, err := uc.Docs.GetEntryDocument(ctx, entryID)
	if err != nil {
		return nil, err
	}
	if doc.Status != entity.EntryStatusPending && doc.Status != entity.EntryStatusConferred {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a nota está %s e não pode mais ser alterada", doc.Status))
	}
	var it *entity.FiscalEntryItem
	for _, x := range doc.Itens {
		if x.ID == dto.ItemID {
			it = x
		}
	}
	if it == nil {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("linha %d não pertence a esta nota", dto.ItemID))
	}
	if it.ItemCode == nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("item %d da nota: concilie o item do cadastro antes de resolver a unidade", it.Sequence))
	}
	memorizar := strings.ToUpper(strings.TrimSpace(dto.Memorizar))
	if memorizar == "" {
		memorizar = MemorizarNenhum
	}
	if memorizar != MemorizarSinonimo && (dto.Fator == nil || !dto.Fator.IsPositive()) {
		return nil, errorsuc.NewValidationError("informe o fator (maior que zero): quanto 1 unidade da nota vale na unidade de estoque")
	}
	s := uc.servico()
	cadastro, err := s.cadastroDosItens(ctx, doc)
	if err != nil {
		return nil, err
	}
	estoque := strings.ToUpper(strings.TrimSpace(cadastro[*it.ItemCode].UOM))
	norm, err := s.normalizacoes(ctx, doc)
	if err != nil {
		return nil, err
	}
	canon, via := canonica(norm, textoUnidade(it.UOM))
	unidadeNota := strings.ToUpper(textoUnidade(it.UOM))

	var vinculos []repository.SupplierItemLink
	switch memorizar {
	case MemorizarNenhum:
		definirFator(it, *dto.Fator, it.Quantity.Mul(*dto.Fator).Round(6), OrigemManual,
			fmt.Sprintf("informado na conferência: 1 %s = %s %s", unidadeNota, dto.Fator.String(), estoque))
	case MemorizarItem:
		if via == "" {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("a unidade %q da nota não está no catálogo: memorize-a como sinônimo de uma unidade do catálogo primeiro", textoUnidade(it.UOM)))
		}
		nota := fmt.Sprintf("memorizada da NF-e %d (%s)", doc.NumeroNF, doc.RazaoSocialEmitente)
		if _, err := uc.Conversoes.Create(ctx, request.CreateItemConversionDTO{
			ItemCode: request.TextCode(fmt.Sprint(*it.ItemCode)), Mask: it.Mask, FromUOM: canon, ToUOM: estoque,
			Factor: *dto.Fator, ToleranceType: "VALUE", Notes: &nota, Origin: "NOTA_FISCAL",
		}); err != nil {
			return nil, err
		}
		limparFator(it) // a resolução de unidades acha a conversão nova
	case MemorizarFornecedor:
		if doc.SupplierCode == nil {
			return nil, errorsuc.NewValidationError("o emitente não está cadastrado como fornecedor: cadastre-o para memorizar o fator no vínculo")
		}
		if it.SupplierItemCode == nil || strings.TrimSpace(*it.SupplierItemCode) == "" {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("item %d da nota não tem código do fornecedor para memorizar", it.Sequence))
		}
		f := *dto.Fator
		u := textoUnidade(it.UOM)
		vinculos = append(vinculos, repository.SupplierItemLink{
			SupplierCode: *doc.SupplierCode, ItemCode: *it.ItemCode, SupplierItemCode: *it.SupplierItemCode,
			SupplierDescription: deref(it.Description), XMLUOM: &u, ConversionFactor: &f, Barcode: it.EAN, CreatedBy: userID,
		})
		aplicarConversao(it, &f, &u)
	case MemorizarSinonimo:
		destino := strings.ToUpper(strings.TrimSpace(dto.UnidadeCatalogo))
		if destino == "" {
			return nil, errorsuc.NewValidationError("informe a unidade do catálogo de que a unidade da nota é sinônimo")
		}
		// Pela nota o sinônimo é sempre do fornecedor; o sinônimo geral (todas
		// as notas) é cadastro do catálogo, feito pelo administrador.
		if doc.SupplierCode == nil {
			return nil, errorsuc.NewValidationError("o emitente não está cadastrado como fornecedor: cadastre-o para memorizar o sinônimo da unidade")
		}
		if _, err := uc.Sinonimos.CreateAlias(ctx, request.UnitAliasDTO{Alias: textoUnidade(it.UOM), UnitCode: destino, SupplierCode: doc.SupplierCode}); err != nil {
			return nil, err
		}
		limparFator(it)
	default:
		return nil, errorsuc.NewValidationError("memorizar deve ser NENHUM, ITEM, FORNECEDOR ou SINONIMO")
	}

	// Mesma preparação da conciliação, sem mexer nos vínculos com o pedido.
	s.SemVinculoAutomatico = map[int64]bool{}
	for _, x := range doc.Itens {
		s.SemVinculoAutomatico[x.ID] = true
	}
	if err := s.Preparar(ctx, doc); err != nil {
		return nil, err
	}
	status, err := s.Status(ctx, doc)
	if err != nil {
		return nil, err
	}
	cab := repository.HeaderConciliation{EntryOperationCode: doc.EntryOperationCode, PurchaseOrderCode: doc.PurchaseOrderCode}
	if err := uc.Docs.SaveConciliation(ctx, entryID, cab, conciliacoesDoDoc(doc), vinculos, nil, status); err != nil {
		return nil, err
	}
	atualizado, err := uc.Docs.GetEntryDocument(ctx, entryID)
	if err != nil {
		return nil, err
	}
	return s.Responder(ctx, atualizado)
}
