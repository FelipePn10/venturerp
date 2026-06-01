package valueobject

import "testing"

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
	d, err := NewDimensions(2, 3, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := d.Volume(); got != 24 {
		t.Errorf("Volume() = %d, want 24", got)
	}
}

func TestWeight(t *testing.T) {
	cases := []struct {
		name              string
		gross, net        float64
		unit              string
		wantValid         bool
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

func TestReorderPoint_Calculate(t *testing.T) {
	// ROP = (TR * CM / CR) + ES = (10*5/2)+3 = 28
	r, err := NewReorderPoint(10, 5, 2, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := r.Calculate()
	if err != nil {
		t.Fatalf("Calculate error: %v", err)
	}
	if got != 28 {
		t.Errorf("Calculate() = %d, want 28", got)
	}

	// CR == 0 is invalid at construction and Calculate guards against it.
	if _, err := NewReorderPoint(10, 5, 0, 3); err == nil {
		t.Error("expected error for CR=0")
	}
}
