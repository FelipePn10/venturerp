// Package lote é o lote de inspeção: a inspeção de recebimento, de processo e
// final num só objeto, com as características copiadas do plano, o plano de
// amostragem NBR 5426, os resultados por unidade da amostra e a decisão de uso.
package lote

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/FelipePn10/panossoerp/internal/domain/quality/amostragem"
)

type Origem string

const (
	OrigemRecebimento Origem = "RECEBIMENTO"
	OrigemProcesso    Origem = "PROCESSO"
	OrigemFinal       Origem = "FINAL"
	OrigemDevolucao   Origem = "DEVOLUCAO"
	OrigemManual      Origem = "MANUAL"
)

func (o Origem) Valida() bool {
	switch o {
	case OrigemRecebimento, OrigemProcesso, OrigemFinal, OrigemDevolucao, OrigemManual:
		return true
	}
	return false
}

type Status string

const (
	Aberto    Status = "ABERTO"
	Decidido  Status = "DECIDIDO"
	Cancelado Status = "CANCELADO"
)

type TipoCaracteristica string

const (
	Variavel  TipoCaracteristica = "VARIAVEL"
	Atributo  TipoCaracteristica = "ATRIBUTO"
	Estrutura TipoCaracteristica = "ESTRUTURA"
)

// Caracteristica copiada do plano na abertura do lote.
type Caracteristica struct {
	ID          int64              `json:"id"`
	Sequencia   int                `json:"sequencia"`
	Nome        string             `json:"nome"`
	Tipo        TipoCaracteristica `json:"tipo"`
	Obrigatoria bool               `json:"obrigatoria"`
	Critica     bool               `json:"critica"`
	Nominal     *decimal.Decimal   `json:"nominal,omitempty"`
	Minimo      *decimal.Decimal   `json:"minimo,omitempty"`
	Maximo      *decimal.Decimal   `json:"maximo,omitempty"`
	Unidade     *string            `json:"unidade,omitempty"`
	Instrumento *string            `json:"instrumento,omitempty"`
	Norma       *string            `json:"norma,omitempty"`
	Referencia  *string            `json:"referencia,omitempty"`
	OrigemStep  *int64             `json:"origem_step_id,omitempty"`
	OrigemCarac *int64             `json:"origem_characteristic_id,omitempty"`
}

// Conformidade do resultado. VARIAVEL: o valor dentro de [mínimo, máximo] (o
// servidor decide, não a tela). ATRIBUTO/ESTRUTURA: o inspetor diz conforme.
func (c Caracteristica) Conformidade(valor *decimal.Decimal, conforme *bool) (bool, error) {
	if c.Tipo == Variavel {
		if valor == nil {
			return false, domainerr.NewValidation(fmt.Sprintf("informe o valor medido de %s", c.Nome))
		}
		if c.Minimo == nil && c.Maximo == nil {
			if conforme == nil {
				return false, domainerr.NewValidation(fmt.Sprintf("%s não tem limites: informe se o valor está conforme", c.Nome))
			}
			return *conforme, nil
		}
		ok := (c.Minimo == nil || valor.GreaterThanOrEqual(*c.Minimo)) && (c.Maximo == nil || valor.LessThanOrEqual(*c.Maximo))
		return ok, nil
	}
	if conforme == nil {
		return false, domainerr.NewValidation(fmt.Sprintf("informe se %s está conforme", c.Nome))
	}
	return *conforme, nil
}

// Resultado de uma característica numa unidade da amostra.
type Resultado struct {
	ID               int64            `json:"id"`
	CaracteristicaID int64            `json:"characteristic_id"`
	AmostraIndice    int              `json:"amostra_indice"`
	Valor            *decimal.Decimal `json:"valor,omitempty"`
	Atributo         *string          `json:"atributo,omitempty"`
	Conforme         bool             `json:"conforme"`
	Observacao       *string          `json:"observacao,omitempty"`
	// InstrumentoID: instrumento usado na medição (metrologia).
	InstrumentoID *int64     `json:"instrument_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	CreatedBy     *uuid.UUID `json:"created_by,omitempty"`
}

// Lote de inspeção.
type Lote struct {
	ID                    int64             `json:"id"`
	Numero                int64             `json:"numero"`
	Origem                Origem            `json:"origem"`
	Status                Status            `json:"status"`
	ItemCode              int64             `json:"item_code"`
	Mask                  string            `json:"mask"`
	Lote                  *string           `json:"lote,omitempty"`
	Serie                 *string           `json:"serie,omitempty"`
	Quantidade            decimal.Decimal   `json:"quantidade"`
	WarehouseID           *int64            `json:"warehouse_id,omitempty"`
	SupplierCode          *int64            `json:"supplier_code,omitempty"`
	PurchaseOrderCode     *int64            `json:"purchase_order_code,omitempty"`
	PurchaseOrderItemCode *int64            `json:"purchase_order_item_code,omitempty"`
	FiscalEntryCode       *int64            `json:"fiscal_entry_code,omitempty"`
	ReceivingNoticeCode   *int64            `json:"receiving_notice_code,omitempty"`
	ProductionOrderID     *int64            `json:"production_order_id,omitempty"`
	RouteOperationID      *int64            `json:"route_operation_id,omitempty"`
	ReceivingRouteID      *int64            `json:"receiving_route_id,omitempty"`
	InspectionPlanID      *int64            `json:"inspection_plan_id,omitempty"`
	ProcedimentoID        *int64            `json:"sampling_procedure_id,omitempty"`
	ChaveAmostragem       *string           `json:"chave_amostragem,omitempty"`
	Plano                 *amostragem.Plano `json:"plano,omitempty"`
	Pulado                bool              `json:"pulado"`
	Certificado           *string           `json:"certificado,omitempty"`
	NotaFornecedor        *string           `json:"nota_fornecedor,omitempty"`
	Modelo                *string           `json:"modelo,omitempty"`
	Observacoes           *string           `json:"observacoes,omitempty"`
	MotivoCancelamento    *string           `json:"motivo_cancelamento,omitempty"`
	LegacyKind            *string           `json:"legacy_kind,omitempty"`
	LegacyID              *int64            `json:"legacy_id,omitempty"`
	Caracteristicas       []Caracteristica  `json:"caracteristicas"`
	Resultados            []Resultado       `json:"resultados"`
	Decisao               *Decisao          `json:"decisao,omitempty"`
	CreatedAt             time.Time         `json:"created_at"`
	CreatedBy             *uuid.UUID        `json:"created_by,omitempty"`
	UpdatedAt             time.Time         `json:"updated_at"`
}

// RetemEstoque: o material espera a decisão no almoxarifado de inspeção.
func (l *Lote) RetemEstoque() bool { return l.WarehouseID != nil && *l.WarehouseID > 0 }

// TamanhoDaAmostra devolve n do plano; sem plano, a quantidade inteira do lote
// (mínimo 1) — a inspeção sem procedimento é por unidade informada.
func (l *Lote) TamanhoDaAmostra() int64 {
	if l.Plano != nil && l.Plano.N > 0 {
		return l.Plano.N
	}
	return 1
}

// Avaliacao resume os resultados contra o plano.
type Avaliacao struct {
	// NaoConformes é d: unidades da amostra com alguma característica não conforme.
	NaoConformes int `json:"nao_conformes"`
	// CriticaReprovada: característica crítica não conforme recusa o lote.
	CriticaReprovada bool `json:"critica_reprovada"`
	// Completa: toda característica obrigatória tem resultado em todas as
	// unidades da amostra.
	Completa  bool     `json:"completa"`
	Faltantes []string `json:"faltantes,omitempty"`
	// AceitoPelaAmostragem: d ≤ Ac e nenhuma crítica reprovada.
	AceitoPelaAmostragem bool `json:"aceito_pela_amostragem"`
	Ac                   int  `json:"ac"`
	Re                   int  `json:"re"`
}

// Avaliar calcula a avaliação do lote a partir dos resultados.
func (l *Lote) Avaliar() Avaliacao {
	n := l.TamanhoDaAmostra()
	ac, re := 0, 1
	if l.Plano != nil {
		ac, re = l.Plano.Ac, l.Plano.Re
	}
	porCarac := map[int64]map[int]bool{}
	unidadesNC := map[int]bool{}
	critica := map[int64]bool{}
	for _, c := range l.Caracteristicas {
		critica[c.ID] = c.Critica
	}
	a := Avaliacao{Ac: ac, Re: re}
	for _, r := range l.Resultados {
		if porCarac[r.CaracteristicaID] == nil {
			porCarac[r.CaracteristicaID] = map[int]bool{}
		}
		porCarac[r.CaracteristicaID][r.AmostraIndice] = true
		if !r.Conforme {
			unidadesNC[r.AmostraIndice] = true
			if critica[r.CaracteristicaID] {
				a.CriticaReprovada = true
			}
		}
	}
	a.NaoConformes = len(unidadesNC)
	a.Completa = true
	for _, c := range l.Caracteristicas {
		if !c.Obrigatoria {
			continue
		}
		if got := int64(len(porCarac[c.ID])); got < n {
			a.Completa = false
			a.Faltantes = append(a.Faltantes, fmt.Sprintf("%s (%d de %d)", c.Nome, got, n))
		}
	}
	sort.Strings(a.Faltantes)
	a.AceitoPelaAmostragem = !a.CriticaReprovada && a.NaoConformes <= ac
	return a
}

// Decisao de uso: para onde vai cada parte do lote.
type Decisao struct {
	Tipo              string          `json:"decisao"`
	Aceita            decimal.Decimal `json:"qtd_aceita"`
	Restrita          decimal.Decimal `json:"qtd_restrita"`
	Retrabalho        decimal.Decimal `json:"qtd_retrabalho"`
	Rejeitada         decimal.Decimal `json:"qtd_rejeitada"`
	Sucata            decimal.Decimal `json:"qtd_sucata"`
	DestinoAceito     *int64          `json:"destino_aceito,omitempty"`
	DestinoRestrito   *int64          `json:"destino_restrito,omitempty"`
	DestinoRetrabalho *int64          `json:"destino_retrabalho,omitempty"`
	DestinoRejeitado  *int64          `json:"destino_rejeitado,omitempty"`
	DestinoSucata     *int64          `json:"destino_sucata,omitempty"`
	Tratamento        *string         `json:"tratamento,omitempty"`
	NaoConformes      *int            `json:"nao_conformes,omitempty"`
	AceitoAmostragem  *bool           `json:"aceito_amostragem,omitempty"`
	AfetaIQF          bool            `json:"afeta_iqf"`
	Motivo            *string         `json:"motivo,omitempty"`
	DecididoEm        time.Time       `json:"decided_at"`
	DecididoPor       *uuid.UUID      `json:"decided_by,omitempty"`
}

const (
	DecisaoAceito          = "ACEITO"
	DecisaoAceitoRestricao = "ACEITO_COM_RESTRICAO"
	DecisaoRejeitado       = "REJEITADO"
	DecisaoParcial         = "PARCIAL"
)

var tratamentos = map[string]bool{"DEVOLVER": true, "SUCATEAR": true, "RETRABALHAR": true, "SELECIONAR": true, "CONCESSAO": true}

// Total das partes.
func (d Decisao) Total() decimal.Decimal {
	return d.Aceita.Add(d.Restrita).Add(d.Retrabalho).Add(d.Rejeitada).Add(d.Sucata)
}

// Aproveitada: o que segue para uso (aceita + restrita).
func (d Decisao) Aproveitada() decimal.Decimal { return d.Aceita.Add(d.Restrita) }

// classificar deriva o tipo da decisão das quantidades.
func (d Decisao) classificar() string {
	naoAceita := d.Retrabalho.Add(d.Rejeitada).Add(d.Sucata)
	switch {
	case naoAceita.IsZero() && d.Restrita.IsZero():
		return DecisaoAceito
	case naoAceita.IsZero():
		return DecisaoAceitoRestricao
	case d.Aproveitada().IsZero():
		return DecisaoRejeitado
	default:
		return DecisaoParcial
	}
}

// Almoxarifado como a decisão precisa conhecê-lo.
type Almoxarifado struct {
	ID          int64
	Localizacao string // INTERNO, INSPECAO, REJEICAO…
	Disposicao  bool
}

// PrepararDecisao valida e completa a decisão: soma igual à quantidade do lote,
// destinos coerentes (aceito vai para almoxarifado disponível; rejeitado não),
// aceitar exige inspeção completa, aceitar o lote todo apesar da recusa da
// amostragem exige concessão com motivo.
func (l *Lote) PrepararDecisao(d *Decisao, almox map[int64]Almoxarifado) error {
	if l.Status != Aberto {
		return domainerr.NewValidation(fmt.Sprintf("o lote %d está %s: a decisão de uso já foi tomada ou ele foi cancelado", l.Numero, strings.ToLower(string(l.Status))))
	}
	for _, q := range []decimal.Decimal{d.Aceita, d.Restrita, d.Retrabalho, d.Rejeitada, d.Sucata} {
		if q.IsNegative() {
			return domainerr.NewValidation("as quantidades da decisão não podem ser negativas")
		}
	}
	if !d.Total().Equal(l.Quantidade) {
		return domainerr.NewValidation(fmt.Sprintf("a decisão soma %s e o lote tem %s: distribua o lote inteiro (aceita, restrita, retrabalho, rejeitada, sucata)",
			d.Total().String(), l.Quantidade.String()))
	}
	if d.Tratamento != nil {
		t := strings.ToUpper(strings.TrimSpace(*d.Tratamento))
		if t == "" {
			d.Tratamento = nil
		} else if !tratamentos[t] {
			return domainerr.NewValidation("tratamento inválido (DEVOLVER, SUCATEAR, RETRABALHAR, SELECIONAR ou CONCESSAO)")
		} else {
			d.Tratamento = &t
		}
	}
	if d.Motivo != nil {
		if m := strings.TrimSpace(*d.Motivo); m == "" {
			d.Motivo = nil
		} else {
			d.Motivo = &m
		}
	}
	av := l.Avaliar()
	d.Tipo = d.classificar()
	if !l.Pulado && d.Aproveitada().IsPositive() && !av.Completa {
		return domainerr.NewValidation("a inspeção não está completa — faltam resultados: " + strings.Join(av.Faltantes, "; ") + ". Só é possível rejeitar sem concluir a inspeção")
	}
	if !l.Pulado && d.Aproveitada().IsPositive() {
		nc, ok := av.NaoConformes, av.AceitoPelaAmostragem
		d.NaoConformes, d.AceitoAmostragem = &nc, &ok
		if !ok && d.Tipo == DecisaoAceito && (d.Tratamento == nil || *d.Tratamento != "CONCESSAO" || d.Motivo == nil) {
			return domainerr.NewValidation(fmt.Sprintf("a amostragem recusou o lote (%d não conforme(s), Ac %d%s): aceitar tudo exige tratamento CONCESSAO e o motivo",
				av.NaoConformes, av.Ac, map[bool]string{true: ", característica crítica reprovada", false: ""}[av.CriticaReprovada]))
		}
	} else if !l.Pulado && len(l.Resultados) > 0 {
		nc, ok := av.NaoConformes, av.AceitoPelaAmostragem
		d.NaoConformes, d.AceitoAmostragem = &nc, &ok
	}
	if d.Tipo != DecisaoAceito && d.Motivo == nil {
		return domainerr.NewValidation("informe o motivo da decisão (há quantidade restrita, em retrabalho, rejeitada ou sucateada)")
	}
	if !l.RetemEstoque() {
		// Inspeção de processo: não há estoque a mover.
		d.DestinoAceito, d.DestinoRestrito, d.DestinoRetrabalho, d.DestinoRejeitado, d.DestinoSucata = nil, nil, nil, nil, nil
		return nil
	}
	origem := *l.WarehouseID
	exigir := func(q decimal.Decimal, dest *int64, nome string, disponivel bool) error {
		if !q.IsPositive() {
			return nil
		}
		if dest == nil {
			return domainerr.NewValidation(fmt.Sprintf("informe o almoxarifado de destino da quantidade %s", nome))
		}
		if *dest == origem {
			return domainerr.NewValidation(fmt.Sprintf("o destino da quantidade %s é o próprio almoxarifado de inspeção", nome))
		}
		a, ok := almox[*dest]
		if !ok {
			return domainerr.NewValidation(fmt.Sprintf("o almoxarifado de destino da quantidade %s não existe nesta empresa", nome))
		}
		bloqueado := !a.Disposicao || a.Localizacao == "INSPECAO" || a.Localizacao == "REJEICAO"
		if disponivel && bloqueado {
			return domainerr.NewValidation(fmt.Sprintf("a quantidade %s precisa ir para um almoxarifado disponível (não de inspeção, rejeição ou sem disposição)", nome))
		}
		if !disponivel && a.Localizacao == "INSPECAO" {
			return domainerr.NewValidation(fmt.Sprintf("a quantidade %s não pode voltar para um almoxarifado de inspeção", nome))
		}
		return nil
	}
	if err := exigir(d.Aceita, d.DestinoAceito, "aceita", true); err != nil {
		return err
	}
	if err := exigir(d.Restrita, d.DestinoRestrito, "restrita", true); err != nil {
		return err
	}
	if err := exigir(d.Retrabalho, d.DestinoRetrabalho, "em retrabalho", false); err != nil {
		return err
	}
	if err := exigir(d.Rejeitada, d.DestinoRejeitado, "rejeitada", false); err != nil {
		return err
	}
	// Sucata sem destino é baixada; com destino, transferida.
	if d.Sucata.IsPositive() && d.DestinoSucata != nil {
		if err := exigir(d.Sucata, d.DestinoSucata, "sucateada", false); err != nil {
			return err
		}
	}
	if d.Rejeitada.IsPositive() {
		if a := almox[*d.DestinoRejeitado]; a.Disposicao && a.Localizacao != "REJEICAO" {
			return domainerr.NewValidation("a quantidade rejeitada precisa ir para um almoxarifado de rejeição (ou sem disposição), não para um disponível")
		}
	}
	return nil
}

// Perna de estoque da decisão.
type Perna struct {
	Destino    *int64 // nil: baixa (sucata)
	Quantidade decimal.Decimal
	Parte      string
}

// Pernas devolve as movimentações da decisão, a partir do almoxarifado do lote.
func (d Decisao) Pernas() []Perna {
	var out []Perna
	add := func(q decimal.Decimal, dest *int64, parte string) {
		if q.IsPositive() {
			out = append(out, Perna{Destino: dest, Quantidade: q, Parte: parte})
		}
	}
	add(d.Aceita, d.DestinoAceito, "ACEITA")
	add(d.Restrita, d.DestinoRestrito, "RESTRITA")
	add(d.Retrabalho, d.DestinoRetrabalho, "RETRABALHO")
	add(d.Rejeitada, d.DestinoRejeitado, "REJEITADA")
	add(d.Sucata, d.DestinoSucata, "SUCATA")
	return out
}
