package financial_uc

import (
	"context"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
)

// LeitorDoPlano lê o plano com o movimento do período (repositório PG).
type LeitorDoPlano interface {
	ContasComMovimento(ctx context.Context, plano *int64, de, ate time.Time) ([]entity.ContaDoResultado, error)
}

// LinhaDaDRE: valor = créditos − débitos (receita positiva, custo e despesa negativos).
type LinhaDaDRE struct {
	Numero     string          `json:"account_number"`
	Nome       string          `json:"description"`
	Nivel      int             `json:"level"`
	Analitica  bool            `json:"analytic"`
	Valor      decimal.Decimal `json:"value"`
	Percentual decimal.Decimal `json:"pct_of_revenue"`
}

// DREGerencial é o resultado do período pelo plano de contas.
type DREGerencial struct {
	Linhas    []LinhaDaDRE    `json:"lines"`
	Receita   decimal.Decimal `json:"revenue"`
	Resultado decimal.Decimal `json:"result"`
}

// DREGerencialUseCase monta a DRE pelo plano de contas (melhoria 10), dos
// lançamentos contábeis — a DRE do financeiro usa só notas e títulos.
type DREGerencialUseCase struct{ Repo LeitorDoPlano }

// Gerar: grupos de resultado pelos prefixos do número da conta (padrão "3" e
// "4"); prefixoReceita marca a receita (padrão o primeiro grupo) para o %.
func (uc *DREGerencialUseCase) Gerar(ctx context.Context, plano *int64, de, ate time.Time, prefixos []string) (*DREGerencial, error) {
	if ate.Before(de) {
		return nil, errorsuc.NewValidationError("o fim do período é anterior ao início")
	}
	if len(prefixos) == 0 {
		prefixos = []string{"3", "4"}
	}
	contas, err := uc.Repo.ContasComMovimento(ctx, plano, de, ate)
	if err != nil {
		return nil, err
	}
	return MontarDRE(contas, prefixos), nil
}

// MontarDRE consolida cada conta nas sintéticas acima dela.
func MontarDRE(contas []entity.ContaDoResultado, prefixos []string) *DREGerencial {
	porID := map[int64]*entity.ContaDoResultado{}
	for i := range contas {
		porID[contas[i].ID] = &contas[i]
	}
	valor := map[int64]decimal.Decimal{}
	nivel := func(c *entity.ContaDoResultado) int {
		n, p := 0, c.ParentID
		for p != 0 && n < 20 {
			n++
			if x := porID[p]; x != nil {
				p = x.ParentID
			} else {
				break
			}
		}
		return n
	}
	for i := range contas {
		c := &contas[i]
		v := c.Creditos.Sub(c.Debitos)
		if v.IsZero() {
			continue
		}
		for id, passos := c.ID, 0; id != 0 && passos < 20; passos++ {
			valor[id] = valor[id].Add(v)
			p := porID[id]
			if p == nil {
				break
			}
			id = p.ParentID
		}
	}
	dentro := func(numero string) bool {
		for _, p := range prefixos {
			if strings.HasPrefix(numero, p) {
				return true
			}
		}
		return false
	}
	out := &DREGerencial{}
	for i := range contas {
		c := &contas[i]
		v, ok := valor[c.ID]
		if !ok || !dentro(c.Numero) {
			continue
		}
		out.Linhas = append(out.Linhas, LinhaDaDRE{Numero: c.Numero, Nome: c.Nome, Nivel: nivel(c), Analitica: c.Analitica, Valor: v.Round(2)})
		if c.ParentID == 0 || porID[c.ParentID] == nil {
			out.Resultado = out.Resultado.Add(v)
			if strings.HasPrefix(c.Numero, prefixos[0]) {
				out.Receita = out.Receita.Add(v)
			}
		}
	}
	out.Resultado = out.Resultado.Round(2)
	out.Receita = out.Receita.Round(2)
	for i := range out.Linhas {
		if out.Receita.IsPositive() {
			out.Linhas[i].Percentual = out.Linhas[i].Valor.Div(out.Receita).Mul(decimal.NewFromInt(100)).Round(2)
		}
	}
	return out
}
