package fiscal_uc

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entrada"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

var (
	cemDec = decimal.NewFromInt(100)
	umDec  = decimal.NewFromInt(1)
)

// montarEntradaDoXML transforma a NF-e lida no documento de entrada. O código
// do produto no XML é o código DO FORNECEDOR: ele nunca é tomado como código
// do nosso cadastro (era o que acontecia, e uma nota com cProd "100" entrava
// como o NOSSO item 100). A ligação com o cadastro é a conciliação.
func montarEntradaDoXML(n *NFeLida, conteudo string, dataEntrada time.Time) (*entity.FiscalEntry, error) {
	numero, err := strconv.ParseInt(n.Numero, 10, 64)
	if err != nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o número da nota no XML (%q) não é um número", n.Numero))
	}
	modelo := n.Modelo
	if modelo == "" {
		modelo = "55"
	}
	serie := n.Serie
	if serie == "" {
		serie = "1"
	}
	e := &entity.FiscalEntry{
		ChaveAcesso:               strPtr(n.ChaveAcesso),
		NumeroNF:                  numero,
		Serie:                     serie,
		Modelo:                    modelo,
		DataEmissao:               n.DataEmissao,
		DataEntrada:               dataEntrada,
		CnpjEmitente:              n.EmitenteCNPJ,
		RazaoSocialEmitente:       truncarTexto(n.EmitenteNome, 200),
		IEEmitente:                strPtr(truncarTexto(soDigitos(n.EmitenteIE), 14)),
		UFEmitente:                strPtr(n.EmitenteUF),
		ValorProdutos:             n.ValorProdutos.InexactFloat64(),
		ValorFrete:                n.ValorFrete.InexactFloat64(),
		ValorSeguro:               n.ValorSeguro.InexactFloat64(),
		ValorDesconto:             n.ValorDesconto.InexactFloat64(),
		ValorIPI:                  n.ValorIPI.InexactFloat64(),
		ValorICMS:                 n.ValorICMS.InexactFloat64(),
		ValorPIS:                  n.ValorPIS.InexactFloat64(),
		ValorCOFINS:               n.ValorCOFINS.InexactFloat64(),
		ValorTotal:                n.ValorTotal.InexactFloat64(),
		ValorICMSST:               n.ValorICMSST,
		ValorOutras:               n.ValorOutras,
		TipoDocumento:             "NFE",
		Status:                    entity.EntryStatusPending,
		NaturezaOperacao:          strPtr(truncarTexto(n.NaturezaOperacao, 60)),
		CnpjDestinatario:          strPtr(n.DestinatarioCNPJ),
		Protocolo:                 strPtr(n.Protocolo),
		ModalidadeFrete:           strPtr(n.ModalidadeFrete),
		InformacoesComplementares: strPtr(n.InformacoesComplementares),
		XMLContent:                strPtr(conteudo),
		SemPagamento:              n.SemPagamento(),
		BaseIBSCBS:                n.BaseIBSCBS,
		ValorIBS:                  n.ValorIBS,
		ValorCBS:                  n.ValorCBS,
		ValorIS:                   n.ValorIS,
		ValorRetPIS:               n.ValorRetPIS,
		ValorRetCOFINS:            n.ValorRetCOFINS,
		ValorRetCSLL:              n.ValorRetCSLL,
		BaseIRRF:                  n.BaseIRRF,
		ValorIRRF:                 n.ValorIRRF,
		BaseRetPrev:               n.BaseRetPrev,
		ValorRetPrev:              n.ValorRetPrev,
		ValorISSRet:               n.ValorISSRet,
		StockStatus:               entity.StockStatusPendente,
	}
	for _, it := range n.Itens {
		item := &entity.FiscalEntryItem{
			Sequence:          it.Numero,
			SupplierItemCode:  strPtr(it.CodigoProduto),
			Description:       strPtr(it.Descricao),
			UOM:               strPtr(it.Unidade),
			Ncm:               strPtr(it.NCM),
			Cfop:              it.CFOP,
			Quantity:          it.Quantidade.InexactFloat64(),
			UnitPrice:         it.ValorUnitario.InexactFloat64(),
			TotalPrice:        it.ValorProduto.InexactFloat64(),
			BaseICMS:          it.BaseICMS.InexactFloat64(),
			AliqICMS:          it.AliqICMS.Div(cemDec).InexactFloat64(),
			ValorICMS:         it.ValorICMS.InexactFloat64(),
			BaseIPI:           it.BaseIPI.InexactFloat64(),
			AliqIPI:           it.AliqIPI.Div(cemDec).InexactFloat64(),
			ValorIPI:          it.ValorIPI.InexactFloat64(),
			ValorPIS:          it.ValorPIS.InexactFloat64(),
			ValorCOFINS:       it.ValorCOFINS.InexactFloat64(),
			CstICMS:           strPtr(it.CSTICMS),
			CstIPI:            strPtr(it.CSTIPI),
			CstPIS:            strPtr(it.CSTPIS),
			CstCOFINS:         strPtr(it.CSTCOFINS),
			GeraCreditoICMS:   it.ValorICMS.IsPositive(),
			GeraCreditoIPI:    it.ValorIPI.IsPositive(),
			GeraCreditoPIS:    it.ValorPIS.IsPositive(),
			GeraCreditoCOFINS: it.ValorCOFINS.IsPositive(),
			EAN:               strPtr(it.EAN),
			CEST:              strPtr(it.CEST),
			Origem:            strPtr(it.Origem),
			ValorFrete:        it.ValorFrete,
			ValorSeguro:       it.ValorSeguro,
			ValorDesconto:     it.ValorDesconto,
			ValorOutras:       it.ValorOutras,
			BaseICMSST:        it.BaseST,
			ValorICMSST:       it.ValorST,
			ValorContabil:     it.ValorContabil(),
			PedidoCompraXML:   strPtr(it.PedidoCompra),
			ItemPedidoXML:     strPtr(it.ItemPedido),
			MovimentaEstoque:  true,
			GeraFinanceiro:    true,
			CSTIBSCBS:         strPtr(it.CSTIBSCBS),
			ClassTrib:         strPtr(it.ClassTrib),
			BaseIBSCBS:        it.BaseIBSCBS,
			AliqIBSUF:         it.AliqIBSUF,
			ValorIBSUF:        it.ValorIBSUF,
			AliqIBSMun:        it.AliqIBSMun,
			ValorIBSMun:       it.ValorIBSMun,
			ValorIBS:          it.ValorIBS,
			AliqCBS:           it.AliqCBS,
			ValorCBS:          it.ValorCBS,
			ValorIS:           it.ValorIS,
			GeraCreditoIBSCBS: it.ValorIBS.IsPositive() || it.ValorCBS.IsPositive(),
		}
		e.Itens = append(e.Itens, item)
	}
	entrada.AjustarValorContabil(e.Itens, n.ValorTotal)
	e.Parcelas = parcelasDoXML(n)
	return e, nil
}

// parcelasDoXML usa as duplicatas da nota. Sem duplicata, a nota é à vista
// (uma parcela vencendo na emissão), a menos que declare não ter pagamento.
func parcelasDoXML(n *NFeLida) []*entity.FiscalEntryInstallment {
	if n.SemPagamento() || !n.ValorTotal.IsPositive() {
		return nil
	}
	if len(n.Duplicatas) > 0 {
		out := make([]*entity.FiscalEntryInstallment, 0, len(n.Duplicatas))
		for i, d := range n.Duplicatas {
			forma := "BOLETO"
			out = append(out, &entity.FiscalEntryInstallment{
				Numero:         i + 1,
				Documento:      strPtr(d.Numero),
				DataVencimento: d.Vencimento,
				Valor:          d.Valor,
				FormaPagamento: &forma,
				Origem:         entity.ParcelaOrigemXML,
			})
		}
		return out
	}
	// Sem duplicata: à vista, pelo líquido (as retenções vão para os títulos
	// dos impostos a recolher).
	liquido := n.ValorTotal.Round(2).Sub(n.TotalRetencoes())
	if !liquido.IsPositive() {
		return nil
	}
	return []*entity.FiscalEntryInstallment{{
		Numero:         1,
		DataVencimento: n.DataEmissao,
		Valor:          liquido,
		Origem:         entity.ParcelaOrigemPadrao,
	}}
}

// conciliacaoAutomatica liga os itens da nota ao cadastro pelo vínculo
// produto × fornecedor: primeiro pelo código do fornecedor, depois pelo
// código de barras. Descrição parecida NÃO concilia sozinha — vira sugestão
// para o usuário confirmar, porque item conciliado errado entra no estoque e
// no custo errados sem ninguém perceber.
func conciliacaoAutomatica(ctx context.Context, docs repository.FiscalEntryDocumentRepository, supplierCode *int64, itens []*entity.FiscalEntryItem) error {
	var vinculos []repository.ItemCandidate
	if supplierCode != nil {
		var err error
		if vinculos, err = docs.SupplierLinksForEntry(ctx, *supplierCode); err != nil {
			return err
		}
	}
	porCodigo := map[string][]repository.ItemCandidate{}
	porBarras := map[string][]repository.ItemCandidate{}
	for _, v := range vinculos {
		if c := normCodigo(v.SupplierItemCode); c != "" {
			porCodigo[c] = append(porCodigo[c], v)
		}
		if b := soDigitos(v.Barcode); b != "" {
			porBarras[b] = append(porBarras[b], v)
		}
	}
	agora := time.Now()
	for _, it := range itens {
		if it.ItemCode != nil {
			continue
		}
		var achado *repository.ItemCandidate
		estrategia := ""
		if it.SupplierItemCode != nil {
			if c := porCodigo[normCodigo(*it.SupplierItemCode)]; len(c) == 1 {
				achado, estrategia = &c[0], "CODIGO_EXATO"
			}
		}
		if achado == nil && it.EAN != nil {
			if c := porBarras[*it.EAN]; len(c) == 1 {
				achado, estrategia = &c[0], "EAN"
			}
		}
		if achado == nil && it.EAN != nil {
			m, err := docs.ResolveByEAN(ctx, supplierCode, *it.EAN)
			if err != nil {
				return err
			}
			if m != nil {
				code := m.ItemCode
				it.ItemCode = &code
				it.ItemSupplierID = m.ItemSupplierID
				s := "EAN"
				it.ResolutionStrategy = &s
				it.ResolvedAt = &agora
				aplicarConversao(it, nil, nil)
				continue
			}
		}
		if achado == nil {
			s := "NAO_RESOLVIDO"
			it.ResolutionStrategy = &s
			continue
		}
		code := achado.Code
		it.ItemCode = &code
		it.ItemSupplierID = achado.ItemSupplierID
		it.ResolutionStrategy = &estrategia
		it.ResolvedAt = &agora
		aplicarConversao(it, achado.ConversionFactor, achado.XMLUOM)
	}
	return nil
}

// aplicarConversao calcula a quantidade na unidade do cadastro. O fator do
// vínculo só vale para a unidade do XML para a qual foi cadastrado.
func aplicarConversao(it *entity.FiscalEntryItem, fator *decimal.Decimal, xmlUOM *string) {
	f := umDec
	if fator != nil && fator.IsPositive() {
		mesmaUnidade := xmlUOM == nil || strings.TrimSpace(*xmlUOM) == "" ||
			(it.UOM != nil && strings.EqualFold(strings.TrimSpace(*xmlUOM), strings.TrimSpace(*it.UOM)))
		if mesmaUnidade {
			f = *fator
		}
	}
	q := decimal.NewFromFloat(it.Quantity).Mul(f).Round(6)
	it.FatorConversao = &f
	it.QuantidadeEstoque = &q
}

// sugerirPlanosDeContas preenche o plano de contas dos itens conciliados que
// ainda não têm: o último usado para o item, ou a conta financeira do
// fornecedor nesta empresa.
func sugerirPlanosDeContas(ctx context.Context, docs repository.FiscalEntryDocumentRepository, contaFornecedor *string, itens []*entity.FiscalEntryItem) error {
	var planoFornecedor *int64
	if contaFornecedor != nil {
		var err error
		if planoFornecedor, err = docs.PlanoContasByCodigo(ctx, *contaFornecedor); err != nil {
			return err
		}
	}
	for _, it := range itens {
		if it.PlanoContasID != nil {
			continue
		}
		if it.ItemCode != nil {
			a, err := docs.LastAccountForItem(ctx, *it.ItemCode)
			if err != nil {
				return err
			}
			if a != nil {
				p := a.PlanoContasID
				it.PlanoContasID = &p
				it.CentroCustoID = a.CentroCustoID
				continue
			}
		}
		if planoFornecedor != nil {
			p := *planoFornecedor
			it.PlanoContasID = &p
		}
	}
	return nil
}

// parcelasDoDTO valida e converte as parcelas informadas pelo usuário.
func parcelasDoDTO(in []request.FiscalEntryInstallmentDTO) ([]*entity.FiscalEntryInstallment, error) {
	out := make([]*entity.FiscalEntryInstallment, 0, len(in))
	numeros := map[int]bool{}
	for i, p := range in {
		num := p.Numero
		if num <= 0 {
			num = i + 1
		}
		if numeros[num] {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("a parcela %d foi informada duas vezes", num))
		}
		numeros[num] = true
		venc, err := time.Parse("2006-01-02", strings.TrimSpace(p.DataVencimento))
		if err != nil {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("parcela %d: vencimento %q inválido, use AAAA-MM-DD", num, p.DataVencimento))
		}
		if !p.Valor.IsPositive() {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("parcela %d: o valor precisa ser maior que zero", num))
		}
		parcela := &entity.FiscalEntryInstallment{
			Numero:         num,
			Documento:      p.Documento,
			DataVencimento: venc,
			Valor:          p.Valor.Round(2),
			FormaPagamento: p.FormaPagamento,
			Origem:         entity.ParcelaOrigemManual,
		}
		for _, a := range p.Distribuicao {
			if a.PlanoContasID <= 0 {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("parcela %d: informe o plano de contas de cada valor distribuído", num))
			}
			if a.Valor.IsNegative() {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("parcela %d: valor distribuído negativo", num))
			}
			if a.Valor.IsZero() {
				continue
			}
			parcela.Distribuicao = append(parcela.Distribuicao, entity.InstallmentAllocation{
				PlanoContasID: a.PlanoContasID, CentroCustoID: a.CentroCustoID, Valor: a.Valor.Round(2),
			})
		}
		out = append(out, parcela)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Numero < out[j].Numero })
	return out, nil
}

// ---- resposta ----

func toFiscalEntryDocumentResponse(e *entity.FiscalEntry) *response.FiscalEntryResponse {
	r := toFiscalEntryResponse(e)
	if r == nil {
		return nil
	}
	r.NaturezaOperacao = e.NaturezaOperacao
	r.CnpjDestinatario = e.CnpjDestinatario
	r.Protocolo = e.Protocolo
	r.ValorICMSST = e.ValorICMSST.InexactFloat64()
	r.ValorOutras = e.ValorOutras.InexactFloat64()
	r.ModalidadeFrete = e.ModalidadeFrete
	r.InformacoesComplementares = e.InformacoesComplementares
	r.SemPagamento = e.SemPagamento
	r.SupplierName = e.SupplierName
	r.ApprovedAt = e.ApprovedAt
	r.EntryOperationCode = e.EntryOperationCode
	r.BaseIBSCBS = e.BaseIBSCBS.InexactFloat64()
	r.ValorIBS = e.ValorIBS.InexactFloat64()
	r.ValorCBS = e.ValorCBS.InexactFloat64()
	r.ValorIS = e.ValorIS.InexactFloat64()
	r.ValorRetPIS = e.ValorRetPIS.InexactFloat64()
	r.ValorRetCOFINS = e.ValorRetCOFINS.InexactFloat64()
	r.ValorRetCSLL = e.ValorRetCSLL.InexactFloat64()
	r.BaseIRRF = e.BaseIRRF.InexactFloat64()
	r.ValorIRRF = e.ValorIRRF.InexactFloat64()
	r.BaseRetPrev = e.BaseRetPrev.InexactFloat64()
	r.ValorRetPrev = e.ValorRetPrev.InexactFloat64()
	r.ValorISSRet = e.ValorISSRet.InexactFloat64()
	r.TotalRetencoes = e.TotalRetencoes().InexactFloat64()
	_, aPagar, _, _ := entrada.PlanoFinanceiro(e)
	r.ValorAPagar = aPagar.InexactFloat64()
	r.StockStatus = e.StockStatus
	r.CancelledAt = e.CancelledAt
	r.CancelReason = e.CancelReason
	for _, ret := range entrada.Retencoes(e) {
		r.Retencoes = append(r.Retencoes, response.FiscalEntryRetencao{
			Tipo: ret.Tipo, Descricao: ret.Descricao, Valor: ret.Valor.InexactFloat64(), Vencimento: ret.Vencimento.Format("2006-01-02"),
		})
	}

	for i, it := range e.Itens {
		if i >= len(r.Itens) {
			break
		}
		ri := &r.Itens[i]
		ri.UOM = it.UOM
		ri.EAN = it.EAN
		ri.CEST = it.CEST
		ri.Origem = it.Origem
		ri.ValorFrete = it.ValorFrete.InexactFloat64()
		ri.ValorSeguro = it.ValorSeguro.InexactFloat64()
		ri.ValorDesconto = it.ValorDesconto.InexactFloat64()
		ri.ValorOutras = it.ValorOutras.InexactFloat64()
		ri.BaseICMSST = it.BaseICMSST.InexactFloat64()
		ri.ValorICMSST = it.ValorICMSST.InexactFloat64()
		ri.ValorContabil = it.ValorContabil.InexactFloat64()
		ri.FatorConversao = decPtrFloat(it.FatorConversao)
		ri.QuantidadeEstoque = decPtrFloat(it.QuantidadeEstoque)
		ri.PedidoCompraXML = it.PedidoCompraXML
		ri.PlanoContasID = it.PlanoContasID
		ri.CentroCustoID = it.CentroCustoID
		ri.ItemName = it.ItemName
		ri.ItemUOM = it.ItemUOM
		ri.PlanoContasCodigo = it.PlanoContasCodigo
		ri.PlanoContasNome = it.PlanoContasNome
		ri.CentroCustoNome = it.CentroCustoNome
		ri.CfopEntrada = it.CfopEntrada
		ri.EntryOperationCode = it.EntryOperationCode
		ri.EntryOperationName = it.EntryOperationName
		ri.MovimentaEstoque = it.MovimentaEstoque
		ri.GeraFinanceiro = it.GeraFinanceiro
		ri.WarehouseID = it.WarehouseID
		ri.WarehouseName = it.WarehouseName
		ri.PurchaseOrderCode = it.PurchaseOrderCode
		ri.PurchaseOrderItemCode = it.PurchaseOrderItemCode
		ri.PurchaseOrderNumber = it.PurchaseOrderNumber
		ri.PurchaseOrderSequence = it.PurchaseOrderSequence
		ri.QtdRecebidaAntes = it.QtdRecebidaAntes.InexactFloat64()
		ri.StockMovementID = it.StockMovementID
		ri.CustoAquisicao = it.CustoAquisicao.InexactFloat64()
		ri.ItemNCM = it.ItemNCM
		ri.CSTIBSCBS = it.CSTIBSCBS
		ri.ClassTrib = it.ClassTrib
		ri.BaseIBSCBS = it.BaseIBSCBS.InexactFloat64()
		ri.ValorIBS = it.ValorIBS.InexactFloat64()
		ri.ValorCBS = it.ValorCBS.InexactFloat64()
		ri.ValorIS = it.ValorIS.InexactFloat64()
		ri.GeraCreditoIBSCBS = it.GeraCreditoIBSCBS
		if it.ItemCode != nil {
			r.ItensConciliados++
		}
		if it.PlanoContasID != nil {
			r.ItensClassificados++
		}
	}

	for _, p := range e.Parcelas {
		pr := response.FiscalEntryInstallmentResponse{
			ID:             p.ID,
			Numero:         p.Numero,
			Documento:      p.Documento,
			DataVencimento: p.DataVencimento.Format("2006-01-02"),
			Valor:          p.Valor.InexactFloat64(),
			FormaPagamento: p.FormaPagamento,
			Origem:         p.Origem,
			ContaPagarID:   p.ContaPagarID,
			Distribuicao:   []response.FiscalEntryInstallmentAllocResult{},
		}
		for _, a := range p.Distribuicao {
			pr.Distribuicao = append(pr.Distribuicao, response.FiscalEntryInstallmentAllocResult{
				PlanoContasID: a.PlanoContasID, CentroCustoID: a.CentroCustoID, Valor: a.Valor.InexactFloat64(),
			})
		}
		r.Parcelas = append(r.Parcelas, pr)
	}

	totais := entrada.TotalPorConta(e.Itens)
	totalGeral := decimal.Zero
	for _, v := range totais {
		totalGeral = totalGeral.Add(v)
	}
	nomes := map[entity.ChaveConta]*entity.FiscalEntryItem{}
	for _, it := range e.Itens {
		if k := it.ChaveConta(); k != nil {
			if _, ok := nomes[*k]; !ok {
				nomes[*k] = it
			}
		}
	}
	for _, k := range entrada.ChavesOrdenadas(totais) {
		t := response.FiscalEntryAccountTotal{
			PlanoContasID: k.PlanoContasID,
			CentroCustoID: k.CentroCusto(),
			Valor:         totais[k].InexactFloat64(),
			Percentual:    entrada.Percentual(totais[k], totalGeral).InexactFloat64(),
		}
		if it := nomes[k]; it != nil {
			t.PlanoContasCodigo = deref(it.PlanoContasCodigo)
			t.PlanoContasNome = deref(it.PlanoContasNome)
			t.CentroCustoNome = deref(it.CentroCustoNome)
		}
		r.TotaisPorConta = append(r.TotaisPorConta, t)
	}

	if e.Status == entity.EntryStatusPending || e.Status == entity.EntryStatusConferred {
		pend := entrada.ConferirEntrada(e)
		for _, p := range pend {
			r.Pendencias = append(r.Pendencias, response.FiscalEntryPendencia{Nivel: p.Nivel, Campo: p.Campo, Mensagem: p.Mensagem})
		}
		r.PodeAprovar = !entrada.TemImpedimento(pend)
	}
	return r
}

// ---- utilidades ----

func normCodigo(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

func truncarTexto(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n])
}

func decPtrFloat(d *decimal.Decimal) *float64 {
	if d == nil {
		return nil
	}
	v := d.InexactFloat64()
	return &v
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func strPtr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
