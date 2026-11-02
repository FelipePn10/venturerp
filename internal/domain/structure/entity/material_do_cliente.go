package entity

// TemMaterialDoCliente diz se a estrutura leva material fornecido pelo cliente
// — ou seja, se a ordem do pai é de BENEFICIAMENTO (industrialização por
// encomenda). Nesse caso o produto processado pertence ao cliente e volta pela
// nota de retorno; ele não entra no estoque próprio.
//
// Co-produto não conta: é saída da estrutura, não insumo.
func TemMaterialDoCliente(children []*ItemStructure) bool {
	for _, c := range children {
		if c != nil && c.CustomerSupplied && !c.IsCoproduct {
			return true
		}
	}
	return false
}
