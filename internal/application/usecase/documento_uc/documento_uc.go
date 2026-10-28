// Package documento_uc conduz o controle de documentos da qualidade.
package documento_uc

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/documento"
)

type UseCase struct {
	Repo dom.Repository
	Auth ports.AuthService
}

type DocumentoDTO struct {
	Codigo           string  `json:"codigo"`
	Titulo           string  `json:"titulo"`
	Tipo             string  `json:"tipo"`
	ItemCode         *int64  `json:"item_code,omitempty"`
	RouteOperationID *int64  `json:"route_operation_id,omitempty"`
	Area             *string `json:"area,omitempty"`
	Ativo            *bool   `json:"ativo,omitempty"`
}

type RascunhoDTO struct {
	DescricaoMudanca string  `json:"descricao_mudanca"`
	Conteudo         *string `json:"conteudo,omitempty"`
	ArquivoNome      *string `json:"-"`
	ArquivoTipo      *string `json:"-"`
	Arquivo          []byte  `json:"-"`
}

func (uc *UseCase) ator(ctx context.Context) (uuid.UUID, error) {
	a, err := uc.Auth.UserID(ctx)
	if err != nil || a == uuid.Nil {
		return uuid.Nil, errorsuc.ErrUnauthorized
	}
	return a, nil
}

func (uc *UseCase) Salvar(ctx context.Context, id int64, dto DocumentoDTO) (*dom.Documento, error) {
	ator, err := uc.ator(ctx)
	if err != nil {
		return nil, err
	}
	d := &dom.Documento{ID: id, Codigo: dto.Codigo, Titulo: dto.Titulo, Tipo: dto.Tipo, ItemCode: dto.ItemCode, RouteOperationID: dto.RouteOperationID,
		Area: dto.Area, Ativo: dto.Ativo == nil || *dto.Ativo}
	if err := d.Validar(); err != nil {
		return nil, err
	}
	if err := uc.Repo.Salvar(ctx, d, ator); err != nil {
		return nil, err
	}
	return uc.Repo.Obter(ctx, d.ID)
}

func (uc *UseCase) Obter(ctx context.Context, id int64) (*dom.Documento, error) {
	return uc.Repo.Obter(ctx, id)
}

func (uc *UseCase) Listar(ctx context.Context, f dom.Filtro) ([]dom.Documento, error) {
	return uc.Repo.Listar(ctx, f)
}

func (uc *UseCase) GravarRascunho(ctx context.Context, docID int64, dto RascunhoDTO) (*dom.Documento, error) {
	ator, err := uc.ator(ctx)
	if err != nil {
		return nil, err
	}
	r := &dom.Revisao{DescricaoMudanca: dto.DescricaoMudanca, Conteudo: dto.Conteudo, ArquivoNome: dto.ArquivoNome, ArquivoTipo: dto.ArquivoTipo,
		Arquivo: dto.Arquivo, ElaboradoPor: ator}
	if r.Conteudo != nil && strings.TrimSpace(*r.Conteudo) == "" {
		r.Conteudo = nil
	}
	if err := r.ValidarRascunho(); err != nil {
		// Rascunho já com arquivo: trocar só o texto não exige reenviar o arquivo.
		atual, errObter := uc.Repo.Obter(ctx, docID)
		if errObter != nil || atual.Rascunho == nil || !atual.Rascunho.TemArquivo || strings.TrimSpace(r.DescricaoMudanca) == "" {
			return nil, err
		}
	}
	if err := uc.Repo.GravarRascunho(ctx, docID, r); err != nil {
		return nil, err
	}
	return uc.Repo.Obter(ctx, docID)
}

// Aprovar (rota APROVAR) torna o rascunho vigente.
func (uc *UseCase) Aprovar(ctx context.Context, docID int64) (*dom.Documento, error) {
	ator, err := uc.ator(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := uc.Repo.Aprovar(ctx, docID, ator); err != nil {
		return nil, err
	}
	return uc.Repo.Obter(ctx, docID)
}

func (uc *UseCase) DescartarRascunho(ctx context.Context, docID int64) (*dom.Documento, error) {
	if err := uc.Repo.DescartarRascunho(ctx, docID); err != nil {
		return nil, err
	}
	return uc.Repo.Obter(ctx, docID)
}

func (uc *UseCase) Arquivo(ctx context.Context, revID int64) (*dom.Revisao, error) {
	return uc.Repo.Arquivo(ctx, revID)
}
