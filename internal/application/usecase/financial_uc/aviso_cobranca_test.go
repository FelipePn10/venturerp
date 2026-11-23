package financial_uc

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
)

func TestTextoDaCobranca(t *testing.T) {
	doc, linha, pix := "NF 10/1", "34191.79001 01043.510047", "00020126PIX"
	txt := TextoDaCobranca(&entity.ContaReceber{ID: 3, NumeroDocumento: &doc, ValorBruto: decimal.RequireFromString("360.50"),
		ValorRecebido: decimal.RequireFromString("60.50"), DataVencimento: time.Date(2026, 12, 5, 0, 0, 0, 0, time.UTC), LinhaDigitavel: &linha, ChavePixGerada: &pix})
	for _, want := range []string{"NF 10/1", "R$ 300,00", "05/12/2026", linha, pix} {
		if !strings.Contains(txt, want) {
			t.Errorf("faltou %q em:\n%s", want, txt)
		}
	}
}
