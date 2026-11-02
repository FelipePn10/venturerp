package entity

import "testing"

func TestTemMaterialDoCliente(t *testing.T) {
	casos := []struct {
		nome   string
		filhos []*ItemStructure
		quer   bool
	}{
		{"sem filhos", nil, false},
		{"só material próprio", []*ItemStructure{{ChildCode: 1}, {ChildCode: 2}}, false},
		{"um componente do cliente", []*ItemStructure{{ChildCode: 1}, {ChildCode: 2, CustomerSupplied: true}}, true},
		{"co-produto marcado não conta", []*ItemStructure{{ChildCode: 3, CustomerSupplied: true, IsCoproduct: true}}, false},
		{"linha nula é ignorada", []*ItemStructure{nil, {ChildCode: 4, CustomerSupplied: true}}, true},
	}
	for _, c := range casos {
		if got := TemMaterialDoCliente(c.filhos); got != c.quer {
			t.Errorf("%s: esperado %v, veio %v", c.nome, c.quer, got)
		}
	}
}
