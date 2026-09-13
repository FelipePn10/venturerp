package entity

import (
	"time"

	"github.com/google/uuid"
)

// StockLot is the registry of a raw-material lot: its supplier lot, heat number
// (corrida) and quality certificate — the traceability metadata a metallurgy
// shop must keep to answer "which heat went into this part?".
// As tags são o contrato da API: esta entidade é devolvida direto pelo handler.
// Sem elas os campos saem em PascalCase e o middleware nem reconhece
// `item_code` para traduzir ao código público.
type StockLot struct {
	ID           int64      `json:"id"`
	ItemCode     int64      `json:"item_code"`
	Mask         string     `json:"mask"`
	Lot          string     `json:"lot"`
	HeatNumber   *string    `json:"heat_number"`
	Certificate  *string    `json:"certificate"`
	SupplierCode *int64     `json:"supplier_code"`
	ReceivedAt   *time.Time `json:"received_at"`
	// ExpiresAt ordena o FEFO. Sem validade o lote vai para o fim da fila e a
	// ordenação cai para a data de recebimento (FIFO).
	ExpiresAt *time.Time `json:"expires_at"`
	Notes     *string    `json:"notes"`
	CreatedAt time.Time  `json:"created_at"`
	CreatedBy uuid.UUID  `json:"created_by"`
}

// StockLotBalance is the on-hand quantity of a single lot in a warehouse.
type StockLotBalance struct {
	ID             int64      `json:"id"`
	ItemCode       int64      `json:"item_code"`
	Mask           string     `json:"mask"`
	WarehouseID    int64      `json:"warehouse_id"`
	Lot            string     `json:"lot"`
	Address        string     `json:"address"`
	Quantity       float64    `json:"quantity"`
	LastCost       float64    `json:"last_cost"`
	LastMovementAt *time.Time `json:"last_movement_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// LotGenealogy is the full traceability of an item lot, in both directions:
// where a raw-material lot was consumed, and what input lots a produced lot is
// made of.
type LotGenealogy struct {
	ItemCode   int64
	Lot        string
	Registry   *StockLot
	Balances   []*StockLotBalance
	ConsumedIn []LotConsumption // production orders that consumed this lot
	ProducedBy []LotProduction  // production orders that produced this lot
}

// LotConsumption is a production order that consumed the queried lot.
type LotConsumption struct {
	ProductionOrderID int64
	OrderNumber       int64
	ProducedItemCode  int64
	ConsumedQty       float64
}

// LotProduction is a production order that produced the queried lot, together
// with the input lots that went into it.
type LotProduction struct {
	ProductionOrderID int64
	OrderNumber       int64
	ProducedQty       float64
	InputLots         []LotInput
}

// LotInput is a raw-material lot consumed by a production order.
type LotInput struct {
	ItemCode    int64
	Lot         string
	ConsumedQty float64
}

// SugestaoFEFO é uma linha da sugestão de separação: qual lote consumir, de que
// endereço e quanto. A ordem é a da regra escolhida (FEFO ou FIFO); a última
// linha pode vir parcial quando o saldo não cobre a necessidade.
type SugestaoFEFO struct {
	Lot         string     `json:"lot"`
	Address     string     `json:"address"`
	WarehouseID int64      `json:"warehouse_id"`
	HeatNumber  *string    `json:"heat_number"`
	Certificate *string    `json:"certificate"`
	ExpiresAt   *time.Time `json:"expires_at"`
	ReceivedAt  *time.Time `json:"received_at"`
	Disponivel  float64    `json:"available_qty"`
	Sugerido    float64    `json:"suggested_qty"`
	Zone        string     `json:"zone"`
	// Ordem física do endereço na rota. O QUE separar é decidido pelo FEFO; a
	// ORDEM em que se caminha é esta — sem ela o separador cruza o galpão de um
	// lado para o outro a cada linha.
	PickSequence int `json:"pick_sequence"`
}

// ResultadoFEFO reúne a sugestão e o que faltou atender.
type ResultadoFEFO struct {
	ItemCode     int64          `json:"item_code"`
	Mask         string         `json:"mask"`
	Regra        string         `json:"rule"`
	Necessario   float64        `json:"required_qty"`
	Atendido     float64        `json:"covered_qty"`
	EmFalta      float64        `json:"missing_qty"`
	Linhas       []SugestaoFEFO `json:"lines"`
	VencidosFora int            `json:"expired_skipped"`
	// Endereço bloqueado (inventário, avaria, quarentena): o saldo existe mas
	// não pode sair. Contado à parte para a tela explicar a falta.
	BloqueadosFora int `json:"blocked_skipped"`
}

// ClasseABC é o resultado da apuração da curva ABC de um item.
type ClasseABC struct {
	ItemCode        int64   `json:"item_code"`
	Classe          string  `json:"abc_class"`
	ValorConsumido  float64 `json:"consumption_value"`
	ParticipacaoPct float64 `json:"share_pct"`
	AcumuladoPct    float64 `json:"cumulative_pct"`
}

// ResumoABC devolve a apuração inteira, para a tela explicar a classe em vez de
// mostrar só a letra.
type ResumoABC struct {
	JanelaMeses   int         `json:"window_months"`
	ValorTotal    float64     `json:"total_value"`
	Itens         []ClasseABC `json:"items"`
	CorteA        float64     `json:"cut_a_pct"`
	CorteB        float64     `json:"cut_b_pct"`
	Classificados int         `json:"classified"`
}

// SugestaoGuarda é uma recomendação de endereço para guardar material.
type SugestaoGuarda struct {
	Address    string   `json:"address"`
	Zone       string   `json:"zone"`
	Motivo     string   `json:"reason"`
	SaldoAtual float64  `json:"current_qty"`
	Capacidade *float64 `json:"capacity"`
	Cabe       bool     `json:"fits"`
	PickSeq    int      `json:"pick_sequence"`
}

// NecessidadeOnda é uma linha de demanda entregue à onda.
type NecessidadeOnda struct {
	ItemCode      int64   `json:"item_code"`
	Mask          string  `json:"mask"`
	Quantity      float64 `json:"quantity"`
	ReferenceType *string `json:"reference_type,omitempty"`
	ReferenceCode *int64  `json:"reference_code,omitempty"`
}

// LinhaOnda é uma parada do separador: um lote, num endereço, com a quantidade.
type LinhaOnda struct {
	ID            int64      `json:"id"`
	ItemCode      int64      `json:"item_code"`
	Mask          string     `json:"mask"`
	Lot           string     `json:"lot"`
	Address       string     `json:"address"`
	Zone          string     `json:"zone"`
	PickSequence  int        `json:"pick_sequence"`
	Quantity      float64    `json:"quantity"`
	PickedQty     float64    `json:"picked_qty"`
	HeatNumber    *string    `json:"heat_number"`
	ExpiresAt     *time.Time `json:"expires_at"`
	ReferenceType *string    `json:"reference_type"`
	ReferenceCode *int64     `json:"reference_code"`
}

// OndaDeSeparacao agrupa várias necessidades numa caminhada só.
type OndaDeSeparacao struct {
	ID          int64       `json:"id"`
	Code        int64       `json:"code"`
	WarehouseID int64       `json:"warehouse_id"`
	Status      string      `json:"status"`
	Rule        string      `json:"rule"`
	Linhas      []LinhaOnda `json:"lines"`
	// O que a onda não conseguiu cobrir. Lista, e não mapa: chave de mapa não
	// passa pela tradução de código de item, então a tela recebia a chave
	// interna (11) no lugar do código que o usuário conhece.
	EmFalta   []FaltaNaOnda `json:"missing"`
	CreatedAt time.Time     `json:"created_at"`
}

// FaltaNaOnda é o que a onda não conseguiu alocar de um item.
type FaltaNaOnda struct {
	ItemCode int64   `json:"item_code"`
	Quantity float64 `json:"quantity"`
}
