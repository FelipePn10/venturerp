package nesting

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/service"
)

func TestHTTPProvider_Type(t *testing.T) {
	if NewHTTPProvider("http://x").Type() != entity.CutTypeTrueShape2D {
		t.Fatal("provider must declare TRUE_SHAPE_2D")
	}
}

func TestHTTPProvider_RoundTrip(t *testing.T) {
	var got wireRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		// Canned engine result: one sheet with the part placed rotated 37°.
		_, _ = w.Write([]byte(`{
			"sheets": [{
				"stock_id": 10, "width": 1000, "height": 1000, "repeat": 1, "used_area": 250000,
				"placements": [{"part_id": 1, "label": "L", "x": 5, "y": 6, "width": 500, "height": 500, "rotation_deg": 37}]
			}],
			"unplaced": [{"label": "X", "width": 999, "height": 999, "qty": 2}]
		}`))
	}))
	defer srv.Close()

	p := NewHTTPProvider(srv.URL)
	poly := []service.Point{{X: 0, Y: 0}, {X: 500, Y: 0}, {X: 500, Y: 500}, {X: 0, Y: 500}}
	sol, err := p.Optimize(
		[]service.DemandPiece{{PartID: 1, Label: "L", Polygon: poly, Width: 500, Height: 500, Qty: 1, AllowRotation: true}},
		[]service.StockPiece{{StockID: 10, Width: 1000, Height: 1000, Qty: 1}},
		service.CutParams{Kerf: 4},
	)
	if err != nil {
		t.Fatal(err)
	}

	// Request was serialised correctly.
	if got.Params.Kerf != 4 || len(got.Parts) != 1 || len(got.Parts[0].Polygon) != 4 {
		t.Fatalf("request not serialised as expected: %+v", got)
	}

	// Response was mapped into a Solution.
	if sol.StockUsed != 1 || len(sol.Patterns) != 1 {
		t.Fatalf("expected 1 sheet/pattern, got used=%d patterns=%d", sol.StockUsed, len(sol.Patterns))
	}
	pl := sol.Patterns[0].Placements[0]
	if pl.RotationDeg != 37 || !pl.Rotated || pl.X != 5 || pl.Y != 6 {
		t.Fatalf("placement mapped wrong: %+v", pl)
	}
	if len(sol.Unplaced) != 1 || sol.Unplaced[0].Qty != 2 {
		t.Fatalf("unplaced mapped wrong: %+v", sol.Unplaced)
	}
	if sol.Utilization < 0.24 || sol.Utilization > 0.26 { // 250000 / 1_000_000
		t.Fatalf("utilization = %v, want ~0.25", sol.Utilization)
	}
}
