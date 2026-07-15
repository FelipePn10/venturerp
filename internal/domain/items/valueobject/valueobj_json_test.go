package valueobject

import (
	"encoding/json"
	"testing"
)

func TestWeightUnmarshalPublicDTO(t *testing.T) {
	var got Weight
	if err := json.Unmarshal([]byte(`{"gross":2.5,"net":2,"unit":"KG"}`), &got); err != nil {
		t.Fatalf("unmarshal weight: %v", err)
	}
	if got.Gross != 2.5 || got.Net != 2 || got.Unit != "KG" {
		t.Fatalf("unexpected weight: %+v", got)
	}
}
