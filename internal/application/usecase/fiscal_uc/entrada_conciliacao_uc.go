package fiscal_uc

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/shopspring/decimal"
	"golang.org/x/text/unicode/norm"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entrada"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

// SaveFiscalEntryConciliationUseCase grava a conciliação da nota de entrada:
// para cada item, o item do cadastro, o plano de contas e o centro de custo; e
// as parcelas com a distribuição por plano de contas. Quando pedido, memoriza
// o vínculo produto × fornecedor para a próxima nota já vir conciliada.
type SaveFiscalEntryConciliationUseCase struct {
	Docs        repository.FiscalEntryDocumentRepository
	Fiscal      repository.FiscalRepository
	Tolerancias ports.PurchaseToleranceEvaluator
	Auth        ports.AuthService
}

func (uc *SaveFiscalEntryConciliationUseCase) Execute(ctx context.Context, entryID int64, dto request.SaveFiscalEntryConciliationDTO) (*response.FiscalEntryResponse, error) {
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
	// Fornecedor cadastrado depois da importação: a nota passa a ser dele antes
	// de memorizar vínculos ou procurar pedido.
	if doc.SupplierCode == nil {
		if _, err := (&EntradaServico{Docs: uc.Docs}).Fornecedor(ctx, doc); err != nil {
			return nil, err
		}
	}
	_, _, alvosAntes, _ := entrada.PlanoFinanceiro(doc)

	porID := map[int64]*entity.FiscalEntryItem{}
	for _, it := range doc.Itens {
		porID[it.ID] = it
	}

	// Confere de uma vez que itens, planos, centros e almoxarifados existem
	// nesta empresa.
	var codigos, planos, centros, depositos []int64
	for _, c := range dto.Itens {
		if c.ItemCode != nil {
			codigos = append(codigos, *c.ItemCode)
		}
		if c.PlanoContasID != nil {
			planos = append(planos, *c.PlanoContasID)
		}
		if c.CentroCustoID != nil {
			centros = append(centros, *c.CentroCustoID)
		}
		if c.WarehouseID != nil {
			depositos = append(depositos, *c.WarehouseID)
		}
	}
	for _, p := range dto.Parcelas {
		for _, a := range p.Distribuicao {
			planos = append(planos, a.PlanoContasID)
			if a.CentroCustoID != nil {
				centros = append(centros, *a.CentroCustoID)
			}
		}
	}
	itensCadastro, err := uc.Docs.ItemsByCode(ctx, codigos)
	if err != nil {
		return nil, err
	}
	planosOK, err := uc.Docs.ExistingPlanos(ctx, planos)
	if err != nil {
		return nil, err
	}
	centrosOK, err := uc.Docs.ExistingCentros(ctx, centros)
	if err != nil {
		return nil, err
	}
	depositosOK, err := uc.Docs.ExistingWarehouses(ctx, depositos)
	if err != nil {
		return nil, err
	}
	for _, p := range dto.Parcelas {
		for _, a := range p.Distribuicao {
			if !planosOK[a.PlanoContasID] {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("parcela %d: plano de contas %d não existe ou está inativo", p.Numero, a.PlanoContasID))
			}
			if a.CentroCustoID != nil && !centrosOK[*a.CentroCustoID] {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("parcela %d: centro de custo %d não existe ou está inativo", p.Numero, *a.CentroCustoID))
			}
		}
	}

	type escolha struct {
		strategy       *string
		itemSupplierID *int64
	}
	escolhas := map[int64]escolha{}
	semVinculoAuto, desvinculados := map[int64]bool{}, map[int64]bool{}
	cfopManual := map[int64]string{}
	var vinculos []repository.SupplierItemLink
	for _, c := range dto.Itens {
		it, ok := porID[c.ID]
		if !ok {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("o item %d não pertence à nota %d", c.ID, entryID))
		}
		if c.ItemCode != nil {
			cad, ok := itensCadastro[*c.ItemCode]
			if !ok {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("item %d da nota: o item %d não existe no cadastro desta empresa", it.Sequence, *c.ItemCode))
			}
			if !cad.IsActive {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("item %d da nota: o item %d (%s) não está ativo", it.Sequence, cad.Code, cad.Name))
			}
		}
		if c.PlanoContasID != nil && !planosOK[*c.PlanoContasID] {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("item %d da nota: plano de contas %d não existe ou está inativo", it.Sequence, *c.PlanoContasID))
		}
		if c.CentroCustoID != nil && !centrosOK[*c.CentroCustoID] {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("item %d da nota: centro de custo %d não existe ou está inativo", it.Sequence, *c.CentroCustoID))
		}
		if c.WarehouseID != nil {
			if _, ok := depositosOK[*c.WarehouseID]; !ok {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("item %d da nota: almoxarifado %d não existe nesta empresa", it.Sequence, *c.WarehouseID))
			}
		}
		if c.FatorConversao != nil && !c.FatorConversao.IsPositive() {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("item %d da nota: o fator de conversão precisa ser maior que zero", it.Sequence))
		}
		if c.CfopEntrada != nil {
			cf := soDigitos(*c.CfopEntrada)
			if cf != "" {
				if len(cf) != 4 || (cf[0] != '1' && cf[0] != '2' && cf[0] != '3') {
					return nil, errorsuc.NewValidationError(fmt.Sprintf("item %d da nota: CFOP de entrada %q inválido (1xxx, 2xxx ou 3xxx)", it.Sequence, *c.CfopEntrada))
				}
				cfopManual[it.ID] = cf
			}
		}

		mesmoItem := mesmoInt(it.ItemCode, c.ItemCode)
		var esc escolha
		switch {
		case c.ItemCode == nil:
			st := "NAO_RESOLVIDO"
			esc.strategy = &st
		case mesmoItem && it.ResolutionStrategy != nil && *it.ResolutionStrategy != "NAO_RESOLVIDO":
			esc.strategy, esc.itemSupplierID = it.ResolutionStrategy, it.ItemSupplierID
		default:
			st := "MANUAL"
			esc.strategy = &st
		}
		escolhas[it.ID] = esc

		fator := c.FatorConversao
		if fator == nil && mesmoItem {
			fator = it.FatorConversao
		}
		if !mesmoItem {
			// Outro item: almoxarifado e pedido escolhidos para o anterior não valem.
			it.WarehouseID, it.PurchaseOrderItemCode, it.PurchaseOrderCode = nil, nil, nil
		}
		it.ItemCode, it.PlanoContasID, it.CentroCustoID = c.ItemCode, c.PlanoContasID, c.CentroCustoID
		it.EntryOperationCode = c.EntryOperationCode
		if c.WarehouseID != nil {
			it.WarehouseID = c.WarehouseID
		}
		// O vínculo automático com o pedido vale para o item recém-conciliado;
		// item que já estava com este cadastro e sem pedido ficou assim de
		// propósito, e "0" é o usuário desfazendo o vínculo.
		if mesmoItem {
			semVinculoAuto[it.ID] = true
		}
		if c.PurchaseOrderItemCode != nil {
			if *c.PurchaseOrderItemCode == 0 {
				it.PurchaseOrderItemCode, it.PurchaseOrderCode = nil, nil
				semVinculoAuto[it.ID] = true
				desvinculados[it.ID] = true
			} else if it.PurchaseOrderItemCode == nil || *it.PurchaseOrderItemCode != *c.PurchaseOrderItemCode {
				l := *c.PurchaseOrderItemCode
				it.PurchaseOrderItemCode, it.PurchaseOrderCode = &l, nil
			}
		}
		it.FatorConversao, it.QuantidadeEstoque = nil, nil
		if c.ItemCode != nil {
			f := umDec
			if fator != nil {
				f = *fator
			}
			q := decimal.NewFromFloat(it.Quantity).Mul(f).Round(6)
			it.FatorConversao, it.QuantidadeEstoque = &f, &q
		}

		if c.LembrarVinculo && c.ItemCode != nil {
			if doc.SupplierCode == nil {
				return nil, errorsuc.NewValidationError("o emitente não está cadastrado como fornecedor: cadastre-o para memorizar o vínculo dos itens")
			}
			if it.SupplierItemCode == nil || strings.TrimSpace(*it.SupplierItemCode) == "" {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("item %d da nota não tem código do fornecedor para memorizar", it.Sequence))
			}
			var fatorVinculo *decimal.Decimal
			var unidade *string
			if c.FatorConversao != nil && !c.FatorConversao.Equal(umDec) {
				fatorVinculo, unidade = c.FatorConversao, it.UOM
			}
			vinculos = append(vinculos, repository.SupplierItemLink{
				SupplierCode:        *doc.SupplierCode,
				ItemCode:            *c.ItemCode,
				SupplierItemCode:    *it.SupplierItemCode,
				SupplierDescription: deref(it.Description),
				XMLUOM:              unidade,
				ConversionFactor:    fatorVinculo,
				Barcode:             it.EAN,
				CreatedBy:           userID,
			})
		}
	}
	doc.EntryOperationCode = dto.EntryOperationCode
	if dto.PurchaseOrderCode != nil {
		if doc.PurchaseOrderCode == nil || *doc.PurchaseOrderCode != *dto.PurchaseOrderCode {
			// Pedido da nota trocado: os itens sem pedido podem casar com o novo,
			// menos os que o usuário desvinculou agora.
			for id := range semVinculoAuto {
				if !desvinculados[id] {
					delete(semVinculoAuto, id)
				}
			}
		}
		doc.PurchaseOrderCode = dto.PurchaseOrderCode
	}

	// Mesma preparação da importação: operação (TES), pedido, almoxarifado e
	// CFOP de entrada. O CFOP digitado prevalece.
	servico := uc.servico()
	servico.SemVinculoAutomatico = semVinculoAuto
	if err := servico.Preparar(ctx, doc); err != nil {
		return nil, err
	}
	for _, it := range doc.Itens {
		if cf, ok := cfopManual[it.ID]; ok {
			v := cf
			it.CfopEntrada = &v
		}
	}

	// Parcelas: as enviadas substituem; senão a distribuição é refeita quando
	// o que cada plano deve receber mudou (classificação, operação, retenção).
	_, _, alvosDepois, _ := entrada.PlanoFinanceiro(doc)
	alvosMudaram := !mesmosAlvos(alvosAntes, alvosDepois)
	var parcelas []*entity.FiscalEntryInstallment
	if len(dto.Parcelas) > 0 {
		if parcelas, err = parcelasDoDTO(dto.Parcelas); err != nil {
			return nil, err
		}
		semDistribuicao := true
		for _, p := range parcelas {
			if len(p.Distribuicao) > 0 {
				semDistribuicao = false
			}
		}
		doc.Parcelas = parcelas
		if semDistribuicao || dto.RecalcularDistribuicao {
			if err := Distribuir(doc); err != nil {
				return nil, errorsuc.NewValidationError(err.Error())
			}
		}
	} else if dto.RecalcularDistribuicao || alvosMudaram {
		parcelas = doc.Parcelas
		for _, p := range parcelas {
			p.ID = 0
		}
		if err := Distribuir(doc); err != nil {
			return nil, errorsuc.NewValidationError(err.Error())
		}
		if parcelas == nil {
			parcelas = []*entity.FiscalEntryInstallment{}
		}
	}

	conciliacoes := make([]repository.ItemConciliation, 0, len(doc.Itens))
	for _, it := range doc.Itens {
		c := repository.ItemConciliation{
			ItemID: it.ID, ItemCode: it.ItemCode, ItemSupplierID: it.ItemSupplierID, ResolutionStrategy: it.ResolutionStrategy,
			PlanoContasID: it.PlanoContasID, CentroCustoID: it.CentroCustoID,
			FatorConversao: it.FatorConversao, QuantidadeEstoque: it.QuantidadeEstoque,
			CfopEntrada: it.CfopEntrada, EntryOperationCode: it.EntryOperationCode,
			MovimentaEstoque: it.MovimentaEstoque, GeraFinanceiro: it.GeraFinanceiro, WarehouseID: it.WarehouseID,
			PurchaseOrderCode: it.PurchaseOrderCode, PurchaseOrderItemCode: it.PurchaseOrderItemCode,
			GeraCreditoICMS: it.GeraCreditoICMS, GeraCreditoIPI: it.GeraCreditoIPI, GeraCreditoPIS: it.GeraCreditoPIS,
			GeraCreditoCOFINS: it.GeraCreditoCOFINS, GeraCreditoIBSCBS: it.GeraCreditoIBSCBS,
		}
		if esc, ok := escolhas[it.ID]; ok {
			c.ResolutionStrategy, c.ItemSupplierID = esc.strategy, esc.itemSupplierID
		}
		conciliacoes = append(conciliacoes, c)
	}

	status, err := servico.Status(ctx, doc)
	if err != nil {
		return nil, err
	}
	cab := repository.HeaderConciliation{EntryOperationCode: doc.EntryOperationCode, PurchaseOrderCode: doc.PurchaseOrderCode}
	if err := uc.Docs.SaveConciliation(ctx, entryID, cab, conciliacoes, vinculos, parcelas, status); err != nil {
		return nil, err
	}
	atualizado, err := uc.Docs.GetEntryDocument(ctx, entryID)
	if err != nil {
		return nil, err
	}
	return servico.Responder(ctx, atualizado)
}

func (uc *SaveFiscalEntryConciliationUseCase) servico() *EntradaServico {
	return &EntradaServico{Docs: uc.Docs, Fiscal: uc.Fiscal, Tolerancias: uc.Tolerancias}
}

func mesmosAlvos(a, b map[entity.ChaveConta]decimal.Decimal) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || !w.Equal(v) {
			return false
		}
	}
	return true
}

func mesmoInt(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// SuggestFiscalEntryItemUseCase sugere itens do cadastro para uma linha da
// nota: primeiro o vínculo do fornecedor, depois descrição parecida com o
// mesmo NCM. O usuário confirma — sugestão nunca concilia sozinha.
type SuggestFiscalEntryItemUseCase struct {
	Docs repository.FiscalEntryDocumentRepository
	Auth ports.AuthService
}

func (uc *SuggestFiscalEntryItemUseCase) Execute(ctx context.Context, entryID, itemID int64, busca string) ([]response.FiscalEntryItemSuggestion, error) {
	if !uc.Auth.CanGetFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	doc, err := uc.Docs.GetEntryDocument(ctx, entryID)
	if err != nil {
		return nil, err
	}
	var linha *entity.FiscalEntryItem
	for _, it := range doc.Itens {
		if it.ID == itemID {
			linha = it
		}
	}
	if linha == nil {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("item %d não encontrado na nota %d", itemID, entryID))
	}

	descricao := deref(linha.Description)
	ncm := soDigitos(deref(linha.Ncm))
	unidade := strings.ToUpper(strings.TrimSpace(deref(linha.UOM)))
	alvo := tokens(descricao)

	melhores := map[int64]response.FiscalEntryItemSuggestion{}
	guardar := func(s response.FiscalEntryItemSuggestion) {
		if atual, ok := melhores[s.ItemCode]; !ok || s.Score > atual.Score {
			melhores[s.ItemCode] = s
		}
	}

	// 1. Vínculos já cadastrados com este fornecedor.
	if doc.SupplierCode != nil {
		vinculos, err := uc.Docs.SupplierLinksForEntry(ctx, *doc.SupplierCode)
		if err != nil {
			return nil, err
		}
		codigoNota := normCodigo(deref(linha.SupplierItemCode))
		for _, v := range vinculos {
			s := response.FiscalEntryItemSuggestion{ItemCode: v.Code, Name: v.Name, UOM: v.UOM, NCM: v.NCM, ItemSupplierID: v.ItemSupplierID, IsActive: v.IsActive}
			switch {
			case codigoNota != "" && normCodigo(v.SupplierItemCode) == codigoNota:
				s.Score, s.Motivo = 1, "código do fornecedor já vinculado a este item"
			case linha.EAN != nil && soDigitos(v.Barcode) == *linha.EAN:
				s.Score, s.Motivo = 0.98, "código de barras do vínculo com o fornecedor"
			default:
				sim := similaridade(alvo, tokens(v.Name+" "+v.SupplierDescription))
				if sim < 0.2 {
					continue
				}
				s.Score, s.Motivo = 0.5+0.4*sim, "item já comprado deste fornecedor com descrição parecida"
			}
			guardar(s)
		}
	}

	// 2. Busca no cadastro pela descrição (ou pelo que o usuário digitou).
	termos := termosDeBusca(alvo)
	if strings.TrimSpace(busca) != "" {
		termos = termosDeBusca(tokens(busca))
		if n, err := strconv.ParseInt(strings.TrimSpace(busca), 10, 64); err == nil {
			termos = append(termos, strconv.FormatInt(n, 10))
		}
	}
	candidatos, err := uc.Docs.SearchItems(ctx, termos, 200)
	if err != nil {
		return nil, err
	}
	referencia := alvo
	if strings.TrimSpace(busca) != "" {
		referencia = tokens(busca)
	}
	for _, c := range candidatos {
		sim := similaridade(referencia, tokens(c.Name))
		score := 0.7 * sim
		motivos := []string{}
		if sim > 0 {
			motivos = append(motivos, fmt.Sprintf("descrição %.0f%% parecida", sim*100))
		}
		if ncm != "" && soDigitos(c.NCM) == ncm {
			score += 0.25
			motivos = append(motivos, "mesmo NCM")
		}
		if unidade != "" && strings.EqualFold(c.UOM, unidade) {
			score += 0.05
			motivos = append(motivos, "mesma unidade")
		}
		if strings.TrimSpace(busca) != "" && strconv.FormatInt(c.Code, 10) == strings.TrimSpace(busca) {
			score, motivos = 1, []string{"código digitado"}
		}
		if score <= 0.05 {
			continue
		}
		guardar(response.FiscalEntryItemSuggestion{
			ItemCode: c.Code, Name: c.Name, UOM: c.UOM, NCM: c.NCM, IsActive: c.IsActive,
			Score: score, Motivo: strings.Join(motivos, ", "),
		})
	}

	out := make([]response.FiscalEntryItemSuggestion, 0, len(melhores))
	for _, s := range melhores {
		s.Score = float64(int(s.Score*1000)) / 1000
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].ItemCode < out[j].ItemCode
	})
	if len(out) > 15 {
		out = out[:15]
	}
	return out, nil
}

var palavrasVazias = map[string]bool{
	"DE": true, "DA": true, "DO": true, "DAS": true, "DOS": true, "COM": true, "PARA": true,
	"EM": true, "E": true, "A": true, "O": true, "UN": true, "PC": true, "PCS": true,
}

// tokens normaliza a descrição: sem acento, maiúscula, sem palavras vazias.
// "Luva de raspa c/ cano longo" e "LUVA RASPA CANO LONGO" viram o mesmo conjunto.
func tokens(s string) map[string]bool {
	semAcento := strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, norm.NFD.String(strings.ToUpper(s)))
	out := map[string]bool{}
	for _, t := range strings.FieldsFunc(semAcento, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if len(t) < 2 || palavrasVazias[t] {
			continue
		}
		out[t] = true
	}
	return out
}

func similaridade(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	comum := 0
	for t := range a {
		if b[t] {
			comum++
		}
	}
	uniao := len(a) + len(b) - comum
	return float64(comum) / float64(uniao)
}

// termosDeBusca escolhe as palavras mais longas (mais distintivas) para a
// busca no banco.
func termosDeBusca(ts map[string]bool) []string {
	lista := make([]string, 0, len(ts))
	for t := range ts {
		if len(t) >= 3 {
			lista = append(lista, t)
		}
	}
	sort.Slice(lista, func(i, j int) bool {
		if len(lista[i]) != len(lista[j]) {
			return len(lista[i]) > len(lista[j])
		}
		return lista[i] < lista[j]
	})
	if len(lista) > 6 {
		lista = lista[:6]
	}
	return lista
}

// PedidosDoItemUseCase lista as linhas de pedido de compra do fornecedor da
// nota, para o item, que ainda têm saldo a faturar — o usuário escolhe a qual
// pedido a linha da nota corresponde (pedido × nota × recebimento).
type PedidosDoItemUseCase struct {
	Docs repository.FiscalEntryDocumentRepository
	Auth ports.AuthService
}

func (uc *PedidosDoItemUseCase) Execute(ctx context.Context, entryID, itemID int64, itemCode *int64) ([]response.LinhaPedidoCompraResponse, error) {
	if !uc.Auth.CanGetFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	doc, err := uc.Docs.GetEntryDocument(ctx, entryID)
	if err != nil {
		return nil, err
	}
	if doc.SupplierCode == nil {
		if _, err := (&EntradaServico{Docs: uc.Docs}).Fornecedor(ctx, doc); err != nil {
			return nil, err
		}
	}
	if doc.SupplierCode == nil {
		return nil, errorsuc.NewValidationError("o emitente não está cadastrado como fornecedor: não há pedido de compra para vincular")
	}
	var linha *entity.FiscalEntryItem
	for _, it := range doc.Itens {
		if it.ID == itemID {
			linha = it
		}
	}
	if linha == nil {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("item %d não encontrado na nota %d", itemID, entryID))
	}
	code := itemCode
	if code == nil {
		code = linha.ItemCode
	}
	if code == nil {
		return nil, errorsuc.NewValidationError("concilie o item com o cadastro antes de escolher o pedido")
	}
	ls, err := uc.Docs.OpenPurchaseOrderLines(ctx, *doc.SupplierCode, *code)
	if err != nil {
		return nil, err
	}
	out := make([]response.LinhaPedidoCompraResponse, 0, len(ls))
	for _, l := range ls {
		f := l.FatorEstoque()
		out = append(out, response.LinhaPedidoCompraResponse{
			Code: l.Code, PurchaseOrderCode: l.PurchaseOrderCode, OrderNumber: l.OrderNumber, Sequence: l.Sequence, ItemCode: l.ItemCode,
			Status: l.Status, RequestedQty: l.RequestedQty.InexactFloat64(), ReceivedQty: l.ReceivedQty.InexactFloat64(),
			InvoicedQty: l.InvoicedQty.InexactFloat64(), SaldoAFaturar: l.RequestedQty.Sub(l.InvoicedQty).Sub(l.CancelledQty).InexactFloat64(),
			UnitPrice: l.UnitPrice.InexactFloat64(), FatorEstoque: f.InexactFloat64(), WarehouseID: l.WarehouseID,
		})
	}
	return out, nil
}
