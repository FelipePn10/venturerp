//go:build integration

package item_conversion_uc_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_conversion_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/service"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	itemrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	convrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_conversion"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Melhoria 3: a barra de 6 m comprada em kg ganha 1 UN = X KG, 1 UN = 6000 MM e
// lote múltiplo 1 de uma vez. A empresa B não alcança o item da A.
func TestAssistenteDeConversaoAplica(t *testing.T) {
	q, pool := testutil.Queries(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, nil)
	item := testutil.UniqueCode()
	testutil.NovosItens(t, pool, a.ID, ator, item)
	t.Cleanup(func() {
		pool.Exec(base, `DELETE FROM item_unit_conversion_events WHERE item_code=$1`, item)
		pool.Exec(base, `DELETE FROM item_unit_conversions WHERE item_code=$1`, item)
	})
	uc := item_conversion_uc.NewItemConversionUseCase(convrepo.New(q, pool), &auth.AuthService{}, itemrepo.NewRepositoryItemSQLC(q).ComPool(pool))
	dto := item_conversion_uc.AssistenteConversaoDTO{ItemCode: request.TextCode(fmt.Sprint(item)), AplicarLoteMultiplo: true,
		MedidasDaPeca: service.MedidasDaPeca{Formato: "BARRA_REDONDA", Material: "ACO", ComprimentoMM: decimal.NewFromInt(6000), DiametroMM: decimal.NewFromInt(25)}}
	if _, err := uc.AplicarAssistente(b.Ctx, dto); err == nil {
		t.Fatal("a empresa B aplicou o assistente no item da A")
	}
	res, err := uc.AplicarAssistente(a.Ctx, dto)
	if err != nil {
		t.Fatal(err)
	}
	if !res.LoteAplicado || len(res.Sugestoes) != 2 || !res.Sugestoes[0].Aplicada || !res.Sugestoes[1].Aplicada {
		t.Fatalf("resultado: %+v", res)
	}
	var kg, mm, lote decimal.Decimal
	if err := pool.QueryRow(base, `SELECT
		(SELECT factor FROM item_unit_conversions WHERE item_code=$1 AND from_uom='UN' AND to_uom='KG' AND is_active),
		(SELECT factor FROM item_unit_conversions WHERE item_code=$1 AND from_uom='UN' AND to_uom='MM' AND is_active),
		(SELECT planning_multiple_lot FROM items WHERE code=$1)`, item).Scan(&kg, &mm, &lote); err != nil {
		t.Fatal(err)
	}
	if !kg.Equal(decimal.RequireFromString("23.120158")) || !mm.Equal(decimal.NewFromInt(6000)) || !lote.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("gravado: 1 UN = %s KG, %s MM, lote %s", kg, mm, lote)
	}
	// Rodar de novo não duplica: as duas já estão certas.
	if res, err = uc.AplicarAssistente(a.Ctx, dto); err != nil || res.Sugestoes[0].Atual == nil || res.Sugestoes[0].Aviso != "" {
		t.Fatalf("segunda vez: %+v err=%v", res, err)
	}
}
