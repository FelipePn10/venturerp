// Package entity holds the Lot/Serial Mask register and its pure code generator.
package entity

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Part types (Tipo da partição da máscara).
const (
	PartCaracter    = "CARACTER"     // fixed text
	PartData        = "DATA"         // current date, formatted
	PartSeqNumerica = "SEQ_NUMERICA" // incrementing numeric sequence
	PartSeqCaracter = "SEQ_CARACTER" // incrementing alphabetic sequence
)

// MaxLotLength is the hard limit for a generated lot code.
const MaxLotLength = 20

func validPartType(t string) bool {
	switch t {
	case PartCaracter, PartData, PartSeqNumerica, PartSeqCaracter:
		return true
	}
	return false
}

// LotMask is the header resolving which mask applies (by customer/item/
// classification/application).
type LotMask struct {
	ID                 int64
	Application        string
	CustomerCode       *int64
	ItemCode           *int64
	ClassificationType string
	ClassificationCode *int64
	ZeroOnYearChange   bool
	IsActive           bool
	Description        string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	CreatedBy          uuid.UUID
	Parts              []LotMaskPart
}

// LotMaskPart is one ordered partition of the mask, carrying the mutable
// sequence state for the incremental types.
type LotMaskPart struct {
	ID               int64
	LotMaskID        int64
	Sequence         int
	PartType         string
	Value            string // fixed text or the sequence's initial value
	Size             int    // 0 = automatic
	DateFormat       string
	ZeroOnYearChange bool
	CurrentValue     string // last generated value (sequence state)
	LastYear         *int
}

func NewLotMask(application string, createdBy uuid.UUID) (*LotMask, error) {
	if application == "" {
		application = "GERAL"
	}
	return &LotMask{Application: application, IsActive: true, CreatedBy: createdBy}, nil
}

func (p *LotMaskPart) Validate() error {
	if !validPartType(p.PartType) {
		return errors.New("tipo de partição inválido")
	}
	if p.Sequence <= 0 {
		return errors.New("sequência da partição deve ser positiva")
	}
	if p.PartType == PartData && p.DateFormat == "" {
		p.DateFormat = "DDMMYYYY"
	}
	if p.Size < 0 {
		return errors.New("tamanho não pode ser negativo")
	}
	return nil
}

// PartUpdate carries the new sequence state to persist after a generation.
type PartUpdate struct {
	PartID     int64
	NewCurrent string
	NewYear    int
}

// GenerateResult is the produced lot code plus the sequence-state updates.
type GenerateResult struct {
	Code    string
	Updates []PartUpdate
}

// Generate builds a lot code from the parts (ordered by Sequence) at time `now`,
// returning the code and the sequence-state updates the caller must persist.
func Generate(parts []LotMaskPart, now time.Time) (GenerateResult, error) {
	ordered := make([]LotMaskPart, len(parts))
	copy(ordered, parts)
	// stable sort by sequence
	for i := 1; i < len(ordered); i++ {
		for j := i; j > 0 && ordered[j-1].Sequence > ordered[j].Sequence; j-- {
			ordered[j-1], ordered[j] = ordered[j], ordered[j-1]
		}
	}

	var sb strings.Builder
	var updates []PartUpdate
	year := now.Year()

	for _, p := range ordered {
		switch p.PartType {
		case PartCaracter:
			sb.WriteString(fitFixed(p.Value, p.Size))
		case PartData:
			sb.WriteString(formatDate(p.DateFormat, now))
		case PartSeqNumerica:
			out, cur := nextNumeric(p, year)
			sb.WriteString(out)
			updates = append(updates, PartUpdate{PartID: p.ID, NewCurrent: cur, NewYear: year})
		case PartSeqCaracter:
			out, cur := nextAlpha(p, year)
			sb.WriteString(out)
			updates = append(updates, PartUpdate{PartID: p.ID, NewCurrent: cur, NewYear: year})
		default:
			return GenerateResult{}, fmt.Errorf("tipo de partição inválido: %s", p.PartType)
		}
	}

	code := sb.String()
	if len(code) > MaxLotLength {
		return GenerateResult{}, fmt.Errorf("código de lote gerado excede %d caracteres: %q", MaxLotLength, code)
	}
	return GenerateResult{Code: code, Updates: updates}, nil
}

// fitFixed right-pads with spaces (or truncates) a fixed value to size; size 0
// keeps the value as-is.
func fitFixed(v string, size int) string {
	if size <= 0 {
		return v
	}
	if len(v) >= size {
		return v[:size]
	}
	return v + strings.Repeat(" ", size-len(v))
}

// nextNumeric computes the numeric sequence value to emit and the new state.
func nextNumeric(p LotMaskPart, year int) (out, newCurrent string) {
	n := atoiOr(p.Value, 0)
	if p.CurrentValue != "" {
		n = atoiOr(p.CurrentValue, 0) + 1
	}
	if p.ZeroOnYearChange && p.LastYear != nil && *p.LastYear != year {
		n = atoiOr(p.Value, 0) // restart at the initial value on year change
	}
	s := strconv.Itoa(n)
	if p.Size > 0 {
		s = leftPadZeros(s, p.Size)
	}
	return s, strconv.Itoa(n)
}

// nextAlpha computes the alphabetic sequence value to emit and the new state.
func nextAlpha(p LotMaskPart, year int) (out, newCurrent string) {
	s := strings.ToUpper(p.Value)
	if s == "" {
		s = "A"
	}
	if p.CurrentValue != "" {
		s = incAlpha(strings.ToUpper(p.CurrentValue))
	}
	if p.ZeroOnYearChange && p.LastYear != nil && *p.LastYear != year {
		s = strings.ToUpper(p.Value)
		if s == "" {
			s = "A"
		}
	}
	out = s
	if p.Size > 0 {
		out = fitFixed(s, p.Size)
	}
	return out, s
}

// incAlpha increments an uppercase A–Z string like an odometer: A→B, Z→AA, AZ→BA.
func incAlpha(s string) string {
	if s == "" {
		return "A"
	}
	r := []byte(s)
	i := len(r) - 1
	for i >= 0 {
		if r[i] < 'A' || r[i] > 'Z' {
			r[i] = 'A'
			break
		}
		if r[i] < 'Z' {
			r[i]++
			return string(r)
		}
		r[i] = 'A'
		i--
	}
	if i < 0 {
		return "A" + string(r)
	}
	return string(r)
}

// formatDate expands DD/MM/YYYY/YY/HH/MI/SS tokens against `now`.
func formatDate(format string, now time.Time) string {
	if format == "" {
		format = "DDMMYYYY"
	}
	rep := strings.NewReplacer(
		"YYYY", fmt.Sprintf("%04d", now.Year()),
		"YY", fmt.Sprintf("%02d", now.Year()%100),
		"MM", fmt.Sprintf("%02d", int(now.Month())),
		"DD", fmt.Sprintf("%02d", now.Day()),
		"HH", fmt.Sprintf("%02d", now.Hour()),
		"MI", fmt.Sprintf("%02d", now.Minute()),
		"SS", fmt.Sprintf("%02d", now.Second()),
	)
	return rep.Replace(format)
}

func leftPadZeros(s string, size int) string {
	if len(s) >= size {
		return s
	}
	return strings.Repeat("0", size-len(s)) + s
}

func atoiOr(s string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return def
	}
	return n
}
