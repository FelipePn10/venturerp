// Package setor: setores da empresa (compras, planejamento, almoxarifado…) e as
// pessoas de cada um. O grupo do PDM aponta para o setor responsável pelo
// planejamento e pelas compras dos seus itens; a pessoa entra e sai do setor sem
// mexer em cadastro de item nenhum.
package setor

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// Setor é um setor da empresa com as pessoas que respondem por ele.
type Setor struct {
	Codigo  string   `json:"code"`
	Nome    string   `json:"name"`
	Ativo   bool     `json:"active"`
	Membros []Membro `json:"members"`
}

// Membro é uma pessoa (usuário da empresa) do setor.
type Membro struct {
	UserID uuid.UUID `json:"user_id"`
	Nome   string    `json:"name"`
	Email  string    `json:"email"`
}

// Conferir normaliza e valida antes de gravar.
func (s *Setor) Conferir() error {
	s.Codigo = strings.ToUpper(strings.TrimSpace(s.Codigo))
	s.Nome = strings.Join(strings.Fields(s.Nome), " ")
	if s.Codigo == "" || utf8.RuneCountInString(s.Codigo) > 60 {
		return domainerr.NewValidation("informe o código do setor, com até 60 caracteres")
	}
	if s.Nome == "" || utf8.RuneCountInString(s.Nome) > 120 {
		return domainerr.NewValidation("informe o nome do setor, com até 120 caracteres")
	}
	return nil
}

// Responsaveis diz quem responde pelo planejamento e pelas compras de um grupo
// do PDM (e, por herança, dos itens dele).
type Responsaveis struct {
	Grupo            int     `json:"group_code"`
	GrupoDescricao   string  `json:"group_description,omitempty"`
	Planejamento     *string `json:"planning_department,omitempty"`
	PlanejamentoNome string  `json:"planning_department_name,omitempty"`
	Compras          *string `json:"purchasing_department,omitempty"`
	ComprasNome      string  `json:"purchasing_department_name,omitempty"`
	// Pessoas de cada setor, para a tela do item mostrar quem é hoje.
	PessoasPlanejamento []string `json:"planning_people,omitempty"`
	PessoasCompras      []string `json:"purchasing_people,omitempty"`
}

// Repository guarda setores, membros e responsáveis por grupo.
type Repository interface {
	Setores(ctx context.Context) ([]Setor, error)
	SalvarSetor(ctx context.Context, s Setor) error
	DefinirMembros(ctx context.Context, codigo string, usuarios []uuid.UUID) error
	ResponsaveisDoGrupo(ctx context.Context, grupo int) (*Responsaveis, error)
	SalvarResponsaveis(ctx context.Context, r Responsaveis, por *uuid.UUID) error
	ResponsaveisDoItem(ctx context.Context, codigoDoItem string) (*Responsaveis, error)
}
