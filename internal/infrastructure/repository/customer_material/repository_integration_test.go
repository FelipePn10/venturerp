//go:build integration

// Testes do repositório de material de terceiro (beneficiamento).
//
// O que importa aqui não é o CRUD: é que o SALDO nunca minta. Cada teste ataca
// uma forma de o saldo passar a contradizer a realidade física do material do
// cliente — movimento sem baixa, baixa dupla, corrida entre dois faturamentos,
// status que discorda das linhas, e vazamento entre empresas.
//
//	TEST_DATABASE_URL=... go test -tags=integration ./internal/infrastructure/repository/customer_material/
package customer_material_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/customer_material/repository"
	repo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer_material"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type ambiente struct {
	repo         *repo.Repository
	ctx          context.Context
	pool         *pgxpool.Pool
	enterpriseID int64
	userID       string
}

func montar(t *testing.T) *ambiente {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	a := &ambiente{pool: pool, repo: repo.New(pool)}

	if err := pool.QueryRow(ctx,
		`INSERT INTO users (id, name, email, password, role, is_active)
		 VALUES (gen_random_uuid(), 'Benef Repo', 'benef-repo-'||$1||'@teste.local', 'x', 'ADMIN', true)
		 RETURNING id::text`, fmt.Sprint(testutil.UniqueCode())).Scan(&a.userID); err != nil {
		t.Fatalf("criando usuário: %v", err)
	}
	t.Cleanup(func() { testutil.Exec(t, pool, `DELETE FROM users WHERE id = $1::uuid`, a.userID) })

	// enterprise.code é INTEGER: a faixa cabe em int32 e não colide com cadastros reais.
	code := 900_000_000 + (testutil.UniqueCode() % 1_000_000_000)
	if err := pool.QueryRow(ctx,
		`INSERT INTO enterprise (code, name, created_by) VALUES ($1, 'BENEF REPO', $2::uuid) RETURNING id`,
		code, a.userID).Scan(&a.enterpriseID); err != nil {
		t.Fatalf("criando empresa: %v", err)
	}
	t.Cleanup(func() { testutil.Exec(t, pool, `DELETE FROM enterprise WHERE id = $1`, a.enterpriseID) })

	a.ctx = context.WithValue(ctx, contextkey.UserKey, &security.AuthUser{
		ID: a.userID, Role: "ADMIN", EnterpriseID: a.enterpriseID, EnterpriseCode: code,
	})

	// Registrado por ÚLTIMO para rodar por PRIMEIRO (t.Cleanup é LIFO): as remessas
	// referenciam empresa e usuário, então apagá-las antes evita deixar lixo na base
	// de teste quando os cleanups de empresa e usuário forem recusados pelas FKs.
	// Os itens e movimentos saem por ON DELETE CASCADE.
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM customer_material_remittances WHERE enterprise_id = $1`, a.enterpriseID)
	})
	return a
}

// remessa registra uma remessa com uma linha da quantidade informada.
func (a *ambiente) remessa(t *testing.T, recebido string) *entity.Remessa {
	t.Helper()
	qtd, _ := decimal.NewFromString(recebido)
	emissao := time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)
	nova := &entity.Remessa{
		CustomerCode:         100,
		NFeNumber:            testutil.UniqueCode(),
		NFeSeries:            "1",
		CFOP:                 "5901",
		IssueDate:            emissao,
		ReceivedAt:           emissao,
		FiscalReturnDeadline: emissao.AddDate(0, 0, 30),
		TotalValue:           decimal.RequireFromString("1069.73"),
		Status:               entity.StatusAberta,
		CreatedBy:            a.userID,
		Itens: []*entity.ItemRemessa{{
			LineNumber:       1,
			CustomerItemCode: "20042554",
			Description:      "SUB CJ SOLDADO DA LAMINA",
			NCM:              "76169900",
			UOM:              "PC",
			QtyInvoiced:      qtd,
			QtyReceived:      qtd,
			UnitValue:        decimal.RequireFromString("178.283333"),
		}},
	}
	criada, err := a.repo.Registrar(a.ctx, nova)
	if err != nil {
		t.Fatalf("registrando remessa: %v", err)
	}
	return criada
}

func (a *ambiente) recarregar(t *testing.T, id int64) *entity.Remessa {
	t.Helper()
	r, err := a.repo.BuscarPorID(a.ctx, id)
	if err != nil {
		t.Fatalf("recarregando remessa: %v", err)
	}
	return r
}

// TestRemessaSemItemEhRecusada: material físico na empresa sem linha de saldo é
// material sem controle, que é justamente o que este módulo existe para acabar.
func TestRemessaSemItemEhRecusada(t *testing.T) {
	a := montar(t)
	emissao := time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)
	_, err := a.repo.Registrar(a.ctx, &entity.Remessa{
		CustomerCode: 100, NFeNumber: testutil.UniqueCode(), NFeSeries: "1", CFOP: "5901",
		IssueDate: emissao, ReceivedAt: emissao, FiscalReturnDeadline: emissao.AddDate(0, 0, 30),
		Status: entity.StatusAberta, CreatedBy: a.userID,
	})
	if err == nil {
		t.Fatal("remessa sem item foi aceita")
	}
	if _, ok := errorsuc.AsValidation(err); !ok {
		t.Fatalf("erro deveria ser de validação, veio: %v", err)
	}
}

// TestMovimentoBaixaSaldoEAtualizaSituacao: o movimento e a baixa da linha vivem
// na mesma transação, e a situação da remessa é derivada do saldo.
func TestMovimentoBaixaSaldoEAtualizaSituacao(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")
	item := r.Itens[0]

	if r.Status != entity.StatusAberta {
		t.Fatalf("remessa nova em %s, esperado ABERTA", r.Status)
	}

	cfop := "5902"
	if _, err := a.repo.RegistrarMovimento(a.ctx, domrepo.NovoMovimento{
		RemittanceItemID: item.ID, MovementType: entity.MovimentoRetorno,
		Quantity: "4", CFOP: &cfop, IdempotencyKey: "ret-parcial-" + fmt.Sprint(item.ID),
	}, a.userID); err != nil {
		t.Fatalf("retorno parcial recusado: %v", err)
	}

	r = a.recarregar(t, r.ID)
	if got := r.Itens[0].BalanceQty.String(); got != "2" {
		t.Fatalf("saldo = %s, esperado 2", got)
	}
	if r.Status != entity.StatusParcial {
		t.Fatalf("situação = %s, esperado PARCIAL", r.Status)
	}

	// Fechando o saldo, a remessa encerra sozinha — sem ninguém mudar status à mão.
	if _, err := a.repo.RegistrarMovimento(a.ctx, domrepo.NovoMovimento{
		RemittanceItemID: item.ID, MovementType: entity.MovimentoRetorno,
		Quantity: "2", CFOP: &cfop, IdempotencyKey: "ret-final-" + fmt.Sprint(item.ID),
	}, a.userID); err != nil {
		t.Fatalf("retorno final recusado: %v", err)
	}
	r = a.recarregar(t, r.ID)
	if got := r.Itens[0].BalanceQty.String(); got != "0" {
		t.Fatalf("saldo final = %s, esperado 0", got)
	}
	if r.Status != entity.StatusEncerrada {
		t.Fatalf("situação final = %s, esperado ENCERRADA", r.Status)
	}
}

// TestMovimentoNaoPassaDoSaldo: a recusa tem de vir com mensagem que diga o saldo,
// não com erro de constraint do PostgreSQL.
func TestMovimentoNaoPassaDoSaldo(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")
	item := r.Itens[0]

	_, err := a.repo.RegistrarMovimento(a.ctx, domrepo.NovoMovimento{
		RemittanceItemID: item.ID, MovementType: entity.MovimentoRetorno,
		Quantity: "7", IdempotencyKey: "excede-" + fmt.Sprint(item.ID),
	}, a.userID)
	if err == nil {
		t.Fatal("devolver 7 de 6 foi aceito")
	}
	if _, ok := errorsuc.AsValidation(err); !ok {
		t.Fatalf("erro deveria ser de validação, veio: %v", err)
	}
	if !strings.Contains(err.Error(), "saldo") {
		t.Fatalf("mensagem não menciona o saldo: %v", err)
	}
	// E nada pode ter sido gravado.
	if movs, err := a.repo.MovimentosDoItem(a.ctx, item.ID); err != nil || len(movs) != 0 {
		t.Fatalf("movimento recusado deixou %d lançamento(s) (err=%v)", len(movs), err)
	}
}

// TestIdempotenciaDevolveOMesmoMovimento: a tela pode reenviar por timeout, e
// baixar o saldo duas vezes seria material do cliente perdido no controle.
func TestIdempotenciaDevolveOMesmoMovimento(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")
	item := r.Itens[0]
	pedido := domrepo.NovoMovimento{
		RemittanceItemID: item.ID, MovementType: entity.MovimentoRetorno,
		Quantity: "2", IdempotencyKey: "repete-" + fmt.Sprint(item.ID),
	}

	primeiro, err := a.repo.RegistrarMovimento(a.ctx, pedido, a.userID)
	if err != nil {
		t.Fatalf("primeiro movimento: %v", err)
	}
	segundo, err := a.repo.RegistrarMovimento(a.ctx, pedido, a.userID)
	if err != nil {
		t.Fatalf("repetição deveria devolver o original: %v", err)
	}
	if primeiro.ID != segundo.ID {
		t.Fatalf("ids diferentes (%d e %d): baixou duas vezes", primeiro.ID, segundo.ID)
	}
	if got := a.recarregar(t, r.ID).Itens[0].BalanceQty.String(); got != "4" {
		t.Fatalf("saldo = %s, esperado 4 — a repetição baixou de novo", got)
	}

	// Mesma chave com outro conteúdo é erro, não repetição.
	outro := pedido
	outro.Quantity = "3"
	if _, err := a.repo.RegistrarMovimento(a.ctx, outro, a.userID); err == nil {
		t.Fatal("mesma chave com quantidade diferente foi aceita")
	} else if _, ok := errorsuc.AsConflict(err); !ok {
		t.Fatalf("erro deveria ser de conflito, veio: %v", err)
	}
}

// TestFaturamentosSimultaneosNaoEstouramOSaldo: dois usuários faturando a mesma
// linha ao mesmo tempo poderiam ler o mesmo saldo e devolver mais do que entrou.
// O FOR UPDATE é o que impede — sem ele, um dos dois passaria.
func TestFaturamentosSimultaneosNaoEstouramOSaldo(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")
	item := r.Itens[0]

	var wg sync.WaitGroup
	erros := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			// Cada um tenta devolver 4: juntos passariam de 6.
			_, erros[n] = a.repo.RegistrarMovimento(a.ctx, domrepo.NovoMovimento{
				RemittanceItemID: item.ID, MovementType: entity.MovimentoRetorno,
				Quantity: "4", IdempotencyKey: fmt.Sprintf("corrida-%d-%d", item.ID, n),
			}, a.userID)
		}(i)
	}
	wg.Wait()

	sucessos := 0
	for _, err := range erros {
		if err == nil {
			sucessos++
		}
	}
	if sucessos != 1 {
		t.Fatalf("%d dos 2 faturamentos simultâneos passaram; esperado exatamente 1 (erros: %v)", sucessos, erros)
	}
	if got := a.recarregar(t, r.ID).Itens[0].BalanceQty.String(); got != "2" {
		t.Fatalf("saldo = %s, esperado 2 — a corrida furou o controle", got)
	}
}

// TestRemessaBloqueadaNaoMovimenta: material sem pedido ou divergente fica retido
// até regularizar, e é a movimentação que precisa ser barrada.
func TestRemessaBloqueadaNaoMovimenta(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")
	item := r.Itens[0]

	if err := a.repo.Bloquear(a.ctx, r.ID, "material recebido sem pedido cadastrado", a.userID); err != nil {
		t.Fatalf("bloqueando: %v", err)
	}
	_, err := a.repo.RegistrarMovimento(a.ctx, domrepo.NovoMovimento{
		RemittanceItemID: item.ID, MovementType: entity.MovimentoRetorno,
		Quantity: "1", IdempotencyKey: "bloq-" + fmt.Sprint(item.ID),
	}, a.userID)
	if err == nil {
		t.Fatal("remessa bloqueada aceitou movimento")
	}
	if _, ok := errorsuc.AsConflict(err); !ok {
		t.Fatalf("erro deveria ser de conflito, veio: %v", err)
	}

	// Bloquear sem motivo não passa.
	if err := a.repo.Bloquear(a.ctx, r.ID, "   ", a.userID); err == nil {
		t.Fatal("bloqueio sem motivo foi aceito")
	}
	// Depois de liberar, movimenta.
	if err := a.repo.Desbloquear(a.ctx, r.ID, a.userID); err != nil {
		t.Fatalf("desbloqueando: %v", err)
	}
	if _, err := a.repo.RegistrarMovimento(a.ctx, domrepo.NovoMovimento{
		RemittanceItemID: item.ID, MovementType: entity.MovimentoRetorno,
		Quantity: "1", IdempotencyKey: "liberado-" + fmt.Sprint(item.ID),
	}, a.userID); err != nil {
		t.Fatalf("remessa liberada recusou movimento: %v", err)
	}
}

// TestSucataExigeDestinacaoEAjusteExigeMotivo: as duas regras que a auditoria
// fiscal do cliente pediu explicitamente.
func TestSucataExigeDestinacaoEAjusteExigeMotivo(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")
	item := r.Itens[0]

	if _, err := a.repo.RegistrarMovimento(a.ctx, domrepo.NovoMovimento{
		RemittanceItemID: item.ID, MovementType: entity.MovimentoSucata,
		Quantity: "1", IdempotencyKey: "sucata-sem-" + fmt.Sprint(item.ID),
	}, a.userID); err == nil {
		t.Fatal("sucata sem destinação foi aceita")
	}
	if _, err := a.repo.RegistrarMovimento(a.ctx, domrepo.NovoMovimento{
		RemittanceItemID: item.ID, MovementType: entity.MovimentoAjuste,
		Quantity: "1", IdempotencyKey: "ajuste-sem-" + fmt.Sprint(item.ID),
	}, a.userID); err == nil {
		t.Fatal("ajuste sem justificativa foi aceito")
	}

	destino := entity.DestinoCliente
	motivo := "sucata pesada e devolvida ao cliente"
	cfop := "5903"
	if _, err := a.repo.RegistrarMovimento(a.ctx, domrepo.NovoMovimento{
		RemittanceItemID: item.ID, MovementType: entity.MovimentoSucata,
		Quantity: "1", CFOP: &cfop, ScrapDestination: &destino, Reason: &motivo,
		IdempotencyKey: "sucata-ok-" + fmt.Sprint(item.ID),
	}, a.userID); err != nil {
		t.Fatalf("sucata com destinação recusada: %v", err)
	}
	if got := a.recarregar(t, r.ID).Itens[0].QtyScrapped.String(); got != "1" {
		t.Fatalf("sucata acumulada = %s, esperado 1", got)
	}
}

// TestEncerrarComSaldoExigeMotivo: encerrar deixando material do cliente aqui é
// exceção aprovada, e o que sobrou tem de ficar registrado.
func TestEncerrarComSaldoExigeMotivo(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")

	if err := a.repo.Encerrar(a.ctx, r.ID, "", a.userID); err == nil {
		t.Fatal("encerrou com saldo sem motivo")
	} else if _, ok := errorsuc.AsValidation(err); !ok {
		t.Fatalf("erro deveria ser de validação, veio: %v", err)
	}
	if err := a.repo.Encerrar(a.ctx, r.ID, "cliente autorizou encerrar com 6 PC em saldo", a.userID); err != nil {
		t.Fatalf("encerramento aprovado recusado: %v", err)
	}
	fechada := a.recarregar(t, r.ID)
	if fechada.Status != entity.StatusEncerrada {
		t.Fatalf("situação = %s, esperado ENCERRADA", fechada.Status)
	}
	if fechada.CloseReason == nil || fechada.ClosedBy == nil {
		t.Fatal("encerramento não registrou motivo e responsável")
	}
	// Uma remessa encerrada não volta a ABERTA por movimento posterior.
	if fechada.StatusCalculado() != entity.StatusEncerrada {
		t.Fatal("status calculado reabriu uma remessa encerrada")
	}
}

// TestSaldoPorItemAgregaPorClienteEItem: é a visão que o inventário mostra ao lado
// do estoque próprio, e ela some quando o saldo zera.
func TestSaldoPorItemAgregaPorClienteEItem(t *testing.T) {
	a := montar(t)
	primeira := a.remessa(t, "6")
	segunda := a.remessa(t, "4")

	saldos, err := a.repo.SaldoPorItem(a.ctx, domrepo.FiltroSaldo{})
	if err != nil {
		t.Fatalf("saldo por item: %v", err)
	}
	if len(saldos) != 1 {
		t.Fatalf("%d linhas de saldo; as duas remessas são do mesmo item e cliente, esperado 1", len(saldos))
	}
	if got := saldos[0].Balance.String(); got != "10" {
		t.Fatalf("saldo agregado = %s, esperado 10", got)
	}
	if saldos[0].RemessasAbertas != 2 {
		t.Fatalf("remessas que sustentam o saldo = %d, esperado 2", saldos[0].RemessasAbertas)
	}
	if saldos[0].PrazoMaisProximo == nil {
		t.Fatal("prazo mais próximo não foi trazido; é o que ordena a fila de retorno")
	}

	// Zerando as duas, o item sai da visão de saldo.
	for _, r := range []*entity.Remessa{primeira, segunda} {
		item := r.Itens[0]
		if _, err := a.repo.RegistrarMovimento(a.ctx, domrepo.NovoMovimento{
			RemittanceItemID: item.ID, MovementType: entity.MovimentoRetorno,
			Quantity: item.QtyReceived.String(), IdempotencyKey: "zera-" + fmt.Sprint(item.ID),
		}, a.userID); err != nil {
			t.Fatalf("zerando remessa: %v", err)
		}
	}
	saldos, err = a.repo.SaldoPorItem(a.ctx, domrepo.FiltroSaldo{})
	if err != nil {
		t.Fatalf("saldo por item: %v", err)
	}
	if len(saldos) != 0 {
		t.Fatalf("%d linhas de saldo após zerar tudo, esperado 0", len(saldos))
	}
}

// TestFiltroDePrazoSustentaACobranca: os 30 dias do retorno fiscal só viram
// controle se houver como listar quem está vencendo.
func TestFiltroDePrazoSustentaACobranca(t *testing.T) {
	a := montar(t)
	a.remessa(t, "6") // prazo em 17/10/2026

	antes := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	depois := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)

	if lista, err := a.repo.Listar(a.ctx, domrepo.FiltroRemessa{VencendoAte: &antes}); err != nil || len(lista) != 0 {
		t.Fatalf("remessa apareceu antes do prazo: %d (err=%v)", len(lista), err)
	}
	if lista, err := a.repo.Listar(a.ctx, domrepo.FiltroRemessa{VencendoAte: &depois}); err != nil || len(lista) != 1 {
		t.Fatalf("remessa não apareceu no prazo: %d (err=%v)", len(lista), err)
	}
	if lista, err := a.repo.Listar(a.ctx, domrepo.FiltroRemessa{SomenteComSaldo: true}); err != nil || len(lista) != 1 {
		t.Fatalf("filtro de saldo: %d (err=%v)", len(lista), err)
	}
}

// TestFilaDePrazoIgnoraEncerradas: remessa encerrada não tem retorno pendente, e
// listá-la poluiria a cobrança. É também o que permite ao Postgres usar o índice
// parcial em vez de varrer a tabela.
func TestFilaDePrazoIgnoraEncerradas(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")
	depois := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	if lista, err := a.repo.Listar(a.ctx, domrepo.FiltroRemessa{VencendoAte: &depois}); err != nil || len(lista) != 1 {
		t.Fatalf("a remessa aberta não apareceu na fila: %d (err=%v)", len(lista), err)
	}

	if err := a.repo.Encerrar(a.ctx, r.ID, "encerrada no teste", a.userID); err != nil {
		t.Fatalf("encerrando: %v", err)
	}
	if lista, err := a.repo.Listar(a.ctx, domrepo.FiltroRemessa{VencendoAte: &depois}); err != nil || len(lista) != 0 {
		t.Fatalf("remessa encerrada apareceu na fila de prazo: %d (err=%v)", len(lista), err)
	}

	// Mas quem pede ENCERRADA explicitamente continua enxergando: o filtro de
	// situação do chamador vence o padrão.
	lista, err := a.repo.Listar(a.ctx, domrepo.FiltroRemessa{
		VencendoAte: &depois, Status: []entity.StatusRemessa{entity.StatusEncerrada},
	})
	if err != nil || len(lista) != 1 {
		t.Fatalf("o filtro explícito de ENCERRADA foi ignorado: %d (err=%v)", len(lista), err)
	}
}

// TestNaoVazaEntreEmpresas: a Usimac e a Tecnofer rodam em bases separadas, mas o
// filtro por empresa é a segunda barreira — e ele tem de valer aqui também.
func TestNaoVazaEntreEmpresas(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")

	outra := context.WithValue(context.Background(), contextkey.UserKey, &security.AuthUser{
		ID: a.userID, Role: "ADMIN", EnterpriseID: a.enterpriseID + 90_000, EnterpriseCode: a.enterpriseID + 90_000,
	})
	if lista, err := a.repo.Listar(outra, domrepo.FiltroRemessa{}); err != nil || len(lista) != 0 {
		t.Fatalf("outra empresa viu %d remessa(s) (err=%v)", len(lista), err)
	}
	if _, err := a.repo.BuscarPorID(outra, r.ID); err == nil {
		t.Fatal("outra empresa leu a remessa pelo id")
	}
	if saldos, err := a.repo.SaldoPorItem(outra, domrepo.FiltroSaldo{}); err != nil || len(saldos) != 0 {
		t.Fatalf("outra empresa viu %d saldo(s) (err=%v)", len(saldos), err)
	}
	if _, err := a.repo.RegistrarMovimento(outra, domrepo.NovoMovimento{
		RemittanceItemID: r.Itens[0].ID, MovementType: entity.MovimentoRetorno,
		Quantity: "1", IdempotencyKey: "vaza-" + fmt.Sprint(r.Itens[0].ID),
	}, a.userID); err == nil {
		t.Fatal("outra empresa movimentou material que não é dela")
	}
	if err := a.repo.Encerrar(outra, r.ID, "tentativa", a.userID); err == nil {
		t.Fatal("outra empresa encerrou a remessa")
	}

	// Os caminhos do FATURAMENTO também são por empresa. Sem isto, a nota de uma
	// empresa poderia baixar o saldo de material do cliente da outra.
	if _, err := a.repo.RegistrarMovimentosDaNota(outra, 4242, []domrepo.MovimentoDaNota{{
		RemittanceItemID: r.Itens[0].ID, MovementType: entity.MovimentoRetorno, Quantity: "1",
	}}, a.userID); err == nil {
		t.Fatal("outra empresa baixou o saldo por nota em material que não é dela")
	}

	// E a baixa da empresa dona não aparece para a outra: é a conferência que libera
	// a transmissão, e ver a baixa alheia liberaria uma nota indevidamente.
	if _, err := a.repo.RegistrarMovimentosDaNota(a.ctx, 4242, []domrepo.MovimentoDaNota{{
		RemittanceItemID: r.Itens[0].ID, MovementType: entity.MovimentoRetorno, Quantity: "1",
	}}, a.userID); err != nil {
		t.Fatalf("a empresa dona não conseguiu baixar por nota: %v", err)
	}
	if movs, err := a.repo.MovimentosDaNota(outra, 4242); err != nil || len(movs) != 0 {
		t.Fatalf("outra empresa viu %d baixa(s) da nota alheia (err=%v)", len(movs), err)
	}
	if movs, err := a.repo.MovimentosDaNota(a.ctx, 4242); err != nil || len(movs) != 1 {
		t.Fatalf("a empresa dona viu %d baixa(s), esperado 1 (err=%v)", len(movs), err)
	}
}

// TestBaixaPorNotaEhIdempotente: é o que torna recuperável uma falha entre criar a
// nota e baixar o saldo. Repetir não pode baixar duas vezes.
func TestBaixaPorNotaEhIdempotente(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "10")
	item := r.Itens[0]
	linhas := []domrepo.MovimentoDaNota{{
		RemittanceItemID: item.ID, MovementType: entity.MovimentoRetorno,
		Quantity: "4", UnitValue: "36.93", CFOP: "5902",
	}}

	primeira, err := a.repo.RegistrarMovimentosDaNota(a.ctx, 9001, linhas, a.userID)
	if err != nil {
		t.Fatalf("primeira baixa: %v", err)
	}
	segunda, err := a.repo.RegistrarMovimentosDaNota(a.ctx, 9001, linhas, a.userID)
	if err != nil {
		t.Fatalf("repetir deveria devolver a baixa original: %v", err)
	}
	if len(primeira) != 1 || len(segunda) != 1 || primeira[0].ID != segunda[0].ID {
		t.Fatalf("a repetição criou movimento novo: %v e %v", primeira, segunda)
	}
	if got := a.recarregar(t, r.ID).Itens[0].BalanceQty.String(); got != "6" {
		t.Fatalf("saldo = %s, esperado 6 — a repetição baixou de novo", got)
	}
}

// TestBaixaPorNotaNaoPassaDoSaldo, somando as linhas da própria nota: conferir uma
// por uma deixaria passar 6+6 sobre um saldo de 10.
func TestBaixaPorNotaNaoPassaDoSaldo(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "10")
	item := r.Itens[0]

	_, err := a.repo.RegistrarMovimentosDaNota(a.ctx, 9002, []domrepo.MovimentoDaNota{
		{RemittanceItemID: item.ID, MovementType: entity.MovimentoRetorno, Quantity: "6"},
		{RemittanceItemID: item.ID, MovementType: entity.MovimentoSobra, Quantity: "6"},
	}, a.userID)
	if err == nil {
		t.Fatal("a nota devolveu 6+6 sobre um saldo de 10")
	}
	if movs, err := a.repo.MovimentosDaNota(a.ctx, 9002); err != nil || len(movs) != 0 {
		t.Fatalf("a recusa deixou %d movimento(s) gravado(s) (err=%v)", len(movs), err)
	}
	if got := a.recarregar(t, r.ID).Itens[0].BalanceQty.String(); got != "10" {
		t.Fatalf("saldo = %s, esperado 10 — a recusa mexeu no saldo", got)
	}
}

// TestBaixaPorNotaRecusaRemessaBloqueada: material retido não pode virar nota.
func TestBaixaPorNotaRecusaRemessaBloqueada(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "10")
	if err := a.repo.Bloquear(a.ctx, r.ID, "sem pedido", a.userID); err != nil {
		t.Fatalf("bloqueando: %v", err)
	}
	if _, err := a.repo.RegistrarMovimentosDaNota(a.ctx, 9003, []domrepo.MovimentoDaNota{{
		RemittanceItemID: r.Itens[0].ID, MovementType: entity.MovimentoRetorno, Quantity: "1",
	}}, a.userID); err == nil {
		t.Fatal("faturou uma remessa bloqueada")
	}
}

// TestNFeDuplicadaEhConflito: o erro operacional mais provável no recebimento tem
// de chegar como conflito explicado, não como violação de índice.
func TestNFeDuplicadaEhConflito(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")

	emissao := time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)
	_, err := a.repo.Registrar(a.ctx, &entity.Remessa{
		CustomerCode: r.CustomerCode, NFeNumber: r.NFeNumber, NFeSeries: r.NFeSeries, CFOP: "5901",
		IssueDate: emissao, ReceivedAt: emissao, FiscalReturnDeadline: emissao.AddDate(0, 0, 30),
		Status: entity.StatusAberta, CreatedBy: a.userID,
		Itens: []*entity.ItemRemessa{{
			LineNumber: 1, CustomerItemCode: "X", Description: "X", NCM: "76169900", UOM: "PC",
			QtyInvoiced: decimal.RequireFromString("1"), QtyReceived: decimal.RequireFromString("1"),
		}},
	})
	if err == nil {
		t.Fatal("a mesma NF-e do mesmo cliente entrou duas vezes")
	}
	if _, ok := errorsuc.AsConflict(err); !ok {
		t.Fatalf("erro deveria ser de conflito, veio: %v", err)
	}
	if !strings.Contains(err.Error(), "NF-e") {
		t.Fatalf("mensagem não explica o conflito: %v", err)
	}
}
