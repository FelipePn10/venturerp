package customer_material_uc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/customer_material/repository"
)

// repoFalso captura o que o caso de uso decidiu, sem banco: as regras cobertas
// aqui são de negócio, não de persistência.
type repoFalso struct {
	registrada  *entity.Remessa
	movimento   domrepo.NovoMovimento
	remessa     *entity.Remessa
	notaBaixada int64
	baixas      []domrepo.MovimentoDaNota
	falharBaixa bool
	estornos    []estornoPedido
}

func (r *repoFalso) Registrar(_ context.Context, remessa *entity.Remessa) (*entity.Remessa, error) {
	r.registrada = remessa
	return remessa, nil
}
func (r *repoFalso) BuscarPorID(context.Context, int64) (*entity.Remessa, error) {
	if r.remessa == nil {
		return nil, errorsuc.NewNotFoundError("remessa não encontrada")
	}
	return r.remessa, nil
}
func (r *repoFalso) Listar(context.Context, domrepo.FiltroRemessa) ([]*entity.Remessa, error) {
	return nil, nil
}
func (r *repoFalso) SaldoPorItem(context.Context, domrepo.FiltroSaldo) ([]*entity.SaldoPorItem, error) {
	return nil, nil
}
func (r *repoFalso) RegistrarMovimento(_ context.Context, mov domrepo.NovoMovimento, _ string) (*entity.Movimento, error) {
	r.movimento = mov
	return &entity.Movimento{}, nil
}
func (r *repoFalso) MovimentosDoItem(context.Context, int64) ([]*entity.Movimento, error) {
	return nil, nil
}
func (r *repoFalso) RegistrarMovimentosDaNota(_ context.Context, fiscalExitID int64, linhas []domrepo.MovimentoDaNota, _ string) ([]*entity.Movimento, error) {
	if r.falharBaixa {
		return nil, errors.New("falha simulada na baixa")
	}
	r.notaBaixada = fiscalExitID
	r.baixas = linhas
	out := make([]*entity.Movimento, 0, len(linhas))
	for i, l := range linhas {
		out = append(out, &entity.Movimento{
			ID: int64(i + 1), RemittanceItemID: l.RemittanceItemID,
			MovementType: l.MovementType, FiscalExitID: &fiscalExitID,
		})
	}
	return out, nil
}

func (r *repoFalso) MovimentosDaNota(_ context.Context, fiscalExitID int64) ([]*entity.Movimento, error) {
	if r.notaBaixada != fiscalExitID {
		return nil, nil
	}
	return []*entity.Movimento{{ID: 1, FiscalExitID: &fiscalExitID}}, nil
}

func (r *repoFalso) Bloquear(context.Context, int64, string, string) error { return nil }
func (r *repoFalso) Desbloquear(context.Context, int64, string) error      { return nil }

func (r *repoFalso) TrilhaDaRemessa(context.Context, int64, int) ([]*entity.EventoDeAuditoria, error) {
	return nil, nil
}

// estornos guarda o que o caso de uso pediu para estornar, para o teste cobrar o
// motivo e o autor que chegaram ao repositório.
func (r *repoFalso) EstornarMovimentosDaNota(_ context.Context, fiscalExitID int64, usuario, motivo string) ([]*entity.Movimento, error) {
	r.estornos = append(r.estornos, estornoPedido{FiscalExitID: fiscalExitID, Usuario: usuario, Motivo: motivo})
	return nil, nil
}

type estornoPedido struct {
	FiscalExitID int64
	Usuario      string
	Motivo       string
}

func (r *repoFalso) Encerrar(context.Context, int64, string, string) error { return nil }

var emissao = time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)

func remessaBase() NovaRemessa {
	pedido := int64(5001)
	return NovaRemessa{
		CustomerCode: 100, NFeNumber: 16933, IssueDate: emissao, ReceivedAt: emissao,
		SalesOrderCode: &pedido,
		Itens: []ItemRecebido{{
			CustomerItemCode: "20042554", Description: "SUB CJ SOLDADO", NCM: "76169900",
			UOM: "PC", QtyInvoiced: "6", QtyReceived: "6", UnitValue: "178.283333",
		}},
	}
}

func novoUC() (*UseCase, *repoFalso) {
	r := &repoFalso{}
	return &UseCase{Repo: r, Agora: func() time.Time { return emissao }}, r
}

// TestPrazoFiscalPadraoSaoTrintaDias: o prazo é o que sustenta a cobrança do
// retorno. Deixá-lo a cargo de quem digita faria cada nota nascer com um prazo.
func TestPrazoFiscalPadraoSaoTrintaDias(t *testing.T) {
	uc, repo := novoUC()
	if _, err := uc.Receber(context.Background(), remessaBase(), "11111111-1111-1111-1111-111111111111"); err != nil {
		t.Fatalf("Receber() error = %v", err)
	}
	esperado := emissao.AddDate(0, 0, 30)
	if !repo.registrada.FiscalReturnDeadline.Equal(esperado) {
		t.Fatalf("prazo = %v, esperado %v", repo.registrada.FiscalReturnDeadline, esperado)
	}
}

// TestMaterialSemPedidoEntraBloqueado: regra explícita do cliente — material sem
// pedido não pode ir para a produção antes da regularização.
func TestMaterialSemPedidoEntraBloqueado(t *testing.T) {
	uc, repo := novoUC()
	nova := remessaBase()
	nova.SalesOrderCode = nil

	if _, err := uc.Receber(context.Background(), nova, "11111111-1111-1111-1111-111111111111"); err != nil {
		t.Fatalf("Receber() error = %v", err)
	}
	if !repo.registrada.Blocked {
		t.Fatal("material sem pedido entrou liberado")
	}
	if repo.registrada.BlockReason == nil || !strings.Contains(*repo.registrada.BlockReason, "sem pedido") {
		t.Fatalf("motivo do bloqueio não explica: %v", repo.registrada.BlockReason)
	}
}

// TestRecebimentoDivergenteEntraBloqueado: quantidade diferente da nota retém o
// material até alguém regularizar, e o motivo tem de ser informado.
func TestRecebimentoDivergenteEntraBloqueado(t *testing.T) {
	uc, repo := novoUC()
	nova := remessaBase()
	nova.Itens[0].QtyReceived = "5"
	motivo := "faltou uma peça na conferência"
	nova.Itens[0].DivergenceReason = &motivo

	if _, err := uc.Receber(context.Background(), nova, "11111111-1111-1111-1111-111111111111"); err != nil {
		t.Fatalf("Receber() error = %v", err)
	}
	if !repo.registrada.Blocked {
		t.Fatal("recebimento divergente entrou liberado")
	}
}

// TestDivergenciaSemMotivoEhRecusada: silenciar a divergência é perder o rastro
// de material que não é nosso.
func TestDivergenciaSemMotivoEhRecusada(t *testing.T) {
	uc, _ := novoUC()
	nova := remessaBase()
	nova.Itens[0].QtyReceived = "5"

	_, err := uc.Receber(context.Background(), nova, "11111111-1111-1111-1111-111111111111")
	if err == nil {
		t.Fatal("divergência sem motivo foi aceita")
	}
	if _, ok := errorsuc.AsValidation(err); !ok {
		t.Fatalf("erro deveria ser de validação: %v", err)
	}
}

// TestSemConferenciaValeAQuantidadeDaNota: o caminho comum não deve exigir
// redigitar a quantidade só para dizer que está certa.
func TestSemConferenciaValeAQuantidadeDaNota(t *testing.T) {
	uc, repo := novoUC()
	nova := remessaBase()
	nova.Itens[0].QtyReceived = ""

	if _, err := uc.Receber(context.Background(), nova, "11111111-1111-1111-1111-111111111111"); err != nil {
		t.Fatalf("Receber() error = %v", err)
	}
	item := repo.registrada.Itens[0]
	if !item.QtyReceived.Equal(item.QtyInvoiced) {
		t.Fatalf("recebido %s, faturado %s", item.QtyReceived, item.QtyInvoiced)
	}
	if repo.registrada.Blocked {
		t.Fatal("recebimento conforme a nota entrou bloqueado")
	}
}

// TestCamposFiscaisObrigatorios: sem NCM, unidade ou descrição, o retorno fiscal
// sai diferente do que o cliente mandou.
func TestCamposFiscaisObrigatorios(t *testing.T) {
	for _, caso := range []struct {
		nome   string
		mutar  func(*NovaRemessa)
		trecho string
	}{
		{"sem NCM", func(n *NovaRemessa) { n.Itens[0].NCM = "" }, "NCM"},
		{"sem unidade", func(n *NovaRemessa) { n.Itens[0].UOM = "" }, "unidade"},
		{"sem descrição", func(n *NovaRemessa) { n.Itens[0].Description = "" }, "descrição"},
		{"sem código do cliente", func(n *NovaRemessa) { n.Itens[0].CustomerItemCode = "" }, "código do item"},
		{"quantidade zero", func(n *NovaRemessa) { n.Itens[0].QtyInvoiced = "0" }, "maior que zero"},
		{"sem itens", func(n *NovaRemessa) { n.Itens = nil }, "ao menos um item"},
		{"sem cliente", func(n *NovaRemessa) { n.CustomerCode = 0 }, "cliente"},
		{"sem número da nota", func(n *NovaRemessa) { n.NFeNumber = 0 }, "NF-e"},
	} {
		t.Run(caso.nome, func(t *testing.T) {
			uc, _ := novoUC()
			nova := remessaBase()
			caso.mutar(&nova)
			_, err := uc.Receber(context.Background(), nova, "11111111-1111-1111-1111-111111111111")
			if err == nil {
				t.Fatalf("%s foi aceito", caso.nome)
			}
			if !strings.Contains(err.Error(), caso.trecho) {
				t.Fatalf("mensagem não cita %q: %v", caso.trecho, err)
			}
		})
	}
}

// TestPadroesDaNotaDeRemessa: série e CFOP da entrada têm padrão porque são
// sempre os mesmos na operação, e digitá-los a cada nota só gera erro.
func TestPadroesDaNotaDeRemessa(t *testing.T) {
	uc, repo := novoUC()
	nova := remessaBase()
	nova.NFeSeries = ""
	nova.CFOP = ""

	if _, err := uc.Receber(context.Background(), nova, "11111111-1111-1111-1111-111111111111"); err != nil {
		t.Fatalf("Receber() error = %v", err)
	}
	if repo.registrada.NFeSeries != "1" {
		t.Fatalf("série = %q, esperado 1", repo.registrada.NFeSeries)
	}
	if repo.registrada.CFOP != "5901" {
		t.Fatalf("CFOP = %q, esperado 5901 (remessa para industrialização)", repo.registrada.CFOP)
	}
	if repo.registrada.Itens[0].LineNumber != 1 {
		t.Fatalf("número da linha = %d, esperado 1", repo.registrada.Itens[0].LineNumber)
	}
}

// TestCFOPAutomaticoPorTipoDeMovimento: 5902 no retorno com o serviço, 5903 em
// sobra e sucata. Deixar isso ao operador em cada lançamento é erro fiscal à espera.
func TestCFOPAutomaticoPorTipoDeMovimento(t *testing.T) {
	for tipo, esperado := range map[entity.TipoMovimento]string{
		entity.MovimentoRetorno: "5902",
		entity.MovimentoSobra:   "5903",
		entity.MovimentoSucata:  "5903",
	} {
		uc, repo := novoUC()
		if _, err := uc.Movimentar(context.Background(), domrepo.NovoMovimento{
			RemittanceItemID: 1, MovementType: tipo, Quantity: "1", IdempotencyKey: "k",
		}, "u"); err != nil {
			t.Fatalf("Movimentar(%s) error = %v", tipo, err)
		}
		if repo.movimento.CFOP == nil || *repo.movimento.CFOP != esperado {
			t.Fatalf("CFOP de %s = %v, esperado %s", tipo, repo.movimento.CFOP, esperado)
		}
	}
}

// TestCFOPInformadoVenceOPadrao: operação atípica precisa poder sair com o CFOP
// que o responsável fiscal determinar.
func TestCFOPInformadoVenceOPadrao(t *testing.T) {
	uc, repo := novoUC()
	cfop := "6902"
	if _, err := uc.Movimentar(context.Background(), domrepo.NovoMovimento{
		RemittanceItemID: 1, MovementType: entity.MovimentoRetorno, Quantity: "1",
		CFOP: &cfop, IdempotencyKey: "k",
	}, "u"); err != nil {
		t.Fatalf("Movimentar() error = %v", err)
	}
	if *repo.movimento.CFOP != "6902" {
		t.Fatalf("CFOP = %s, o informado deveria vencer", *repo.movimento.CFOP)
	}
}

// TestAjusteNaoGanhaCFOPAutomatico: ajuste é correção interna e não tem CFOP
// óbvio; inventar um seria criar documento fiscal errado.
func TestAjusteNaoGanhaCFOPAutomatico(t *testing.T) {
	uc, repo := novoUC()
	motivo := "correção de conferência"
	if _, err := uc.Movimentar(context.Background(), domrepo.NovoMovimento{
		RemittanceItemID: 1, MovementType: entity.MovimentoAjuste, Quantity: "1",
		Reason: &motivo, IdempotencyKey: "k",
	}, "u"); err != nil {
		t.Fatalf("Movimentar() error = %v", err)
	}
	if repo.movimento.CFOP != nil {
		t.Fatalf("ajuste recebeu CFOP %v", *repo.movimento.CFOP)
	}
}

// TestPrazoAnteriorAEmissaoEhRecusado: um prazo assim tornaria a fila de cobrança
// sem sentido desde o primeiro dia.
func TestPrazoAnteriorAEmissaoEhRecusado(t *testing.T) {
	uc, _ := novoUC()
	nova := remessaBase()
	anterior := emissao.AddDate(0, 0, -1)
	nova.FiscalReturnDeadline = &anterior

	if _, err := uc.Receber(context.Background(), nova, "11111111-1111-1111-1111-111111111111"); err == nil {
		t.Fatal("prazo anterior à emissão foi aceito")
	}
}

// TestReceberExigeUsuarioDaSessao: sem autor, o insert chegava ao banco com
// created_by vazio e falhava na conversão para uuid — erro de domínio genérico
// em vez de mensagem útil. A recusa tem de acontecer antes de tocar no banco.
func TestReceberExigeUsuarioDaSessao(t *testing.T) {
	uc, repo := novoUC()
	_, err := uc.Receber(context.Background(), remessaBase(), "  ")
	if err == nil {
		t.Fatal("remessa sem usuário da sessão foi aceita")
	}
	if _, ok := errorsuc.AsValidation(err); !ok {
		t.Fatalf("erro deveria ser de validação: %v", err)
	}
	if repo.registrada != nil {
		t.Fatal("chegou a gravar apesar de não haver usuário")
	}
}

// TestReceberGravaOAutor: é o dado que responde "quem deu entrada neste material".
func TestReceberGravaOAutor(t *testing.T) {
	uc, repo := novoUC()
	const autor = "11111111-1111-1111-1111-111111111111"
	if _, err := uc.Receber(context.Background(), remessaBase(), autor); err != nil {
		t.Fatalf("Receber() error = %v", err)
	}
	if repo.registrada.CreatedBy != autor {
		t.Fatalf("autor = %q, esperado %q", repo.registrada.CreatedBy, autor)
	}
}

// TestEstornoExigeMotivoDescritivo: o estorno desfaz a baixa de material que não é
// da empresa. "erro" no motivo não explica nada a quem conferir depois, e o mínimo
// é o mesmo do cancelamento fiscal.
func TestEstornoExigeMotivoDescritivo(t *testing.T) {
	repo := &repoFalso{}
	uc := &UseCase{Repo: repo}
	ctx := context.Background()

	if _, err := uc.EstornarNota(ctx, 9001, "", "cancelamento da NF-e 5911 por erro de quantidade"); err == nil {
		t.Fatal("estorno sem usuário da sessão foi aceito")
	}
	if _, err := uc.EstornarNota(ctx, 9001, "usuario-1", "erro"); err == nil {
		t.Fatal("estorno com motivo de 4 caracteres foi aceito")
	}
	if len(repo.estornos) != 0 {
		t.Fatalf("o repositório foi chamado %d vez(es) numa recusa", len(repo.estornos))
	}

	motivo := "  cancelamento da NF-e 5911: quantidade errada  "
	if _, err := uc.EstornarNota(ctx, 9001, "usuario-1", motivo); err != nil {
		t.Fatalf("estorno válido recusado: %v", err)
	}
	if len(repo.estornos) != 1 {
		t.Fatalf("o repositório recebeu %d pedido(s), esperado 1", len(repo.estornos))
	}
	// O motivo chega aparado: espaço em volta vira ruído no histórico.
	if got := repo.estornos[0].Motivo; got != "cancelamento da NF-e 5911: quantidade errada" {
		t.Fatalf("motivo repassado = %q", got)
	}
	if repo.estornos[0].Usuario != "usuario-1" || repo.estornos[0].FiscalExitID != 9001 {
		t.Fatalf("pedido repassado errado: %+v", repo.estornos[0])
	}
}
