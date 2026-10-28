// Package documento é o controle de documentos da qualidade: instrução de
// trabalho, procedimento, formulário, plano e especificação, com revisões
// RASCUNHO → VIGENTE → OBSOLETA e aprovação por quem não elaborou.
package documento

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

var tipos = map[string]bool{"IT": true, "PROCEDIMENTO": true, "FORMULARIO": true, "PLANO_INSPECAO": true, "ESPECIFICACAO": true, "OUTRO": true}

type Documento struct {
	ID               int64     `json:"id"`
	Codigo           string    `json:"codigo"`
	Titulo           string    `json:"titulo"`
	Tipo             string    `json:"tipo"`
	ItemCode         *int64    `json:"item_code,omitempty"`
	RouteOperationID *int64    `json:"route_operation_id,omitempty"`
	Area             *string   `json:"area,omitempty"`
	Ativo            bool      `json:"ativo"`
	CreatedAt        time.Time `json:"created_at"`
	Vigente          *Revisao  `json:"vigente,omitempty"`
	Rascunho         *Revisao  `json:"rascunho,omitempty"`
	Revisoes         []Revisao `json:"revisoes,omitempty"`
}

type Revisao struct {
	ID               int64      `json:"id"`
	DocumentoID      int64      `json:"document_id"`
	Revisao          int        `json:"revisao"`
	Status           string     `json:"status"`
	DescricaoMudanca string     `json:"descricao_mudanca"`
	Conteudo         *string    `json:"conteudo,omitempty"`
	ArquivoNome      *string    `json:"arquivo_nome,omitempty"`
	ArquivoTipo      *string    `json:"arquivo_tipo,omitempty"`
	TemArquivo       bool       `json:"tem_arquivo"`
	Arquivo          []byte     `json:"-"`
	ElaboradoPor     uuid.UUID  `json:"elaborado_por"`
	ElaboradoEm      time.Time  `json:"elaborado_em"`
	AprovadoPor      *uuid.UUID `json:"aprovado_por,omitempty"`
	VigenteDesde     *time.Time `json:"vigente_desde,omitempty"`
	ObsoletaEm       *time.Time `json:"obsoleta_em,omitempty"`
}

// Validar confere o cabeçalho.
func (d *Documento) Validar() error {
	d.Codigo = strings.ToUpper(strings.TrimSpace(d.Codigo))
	d.Titulo = strings.TrimSpace(d.Titulo)
	d.Tipo = strings.ToUpper(strings.TrimSpace(d.Tipo))
	switch {
	case d.Codigo == "" || len(d.Codigo) > 30:
		return domainerr.NewValidation("informe o código do documento (até 30 caracteres)")
	case d.Titulo == "":
		return domainerr.NewValidation("informe o título do documento")
	case !tipos[d.Tipo]:
		return domainerr.NewValidation("tipo inválido (IT, PROCEDIMENTO, FORMULARIO, PLANO_INSPECAO, ESPECIFICACAO ou OUTRO)")
	}
	return nil
}

// ArquivoMaximo: 10 MB por revisão (o documento vai no banco).
const ArquivoMaximo = 10 << 20

// ValidarRascunho confere a revisão em elaboração.
func (r *Revisao) ValidarRascunho() error {
	r.DescricaoMudanca = strings.TrimSpace(r.DescricaoMudanca)
	if r.DescricaoMudanca == "" {
		return domainerr.NewValidation("descreva o que muda nesta revisão")
	}
	semTexto := r.Conteudo == nil || strings.TrimSpace(*r.Conteudo) == ""
	if semTexto && len(r.Arquivo) == 0 {
		return domainerr.NewValidation("a revisão precisa de texto ou de arquivo")
	}
	if len(r.Arquivo) > ArquivoMaximo {
		return domainerr.NewValidation("o arquivo passa de 10 MB")
	}
	return nil
}

// PodeAprovar: só rascunho, e quem elaborou não aprova (segregação).
func (r *Revisao) PodeAprovar(aprovador uuid.UUID) error {
	if r.Status != "RASCUNHO" {
		return domainerr.NewValidation("só a revisão em rascunho é aprovada")
	}
	if aprovador == r.ElaboradoPor {
		return domainerr.NewValidation("quem elaborou a revisão não pode aprová-la")
	}
	return nil
}

type Filtro struct {
	Tipo     string
	ItemCode int64
	Busca    string
	Todos    bool // inclui inativos
}

type Repository interface {
	Salvar(ctx context.Context, d *Documento, ator uuid.UUID) error
	Obter(ctx context.Context, id int64) (*Documento, error)
	Listar(ctx context.Context, f Filtro) ([]Documento, error)
	// GravarRascunho cria (ou substitui) o rascunho do documento.
	GravarRascunho(ctx context.Context, documentoID int64, r *Revisao) error
	// Aprovar torna o rascunho vigente e a vigente anterior obsoleta, numa
	// transação; confere a segregação sobre o rascunho travado.
	Aprovar(ctx context.Context, documentoID int64, aprovador uuid.UUID) (*Revisao, error)
	DescartarRascunho(ctx context.Context, documentoID int64) error
	Arquivo(ctx context.Context, revisaoID int64) (*Revisao, error)
}
