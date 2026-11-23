package ports

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

// ErrIntegracaoDesligada: o adaptador externo ainda não foi contratado/configurado.
var ErrIntegracaoDesligada = errors.New("integração não configurada nesta instalação")

// CobrancaPix é o título a cobrar (PIX, ou boleto com PIX no banco que emite os dois).
type CobrancaPix struct {
	TituloID   int64
	SeuNumero  string // identificador do título no banco (até 15 caracteres)
	Documento  string
	Valor      decimal.Decimal
	Vencimento time.Time
	Pagador    Pagador
}

// Pagador é quem paga o título (cadastro do cliente).
type Pagador struct {
	Nome, Documento, Email          string
	Logradouro, Numero, Complemento string
	Bairro, Cidade, UF, CEP         string
}

// PixGerado é a cobrança criada no banco/PSP.
type PixGerado struct {
	IDExterno      string     `json:"external_id"` // o identificador da cobrança no banco (ex.: codigoSolicitacao)
	TxID           string     `json:"txid,omitempty"`
	CopiaECola     string     `json:"pix_copy_paste,omitempty"`
	NossoNumero    string     `json:"our_number,omitempty"`
	LinhaDigitavel string     `json:"digitable_line,omitempty"`
	CodigoBarras   string     `json:"barcode,omitempty"`
	Expira         *time.Time `json:"expires_at,omitempty"`
	// Pendente: o banco ainda processa a emissão; os códigos chegam depois.
	Pendente bool `json:"pending"`
	// Situacao: a do banco (A_RECEBER, RECEBIDO, CANCELADO…), quando consultada.
	Situacao string `json:"bank_status,omitempty"`
}

// PagamentoConsultado é a situação da cobrança no banco.
type PagamentoConsultado struct {
	IDExterno, SeuNumero string
	Pago                 bool
	Valor                decimal.Decimal
	Data                 time.Time
	Forma                string // BOLETO ou PIX
	Situacao             string
}

// CobrancaBancaria emite a cobrança no banco/PSP e consulta se foi paga. A baixa
// do que foi pago volta pelo retorno (CNAB) ou pelo aviso do banco (webhook),
// sempre confirmada pela consulta.
type CobrancaBancaria interface {
	GerarPix(ctx context.Context, c CobrancaPix) (*PixGerado, error)
	// ConsultarCobranca relê a cobrança já emitida (pelo IDExterno).
	ConsultarCobranca(ctx context.Context, idExterno string) (*PixGerado, error)
	ConsultarPagamento(ctx context.Context, idExterno string) (*PagamentoConsultado, error)
	// CancelarCobranca tira do banco a cobrança do título cancelado ou recebido por outro meio.
	CancelarCobranca(ctx context.Context, idExterno, motivo string) error
	// RegistrarWebhook cadastra a URL que o banco chama ao receber.
	RegistrarWebhook(ctx context.Context, url string) error
}

// MensagemCurta é uma mensagem de WhatsApp/SMS.
type MensagemCurta struct {
	Para  string // telefone com DDI e DDD, só dígitos
	Texto string
}

// Mensageiro envia mensagem curta (WhatsApp Business/SMS).
type Mensageiro interface {
	Enviar(ctx context.Context, m MensagemCurta) error
}

// EventoDeRastreio é uma posição da carga na transportadora.
type EventoDeRastreio struct {
	Quando   time.Time `json:"at"`
	Situacao string    `json:"status"`
	Local    string    `json:"place,omitempty"`
}

// RastreioDeCarga consulta a carga na transportadora (API dela).
type RastreioDeCarga interface {
	Rastrear(ctx context.Context, transportadoraCNPJ, documento string) ([]EventoDeRastreio, error)
}

// VolumeDeFrete é um volume da carga (medidas em metros, peso em kg).
type VolumeDeFrete struct {
	Quantidade            int
	Altura, Largura, Comp decimal.Decimal
	PesoKg, ValorUnitario decimal.Decimal
}

// PedidoDeCotacao é a carga a cotar.
type PedidoDeCotacao struct {
	CNPJEmbarcador, CEPOrigem   string
	DocDestinatario, CEPDestino string
	DestinatarioPJ              bool
	ValorNota                   decimal.Decimal
	Volumes                     []VolumeDeFrete
}

// OfertaDeFrete é uma cotação devolvida pela transportadora/hub.
type OfertaDeFrete struct {
	Transportadora  string          `json:"carrier"`
	CNPJ            string          `json:"carrier_document"`
	Servico         string          `json:"service"`
	PrazoDias       int             `json:"delivery_days"`
	PrevisaoEntrega string          `json:"estimated_date"`
	Preco           decimal.Decimal `json:"price"`
	Validade        *time.Time      `json:"expires_at,omitempty"`
}

// CotacaoDeFrete cota a carga nas transportadoras (hub de frete).
type CotacaoDeFrete interface {
	Cotar(ctx context.Context, p PedidoDeCotacao) ([]OfertaDeFrete, error)
}
