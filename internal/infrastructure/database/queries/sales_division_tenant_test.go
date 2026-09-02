package queries_test

import (
	"os"
	"regexp"
	"testing"
)

func TestSalesDivisionQueriesAlwaysScopeTenant(t *testing.T) {
	raw, err := os.ReadFile("sales_division.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	for _, operation := range []string{
		`(?s)-- name: CreateSalesDivision.*?enterprise_id`,
		`(?s)-- name: UpdateSalesDivision.*?WHERE code = \$1 AND enterprise_id`,
		`(?s)-- name: GetSalesDivisionByCode.*?WHERE code = \$1 AND enterprise_id`,
		`(?s)-- name: ListSalesDivisions.*?WHERE enterprise_id`,
		`(?s)-- name: ListActiveSalesDivisions.*?WHERE enterprise_id`,
		`(?s)-- name: DeleteSalesDivision.*?WHERE code = \$1 AND enterprise_id`,
	} {
		if !regexp.MustCompile(operation).MatchString(sql) {
			t.Errorf("query de divisão sem isolamento obrigatório: %s", operation)
		}
	}
}
