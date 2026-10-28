// Package certificado é o certificado de qualidade entregue ao cliente: o que
// foi inspecionado no lote vendido (medições resumidas por característica),
// a decisão de uso e a corrida/certificado da usina — congelado na emissão.
package certificado

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/FelipePn10/panossoerp/internal/domain/quality/lote"
)

// CaracResumo resume uma característica: especificação e o que foi medido.
type CaracResumo struct {
	Nome         string   `json:"nome"`
	Tipo         string   `json:"tipo"`
	Nominal      string   `json:"nominal,omitempty"`
	Minimo       string   `json:"minimo,omitempty"`
	Maximo       string   `json:"maximo,omitempty"`
	Unidade      string   `json:"unidade,omitempty"`
	N            int      `json:"n"`
	MedidoMinimo string   `json:"medido_minimo,omitempty"`
	MedidoMaximo string   `json:"medido_maximo,omitempty"`
	Media        string   `json:"media,omitempty"`
	NaoConformes int      `json:"nao_conformes"`
	Instrumentos []string `json:"instrumentos,omitempty"`
	Critica      bool     `json:"critica"`
}

// LoteResumo é um lote de inspeção no certificado.
type LoteResumo struct {
	ID              int64         `json:"id"`
	Numero          int64         `json:"numero"`
	Origem          string        `json:"origem"`
	Decisao         string        `json:"decisao"`
	Concessao       bool          `json:"concessao"`
	Tratamento      string        `json:"tratamento,omitempty"`
	Aproveitada     string        `json:"qtd_aproveitada"`
	DecididoEm      time.Time     `json:"decidido_em"`
	Amostra         string        `json:"amostra,omitempty"`
	Caracteristicas []CaracResumo `json:"caracteristicas"`
}

// Conteudo é o que vai impresso — gravado como está no momento da emissão.
type Conteudo struct {
	Empresa     Parte        `json:"empresa"`
	Cliente     *Parte       `json:"cliente,omitempty"`
	ItemCode    int64        `json:"item_code"`
	ItemNome    string       `json:"item_nome"`
	Mask        string       `json:"mask,omitempty"`
	Lote        string       `json:"lote,omitempty"`
	Corrida     string       `json:"corrida,omitempty"`
	CertUsina   string       `json:"certificado_usina,omitempty"`
	Quantidade  string       `json:"quantidade"`
	NF          *NotaFiscal  `json:"nf,omitempty"`
	PedidoVenda *int64       `json:"pedido_venda,omitempty"`
	Lotes       []LoteResumo `json:"lotes"`
	Observacoes string       `json:"observacoes,omitempty"`
	Declaracao  string       `json:"declaracao"`
}

type Parte struct {
	Codigo int64  `json:"codigo,omitempty"`
	Nome   string `json:"nome"`
	CNPJ   string `json:"cnpj,omitempty"`
}

type NotaFiscal struct {
	Numero int64     `json:"numero"`
	Serie  string    `json:"serie"`
	Data   time.Time `json:"data"`
}

// Certificado emitido.
type Certificado struct {
	ID                 int64           `json:"id"`
	Numero             int64           `json:"numero"`
	CustomerCode       *int64          `json:"customer_code,omitempty"`
	FiscalExitID       *int64          `json:"fiscal_exit_id,omitempty"`
	SalesOrderCode     *int64          `json:"sales_order_code,omitempty"`
	ItemCode           int64           `json:"item_code"`
	Mask               string          `json:"mask"`
	Lote               *string         `json:"lote,omitempty"`
	Quantidade         decimal.Decimal `json:"quantidade"`
	Corrida            *string         `json:"corrida,omitempty"`
	CertificadoUsina   *string         `json:"certificado_usina,omitempty"`
	LotesInspecao      []int64         `json:"lotes_inspecao"`
	Conteudo           Conteudo        `json:"conteudo"`
	Observacoes        *string         `json:"observacoes,omitempty"`
	Status             string          `json:"status"`
	MotivoCancelamento *string         `json:"motivo_cancelamento,omitempty"`
	EmitidoEm          time.Time       `json:"emitido_em"`
	EmitidoPor         *uuid.UUID      `json:"emitido_por,omitempty"`
	CanceladoEm        *time.Time      `json:"cancelado_em,omitempty"`
}

func texto(d *decimal.Decimal) string {
	if d == nil {
		return ""
	}
	return d.String()
}

// Resumir calcula, por característica, n, mínimo, máximo, média e não
// conformes do lote, com os instrumentos usados (códigos).
func Resumir(l *lote.Lote, instrumentos map[int64]string) LoteResumo {
	r := LoteResumo{ID: l.ID, Numero: l.Numero, Origem: string(l.Origem)}
	if d := l.Decisao; d != nil {
		r.Decisao, r.Aproveitada, r.DecididoEm = d.Tipo, d.Aproveitada().String(), d.DecididoEm
		if d.Tratamento != nil {
			r.Tratamento = *d.Tratamento
			r.Concessao = *d.Tratamento == "CONCESSAO"
		}
	}
	if p := l.Plano; p != nil {
		r.Amostra = fmt.Sprintf("n=%d Ac=%d Re=%d", p.N, p.Ac, p.Re)
		if p.Letra != "" {
			r.Amostra = fmt.Sprintf("NBR 5426 letra %s, %s, %s", p.Letra, strings.ToLower(string(p.Severidade)), r.Amostra)
		}
	}
	if l.Pulado {
		r.Amostra = "liberado por skip-lot"
	}
	for _, c := range l.Caracteristicas {
		cr := CaracResumo{Nome: c.Nome, Tipo: string(c.Tipo), Nominal: texto(c.Nominal), Minimo: texto(c.Minimo), Maximo: texto(c.Maximo), Critica: c.Critica}
		if c.Unidade != nil {
			cr.Unidade = *c.Unidade
		}
		var min, max, soma *decimal.Decimal
		usados := map[string]bool{}
		for _, res := range l.Resultados {
			if res.CaracteristicaID != c.ID {
				continue
			}
			cr.N++
			if !res.Conforme {
				cr.NaoConformes++
			}
			if res.InstrumentoID != nil {
				if cod, ok := instrumentos[*res.InstrumentoID]; ok {
					usados[cod] = true
				}
			}
			if res.Valor == nil {
				continue
			}
			v := *res.Valor
			if min == nil || v.LessThan(*min) {
				min = &v
			}
			if max == nil || v.GreaterThan(*max) {
				max = &v
			}
			s := v
			if soma != nil {
				s = soma.Add(v)
			}
			soma = &s
		}
		if soma != nil {
			medidos := 0
			for _, res := range l.Resultados {
				if res.CaracteristicaID == c.ID && res.Valor != nil {
					medidos++
				}
			}
			media := soma.Div(decimal.NewFromInt(int64(medidos))).Round(6)
			cr.MedidoMinimo, cr.MedidoMaximo, cr.Media = min.String(), max.String(), media.String()
		}
		for cod := range usados {
			cr.Instrumentos = append(cr.Instrumentos, cod)
		}
		sort.Strings(cr.Instrumentos)
		r.Caracteristicas = append(r.Caracteristicas, cr)
	}
	return r
}

// ConferirLotes: lotes decididos do mesmo item com parte aproveitada, e a
// quantidade certificada — somada à dos certificados já emitidos sobre esses
// lotes — cabe no que eles aproveitaram. Certificado parcial (entregas
// separadas) é normal; certificar mais do que foi inspecionado, não.
func ConferirLotes(item int64, quantidade, jaCertificado decimal.Decimal, lotes []*lote.Lote) error {
	if len(lotes) == 0 {
		return domainerr.NewValidation("não há lote de inspeção decidido e aproveitado para este item/lote: o certificado declara uma inspeção que precisa existir")
	}
	total := decimal.Zero
	for _, l := range lotes {
		if l.ItemCode != item {
			return domainerr.NewValidation(fmt.Sprintf("o lote de inspeção %d é de outro item", l.Numero))
		}
		if l.Status != lote.Decidido || l.Decisao == nil || !l.Decisao.Aproveitada().IsPositive() {
			return domainerr.NewValidation(fmt.Sprintf("o lote de inspeção %d não foi aprovado para uso", l.Numero))
		}
		total = total.Add(l.Decisao.Aproveitada())
	}
	if !quantidade.IsPositive() {
		return domainerr.NewValidation("informe a quantidade certificada")
	}
	if quantidade.Add(jaCertificado).GreaterThan(total) {
		if jaCertificado.IsPositive() {
			return domainerr.NewValidation(fmt.Sprintf("a quantidade certificada (%s) mais a já certificada nesses lotes (%s) passa do que os lotes de inspeção aprovaram (%s)",
				quantidade, jaCertificado, total))
		}
		return domainerr.NewValidation(fmt.Sprintf("a quantidade certificada (%s) passa do que os lotes de inspeção aprovaram (%s)", quantidade, total))
	}
	return nil
}

// Declaracao padrão do certificado.
const Declaracao = "Declaramos que o material acima foi inspecionado conforme o plano de inspeção vigente e atende às especificações indicadas, " +
	"salvo o que estiver registrado como concessão."

// Fontes reúne o que o certificado lê de outros cadastros.
type Fontes interface {
	Empresa(ctx context.Context) (Parte, error)
	Cliente(ctx context.Context, codigo int64) (*Parte, error)
	Item(ctx context.Context, codigo int64) (string, error)
	NotaFiscal(ctx context.Context, exitID int64) (*NotaFiscal, *int64, error) // nota e cliente dela
	Corrida(ctx context.Context, item int64, lote string) (corrida, certificado string, err error)
	Instrumentos(ctx context.Context, ids []int64) (map[int64]string, error)
}

type Filtro struct {
	ItemCode     int64
	CustomerCode int64
	Status       string
}

type Repository interface {
	// Emitir grava sob trava e confere de novo, dentro dela, que a quantidade
	// mais a já certificada nos lotes não passa de limite (o aprovado).
	Emitir(ctx context.Context, c *Certificado, limite decimal.Decimal) error
	Obter(ctx context.Context, id int64) (*Certificado, error)
	Listar(ctx context.Context, f Filtro) ([]Certificado, error)
	Cancelar(ctx context.Context, id int64, motivo string, ator uuid.UUID) error
	// JaCertificado soma os certificados emitidos (não cancelados) que usam
	// algum desses lotes de inspeção.
	JaCertificado(ctx context.Context, lotes []int64) (decimal.Decimal, error)
}
