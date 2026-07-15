package valueobject

import (
	"errors"
)

//
// ItemCode
//

type ItemCode int64

func NewItemCode(code int64) (ItemCode, error) {
	if code <= 0 {
		return 0, errors.New("item code must be greater than zero")
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
	Height int
}

func NewDimensions(length, width, height int) (*Dimensions, error) {
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

func (d Dimensions) Volume() int {
	return d.Length * d.Width * d.Height
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
	DaysInterval int
}

func NewCyclicalCountConfig(days int) (*CyclicalCountConfig, error) {
	c := &CyclicalCountConfig{
		DaysInterval: days,
	}

	if !c.IsValid() {
		return nil, errors.New("invalid cyclical count config")
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
		return nil, errors.New("invalid reorder point")
	}

	return r, nil
}

func (r ReorderPoint) IsValid() bool {
	return r.CR > 0
}

func (r ReorderPoint) Calculate() (int, error) {
	if r.CR == 0 {
		return 0, errors.New("CR cannot be zero")
	}

	result := (int(r.TR) * int(r.CM) / r.CR) + int(r.ES)
	return result, nil
}
