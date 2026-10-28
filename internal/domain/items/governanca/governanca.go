// Package governanca: governança do cadastro de item (Fase 4) — status do
// cadastro, liberação por visão, campos obrigatórios por tipo de item,
// duplicidade e descontinuação com substituto.
package governanca

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// Status do cadastro. Item sem governança gravada é LIBERADO.
const (
	EmCadastro    = "EM_CADASTRO"
	Liberado      = "LIBERADO"
	Bloqueado     = "BLOQUEADO"
	Descontinuado = "DESCONTINUADO"
)

// Visões do cadastro, cada uma liberada pela sua área.
const (
	Engenharia   = "ENGENHARIA"
	Fiscal       = "FISCAL"
	Suprimentos  = "SUPRIMENTOS"
	Planejamento = "PLANEJAMENTO"
	Comercial    = "COMERCIAL"
	Custos       = "CUSTOS"
)

var Visoes = []string{Engenharia, Fiscal, Suprimentos, Planejamento, Comercial, Custos}

// Tipos do item (items.engineering_type).
const (
	Fabricado  = 0
	Comprado   = 1
	DeTerceiro = 2
	Servico    = 3
)

func VisaoValida(v string) bool {
	for _, x := range Visoes {
		if x == v {
			return true
		}
	}
	return false
}

func RotuloStatus(s string) string {
	switch s {
	case EmCadastro:
		return "em cadastro"
	case Liberado:
		return "liberado"
	case Bloqueado:
		return "bloqueado"
	case Descontinuado:
		return "descontinuado"
	}
	return strings.ToLower(s)
}

func RotuloTipo(t int) string {
	switch t {
	case Fabricado:
		return "fabricado"
	case Comprado:
		return "comprado"
	case DeTerceiro:
		return "de terceiro"
	case Servico:
		return "serviço"
	}
	return fmt.Sprint(t)
}

// ─── Ficha e campos obrigatórios ─────────────────────────────────────────────

// Ficha: o cadastro do item (as colunas de items) e os fatos de outras tabelas
// que a completude precisa.
type Ficha struct {
	Dados           map[string]json.RawMessage
	Tipo            int
	TemEstrutura    bool
	TemRoteiro      bool
	TemFornecedor   bool
	TemCustoPadrao  bool
	TemConversao    bool // conversão da unidade de compra para a de estoque
	CompraEmOutraUM bool
}

func (f Ficha) texto(campo string) string {
	raw, ok := f.Dados[campo]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(string(raw))
}

// preenchido: texto não vazio, número diferente de zero, JSON com conteúdo.
func (f Ficha) preenchido(campo string) bool {
	v := f.texto(campo)
	switch v {
	case "", "0", "{}", "[]", "false", `""`:
		return false
	}
	if strings.HasPrefix(v, "{") {
		var m map[string]any
		if json.Unmarshal([]byte(v), &m) == nil {
			for _, x := range m {
				switch y := x.(type) {
				case nil:
				case float64:
					if y != 0 {
						return true
					}
				case string:
					if strings.TrimSpace(y) != "" {
						return true
					}
				default:
					return true
				}
			}
			return false
		}
	}
	return true
}

// Campo do catálogo de completude.
type Campo struct {
	Chave  string `json:"campo"`
	Rotulo string `json:"rotulo"`
	Visao  string `json:"visao"`
	// Tipos de item a que se aplica (vazio = todos).
	Tipos []int `json:"tipos,omitempty"`
	// Padrao: obrigatório quando a empresa não ajustou.
	Padrao bool `json:"padrao"`
	checar func(Ficha) bool
}

func coluna(nome string) func(Ficha) bool { return func(f Ficha) bool { return f.preenchido(nome) } }

var fisicos = []int{Fabricado, Comprado, DeTerceiro}

// Catalogo de campos por visão (as chaves de coluna são as de items).
var Catalogo = []Campo{
	{Chave: "name", Rotulo: "Nome do item", Visao: Engenharia, Padrao: true, checar: coluna("name")},
	{Chave: "pdm_group_code", Rotulo: "Grupo do PDM", Visao: Engenharia, Padrao: true, checar: coluna("pdm_group_code")},
	{Chave: "pdm_description_technique", Rotulo: "Descrição técnica", Visao: Engenharia, Padrao: true, checar: coluna("pdm_description_technique")},
	{Chave: "engineering_weight", Rotulo: "Peso", Visao: Engenharia, Tipos: []int{Fabricado, Comprado}, Padrao: false, checar: coluna("engineering_weight")},
	{Chave: "estrutura", Rotulo: "Estrutura (lista de materiais)", Visao: Engenharia, Tipos: []int{Fabricado}, Padrao: true, checar: func(f Ficha) bool { return f.TemEstrutura }},
	{Chave: "roteiro", Rotulo: "Roteiro de fabricação", Visao: Engenharia, Tipos: []int{Fabricado}, Padrao: true, checar: func(f Ficha) bool { return f.TemRoteiro }},

	{Chave: "accounting_sale_fiscal_classification_code", Rotulo: "NCM (classificação fiscal)", Visao: Fiscal, Tipos: fisicos, Padrao: true, checar: coluna("accounting_sale_fiscal_classification_code")},
	{Chave: "accounting_origin", Rotulo: "Origem da mercadoria", Visao: Fiscal, Tipos: fisicos, Padrao: true, checar: func(f Ficha) bool { return f.texto("accounting_origin") != "" }},
	{Chave: "accounting_purchase_fiscal_classification_code", Rotulo: "Classificação fiscal de compra", Visao: Fiscal, Tipos: []int{Comprado}, Padrao: false, checar: coluna("accounting_purchase_fiscal_classification_code")},
	{Chave: "accounting_cest", Rotulo: "CEST", Visao: Fiscal, Tipos: fisicos, Padrao: false, checar: coluna("accounting_cest")},

	{Chave: "fornecedor", Rotulo: "Fornecedor preferencial", Visao: Suprimentos, Tipos: []int{Comprado}, Padrao: true, checar: func(f Ficha) bool { return f.TemFornecedor }},
	{Chave: "supplies_purchase_uom", Rotulo: "Unidade de compra", Visao: Suprimentos, Tipos: []int{Comprado}, Padrao: true, checar: coluna("supplies_purchase_uom")},
	{Chave: "conversao_compra", Rotulo: "Conversão da unidade de compra", Visao: Suprimentos, Tipos: []int{Comprado}, Padrao: true,
		checar: func(f Ficha) bool { return !f.CompraEmOutraUM || f.TemConversao }},
	{Chave: "supplies_warehouse_code", Rotulo: "Almoxarifado de recebimento", Visao: Suprimentos, Tipos: []int{Comprado}, Padrao: false, checar: coluna("supplies_warehouse_code")},

	{Chave: "warehouse_code", Rotulo: "Almoxarifado padrão", Visao: Planejamento, Tipos: fisicos, Padrao: true, checar: coluna("warehouse_code")},
	{Chave: "planning_minimum_lot", Rotulo: "Lote mínimo", Visao: Planejamento, Tipos: []int{Fabricado, Comprado}, Padrao: false, checar: coluna("planning_minimum_lot")},
	{Chave: "planning_safety_stock", Rotulo: "Estoque de segurança", Visao: Planejamento, Tipos: []int{Fabricado, Comprado}, Padrao: false, checar: coluna("planning_safety_stock")},
	{Chave: "planner_employee_code", Rotulo: "Planejador responsável", Visao: Planejamento, Tipos: []int{Fabricado, Comprado}, Padrao: false, checar: coluna("planner_employee_code")},

	{Chave: "commercial_sale_type", Rotulo: "Tipo de venda (venda/revenda)", Visao: Comercial, Tipos: []int{Fabricado}, Padrao: true, checar: coluna("commercial_sale_type")},
	{Chave: "commercial_description", Rotulo: "Descrição comercial", Visao: Comercial, Padrao: false, checar: coluna("commercial_description")},

	{Chave: "custo_padrao", Rotulo: "Custo padrão", Visao: Custos, Tipos: []int{Fabricado}, Padrao: true, checar: func(f Ficha) bool { return f.TemCustoPadrao }},
	{Chave: "accounting_inventory_group_code", Rotulo: "Grupo contábil de estoque", Visao: Custos, Tipos: fisicos, Padrao: false, checar: coluna("accounting_inventory_group_code")},
}

func CampoDoCatalogo(chave string) (Campo, bool) {
	for _, c := range Catalogo {
		if c.Chave == chave {
			return c, true
		}
	}
	return Campo{}, false
}

func (c Campo) aplica(tipo int) bool {
	if len(c.Tipos) == 0 {
		return true
	}
	for _, t := range c.Tipos {
		if t == tipo {
			return true
		}
	}
	return false
}

// Pendencia: campo obrigatório faltando numa visão.
type Pendencia struct {
	Visao  string `json:"visao"`
	Campo  string `json:"campo"`
	Rotulo string `json:"rotulo"`
}

// Pendencias da ficha nas visões pedidas (vazio = todas), com os ajustes da
// empresa sobre a obrigatoriedade (campo → obrigatório).
func Pendencias(f Ficha, visoes []string, ajustes map[string]bool) []Pendencia {
	quer := map[string]bool{}
	for _, v := range visoes {
		quer[v] = true
	}
	out := []Pendencia{}
	for _, c := range Catalogo {
		if len(quer) > 0 && !quer[c.Visao] {
			continue
		}
		obrigatorio := c.Padrao
		if v, ok := ajustes[c.Chave]; ok {
			obrigatorio = v
		}
		if !obrigatorio || !c.aplica(f.Tipo) || c.checar(f) {
			continue
		}
		out = append(out, Pendencia{Visao: c.Visao, Campo: c.Chave, Rotulo: c.Rotulo})
	}
	return out
}

// ─── Status ──────────────────────────────────────────────────────────────────

// Mudanca de status pedida pelo usuário.
type Mudanca struct {
	Para       string
	Motivo     string
	Substituto *int64
	Por        *uuid.UUID
}

// Conferir a mudança a partir do status atual. EM_CADASTRO → LIBERADO não é
// manual: acontece quando a última visão exigida é liberada.
func (m Mudanca) Conferir(atual string) error {
	permitido := map[string][]string{
		EmCadastro:    {Bloqueado, Descontinuado},
		Liberado:      {Bloqueado, Descontinuado},
		Bloqueado:     {Liberado, Descontinuado},
		Descontinuado: {Liberado},
	}
	ok := false
	for _, p := range permitido[atual] {
		if p == m.Para {
			ok = true
		}
	}
	if !ok {
		if atual == EmCadastro && m.Para == Liberado {
			return domainerr.NewValidation("o item em cadastro é liberado pelas áreas: libere as visões pendentes")
		}
		return domainerr.NewValidation(fmt.Sprintf("o item está %s e não pode ficar %s", RotuloStatus(atual), RotuloStatus(m.Para)))
	}
	if strings.TrimSpace(m.Motivo) == "" {
		return domainerr.NewValidation("informe o motivo")
	}
	if m.Substituto != nil && m.Para != Descontinuado {
		return domainerr.NewValidation("o substituto só se informa ao descontinuar")
	}
	return nil
}

// StatusAposLiberacao: com todas as visões exigidas liberadas, o item em
// cadastro fica liberado; os outros status não mudam por liberação.
func StatusAposLiberacao(atual string, exigidas []string, liberadas map[string]bool) string {
	if atual != EmCadastro {
		return atual
	}
	for _, v := range exigidas {
		if !liberadas[v] {
			return atual
		}
	}
	return Liberado
}

// ─── Tipos de leitura ────────────────────────────────────────────────────────

type Parametros struct {
	ExigeLiberacao bool     `json:"exige_liberacao"`
	Visoes         []string `json:"visoes"`
}

type Situacao struct {
	ItemCode     int64            `json:"item_code"`
	Codigo       string           `json:"codigo"`
	Nome         string           `json:"nome"`
	Tipo         int              `json:"tipo"`
	Status       string           `json:"status"`
	Motivo       *string          `json:"motivo,omitempty"`
	Substituto   *int64           `json:"substituto_code,omitempty"`
	SubstitutoCd *string          `json:"substituto,omitempty"`
	AlteradoEm   *time.Time       `json:"alterado_em,omitempty"`
	Liberadas    map[string]Marca `json:"liberadas"`
}

type Marca struct {
	Por  *uuid.UUID `json:"por,omitempty"`
	Nome string     `json:"nome,omitempty"`
	Em   time.Time  `json:"em"`
}

type Evento struct {
	ID     int64           `json:"id"`
	Evento string          `json:"evento"`
	Campo  *string         `json:"campo,omitempty"`
	Antes  json.RawMessage `json:"antes,omitempty"`
	Depois json.RawMessage `json:"depois,omitempty"`
	Motivo *string         `json:"motivo,omitempty"`
	Por    *uuid.UUID      `json:"por,omitempty"`
	Nome   string          `json:"nome,omitempty"`
	Em     time.Time       `json:"ocorrido_em"`
}

// Uso: onde o item aparece, para decidir a descontinuação.
type Uso struct {
	Saldo            string   `json:"saldo"`
	PedidosVenda     int64    `json:"pedidos_venda_abertos"`
	PedidosCompra    int64    `json:"pedidos_compra_abertos"`
	OrdensFabricacao int64    `json:"ordens_fabricacao_abertas"`
	EstruturasPai    []string `json:"estruturas_pai"`
}

// Candidato a duplicado.
type Candidato struct {
	ItemCode int64   `json:"item_code"`
	Codigo   string  `json:"codigo"`
	Nome     string  `json:"nome"`
	NCM      string  `json:"ncm,omitempty"`
	Status   string  `json:"status"`
	Nota     float64 `json:"semelhanca"`
	Motivo   string  `json:"motivo"`
}

type Repository interface {
	Parametros(ctx context.Context) (Parametros, error)
	SalvarParametros(ctx context.Context, p Parametros, por *uuid.UUID) error
	Ajustes(ctx context.Context) (map[string]bool, error)
	SalvarAjuste(ctx context.Context, campo string, obrigatorio *bool, por *uuid.UUID) error
	Ficha(ctx context.Context, itemCode int64) (*Ficha, error)
	Situacao(ctx context.Context, itemCode int64) (*Situacao, error)
	// IniciarCadastro grava EM_CADASTRO para o item novo.
	IniciarCadastro(ctx context.Context, itemCode int64, por *uuid.UUID) error
	// Liberar grava a visão e, se pedido, o novo status — na mesma transação.
	Liberar(ctx context.Context, itemCode int64, visao string, novoStatus string, por *uuid.UUID) error
	EstornarLiberacao(ctx context.Context, itemCode int64, visao, motivo string, por *uuid.UUID) error
	MudarStatus(ctx context.Context, itemCode int64, atual string, m Mudanca) error
	Historico(ctx context.Context, itemCode int64, limite int) ([]Evento, error)
	Uso(ctx context.Context, itemCode int64) (*Uso, error)
	// EmCadastro: itens da empresa com status EM_CADASTRO.
	EmCadastro(ctx context.Context) ([]Situacao, error)
	// CandidatosParecidos: itens da empresa com alguma palavra em comum ou o
	// mesmo NCM (o domínio dá a nota).
	CandidatosParecidos(ctx context.Context, palavras []string, ncm string, excluir *int64) ([]Candidato, error)
}
