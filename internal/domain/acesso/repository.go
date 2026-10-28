package acesso

import (
	"context"

	"github.com/google/uuid"
)

// Repository guarda perfis, permissões e vínculos da empresa da sessão.
type Repository interface {
	ListarPerfis(ctx context.Context) ([]*Perfil, error)
	BuscarPerfil(ctx context.Context, id int64) (*Perfil, error)
	CriarPerfil(ctx context.Context, p *Perfil, actor uuid.UUID) (*Perfil, error)
	// AtualizarPerfil grava nome, descrição, ativo e substitui as permissões.
	AtualizarPerfil(ctx context.Context, p *Perfil, actor uuid.UUID) error
	ExcluirPerfil(ctx context.Context, id int64) error

	ListarUsuarios(ctx context.Context) ([]UsuarioDaEmpresa, error)
	// DefinirPerfisDoUsuario substitui os perfis do usuário na empresa.
	DefinirPerfisDoUsuario(ctx context.Context, userID uuid.UUID, perfis []int64, actor uuid.UUID) error

	// AcessoDoUsuario lê os perfis ATIVOS do usuário na empresa informada.
	AcessoDoUsuario(ctx context.Context, userID uuid.UUID, enterpriseID int64) (AcessoDoUsuario, error)
}
