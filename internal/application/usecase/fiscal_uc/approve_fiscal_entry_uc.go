package fiscal_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	financialEntity "github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	financialrepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entrada"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

// InspecaoDeRecebimento é o roteiro de inspeção do recebimento (FINS0212):
// material com roteiro ativo entra no almoxarifado de inspeção e abre ordem
// de inspeção, em vez de ir direto para o estoque disponível.
type InspecaoDeRecebimento interface {
	ResolveInspectionRoute(ctx context.Context, itemCode int64, mask string) (inspectionWarehouseID int64, matched bool)
	OpenInspectionOrderFromReceipt(ctx context.Context, itemCode int64, mask string, quantity float64, inspectionWarehouseID int64, supplierCode, purchaseOrderCode, purchaseOrderItemCode *int64, lot *string) (int64, int64, error)
}

// ApproveFiscalEntryUseCase efetiva a nota de entrada. Conferida (itens
// conciliados e classificados, parcelas fechando, nenhuma divergência que
// bloqueie), gera numa transação só:
//
//   - um título do contas a pagar por parcela, pelo líquido, com o rateio por
//     plano de contas, e um título por imposto retido (a recolher);
//   - a entrada no estoque pelo custo de aquisição — só do que não teve
//     recebimento físico pelo pedido de compra (3-way) — no almoxarifado de
//     inspeção quando o item tem roteiro;
//   - o faturado/recebido do pedido de compra;
//   - a contabilização (quando configurada).
type ApproveFiscalEntryUseCase struct {
	FiscalRepo    repository.FiscalRepository
	Docs          repository.FiscalEntryDocumentRepository
	FinancialRepo financialrepo.FinancialRepository
	Tolerancias   ports.PurchaseToleranceEvaluator
	Inspecao      InspecaoDeRecebimento
	Auth          ports.AuthService
}

func (uc *ApproveFiscalEntryUseCase) servico() *EntradaServico {
	return &EntradaServico{Docs: uc.Docs, Fiscal: uc.FiscalRepo, Tolerancias: uc.Tolerancias}
}

func (uc *ApproveFiscalEntryUseCase) Execute(ctx context.Context, dto request.ApproveFiscalEntryDTO) (*response.FiscalEntryResponse, error) {
	if !uc.Auth.CanApproveFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if uc.Docs == nil {
		return nil, fmt.Errorf("repositório da nota de entrada não configurado")
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	servico := uc.servico()

	doc, err := uc.Docs.GetEntryDocument(ctx, dto.ID)
	if err != nil {
		return nil, err
	}
	if doc.Status != entity.EntryStatusPending && doc.Status != entity.EntryStatusConferred {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a nota está %s: só nota pendente ou conferida pode ser aprovada", doc.Status))
	}

	// Nota antiga, de antes das parcelas: nasce com a parcela única de 30 dias
	// que a aprovação sempre gerou, distribuída pelos planos dos itens.
	_, aPagar, _, _ := entrada.PlanoFinanceiro(doc)
	if len(doc.Parcelas) == 0 && !doc.SemPagamento && aPagar.IsPositive() {
		doc.Parcelas = []*entity.FiscalEntryInstallment{{
			Numero:         1,
			DataVencimento: doc.DataEmissao.AddDate(0, 0, 30),
			Valor:          aPagar,
			Origem:         entity.ParcelaOrigemPadrao,
		}}
		if err := Distribuir(doc); err != nil {
			return nil, errorsuc.NewValidationError(err.Error())
		}
		cab := repository.HeaderConciliation{EntryOperationCode: doc.EntryOperationCode, PurchaseOrderCode: doc.PurchaseOrderCode}
		if err := uc.Docs.SaveConciliation(ctx, doc.ID, cab, conciliacoesDoDoc(doc), nil, doc.Parcelas, doc.Status); err != nil {
			return nil, err
		}
		if doc, err = uc.Docs.GetEntryDocument(ctx, dto.ID); err != nil {
			return nil, err
		}
	}

	pend, _, err := servico.Conferir(ctx, doc)
	if err != nil {
		return nil, err
	}
	if entrada.TemImpedimento(pend) {
		var msgs []string
		for _, p := range pend {
			if p.Nivel == entrada.NivelImpede {
				msgs = append(msgs, p.Mensagem)
			}
		}
		return nil, errorsuc.NewValidationError("a nota ainda não pode ser aprovada: " + strings.Join(msgs, "; "))
	}

	movimentos, custos, err := uc.montarMovimentos(ctx, doc)
	if err != nil {
		return nil, err
	}
	lancamentos, err := uc.montarContabilizacao(ctx, doc, custos)
	if err != nil {
		return nil, err
	}
	titulos := montarTitulos(doc)
	titulosRet, err := montarTitulosRetencao(doc)
	if err != nil {
		return nil, err
	}
	titulos = append(titulos, titulosRet...)

	res, err := uc.Docs.AprovarEntrada(ctx, repository.AprovacaoEntrada{
		EntryID: doc.ID, UserID: userID, Titulos: titulos, Movimentos: movimentos, Lancamentos: lancamentos, Custos: custos,
	})
	if err != nil {
		return nil, err
	}

	var avisos []string
	// Ordens de inspeção para o que entrou no almoxarifado de inspeção.
	if uc.Inspecao != nil {
		inspecao := map[int64]int64{}
		for _, m := range movimentos {
			if wh, ok := uc.inspecaoDoItem(ctx, m.ItemCode); ok && wh == m.WarehouseID {
				inspecao[m.ItemID] = wh
			}
		}
		for _, mg := range res.Movimentados {
			if wh, ok := inspecao[mg.ItemID]; ok {
				if _, numero, err := uc.Inspecao.OpenInspectionOrderFromReceipt(ctx, mg.ItemCode, "", mg.Quantidade.InexactFloat64(), wh,
					doc.SupplierCode, mg.PurchaseOrderCode, mg.PurchaseOrderItemCode, nil); err != nil {
					avisos = append(avisos, fmt.Sprintf("o item %d entrou no almoxarifado de inspeção, mas a ordem de inspeção não foi aberta: %v", mg.ItemCode, err))
				} else {
					avisos = append(avisos, fmt.Sprintf("item %d aguardando inspeção (ordem %d)", mg.ItemCode, numero))
				}
			}
		}
	}
	if res.QtdJaRecebida.IsPositive() {
		avisos = append(avisos, "parte da nota já tinha entrado no estoque pelo recebimento do pedido de compra e não entrou de novo")
	}

	// Créditos de imposto da competência da entrada. Vêm depois da aprovação
	// (estão em outro agregado); uma falha aqui não desfaz a nota aprovada, mas
	// é devolvida como aviso para alguém lançar o crédito.
	avisos = append(avisos, uc.lancarCreditos(ctx, doc, decimal.NewFromInt(1))...)

	aprovado, err := uc.Docs.GetEntryDocument(ctx, doc.ID)
	if err != nil {
		return nil, err
	}
	aprovado.Warnings = avisos
	return servico.Responder(ctx, aprovado)
}

func (uc *ApproveFiscalEntryUseCase) inspecaoDoItem(ctx context.Context, itemCode int64) (int64, bool) {
	if uc.Inspecao == nil {
		return 0, false
	}
	wh, ok := uc.Inspecao.ResolveInspectionRoute(ctx, itemCode, "")
	return wh, ok && wh > 0
}

// montarMovimentos diz o que cada item faz no estoque e no pedido de compra.
func (uc *ApproveFiscalEntryUseCase) montarMovimentos(ctx context.Context, doc *entity.FiscalEntry) ([]repository.MovimentoEntrada, map[int64]decimal.Decimal, error) {
	fatores := map[int64]decimal.Decimal{}
	linhas := []int64{}
	for _, it := range doc.Itens {
		if it.PurchaseOrderItemCode != nil {
			linhas = append(linhas, *it.PurchaseOrderItemCode)
		}
	}
	if len(linhas) > 0 && doc.SupplierCode != nil {
		ls, err := uc.Docs.PurchaseOrderLines(ctx, *doc.SupplierCode, nil, linhas)
		if err != nil {
			return nil, nil, err
		}
		for _, l := range ls {
			fatores[l.Code] = l.FatorEstoque()
		}
	}
	custos := map[int64]decimal.Decimal{}
	var out []repository.MovimentoEntrada
	for _, it := range doc.Itens {
		custo := entrada.CustoAquisicao(it)
		custos[it.ID] = custo
		if it.ItemCode == nil {
			continue
		}
		qtd := decimal.NewFromFloat(it.Quantity)
		if it.QuantidadeEstoque != nil {
			qtd = *it.QuantidadeEstoque
		}
		m := repository.MovimentoEntrada{
			ItemID: it.ID, ItemCode: *it.ItemCode, Quantidade: qtd, CustoTotal: custo,
			MovimentaEstoque: it.MovimentaEstoque, PurchaseOrderItemCode: it.PurchaseOrderItemCode,
			FatorPedido: decimal.NewFromInt(1),
		}
		if it.PurchaseOrderItemCode != nil {
			if f, ok := fatores[*it.PurchaseOrderItemCode]; ok {
				m.FatorPedido = f
			}
		}
		if it.WarehouseID != nil {
			m.WarehouseID = *it.WarehouseID
		}
		if it.MovimentaEstoque {
			if wh, ok := uc.inspecaoDoItem(ctx, *it.ItemCode); ok {
				m.WarehouseID = wh
			}
		}
		if !m.MovimentaEstoque && m.PurchaseOrderItemCode == nil {
			continue
		}
		out = append(out, m)
	}
	return out, custos, nil
}

// montarContabilizacao gera as partidas da nota: débito no custo/despesa
// (conta contábil do plano de contas do item) e nos impostos a recuperar,
// crédito em fornecedores; e a transferência de fornecedores para as
// retenções a recolher. Sem parâmetros contábeis, a nota não é contabilizada.
func (uc *ApproveFiscalEntryUseCase) montarContabilizacao(ctx context.Context, doc *entity.FiscalEntry, custos map[int64]decimal.Decimal) ([]repository.LancamentoContabil, error) {
	p, err := uc.Docs.AccountingParams(ctx)
	if err != nil {
		return nil, err
	}
	if p == nil || !p.ContabilizarEntrada {
		return nil, nil
	}
	planos := []int64{}
	for _, it := range doc.Itens {
		if it.PlanoContasID != nil {
			planos = append(planos, *it.PlanoContasID)
		}
	}
	contaDoPlano, err := uc.Docs.PlanoContasAccounts(ctx, planos)
	if err != nil {
		return nil, err
	}
	fornecedor := doc.RazaoSocialEmitente
	hist := func(s string) string {
		return fmt.Sprintf("NF %d/%s %s — %s", doc.NumeroNF, doc.Serie, fornecedor, s)
	}

	var out []repository.LancamentoContabil
	var faltando []string
	add := func(debito int64, credito int64, cc *int64, valor decimal.Decimal, h string) {
		if valor.IsPositive() {
			out = append(out, repository.LancamentoContabil{PlanID: p.PlanID, DebitoID: debito, CreditoID: credito, DebitoCC: cc, Valor: valor.Round(2), Historico: h})
		}
	}
	for _, it := range doc.Itens {
		if !it.GeraFinanceiro {
			continue
		}
		conta := int64(0)
		if it.PlanoContasID != nil {
			conta = contaDoPlano[*it.PlanoContasID]
		}
		if conta == 0 && p.DespesaPadraoAccountID != nil {
			conta = *p.DespesaPadraoAccountID
		}
		if conta == 0 {
			faltando = append(faltando, fmt.Sprintf("o plano de contas do item %d não tem conta contábil (e não há conta de despesa padrão)", it.Sequence))
			continue
		}
		custo := custos[it.ID]
		// Crédito sem conta de "a recuperar" configurada vira custo: não some.
		recuperar := func(gera bool, valor decimal.Decimal, contaRec *int64, nome string) {
			if !gera || !valor.IsPositive() {
				return
			}
			if contaRec == nil {
				custo = custo.Add(valor)
				return
			}
			add(*contaRec, p.FornecedoresAccountID, nil, valor, hist(nome+" a recuperar"))
		}
		recuperar(it.GeraCreditoICMS, decimal.NewFromFloat(it.ValorICMS), p.ICMSRecuperarAccountID, "ICMS")
		recuperar(it.GeraCreditoIPI, decimal.NewFromFloat(it.ValorIPI), p.IPIRecuperarAccountID, "IPI")
		recuperar(it.GeraCreditoPIS, decimal.NewFromFloat(it.ValorPIS), p.PISRecuperarAccountID, "PIS")
		recuperar(it.GeraCreditoCOFINS, decimal.NewFromFloat(it.ValorCOFINS), p.COFINSRecuperarAccountID, "COFINS")
		recuperar(it.GeraCreditoIBSCBS, it.ValorIBS, p.IBSRecuperarAccountID, "IBS")
		recuperar(it.GeraCreditoIBSCBS, it.ValorCBS, p.CBSRecuperarAccountID, "CBS")
		add(conta, p.FornecedoresAccountID, it.CentroCustoID, custo, hist(fmt.Sprintf("item %d %s", it.Sequence, deref(it.Description))))
	}
	contaRet := map[string]*int64{
		"IRRF": p.IRRFRecolherAccountID, "PIS": p.PCCRecolherAccountID, "COFINS": p.PCCRecolherAccountID,
		"CSLL": p.PCCRecolherAccountID, "INSS": p.INSSRecolherAccountID, "ISS": p.ISSRecolherAccountID,
	}
	for _, r := range entrada.Retencoes(doc) {
		c := contaRet[r.Tipo]
		if c == nil {
			faltando = append(faltando, fmt.Sprintf("falta a conta contábil de %s retido a recolher", r.Tipo))
			continue
		}
		add(p.FornecedoresAccountID, *c, nil, r.Valor, hist(r.Descricao))
	}
	if len(faltando) > 0 {
		return nil, errorsuc.NewValidationError("a contabilização automática está ligada, mas " + strings.Join(faltando, "; ") +
			". Ajuste os parâmetros contábeis ou o vínculo do plano de contas com a conta contábil")
	}
	ids := map[int64]bool{}
	for _, l := range out {
		ids[l.DebitoID], ids[l.CreditoID] = true, true
	}
	lista := make([]int64, 0, len(ids))
	for id := range ids {
		lista = append(lista, id)
	}
	validas, err := uc.Docs.ValidAccountingAccounts(ctx, p.PlanID, lista)
	if err != nil {
		return nil, err
	}
	for _, id := range lista {
		if !validas[id] {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("a conta contábil %d não é analítica do plano contábil configurado: lançamento em conta sintética desequilibra o balancete", id))
		}
	}
	return out, nil
}

// lancarCreditos registra os créditos de imposto na apuração; sinal -1 estorna.
func (uc *ApproveFiscalEntryUseCase) lancarCreditos(ctx context.Context, doc *entity.FiscalEntry, sinal decimal.Decimal) []string {
	return lancarCreditosDaNota(ctx, uc.FinancialRepo, doc, sinal)
}

func lancarCreditosDaNota(ctx context.Context, fin financialrepo.FinancialRepository, doc *entity.FiscalEntry, sinal decimal.Decimal) []string {
	if fin == nil {
		return nil
	}
	var avisos []string
	competencia := doc.DataEntrada.Format("01/2006")
	creditos := map[string]decimal.Decimal{}
	for _, item := range doc.Itens {
		if item.GeraCreditoICMS {
			creditos["ICMS"] = creditos["ICMS"].Add(decimal.NewFromFloat(item.ValorICMS))
		}
		if item.GeraCreditoIPI {
			creditos["IPI"] = creditos["IPI"].Add(decimal.NewFromFloat(item.ValorIPI))
		}
		if item.GeraCreditoPIS {
			creditos["PIS"] = creditos["PIS"].Add(decimal.NewFromFloat(item.ValorPIS))
		}
		if item.GeraCreditoCOFINS {
			creditos["COFINS"] = creditos["COFINS"].Add(decimal.NewFromFloat(item.ValorCOFINS))
		}
		if item.GeraCreditoIBSCBS {
			creditos["IBS"] = creditos["IBS"].Add(item.ValorIBS)
			creditos["CBS"] = creditos["CBS"].Add(item.ValorCBS)
		}
	}
	for _, imposto := range []string{"ICMS", "IPI", "PIS", "COFINS", "IBS", "CBS"} {
		valor := creditos[imposto]
		if !valor.IsPositive() {
			continue
		}
		ta := &financialEntity.TaxAssessment{
			Imposto:     imposto,
			Competencia: competencia,
			Creditos:    valor.Mul(sinal),
			Debitos:     decimal.Zero,
			Status:      financialEntity.TaxStatusApurar,
		}
		if err := fin.UpsertTaxAssessmentCredito(ctx, ta); err != nil {
			avisos = append(avisos, fmt.Sprintf("o crédito de %s (%s) não foi registrado na apuração de %s: %v", imposto, entrada.Moeda(valor.Mul(sinal)), competencia, err))
		}
	}
	return avisos
}

// conciliacoesDoDoc devolve o estado atual dos itens para regravação.
func conciliacoesDoDoc(doc *entity.FiscalEntry) []repository.ItemConciliation {
	out := make([]repository.ItemConciliation, 0, len(doc.Itens))
	for _, it := range doc.Itens {
		out = append(out, repository.ItemConciliation{
			ItemID: it.ID, ItemCode: it.ItemCode, ItemSupplierID: it.ItemSupplierID, ResolutionStrategy: it.ResolutionStrategy,
			PlanoContasID: it.PlanoContasID, CentroCustoID: it.CentroCustoID,
			FatorConversao: it.FatorConversao, QuantidadeEstoque: it.QuantidadeEstoque,
			CfopEntrada: it.CfopEntrada, EntryOperationCode: it.EntryOperationCode,
			MovimentaEstoque: it.MovimentaEstoque, GeraFinanceiro: it.GeraFinanceiro, WarehouseID: it.WarehouseID,
			PurchaseOrderCode: it.PurchaseOrderCode, PurchaseOrderItemCode: it.PurchaseOrderItemCode,
			GeraCreditoICMS: it.GeraCreditoICMS, GeraCreditoIPI: it.GeraCreditoIPI, GeraCreditoPIS: it.GeraCreditoPIS,
			GeraCreditoCOFINS: it.GeraCreditoCOFINS, GeraCreditoIBSCBS: it.GeraCreditoIBSCBS,
		})
	}
	return out
}

// montarTitulos gera um título por parcela. O título leva o plano de contas
// quando a parcela vai inteira para um só; com mais de um, o plano fica no
// rateio (e o filtro por plano de contas do contas a pagar olha o rateio).
func montarTitulos(doc *entity.FiscalEntry) []repository.TituloAPagar {
	total := len(doc.Parcelas)
	fornecedor := doc.RazaoSocialEmitente
	if doc.SupplierName != nil && *doc.SupplierName != "" {
		fornecedor = *doc.SupplierName
	}
	out := make([]repository.TituloAPagar, 0, total)
	for _, p := range doc.Parcelas {
		numDoc := fmt.Sprintf("NF-%d/%s", doc.NumeroNF, doc.Serie)
		if total > 1 {
			numDoc = fmt.Sprintf("NF-%d/%s %d/%d", doc.NumeroNF, doc.Serie, p.Numero, total)
		}
		obs := fmt.Sprintf("Nota de entrada %d (%s)", doc.NumeroNF, fornecedor)
		if p.Documento != nil && *p.Documento != "" {
			obs += " — duplicata " + *p.Documento
		}
		t := repository.TituloAPagar{
			InstallmentID:   p.ID,
			TipoDocumento:   "NFE",
			NumeroDocumento: numDoc,
			FornecedorCode:  doc.SupplierCode,
			FornecedorCNPJ:  doc.CnpjEmitente,
			DataEmissao:     doc.DataEmissao,
			DataVencimento:  p.DataVencimento,
			Valor:           p.Valor,
			ParcelaNumero:   p.Numero,
			ParcelaTotal:    total,
			FormaPagamento:  p.FormaPagamento,
			Observacao:      &obs,
			Rateio:          p.Distribuicao,
		}
		t.PlanoContasID, t.CentroCustoID = planoUnico(p.Distribuicao)
		out = append(out, t)
	}
	return out
}

// montarTitulosRetencao gera um título por imposto retido, com o rateio pela
// parte de cada plano de contas que ficou fora das parcelas do fornecedor —
// assim o contas a pagar por plano de contas fecha com o total da nota.
func montarTitulosRetencao(doc *entity.FiscalEntry) ([]repository.TituloAPagar, error) {
	rets := entrada.Retencoes(doc)
	if len(rets) == 0 {
		return nil, nil
	}
	_, _, _, residuo := entrada.PlanoFinanceiro(doc)
	parcelas := make([]*entity.FiscalEntryInstallment, 0, len(rets))
	for i, r := range rets {
		parcelas = append(parcelas, &entity.FiscalEntryInstallment{Numero: i + 1, Valor: r.Valor})
	}
	if err := entrada.DistribuirProporcional(parcelas, residuo); err != nil {
		return nil, errorsuc.NewValidationError(err.Error())
	}
	out := make([]repository.TituloAPagar, 0, len(rets))
	for i, r := range rets {
		obs := fmt.Sprintf("%s na NF %d/%s de %s (CNPJ %s): recolher em guia própria", r.Descricao, doc.NumeroNF, doc.Serie, doc.RazaoSocialEmitente, doc.CnpjEmitente)
		t := repository.TituloAPagar{
			TipoDocumento:   "RETENCAO",
			RetencaoTipo:    r.Tipo,
			NumeroDocumento: fmt.Sprintf("RET %s NF-%d/%s", r.Tipo, doc.NumeroNF, doc.Serie),
			DataEmissao:     doc.DataEmissao,
			DataVencimento:  r.Vencimento,
			Valor:           r.Valor,
			ParcelaNumero:   1,
			ParcelaTotal:    1,
			Observacao:      &obs,
			Rateio:          parcelas[i].Distribuicao,
		}
		t.PlanoContasID, t.CentroCustoID = planoUnico(t.Rateio)
		out = append(out, t)
	}
	return out, nil
}

func planoUnico(dist []entity.InstallmentAllocation) (*int64, *int64) {
	contas := map[entity.ChaveConta]bool{}
	for _, a := range dist {
		if a.Valor.IsPositive() {
			contas[a.Chave()] = true
		}
	}
	if len(contas) != 1 {
		return nil, nil
	}
	for k := range contas {
		plano := k.PlanoContasID
		return &plano, k.CentroCusto()
	}
	return nil, nil
}
