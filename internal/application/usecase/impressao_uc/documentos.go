// Package impressao_uc monta os documentos impressos do ERP (pedido de venda,
// espelho da nota de entrada, títulos, plano de materiais, capacidade,
// programação, kardex, custo de produção e romaneio). Cada documento é um
// modelo de leitura completo: a consulta vem da infraestrutura e o desenho do
// PDF também, cada um com o layout próprio do documento.
package impressao_uc

import (
	"time"

	"github.com/shopspring/decimal"
)

// Parte identifica uma empresa no documento: emitente, cliente, fornecedor,
// transportadora.
type Parte struct {
	Codigo   int64
	Nome     string
	Fantasia string
	CNPJCPF  string
	IE       string
	Endereco string // logradouro, número e complemento
	Bairro   string
	Cidade   string
	UF       string
	CEP      string
	Telefone string
	Email    string
}

// Emitente é a empresa que imprime, com a identidade visual dela.
type Emitente struct {
	Parte
	Logo     []byte
	CorMarca string
}

// Cabecalho comum de todo documento impresso.
type Cabecalho struct {
	Emitente   Emitente
	Gerado     time.Time
	EmitidoPor string
}

// ─── Pedido de venda ────────────────────────────────────────────────────────

type ItemPedidoVenda struct {
	Seq                        int
	Codigo, Descricao, NCM, UM string
	Quantidade, Atendida       decimal.Decimal
	Cancelada                  decimal.Decimal
	Preco, DescontoPct, Total  decimal.Decimal
	IPIPct, IPI, ICMSPct, ST   decimal.Decimal
	PISPct, COFINSPct          decimal.Decimal
	Entrega                    *time.Time
	Deposito                   string
	PesoLiquido, PesoBruto     decimal.Decimal // unitários
	Situacao                   string
}

type Parcela struct {
	Numero     int
	Dias       int
	Vencimento time.Time
	Percentual decimal.Decimal
	Valor      decimal.Decimal
	Descricao  string
}

type NotaVinculada struct {
	Numero   int64
	Serie    string
	Emissao  time.Time
	Valor    decimal.Decimal
	Situacao string
	Chave    string
}

type PedidoVenda struct {
	Cabecalho
	Codigo, Numero    int64
	Emissao           time.Time
	Entrega           *time.Time
	EntregaFirme      bool
	Situacao          string // R, A, P, F, C…
	Bloqueado         bool
	MotivoBloqueio    string
	Origem, Canal     string
	Divisao           string
	Representante     string
	ComissaoPct       decimal.Decimal
	Tabela            string
	Condicao          string
	Moeda             string
	TipoFrete         string
	Frete, Seguro     decimal.Decimal
	Desconto          decimal.Decimal
	Observacao        string
	ObservacaoCliente string
	Plano             string
	Cliente           Parte
	EnderecoEntrega   string
	Transportadora    string
	Itens             []ItemPedidoVenda
	Parcelas          []Parcela
	Notas             []NotaVinculada
}

// ─── Nota fiscal de entrada (espelho) ───────────────────────────────────────

type ItemNotaEntrada struct {
	Seq                              int
	Codigo, Descricao, NCM, CFOP, UM string
	CST                              string
	Quantidade, Unitario, Total      decimal.Decimal
	BaseICMS, AliqICMS, ICMS         decimal.Decimal
	AliqIPI, IPI, PIS, COFINS        decimal.Decimal
	CreditaICMS, CreditaIPI          bool
	CreditaPIS, CreditaCOFINS        bool
	PlanoContas, Deposito            string
	Pedido, LinhaPedido              int64
	QtdPedido, PrecoPedido           decimal.Decimal
	QtdEstoque                       decimal.Decimal
	CustoAquisicao                   decimal.Decimal
}

type ParcelaNota struct {
	Numero     int
	Documento  string
	Vencimento time.Time
	Valor      decimal.Decimal
	Titulo     string // situação do contas a pagar gerado
}

type Divergencia struct{ Nivel, Mensagem string }

type NotaEntrada struct {
	Cabecalho
	ID                              int64
	Numero                          int64
	Serie, Modelo, Chave, Protocolo string
	Natureza                        string
	Emissao, Entrada                time.Time
	Situacao                        string
	Fornecedor                      Parte
	PedidoCompra                    int64
	Produtos, Frete, Seguro         decimal.Decimal
	Desconto, Outras                decimal.Decimal
	BaseICMS, ICMS, ICMSST, IPI     decimal.Decimal
	PIS, COFINS, Total              decimal.Decimal
	Observacao                      string
	AprovadaEm                      *time.Time
	AprovadaPor                     string
	Itens                           []ItemNotaEntrada
	Parcelas                        []ParcelaNota
	Divergencias                    []Divergencia
}

// ─── Títulos (contas a receber e a pagar) ───────────────────────────────────

type Titulo struct {
	ID                       int64
	Documento                string
	Parte                    Parte
	Emissao, Vencimento      time.Time
	Pagamento                *time.Time
	Parcela, Parcelas        int
	Valor, Desconto          decimal.Decimal
	Juros, Multa, Pago       decimal.Decimal
	Situacao                 string
	Aprovacao                string
	PlanoContas, CentroCusto string
	Origem                   string // "NF-e 66", "NF 900101 · pedido 4"
	Forma                    string
}

type ContaBancaria struct {
	Banco, Agencia, Conta, Titular, Pix, Descricao string
}

// FiltroTitulos escolhe os títulos do relatório.
type FiltroTitulos struct {
	ParteID        *int64
	NotaFiscalID   *int64
	De, Ate        *time.Time
	SomenteAbertos bool
	// Os mesmos filtros da carteira na tela, para o PDF trazer o que a lista
	// mostra: período pela emissão (o padrão é o vencimento), situação exata e
	// só os vencidos (em aberto com vencimento anterior a hoje).
	PorEmissao      bool
	Situacao        string
	SomenteVencidos bool
	// Filtros secundários da carteira. Plano de contas e centro de custo valem
	// pelo título ou pelo rateio (só contas a pagar têm rateio); aprovação e
	// tipo de documento existem só em contas a pagar, pedido só em receber.
	Documento     string
	TipoDocumento string
	Aprovacao     string
	PlanoContasID *int64
	CentroCustoID *int64
	PedidoID      *int64
	ValorMinimo   *decimal.Decimal
	ValorMaximo   *decimal.Decimal
}

type Titulos struct {
	Cabecalho
	Receber    bool
	Filtro     string // descrição legível do filtro
	Referencia time.Time
	Titulos    []Titulo
	Contas     []ContaBancaria
}

// ─── Plano de materiais (MRP) ───────────────────────────────────────────────

type DemandaPlano struct {
	Origem, Documento, Cliente string
	Codigo, Descricao, UM      string
	Quantidade                 decimal.Decimal
	Data                       time.Time
}

type SugestaoPlano struct {
	Codigo, Numero      int64
	Item, Descricao, UM string
	Tipo                string // FABRICACAO | COMPRA
	Nivel               int
	Pai                 string // código do item pai
	Quantidade          decimal.Decimal
	Inicio, Necessidade time.Time
	Origem              string
	Firmada             bool
	Gerou               string // "OF 5", "PC 3"
	Fornecedor          string
	PrecoReferencia     decimal.Decimal
	Roteiro             string
	HorasPrevistas      decimal.Decimal
}

type ExcecaoPlano struct {
	Item, Tipo, Descricao string
	Data                  *time.Time
	Tratada               bool
}

type PlanoMateriais struct {
	Cabecalho
	Plano       int64
	Nome        string
	CalculadoEm *time.Time
	Status      string
	Erros       string
	Demandas    []DemandaPlano
	Sugestoes   []SugestaoPlano
	Excecoes    []ExcecaoPlano
}

// ─── Capacidade (CRP) ───────────────────────────────────────────────────────

type CargaCentro struct {
	Centro     string
	Data       time.Time
	Requerido  decimal.Decimal
	Disponivel decimal.Decimal
	CargaPct   decimal.Decimal
}

type OrdemNaCarga struct {
	Centro, Ordem, Item, Operacao string
	Data                          time.Time
	Horas                         decimal.Decimal
}

type Capacidade struct {
	Cabecalho
	Plano    int64
	Nome     string
	Cargas   []CargaCentro
	Ordens   []OrdemNaCarga
	Maquinas map[string][]string // centro → máquinas
}

// ─── Programação (APS) ──────────────────────────────────────────────────────

type OperacaoProgramada struct {
	Maquina, Centro string
	Ordem           int64
	Item, Descricao string
	Etapa           int
	Operacao        string
	Quantidade      decimal.Decimal
	Inicio, Fim     time.Time
	Situacao        string
	Necessidade     *time.Time
}

type Programacao struct {
	Cabecalho
	De, Ate   time.Time
	Grupo     string
	Operacoes []OperacaoProgramada
	// Agrupamento nomeia as linhas do Gantt: "máquina" (padrão), "centro de
	// trabalho" ou "ordem de produção" (quadro do APS agrupado por ordem).
	Agrupamento string
}

// ─── Kardex ─────────────────────────────────────────────────────────────────

type MovimentoKardex struct {
	Data              time.Time
	Deposito, Tipo    string
	Entrada           bool
	Quantidade, Custo decimal.Decimal
	// Valor: efeito do movimento no valor do estoque, com sinal. O ajuste de
	// custo tem quantidade zero e só valor.
	Valor             decimal.Decimal
	Origem, Lote, Obs string
}

type ItemKardex struct {
	Codigo, Descricao, UM      string
	SaldoInicial, CustoInicial decimal.Decimal
	Movimentos                 []MovimentoKardex
	SaldoFinal, CustoMedio     decimal.Decimal
	// ValorFinal é o valor acumulado do estoque no fim do período, o mesmo
	// do kardex da tela (com saldo negativo o custo médio não existe).
	ValorFinal decimal.Decimal
}

type FiltroKardex struct {
	Item     string
	Deposito *int64
	De, Ate  time.Time
}

type Kardex struct {
	Cabecalho
	De, Ate  time.Time
	Deposito string
	Itens    []ItemKardex
}

// ─── Custo de produção ──────────────────────────────────────────────────────

type ConsumoOrdem struct {
	Codigo, Descricao, UM string
	Quantidade, Custo     decimal.Decimal
}

type EtapaOrdem struct {
	Seq              int
	Operacao, Centro string
	Prevista, Real   decimal.Decimal
	Situacao         string
}

type CustoOrdem struct {
	Ordem                                int64
	Item, Descricao, UM                  string
	Quantidade                           decimal.Decimal
	Inicio, Fim                          *time.Time
	Situacao                             string
	Material, Maquina, MaoDeObra         decimal.Decimal
	Overhead, Terceiros, Total, Unitario decimal.Decimal
	PadraoMaterial, PadraoMaquina        decimal.Decimal
	PadraoMaoDeObra, PadraoOverhead      decimal.Decimal
	PadraoTerceiros, PadraoTotal         decimal.Decimal
	Apurado                              *time.Time
	Consumos                             []ConsumoOrdem
	Etapas                               []EtapaOrdem
}

type CustoProducao struct {
	Cabecalho
	Filtro string
	Ordens []CustoOrdem
}

// ─── Romaneio ───────────────────────────────────────────────────────────────

type ItemRomaneio struct {
	Seq                    int
	Codigo, Descricao, UM  string
	Deposito               string
	Quantidade, Conferida  decimal.Decimal
	Conferido              bool
	PesoLiquido, PesoBruto decimal.Decimal // unitários
}

type VolumeRomaneio struct {
	Numero                        int
	Embalagem, Marcacao, Conteudo string
	PesoLiquido, PesoBruto        decimal.Decimal
	Comprimento, Largura, Altura  decimal.Decimal
	Cubagem                       decimal.Decimal
}

type Romaneio struct {
	Cabecalho
	Codigo                          int64
	Situacao                        string
	Criado                          time.Time
	Separado, Conferido, Despachado *time.Time
	PrevisaoEntrega                 *time.Time
	PedidoVenda                     int64
	Destinatario                    Parte
	EnderecoEntrega                 string
	Transportadora                  Parte
	Placa, Motorista                string
	DocMotorista, ANTT              string
	Lacres                          string
	ModalidadeFrete                 string
	Frete, Seguro                   decimal.Decimal
	NFNumero                        int64
	NFSerie, NFChave                string
	NFValor                         decimal.Decimal
	Volumes                         int
	PesoLiquido                     decimal.Decimal
	PesoBruto                       decimal.Decimal
	Cubagem                         decimal.Decimal
	Observacao                      string
	Itens                           []ItemRomaneio
	Embalagens                      []VolumeRomaneio
}

// ─── Proposta comercial (orçamento de venda) ────────────────────────────────

type Proposta struct {
	Cabecalho
	Codigo, Numero    int64
	Emissao           time.Time
	Validade, Entrega *time.Time
	Situacao          string
	PedidoCliente     string
	Cliente           Parte
	Representante     string
	Tabela, Condicao  string
	Moeda, TipoFrete  string
	Frete, Seguro     decimal.Decimal
	Desconto          decimal.Decimal
	Observacao        string
	ObservacaoCliente string
	Convertida        int64 // pedido de venda gerado, quando houver
	Itens             []ItemPedidoVenda
	Parcelas          []Parcela
}

// comoPedido reaproveita totais e parcelas do pedido de venda.
func (p Proposta) comoPedido() PedidoVenda {
	return PedidoVenda{Emissao: p.Emissao, Frete: p.Frete, Seguro: p.Seguro, Desconto: p.Desconto, Itens: p.Itens, Parcelas: p.Parcelas}
}

func (p Proposta) Totais() TotaisPedido { return p.comoPedido().Totais() }

// ─── Cotação de compra ──────────────────────────────────────────────────────

type PrecoCotacao struct {
	Fornecedor int64
	Unitario   decimal.Decimal
	PrazoDias  int
	Condicao   string
	Obs        string
	Escolhido  bool
}

type ItemCotacao struct {
	Seq                   int
	ID                    int64
	Codigo, Descricao, UM string
	Quantidade            decimal.Decimal
	Entrega               *time.Time
	Precos                []PrecoCotacao
}

type Cotacao struct {
	Cabecalho
	Codigo       int64
	Emissao      time.Time
	Situacao     string
	Observacao   string
	Comprador    string
	Fornecedores []Parte
	Fornecedor   *Parte // solicitação para um fornecedor; nil = mapa comparativo
	Itens        []ItemCotacao
}

// ─── Ficha cadastral do cliente ─────────────────────────────────────────────

type EnderecoCliente struct {
	Tipo, Endereco, Bairro, Cidade, UF, CEP string
	Padrao                                  bool
}

type ContatoCliente struct {
	Nome, Cargo, Email, Telefone, Celular string
	Principal                             bool
}

type FichaCliente struct {
	Cabecalho
	Cliente                Parte
	Matriz                 string // "código — nome" quando o cliente é filial
	Tipo, Regiao, Segmento string
	Finalidade             string
	InscricaoMunicipal     string
	Suframa, Site          string
	Ativo, Bloqueado       bool
	MotivoBloqueio         string
	Cadastro               time.Time
	LimiteCredito          decimal.Decimal
	Condicao, Tabela       string
	Transportadora         string
	Enderecos              []EnderecoCliente
	Contatos               []ContatoCliente
	Filiais                []Parte
	// Posição comercial na data da impressão.
	TitulosAbertos    int
	EmAberto, Vencido decimal.Decimal
	PedidosAbertos    int
	ValorPedidos      decimal.Decimal
	Faturado12m       decimal.Decimal
	UltimaCompra      *time.Time
	UltimaNF          int64
}
