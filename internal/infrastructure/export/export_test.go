package export

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
	"time"
)

func sampleTable() *Table {
	t := &Table{
		Title:       "Clientes",
		Subtitle:    "Tecnofer • somente ativos",
		Columns:     []string{"Código", "Razão Social", "CNPJ", "Ativo"},
		GeneratedAt: time.Date(2026, 6, 21, 10, 30, 0, 0, time.UTC),
	}
	for i := 0; i < 120; i++ { // force PDF pagination
		t.Rows = append(t.Rows, []string{
			"100" + string(rune('0'+i%10)),
			"Indústria Metalúrgica São José Ltda — Filial",
			"12.345.678/0001-90",
			"Sim",
		})
	}
	return t
}

func TestEncodeCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := EncodeCSV(&buf, sampleTable()); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "\xEF\xBB\xBF") {
		t.Error("missing UTF-8 BOM")
	}
	if !strings.Contains(out, "Razão Social") {
		t.Error("header missing")
	}
	if !strings.Contains(out, ";") {
		t.Error("expected semicolon delimiter")
	}
}

func TestEncodeXLSXIsValidZip(t *testing.T) {
	var buf bytes.Buffer
	if err := EncodeXLSX(&buf, sampleTable()); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}
	want := map[string]bool{
		"[Content_Types].xml":        false,
		"xl/workbook.xml":            false,
		"xl/worksheets/sheet1.xml":   false,
		"xl/_rels/workbook.xml.rels": false,
	}
	for _, f := range zr.File {
		if _, ok := want[f.Name]; ok {
			want[f.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing part %s", name)
		}
	}
}

func TestEncodePDFStructure(t *testing.T) {
	var buf bytes.Buffer
	if err := EncodePDF(&buf, sampleTable()); err != nil {
		t.Fatal(err)
	}
	out := buf.Bytes()
	if !bytes.HasPrefix(out, []byte("%PDF-1.4")) {
		t.Error("missing PDF header")
	}
	if !bytes.Contains(out, []byte("%%EOF")) {
		t.Error("missing EOF marker")
	}
	if !bytes.Contains(out, []byte("/Type /Catalog")) {
		t.Error("missing catalog")
	}
	if !bytes.Contains(out, []byte("startxref")) {
		t.Error("missing xref")
	}
}

func TestTableFromSlice(t *testing.T) {
	type row struct {
		Code int     `json:"codigo"`
		Name string  `json:"nome"`
		Rate float64 `json:"-"`
	}
	tbl, err := TableFromSlice("Itens", []row{{1, "Parafuso", 9.9}, {2, "Porca", 1.1}})
	if err != nil {
		t.Fatal(err)
	}
	if len(tbl.Columns) != 2 || tbl.Columns[0] != "codigo" || tbl.Columns[1] != "nome" {
		t.Fatalf("unexpected columns: %v", tbl.Columns)
	}
	if len(tbl.Rows) != 2 || tbl.Rows[0][1] != "Parafuso" {
		t.Fatalf("unexpected rows: %v", tbl.Rows)
	}
}
