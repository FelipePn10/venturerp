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

func TestAmbiguousItemReferenceKeysAreScopedByRoute(t *testing.T) {
	classification := httptest.NewRequest("POST", "/api/items/classifications/", nil)
	if isItemReferenceKey(classification, "parent_code") {
		t.Fatal("parent_code de classificação não pode ser tratado como código de item")
	}
	productionPlan := httptest.NewRequest("POST", "/api/production-plan/create", nil)
	if isItemReferenceKey(productionPlan, "class_item_codes") {
		t.Fatal("class_item_codes contém códigos de classificação, não códigos de itens")
	}
	structure := httptest.NewRequest("POST", "/api/items/structure/create", nil)
	for _, key := range []string{"parent_code", "child_code", "item_code"} {
		if !isItemReferenceKey(structure, key) {
			t.Fatalf("%s da estrutura deveria continuar sendo tratado como código de item", key)
		}
	}
}
