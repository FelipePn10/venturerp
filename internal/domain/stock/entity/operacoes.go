package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TransferenciaEstoque move um item de um almoxarifado para outro numa
// operação só: a saída da origem ao custo médio e a entrada no destino pelo
// mesmo custo, na mesma transação. Antes eram dois lançamentos manuais soltos —
// um podia ficar sem o outro, e a entrada levava o preço que a pessoa digitasse.
type TransferenciaEstoque struct {
	ItemCode     int64
	Mask         string
	DeAlmox      int64
	ParaAlmox    int64
	Quantidade   decimal.Decimal
	Lote         *string
	EnderecoDe   *string
	EnderecoPara *string
	Notas        *string
	Ator         uuid.UUID
	// Referência do documento que originou a transferência (ex.: a ordem de
	// inspeção que liberou o material). Vazia = TRANSFERENCIA, ligada pelo id da
	// saída, como na transferência manual.
	ReferenciaTipo   *string
	ReferenciaCodigo *int64
}

type ResultadoTransferencia struct {
	Saida   *StockMovement
	Entrada *StockMovement
}

// LinhaKardex é uma linha do extrato do item com o saldo acumulado.
type LinhaKardex struct {
	MovementID int64     `json:"movement_id"`
	Data       time.Time `json:"date"`
	// DataDocumento é a que vale para o saldo; Data é quando foi gravado.
	DataDocumento    string          `json:"movement_date"`
	Tipo             string          `json:"movement_type"`
	TipoCodigo       *string         `json:"movement_type_code,omitempty"`
	Referencia       *string         `json:"reference_type,omitempty"`
	ReferenciaCodigo *int64          `json:"reference_code,omitempty"`
	Almoxarifado     int64           `json:"warehouse_id"`
	Lote             *string         `json:"lot,omitempty"`
	Endereco         *string         `json:"address,omitempty"`
	Entrada          decimal.Decimal `json:"in_qty"`
	Saida            decimal.Decimal `json:"out_qty"`
	Saldo            decimal.Decimal `json:"balance_qty"`
	CustoUnitario    decimal.Decimal `json:"unit_cost"`
	Valor            decimal.Decimal `json:"value"`
	ValorSaldo       decimal.Decimal `json:"balance_value"`
	CentroCusto      *int32          `json:"cost_center_code,omitempty"`
	Estornado        bool            `json:"reversed"`
	EstornaOutro     bool            `json:"is_reversal"`
	Notas            *string         `json:"notes,omitempty"`
}

// Kardex é o extrato do item num período: saldo anterior, movimentos com o
// saldo acumulado e saldo final, em quantidade e valor.
type Kardex struct {
	ItemCode      int64           `json:"item_code"`
	Codigo        string          `json:"business_code"`
	Nome          string          `json:"name"`
	Unidade       string          `json:"uom"`
	Almoxarifado  *int64          `json:"warehouse_id,omitempty"`
	De            time.Time       `json:"from"`
	Ate           time.Time       `json:"to"`
	SaldoAnterior decimal.Decimal `json:"opening_qty"`
	ValorAnterior decimal.Decimal `json:"opening_value"`
	Entradas      decimal.Decimal `json:"total_in"`
	Saidas        decimal.Decimal `json:"total_out"`
	SaldoFinal    decimal.Decimal `json:"closing_qty"`
	ValorFinal    decimal.Decimal `json:"closing_value"`
	Linhas        []LinhaKardex   `json:"lines"`
}

// PosicaoItem é o saldo de um item num almoxarifado numa data, com a segunda
// unidade quando há conversão cadastrada.
type PosicaoItem struct {
	ItemCode           int64            `json:"item_code"`
	Codigo             string           `json:"business_code"`
	Nome               string           `json:"name"`
	Unidade            string           `json:"uom"`
	Almoxarifado       int64            `json:"warehouse_id"`
	Quantidade         decimal.Decimal  `json:"quantity"`
	Valor              decimal.Decimal  `json:"value"`
	CustoMedio         decimal.Decimal  `json:"avg_cost"`
	UnidadeAlternativa *string          `json:"alt_uom,omitempty"`
	QuantidadeAlt      *decimal.Decimal `json:"alt_quantity,omitempty"`
}

// SugestaoReposicao: item no ponto de pedido ou abaixo dele, com o quanto repor e
// de onde. WarehouseID nil é o item sem política, calculado sobre o saldo somado.
type SugestaoReposicao struct {
	ItemCode         int64           `json:"item_code"`
	Mask             string          `json:"mask"`
	Codigo           string          `json:"business_code"`
	Nome             string          `json:"name"`
	Unidade          string          `json:"uom"`
	WarehouseID      *int64          `json:"warehouse_id,omitempty"`
	Origem           string          `json:"source_type"`
	AlmoxOrigem      *int64          `json:"source_warehouse_id,omitempty"`
	Disponivel       decimal.Decimal `json:"on_hand"`
	Reservado        decimal.Decimal `json:"reserved"`
	Retido           decimal.Decimal `json:"retained"`
	Requisitado      decimal.Decimal `json:"requested"`
	EmPedido         decimal.Decimal `json:"on_order"`
	Projetado        decimal.Decimal `json:"projected"`
	PontoPedido      decimal.Decimal `json:"reorder_point"`
	EstoqueSeguranca decimal.Decimal `json:"safety_stock"`
	Maximo           decimal.Decimal `json:"maximum"`
	LoteMinimo       decimal.Decimal `json:"minimum_lot"`
	LoteMultiplo     decimal.Decimal `json:"multiple_lot"`
	ConsumoMedio     decimal.Decimal `json:"avg_monthly_consumption"`
	Sugerido         decimal.Decimal `json:"suggested_qty"`
	Motivo           string          `json:"reason"`
	Aviso            string          `json:"warning,omitempty"`
}

// AnaliseItem junta giro, cobertura, valor e tempo parado de um item.
type AnaliseItem struct {
	ItemCode           int64           `json:"item_code"`
	Codigo             string          `json:"business_code"`
	Nome               string          `json:"name"`
	Unidade            string          `json:"uom"`
	Saldo              decimal.Decimal `json:"quantity"`
	Valor              decimal.Decimal `json:"value"`
	ConsumoQtd         decimal.Decimal `json:"consumed_qty"`
	ConsumoValor       decimal.Decimal `json:"consumed_value"`
	ConsumoMedioMensal decimal.Decimal `json:"avg_monthly_consumption"`
	CoberturaMeses     *float64        `json:"coverage_months,omitempty"`
	Giro               *float64        `json:"turnover,omitempty"`
	UltimoMovimento    *time.Time      `json:"last_movement_at,omitempty"`
	DiasSemMovimento   *int            `json:"idle_days,omitempty"`
	Parado             bool            `json:"idle"`
	ClasseABC          *string         `json:"abc_class,omitempty"`
	// DemandaAberta soma pedido de venda, ordem de produção e demanda independente.
	DemandaAberta decimal.Decimal `json:"open_demand"`
	Falta         decimal.Decimal `json:"shortage"`
	// Provisao é o valor em risco pela regra de obsolescência da empresa.
	Provisao decimal.Decimal `json:"provision_value"`
}

// SeparadoDaOnda é o que uma linha da onda separou e ainda não foi faturado.
type SeparadoDaOnda struct {
	Linha      int64
	Lote       string
	Endereco   string
	Quantidade decimal.Decimal
}

// CartaoKanban é o circuito de reposição de um item num almoxarifado.
type CartaoKanban struct {
	ID              int64           `json:"id"`
	ItemCode        int64           `json:"item_code"`
	Nome            string          `json:"name"`
	WarehouseID     *int64          `json:"warehouse_id"`
	CardCount       int             `json:"card_count"`
	QuantityPerCard decimal.Decimal `json:"quantity_per_card"`
	ReorderPoint    decimal.Decimal `json:"reorder_point"`
	ContainerType   *string         `json:"container_type"`
	Status          string          `json:"status"`
	Saldo           decimal.Decimal `json:"available_qty"`
	Fabricado       bool            `json:"manufactured"`
}

// IdadeDoLote é o tempo do lote no estoque, desde o recebimento.
type IdadeDoLote struct {
	ItemCode    int64           `json:"item_code"`
	Codigo      string          `json:"business_code"`
	Nome        string          `json:"name"`
	Lote        string          `json:"lot"`
	WarehouseID int64           `json:"warehouse_id"`
	Quantidade  decimal.Decimal `json:"quantity"`
	Valor       decimal.Decimal `json:"value"`
	Recebido    *time.Time      `json:"received_at,omitempty"`
	Dias        *int            `json:"age_days,omitempty"`
}

// NiveisSugeridos é o mínimo, a segurança e o máximo calculados pelo consumo.
type NiveisSugeridos struct {
	ConsumoDiario decimal.Decimal `json:"daily_consumption"`
	PrazoDias     int             `json:"lead_time_days"`
	DiasSeguranca int             `json:"safety_days"`
	Seguranca     decimal.Decimal `json:"safety_qty"`
	Minimo        decimal.Decimal `json:"minimum_qty"`
	Maximo        decimal.Decimal `json:"maximum_qty"`
}

// SugerirNiveis: segurança são os dias de folga pelo consumo; mínimo cobre o prazo
// de entrega e a segurança; máximo soma um mês de consumo.
func SugerirNiveis(consumo decimal.Decimal, dias, prazo, diasSeguranca int) NiveisSugeridos {
	n := NiveisSugeridos{PrazoDias: prazo, DiasSeguranca: diasSeguranca}
	if dias <= 0 || !consumo.IsPositive() {
		return n
	}
	n.ConsumoDiario = consumo.DivRound(decimal.NewFromInt(int64(dias)), 6)
	n.Seguranca = n.ConsumoDiario.Mul(decimal.NewFromInt(int64(diasSeguranca))).Round(4)
	n.Minimo = n.ConsumoDiario.Mul(decimal.NewFromInt(int64(prazo))).Add(n.Seguranca).Round(4)
	n.Maximo = n.Minimo.Add(n.ConsumoDiario.Mul(decimal.NewFromInt(30))).Round(4)
	return n
}

// LoteAVencer: saldo em lote vencido ou perto de vencer.
type LoteAVencer struct {
	ItemCode     int64           `json:"item_code"`
	Codigo       string          `json:"business_code"`
	Nome         string          `json:"name"`
	Lote         string          `json:"lot"`
	Almoxarifado int64           `json:"warehouse_id"`
	Quantidade   decimal.Decimal `json:"quantity"`
	Validade     time.Time       `json:"expires_at"`
	Dias         int             `json:"days_to_expire"`
	Vencido      bool            `json:"expired"`
}

// SaldoComSituacao é o saldo de um almoxarifado dizendo se ele conta para
// promessa (ATP): almoxarifado disponível que não seja de inspeção nem rejeição.
type SaldoComSituacao struct {
	StockBalance
	Prometivel bool
	Situacao   string // LIVRE, INSPECAO, BLOQUEADO
}

// TipoMovimentoConfig é o que o cadastro de tipos de movimento (VEST0200)
// decide no lançamento manual.
type TipoMovimentoConfig struct {
	Sigla            string
	Descricao        string
	Direcao          string // ENTRADA, SAIDA, TRANSFERENCIA, AMBOS
	AtualizaCusto    bool
	ExigeCentroCusto bool
	PermiteManual    bool
	Ativo            bool
}
