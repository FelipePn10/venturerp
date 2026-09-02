package response

// Painel do configurador embutido na Estrutura de Produto (VENT0210). Não existe
// tela própria: a estrutura abre este painel por um botão, com tudo o que ele
// precisa em uma única leitura — as perguntas do item, as respostas possíveis,
// as configurações já geradas e os componentes cuja quantidade sai de fórmula.

// StructureConfiguratorOption é uma resposta possível de uma pergunta.
type StructureConfiguratorOption struct {
	VariableID int64 `json:"variable_id"`
	// Code é o valor canônico comparado pelas restrições.
	Code string `json:"code"`
	// MaskComposition é o pedaço que essa resposta acrescenta à máscara.
	MaskComposition string `json:"mask_composition"`
	Description     string `json:"description"`
	IsDefault       bool   `json:"is_default"`
}

// StructureConfiguratorQuestion é uma pergunta do item, na ordem em que a tela
// deve apresentá-la.
type StructureConfiguratorQuestion struct {
	CharacteristicID int64  `json:"characteristic_id"`
	ItemCharID       int64  `json:"item_characteristic_id"`
	Sequence         int    `json:"sequence"`
	Code             string `json:"code"`
	Description      string `json:"description"`
	// Type é o tipo da pergunta (ESCOLHA, INF_NUMERICA, OPCAO, …) e TypeLabel a
	// mesma informação já pronta para exibição.
	Type      string `json:"type"`
	TypeLabel string `json:"type_label"`
	Required  bool   `json:"required"`
	// AllowsMultiple indica pergunta de escolha múltipla.
	AllowsMultiple bool     `json:"allows_multiple"`
	Mask           string   `json:"mask,omitempty"`
	Formula        string   `json:"formula,omitempty"`
	NumMin         *float64 `json:"num_min,omitempty"`
	NumMax         *float64 `json:"num_max,omitempty"`
	NumMultiple    *float64 `json:"num_multiple,omitempty"`
	OptionTrue     string   `json:"option_true,omitempty"`
	OptionFalse    string   `json:"option_false,omitempty"`
	// UsedByFormula marca as perguntas que alimentam alguma fórmula de
	// quantidade da estrutura — a tela destaca essas como essenciais.
	UsedByFormula bool                          `json:"used_by_formula"`
	Options       []StructureConfiguratorOption `json:"options"`
}

// StructureConfiguratorFormula é um componente da estrutura cuja quantidade sai
// de fórmula.
type StructureConfiguratorFormula struct {
	ChildCode        int64    `json:"child_code"`
	ChildDescription string   `json:"child_description"`
	Formula          string   `json:"formula"`
	Rounding         string   `json:"quantity_rounding"`
	Scale            int16    `json:"quantity_scale"`
	NominalQuantity  float64  `json:"nominal_quantity"`
	UnitOfMeasure    string   `json:"unit_of_measurement"`
	Variables        []string `json:"variables"`
}

// StructureConfiguratorMask é uma configuração já gerada para o item.
type StructureConfiguratorMask struct {
	ID   int64  `json:"id"`
	Mask string `json:"mask"`
	Hash string `json:"mask_hash"`
	// Answered distingue as máscaras geradas pelo configurador (com respostas
	// gravadas) das criadas por propagação.
	Answered bool `json:"answered"`
}

// StructureConfiguratorPanelResponse é a carga completa do botão "Configurador"
// dentro da Estrutura de Produto.
type StructureConfiguratorPanelResponse struct {
	ItemCode string `json:"item_code"`
	ItemName string `json:"item_name,omitempty"`
	// Configurable é falso quando o item não tem perguntas cadastradas: a tela
	// mantém o botão desabilitado e explica o motivo em Message.
	Configurable bool   `json:"configurable"`
	Message      string `json:"message,omitempty"`
	// RestrictionsEnabled informa se o motor de restrições (dependências entre
	// respostas) está ativo para validar as combinações.
	RestrictionsEnabled bool                            `json:"restrictions_enabled"`
	Questions           []StructureConfiguratorQuestion `json:"questions"`
	Masks               []StructureConfiguratorMask     `json:"masks"`
	Formulas            []StructureConfiguratorFormula  `json:"formulas"`
	// MissingFormulaVariables lista variáveis usadas por fórmulas da estrutura
	// que não têm pergunta cadastrada — a estrutura nunca conseguiria calculá-las.
	MissingFormulaVariables []string `json:"missing_formula_variables,omitempty"`
}

// StructureConfiguratorViolation é uma restrição/dependência violada pela
// combinação de respostas (equivalente ao FENG0116).
type StructureConfiguratorViolation struct {
	RestrictionCode  int64  `json:"restriction_code"`
	CharacteristicID int64  `json:"characteristic_id"`
	Question         string `json:"question,omitempty"`
	Operator         string `json:"operator"`
	ExpectedValue    string `json:"expected_value,omitempty"`
	AnsweredValue    string `json:"answered_value,omitempty"`
	Message          string `json:"message"`
}

// StructureConfiguratorApplyResponse devolve a configuração aplicada: a máscara
// gerada, as respostas e a estrutura já resolvida com as fórmulas avaliadas.
type StructureConfiguratorApplyResponse struct {
	ItemCode string `json:"item_code"`
	Mask     string `json:"mask"`
	MaskHash string `json:"mask_hash"`
	// Persisted diz se a configuração foi gravada (e passa a valer para MRP,
	// custo e ordens) ou se foi apenas uma simulação.
	Persisted bool                    `json:"persisted"`
	MaskID    *int64                  `json:"mask_id,omitempty"`
	Answers   []CfgMaskAnswerResponse `json:"answers"`
	// Variables são os valores numéricos que alimentaram as fórmulas.
	Variables map[string]float64 `json:"variables,omitempty"`
	// Structure é a estrutura resolvida para esta configuração; vem preenchida
	// apenas quando a configuração foi gravada.
	Structure *StructureTreeResponse `json:"structure,omitempty"`
	// Warnings reúne avisos que não impedem a configuração (por exemplo, uma
	// fórmula que ficou sem variável).
	Warnings []string `json:"warnings,omitempty"`
}
