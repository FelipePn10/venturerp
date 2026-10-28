// Package amostragem_uc expõe a amostragem NBR 5426: procedimentos da empresa,
// calculadora, plano do lote pela severidade corrente e o registro do
// resultado, que aplica as regras de troca e grava o histórico.
package amostragem_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/amostragem"
)

type UseCase struct {
	Repo dom.Repository
	Auth ports.AuthService
}

// ─── DTOs ─────────────────────────────────────────────────────────────────────

type ProcedimentoDTO struct {
	Codigo          string `json:"codigo"`
	Descricao       string `json:"descricao"`
	Tipo            string `json:"tipo"`
	Nivel           string `json:"nivel,omitempty"`
	NQA             string `json:"nqa,omitempty"`
	PermiteAtenuada bool   `json:"permite_atenuada"`
	AmostraFixa     *int64 `json:"amostra_fixa,omitempty"`
	AcFixo          *int   `json:"ac_fixo,omitempty"`
	ReFixo          *int   `json:"re_fixo,omitempty"`
	SkipApos        int    `json:"skip_apos,omitempty"`
	SkipFrequencia  int    `json:"skip_frequencia,omitempty"`
	Ativo           *bool  `json:"ativo,omitempty"`
}

type LiberacaoDTO struct {
	ProcedimentoID int64  `json:"procedure_id"`
	ItemCode       int64  `json:"item_code"`
	Chave          string `json:"chave"`
	Motivo         string `json:"motivo"`
}

// PlanoDoLote é o que a inspeção precisa saber antes de amostrar.
type PlanoDoLote struct {
	Plano dom.Plano `json:"plano"`
	// Inspecionar falso: o skip-lot libera este lote sem amostra.
	Inspecionar bool `json:"inspecionar"`
	// Suspensa: 5 recusas em apertada (9.4). O lote não pode ser aceito pela
	// norma até a liberação.
	Suspensa bool       `json:"suspensa"`
	Estado   dom.Estado `json:"estado"`
}

// ResultadoRegistrado devolve o estado depois do lote e a troca, se houve.
type ResultadoRegistrado struct {
	Aceito    bool           `json:"aceito"`
	Estado    dom.Estado     `json:"estado"`
	Transicao *dom.Transicao `json:"transicao,omitempty"`
}

// ─── calculadora ──────────────────────────────────────────────────────────────

// Calcular é a consulta da norma, sem estado: letra, n, Ac e Re.
func (uc *UseCase) Calcular(lote int64, nivel, nqa, severidade string) (dom.Plano, error) {
	sev := dom.Severidade(strings.ToUpper(strings.TrimSpace(severidade)))
	if sev == "" {
		sev = dom.Normal
	}
	if sev != dom.Normal && sev != dom.Apertada && sev != dom.Atenuada {
		return dom.Plano{}, errorsuc.NewValidationError("severidade inválida (NORMAL, APERTADA ou ATENUADA)")
	}
	niv := dom.Nivel(strings.ToUpper(strings.TrimSpace(nivel)))
	if niv == "" {
		niv = dom.NivelII
	}
	return dom.PlanoParaLote(lote, niv, nqa, sev)
}

// ─── procedimentos ────────────────────────────────────────────────────────────

func (uc *UseCase) ListarProcedimentos(ctx context.Context, somenteAtivos bool) ([]dom.Procedimento, error) {
	return uc.Repo.ListarProcedimentos(ctx, somenteAtivos)
}

func (uc *UseCase) ObterProcedimento(ctx context.Context, id int64) (*dom.Procedimento, error) {
	return uc.Repo.ObterProcedimento(ctx, id)
}

func (uc *UseCase) Salvar(ctx context.Context, id int64, dto ProcedimentoDTO) (*dom.Procedimento, error) {
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	p := &dom.Procedimento{
		ID: id, Codigo: dto.Codigo, Descricao: dto.Descricao,
		Tipo:  dom.TipoProcedimento(strings.ToUpper(strings.TrimSpace(dto.Tipo))),
		Nivel: dom.Nivel(strings.ToUpper(strings.TrimSpace(dto.Nivel))), NQA: dto.NQA,
		PermiteAtenuada: dto.PermiteAtenuada, AmostraFixa: dto.AmostraFixa, AcFixo: dto.AcFixo, ReFixo: dto.ReFixo,
		Skip:  dom.SkipLot{AposAceitos: dto.SkipApos, Frequencia: dto.SkipFrequencia},
		Ativo: dto.Ativo == nil || *dto.Ativo, UpdatedBy: ator,
	}
	if id != 0 {
		if _, err := uc.Repo.ObterProcedimento(ctx, id); err != nil {
			return nil, err
		}
	}
	p.Normalizar()
	if err := p.Validar(); err != nil {
		return nil, err
	}
	if err := uc.Repo.SalvarProcedimento(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// ─── lote ─────────────────────────────────────────────────────────────────────

// PlanoDoLote calcula o plano do lote pela severidade corrente da unidade de
// controle. quantidade vem como texto decimal (o estoque é NUMERIC).
func (uc *UseCase) PlanoDoLote(ctx context.Context, procedimentoID, item int64, chave, quantidade string) (*PlanoDoLote, error) {
	p, err := uc.Repo.ObterProcedimento(ctx, procedimentoID)
	if err != nil {
		return nil, err
	}
	if !p.Ativo {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o procedimento de amostragem %s está inativo", p.Codigo))
	}
	tamanho, err := tamanhoDoLote(p, quantidade)
	if err != nil {
		return nil, err
	}
	e, err := uc.Repo.Estado(ctx, procedimentoID, item, chaveOuGeral(chave))
	if err != nil {
		return nil, err
	}
	pl, err := p.PlanoPara(tamanho, e)
	if err != nil {
		return nil, err
	}
	return &PlanoDoLote{Plano: pl, Inspecionar: e.DeveInspecionar(p.Skip), Suspensa: e.Suspensa, Estado: e}, nil
}

// tamanhoDoLote: a norma conta unidades; plano fixo e 100 % aceitam lote
// fracionado (o 100 % inspeciona o lote inteiro, 1 "unidade" de controle).
func tamanhoDoLote(p *dom.Procedimento, quantidade string) (int64, error) {
	n, err := dom.TamanhoDoLote(quantidade)
	if err == nil || p.Tipo == dom.TipoNorma {
		return n, err
	}
	return 1, nil
}

func chaveOuGeral(c string) string {
	if c = strings.TrimSpace(c); c == "" {
		return dom.ChaveGeral
	}
	return c
}

// RegistrarResultado aplica o resultado do lote inspecionado com o plano que
// foi usado (o gravado no lote, não o de agora: a severidade pode ter mudado
// com outro lote no meio). Lote suspenso não entra na norma até a liberação.
func (uc *UseCase) RegistrarResultado(ctx context.Context, procedimentoID, item int64, chave string, pl dom.Plano, naoConformes int, producaoRegular bool, referencia string) (*ResultadoRegistrado, error) {
	if naoConformes < 0 || int64(naoConformes) > pl.N {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("não conformes (%d) fora da amostra de %d", naoConformes, pl.N))
	}
	ator, _ := uc.Auth.UserID(ctx)
	p, err := uc.Repo.ObterProcedimento(ctx, procedimentoID)
	if err != nil {
		return nil, err
	}
	r := p.ResultadoDoLote(pl, naoConformes)
	ref := refOuNil(referencia)
	e, tr, err := uc.Repo.AtualizarEstado(ctx, procedimentoID, item, chaveOuGeral(chave), ref, ator, func(e dom.Estado) (dom.Estado, *dom.Transicao, error) {
		if e.Suspensa {
			return e, nil, errorsuc.NewValidationError("a inspeção deste item/fornecedor está suspensa (5 lotes recusados em apertada): libere a suspensão antes de registrar lotes")
		}
		novo, tr := p.Registrar(e, r, producaoRegular)
		return novo, tr, nil
	})
	if err != nil {
		return nil, err
	}
	return &ResultadoRegistrado{Aceito: r.Aceito, Estado: e, Transicao: tr}, nil
}

// PularLote registra o lote liberado pelo skip-lot.
func (uc *UseCase) PularLote(ctx context.Context, procedimentoID, item int64, chave, referencia string) (dom.Estado, error) {
	ator, _ := uc.Auth.UserID(ctx)
	p, err := uc.Repo.ObterProcedimento(ctx, procedimentoID)
	if err != nil {
		return dom.Estado{}, err
	}
	e, _, err := uc.Repo.AtualizarEstado(ctx, procedimentoID, item, chaveOuGeral(chave), refOuNil(referencia), ator, func(e dom.Estado) (dom.Estado, *dom.Transicao, error) {
		if e.DeveInspecionar(p.Skip) {
			return e, nil, errorsuc.NewValidationError("este lote precisa ser inspecionado: o skip-lot não o libera")
		}
		return e.Pular(), nil, nil
	})
	return e, err
}

// LiberarSuspensao levanta a suspensão de 9.4 depois da ação do fornecedor; a
// inspeção recomeça em apertada. Exige alçada de aprovação (rota APROVAR).
func (uc *UseCase) LiberarSuspensao(ctx context.Context, dto LiberacaoDTO) (dom.Estado, error) {
	ator, err := uc.Auth.UserID(ctx)
	if err != nil || ator == uuid.Nil {
		return dom.Estado{}, errorsuc.ErrUnauthorized
	}
	motivo := strings.TrimSpace(dto.Motivo)
	if motivo == "" {
		return dom.Estado{}, errorsuc.NewValidationError("informe a ação do fornecedor que justifica a liberação")
	}
	if dto.ItemCode <= 0 || strings.TrimSpace(dto.Chave) == "" {
		return dom.Estado{}, errorsuc.NewValidationError("informe o item e a unidade de controle")
	}
	if _, err := uc.Repo.ObterProcedimento(ctx, dto.ProcedimentoID); err != nil {
		return dom.Estado{}, err
	}
	e, _, err := uc.Repo.AtualizarEstado(ctx, dto.ProcedimentoID, dto.ItemCode, strings.TrimSpace(dto.Chave), nil, ator, func(e dom.Estado) (dom.Estado, *dom.Transicao, error) {
		if !e.Suspensa {
			return e, nil, errorsuc.NewValidationError("a inspeção deste item/fornecedor não está suspensa")
		}
		return e.Liberar(), &dom.Transicao{De: e.Severidade, Para: dom.Apertada, Liberacao: true, Motivo: "liberada: " + motivo}, nil
	})
	return e, err
}

func (uc *UseCase) Estados(ctx context.Context, item int64) ([]dom.EstadoDoItem, error) {
	return uc.Repo.EstadosDoItem(ctx, item)
}

func (uc *UseCase) Eventos(ctx context.Context, item int64) ([]dom.Evento, error) {
	return uc.Repo.Eventos(ctx, item, 200)
}

func refOuNil(s string) *string {
	if s = strings.TrimSpace(s); s == "" {
		return nil
	}
	return &s
}
