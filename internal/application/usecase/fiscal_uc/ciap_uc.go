package fiscal_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/ciap"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/sped"
)

// CIAPUseCase: o Controle de Crédito do ICMS do Ativo Permanente (VFIS0345).
// Os bens das notas de entrada aprovadas entram sozinhos; a apuração do mês
// calcula as parcelas, o índice das saídas e o crédito, que vai para a EFD
// (Bloco G e E111 da apuração do ICMS).
type CIAPUseCase struct {
	Repo   ciap.Repository
	Fiscal repository.FiscalRepository
	Auth   ports.AuthService
}

type CIAPParametrosDTO struct {
	CodigoAjuste   string `json:"codigo_ajuste"`
	ContaAtivoID   *int64 `json:"conta_ativo_id"`
	NumeroParcelas int    `json:"numero_parcelas"`
}

type CIAPBemDTO struct {
	Codigo         string `json:"codigo"`
	Descricao      string `json:"descricao"`
	Funcao         string `json:"funcao"`
	VidaUtilMeses  int    `json:"vida_util_meses"`
	ContaAtivoID   *int64 `json:"conta_ativo_id"`
	CostCenterID   *int64 `json:"cost_center_id"`
	DataEntrada    string `json:"data_entrada"`
	ICMSOp         string `json:"icms_op"`
	ICMSST         string `json:"icms_st"`
	ICMSFrete      string `json:"icms_frete"`
	ICMSDifal      string `json:"icms_difal"`
	NumeroParcelas int    `json:"numero_parcelas"`
}

type CIAPBaixaDTO struct {
	Tipo   string `json:"tipo"`
	Data   string `json:"data"`
	Motivo string `json:"motivo"`
}

// CIAPApurarDTO: as saídas do mês (texto decimal). Vazias = as calculadas das
// NF-e de saída autorizadas.
type CIAPApurarDTO struct {
	Ano           int    `json:"ano"`
	Mes           int    `json:"mes"`
	SaidasTribExp string `json:"saidas_trib_exp"`
	SaidasTotal   string `json:"saidas_total"`
}

type CIAPBemResponse struct {
	ID             int64           `json:"id"`
	Codigo         string          `json:"codigo"`
	Descricao      string          `json:"descricao"`
	Funcao         string          `json:"funcao,omitempty"`
	VidaUtilMeses  int             `json:"vida_util_meses,omitempty"`
	ContaAtivoID   *int64          `json:"conta_ativo_id,omitempty"`
	Conta          string          `json:"conta,omitempty"`
	CostCenterID   *int64          `json:"cost_center_id,omitempty"`
	Centro         string          `json:"centro,omitempty"`
	Origem         string          `json:"origem"`
	Documento      string          `json:"documento,omitempty"`
	DataEntrada    string          `json:"data_entrada"`
	ICMSOp         decimal.Decimal `json:"icms_op"`
	ICMSST         decimal.Decimal `json:"icms_st"`
	ICMSFrete      decimal.Decimal `json:"icms_frete"`
	ICMSDifal      decimal.Decimal `json:"icms_difal"`
	Total          decimal.Decimal `json:"total"`
	ValorParcela   decimal.Decimal `json:"valor_parcela"`
	NumeroParcelas int             `json:"numero_parcelas"`
	ParcelasUsadas int             `json:"parcelas_usadas"`
	Situacao       string          `json:"situacao"`
	DataBaixa      *string         `json:"data_baixa,omitempty"`
	TipoBaixa      string          `json:"tipo_baixa,omitempty"`
}

type CIAPMovimentoResponse struct {
	BemID         int64           `json:"bem_id"`
	Codigo        string          `json:"codigo"`
	Descricao     string          `json:"descricao"`
	TipoMov       string          `json:"tipo_mov"`
	Data          string          `json:"data"`
	NumeroParcela int             `json:"numero_parcela,omitempty"`
	ValorParcela  decimal.Decimal `json:"valor_parcela"`
}

type CIAPApuracaoResponse struct {
	Ano              int                     `json:"ano"`
	Mes              int                     `json:"mes"`
	SaldoInicial     decimal.Decimal         `json:"saldo_inicial"`
	SomaParcelas     decimal.Decimal         `json:"soma_parcelas"`
	SaidasTribExp    decimal.Decimal         `json:"saidas_trib_exp"`
	SaidasTotal      decimal.Decimal         `json:"saidas_total"`
	Indice           decimal.Decimal         `json:"indice"`
	ICMSApropriado   decimal.Decimal         `json:"icms_apropriado"`
	SaidasInformadas bool                    `json:"saidas_informadas"`
	Gravada          bool                    `json:"gravada"`
	Fechada          bool                    `json:"fechada"`
	ApuradoEm        *time.Time              `json:"apurado_em,omitempty"`
	Movimentos       []CIAPMovimentoResponse `json:"movimentos,omitempty"`
	// Sugerido: as saídas calculadas das NF-e, para comparar com as informadas.
	SugeridoTribExp decimal.Decimal `json:"sugerido_trib_exp"`
	SugeridoTotal   decimal.Decimal `json:"sugerido_total"`
}

func (uc *CIAPUseCase) podeAlterar(ctx context.Context) error {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return nil
}

// regimeComCredito: o Simples Nacional não toma crédito de ICMS (nem CIAP).
func (uc *CIAPUseCase) regimeComCredito(ctx context.Context) (*string, error) {
	if uc.Fiscal == nil {
		return nil, nil
	}
	cfg, err := uc.Fiscal.GetFiscalConfig(ctx)
	if err != nil {
		if errorsuc.EhNaoEncontrado(err) {
			return nil, nil
		}
		return nil, err
	}
	if regimeSimples(cfg) {
		return nil, errorsuc.NewValidationError("o optante do Simples Nacional não toma crédito de ICMS: não há CIAP")
	}
	uf := strings.ToUpper(strings.TrimSpace(cfg.UFEmpresa))
	return &uf, nil
}

func (uc *CIAPUseCase) Parametros(ctx context.Context) (ciap.Parametros, error) {
	return uc.Repo.Parametros(ctx)
}

func (uc *CIAPUseCase) SalvarParametros(ctx context.Context, dto CIAPParametrosDTO) (ciap.Parametros, error) {
	if err := uc.podeAlterar(ctx); err != nil {
		return ciap.Parametros{}, err
	}
	por, err := uc.Auth.UserID(ctx)
	if err != nil {
		return ciap.Parametros{}, err
	}
	uf, err := uc.regimeComCredito(ctx)
	if err != nil {
		return ciap.Parametros{}, err
	}
	cod := strings.ToUpper(strings.TrimSpace(dto.CodigoAjuste))
	if cod != "" {
		// O crédito do CIAP é "outros créditos" (3º/4º caractere = 02) da
		// tabela 5.1.1 da UF da empresa.
		if t, ok := sped.TipoAjuste(cod); !ok || t != '2' {
			return ciap.Parametros{}, errorsuc.NewValidationError(fmt.Sprintf("o código %q não é de outros créditos da apuração do ICMS (tabela 5.1.1, 8 caracteres, 4º = 2): confira na VFIS0510", cod))
		}
		if uf != nil && *uf != "" && cod[:2] != *uf {
			return ciap.Parametros{}, errorsuc.NewValidationError(fmt.Sprintf("o código %s é de outra UF: a empresa é do %s", cod, *uf))
		}
	}
	n := dto.NumeroParcelas
	if n == 0 {
		n = 48
	}
	if n < 1 || n > 999 {
		return ciap.Parametros{}, errorsuc.NewValidationError("número de parcelas de 1 a 999 (48 pela LC 87/1996)")
	}
	p := ciap.Parametros{CodigoAjuste: cod, ContaAtivoID: dto.ContaAtivoID, NumeroParcelas: n}
	if err := uc.Repo.SalvarParametros(ctx, p, por); err != nil {
		return ciap.Parametros{}, err
	}
	return uc.Repo.Parametros(ctx)
}

func diaISOPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.DateOnly)
	return &s
}

func toCIAPBem(b ciap.BemCadastrado) CIAPBemResponse {
	situacao := "ATIVO"
	if b.Baixado {
		situacao = "BAIXADO"
	} else if b.ParcelasUsadas >= b.NumeroParcelas {
		situacao = "ENCERRADO"
	}
	r := CIAPBemResponse{ID: b.ID, Codigo: b.Codigo, Descricao: b.Descricao, Funcao: b.Funcao, VidaUtilMeses: b.VidaUtilMeses,
		ContaAtivoID: b.ContaAtivoID, CostCenterID: b.CostCenterID, Origem: b.Origem, DataEntrada: b.DataEntrada.Format(time.DateOnly),
		ICMSOp: b.ICMSOp, ICMSST: b.ICMSST, ICMSFrete: b.ICMSFrete, ICMSDifal: b.ICMSDifal, Total: b.Total(), ValorParcela: b.ValorParcela(),
		NumeroParcelas: b.NumeroParcelas, ParcelasUsadas: b.ParcelasUsadas, Situacao: situacao, DataBaixa: diaISOPtr(b.DataBaixa), TipoBaixa: b.TipoBaixa}
	if b.ContaCodigo != "" {
		r.Conta = b.ContaCodigo + " — " + b.ContaNome
	}
	if b.CentroCodigo != "" {
		r.Centro = b.CentroCodigo + " — " + b.CentroNome
	}
	if d := b.Documento; d != nil {
		r.Documento = fmt.Sprintf("NF %s/%s, item %d", d.Numero, d.Serie, d.NumItem)
	}
	return r
}

func (uc *CIAPUseCase) Bens(ctx context.Context) ([]CIAPBemResponse, error) {
	if err := uc.Repo.Sincronizar(ctx); err != nil {
		return nil, err
	}
	bens, err := uc.Repo.Bens(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CIAPBemResponse, 0, len(bens))
	for _, b := range bens {
		out = append(out, toCIAPBem(b))
	}
	return out, nil
}

func lerValor(nome, s string) (decimal.Decimal, error) {
	t := strings.TrimSpace(s)
	if t == "" {
		return decimal.Zero, nil
	}
	if strings.Contains(t, ",") {
		t = strings.ReplaceAll(strings.ReplaceAll(t, ".", ""), ",", ".")
	}
	v, err := decimal.NewFromString(t)
	if err != nil {
		return decimal.Zero, errorsuc.NewValidationError(fmt.Sprintf("%s inválido: %q", nome, s))
	}
	return v, nil
}

func (uc *CIAPUseCase) montarBem(ctx context.Context, dto CIAPBemDTO) (ciap.BemManual, error) {
	var b ciap.BemManual
	b.Codigo, b.Descricao, b.Funcao, b.VidaUtilMeses = dto.Codigo, dto.Descricao, strings.TrimSpace(dto.Funcao), dto.VidaUtilMeses
	b.ContaAtivoID, b.CostCenterID = dto.ContaAtivoID, dto.CostCenterID
	if dto.DataEntrada != "" {
		d, err := time.Parse(time.DateOnly, dto.DataEntrada)
		if err != nil {
			return b, errorsuc.NewValidationError("data de entrada inválida (AAAA-MM-DD)")
		}
		b.DataEntrada = d
	}
	var err error
	if b.ICMSOp, err = lerValor("ICMS da operação", dto.ICMSOp); err != nil {
		return b, err
	}
	if b.ICMSST, err = lerValor("ICMS-ST", dto.ICMSST); err != nil {
		return b, err
	}
	if b.ICMSFrete, err = lerValor("ICMS do frete", dto.ICMSFrete); err != nil {
		return b, err
	}
	if b.ICMSDifal, err = lerValor("ICMS do diferencial de alíquota", dto.ICMSDifal); err != nil {
		return b, err
	}
	b.NumeroParcelas = dto.NumeroParcelas
	if b.NumeroParcelas == 0 {
		p, err := uc.Repo.Parametros(ctx)
		if err != nil {
			return b, err
		}
		b.NumeroParcelas = p.NumeroParcelas
	}
	if dto.VidaUtilMeses < 0 || dto.VidaUtilMeses > 999 {
		return b, errorsuc.NewValidationError("vida útil em meses, de 0 a 999")
	}
	return b, nil
}

// CriarBem: o bem anterior ao sistema (ou sem nota no sistema) entra à mão.
func (uc *CIAPUseCase) CriarBem(ctx context.Context, dto CIAPBemDTO) (*CIAPBemResponse, error) {
	if err := uc.podeAlterar(ctx); err != nil {
		return nil, err
	}
	if _, err := uc.regimeComCredito(ctx); err != nil {
		return nil, err
	}
	por, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	b, err := uc.montarBem(ctx, dto)
	if err != nil {
		return nil, err
	}
	if err := b.Conferir(); err != nil {
		return nil, err
	}
	id, err := uc.Repo.CriarBem(ctx, b, por)
	if err != nil {
		return nil, err
	}
	c, err := uc.Repo.Bem(ctx, id)
	if err != nil {
		return nil, err
	}
	out := toCIAPBem(*c)
	return &out, nil
}

// AtualizarBem: identificação, conta, centro, função e vida útil sempre; os
// valores, a entrada e as parcelas só no bem manual sem parcela apropriada.
func (uc *CIAPUseCase) AtualizarBem(ctx context.Context, id int64, dto CIAPBemDTO) (*CIAPBemResponse, error) {
	if err := uc.podeAlterar(ctx); err != nil {
		return nil, err
	}
	atual, err := uc.Repo.Bem(ctx, id)
	if err != nil {
		return nil, err
	}
	b, err := uc.montarBem(ctx, dto)
	if err != nil {
		return nil, err
	}
	b.ID = id
	valores := atual.Origem == "MANUAL" && atual.ParcelasUsadas == 0
	if !valores {
		b.DataEntrada, b.ICMSOp, b.ICMSST, b.ICMSFrete, b.ICMSDifal, b.NumeroParcelas =
			atual.DataEntrada, atual.ICMSOp, atual.ICMSST, atual.ICMSFrete, atual.ICMSDifal, atual.NumeroParcelas
	}
	if err := b.Conferir(); err != nil {
		return nil, err
	}
	if err := uc.Repo.AtualizarBem(ctx, b, valores); err != nil {
		return nil, err
	}
	c, err := uc.Repo.Bem(ctx, id)
	if err != nil {
		return nil, err
	}
	out := toCIAPBem(*c)
	return &out, nil
}

func (uc *CIAPUseCase) Baixar(ctx context.Context, id int64, dto CIAPBaixaDTO) error {
	if err := uc.podeAlterar(ctx); err != nil {
		return err
	}
	tipo := strings.ToUpper(strings.TrimSpace(dto.Tipo))
	if !ciap.TipoBaixaValido(tipo) {
		return errorsuc.NewValidationError("tipo da baixa: AT (alienação ou transferência), PE (perecimento, extravio ou deterioração) ou OT (outras saídas)")
	}
	d, err := time.Parse(time.DateOnly, dto.Data)
	if err != nil {
		return errorsuc.NewValidationError("data da baixa inválida (AAAA-MM-DD)")
	}
	if strings.TrimSpace(dto.Motivo) == "" {
		return errorsuc.NewValidationError("informe o motivo da baixa")
	}
	if a, err := uc.Repo.Apuracao(ctx, d.Year(), int(d.Month())); err != nil {
		return err
	} else if a != nil && a.Fechada {
		return errorsuc.NewConflictError(fmt.Sprintf("a apuração do CIAP de %02d/%d está fechada: a baixa nesse mês muda o crédito", d.Month(), d.Year()))
	}
	return uc.Repo.Baixar(ctx, id, tipo, d, strings.TrimSpace(dto.Motivo))
}

func (uc *CIAPUseCase) ExcluirBem(ctx context.Context, id int64) error {
	if err := uc.podeAlterar(ctx); err != nil {
		return err
	}
	return uc.Repo.ExcluirBem(ctx, id)
}

// calcular monta o mês com as saídas informadas (ou as calculadas).
func (uc *CIAPUseCase) calcular(ctx context.Context, dto CIAPApurarDTO) (*ciap.Apuracao, bool, decimal.Decimal, decimal.Decimal, error) {
	if _, err := uc.regimeComCredito(ctx); err != nil {
		return nil, false, decimal.Zero, decimal.Zero, err
	}
	if err := uc.Repo.Sincronizar(ctx); err != nil {
		return nil, false, decimal.Zero, decimal.Zero, err
	}
	sugTrib, sugTotal, err := uc.Repo.Saidas(ctx, dto.Ano, dto.Mes)
	if err != nil {
		return nil, false, decimal.Zero, decimal.Zero, err
	}
	trib, total, informadas := sugTrib, sugTotal, false
	if strings.TrimSpace(dto.SaidasTribExp) != "" || strings.TrimSpace(dto.SaidasTotal) != "" {
		if trib, err = lerValor("saídas tributadas e de exportação", dto.SaidasTribExp); err != nil {
			return nil, false, decimal.Zero, decimal.Zero, err
		}
		if total, err = lerValor("total de saídas", dto.SaidasTotal); err != nil {
			return nil, false, decimal.Zero, decimal.Zero, err
		}
		informadas = true
	}
	cads, err := uc.Repo.Bens(ctx)
	if err != nil {
		return nil, false, decimal.Zero, decimal.Zero, err
	}
	bens := make([]ciap.Bem, 0, len(cads))
	for _, b := range cads {
		bens = append(bens, b.Bem)
	}
	a, err := ciap.Apurar(bens, dto.Ano, dto.Mes, trib, total)
	return a, informadas, sugTrib, sugTotal, err
}

func toCIAPApuracao(a ciap.Apuracao, informadas bool, sugTrib, sugTotal decimal.Decimal) *CIAPApuracaoResponse {
	r := &CIAPApuracaoResponse{Ano: a.Ano, Mes: a.Mes, SaldoInicial: a.SaldoInicial, SomaParcelas: a.SomaParcelas, SaidasTribExp: a.SaidasTribExp,
		SaidasTotal: a.SaidasTotal, Indice: a.Indice, ICMSApropriado: a.ICMSApropriado, SaidasInformadas: informadas,
		SugeridoTribExp: sugTrib, SugeridoTotal: sugTotal}
	for _, m := range a.Movimentos {
		r.Movimentos = append(r.Movimentos, CIAPMovimentoResponse{BemID: m.Bem.ID, Codigo: m.Bem.Codigo, Descricao: m.Bem.Descricao,
			TipoMov: m.TipoMov, Data: m.Data.Format(time.DateOnly), NumeroParcela: m.NumeroParcela, ValorParcela: m.ValorParcela})
	}
	return r
}

// Previa: o mês calculado, sem gravar — com a apuração gravada, se houver.
func (uc *CIAPUseCase) Previa(ctx context.Context, dto CIAPApurarDTO) (*CIAPApuracaoResponse, error) {
	if g, err := uc.Repo.Apuracao(ctx, dto.Ano, dto.Mes); err != nil {
		return nil, err
	} else if g != nil && strings.TrimSpace(dto.SaidasTribExp) == "" && strings.TrimSpace(dto.SaidasTotal) == "" {
		sugTrib, sugTotal, err := uc.Repo.Saidas(ctx, dto.Ano, dto.Mes)
		if err != nil {
			return nil, err
		}
		r := toCIAPApuracao(g.Apuracao, g.SaidasInformadas, sugTrib, sugTotal)
		r.Gravada, r.Fechada = true, g.Fechada
		em := g.ApuradoEm
		r.ApuradoEm = &em
		return r, nil
	}
	a, informadas, sugTrib, sugTotal, err := uc.calcular(ctx, dto)
	if err != nil {
		return nil, err
	}
	return toCIAPApuracao(*a, informadas, sugTrib, sugTotal), nil
}

// Apurar grava o mês (refaz enquanto não fechado).
func (uc *CIAPUseCase) Apurar(ctx context.Context, dto CIAPApurarDTO) (*CIAPApuracaoResponse, error) {
	if err := uc.podeAlterar(ctx); err != nil {
		return nil, err
	}
	por, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	a, informadas, _, _, err := uc.calcular(ctx, dto)
	if err != nil {
		return nil, err
	}
	if err := uc.Repo.GravarApuracao(ctx, a, informadas, por); err != nil {
		return nil, err
	}
	return uc.Previa(ctx, CIAPApurarDTO{Ano: dto.Ano, Mes: dto.Mes})
}

func (uc *CIAPUseCase) Fechar(ctx context.Context, ano, mes int, fechar bool) error {
	if err := uc.podeAlterar(ctx); err != nil {
		return err
	}
	return uc.Repo.Fechar(ctx, ano, mes, fechar)
}

func (uc *CIAPUseCase) Apuracoes(ctx context.Context, ano int) ([]CIAPApuracaoResponse, error) {
	as, err := uc.Repo.ApuracoesDoAno(ctx, ano)
	if err != nil {
		return nil, err
	}
	out := make([]CIAPApuracaoResponse, 0, len(as))
	for _, a := range as {
		r := toCIAPApuracao(a.Apuracao, a.SaidasInformadas, decimal.Zero, decimal.Zero)
		r.Gravada, r.Fechada = true, a.Fechada
		em := a.ApuradoEm
		r.ApuradoEm = &em
		out = append(out, *r)
	}
	return out, nil
}
