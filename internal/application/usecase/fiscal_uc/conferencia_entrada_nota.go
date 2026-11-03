package fiscal_uc

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/pkg/validation"
)

// conferirXMLDaEntrada recusa o arquivo que não pode virar título a pagar e
// crédito de imposto: nota sem valor fiscal, de outra empresa, ou com a chave que
// não é a dela. O XML é de um terceiro, então vale desconfiar do arquivo.
func conferirXMLDaEntrada(n *NFeLida, cfg *entity.FiscalConfig) ([]string, error) {
	var avisos []string
	if n.Modelo != "" && n.Modelo != "55" {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("este XML é do modelo %s: aqui entram só notas fiscais eletrônicas (modelo 55)", n.Modelo))
	}
	if n.ChaveAcesso != "" {
		if msg := divergenciaDaChave(n.ChaveAcesso, n.EmitenteCNPJ, n.Serie, n.Numero, n.Modelo, n.EmitenteUF, n.DataEmissao); msg != "" {
			return nil, errorsuc.NewValidationError("a chave de acesso deste XML não confere com a nota: " + msg + ". O arquivo pode ter sido alterado: peça o XML original ao fornecedor")
		}
	}

	switch n.SituacaoProtocolo {
	case "100", "150":
	case "":
		avisos = append(avisos, "este XML não traz o protocolo de autorização da SEFAZ: confirme a situação da nota antes de aprovar, ou importe pela chave")
	default:
		return nil, errorsuc.NewValidationError(fmt.Sprintf("este XML traz uma nota que a SEFAZ não autorizou (situação %s): ela não tem valor fiscal e não pode ser lançada", n.SituacaoProtocolo))
	}

	if n.Ambiente == "2" {
		if cfg != nil && cfg.FocusNfeAmbiente == "producao" {
			return nil, errorsuc.NewValidationError("este XML é de uma nota de homologação, sem valor fiscal: ela não pode gerar contas a pagar nem crédito de imposto")
		}
		avisos = append(avisos, "este XML é de uma nota de homologação, sem valor fiscal")
	}

	if cfg != nil && cfg.CnpjEmpresa != "" && n.DestinatarioCNPJ != "" {
		empresa := soDigitos(cfg.CnpjEmpresa)
		switch {
		case empresa == n.DestinatarioCNPJ:
		case len(empresa) == 14 && len(n.DestinatarioCNPJ) == 14 && empresa[:8] == n.DestinatarioCNPJ[:8]:
			avisos = append(avisos, fmt.Sprintf("o destinatário da nota é outro estabelecimento desta empresa (%s, e este é %s): confira se ela deve entrar aqui", n.DestinatarioCNPJ, empresa))
		default:
			return nil, errorsuc.NewValidationError(fmt.Sprintf("esta nota foi emitida para outro destinatário (%s) e o CNPJ desta empresa é %s: ela não pode entrar aqui", n.DestinatarioCNPJ, empresa))
		}
	}
	return avisos, nil
}

// divergenciaDaChave compara a chave com a nota. Vazio: confere.
func divergenciaDaChave(chave, documento, serie, numero, modelo, uf string, emissao time.Time) string {
	c, ok := validation.LerChaveNFe(chave)
	if !ok {
		return "ela não tem 44 dígitos ou o dígito verificador está errado"
	}
	if d := fmt.Sprintf("%014s", soDigitos(documento)); d != c.Documento {
		return fmt.Sprintf("ela é do emitente %s e a nota é de %s", c.Documento, soDigitos(documento))
	}
	if n, err := strconv.ParseInt(strings.TrimSpace(numero), 10, 64); err == nil && n != c.Numero {
		return fmt.Sprintf("ela é da nota %d e a nota informada é %d", c.Numero, n)
	}
	if s, err := strconv.Atoi(strings.TrimSpace(serie)); err == nil && s != c.Serie {
		return fmt.Sprintf("ela é da série %d e a nota informada é da série %d", c.Serie, s)
	}
	if m := strings.TrimSpace(modelo); m != "" && m != c.Modelo {
		return fmt.Sprintf("ela é do modelo %s e a nota informada é do modelo %s", c.Modelo, m)
	}
	if cod, ok := validation.CodigoIBGEDaUF[strings.ToUpper(strings.TrimSpace(uf))]; ok && cod != c.CodigoUF {
		return fmt.Sprintf("ela é de outra UF (código %s) e o emitente é de %s", c.CodigoUF, strings.ToUpper(strings.TrimSpace(uf)))
	}
	if !emissao.IsZero() && emissao.Format("0601") != c.AnoMes {
		return fmt.Sprintf("ela é de %s/20%s e a emissão informada é %s", c.AnoMes[2:], c.AnoMes[:2], emissao.Format("01/2006"))
	}
	return ""
}

// conferirNotaManual: o que foi digitado à mão não passou pela SEFAZ. A nota do
// fornecedor fecha na conta; se aqui não fecha, houve erro de digitação, e ele
// viraria estoque, custo e título a pagar errados.
func conferirNotaManual(dto request.CreateFiscalEntryDTO, emissao, entrada time.Time, ufEmpresa string, agora time.Time) error {
	erro := func(msg string, args ...any) error { return errorsuc.NewValidationError(fmt.Sprintf(msg, args...)) }

	if dto.NumeroNF <= 0 {
		return erro("informe o número da nota")
	}
	if !validation.ValidateCNPJOrCPF(dto.CnpjEmitente) {
		return erro("o CNPJ ou CPF do emitente não é válido: confira os dígitos")
	}
	if strings.TrimSpace(dto.RazaoSocialEmitente) == "" {
		return erro("informe a razão social do emitente")
	}
	ufEmitente := strings.ToUpper(strings.TrimSpace(derefStr(dto.UFEmitente)))
	if ufEmitente != "" && !validation.UFValida(ufEmitente) {
		return erro("a UF do emitente (%s) não existe", ufEmitente)
	}

	// Um dia de folga: o processo pode rodar em outro fuso.
	limite := dataSemHora(agora.In(fusoDaNota)).AddDate(0, 0, 1)
	if dataSemHora(emissao).After(limite) {
		return erro("a data de emissão (%s) está no futuro", emissao.Format("02/01/2006"))
	}
	if dataSemHora(entrada).After(limite) {
		return erro("a data de entrada (%s) está no futuro", entrada.Format("02/01/2006"))
	}
	if dataSemHora(entrada).Before(dataSemHora(emissao)) {
		return erro("a data de entrada não pode ser anterior à data de emissão")
	}

	if dto.ChaveAcesso != nil && soDigitos(*dto.ChaveAcesso) != "" {
		if msg := divergenciaDaChave(*dto.ChaveAcesso, dto.CnpjEmitente, dto.Serie, fmt.Sprint(dto.NumeroNF), dto.Modelo, ufEmitente, emissao); msg != "" {
			return erro("a chave de acesso não confere com a nota: %s", msg)
		}
	}

	if len(dto.Itens) == 0 {
		return erro("inclua ao menos um item na nota")
	}
	centavo := decimal.RequireFromString("0.01")
	somaProdutos, somaIPI, somaICMS := decimal.Zero, decimal.Zero, decimal.Zero
	for i, it := range dto.Itens {
		n := it.Sequence
		if n <= 0 {
			n = i + 1
		}
		qtd, preco, total := it.Quantity, it.UnitPrice, it.TotalPrice
		if !qtd.IsPositive() {
			return erro("linha %d: a quantidade precisa ser maior que zero", n)
		}
		if preco.IsNegative() || total.IsNegative() {
			return erro("linha %d: o valor não pode ser negativo", n)
		}
		if it.ValorICMS.IsNegative() || it.ValorIPI.IsNegative() || it.ValorPIS.IsNegative() || it.ValorCOFINS.IsNegative() {
			return erro("linha %d: os impostos não podem ser negativos", n)
		}
		if conta := qtd.Mul(preco); conta.Sub(total).Abs().GreaterThan(centavo.Add(qtd.Mul(decimal.RequireFromString("0.00005")))) {
			return erro("linha %d: quantidade × preço dá %s, mas o total da linha é %s", n, conta.StringFixed(2), total.StringFixed(2))
		}
		if ncm := soDigitos(derefStr(it.Ncm)); ncm != "" && len(ncm) != 8 {
			return erro("linha %d: o NCM tem 8 dígitos", n)
		}
		if msg := cfopDoFornecedorForaDaUF(it.Cfop, ufEmitente, ufEmpresa); msg != "" {
			return erro("linha %d: %s", n, msg)
		}
		somaProdutos, somaIPI, somaICMS = somaProdutos.Add(total), somaIPI.Add(it.ValorIPI), somaICMS.Add(it.ValorICMS)
	}

	dois := centavo.Mul(decimal.NewFromInt(2))
	produtos := dto.ValorProdutos
	if somaProdutos.Sub(produtos).Abs().GreaterThan(dois) {
		return erro("a soma das linhas (%s) não fecha com o total de produtos da nota (%s)", somaProdutos.StringFixed(2), produtos.StringFixed(2))
	}
	if ipi := dto.ValorIPI; somaIPI.Sub(ipi).Abs().GreaterThan(dois) {
		return erro("o IPI das linhas (%s) não fecha com o IPI da nota (%s)", somaIPI.StringFixed(2), ipi.StringFixed(2))
	}
	if icms := dto.ValorICMS; somaICMS.Sub(icms).Abs().GreaterThan(dois) {
		return erro("o ICMS das linhas (%s) não fecha com o ICMS da nota (%s)", somaICMS.StringFixed(2), icms.StringFixed(2))
	}
	if dto.ValorFrete.IsNegative() || dto.ValorSeguro.IsNegative() || dto.ValorDesconto.IsNegative() || dto.ValorICMSST.IsNegative() || dto.ValorOutras.IsNegative() {
		return erro("frete, seguro, desconto, ICMS-ST e outras despesas não podem ser negativos")
	}
	esperado := produtos.Add(dto.ValorIPI).Add(dto.ValorFrete).
		Add(dto.ValorSeguro).Add(dto.ValorICMSST).Add(dto.ValorOutras).Sub(dto.ValorDesconto)
	if total := dto.ValorTotal; total.Sub(esperado).Abs().GreaterThan(dois) {
		return erro("o total da nota (%s) não fecha com produtos + IPI + ICMS-ST + frete + seguro + outras despesas − desconto (%s)", total.StringFixed(2), esperado.StringFixed(2))
	}
	if !dto.ValorTotal.IsPositive() {
		return erro("o total da nota precisa ser maior que zero")
	}
	return nil
}

// cfopDoFornecedorForaDaUF: a nota do fornecedor traz o CFOP de saída dele: 5
// dentro do estado, 6 para outro. CFOP 1, 2 e 3 são de nota de entrada emitida
// pelo próprio fornecedor, sem esta relação a conferir.
func cfopDoFornecedorForaDaUF(cfop, ufEmitente, ufEmpresa string) string {
	c := strings.TrimSpace(cfop)
	if c == "" {
		return ""
	}
	if len(c) != 4 || !cfopSoDigitos.MatchString(c) || !strings.ContainsRune("123567", rune(c[0])) {
		return "o CFOP tem 4 dígitos e começa com 1, 2, 3, 5, 6 ou 7"
	}
	if ufEmitente == "" || strings.TrimSpace(ufEmpresa) == "" {
		return ""
	}
	mesmaUF := strings.EqualFold(ufEmitente, strings.TrimSpace(ufEmpresa))
	if c[0] == '5' && !mesmaUF {
		return fmt.Sprintf("o CFOP %s é de operação dentro do estado, mas o emitente é de %s e a empresa de %s", c, ufEmitente, strings.ToUpper(strings.TrimSpace(ufEmpresa)))
	}
	if c[0] == '6' && mesmaUF {
		return fmt.Sprintf("o CFOP %s é de operação entre estados, mas emitente e empresa são de %s", c, ufEmitente)
	}
	return ""
}
