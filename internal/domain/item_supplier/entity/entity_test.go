package entity

import (
	"testing"

	"github.com/google/uuid"
)

func TestDirectBillingDependencies(t *testing.T) {
	s, err := NewItemPreferredSupplier(1, 2, 3, "", 2, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	s.DirectBilling = true
	if s.Validate() == nil {
		t.Fatal("direct billing without preferred supplier accepted")
	}
	s.IsPreferred = true
	s.ThirdPartyOrder = true
	if err = s.Validate(); err != nil {
		t.Fatalf("valid dependency rejected: %v", err)
	}
	if s.Ranking != 1 {
		t.Fatalf("preferred ranking=%d", s.Ranking)
	}
}
