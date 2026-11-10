package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// FichaDaOrdem é a ordem de fabricação como ela vai para o chão de fábrica:
// cabeçalho, as etapas na sequência do roteiro (com o centro ou o terceiro que
// executa), o material de cada etapa, as ferramentas e as ordens de serviço de
// terceiro. É um modelo de LEITURA — nada aqui é gravado.
type FichaDaOrdem struct {
	ID           int64
	Numero       int64
	ItemCode     int64
	ItemCodigo   string // código de negócio (PORTA-01); vazio quando o item não tem
	ItemNome     string
	Mascara      string
	Unidade      string
	Revisao      string
	Status       string
	QtdPlanejada decimal.Decimal
	QtdProduzida decimal.Decimal
	QtdRefugada  decimal.Decimal
	Inicio       *time.Time
	Fim          *time.Time
	Prioridade   string
	Observacoes  string
	Origem       string
	PedidoVenda  *int64
	// Deposito: almoxarifado que recebe o produzido.
	Deposito       *int64
	Roteiro        *RoteiroDaFicha
	Etapas         []EtapaDaFicha
	Materiais      []MaterialDaFicha
	Ferramentas    []FerramentaDaFicha
	OrdensTerceiro []OrdemTerceiroDaFicha
}

type RoteiroDaFicha struct {
	Codigo      int64
	Alternativa int16
	Descricao   string
}

type EtapaDaFicha struct {
	Sequencia     int16
	Operacao      string
	Centro        string
	Terceiro      string
	Externa       bool
	HorasPrep     decimal.Decimal
	HorasPrevista decimal.Decimal
	HorasReais    decimal.Decimal
	QtdEntra      *decimal.Decimal
	Inspecao      bool
	Status        string
	Observacao    string
	Inicio        *time.Time
	Fim           *time.Time
}

type MaterialDaFicha struct {
	ItemCode   int64
	ItemCodigo string
	ItemNome   string
	Mascara    string
	Quantidade decimal.Decimal
	Atendida   decimal.Decimal
	Unidade    string
	Etapa      *int16
	Deposito   int64
	Tipo       string
}

type FerramentaDaFicha struct {
	Etapa      int16
	Codigo     int64
	Nome       string
	Quantidade decimal.Decimal
}

type OrdemTerceiroDaFicha struct {
	Codigo     int64
	Etapa      *int16
	Fornecedor string
	Quantidade decimal.Decimal
	Recebida   decimal.Decimal
	Prazo      *time.Time
	Status     string
}

// CadastroDoPosto: o que o posto vê de máquina, operador e depósito.
type CadastroDoPosto struct {
	ID     int64  `json:"id"`
	Codigo string `json:"code"`
	Nome   string `json:"name"`
	Ativo  bool   `json:"is_active"`
}

type CadastrosDoPosto struct {
	Maquinas   []CadastroDoPosto `json:"machines"`
	Operadores []CadastroDoPosto `json:"employees"`
	Depositos  []CadastroDoPosto `json:"warehouses"`
}
