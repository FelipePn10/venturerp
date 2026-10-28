// Package acesso_uc mantém os perfis de acesso da empresa e os vínculos com os
// usuários. As rotas são exclusivas de ADMIN; o caso de uso valida o conteúdo
// e invalida o cache do controle de acesso a cada gravação.
package acesso_uc

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/acesso"
)

// Invalidador esquece o acesso guardado dos usuários.
type Invalidador interface{ Invalidar() }

type UseCase struct {
	Repo  acesso.Repository
	Auth  ports.AuthService
	Cache Invalidador
}

// ─── DTOs ─────────────────────────────────────────────────────────────────────

type PermissaoDTO struct {
	Modulo string `json:"modulo"`
	Acao   string `json:"acao"`
}

type ModuloResponse struct {
	Modulo string   `json:"modulo"`
	Nome   string   `json:"nome"`
	Acoes  []string `json:"acoes"`
}

type PerfilDTO struct {
	Nome       string         `json:"nome"`
	Descricao  *string        `json:"descricao,omitempty"`
	Ativo      *bool          `json:"ativo,omitempty"`
	Permissoes []PermissaoDTO `json:"permissoes"`
}

type PerfilResponse struct {
	ID         int64          `json:"id"`
	Nome       string         `json:"nome"`
	Descricao  *string        `json:"descricao,omitempty"`
	Ativo      bool           `json:"ativo"`
	Usuarios   int            `json:"usuarios"`
	Permissoes []PermissaoDTO `json:"permissoes"`
	UpdatedAt  string         `json:"updated_at"`
}

type UsuarioResponse struct {
	UserID string  `json:"user_id"`
	Nome   string  `json:"nome"`
	Email  string  `json:"email"`
	Papel  string  `json:"papel"`
	Ativo  bool    `json:"ativo"`
	Perfis []int64 `json:"perfis"`
}

type PerfisDoUsuarioDTO struct {
	Perfis []int64 `json:"perfis"`
}

// ─── catálogo ─────────────────────────────────────────────────────────────────

func Catalogo() []ModuloResponse {
	out := make([]ModuloResponse, 0, len(acesso.Catalogo))
	for _, m := range acesso.Catalogo {
		r := ModuloResponse{Modulo: string(m.Modulo), Nome: m.Nome}
		for _, a := range m.Acoes {
			r.Acoes = append(r.Acoes, string(a))
		}
		out = append(out, r)
	}
	return out
}

func permissoesDTO(p acesso.Permissoes) []PermissaoDTO {
	out := []PermissaoDTO{}
	for _, item := range p.Lista() {
		out = append(out, PermissaoDTO{Modulo: string(item.Modulo), Acao: string(item.Acao)})
	}
	return out
}

func perfilResponse(p *acesso.Perfil) PerfilResponse {
	return PerfilResponse{ID: p.ID, Nome: p.Nome, Descricao: p.Descricao, Ativo: p.Ativo, Usuarios: p.Usuarios,
		Permissoes: permissoesDTO(p.Permissoes), UpdatedAt: p.UpdatedAt.Format(time.RFC3339)}
}

// ─── perfis ───────────────────────────────────────────────────────────────────

func (uc *UseCase) ListarPerfis(ctx context.Context) ([]PerfilResponse, error) {
	perfis, err := uc.Repo.ListarPerfis(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PerfilResponse, 0, len(perfis))
	for _, p := range perfis {
		out = append(out, perfilResponse(p))
	}
	return out, nil
}

func (uc *UseCase) montar(dto PerfilDTO) (*acesso.Perfil, error) {
	lista := make([]acesso.Permissao, 0, len(dto.Permissoes))
	for _, p := range dto.Permissoes {
		lista = append(lista, acesso.Permissao{Modulo: acesso.Modulo(p.Modulo), Acao: acesso.Acao(p.Acao)})
	}
	perms, err := acesso.NovasPermissoes(lista)
	if err != nil {
		return nil, err
	}
	p := &acesso.Perfil{Nome: dto.Nome, Descricao: dto.Descricao, Ativo: true, Permissoes: perms}
	if dto.Ativo != nil {
		p.Ativo = *dto.Ativo
	}
	if err := p.Validar(); err != nil {
		return nil, err
	}
	return p, nil
}

func (uc *UseCase) CriarPerfil(ctx context.Context, dto PerfilDTO) (*PerfilResponse, error) {
	p, err := uc.montar(dto)
	if err != nil {
		return nil, err
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	criado, err := uc.Repo.CriarPerfil(ctx, p, actor)
	if err != nil {
		return nil, err
	}
	uc.invalidar()
	r := perfilResponse(criado)
	return &r, nil
}

func (uc *UseCase) AtualizarPerfil(ctx context.Context, id int64, dto PerfilDTO) (*PerfilResponse, error) {
	p, err := uc.montar(dto)
	if err != nil {
		return nil, err
	}
	p.ID = id
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := uc.Repo.AtualizarPerfil(ctx, p, actor); err != nil {
		return nil, err
	}
	uc.invalidar()
	atual, err := uc.Repo.BuscarPerfil(ctx, id)
	if err != nil {
		return nil, err
	}
	r := perfilResponse(atual)
	return &r, nil
}

func (uc *UseCase) ExcluirPerfil(ctx context.Context, id int64) error {
	if err := uc.Repo.ExcluirPerfil(ctx, id); err != nil {
		return err
	}
	uc.invalidar()
	return nil
}

// ─── usuários ─────────────────────────────────────────────────────────────────

func (uc *UseCase) ListarUsuarios(ctx context.Context) ([]UsuarioResponse, error) {
	us, err := uc.Repo.ListarUsuarios(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]UsuarioResponse, 0, len(us))
	for _, u := range us {
		perfis := u.Perfis
		if perfis == nil {
			perfis = []int64{}
		}
		out = append(out, UsuarioResponse{UserID: u.UserID.String(), Nome: u.Nome, Email: u.Email, Papel: u.Papel, Ativo: u.Ativo, Perfis: perfis})
	}
	return out, nil
}

func (uc *UseCase) DefinirPerfisDoUsuario(ctx context.Context, userID string, dto PerfisDoUsuarioDTO) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return errorsuc.NewValidationError("usuário inválido")
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	vistos := map[int64]bool{}
	perfis := []int64{}
	for _, id := range dto.Perfis {
		if id <= 0 {
			return errorsuc.NewValidationError("perfil inválido")
		}
		if !vistos[id] {
			vistos[id] = true
			perfis = append(perfis, id)
		}
	}
	if err := uc.Repo.DefinirPerfisDoUsuario(ctx, uid, perfis, actor); err != nil {
		return err
	}
	uc.invalidar()
	return nil
}

func (uc *UseCase) invalidar() {
	if uc.Cache != nil {
		uc.Cache.Invalidar()
	}
}
