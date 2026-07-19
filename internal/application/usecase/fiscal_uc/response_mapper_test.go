package fiscal_uc

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

func TestFiscalConfigResponseNeverSerializesFocusToken(t *testing.T) {
	token := "focus-secret-token"
	response := toFiscalConfigResponse(&entity.FiscalConfig{FocusNfeToken: &token})
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	body := string(encoded)
	if strings.Contains(body, token) || strings.Contains(body, "focus_nfe_token") {
		t.Fatalf("FocusNFe token leaked in response: %s", body)
	}
	if !strings.Contains(body, `"focus_nfe_configured":true`) {
		t.Fatalf("configured indicator missing: %s", body)
	}
}
