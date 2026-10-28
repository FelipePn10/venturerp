// Package devolucao_venda_uc: devolução de venda do cliente (comercial,
// Fase 5.1). Nasce da NF de saída autorizada; é efetivada pela NF-e de
// entrada emitida pela empresa (autorizada na SEFAZ) ou pela NF de devolução
// do próprio cliente.
package devolucao_venda_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/devolucaovenda"
	financialEntity "github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	financialrepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

const SourceDevolucaoVenda = "DEVOLUCAO_VENDA"

type UseCase struct {
	Repo          dom.Repository
	Fiscal        fiscalrepo.FiscalRepository
	FinancialRepo financialrepo.FinancialRepository
	Auth          ports.AuthService
	Agora         func() time.Time
}

func (uc *UseCase) agora() time.Time {
	if uc.Agora != nil {
		return uc.Agora()
	}
	return time.Now()
}

func (uc *UseCase) ator(ctx context.Context) (uuid.UUID, error) {
	a, err := uc.Auth.UserID(ctx)
	if err != nil || a == uuid.Nil {
		return uuid.Nil, errorsuc.ErrUnauthorized
	}
	return a, nil
}

type Previa struct {
	FiscalExitID int64             `json:"fiscal_exit_id"`
	NumeroNF     int64             `json:"numero_nf"`
	CustomerCode *int64            `json:"customer_code,omitempty"`
	Linhas       []dom.LinhaDaNota `json:"linhas"`
}

// Previa: as linhas da NF com o vendido, o já devolvido e as séries no cliente.
func (uc *UseCase) Previa(ctx context.Context, fiscalExitID int64) (*Previa, error) {
	n, cliente, _, linhas, err := uc.Repo.LinhasDaNota(ctx, fiscalExitID)
	if err != nil {
		return nil, err
	}
	return &Previa{FiscalExitID: fiscalExitID, NumeroNF: n, CustomerCode: cliente, Linhas: linhas}, nil
}

type CriarDTO struct {
	FiscalExitID   int64        `json:"fiscal_exit_id"`
	Modo           string       `json:"modo"`
	Motivo         string       `json:"motivo"`
	Observacao     *string      `json:"observacao,omitempty"`
	WarehouseID    int64        `json:"warehouse_id"`
	NFClienteChave *string      `json:"nf_cliente_chave,omitempty"`
	Itens          []dom.Pedido `json:"itens"`
}

func (uc *UseCase) Criar(ctx context.Context, dto CriarDTO) (*dom.Devolucao, error) {
	por, err := uc.ator(ctx)
	if err != nil {
		return nil, err
	}
	modo, motivo := strings.ToUpper(strings.TrimSpace(dto.Modo)), strings.ToUpper(strings.TrimSpace(dto.Motivo))
	var chave *string
	if dto.NFClienteChave != nil {
		c := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, *dto.NFClienteChave)
		chave = &c
	}
	if err := dom.ConferirAbertura(modo, motivo, chave); err != nil {
		return nil, err
	}
	if dto.WarehouseID <= 0 {
		return nil, errorsuc.NewValidationError("informe o almoxarifado que recebe a mercadoria devolvida")
	}
	numero, cliente, chaveOrigem, linhas, err := uc.Repo.LinhasDaNota(ctx, dto.FiscalExitID)
	if err != nil {
		return nil, err
	}
	if modo == dom.NFCliente && chave != nil && *chave == chaveOrigem {
		return nil, errorsuc.NewValidationError("a chave informada é a da própria NF de venda: informe a da NF de devolução do cliente")
	}
	itens, total, err := dom.Calcular(linhas, dto.Itens)
	if err != nil {
		return nil, err
	}
	d := &dom.Devolucao{FiscalExitID: dto.FiscalExitID, NumeroNF: numero, CustomerCode: cliente, Modo: modo, Situacao: dom.Rascunho, Motivo: motivo,
		Observacao: dto.Observacao, WarehouseID: dto.WarehouseID, NFClienteChave: chave, ValorTotal: total, Itens: itens}
	if chave != nil && len(*chave) == 44 {
		var n int64
		if _, err := fmt.Sscan((*chave)[25:34], &n); err == nil {
			d.NFClienteNumero = &n
		}
	}
	if err := uc.Repo.Criar(ctx, d, &por); err != nil {
		return nil, err
	}
	return uc.Repo.Obter(ctx, d.ID)
}

func (uc *UseCase) Obter(ctx context.Context, id int64) (*dom.Devolucao, error) {
	return uc.Repo.Obter(ctx, id)
}

func (uc *UseCase) Listar(ctx context.Context, fiscalExitID *int64, situacao string) ([]dom.Devolucao, error) {
	return uc.Repo.Listar(ctx, fiscalExitID, strings.ToUpper(strings.TrimSpace(situacao)))
}

// GerarNF (emissão própria): a NF-e de entrada da devolução em rascunho, com
// os itens e impostos proporcionais, o CFOP de devolução e a NF de venda
// referenciada. É transmitida pela tela da NF-e (VFIS0200) como as outras.
func (uc *UseCase) GerarNF(ctx context.Context, id int64) (*dom.Devolucao, error) {
	if !uc.Auth.CanCreateFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	por, err := uc.ator(ctx)
	if err != nil {
		return nil, err
	}
	d, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	if d.Modo != dom.EmissaoPropria {
		return nil, errorsuc.NewValidationError("a devolução é pela NF do cliente: não se emite NF-e própria")
	}
	if d.Situacao != dom.Rascunho {
		return nil, errorsuc.NewValidationError("só a devolução em rascunho gera a NF-e de entrada")
	}
	if d.NFPropriaID != nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a NF-e de entrada da devolução já foi gerada (id %d)", *d.NFPropriaID))
	}
	orig, err := uc.Fiscal.GetExitByID(ctx, d.FiscalExitID)
	if err != nil {
		return nil, err
	}
	if orig.ChaveAcesso == nil || len(strings.TrimSpace(*orig.ChaveAcesso)) != 44 {
		return nil, errorsuc.NewValidationError("a NF de venda não tem chave de acesso para ser referenciada")
	}
	origItens, err := uc.Fiscal.GetExitItems(ctx, d.FiscalExitID)
	if err != nil {
		return nil, err
	}
	porID := map[int64]*entity.FiscalExitItem{}
	for _, it := range origItens {
		porID[it.ID] = it
	}
	fl := func(v decimal.Decimal) float64 { return v.Round(2).InexactFloat64() }
	var produtos, icms, ipi, pis, cofins, st, baseST decimal.Decimal
	for _, it := range d.Itens {
		produtos, icms, ipi, pis, cofins, st = produtos.Add(it.ValorProdutos), icms.Add(it.ICMS), ipi.Add(it.IPI), pis.Add(it.PIS), cofins.Add(it.COFINS), st.Add(it.ST)
		baseST = baseST.Add(it.BaseST)
	}
	tipo := 0
	src := SourceDevolucaoVenda
	devID := d.ID
	exit := &entity.FiscalExit{
		Serie: orig.Serie, DataEmissao: uc.agora(),
		CnpjDestinatario: orig.CnpjDestinatario, RazaoSocialDestinatario: orig.RazaoSocialDestinatario, IEDestinatario: orig.IEDestinatario, UFDestinatario: orig.UFDestinatario,
		DestLogradouro: orig.DestLogradouro, DestNumero: orig.DestNumero, DestComplemento: orig.DestComplemento, DestBairro: orig.DestBairro,
		DestMunicipio: orig.DestMunicipio, DestCodigoMunicipio: orig.DestCodigoMunicipio, DestCEP: orig.DestCEP, DestEmail: orig.DestEmail, DestTelefone: orig.DestTelefone,
		CustomerCode: orig.CustomerCode, NaturezaOperacao: "Devolução de venda", Finalidade: 4, NFeReferenciada: orig.ChaveAcesso,
		TipoDocumento: &tipo, DevolucaoVendaID: &devID, SourceType: &src, Status: entity.ExitStatusDraft, CreatedBy: por,
		ValorProdutos: fl(produtos), ValorICMS: fl(icms), ValorIPI: fl(ipi), ValorPIS: fl(pis), ValorCOFINS: fl(cofins),
		BaseICMSST: fl(baseST), ValorICMSST: fl(st), ValorTotal: fl(d.ValorTotal),
	}
	if len(d.Itens) > 0 {
		exit.Cfop = d.Itens[0].CFOP
	}
	if exit.NumeroNF, err = uc.Fiscal.GetNextNFNumber(ctx); err != nil {
		return nil, err
	}
	criada, err := uc.Fiscal.CreateExit(ctx, exit)
	if err != nil {
		return nil, err
	}
	for i, it := range d.Itens {
		o := porID[it.FiscalExitItemID]
		if o == nil {
			return nil, fmt.Errorf("linha %d da NF de venda não encontrada", it.FiscalExitItemID)
		}
		origem := it.FiscalExitItemID
		cfop := it.CFOP
		item := &entity.FiscalExitItem{
			FiscalExitID: criada.ID, Sequence: i + 1, ItemCode: o.ItemCode, Ncm: o.Ncm, Cfop: cfop,
			Quantity: it.Quantidade.InexactFloat64(), UnitPrice: o.UnitPrice, TotalPrice: fl(it.ValorProdutos),
			BaseICMS: fl(it.BaseICMS), AliqICMS: o.AliqICMS, ValorICMS: fl(it.ICMS), BaseIPI: fl(it.BaseIPI), AliqIPI: o.AliqIPI, ValorIPI: fl(it.IPI),
			AliqPIS: o.AliqPIS, ValorPIS: fl(it.PIS), AliqCOFINS: o.AliqCOFINS, ValorCOFINS: fl(it.COFINS), BaseICMSST: fl(it.BaseST), AliqICMSST: o.AliqICMSST, ValorICMSST: fl(it.ST),
			CstICMS: o.CstICMS, CstIPI: o.CstIPI, CstPIS: o.CstPIS, CstCOFINS: o.CstCOFINS, OrigemMercadoria: o.OrigemMercadoria,
			Description: o.Description, UnidadeComercial: o.UnidadeComercial, CodigoProduto: o.CodigoProduto, FiscalExitItemOrigemID: &origem,
		}
		if _, err := uc.Fiscal.CreateExitItem(ctx, item); err != nil {
			return nil, err
		}
	}
	if err := uc.Repo.VincularNFPropria(ctx, d.ID, criada.ID); err != nil {
		return nil, err
	}
	return uc.Repo.Obter(ctx, d.ID)
}

// AoAutorizar: a NF-e de entrada da devolução foi autorizada na SEFAZ — marca
// emitida e efetiva. A nota já está na SEFAZ: falhas voltam como aviso para
// alguém concluir pelo botão "Efetivar" (repetir não duplica).
func (uc *UseCase) AoAutorizar(ctx context.Context, exit *entity.FiscalExit, userID uuid.UUID) []string {
	if exit.DevolucaoVendaID == nil {
		return nil
	}
	if err := uc.Repo.MarcarEmitida(ctx, *exit.DevolucaoVendaID); err != nil {
		return []string{"a NF-e de entrada foi autorizada, mas a devolução não foi marcada como emitida: " + err.Error()}
	}
	d, err := uc.efetivar(ctx, *exit.DevolucaoVendaID, userID)
	if err != nil {
		return []string{"a NF-e de entrada foi autorizada, mas a devolução não foi efetivada no estoque/financeiro: " + err.Error() + ` — use "Efetivar" na devolução`}
	}
	return d
}

// Efetivar (modo NF do cliente, ou retomada da emissão própria).
func (uc *UseCase) Efetivar(ctx context.Context, id int64) (*dom.Devolucao, []string, error) {
	por, err := uc.ator(ctx)
	if err != nil {
		return nil, nil, err
	}
	avisos, err := uc.efetivar(ctx, id, por)
	if err != nil {
		return nil, nil, err
	}
	d, err := uc.Repo.Obter(ctx, id)
	return d, avisos, err
}

func (uc *UseCase) efetivar(ctx context.Context, id int64, por uuid.UUID) ([]string, error) {
	d, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := dom.PodeEfetivar(d); err != nil {
		return nil, err
	}
	if d.Situacao == dom.Efetivada {
		return nil, nil
	}
	hoje := uc.agora()
	res, err := uc.Repo.Efetivar(ctx, dom.Efetivacao{Devolucao: d, Data: hoje, Por: por})
	if err != nil {
		return nil, err
	}
	avisos := []string{}
	if res.Abatido.IsPositive() {
		avisos = append(avisos, fmt.Sprintf("R$ %s abatidos dos títulos em aberto da NF %d", res.Abatido.StringFixed(2), d.NumeroNF))
	}
	if res.CreditoCliente.IsPositive() {
		avisos = append(avisos, fmt.Sprintf("R$ %s viraram crédito do cliente (adiantamento %d)", res.CreditoCliente.StringFixed(2), *res.AdiantamentoID))
	}
	if res.Comissao.IsNegative() {
		avisos = append(avisos, fmt.Sprintf("comissão estornada: R$ %s", res.Comissao.Neg().StringFixed(2)))
	}
	// Imposto: a devolução vira crédito na apuração do mês (estorna o débito da venda).
	if uc.FinancialRepo != nil {
		creditos := map[string]decimal.Decimal{}
		for _, it := range d.Itens {
			creditos["ICMS"] = creditos["ICMS"].Add(it.ICMS)
			creditos["IPI"] = creditos["IPI"].Add(it.IPI)
			creditos["PIS"] = creditos["PIS"].Add(it.PIS)
			creditos["COFINS"] = creditos["COFINS"].Add(it.COFINS)
		}
		competencia := hoje.Format("01/2006")
		for _, imposto := range []string{"ICMS", "IPI", "PIS", "COFINS"} {
			v := creditos[imposto].Round(2)
			if !v.IsPositive() {
				continue
			}
			ta := &financialEntity.TaxAssessment{Imposto: imposto, Competencia: competencia, Creditos: v, Debitos: decimal.Zero, Status: financialEntity.TaxStatusApurar}
			if err := uc.FinancialRepo.UpsertTaxAssessmentCredito(ctx, ta); err != nil {
				avisos = append(avisos, fmt.Sprintf("o crédito de %s da devolução não foi registrado na apuração de %s: %v", imposto, competencia, err))
			}
		}
	}
	return avisos, nil
}

func (uc *UseCase) Cancelar(ctx context.Context, id int64, motivo string) (*dom.Devolucao, error) {
	por, err := uc.ator(ctx)
	if err != nil {
		return nil, err
	}
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return nil, errorsuc.NewValidationError("informe o motivo do cancelamento")
	}
	if err := uc.Repo.Cancelar(ctx, id, motivo, &por); err != nil {
		return nil, err
	}
	return uc.Repo.Obter(ctx, id)
}

// PodeCancelarNF: a NF-e de entrada de uma devolução já efetivada não se
// cancela pela tela da NF-e (o estoque e o financeiro já mudaram).
func (uc *UseCase) PodeCancelarNF(ctx context.Context, exit *entity.FiscalExit) error {
	if exit.DevolucaoVendaID == nil {
		return nil
	}
	d, err := uc.Repo.Obter(ctx, *exit.DevolucaoVendaID)
	if err != nil {
		return err
	}
	if d.Situacao == dom.Efetivada {
		return errorsuc.NewValidationError(fmt.Sprintf("a NF-e é da devolução de venda %d, já efetivada no estoque e no financeiro: não se cancela", d.Numero))
	}
	return nil
}
