package entity

import "testing"

// Bug AC: o rascunho não tem número — a mensagem dizia "NF-e 0".
func TestNotaDoRomaneio(t *testing.T) {
	id, zero, numero := int64(77), int64(0), int64(1234)
	for _, c := range []struct {
		s    Shipment
		quer string
	}{
		{Shipment{FiscalExitID: &id, NFeNumber: &numero}, "a NF-e 1234"},
		{Shipment{FiscalExitID: &id, NFeNumber: &zero}, "uma NF-e em rascunho (id 77)"},
		{Shipment{FiscalExitID: &id}, "uma NF-e em rascunho (id 77)"},
		{Shipment{}, "uma NF-e"},
	} {
		if got := c.s.NotaDoRomaneio(); got != c.quer {
			t.Errorf("%+v: %q, quer %q", c.s, got, c.quer)
		}
	}
}
