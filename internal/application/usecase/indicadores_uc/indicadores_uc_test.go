package indicadores_uc

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/acesso"
)

type repoFixo struct{}

func (repoFixo) Vendas(context.Context, time.Time, time.Time) (*Margem, error) {
	return &Margem{Receita: decimal.RequireFromString("360.50"), CMV: decimal.RequireFromString("230.22")}, nil
}
func (repoFixo) OEE(context.Context, time.Time, time.Time) ([]OEE, error) {
	return []OEE{{HorasApontadas: decimal.NewFromInt(30), HorasDisponiveis: decimal.NewFromInt(40), Boas: decimal.NewFromInt(95), Refugo: decimal.NewFromInt(5)}}, nil
}
func (repoFixo) Giro(context.Context, time.Time, time.Time) (*Giro, error) { return &Giro{}, nil }
func (repoFixo) PontualidadeFornecedores(context.Context, time.Time, time.Time) ([]Pontualidade, error) {
	return nil, nil
}
func (repoFixo) AcertoDaPromessa(context.Context, time.Time, time.Time) (*Pontualidade, error) {
	return &Pontualidade{}, nil
}

type soProducao struct{}

func (soProducao) PodeNoModulo(_ context.Context, m acesso.Modulo, _ acesso.Acao) (bool, error) {
	return m == acesso.Producao, nil
}

func TestIndicadores(t *testing.T) {
	uc := &UseCase{Repo: repoFixo{}}
	ind, err := uc.Calcular(context.Background(), time.Now().AddDate(0, -1, 0), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !ind.Vendas.Percentual.Equal(decimal.RequireFromString("36.14")) {
		t.Fatalf("margem: %+v", ind.Vendas)
	}
	// 75% disponível × 95% qualidade = 71,25%.
	if o := ind.OEE[0]; !o.Disponibilidade.Equal(decimal.NewFromInt(75)) || !o.Qualidade.Equal(decimal.NewFromInt(95)) || !o.OEE.Equal(decimal.RequireFromString("71.25")) {
		t.Fatalf("OEE: %+v", o)
	}
	uc.Acesso = soProducao{}
	ind, _ = uc.Calcular(context.Background(), time.Now().AddDate(0, -1, 0), time.Now())
	if ind.Vendas != nil || ind.Giro != nil || len(ind.OEE) != 1 {
		t.Fatalf("perfil da produção só vê o OEE: %+v", ind)
	}
}
