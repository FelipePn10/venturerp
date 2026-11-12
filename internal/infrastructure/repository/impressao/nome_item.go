package impressao

import "fmt"

// O nome do item mora em campos diferentes conforme o cadastro: itens criados
// pelo PDM costumam ter só a descrição técnica (name e commercial_description
// vazios). Sem esse fallback o documento saía com a descrição em branco.

// nomeComercial é o nome que vai para o cliente (pedido, proposta, romaneio).
func nomeComercial(a string) string {
	return fmt.Sprintf(`COALESCE(NULLIF(btrim(%[1]s.commercial_description),''), NULLIF(btrim(%[1]s.pdm_description_technique),''), NULLIF(btrim(%[1]s.name),''), '')`, a)
}

// nomeInterno é o nome dos documentos de uso interno (planejamento, estoque,
// custo, cotação) — a mesma ordem da NF de entrada.
func nomeInterno(a string) string {
	return fmt.Sprintf(`COALESCE(NULLIF(btrim(%[1]s.name),''), NULLIF(btrim(%[1]s.pdm_description_technique),''), NULLIF(btrim(%[1]s.commercial_description),''), '')`, a)
}
