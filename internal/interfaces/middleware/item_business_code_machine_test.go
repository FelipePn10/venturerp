package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestMachineTimeUsesNativeBusinessItemCodeRequest(t *testing.T) {
	for _, target := range []string{
		"/api/machine/time/create",
		"/api/machine/time/list?item_code=TEA452-0",
	} {
		if !nativeItemBusinessCodeRequest(httptest.NewRequest("GET", target, nil)) {
			t.Fatalf("rota %s deveria preservar o código comercial", target)
		}
	}
	if nativeItemBusinessCodeRequest(httptest.NewRequest("POST", "/api/machine/time/production/calculate", nil)) {
		t.Fatal("cálculo legado não deveria mudar de contrato neste ajuste")
	}
}
