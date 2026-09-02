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

func TestItemClassificationPreservesClassificationParentCode(t *testing.T) {
	for _, target := range []string{
		"/api/items/classifications/",
		"/api/items/classifications/masks/1/items",
		"/api/items/classifications/1/children",
	} {
		if !nativeItemBusinessCodeRequest(httptest.NewRequest("POST", target, nil)) {
			t.Fatalf("rota %s deveria preservar parent_code como código de classificação", target)
		}
	}
	if nativeItemBusinessCodeRequest(httptest.NewRequest("POST", "/api/items/classifications-report", nil)) {
		t.Fatal("rota não relacionada não deveria ignorar a tradução de códigos de item")
	}
}
