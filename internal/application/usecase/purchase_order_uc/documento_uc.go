package purchase_order_uc

import (
	"context"
	"fmt"
	"html"
	"net/mail"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

// DocumentoPedidoUseCase gera o PDF do pedido e o envia ao fornecedor.
type DocumentoPedidoUseCase struct {
	Repo      repository.PurchaseOrderRepository
	Consultas ConsultasCompras
	Previsao  *PrevisaoPagamentosUseCase
	Gerador   GeradorPDFPedido
	Email     ports.EmailProvider
	Auth      ports.AuthService
	Agora     func() time.Time
}

func (uc *DocumentoPedidoUseCase) agora() time.Time {
	if uc.Agora != nil {
		return uc.Agora()
	}
	return time.Now()
}

// montar junta capa, linhas, dados de terceiros e a previsão de pagamento.
func (uc *DocumentoPedidoUseCase) montar(ctx context.Context, code int64) (*entity.PurchaseOrder, *DocumentoPedido, error) {
	po, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, nil, err
	}
	itens, err := uc.Repo.ListItems(ctx, code)
	if err != nil {
		return nil, nil, err
	}
	dados, err := uc.Consultas.DadosDocumento(ctx, code)
	if err != nil {
		return nil, nil, err
	}
	t := entity.CalcularTotais(po, itens)
	d := DocumentoPedido{
		Dados: dados, Numero: po.OrderNumber, Codigo: po.Code, Emissao: po.EmissionDate, Entrega: po.DeliveryDate,
		Situacao: entity.RotuloSituacao(po.Status), Moeda: po.CurrencyCode, Frete: po.FreightType, ValorFrete: po.FreightValue,
		Bruto: t.Bruto, Desconto: t.Desconto, IPI: t.IPI, FreteFOB: t.Frete, Liquido: t.Liquido, Gerado: uc.agora(),
		Rascunho: po.Status == entity.PurchaseOrderStatusDRAFT || po.Status == entity.PurchaseOrderStatusREQUESTED,
	}
	if po.Notes != nil {
		d.Observacao = *po.Notes
	}
	if uc.Previsao != nil {
		prev, err := uc.Previsao.prever(ctx, po, itens, map[int64]*customerentity.PaymentCondition{})
		if err != nil {
			return nil, nil, err
		}
		d.Parcelas = prev.Parcelas
	}
	return po, &d, nil
}

// PDF devolve o documento e o nome do arquivo.
func (uc *DocumentoPedidoUseCase) PDF(ctx context.Context, code int64) ([]byte, string, error) {
	if !uc.Auth.CanGetPurchaseOrder(ctx) {
		return nil, "", errorsuc.ErrUnauthorized
	}
	po, d, err := uc.montar(ctx, code)
	if err != nil {
		return nil, "", err
	}
	b, err := uc.desenhar(d)
	if err != nil {
		return nil, "", err
	}
	return b, nomeArquivoPedido(po), nil
}

func (uc *DocumentoPedidoUseCase) desenhar(d *DocumentoPedido) ([]byte, error) {
	if uc.Gerador == nil {
		return nil, fmt.Errorf("gerador de PDF do pedido não configurado")
	}
	return uc.Gerador.PedidoCompra(*d)
}

func nomeArquivoPedido(po *entity.PurchaseOrder) string {
	return fmt.Sprintf("pedido-compra-%d.pdf", po.OrderNumber)
}

// Destinatarios sugere os e-mails do fornecedor do pedido.
func (uc *DocumentoPedidoUseCase) Destinatarios(ctx context.Context, code int64) ([]Destinatario, error) {
	if !uc.Auth.CanGetPurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	po, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if po.SupplierCode == nil {
		return []Destinatario{}, nil
	}
	return uc.Consultas.Destinatarios(ctx, *po.SupplierCode)
}

// Envios lista o histórico de envios do pedido.
func (uc *DocumentoPedidoUseCase) Envios(ctx context.Context, code int64) ([]Envio, error) {
	if !uc.Auth.CanGetPurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if _, err := uc.Repo.GetByCode(ctx, code); err != nil {
		return nil, err
	}
	return uc.Consultas.Envios(ctx, code)
}

// EnviarDTO é o envio pedido pela tela.
type EnviarDTO struct {
	Para     []string `json:"para"`
	Mensagem string   `json:"mensagem"`
}

// Enviar manda o PDF do pedido aprovado ao fornecedor e registra o envio — o
// que falhou também fica registrado, com o motivo.
func (uc *DocumentoPedidoUseCase) Enviar(ctx context.Context, code int64, dto EnviarDTO) (*Envio, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if uc.Email == nil {
		return nil, errorsuc.NewValidationError("o envio de e-mail não está configurado neste servidor")
	}
	po, d, err := uc.montar(ctx, code)
	if err != nil {
		return nil, err
	}
	switch po.Status {
	case entity.PurchaseOrderStatusAPPROVED, entity.PurchaseOrderStatusPARTIAL:
	default:
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o pedido %d está %s: só pedido aprovado vai ao fornecedor", po.Code, entity.RotuloSituacao(po.Status)))
	}
	para, err := validarEmails(dto.Para)
	if err != nil {
		return nil, err
	}
	pdf, err := uc.desenhar(d)
	if err != nil {
		return nil, err
	}
	empresa := strings.TrimSpace(d.Dados.Empresa.Nome)
	assunto := fmt.Sprintf("Pedido de compra nº %d", po.OrderNumber)
	if empresa != "" {
		assunto += " — " + empresa
	}
	msg := strings.TrimSpace(dto.Mensagem)
	if msg == "" {
		msg = fmt.Sprintf("Prezados,\n\nSegue em anexo o pedido de compra nº %d. Pedimos a gentileza de confirmar o recebimento e a data de entrega.\n\nAtenciosamente,\n%s", po.OrderNumber, empresa)
	}
	enterpriseID, _ := uc.Auth.EnterpriseID(ctx)
	por, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	envio := Envio{PurchaseOrderCode: po.Code, EnviadoEm: uc.agora(), EnviadoPor: &por, Destinatarios: strings.Join(para, ", "), Assunto: assunto, Situacao: "ENVIADO"}
	sendErr := uc.Email.Send(ctx, ports.EmailMessage{
		EnterpriseID: enterpriseID,
		MessageID:    fmt.Sprintf("<pedido-%d-%d@venturerp.local>", po.Code, uc.agora().UnixNano()),
		FromName:     empresa,
		To:           para,
		Subject:      assunto,
		Text:         msg,
		HTML:         "<p>" + strings.ReplaceAll(html.EscapeString(msg), "\n", "<br>") + "</p>",
		Attachments:  []ports.EmailAttachment{{FileName: nomeArquivoPedido(po), MIMEType: "application/pdf", Content: pdf}},
	})
	if sendErr != nil {
		envio.Situacao, envio.Erro = "FALHOU", sendErr.Error()
	}
	gravado, err := uc.Consultas.RegistrarEnvio(ctx, envio)
	if err != nil {
		return nil, err
	}
	if sendErr != nil {
		return gravado, errorsuc.NewValidationError("o e-mail não foi enviado: " + sendErr.Error())
	}
	return gravado, nil
}

func validarEmails(lista []string) ([]string, error) {
	vistos := map[string]bool{}
	var out []string
	for _, bruto := range lista {
		for _, parte := range strings.FieldsFunc(bruto, func(r rune) bool { return r == ',' || r == ';' || r == ' ' || r == '\n' }) {
			e := strings.ToLower(strings.TrimSpace(parte))
			if e == "" || vistos[e] {
				continue
			}
			a, err := mail.ParseAddress(e)
			if err != nil || a.Address != e || !strings.Contains(e[strings.LastIndex(e, "@")+1:], ".") {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("e-mail inválido: %s", parte))
			}
			vistos[e] = true
			out = append(out, e)
		}
	}
	if len(out) == 0 {
		return nil, errorsuc.NewValidationError("informe ao menos um e-mail do fornecedor")
	}
	if len(out) > 10 {
		return nil, errorsuc.NewValidationError("no máximo 10 destinatários por envio")
	}
	return out, nil
}
