// Package metrologia: instrumentos de medição, calibrações e a regra de quem
// pode medir — instrumento ativo, do grupo pedido pela característica e com a
// calibração em dia.
package metrologia

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

type Instrumento struct {
	ID                int64            `json:"id"`
	Codigo            string           `json:"codigo"`
	Descricao         string           `json:"descricao"`
	Grupo             string           `json:"grupo"`
	Fabricante        *string          `json:"fabricante,omitempty"`
	NumeroSerie       *string          `json:"numero_serie,omitempty"`
	Faixa             *string          `json:"faixa,omitempty"`
	Resolucao         *decimal.Decimal `json:"resolucao,omitempty"`
	EMA               *decimal.Decimal `json:"ema,omitempty"`
	Unidade           *string          `json:"unidade,omitempty"`
	Localizacao       *string          `json:"localizacao,omitempty"`
	FrequenciaDias    int              `json:"frequencia_dias"`
	Status            string           `json:"status"`
	MotivoStatus      *string          `json:"motivo_status,omitempty"`
	UltimaCalibracao  *time.Time       `json:"ultima_calibracao,omitempty"`
	ProximaCalibracao *time.Time       `json:"proxima_calibracao,omitempty"`
	UpdatedAt         time.Time        `json:"updated_at"`
	UpdatedBy         *uuid.UUID       `json:"updated_by,omitempty"`
}

type Calibracao struct {
	ID             int64            `json:"id"`
	InstrumentoID  int64            `json:"instrument_id"`
	Data           time.Time        `json:"data"`
	Executor       string           `json:"executor"`
	Laboratorio    *string          `json:"laboratorio,omitempty"`
	Certificado    *string          `json:"certificado,omitempty"`
	ErroEncontrado *decimal.Decimal `json:"erro_encontrado,omitempty"`
	Incerteza      *decimal.Decimal `json:"incerteza,omitempty"`
	EMA            *decimal.Decimal `json:"ema,omitempty"`
	Resultado      string           `json:"resultado"`
	Ajustado       bool             `json:"ajustado"`
	Proxima        *time.Time       `json:"proxima,omitempty"`
	Observacoes    *string          `json:"observacoes,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	CreatedBy      *uuid.UUID       `json:"created_by,omitempty"`
}

// NormalizarGrupo: maiúsculas, sem acento e espaços colapsados — "Paquímetro"
// e "PAQUIMETRO" são o mesmo grupo.
func NormalizarGrupo(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, _ := transform.String(t, s)
	return strings.ToUpper(strings.Join(strings.Fields(out), " "))
}

// Validar confere o cadastro.
func (i *Instrumento) Validar() error {
	i.Codigo = strings.ToUpper(strings.TrimSpace(i.Codigo))
	i.Descricao = strings.TrimSpace(i.Descricao)
	i.Grupo = NormalizarGrupo(i.Grupo)
	switch {
	case i.Codigo == "" || len(i.Codigo) > 30:
		return domainerr.NewValidation("informe o código do instrumento (até 30 caracteres)")
	case i.Descricao == "":
		return domainerr.NewValidation("informe a descrição do instrumento")
	case i.Grupo == "":
		return domainerr.NewValidation("informe o grupo do instrumento (o mesmo do roteiro de inspeção, ex.: PAQUIMETRO)")
	case i.FrequenciaDias <= 0:
		return domainerr.NewValidation("informe a frequência de calibração em dias")
	case i.EMA != nil && !i.EMA.IsPositive():
		return domainerr.NewValidation("o erro máximo admissível precisa ser maior que zero")
	case i.Resolucao != nil && !i.Resolucao.IsPositive():
		return domainerr.NewValidation("a resolução precisa ser maior que zero")
	}
	if i.Status == "" {
		i.Status = "ATIVO"
	}
	return nil
}

func dia(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }

// Avaliar decide o resultado da calibração e a próxima data.
//
// Com erro encontrado e EMA, o servidor decide: aprovado se |erro| + incerteza
// ≤ EMA. Somar a incerteza é a regra conservadora (zona de guarda): um erro no
// limite com incerteza grande não aprova. Sem os números, vale o resultado
// informado pelo laboratório. Aprovada, a próxima é data + frequência.
func (c *Calibracao) Avaliar(i *Instrumento, hoje time.Time) error {
	if c.Data.IsZero() {
		return domainerr.NewValidation("informe a data da calibração")
	}
	if dia(c.Data).After(dia(hoje)) {
		return domainerr.NewValidation("a data da calibração não pode ser futura")
	}
	if c.Executor != "INTERNO" && c.Executor != "EXTERNO" {
		return domainerr.NewValidation("informe quem calibrou (INTERNO ou EXTERNO)")
	}
	if c.Executor == "EXTERNO" && (c.Certificado == nil || strings.TrimSpace(*c.Certificado) == "") {
		return domainerr.NewValidation("calibração externa precisa do número do certificado")
	}
	if c.EMA == nil {
		c.EMA = i.EMA
	}
	if c.Incerteza != nil && c.Incerteza.IsNegative() {
		return domainerr.NewValidation("a incerteza não pode ser negativa")
	}
	if c.ErroEncontrado != nil && c.EMA != nil {
		u := decimal.Zero
		if c.Incerteza != nil {
			u = *c.Incerteza
		}
		if c.ErroEncontrado.Abs().Add(u).LessThanOrEqual(*c.EMA) {
			c.Resultado = "APROVADO"
		} else {
			c.Resultado = "REPROVADO"
		}
	} else if c.Resultado != "APROVADO" && c.Resultado != "REPROVADO" {
		return domainerr.NewValidation("sem erro encontrado e erro máximo admissível, informe o resultado (APROVADO ou REPROVADO)")
	}
	c.Proxima = nil
	if c.Resultado == "APROVADO" {
		p := dia(c.Data).AddDate(0, 0, i.FrequenciaDias)
		c.Proxima = &p
	}
	return nil
}

// Aplicar leva o resultado ao instrumento: aprovada libera (e reativa o
// bloqueado), reprovada bloqueia. Calibração mais antiga que a última não
// mexe nas datas.
func (i *Instrumento) Aplicar(c *Calibracao) {
	if i.UltimaCalibracao != nil && dia(c.Data).Before(dia(*i.UltimaCalibracao)) {
		return
	}
	d := dia(c.Data)
	i.UltimaCalibracao = &d
	if c.Resultado == "APROVADO" {
		i.ProximaCalibracao = c.Proxima
		if i.Status == "BLOQUEADO" {
			i.Status, i.MotivoStatus = "ATIVO", nil
		}
		return
	}
	i.ProximaCalibracao = nil
	if i.Status != "BAIXADO" {
		m := fmt.Sprintf("calibração reprovada em %s", d.Format("02/01/2006"))
		i.Status, i.MotivoStatus = "BLOQUEADO", &m
	}
}

// PodeMedir: ativo, calibração em dia e do grupo pedido (quando a
// característica pede um).
func (i *Instrumento) PodeMedir(grupoExigido string, hoje time.Time) error {
	if i.Status != "ATIVO" {
		motivo := ""
		if i.MotivoStatus != nil {
			motivo = ": " + *i.MotivoStatus
		}
		return domainerr.NewValidation(fmt.Sprintf("o instrumento %s está %s%s", i.Codigo, strings.ToLower(i.Status), motivo))
	}
	if i.ProximaCalibracao == nil {
		return domainerr.NewValidation(fmt.Sprintf("o instrumento %s não tem calibração aprovada", i.Codigo))
	}
	if dia(*i.ProximaCalibracao).Before(dia(hoje)) {
		return domainerr.NewValidation(fmt.Sprintf("a calibração do instrumento %s venceu em %s", i.Codigo, i.ProximaCalibracao.Format("02/01/2006")))
	}
	if g := NormalizarGrupo(grupoExigido); g != "" && g != i.Grupo {
		return domainerr.NewValidation(fmt.Sprintf("a característica pede instrumento do grupo %s e o %s é do grupo %s", g, i.Codigo, i.Grupo))
	}
	return nil
}

// Impacto: lote medido com o instrumento depois da última calibração aprovada.
type Impacto struct {
	LoteID     int64     `json:"lote_id"`
	LoteNumero int64     `json:"lote_numero"`
	ItemCode   int64     `json:"item_code"`
	Status     string    `json:"status"`
	Decisao    *string   `json:"decisao,omitempty"`
	Medicoes   int       `json:"medicoes"`
	UltimaEm   time.Time `json:"ultima_medicao"`
}

// Filtro da lista de instrumentos.
type Filtro struct {
	Grupo     string
	Situacao  string // VENCIDOS, A_VENCER, BLOQUEADOS
	Dias      int
	Hoje      time.Time
	Inclusive bool // inclui baixados
}

type Repository interface {
	Salvar(ctx context.Context, i *Instrumento) error
	Obter(ctx context.Context, id int64) (*Instrumento, error)
	Listar(ctx context.Context, f Filtro) ([]Instrumento, error)
	// RegistrarCalibracao grava a calibração e aplica no instrumento (travado).
	RegistrarCalibracao(ctx context.Context, instrumentoID int64, c *Calibracao, hoje time.Time) (*Instrumento, error)
	Calibracoes(ctx context.Context, instrumentoID int64) ([]Calibracao, error)
	// Impacto: lotes com resultado medido pelo instrumento desde a data.
	Impacto(ctx context.Context, instrumentoID int64, desde time.Time) ([]Impacto, error)
	PorIDs(ctx context.Context, ids []int64) (map[int64]Instrumento, error)
}
