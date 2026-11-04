package entity

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/google/uuid"
)

// ItemStandardCost é o custo-padrão GRAVADO do item.
//
// ⚠️ `MaterialCost` é o material CHEIO: o material próprio mais o custo total dos
// componentes de níveis inferiores. `LaborCost` é só a mão de obra DIRETA —
// preparação, máquina e serviço de terceiro têm colunas próprias desde a migração
// 000373. `TotalCost` é gerado pelo banco somando os seis componentes (migração
// 000374); `OwnLevelCost` e `LowerLevelCost` são um CORTE dos mesmos valores e
// ficam de fora da soma.
type ItemStandardCost struct {
	ID              int64
	ItemCode        int64
	Mask            string
	MaterialCost    decimal.Decimal
	SetupCost       decimal.Decimal
	MachineCost     decimal.Decimal
	LaborCost       decimal.Decimal
	SubcontractCost decimal.Decimal
	OverheadCost    decimal.Decimal
	OwnLevelCost    decimal.Decimal
	LowerLevelCost  decimal.Decimal
	LotSize         decimal.Decimal
	TotalCost       decimal.Decimal
	Currency        string
	CalculatedAt    time.Time
	CalculatedBy    uuid.UUID
}

type WorkCenterCost struct {
	ID           int64
	WorkCenterID int64
	CostPerHour  decimal.Decimal // legacy blended rate (kept as machine-rate fallback)
	// Enterprise+ split: machine occupancy rate vs. direct-labor rate per hour.
	MachineCostPerHour decimal.Decimal
	LaborCostPerHour   decimal.Decimal
	Currency           string
	UpdatedAt          time.Time
	UpdatedBy          uuid.UUID
}

// MachineRate returns the effective machine hourly rate (falls back to the blended rate).
func (w *WorkCenterCost) MachineRate() decimal.Decimal {
	if w.MachineCostPerHour.IsPositive() {
		return w.MachineCostPerHour
	}
	return w.CostPerHour
}

// LaborRate returns the effective labor hourly rate.
func (w *WorkCenterCost) LaborRate() decimal.Decimal { return w.LaborCostPerHour }

type ItemPurchaseCost struct {
	ID        int64
	ItemCode  int64
	UnitCost  decimal.Decimal
	Currency  string
	UpdatedAt time.Time
	UpdatedBy uuid.UUID
}

type CostRollupLogEntry struct {
	ID           int64
	ItemCode     int64
	Mask         string
	BOMLevel     int
	MaterialCost decimal.Decimal
	LaborCost    decimal.Decimal
	OverheadCost decimal.Decimal
	// Componentes acrescentados na migração 000373: é o que permite a tela mostrar
	// a árvore de custo aberta em vez de dois números por nível.
	SetupCost       decimal.Decimal
	MachineCost     decimal.Decimal
	SubcontractCost decimal.Decimal
	LowerLevelCost  decimal.Decimal
	Quantity        decimal.Decimal
	ParentCode      *int64
	RunAt           time.Time
}

// Total do nó do log: componentes próprios mais o que veio de baixo.
func (e CostRollupLogEntry) Total() decimal.Decimal {
	return e.MaterialCost.Add(e.SetupCost).Add(e.MachineCost).Add(e.LaborCost).
		Add(e.SubcontractCost).Add(e.OverheadCost).Add(e.LowerLevelCost)
}

type RollupResult struct {
	ItemCode     int64
	Mask         string
	MaterialCost decimal.Decimal
	LaborCost    decimal.Decimal
	OverheadCost decimal.Decimal
	TotalCost    decimal.Decimal
	Detail       []CostRollupLogEntry
}

// ─── componentes de custo e esquema de rateio ────────────────────────────────

// ComponentesDeCusto é o custo unitário aberto por origem, no lugar dos dois
// números que existiam ("material" e "operação").
//
// Cada componente se ataca de um jeito diferente, e é por isso que eles ficam
// separados: setup alto pede lote maior, máquina alta pede outro recurso,
// terceiro alto pede internalizar, material alto pede negociar compra. Um total
// só não diz nada disso. Mesmo raciocínio da estrutura de componentes de custo do
// SAP e das taxas de CIF do TOTVS.
type ComponentesDeCusto struct {
	// Material comprado consumido por este item e pelos níveis abaixo.
	Material decimal.Decimal
	// Preparação de máquina, já diluída pelo lote de referência.
	Setup decimal.Decimal
	// Ocupação de máquina.
	Maquina decimal.Decimal
	// Mão de obra direta.
	MaoDeObra decimal.Decimal
	// Serviço de terceiro (operação externa do roteiro).
	Subcontratacao decimal.Decimal
	// Indiretos aplicados pelo esquema de rateio.
	Overhead decimal.Decimal
	// NivelProprio é o que ESTA etapa agrega: conversão + indiretos.
	// NivelInferior é o custo total dos componentes fabricados abaixo.
	// A soma dos dois é o total — é o corte que diz se o custo subiu aqui ou lá
	// embaixo.
	NivelProprio  decimal.Decimal
	NivelInferior decimal.Decimal
}

// Total soma os componentes. Nunca somar NivelProprio + NivelInferior aqui: eles
// são uma LEITURA dos mesmos valores por outro corte, e somar tudo dobraria o
// custo.
func (c ComponentesDeCusto) Total() decimal.Decimal {
	return c.Material.Add(c.Setup).Add(c.Maquina).Add(c.MaoDeObra).Add(c.Subcontratacao).Add(c.Overhead)
}

// Conversao é o que a fábrica agrega em horas e preparação, sem indiretos.
func (c ComponentesDeCusto) Conversao() decimal.Decimal {
	return c.Setup.Add(c.Maquina).Add(c.MaoDeObra)
}

// BaseDeRateio diz sobre qual componente a taxa de indireto incide.
type BaseDeRateio string

const (
	BaseMaterial       BaseDeRateio = "MATERIAL"
	BaseSetup          BaseDeRateio = "SETUP"
	BaseMaquina        BaseDeRateio = "MAQUINA"
	BaseMaoDeObra      BaseDeRateio = "MAO_DE_OBRA"
	BaseConversao      BaseDeRateio = "CONVERSAO"
	BaseSubcontratacao BaseDeRateio = "SUBCONTRATACAO"
	BaseTotal          BaseDeRateio = "TOTAL"
)

// MetodoDeRateio diz como a taxa é aplicada.
type MetodoDeRateio string

const (
	// MetodoPercentual: taxa em fração sobre o valor da base (0,12 = 12%).
	MetodoPercentual MetodoDeRateio = "PERCENTUAL"
	// MetodoValorPorHora: reais por hora da base. Exige base medida em horas.
	MetodoValorPorHora MetodoDeRateio = "VALOR_POR_HORA"
	// MetodoValorPorUnidade: reais fixos por unidade produzida.
	MetodoValorPorUnidade MetodoDeRateio = "VALOR_POR_UNIDADE"
)

// RegraDeRateio é uma linha do esquema de cálculo de indiretos.
type RegraDeRateio struct {
	ID            int64
	EnterpriseID  int64
	Code          string
	Description   string
	Base          BaseDeRateio
	Method        MetodoDeRateio
	Rate          decimal.Decimal
	WorkCenterID  *int64
	ItemCode      *int64
	PlanoContasID *int64
	CentroCustoID *int64
	ValidFrom     time.Time
	ValidTo       *time.Time
	IsActive      bool
	Notes         *string
	CreatedBy     uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// VigenteEm diz se a regra vale na data da apuração. Regra fora de vigência
// aplicada é custo errado que ninguém questiona, porque o número sai plausível.
func (r *RegraDeRateio) VigenteEm(data time.Time) bool {
	if !r.IsActive {
		return false
	}
	dia := data.Truncate(24 * time.Hour)
	if dia.Before(r.ValidFrom.Truncate(24 * time.Hour)) {
		return false
	}
	if r.ValidTo != nil && dia.After(r.ValidTo.Truncate(24*time.Hour)) {
		return false
	}
	return true
}

// AplicaAoItem diz se a regra alcança este item e centro de trabalho. Escopo nulo
// é curinga: regra sem item nem centro vale para toda a fábrica.
func (r *RegraDeRateio) AplicaAoItem(itemCode int64, centrosDoRoteiro map[int64]bool) bool {
	if r.ItemCode != nil && *r.ItemCode != itemCode {
		return false
	}
	if r.WorkCenterID != nil && !centrosDoRoteiro[*r.WorkCenterID] {
		return false
	}
	return true
}

// HorasDaBase é o denominador do método por hora.
func (r *RegraDeRateio) HorasDaBase(maquina, maoDeObra, setup float64) float64 {
	switch r.Base {
	case BaseMaquina:
		return maquina
	case BaseMaoDeObra:
		return maoDeObra
	case BaseSetup:
		return setup
	case BaseConversao:
		return maquina + maoDeObra + setup
	default:
		return 0
	}
}

// ValorDaBase é o denominador do método percentual.
func (r *RegraDeRateio) ValorDaBase(c ComponentesDeCusto) decimal.Decimal {
	switch r.Base {
	case BaseMaterial:
		return c.Material
	case BaseSetup:
		return c.Setup
	case BaseMaquina:
		return c.Maquina
	case BaseMaoDeObra:
		return c.MaoDeObra
	case BaseConversao:
		return c.Conversao()
	case BaseSubcontratacao:
		return c.Subcontratacao
	case BaseTotal:
		// Sobre tudo que veio ANTES do rateio. Incluir Overhead aqui faria a
		// segunda regra incidir sobre a primeira, e a ordem das regras passaria a
		// mudar o custo.
		return c.Material.Add(c.Conversao()).Add(c.Subcontratacao)
	default:
		return decimal.Zero
	}
}

// RateioAplicado é o rastro de uma regra no custo do item: sem isso o indireto é
// um número que aparece no total e ninguém sabe de onde veio.
type RateioAplicado struct {
	RuleID      int64
	Code        string
	Description string
	Base        BaseDeRateio
	Method      MetodoDeRateio
	Rate        decimal.Decimal
	BaseValue   decimal.Decimal
	Applied     decimal.Decimal
}

// HorasDoRoteiro são as horas por unidade que a apuração mediu, usadas pelas
// regras de valor por hora.
type HorasDoRoteiro struct {
	Maquina   float64
	MaoDeObra float64
	Setup     float64
}

// HistoricoDeCusto é uma apuração gravada, aberta por componente. Uma linha por
// execução, nunca sobrescrita: é o que responde "por que o custo deste item subiu
// 12% este mês" comparando duas apurações componente a componente.
type HistoricoDeCusto struct {
	ID           int64
	EnterpriseID int64
	ItemCode     int64
	Mask         string
	LotSize      decimal.Decimal
	Componentes  ComponentesDeCusto
	TotalCost    decimal.Decimal
	Currency     string
	Rateios      []RateioAplicado
	CalculatedAt time.Time
	CalculatedBy uuid.UUID
}
