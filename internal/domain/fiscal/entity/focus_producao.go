package entity

import (
	"strings"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// AmbienteProducao: com ele a Focus transmite para a SEFAZ de verdade.
const AmbienteProducao = "producao"

// MsgTokenProducaoPelaVenture: o que a tela mostra ao tentar mexer no token
// de produção.
const MsgTokenProducaoPelaVenture = "o token Focus NF-e de produção só é gravado ou trocado pela Venture ERP (comando no servidor): peça à Venture"

// EmProducao: a empresa já transmite em produção.
func (c *FiscalConfig) EmProducao() bool {
	return c != nil && strings.EqualFold(strings.TrimSpace(c.FocusNfeAmbiente), AmbienteProducao)
}

// ConferirTrocaDoFocus: pela tela, a empresa só mexe no Focus de homologação.
// Entrar em produção, sair dela ou trocar o token de produção é com a Venture
// ERP, por comando no servidor (cmd/venture-admin). Trocar o ambiente para
// homologação também é bloqueado: senão bastava ir à homologação, gravar
// outro token e voltar.
func ConferirTrocaDoFocus(atual *FiscalConfig, ambiente string, token *string) error {
	ambiente = strings.ToLower(strings.TrimSpace(ambiente))
	tokenNovo := token != nil && strings.TrimSpace(*token) != ""
	if atual.EmProducao() {
		if ambiente != AmbienteProducao || tokenNovo {
			return domainerr.NewValidation(MsgTokenProducaoPelaVenture)
		}
		return nil
	}
	if ambiente == AmbienteProducao {
		return domainerr.NewValidation(MsgTokenProducaoPelaVenture)
	}
	return nil
}
