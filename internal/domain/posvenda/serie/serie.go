// Package serie: número de série e base instalada (pós-venda, Fase 3.1).
//
// A série é o equipamento: nasce em estoque (produção, compra ou cadastro),
// vai ao cliente pela NF de saída, pode entrar em assistência e ser baixada.
// Cada passo fica no histórico, que não muda.
package serie

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/FelipePn10/panossoerp/internal/domain/posvenda/garantia"
)

const (
	EmEstoque     = "EM_ESTOQUE"
	NoCliente     = "NO_CLIENTE"
	EmAssistencia = "EM_ASSISTENCIA"
	Baixado       = "BAIXADO"

	OrigemProducao = "PRODUCAO"
	OrigemCompra   = "COMPRA"
	OrigemCadastro = "CADASTRO"

	EventoCadastro      = "CADASTRO"
	EventoProducao      = "PRODUCAO"
	EventoVenda         = "VENDA"
	EventoEstornoVenda  = "ESTORNO_VENDA"
	EventoInstalacao    = "INSTALACAO"
	EventoTransferencia = "TRANSFERENCIA"
	EventoBaixa         = "BAIXA"
	EventoCorrecao      = "CORRECAO"
	EventoGarantia      = "GARANTIA_ESTENDIDA"
	// A série vai à fábrica por uma ordem de serviço e volta ao cliente.
	EventoAssistencia        = "ASSISTENCIA"
	EventoRetornoAssistencia = "RETORNO_ASSISTENCIA"
	// O cliente devolveu o equipamento (devolução de venda).
	EventoDevolucao = "DEVOLUCAO_VENDA"
)

// Perfil: o que o pós-venda precisa saber do item.
type Perfil struct {
	ItemCode       int64   `json:"item_code"`
	ItemCodigo     string  `json:"item_codigo,omitempty"`
	ItemNome       string  `json:"item_nome,omitempty"`
	ControlaSerie  bool    `json:"controla_serie"`
	MascaraSerieID *int64  `json:"mascara_serie_id,omitempty"`
	Observacoes    *string `json:"observacoes,omitempty"`
	// Garantia (3.2).
	Garantia  garantia.Regra `json:"garantia"`
	UpdatedAt *time.Time     `json:"updated_at,omitempty"`
	UpdatedBy *uuid.UUID     `json:"updated_by,omitempty"`
}

type Serie struct {
	ID                int64      `json:"id"`
	ItemCode          int64      `json:"item_code"`
	ItemCodigo        string     `json:"item_codigo,omitempty"`
	ItemNome          string     `json:"item_nome,omitempty"`
	Mask              *string    `json:"mask,omitempty"`
	NumeroSerie       string     `json:"numero_serie"`
	Situacao          string     `json:"situacao"`
	Origem            string     `json:"origem"`
	WarehouseID       *int64     `json:"warehouse_id,omitempty"`
	Lote              *string    `json:"lote,omitempty"`
	ProductionOrderID *int64     `json:"production_order_id,omitempty"`
	FiscalEntryID     *int64     `json:"fiscal_entry_id,omitempty"`
	FiscalExitID      *int64     `json:"fiscal_exit_id,omitempty"`
	NumeroNF          *int64     `json:"numero_nf,omitempty"`
	CustomerCode      *int64     `json:"customer_code,omitempty"`
	ClienteNome       string     `json:"cliente_nome,omitempty"`
	DataVenda         *time.Time `json:"data_venda,omitempty"`
	DataInstalacao    *time.Time `json:"data_instalacao,omitempty"`
	LocalInstalacao   *string    `json:"local_instalacao,omitempty"`
	Observacoes       *string    `json:"observacoes,omitempty"`
	EstendidaAte      *time.Time `json:"garantia_estendida_ate,omitempty"`
	EstendidaMotivo   *string    `json:"garantia_estendida_motivo,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type Evento struct {
	ID             int64      `json:"id"`
	Evento         string     `json:"evento"`
	SituacaoAntes  *string    `json:"situacao_antes,omitempty"`
	SituacaoDepois string     `json:"situacao_depois"`
	CustomerCode   *int64     `json:"customer_code,omitempty"`
	WarehouseID    *int64     `json:"warehouse_id,omitempty"`
	ReferenciaTipo *string    `json:"referencia_tipo,omitempty"`
	ReferenciaID   *int64     `json:"referencia_id,omitempty"`
	Motivo         *string    `json:"motivo,omitempty"`
	OcorridoEm     time.Time  `json:"ocorrido_em"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
}

// Mudanca: uma transição da série, gravada com o evento na mesma operação.
type Mudanca struct {
	Evento          string
	Situacao        string
	WarehouseID     *int64
	CustomerCode    *int64
	FiscalExitID    *int64
	DataVenda       *time.Time
	DataInstalacao  *time.Time
	LocalInstalacao *string
	ReferenciaTipo  string
	ReferenciaID    *int64
	Motivo          *string
	Por             *uuid.UUID
	// LimparVenda tira cliente, nota e data de venda (estorno da venda).
	LimparVenda bool
}

type Filtro struct {
	ItemCode     *int64
	CustomerCode *int64
	Situacao     string
	Busca        string
	Limite       int
}

// LinhaNota: uma linha da NF de saída de item com série.
type LinhaNota struct {
	FiscalExitItemID int64           `json:"fiscal_exit_item_id"`
	ItemCode         int64           `json:"item_code"`
	Quantidade       decimal.Decimal `json:"quantidade"`
	Series           []Serie         `json:"series"`
}

type Repository interface {
	Perfil(ctx context.Context, itemCode int64) (*Perfil, error)
	SalvarPerfil(ctx context.Context, p *Perfil) error
	Perfis(ctx context.Context) ([]Perfil, error)

	Obter(ctx context.Context, id int64) (*Serie, error)
	Listar(ctx context.Context, f Filtro) ([]Serie, error)
	Historico(ctx context.Context, id int64) ([]Evento, error)
	Cadastrar(ctx context.Context, s *Serie, por *uuid.UUID, motivo *string) error
	Mudar(ctx context.Context, id int64, m Mudanca) (*Serie, error)

	// Nota de saída.
	LinhasComSerie(ctx context.Context, fiscalExitID int64) ([]LinhaNota, error)
	DefinirSeriesDaLinha(ctx context.Context, fiscalExitID, fiscalExitItemID int64, series []int64, por *uuid.UUID) error

	// Garantia.
	PorNumero(ctx context.Context, itemCode int64, numero string) (*Serie, error)
	NotaDoCliente(ctx context.Context, cliente int64, numeroNF int64) (*NotaVendida, error)
	EstenderGarantia(ctx context.Context, id int64, ate time.Time, motivo string, por *uuid.UUID) error
}

// NotaVendida: a NF de saída autorizada que vendeu a peça ao cliente.
type NotaVendida struct {
	FiscalExitID int64     `json:"fiscal_exit_id"`
	NumeroNF     int64     `json:"numero_nf"`
	Emissao      time.Time `json:"emissao"`
}

// Transicao confere se a série pode ir de uma situação para outra pelo evento.
func Transicao(atual, evento string) (string, error) {
	invalida := func() (string, error) {
		return "", domainerr.NewValidation(fmt.Sprintf("a série está %s e não aceita %s", Rotulo(atual), strings.ToLower(RotuloEvento(evento))))
	}
	switch evento {
	case EventoVenda:
		if atual != EmEstoque {
			return invalida()
		}
		return NoCliente, nil
	case EventoEstornoVenda, EventoDevolucao:
		if atual != NoCliente {
			return invalida()
		}
		return EmEstoque, nil
	case EventoInstalacao, EventoTransferencia:
		if atual != NoCliente {
			return invalida()
		}
		return NoCliente, nil
	case EventoAssistencia:
		if atual != NoCliente {
			return invalida()
		}
		return EmAssistencia, nil
	case EventoRetornoAssistencia:
		if atual != EmAssistencia {
			return invalida()
		}
		return NoCliente, nil
	case EventoBaixa:
		if atual == Baixado {
			return invalida()
		}
		return Baixado, nil
	}
	return "", fmt.Errorf("evento de série desconhecido: %s", evento)
}

func Rotulo(situacao string) string {
	switch situacao {
	case EmEstoque:
		return "em estoque"
	case NoCliente:
		return "no cliente"
	case EmAssistencia:
		return "em assistência"
	case Baixado:
		return "baixada"
	}
	return strings.ToLower(situacao)
}

func RotuloEvento(evento string) string {
	switch evento {
	case EventoVenda:
		return "Venda"
	case EventoEstornoVenda:
		return "Estorno da venda"
	case EventoInstalacao:
		return "Instalação"
	case EventoTransferencia:
		return "Transferência de cliente"
	case EventoBaixa:
		return "Baixa"
	case EventoDevolucao:
		return "Devolução de venda"
	case EventoAssistencia:
		return "Entrada na assistência"
	case EventoRetornoAssistencia:
		return "Retorno da assistência"
	}
	return evento
}

// ConferirLista limpa a lista de séries informada para uma quantidade: sem
// vazio, sem repetida e uma série por unidade (quantidade inteira).
func ConferirLista(series []string, quantidade decimal.Decimal) ([]string, error) {
	if !quantidade.Equal(quantidade.Truncate(0)) {
		return nil, domainerr.NewValidation(fmt.Sprintf("item com número de série só aceita quantidade inteira (informado %s)", quantidade.String()))
	}
	out := make([]string, 0, len(series))
	vistas := map[string]bool{}
	for _, s := range series {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		chave := strings.ToUpper(s)
		if vistas[chave] {
			return nil, domainerr.NewValidation(fmt.Sprintf("a série %s foi informada duas vezes", s))
		}
		vistas[chave] = true
		out = append(out, s)
	}
	if int64(len(out)) != quantidade.IntPart() {
		return nil, domainerr.NewValidation(fmt.Sprintf("informe uma série por unidade: %s unidade(s) e %d série(s)", quantidade.String(), len(out)))
	}
	return out, nil
}

// PendenciasDaNota: o que impede a nota de sair por causa de série (prévia).
func PendenciasDaNota(linhas []LinhaNota) []string {
	var out []string
	for _, l := range linhas {
		qtd := l.Quantidade
		if !qtd.Equal(qtd.Truncate(0)) {
			out = append(out, fmt.Sprintf("o item %d controla número de série e a quantidade da nota não é inteira (%s)", l.ItemCode, qtd.String()))
			continue
		}
		if int64(len(l.Series)) != qtd.IntPart() {
			out = append(out, fmt.Sprintf("o item %d controla número de série: escolha %s série(s) para a linha (escolhidas: %d)", l.ItemCode, qtd.String(), len(l.Series)))
			continue
		}
		for _, s := range l.Series {
			if s.Situacao != EmEstoque {
				out = append(out, fmt.Sprintf("a série %s do item %d está %s", s.NumeroSerie, l.ItemCode, Rotulo(s.Situacao)))
			}
		}
	}
	return out
}
