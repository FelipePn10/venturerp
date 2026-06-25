package romaneio

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
	"time"
)

func sampleRomaneio() *RomaneioData {
	now := time.Date(2026, 6, 23, 14, 30, 0, 0, time.UTC)
	return &RomaneioData{
		Title:         "ROMANEIO DE EXPEDICAO",
		Code:          1001,
		Date:          now,
		Status:        "OPEN",
		ReferenceType: "SALES_ORDER",
		ReferenceCode: 500,
		Enterprise: CompanyInfo{
			Name:     "Tecnofer Industria Metalurgica Ltda",
			CNPJCPF:  "12.345.678/0001-90",
			IE:       "123.456.789",
			Street:   "Rua dos Metais",
			Number:   "1000",
			District: "Distrito Industrial",
			City:     "São Paulo",
			UF:       "SP",
			CEP:      "01234-567",
			Phone:    "(11) 3333-4444",
			Email:    "vendas@tecnofer.com.br",
		},
		Destinatario: CompanyInfo{
			Name:     "Cliente Bom S.A.",
			CNPJCPF:  "98.765.432/0001-10",
			Street:   "Av. Paulista",
			Number:   "1500",
			District: "Bela Vista",
			City:     "São Paulo",
			UF:       "SP",
		},
		Carrier: CarrierInfo{
			Name:        "Transportadora Rápida Ltda",
			CNPJCPF:     "11.222.333/0001-44",
			Plate:       "ABC-1234",
			Driver:      "José da Silva",
			ANTT:        "123456789",
			FreightType: "FOB",
		},
		Items: []RomaneioItem{
			{
				Sequence:    1,
				ItemCode:    100,
				Description: "Chapa de Aço SAE 1020 3mm",
				NCM:         "7208.51.00",
				CFOP:        "5102",
				Quantity:    100,
				Unit:        "KG",
				UnitPrice:   12.50,
				TotalPrice:  1250.00,
				ICMSPct:     18,
				IPIPct:      5,
				PISPct:      1.65,
				COFINSPct:   7.6,
				WeightNet:   100,
			},
			{
				Sequence:    2,
				ItemCode:    200,
				Description: "Parafuso Sextavado M8x30",
				NCM:         "7318.15.00",
				CFOP:        "5102",
				Quantity:    500,
				Unit:        "UN",
				UnitPrice:   0.35,
				TotalPrice:  175.00,
				ICMSPct:     18,
				IPIPct:      5,
				WeightNet:   5,
			},
		},
		TotalVolumes: 3,
		TotalWeight:  105,
		TotalGross:   1425.00,
		TotalNet:     1425.00,
		TransportInfo: TransportInfo{
			FreightType:       "FOB",
			FreightValue:      150.00,
			InsuranceValue:    50.00,
			VolumeQuantity:    3,
			VolumeType:        "CX",
			NetWeight:         105,
			GrossWeight:       115,
			EstimatedDelivery: "28/06/2026",
		},
		GeneratedAt: now,
	}
}

func TestGenerateRomaneioPDF(t *testing.T) {
	data := sampleRomaneio()
	pdf, err := GenerateRomaneioPDF(data)
	if err != nil {
		t.Fatalf("GenerateRomaneioPDF error: %v", err)
	}

	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) {
		t.Error("missing PDF header")
	}
	if !bytes.Contains(pdf, []byte("%%EOF")) {
		t.Error("missing EOF marker")
	}
	if !bytes.Contains(pdf, []byte("/Type /Catalog")) {
		t.Error("missing catalog")
	}
	if !bytes.Contains(pdf, []byte("startxref")) {
		t.Error("missing xref")
	}
	if !bytes.Contains(pdf, []byte("/Helvetica-Bold")) {
		t.Error("missing Helvetica-Bold font")
	}

	content := string(pdf)
	if !strings.Contains(content, "ROMANEIO DE EXPEDICAO") {
		t.Error("missing title in PDF content")
	}
	if !strings.Contains(content, "Tecnofer") {
		t.Error("missing enterprise name")
	}
}

func TestGenerateRomaneioPDF_Minimal(t *testing.T) {
	d := &RomaneioData{
		Title:       "ROMANEIO",
		Code:        1,
		Date:        time.Now(),
		Status:      "OPEN",
		Enterprise:  CompanyInfo{Name: "Teste"},
		GeneratedAt: time.Now(),
	}
	pdf, err := GenerateRomaneioPDF(d)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !bytes.Contains(pdf, []byte("%%EOF")) {
		t.Error("minimal PDF should still be valid")
	}
}

func TestGenerateRomaneioXLSX(t *testing.T) {
	data := sampleRomaneio()
	xlsx, err := GenerateRomaneioXLSX(data)
	if err != nil {
		t.Fatalf("GenerateRomaneioXLSX error: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(xlsx), int64(len(xlsx)))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}

	want := map[string]bool{
		"[Content_Types].xml":        false,
		"xl/workbook.xml":            false,
		"xl/worksheets/sheet1.xml":   false,
		"xl/_rels/workbook.xml.rels": false,
		"xl/styles.xml":              false,
	}
	for _, f := range zr.File {
		if _, ok := want[f.Name]; ok {
			want[f.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing OOXML part: %s", name)
		}
	}
}

func TestGenerateRomaneioXLSX_Content(t *testing.T) {
	data := sampleRomaneio()
	xlsx, err := GenerateRomaneioXLSX(data)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(xlsx), int64(len(xlsx)))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}

	var sheetContent string
	for _, f := range zr.File {
		if f.Name == "xl/worksheets/sheet1.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("cannot open sheet: %v", err)
			}
			var buf bytes.Buffer
			buf.ReadFrom(rc)
			rc.Close()
			sheetContent = buf.String()
		}
	}

	if sheetContent == "" {
		t.Fatal("sheet1.xml is empty")
	}
	if !strings.Contains(sheetContent, "ROMANEIO DE EXPEDICAO") {
		t.Error("missing title in sheet XML")
	}
	if !strings.Contains(sheetContent, "Chapa de A") {
		t.Error("missing item description")
	}
	if !strings.Contains(sheetContent, "7208.51.00") {
		t.Error("missing NCM")
	}
}

func TestMoneyFormat(t *testing.T) {
	tests := []struct{ in float64; want string }{
		{0, "0,00"},
		{12.5, "12,50"},
		{1250, "1.250,00"},
		{1234567.89, "1.234.567,89"},
		{-99.9, "-99,90"},
	}
	for _, tt := range tests {
		if got := money(tt.in); got != tt.want {
			t.Errorf("money(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestXmlEscape(t *testing.T) {
	tests := []struct {
		in  string
		exp string
	}{
		{"normal", "normal"},
		{"a & b", "a &amp; b"},
		{"<tag>", "&lt;tag&gt;"},
		{`"quoted"`, "&quot;quoted&quot;"},
	}
	for _, tt := range tests {
		out := xmlEscape(tt.in)
		if out != tt.exp {
			t.Errorf("xmlEscape(%q) = %q, want %q", tt.in, out, tt.exp)
		}
	}
}

func TestCompanyInfo_Defaults(t *testing.T) {
	c := CompanyInfo{}
	if c.Name != "" {
		t.Error("name should default to empty")
	}
}

func TestCarrierInfo_Defaults(t *testing.T) {
	c := CarrierInfo{}
	if c.Name != "" {
		t.Error("name should default to empty")
	}
}
