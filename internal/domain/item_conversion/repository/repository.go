package repository

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/service"
)

// ItemConversionRepository: conversões do item. A empresa é a do ITEM
// (items.enterprise_id) — toda leitura e escrita confere por EXISTS.
type ItemConversionRepository interface {
	Create(ctx context.Context, c *entity.ItemUnitConversion) (*entity.ItemUnitConversion, error)
	// ListByItem: versões ativas (vigentes e futuras).
	ListByItem(ctx context.Context, itemCode int64) ([]*entity.ItemUnitConversion, error)
	// Get devolve a conversão direta (item, de, para) vigente hoje, sem máscara.
	Get(ctx context.Context, itemCode int64, fromUOM, toUOM string) (*entity.ItemUnitConversion, error)
	Delete(ctx context.Context, id int64) error
}

// Impacto é o que depende de uma conversão — mostrado antes de alterar/excluir.
type Impacto struct {
	EstruturasNaUnidade   int64 // linhas de estrutura (item como componente) em uma das duas unidades
	EstruturasDivergentes int64 // dessas, as que congelaram um fator diferente do atual
	PedidosAbertos        int64 // linhas de pedido em aberto comprando numa das unidades
	NotasPendentes        int64 // linhas de nota de entrada ainda não aprovadas
	VinculosFornecedor    int64 // vínculos item-fornecedor com unidade do XML no par
	NotasAprovadasNoMes   int64 // notas aprovadas do item no mês corrente (SPED: fator não muda no mês)
	// FatoresCongelados: fatores gravados nas linhas de estrutura que usam o
	// item fora da unidade de estoque — o caso de uso compara com o vigente.
	FatoresCongelados []FatorCongelado
}

// FatorCongelado: linhas de estrutura numa unidade, com o fator que gravaram.
type FatorCongelado struct {
	Unidade string
	Fator   decimal.Decimal
	Linhas  int64
}

// UsoDeUnidade: uma unidade em que o item aparece fora do cadastro.
type UsoDeUnidade struct {
	Unidade    string
	Onde       string // ESTRUTURA, FORNECEDOR, PEDIDO, NOTA
	Quantidade int64
}

// UnidadesDoItem: os papéis de unidade do cadastro do item e onde mais ele
// aparece em outra unidade.
type UnidadesDoItem struct {
	ItemCode       int64
	Estoque        string
	Compra         string
	CompraContabil string
	VendaContabil  string
	Secundaria     string
	AceitaFracao   bool
	Usos           []UsoDeUnidade
}

// ManagedItemConversionRepository é o contrato completo usado pela VSUP0110.
type ManagedItemConversionRepository interface {
	ItemConversionRepository
	GetByID(ctx context.Context, id int64) (*entity.ItemUnitConversion, error)
	ListVersions(ctx context.Context, itemCode int64) ([]*entity.ItemUnitConversion, error)
	// Update altera a linha no lugar e grava o evento — na mesma transação.
	Update(ctx context.Context, c *entity.ItemUnitConversion, ev *entity.Event) (*entity.ItemUnitConversion, error)
	// NewVersion encerra `atual` em closeAt e cria `nova` — na mesma transação.
	NewVersion(ctx context.Context, atual *entity.ItemUnitConversion, closeAt time.Time, nova *entity.ItemUnitConversion, ev *entity.Event) (*entity.ItemUnitConversion, error)
	CreateWithEvent(ctx context.Context, c *entity.ItemUnitConversion, ev *entity.Event) (*entity.ItemUnitConversion, error)
	Deactivate(ctx context.Context, id int64, ev *entity.Event) error

	ListLotConversions(ctx context.Context, itemCode int64) ([]*entity.LotConversion, error)
	GetLotConversion(ctx context.Context, id int64) (*entity.LotConversion, error)
	UpsertLotConversion(ctx context.Context, l *entity.LotConversion, ev *entity.Event) (*entity.LotConversion, error)
	DeactivateLotConversion(ctx context.Context, id int64, ev *entity.Event) error
	LotExists(ctx context.Context, itemCode int64, lot string) (bool, error)

	Events(ctx context.Context, itemCode int64, limit int) ([]*entity.Event, error)

	// Graph carrega as arestas do item (lote, máscara, item) e as padrão da
	// empresa, vigentes na data.
	Graph(ctx context.Context, itemCode int64, mask, lot string, at time.Time) ([]service.Aresta, error)
	AcceptsFractional(ctx context.Context, itemCode int64) (bool, error)
	UnitsOfItem(ctx context.Context, itemCode int64) (*UnidadesDoItem, error)
	Impact(ctx context.Context, c *entity.ItemUnitConversion) (*Impacto, error)
	// UnitsExist devolve as unidades informadas que NÃO estão ativas no
	// catálogo da empresa do item.
	MissingUnits(ctx context.Context, itemCode int64, units ...string) ([]string, error)
}
