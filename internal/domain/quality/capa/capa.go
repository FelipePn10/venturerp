// Package capa é o tratamento da não conformidade: ação corretiva/preventiva
// pelo método 8D (D0 a D8) ou simples (análise → ação → verificação), com
// causa raiz por 5 porquês ou Ishikawa, ações com responsável e prazo e
// verificação de eficácia.
package capa

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

type Metodo string

const (
	Metodo8D      Metodo = "8D"
	MetodoSimples Metodo = "SIMPLES"
)

// Etapas do 8D e do método simples, na ordem.
var (
	etapas8D      = []string{"D0", "D1", "D2", "D3", "D4", "D5", "D6", "D7", "D8"}
	etapasSimples = []string{"ANALISE", "ACAO", "VERIFICACAO"}
)

// NomeDaEtapa para a tela e o histórico.
var NomeDaEtapa = map[string]string{
	"D0": "D0 — Preparação e ação de emergência", "D1": "D1 — Equipe", "D2": "D2 — Descrição do problema",
	"D3": "D3 — Contenção", "D4": "D4 — Causa raiz e ponto de escape", "D5": "D5 — Ações corretivas permanentes",
	"D6": "D6 — Implementação e validação", "D7": "D7 — Prevenção da recorrência", "D8": "D8 — Encerramento e eficácia",
	"ANALISE": "Análise da causa", "ACAO": "Ação corretiva", "VERIFICACAO": "Verificação de eficácia",
}

func (m Metodo) etapas() []string {
	if m == MetodoSimples {
		return etapasSimples
	}
	return etapas8D
}

// PrimeiraEtapa do método.
func (m Metodo) PrimeiraEtapa() string { return m.etapas()[0] }

// EtapaDeCausa é para onde volta a CAPA cuja ação foi ineficaz.
func (m Metodo) EtapaDeCausa() string {
	if m == MetodoSimples {
		return "ANALISE"
	}
	return "D4"
}

// EtapaFinal: onde se verifica a eficácia e se encerra.
func (m Metodo) EtapaFinal() string {
	e := m.etapas()
	return e[len(e)-1]
}

func (m Metodo) indice(etapa string) int {
	for i, e := range m.etapas() {
		if e == etapa {
			return i
		}
	}
	return -1
}

// Depois diz se a etapa a vem depois (ou é) da b.
func (m Metodo) Depois(a, b string) bool { return m.indice(a) >= m.indice(b) }

// Problema é a descrição 5W2H (D2).
type Problema struct {
	OQue           string `json:"o_que"`
	Onde           string `json:"onde"`
	Quando         string `json:"quando"`
	Quem           string `json:"quem"`
	Quanto         string `json:"quanto"`
	Como           string `json:"como"`
	PorQueDetectou string `json:"por_que_detectou"`
}

// CategoriasIshikawa: os 6M.
var CategoriasIshikawa = []string{"MAO_DE_OBRA", "METODO", "MATERIAL", "MAQUINA", "MEDICAO", "MEIO_AMBIENTE"}

type Status string

const (
	Aberta    Status = "ABERTA"
	Encerrada Status = "ENCERRADA"
	Cancelada Status = "CANCELADA"
)

// Capa é o tratamento.
type Capa struct {
	ID                  int64               `json:"id"`
	Numero              int64               `json:"numero"`
	Metodo              Metodo              `json:"metodo"`
	Tipo                string              `json:"tipo"`
	Origem              string              `json:"origem"`
	Titulo              string              `json:"titulo"`
	ItemCode            *int64              `json:"item_code,omitempty"`
	SupplierCode        *int64              `json:"supplier_code,omitempty"`
	CustomerCode        *int64              `json:"customer_code,omitempty"`
	Referencia          *string             `json:"referencia,omitempty"`
	Responsavel         uuid.UUID           `json:"responsavel"`
	Prazo               *time.Time          `json:"prazo,omitempty"`
	Etapa               string              `json:"etapa"`
	Status              Status              `json:"status"`
	Problema            Problema            `json:"problema"`
	ContencaoDispensada *string             `json:"contencao_dispensada,omitempty"`
	MetodoCausa         *string             `json:"metodo_causa,omitempty"`
	CincoPorques        []string            `json:"cinco_porques"`
	Ishikawa            map[string][]string `json:"ishikawa"`
	CausaRaiz           *string             `json:"causa_raiz,omitempty"`
	PontoDeEscape       *string             `json:"ponto_de_escape,omitempty"`
	Prevencao           *string             `json:"prevencao,omitempty"`
	Eficaz              *bool               `json:"eficaz,omitempty"`
	Verificacao         *string             `json:"verificacao,omitempty"`
	Licoes              *string             `json:"licoes,omitempty"`
	Ciclo               int                 `json:"ciclo"`
	CreatedAt           time.Time           `json:"created_at"`
	CreatedBy           *uuid.UUID          `json:"created_by,omitempty"`
	UpdatedAt           time.Time           `json:"updated_at"`
	ClosedAt            *time.Time          `json:"closed_at,omitempty"`
	ClosedBy            *uuid.UUID          `json:"closed_by,omitempty"`
	NCs                 []int64             `json:"ncs"`
	Equipe              []Membro            `json:"equipe"`
	Acoes               []Acao              `json:"acoes"`
	Eventos             []Evento            `json:"eventos,omitempty"`
}

type Membro struct {
	UserID uuid.UUID `json:"user_id"`
	Nome   string    `json:"nome,omitempty"`
	Papel  string    `json:"papel"`
}

type Acao struct {
	ID           int64      `json:"id"`
	Tipo         string     `json:"tipo"`
	Descricao    string     `json:"descricao"`
	Responsavel  uuid.UUID  `json:"responsavel"`
	Prazo        time.Time  `json:"prazo"`
	Status       string     `json:"status"`
	Evidencia    *string    `json:"evidencia,omitempty"`
	ConcluidaEm  *time.Time `json:"concluida_em,omitempty"`
	ConcluidaPor *uuid.UUID `json:"concluida_por,omitempty"`
	Ciclo        int        `json:"ciclo"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Evento struct {
	ID        int64      `json:"id"`
	Evento    string     `json:"evento"`
	De        *string    `json:"de,omitempty"`
	Para      *string    `json:"para,omitempty"`
	Detalhe   *string    `json:"detalhe,omitempty"`
	Ator      *uuid.UUID `json:"ator,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

var origens = map[string]bool{"NC": true, "RECLAMACAO_CLIENTE": true, "FORNECEDOR": true, "AUDITORIA": true, "INTERNA": true}

// ValidarAbertura confere a CAPA nova.
func (c *Capa) ValidarAbertura() error {
	c.Titulo = strings.TrimSpace(c.Titulo)
	if c.Titulo == "" {
		return domainerr.NewValidation("informe o título da CAPA")
	}
	if c.Metodo != Metodo8D && c.Metodo != MetodoSimples {
		return domainerr.NewValidation("método inválido (8D ou SIMPLES)")
	}
	if c.Tipo == "" {
		c.Tipo = "CORRETIVA"
	}
	if c.Tipo != "CORRETIVA" && c.Tipo != "PREVENTIVA" {
		return domainerr.NewValidation("tipo inválido (CORRETIVA ou PREVENTIVA)")
	}
	if !origens[c.Origem] {
		return domainerr.NewValidation("origem inválida (NC, RECLAMACAO_CLIENTE, FORNECEDOR, AUDITORIA ou INTERNA)")
	}
	if c.Origem == "NC" && len(c.NCs) == 0 {
		return domainerr.NewValidation("CAPA de origem NC precisa de ao menos uma não conformidade")
	}
	if c.Origem == "FORNECEDOR" && c.SupplierCode == nil {
		return domainerr.NewValidation("CAPA de fornecedor precisa do fornecedor")
	}
	if c.Origem == "RECLAMACAO_CLIENTE" && c.CustomerCode == nil {
		return domainerr.NewValidation("CAPA de reclamação de cliente precisa do cliente")
	}
	if c.Responsavel == uuid.Nil {
		return domainerr.NewValidation("informe o responsável pela CAPA")
	}
	c.Etapa, c.Status, c.Ciclo = c.Metodo.PrimeiraEtapa(), Aberta, 1
	if c.CincoPorques == nil {
		c.CincoPorques = []string{}
	}
	if c.Ishikawa == nil {
		c.Ishikawa = map[string][]string{}
	}
	return nil
}

func vazio(s *string) bool { return s == nil || strings.TrimSpace(*s) == "" }

// acoesDoCiclo filtra as ações não canceladas do ciclo corrente.
func (c *Capa) acoesDoCiclo(tipo string) []Acao {
	var out []Acao
	for _, a := range c.Acoes {
		if a.Tipo == tipo && a.Ciclo == c.Ciclo && a.Status != "CANCELADA" {
			out = append(out, a)
		}
	}
	return out
}

// Pendencias diz o que falta para sair da etapa corrente.
func (c *Capa) Pendencias() []string {
	var p []string
	falta := func(cond bool, msg string) {
		if cond {
			p = append(p, msg)
		}
	}
	causa := func() {
		falta(c.MetodoCausa == nil, "escolha o método da causa raiz (5 porquês ou Ishikawa)")
		if c.MetodoCausa != nil && *c.MetodoCausa == "5PORQUES" {
			n := 0
			for _, x := range c.CincoPorques {
				if strings.TrimSpace(x) != "" {
					n++
				}
			}
			falta(n < 3, "responda ao menos 3 porquês")
		}
		if c.MetodoCausa != nil && *c.MetodoCausa == "ISHIKAWA" {
			n := 0
			for _, xs := range c.Ishikawa {
				for _, x := range xs {
					if strings.TrimSpace(x) != "" {
						n++
					}
				}
			}
			falta(n == 0, "registre ao menos uma causa no diagrama de Ishikawa")
		}
		falta(vazio(c.CausaRaiz), "escreva a causa raiz")
	}
	concluidas := func(tipo, nome string) {
		for _, a := range c.acoesDoCiclo(tipo) {
			falta(a.Status != "CONCLUIDA", fmt.Sprintf("conclua a ação %s \"%s\" (com evidência)", nome, a.Descricao))
		}
	}
	switch c.Etapa {
	case "D0":
		// Preparação: título e responsável já exigidos na abertura.
	case "D1":
		lider := false
		for _, m := range c.Equipe {
			if m.Papel == "LIDER" {
				lider = true
			}
		}
		falta(!lider, "indique o líder da equipe")
		falta(len(c.Equipe) < 2, "a equipe do 8D precisa de ao menos 2 pessoas")
	case "D2":
		falta(strings.TrimSpace(c.Problema.OQue) == "", "descreva o problema (o quê)")
		falta(strings.TrimSpace(c.Problema.Onde) == "", "diga onde o problema aparece")
		falta(strings.TrimSpace(c.Problema.Quando) == "", "diga quando o problema aparece")
		falta(strings.TrimSpace(c.Problema.Quanto) == "", "diga quanto (quantidade, frequência)")
	case "D3":
		falta(len(c.acoesDoCiclo("CONTENCAO")) == 0 && vazio(c.ContencaoDispensada),
			"registre ao menos uma ação de contenção ou justifique por que não é necessária")
	case "D4":
		causa()
		falta(vazio(c.PontoDeEscape), "diga o ponto de escape: por que o controle não detectou")
	case "D5":
		falta(len(c.acoesDoCiclo("CORRETIVA")) == 0, "registre ao menos uma ação corretiva permanente")
	case "D6":
		falta(len(c.acoesDoCiclo("CORRETIVA")) == 0, "registre ao menos uma ação corretiva permanente")
		concluidas("CORRETIVA", "corretiva")
		concluidas("CONTENCAO", "de contenção")
	case "D7":
		falta(vazio(c.Prevencao), "diga o que muda para não acontecer de novo (plano de inspeção, IT, dispositivo à prova de erro…)")
		concluidas("PREVENTIVA", "preventiva")
	case "ANALISE":
		falta(strings.TrimSpace(c.Problema.OQue) == "", "descreva o problema (o quê)")
		causa()
	case "ACAO":
		falta(len(c.acoesDoCiclo("CORRETIVA")) == 0, "registre ao menos uma ação corretiva")
		concluidas("CORRETIVA", "corretiva")
	}
	return p
}

// Avancar leva a CAPA à próxima etapa, se a corrente está completa.
func (c *Capa) Avancar() (de, para string, err error) {
	if c.Status != Aberta {
		return "", "", domainerr.NewValidation("a CAPA não está aberta")
	}
	if c.Etapa == c.Metodo.EtapaFinal() {
		return "", "", domainerr.NewValidation("esta é a etapa de verificação: encerre a CAPA informando se a ação foi eficaz")
	}
	if p := c.Pendencias(); len(p) > 0 {
		return "", "", domainerr.NewValidation("para concluir " + NomeDaEtapa[c.Etapa] + ": " + strings.Join(p, "; "))
	}
	i := c.Metodo.indice(c.Etapa)
	de, para = c.Etapa, c.Metodo.etapas()[i+1]
	c.Etapa = para
	return de, para, nil
}

// Encerrar registra a verificação de eficácia. Eficaz encerra; ineficaz volta
// à causa raiz num ciclo novo (as ações do ciclo anterior ficam no histórico).
func (c *Capa) Encerrar(eficaz bool, verificacao string, licoes *string, ator uuid.UUID, agora time.Time) (encerrou bool, err error) {
	if c.Status != Aberta {
		return false, domainerr.NewValidation("a CAPA não está aberta")
	}
	if c.Etapa != c.Metodo.EtapaFinal() {
		return false, domainerr.NewValidation("a verificação de eficácia é a última etapa: conclua as anteriores")
	}
	if strings.TrimSpace(verificacao) == "" {
		return false, domainerr.NewValidation("descreva como a eficácia foi verificada (lotes seguintes, indicadores, auditoria)")
	}
	v := strings.TrimSpace(verificacao)
	c.Verificacao, c.Eficaz = &v, &eficaz
	if !eficaz {
		c.Etapa = c.Metodo.EtapaDeCausa()
		c.Ciclo++
		c.CausaRaiz, c.PontoDeEscape = nil, nil
		return false, nil
	}
	c.Status, c.Licoes, c.ClosedAt, c.ClosedBy = Encerrada, licoes, &agora, &ator
	return true, nil
}

// ValidarAcao confere a ação nova.
func ValidarAcao(a *Acao) error {
	a.Descricao = strings.TrimSpace(a.Descricao)
	if a.Tipo != "CONTENCAO" && a.Tipo != "CORRETIVA" && a.Tipo != "PREVENTIVA" {
		return domainerr.NewValidation("tipo de ação inválido (CONTENCAO, CORRETIVA ou PREVENTIVA)")
	}
	if a.Descricao == "" {
		return domainerr.NewValidation("descreva a ação")
	}
	if a.Responsavel == uuid.Nil {
		return domainerr.NewValidation("informe o responsável pela ação")
	}
	if a.Prazo.IsZero() {
		return domainerr.NewValidation("informe o prazo da ação")
	}
	return nil
}
