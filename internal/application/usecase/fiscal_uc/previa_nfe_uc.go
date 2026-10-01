package fiscal_uc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/engine"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

const (
	NivelImpede  = "IMPEDE"
	NivelAtencao = "ATENCAO"
)

// PreviaNFeUseCase monta a nota como ela será transmitida e lista o que falta.
//
// Não chama a SEFAZ, não muda o status e não grava nada: é seguro rodar quantas
// vezes o time quiser, antes de cada emissão.
type PreviaNFeUseCase struct {
	Repo      repository.FiscalRepository
	Auth      ports.AuthService
	Customers customerrepo.CustomerRepository
	Orders    salesrepo.SalesOrderRepository
}

func (uc *PreviaNFeUseCase) Execute(ctx context.Context, id int64) (*response.PreviaNFeResponse, error) {
	if !uc.Auth.CanCreateFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	exit, err := uc.Repo.GetExitByID(ctx, id)
	if err != nil {
		return nil, err
	}
	items, err := uc.Repo.GetExitItems(ctx, id)
	if err != nil {
		return nil, err
	}
	cfg, err := uc.Repo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, err
	}

	plano := resolverPlanoDaNota(ctx, exit, uc.Customers, uc.Orders)
	payload := montarPayloadNFe(exit, items, cfg, plano)
	payloadJSON, _ := json.MarshalIndent(payload, "", "  ")

	out := &response.PreviaNFeResponse{
		FiscalExitID:     exit.ID,
		NumeroNF:         exit.NumeroNF,
		Serie:            exit.Serie,
		Status:           string(exit.Status),
		DataEmissao:      exit.DataEmissao.Format("2006-01-02"),
		NaturezaOperacao: exit.NaturezaOperacao,
		Cfop:             exit.Cfop,
		Ambiente:         cfg.FocusNfeAmbiente,
		ConsumidorFinal:  payload.ConsumidorFinal == 1,
		SalesOrderCode:   exit.SalesOrderCode,
		ShipmentLoadCode: exit.ShipmentLoadCode,
		PayloadEnviado:   string(payloadJSON),
	}
	if exit.DataSaida != nil {
		out.DataSaida = exit.DataSaida.Format("2006-01-02")
	}
	out.TipoOperacao = "Operação interna (mesmo estado)"
	if payload.LocalDestino == 2 {
		out.TipoOperacao = "Operação interestadual"
	}

	out.Emitente = response.PreviaParteNFe{
		Documento: cfg.CnpjEmpresa, Nome: cfg.RazaoSocial, IE: derefStr(cfg.IEEmpresa),
		Logradouro: cfg.Logradouro, Numero: cfg.Numero, Complemento: derefStr(cfg.Complemento),
		Bairro: cfg.Bairro, Municipio: cfg.Municipio, CodigoMunicipio: cfg.CodigoMunicipio,
		UF: cfg.UFEmpresa, CEP: cfg.CEP, Email: cfg.Email, Telefone: derefStr(cfg.Telefone),
	}
	out.Destinatario = response.PreviaParteNFe{
		Documento: derefStr(exit.CnpjDestinatario), Nome: derefStr(exit.RazaoSocialDestinatario),
		IE: derefStr(exit.IEDestinatario), Logradouro: derefStr(exit.DestLogradouro),
		Numero: derefStr(exit.DestNumero), Complemento: derefStr(exit.DestComplemento),
		Bairro: derefStr(exit.DestBairro), Municipio: derefStr(exit.DestMunicipio),
		CodigoMunicipio: derefStr(exit.DestCodigoMunicipio), UF: derefStr(exit.UFDestinatario),
		CEP: derefStr(exit.DestCEP), Email: derefStr(exit.DestEmail), Telefone: derefStr(exit.DestTelefone),
	}

	for _, it := range items {
		out.Itens = append(out.Itens, response.PreviaItemNFe{
			Sequence: it.Sequence, ItemCode: it.ItemCode, Descricao: derefStr(it.Description),
			Ncm: derefStr(it.Ncm), Cfop: it.Cfop, UM: "UN",
			Quantidade: it.Quantity, ValorUnit: it.UnitPrice, ValorTotal: it.TotalPrice,
			Origem:  it.OrigemMercadoria,
			CstICMS: derefStr(it.CstICMS), BaseICMS: it.BaseICMS, AliqICMS: it.AliqICMS, ValorICMS: it.ValorICMS,
			CstIPI: derefStr(it.CstIPI), AliqIPI: it.AliqIPI, ValorIPI: it.ValorIPI,
			CstPIS: derefStr(it.CstPIS), ValorPIS: it.ValorPIS,
			CstCOFINS: derefStr(it.CstCOFINS), ValorCOFINS: it.ValorCOFINS,
			BaseICMSST: it.BaseICMSST, ValorICMSST: it.ValorICMSST, MVA: it.MVA,
		})
	}

	out.Totais = response.PreviaTotaisNFe{
		ValorProdutos: exit.ValorProdutos, ValorFrete: exit.ValorFrete, ValorSeguro: exit.ValorSeguro,
		ValorDesconto: exit.ValorDesconto, ValorIPI: exit.ValorIPI, ValorICMS: exit.ValorICMS,
		BaseICMSST: exit.BaseICMSST, ValorICMSST: exit.ValorICMSST,
		ValorPIS: exit.ValorPIS, ValorCOFINS: exit.ValorCOFINS, ValorTotalNF: exit.ValorTotal,
		Conferencia: fmt.Sprintf(
			"produtos %s + IPI %s + ICMS-ST %s + frete %s + seguro %s − desconto %s = %s (o ICMS próprio, %s, está embutido no preço e não soma)",
			reais(exit.ValorProdutos), reais(exit.ValorIPI), reais(exit.ValorICMSST), reais(exit.ValorFrete),
			reais(exit.ValorSeguro), reais(exit.ValorDesconto), reais(exit.ValorTotal), reais(exit.ValorICMS)),
	}

	out.Pagamento = response.PreviaPagamentoNFe{
		CondicaoCode: plano.CondicaoCode, CondicaoDescricao: plano.CondicaoDescricao,
		Origem: rotuloOrigemCondicao(plano.Origem), Aviso: plano.Aviso,
	}
	for _, p := range plano.Parcelas {
		valor, _ := p.Valor.Float64()
		pct, _ := p.Percentual.Float64()
		out.Pagamento.Parcelas = append(out.Pagamento.Parcelas, response.PreviaParcelaNFe{
			Numero: p.Numero, Percentual: pct, Valor: valor,
			Vencimento: p.Vencimento.Format("2006-01-02"), Descricao: p.Descricao,
			FormaPagamentoNF: rotuloFormaPagamento(formaPagamentoNFe(p)), Estimado: p.Estimado,
		})
	}

	out.Pendencias = conferirNota(exit, items, cfg, plano)
	out.PodeAutorizar = true
	for _, p := range out.Pendencias {
		if p.Nivel == NivelImpede {
			out.PodeAutorizar = false
			break
		}
	}
	return out, nil
}

// conferirNota é a conferência que antes só existia na recusa da SEFAZ.
//
// A ordem das verificações segue a da nota: primeiro o que impede transmitir,
// depois o que passa mas tem consequência.
func conferirNota(
	exit *entity.FiscalExit,
	items []*entity.FiscalExitItem,
	cfg *entity.FiscalConfig,
	plano PlanoDaNota,
) []response.PreviaPendenciaNFe {
	var p []response.PreviaPendenciaNFe
	impede := func(campo, msg, como string) {
		p = append(p, response.PreviaPendenciaNFe{Nivel: NivelImpede, Campo: campo, Mensagem: msg, ComoResolver: como})
	}
	atencao := func(campo, msg, como string) {
		p = append(p, response.PreviaPendenciaNFe{Nivel: NivelAtencao, Campo: campo, Mensagem: msg, ComoResolver: como})
	}

	// ── Situação da nota
	if exit.Status != entity.ExitStatusDraft && exit.Status != entity.ExitStatusAwaitingAuthorization {
		impede("status", fmt.Sprintf("a nota está em %s e só rascunho pode ser autorizado", exit.Status),
			"gere uma nova nota; nota autorizada se resolve por cancelamento ou carta de correção")
	}

	// ── Provedor
	if cfg.FocusNfeToken == nil || strings.TrimSpace(*cfg.FocusNfeToken) == "" {
		impede("token_focus", "o token da Focus NF-e não está configurado, então a nota não pode ser transmitida",
			"informe o token em VFIS0100 — Configuração Fiscal")
	}
	if strings.EqualFold(cfg.FocusNfeAmbiente, "homologacao") {
		atencao("ambiente", "o ambiente é HOMOLOGAÇÃO: a nota é um teste e não tem valor fiscal",
			"para valer de verdade, troque o ambiente para produção em VFIS0100 — Configuração Fiscal")
	}

	// ── Emitente
	exigir := func(valor, campo, rotulo, tela string) {
		if strings.TrimSpace(valor) == "" {
			impede(campo, fmt.Sprintf("%s não está preenchido e a NF-e exige esse campo", rotulo), tela)
		}
	}
	telaFiscal := "preencha em VFIS0100 — Configuração Fiscal"
	exigir(cfg.CnpjEmpresa, "emitente.cnpj", "o CNPJ da empresa", telaFiscal)
	exigir(cfg.RazaoSocial, "emitente.razao_social", "a razão social da empresa", telaFiscal)
	exigir(cfg.Logradouro, "emitente.logradouro", "o logradouro da empresa", telaFiscal)
	exigir(cfg.Numero, "emitente.numero", "o número do endereço da empresa", telaFiscal)
	exigir(cfg.Bairro, "emitente.bairro", "o bairro da empresa", telaFiscal)
	exigir(cfg.Municipio, "emitente.municipio", "o município da empresa", telaFiscal)
	exigir(cfg.UFEmpresa, "emitente.uf", "a UF da empresa", telaFiscal)
	exigir(cfg.CEP, "emitente.cep", "o CEP da empresa", telaFiscal)
	if codigoMunicipioVazio(cfg.CodigoMunicipio) {
		// A NF-e sai sem ele (o provedor resolve o município pelo nome + UF), mas
		// CT-e, NFS-e e SPED levam o código direto no arquivo.
		atencao("emitente.codigo_municipio",
			"o código IBGE do município da empresa está zerado; a NF-e sai, mas CT-e, NFS-e e os arquivos do SPED (EFD e ECD) vão com código inválido",
			"informe o código do município em VFIS0100 — Configuração Fiscal (Mandaguari/PR é 4114302)")
	}

	// ── Destinatário
	exigirDest := func(valor *string, campo, rotulo string) {
		if valor == nil || strings.TrimSpace(*valor) == "" {
			impede(campo, fmt.Sprintf("%s não está preenchido e a NF-e exige esse campo", rotulo),
				"informe na própria nota ou complete o endereço do cliente em VCLI0500 — Cadastro de Cliente e gere a nota novamente")
		}
	}
	exigirDest(exit.CnpjDestinatario, "destinatario.documento", "o CNPJ/CPF do destinatário")
	exigirDest(exit.RazaoSocialDestinatario, "destinatario.nome", "o nome do destinatário")
	exigirDest(exit.DestLogradouro, "destinatario.logradouro", "o logradouro do destinatário")
	exigirDest(exit.DestNumero, "destinatario.numero", "o número do endereço do destinatário")
	exigirDest(exit.DestBairro, "destinatario.bairro", "o bairro do destinatário")
	exigirDest(exit.DestMunicipio, "destinatario.municipio", "o município do destinatário")
	exigirDest(exit.UFDestinatario, "destinatario.uf", "a UF do destinatário")
	exigirDest(exit.DestCEP, "destinatario.cep", "o CEP do destinatário")
	if exit.IEDestinatario == nil || strings.TrimSpace(*exit.IEDestinatario) == "" || *exit.IEDestinatario == "ISENTO" {
		atencao("destinatario.ie",
			"o destinatário está sem inscrição estadual, então a nota sai como consumidor final não contribuinte",
			"se o cliente é contribuinte, informe a inscrição estadual em VCLI0500 — Cadastro de Cliente; isso muda o cálculo do ICMS e do DIFAL")
	}
	if exit.DestEmail == nil || strings.TrimSpace(*exit.DestEmail) == "" {
		atencao("destinatario.email", "o destinatário não tem e-mail, então não receberá a nota automaticamente",
			"cadastre um contato com e-mail em VCLI0500 — Cadastro de Cliente")
	}

	// ── Operação e itens
	exigir(exit.NaturezaOperacao, "natureza_operacao", "a natureza da operação", "informe na própria nota")
	if len(items) == 0 {
		impede("itens", "a nota não tem nenhum item", "inclua os itens antes de emitir")
	}
	for _, it := range items {
		ref := fmt.Sprintf("item %d", it.Sequence)
		if it.Ncm == nil || strings.TrimSpace(*it.Ncm) == "" {
			impede(fmt.Sprintf("itens[%d].ncm", it.Sequence),
				fmt.Sprintf("%s está sem NCM e a SEFAZ recusa nota com item sem classificação fiscal", ref),
				"preencha a classificação fiscal do item em VENT0200 — Cadastro de Itens, aba Contábil, campo Classif. Fiscal Venda")
		} else if d := engine.NormalizarNCM(*it.Ncm); len(d) != 8 {
			// Vale apontar aqui: NCM torto só apareceria na recusa da SEFAZ, depois de
			// a nota já ter consumido numeração.
			impede(fmt.Sprintf("itens[%d].ncm", it.Sequence),
				fmt.Sprintf("%s tem o NCM %q, com %d dígitos — o NCM tem 8", ref, *it.Ncm, len(d)),
				"corrija a classificação fiscal do item em VENT0200 — Cadastro de Itens, aba Contábil, conferindo o código na tabela TIPI")
		}
		if strings.TrimSpace(it.Cfop) == "" {
			impede(fmt.Sprintf("itens[%d].cfop", it.Sequence), fmt.Sprintf("%s está sem CFOP", ref),
				"informe o CFOP da operação na nota (venda no estado 5101/5102, fora do estado 6101/6102)")
		}
		if it.Quantity <= 0 {
			impede(fmt.Sprintf("itens[%d].quantidade", it.Sequence), fmt.Sprintf("%s está com quantidade zero", ref),
				"corrija a quantidade do item")
		}
		if it.UnitPrice <= 0 {
			impede(fmt.Sprintf("itens[%d].valor_unitario", it.Sequence), fmt.Sprintf("%s está com preço zero", ref),
				"informe o preço do item; verifique se há preço vigente na tabela de venda em VCST0202")
		}
		if it.ValorICMSST > 0 {
			atencao(fmt.Sprintf("itens[%d].cest", it.Sequence),
				fmt.Sprintf("%s tem ICMS-ST calculado; itens com substituição tributária normalmente exigem CEST", ref),
				"informe o CEST na classificação fiscal do item em VENT0200 — Cadastro de Itens, aba Contábil")
		}
	}

	// ── Totais
	if exit.ValorTotal <= 0 {
		impede("valor_total", "o valor total da nota é zero", "verifique preços e quantidades dos itens")
	}

	// ── Pagamento
	if plano.Aviso != "" {
		atencao("pagamento", "a condição de pagamento não pôde ser dividida em parcelas: "+plano.Aviso,
			"ajuste os percentuais das parcelas em VCLI0520 — Apoio de Cliente (Comercial); a nota sai como parcela única enquanto isso")
	}
	if plano.Origem == "SEM_CONDICAO" {
		atencao("pagamento", "a nota não tem condição de pagamento, então sai como à vista e gera um único título",
			"vincule a condição de pagamento ao pedido de venda, ou defina a condição padrão do cliente em VCLI0500 — Cadastro de Cliente")
	}
	for _, parcela := range plano.Parcelas {
		if parcela.Estimado {
			atencao("pagamento",
				fmt.Sprintf("a parcela %d conta da entrega ou do faturamento e essa data não está definida: o vencimento mostrado é projetado", parcela.Numero),
				"informe a data de saída da nota para o vencimento sair correto")
		}
	}

	// ── Efeitos fora da nota
	if exit.SalesOrderCode == nil {
		atencao("pedido_de_venda",
			"a nota não está vinculada a um pedido de venda: o estoque não será baixado e o pedido não será marcado como faturado",
			"fature a partir do pedido ou da carga (VFIS0640 — Faturamento de Carga e DANFE) para o estoque e o pedido acompanharem a nota")
	}

	return p
}

func codigoMunicipioVazio(codigo string) bool {
	codigo = strings.TrimSpace(codigo)
	if codigo == "" {
		return true
	}
	for _, r := range codigo {
		if r != '0' {
			return false
		}
	}
	return true
}

func rotuloOrigemCondicao(origem string) string {
	switch origem {
	case "PEDIDO_DE_VENDA":
		return "condição de pagamento do pedido de venda"
	case "CADASTRO_DO_CLIENTE":
		return "condição de pagamento padrão do cliente"
	default:
		return "sem condição de pagamento (à vista)"
	}
}

func rotuloFormaPagamento(codigo string) string {
	switch codigo {
	case "01":
		return "01 — Dinheiro"
	case "02":
		return "02 — Cheque"
	case "03":
		return "03 — Cartão de crédito"
	case "04":
		return "04 — Cartão de débito"
	case "15":
		return "15 — Boleto bancário"
	case "16":
		return "16 — Transferência bancária"
	case "17":
		return "17 — PIX"
	default:
		return codigo + " — Outros"
	}
}

func reais(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	return "R$ " + strings.Replace(s, ".", ",", 1)
}
