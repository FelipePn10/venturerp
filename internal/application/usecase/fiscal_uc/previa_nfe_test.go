package fiscal_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/google/uuid"
)

func notaBase() *fiscalentity.FiscalExit {
	return &fiscalentity.FiscalExit{
		ID: 7, NumeroNF: 101, Serie: "1",
		DataEmissao:             time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC),
		CnpjDestinatario:        strp("11222333000181"),
		RazaoSocialDestinatario: strp("CLIENTE TESTE LTDA"),
		IEDestinatario:          strp("9012345678"),
		UFDestinatario:          strp("SP"),
		DestLogradouro:          strp("RUA DAS OFICINAS"),
		DestNumero:              strp("450"),
		DestBairro:              strp("DISTRITO INDUSTRIAL"),
		DestMunicipio:           strp("CAMPINAS"),
		DestCEP:                 strp("13050000"),
		DestEmail:               strp("compras@cliente.com.br"),
		Cfop:                    "6101",
		NaturezaOperacao:        "VENDA DE PRODUCAO DO ESTABELECIMENTO",
		ValorProdutos:           1000,
		ValorIPI:                50,
		ValorICMS:               120,
		ValorTotal:              1050,
		Status:                  fiscalentity.ExitStatusDraft,
	}
}

func configBase() *fiscalentity.FiscalConfig {
	return &fiscalentity.FiscalConfig{
		CnpjEmpresa: "22333444000155", RazaoSocial: "TECNOFER LTDA",
		IEEmpresa: strp("9099999999"), UFEmpresa: "PR",
		Logradouro: "AVENIDA DAS INDUSTRIAS", Numero: "1200", Bairro: "CENTRO",
		Municipio: "MANDAGUARI", CodigoMunicipio: "4114302", CEP: "86975000",
		FocusNfeToken: strp("token-valido"), FocusNfeAmbiente: "producao",
	}
}

func itemBase() []*fiscalentity.FiscalExitItem {
	return []*fiscalentity.FiscalExitItem{{
		Sequence: 1, ItemCode: i64p(5), Ncm: strp("73269000"), Cfop: "6101",
		Quantity: 10, UnitPrice: 100, TotalPrice: 1000, Description: strp("BALANCA RN-01001"),
	}}
}

// ─── Endereço do destinatário no payload ────────────────────────────────────

// O endereço do destinatário é obrigatório na NF-e. Antes ele não existia nem no
// banco, então o payload saía sem logradouro/bairro/município/CEP e a SEFAZ
// recusava a autorização.
func TestPayloadLevaEnderecoDoDestinatario(t *testing.T) {
	p := montarPayloadNFe(notaBase(), itemBase(), configBase(), PlanoDaNota{})
	d := p.Destinatario
	if d.Logradouro != "RUA DAS OFICINAS" || d.Numero != "450" || d.Bairro != "DISTRITO INDUSTRIAL" ||
		d.Municipio != "CAMPINAS" || d.CEP != "13050000" {
		t.Fatalf("endereço do destinatário não foi para o payload: %+v", d)
	}
	if d.Email != "compras@cliente.com.br" {
		t.Fatalf("e-mail do destinatário ausente: %+v", d)
	}
	if p.LocalDestino != 2 {
		t.Fatalf("PR → SP deveria ser interestadual (2), veio %d", p.LocalDestino)
	}
}

// ─── Forma de pagamento e duplicatas ────────────────────────────────────────

func planoEm3Parcelas(total float64, emissao time.Time) PlanoDaNota {
	t30 := emissao.AddDate(0, 0, 28)
	t60 := emissao.AddDate(0, 0, 56)
	return PlanoDaNota{
		CondicaoCode: i64p(3), CondicaoDescricao: "30% entrada, 28/56", Origem: "PEDIDO_DE_VENDA",
		Parcelas: []customerentity.ParcelaCalculada{
			{Numero: 1, Percentual: decimal.NewFromInt(30), Valor: decimal.NewFromFloat(total * 0.30), Vencimento: emissao, Evento: customerentity.BaseEntrada, DiasPrazo: 0, Descricao: "entrada"},
			{Numero: 2, Percentual: decimal.NewFromInt(35), Valor: decimal.NewFromFloat(total * 0.35), Vencimento: t30, Evento: customerentity.BaseEmissao, DiasPrazo: 28, Descricao: "28 dias"},
			{Numero: 3, Percentual: decimal.NewFromInt(35), Valor: decimal.NewFromFloat(total * 0.35), Vencimento: t60, Evento: customerentity.BaseEmissao, DiasPrazo: 56, Descricao: "56 dias"},
		},
	}
}

// Uma venda a prazo declarada como "dinheiro no ato" é informação fiscal errada,
// e sem duplicata o cliente recebe a nota sem saber quanto paga em cada data.
func TestVendaAPrazoLevaDuplicataEBoleto(t *testing.T) {
	nota := notaBase()
	p := montarPayloadNFe(nota, itemBase(), configBase(), planoEm3Parcelas(nota.ValorTotal, nota.DataEmissao))

	if len(p.FormaPagamento) != 3 {
		t.Fatalf("esperava 3 formas de pagamento (uma por parcela), veio %d", len(p.FormaPagamento))
	}
	if p.FormaPagamento[0].FormaPagamento != "01" {
		t.Fatalf("a entrada no ato deveria ser dinheiro (01), veio %s", p.FormaPagamento[0].FormaPagamento)
	}
	if p.FormaPagamento[1].FormaPagamento != "15" || p.FormaPagamento[2].FormaPagamento != "15" {
		t.Fatalf("parcelas a prazo deveriam ser boleto (15): %+v", p.FormaPagamento)
	}
	if len(p.Duplicatas) != 2 {
		t.Fatalf("esperava duplicata só para as 2 parcelas a prazo, veio %d", len(p.Duplicatas))
	}
	if p.Duplicatas[0].DataVencimento != nota.DataEmissao.AddDate(0, 0, 28).Format("2006-01-02") {
		t.Fatalf("vencimento da duplicata errado: %s", p.Duplicatas[0].DataVencimento)
	}
	var soma float64
	for _, f := range p.FormaPagamento {
		soma += f.Valor
	}
	if soma < nota.ValorTotal-0.01 || soma > nota.ValorTotal+0.01 {
		t.Fatalf("as formas de pagamento somam %.2f e a nota vale %.2f", soma, nota.ValorTotal)
	}
}

func TestVendaAVistaNaoLevaDuplicata(t *testing.T) {
	nota := notaBase()
	plano := PlanoDaNota{Parcelas: []customerentity.ParcelaCalculada{{
		Numero: 1, Valor: decimal.NewFromFloat(nota.ValorTotal), Vencimento: nota.DataEmissao,
		Evento: customerentity.BaseEmissao, DiasPrazo: 0,
	}}}
	p := montarPayloadNFe(nota, itemBase(), configBase(), plano)
	if len(p.Duplicatas) != 0 {
		t.Fatalf("venda à vista não tem duplicata, veio %d", len(p.Duplicatas))
	}
	if p.FormaPagamento[0].FormaPagamento != "01" {
		t.Fatalf("à vista deveria ser dinheiro (01), veio %s", p.FormaPagamento[0].FormaPagamento)
	}
}

// ─── Conferência prévia ─────────────────────────────────────────────────────

func TestNotaCompletaNaoTemImpedimento(t *testing.T) {
	nota := notaBase()
	nota.SalesOrderCode = i64p(900)
	pend := conferirNota(nota, itemBase(), configBase(), planoEm3Parcelas(nota.ValorTotal, nota.DataEmissao))
	for _, p := range pend {
		if p.Nivel == NivelImpede {
			t.Fatalf("nota completa não deveria ter impedimento, veio %s: %s", p.Campo, p.Mensagem)
		}
	}
}

func TestItemSemNcmImpedeEmissao(t *testing.T) {
	items := itemBase()
	items[0].Ncm = nil
	pend := conferirNota(notaBase(), items, configBase(), PlanoDaNota{})
	if !temPendencia(pend, NivelImpede, "itens[1].ncm") {
		t.Fatalf("item sem NCM tem de impedir a emissão: %+v", pend)
	}
}

func TestDestinatarioSemEnderecoImpedeEmissao(t *testing.T) {
	nota := notaBase()
	nota.DestLogradouro, nota.DestBairro, nota.DestMunicipio, nota.DestCEP = nil, nil, nil, nil
	pend := conferirNota(nota, itemBase(), configBase(), PlanoDaNota{})
	for _, campo := range []string{"destinatario.logradouro", "destinatario.bairro", "destinatario.municipio", "destinatario.cep"} {
		if !temPendencia(pend, NivelImpede, campo) {
			t.Fatalf("%s deveria impedir a emissão: %+v", campo, pend)
		}
	}
}

// O código do município zerado NÃO impede a NF-e (o provedor resolve o município
// pelo nome + UF), mas quebra CT-e, NFS-e e os arquivos do SPED. Dizer "a nota
// será rejeitada" seria mentira, e o time perderia a confiança na conferência.
func TestCodigoDeMunicipioZeradoEApenasAtencao(t *testing.T) {
	cfg := configBase()
	cfg.CodigoMunicipio = "0000000"
	pend := conferirNota(notaBase(), itemBase(), cfg, PlanoDaNota{})
	if temPendencia(pend, NivelImpede, "emitente.codigo_municipio") {
		t.Fatal("código de município zerado não pode impedir a NF-e")
	}
	if !temPendencia(pend, NivelAtencao, "emitente.codigo_municipio") {
		t.Fatalf("código de município zerado tem de aparecer como atenção: %+v", pend)
	}
	for _, p := range pend {
		if p.Campo == "emitente.codigo_municipio" && !strings.Contains(p.Mensagem, "SPED") {
			t.Fatalf("a mensagem precisa dizer o que realmente quebra: %s", p.Mensagem)
		}
	}
}

func TestHomologacaoAvisaQueNotaNaoTemValorFiscal(t *testing.T) {
	cfg := configBase()
	cfg.FocusNfeAmbiente = "homologacao"
	pend := conferirNota(notaBase(), itemBase(), cfg, PlanoDaNota{})
	if !temPendencia(pend, NivelAtencao, "ambiente") {
		t.Fatalf("homologação precisa avisar: %+v", pend)
	}
}

func TestSemTokenImpedeEmissao(t *testing.T) {
	cfg := configBase()
	cfg.FocusNfeToken = nil
	pend := conferirNota(notaBase(), itemBase(), cfg, PlanoDaNota{})
	if !temPendencia(pend, NivelImpede, "token_focus") {
		t.Fatalf("sem token não há emissão: %+v", pend)
	}
}

func TestNotaSemPedidoAvisaQueEstoqueNaoBaixa(t *testing.T) {
	pend := conferirNota(notaBase(), itemBase(), configBase(), PlanoDaNota{})
	if !temPendencia(pend, NivelAtencao, "pedido_de_venda") {
		t.Fatalf("nota solta precisa avisar sobre estoque e pedido: %+v", pend)
	}
}

func temPendencia(pend []response.PreviaPendenciaNFe, nivel, campo string) bool {
	for _, p := range pend {
		if p.Nivel == nivel && p.Campo == campo {
			return true
		}
	}
	return false
}

// ─── Numeração da nota ──────────────────────────────────────────────────────

type repoNumeracao struct {
	repository.FiscalRepository
	proximo    int64
	criada     *fiscalentity.FiscalExit
	chamouNext int
}

func (r *repoNumeracao) GetNextNFNumber(context.Context) (int64, error) {
	r.chamouNext++
	return r.proximo, nil
}
func (r *repoNumeracao) GetFiscalConfig(context.Context) (*fiscalentity.FiscalConfig, error) {
	return configBase(), nil
}
func (r *repoNumeracao) ListNcmTaxes(context.Context) ([]*fiscalentity.NcmTaxTable, error) {
	return nil, nil
}
func (r *repoNumeracao) ListICMSInterstate(context.Context) (map[string]float64, error) {
	return map[string]float64{}, nil
}
func (r *repoNumeracao) ListICMSInternal(context.Context) (map[string]struct{ ICMS, FCP float64 }, error) {
	return map[string]struct{ ICMS, FCP float64 }{}, nil
}
func (r *repoNumeracao) CreateExit(_ context.Context, e *fiscalentity.FiscalExit) (*fiscalentity.FiscalExit, error) {
	e.ID = 1
	r.criada = e
	return e, nil
}
func (r *repoNumeracao) CreateExitItem(_ context.Context, it *fiscalentity.FiscalExitItem) (*fiscalentity.FiscalExitItem, error) {
	return it, nil
}
func (r *repoNumeracao) GetExitItems(context.Context, int64) ([]*fiscalentity.FiscalExitItem, error) {
	return nil, nil
}

type authLiberado struct{ ports.AuthService }

func (authLiberado) CanCreateFiscalExit(context.Context) bool { return true }
func (authLiberado) UserID(context.Context) (uuid.UUID, error) {
	return uuid.MustParse("00000000-0000-0000-0000-000000000001"), nil
}

// O número da NF-e é sequência fiscal. Toda nota digitada nascia com número 0 —
// e duas notas com o mesmo número é problema com a Receita.
func TestNotaSemNumeroRecebeOProximoDaSequencia(t *testing.T) {
	repo := &repoNumeracao{proximo: 42}
	uc := &CreateFiscalExitUseCase{Repo: repo, Auth: authLiberado{}}
	_, err := uc.Execute(context.Background(), request.CreateFiscalExitDTO{
		DataEmissao: "2026-09-27", Cfop: "5101", NaturezaOperacao: "VENDA",
		Itens: []request.CreateFiscalExitItemDTO{{Sequence: 1, Cfop: "5101", Quantity: 1, UnitPrice: 10, TotalPrice: 10}},
	})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if repo.criada.NumeroNF != 42 {
		t.Fatalf("número da nota = %d, queria 42 (próximo da sequência)", repo.criada.NumeroNF)
	}
	if repo.criada.Serie != "1" {
		t.Fatalf("série vazia deveria virar \"1\", veio %q", repo.criada.Serie)
	}
}

// Quem precisa retomar uma numeração existente informa o número, e o sistema
// respeita: não pode sobrescrever a escolha do fiscal.
func TestNumeroInformadoEhRespeitado(t *testing.T) {
	repo := &repoNumeracao{proximo: 42}
	uc := &CreateFiscalExitUseCase{Repo: repo, Auth: authLiberado{}}
	_, err := uc.Execute(context.Background(), request.CreateFiscalExitDTO{
		NumeroNF: 907, Serie: "2", DataEmissao: "2026-09-27", Cfop: "5101", NaturezaOperacao: "VENDA",
		Itens: []request.CreateFiscalExitItemDTO{{Sequence: 1, Cfop: "5101", Quantity: 1, UnitPrice: 10, TotalPrice: 10}},
	})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if repo.criada.NumeroNF != 907 || repo.criada.Serie != "2" {
		t.Fatalf("número/série informados não foram respeitados: %d/%s", repo.criada.NumeroNF, repo.criada.Serie)
	}
	if repo.chamouNext != 0 {
		t.Fatal("não deveria consultar a sequência quando o número foi informado")
	}
}

func TestDataHoraEmissaoNoFusoDeBrasilia(t *testing.T) {
	// 03:11 UTC = 00:11 em Brasília do mesmo dia: hora da transmissão.
	agora := time.Date(2026, 10, 7, 3, 11, 49, 0, time.UTC)
	if got := dataHoraEmissao(time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC), agora); got != "2026-10-07T00:11:49-03:00" {
		t.Errorf("hoje = %s", got)
	}
	if got := dataHoraEmissao(time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), agora); got != "2026-10-05T00:00:00-03:00" {
		t.Errorf("data passada = %s", got)
	}
}
