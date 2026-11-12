package documentos

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/cutting_plan_uc"
	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/certificado"
	"github.com/shopspring/decimal"
)

var (
	agora = time.Date(2026, 11, 11, 14, 0, 0, 0, time.UTC)
	cab   = uc.Cabecalho{Emitente: uc.Emitente{Parte: uc.Parte{Nome: "TECNOFER FABRICACAO E MONTAGEM DE ESTRUTURAS METALICAS LTDA", CNPJCPF: "52454668000102", Cidade: "Mandaguari", UF: "PR"}}, Gerado: agora, EmitidoPor: "Teste"}
	D     = decimal.RequireFromString
)

func paginas(b []byte) int { return len(regexp.MustCompile(`/Type /Page[^s]`).FindAll(b, -1)) }

// conferir valida o PDF e procura os textos (ASCII) no conteúdo.
func conferir(t *testing.T, nome string, b []byte, minPag int, textos ...string) {
	t.Helper()
	if !bytes.HasPrefix(b, []byte("%PDF")) {
		t.Fatalf("%s: não é PDF", nome)
	}
	if n := paginas(b); n < minPag {
		t.Fatalf("%s: %d página(s), esperava ao menos %d", nome, n, minPag)
	}
	for _, s := range textos {
		if !bytes.Contains(b, []byte(s)) {
			t.Errorf("%s: texto %q ausente", nome, s)
		}
	}
}

func TestPedidoVendaPaginaECalculaTotais(t *testing.T) {
	ent := agora.AddDate(0, 1, 0)
	d := uc.PedidoVenda{Cabecalho: cab, Numero: 4, Emissao: agora, Entrega: &ent, Situacao: "F", Condicao: "30/60/90",
		Cliente:  uc.Parte{Nome: "VP - COMERCIO DE PECAS", CNPJCPF: "13934028000229", Cidade: "Ribeirão Preto", UF: "SP"},
		Parcelas: []uc.Parcela{{Numero: 1, Dias: 30, Vencimento: ent, Percentual: D("100"), Valor: D("7444.33")}}}
	for i := 1; i <= 40; i++ {
		d.Itens = append(d.Itens, uc.ItemPedidoVenda{Seq: i, Codigo: "RN-01001", Descricao: strings.Repeat("BALANÇA ASA DELTA ", 6), UM: "UN",
			Quantidade: D("20"), Preco: D("360.5"), Total: D("7210"), IPIPct: D("3.25"), IPI: D("234.33"), ICMSPct: D("12"), Entrega: &ent})
	}
	b := Gerador{}.PedidoVenda(d)
	conferir(t, "pedido de venda", b, 2, "PEDIDO DE VENDA", "TOTAL DO PEDIDO", "R$ 297.773,20", "FATURADO", "continua")
	conferir(t, "pedido vazio", Gerador{}.PedidoVenda(uc.PedidoVenda{Cabecalho: cab, Numero: 1, Emissao: agora}), 1, "vista")
}

func TestNotaEntradaConfereComPedido(t *testing.T) {
	d := uc.NotaEntrada{Cabecalho: cab, Numero: 900102, Serie: "1", Emissao: agora, Entrada: agora, Situacao: "APPROVED", PedidoCompra: 4,
		Fornecedor: uc.Parte{Nome: "JANDAIA"}, Total: D("100"), Chave: strings.Repeat("4", 44),
		Itens: []uc.ItemNotaEntrada{
			{Seq: 1, Codigo: "8", Quantidade: D("10"), Unitario: D("7.4"), Total: D("74"), Pedido: 4, LinhaPedido: 1, QtdPedido: D("10"), PrecoPedido: D("7.4"), CreditaICMS: true, ICMS: D("14.43"), QtdEstoque: D("10"), CustoAquisicao: D("59.57")},
			{Seq: 2, Codigo: "9", Quantidade: D("12"), Unitario: D("8"), Total: D("96"), Pedido: 4, LinhaPedido: 2, QtdPedido: D("10"), PrecoPedido: D("7.4")},
		}}
	conferir(t, "nota de entrada", Gerador{}.NotaEntrada(d), 1, "ESPELHO DA NOTA FISCAL DE ENTRADA", "Confere", "Divergente", "5,9570", "APROVADA")
}

func TestTitulosReceberEPagar(t *testing.T) {
	var ts []uc.Titulo
	for i := 1; i <= 30; i++ {
		ts = append(ts, uc.Titulo{Documento: fmt.Sprintf("NF-%d/1", i), Parte: uc.Parte{Codigo: int64(i%3 + 1), Nome: fmt.Sprintf("Parte %d", i%3)},
			Emissao: agora, Vencimento: agora.AddDate(0, 0, i*3-10), Parcela: 1, Parcelas: 1, Valor: D("100"), Situacao: "PENDENTE", Aprovacao: "PENDENTE"})
	}
	conferir(t, "a receber", Gerador{}.Titulos(uc.Titulos{Cabecalho: cab, Receber: true, Titulos: ts, Referencia: agora}), 1, "Vencido", "COMO PAGAR", "Nenhuma conta")
	conferir(t, "a pagar", Gerador{}.Titulos(uc.Titulos{Cabecalho: cab, Titulos: ts, Referencia: agora}), 2, "PROGRAMA", "RESUMO POR FORNECEDOR", "AUTORIZADO POR")
	conferir(t, "a pagar vazio", Gerador{}.Titulos(uc.Titulos{Cabecalho: cab, Referencia: agora}), 1)
}

func TestPlanoMateriaisArvore(t *testing.T) {
	d := uc.PlanoMateriais{Cabecalho: cab, Plano: 2, Nome: "GERAL", Status: "COMPLETED",
		Sugestoes: []uc.SugestaoPlano{
			{Codigo: 1, Item: "RN-01001", Tipo: "FABRICACAO", Quantidade: D("20"), Inicio: agora, Necessidade: agora.AddDate(0, 0, 2), Gerou: "OF 3", Firmada: true},
			{Codigo: 2, Item: "TP-A", Pai: "RN-01001", Tipo: "FABRICACAO", Quantidade: D("40"), Inicio: agora, Necessidade: agora.AddDate(0, 0, 1)},
			{Codigo: 3, Item: "CHAPA", Pai: "TP-A", Tipo: "COMPRA", Quantidade: D("376.79"), PrecoReferencia: D("7.4"), Inicio: agora.AddDate(0, 0, -5), Necessidade: agora},
			{Codigo: 4, Item: "ORFAO", Pai: "NAO-EXISTE", Tipo: "COMPRA", Quantidade: D("1"), Inicio: agora, Necessidade: agora},
		}}
	conferir(t, "plano de materiais", Gerador{}.PlanoMateriais(d), 1, "rvore de explos", "RN-01001", "ORFAO", "2.788,25", "Nenhuma exce")
}

func TestCapacidadeProgramacaoKardexCustoRomaneio(t *testing.T) {
	cap := uc.Capacidade{Cabecalho: cab, Plano: 2, Maquinas: map[string][]string{}}
	for i := 0; i < 25; i++ {
		cap.Cargas = append(cap.Cargas, uc.CargaCentro{Centro: "SOLDA", Data: agora.AddDate(0, 0, i), Requerido: D("9"), Disponivel: D("8"), CargaPct: D("112.5")})
	}
	conferir(t, "capacidade", Gerador{}.Capacidade(cap), 1, "MAPA DE CARGA", "113%")
	conferir(t, "capacidade vazia", Gerador{}.Capacidade(uc.Capacidade{Cabecalho: cab}), 1, "ainda n")

	ne := agora.AddDate(0, 0, 1)
	prog := uc.Programacao{Cabecalho: cab, De: agora, Ate: agora.AddDate(0, 0, 3), Operacoes: []uc.OperacaoProgramada{
		{Maquina: "LASER", Ordem: 9, Etapa: 10, Item: "TP", Quantidade: D("20"), Inicio: agora, Fim: agora.Add(2 * time.Hour), Necessidade: &ne},
		{Maquina: "PRENSA", Ordem: 8, Etapa: 20, Item: "BU", Quantidade: D("20"), Inicio: agora.Add(20 * time.Hour), Fim: agora.Add(28 * time.Hour), Situacao: "DONE"},
	}}
	conferir(t, "programação", Gerador{}.Programacao(prog), 1, "GANTT POR M", "SEQU", "2h00")

	k := uc.Kardex{Cabecalho: cab, De: agora, Ate: agora, Itens: []uc.ItemKardex{{Codigo: "8", Descricao: "CHAPA", UM: "KG", SaldoInicial: D("10"), CustoInicial: D("60"),
		Movimentos: []uc.MovimentoKardex{{Data: agora, Tipo: "IN", Entrada: true, Quantidade: D("5"), Custo: D("6")}, {Data: agora, Tipo: "REP", Quantidade: D("15"), Custo: D("6")}},
		SaldoFinal: D("0")}}}
	conferir(t, "kardex", Gerador{}.Kardex(k), 1, "KARDEX", "Saldo anterior", "Requisi")

	c := uc.CustoProducao{Cabecalho: cab, Ordens: []uc.CustoOrdem{{Ordem: 3, Item: "RN-01001", Quantidade: D("20"), Material: D("4855.58"), Maquina: D("266.52"),
		Total: D("5122.10"), Unitario: D("256.10"), PadraoTotal: D("480"), Consumos: []uc.ConsumoOrdem{{Codigo: "TP", Quantidade: D("40"), Custo: D("2399")}},
		Etapas: []uc.EtapaOrdem{{Seq: 10, Operacao: "Montagem", Prevista: D("1"), Real: D("1")}}}}}
	conferir(t, "custo", Gerador{}.CustoProducao(c), 1, "+967,1%", "overhead sa", "REAL")

	r := uc.Romaneio{Cabecalho: cab, Codigo: 1, Situacao: "SHIPPED", Criado: agora, Despachado: &agora, NFNumero: 66, NFChave: strings.Repeat("1", 44),
		Itens: []uc.ItemRomaneio{{Seq: 1, Codigo: "RN-01001", Quantidade: D("20"), PesoLiquido: D("14.45"), PesoBruto: D("14.46"), Conferido: true}}}
	conferir(t, "romaneio", Gerador{}.Romaneio(r), 1, "Romaneio de carga", "DESPACHADO", "289,00", "destaque aqui")
}

func TestPedidoCompraNovoLayout(t *testing.T) {
	d := purchase_order_uc.DocumentoPedido{Dados: &purchase_order_uc.DadosDocumento{Empresa: purchase_order_uc.Parte{Nome: "TECNOFER"}, Fornecedor: purchase_order_uc.Parte{Nome: "JANDAIA"}},
		Numero: 4, Emissao: agora, Rascunho: true, Liquido: D("2788.27")}
	conferir(t, "pedido de compra", PedidoCompra(d), 1, "PEDIDO DE COMPRA", "RASCUNHO", "R$ 2.788,27", "Endere")
}

func TestFormatos(t *testing.T) {
	if moeda(D("1234567.891")) != "R$ 1.234.567,89" || qtd(D("10.50")) != "10,5" || qtd(D("3")) != "3" || num(D("-5"), 2) != "-5,00" ||
		pctCurto(D("12.00")) != "12%" || cnpj("52454668000102") != "52.454.668/0001-02" || chaveNFe("12345678") != "1234 5678" {
		t.Fatalf("formatos: %s %s %s %s %s", moeda(D("1234567.891")), qtd(D("10.50")), num(D("-5"), 2), pctCurto(D("12.00")), cnpj("52454668000102"))
	}
	if ls := quebrar("uma frase longa que precisa quebrar em mais de uma linha", reg, 8, 60); len(ls) < 3 {
		t.Fatalf("quebrar: %v", ls)
	}
}

func TestPropostaECotacao(t *testing.T) {
	val := agora.AddDate(0, 0, 10)
	pr := uc.Proposta{Cabecalho: cab, Numero: 7, Emissao: agora, Validade: &val, Cliente: uc.Parte{Nome: "VP PEÇAS"}, Convertida: 4,
		Itens: []uc.ItemPedidoVenda{{Seq: 1, Codigo: "RN-01001", Descricao: "BALANÇA", Quantidade: D("20"), Preco: D("360.5"), Total: D("7210"), IPI: D("234.33")}}}
	conferir(t, "proposta", Gerador{}.Proposta(pr), 1, "PROPOSTA COMERCIAL", "R$ 7.444,33", "10 dia", "pedido de venda n")

	ent := agora.AddDate(0, 0, 20)
	c := uc.Cotacao{Cabecalho: cab, Codigo: 1, Emissao: agora, Fornecedores: []uc.Parte{{Codigo: 1, Nome: "JANDAIA"}, {Codigo: 25, Nome: "ACOBARRA"}},
		Itens: []uc.ItemCotacao{
			{ID: 10, Seq: 1, Codigo: "8", Quantidade: D("400"), UM: "KG", Entrega: &ent, Precos: []uc.PrecoCotacao{{Fornecedor: 1, Unitario: D("7.4"), Escolhido: true}, {Fornecedor: 25, Unitario: D("7.25")}}},
			{ID: 11, Seq: 2, Codigo: "9", Quantidade: D("120"), UM: "KG"},
		}}
	conferir(t, "mapa", Gerador{}.MapaCotacao(c), 1, "MAPA COMPARATIVO", "R$ 2.960,00", "R$ 2.900,00", "sem pre")
	c.Fornecedor = &c.Fornecedores[1]
	conferir(t, "solicitação", Gerador{}.SolicitacaoCotacao(c), 1, "Solicita", "ACOBARRA", "CONDI")
}

func TestFichaCertificadoMapa(t *testing.T) {
	ult := agora.AddDate(0, -1, 0)
	fc := uc.FichaCliente{Cabecalho: cab, Cliente: uc.Parte{Codigo: 4, Nome: "VP PEÇAS", CNPJCPF: "13934028000229"}, Ativo: true, Bloqueado: true, MotivoBloqueio: "limite",
		LimiteCredito: D("10000"), EmAberto: D("7444.33"), ValorPedidos: D("5000"), UltimaCompra: &ult, UltimaNF: 66,
		Enderecos: []uc.EnderecoCliente{{Tipo: "COBRANCA", Endereco: "RUA A, 1", Cidade: "Ribeirão Preto", UF: "SP", Padrao: true}}}
	conferir(t, "ficha do cliente", Gerador{}.FichaCliente(fc), 1, "FICHA CADASTRAL", "13.934.028/0002-29", "BLOQUEADO", "124,4% do limite", "Sem endere")

	cert := &dom.Certificado{Numero: 12, Status: "CANCELADO", EmitidoEm: agora, Conteudo: dom.Conteudo{Empresa: dom.Parte{Nome: "TECNOFER"}, ItemCode: 5, ItemNome: "BALANÇA", Quantidade: "20",
		Lotes: []dom.LoteResumo{{Numero: 31, Origem: "FINAL", Decisao: "APROVADO", Concessao: true, Aproveitada: "20", DecididoEm: agora,
			Caracteristicas: []dom.CaracResumo{{Nome: "Furo", Tipo: "VARIAVEL", Minimo: "50.1", Maximo: "50.4", N: 5, Media: "50.25"}, {Nome: "Solda", Tipo: "ATRIBUTO", N: 5, NaoConformes: 1}}}}}}
	conferir(t, "certificado", CertificadoQualidade(cert, agora, nil), 1, "CERTIFICADO DE QUALIDADE", "50,1 a 50,4", "CONCESS", "CANCELADO", "Conforme")

	pts := []*entity.CuttingPattern{
		{Sequence: 1, StockWidthMM: 3000, StockHeightMM: 1200, RepeatCount: 3, UtilizationPct: 70, UsedAreaMM2: 2520000,
			Placements: []*entity.PatternPlacement{{Label: "TP-A", PosXMM: 0, PosYMM: 0, WidthMM: 480, HeightMM: 380}, {Label: "TP-A", PosXMM: 485, WidthMM: 480, HeightMM: 380}}},
		{Sequence: 2, StockLengthMM: 6000, RepeatCount: 2, UtilizationPct: 90, UsedMM: 5400, RemnantMM: 600,
			Placements: []*entity.PatternPlacement{{Label: "VIGA", LengthMM: 2700}, {Label: "VIGA", OffsetMM: 2703, LengthMM: 2700}}},
	}
	m := cutting_plan_uc.MapaCorte{Plano: &entity.CuttingPlan{Code: 7, CutType: entity.CutTypeGuillotine2D, KerfMM: 3}, MaterialCodigo: "8", Padroes: pts, Gerado: agora}
	conferir(t, "mapa de corte", MapaCorte{}.MapaCorte(m), 3, "MAPA DE CORTE", "5 chapa", "2", "Cortado ______ de 3", "sobra 600 mm")
}
