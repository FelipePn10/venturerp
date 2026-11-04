package fiscal_uc

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/pkg/validation"
)

// prazoEmissaoNFe: a SEFAZ recusa a nota com emissão muito atrasada.
const prazoEmissaoNFe = 30

var fusoDaNota = time.FixedZone("BRT", -3*60*60)

var cfopSoDigitos = regexp.MustCompile(`^\d+$`)

// conferirDocumentoDaNota confere o documento antes de ele sair da empresa:
// identificação do emitente e do destinatário, contas da nota, datas e nota
// referenciada. Vale para a prévia e para a transmissão: o que está errado
// aparece na conferência, e não como rejeição da SEFAZ ou nota com valor errado.
//
// A tela também confere, mas nota nasce de pedido, carga, devolução e
// beneficiamento sem passar por ela. É aqui que a regra vale para todos.
func conferirDocumentoDaNota(exit *entity.FiscalExit, items []*entity.FiscalExitItem, cfg *entity.FiscalConfig, agora time.Time) []response.PreviaPendenciaNFe {
	if exit == nil || cfg == nil {
		return nil
	}
	var p []response.PreviaPendenciaNFe
	impede := func(campo, msg, como string) {
		p = append(p, response.PreviaPendenciaNFe{Nivel: NivelImpede, Campo: campo, Mensagem: msg, ComoResolver: como})
	}

	// ── Emitente
	const naConfig = "complete em VFIS0100 — Configuração Fiscal"
	if !validation.ValidateCNPJ(cfg.CnpjEmpresa) {
		impede("emitente.cnpj", "o CNPJ da empresa não é válido", naConfig)
	}
	for _, f := range []struct{ campo, nome, valor string }{
		{"razao_social", "a razão social", cfg.RazaoSocial}, {"ie", "a inscrição estadual", derefStr(cfg.IEEmpresa)},
		{"logradouro", "o logradouro", cfg.Logradouro}, {"numero", "o número do endereço", cfg.Numero},
		{"bairro", "o bairro", cfg.Bairro}, {"municipio", "o município", cfg.Municipio},
	} {
		if strings.TrimSpace(f.valor) == "" {
			impede("emitente."+f.campo, "falta "+f.nome+" da empresa emitente", naConfig)
		}
	}
	if !validation.UFValida(cfg.UFEmpresa) {
		impede("emitente.uf", "falta a UF da empresa emitente", naConfig)
	}
	if len(soDigitos(cfg.CEP)) != 8 {
		impede("emitente.cep", "o CEP da empresa emitente precisa ter 8 dígitos", naConfig)
	}

	// ── Destinatário
	ufDest := strings.ToUpper(strings.TrimSpace(derefStr(exit.UFDestinatario)))
	exterior := ufDest == "EX"
	const naNota = "corrija o cadastro do cliente, descarte este rascunho e gere a nota novamente"
	if !exterior {
		if !validation.ValidateCNPJOrCPF(derefStr(exit.CnpjDestinatario)) {
			impede("destinatario.documento", "o CNPJ ou CPF do destinatário não é válido", naNota)
		}
		if !validation.UFValida(ufDest) {
			impede("destinatario.uf", "falta a UF do destinatário", naNota)
		}
		if len(soDigitos(derefStr(exit.DestCEP))) != 8 {
			impede("destinatario.cep", "o CEP do destinatário precisa ter 8 dígitos", naNota)
		}
	}
	for _, f := range []struct{ campo, nome, valor string }{
		{"nome", "o nome ou a razão social", derefStr(exit.RazaoSocialDestinatario)},
		{"logradouro", "o logradouro", derefStr(exit.DestLogradouro)}, {"numero", "o número do endereço", derefStr(exit.DestNumero)},
		{"bairro", "o bairro", derefStr(exit.DestBairro)}, {"municipio", "o município", derefStr(exit.DestMunicipio)},
	} {
		if strings.TrimSpace(f.valor) == "" {
			impede("destinatario."+f.campo, "falta "+f.nome+" do destinatário", naNota)
		}
	}
	if ie := strings.TrimSpace(derefStr(exit.IEDestinatario)); ie != "" && !strings.EqualFold(ie, "ISENTO") && soDigitos(ie) == "" {
		impede("destinatario.ie", fmt.Sprintf("a inscrição estadual do destinatário (%s) não é válida: use só os números ou ISENTO", ie), naNota)
	}

	// ── Datas
	const refazer = "descarte este rascunho e gere a nota novamente com a data correta"
	hoje := dataSemHora(agora.In(fusoDaNota))
	emissao := dataSemHora(exit.DataEmissao)
	// Um dia de folga: o processo pode rodar em outro fuso, e a SEFAZ continua
	// sendo a última palavra sobre a nota datada de amanhã.
	if emissao.After(hoje.AddDate(0, 0, 1)) {
		impede("data_emissao", fmt.Sprintf("a data de emissão (%s) está no futuro", emissao.Format("02/01/2006")), refazer)
	}
	if emissao.Before(hoje.AddDate(0, 0, -prazoEmissaoNFe)) {
		impede("data_emissao", fmt.Sprintf("a data de emissão (%s) tem mais de %d dias: a SEFAZ recusa nota com emissão muito atrasada", emissao.Format("02/01/2006"), prazoEmissaoNFe), refazer)
	}
	if exit.DataSaida != nil {
		saida := dataSemHora(*exit.DataSaida)
		if saida.Before(emissao) {
			impede("data_saida", "a data de saída é anterior à data de emissão", refazer)
		}
		if saida.After(emissao.AddDate(0, 0, prazoEmissaoNFe)) {
			impede("data_saida", fmt.Sprintf("a data de saída é mais de %d dias depois da emissão", prazoEmissaoNFe), refazer)
		}
	}

	// ── Nota referenciada: devolução, complemento e ajuste apontam a nota de origem.
	if exit.Finalidade >= 2 && exit.Finalidade <= 4 {
		chave := strings.TrimSpace(derefStr(exit.NFeReferenciada))
		if chave == "" {
			impede("nfe_referenciada", "esta nota complementa, ajusta ou devolve outra e precisa da chave da nota de origem", "informe a chave da nota referenciada")
		} else if _, ok := validation.LerChaveNFe(chave); !ok {
			impede("nfe_referenciada", "a chave da nota referenciada não é válida: confira os 44 dígitos", "informe a chave correta da nota referenciada")
		}
	}

	// ── Linhas: nota complementar e de ajuste podem ter quantidade ou preço zerados.
	const refazerLinha = "descarte este rascunho e gere a nota novamente com a linha correta"
	normal := exit.Finalidade == 0 || exit.Finalidade == 1 || exit.Finalidade == 4
	entradaPropria := exit.TipoDocumento != nil && *exit.TipoDocumento == 0
	somaLinhas := decimal.Zero
	if len(items) == 0 {
		impede("itens", "a nota não tem itens", "inclua ao menos um item")
	}
	for _, it := range items {
		campo := fmt.Sprintf("itens[%d]", it.Sequence)
		linha := fmt.Sprintf("linha %d", it.Sequence)
		if strings.TrimSpace(descricaoDaLinha(it)) == "" {
			impede(campo+".descricao", linha+": falta a descrição do produto", refazerLinha)
		}
		if ncm := soDigitos(derefStr(it.Ncm)); len(ncm) != 8 {
			impede(campo+".ncm", linha+": o NCM precisa ter 8 dígitos", "complete o NCM do item em VENT0200 e gere a nota novamente")
		}
		if msg := cfopForaDoDestino(it.Cfop, ufDest, cfg.UFEmpresa, entradaPropria, exit.Finalidade); msg != "" {
			impede(campo+".cfop", linha+": "+msg, refazerLinha)
		}
		qtd, preco, total := it.Quantity, it.UnitPrice, it.TotalPrice
		somaLinhas = somaLinhas.Add(total)
		if normal && !qtd.IsPositive() {
			impede(campo+".quantidade", linha+": informe a quantidade", refazerLinha)
		}
		if normal && !preco.IsPositive() {
			impede(campo+".preco", linha+": informe o preço unitário", refazerLinha)
		}
		// Preço com mais de 2 casas arredonda: a folga cresce com a quantidade.
		folga := decimal.RequireFromString("0.01").Add(qtd.Mul(decimal.RequireFromString("0.00005")))
		if normal && qtd.Mul(preco).Sub(total).Abs().GreaterThan(folga) {
			impede(campo+".total", fmt.Sprintf("%s: quantidade × preço dá %s, mas o total da linha é %s", linha, qtd.Mul(preco).StringFixed(2), total.StringFixed(2)), refazerLinha)
		}
	}

	// ── Totais: a SEFAZ soma de novo e recusa a nota que não fecha.
	toleranciaTotal := decimal.RequireFromString("0.02")
	produtos := exit.ValorProdutos
	if len(items) > 0 && somaLinhas.Sub(produtos).Abs().GreaterThan(toleranciaTotal) {
		impede("valor_produtos", fmt.Sprintf("a soma das linhas (%s) não fecha com o total de produtos da nota (%s)", somaLinhas.StringFixed(2), produtos.StringFixed(2)), refazerLinha)
	}
	esperado := produtos.Add(exit.ValorIPI).Add(exit.ValorICMSST).
		Add(exit.ValorFrete).Add(exit.ValorSeguro).Sub(exit.ValorDesconto)
	total := exit.ValorTotal
	if total.Sub(esperado).Abs().GreaterThan(toleranciaTotal) {
		impede("valor_total", fmt.Sprintf("o total da nota (%s) não fecha com produtos + IPI + ICMS-ST + frete + seguro − desconto (%s)", total.StringFixed(2), esperado.StringFixed(2)), refazerLinha)
	}
	if normal && !total.IsPositive() {
		impede("valor_total", "o total da nota precisa ser maior que zero", "confira o desconto e gere a nota novamente")
	}
	return p
}

// cfopForaDoDestino: o primeiro dígito do CFOP diz o sentido e o destino.
// Na venda normal quem confere é conferirTributacao; aqui entram a nota de
// entrada própria e as de outras finalidades, e o CFOP mal preenchido.
func cfopForaDoDestino(cfop, ufDest, ufEmpresa string, entradaPropria bool, finalidade int) string {
	c := strings.TrimSpace(cfop)
	if len(c) != 4 || !cfopSoDigitos.MatchString(c) {
		return "o CFOP precisa ter 4 dígitos"
	}
	if entradaPropria {
		if c[0] != '1' && c[0] != '2' && c[0] != '3' {
			return fmt.Sprintf("a nota é de entrada e o CFOP %s é de saída", c)
		}
		return ""
	}
	if c[0] != '5' && c[0] != '6' && c[0] != '7' {
		return fmt.Sprintf("a nota é de saída e o CFOP %s é de entrada", c)
	}
	if (finalidade == 0 || finalidade == 1) || ufDest == "" {
		return ""
	}
	quer := byte('6')
	switch {
	case ufDest == "EX":
		quer = '7'
	case strings.EqualFold(ufDest, ufEmpresa):
		quer = '5'
	}
	if c[0] != quer {
		return fmt.Sprintf("o CFOP %s não combina com o destino %s: use um CFOP iniciado por %c", c, ufDest, quer)
	}
	return ""
}

// descricaoDaLinha acompanha o que a nota leva: sem descrição gravada mas com item
// cadastrado, o XML sai com a identificação do item (montarPayloadNFe); só a linha
// sem nenhum dos dois não tem o que transmitir.
func descricaoDaLinha(it *entity.FiscalExitItem) string {
	if it.Description != nil && strings.TrimSpace(*it.Description) != "" {
		return *it.Description
	}
	if safeInt64(it.ItemCode) != 0 {
		return fmt.Sprintf("Produto %d", safeInt64(it.ItemCode))
	}
	return ""
}

func dataSemHora(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
