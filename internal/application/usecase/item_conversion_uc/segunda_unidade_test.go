package item_conversion_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/repository"
)

type repoSincroniza struct {
	repository.ManagedItemConversionRepository
	chamadas []string
	antes    *decimal.Decimal
}

func (r *repoSincroniza) SincronizarSegundaUnidade(_ context.Context, _ int64, de, para string, f decimal.Decimal) (*decimal.Decimal, bool, error) {
	r.chamadas = append(r.chamadas, de+">"+para+"="+f.String())
	return r.antes, r.antes != nil, nil
}

func conv(t *testing.T, mask, qPara string) *entity.ItemUnitConversion {
	t.Helper()
	c, err := entity.NewItemUnitConversionFraction(9, mask, "UN", "KG", decimal.NewFromInt(1), decimal.RequireFromString(qPara), decimal.Zero, decimal.Zero, "VALUE", [16]byte{1})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestSegundaUnidadeSegueAConversaoDeHoje(t *testing.T) {
	ctx := context.Background()
	hoje := time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)
	vinteCinco := decimal.NewFromInt(25)
	repo := &repoSincroniza{antes: &vinteCinco}
	uc := &ItemConversionUseCase{repo: repo, now: func() time.Time { return hoje }}

	avisos := uc.segundaUnidade(ctx, conv(t, "", "24"))
	if len(repo.chamadas) != 1 || repo.chamadas[0] != "UN>KG=24" {
		t.Fatalf("deveria sincronizar UN→KG = 24, chamou %v", repo.chamadas)
	}
	if len(avisos) != 1 || !strings.Contains(avisos[0], "25") {
		t.Fatalf("o aviso conta o fator anterior (25): %v", avisos)
	}

	// Máscara, vigência futura e versão já encerrada não mexem no cadastro do item.
	futura := conv(t, "", "26")
	futura.ValidFrom = hoje.AddDate(0, 0, 3)
	encerrada := conv(t, "", "23")
	fim := hoje.AddDate(0, 0, -1)
	encerrada.ValidTo = &fim
	for _, c := range []*entity.ItemUnitConversion{conv(t, "ESP5", "141.3"), futura, encerrada} {
		if a := uc.segundaUnidade(ctx, c); a != nil {
			t.Fatalf("não deveria avisar: %v", a)
		}
	}
	if len(repo.chamadas) != 1 {
		t.Fatalf("só a conversão sem máscara vigente hoje sincroniza; chamadas: %v", repo.chamadas)
	}

	// Mesmo fator: nada mudou, nada a avisar.
	repo.antes = nil
	if a := uc.segundaUnidade(ctx, conv(t, "", "24")); a != nil {
		t.Fatalf("sem mudança não há aviso: %v", a)
	}
}
