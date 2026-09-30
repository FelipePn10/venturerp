package customer_material_uc

import (
	"context"
	"errors"
	"strings"
	"testing"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

// notasFalsas registra o que o faturamento gravou no módulo fiscal.
type notasFalsas struct {
	proximo    int64
	saida      *fiscalentity.FiscalExit
	itens      []*fiscalentity.FiscalExitItem
	falharNum  bool
	falharNota bool
	// rascunhoPendente simula uma nota que ficou em rascunho porque o faturamento
	// anterior falhou depois de criá-la. É o cenário em que a repetição tem de
	// RETOMAR a nota, não criar outra.
	rascunhoPendente *fiscalentity.FiscalExit
	// falharLinha faz a gravação da linha de sequência informada falhar, para
	// reproduzir a parada no meio.
	falharLinha    int
	numerosPedidos int
}

// RascunhoDeBeneficiamento devolve o rascunho pendente da remessa, quando o cenário
// do teste tiver um.
func (n *notasFalsas) RascunhoDeBeneficiamento(context.Context, int64) (*fiscalentity.FiscalExit, error) {
	return n.rascunhoPendente, nil
}

// GetExitItems devolve as linhas já gravadas, para a retomada criar só o que falta.
func (n *notasFalsas) GetExitItems(_ context.Context, fiscalExitID int64) ([]*fiscalentity.FiscalExitItem, error) {
	out := make([]*fiscalentity.FiscalExitItem, 0, len(n.itens))
	for _, it := range n.itens {
		if it.FiscalExitID == fiscalExitID {
			out = append(out, it)
		}
	}
	return out, nil
}

func (n *notasFalsas) GetNextNFNumber(context.Context) (int64, error) {
	n.numerosPedidos++
	if n.falharNum {
		return 0, errors.New("falha ao numerar")
	}
	if n.proximo == 0 {
		n.proximo = 5912
	}
	return n.proximo, nil
}

func (n *notasFalsas) CreateExit(_ context.Context, e *fiscalentity.FiscalExit) (*fiscalentity.FiscalExit, error) {
	if n.falharNota {
		return nil, errors.New("falha ao criar a nota")
	}
	e.ID = 777
	n.saida = e
	return e, nil
}

func (n *notasFalsas) CreateExitItem(_ context.Context, item *fiscalentity.FiscalExitItem) (*fiscalentity.FiscalExitItem, error) {
	if n.falharLinha != 0 && item.Sequence == n.falharLinha {
		return nil, errors.New("falha ao gravar a linha")
	}
	item.ID = int64(len(n.itens) + 1)
	n.itens = append(n.itens, item)
	return item, nil
}

const autorValido = "11111111-1111-1111-1111-111111111111"

func destinatarioUsimac() DadosDoDestinatario {
	return DadosDoDestinatario{
		CNPJ: "43488105000578", RazaoSocial: "HUBBELL DO BRASIL", IE: "687124928115",
		UF: "SP", Logradouro: "Rua Anna Inghes del Fiol", Numero: "415",
		Bairro: "Vila Sao Cristovao", Municipio: "Tatui", CodigoMunicipio: "3553708",
		CEP: "18280005",
	}
}

func montarFaturamento() (*UseCase, *repoFalso, *notasFalsas) {
	repo := &repoFalso{remessa: remessaDaNF5911()}
	notas := &notasFalsas{}
	return &UseCase{Repo: repo, Notas: notas}, repo, notas
}

func pedidoPadrao() PedidoDeFaturamento {
	destinatario := destinatarioUsimac()
	return PedidoDeFaturamento{
		RemittanceID: 1,
		Servico:      servicoDaNF5911(),
		Destinatario: &destinatario,
		Devolucoes: []DevolucaoDeMaterial{
			{RemittanceItemID: 10, Quantidade: "3.848", Tipo: entity.MovimentoRetorno},
			{RemittanceItemID: 11, Quantidade: "1.304", Tipo: entity.MovimentoRetorno},
		},
	}
}

// TestFaturarGravaNotaComTresLinhas é o caminho feliz: a nota sai com o serviço e o
// material, com os CFOPs e CSTs que a contadora definiu.
func TestFaturarGravaNotaComTresLinhas(t *testing.T) {
	uc, repo, notas := montarFaturamento()

	resultado, err := uc.Faturar(context.Background(), pedidoPadrao(), autorValido)
	if err != nil {
		t.Fatalf("Faturar() error = %v", err)
	}
	if resultado.NumeroNF != 5912 {
		t.Errorf("número da nota = %d, esperado 5912", resultado.NumeroNF)
	}
	if len(notas.itens) != 3 {
		t.Fatalf("%d linha(s) gravada(s), esperado 3", len(notas.itens))
	}

	// Linha do serviço.
	if notas.itens[0].Cfop != "5124" || *notas.itens[0].CstICMS != "51" {
		t.Errorf("serviço: CFOP %s / CST %s, esperado 5124 / 051",
			notas.itens[0].Cfop, *notas.itens[0].CstICMS)
	}
	if notas.itens[0].CstPIS == nil || *notas.itens[0].CstPIS != "01" {
		t.Error("a linha do serviço saiu sem CST 01 de PIS")
	}
	if notas.itens[0].ValorPIS != 1.5 || notas.itens[0].ValorCOFINS != 6.9 {
		t.Errorf("PIS/COFINS = %.2f/%.2f, esperado 1.50/6.90",
			notas.itens[0].ValorPIS, notas.itens[0].ValorCOFINS)
	}

	// Linhas de material: 5902, CST ICMS 50 (suspensão), CST PIS/COFINS 49 sem valor,
	// unidade e código DO CLIENTE.
	for _, item := range notas.itens[1:] {
		if item.Cfop != "5902" || *item.CstICMS != "50" {
			t.Errorf("material: CFOP %s / CST %s, esperado 5902 / 50", item.Cfop, *item.CstICMS)
		}
		// O CST vai, o VALOR não: campo vazio cairia no padrão "01" do autorizador e
		// declararia o material do cliente como operação tributável.
		if item.CstPIS == nil || *item.CstPIS != "49" {
			t.Errorf("CST de PIS do material = %v, esperado 49", item.CstPIS)
		}
		if item.ValorPIS != 0 || item.ValorCOFINS != 0 {
			t.Errorf("a linha de material saiu com valor de PIS/COFINS: %.2f/%.2f", item.ValorPIS, item.ValorCOFINS)
		}
		if item.ValorICMS != 0 {
			t.Errorf("a linha de material saiu com ICMS %.2f", item.ValorICMS)
		}
		// ⚠️ Unidade e código do produto: sem eles o autorizador manda "UN" fixo e
		// `cProd` "0". O material da Usimac é em KG, com o código do cliente.
		if item.UnidadeComercial == nil || *item.UnidadeComercial == "" {
			t.Error("a linha de material saiu sem unidade: o autorizador cairia em UN")
		}
		if item.CodigoProduto == nil || *item.CodigoProduto == "" {
			t.Error("a linha de material saiu sem código de produto: o autorizador serializaria \"0\"")
		}
	}

	// Total da nota real.
	if notas.saida.ValorTotal != 412.6 {
		t.Errorf("total da nota = %.2f, esperado 412.60", notas.saida.ValorTotal)
	}
	// A nota nasce em RASCUNHO: a transmissão é passo separado.
	if notas.saida.Status != fiscalentity.ExitStatusDraft {
		t.Errorf("situação da nota = %s, esperado DRAFT", notas.saida.Status)
	}
	if notas.saida.SourceType == nil || *notas.saida.SourceType != "BENEFICIAMENTO" {
		t.Error("a nota não ficou marcada como origem BENEFICIAMENTO")
	}

	// E o saldo foi baixado, vinculado a esta nota.
	if repo.notaBaixada != 777 {
		t.Errorf("baixa vinculada à nota %d, esperado 777", repo.notaBaixada)
	}
	if len(repo.baixas) != 2 {
		t.Fatalf("%d baixa(s), esperado 2 — a linha de serviço não baixa saldo", len(repo.baixas))
	}
	for _, baixa := range repo.baixas {
		if baixa.MovementType != entity.MovimentoRetorno {
			t.Errorf("tipo da baixa = %s, esperado RETURN", baixa.MovementType)
		}
		if baixa.CFOP != "5902" {
			t.Errorf("CFOP da baixa = %s, esperado 5902", baixa.CFOP)
		}
	}
}

// TestFaturarPreencheODestinatario: sem endereço completo a SEFAZ rejeita por campo
// obrigatório ausente, e o erro chega tarde.
func TestFaturarPreencheODestinatario(t *testing.T) {
	uc, _, notas := montarFaturamento()
	if _, err := uc.Faturar(context.Background(), pedidoPadrao(), autorValido); err != nil {
		t.Fatalf("Faturar() error = %v", err)
	}
	for nome, valor := range map[string]*string{
		"CNPJ": notas.saida.CnpjDestinatario, "razão social": notas.saida.RazaoSocialDestinatario,
		"UF": notas.saida.UFDestinatario, "logradouro": notas.saida.DestLogradouro,
		"número": notas.saida.DestNumero, "bairro": notas.saida.DestBairro,
		"município": notas.saida.DestMunicipio, "código do município": notas.saida.DestCodigoMunicipio,
		"CEP": notas.saida.DestCEP,
	} {
		if valor == nil || *valor == "" {
			t.Errorf("a nota saiu sem %s do destinatário", nome)
		}
	}
}

// TestFaturarSemBaixaAvisaComoResolver: se a baixa falhar, a nota existe em rascunho.
// A mensagem tem de dizer que repetir resolve — senão o operador emite outra nota.
func TestFaturarSemBaixaAvisaComoResolver(t *testing.T) {
	uc, repo, _ := montarFaturamento()
	repo.falharBaixa = true

	_, err := uc.Faturar(context.Background(), pedidoPadrao(), autorValido)
	if err == nil {
		t.Fatal("a falha na baixa não foi reportada")
	}
	if !strings.Contains(err.Error(), "Repita o faturamento") {
		t.Fatalf("a mensagem não diz como resolver: %v", err)
	}
	if !strings.Contains(err.Error(), "não duplica") {
		t.Fatalf("a mensagem não tranquiliza sobre duplicar: %v", err)
	}
}

// TestFaturarSemNotasConfiguradas: ambiente sem emissão fiscal recusa com mensagem
// clara, em vez de estourar em nil.
func TestFaturarSemNotasConfiguradas(t *testing.T) {
	uc := &UseCase{Repo: &repoFalso{remessa: remessaDaNF5911()}}
	if _, err := uc.Faturar(context.Background(), pedidoPadrao(), autorValido); err == nil {
		t.Fatal("faturou sem emissor de nota configurado")
	} else if _, ok := errorsuc.AsValidation(err); !ok {
		t.Fatalf("erro deveria ser de validação: %v", err)
	}
}

// TestFaturarExigeAutorValido: a nota grava quem emitiu; um usuário inválido não
// pode virar um UUID zerado em documento fiscal.
func TestFaturarExigeAutorValido(t *testing.T) {
	for _, autor := range []string{"", "   ", "nao-e-uuid"} {
		uc, _, _ := montarFaturamento()
		if _, err := uc.Faturar(context.Background(), pedidoPadrao(), autor); err == nil {
			t.Fatalf("faturou com autor %q", autor)
		}
	}
}

// TestFaturarNaoGravaNotaQuandoAValidacaoFalha: a validação vem ANTES de qualquer
// gravação, então uma recusa não deixa nota órfã.
func TestFaturarNaoGravaNotaQuandoAValidacaoFalha(t *testing.T) {
	uc, _, notas := montarFaturamento()
	pedido := pedidoPadrao()
	pedido.Devolucoes[0].Quantidade = "999" // acima do saldo

	if _, err := uc.Faturar(context.Background(), pedido, autorValido); err == nil {
		t.Fatal("faturou acima do saldo")
	}
	if notas.saida != nil {
		t.Error("a nota foi gravada apesar de a validação falhar")
	}
	if len(notas.itens) != 0 {
		t.Errorf("%d linha(s) gravada(s) apesar da falha", len(notas.itens))
	}
}

// TestFaturarInterestadualNaoGravaNada: o CFOP interestadual não foi validado pelo
// responsável fiscal, e a recusa tem de acontecer antes de gravar.
func TestFaturarInterestadualNaoGravaNada(t *testing.T) {
	uc, _, notas := montarFaturamento()
	pedido := pedidoPadrao()
	pedido.Destinatario.UF = "MG"

	if _, err := uc.Faturar(context.Background(), pedido, autorValido); err == nil {
		t.Fatal("faturou para outra UF com CFOP interno")
	}
	if notas.saida != nil {
		t.Error("a nota interestadual foi gravada")
	}
}

// TestConferirBaixaDaNotaBloqueiaTransmissao é a trava final: nota de beneficiamento
// sem baixa não pode ir para a SEFAZ.
func TestConferirBaixaDaNotaBloqueiaTransmissao(t *testing.T) {
	uc, repo, _ := montarFaturamento()

	if err := uc.ConferirBaixaDaNota(context.Background(), 777); err == nil {
		t.Fatal("autorizou uma nota sem baixa de saldo")
	} else if _, ok := errorsuc.AsConflict(err); !ok {
		t.Fatalf("erro deveria ser de conflito: %v", err)
	}

	// Depois do faturamento, a conferência passa.
	if _, err := uc.Faturar(context.Background(), pedidoPadrao(), autorValido); err != nil {
		t.Fatalf("Faturar() error = %v", err)
	}
	if err := uc.ConferirBaixaDaNota(context.Background(), repo.notaBaixada); err != nil {
		t.Fatalf("a conferência recusou uma nota já baixada: %v", err)
	}
}

// TestFaturarUsaSerieUmPorPadrao: a operação usa série 1; deixar em branco não pode
// gerar nota sem série.
func TestFaturarUsaSerieUmPorPadrao(t *testing.T) {
	uc, _, notas := montarFaturamento()
	if _, err := uc.Faturar(context.Background(), pedidoPadrao(), autorValido); err != nil {
		t.Fatalf("Faturar() error = %v", err)
	}
	if notas.saida.Serie != "1" {
		t.Errorf("série = %q, esperado 1", notas.saida.Serie)
	}
}

// TestRepetirFaturamentoRetomaARascunhoEmVezDeDuplicar é o teste do defeito mais
// sutil deste fluxo.
//
// O faturamento não é atômico — cria a nota em RASCUNHO, grava as linhas e depois
// baixa o saldo, sem transação possível entre os dois módulos. Quando falhava no
// meio, a mensagem mandava repetir; e repetir pedia número NOVO e criava OUTRA nota.
// A repetição anunciada como segura produzia duplicata.
func TestRepetirFaturamentoRetomaARascunhoEmVezDeDuplicar(t *testing.T) {
	// 1ª tentativa: a gravação da SEGUNDA linha falha, deixando a nota e a linha 1.
	notas := &notasFalsas{falharLinha: 2}
	uc := &UseCase{Repo: &repoFalso{remessa: remessaDaNF5911()}, Notas: notas}

	_, err := uc.Faturar(context.Background(), pedidoPadrao(), autorValido)
	if err == nil {
		t.Fatal("a falha ao gravar a linha deveria ter interrompido o faturamento")
	}
	if notas.saida == nil {
		t.Fatal("o rascunho não foi criado: o cenário da retomada não se formou")
	}
	if len(notas.itens) != 1 {
		t.Fatalf("a 1ª tentativa gravou %d linha(s), esperado 1 antes da falha", len(notas.itens))
	}
	numeroOriginal := notas.saida.NumeroNF
	idOriginal := notas.saida.ID
	pedidosDeNumero := notas.numerosPedidos

	// 2ª tentativa: o rascunho existe e a gravação não falha mais.
	notas.falharLinha = 0
	notas.rascunhoPendente = notas.saida

	resultado, err := uc.Faturar(context.Background(), pedidoPadrao(), autorValido)
	if err != nil {
		t.Fatalf("a repetição deveria concluir a nota original: %v", err)
	}

	// A prova: MESMA nota, MESMO número, e nenhum número novo pedido.
	if resultado.FiscalExitID != idOriginal {
		t.Fatalf("a repetição usou a nota %d, esperado retomar a %d", resultado.FiscalExitID, idOriginal)
	}
	if resultado.NumeroNF != numeroOriginal {
		t.Fatalf("a repetição numerou a nota como %d, esperado manter %d", resultado.NumeroNF, numeroOriginal)
	}
	if notas.numerosPedidos != pedidosDeNumero {
		t.Fatalf("a repetição pediu número novo (%d pedidos, era %d): é isso que produzia nota duplicada",
			notas.numerosPedidos, pedidosDeNumero)
	}

	// E as linhas não foram duplicadas: a linha 1 já existia e não foi recriada.
	porSequencia := map[int]int{}
	for _, it := range notas.itens {
		porSequencia[it.Sequence]++
	}
	for seq, vezes := range porSequencia {
		if vezes != 1 {
			t.Fatalf("a linha de sequência %d foi gravada %d vezes", seq, vezes)
		}
	}
	if len(notas.itens) != 3 {
		t.Fatalf("a nota ficou com %d linha(s), esperado 3 (serviço + dois materiais)", len(notas.itens))
	}
}
