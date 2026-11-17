package fiscal_uc

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	ibptentity "github.com/FelipePn10/panossoerp/internal/domain/ibpt/entity"
)

type ibptFake map[string]*ibptentity.IBPTRate

func (f ibptFake) GetByNCM(_ context.Context, ncm, uf string) (*ibptentity.IBPTRate, error) {
	if r, ok := f[ncm+uf]; ok {
		return r, nil
	}
	return nil, errorsuc.NewNotFoundError("sem IBPT")
}

func TestTextoTransparencia(t *testing.T) {
	hoje := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	ontem := hoje.AddDate(0, 0, -1)
	chave := "ABC123"
	f := ibptFake{
		"73089010PR": {NacionalFederal: 10, ImportadoFederal: 20, Estadual: 12, Municipal: 0, Fonte: "IBPT", Chave: &chave},
		"94036000PR": {NacionalFederal: 15, Estadual: 18, VigenciaFim: &ontem}, // vencida: fora
	}
	txt, err := textoTransparencia(context.Background(), f, "pr", []linhaTransparencia{
		{ncm: "7308.90.10", origem: "0", valor: decimal.NewFromInt(1000)}, // 100 fed, 120 est
		{ncm: "73089010", origem: "1", valor: decimal.NewFromInt(500)},    // importada: 100 fed, 60 est
		{ncm: "94036000", origem: "0", valor: decimal.NewFromInt(900)},    // IBPT vencida
		{ncm: "11111111", origem: "0", valor: decimal.NewFromInt(900)},    // sem IBPT
	}, hoje)
	if err != nil {
		t.Fatal(err)
	}
	quer := "Valor aproximado dos tributos: R$ 200,00 federais, R$ 180,00 estaduais e R$ 0,00 municipais (Lei 12.741/2012). Fonte: IBPT ABC123."
	if txt != quer {
		t.Fatalf("\n got %q\nquer %q", txt, quer)
	}
	if txt, _ := textoTransparencia(context.Background(), f, "PR", []linhaTransparencia{{ncm: "11111111", valor: decimal.NewFromInt(1)}}, hoje); txt != "" {
		t.Fatalf("sem IBPT não há texto: %q", txt)
	}
}
