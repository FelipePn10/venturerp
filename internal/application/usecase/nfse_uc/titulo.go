package nfse_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	financialentity "github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/nfse/entity"
)

// vencimentoDaNFSe: vazio vale emissão + 30 dias (o mesmo padrão da NF-e sem
// condição de pagamento); antes da emissão é recusado.
func vencimentoDaNFSe(informado *string, emissao time.Time) (*time.Time, error) {
	if informado == nil || strings.TrimSpace(*informado) == "" {
		v := emissao.AddDate(0, 0, 30)
		return &v, nil
	}
	v, err := time.Parse("2006-01-02", strings.TrimSpace(*informado))
	if err != nil {
		return nil, errorsuc.NewValidationError("data de vencimento inválida: use AAAA-MM-DD")
	}
	if v.Before(emissao) {
		return nil, errorsuc.NewValidationError("o vencimento não pode ser antes da emissão da NFS-e")
	}
	return &v, nil
}

// conferirCliente: o cliente tem de existir nesta empresa. Tomador em branco
// é preenchido com o documento e o nome do cadastro.
func (uc *CreateNFSeUseCase) conferirCliente(ctx context.Context, dto *request.CreateNFSeDTO) error {
	if uc.Clientes == nil {
		return errorsuc.NewValidationError("não foi possível conferir o cliente da NFS-e")
	}
	c, err := uc.Clientes.GetCustomerByCode(ctx, *dto.CustomerCode)
	if err != nil {
		if errorsuc.EhNaoEncontrado(err) {
			return errorsuc.NewValidationError(fmt.Sprintf("cliente %d não encontrado nesta empresa", *dto.CustomerCode))
		}
		return err
	}
	if c == nil {
		return errorsuc.NewValidationError(fmt.Sprintf("cliente %d não encontrado nesta empresa", *dto.CustomerCode))
	}
	if dto.TomadorCnpjCpf == nil || strings.TrimSpace(*dto.TomadorCnpjCpf) == "" {
		doc := soDigitos(c.DocumentNumber)
		dto.TomadorCnpjCpf = &doc
	}
	if dto.TomadorRazaoSocial == nil || strings.TrimSpace(*dto.TomadorRazaoSocial) == "" {
		nome := c.Name
		dto.TomadorRazaoSocial = &nome
	}
	return nil
}

func soDigitos(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// tituloDaNFSe: uma parcela pelo valor líquido (serviço menos o ISS retido),
// que é o que o tomador paga.
func tituloDaNFSe(n *entity.NFSe, por uuid.UUID, agora time.Time) *financialentity.ContaReceber {
	doc := fmt.Sprintf("NFS-e %s", deref(n.NumeroNFSe))
	if deref(n.NumeroNFSe) == "" {
		doc = fmt.Sprintf("NFS-e #%d", n.ID)
	}
	venc := n.DataEmissao.AddDate(0, 0, 30)
	if n.DataVencimento != nil {
		venc = *n.DataVencimento
	}
	return &financialentity.ContaReceber{
		NumeroDocumento: &doc,
		ClienteID:       n.CustomerCode,
		SalesOrderID:    n.SalesOrderCode,
		DataLancamento:  agora,
		DataEmissao:     n.DataEmissao,
		DataVencimento:  venc,
		ValorBruto:      n.ValorLiquido,
		Desconto:        decimal.Zero,
		Juros:           decimal.Zero,
		Multa:           decimal.Zero,
		ValorRecebido:   decimal.Zero,
		ParcelaNumero:   1,
		ParcelaTotal:    1,
		Status:          financialentity.ContaReceberStatusPendente,
		IsActive:        true,
		CriadoPor:       por,
	}
}

// recusarComTituloRecebido: cancelar a nota com o título já recebido deixaria
// o dinheiro no caixa sem o serviço que o explica — a mesma regra da NF-e.
func recusarComTituloRecebido(titulos []*financialentity.ContaReceber) error {
	for _, t := range titulos {
		if t != nil && t.ValorRecebido.IsPositive() {
			return errorsuc.NewValidationError(fmt.Sprintf(
				"o título %s desta NFS-e já teve recebimento de %s: estorne o recebimento antes de cancelar a nota",
				deref(t.NumeroDocumento), t.ValorRecebido.StringFixed(2)))
		}
	}
	return nil
}
