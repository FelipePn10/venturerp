package fiscal_uc

import (
	"context"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/accounting/contabilizacao"
	financialEntity "github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	financialrepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/devolucao"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

// SourceDevolucaoCompra marca a NF-e de devolução de compra.
const SourceDevolucaoCompra = "DEVOLUCAO_COMPRA"

// EstoqueDaSaida devolve ao estoque o que uma NF-e de saída cancelada baixou.
type EstoqueDaSaida interface {
	EstornarEstoqueDaSaida(ctx context.Context, exitID int64, userID uuid.UUID) (int, error)
}

// DevolucaoCompraUseCase é a devolução de compra ao fornecedor: a NF-e de
// saída (finalidade 4) referenciando a nota de entrada, com itens e impostos
// proporcionais ao que é devolvido. Na autorização, a mercadoria sai do
// estoque, o valor abate os títulos em aberto da nota (o que sobrar vira
// crédito a receber do fornecedor), os créditos de imposto são estornados e a
// contabilização é feita. O cancelamento desfaz tudo.
type DevolucaoCompraUseCase struct {
	Repo          repository.FiscalRepository
	Docs          repository.FiscalEntryDocumentRepository
	Devolucoes    repository.DevolucaoRepository
	Estoque       EstoqueDaSaida
	FinancialRepo financialrepo.FinancialRepository
	Auth          ports.AuthService
}

type DevolucaoItemDTO struct {
	FiscalEntryItemID int64           `json:"fiscal_entry_item_id"`
	Quantidade        decimal.Decimal `json:"quantidade"`
}

type DevolucaoCompraDTO struct {
	FiscalEntryID    int64              `json:"fiscal_entry_id"`
	DataEmissao      string             `json:"data_emissao"`
	Serie            string             `json:"serie"`
	NaturezaOperacao string             `json:"natureza_operacao"`
	Itens            []DevolucaoItemDTO `json:"itens"`
	// Endereço do fornecedor. Vazio, vem do XML da nota de entrada.
	DestLogradouro      *string `json:"dest_logradouro,omitempty"`
	DestNumero          *string `json:"dest_numero,omitempty"`
	DestComplemento     *string `json:"dest_complemento,omitempty"`
	DestBairro          *string `json:"dest_bairro,omitempty"`
	DestMunicipio       *string `json:"dest_municipio,omitempty"`
	DestCodigoMunicipio *string `json:"dest_codigo_municipio,omitempty"`
	DestCEP             *string `json:"dest_cep,omitempty"`
}

func f2d(v float64) decimal.Decimal { return decimal.NewFromFloat(v) }

func origemDoItem(it *entity.FiscalEntryItem, devolvida decimal.Decimal) devolucao.Origem {
	o := devolucao.Origem{
		ItemID: it.ID, Descricao: firstNonEmpty(deref(it.Description), fmt.Sprintf("item %d", it.Sequence)),
		Quantidade: f2d(it.Quantity), Devolvida: devolvida, ValorUnitario: f2d(it.UnitPrice), Total: f2d(it.TotalPrice),
		Frete: it.ValorFrete, Seguro: it.ValorSeguro, Desconto: it.ValorDesconto, Outras: it.ValorOutras,
		BaseICMS: f2d(it.BaseICMS), ICMS: f2d(it.ValorICMS), BaseIPI: f2d(it.BaseIPI), IPI: f2d(it.ValorIPI),
		PIS: f2d(it.ValorPIS), COFINS: f2d(it.ValorCOFINS), BaseST: it.BaseICMSST, ST: it.ValorICMSST, CustoAquisicao: it.CustoAquisicao,
	}
	if it.GeraCreditoICMS {
		o.CreditoICMS = o.ICMS
	}
	if it.GeraCreditoIPI {
		o.CreditoIPI = o.IPI
	}
	if it.GeraCreditoPIS {
		o.CreditoPIS = o.PIS
	}
	if it.GeraCreditoCOFINS {
		o.CreditoCOFI = o.COFINS
	}
	return o
}

// Previa mostra o que pode ser devolvido da nota.
func (uc *DevolucaoCompraUseCase) Previa(ctx context.Context, entryID int64) (*response.DevolucaoPreviaResponse, error) {
	if !uc.Auth.CanGetFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	doc, err := uc.Docs.GetEntryDocument(ctx, entryID)
	if err != nil {
		return nil, err
	}
	devolvidas, err := uc.Devolucoes.QuantidadesDevolvidas(ctx, entryID)
	if err != nil {
		return nil, err
	}
	mesmaUF, exterior := uc.ufs(ctx, doc)
	r := &response.DevolucaoPreviaResponse{FiscalEntryID: doc.ID, NumeroNF: doc.NumeroNF, Serie: doc.Serie, Fornecedor: doc.RazaoSocialEmitente,
		Status: string(doc.Status), ChaveAcesso: doc.ChaveAcesso}
	end := uc.enderecoDoFornecedor(ctx, doc.ID)
	r.Endereco = response.DevolucaoEndereco{Logradouro: end.Logradouro, Numero: end.Numero, Complemento: end.Complemento, Bairro: end.Bairro,
		Municipio: end.Municipio, CodigoMunicipio: end.CodigoMunicipio, CEP: end.CEP, Faltando: end.faltando()}
	for _, it := range doc.Itens {
		o := origemDoItem(it, devolvidas[it.ID])
		r.Itens = append(r.Itens, response.DevolucaoItemPrevia{
			FiscalEntryItemID: it.ID, Sequence: it.Sequence, ItemCode: it.ItemCode, Descricao: o.Descricao, UOM: it.UOM,
			Quantidade: o.Quantidade.InexactFloat64(), Devolvida: o.Devolvida.InexactFloat64(), Disponivel: o.Disponivel().InexactFloat64(),
			ValorUnitario: it.UnitPrice, CFOPDevolucao: devolucao.CFOP(deref(it.CfopEntrada), mesmaUF, exterior),
		})
	}
	return r, nil
}

func (uc *DevolucaoCompraUseCase) ufs(ctx context.Context, doc *entity.FiscalEntry) (mesmaUF, exterior bool) {
	ufEmp := ""
	if cfg, err := uc.Repo.GetFiscalConfig(ctx); err == nil && cfg != nil {
		ufEmp = strings.ToUpper(strings.TrimSpace(cfg.UFEmpresa))
	}
	ufForn := strings.ToUpper(strings.TrimSpace(deref(doc.UFEmitente)))
	return ufForn != "" && ufForn == ufEmp, ufForn == "EX"
}

// Criar gera a NF-e de devolução em rascunho (autorize como qualquer NF-e de saída).
func (uc *DevolucaoCompraUseCase) Criar(ctx context.Context, dto DevolucaoCompraDTO) (*response.FiscalExitResponse, error) {
	if !uc.Auth.CanCreateFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if len(dto.Itens) == 0 {
		return nil, errorsuc.NewValidationError("informe os itens e as quantidades devolvidas")
	}
	emissao, err := time.Parse("2006-01-02", strings.TrimSpace(dto.DataEmissao))
	if err != nil {
		return nil, errorsuc.NewValidationError("data de emissão inválida: use AAAA-MM-DD")
	}
	doc, err := uc.Docs.GetEntryDocument(ctx, dto.FiscalEntryID)
	if err != nil {
		return nil, err
	}
	if doc.Status != entity.EntryStatusApproved {
		return nil, errorsuc.NewValidationError("só nota de entrada aprovada pode ser devolvida")
	}
	if doc.ChaveAcesso == nil || len(soDigitos(*doc.ChaveAcesso)) != 44 {
		return nil, errorsuc.NewValidationError("a nota de entrada não tem chave de acesso: a NF-e de devolução precisa referenciar a NF-e de origem")
	}
	devolvidas, err := uc.Devolucoes.QuantidadesDevolvidas(ctx, doc.ID)
	if err != nil {
		return nil, err
	}
	porID := map[int64]*entity.FiscalEntryItem{}
	for _, it := range doc.Itens {
		porID[it.ID] = it
	}
	mesmaUF, exterior := uc.ufs(ctx, doc)
	emitenteSimples := false
	if cfg, err := uc.Repo.GetFiscalConfig(ctx); err == nil && cfg != nil {
		emitenteSimples = regimeSimples(cfg)
	}
	var linhas []devolucao.Linha
	vistos := map[int64]bool{}
	for _, d := range dto.Itens {
		it := porID[d.FiscalEntryItemID]
		if it == nil {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("o item %d não é desta nota", d.FiscalEntryItemID))
		}
		if vistos[it.ID] {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("o item %d foi informado duas vezes", it.Sequence))
		}
		vistos[it.ID] = true
		l, err := devolucao.Montar(origemDoItem(it, devolvidas[it.ID]), d.Quantidade)
		if err != nil {
			return nil, errorsuc.NewValidationError(err.Error())
		}
		linhas = append(linhas, l)
	}

	// Destinatário: o fornecedor, com o endereço do XML da nota (o endereço
	// fiscal que ele declarou) — ou o informado na tela.
	exit := &entity.FiscalExit{
		Serie: firstNonEmpty(strings.TrimSpace(dto.Serie), "1"), DataEmissao: emissao,
		CnpjDestinatario: &doc.CnpjEmitente, RazaoSocialDestinatario: &doc.RazaoSocialEmitente, IEDestinatario: doc.IEEmitente, UFDestinatario: doc.UFEmitente,
		NaturezaOperacao: firstNonEmpty(strings.TrimSpace(dto.NaturezaOperacao), "Devolução de compra"),
		Finalidade:       4, NFeReferenciada: doc.ChaveAcesso, FiscalEntryID: &doc.ID, SupplierCode: doc.SupplierCode,
		Status: entity.ExitStatusDraft, CreatedBy: userID,
	}
	src := SourceDevolucaoCompra
	exit.SourceType = &src
	end := uc.enderecoDoFornecedor(ctx, doc.ID)
	end.sobrepor(dto)
	if falta := end.faltando(); len(falta) > 0 {
		return nil, errorsuc.NewValidationError("o endereço do fornecedor está incompleto (" + strings.Join(falta, ", ") + "): informe-o na devolução")
	}
	end.aplicar(exit)

	total := decimal.Zero
	var produtos, frete, seguro, desconto, outras, ipi, icms, pis, cofins, baseST, st decimal.Decimal
	for _, l := range linhas {
		produtos, frete, seguro, desconto, outras = produtos.Add(l.Total), frete.Add(l.Frete), seguro.Add(l.Seguro), desconto.Add(l.Desconto), outras.Add(l.Outras)
		ipi, icms, pis, cofins, baseST, st = ipi.Add(l.IPI), icms.Add(l.ICMS), pis.Add(l.PIS), cofins.Add(l.COFINS), baseST.Add(l.BaseST), st.Add(l.ST)
		total = total.Add(l.ValorContabil())
	}
	fl := func(v decimal.Decimal) float64 { return v.Round(2).InexactFloat64() }
	exit.ValorProdutos, exit.ValorFrete, exit.ValorSeguro, exit.ValorDesconto = fl(produtos), fl(frete), fl(seguro), fl(desconto)
	exit.ValorIPI, exit.ValorICMS, exit.ValorPIS, exit.ValorCOFINS = fl(ipi), fl(icms), fl(pis), fl(cofins)
	exit.BaseICMSST, exit.ValorICMSST, exit.ValorTotal = fl(baseST), fl(st), fl(total)
	exit.Cfop = devolucao.CFOP(deref(porID[linhas[0].Origem.ItemID].CfopEntrada), mesmaUF, exterior)
	if exit.NumeroNF, err = uc.Repo.GetNextNFNumber(ctx); err != nil {
		return nil, err
	}
	criada, err := uc.Repo.CreateExit(ctx, exit)
	if err != nil {
		return nil, err
	}
	for i, l := range linhas {
		it := porID[l.Origem.ItemID]
		q := l.Quantidade.InexactFloat64()
		entryItem := it.ID
		cod := ""
		if it.ItemCode != nil {
			cod = fmt.Sprint(*it.ItemCode)
		}
		// A devolução é emitida pela empresa: o CST é o dela, não o do fornecedor.
		cstICMS := devolucao.CSTICMS(deref(it.CstICMS), emitenteSimples)
		item := &entity.FiscalExitItem{
			FiscalExitID: criada.ID, Sequence: i + 1, ItemCode: it.ItemCode, Ncm: it.Ncm,
			Cfop: devolucao.CFOP(deref(it.CfopEntrada), mesmaUF, exterior), Quantity: q, UnitPrice: it.UnitPrice, TotalPrice: fl(l.Total),
			BaseICMS: fl(l.BaseICMS), AliqICMS: it.AliqICMS, ValorICMS: fl(l.ICMS), BaseIPI: fl(l.BaseIPI), AliqIPI: it.AliqIPI, ValorIPI: fl(l.IPI),
			ValorPIS: fl(l.PIS), ValorCOFINS: fl(l.COFINS), BaseICMSST: fl(l.BaseST), ValorICMSST: fl(l.ST),
			CstICMS: &cstICMS, CstIPI: it.CstIPI, CstPIS: it.CstPIS, CstCOFINS: it.CstCOFINS,
			OrigemMercadoria: firstNonEmpty(deref(it.Origem), "0"), Description: it.Description, UnidadeComercial: it.UOM,
			FiscalEntryItemID: &entryItem,
		}
		if cod != "" {
			item.CodigoProduto = &cod
		}
		if _, err := uc.Repo.CreateExitItem(ctx, item); err != nil {
			return nil, err
		}
	}
	saida, err := uc.Repo.GetExitByID(ctx, criada.ID)
	if err != nil {
		return nil, err
	}
	return toFiscalExitResponse(saida), nil
}

// Efetivar faz, depois da autorização na SEFAZ, o que a devolução muda no
// estoque, no financeiro, na apuração e na contabilidade. A NF-e já está na
// SEFAZ: falhas voltam como aviso para alguém concluir.
// ReferenciarItens liga cada item da NF-e de devolução ao item da nota de
// compra (grupo DFeReferenciado: chave + número do item), exigido pela SEFAZ
// desde a reforma tributária.
func (uc *DevolucaoCompraUseCase) ReferenciarItens(ctx context.Context, exit *entity.FiscalExit, items []*entity.FiscalExitItem, payload *focusnfe.NFEPayload) error {
	if exit.Finalidade != 4 || exit.FiscalEntryID == nil || exit.NFeReferenciada == nil {
		return nil
	}
	doc, err := uc.Docs.GetEntryDocument(ctx, *exit.FiscalEntryID)
	if err != nil {
		return err
	}
	seq := make(map[int64]int, len(doc.Itens))
	for _, it := range doc.Itens {
		seq[it.ID] = it.Sequence
	}
	for i, x := range items {
		if i >= len(payload.Items) || x.FiscalEntryItemID == nil {
			continue
		}
		if n := seq[*x.FiscalEntryItemID]; n > 0 {
			payload.Items[i].DFeRefChave = *exit.NFeReferenciada
			payload.Items[i].DFeRefItem = n
		}
	}
	return nil
}

// PodeAutorizar confere, antes de transmitir à SEFAZ, que a devolução ainda
// cabe: a nota de entrada continua aprovada e nenhuma linha passa do que foi
// comprado somadas as outras devoluções vivas (criadas depois desta, por
// exemplo). Autorizar o que o ERP não consegue efetivar deixaria a SEFAZ e o
// estoque discordando.
func (uc *DevolucaoCompraUseCase) PodeAutorizar(ctx context.Context, exit *entity.FiscalExit) error {
	if exit.Finalidade != 4 || exit.FiscalEntryID == nil {
		return nil
	}
	doc, err := uc.Docs.GetEntryDocument(ctx, *exit.FiscalEntryID)
	if err != nil {
		return err
	}
	if doc.Status != entity.EntryStatusApproved && doc.Status != entity.EntryStatusWrittenOff {
		return errorsuc.NewValidationError(fmt.Sprintf("a nota de entrada %d está %s: a devolução só sai de nota aprovada", doc.NumeroNF, doc.Status))
	}
	// A SEFAZ exige que o destinatário da devolução seja o emitente da nota
	// referenciada (1194): o CNPJ do emitente está na própria chave (posições 7 a 20).
	if ref := soDigitos(deref(exit.NFeReferenciada)); len(ref) == 44 {
		if dest := soDigitos(deref(exit.CnpjDestinatario)); dest != "" && ref[6:20] != dest {
			return errorsuc.NewValidationError(fmt.Sprintf("o destinatário da devolução (%s) não é o emitente da nota referenciada (%s, pela chave): a devolução vai para quem vendeu",
				dest, ref[6:20]))
		}
	}
	devolvidas, err := uc.Devolucoes.QuantidadesDevolvidas(ctx, doc.ID)
	if err != nil {
		return err
	}
	for _, it := range doc.Itens {
		comprado := decimal.NewFromFloat(it.Quantity)
		// QuantidadesDevolvidas já inclui esta nota (rascunho não cancelado).
		if devolvidas[it.ID].GreaterThan(comprado.Add(decimal.New(1, -6))) {
			return errorsuc.NewValidationError(fmt.Sprintf("o item %d da nota %d já tem %s devolvidos em outras NF-e de devolução e só %s foram comprados: cancele a devolução excedente",
				it.Sequence, doc.NumeroNF, devolvidas[it.ID].String(), comprado.String()))
		}
	}
	return nil
}

func (uc *DevolucaoCompraUseCase) Efetivar(ctx context.Context, exit *entity.FiscalExit, userID uuid.UUID) []string {
	if exit.FiscalEntryID == nil {
		return nil
	}
	falha := func(err error) []string {
		return []string{"a devolução foi autorizada, mas não foi efetivada no estoque/financeiro: " + err.Error() + ". Corrija e use \"Efetivar devolução\" (POST /api/fiscal/devolucoes/{id}/efetivar) — repetir não duplica"}
	}
	itens, err := uc.Repo.GetExitItems(ctx, exit.ID)
	if err != nil {
		return falha(err)
	}
	doc, err := uc.Docs.GetEntryDocument(ctx, *exit.FiscalEntryID)
	if err != nil {
		return falha(err)
	}
	porID := map[int64]*entity.FiscalEntryItem{}
	for _, it := range doc.Itens {
		porID[it.ID] = it
	}
	var saidas []repository.SaidaDevolucao
	creditos := map[string]decimal.Decimal{}
	for _, x := range itens {
		if x.FiscalEntryItemID == nil {
			continue
		}
		it := porID[*x.FiscalEntryItemID]
		if it == nil {
			continue
		}
		if it.MovimentaEstoque && it.ItemCode != nil && it.WarehouseID != nil {
			fator := decimal.NewFromInt(1)
			if it.FatorConversao != nil && it.FatorConversao.IsPositive() {
				fator = *it.FatorConversao
			}
			saidas = append(saidas, repository.SaidaDevolucao{FiscalEntryItemID: it.ID, ItemCode: *it.ItemCode, WarehouseID: *it.WarehouseID,
				Quantidade: f2d(x.Quantity).Mul(fator).Round(6)})
		}
		if it.GeraCreditoICMS {
			creditos["ICMS"] = creditos["ICMS"].Add(f2d(x.ValorICMS))
		}
		if it.GeraCreditoIPI {
			creditos["IPI"] = creditos["IPI"].Add(f2d(x.ValorIPI))
		}
		if it.GeraCreditoPIS {
			creditos["PIS"] = creditos["PIS"].Add(f2d(x.ValorPIS))
		}
		if it.GeraCreditoCOFINS {
			creditos["COFINS"] = creditos["COFINS"].Add(f2d(x.ValorCOFINS))
		}
	}
	p, err := uc.Docs.AccountingParams(ctx)
	if err != nil {
		return falha(err)
	}
	montar := func(custos map[int64]decimal.Decimal) (*contabilizacao.Lote, error) {
		if p == nil || !p.ContabilizarEntrada {
			return nil, nil
		}
		return uc.loteDevolucao(ctx, exit, doc, porID, custos, creditos, p)
	}
	res, err := uc.Devolucoes.EfetivarDevolucao(ctx, repository.EfetivacaoDevolucao{
		ExitID: exit.ID, EntryID: doc.ID, NumeroNF: exit.NumeroNF, Valor: f2d(exit.ValorTotal).Round(2),
		FornecedorCode: doc.SupplierCode, FornecedorCNPJ: doc.CnpjEmitente, Data: exit.DataEmissao, Saidas: saidas, Montar: montar, UserID: userID,
	})
	if err != nil {
		return falha(err)
	}
	avisos := uc.apuracao(ctx, exit.DataEmissao, creditos, decimal.NewFromInt(-1))
	if res.Abatido.IsPositive() {
		avisos = append(avisos, fmt.Sprintf("R$ %s abatidos dos títulos em aberto da NF %d", res.Abatido.StringFixed(2), doc.NumeroNF))
	}
	if res.CreditoReceber.IsPositive() {
		avisos = append(avisos, fmt.Sprintf("R$ %s viraram crédito a receber do fornecedor (títulos da nota já pagos)", res.CreditoReceber.StringFixed(2)))
	}
	return avisos
}

func (uc *DevolucaoCompraUseCase) loteDevolucao(ctx context.Context, exit *entity.FiscalExit, doc *entity.FiscalEntry, porID map[int64]*entity.FiscalEntryItem,
	custos map[int64]decimal.Decimal, creditos map[string]decimal.Decimal, p *repository.AccountingPostingParams) (*contabilizacao.Lote, error) {
	planos := []int64{}
	for id := range custos {
		if it := porID[id]; it != nil && it.PlanoContasID != nil {
			planos = append(planos, *it.PlanoContasID)
		}
	}
	contas, err := uc.Docs.PlanoContasAccounts(ctx, planos)
	if err != nil {
		return nil, err
	}
	d := contabilizacao.Devolucao{
		Total: f2d(exit.ValorTotal).Round(2), Fornecedores: ptrConta(p.FornecedoresAccountID), Variacao: p.DespesaPadraoAccountID,
		Historico: fmt.Sprintf("Devolução NF-e %d/%s da NF %d de %s", exit.NumeroNF, exit.Serie, doc.NumeroNF, doc.RazaoSocialEmitente),
	}
	for id, custo := range custos {
		it := porID[id]
		var conta *int64
		var cc *int64
		if it != nil {
			if it.PlanoContasID != nil {
				conta = ptrConta(contas[*it.PlanoContasID])
			}
			cc = it.CentroCustoID
		}
		if conta == nil {
			conta = p.DespesaPadraoAccountID
		}
		nome := fmt.Sprintf("item %d", id)
		if it != nil {
			nome = fmt.Sprintf("item %d %s", it.Sequence, deref(it.Description))
		}
		d.Linhas = append(d.Linhas, contabilizacao.LinhaDevolucao{Conta: conta, CC: cc, Custo: custo, Nome: nome})
	}
	contaCredito := map[string]*int64{"ICMS": p.ICMSRecuperarAccountID, "IPI": p.IPIRecuperarAccountID, "PIS": p.PISRecuperarAccountID, "COFINS": p.COFINSRecuperarAccountID}
	for _, imposto := range []string{"ICMS", "IPI", "PIS", "COFINS"} {
		d.Creditos = append(d.Creditos, contabilizacao.CreditoEstornado{Conta: contaCredito[imposto], Valor: creditos[imposto], Nome: imposto})
	}
	ls, faltando := contabilizacao.NotaDevolucao(d)
	if len(faltando) > 0 {
		return nil, errorsuc.NewValidationError("a contabilização automática está ligada, mas " + strings.Join(faltando, "; "))
	}
	ids := map[int64]bool{}
	for _, l := range ls {
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
			return nil, errorsuc.NewValidationError(fmt.Sprintf("a conta contábil %d não é analítica do plano contábil configurado", id))
		}
	}
	return &contabilizacao.Lote{PlanID: p.PlanID, Data: exit.DataEmissao, Prefixo: "DEV", SourceType: repository.OrigemDevolucao, SourceID: exit.ID, Lancamentos: ls}, nil
}

func ptrConta(v int64) *int64 {
	if v <= 0 {
		return nil
	}
	return &v
}

func (uc *DevolucaoCompraUseCase) apuracao(ctx context.Context, data time.Time, creditos map[string]decimal.Decimal, sinal decimal.Decimal) []string {
	if uc.FinancialRepo == nil {
		return nil
	}
	var avisos []string
	competencia := data.Format("01/2006")
	for _, imposto := range []string{"ICMS", "IPI", "PIS", "COFINS"} {
		v := creditos[imposto]
		if !v.IsPositive() {
			continue
		}
		ta := &financialEntity.TaxAssessment{Imposto: imposto, Competencia: competencia, Creditos: v.Mul(sinal), Debitos: decimal.Zero, Status: financialEntity.TaxStatusApurar}
		if err := uc.FinancialRepo.UpsertTaxAssessmentCredito(ctx, ta); err != nil {
			avisos = append(avisos, fmt.Sprintf("o estorno do crédito de %s não foi registrado na apuração de %s: %v", imposto, competencia, err))
		}
	}
	return avisos
}

// PodeDesfazer é a checagem antes de cancelar a devolução na SEFAZ.
func (uc *DevolucaoCompraUseCase) PodeDesfazer(ctx context.Context, exit *entity.FiscalExit) error {
	if exit.FiscalEntryID == nil {
		return nil
	}
	return uc.Devolucoes.ChecarEstornoDevolucao(ctx, exit.ID)
}

// Desfazer, depois do cancelamento na SEFAZ: o estoque volta, os títulos
// abatidos reabrem, o crédito a receber é cancelado, a apuração e a
// contabilidade são estornadas.
func (uc *DevolucaoCompraUseCase) Desfazer(ctx context.Context, exit *entity.FiscalExit, userID uuid.UUID) []string {
	if exit.FiscalEntryID == nil {
		return nil
	}
	var avisos []string
	if uc.Estoque != nil {
		if _, err := uc.Estoque.EstornarEstoqueDaSaida(ctx, exit.ID, userID); err != nil {
			avisos = append(avisos, "a devolução foi cancelada, mas o estoque não voltou: "+err.Error())
		}
	}
	if err := uc.Devolucoes.EstornarDevolucao(ctx, exit.ID, time.Now()); err != nil {
		return append(avisos, "a devolução foi cancelada, mas o financeiro/contabilidade não foi desfeito: "+err.Error())
	}
	itens, err := uc.Repo.GetExitItems(ctx, exit.ID)
	if err != nil {
		return append(avisos, err.Error())
	}
	doc, err := uc.Docs.GetEntryDocument(ctx, *exit.FiscalEntryID)
	if err != nil {
		return append(avisos, err.Error())
	}
	porID := map[int64]*entity.FiscalEntryItem{}
	for _, it := range doc.Itens {
		porID[it.ID] = it
	}
	creditos := map[string]decimal.Decimal{}
	for _, x := range itens {
		if x.FiscalEntryItemID == nil || porID[*x.FiscalEntryItemID] == nil {
			continue
		}
		it := porID[*x.FiscalEntryItemID]
		if it.GeraCreditoICMS {
			creditos["ICMS"] = creditos["ICMS"].Add(f2d(x.ValorICMS))
		}
		if it.GeraCreditoIPI {
			creditos["IPI"] = creditos["IPI"].Add(f2d(x.ValorIPI))
		}
		if it.GeraCreditoPIS {
			creditos["PIS"] = creditos["PIS"].Add(f2d(x.ValorPIS))
		}
		if it.GeraCreditoCOFINS {
			creditos["COFINS"] = creditos["COFINS"].Add(f2d(x.ValorCOFINS))
		}
	}
	return append(avisos, uc.apuracao(ctx, exit.DataEmissao, creditos, decimal.NewFromInt(1))...)
}

// Reprocessar efetiva de novo a devolução autorizada (idempotente: o que já
// foi feito não se repete). Para quando a efetivação falhou na autorização.
func (uc *DevolucaoCompraUseCase) Reprocessar(ctx context.Context, exitID int64) (*response.FiscalExitResponse, error) {
	if !uc.Auth.CanAuthorizeFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	exit, err := uc.Repo.GetExitByID(ctx, exitID)
	if err != nil {
		return nil, err
	}
	if exit.FiscalEntryID == nil {
		return nil, errorsuc.NewValidationError("esta NF-e não é uma devolução de compra")
	}
	if exit.Status != entity.ExitStatusAuthorized {
		return nil, errorsuc.NewValidationError("só a devolução autorizada na SEFAZ é efetivada")
	}
	resp := toFiscalExitResponse(exit)
	resp.Warnings = uc.Efetivar(ctx, exit, userID)
	return resp, nil
}

// enderecoFornecedor é o destinatário da devolução: o endereço que o
// fornecedor declarou no XML da nota, completado pelo que a tela informar.
type enderecoFornecedor struct {
	Logradouro, Numero, Complemento, Bairro, Municipio, CodigoMunicipio, CEP, Telefone string
}

func (uc *DevolucaoCompraUseCase) enderecoDoFornecedor(ctx context.Context, entryID int64) enderecoFornecedor {
	var e enderecoFornecedor
	if _, xmlTxt, err := uc.Docs.GetEntryXML(ctx, entryID); err == nil && strings.TrimSpace(xmlTxt) != "" {
		if n, err := LerNFe([]byte(xmlTxt)); err == nil {
			e = enderecoFornecedor{Logradouro: strings.TrimSpace(n.EmitenteLogradouro), Numero: strings.TrimSpace(n.EmitenteNumero),
				Complemento: strings.TrimSpace(n.EmitenteComplemento), Bairro: strings.TrimSpace(n.EmitenteBairro),
				Municipio: strings.TrimSpace(n.EmitenteMunicipio), CodigoMunicipio: strings.TrimSpace(n.EmitenteCodigoMunicipio),
				CEP: strings.TrimSpace(n.EmitenteCEP), Telefone: strings.TrimSpace(n.EmitenteFone)}
		}
	}
	return e
}

func (e *enderecoFornecedor) sobrepor(dto DevolucaoCompraDTO) {
	for _, par := range []struct {
		dst *string
		src *string
	}{
		{&e.Logradouro, dto.DestLogradouro}, {&e.Numero, dto.DestNumero}, {&e.Complemento, dto.DestComplemento}, {&e.Bairro, dto.DestBairro},
		{&e.Municipio, dto.DestMunicipio}, {&e.CodigoMunicipio, dto.DestCodigoMunicipio}, {&e.CEP, dto.DestCEP},
	} {
		if par.src != nil && strings.TrimSpace(*par.src) != "" {
			*par.dst = strings.TrimSpace(*par.src)
		}
	}
	e.CEP, e.CodigoMunicipio = soDigitos(e.CEP), soDigitos(e.CodigoMunicipio)
}

// faltando lista, em ordem fixa, o que a NF-e exige do endereço do destinatário.
func (e enderecoFornecedor) faltando() []string {
	var falta []string
	for _, c := range []struct{ nome, valor string }{
		{"logradouro", e.Logradouro}, {"número", e.Numero}, {"bairro", e.Bairro}, {"município", e.Municipio},
		{"código do município (IBGE)", e.CodigoMunicipio}, {"CEP", e.CEP},
	} {
		if strings.TrimSpace(c.valor) == "" {
			falta = append(falta, c.nome)
		}
	}
	return falta
}

func (e enderecoFornecedor) aplicar(x *entity.FiscalExit) {
	opt := func(s string) *string {
		if s = strings.TrimSpace(s); s == "" {
			return nil
		}
		return &s
	}
	x.DestLogradouro, x.DestNumero, x.DestComplemento = opt(e.Logradouro), opt(e.Numero), opt(e.Complemento)
	x.DestBairro, x.DestMunicipio, x.DestCodigoMunicipio = opt(e.Bairro), opt(e.Municipio), opt(e.CodigoMunicipio)
	x.DestCEP, x.DestTelefone = opt(e.CEP), opt(e.Telefone)
}
