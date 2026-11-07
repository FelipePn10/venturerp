package governanca

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

// Embalagem é um nível de embalagem do item: a caixa tem 10 unidades, o palete tem
// 40 caixas. Quantidade é sempre o total de unidades do item naquele nível.
type Embalagem struct {
	GTIN       string          `json:"gtin"`
	Unidade    string          `json:"uom"`
	Quantidade decimal.Decimal `json:"package_quantity"`
	Descricao  *string         `json:"description,omitempty"`
	// Contem e ContemQuantidade dizem do que o nível é feito: 40 de CX. Vazios, a
	// embalagem é informada direto em unidades do item.
	Contem           *string          `json:"parent_uom,omitempty"`
	ContemQuantidade *decimal.Decimal `json:"parent_quantity,omitempty"`
}

// Limites das colunas de item_gtins: passar deles seria erro de banco, não aviso.
const (
	maximoUnidadeEmbalagem   = 20
	maximoDescricaoEmbalagem = 120
)

var limiteQuantidadeEmbalagem = decimal.New(1, 12)

// ConferirEmbalagens valida a lista inteira antes de gravar e calcula o total de
// unidades de cada nível. unidadeBase é a unidade de estoque do item.
func ConferirEmbalagens(lista []Embalagem, unidadeBase string) error {
	gtins, porUnidade := map[string]bool{}, map[string]*Embalagem{}
	unidadeBase = strings.ToUpper(strings.TrimSpace(unidadeBase))
	for i := range lista {
		e := &lista[i]
		e.GTIN = strings.TrimSpace(e.GTIN)
		e.Unidade = strings.ToUpper(strings.TrimSpace(e.Unidade))
		if e.Contem != nil {
			c := strings.ToUpper(strings.TrimSpace(*e.Contem))
			e.Contem = &c
			if c == "" {
				e.Contem, e.ContemQuantidade = nil, nil
			}
		}
		if e.Contem == nil {
			e.ContemQuantidade = nil
		}
		if e.Descricao != nil {
			d := strings.TrimSpace(*e.Descricao)
			e.Descricao = &d
			if d == "" {
				e.Descricao = nil
			} else if utf8.RuneCountInString(d) > maximoDescricaoEmbalagem {
				return domainerr.NewValidation("a descrição da embalagem " + e.Unidade + " passa de 120 caracteres")
			}
		}
		if e.GTIN != "" && !valueobject.GTINValido(e.GTIN) {
			return domainerr.NewValidation("o GTIN " + e.GTIN + " é inválido: confira os dígitos e o verificador")
		}
		if e.Unidade == "" {
			return domainerr.NewValidation("informe a unidade da embalagem")
		}
		if utf8.RuneCountInString(e.Unidade) > maximoUnidadeEmbalagem || (e.Contem != nil && utf8.RuneCountInString(*e.Contem) > maximoUnidadeEmbalagem) {
			return domainerr.NewValidation("a unidade da embalagem tem no máximo 20 caracteres")
		}
		if e.Unidade == unidadeBase {
			if e.GTIN == "" {
				return domainerr.NewValidation("a unidade " + e.Unidade + " é a do próprio item: informe o GTIN ou tire a linha")
			}
			if e.Contem != nil {
				return domainerr.NewValidation("a unidade " + e.Unidade + " é a do próprio item: ela não contém outra embalagem")
			}
			if !e.Quantidade.Equal(decimal.NewFromInt(1)) {
				return domainerr.NewValidation("a unidade " + e.Unidade + " é a do próprio item: a quantidade é 1")
			}
		}
		if e.Contem == nil && e.GTIN == "" {
			return domainerr.NewValidation("a embalagem " + e.Unidade + " precisa de GTIN ou de dizer o que contém")
		}
		if e.Contem == nil && !e.Quantidade.IsPositive() {
			return domainerr.NewValidation("a quantidade da embalagem deve ser maior que zero")
		}
		if e.Contem != nil && (e.ContemQuantidade == nil || !e.ContemQuantidade.IsPositive()) {
			return domainerr.NewValidation("informe quantas " + *e.Contem + " cabem em " + e.Unidade)
		}
		if e.Contem != nil && e.ContemQuantidade.GreaterThanOrEqual(limiteQuantidadeEmbalagem) {
			return domainerr.NewValidation("a quantidade da embalagem " + e.Unidade + " passa do limite de 999.999.999.999")
		}
		if e.GTIN != "" && gtins[e.GTIN] {
			return domainerr.NewValidation("o GTIN " + e.GTIN + " está repetido")
		}
		if porUnidade[e.Unidade] != nil {
			return domainerr.NewValidation("a unidade " + e.Unidade + " está repetida: uma linha por embalagem")
		}
		gtins[e.GTIN], porUnidade[e.Unidade] = e.GTIN != "", e
	}
	for i := range lista {
		q, err := totalDaEmbalagem(&lista[i], porUnidade, unidadeBase, map[string]bool{})
		if err != nil {
			return err
		}
		if q.GreaterThanOrEqual(limiteQuantidadeEmbalagem) {
			return domainerr.NewValidation("a quantidade da embalagem " + lista[i].Unidade + " passa do limite de 999.999.999.999")
		}
		lista[i].Quantidade = q
	}
	return nil
}

func totalDaEmbalagem(e *Embalagem, porUnidade map[string]*Embalagem, base string, caminho map[string]bool) (decimal.Decimal, error) {
	if e.Contem == nil {
		return e.Quantidade, nil
	}
	if caminho[e.Unidade] {
		return decimal.Zero, domainerr.NewValidation("a embalagem " + e.Unidade + " não pode conter a si mesma")
	}
	caminho[e.Unidade] = true
	if *e.Contem == base {
		return *e.ContemQuantidade, nil
	}
	nivel := porUnidade[*e.Contem]
	if nivel == nil {
		return decimal.Zero, domainerr.NewValidation("a embalagem " + e.Unidade + " contém " + *e.Contem + ", que não está cadastrada neste item")
	}
	q, err := totalDaEmbalagem(nivel, porUnidade, base, caminho)
	if err != nil {
		return decimal.Zero, err
	}
	return e.ContemQuantidade.Mul(q), nil
}

// AtributoLivre é uma característica do item fora do padrão do PDM.
type AtributoLivre struct {
	Nome  string `json:"name"`
	Valor string `json:"value"`
}

// ResumoDeAtributos é o texto que torna o item encontrável pelos atributos livres.
type ResumoDeAtributos struct {
	Item  string `json:"item"`
	Texto string `json:"text"`
	// Apelidos diz o que cada código é, para a busca mostrar por que achou o item.
	Apelidos []Apelido `json:"aliases"`
}

// CuidadoDoItem é o que quem recebe, fabrica, separa ou expede o item precisa
// ler antes de mexer nele: risco do produto perigoso e como armazenar.
type CuidadoDoItem struct {
	Item        string  `json:"item"`
	Nome        string  `json:"name"`
	Perigoso    bool    `json:"hazardous"`
	ONU         *string `json:"un_number,omitempty"`
	Classe      *string `json:"hazard_class,omitempty"`
	Armazenagem *string `json:"storage_conditions,omitempty"`
}

// Apelido é um código pelo qual o item também é achado.
type Apelido struct {
	Tipo  string `json:"kind"`
	Valor string `json:"value"`
}

// MaximoDeAtributosLivres evita que o livre vire um segundo cadastro do item.
const MaximoDeAtributosLivres = 30

// ConferirAtributosLivres normaliza e valida a lista inteira antes de gravar.
func ConferirAtributosLivres(lista []AtributoLivre) error {
	if len(lista) > MaximoDeAtributosLivres {
		return domainerr.NewValidation("use no máximo 30 atributos livres por item")
	}
	vistos := map[string]bool{}
	for i := range lista {
		a := &lista[i]
		a.Nome = strings.ToUpper(strings.Join(strings.Fields(a.Nome), " "))
		a.Valor = strings.Join(strings.Fields(a.Valor), " ")
		if a.Nome == "" || a.Valor == "" {
			return domainerr.NewValidation("informe o nome e o valor de cada atributo livre")
		}
		if utf8.RuneCountInString(a.Nome) > 60 || utf8.RuneCountInString(a.Valor) > 200 {
			return domainerr.NewValidation("o atributo " + a.Nome + " passa do tamanho: 60 no nome e 200 no valor")
		}
		if vistos[a.Nome] {
			return domainerr.NewValidation("o atributo " + a.Nome + " está repetido")
		}
		vistos[a.Nome] = true
	}
	return nil
}

// Revisao é uma versão do item com vigência.
type Revisao struct {
	Revisao  string     `json:"revision"`
	De       time.Time  `json:"valid_from"`
	Ate      *time.Time `json:"valid_until,omitempty"`
	Motivo   string     `json:"reason"`
	Por      *uuid.UUID `json:"created_by,omitempty"`
	CriadaEm time.Time  `json:"created_at"`
}

// NovaRevisao é o pedido de uma revisão: a anterior termina na véspera do início.
type NovaRevisao struct {
	Revisao string
	De      time.Time
	Motivo  string
}

// Conferir valida contra as revisões que o item já tem.
func (n *NovaRevisao) Conferir(existentes []Revisao) error {
	n.Revisao = strings.ToUpper(strings.TrimSpace(n.Revisao))
	n.Motivo = strings.TrimSpace(n.Motivo)
	if n.Revisao == "" || len(n.Revisao) > 20 {
		return domainerr.NewValidation("informe a revisão, com até 20 caracteres")
	}
	if n.Motivo == "" {
		return domainerr.NewValidation("informe o motivo da revisão")
	}
	if n.De.IsZero() {
		return domainerr.NewValidation("informe o início da vigência")
	}
	for _, r := range existentes {
		if r.Revisao == n.Revisao {
			return domainerr.NewValidation("o item já tem a revisão " + n.Revisao)
		}
		if !n.De.After(r.De) {
			return domainerr.NewValidation("a vigência começa depois da revisão " + r.Revisao)
		}
	}
	return nil
}

// RevisaoVigente devolve a revisão que vale na data.
func RevisaoVigente(lista []Revisao, data time.Time) *Revisao {
	var vale *Revisao
	for i := range lista {
		r := &lista[i]
		if r.De.After(data) || (r.Ate != nil && r.Ate.Before(data)) {
			continue
		}
		if vale == nil || r.De.After(vale.De) {
			vale = r
		}
	}
	return vale
}

// SaldoDuasUnidades é o saldo do item na unidade do estoque e na segunda.
type SaldoDuasUnidades struct {
	Unidade        string  `json:"uom"`
	Saldo          string  `json:"quantity"`
	SegundaUnidade *string `json:"secondary_uom,omitempty"`
	SegundoSaldo   string  `json:"secondary_quantity"`
	// Detalhes abre o saldo por almoxarifado e lote: a chapa do lote A pesa
	// diferente da do lote B, e é esse peso que o corte precisa saber.
	Detalhes []SaldoPorLote `json:"details"`
}

// SaldoPorLote é o saldo do item num almoxarifado e lote, nas duas unidades.
type SaldoPorLote struct {
	AlmoxarifadoID int64   `json:"warehouse_id"`
	Almoxarifado   string  `json:"warehouse"`
	Lote           *string `json:"lot,omitempty"`
	Saldo          string  `json:"quantity"`
	SegundoSaldo   string  `json:"secondary_quantity"`
}

// RepositorioDoCadastro grava embalagens e revisões, da empresa da sessão.
type RepositorioDoCadastro interface {
	Embalagens(ctx context.Context, item int64) ([]Embalagem, error)
	SalvarEmbalagens(ctx context.Context, item int64, lista []Embalagem) error
	Revisoes(ctx context.Context, item int64) ([]Revisao, error)
	CriarRevisao(ctx context.Context, item int64, n NovaRevisao, por *uuid.UUID) error
	SaldoDuasUnidades(ctx context.Context, item int64) (*SaldoDuasUnidades, error)
	AtributosLivres(ctx context.Context, item int64) ([]AtributoLivre, error)
	SalvarAtributosLivres(ctx context.Context, item int64, lista []AtributoLivre) error
	NomesDeAtributosLivres(ctx context.Context) ([]string, error)
	ResumoDeAtributosLivres(ctx context.Context) ([]ResumoDeAtributos, error)
	CuidadosDosItens(ctx context.Context) ([]CuidadoDoItem, error)
}
