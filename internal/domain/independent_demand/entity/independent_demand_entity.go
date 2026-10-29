package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Motivo da demanda independente. A família decide como ela se encerra (ver
// ConsumidaPelaVenda).
const (
	MotivoEstoque          = "ESTOQUE"
	MotivoReposicao        = "REPOSICAO"
	MotivoAntecipacaoVenda = "ANTECIPACAO_VENDA"
	MotivoAmostra          = "AMOSTRA"
	MotivoFeira            = "FEIRA"
	MotivoConsumoInterno   = "CONSUMO_INTERNO"
	MotivoAssistencia      = "ASSISTENCIA"
	MotivoProjeto          = "PROJETO"
)

// Situação gravada. Atendida, consumida e vencida são calculadas.
const (
	SituacaoAberta    = "ABERTA"
	SituacaoEncerrada = "ENCERRADA"
	SituacaoCancelada = "CANCELADA"
)

// ReferenciaAtendimento é a referência da saída de estoque que atende a
// demanda de finalidade própria (reference_code = código da demanda).
const ReferenciaAtendimento = "DEMANDA_INDEPENDENTE"

var motivos = map[string]string{
	MotivoEstoque:          "Produção para estoque",
	MotivoReposicao:        "Reposição de estoque",
	MotivoAntecipacaoVenda: "Antecipação de venda",
	MotivoAmostra:          "Amostra / protótipo",
	MotivoFeira:            "Feira / demonstração",
	MotivoConsumoInterno:   "Consumo interno",
	MotivoAssistencia:      "Assistência / peça de reposição",
	MotivoProjeto:          "Projeto / obra",
}

// MotivoValido diz se o motivo existe.
func MotivoValido(m string) bool { _, ok := motivos[m]; return ok }

// DescricaoDoMotivo é o nome do motivo para mensagens.
func DescricaoDoMotivo(m string) string { return motivos[m] }

// ConsumidaPelaVenda: a demanda é expectativa de venda — o pedido de venda do
// item a consome (SAP, estratégia 40) e ela vence quando nenhuma venda pode
// mais consumi-la. As de finalidade própria (amostra, feira, consumo interno,
// assistência, projeto) seguram o estoque até a saída para a finalidade.
func ConsumidaPelaVenda(motivo string) bool {
	switch motivo {
	case MotivoEstoque, MotivoReposicao, MotivoAntecipacaoVenda:
		return true
	}
	return false
}

type IndependentDemand struct {
	ID             int64
	CodeDemand     int64
	ItemCode       int64
	Mask           *string
	CostCenterCode *int64
	Quantity       decimal.Decimal
	DemandDate     time.Time
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CreatedBy      uuid.UUID

	Motivo       string
	WarehouseID  *int64
	CustomerCode *int64
	Projeto      *string
	Prioridade   int
	Observacao   *string
	// Programacao reúne as entregas cadastradas de uma vez.
	Programacao        *uuid.UUID
	Situacao           string
	EncerradaEm        *time.Time
	EncerradaPor       *uuid.UUID
	MotivoEncerramento *string
	UpdatedBy          *uuid.UUID

	// Atendida: saídas de estoque para a finalidade (referência
	// DEMANDA_INDEPENDENTE), sem as estornadas. Preenchida pela leitura.
	Atendida decimal.Decimal
	// Código de negócio e nome do item, para a tela (preenchidos pela leitura).
	ItemBusinessCode string
	ItemName         string
	// Nomes do almoxarifado e do cliente, para a tela (preenchidos pela leitura).
	WarehouseNome string
	CustomerNome  string
}

// Saldo é o que ainda falta atender pela finalidade.
func (d *IndependentDemand) Saldo() decimal.Decimal {
	return decimal.Max(decimal.Zero, d.Quantity.Sub(d.Atendida))
}

// Validar confere o que não depende do banco.
func (d *IndependentDemand) Validar() string {
	switch {
	case d.ItemCode <= 0:
		return "informe o item"
	case !d.Quantity.IsPositive():
		return "a quantidade precisa ser maior que zero"
	case d.DemandDate.IsZero():
		return "informe a data da necessidade"
	case !MotivoValido(d.Motivo):
		return "motivo inválido"
	case d.Prioridade < 1 || d.Prioridade > 5:
		return "a prioridade vai de 1 (mais urgente) a 5"
	}
	if (d.Motivo == MotivoProjeto) && (d.Projeto == nil || *d.Projeto == "") {
		return "informe o projeto ou a obra da demanda"
	}
	return ""
}

// Entrega de uma programação.
type Entrega struct {
	Data       time.Time
	Quantidade decimal.Decimal
}

// Programar divide a quantidade em n entregas a cada intervaloDias, a partir da
// primeira data. A divisão é exata até 4 casas; a sobra do arredondamento vai
// para a última entrega, para a soma bater com o total.
func Programar(total decimal.Decimal, primeira time.Time, n, intervaloDias int) ([]Entrega, string) {
	switch {
	case n < 1 || n > 104:
		return nil, "a programação vai de 1 a 104 entregas"
	case n > 1 && intervaloDias < 1:
		return nil, "informe o intervalo entre as entregas, em dias"
	case !total.IsPositive():
		return nil, "a quantidade precisa ser maior que zero"
	}
	parte := total.Div(decimal.NewFromInt(int64(n))).Truncate(4)
	if !parte.IsPositive() {
		return nil, "a quantidade é pequena demais para tantas entregas"
	}
	out := make([]Entrega, n)
	soma := decimal.Zero
	for i := 0; i < n; i++ {
		out[i] = Entrega{Data: primeira.AddDate(0, 0, i*intervaloDias), Quantidade: parte}
		soma = soma.Add(parte)
	}
	out[n-1].Quantidade = out[n-1].Quantidade.Add(total.Sub(soma))
	return out, ""
}

// OrdemDaDemanda é a ordem que cobre a demanda: a planejada nascida da
// sugestão firmada (pelo pegging) ou a OF de "Gerar OF agora".
type OrdemDaDemanda struct {
	PlannedOrderCode  *int64
	ProductionOrderID *int64
	OrderNumber       *int64
	Status            string
	Quantidade        decimal.Decimal
	Planejada         decimal.Decimal
	Produzida         decimal.Decimal
}

// Coberta é o que as ordens ligadas cobrem da demanda; a cancelada não cobre.
func Coberta(ordens []OrdemDaDemanda) decimal.Decimal {
	total := decimal.Zero
	for _, o := range ordens {
		if o.Status == "CANCELLED" {
			continue
		}
		total = total.Add(o.Quantidade)
	}
	return total
}

// SemOrdem é o que o MRP ainda vai planejar e nenhuma ordem ligada cobre: é o
// que "Gerar OF agora" oferece.
func SemOrdem(aPlanejar, coberta decimal.Decimal) decimal.Decimal {
	return decimal.Max(decimal.Zero, aPlanejar.Sub(coberta))
}

// Filtro da lista de acompanhamento.
type Filtro struct {
	ItemCode *int64
	Motivo   string
	Situacao string
	De, Ate  *time.Time
	Limite   int
}
