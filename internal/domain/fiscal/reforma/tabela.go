package reforma

import (
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ClassificacaoOficial é uma linha da tabela nacional de classificação
// tributária do IBS/CBS (cClassTrib), com o CST a que pertence.
type ClassificacaoOficial struct {
	Classificacao
	CSTDescricao      string
	Descricao         string
	ReducaoAliquota   bool
	Diferimento       bool
	TipoAliquota      int
	TributacaoRegular bool
	AceitaNFe         bool
	VigenciaInicio    time.Time
	VigenciaFim       *time.Time
	URLLegislacao     *string
	VersaoTabela      string
}

// cstsSuportados são os CST cujo grupo o emissor monta por completo: o grupo
// padrão (gIBSCBS), a redução de alíquota (gRed) ou nenhum grupo de valores. Os
// demais exigem grupos que o emissor ainda não preenche (IT 2025.002, tabela
// CST: ind_gDif, ind_gIBSCBSMono, ind_gTransfCred, ind_gAjusteCompet,
// ind_gCredPresIBSZFM, ind_RedutorBC, alíquota fixa) — a nota sairia
// incompleta e a SEFAZ rejeitaria.
var cstsSuportados = map[string]bool{"000": true, "010": true, "011": true, "200": true, "400": true, "410": true, "550": true}

// CSTSuportadoNaEmissao diz se o emissor monta o grupo do CST.
func CSTSuportadoNaEmissao(cst string) bool { return cstsSuportados[cst] }

// ValidarParaNFe recusa a classificação que a SEFAZ rejeitaria na NF-e da data:
// fora da vigência, não admitida no modelo 55 ou com grupo que o emissor não monta.
func (c ClassificacaoOficial) ValidarParaNFe(emissao time.Time) error {
	if !c.AceitaNFe {
		return domainerr.NewValidation(fmt.Sprintf("o cClassTrib %s (%s) não é admitido na NF-e", c.ClassTrib, c.Descricao))
	}
	if !CSTSuportadoNaEmissao(c.CST) {
		return domainerr.NewValidation(fmt.Sprintf(
			"o CST %s (%s) exige grupos do IBS/CBS que a emissão ainda não preenche (diferimento, monofásico, crédito, ajuste ou alíquota fixa): use um cClassTrib de CST 000, 010, 011, 200, 400, 410 ou 550",
			c.CST, c.CSTDescricao))
	}
	dia := emissao.Truncate(24 * time.Hour)
	if dia.Before(c.VigenciaInicio) {
		return domainerr.NewValidation(fmt.Sprintf("o cClassTrib %s só vale a partir de %s", c.ClassTrib, c.VigenciaInicio.Format("02/01/2006")))
	}
	if c.VigenciaFim != nil && dia.After(*c.VigenciaFim) {
		return domainerr.NewValidation(fmt.Sprintf("o cClassTrib %s deixou de valer em %s", c.ClassTrib, c.VigenciaFim.Format("02/01/2006")))
	}
	return nil
}

// AliquotasCadastradas são as alíquotas que a empresa gravou para um ano.
type AliquotasCadastradas struct {
	Ano        int
	Aliquotas  Aliquotas
	Fundamento string
	UpdatedAt  time.Time
	UpdatedBy  uuid.UUID
}

// ValidarCadastro: percentuais entre 0 e 100 e fundamento informado (a
// alíquota da CBS de 2027 é ato da União; quem cadastra diz qual).
func ValidarCadastro(ano int, a Aliquotas, fundamento string) error {
	if ano < 2026 || ano > 2100 {
		return domainerr.NewValidation("o ano das alíquotas vai de 2026 em diante")
	}
	cem := decimal.NewFromInt(100)
	for nome, v := range map[string]decimal.Decimal{"IBS estadual": a.IBSUF, "IBS municipal": a.IBSMun, "CBS": a.CBS} {
		if v.IsNegative() || !v.LessThan(cem) {
			return domainerr.NewValidation(fmt.Sprintf("a alíquota do %s deve ficar entre 0 e 100%%", nome))
		}
	}
	if len([]rune(fundamento)) < 3 {
		return domainerr.NewValidation("informe o fundamento das alíquotas (lei, resolução ou ato que as fixou)")
	}
	return nil
}

// CFOPClassificacao é o cClassTrib padrão de um CFOP na empresa.
type CFOPClassificacao struct {
	CFOP       string
	ClassTrib  string
	Descricao  string
	Observacao *string
	UpdatedAt  time.Time
	UpdatedBy  uuid.UUID
}

// Candidatos são os cClassTrib que podem tratar uma linha da nota, um por
// origem; vazio quando a origem não informa.
type Candidatos struct {
	DoItem, DoCFOP, DaClassificacao string
}

// RetratoItem é o grupo IBS/CBS transmitido em uma linha da nota.
type RetratoItem struct {
	ItemID int64
	Origem string
	Grupo  Grupo
}

// Modo de convivência dos dois modelos na NF-e de saída da empresa.
type Modo string

const (
	// ModoAutomatico: modelo atual + grupo IBS/CBS do regime normal (padrão).
	ModoAutomatico Modo = "AUTOMATICO"
	// ModoSomenteAtual: só o modelo atual (ICMS, IPI, PIS/COFINS), sem o grupo.
	ModoSomenteAtual Modo = "SOMENTE_ATUAL"
)

// Parametros é o modo da empresa e quem o definiu.
type Parametros struct {
	Modo      Modo
	Motivo    *string
	UpdatedAt *time.Time
	UpdatedBy *uuid.UUID
}
