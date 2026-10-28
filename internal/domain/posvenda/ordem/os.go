// Package ordem: ordem de serviço da assistência técnica (pós-venda, Fase 3.3).
//
// A OS nasce de um chamado, vai ao cliente (técnico externo) ou acontece na
// fábrica (o produto volta), recebe horas, peças e despesas e termina
// concluída e, se houver o que cobrar, faturada. O que a garantia cobre não é
// cobrado — é custo de garantia.
package ordem

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
	LocalCliente = "CLIENTE"
	LocalFabrica = "FABRICA"

	Aberta     = "ABERTA"
	Agendada   = "AGENDADA"
	EmExecucao = "EM_EXECUCAO"
	Concluida  = "CONCLUIDA"
	Faturada   = "FATURADA"
	Cancelada  = "CANCELADA"

	HoraTrabalho     = "TRABALHO"
	HoraDeslocamento = "DESLOCAMENTO"

	DespesaKM = "KM"
)

// Cobertura: o que a garantia do item do chamado cobre nesta OS.
type Cobertura struct {
	EmGarantia        bool `json:"em_garantia"`
	CobrePecas        bool `json:"cobre_pecas"`
	CobreMaoObra      bool `json:"cobre_mao_obra"`
	CobreDeslocamento bool `json:"cobre_deslocamento"`
}

// PecaCobrada: peça é garantia quando a OS está em garantia e a cobertura
// inclui peças.
func (c Cobertura) PecaCobrada() bool { return !(c.EmGarantia && c.CobrePecas) }

// HoraCobrada: trabalho segue a mão de obra; deslocamento, o deslocamento.
func (c Cobertura) HoraCobrada(tipo string) bool {
	if tipo == HoraDeslocamento {
		return !(c.EmGarantia && c.CobreDeslocamento)
	}
	return !(c.EmGarantia && c.CobreMaoObra)
}

// DespesaCobrada: despesa de viagem segue o deslocamento.
func (c Cobertura) DespesaCobrada() bool { return !(c.EmGarantia && c.CobreDeslocamento) }

// Transicao confere a mudança de situação da OS.
func Transicao(de, para string) error {
	permitido := map[string][]string{
		Aberta:     {Agendada, EmExecucao, Cancelada},
		Agendada:   {Agendada, EmExecucao, Cancelada},
		EmExecucao: {Concluida, Cancelada},
		Concluida:  {Faturada, EmExecucao},
	}
	for _, p := range permitido[de] {
		if p == para {
			return nil
		}
	}
	return domainerr.NewValidation(fmt.Sprintf("a OS está %s e não pode ir para %s", Rotulo(de), Rotulo(para)))
}

func Rotulo(s string) string {
	switch s {
	case Aberta:
		return "aberta"
	case Agendada:
		return "agendada"
	case EmExecucao:
		return "em execução"
	case Concluida:
		return "concluída"
	case Faturada:
		return "faturada"
	case Cancelada:
		return "cancelada"
	}
	return strings.ToLower(s)
}

// Lancamentos aceita horas, peças e despesas.
func Lancamentos(situacao string) error {
	if situacao != Agendada && situacao != EmExecucao && situacao != Aberta {
		return domainerr.NewValidation(fmt.Sprintf("a OS está %s: horas, peças e despesas só entram com a OS aberta, agendada ou em execução", Rotulo(situacao)))
	}
	return nil
}

// Valores: custo e preço de cada lançamento.
type Linha struct {
	Custo   decimal.Decimal
	Preco   decimal.Decimal
	Cobrado bool
}

// Totais: o custo total da OS, o que vai faturado e o custo de garantia.
type Totais struct {
	CustoHoras    decimal.Decimal `json:"custo_horas"`
	CustoPecas    decimal.Decimal `json:"custo_pecas"`
	CustoDespesas decimal.Decimal `json:"custo_despesas"`
	CustoTotal    decimal.Decimal `json:"custo_total"`
	ACobrar       decimal.Decimal `json:"a_cobrar"`
	CustoGarantia decimal.Decimal `json:"custo_garantia"`
	HorasTrabalho decimal.Decimal `json:"horas_trabalho"`
	HorasViagem   decimal.Decimal `json:"horas_deslocamento"`
}

func Somar(horas, pecas, despesas []Linha) Totais {
	var t Totais
	soma := func(ls []Linha, custo *decimal.Decimal) {
		for _, l := range ls {
			*custo = custo.Add(l.Custo)
			if l.Cobrado {
				t.ACobrar = t.ACobrar.Add(l.Preco)
			} else {
				t.CustoGarantia = t.CustoGarantia.Add(l.Custo)
			}
		}
	}
	soma(horas, &t.CustoHoras)
	soma(pecas, &t.CustoPecas)
	soma(despesas, &t.CustoDespesas)
	t.CustoTotal = t.CustoHoras.Add(t.CustoPecas).Add(t.CustoDespesas)
	return t
}

// Conclusao exige o defeito constatado e a solução.
func Conclusao(defeito, solucao string) error {
	if strings.TrimSpace(defeito) == "" || strings.TrimSpace(solucao) == "" {
		return domainerr.NewValidation("para concluir a OS informe o defeito constatado e a solução aplicada")
	}
	return nil
}

// Agenda não aceita data no passado distante (erro de digitação).
func Agenda(quando, agora time.Time) error {
	if quando.Before(agora.AddDate(0, 0, -1)) {
		return domainerr.NewValidation("a data do agendamento já passou")
	}
	return nil
}

type Parametros struct {
	CustoHora       decimal.Decimal `json:"custo_hora"`
	PrecoHora       decimal.Decimal `json:"preco_hora"`
	CustoKM         decimal.Decimal `json:"custo_km"`
	PrecoKM         decimal.Decimal `json:"preco_km"`
	WarehouseID     *int64          `json:"warehouse_id,omitempty"`
	ItemServicoCode *int64          `json:"-"`
	// ItemServico: código de negócio do item de serviço (o que a tela usa).
	ItemServico     string `json:"item_servico"`
	ItemServicoNome string `json:"item_servico_nome"`
}

type OS struct {
	ID           int64      `json:"id"`
	Numero       int64      `json:"numero"`
	CallCode     int64      `json:"call_code"`
	CallNumero   *int64     `json:"call_number,omitempty"`
	CallItemCode *int64     `json:"call_item_code,omitempty"`
	SerieID      *int64     `json:"serie_id,omitempty"`
	NumeroSerie  *string    `json:"numero_serie,omitempty"`
	ItemCode     *int64     `json:"item_code,omitempty"`
	ItemNome     string     `json:"item_nome,omitempty"`
	CustomerCode int64      `json:"customer_code"`
	ClienteNome  string     `json:"cliente_nome,omitempty"`
	Local        string     `json:"local"`
	Situacao     string     `json:"situacao"`
	TecnicoCode  *int64     `json:"tecnico_code,omitempty"`
	TecnicoNome  string     `json:"tecnico_nome,omitempty"`
	AgendadaPara *time.Time `json:"agendada_para,omitempty"`
	IniciadaEm   *time.Time `json:"iniciada_em,omitempty"`
	ConcluidaEm  *time.Time `json:"concluida_em,omitempty"`
	Cobertura
	DefeitoConstatado  *string   `json:"defeito_constatado,omitempty"`
	Solucao            *string   `json:"solucao,omitempty"`
	Observacoes        *string   `json:"observacoes,omitempty"`
	MotivoCancelamento *string   `json:"motivo_cancelamento,omitempty"`
	SalesOrderCode     *int64    `json:"sales_order_code,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type Hora struct {
	ID          int64           `json:"id"`
	TecnicoCode *int64          `json:"tecnico_code,omitempty"`
	Data        time.Time       `json:"data"`
	Tipo        string          `json:"tipo"`
	Horas       decimal.Decimal `json:"horas"`
	CustoHora   decimal.Decimal `json:"custo_hora"`
	PrecoHora   decimal.Decimal `json:"preco_hora"`
	Cobrado     bool            `json:"cobrado"`
	Observacao  *string         `json:"observacao,omitempty"`
}

type Peca struct {
	ID              int64           `json:"id"`
	ItemCode        int64           `json:"item_code"`
	ItemNome        string          `json:"item_nome,omitempty"`
	Quantidade      decimal.Decimal `json:"quantidade"`
	WarehouseID     int64           `json:"warehouse_id"`
	CustoUnitario   decimal.Decimal `json:"custo_unitario"`
	PrecoUnitario   decimal.Decimal `json:"preco_unitario"`
	Cobrado         bool            `json:"cobrado"`
	StockMovementID *int64          `json:"stock_movement_id,omitempty"`
	Observacao      *string         `json:"observacao,omitempty"`
}

type Despesa struct {
	ID         int64           `json:"id"`
	Tipo       string          `json:"tipo"`
	Quantidade decimal.Decimal `json:"quantidade"`
	Custo      decimal.Decimal `json:"custo"`
	Preco      decimal.Decimal `json:"preco"`
	Cobrado    bool            `json:"cobrado"`
	Observacao *string         `json:"observacao,omitempty"`
}

type Evento struct {
	ID         int64      `json:"id"`
	Evento     string     `json:"evento"`
	De         *string    `json:"de,omitempty"`
	Para       *string    `json:"para,omitempty"`
	Detalhe    *string    `json:"detalhe,omitempty"`
	OcorridoEm time.Time  `json:"ocorrido_em"`
	CreatedBy  *uuid.UUID `json:"created_by,omitempty"`
}

type Filtro struct {
	Situacao string
	Cliente  *int64
	Chamado  *int64
	Tecnico  *int64
}

// Chamado: o que a OS herda do chamado e do item (cliente, série, garantia).
type Chamado struct {
	CallCode     int64
	CustomerCode int64
	CallItemCode *int64
	ItemCode     *int64
	SerieID      *int64
	Cobertura    Cobertura
}

// Mudanca: troca de situação e os campos que vão junto.
type Mudanca struct {
	Para               string
	TecnicoCode        *int64
	AgendadaPara       *time.Time
	DefeitoConstatado  *string
	Solucao            *string
	MotivoCancelamento *string
	SalesOrderCode     *int64
	Detalhe            *string
	Por                *uuid.UUID
	// SerieEmAssistencia: true leva a série da OS para "em assistência"
	// (produto na fábrica), false a devolve ao cliente; nil não mexe.
	SerieEmAssistencia *bool
}

// Faturamento: o pedido de venda que fatura a OS e a mudança para FATURADA
// (o código do pedido é preenchido pelo repositório).
type Faturamento struct {
	Pedido  *orderentity.SalesOrder
	Itens   []*orderentity.SalesOrderItem
	Mudanca Mudanca
}

// Repository: a OS e seus lançamentos, da empresa da sessão.
type Repository interface {
	Parametros(ctx context.Context) (*Parametros, error)
	SalvarParametros(ctx context.Context, p *Parametros, por *uuid.UUID) error
	DoChamado(ctx context.Context, callCode int64, itemCode *int64) (*Chamado, error)
	Abrir(ctx context.Context, o *OS, por *uuid.UUID) error
	Obter(ctx context.Context, id int64) (*OS, error)
	Listar(ctx context.Context, f Filtro) ([]OS, error)
	Mudar(ctx context.Context, id int64, m Mudanca) error
	// Faturar grava, numa transação com a OS travada, o pedido de venda de
	// faturamento (número, capa, linhas e totais) e a passagem a FATURADA.
	Faturar(ctx context.Context, id int64, f Faturamento) (*orderentity.SalesOrder, error)
	AddHora(ctx context.Context, osID int64, h *Hora, por *uuid.UUID) error
	AddDespesa(ctx context.Context, osID int64, d *Despesa, por *uuid.UUID) error
	AddPeca(ctx context.Context, osID int64, p *Peca, por uuid.UUID) error
	RemoverLancamento(ctx context.Context, osID int64, tipo string, id int64, por uuid.UUID) error
	Lancamentos(ctx context.Context, osID int64) ([]Hora, []Peca, []Despesa, []Evento, error)
}
