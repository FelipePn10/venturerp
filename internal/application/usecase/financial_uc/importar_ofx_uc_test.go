package financial_uc

import (
	"testing"
	"time"
)

const sampleOFX1 = `OFXHEADER:100
DATA:OFXSGML
VERSION:151
SECURITY:NONE
ENCODING:UTF-8
CHARSET:1252
COMPRESSION:NONE
OLDFILEUID:NONE
NEWFILEUID:NONE

<OFX>
<BANKMSGSRSV1>
<STMTTRNRS>
<STMTRS>
<BANKTRANLIST>
<DTSTART>20240101120000</DTSTART>
<DTEND>20240131120000</DTEND>
<STMTTRN>
<TRNTYPE>DEBIT</TRNTYPE>
<DTPOSTED>20240115</DTPOSTED>
<TRNAMT>-150.00</TRNAMT>
<FITID>TXN001</FITID>
<MEMO>Pagamento fornecedor</MEMO>
</STMTTRN>
<STMTTRN>
<TRNTYPE>CREDIT</TRNTYPE>
<DTPOSTED>20240120</DTPOSTED>
<TRNAMT>500.00</TRNAMT>
<FITID>TXN002</FITID>
<MEMO>Recebimento cliente</MEMO>
</STMTTRN>
</BANKTRANLIST>
</STMTRS>
</STMTTRNRS>
</BANKMSGSRSV1>
</OFX>`

const sampleOFX2XML = `<?xml version="1.0" encoding="UTF-8"?>
<OFX>
<BANKMSGSRSV1>
<STMTTRNRS>
<STMTRS>
<BANKTRANLIST>
<STMTTRN>
<TRNTYPE>CREDIT</TRNTYPE>
<DTPOSTED>20240205120000.000</DTPOSTED>
<TRNAMT>1200.50</TRNAMT>
<FITID>XML001</FITID>
<MEMO>Transferência recebida</MEMO>
</STMTTRN>
<STMTTRN>
<TRNTYPE>DEBIT</TRNTYPE>
<DTPOSTED>20240210</DTPOSTED>
<TRNAMT>-300.00</TRNAMT>
<FITID>XML002</FITID>
<MEMO>IOF</MEMO>
</STMTTRN>
</BANKTRANLIST>
</STMTRS>
</STMTTRNRS>
</BANKMSGSRSV1>
</OFX>`

func TestParseOFX_SGML_TwoTransactions(t *testing.T) {
	txns, err := parseOFX(sampleOFX1)
	if err != nil {
		t.Fatal(err)
	}
	if len(txns) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txns))
	}

	t.Run("debit transaction", func(t *testing.T) {
		tx := txns[0]
		if tx.TrnType != "DEBIT" {
			t.Errorf("TrnType: want DEBIT, got %s", tx.TrnType)
		}
		if tx.FitID != "TXN001" {
			t.Errorf("FitID: want TXN001, got %s", tx.FitID)
		}
		if tx.TrnAmt != "-150.00" {
			t.Errorf("TrnAmt: want -150.00, got %s", tx.TrnAmt)
		}
		if tx.Memo != "Pagamento fornecedor" {
			t.Errorf("Memo: want 'Pagamento fornecedor', got %s", tx.Memo)
		}
	})

	t.Run("credit transaction", func(t *testing.T) {
		tx := txns[1]
		if tx.TrnType != "CREDIT" {
			t.Errorf("TrnType: want CREDIT, got %s", tx.TrnType)
		}
		if tx.FitID != "TXN002" {
			t.Errorf("FitID: want TXN002, got %s", tx.FitID)
		}
		if tx.TrnAmt != "500.00" {
			t.Errorf("TrnAmt: want 500.00, got %s", tx.TrnAmt)
		}
	})
}

func TestParseOFX_XML_TwoTransactions(t *testing.T) {
	txns, err := parseOFX(sampleOFX2XML)
	if err != nil {
		t.Fatal(err)
	}
	if len(txns) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txns))
	}

	if txns[0].FitID != "XML001" {
		t.Errorf("FitID: want XML001, got %s", txns[0].FitID)
	}
	if txns[1].FitID != "XML002" {
		t.Errorf("FitID: want XML002, got %s", txns[1].FitID)
	}
}

func TestParseOFXDate_Formats(t *testing.T) {
	cases := []struct {
		input string
		year  int
		month time.Month
		day   int
	}{
		{"20240115", 2024, time.January, 15},
		{"20240205120000", 2024, time.February, 5},
		{"20240205120000.000", 2024, time.February, 5},
		{"20240205120000.000[-03:BRT]", 2024, time.February, 5},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseOFXDate(tc.input)
			if err != nil {
				t.Fatalf("parseOFXDate(%q): %v", tc.input, err)
			}
			if got.Year() != tc.year || got.Month() != tc.month || got.Day() != tc.day {
				t.Errorf("want %d-%02d-%02d, got %s", tc.year, tc.month, tc.day, got.Format("2006-01-02"))
			}
		})
	}
}

func TestParseOFXDate_Invalid(t *testing.T) {
	_, err := parseOFXDate("not-a-date")
	if err == nil {
		t.Error("expected error for invalid date, got nil")
	}
}

func TestParseOFX_EmptyContent(t *testing.T) {
	txns, err := parseOFX("")
	if err != nil {
		t.Fatal(err)
	}
	if len(txns) != 0 {
		t.Errorf("expected 0 transactions for empty content, got %d", len(txns))
	}
}

func TestParseOFX_NoTransactions(t *testing.T) {
	content := `<OFX><BANKMSGSRSV1></BANKMSGSRSV1></OFX>`
	txns, err := parseOFX(content)
	if err != nil {
		t.Fatal(err)
	}
	if len(txns) != 0 {
		t.Errorf("expected 0 transactions, got %d", len(txns))
	}
}
