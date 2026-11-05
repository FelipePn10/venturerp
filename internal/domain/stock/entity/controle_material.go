package entity

import "github.com/shopspring/decimal"

// Ação da empresa para o lote vencido.
const (
	LoteVencidoBloqueia = "BLOQUEAR"
	LoteVencidoAvisa    = "AVISAR"
)

// Regras de baixa do lote.
const (
	BaixaFEFO = "FEFO"
	BaixaFIFO = "FIFO"
)

// SituacaoLote é uma situação do cadastro da empresa e o que ela permite.
type SituacaoLote struct {
	Code            string `json:"code"`
	Description     string `json:"description"`
	AllowsIssue     bool   `json:"allows_issue"`
	AllowsTransfer  bool   `json:"allows_transfer"`
	CountsAvailable bool   `json:"counts_available"`
	IsActive        bool   `json:"is_active"`
}

// ParametrosEstoque são as regras de estoque da empresa.
type ParametrosEstoque struct {
	ExpiredLotAction string `json:"expired_lot_action"`
	// RetroactiveMonths: quantos meses antes do atual o lançamento pode ser datado.
	// Vazio não limita; 0 só aceita o mês corrente.
	RetroactiveMonths *int `json:"retroactive_months"`
	// ToleranceA/B/C: diferença aceita por classe ABC, em percentual. Valem para a
	// acuracidade e para o inventário que não traz tolerância própria.
	ToleranceA *decimal.Decimal `json:"tolerance_a"`
	ToleranceB *decimal.Decimal `json:"tolerance_b"`
	ToleranceC *decimal.Decimal `json:"tolerance_c"`
	// ObsoleteDays e ObsoleteProvisionPct: sem movimento há tantos dias, o item entra
	// na provisão por este percentual do valor.
	ObsoleteDays         *int             `json:"obsolete_days"`
	ObsoleteProvisionPct *decimal.Decimal `json:"obsolete_provision_pct"`
}

// ToleranciaDaClasse devolve a tolerância da classe ABC; item sem classe usa a C.
func (p ParametrosEstoque) ToleranciaDaClasse(classe string) *decimal.Decimal {
	switch classe {
	case "A":
		return p.ToleranceA
	case "B":
		return p.ToleranceB
	}
	return p.ToleranceC
}

// TemToleranciaPorClasse diz se a empresa cadastrou tolerância em alguma classe.
func (p ParametrosEstoque) TemToleranciaPorClasse() bool {
	return p.ToleranceA != nil || p.ToleranceB != nil || p.ToleranceC != nil
}

// MotivoDivergencia é um motivo de diferença de contagem ou de acerto de saldo.
type MotivoDivergencia struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

// ControleDoItem diz como o estoque do item é controlado.
type ControleDoItem struct {
	ItemCode        int64  `json:"item_code"`
	ControlsLot     bool   `json:"controls_lot"`
	ControlsAddress bool   `json:"controls_address"`
	LotIssueRule    string `json:"lot_issue_rule"`
}

// MudancaSituacaoLote troca a situação e o grau de um lote.
type MudancaSituacaoLote struct {
	ItemCode int64
	Mask     string
	Lot      string
	Status   *string // nil = liberado
	Grade    *string
	Reason   string
}

// ImplantacaoLote dá lote a um saldo que entrou sem lote.
type ImplantacaoLote struct {
	ItemCode    int64
	Mask        string
	WarehouseID int64
	Address     string
	Lot         string
	Quantity    decimal.Decimal
}
