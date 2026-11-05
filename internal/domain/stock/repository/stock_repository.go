package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

var ErrInsufficientStock = domainerr.NewValidation("estoque disponível insuficiente para a reserva")

// ErrAlmoxSemReserva: o cadastro do almoxarifado não aceita reservas.
var ErrAlmoxSemReserva = domainerr.NewValidation("este almoxarifado não aceita reservas: reserve em outro ou mude o cadastro do almoxarifado")

type StockRepository interface {
	// Stock movements
	CreateMovement(ctx context.Context, m *entity.StockMovement) (*entity.StockMovement, error)
	ListMovements(ctx context.Context) ([]*entity.StockMovement, error)
	ListMovementsByItem(ctx context.Context, itemCode int64) ([]*entity.StockMovement, error)
	ListMovementsByWarehouse(ctx context.Context, warehouseID int64) ([]*entity.StockMovement, error)
	ListMovementsByDateRange(ctx context.Context, from, to time.Time) ([]*entity.StockMovement, error)

	// Stock balance
	GetBalance(ctx context.Context, itemCode int64, mask string, warehouseID int64) (*entity.StockBalance, error)
	ListBalances(ctx context.Context) ([]*entity.StockBalance, error)
	ListBalancesByWarehouse(ctx context.Context, warehouseID int64) ([]*entity.StockBalance, error)
	ListBalancesByItem(ctx context.Context, itemCode int64) ([]*entity.StockBalance, error)

	// Stock reservations
	CreateReservation(ctx context.Context, r *entity.StockReservation) (*entity.StockReservation, error)
	GetReservation(ctx context.Context, id int64) (*entity.StockReservation, error)
	ListReservations(ctx context.Context) ([]*entity.StockReservation, error)
	ListMovementsPage(ctx context.Context, f entity.FiltroMovimentos) ([]*entity.StockMovement, error)
	ListReservationsByItem(ctx context.Context, itemCode int64) ([]*entity.StockReservation, error)
	ListActiveReservations(ctx context.Context) ([]*entity.StockReservation, error)
	CancelReservation(ctx context.Context, id int64) error
	ConsumeReservation(ctx context.Context, id int64) error
	// ConsumirReservaParcial consome até qtd da reserva e devolve o consumido; o
	// resto segue reservado. A nota parcial não pode soltar o que falta faturar.
	ConsumirReservaParcial(ctx context.Context, id int64, qtd decimal.Decimal) (decimal.Decimal, error)
	// HasActiveReservationByReference reports whether an originating document
	// already has active reservations, so reserving is idempotent.
	HasActiveReservationByReference(ctx context.Context, referenceType string, referenceCode int64) (bool, error)

	// Consumption average (consumo médio mensal, alimenta o ROP)
	RecalcConsumptionAverage(ctx context.Context, itemCode int64, windowMonths int) (*entity.ItemConsumptionAverage, error)
	RecalcAllConsumptionAverages(ctx context.Context, windowMonths int) (int, error)
	GetConsumptionAverage(ctx context.Context, itemCode int64) (*entity.ItemConsumptionAverage, error)

	// Lot traceability (rastreabilidade de lote/corrida)
	UpsertLot(ctx context.Context, lot *entity.StockLot) (*entity.StockLot, error)
	GetLot(ctx context.Context, itemCode int64, lot string) (*entity.StockLot, error)
	ListLotBalancesByItem(ctx context.Context, itemCode int64) ([]*entity.StockLotBalance, error)
	GetLotGenealogy(ctx context.Context, itemCode int64, lot string) (*entity.LotGenealogy, error)

	// Endereçamento e separação (migração 344)
	SugerirSeparacaoFEFO(ctx context.Context, itemCode int64, mask string, warehouseID int64, necessario decimal.Decimal, regra string) (*entity.ResultadoFEFO, error)
	ListarSaldoPorEndereco(ctx context.Context, warehouseID int64, itemCode int64) ([]*entity.StockLotBalance, error)
	TransferirEntreEnderecos(ctx context.Context, m *entity.StockMovement, origem, destino string) (*entity.StockMovement, error)
	ApurarCurvaABC(ctx context.Context, janelaMeses int, corteA, corteB float64) (*entity.ResumoABC, error)
	SugerirEnderecoDeGuarda(ctx context.Context, itemCode int64, mask string, warehouseID int64, quantidade float64, zona string) ([]*entity.SugestaoGuarda, error)
	CriarOndaDeSeparacao(ctx context.Context, codigo, warehouseID int64, regra string, necessidades []entity.NecessidadeOnda, ator uuid.UUID) (*entity.OndaDeSeparacao, error)
	// ConfirmarOndaDeSeparacao: separado diz quanto saiu de cada linha, pelo id;
	// linha fora do mapa foi separada inteira.
	ConfirmarOndaDeSeparacao(ctx context.Context, codigo int64, separado map[int64]decimal.Decimal, ator uuid.UUID) (*entity.OndaDeSeparacao, error)
	ListarOndas(ctx context.Context, status string) ([]*entity.OndaDeSeparacao, error)
	// LinhasDePedidoParaOnda: o que falta separar do pedido neste almoxarifado.
	LinhasDePedidoParaOnda(ctx context.Context, pedido, almox int64) ([]entity.LinhaPedidoOnda, error)
	CancelarOndaDeSeparacao(ctx context.Context, codigo int64, ator uuid.UUID) (*entity.OndaDeSeparacao, error)

	// Physical inventory
	CreateInventory(ctx context.Context, inv *entity.PhysicalInventory) (*entity.PhysicalInventory, error)
	GetInventory(ctx context.Context, id int64) (*entity.PhysicalInventory, error)
	GetInventoryByCode(ctx context.Context, code int64) (*entity.PhysicalInventory, error)
	ListInventories(ctx context.Context) ([]*entity.PhysicalInventory, error)
	ListInventoriesByStatus(ctx context.Context, status string) ([]*entity.PhysicalInventory, error)
	UpdateInventoryStatus(ctx context.Context, id int64, status string) error
	CloseInventory(ctx context.Context, id int64) error
	// FecharInventario lança o ajuste de cada diferença (contado − foto da
	// abertura) e fecha o inventário, numa transação.
	FecharInventario(ctx context.Context, id int64, ator uuid.UUID, zerarNaoContados bool, motivo string) (*entity.ResultadoFechamentoInventario, error)
	// ResumoDoFechamento diz o que o inventário ajustaria se fechasse agora.
	ResumoDoFechamento(ctx context.Context, inv *entity.PhysicalInventory, zerarNaoContados bool) (entity.ResumoFechamento, error)
	// EncerrarContagem leva o inventário à conferência; sem isso a contagem cega
	// não revela as diferenças.
	EncerrarContagem(ctx context.Context, id int64) error
	// MarcarRecontagem devolve à contagem as linhas escolhidas ou, sem escolha,
	// as fora da tolerância (todas com diferença, se não houver tolerância).
	MarcarRecontagem(ctx context.Context, inv *entity.PhysicalInventory, linhas []int64) (int, error)
	AguardarAprovacaoDoInventario(ctx context.Context, id int64, zerarNaoContados bool, motivo string) error
	DevolverInventarioParaConferencia(ctx context.Context, id int64) error

	// Physical inventory items
	UpsertInventoryItem(ctx context.Context, item *entity.PhysicalInventoryItem) error
	ListInventoryItems(ctx context.Context, inventoryID int64) ([]*entity.PhysicalInventoryItem, error)
	CountInventoryItem(ctx context.Context, item *entity.PhysicalInventoryItem) error
	AdjustInventoryItem(ctx context.Context, item *entity.PhysicalInventoryItem) error
}
