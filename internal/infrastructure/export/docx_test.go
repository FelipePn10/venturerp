package export

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func brandedSampleTable() *Table {
	t := sampleTable()
	t.Branding = &Branding{
		CompanyName: "Tecnofer Indústria Metalúrgica Ltda",
		CNPJ:        "52.454.668/0001-02",
		IE:          "9103144679",
		Address:     "Rua das Indústrias, 1000 - Centro - Sertanópolis/PR  CEP 86975-000",
		Phone:       "(43) 3232-0000",
		Email:       "contato@tecnofer.com.br",
	}
	return t
}

func TestParseFormatDOCX(t *testing.T) {
	for _, in := range []string{"docx", "DOCX", "word", "doc", " Word "} {
		f, ok := ParseFormat(in)
		if !ok || f != FormatDOCX {
			t.Errorf("ParseFormat(%q) = %v, %v; want docx, true", in, f, ok)
		}
	}
	if ct := FormatDOCX.ContentType(); !strings.Contains(ct, "wordprocessingml") {
		t.Errorf("unexpected content type: %s", ct)
	}
	if FormatDOCX.Extension() != "docx" {
		t.Errorf("unexpected extension: %s", FormatDOCX.Extension())
	}
}

func TestEncodeDOCXIsValidZip(t *testing.T) {
	var buf bytes.Buffer
	if err := EncodeDOCX(&buf, brandedSampleTable()); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}

	parts := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		b, _ := io.ReadAll(rc)
		rc.Close()
		parts[f.Name] = string(b)
	}

	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml"} {
		if _, ok := parts[name]; !ok {
			t.Errorf("missing part %s", name)
		}
	}

	doc := parts["word/document.xml"]
	if !strings.Contains(doc, "<w:tbl>") {
		t.Error("document has no table")
	}
	if !strings.Contains(doc, "Tecnofer Indústria Metalúrgica Ltda") {
		t.Error("letterhead company name missing")
	}
	if !strings.Contains(doc, "CNPJ: 52.454.668/0001-02") {
		t.Error("letterhead CNPJ line missing")
	}
	if !strings.Contains(doc, "Razão Social") {
		t.Error("column header missing")
	}
}

func TestPDFAndXLSXCarryBranding(t *testing.T) {
	var pdf bytes.Buffer
	if err := EncodePDF(&pdf, brandedSampleTable()); err != nil {
		t.Fatal(err)
	}
	// The company name is WinAnsi-encoded in the PDF content stream; check a
	// plain-ASCII fragment that survives that encoding.
	if !bytes.Contains(pdf.Bytes(), []byte("Tecnofer Ind")) {
		t.Error("PDF missing company letterhead")
	}

	var xlsx bytes.Buffer
	if err := EncodeXLSX(&xlsx, brandedSampleTable()); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(xlsx.Bytes()), int64(xlsx.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var sheet string
	for _, f := range zr.File {
		if f.Name == "xl/worksheets/sheet1.xml" {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			sheet = string(b)
		}
	}
	if !strings.Contains(sheet, "Tecnofer Indústria Metalúrgica Ltda") {
		t.Error("XLSX missing company letterhead")
	}
}
