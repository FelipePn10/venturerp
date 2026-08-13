package manufacturing

import (
	"bytes"
	"strings"
	"testing"
)

func TestSpecificDocumentsUseSharedPDFEngineAndCleanHierarchy(t *testing.T) {
	b := Branding{BrandColorHex: "#1B5E36"}
	token := "OF1_abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
	order := RenderManufacturingOrder(ManufacturingOrder{Branding: b, Title: "ORDEM DE FABRICAÇÃO", Number: "OF-1", Status: "LIBERADA", BarcodeValue: token, Summary: []Field{{"Item", "TEA452-0"}}, Sections: []OrderSection{{Title: "Operações", Columns: []Column{{Title: "Código", Weight: 1}}, Rows: [][]string{{"010"}}}}})
	structure := RenderItemStructure(ItemStructure{Branding: b, Title: "ESTRUTURA DO ITEM", RootCode: "TEA452-0", Revision: "C", BaseQty: "1 UN", Nodes: []StructureNode{{Level: 0, Code: "TEA452-0", Description: "Conjunto"}, {Level: 1, Code: "MP-01", Description: "Componente"}}})
	for name, pdf := range map[string][]byte{"order": order, "structure": structure} {
		if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) || !bytes.HasSuffix(pdf, []byte("%%EOF\n")) {
			t.Fatalf("%s is not a valid PDF envelope", name)
		}
		for _, forbidden := range []string{"|  |", "+--", "├", "└"} {
			if strings.Contains(string(pdf), forbidden) {
				t.Errorf("%s contains forbidden marker %q", name, forbidden)
			}
		}
	}
	if bytes.Contains(order, []byte(token)) {
		t.Fatal("PDF must not expose the complete opaque token as human-readable text")
	}
}
