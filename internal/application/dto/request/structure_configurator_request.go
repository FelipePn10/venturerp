package request

// Configurador embutido na Estrutura de Produto (VENT0210). Não há tela própria:
// a estrutura abre o painel por um botão e envia as respostas por aqui.

// ApplyStructureConfigurationDTO são as respostas da configuração de um item.
type ApplyStructureConfigurationDTO struct {
	// Answers traz uma entrada por característica respondida; escolha múltipla
	// repete a mesma characteristic_id com variable_id diferentes.
	Answers []CfgMaskAnswerInput `json:"answers"`
	// Persist grava a configuração (máscara + respostas). Sem ele a chamada é
	// apenas uma simulação: valida as restrições e devolve a máscara que sairia.
	Persist bool `json:"persist"`
}
