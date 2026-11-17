package entity

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/pkg/validation"
)

// Saneamento do cadastro de clientes (VCLI0510): o que no cadastro faz a NF-e
// ser rejeitada, o pedido travar ou a análise de crédito errar. Só clientes
// ativos; cada pendência diz como resolver.

// SaneamentoCliente é o que o relatório lê de cada cliente ativo.
type SaneamentoCliente struct {
	Code                  int64
	Name                  string
	DocumentType          DocumentType
	DocumentNumber        string
	StateRegistration     *string
	SuframaCode           *string
	SuframaExpiry         *time.Time
	UF                    string // do endereço principal; vazio = sem endereço
	NFeEmail              *string
	RegistrationStatus    *string
	RegistrationCheckedAt *time.Time
	Blocked               bool
	CreditLimit           decimal.Decimal
	CreditLimitExpiresAt  *time.Time
	LastReviewOrigin      string // origem da última análise de crédito ("" = nenhuma)
}

// Gravidade: IMPEDE = a nota é rejeitada ou o pedido trava; ATENCAO = vale
// corrigir, mas não para nada hoje.
const (
	SaneamentoImpede  = "IMPEDE"
	SaneamentoAtencao = "ATENCAO"
)

type PendenciaCadastro struct {
	Categoria    string
	Gravidade    string
	Detalhe      string
	ComoResolver string
}

// CategoriasSaneamento: título de cada categoria, na ordem do relatório.
var CategoriasSaneamento = []struct{ Code, Titulo string }{
	{"DOCUMENTO_INVALIDO", "CNPJ/CPF com dígito errado"},
	{"IE_INVALIDA", "Inscrição estadual inválida para a UF"},
	{"SEM_ENDERECO", "Sem endereço"},
	{"SUFRAMA_INVALIDA", "SUFRAMA com dígito errado"},
	{"SUFRAMA_VENCIDA", "SUFRAMA vencida"},
	{"RECEITA_IRREGULAR", "CNPJ irregular na Receita e não bloqueado"},
	{"RECEITA_NAO_CONFERIDA", "CNPJ sem conferência recente na Receita"},
	{"LIMITE_SEM_ANALISE", "Limite antigo, nunca analisado"},
	{"LIMITE_VENCIDO", "Limite de crédito vencido"},
	{"LIMITE_VENCE_30_DIAS", "Limite vence em até 30 dias"},
	{"SEM_EMAIL_NFE", "Sem e-mail para a NF-e"},
}

// PendenciasDoCliente aponta o que está errado no cadastro de um cliente.
func PendenciasDoCliente(c SaneamentoCliente, pol CreditPolicy, today time.Time) []PendenciaCadastro {
	var out []PendenciaCadastro
	add := func(cat, grav, det, como string) {
		out = append(out, PendenciaCadastro{Categoria: cat, Gravidade: grav, Detalhe: det, ComoResolver: como})
	}
	doc := c.DocumentNumber
	switch c.DocumentType {
	case DocumentCNPJ:
		if !validation.ValidateCNPJ(doc) {
			add("DOCUMENTO_INVALIDO", SaneamentoImpede, "CNPJ "+doc, "corrija o CNPJ na VCLI0500: a SEFAZ rejeita a NF-e e a Receita não confere")
		}
	case DocumentCPF:
		if !validation.ValidateCPF(doc) {
			add("DOCUMENTO_INVALIDO", SaneamentoImpede, "CPF "+doc, "corrija o CPF na VCLI0500: a SEFAZ rejeita a NF-e")
		}
	}
	if c.UF == "" {
		add("SEM_ENDERECO", SaneamentoImpede, "nenhum endereço cadastrado", "inclua o endereço na VCLI0500 → Endereços: a NF-e exige o endereço do destinatário")
	}
	if ie := strings.TrimSpace(deref(c.StateRegistration)); ie != "" && !strings.EqualFold(ie, "ISENTO") && c.UF != "" &&
		!validation.IEValida(ie, c.UF) {
		add("IE_INVALIDA", SaneamentoImpede, fmt.Sprintf("IE %s não confere para %s", ie, c.UF),
			"confira a IE (ou a UF do endereço principal) na VCLI0500: a SEFAZ rejeita a NF-e")
	}
	if isuf := strings.TrimSpace(deref(c.SuframaCode)); isuf != "" {
		switch {
		case !validation.SuframaValida(isuf):
			add("SUFRAMA_INVALIDA", SaneamentoImpede, "SUFRAMA "+isuf,
				"corrija a inscrição na VCLI0500 → Fiscal: com o dígito errado ela não vai na nota e a venda para a Zona Franca sai tributada")
		case c.SuframaExpiry != nil && c.SuframaExpiry.Before(today):
			add("SUFRAMA_VENCIDA", SaneamentoImpede, fmt.Sprintf("SUFRAMA %s vencida em %s", isuf, c.SuframaExpiry.Format("02/01/2006")),
				"atualize a validade na VCLI0500 → Fiscal: vencida, a venda para a Zona Franca sai tributada")
		}
	}
	if c.DocumentType == DocumentCNPJ {
		status := strings.TrimSpace(deref(c.RegistrationStatus))
		switch {
		case status != "" && !strings.EqualFold(status, "ATIVA") && !c.Blocked:
			add("RECEITA_IRREGULAR", SaneamentoImpede, "situação na Receita: "+status,
				"bloqueie o cliente ou reconsulte na VCLI0500 (Consultar CNPJ): CNPJ irregular zera o crédito")
		case c.RegistrationCheckedAt == nil || c.RegistrationCheckedAt.Before(today.AddDate(0, 0, -2*pol.RegistrationCheckDays)):
			quando := "nunca conferido"
			if c.RegistrationCheckedAt != nil {
				quando = "última conferência em " + c.RegistrationCheckedAt.Format("02/01/2006")
			}
			add("RECEITA_NAO_CONFERIDA", SaneamentoAtencao, quando,
				"o agendador confere aos poucos; para adiantar, use Consultar CNPJ na VCLI0500")
		}
	}
	if c.CreditLimit.IsPositive() {
		switch {
		case c.CreditLimitExpiresAt != nil && c.CreditLimitExpiresAt.Before(today):
			add("LIMITE_VENCIDO", SaneamentoImpede, "vencido em "+c.CreditLimitExpiresAt.Format("02/01/2006"),
				"revise o limite na VCLI0500 → Crédito: com ele vencido, todo pedido a prazo vai para a análise de crédito")
		case c.LastReviewOrigin == "LEGADO":
			add("LIMITE_SEM_ANALISE", SaneamentoAtencao, "limite de "+Reais(c.CreditLimit)+" digitado antes da política",
				"faça a análise na VCLI0500 → Crédito (sem histórico de pagamento a revisão automática não o mexe)")
		case c.CreditLimitExpiresAt != nil && !c.CreditLimitExpiresAt.After(today.AddDate(0, 0, 30)):
			add("LIMITE_VENCE_30_DIAS", SaneamentoAtencao, "vence em "+c.CreditLimitExpiresAt.Format("02/01/2006"),
				"a revisão automática renova antes de vencer; se ela estiver desligada na política, revise na VCLI0500 → Crédito")
		}
	}
	if strings.TrimSpace(deref(c.NFeEmail)) == "" {
		add("SEM_EMAIL_NFE", SaneamentoAtencao, "sem e-mail de NF-e", "informe o e-mail na VCLI0500 → Fiscal para o cliente receber o XML e o DANFE")
	}
	return out
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
