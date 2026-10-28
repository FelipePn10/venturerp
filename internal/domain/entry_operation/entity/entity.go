package entity

import (
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ─── State Group ───────────────────────────────────────────────────────────────

type StateGroup struct {
	ID          int64
	Code        int64
	Description string
	IsActive    bool
	CreatedAt   time.Time
	CreatedBy   uuid.UUID
	UFs         []string
}

func NewStateGroup(code int64, description string, createdBy uuid.UUID) (*StateGroup, error) {
	if code == 0 {
		return nil, domainerr.NewValidation("informe o código")
	}
	if description == "" {
		return nil, domainerr.NewValidation("informe a descrição")
	}
	return &StateGroup{
		Code:        code,
		Description: description,
		IsActive:    true,
		CreatedAt:   time.Now(),
		CreatedBy:   createdBy,
	}, nil
}

// ─── Entry Operation Type ──────────────────────────────────────────────────────

type EntryOperationType struct {
	ID                 int64
	Code               int64
	Description        string
	InvoiceTypeCode    *int64
	NatureOperation    string
	ClassificationType *string
	ClassificationCode *string
	StateGroupCode     *int64
	SupplierTypeCode   *int64
	// Comportamento da nota (equivalente à TES): movimenta estoque, gera
	// contas a pagar e quais impostos viram crédito em vez de custo.
	MovimentaEstoque bool
	GeraFinanceiro   bool
	CreditaICMS      bool
	CreditaIPI       bool
	CreditaPISCOFINS bool
	IsActive         bool
	CreatedAt        time.Time
	CreatedBy        uuid.UUID
}

func NewEntryOperationType(code int64, description, natureOperation string, createdBy uuid.UUID) (*EntryOperationType, error) {
	if code == 0 {
		return nil, domainerr.NewValidation("informe o código")
	}
	if description == "" {
		return nil, domainerr.NewValidation("informe a descrição")
	}
	natureOperation, err := NormalizarNatureza(natureOperation)
	if err != nil {
		return nil, err
	}
	return &EntryOperationType{
		Code:             code,
		Description:      description,
		NatureOperation:  natureOperation,
		MovimentaEstoque: true,
		GeraFinanceiro:   true,
		CreditaICMS:      true,
		CreditaIPI:       true,
		CreditaPISCOFINS: true,
		IsActive:         true,
		CreatedAt:        time.Now(),
		CreatedBy:        createdBy,
	}, nil
}

// ValidateUF applies the UF × Grupo de Estado rule based on the nature's first
// digit. ufInGroup tells whether the enterprise UF belongs to the operation's
// state group.
func (o *EntryOperationType) ValidateUF(enterpriseUF string, ufInGroup bool) error {
	if o.NatureOperation == "" {
		return nil
	}
	switch o.NatureOperation[0] {
	case '1': // dentro do estado → UF deve pertencer ao grupo
		if !ufInGroup {
			return domainerr.NewValidation(fmt.Sprintf("natureza 1 (dentro do estado): a UF %s da empresa deve pertencer ao grupo de estado da operação", enterpriseUF))
		}
	case '2': // fora do estado → UF NÃO deve pertencer ao grupo
		if ufInGroup {
			return domainerr.NewValidation(fmt.Sprintf("natureza 2 (fora do estado): a UF %s da empresa não deve pertencer ao grupo de estado da operação", enterpriseUF))
		}
	case '3': // fora do país → operação estrangeira (sem validação de grupo)
	}
	return nil
}

// NormalizarNatureza: a natureza é o CFOP de entrada da operação, com 4 dígitos
// e iniciado por 1 (dentro do estado), 2 (fora do estado) ou 3 (exterior). Na
// nota, o primeiro dígito acompanha a UF do fornecedor.
func NormalizarNatureza(n string) (string, error) {
	var b strings.Builder
	for _, r := range n {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	d := b.String()
	if d == "" {
		return "", domainerr.NewValidation("informe a natureza da operação (o CFOP de entrada, como 1101)")
	}
	if len(d) != 4 || (d[0] != '1' && d[0] != '2' && d[0] != '3') {
		return "", domainerr.NewValidation(fmt.Sprintf("natureza %q inválida: é o CFOP de entrada com 4 dígitos, iniciado por 1 (dentro do estado), 2 (fora do estado) ou 3 (exterior)", strings.TrimSpace(n)))
	}
	return d, nil
}

// ConferirCreditos: o crédito marcado tem de caber na natureza. Uso e consumo
// (x556, x407) não dá crédito de ICMS nem de IPI; bem do ativo (x551, x406) não
// dá crédito de IPI, e o ICMS dele é apropriado em parcelas (CIAP), fora da
// nota. Marcado errado, o imposto sairia do custo e iria para a apuração.
func (o *EntryOperationType) ConferirCreditos() error {
	if len(o.NatureOperation) != 4 {
		return nil
	}
	switch o.NatureOperation[1:] {
	case "556", "407":
		if o.CreditaICMS || o.CreditaIPI {
			return domainerr.NewValidation(fmt.Sprintf("a natureza %s é de uso e consumo: não credita ICMS nem IPI (desmarque os dois)", o.NatureOperation))
		}
	case "551", "406":
		if o.CreditaIPI {
			return domainerr.NewValidation(fmt.Sprintf("a natureza %s é de ativo imobilizado: não credita IPI (desmarque)", o.NatureOperation))
		}
		if o.CreditaICMS {
			return domainerr.NewValidation(fmt.Sprintf("a natureza %s é de ativo imobilizado: o ICMS é apropriado em parcelas (CIAP), não de uma vez na nota — desmarque o crédito de ICMS", o.NatureOperation))
		}
	}
	return nil
}

// ValeParaFornecedor: com grupo de estados, a operação só vale para fornecedor
// de UF do grupo (ufNoGrupo diz se a UF dele está lá).
func (o *EntryOperationType) ValeParaFornecedor(ufFornecedor string, ufNoGrupo bool) error {
	if o.StateGroupCode == nil || strings.TrimSpace(ufFornecedor) == "" {
		return nil
	}
	if !ufNoGrupo {
		return domainerr.NewValidation(fmt.Sprintf("o tipo de operação %d (%s) vale só para fornecedor das UFs do grupo de estados %d; o fornecedor é de %s",
			o.Code, o.Description, *o.StateGroupCode, strings.ToUpper(strings.TrimSpace(ufFornecedor))))
	}
	return nil
}
