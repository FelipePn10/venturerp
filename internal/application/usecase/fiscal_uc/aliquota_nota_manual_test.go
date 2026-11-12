package fiscal_uc

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entrada"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// A nota digitada à mão traz a alíquota como na nota (18 = 18%); a conferência
// faz base × alíquota em fração. Sem a conversão, 22.500 × 18 acusava um ICMS
// de 405.000,00 contra os 4.050,00 da nota.
func TestAliquotaDaNotaManualViraFracao(t *testing.T) {
	casos := []struct {
		nome, pct, base, valor, esperado string
	}{
		{"percentual digitado", "18", "22500", "4050", "0.18"},
		{"fração da tela do desktop", "0.18", "22500", "4050", "0.18"},
		{"fração de 1% da tela", "0.01", "1000", "10", "0.01"},
		{"percentual de 1% pela API", "1", "1000", "10", "0.01"},
		{"IPI acima de 100% em percentual", "300", "100", "300", "3"},
		{"percentual sem base", "12", "0", "0", "0.12"},
		{"fração sem base", "0.12", "0", "0", "0.12"},
		{"percentual com casas", "7.5", "1000", "75", "0.075"},
		{"sem alíquota, sai da nota", "0", "1000", "120", "0.12"},
		{"sem imposto", "0", "0", "0", "0"},
		{"base sem valor", "0", "1000", "0", "0"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := aliquotaDaNotaManual(d(c.pct), d(c.base), d(c.valor)); !got.Equal(d(c.esperado)) {
				t.Fatalf("alíquota = %s, esperado %s", got, c.esperado)
			}
		})
	}
}

func TestNotaManualNaoGeraDivergenciaFalsaDeICMS(t *testing.T) {
	for _, aliq := range []string{"18", "0.18", "0"} { // API (percentual), tela (fração) e só o valor
		item := entryItemFromDTO(request.CreateFiscalEntryItemDTO{
			Sequence: 1, Cfop: "1101", Quantity: d("1000"), UnitPrice: d("22.5"), TotalPrice: d("22500"),
			BaseICMS: d("22500"), AliqICMS: d(aliq), ValorICMS: d("4050"),
			BaseIPI: d("22500"), AliqIPI: d("0.05"), ValorIPI: d("1125"),
		})
		divs := entrada.Divergencias(&entity.FiscalEntry{Itens: []*entity.FiscalEntryItem{item}}, entrada.ParametrosConferencia{})
		for _, dv := range divs {
			if dv.Tipo == "CALCULO_ICMS" || dv.Tipo == "CALCULO_IPI" {
				t.Fatalf("aliq_icms=%s: divergência falsa %+v", aliq, dv)
			}
		}
	}
}
