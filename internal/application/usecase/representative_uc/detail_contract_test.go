package representative_uc

import (
	"encoding/json"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/representative/entity"
)

func TestRepresentativeDetailAlwaysContainsCollectionFields(t *testing.T) {
	encoded, err := json.Marshal(toResponse(&entity.Representative{}))
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"enterprises", "accounting", "regions", "segments", "sales_plans", "interests", "phones", "emails", "correspondence_addresses", "contacts"} {
		value, exists := fields[name]
		if !exists {
			t.Errorf("coleção %s ausente do detalhe", name)
			continue
		}
		if rows, ok := value.([]any); !ok || len(rows) != 0 {
			t.Errorf("coleção %s deveria ser um array vazio, recebido %#v", name, value)
		}
	}
}
