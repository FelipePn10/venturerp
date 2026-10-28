package fiscal_uc

import (
	"context"
	"errors"
	"fmt"
	poentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	"slices"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/engine"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entrada"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

// EntradaServico é o que todos os casos de uso da nota de entrada fazem do
// mesmo jeito — importar, lançar à mão, conferir e aprovar enxergam a nota com
// as mesmas regras:
//
//   - Preparar: tipo de operação de entrada (o "TES") de cada item — CFOP de
//     entrada, se movimenta estoque, se gera financeiro, quais créditos toma —,
//     a linha do pedido de compra (3-way), o almoxarifado e o plano de contas
//     sugeridos pelo pedido;
//   - Distribuir: parcelas pelo valor a pagar ao fornecedor (itens que geram
//     financeiro menos as retenções), na proporção dos planos de contas;
//   - Conferir: pendências da nota + divergências contra as tabelas fiscais, o
//     cadastro e o pedido (preço e quantidade acima da tolerância de compras
//     configurada como bloqueio impedem a aprovação).
type EntradaServico struct {
	Docs        repository.FiscalEntryDocumentRepository
	Fiscal      repository.FiscalRepository
	Tolerancias ports.PurchaseToleranceEvaluator
	// SemVinculoAutomatico: itens (pelo id) que o vínculo automático com o
	// pedido não deve tocar — o usuário desfez o vínculo, ou o item já foi
	// conferido antes e ficou sem pedido de propósito. Nulo = todos (importação).
	SemVinculoAutomatico map[int64]bool
}

// contexto fiscal da empresa: UF (para o CFOP de entrada e a alíquota
// interestadual). Sem configuração fiscal a nota segue, sem essas conferências.
func (s *EntradaServico) ufEmpresa(ctx context.Context) (string, error) {
	cfg, err := s.configFiscal(ctx)
	if err != nil || cfg == nil {
		return "", err
	}
	return strings.ToUpper(strings.TrimSpace(cfg.UFEmpresa)), nil
}

// configFiscal: a configuração fiscal da empresa; nil quando não há.
func (s *EntradaServico) configFiscal(ctx context.Context) (*entity.FiscalConfig, error) {
	if s.Fiscal == nil {
		return nil, nil
	}
	cfg, err := s.Fiscal.GetFiscalConfig(ctx)
	var semConfig *errorsuc.NotFoundError
	if err != nil {
		if errors.As(err, &semConfig) {
			return nil, nil
		}
		return nil, err
	}
	return cfg, nil
}

// Preparar aplica operação, pedido, almoxarifado e CFOP de entrada aos itens.
// `manual` diz quais itens tiveram operação/almoxarifado/linha escolhidos à
// mão (não são sobrescritos pelos padrões).
func (s *EntradaServico) Preparar(ctx context.Context, e *entity.FiscalEntry) error {
	cfg, err := s.configFiscal(ctx)
	if err != nil {
		return err
	}
	uf, regime := "", engine.RegimeReal
	if cfg != nil {
		uf, regime = strings.ToUpper(strings.TrimSpace(cfg.UFEmpresa)), engine.RegimeDoCadastro(cfg.RegimeTributario)
	}

	// Linhas de pedido de compra candidatas: pelo pedido informado na nota,
	// pelo xPed dos itens e pelas linhas já escolhidas.
	var linhas []repository.PurchaseOrderLine
	if e.SupplierCode != nil {
		refs := []string{}
		if e.PurchaseOrderCode != nil {
			refs = append(refs, fmt.Sprint(*e.PurchaseOrderCode))
		}
		lineCodes := []int64{}
		for _, it := range e.Itens {
			if it.PedidoCompraXML != nil {
				refs = append(refs, *it.PedidoCompraXML)
			}
			if it.PurchaseOrderItemCode != nil {
				lineCodes = append(lineCodes, *it.PurchaseOrderItemCode)
			}
		}
		if linhas, err = s.Docs.PurchaseOrderLines(ctx, *e.SupplierCode, refs, lineCodes); err != nil {
			return err
		}
	}
	linhaPorCodigo := map[int64]repository.PurchaseOrderLine{}
	for _, l := range linhas {
		linhaPorCodigo[l.Code] = l
	}
	usadas := map[int64]bool{}
	for _, it := range e.Itens {
		if it.PurchaseOrderItemCode != nil {
			usadas[*it.PurchaseOrderItemCode] = true
		}
	}

	// Vínculo automático com o pedido: mesmo item, preferindo a linha que o
	// fornecedor citou (nItemPed) e a que ainda tem saldo a faturar.
	for _, it := range e.Itens {
		if it.PurchaseOrderItemCode != nil || it.ItemCode == nil || s.SemVinculoAutomatico[it.ID] {
			continue
		}
		var escolhida *repository.PurchaseOrderLine
		for i := range linhas {
			l := linhas[i]
			if l.ItemCode != *it.ItemCode || usadas[l.Code] || l.Status == "CANCELLED" ||
				poentity.SituacaoAceitaRecebimento(l.PurchaseOrderCode, poentity.PurchaseOrderStatus(l.OrderStatus)) != nil {
				continue
			}
			saldo := l.RequestedQty.Sub(l.InvoicedQty).Sub(l.CancelledQty)
			if !saldo.IsPositive() {
				continue
			}
			if it.PedidoCompraXML != nil && !refCasa(*it.PedidoCompraXML, l) && (e.PurchaseOrderCode == nil || *e.PurchaseOrderCode != l.PurchaseOrderCode) {
				continue
			}
			if escolhida == nil || (it.ItemPedidoXML != nil && fmt.Sprint(l.Sequence) == strings.TrimLeft(*it.ItemPedidoXML, "0")) {
				escolhida = &linhas[i]
			}
		}
		if escolhida != nil {
			code, po := escolhida.Code, escolhida.PurchaseOrderCode
			it.PurchaseOrderItemCode, it.PurchaseOrderCode = &code, &po
			usadas[code] = true
		}
	}

	// Operações (TES): do item, da nota ou da linha do pedido.
	codigosOp := []int64{}
	for _, it := range e.Itens {
		if op := operacaoDoItem(e, it, linhaPorCodigo); op != nil {
			codigosOp = append(codigosOp, *op)
		}
	}
	ops, err := s.Docs.EntryOperations(ctx, codigosOp)
	if err != nil {
		return err
	}

	// Padrões do cadastro do item (almoxarifado).
	codigos := []int64{}
	for _, it := range e.Itens {
		if it.ItemCode != nil {
			codigos = append(codigos, *it.ItemCode)
		}
	}
	cadastro, err := s.Docs.ItemsByCode(ctx, codigos)
	if err != nil {
		return err
	}

	for _, it := range e.Itens {
		natureza, finalidade := "", ""
		if code := operacaoDoItem(e, it, linhaPorCodigo); code != nil {
			op, ok := ops[*code]
			if !ok {
				return errorsuc.NewValidationError(fmt.Sprintf("item %d: o tipo de operação de entrada %d não existe", it.Sequence, *code))
			}
			if !op.IsActive {
				return errorsuc.NewValidationError(fmt.Sprintf("item %d: o tipo de operação de entrada %d (%s) está inativo", it.Sequence, op.Code, op.Description))
			}
			if ufF := strings.ToUpper(strings.TrimSpace(derefStr(e.UFEmitente))); op.StateGroupCode != nil && ufF != "" && !slices.Contains(op.GrupoUFs, ufF) {
				return errorsuc.NewValidationError(fmt.Sprintf("item %d: o tipo de operação de entrada %d (%s) vale só para fornecedor das UFs do grupo de estados %d (%s); o fornecedor é de %s",
					it.Sequence, op.Code, op.Description, *op.StateGroupCode, strings.Join(op.GrupoUFs, ", "), ufF))
			}
			c := op.Code
			it.EntryOperationCode = &c
			natureza = op.Natureza
			it.MovimentaEstoque = op.MovimentaEstoque
			it.GeraFinanceiro = op.GeraFinanceiro
			it.GeraCreditoICMS = op.CreditaICMS && it.ValorICMS > 0
			it.GeraCreditoIPI = op.CreditaIPI && it.ValorIPI > 0
			it.GeraCreditoPIS = op.CreditaPISCOFINS && it.ValorPIS > 0
			it.GeraCreditoCOFINS = op.CreditaPISCOFINS && it.ValorCOFINS > 0
		} else {
			// Sem operação: entra no estoque, gera financeiro e toma os
			// créditos destacados que a finalidade da compra admite.
			it.MovimentaEstoque, it.GeraFinanceiro = true, true
			it.GeraCreditoICMS = it.ValorICMS > 0
			it.GeraCreditoIPI = it.ValorIPI > 0
			it.GeraCreditoPIS = it.ValorPIS > 0
			it.GeraCreditoCOFINS = it.ValorCOFINS > 0
			finalidade = finalidadeDaCompra(it, linhaPorCodigo, cadastro)
			entrada.CreditosDaFinalidade(finalidade, it)
		}
		// O regime restringe: Simples não toma crédito; Presumido não toma
		// PIS/COFINS. A conferência explica o que ficou sem crédito.
		entrada.CreditosDoRegime(regime, it)
		ufForn := ""
		if e.UFEmitente != nil {
			ufForn = *e.UFEmitente
		}
		if cfop := entrada.CFOPEntradaComFinalidade(it.Cfop, natureza, finalidade, ufForn, uf); cfop != "" {
			it.CfopEntrada = &cfop
		}
		if it.PurchaseOrderItemCode != nil {
			if l, ok := linhaPorCodigo[*it.PurchaseOrderItemCode]; ok {
				if it.WarehouseID == nil && l.WarehouseID != nil {
					w := *l.WarehouseID
					it.WarehouseID = &w
				}
				if it.PlanoContasID == nil && l.AccountingAccount != nil {
					if p, err := s.Docs.PlanoContasByCodigo(ctx, *l.AccountingAccount); err != nil {
						return err
					} else if p != nil {
						it.PlanoContasID = p
					}
				}
				// Quantidade na unidade de estoque pelo fator do pedido, quando
				// o vínculo com o fornecedor não trouxe conversão.
				if (it.FatorConversao == nil || it.FatorConversao.Equal(umDec)) && l.InternalQty.IsPositive() {
					f := l.FatorEstoque()
					q := decimal.NewFromFloat(it.Quantity).Mul(f).Round(6)
					it.FatorConversao, it.QuantidadeEstoque = &f, &q
				}
			}
		}
		// Sem conta da linha nem do pedido, vale a conta padrão do centro de custo.
		if it.PlanoContasID == nil && it.CentroCustoID != nil {
			p, err := s.Docs.ContaPadraoDoCentro(ctx, *it.CentroCustoID)
			if err != nil {
				return err
			}
			it.PlanoContasID = p
		}
		if it.WarehouseID == nil && it.ItemCode != nil {
			if c, ok := cadastro[*it.ItemCode]; ok && c.DefaultWarehouseID != nil {
				w := *c.DefaultWarehouseID
				it.WarehouseID = &w
			}
		}
		if it.ItemCode != nil && it.QuantidadeEstoque == nil {
			aplicarConversao(it, nil, nil)
		}
	}
	if _, err := s.aplicarConversoesDoCadastro(ctx, e, cadastro); err != nil {
		return err
	}
	return nil
}

// unidadeDifere: a unidade da nota não é a de estoque do item.
func unidadeDifere(daNota *string, doEstoque string) bool {
	if daNota == nil || strings.TrimSpace(*daNota) == "" || strings.TrimSpace(doEstoque) == "" {
		return false
	}
	return !strings.EqualFold(strings.TrimSpace(*daNota), strings.TrimSpace(doEstoque))
}

// semFator: a linha não tem conversão própria (nula ou 1, que é o padrão
// gravado quando nada converteu).
func semFator(it *entity.FiscalEntryItem) bool {
	return it.FatorConversao == nil || it.FatorConversao.Equal(umDec)
}

// aplicarConversoesDoCadastro converte para a unidade de estoque as linhas que
// vieram em outra unidade e não têm fator — o do vínculo com o fornecedor ou o
// do pedido têm precedência.
//
// Sem isto o cadastro de conversões do item era ignorado na entrada: a nota de
// 115,8975 KG de um tubo estocado em metros entrava como 115,8975 METROS, ao
// preço do quilo. O estoque ficava 6,4 vezes maior e o custo do metro 6,4
// vezes menor, e nada avisava.
func (s *EntradaServico) aplicarConversoesDoCadastro(ctx context.Context, e *entity.FiscalEntry, cadastro map[int64]repository.ItemCandidate) (bool, error) {
	mudou := false
	for _, it := range e.Itens {
		if it.ItemCode == nil || !semFator(it) {
			continue
		}
		c, ok := cadastro[*it.ItemCode]
		if !ok || !unidadeDifere(it.UOM, c.UOM) {
			continue
		}
		f, err := s.Docs.FatorDeConversao(ctx, *it.ItemCode, *it.UOM, c.UOM)
		if err != nil {
			return false, err
		}
		if f == nil || !f.IsPositive() || f.Equal(umDec) {
			continue
		}
		q := decimal.NewFromFloat(it.Quantity).Mul(*f).Round(6)
		fator := *f
		it.FatorConversao, it.QuantidadeEstoque = &fator, &q
		mudou = true
	}
	return mudou, nil
}

// AplicarConversoes reaplica as conversões do cadastro numa nota já gravada
// (nota importada antes da correção, ou conversão cadastrada depois da
// importação). Devolve se alguma linha mudou.
func (s *EntradaServico) AplicarConversoes(ctx context.Context, e *entity.FiscalEntry) (bool, error) {
	cadastro, err := s.cadastroDosItens(ctx, e)
	if err != nil {
		return false, err
	}
	return s.aplicarConversoesDoCadastro(ctx, e, cadastro)
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

// pendenciasDeUnidade: linha que entra no estoque numa unidade diferente da de
// estoque, sem conversão, não pode ser aprovada — entraria com o número errado.
func (s *EntradaServico) pendenciasDeUnidade(ctx context.Context, e *entity.FiscalEntry) ([]entrada.Pendencia, error) {
	if s.Docs == nil {
		return nil, nil
	}
	cadastro, err := s.cadastroDosItens(ctx, e)
	if err != nil {
		return nil, err
	}
	var out []entrada.Pendencia
	for _, it := range e.Itens {
		if it.ItemCode == nil || !it.MovimentaEstoque || !semFator(it) {
			continue
		}
		c, ok := cadastro[*it.ItemCode]
		if !ok || !unidadeDifere(it.UOM, c.UOM) {
			continue
		}
		// Conversão cadastrada com fator 1 (PC → UN) é legítima: o que impede é
		// não haver conversão nenhuma.
		if f, err := s.Docs.FatorDeConversao(ctx, *it.ItemCode, *it.UOM, c.UOM); err != nil {
			return nil, err
		} else if f != nil {
			continue
		}
		codigo := c.BusinessCode
		if codigo == "" {
			codigo = fmt.Sprint(c.Code)
		}
		out = append(out, entrada.Pendencia{
			Nivel: entrada.NivelImpede,
			Campo: fmt.Sprintf("itens[%d].fator_conversao", it.Sequence),
			Mensagem: fmt.Sprintf("item %d (%s): a nota vem em %s e o item é estocado em %s, sem conversão — cadastre a conversão %s→%s do item (VSUP0110) ou informe o fator na conciliação",
				it.Sequence, codigo, strings.ToUpper(strings.TrimSpace(*it.UOM)), c.UOM, strings.ToUpper(strings.TrimSpace(*it.UOM)), c.UOM),
		})
	}
	return out, nil
}

func operacaoDoItem(e *entity.FiscalEntry, it *entity.FiscalEntryItem, linhas map[int64]repository.PurchaseOrderLine) *int64 {
	if it.EntryOperationCode != nil {
		return it.EntryOperationCode
	}
	if e.EntryOperationCode != nil {
		return e.EntryOperationCode
	}
	if it.PurchaseOrderItemCode != nil {
		if l, ok := linhas[*it.PurchaseOrderItemCode]; ok && l.OperationCode != nil {
			return l.OperationCode
		}
	}
	return nil
}

func refCasa(ref string, l repository.PurchaseOrderLine) bool {
	n := strings.TrimLeft(soDigitos(ref), "0")
	return n != "" && (n == fmt.Sprint(l.OrderNumber) || n == fmt.Sprint(l.PurchaseOrderCode))
}

// Distribuir refaz a distribuição das parcelas pelos alvos financeiros.
func Distribuir(e *entity.FiscalEntry) error {
	_, _, alvos, _ := entrada.PlanoFinanceiro(e)
	return entrada.DistribuirProporcional(e.Parcelas, alvos)
}

// Conferir junta as pendências da nota e as divergências que impedem.
func (s *EntradaServico) Conferir(ctx context.Context, e *entity.FiscalEntry) ([]entrada.Pendencia, []entrada.Divergencia, error) {
	pendForn, err := s.Fornecedor(ctx, e)
	if err != nil {
		return nil, nil, err
	}
	pend := append(entrada.ConferirEntrada(e), pendForn...)
	pendUnidade, err := s.pendenciasDeUnidade(ctx, e)
	if err != nil {
		return nil, nil, err
	}
	pend = append(pend, pendUnidade...)
	pendTrib, err := s.pendenciasTributarias(ctx, e)
	if err != nil {
		return nil, nil, err
	}
	pend = append(pend, pendTrib...)
	pendCentro, err := s.pendenciasDeCentro(ctx, e)
	if err != nil {
		return nil, nil, err
	}
	pend = append(pend, pendCentro...)
	divs, err := s.Divergencias(ctx, e)
	if err != nil {
		return nil, nil, err
	}
	for _, d := range divs {
		if d.Nivel == entrada.NivelImpede {
			pend = append(pend, entrada.Pendencia{Nivel: entrada.NivelImpede, Campo: fmt.Sprintf("itens[%d].divergencia", d.Item), Mensagem: d.Mensagem})
		}
	}
	return pend, divs, nil
}

// Fornecedor liga a nota ao fornecedor cadastrado depois da importação (o
// mesmo CNPJ do emitente) e confere a situação dele: fornecedor bloqueado ou
// inativo não recebe nota nova. Só mexe em nota ainda não aprovada.
func (s *EntradaServico) Fornecedor(ctx context.Context, e *entity.FiscalEntry) ([]entrada.Pendencia, error) {
	if s.Docs == nil || (e.Status != entity.EntryStatusPending && e.Status != entity.EntryStatusConferred) {
		return nil, nil
	}
	if e.SupplierCode == nil {
		f, err := s.Docs.FindSupplierByDocument(ctx, e.CnpjEmitente)
		if err != nil || f == nil {
			return nil, err
		}
		if _, err := s.Docs.LinkSupplierToPendingEntries(ctx, e.CnpjEmitente, f.Code); err != nil {
			return nil, err
		}
		code, nome := f.Code, f.Name
		e.SupplierCode, e.SupplierName = &code, &nome
	}
	f, err := s.Docs.SupplierByCode(ctx, *e.SupplierCode)
	if err != nil {
		return nil, err
	}
	switch {
	case f == nil:
		return []entrada.Pendencia{{Nivel: entrada.NivelImpede, Campo: "supplier_code",
			Mensagem: fmt.Sprintf("o fornecedor %d da nota não existe mais no cadastro desta empresa", *e.SupplierCode)}}, nil
	case f.Blocked:
		return []entrada.Pendencia{{Nivel: entrada.NivelImpede, Campo: "supplier_code",
			Mensagem: fmt.Sprintf("o fornecedor %d (%s) está bloqueado: desbloqueie-o em VSUP0500 para aprovar a nota", f.Code, f.Name)}}, nil
	case !f.IsActive:
		return []entrada.Pendencia{{Nivel: entrada.NivelImpede, Campo: "supplier_code",
			Mensagem: fmt.Sprintf("o fornecedor %d (%s) está inativo: reative-o em VSUP0500 para aprovar a nota", f.Code, f.Name)}}, nil
	}
	return nil, nil
}

// Status diz se a nota está conferida (sem impedimento) ou pendente.
func (s *EntradaServico) Status(ctx context.Context, e *entity.FiscalEntry) (entity.FiscalEntryStatus, error) {
	pend, _, err := s.Conferir(ctx, e)
	if err != nil {
		return "", err
	}
	if entrada.TemImpedimento(pend) {
		return entity.EntryStatusPending, nil
	}
	return entity.EntryStatusConferred, nil
}

// Divergencias confere a nota contra as tabelas fiscais, o cadastro e o
// pedido de compra. Preço e quantidade acima do pedido passam pela regra de
// tolerância de compras do fornecedor: BLOCK vira impedimento.
func (s *EntradaServico) Divergencias(ctx context.Context, e *entity.FiscalEntry) ([]entrada.Divergencia, error) {
	uf, err := s.ufEmpresa(ctx)
	if err != nil {
		return nil, err
	}
	p := entrada.ParametrosConferencia{UFEmpresa: uf, NCMDoItem: map[int64]string{}, LinhasPedido: map[int64]entrada.LinhaPedido{}}

	ncms := []string{}
	codigos := []int64{}
	linhas := []int64{}
	for _, it := range e.Itens {
		if it.Ncm != nil {
			ncms = append(ncms, soDigitos(*it.Ncm))
		}
		if it.ItemCode != nil {
			codigos = append(codigos, *it.ItemCode)
		}
		if it.PurchaseOrderItemCode != nil {
			linhas = append(linhas, *it.PurchaseOrderItemCode)
		}
	}
	if p.AliqIPIPorNCM, err = s.Docs.NCMIPIRates(ctx, ncms); err != nil {
		return nil, err
	}
	if s.Fiscal != nil {
		tabela, err := s.Fiscal.ListICMSInterstate(ctx)
		if err != nil {
			return nil, err
		}
		p.ICMSInterestadual = map[string]decimal.Decimal{}
		for k, v := range tabela {
			p.ICMSInterestadual[strings.ToUpper(k)] = decimal.NewFromFloat(v)
		}
	}
	cadastro, err := s.Docs.ItemsByCode(ctx, codigos)
	if err != nil {
		return nil, err
	}
	for code, c := range cadastro {
		p.NCMDoItem[code] = c.NCM
	}
	if len(linhas) > 0 && e.SupplierCode != nil {
		ls, err := s.Docs.PurchaseOrderLines(ctx, *e.SupplierCode, nil, linhas)
		if err != nil {
			return nil, err
		}
		for _, l := range ls {
			f := l.FatorEstoque()
			preco := l.UnitPrice
			if l.InternalPrice.IsPositive() {
				preco = l.InternalPrice
			} else if f.IsPositive() {
				preco = l.UnitPrice.Div(f)
			}
			saldo := l.RequestedQty.Sub(l.InvoicedQty).Sub(l.CancelledQty).Mul(f)
			p.LinhasPedido[l.Code] = entrada.LinhaPedido{
				Codigo: l.Code, PedidoCodigo: l.PurchaseOrderCode, ItemCode: l.ItemCode,
				PrecoUnitario: preco.Round(6), SaldoAFaturar: saldo, TolerancePct: l.TolerancePct,
				Cancelada:         l.Status == "CANCELLED",
				PedidoNaoAprovado: poentity.SituacaoAceitaRecebimento(l.PurchaseOrderCode, poentity.PurchaseOrderStatus(l.OrderStatus)) != nil,
				SituacaoPedido:    poentity.RotuloSituacao(poentity.PurchaseOrderStatus(l.OrderStatus)),
			}
		}
	}

	divs := entrada.Divergencias(e, p)
	if s.Tolerancias != nil {
		for i := range divs {
			kind := ""
			switch divs[i].Tipo {
			case "PEDIDO_PRECO":
				kind = "ITEM_PRICE"
			case "PEDIDO_QUANTIDADE":
				kind = "QUANTITY"
			default:
				continue
			}
			esperado, e1 := decimal.NewFromString(divs[i].Esperado)
			informado, e2 := decimal.NewFromString(divs[i].Informado)
			if e1 != nil || e2 != nil {
				continue
			}
			acao, msg, excedeu, err := s.Tolerancias.EvaluatePurchaseTolerance(ctx, e.SupplierCode, kind, "ENTRY_INVOICE", esperado, informado)
			if err != nil {
				return nil, err
			}
			if excedeu && acao == "BLOCK" {
				divs[i].Nivel = entrada.NivelImpede
				if msg != "" {
					divs[i].Mensagem += " — " + msg
				}
			}
		}
	}
	return divs, nil
}

// Responder monta a resposta completa da nota (com divergências).
func (s *EntradaServico) Responder(ctx context.Context, e *entity.FiscalEntry) (*response.FiscalEntryResponse, error) {
	if e == nil {
		return nil, nil
	}
	pendForn, err := s.Fornecedor(ctx, e)
	if err != nil {
		return nil, err
	}
	r := toFiscalEntryDocumentResponse(e)
	if e.Status == entity.EntryStatusPending || e.Status == entity.EntryStatusConferred {
		for _, p := range pendForn {
			r.Pendencias = append(r.Pendencias, response.FiscalEntryPendencia{Nivel: p.Nivel, Campo: p.Campo, Mensagem: p.Mensagem})
			if p.Nivel == entrada.NivelImpede {
				r.PodeAprovar = false
			}
		}
	}
	divs, err := s.Divergencias(ctx, e)
	if err != nil {
		return nil, err
	}
	for _, d := range divs {
		r.Divergencias = append(r.Divergencias, response.FiscalEntryDivergencia{
			Nivel: d.Nivel, Item: d.Item, Tipo: d.Tipo, Mensagem: d.Mensagem, Esperado: d.Esperado, Informado: d.Informado,
		})
		if d.Nivel == entrada.NivelImpede && (e.Status == entity.EntryStatusPending || e.Status == entity.EntryStatusConferred) {
			r.Pendencias = append(r.Pendencias, response.FiscalEntryPendencia{Nivel: d.Nivel, Campo: fmt.Sprintf("itens[%d].divergencia", d.Item), Mensagem: d.Mensagem})
			r.PodeAprovar = false
		}
	}
	return r, nil
}

// pendenciasTributarias explica a tributação da entrada que depende da
// empresa: os créditos que o regime não deixa aproveitar e o DIFAL que a
// empresa recolhe na compra interestadual para uso, consumo ou ativo.
func (s *EntradaServico) pendenciasTributarias(ctx context.Context, e *entity.FiscalEntry) ([]entrada.Pendencia, error) {
	cfg, err := s.configFiscal(ctx)
	if err != nil || cfg == nil {
		return nil, err
	}
	var out []entrada.Pendencia
	regime := engine.RegimeDoCadastro(cfg.RegimeTributario)
	destacado := false
	for _, it := range e.Itens {
		if it.ValorICMS > 0 || it.ValorIPI > 0 || it.ValorPIS > 0 || it.ValorCOFINS > 0 {
			destacado = true
		}
	}
	switch {
	case regime == engine.RegimeSimples && destacado:
		out = append(out, entrada.Pendencia{Nivel: entrada.NivelAtencao, Campo: "itens.creditos",
			Mensagem: "empresa do Simples Nacional: o ICMS, o IPI, o PIS e a COFINS destacados pelo fornecedor não geram crédito e entram no custo"})
	case regime == engine.RegimePresumido:
		for _, it := range e.Itens {
			if it.ValorPIS > 0 || it.ValorCOFINS > 0 {
				out = append(out, entrada.Pendencia{Nivel: entrada.NivelAtencao, Campo: "itens.creditos",
					Mensagem: "empresa do Lucro Presumido: PIS e COFINS são cumulativos e não geram crédito na entrada"})
				break
			}
		}
	}

	// DIFAL de uso, consumo e ativo (base dupla), pela alíquota interna da UF
	// da empresa na tabela de ICMS (ou a da configuração).
	ufEmpresa := strings.ToUpper(strings.TrimSpace(cfg.UFEmpresa))
	aliqInterna := decimal.NewFromFloat(cfg.IcmsInternoAliquota)
	internas, err := s.Fiscal.ListICMSInternal(ctx)
	if err != nil {
		return nil, err
	}
	if v, ok := internas[ufEmpresa]; ok && v.ICMS > 0 {
		aliqInterna = decimal.NewFromFloat(v.ICMS)
	}
	var inter map[string]float64
	for _, it := range e.Itens {
		if it.CfopEntrada == nil || !entrada.CFOPDeUsoConsumoInterestadual(*it.CfopEntrada) {
			continue
		}
		icmsOrigem := decimal.NewFromFloat(it.ValorICMS)
		if !icmsOrigem.IsPositive() && e.UFEmitente != nil {
			// Fornecedor sem ICMS destacado (Simples): vale a alíquota
			// interestadual entre as UFs.
			if inter == nil {
				if inter, err = s.Fiscal.ListICMSInterstate(ctx); err != nil {
					return nil, err
				}
			}
			if a, ok := inter[strings.ToUpper(strings.TrimSpace(*e.UFEmitente))+ufEmpresa]; ok && a > 0 {
				icmsOrigem = it.ValorContabil.Mul(decimal.NewFromFloat(a)).Round(2)
			}
		}
		base, difal := entrada.DifalUsoConsumo(it.ValorContabil, icmsOrigem, aliqInterna)
		if !difal.IsPositive() {
			continue
		}
		out = append(out, entrada.Pendencia{Nivel: entrada.NivelAtencao, Campo: fmt.Sprintf("itens[%d].difal", it.Sequence),
			Mensagem: fmt.Sprintf("linha %d (CFOP %s, uso/consumo/ativo de outra UF): DIFAL a recolher para %s de %s (base dupla %s, interna %s%%, ICMS da origem %s)",
				it.Sequence, *it.CfopEntrada, ufEmpresa, entrada.Moeda(difal), entrada.Moeda(base), aliqInterna.Mul(cemDec).Round(2).String(), entrada.Moeda(icmsOrigem))})
	}
	return out, nil
}

// finalidadeDaCompra: a da linha do pedido de compra; sem pedido, o tipo de uso
// do cadastro do item; sem item, nenhuma (vale a troca de dígito do CFOP).
func finalidadeDaCompra(it *entity.FiscalEntryItem, linhas map[int64]repository.PurchaseOrderLine, cadastro map[int64]repository.ItemCandidate) string {
	if it.PurchaseOrderItemCode != nil {
		if l, ok := linhas[*it.PurchaseOrderItemCode]; ok && l.UtilizationType != nil {
			switch f := strings.ToUpper(strings.TrimSpace(*l.UtilizationType)); f {
			case entrada.FinalidadeIndustrializacao, entrada.FinalidadeConsumo, entrada.FinalidadeImobilizado:
				return f
			}
		}
	}
	if it.ItemCode != nil {
		if c, ok := cadastro[*it.ItemCode]; ok {
			return entrada.FinalidadeDoTipoDeUso(c.TipoUso)
		}
	}
	return ""
}

// pendenciasDeCentro: a conta que exige centro de custo (VFIN0120) não aceita
// item nem rateio de parcela da nota sem centro — o título a pagar que a
// aprovação gera sairia sem ele.
func (s *EntradaServico) pendenciasDeCentro(ctx context.Context, e *entity.FiscalEntry) ([]entrada.Pendencia, error) {
	var ids []int64
	for _, it := range e.Itens {
		if it.GeraFinanceiro && it.PlanoContasID != nil {
			ids = append(ids, *it.PlanoContasID)
		}
	}
	for _, p := range e.Parcelas {
		for _, d := range p.Distribuicao {
			ids = append(ids, d.PlanoContasID)
		}
	}
	exige, err := s.Docs.ContasQueExigemCentro(ctx, ids)
	if err != nil || len(exige) == 0 {
		return nil, err
	}
	var out []entrada.Pendencia
	for _, it := range e.Itens {
		if it.GeraFinanceiro && it.PlanoContasID != nil && it.CentroCustoID == nil {
			if conta, ok := exige[*it.PlanoContasID]; ok {
				out = append(out, entrada.Pendencia{Nivel: entrada.NivelImpede, Campo: fmt.Sprintf("itens[%d].centro_custo_id", it.Sequence),
					Mensagem: fmt.Sprintf("o item %d está na conta %s, que exige centro de custo: informe o centro", it.Sequence, conta)})
			}
		}
	}
	for i, p := range e.Parcelas {
		for _, d := range p.Distribuicao {
			if conta, ok := exige[d.PlanoContasID]; ok && d.CentroCustoID == nil {
				out = append(out, entrada.Pendencia{Nivel: entrada.NivelImpede, Campo: fmt.Sprintf("parcelas[%d].distribuicao", i+1),
					Mensagem: fmt.Sprintf("a parcela %d tem rateio na conta %s, que exige centro de custo: informe o centro", i+1, conta)})
			}
		}
	}
	return out, nil
}
