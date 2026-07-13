package third_party_service_uc

import (
	"context"
	"testing"
	"time"

	domain "github.com/FelipePn10/panossoerp/internal/domain/third_party_service"
	"github.com/shopspring/decimal"
)

type priceResolver struct {
	domain.Repository
	price *domain.Price
}

func (r *priceResolver) ResolvePrice(context.Context, int64, string, int64, int64, time.Time, map[string]string) (*domain.Price, error) {
	return r.price, nil
}
func TestStandardCostIncludesFreightAndConversion(t *testing.T) {
	factor := decimal.NewFromInt(2)
	uc := New(&priceResolver{price: &domain.Price{UnitPrice: decimal.NewFromInt(100), FreightType: "PERCENT", FreightValue: decimal.NewFromInt(10), ConversionFactor: &factor}})
	got, e := uc.StandardCostPerUnit(context.Background(), 1, "", 2, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	if !got.Equal(decimal.NewFromInt(55)) {
		t.Fatalf("cost=%s want=55", got)
	}
}

func TestRealCostSubtractsRecoverableTaxAndIgnoresStandardFreight(t *testing.T) {
	factor := decimal.NewFromInt(1)
	uc := New(&priceResolver{price: &domain.Price{UnitPrice: decimal.NewFromInt(100), FreightType: "FIXED", FreightValue: decimal.NewFromInt(30), TaxPercent: decimal.NewFromInt(15), ConversionFactor: &factor}})
	got, err := uc.CostPerUnit(context.Background(), 1, "", 2, time.Now(), "REAL")
	if err != nil {
		t.Fatal(err)
	}
	if !got.RecoverableTaxes.Equal(decimal.NewFromInt(15)) || !got.Freight.IsZero() || !got.EffectiveUnitCost.Equal(decimal.NewFromInt(85)) {
		t.Fatalf("real cost=%+v", got)
	}
}
