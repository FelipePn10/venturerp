package fiscal_uc

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/pkg/validation"
)

const (
	cnpjEmitenteTeste = "11444777000161"
	cnpjEmpresaTeste  = "11222333000181"
)

func notaBoa(agora time.Time) (*entity.FiscalExit, []*entity.FiscalExitItem, *entity.FiscalConfig) {
	cfg := &entity.FiscalConfig{CnpjEmpresa: cnpjEmpresaTeste, RazaoSocial: "EMPRESA", IEEmpresa: sp("123"), Logradouro: "RUA A", Numero: "1",
		Bairro: "CENTRO", Municipio: "CURITIBA", UFEmpresa: "PR", CEP: "80010000", RegimeTributario: "3"}
	nota := &entity.FiscalExit{Finalidade: 1, DataEmissao: agora, CnpjDestinatario: sp(cnpjEmitenteTeste), UFDestinatario: sp("PR"), DestCEP: sp("80010000"),
		RazaoSocialDestinatario: sp("CLIENTE"), DestLogradouro: sp("RUA B"), DestNumero: sp("2"), DestBairro: sp("CENTRO"), DestMunicipio: sp("CURITIBA"),
		ValorProdutos: 100, ValorTotal: 110, ValorIPI: 5, ValorFrete: 5}
	item := int64(7)
	linha := &entity.FiscalExitItem{Sequence: 1, ItemCode: &item, Description: sp("PECA"), Ncm: sp("73181500"), Cfop: "5101", Quantity: 2, UnitPrice: 50, TotalPrice: 100}
	return nota, []*entity.FiscalExitItem{linha}, cfg
}

func TestConferirDocumentoDaNota(t *testing.T) {
	agora := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	casos := []struct {
		nome  string
		mexe  func(*entity.FiscalExit, *entity.FiscalExitItem, *entity.FiscalConfig)
		campo string
	}{
		{"nota correta", func(*entity.FiscalExit, *entity.FiscalExitItem, *entity.FiscalConfig) {}, ""},
		{"emitente com CNPJ inválido", func(_ *entity.FiscalExit, _ *entity.FiscalExitItem, c *entity.FiscalConfig) {
			c.CnpjEmpresa = "11222333000100"
		}, "emitente.cnpj"},
		{"destinatário com documento inválido", func(n *entity.FiscalExit, _ *entity.FiscalExitItem, _ *entity.FiscalConfig) {
			n.CnpjDestinatario = sp("11111111111111")
		}, "destinatario.documento"},
		{"destinatário sem CEP", func(n *entity.FiscalExit, _ *entity.FiscalExitItem, _ *entity.FiscalConfig) { n.DestCEP = sp("") }, "destinatario.cep"},
		{"emissão no futuro", func(n *entity.FiscalExit, _ *entity.FiscalExitItem, _ *entity.FiscalConfig) {
			n.DataEmissao = agora.AddDate(0, 0, 5)
		}, "data_emissao"},
		{"emissão muito atrasada", func(n *entity.FiscalExit, _ *entity.FiscalExitItem, _ *entity.FiscalConfig) {
			n.DataEmissao = agora.AddDate(0, 0, -40)
		}, "data_emissao"},
		{"saída antes da emissão", func(n *entity.FiscalExit, _ *entity.FiscalExitItem, _ *entity.FiscalConfig) {
			d := agora.AddDate(0, 0, -3)
			n.DataSaida = &d
		}, "data_saida"},
		{"devolução sem nota de origem", func(n *entity.FiscalExit, _ *entity.FiscalExitItem, _ *entity.FiscalConfig) { n.Finalidade = 4 }, "nfe_referenciada"},
		{"nota de origem com chave errada", func(n *entity.FiscalExit, _ *entity.FiscalExitItem, _ *entity.FiscalConfig) {
			n.Finalidade = 4
			n.NFeReferenciada = sp(strings.Repeat("1", 44))
		}, "nfe_referenciada"},
		{"linha sem NCM", func(_ *entity.FiscalExit, l *entity.FiscalExitItem, _ *entity.FiscalConfig) { l.Ncm = nil }, "itens[1].ncm"},
		{"CFOP sem 4 dígitos", func(_ *entity.FiscalExit, l *entity.FiscalExitItem, _ *entity.FiscalConfig) { l.Cfop = "510" }, "itens[1].cfop"},
		{"quantidade zero", func(_ *entity.FiscalExit, l *entity.FiscalExitItem, _ *entity.FiscalConfig) { l.Quantity = 0 }, "itens[1].quantidade"},
		{"conta da linha não fecha", func(_ *entity.FiscalExit, l *entity.FiscalExitItem, _ *entity.FiscalConfig) { l.UnitPrice = 60 }, "itens[1].total"},
		{"soma das linhas não fecha", func(n *entity.FiscalExit, _ *entity.FiscalExitItem, _ *entity.FiscalConfig) {
			n.ValorProdutos = 90
			n.ValorTotal = 100
		}, "valor_produtos"},
		{"total não fecha", func(n *entity.FiscalExit, _ *entity.FiscalExitItem, _ *entity.FiscalConfig) { n.ValorTotal = 111 }, "valor_total"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			nota, itens, cfg := notaBoa(agora)
			c.mexe(nota, itens[0], cfg)
			n := niveis(conferirDocumentoDaNota(nota, itens, cfg, agora))
			if c.campo == "" {
				if len(n) != 0 {
					t.Fatalf("nota correta não pode ter pendência: %v", n)
				}
				return
			}
			if n[c.campo] != NivelImpede {
				t.Fatalf("%s deveria impedir: %v", c.campo, n)
			}
		})
	}
}

// Sem descrição gravada a nota leva a identificação do item; sem item, não há o que levar.
func TestConferirDocumentoDescricao(t *testing.T) {
	agora := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	nota, itens, cfg := notaBoa(agora)
	itens[0].Description = nil
	if n := niveis(conferirDocumentoDaNota(nota, itens, cfg, agora)); n["itens[1].descricao"] != "" {
		t.Fatalf("item cadastrado tem o que transmitir: %v", n)
	}
	itens[0].ItemCode = nil
	if n := niveis(conferirDocumentoDaNota(nota, itens, cfg, agora)); n["itens[1].descricao"] != NivelImpede {
		t.Fatalf("linha sem descrição e sem item: %v", n)
	}
}

func xmlBom() *NFeLida {
	return &NFeLida{
		ChaveAcesso: validation.MontarChaveNFe("41", "2610", cnpjEmitenteTeste, 1, 555, 12345678),
		Numero:      "555", Serie: "1", Modelo: "55", EmitenteUF: "PR", EmitenteCNPJ: cnpjEmitenteTeste,
		DataEmissao:      time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC),
		DestinatarioCNPJ: cnpjEmpresaTeste, SituacaoProtocolo: "100", Ambiente: "1",
	}
}

func TestConferirXMLDaEntrada(t *testing.T) {
	cfg := &entity.FiscalConfig{CnpjEmpresa: cnpjEmpresaTeste, FocusNfeAmbiente: "producao"}
	if avisos, err := conferirXMLDaEntrada(xmlBom(), cfg); err != nil || len(avisos) != 0 {
		t.Fatalf("XML correto: %v %v", avisos, err)
	}
	recusa := []struct {
		nome string
		mexe func(*NFeLida)
	}{
		{"modelo 65", func(n *NFeLida) { n.Modelo = "65" }},
		{"chave com dígito errado", func(n *NFeLida) { n.ChaveAcesso = n.ChaveAcesso[:43] + string('0'+(n.ChaveAcesso[43]-'0'+1)%10) }},
		{"número diferente da chave", func(n *NFeLida) { n.Numero = "556" }},
		{"série diferente da chave", func(n *NFeLida) { n.Serie = "2" }},
		{"emitente diferente da chave", func(n *NFeLida) { n.EmitenteCNPJ = cnpjEmpresaTeste }},
		{"UF diferente da chave", func(n *NFeLida) { n.EmitenteUF = "SP" }},
		{"mês diferente da chave", func(n *NFeLida) { n.DataEmissao = n.DataEmissao.AddDate(0, 2, 0) }},
		{"nota não autorizada", func(n *NFeLida) { n.SituacaoProtocolo = "302" }},
		{"homologação em produção", func(n *NFeLida) { n.Ambiente = "2" }},
		{"outro destinatário", func(n *NFeLida) { n.DestinatarioCNPJ = "99888777000166" }},
	}
	for _, c := range recusa {
		t.Run(c.nome, func(t *testing.T) {
			n := xmlBom()
			c.mexe(n)
			if _, err := conferirXMLDaEntrada(n, cfg); err == nil {
				t.Fatal("deveria recusar")
			}
		})
	}

	n := xmlBom()
	n.SituacaoProtocolo = ""
	if avisos, err := conferirXMLDaEntrada(n, cfg); err != nil || len(avisos) != 1 {
		t.Fatalf("sem protocolo só avisa: %v %v", avisos, err)
	}
	n = xmlBom()
	n.DestinatarioCNPJ = cnpjEmpresaTeste[:8] + "000299"
	if avisos, err := conferirXMLDaEntrada(n, cfg); err != nil || len(avisos) != 1 {
		t.Fatalf("outro estabelecimento só avisa: %v %v", avisos, err)
	}
	n = xmlBom()
	n.Ambiente = "2"
	if avisos, err := conferirXMLDaEntrada(n, &entity.FiscalConfig{CnpjEmpresa: cnpjEmpresaTeste, FocusNfeAmbiente: "homologacao"}); err != nil || len(avisos) != 1 {
		t.Fatalf("homologação em homologação só avisa: %v %v", avisos, err)
	}
}

func manualBoa() request.CreateFiscalEntryDTO {
	return request.CreateFiscalEntryDTO{
		NumeroNF: 555, Serie: "1", Modelo: "55", CnpjEmitente: cnpjEmitenteTeste, RazaoSocialEmitente: "FORNECEDOR", UFEmitente: sp("PR"),
		ValorProdutos: 100, ValorIPI: 5, ValorICMS: 12, ValorFrete: 10, ValorTotal: 115,
		Itens: []request.CreateFiscalEntryItemDTO{{Sequence: 1, Cfop: "5101", Quantity: 2, UnitPrice: 50, TotalPrice: 100, ValorIPI: 5, ValorICMS: 12}},
	}
}

func TestConferirNotaManual(t *testing.T) {
	agora := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	emissao, entrada := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)
	if err := conferirNotaManual(manualBoa(), emissao, entrada, "PR", agora); err != nil {
		t.Fatalf("nota correta: %v", err)
	}
	st := manualBoa()
	st.ValorICMSST, st.ValorOutras, st.ValorTotal = decimal.NewFromInt(8), decimal.NewFromInt(2), 125
	if err := conferirNotaManual(st, emissao, entrada, "PR", agora); err != nil {
		t.Fatalf("nota com ICMS-ST e outras despesas fecha: %v", err)
	}
	chaveBoa := validation.MontarChaveNFe("41", "2610", cnpjEmitenteTeste, 1, 555, 1)
	comChave := manualBoa()
	comChave.ChaveAcesso = &chaveBoa
	if err := conferirNotaManual(comChave, emissao, entrada, "PR", agora); err != nil {
		t.Fatalf("chave correta: %v", err)
	}

	recusa := []struct {
		nome string
		mexe func(*request.CreateFiscalEntryDTO)
	}{
		{"sem número", func(d *request.CreateFiscalEntryDTO) { d.NumeroNF = 0 }},
		{"CNPJ inválido", func(d *request.CreateFiscalEntryDTO) { d.CnpjEmitente = "11444777000100" }},
		{"sem razão social", func(d *request.CreateFiscalEntryDTO) { d.RazaoSocialEmitente = " " }},
		{"UF inexistente", func(d *request.CreateFiscalEntryDTO) { d.UFEmitente = sp("XX") }},
		{"chave de outra nota", func(d *request.CreateFiscalEntryDTO) {
			c := validation.MontarChaveNFe("41", "2610", cnpjEmitenteTeste, 1, 556, 1)
			d.ChaveAcesso = &c
		}},
		{"sem itens", func(d *request.CreateFiscalEntryDTO) { d.Itens = nil }},
		{"quantidade zero", func(d *request.CreateFiscalEntryDTO) { d.Itens[0].Quantity = 0 }},
		{"conta da linha", func(d *request.CreateFiscalEntryDTO) { d.Itens[0].UnitPrice = 60 }},
		{"NCM curto", func(d *request.CreateFiscalEntryDTO) { d.Itens[0].Ncm = sp("7318") }},
		{"CFOP interestadual em fornecedor da mesma UF", func(d *request.CreateFiscalEntryDTO) { d.Itens[0].Cfop = "6101" }},
		{"soma das linhas", func(d *request.CreateFiscalEntryDTO) { d.ValorProdutos = 90; d.ValorTotal = 105 }},
		{"IPI do cabeçalho", func(d *request.CreateFiscalEntryDTO) { d.ValorIPI = 9; d.ValorTotal = 119 }},
		{"ICMS do cabeçalho", func(d *request.CreateFiscalEntryDTO) { d.ValorICMS = 20 }},
		{"total digitado errado", func(d *request.CreateFiscalEntryDTO) { d.ValorTotal = 200 }},
		{"ST negativo", func(d *request.CreateFiscalEntryDTO) { d.ValorICMSST = decimal.NewFromInt(-1) }},
	}
	for _, c := range recusa {
		t.Run(c.nome, func(t *testing.T) {
			d := manualBoa()
			c.mexe(&d)
			if err := conferirNotaManual(d, emissao, entrada, "PR", agora); err == nil {
				t.Fatal("deveria recusar")
			}
		})
	}
	if err := conferirNotaManual(manualBoa(), agora.AddDate(0, 0, 5), agora.AddDate(0, 0, 5), "PR", agora); err == nil {
		t.Fatal("emissão no futuro")
	}
	if err := conferirNotaManual(manualBoa(), entrada, emissao, "PR", agora); err == nil {
		t.Fatal("entrada antes da emissão")
	}
}

func TestPlanoDoTipoNota(t *testing.T) {
	plano := PlanoDaNota{Origem: "CONDICAO"}
	if got := planoDoTipoNota(plano, entity.EfeitosPadraoDaNota()); got.SemCobranca {
		t.Fatal("venda comum tem cobrança")
	}
	if got := planoDoTipoNota(plano, entity.EfeitosDoTipoNota{}); !got.SemCobranca || got.Origem != "SEM_COBRANCA" || got.Parcelas != nil {
		t.Fatalf("tipo sem título: %+v", got)
	}
}
