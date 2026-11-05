package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// ChaveCusto identifica o saldo que tem custo próprio.
type ChaveCusto struct {
	Almoxarifado int64
	Mask         string
}

// MovimentoCusto é um movimento visto pelo recálculo, na ordem de data e número.
type MovimentoCusto struct {
	ID           int64
	Almoxarifado int64
	Mask         string
	Tipo         string
	Quantidade   decimal.Decimal
	Unitario     decimal.Decimal
	Total        decimal.Decimal
	Referencia   string
	EstornaDe    *int64
	// SemMedia: tipo de movimento que não atualiza o custo médio.
	SemMedia bool
	Data     time.Time
}

// MudancaCusto é o valor novo de um movimento.
type MudancaCusto struct {
	MovimentoID    int64
	Almoxarifado   int64
	Mask           string
	Data           time.Time
	UnitarioAntes  decimal.Decimal
	UnitarioDepois decimal.Decimal
	TotalAntes     decimal.Decimal
	TotalDepois    decimal.Decimal
	// DiferencaResultado: quanto a mais (ou a menos) saiu para o resultado. Só a
	// saída de verdade conta: a transferência sai de um saldo e entra em outro.
	DiferencaResultado decimal.Decimal
}

// RecalcularCusto refaz o custo médio na ordem dos movimentos, partindo dos
// estados informados. É o que corrige o lançamento retroativo: em tempo real ele
// entrou fora de ordem e as saídas seguintes ficaram com o médio antigo.
//
// Entrada fica pelo próprio preço; saída sai pelo médio do momento; desfazer uma
// entrada sai pelo preço dela; transferência entra pelo que saiu da origem.
func RecalcularCusto(estados map[ChaveCusto]CostingState, movs []MovimentoCusto) []MudancaCusto {
	var mudancas []MudancaCusto
	novoPreco := map[int64]decimal.Decimal{}
	precoTransferido := decimal.Zero
	temTransferencia := false
	for _, m := range movs {
		k := ChaveCusto{Almoxarifado: m.Almoxarifado, Mask: m.Mask}
		st := estados[k]
		if m.Tipo == MovementTypeCostAdjustment {
			st.TotalCost = st.TotalCost.Add(m.Total).Round(CasasValor)
			if st.Quantity.IsPositive() {
				st.AvgCost = st.TotalCost.DivRound(st.Quantity, CasasCusto)
			}
			estados[k] = st
			continue
		}
		delta := SignedQuantity(m.Tipo, m.Quantidade)
		if delta.IsZero() {
			continue
		}
		preco := m.Unitario
		diferenca := decimal.Zero
		if m.EstornaDe != nil {
			if p, ok := novoPreco[*m.EstornaDe]; ok {
				preco = p
			}
		}
		if delta.IsPositive() {
			switch {
			case m.Tipo == MovementTypeTransferIn && temTransferencia:
				preco = precoTransferido
			case m.EstornaDe == nil && (m.Tipo == MovementTypeAdjustment || m.SemMedia) && st.AvgCost.IsPositive():
				preco = st.AvgCost
			}
			st, _ = ApplyMovementCosting(st, delta, preco)
		} else if m.EstornaDe != nil || desfazEntrada(m) {
			st, _ = ApplyEntryReversalCosting(st, delta, preco)
		} else {
			if st.AvgCost.IsPositive() {
				preco = st.AvgCost
			}
			st, _ = ApplyMovementCosting(st, delta, preco)
			if m.Tipo == MovementTypeTransferOut {
				precoTransferido, temTransferencia = preco, true
			}
		}
		estados[k] = st
		novoPreco[m.ID] = preco
		total := delta.Abs().Mul(preco).Round(CasasValor)
		if preco.Equal(m.Unitario) && total.Equal(m.Total.Abs()) {
			continue
		}
		if delta.IsNegative() && m.Tipo != MovementTypeTransferOut {
			diferenca = total.Sub(m.Total.Abs())
		}
		mudancas = append(mudancas, MudancaCusto{
			MovimentoID: m.ID, Almoxarifado: m.Almoxarifado, Mask: m.Mask, Data: m.Data,
			UnitarioAntes: m.Unitario, UnitarioDepois: preco, TotalAntes: m.Total.Abs(), TotalDepois: total,
			DiferencaResultado: diferenca,
		})
	}
	return mudancas
}

func desfazEntrada(m MovimentoCusto) bool {
	return m.Referencia == ReferenceTypeNFEntry || m.Referencia == ReferenceTypePurchaseOrder
}

// Fechamento é um mês de estoque fechado.
type Fechamento struct {
	ID                int64           `json:"id"`
	Competencia       string          `json:"competence"`
	Status            string          `json:"status"`
	ValorTotal        decimal.Decimal `json:"total_value"`
	ItensRecalculados int             `json:"recalculated_items"`
	Diferenca         decimal.Decimal `json:"variance_value"`
	Contabilizado     bool            `json:"journal_posted"`
	FechadoEm         time.Time       `json:"closed_at"`
	ReabertoEm        *time.Time      `json:"reopened_at,omitempty"`
	MotivoReabertura  *string         `json:"reopen_reason,omitempty"`
}

// ResultadoFechamento diz o que o fechamento faz ou fez.
type ResultadoFechamento struct {
	Competencia         string          `json:"competence"`
	Simulacao           bool            `json:"simulation"`
	ItensNaFoto         int             `json:"items_in_snapshot"`
	ValorTotal          decimal.Decimal `json:"total_value"`
	ItensRecalculados   int             `json:"recalculated_items"`
	MovimentosAlterados int             `json:"changed_movements"`
	DiferencaRecalculo  decimal.Decimal `json:"recalc_variance"`
	DiferencaSaldoNeg   decimal.Decimal `json:"negative_stock_variance"`
	Contabilizado       bool            `json:"journal_posted"`
	// Pendencias impedem fechar. Avisos não.
	Pendencias []string `json:"blocking"`
	Avisos     []string `json:"warnings"`
}
