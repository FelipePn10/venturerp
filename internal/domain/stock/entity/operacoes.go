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
	MovementID       int64           `json:"movement_id"`
	Data             time.Time       `json:"date"`
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

// SugestaoReposicao: item abaixo do ponto de pedido (ou do estoque mínimo),
// com o quanto comprar respeitando lote mínimo e múltiplo.
type SugestaoReposicao struct {
	ItemCode         int64           `json:"item_code"`
	Codigo           string          `json:"business_code"`
	Nome             string          `json:"name"`
	Unidade          string          `json:"uom"`
	Disponivel       decimal.Decimal `json:"on_hand"`
	Reservado        decimal.Decimal `json:"reserved"`
	EmPedido         decimal.Decimal `json:"on_order"`
	Projetado        decimal.Decimal `json:"projected"`
	PontoPedido      decimal.Decimal `json:"reorder_point"`
	EstoqueSeguranca decimal.Decimal `json:"safety_stock"`
	LoteMinimo       decimal.Decimal `json:"minimum_lot"`
	LoteMultiplo     decimal.Decimal `json:"multiple_lot"`
	ConsumoMedio     decimal.Decimal `json:"avg_monthly_consumption"`
	Sugerido         decimal.Decimal `json:"suggested_qty"`
	Motivo           string          `json:"reason"`
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
