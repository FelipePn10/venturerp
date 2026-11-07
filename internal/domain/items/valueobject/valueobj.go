package valueobject

import (
	"encoding/json"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/shopspring/decimal"
	"regexp"
	"strings"
)

//
// ItemCode
//

type ItemCode int64

var businessCodePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9._/-]{0,59}$`)

type BusinessCode string

func NewBusinessCode(code string) (BusinessCode, error) {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	if !businessCodePattern.MatchString(normalized) {
		return "", domainerr.NewValidation(fmt.Sprintf("código de item inválido (%q): use de 1 a 60 caracteres — letras, números, ponto, hífen, barra ou sublinhado, sem espaço. Para procurar pela descrição, use a busca por descrição", strings.TrimSpace(code)))
	}
	return BusinessCode(normalized), nil
}

func (c BusinessCode) IsValid() bool {
	return businessCodePattern.MatchString(string(c))
}

func NewItemCode(code int64) (ItemCode, error) {
	if code <= 0 {
		return 0, domainerr.NewValidation("o código do item deve ser maior que zero")
	}
	return ItemCode(code), nil
}

func (c ItemCode) IsValid() bool {
	return c > 0
}

//
// GTIN
//

// GTINValido confere o tamanho (8, 12, 13 ou 14 dígitos) e o dígito verificador
// GS1. Com o dígito errado a SEFAZ rejeita a nota, então o erro precisa aparecer
// no cadastro, não na emissão.
func GTINValido(gtin string) bool {
	switch len(gtin) {
	case 8, 12, 13, 14:
	default:
		return false
	}
	soma := 0
	for i := len(gtin) - 2; i >= 0; i-- {
		c := gtin[i]
		if c < '0' || c > '9' {
			return false
		}
		peso := 3
		if (len(gtin)-2-i)%2 == 1 {
			peso = 1
		}
		soma += int(c-'0') * peso
	}
	ultimo := gtin[len(gtin)-1]
	if ultimo < '0' || ultimo > '9' {
		return false
	}
	return (10-soma%10)%10 == int(ultimo-'0')
}

//
// Dimensions
//

type Dimensions struct {
	Length int
	Width  int
	Height float64
}

func NewDimensions(length, width int, height float64) (*Dimensions, error) {
	d := &Dimensions{
		Length: length,
		Width:  width,
		Height: height,
	}

	if !d.IsValid() {
		return nil, domainerr.NewValidation("dimensões inválidas: informe comprimento, largura e altura maiores que zero")
	}

	return d, nil
}

func (d Dimensions) IsValid() bool {
	return d.Length > 0 && d.Width > 0 && d.Height > 0
}

func (d Dimensions) Volume() float64 {
	return float64(d.Length) * float64(d.Width) * d.Height
}

//
// Weight
//

type Weight struct {
	Gross float64 `json:"gross"`
	Net   float64 `json:"net"`
	Unit  string  `json:"unit"`
}

func NewWeight(gross, net float64, unit string) (Weight, error) {
	w := Weight{
		Gross: gross,
		Net:   net,
		Unit:  unit,
	}

	if !w.IsValid() {
		return Weight{}, domainerr.NewValidation("peso inválido: informe o peso líquido e o bruto maiores que zero, com o bruto não menor que o líquido")
	}

	return w, nil
}

func (w Weight) IsValid() bool {
	if w.Unit == "" {
		return false
	}
	if w.Net < 0 {
		return false
	}
	if w.Gross < w.Net {
		return false
	}
	return true
}

//
// Attribute
//

type Attribute struct {
	Name  string
	Value string
}

func NewAttribute(name, value string) (Attribute, error) {
	a := Attribute{
		Name:  name,
		Value: value,
	}

	if !a.IsValid() {
		return Attribute{}, domainerr.NewValidation("atributo do PDM inválido: informe nome e valor")
	}

	return a, nil
}

func (a Attribute) IsValid() bool {
	return a.Name != "" && a.Value != ""
}

//
// CyclicalCountConfig
//

type CyclicalCountConfig struct {
	DaysInterval int `json:"days_interval"`
}

// UnmarshalJSON accepts the public snake_case contract and the temporary
// legacy spelling used by already installed Desktop versions. Unknown keys
// intentionally leave the value invalid so domain validation rejects them.
func (c *CyclicalCountConfig) UnmarshalJSON(data []byte) error {
	var wire struct {
		DaysInterval       *int `json:"days_interval"`
		LegacyDaysInterval *int `json:"DaysInterval"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	switch {
	case wire.DaysInterval != nil:
		c.DaysInterval = *wire.DaysInterval
	case wire.LegacyDaysInterval != nil:
		c.DaysInterval = *wire.LegacyDaysInterval
	default:
		c.DaysInterval = 0
	}
	return nil
}

func NewCyclicalCountConfig(days int) (*CyclicalCountConfig, error) {
	c := &CyclicalCountConfig{
		DaysInterval: days,
	}

	if !c.IsValid() {
		return nil, domainerr.NewValidation("configuração de contagem cíclica inválida")
	}

	return c, nil
}

func (c CyclicalCountConfig) IsValid() bool {
	return c.DaysInterval > 0
}

//
// ReorderPoint
//

// TR (tempo de reposição) e CR (ciclo de revisão) são dias; CM (consumo
// mensal) e ES (estoque de segurança) estão na unidade de estoque e aceitam
// fração — item em KG ou M consome 12,5 por mês.
type ReorderPoint struct {
	TR int
	CM decimal.Decimal
	CR int
	ES decimal.Decimal
	// Modo DIA_TRABALHADO: CM é o consumo por dia trabalhado (0 = apurar do
	// histórico), CR é 1 e TR é o prazo de reposição em dias úteis. A reposição
	// troca TR pelos dias que a empresa trabalha no prazo, pelo calendário
	// industrial (sábado trabalhado conta, feriado não). Sem modo, é o formato
	// antigo: CM consumido em CR dias.
	Modo string
}

// ModoPorDiaTrabalhado é o ponto de pedido pelo consumo por dia trabalhado.
const ModoPorDiaTrabalhado = "DIA_TRABALHADO"

// reorderPointJSON grava números no JSONB (o decimal sairia como texto).
type reorderPointJSON struct {
	TR   int             `json:"TR"`
	CM   json.RawMessage `json:"CM"`
	CR   int             `json:"CR"`
	ES   json.RawMessage `json:"ES"`
	Modo string          `json:"modo,omitempty"`
}

func (r ReorderPoint) MarshalJSON() ([]byte, error) {
	return json.Marshal(reorderPointJSON{TR: r.TR, CM: json.RawMessage(r.CM.String()), CR: r.CR, ES: json.RawMessage(r.ES.String()), Modo: r.Modo})
}

// UnmarshalJSON aceita número ou texto, e o formato antigo (inteiros).
func (r *ReorderPoint) UnmarshalJSON(b []byte) error {
	var raw struct {
		TR, CR int
		CM, ES decimal.NullDecimal
		Modo   string `json:"modo"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	r.TR, r.CR, r.Modo = raw.TR, raw.CR, raw.Modo
	if r.Modo != "" && r.Modo != ModoPorDiaTrabalhado {
		return domainerr.NewValidation("modo do ponto de pedido inválido")
	}
	r.CM, r.ES = raw.CM.Decimal, raw.ES.Decimal
	return nil
}

func NewReorderPoint(tr int, cm decimal.Decimal, cr int, es decimal.Decimal) (*ReorderPoint, error) {
	r := &ReorderPoint{TR: tr, CM: cm, CR: cr, ES: es}
	if !r.IsValid() {
		return nil, domainerr.NewValidation("ponto de reposição inválido")
	}
	return r, nil
}

func (r ReorderPoint) IsValid() bool {
	return r.CR > 0 && r.TR >= 0 && !r.CM.IsNegative() && !r.ES.IsNegative()
}

// Calculate é o ponto de pedido TR × CM / CR + ES, arredondado para cima (a
// mesma conta da sugestão de reposição do estoque; antes o MRP truncava a
// divisão inteira e as duas telas davam números diferentes).
func (r ReorderPoint) Calculate() (decimal.Decimal, error) {
	if r.CR == 0 {
		return decimal.Zero, domainerr.NewValidation("o CR não pode ser zero")
	}
	return PontoDePedido(r.TR, r.CM, r.CR, r.ES), nil
}

// PontoDePedido é TR × CM / CR + ES arredondado para cima.
func PontoDePedido(tr int, cm decimal.Decimal, cr int, es decimal.Decimal) decimal.Decimal {
	return decimal.NewFromInt(int64(tr)).Mul(cm).Div(decimal.NewFromInt(int64(cr))).Add(es).Ceil()
}
