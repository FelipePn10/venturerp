// Package metrologia_uc: cadastro de instrumentos, registro de calibrações e
// a análise de impacto da calibração reprovada.
package metrologia_uc

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/metrologia"
)

type UseCase struct {
	Repo  dom.Repository
	Auth  ports.AuthService
	Agora func() time.Time
}

func (uc *UseCase) hoje() time.Time {
	if uc.Agora != nil {
		return uc.Agora()
	}
	return time.Now()
}

type InstrumentoDTO struct {
	Codigo         string           `json:"codigo"`
	Descricao      string           `json:"descricao"`
	Grupo          string           `json:"grupo"`
	Fabricante     *string          `json:"fabricante,omitempty"`
	NumeroSerie    *string          `json:"numero_serie,omitempty"`
	Faixa          *string          `json:"faixa,omitempty"`
	Resolucao      *decimal.Decimal `json:"resolucao,omitempty"`
	EMA            *decimal.Decimal `json:"ema,omitempty"`
	Unidade        *string          `json:"unidade,omitempty"`
	Localizacao    *string          `json:"localizacao,omitempty"`
	FrequenciaDias int              `json:"frequencia_dias"`
}

type CalibracaoDTO struct {
	Data           string           `json:"data"`
	Executor       string           `json:"executor"`
	Laboratorio    *string          `json:"laboratorio,omitempty"`
	Certificado    *string          `json:"certificado,omitempty"`
	ErroEncontrado *decimal.Decimal `json:"erro_encontrado,omitempty"`
	Incerteza      *decimal.Decimal `json:"incerteza,omitempty"`
	EMA            *decimal.Decimal `json:"ema,omitempty"`
	Resultado      string           `json:"resultado,omitempty"`
	Ajustado       bool             `json:"ajustado"`
	Observacoes    *string          `json:"observacoes,omitempty"`
}

// InstrumentoResponse: o instrumento com as calibrações.
type InstrumentoResponse struct {
	*dom.Instrumento
	Calibracoes []dom.Calibracao `json:"calibracoes"`
}

// CalibracaoResponse: o instrumento depois da calibração e, se reprovou, os
// lotes medidos com ele desde a última calibração aprovada.
type CalibracaoResponse struct {
	Calibracao  *dom.Calibracao  `json:"calibracao"`
	Instrumento *dom.Instrumento `json:"instrumento"`
	Impacto     []dom.Impacto    `json:"impacto,omitempty"`
	Desde       *time.Time       `json:"impacto_desde,omitempty"`
}

func (uc *UseCase) ator(ctx context.Context) *uuid.UUID {
	a, err := uc.Auth.UserID(ctx)
	if err != nil || a == uuid.Nil {
		return nil
	}
	return &a
}

func limpo(s *string) *string {
	if s == nil {
		return nil
	}
	if v := strings.TrimSpace(*s); v != "" {
		return &v
	}
	return nil
}

func (uc *UseCase) Salvar(ctx context.Context, id int64, dto InstrumentoDTO) (*InstrumentoResponse, error) {
	i := &dom.Instrumento{ID: id}
	if id != 0 {
		atual, err := uc.Repo.Obter(ctx, id)
		if err != nil {
			return nil, err
		}
		if atual.Status == "BAIXADO" {
			return nil, errorsuc.NewValidationError("instrumento baixado não muda")
		}
		i.Status, i.MotivoStatus = atual.Status, atual.MotivoStatus
	}
	i.Codigo, i.Descricao, i.Grupo = dto.Codigo, dto.Descricao, dto.Grupo
	i.Fabricante, i.NumeroSerie, i.Faixa = limpo(dto.Fabricante), limpo(dto.NumeroSerie), limpo(dto.Faixa)
	i.Resolucao, i.EMA, i.Unidade, i.Localizacao, i.FrequenciaDias = dto.Resolucao, dto.EMA, limpo(dto.Unidade), limpo(dto.Localizacao), dto.FrequenciaDias
	i.UpdatedBy = uc.ator(ctx)
	if err := i.Validar(); err != nil {
		return nil, err
	}
	if err := uc.Repo.Salvar(ctx, i); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, i.ID)
}

func (uc *UseCase) Obter(ctx context.Context, id int64) (*InstrumentoResponse, error) {
	i, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	cs, err := uc.Repo.Calibracoes(ctx, id)
	if err != nil {
		return nil, err
	}
	return &InstrumentoResponse{Instrumento: i, Calibracoes: cs}, nil
}

func (uc *UseCase) Listar(ctx context.Context, f dom.Filtro) ([]dom.Instrumento, error) {
	f.Hoje = uc.hoje()
	f.Situacao = strings.ToUpper(f.Situacao)
	return uc.Repo.Listar(ctx, f)
}

// Baixar retira o instrumento de uso (perdido, quebrado, sucateado).
func (uc *UseCase) Baixar(ctx context.Context, id int64, motivo string) (*InstrumentoResponse, error) {
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return nil, errorsuc.NewValidationError("informe o motivo da baixa")
	}
	i, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	i.Status, i.MotivoStatus, i.UpdatedBy = "BAIXADO", &motivo, uc.ator(ctx)
	if err := uc.Repo.Salvar(ctx, i); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

func (uc *UseCase) RegistrarCalibracao(ctx context.Context, id int64, dto CalibracaoDTO) (*CalibracaoResponse, error) {
	data, err := time.Parse("2006-01-02", strings.TrimSpace(dto.Data))
	if err != nil {
		return nil, errorsuc.NewValidationError("data da calibração inválida (use AAAA-MM-DD)")
	}
	c := &dom.Calibracao{
		Data: data, Executor: strings.ToUpper(strings.TrimSpace(dto.Executor)), Laboratorio: limpo(dto.Laboratorio), Certificado: limpo(dto.Certificado),
		ErroEncontrado: dto.ErroEncontrado, Incerteza: dto.Incerteza, EMA: dto.EMA, Resultado: strings.ToUpper(strings.TrimSpace(dto.Resultado)),
		Ajustado: dto.Ajustado, Observacoes: limpo(dto.Observacoes), CreatedBy: uc.ator(ctx),
	}
	// A última aprovada antes desta é o início da análise de impacto.
	anteriores, err := uc.Repo.Calibracoes(ctx, id)
	if err != nil {
		return nil, err
	}
	i, err := uc.Repo.RegistrarCalibracao(ctx, id, c, uc.hoje())
	if err != nil {
		return nil, err
	}
	out := &CalibracaoResponse{Calibracao: c, Instrumento: i}
	if c.Resultado == "REPROVADO" {
		desde := time.Time{}
		for _, a := range anteriores {
			if a.Resultado == "APROVADO" && !a.Data.After(data) {
				desde = a.Data
				break
			}
		}
		if out.Impacto, err = uc.Repo.Impacto(ctx, id, desde); err != nil {
			return nil, err
		}
		out.Desde = &desde
	}
	return out, nil
}

// Impacto sob demanda: lotes medidos com o instrumento desde a data.
func (uc *UseCase) Impacto(ctx context.Context, id int64, desde string) ([]dom.Impacto, error) {
	d := time.Time{}
	if strings.TrimSpace(desde) != "" {
		var err error
		if d, err = time.Parse("2006-01-02", desde); err != nil {
			return nil, errorsuc.NewValidationError("data inválida (use AAAA-MM-DD)")
		}
	}
	if _, err := uc.Repo.Obter(ctx, id); err != nil {
		return nil, err
	}
	return uc.Repo.Impacto(ctx, id, d)
}
