package pdfkit

import "testing"

func TestCode128BKnownChecksumAndValidation(t *testing.T) {
	symbols, err := Code128B("OF1_TEST")
	if err != nil {
		t.Fatal(err)
	}
	if symbols[0] != 104 || symbols[len(symbols)-1] != 106 {
		t.Fatalf("unexpected boundaries: %v", symbols)
	}
	checksum := 104
	for i, r := range "OF1_TEST" {
		checksum += (int(r) - 32) * (i + 1)
	}
	if got := symbols[len(symbols)-2]; got != checksum%103 {
		t.Fatalf("checksum=%d, want %d", got, checksum%103)
	}
	if _, err = Code128B("inválido"); err == nil {
		t.Fatal("expected non-ASCII validation error")
	}
}

func TestDrawCode128BRejectsUnreadableWidth(t *testing.T) {
	doc := NewLandscape()
	page := doc.AddPage()
	if err := page.DrawCode128B("OF1_abcdefghijklmnopqrstuvwxyz0123456789", 10, 10, 40, 30, Black); err == nil {
		t.Fatal("expected insufficient width error")
	}
	if err := page.DrawCode128B("OF1_abcdefghijklmnopqrstuvwxyz0123456789", 10, 10, 700, 30, Black); err != nil {
		t.Fatal(err)
	}
}
