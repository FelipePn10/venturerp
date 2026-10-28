package fiscal_uc

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/reforma"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

// ReformaTributariaUseCase mantém a parametrização da reforma tributária da
// empresa: alíquotas por ano, cClassTrib padrão por CFOP e o cClassTrib próprio
// de uma linha de nota. A tabela oficial de classificação só é consultada.
type ReformaTributariaUseCase struct {
	Repo repository.ReformaTributariaRepository
	Auth ports.AuthService
}

// ─── DTOs ─────────────────────────────────────────────────────────────────────

type ClassificacaoIBSCBSResponse struct {
	ClassTrib         string  `json:"cclass_trib"`
	CST               string  `json:"cst"`
	CSTDescricao      string  `json:"cst_descricao"`
	Descricao         string  `json:"descricao"`
	ExigeTributacao   bool    `json:"exige_tributacao"`
	ReducaoAliquota   bool    `json:"reducao_aliquota"`
	Diferimento       bool    `json:"diferimento"`
	PercReducaoIBS    float64 `json:"perc_reducao_ibs"`
	PercReducaoCBS    float64 `json:"perc_reducao_cbs"`
	TributacaoRegular bool    `json:"tributacao_regular"`
	AceitaNFe         bool    `json:"aceita_nfe"`
	VigenciaInicio    string  `json:"vigencia_inicio"`
	VigenciaFim       *string `json:"vigencia_fim,omitempty"`
	URLLegislacao     *string `json:"url_legislacao,omitempty"`
	VersaoTabela      string  `json:"versao_tabela"`
	VigenteHoje       bool    `json:"vigente_hoje"`
	// SuportadoEmissao: o emissor monta o grupo deste CST (ver reforma.CSTSuportadoNaEmissao).
	SuportadoEmissao bool `json:"suportado_emissao"`
}

// AliquotasIBSCBSResponse é o ano com as alíquotas que valem nele e de onde
// vieram: EMPRESA (cadastro), LEI (padrão da LC 214) ou PENDENTE (falta a CBS).
type AliquotasIBSCBSResponse struct {
	Ano        int      `json:"ano"`
	IBSUF      *float64 `json:"ibs_uf"`
	IBSMun     *float64 `json:"ibs_mun"`
	CBS        *float64 `json:"cbs"`
	Fonte      string   `json:"fonte"`
	Fundamento string   `json:"fundamento"`
	Pendencia  string   `json:"pendencia,omitempty"`
	UpdatedAt  *string  `json:"updated_at,omitempty"`
}

type SalvarAliquotasIBSCBSDTO struct {
	IBSUF      decimal.Decimal `json:"ibs_uf"`
	IBSMun     decimal.Decimal `json:"ibs_mun"`
	CBS        decimal.Decimal `json:"cbs"`
	Fundamento string          `json:"fundamento"`
}

type CFOPClassTribResponse struct {
	CFOP       string  `json:"cfop"`
	ClassTrib  string  `json:"cclass_trib"`
	Descricao  string  `json:"descricao"`
	Observacao *string `json:"observacao,omitempty"`
	UpdatedAt  string  `json:"updated_at"`
}

type SalvarCFOPClassTribDTO struct {
	ClassTrib  string  `json:"cclass_trib"`
	Observacao *string `json:"observacao,omitempty"`
}

type DefinirClassTribDaLinhaDTO struct {
	// ClassTrib nulo ou vazio volta a linha para o padrão (CFOP, classificação).
	ClassTrib *string `json:"cclass_trib"`
}

// ─── tabela oficial ───────────────────────────────────────────────────────────

func (uc *ReformaTributariaUseCase) ListarClassificacoes(ctx context.Context, cst, busca string, somenteNFe bool) ([]ClassificacaoIBSCBSResponse, error) {
	if !uc.Auth.CanListFiscalExits(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if cst != "" && !regexp.MustCompile(`^[0-9]{3}$`).MatchString(cst) {
		return nil, errorsuc.NewValidationError("o CST do IBS/CBS tem 3 dígitos")
	}
	linhas, err := uc.Repo.ListarClassificacoes(ctx, cst, busca, somenteNFe)
	if err != nil {
		return nil, err
	}
	hoje := time.Now()
	out := make([]ClassificacaoIBSCBSResponse, 0, len(linhas))
	for _, c := range linhas {
		out = append(out, classificacaoResponse(c, hoje))
	}
	return out, nil
}

func classificacaoResponse(c reforma.ClassificacaoOficial, hoje time.Time) ClassificacaoIBSCBSResponse {
	r := ClassificacaoIBSCBSResponse{
		ClassTrib: c.ClassTrib, CST: c.CST, CSTDescricao: c.CSTDescricao, Descricao: c.Descricao,
		ExigeTributacao: c.ExigeTributacao, ReducaoAliquota: c.ReducaoAliquota, Diferimento: c.Diferimento,
		PercReducaoIBS: c.PercReducaoIBS.InexactFloat64(), PercReducaoCBS: c.PercReducaoCBS.InexactFloat64(),
		TributacaoRegular: c.TributacaoRegular, AceitaNFe: c.AceitaNFe,
		VigenciaInicio: c.VigenciaInicio.Format("2006-01-02"), URLLegislacao: c.URLLegislacao, VersaoTabela: c.VersaoTabela,
	}
	if c.VigenciaFim != nil {
		f := c.VigenciaFim.Format("2006-01-02")
		r.VigenciaFim = &f
	}
	r.SuportadoEmissao = reforma.CSTSuportadoNaEmissao(c.CST)
	r.VigenteHoje = !hoje.Before(c.VigenciaInicio) && (c.VigenciaFim == nil || !hoje.After(c.VigenciaFim.Add(24*time.Hour)))
	return r
}

// classificacaoParaNFe confere que o código existe na tabela oficial e é
// admitido na NF-e (a vigência é conferida na emissão, contra a data da nota).
func (uc *ReformaTributariaUseCase) classificacaoParaNFe(ctx context.Context, codigo string) (string, error) {
	codigo = strings.TrimSpace(codigo)
	if !regexp.MustCompile(`^[0-9]{6}$`).MatchString(codigo) {
		return "", errorsuc.NewValidationError("o cClassTrib tem 6 dígitos (os 3 primeiros são o CST)")
	}
	tabela, err := uc.Repo.BuscarClassificacoes(ctx, []string{codigo})
	if err != nil {
		return "", err
	}
	c, ok := tabela[codigo]
	if !ok {
		return "", errorsuc.NewValidationError(fmt.Sprintf("o cClassTrib %s não existe na tabela oficial de classificação tributária", codigo))
	}
	if !c.AceitaNFe {
		return "", errorsuc.NewValidationError(fmt.Sprintf("o cClassTrib %s (%s) não é admitido na NF-e", codigo, c.Descricao))
	}
	if !reforma.CSTSuportadoNaEmissao(c.CST) {
		return "", errorsuc.NewValidationError(fmt.Sprintf(
			"o cClassTrib %s é do CST %s (%s), que exige grupos do IBS/CBS que a emissão ainda não preenche: use um de CST 000, 010, 011, 200, 400, 410 ou 550",
			codigo, c.CST, c.CSTDescricao))
	}
	return codigo, nil
}

// ValidarClassTribDaClassificacao é usado pelo cadastro de classificação fiscal.
func (uc *ReformaTributariaUseCase) ValidarClassTribDaClassificacao(ctx context.Context, codigo string) error {
	_, err := uc.classificacaoParaNFe(ctx, codigo)
	return err
}

// ─── alíquotas ────────────────────────────────────────────────────────────────

// anosExibidos: a transição inteira (2026 a 2033) mais os anos cadastrados.
const primeiroAnoReforma, ultimoAnoTransicao = 2026, 2033

func (uc *ReformaTributariaUseCase) ListarAliquotas(ctx context.Context) ([]AliquotasIBSCBSResponse, error) {
	if !uc.Auth.CanListFiscalExits(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	cadastradas, err := uc.Repo.ListarAliquotas(ctx)
	if err != nil {
		return nil, err
	}
	porAno := map[int]reforma.AliquotasCadastradas{}
	ultimo := ultimoAnoTransicao
	for _, c := range cadastradas {
		porAno[c.Ano] = c
		if c.Ano > ultimo {
			ultimo = c.Ano
		}
	}
	f := func(d decimal.Decimal) *float64 { v := d.InexactFloat64(); return &v }
	out := make([]AliquotasIBSCBSResponse, 0, ultimo-primeiroAnoReforma+1)
	for ano := primeiroAnoReforma; ano <= ultimo; ano++ {
		if c, ok := porAno[ano]; ok {
			em := c.UpdatedAt.Format(time.RFC3339)
			out = append(out, AliquotasIBSCBSResponse{Ano: ano, IBSUF: f(c.Aliquotas.IBSUF), IBSMun: f(c.Aliquotas.IBSMun), CBS: f(c.Aliquotas.CBS),
				Fonte: reforma.FonteEmpresa, Fundamento: c.Fundamento, UpdatedAt: &em})
			continue
		}
		linha := AliquotasIBSCBSResponse{Ano: ano, Fonte: "PENDENTE"}
		ibsUF, ibsMun, cbs, ok := reforma.AliquotasDaLei(ano)
		if ok {
			linha.IBSUF, linha.IBSMun = f(ibsUF), f(ibsMun)
			linha.Fundamento = "LC 214/2025, arts. 343, 344 e 346"
			if cbs != nil {
				linha.CBS, linha.Fonte = f(*cbs), reforma.FonteLei
			} else {
				linha.Pendencia = "a CBS do ano é a alíquota de referência fixada pela União menos 0,1 ponto (art. 347): cadastre-a antes da primeira nota do ano"
			}
		} else {
			linha.Pendencia = "alíquotas do ano ainda não cadastradas"
		}
		out = append(out, linha)
	}
	return out, nil
}

func (uc *ReformaTributariaUseCase) SalvarAliquotas(ctx context.Context, ano int, dto SalvarAliquotasIBSCBSDTO) error {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return errorsuc.ErrUnauthorized
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	a := reforma.Aliquotas{IBSUF: dto.IBSUF, IBSMun: dto.IBSMun, CBS: dto.CBS}
	fundamento := strings.TrimSpace(dto.Fundamento)
	if err := reforma.ValidarCadastro(ano, a, fundamento); err != nil {
		return err
	}
	return uc.Repo.SalvarAliquotas(ctx, ano, a, fundamento, actor)
}

func (uc *ReformaTributariaUseCase) ExcluirAliquotas(ctx context.Context, ano int) error {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.ExcluirAliquotas(ctx, ano)
}

// ─── CFOP ─────────────────────────────────────────────────────────────────────

func (uc *ReformaTributariaUseCase) ListarCFOP(ctx context.Context) ([]CFOPClassTribResponse, error) {
	if !uc.Auth.CanListFiscalExits(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	linhas, err := uc.Repo.ListarCFOP(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CFOPClassTribResponse, 0, len(linhas))
	for _, m := range linhas {
		out = append(out, CFOPClassTribResponse{CFOP: m.CFOP, ClassTrib: m.ClassTrib, Descricao: m.Descricao,
			Observacao: m.Observacao, UpdatedAt: m.UpdatedAt.Format(time.RFC3339)})
	}
	return out, nil
}

var cfopValido = regexp.MustCompile(`^[0-9]{4}$`)

func (uc *ReformaTributariaUseCase) SalvarCFOP(ctx context.Context, cfop string, dto SalvarCFOPClassTribDTO) error {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return errorsuc.ErrUnauthorized
	}
	cfop = strings.ReplaceAll(strings.TrimSpace(cfop), ".", "")
	if !cfopValido.MatchString(cfop) {
		return errorsuc.NewValidationError("o CFOP tem 4 dígitos")
	}
	if cfop[0] != '5' && cfop[0] != '6' && cfop[0] != '7' {
		return errorsuc.NewValidationError("o cClassTrib padrão por CFOP vale para a NF-e de saída: use CFOP iniciado em 5, 6 ou 7")
	}
	codigo, err := uc.classificacaoParaNFe(ctx, dto.ClassTrib)
	if err != nil {
		return err
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	var obs *string
	if dto.Observacao != nil && strings.TrimSpace(*dto.Observacao) != "" {
		o := strings.TrimSpace(*dto.Observacao)
		obs = &o
	}
	return uc.Repo.SalvarCFOP(ctx, cfop, codigo, obs, actor)
}

func (uc *ReformaTributariaUseCase) ExcluirCFOP(ctx context.Context, cfop string) error {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.ExcluirCFOP(ctx, strings.ReplaceAll(strings.TrimSpace(cfop), ".", ""))
}

// ─── linha da nota ────────────────────────────────────────────────────────────

func (uc *ReformaTributariaUseCase) DefinirClassTribDaLinha(ctx context.Context, exitID, itemID int64, dto DefinirClassTribDaLinhaDTO) error {
	if !uc.Auth.CanCreateFiscalExit(ctx) {
		return errorsuc.ErrUnauthorized
	}
	var codigo *string
	if dto.ClassTrib != nil && strings.TrimSpace(*dto.ClassTrib) != "" {
		c, err := uc.classificacaoParaNFe(ctx, *dto.ClassTrib)
		if err != nil {
			return err
		}
		codigo = &c
	}
	return uc.Repo.DefinirClassTribDoItem(ctx, exitID, itemID, codigo)
}

// ─── modo de convivência ──────────────────────────────────────────────────────

type ModoReformaResponse struct {
	Modo      string  `json:"modo"`
	Motivo    *string `json:"motivo,omitempty"`
	UpdatedAt *string `json:"updated_at,omitempty"`
}

type SalvarModoReformaDTO struct {
	Modo   string  `json:"modo"`
	Motivo *string `json:"motivo,omitempty"`
}

func (uc *ReformaTributariaUseCase) Modo(ctx context.Context) (*ModoReformaResponse, error) {
	if !uc.Auth.CanListFiscalExits(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	p, err := uc.Repo.Parametros(ctx)
	if err != nil {
		return nil, err
	}
	r := &ModoReformaResponse{Modo: string(p.Modo), Motivo: p.Motivo}
	if p.UpdatedAt != nil {
		em := p.UpdatedAt.Format(time.RFC3339)
		r.UpdatedAt = &em
	}
	return r, nil
}

// SalvarModo: desligar o grupo IBS/CBS exige motivo (fica registrado quem e por
// quê); voltar ao automático não.
func (uc *ReformaTributariaUseCase) SalvarModo(ctx context.Context, dto SalvarModoReformaDTO) error {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return errorsuc.ErrUnauthorized
	}
	modo := reforma.Modo(strings.ToUpper(strings.TrimSpace(dto.Modo)))
	if modo != reforma.ModoAutomatico && modo != reforma.ModoSomenteAtual {
		return errorsuc.NewValidationError("modo inválido: use AUTOMATICO ou SOMENTE_ATUAL")
	}
	var motivo *string
	if dto.Motivo != nil && strings.TrimSpace(*dto.Motivo) != "" {
		m := strings.TrimSpace(*dto.Motivo)
		motivo = &m
	}
	if modo == reforma.ModoSomenteAtual && motivo == nil {
		return errorsuc.NewValidationError("informe o motivo para emitir só com o modelo atual (ex.: rejeição da SEFAZ no grupo IBS/CBS)")
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	return uc.Repo.SalvarParametros(ctx, reforma.Parametros{Modo: modo, Motivo: motivo}, actor)
}
