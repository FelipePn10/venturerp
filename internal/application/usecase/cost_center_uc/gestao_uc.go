package cost_center_uc

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/gestao"
)

// GestaoUseCase: o centro de custo único como instrumento de gestão — código da
// empresa, responsável, aprovador, orçamento × realizado, árvore consolidada e
// aprovação das requisições de compra pelo centro.
type GestaoUseCase struct {
	Repo gestao.Repository
	Auth ports.AuthService
}

// GestaoDTO: o que a tela altera na gestão do centro.
type GestaoDTO struct {
	Codigo            string     `json:"codigo"`
	ResponsavelID     *uuid.UUID `json:"responsavel_id"`
	AprovadorID       *uuid.UUID `json:"aprovador_id"`
	AprovaRequisicoes bool       `json:"aprova_requisicoes"`
	ContaPadraoID     *int64     `json:"conta_padrao_id"`
}

// OrcamentoLinhaDTO: um mês de um centro (valor em texto, decimal).
type OrcamentoLinhaDTO struct {
	CostCenterID int64  `json:"cost_center_id"`
	Mes          int    `json:"mes"`
	Valor        string `json:"valor"`
}

type OrcamentoDTO struct {
	Ano    int                 `json:"ano"`
	Linhas []OrcamentoLinhaDTO `json:"linhas"`
}

func dia(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.DateOnly)
	return &s
}

func toCentroResponse(c gestao.Centro) response.CentroDeCustoResponse {
	return response.CentroDeCustoResponse{
		ID: c.ID, Code: c.Code, Codigo: c.Codigo, Descricao: c.Descricao, ParentCode: c.ParentCode, Tipo: c.Tipo,
		IsRatio: c.IsRatio, StartDate: c.StartDate.Format(time.DateOnly), EndDate: dia(c.EndDate), IsActive: c.IsActive,
		ResponsavelID: c.Gestao.ResponsavelID, Responsavel: c.Responsavel, AprovadorID: c.Gestao.AprovadorID, Aprovador: c.Aprovador,
		AprovaRequisicoes: c.Gestao.AprovaRequisicoes, ContaPadraoID: c.Gestao.ContaPadraoID,
	}
}

func (uc *GestaoUseCase) Centros(ctx context.Context) ([]response.CentroDeCustoResponse, error) {
	if !uc.Auth.CanListCostCenter(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	cs, err := uc.Repo.Centros(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]response.CentroDeCustoResponse, 0, len(cs))
	for _, c := range cs {
		out = append(out, toCentroResponse(c))
	}
	return out, nil
}

func (uc *GestaoUseCase) SalvarGestao(ctx context.Context, code int32, dto GestaoDTO) (*response.CentroDeCustoResponse, error) {
	if !uc.Auth.CanEditCostCenter(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	por, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	c, err := uc.Repo.Centro(ctx, code)
	if err != nil {
		return nil, err
	}
	g := gestao.Gestao{CostCenterID: c.ID, Codigo: dto.Codigo, ResponsavelID: dto.ResponsavelID, AprovadorID: dto.AprovadorID,
		AprovaRequisicoes: dto.AprovaRequisicoes, ContaPadraoID: dto.ContaPadraoID}
	if err := g.Conferir(); err != nil {
		return nil, err
	}
	if err := uc.Repo.SalvarGestao(ctx, g, por); err != nil {
		return nil, err
	}
	atual, err := uc.Repo.Centro(ctx, code)
	if err != nil {
		return nil, err
	}
	out := toCentroResponse(*atual)
	return &out, nil
}

// Orcamento: os 12 meses de cada centro, orçado e realizado.
func (uc *GestaoUseCase) Orcamento(ctx context.Context, ano int) ([]response.OrcamentoCentroResponse, error) {
	if !uc.Auth.CanListCostCenter(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if ano < 2000 || ano > 2100 {
		return nil, errorsuc.NewValidationError("informe o ano do orçamento")
	}
	cs, err := uc.Repo.Centros(ctx)
	if err != nil {
		return nil, err
	}
	orc, err := uc.Repo.Orcamentos(ctx, ano)
	if err != nil {
		return nil, err
	}
	linhas := make(map[int64]*response.OrcamentoCentroResponse, len(cs))
	out := make([]response.OrcamentoCentroResponse, 0, len(cs))
	for _, c := range cs {
		out = append(out, response.OrcamentoCentroResponse{CostCenterID: c.ID, Code: c.Code, Codigo: c.Codigo, Descricao: c.Descricao,
			Orcado: zeros(), Realizado: zeros()})
	}
	for i := range out {
		linhas[out[i].CostCenterID] = &out[i]
	}
	for _, o := range orc {
		if l, ok := linhas[o.CostCenterID]; ok && o.Mes >= 1 && o.Mes <= 12 {
			l.Orcado[o.Mes-1] = o.Valor
		}
	}
	for mes := 1; mes <= 12; mes++ {
		real, err := uc.Repo.Realizado(ctx, ano, mes)
		if err != nil {
			return nil, err
		}
		for id, v := range real {
			if l, ok := linhas[id]; ok {
				l.Realizado[mes-1] = v
			}
		}
	}
	return out, nil
}

func zeros() []decimal.Decimal {
	z := make([]decimal.Decimal, 12)
	for i := range z {
		z[i] = decimal.Zero
	}
	return z
}

func (uc *GestaoUseCase) SalvarOrcamento(ctx context.Context, dto OrcamentoDTO) error {
	if !uc.Auth.CanEditCostCenter(ctx) {
		return errorsuc.ErrUnauthorized
	}
	por, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	if len(dto.Linhas) == 0 {
		return errorsuc.NewValidationError("informe os meses do orçamento")
	}
	os := make([]gestao.Orcamento, 0, len(dto.Linhas))
	for _, l := range dto.Linhas {
		txt := strings.TrimSpace(l.Valor)
		if txt == "" {
			txt = "0"
		}
		v, err := decimal.NewFromString(txt)
		if err != nil {
			return errorsuc.NewValidationError("valor do orçamento inválido: " + l.Valor)
		}
		o := gestao.Orcamento{CostCenterID: l.CostCenterID, Ano: dto.Ano, Mes: l.Mes, Valor: v}
		if err := gestao.ConferirOrcamento(o); err != nil {
			return err
		}
		os = append(os, o)
	}
	return uc.Repo.SalvarOrcamentos(ctx, os, por)
}

// Arvore: a hierarquia com orçado × realizado próprios e consolidados; mes 0 =
// o ano todo.
func (uc *GestaoUseCase) Arvore(ctx context.Context, ano, mes int) ([]response.NoCentroResponse, error) {
	if !uc.Auth.CanListCostCenter(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if ano < 2000 || ano > 2100 || mes < 0 || mes > 12 {
		return nil, errorsuc.NewValidationError("informe o ano e o mês (0 = ano todo)")
	}
	cs, err := uc.Repo.Centros(ctx)
	if err != nil {
		return nil, err
	}
	orcs, err := uc.Repo.Orcamentos(ctx, ano)
	if err != nil {
		return nil, err
	}
	orcado := map[int64]decimal.Decimal{}
	for _, o := range orcs {
		if mes == 0 || o.Mes == mes {
			orcado[o.CostCenterID] = orcado[o.CostCenterID].Add(o.Valor)
		}
	}
	real, err := uc.Repo.Realizado(ctx, ano, mes)
	if err != nil {
		return nil, err
	}
	return toNos(gestao.MontarArvore(cs, orcado, real)), nil
}

func toNos(ns []*gestao.No) []response.NoCentroResponse {
	out := make([]response.NoCentroResponse, 0, len(ns))
	for _, n := range ns {
		out = append(out, response.NoCentroResponse{ID: n.ID, Code: n.Code, Codigo: n.Codigo, Descricao: n.Descricao, Tipo: n.Tipo,
			IsActive: n.IsActive, Responsavel: n.Responsavel, Orcado: n.Orcado, Realizado: n.Realizado,
			OrcadoConsolidado: n.OrcadoConsolidado, RealizadoConsolidado: n.RealizadoConsolidado, Estourado: n.Estourado(), Filhos: toNos(n.Filhos)})
	}
	return out
}

// Aprovacoes: minhas = só as que o usuário decide.
func (uc *GestaoUseCase) Aprovacoes(ctx context.Context, status string, minhas bool, requisicao *int64) ([]response.AprovacaoRequisicaoResponse, error) {
	if !uc.Auth.CanListCostCenter(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	eu, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	var filtro *uuid.UUID
	if minhas {
		filtro = &eu
	}
	as, err := uc.Repo.Aprovacoes(ctx, status, filtro, requisicao)
	if err != nil {
		return nil, err
	}
	out := make([]response.AprovacaoRequisicaoResponse, 0, len(as))
	for _, a := range as {
		pode := a.Status == gestao.AprovacaoPendente && a.AprovadorID == eu && (a.Solicitante == nil || *a.Solicitante != eu)
		out = append(out, response.AprovacaoRequisicaoResponse{ID: a.ID, RequisitionCode: a.RequisitionCode, CostCenterID: a.CostCenterID,
			CentroCodigo: a.CentroCodigo, CentroDescricao: a.CentroDescricao, Status: a.Status, AprovadorID: a.AprovadorID,
			DecididoEm: a.DecididoEm, Justificativa: a.Justificativa, CriadoEm: a.CriadoEm, PodeDecidir: pode})
	}
	return out, nil
}

func (uc *GestaoUseCase) Decidir(ctx context.Context, id int64, aprovar bool, justificativa string) error {
	eu, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	a, err := uc.Repo.Aprovacao(ctx, id)
	if err != nil {
		return err
	}
	if err := a.Decidir(eu, aprovar, justificativa); err != nil {
		return err
	}
	return uc.Repo.GravarDecisao(ctx, a)
}

func (uc *GestaoUseCase) Usuarios(ctx context.Context) ([]gestao.Usuario, error) {
	if !uc.Auth.CanListCostCenter(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.Usuarios(ctx)
}
