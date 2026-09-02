package entity

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestPurchasePriceValidationAndPrecision(t *testing.T) {
	supplier := int64(10)
	table, err := NewPurchasePriceTable(1, 1, &supplier, " Fornecedor ", "brl", uuid.New())
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

// Uma tabela sem fornecedor é válida: vale para qualquer fornecedor e o preço
// por item define o seu, se houver.
func TestPurchasePriceTableWithoutSupplier(t *testing.T) {
	table, err := NewPurchasePriceTable(1, 1, nil, "Tabela padrão 2026", "", uuid.New())
	if err != nil {
		t.Fatalf("tabela sem fornecedor rejeitada: %v", err)
	}
	if table.SupplierCode != nil {
		t.Fatalf("fornecedor preenchido indevidamente: %v", *table.SupplierCode)
	}
	if table.CurrencyCode != "BRL" {
		t.Fatalf("moeda padrão = %q, quer BRL", table.CurrencyCode)
	}
}

func TestPurchasePriceTableRejectsInvalidSupplier(t *testing.T) {
	invalid := int64(0)
	if _, err := NewPurchasePriceTable(1, 1, &invalid, "Tabela", "BRL", uuid.New()); err == nil {
		t.Fatal("fornecedor zerado aceito")
	}
}
