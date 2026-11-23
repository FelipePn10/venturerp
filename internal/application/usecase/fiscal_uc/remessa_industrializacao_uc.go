package fiscal_uc

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

// SourceRemessaIndustrializacao marca a NF-e de remessa da OS de terceiro
// (CFOP 5901/6901). Não é venda: não gera título, não baixa pedido e não mexe
// no estoque — a peça vai para o almoxarifado do terceiro pela própria OS.
const SourceRemessaIndustrializacao = "REMESSA_INDUSTRIALIZACAO"

// Tributação da remessa para industrialização por encomenda (regime normal):
// ICMS e IPI suspensos, PIS/COFINS fora do campo (outras saídas).
// shortcut: CST/CSOSN fixos; validar com a contadora de cada cliente e, se a UF
// exigir, mover para a configuração fiscal (como o texto do retorno do beneficiamento).
const (
	cstICMSRemessa        = "50"  // suspensão
	csosnRemessa          = "900" // Simples: outros
	cstIPIRemessa         = "55"  // saída com suspensão
	cstPISCOFINSRemessa   = "49"  // outras operações de saída
	naturezaRemessaPadrao = "Remessa para industrialização por encomenda"
)

// RemessaParaIndustrializar é o que a NF-e precisa da OS de terceiro.
type RemessaParaIndustrializar struct {
	OSID, OSCode, ItemCode int64
	Mask, Descricao, NCM   string
	Unidade                string
	Fornecedor             int64
	CustoMedio             decimal.Decimal
	PrevisaoRetorno        time.Time
}

// DestinatarioDaRemessa é o fornecedor que industrializa.
type DestinatarioDaRemessa struct {
	CNPJ, RazaoSocial, IE, UF                               string
	Logradouro, Numero, Complemento, Bairro, Municipio, CEP string
}

// LeitorDaRemessa lê a OS de terceiro e liga a nota ao movimento de remessa.
type LeitorDaRemessa interface {
	// RemessaDaOS confere também que o movimento (quando informado) é remessa da OS.
	RemessaDaOS(ctx context.Context, osID int64, movimento *int64) (*RemessaParaIndustrializar, error)
	FornecedorDestinatario(ctx context.Context, fornecedor int64) (*DestinatarioDaRemessa, error)
	LigarNotaAoMovimento(ctx context.Context, osID, movimentoID, exitID int64) error
}

// RemessaIndustrializacaoDTO pede a NF-e de remessa de uma OS de terceiro.
type RemessaIndustrializacaoDTO struct {
	// MovementID: a remessa da OS que a nota acoberta (opcional).
	MovementID *int64 `json:"movement_id,omitempty"`
	Quantity   string `json:"quantity"`
	// UnitValue: valor unitário da peça; vazio = custo médio do item.
	UnitValue        string `json:"unit_value,omitempty"`
	DataEmissao      string `json:"data_emissao"`
	Serie            string `json:"serie,omitempty"`
	NaturezaOperacao string `json:"natureza_operacao,omitempty"`
	// Tributação definida pela contadora; vazio = padrão (ICMS 50 ou CSOSN 900,
	// IPI 55, PIS/COFINS 49).
	CSTICMS      string `json:"cst_icms,omitempty"`
	CSTIPI       string `json:"cst_ipi,omitempty"`
	CSTPISCOFINS string `json:"cst_pis_cofins,omitempty"`
	// CBenef: código de benefício fiscal, na UF que o exige para a suspensão.
	CBenef string `json:"cbenef,omitempty"`
	// TextoLegal: o dispositivo da suspensão no RICMS da UF, nos dados adicionais.
	TextoLegal string `json:"texto_legal,omitempty"`
}

var soDigitosCST = regexp.MustCompile(`^[0-9]{2,3}$`)

// tributacaoDaRemessa: o padrão, sobreposto pelo que foi informado.
func tributacaoDaRemessa(dto RemessaIndustrializacaoDTO, simples bool) (icms, ipi, pis string, err error) {
	icms, ipi, pis = cstICMSRemessa, cstIPIRemessa, cstPISCOFINSRemessa
	if simples {
		icms = csosnRemessa
	}
	for _, c := range []struct {
		dst  *string
		v    string
		nome string
	}{{&icms, dto.CSTICMS, "CST/CSOSN do ICMS"}, {&ipi, dto.CSTIPI, "CST do IPI"}, {&pis, dto.CSTPISCOFINS, "CST do PIS/COFINS"}} {
		v := strings.TrimSpace(c.v)
		if v == "" {
			continue
		}
		if !soDigitosCST.MatchString(v) {
			return "", "", "", errorsuc.NewValidationError(fmt.Sprintf("%s inválido: %s", c.nome, v))
		}
		*c.dst = v
	}
	return icms, ipi, pis, nil
}

// RemessaIndustrializacaoUseCase gera a NF-e de remessa (rascunho) da OS de
// terceiro. Antes a remessa para a TRATE era só um registro, sem nota.
type RemessaIndustrializacaoUseCase struct {
	Repo   repository.FiscalRepository
	Auth   ports.AuthService
	Leitor LeitorDaRemessa
}

func (uc *RemessaIndustrializacaoUseCase) Criar(ctx context.Context, osID int64, dto RemessaIndustrializacaoDTO) (*response.FiscalExitResponse, error) {
	if !uc.Auth.CanCreateFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	qtd, err := decimal.NewFromString(strings.TrimSpace(dto.Quantity))
	if err != nil || !qtd.IsPositive() {
		return nil, errorsuc.NewValidationError("informe a quantidade remetida")
	}
	emissao, err := time.Parse("2006-01-02", strings.TrimSpace(dto.DataEmissao))
	if err != nil {
		return nil, errorsuc.NewValidationError("data de emissão inválida: use AAAA-MM-DD")
	}
	os, err := uc.Leitor.RemessaDaOS(ctx, osID, dto.MovementID)
	if err != nil {
		return nil, err
	}
	valor := os.CustoMedio
	if v := strings.TrimSpace(dto.UnitValue); v != "" {
		if valor, err = decimal.NewFromString(v); err != nil || valor.IsNegative() {
			return nil, errorsuc.NewValidationError("valor unitário inválido")
		}
	}
	if !valor.IsPositive() {
		return nil, errorsuc.NewValidationError("o item não tem custo médio: informe o valor unitário da remessa")
	}
	dest, err := uc.Leitor.FornecedorDestinatario(ctx, os.Fornecedor)
	if err != nil {
		return nil, err
	}
	cfg, err := uc.Repo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("lendo a configuração fiscal: %w", err)
	}
	cfop, err := cfopDaRemessa(cfg.UFEmpresa, dest.UF)
	if err != nil {
		return nil, err
	}
	total := qtd.Mul(valor).Round(2)
	opt := func(s string) *string {
		if s = strings.TrimSpace(s); s == "" {
			return nil
		}
		return &s
	}
	info := fmt.Sprintf("Remessa para industrialização por encomenda - OS de terceiro %d. Retorno previsto em %s.", os.OSCode, os.PrevisaoRetorno.Format("02/01/2006"))
	if t := strings.TrimSpace(dto.TextoLegal); t != "" {
		info += " " + t
	}
	semFrete := "SEM_FRETE"
	src := SourceRemessaIndustrializacao
	exit := &entity.FiscalExit{
		Serie: firstNonEmpty(strings.TrimSpace(dto.Serie), "1"), DataEmissao: emissao,
		CnpjDestinatario: opt(dest.CNPJ), RazaoSocialDestinatario: opt(dest.RazaoSocial), IEDestinatario: opt(dest.IE), UFDestinatario: opt(dest.UF),
		DestLogradouro: opt(dest.Logradouro), DestNumero: opt(dest.Numero), DestComplemento: opt(dest.Complemento),
		DestBairro: opt(dest.Bairro), DestMunicipio: opt(dest.Municipio), DestCEP: opt(dest.CEP),
		Cfop: cfop, NaturezaOperacao: firstNonEmpty(strings.TrimSpace(dto.NaturezaOperacao), naturezaRemessaPadrao),
		ValorProdutos: total, ValorTotal: total, Finalidade: 1, SupplierCode: &os.Fornecedor,
		InformacoesComplementares: &info, TipoFrete: &semFrete, SourceType: &src,
		Status: entity.ExitStatusDraft, CreatedBy: userID,
	}
	cst, cstIPI, cstPIS, err := tributacaoDaRemessa(dto, regimeSimples(cfg))
	if err != nil {
		return nil, err
	}
	if exit.NumeroNF, err = uc.Repo.GetNextNFNumber(ctx); err != nil {
		return nil, err
	}
	criada, err := uc.Repo.CreateExit(ctx, exit)
	if err != nil {
		return nil, err
	}
	item := os.ItemCode
	cod := fmt.Sprint(os.ItemCode)
	linha := &entity.FiscalExitItem{
		FiscalExitID: criada.ID, Sequence: 1, ItemCode: &item, Mask: os.Mask, Ncm: opt(os.NCM), Cfop: cfop,
		Quantity: qtd, UnitPrice: valor, TotalPrice: total,
		CstICMS: &cst, CstIPI: &cstIPI, CstPIS: &cstPIS, CstCOFINS: &cstPIS, OrigemMercadoria: "0",
		Description: opt(os.Descricao), UnidadeComercial: opt(os.Unidade), CodigoProduto: &cod,
		CodigoBeneficioFiscal: opt(dto.CBenef),
	}
	if _, err := uc.Repo.CreateExitItem(ctx, linha); err != nil {
		return nil, err
	}
	if dto.MovementID != nil {
		if err := uc.Leitor.LigarNotaAoMovimento(ctx, osID, *dto.MovementID, criada.ID); err != nil {
			return nil, err
		}
	}
	saida, err := uc.Repo.GetExitByID(ctx, criada.ID)
	if err != nil {
		return nil, err
	}
	return toFiscalExitResponse(saida), nil
}

// cfopDaRemessa: 5901 dentro da UF, 6901 para outra UF. Exterior não se aplica.
func cfopDaRemessa(ufEmpresa, ufDestino string) (string, error) {
	e, d := strings.ToUpper(strings.TrimSpace(ufEmpresa)), strings.ToUpper(strings.TrimSpace(ufDestino))
	switch {
	case e == "" || d == "":
		return "", errorsuc.NewValidationError("informe a UF da empresa (Configuração Fiscal) e a do fornecedor (endereço principal) para escolher o CFOP da remessa")
	case d == "EX":
		return "", errorsuc.NewValidationError("remessa para industrialização no exterior não é suportada")
	case e == d:
		return "5901", nil
	default:
		return "6901", nil
	}
}

func ehRemessaIndustrializacao(exit *entity.FiscalExit) bool {
	return exit != nil && exit.SourceType != nil && *exit.SourceType == SourceRemessaIndustrializacao
}
