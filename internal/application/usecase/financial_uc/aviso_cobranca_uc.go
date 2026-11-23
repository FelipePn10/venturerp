package financial_uc

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

// ContatosDoCliente lê e-mail de cobrança e celular do cliente (repositório PG).
type ContatosDoCliente interface {
	ContatosDeCobranca(ctx context.Context, clienteID int64) ([]string, string, error)
}

// AvisoDeCobrancaUseCase manda ao cliente o aviso do título — valor, vencimento,
// linha digitável e PIX — por e-mail ou WhatsApp (melhoria 13).
type AvisoDeCobrancaUseCase struct {
	Repo       repository.FinancialRepository
	Auth       ports.AuthService
	Contatos   ContatosDoCliente
	Email      ports.EmailProvider
	Mensageiro ports.Mensageiro
}

// AvisoDTO: canal EMAIL (padrão) ou WHATSAPP; destinos vazios = os do cadastro.
type AvisoDTO struct {
	Canal string   `json:"channel"`
	Para  []string `json:"to,omitempty"`
}

// AvisoEnviado diz para quem foi.
type AvisoEnviado struct {
	Canal string   `json:"channel"`
	Para  []string `json:"to"`
}

func (uc *AvisoDeCobrancaUseCase) Enviar(ctx context.Context, id int64, dto AvisoDTO) (*AvisoEnviado, error) {
	if !uc.Auth.CanBaixarContaReceber(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	cr, err := uc.Repo.GetContaReceber(ctx, id)
	if err != nil {
		return nil, err
	}
	if cr.Status != entity.ContaReceberStatusPendente && cr.Status != entity.ContaReceberStatusAprovado && cr.Status != entity.ContaReceberStatusVencido {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("só título em aberto tem aviso de cobrança; este está %s", cr.Status))
	}
	canal := strings.ToUpper(strings.TrimSpace(dto.Canal))
	if canal == "" {
		canal = "EMAIL"
	}
	para := dto.Para
	if len(para) == 0 && cr.ClienteID != nil && uc.Contatos != nil {
		emails, celular, err := uc.Contatos.ContatosDeCobranca(ctx, *cr.ClienteID)
		if err != nil {
			return nil, err
		}
		if canal == "WHATSAPP" {
			if celular != "" {
				para = []string{celular}
			}
		} else {
			para = emails
		}
	}
	if len(para) == 0 {
		return nil, errorsuc.NewValidationError("o cliente não tem e-mail de cobrança nem contato com celular: informe o destino")
	}
	texto := TextoDaCobranca(cr)
	switch canal {
	case "EMAIL":
		if uc.Email == nil {
			return nil, errorsuc.NewValidationError("o envio de e-mail não está configurado")
		}
		emails, err := validarEmailsCobranca(para)
		if err != nil {
			return nil, err
		}
		empresa, _ := uc.Auth.EnterpriseID(ctx)
		if err := uc.Email.Send(ctx, ports.EmailMessage{EnterpriseID: empresa, To: emails, Subject: "Aviso de cobrança — " + documento(cr),
			Text: texto, HTML: "<pre style=\"font-family:inherit\">" + html.EscapeString(texto) + "</pre>"}); err != nil {
			return nil, fmt.Errorf("enviando o aviso de cobrança: %w", err)
		}
		para = emails
	case "WHATSAPP":
		if uc.Mensageiro == nil {
			return nil, errorsuc.NewValidationError("o WhatsApp não está configurado")
		}
		for _, p := range para {
			fone := soDigitosFinanceiro(p)
			if len(fone) < 10 {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("telefone inválido: %s", p))
			}
			if err := uc.Mensageiro.Enviar(ctx, ports.MensagemCurta{Para: fone, Texto: texto}); err != nil {
				if errors.Is(err, ports.ErrIntegracaoDesligada) {
					return nil, errorsuc.NewValidationError("o WhatsApp (Cloud API da Meta) não está configurado: o administrador precisa ligar a integração")
				}
				return nil, fmt.Errorf("enviando o aviso por WhatsApp: %w", err)
			}
		}
	default:
		return nil, errorsuc.NewValidationError("canal inválido: use EMAIL ou WHATSAPP")
	}
	return &AvisoEnviado{Canal: canal, Para: para}, nil
}

func documento(cr *entity.ContaReceber) string {
	if cr.NumeroDocumento != nil && strings.TrimSpace(*cr.NumeroDocumento) != "" {
		return *cr.NumeroDocumento
	}
	return fmt.Sprintf("título %d", cr.ID)
}

// TextoDaCobranca é o aviso: documento, saldo, vencimento e como pagar.
func TextoDaCobranca(cr *entity.ContaReceber) string {
	saldo := cr.ValorBruto.Sub(cr.ValorRecebido).Sub(cr.Desconto)
	var b strings.Builder
	fmt.Fprintf(&b, "Documento: %s\nValor: R$ %s\nVencimento: %s\n", documento(cr), strings.Replace(saldo.StringFixed(2), ".", ",", 1), cr.DataVencimento.Format("02/01/2006"))
	if cr.LinhaDigitavel != nil && strings.TrimSpace(*cr.LinhaDigitavel) != "" {
		fmt.Fprintf(&b, "Linha digitável: %s\n", *cr.LinhaDigitavel)
	}
	if cr.ChavePixGerada != nil && strings.TrimSpace(*cr.ChavePixGerada) != "" {
		fmt.Fprintf(&b, "PIX copia e cola: %s\n", *cr.ChavePixGerada)
	}
	return b.String()
}

func validarEmailsCobranca(lista []string) ([]string, error) {
	var out []string
	for _, e := range lista {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if !strings.Contains(e, "@") || strings.ContainsAny(e, " \r\n") {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("e-mail inválido: %s", e))
		}
		out = append(out, e)
	}
	if len(out) == 0 {
		return nil, errorsuc.NewValidationError("informe ao menos um e-mail")
	}
	return out, nil
}

func soDigitosFinanceiro(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
