// Package contratovenda: contrato de venda com o cliente (comercial, Fase
// 5.2) — itens com quantidade e preço fixados, teto de valor opcional,
// vigência, programação de entregas e pedidos de chamada que consomem o saldo.
package contratovenda

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
)

const (
	Rascunho  = "RASCUNHO"
	Ativo     = "ATIVO"
	Encerrado = "ENCERRADO"
	Cancelado = "CANCELADO"
)

func Rotulo(s string) string {
	switch s {
	case Rascunho:
		return "em rascunho"
	case Ativo:
		return "ativo"
	case Encerrado:
		return "encerrado"
	case Cancelado:
		return "cancelado"
	}
	return strings.ToLower(s)
}

type Item struct {
	ID            int64           `json:"id"`
	ItemCode      int64           `json:"item_code"`
	ItemNome      string          `json:"item_nome,omitempty"`
	Quantidade    decimal.Decimal `json:"quantidade"`
	PrecoUnitario decimal.Decimal `json:"preco_unitario"`
	Consumido     decimal.Decimal `json:"consumido"`
	Saldo         decimal.Decimal `json:"saldo"`
}

type Parcela struct {
	ID             int64           `json:"id"`
	ContratoItemID int64           `json:"contrato_item_id"`
	ItemCode       int64           `json:"item_code"`
	DataPrevista   time.Time       `json:"data_prevista"`
	Quantidade     decimal.Decimal `json:"quantidade"`
	SalesOrderCode *int64          `json:"sales_order_code,omitempty"`
}

type Contrato struct {
	ID              int64            `json:"id"`
	Numero          int64            `json:"numero"`
	CustomerCode    int64            `json:"customer_code"`
	ClienteNome     string           `json:"cliente_nome,omitempty"`
	Descricao       string           `json:"descricao"`
	VigenciaInicio  time.Time        `json:"vigencia_inicio"`
	VigenciaFim     time.Time        `json:"vigencia_fim"`
	ValorContratado *decimal.Decimal `json:"valor_contratado,omitempty"`
	ValorConsumido  decimal.Decimal  `json:"valor_consumido"`
	Situacao        string           `json:"situacao"`
	AlertaSaldoPct  int              `json:"alerta_saldo_pct"`
	AlertaDias      int              `json:"alerta_dias"`
	PaymentTermCode *int64           `json:"payment_term_code,omitempty"`
	Observacao      *string          `json:"observacao,omitempty"`
	MotivoEncerra   *string          `json:"motivo_encerramento,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	Itens           []Item           `json:"itens"`
	Programacao     []Parcela        `json:"programacao"`
}

// ConferirCadastro: descrição, vigência coerente, itens sem repetição com
// quantidade positiva e preço não negativo.
func (c Contrato) ConferirCadastro() error {
	if strings.TrimSpace(c.Descricao) == "" {
		return domainerr.NewValidation("descreva o contrato")
	}
	if c.CustomerCode <= 0 {
		return domainerr.NewValidation("informe o cliente")
	}
	if c.VigenciaFim.Before(c.VigenciaInicio) {
		return domainerr.NewValidation("o fim da vigência é anterior ao início")
	}
	if c.ValorContratado != nil && !c.ValorContratado.IsPositive() {
		return domainerr.NewValidation("o valor contratado, quando informado, é positivo")
	}
	if c.AlertaSaldoPct < 0 || c.AlertaSaldoPct > 99 || c.AlertaDias < 0 || c.AlertaDias > 365 {
		return domainerr.NewValidation("alerta: saldo entre 0 e 99% e vencimento entre 0 e 365 dias")
	}
	visto := map[int64]bool{}
	for _, it := range c.Itens {
		if visto[it.ItemCode] {
			return domainerr.NewValidation("o mesmo item aparece duas vezes no contrato")
		}
		visto[it.ItemCode] = true
		if !it.Quantidade.IsPositive() {
			return domainerr.NewValidation("a quantidade contratada do item é positiva")
		}
		if it.PrecoUnitario.IsNegative() {
			return domainerr.NewValidation("o preço do item não pode ser negativo")
		}
	}
	return nil
}

// Transicao do contrato. Ativar exige item; cancelar só sem chamada.
func Transicao(c *Contrato, para string, temChamada bool) error {
	permitido := map[string][]string{Rascunho: {Ativo, Cancelado}, Ativo: {Encerrado, Cancelado}}
	ok := false
	for _, p := range permitido[c.Situacao] {
		if p == para {
			ok = true
		}
	}
	if !ok {
		return domainerr.NewValidation(fmt.Sprintf("o contrato está %s e não pode ficar %s", Rotulo(c.Situacao), Rotulo(para)))
	}
	if para == Ativo && len(c.Itens) == 0 {
		return domainerr.NewValidation("inclua ao menos um item antes de ativar o contrato")
	}
	if para == Cancelado && temChamada {
		return domainerr.NewValidation("o contrato já tem pedidos de chamada: encerre-o em vez de cancelar")
	}
	return nil
}

// LinhaChamada: o que se pede de um item na chamada.
type LinhaChamada struct {
	ContratoItemID int64           `json:"contrato_item_id"`
	Quantidade     decimal.Decimal `json:"quantidade"`
	ParcelaID      *int64          `json:"parcela_id,omitempty"`
}

// ConferirChamada: contrato ativo, data dentro da vigência, cada quantidade
// dentro do saldo do item e o total dentro do teto de valor.
func ConferirChamada(c *Contrato, linhas []LinhaChamada, data time.Time) (decimal.Decimal, error) {
	if c.Situacao != Ativo {
		return decimal.Zero, domainerr.NewValidation(fmt.Sprintf("o contrato está %s: só contrato ativo recebe pedido de chamada", Rotulo(c.Situacao)))
	}
	dia := time.Date(data.Year(), data.Month(), data.Day(), 0, 0, 0, 0, time.UTC)
	ini := time.Date(c.VigenciaInicio.Year(), c.VigenciaInicio.Month(), c.VigenciaInicio.Day(), 0, 0, 0, 0, time.UTC)
	fim := time.Date(c.VigenciaFim.Year(), c.VigenciaFim.Month(), c.VigenciaFim.Day(), 0, 0, 0, 0, time.UTC)
	if dia.Before(ini) || dia.After(fim) {
		return decimal.Zero, domainerr.NewValidation(fmt.Sprintf("a chamada de %s está fora da vigência do contrato (%s a %s)",
			dia.Format("02/01/2006"), ini.Format("02/01/2006"), fim.Format("02/01/2006")))
	}
	if len(linhas) == 0 {
		return decimal.Zero, domainerr.NewValidation("informe ao menos um item na chamada")
	}
	porID := map[int64]Item{}
	for _, it := range c.Itens {
		porID[it.ID] = it
	}
	visto := map[int64]bool{}
	total := decimal.Zero
	for _, l := range linhas {
		it, ok := porID[l.ContratoItemID]
		if !ok {
			return decimal.Zero, domainerr.NewValidation(fmt.Sprintf("o item %d não é deste contrato", l.ContratoItemID))
		}
		if visto[l.ContratoItemID] {
			return decimal.Zero, domainerr.NewValidation("o mesmo item aparece duas vezes na chamada")
		}
		visto[l.ContratoItemID] = true
		if !l.Quantidade.IsPositive() {
			return decimal.Zero, domainerr.NewValidation("a quantidade da chamada é positiva")
		}
		if l.Quantidade.GreaterThan(it.Saldo) {
			return decimal.Zero, domainerr.NewValidation(fmt.Sprintf("o item %s tem saldo de %s no contrato: a chamada pede %s", it.ItemNome, it.Saldo, l.Quantidade))
		}
		total = total.Add(l.Quantidade.Mul(it.PrecoUnitario))
	}
	total = total.Round(2)
	if c.ValorContratado != nil {
		saldoValor := c.ValorContratado.Sub(c.ValorConsumido)
		if total.GreaterThan(saldoValor) {
			return decimal.Zero, domainerr.NewValidation(fmt.Sprintf("a chamada soma R$ %s e o contrato só tem R$ %s de saldo em valor", total.StringFixed(2), saldoValor.StringFixed(2)))
		}
	}
	return total, nil
}

type Filtro struct {
	Cliente  *int64
	Situacao string
}

// Chamada: o pedido de venda montado pelo caso de uso (um item de pedido por
// linha, na mesma ordem). O repositório trava o contrato, confere o saldo de
// novo e grava pedido, itens e vínculos numa transação.
type Chamada struct {
	ContratoID int64
	Linhas     []LinhaChamada
	Data       time.Time
	Pedido     *orderentity.SalesOrder
	Itens      []*orderentity.SalesOrderItem
	Por        uuid.UUID
}

type Repository interface {
	Criar(ctx context.Context, c *Contrato, por *uuid.UUID) error
	Obter(ctx context.Context, id int64) (*Contrato, error)
	Listar(ctx context.Context, f Filtro) ([]Contrato, error)
	TemChamada(ctx context.Context, id int64) (bool, error)
	Mudar(ctx context.Context, id int64, de, para string, motivo *string, por *uuid.UUID) error
	Programar(ctx context.Context, contratoItemID int64, data time.Time, qtd decimal.Decimal) error
	RemoverParcela(ctx context.Context, parcelaID int64) error
	Chamar(ctx context.Context, c Chamada) (*orderentity.SalesOrder, error)
}
