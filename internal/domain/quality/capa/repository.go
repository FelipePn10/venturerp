package capa

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Filtro da lista de CAPAs.
type Filtro struct {
	Status      string
	Responsavel *uuid.UUID
	Atrasadas   bool
	Hoje        time.Time
}

// NC como a CAPA e a tela de NC a mostram (com lote de inspeção e CAPAs).
type NC struct {
	ID                int64     `json:"id"`
	Code              int64     `json:"code"`
	ItemCode          int64     `json:"item_code"`
	Lot               *string   `json:"lot,omitempty"`
	LotID             *int64    `json:"lot_id,omitempty"`
	LoteNumero        *int64    `json:"lote_numero,omitempty"`
	SupplierCode      *int64    `json:"supplier_code,omitempty"`
	ProductionOrderID *int64    `json:"production_order_id,omitempty"`
	Quantidade        string    `json:"nonconform_qty"`
	Descricao         string    `json:"description"`
	Severidade        string    `json:"severity"`
	Disposicao        *string   `json:"disposition,omitempty"`
	Aberta            bool      `json:"is_open"`
	CreatedAt         time.Time `json:"created_at"`
	Capas             []int64   `json:"capas"`
	CapaNumeros       []int64   `json:"capa_numeros"`
}

// FiltroNC da lista de NCs.
type FiltroNC struct {
	SoAbertas    bool
	SemCapa      bool
	ItemCode     int64
	SupplierCode int64
	Limite       int
}

// Mudanca altera a CAPA travada; devolve o evento do histórico (opcional).
type Mudanca func(c *Capa) (*Evento, error)

// Repository da CAPA, na empresa da sessão.
type Repository interface {
	Abrir(ctx context.Context, c *Capa, ator uuid.UUID) error
	Obter(ctx context.Context, id int64) (*Capa, error)
	Listar(ctx context.Context, f Filtro) ([]Capa, error)
	// Atualizar trava a CAPA, aplica fn e grava cabeçalho, equipe e evento.
	Atualizar(ctx context.Context, id int64, ator uuid.UUID, fn Mudanca) (*Capa, error)
	AdicionarAcao(ctx context.Context, capaID int64, a *Acao, ator uuid.UUID) error
	// AtualizarAcao trava a ação e a CAPA (aberta) e aplica fn.
	AtualizarAcao(ctx context.Context, capaID, acaoID int64, ator uuid.UUID, fn func(c *Capa, a *Acao) (*Evento, error)) (*Capa, error)
	VincularNC(ctx context.Context, capaID, ncID int64, ator uuid.UUID) error
	AcoesPendentes(ctx context.Context, responsavel *uuid.UUID) ([]AcaoPendente, error)
	NCs(ctx context.Context, f FiltroNC) ([]NC, error)
	// UsuariosDaEmpresa confere que equipe e responsáveis são da empresa.
	UsuariosDaEmpresa(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error)
	// Usuarios ativos da empresa, para escolher equipe e responsáveis.
	Usuarios(ctx context.Context) ([]Membro, error)
}

// AcaoPendente para o painel "minhas ações".
type AcaoPendente struct {
	Acao
	CapaID     int64  `json:"capa_id"`
	CapaNumero int64  `json:"capa_numero"`
	CapaTitulo string `json:"capa_titulo"`
	Atrasada   bool   `json:"atrasada"`
}
