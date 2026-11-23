// Package integracoes: adaptadores das integrações externas (cobrança,
// mensagens, transportadoras). Até haver contrato e credencial, o adaptador
// responde "integração não configurada" — a porta e a regra já existem.
package integracoes

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

// CobrancaDesligada é o PSP/banco ainda não contratado.
type CobrancaDesligada struct{}

func (CobrancaDesligada) GerarPix(context.Context, ports.CobrancaPix) (*ports.PixGerado, error) {
	return nil, ports.ErrIntegracaoDesligada
}

func (CobrancaDesligada) ConsultarCobranca(context.Context, string) (*ports.PixGerado, error) {
	return nil, ports.ErrIntegracaoDesligada
}
func (CobrancaDesligada) CancelarCobranca(context.Context, string, string) error {
	return ports.ErrIntegracaoDesligada
}
func (CobrancaDesligada) RegistrarWebhook(context.Context, string) error {
	return ports.ErrIntegracaoDesligada
}
func (CobrancaDesligada) ConsultarPagamento(context.Context, string) (*ports.PagamentoConsultado, error) {
	return nil, ports.ErrIntegracaoDesligada
}

var _ ports.CobrancaBancaria = CobrancaDesligada{}

// MensageiroDesligado é o WhatsApp/SMS ainda não contratado.
type MensageiroDesligado struct{}

func (MensageiroDesligado) Enviar(context.Context, ports.MensagemCurta) error {
	return ports.ErrIntegracaoDesligada
}

// RastreioDesligado é a API da transportadora ainda não integrada (o link de
// rastreio do cadastro da transportadora continua valendo).
type RastreioDesligado struct{}

func (RastreioDesligado) Rastrear(context.Context, string, string) ([]ports.EventoDeRastreio, error) {
	return nil, ports.ErrIntegracaoDesligada
}

var (
	_ ports.Mensageiro      = MensageiroDesligado{}
	_ ports.RastreioDeCarga = RastreioDesligado{}
)

// CotacaoDesligada é o hub de frete ainda não contratado.
type CotacaoDesligada struct{}

func (CotacaoDesligada) Cotar(context.Context, ports.PedidoDeCotacao) ([]ports.OfertaDeFrete, error) {
	return nil, ports.ErrIntegracaoDesligada
}

var _ ports.CotacaoDeFrete = CotacaoDesligada{}
