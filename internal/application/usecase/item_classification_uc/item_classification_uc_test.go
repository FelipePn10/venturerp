package item_classification_uc

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
)

func TestValidateHierarchicalClassification(t *testing.T) {
	if err := validateMask("99.99.99"); err != nil {
		t.Fatal(err)
	}
	if _, err := validateClassificationCode("10", "99.99.99", nil); err != nil {
		t.Fatal(err)
	}
	parent := &entity.ItemClassification{Code: "10", Level: 1}
	if level, err := validateClassificationCode("10.20", "99.99.99", parent); err != nil || level != 2 {
		t.Fatalf("filho valido rejeitado: level=%d err=%v", level, err)
	}
	if _, err := validateClassificationCode("11.20", "99.99.99", parent); err == nil {
		t.Fatal("filho fora do prefixo do pai foi aceito")
	}
	if _, err := validateClassificationCode("10.2", "99.99.99", parent); err == nil {
		t.Fatal("segmento fora da mascara foi aceito")
	}
}

func TestValidateMaskRejectsNonBrazilianHierarchyPattern(t *testing.T) {
	for _, mask := range []string{"", "XX.XX", "99-99", "99..99"} {
		if validateMask(mask) == nil {
			t.Fatalf("mascara invalida aceita: %q", mask)
		}
	}
}
