package valueobject

import (
	"encoding/json"
	"errors"
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
		return "", errors.New("codigo do item deve ter de 1 a 60 caracteres alfanumericos e pode conter ponto, hifen, barra ou sublinhado")
	}
	return BusinessCode(normalized), nil
}

func (c BusinessCode) IsValid() bool {
	return businessCodePattern.MatchString(string(c))
}

func NewItemCode(code int64) (ItemCode, error) {
	if code <= 0 {
		return 0, errors.New("o código do item deve ser maior que zero")
	}
	return ItemCode(code), nil
}

func (c ItemCode) IsValid() bool {
	return c > 0
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
		return nil, errors.New("invalid dimensions")
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
		return Weight{}, errors.New("invalid weight")
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
		return Attribute{}, errors.New("invalid attribute")
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
		return nil, errors.New("configuração de contagem cíclica inválida")
	}

	return c, nil
}

func (c CyclicalCountConfig) IsValid() bool {
	return c.DaysInterval > 0
}

//
// ReorderPoint
//

type ReorderPoint struct {
	TR int16
	CM int16
	CR int
	ES int16
}

func NewReorderPoint(tr, cm int16, cr int, es int16) (*ReorderPoint, error) {
	r := &ReorderPoint{
		TR: tr,
		CM: cm,
		CR: cr,
		ES: es,
	}

	if !r.IsValid() {
		return nil, errors.New("ponto de reposição inválido")
	}

	return r, nil
}

func (r ReorderPoint) IsValid() bool {
	return r.CR > 0
}

func (r ReorderPoint) Calculate() (int, error) {
	if r.CR == 0 {
		return 0, errors.New("o CR não pode ser zero")
	}

	result := (int(r.TR) * int(r.CM) / r.CR) + int(r.ES)
	return result, nil
}
