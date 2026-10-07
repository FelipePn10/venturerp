package fiscal_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/accounting/contabilizacao"
	financialEntity "github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	financialrepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/frete"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

// FreteUseCase é o frete sobre compras (conhecimento de frete): o CT-e da
// transportadora que trouxe a mercadoria. Ligado às notas de entrada que
// transportou, ao ser lançado rateia o custo entre os itens, complementa o
// custo do estoque, gera o título da transportadora e contabiliza.
type FreteUseCase struct {
	Repo          repository.FreightRepository
	Docs          repository.FiscalEntryDocumentRepository
	Fiscal        repository.FiscalRepository
	FinancialRepo financialrepo.FinancialRepository
	Auth          ports.AuthService
}

// FreteDTO é o frete digitado ou ajustado na tela.
type FreteDTO struct {
	SupplierCode   *int64          `json:"supplier_code,omitempty"`
	ChaveCTe       *string         `json:"chave_cte,omitempty"`
	Numero         int64           `json:"numero"`
	Serie          string          `json:"serie"`
	DataEmissao    string          `json:"data_emissao"`
	CNPJ           string          `json:"cnpj_transportadora"`
	Nome           string          `json:"nome_transportadora"`
	UF             *string         `json:"uf_transportadora,omitempty"`
	CFOP           *string         `json:"cfop,omitempty"`
	ValorFrete     decimal.Decimal `json:"valor_frete"`
	BaseICMS       decimal.Decimal `json:"base_icms"`
	AliqICMS       decimal.Decimal `json:"aliq_icms"`
	ValorICMS      decimal.Decimal `json:"valor_icms"`
	CreditaICMS    *bool           `json:"credita_icms,omitempty"`
	TipoRateio     string          `json:"tipo_rateio"`
	DataVencimento string          `json:"data_vencimento"`
	Observacao     *string         `json:"observacao,omitempty"`
	NotasIDs       []int64         `json:"notas_ids"`
}

func dataISO(s, campo string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, errorsuc.NewValidationError(fmt.Sprintf("%s inválida: use AAAA-MM-DD", campo))
	}
	return t, nil
}

func (uc *FreteUseCase) validar(f *repository.FreightDocument) error {
	if f.Numero <= 0 {
		return errorsuc.NewValidationError("informe o número do CT-e")
	}
	if len(soDigitos(f.CNPJTransportadora)) != 14 {
		return errorsuc.NewValidationError("informe o CNPJ da transportadora")
	}
	if !f.ValorFrete.IsPositive() {
		return errorsuc.NewValidationError("o valor do frete precisa ser maior que zero")
	}
	if f.ValorICMS.IsNegative() || f.ValorICMS.GreaterThan(f.ValorFrete) {
		return errorsuc.NewValidationError("o ICMS do frete não pode ser negativo nem maior que o frete")
	}
	switch f.TipoRateio {
	case frete.RateioValor, frete.RateioQuantidade, frete.RateioPeso:
	default:
		return errorsuc.NewValidationError("tipo de rateio: VALOR, QUANTIDADE ou PESO")
	}
	if f.ChaveCTe != nil && *f.ChaveCTe != "" && len(soDigitos(*f.ChaveCTe)) != 44 {
		return errorsuc.NewValidationError("a chave do CT-e tem 44 dígitos")
	}
	return nil
}

func (uc *FreteUseCase) doDTO(ctx context.Context, f *repository.FreightDocument, dto FreteDTO) error {
	var err error
	f.Numero, f.Serie = dto.Numero, firstNonEmpty(strings.TrimSpace(dto.Serie), "1")
	if f.DataEmissao, err = dataISO(dto.DataEmissao, "data de emissão"); err != nil {
		return err
	}
	if f.DataVencimento, err = dataISO(dto.DataVencimento, "data de vencimento"); err != nil {
		return err
	}
	f.CNPJTransportadora, f.NomeTransportadora, f.UFTransportadora, f.CFOP = soDigitos(dto.CNPJ), strings.TrimSpace(dto.Nome), dto.UF, dto.CFOP
	f.ValorFrete, f.BaseICMS, f.AliqICMS, f.ValorICMS = dto.ValorFrete.Round(2), dto.BaseICMS.Round(2), dto.AliqICMS, dto.ValorICMS.Round(2)
	f.CreditaICMS = dto.CreditaICMS == nil || *dto.CreditaICMS
	f.TipoRateio = strings.ToUpper(firstNonEmpty(strings.TrimSpace(dto.TipoRateio), frete.RateioValor))
	f.Observacao = dto.Observacao
	if dto.ChaveCTe != nil {
		ch := soDigitos(*dto.ChaveCTe)
		f.ChaveCTe = &ch
	}
	f.SupplierCode = dto.SupplierCode
	if f.SupplierCode == nil && f.CNPJTransportadora != "" {
		if s, err := uc.Docs.FindSupplierByDocument(ctx, f.CNPJTransportadora); err == nil && s != nil {
			code := s.Code
			f.SupplierCode = &code
		}
	}
	return uc.validar(f)
}

// Criar registra o frete digitado.
func (uc *FreteUseCase) Criar(ctx context.Context, dto FreteDTO) (*response.FreteResponse, error) {
	if !uc.Auth.CanCreateFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	f := &repository.FreightDocument{CreatedBy: userID}
	if err := uc.doDTO(ctx, f, dto); err != nil {
		return nil, err
	}
	criado, err := uc.Repo.CreateFreight(ctx, f, dto.NotasIDs)
	if err != nil {
		return nil, err
	}
	return uc.responder(ctx, criado)
}

// ImportarXML registra o frete a partir do XML do CT-e: as notas
// transportadas são ligadas pelas chaves que o CT-e cita. O tomador (quem paga)
// precisa ser a empresa.
func (uc *FreteUseCase) ImportarXML(ctx context.Context, conteudo []byte, vencimento, tipoRateio string) (*response.FreteResponse, error) {
	if !uc.Auth.CanCreateFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	c, err := LerCTe(conteudo)
	if err != nil {
		return nil, err
	}
	if cfg, err := uc.Fiscal.GetFiscalConfig(ctx); err == nil && cfg != nil && c.TomadorCNPJ != "" {
		if empresa := soDigitos(cfg.CnpjEmpresa); empresa != "" && empresa != c.TomadorCNPJ {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("o tomador do CT-e (quem paga o frete) é o CNPJ %s, não a empresa (%s): este frete não é da empresa", c.TomadorCNPJ, empresa))
		}
	}
	if strings.TrimSpace(vencimento) == "" {
		vencimento = c.DataEmissao.AddDate(0, 0, 30).Format("2006-01-02")
	}
	notas, err := uc.Repo.EntriesByChaves(ctx, c.ChavesNFe)
	if err != nil {
		return nil, err
	}
	chave := c.Chave
	cfop := c.CFOP
	uf := c.EmitenteUF
	dto := FreteDTO{
		ChaveCTe: &chave, Numero: c.Numero, Serie: c.Serie, DataEmissao: c.DataEmissao.Format("2006-01-02"),
		CNPJ: c.EmitenteCNPJ, Nome: c.EmitenteNome, UF: &uf, CFOP: &cfop,
		ValorFrete: c.ValorPrestacao, BaseICMS: c.BaseICMS, AliqICMS: c.AliqICMS, ValorICMS: c.ValorICMS,
		TipoRateio: tipoRateio, DataVencimento: vencimento, NotasIDs: notas,
	}
	f := &repository.FreightDocument{CreatedBy: userID}
	if err := uc.doDTO(ctx, f, dto); err != nil {
		return nil, err
	}
	xmlTxt := c.ConteudoXML
	f.XMLContent = &xmlTxt
	criado, err := uc.Repo.CreateFreight(ctx, f, notas)
	if err != nil {
		return nil, err
	}
	resp, err := uc.responder(ctx, criado)
	if err != nil {
		return nil, err
	}
	if faltam := len(c.ChavesNFe) - len(notas); faltam > 0 {
		resp.Avisos = append(resp.Avisos, fmt.Sprintf("%d nota(s) citada(s) no CT-e ainda não foram lançadas como entrada: importe-as e vincule ao frete", faltam))
	}
	return resp, nil
}

// Atualizar altera o frete pendente (notas, rateio, vencimento, ICMS).
func (uc *FreteUseCase) Atualizar(ctx context.Context, id int64, dto FreteDTO) (*response.FreteResponse, error) {
	if !uc.Auth.CanCreateFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	f, err := uc.Repo.GetFreight(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := uc.doDTO(ctx, f, dto); err != nil {
		return nil, err
	}
	if err := uc.Repo.UpdateFreight(ctx, f, dto.NotasIDs); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

func (uc *FreteUseCase) Obter(ctx context.Context, id int64) (*response.FreteResponse, error) {
	if !uc.Auth.CanGetFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	f, err := uc.Repo.GetFreight(ctx, id)
	if err != nil {
		return nil, err
	}
	return uc.responder(ctx, f)
}

func (uc *FreteUseCase) Listar(ctx context.Context, status string) ([]response.FreteResponse, error) {
	if !uc.Auth.CanGetFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	lista, err := uc.Repo.ListFreights(ctx, status)
	if err != nil {
		return nil, err
	}
	out := make([]response.FreteResponse, 0, len(lista))
	for _, f := range lista {
		out = append(out, *toFreteResponse(f, decimal.Zero))
	}
	return out, nil
}

// creditoEfetivo: o ICMS do CT-e só deixa de ser custo quando é creditado e,
// com a contabilização ligada, há conta de ICMS a recuperar (sem ela o
// crédito vira custo — a mesma regra da nota de entrada).
func creditoEfetivo(f *repository.FreightDocument, p *repository.AccountingPostingParams) bool {
	if !f.CreditaICMS || !f.ValorICMS.IsPositive() {
		return false
	}
	if p != nil && p.ContabilizarEntrada && p.ICMSRecuperarAccountID == nil {
		return false
	}
	return true
}

func (uc *FreteUseCase) responder(ctx context.Context, f *repository.FreightDocument) (*response.FreteResponse, error) {
	p, err := uc.Docs.AccountingParams(ctx)
	if err != nil {
		return nil, err
	}
	custo := frete.CustoDoFrete(f.ValorFrete, f.ValorICMS, creditoEfetivo(f, p))
	r := toFreteResponse(f, custo)
	if f.Status != repository.FreteStatusPendente || len(f.Notas) == 0 {
		return r, nil
	}
	ids := make([]int64, 0, len(f.Notas))
	pendentes := []string{}
	for _, n := range f.Notas {
		ids = append(ids, n.FiscalEntryID)
		if n.Status != "APPROVED" {
			pendentes = append(pendentes, fmt.Sprintf("NF %d", n.NumeroNF))
		}
	}
	if len(pendentes) > 0 {
		r.Avisos = append(r.Avisos, "aprove as notas antes de lançar o frete: "+strings.Join(pendentes, ", "))
	}
	itens, err := uc.Repo.FreteItens(ctx, ids)
	if err != nil {
		return nil, err
	}
	partes, err := frete.Ratear(custo, f.TipoRateio, itens)
	if err != nil {
		r.Avisos = append(r.Avisos, err.Error())
		return r, nil
	}
	for _, pt := range partes {
		r.Previa = append(r.Previa, response.FreteAlocacaoResponse{
			FiscalEntryID: pt.Item.EntryID, FiscalEntryItemID: pt.Item.ItemID, ItemCode: pt.Item.ItemCode, Descricao: pt.Item.Descricao,
			Valor: pt.Valor.InexactFloat64(), ValorEstoque: pt.ValorEstoque.InexactFloat64(), ValorDespesa: pt.ValorDespesa.InexactFloat64(),
		})
	}
	return r, nil
}

// Lancar efetiva o frete (custo, título, contabilidade) e registra o crédito
// de ICMS na apuração.
func (uc *FreteUseCase) Lancar(ctx context.Context, id int64) (*response.FreteResponse, error) {
	if !uc.Auth.CanApproveFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	f, err := uc.Repo.GetFreight(ctx, id)
	if err != nil {
		return nil, err
	}
	p, err := uc.Docs.AccountingParams(ctx)
	if err != nil {
		return nil, err
	}
	credita := creditoEfetivo(f, p)
	custo := frete.CustoDoFrete(f.ValorFrete, f.ValorICMS, credita)
	montar := func(partes []frete.Parte) (repository.TituloAPagar, *contabilizacao.Lote, error) {
		titulo := tituloDoFrete(f, partes)
		if p == nil || !p.ContabilizarEntrada {
			return titulo, nil, nil
		}
		lote, err := uc.loteDoFrete(ctx, f, p, partes, credita)
		return titulo, lote, err
	}
	if err := uc.Repo.LancarFrete(ctx, repository.LancamentoFrete{FreightID: id, Custo: custo, Montar: montar, UserID: userID}); err != nil {
		return nil, err
	}
	lancado, err := uc.Repo.GetFreight(ctx, id)
	if err != nil {
		return nil, err
	}
	resp, err := uc.responder(ctx, lancado)
	if err != nil {
		return nil, err
	}
	if credita {
		resp.Avisos = append(resp.Avisos, uc.creditoICMS(ctx, lancado, decimal.NewFromInt(1))...)
	}
	return resp, nil
}

// Cancelar desfaz o frete (recusado se o título da transportadora foi pago).
func (uc *FreteUseCase) Cancelar(ctx context.Context, id int64, motivo string) (*response.FreteResponse, error) {
	if !uc.Auth.CanApproveFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if len([]rune(strings.TrimSpace(motivo))) < 10 {
		return nil, errorsuc.NewValidationError("informe o motivo do cancelamento (pelo menos 10 caracteres)")
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	f, err := uc.Repo.GetFreight(ctx, id)
	if err != nil {
		return nil, err
	}
	p, err := uc.Docs.AccountingParams(ctx)
	if err != nil {
		return nil, err
	}
	estavaLancado := f.Status == repository.FreteStatusLancado
	if err := uc.Repo.CancelarFrete(ctx, repository.CancelamentoFrete{FreightID: id, Motivo: strings.TrimSpace(motivo), UserID: userID, DataEstorno: time.Now()}); err != nil {
		return nil, err
	}
	cancelado, err := uc.Repo.GetFreight(ctx, id)
	if err != nil {
		return nil, err
	}
	resp, err := uc.responder(ctx, cancelado)
	if err != nil {
		return nil, err
	}
	if estavaLancado && creditoEfetivo(f, p) {
		resp.Avisos = append(resp.Avisos, uc.creditoICMS(ctx, cancelado, decimal.NewFromInt(-1))...)
	}
	return resp, nil
}

// tituloDoFrete: um título para a transportadora pelo valor do CT-e, com o
// rateio por plano de contas na proporção do frete de cada item.
func tituloDoFrete(f *repository.FreightDocument, partes []frete.Parte) repository.TituloAPagar {
	type chave struct{ plano, cc int64 }
	pesos := map[chave]decimal.Decimal{}
	var ordem []chave
	for _, p := range partes {
		if p.Item.PlanoContasID == nil {
			continue
		}
		k := chave{plano: *p.Item.PlanoContasID}
		if p.Item.CentroCustoID != nil {
			k.cc = *p.Item.CentroCustoID
		}
		if _, ok := pesos[k]; !ok {
			ordem = append(ordem, k)
		}
		pesos[k] = pesos[k].Add(p.Valor)
	}
	soma := decimal.Zero
	for _, v := range pesos {
		soma = soma.Add(v)
	}
	var rateio []entity.InstallmentAllocation
	acumulado := decimal.Zero
	for i, k := range ordem {
		v := f.ValorFrete.Mul(pesos[k]).Div(soma).Round(2)
		if i == len(ordem)-1 {
			v = f.ValorFrete.Sub(acumulado)
		}
		acumulado = acumulado.Add(v)
		a := entity.InstallmentAllocation{PlanoContasID: k.plano, Valor: v}
		if k.cc > 0 {
			cc := k.cc
			a.CentroCustoID = &cc
		}
		rateio = append(rateio, a)
	}
	obs := fmt.Sprintf("Frete do CT-e %d/%s (%s) sobre compras", f.Numero, f.Serie, f.NomeTransportadora)
	t := repository.TituloAPagar{
		TipoDocumento: "CTE", NumeroDocumento: fmt.Sprintf("CTE-%d/%s", f.Numero, f.Serie),
		FornecedorCode: f.SupplierCode, FornecedorCNPJ: f.CNPJTransportadora, DataEmissao: f.DataEmissao,
		DataVencimento: f.DataVencimento, Valor: f.ValorFrete, ParcelaNumero: 1, ParcelaTotal: 1, Observacao: &obs, Rateio: rateio,
	}
	t.PlanoContasID, t.CentroCustoID = planoUnico(rateio)
	return t
}

// loteDoFrete: D conta do plano do item (parte em estoque) ou despesa padrão
// (parte consumida) × C Fornecedores; D ICMS a recuperar × C Fornecedores.
func (uc *FreteUseCase) loteDoFrete(ctx context.Context, f *repository.FreightDocument, p *repository.AccountingPostingParams, partes []frete.Parte, credita bool) (*contabilizacao.Lote, error) {
	planos := []int64{}
	for _, pt := range partes {
		if pt.Item.PlanoContasID != nil {
			planos = append(planos, *pt.Item.PlanoContasID)
		}
	}
	contas, err := uc.Docs.PlanoContasAccounts(ctx, planos)
	if err != nil {
		return nil, err
	}
	hist := fmt.Sprintf("Frete CT-e %d/%s %s", f.Numero, f.Serie, f.NomeTransportadora)
	var ls []contabilizacao.Lancamento
	var faltando []string
	for _, pt := range partes {
		var conta int64
		if pt.Item.PlanoContasID != nil {
			conta = contas[*pt.Item.PlanoContasID]
		}
		despesa := conta
		if p.DespesaPadraoAccountID != nil {
			despesa = *p.DespesaPadraoAccountID
		}
		if conta == 0 && pt.ValorEstoque.IsPositive() {
			conta = despesa
		}
		if (pt.ValorEstoque.IsPositive() && conta == 0) || (pt.ValorDespesa.IsPositive() && despesa == 0) {
			faltando = append(faltando, fmt.Sprintf("o item %q não tem conta contábil pelo plano de contas (e não há despesa padrão)", pt.Item.Descricao))
			continue
		}
		if pt.ValorEstoque.IsPositive() {
			ls = append(ls, contabilizacao.Lancamento{DebitoID: conta, CreditoID: p.FornecedoresAccountID, DebitoCC: pt.Item.CentroCustoID, Valor: pt.ValorEstoque, Historico: hist + " — custo do estoque"})
		}
		if pt.ValorDespesa.IsPositive() {
			ls = append(ls, contabilizacao.Lancamento{DebitoID: despesa, CreditoID: p.FornecedoresAccountID, DebitoCC: pt.Item.CentroCustoID, Valor: pt.ValorDespesa, Historico: hist + " — material já consumido"})
		}
	}
	if credita {
		ls = append(ls, contabilizacao.Lancamento{DebitoID: *p.ICMSRecuperarAccountID, CreditoID: p.FornecedoresAccountID, Valor: f.ValorICMS, Historico: hist + " — ICMS a recuperar"})
	}
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
	return &contabilizacao.Lote{PlanID: p.PlanID, Data: f.DataEmissao, Prefixo: "FRE", SourceType: repository.OrigemFrete, SourceID: f.ID, Lancamentos: ls}, nil
}

func (uc *FreteUseCase) creditoICMS(ctx context.Context, f *repository.FreightDocument, sinal decimal.Decimal) []string {
	if uc.FinancialRepo == nil {
		return nil
	}
	competencia := f.DataEmissao.Format("01/2006")
	ta := &financialEntity.TaxAssessment{Imposto: "ICMS", Competencia: competencia, Creditos: f.ValorICMS.Mul(sinal), Debitos: decimal.Zero, Status: financialEntity.TaxStatusApurar}
	if err := uc.FinancialRepo.UpsertTaxAssessmentCredito(ctx, ta); err != nil {
		return []string{fmt.Sprintf("o crédito de ICMS do frete não foi registrado na apuração de %s: %v", competencia, err)}
	}
	return nil
}

func toFreteResponse(f *repository.FreightDocument, custo decimal.Decimal) *response.FreteResponse {
	r := &response.FreteResponse{
		ID: f.ID, ChaveCTe: f.ChaveCTe, Numero: f.Numero, Serie: f.Serie, DataEmissao: f.DataEmissao.Format("2006-01-02"),
		CNPJTransportadora: f.CNPJTransportadora, NomeTransportadora: f.NomeTransportadora, UFTransportadora: f.UFTransportadora,
		SupplierCode: f.SupplierCode, SupplierName: f.SupplierName, CFOP: f.CFOP,
		ValorFrete: f.ValorFrete.InexactFloat64(), BaseICMS: f.BaseICMS.InexactFloat64(), AliqICMS: f.AliqICMS.InexactFloat64(),
		ValorICMS: f.ValorICMS.InexactFloat64(), CreditaICMS: f.CreditaICMS, CustoFrete: custo.InexactFloat64(), TipoRateio: f.TipoRateio,
		DataVencimento: f.DataVencimento.Format("2006-01-02"), Status: f.Status, ContaPagarID: f.ContaPagarID, Observacao: f.Observacao,
		CreatedAt: f.CreatedAt, LancadoEm: f.LancadoEm, CanceladoEm: f.CanceladoEm, CancelReason: f.CancelReason,
	}
	for _, n := range f.Notas {
		r.Notas = append(r.Notas, response.FreteNotaResponse{FiscalEntryID: n.FiscalEntryID, NumeroNF: n.NumeroNF, Serie: n.Serie,
			Emitente: n.Emitente, ValorTotal: n.ValorTotal.InexactFloat64(), Status: n.Status, ChaveAcesso: n.ChaveAcesso})
	}
	for _, a := range f.Alocacoes {
		r.Alocacoes = append(r.Alocacoes, response.FreteAlocacaoResponse{FiscalEntryID: a.FiscalEntryID, FiscalEntryItemID: a.FiscalEntryItemID,
			ItemCode: a.ItemCode, Descricao: a.Descricao, Valor: a.Valor.InexactFloat64(), ValorEstoque: a.ValorEstoque.InexactFloat64(),
			ValorDespesa: a.ValorDespesa.InexactFloat64()})
	}
	return r
}
