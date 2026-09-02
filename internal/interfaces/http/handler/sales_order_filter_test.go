package handler

import (
	"net/http/httptest"
	"testing"
)

func TestParseSalesOrderFilter(t *testing.T) {
	t.Run("aceita filtros e paginação", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/search?search=cliente&customer_code=10&item_code=20&workflow_status=ATENDIDO&limit=50&offset=5&emission_from=2026-08-01&emission_to=2026-08-31", nil)
		filter, err := parseSalesOrderFilter(r, true)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if filter.Search != "cliente" || filter.CustomerCode == nil || *filter.CustomerCode != 10 || filter.ItemCode == nil || *filter.ItemCode != 20 || filter.WorkflowStatus == nil || *filter.WorkflowStatus != "ATENDIDO" || filter.Limit != 50 || filter.Offset != 5 {
			t.Fatalf("filtro inesperado: %+v", filter)
		}
	})

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{"código inválido", "?customer_code=x", "customer_code"},
		{"booleano inválido", "?is_blocked=talvez", "is_blocked"},
		{"data inválida", "?emission_from=31-08-2026", "emission_from"},
		{"período invertido", "?delivery_from=2026-09-01&delivery_to=2026-08-01", "delivery_from"},
		{"limite excessivo", "?limit=501", "limit"},
		{"offset negativo", "?offset=-1", "offset"},
		{"workflow inválido", "?workflow_status=desconhecido", "workflow_status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/search"+tt.query, nil)
			_, err := parseSalesOrderFilter(r, true)
			if err == nil || !contains(err.Error(), tt.want) {
				t.Fatalf("esperava erro contendo %q, recebeu %v", tt.want, err)
			}
		})
	}
}

func TestParseSalesOrderReportIgnoresPagination(t *testing.T) {
	r := httptest.NewRequest("GET", "/report?limit=inválido&offset=-1", nil)
	filter, err := parseSalesOrderFilter(r, false)
	if err != nil {
		t.Fatalf("relatório não deve paginar: %v", err)
	}
	if filter.Limit != 0 || filter.Offset != 0 {
		t.Fatalf("relatório recebeu paginação: %+v", filter)
	}
}

func contains(value, fragment string) bool {
	for i := 0; i+len(fragment) <= len(value); i++ {
		if value[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
