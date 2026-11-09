// Package entity modela o catálogo de unidades de medida da empresa, os
// sinônimos (texto recebido → unidade) e as conversões padrão.
package entity

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
)

// Dimensão é a grandeza física: só se converte livremente (conversão padrão)
// dentro da mesma dimensão. KG→UN depende do item; M→MM não.
type Dimensao string

const (
	DimContagem    Dimensao = "CONTAGEM"
	DimComprimento Dimensao = "COMPRIMENTO"
	DimMassa       Dimensao = "MASSA"
	DimArea        Dimensao = "AREA"
	DimVolume      Dimensao = "VOLUME"
	DimTempo       Dimensao = "TEMPO"
	DimOutra       Dimensao = "OUTRA"
)

func (d Dimensao) Valida() bool {
	switch d {
	case DimContagem, DimComprimento, DimMassa, DimArea, DimVolume, DimTempo, DimOutra:
		return true
	}
	return false
}

// Unit é uma unidade do catálogo da empresa.
type Unit struct {
	ID          int64
	Code        string
	Description string
	Dimension   Dimensao
	FiscalCode  string // como sai na NF-e/SPED (até 6 caracteres)
	Decimals    int
	IsSystem    bool
	IsActive    bool
	CreatedBy   *uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewUnit valida uma unidade nova ou alterada.
func NewUnit(code, description string, dim Dimensao, fiscalCode string, decimals int) (*Unit, error) {
	code = NormalizarCodigo(code)
	if !types.CodigoDeUnidadeValido(code) {
		return nil, domainerr.NewValidation("o código da unidade tem de 1 a 10 letras ou números, sem espaço nem acento (ex.: BARRA, M2)")
	}
	description = strings.TrimSpace(description)
	if description == "" {
		return nil, domainerr.NewValidation("informe a descrição da unidade")
	}
	if len([]rune(description)) > 80 {
		return nil, domainerr.NewValidation("a descrição da unidade passa de 80 caracteres")
	}
	if dim == "" {
		dim = DimOutra
	}
	if !dim.Valida() {
		return nil, domainerr.NewValidation("dimensão inválida: use CONTAGEM, COMPRIMENTO, MASSA, AREA, VOLUME, TEMPO ou OUTRA")
	}
	fiscalCode = NormalizarCodigo(fiscalCode)
	if fiscalCode == "" {
		if len(code) > 6 {
			return nil, domainerr.NewValidation("o código tem mais de 6 caracteres: informe o código fiscal (como a unidade sai na NF-e, até 6)")
		}
		fiscalCode = code
	}
	if len(fiscalCode) > 6 {
		return nil, domainerr.NewValidation("o código fiscal da unidade aceita no máximo 6 caracteres (limite da NF-e e do SPED)")
	}
	if decimals < 0 || decimals > 6 {
		return nil, domainerr.NewValidation("as casas decimais da unidade vão de 0 a 6")
	}
	return &Unit{Code: code, Description: description, Dimension: dim, FiscalCode: fiscalCode, Decimals: decimals, IsActive: true}, nil
}

// Alias: texto recebido (NF-e, planilha) que significa uma unidade do catálogo.
type Alias struct {
	ID           int64
	Alias        string
	UnitCode     string
	SupplierCode *int64 // nulo = vale para todos os fornecedores
	SupplierName string
	IsSystem     bool
	CreatedBy    *uuid.UUID
	CreatedAt    time.Time
}

func NewAlias(alias, unitCode string, supplier *int64) (*Alias, error) {
	alias = NormalizarTexto(alias)
	unitCode = NormalizarCodigo(unitCode)
	if alias == "" || len(alias) > 10 {
		return nil, domainerr.NewValidation("o sinônimo tem de 1 a 10 caracteres (é o texto que vem na nota, ex.: UND, PÇ)")
	}
	if unitCode == "" {
		return nil, domainerr.NewValidation("informe a unidade do catálogo que o sinônimo representa")
	}
	if alias == unitCode {
		return nil, domainerr.NewValidation("o sinônimo é igual ao próprio código da unidade: não precisa cadastrar")
	}
	if supplier != nil && *supplier <= 0 {
		supplier = nil
	}
	return &Alias{Alias: alias, UnitCode: unitCode, SupplierCode: supplier}, nil
}

// StandardConversion: conversão padrão da empresa, entre unidades da MESMA
// dimensão — vale para todos os itens que não tenham a sua própria.
type StandardConversion struct {
	ID        int64
	FromUOM   string
	ToUOM     string
	FromQty   decimal.Decimal
	ToQty     decimal.Decimal
	Factor    decimal.Decimal
	IsSystem  bool
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
	FromDim   Dimensao
	ToDim     Dimensao
}

func NewStandardConversion(from, to string, fromQty, toQty decimal.Decimal, fromDim, toDim Dimensao) (*StandardConversion, error) {
	from, to = NormalizarCodigo(from), NormalizarCodigo(to)
	if from == "" || to == "" {
		return nil, domainerr.NewValidation("informe a unidade de origem e a de destino")
	}
	if from == to {
		return nil, domainerr.NewValidation("a unidade de origem e a de destino devem ser diferentes")
	}
	if !fromQty.IsPositive() || !toQty.IsPositive() {
		return nil, domainerr.NewValidation("as quantidades da conversão devem ser maiores que zero")
	}
	if fromDim != toDim || fromDim == DimOutra || fromDim == "" {
		return nil, domainerr.NewValidation("conversão padrão só entre unidades da mesma dimensão (comprimento com comprimento, massa com massa…). " +
			"Entre dimensões diferentes, como KG e UN, o fator depende do item: cadastre na conversão do item")
	}
	return &StandardConversion{FromUOM: from, ToUOM: to, FromQty: fromQty, ToQty: toQty,
		Factor: toQty.DivRound(fromQty, 10), IsActive: true, FromDim: fromDim, ToDim: toDim}, nil
}

// NormalizarCodigo: código do catálogo em maiúsculas, sem espaços.
func NormalizarCodigo(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

// NormalizarTexto põe o texto livre de uma nota na forma em que sinônimos são
// guardados e comparados: maiúsculas, sem acento, sem pontuação e sem espaço;
// "²" e "³" viram 2 e 3. "Pç." → "PC", "m²" → "M2", "Kg " → "KG".
func NormalizarTexto(s string) string {
	s = semAcento.Replace(strings.ToUpper(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 10 {
		out = out[:10]
	}
	return out
}

var semAcento = strings.NewReplacer(
	"²", "2", "³", "3", "º", "", "ª", "",
	"Á", "A", "À", "A", "Â", "A", "Ã", "A", "Ä", "A",
	"É", "E", "È", "E", "Ê", "E", "Ë", "E",
	"Í", "I", "Ì", "I", "Î", "I", "Ï", "I",
	"Ó", "O", "Ò", "O", "Ô", "O", "Õ", "O", "Ö", "O",
	"Ú", "U", "Ù", "U", "Û", "U", "Ü", "U",
	"Ç", "C", "Ñ", "N",
)
