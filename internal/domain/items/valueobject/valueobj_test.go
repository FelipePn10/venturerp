package valueobject

import (
	"encoding/json"
	"github.com/shopspring/decimal"
	"math"
	"testing"
)

func TestNewItemCode(t *testing.T) {
	if _, err := NewItemCode(0); err == nil {
		t.Error("expected error for code 0")
	}
	if _, err := NewItemCode(-5); err == nil {
		t.Error("expected error for negative code")
	}
	c, err := NewItemCode(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c != 42 || !c.IsValid() {
		t.Errorf("unexpected item code: %v", c)
	}
}

func TestDimensions(t *testing.T) {
	if _, err := NewDimensions(0, 1, 1); err == nil {
		t.Error("expected error for zero length")
	}
	d, err := NewDimensions(2, 3, 6.35)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := d.Volume(); math.Abs(got-38.1) > 1e-9 {
		t.Errorf("Volume() = %v, want 38.1", got)
	}
}

func TestWeight(t *testing.T) {
	cases := []struct {
		name       string
		gross, net float64
		unit       string
		wantValid  bool
	}{
		{"valid", 10, 8, "KG", true},
		{"empty unit", 10, 8, "", false},
		{"negative net", 10, -1, "KG", false},
		{"gross < net", 5, 8, "KG", false},
		{"gross == net", 8, 8, "KG", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewWeight(tc.gross, tc.net, tc.unit)
			if (err == nil) != tc.wantValid {
				t.Errorf("NewWeight(%v,%v,%q) valid=%v, want %v", tc.gross, tc.net, tc.unit, err == nil, tc.wantValid)
			}
		})
	}
}

func TestAttribute(t *testing.T) {
	if _, err := NewAttribute("", "v"); err == nil {
		t.Error("expected error for empty name")
	}
	if _, err := NewAttribute("n", ""); err == nil {
		t.Error("expected error for empty value")
	}
	if _, err := NewAttribute("cor", "azul"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCyclicalCountConfig(t *testing.T) {
	if _, err := NewCyclicalCountConfig(0); err == nil {
		t.Error("expected error for 0 days")
	}
	if _, err := NewCyclicalCountConfig(30); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCyclicalCountConfigJSONContractAndLegacyRead(t *testing.T) {
	for _, tc := range []struct {
		body string
		want int
	}{{`{"days_interval":15}`, 15}, {`{"DaysInterval":7}`, 7}} {
		var got CyclicalCountConfig
		if err := json.Unmarshal([]byte(tc.body), &got); err != nil {
			t.Fatal(err)
		}
		if got.DaysInterval != tc.want || !got.IsValid() {
			t.Fatalf("json=%s config=%+v", tc.body, got)
		}
	}
	for _, invalid := range []string{`{"days":1}`, `{"days_interval":0}`, `{"days_interval":-1}`} {
		var got CyclicalCountConfig
		if err := json.Unmarshal([]byte(invalid), &got); err != nil {
			t.Fatal(err)
		}
		if got.IsValid() {
			t.Fatalf("contrato inválido aceito: %s", invalid)
		}
	}
	encoded, err := json.Marshal(CyclicalCountConfig{DaysInterval: 30})
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"days_interval":30}` {
		t.Fatalf("resposta fora do contrato: %s", encoded)
	}
}

func TestReorderPoint_Calculate(t *testing.T) {
	d := decimal.RequireFromString
	// ROP = (TR * CM / CR) + ES = (10*5/2)+3 = 28
	r, err := NewReorderPoint(10, d("5"), 2, d("3"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, _ := r.Calculate(); !got.Equal(d("28")) {
		t.Errorf("Calculate() = %s, want 28", got)
	}
	// Item em KG: 7 dias × 12,5 kg/mês ÷ 30 + 2,5 = 5,4166… → 6 (sobe; antes o
	// MRP truncava a divisão inteira).
	r, _ = NewReorderPoint(7, d("12.5"), 30, d("2.5"))
	if got, _ := r.Calculate(); !got.Equal(d("6")) {
		t.Errorf("fracionado = %s, want 6", got)
	}
	// CR == 0 is invalid at construction and Calculate guards against it.
	if _, err := NewReorderPoint(10, d("5"), 0, d("3")); err == nil {
		t.Error("expected error for CR=0")
	}
	if _, err := NewReorderPoint(10, d("-1"), 2, d("3")); err == nil {
		t.Error("consumo negativo")
	}
}

func TestReorderPointJSONNumericoECompativel(t *testing.T) {
	r, _ := NewReorderPoint(7, decimal.RequireFromString("12.5"), 30, decimal.RequireFromString("2.5"))
	b, err := json.Marshal(r)
	if err != nil || string(b) != `{"TR":7,"CM":12.5,"CR":30,"ES":2.5}` {
		t.Fatalf("JSONB deve guardar números: %s %v", b, err)
	}
	// Formato gravado antes (inteiros) e o que o desktop envia continuam lidos.
	for _, raw := range []string{`{"TR":7,"CM":200,"CR":2,"ES":50}`, `{"TR":7,"CM":"200","CR":2,"ES":"50"}`} {
		var v ReorderPoint
		if err := json.Unmarshal([]byte(raw), &v); err != nil || v.TR != 7 || !v.CM.Equal(decimal.NewFromInt(200)) || !v.ES.Equal(decimal.NewFromInt(50)) {
			t.Fatalf("%s: %+v %v", raw, v, err)
		}
	}
}
