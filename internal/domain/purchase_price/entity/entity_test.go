package entity

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestPurchasePriceValidationAndPrecision(t *testing.T) {
	table, err := NewPurchasePriceTable(1, 1, 10, " Fornecedor ", "brl", uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if table.CurrencyCode != "BRL" || table.Description != "Fornecedor" {
		t.Fatalf("normalization failed: %+v", table)
	}
	start, end := time.Now(), time.Now().Add(-time.Hour)
	table.ValidityStart, table.ValidityEnd = &start, &end
	if table.ValidateValidity() == nil {
		t.Fatal("inverted validity accepted")
	}
	price := decimal.RequireFromString("123.456789")
	item, err := NewPurchasePriceTableItem(1, 2, price)
	if err != nil {
		t.Fatal(err)
	}
	if !item.Price.Equal(price) {
		t.Fatalf("precision lost: %s", item.Price)
	}
}
