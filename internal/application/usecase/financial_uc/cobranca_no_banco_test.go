package financial_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

type bancoRegistra struct {
	cancelados []string
	falha      error
}

func (b *bancoRegistra) GerarPix(context.Context, ports.CobrancaPix) (*ports.PixGerado, error) {
	return nil, nil
}
func (b *bancoRegistra) ConsultarCobranca(context.Context, string) (*ports.PixGerado, error) {
	return nil, nil
}
func (b *bancoRegistra) ConsultarPagamento(context.Context, string) (*ports.PagamentoConsultado, error) {
	return nil, nil
}
func (b *bancoRegistra) CancelarCobranca(_ context.Context, id, _ string) error {
	if b.falha != nil {
		return b.falha
	}
	b.cancelados = append(b.cancelados, id)
	return nil
}
func (b *bancoRegistra) RegistrarWebhook(context.Context, string) error { return nil }

type titulosComCobranca map[int64]string

func (t titulosComCobranca) CobrancaDoTitulo(_ context.Context, id int64) (string, error) {
	return t[id], nil
}

// Título sem cobrança no banco: nada a cancelar. Com cobrança: cancela a dela.
// Integração desligada não impede; falha do banco volta ao chamador.
func TestCobrancaNoBancoCancela(t *testing.T) {
	b := &bancoRegistra{}
	c := &CobrancaNoBanco{Cobranca: b, Titulos: titulosComCobranca{7: "inter-7"}}
	if err := c.Cancelar(context.Background(), 8, "x"); err != nil || len(b.cancelados) != 0 {
		t.Fatalf("título sem cobrança: %v %v", err, b.cancelados)
	}
	if err := c.Cancelar(context.Background(), 7, "x"); err != nil || len(b.cancelados) != 1 || b.cancelados[0] != "inter-7" {
		t.Fatalf("cancelamento: %v %v", err, b.cancelados)
	}
	b.falha = ports.ErrIntegracaoDesligada
	if err := c.Cancelar(context.Background(), 7, "x"); err != nil {
		t.Fatalf("integração desligada não pode impedir: %v", err)
	}
	b.falha = errors.New("HTTP 500")
	if err := c.Cancelar(context.Background(), 7, "x"); err == nil {
		t.Fatal("falha do banco precisa voltar ao chamador")
	}
	var nada *CobrancaNoBanco
	if err := nada.Cancelar(context.Background(), 7, "x"); err != nil {
		t.Fatal("sem banco configurado no caso de uso, não faz nada")
	}
}
